package openrouter

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"llm-budget-tracker/internal/catalog"
	"llm-budget-tracker/internal/config"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
	"llm-budget-tracker/internal/service"
)

func TestOpenRouterFetchCatalogRefreshesCacheWithoutClobberingOverrides(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/models" {
			t.Fatalf("path = %q, want /api/v1/models", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}

		_ = json.NewEncoder(w).Encode(modelsResponse{Data: []modelPayload{{
			ID:            "anthropic/claude-3.5-sonnet",
			CanonicalSlug: "anthropic/claude-3.5-sonnet-20241022",
			Pricing: pricingPayload{
				Prompt:          "0.000003",
				Completion:      "0.000015",
				InputCacheRead:  "0.0000003",
				InputCacheWrite: "0.00000375",
				WebSearch:       "0.01",
			},
		}}})
	}))
	defer server.Close()

	loader, err := catalog.New(catalog.Options{OverridePath: filepath.Join("testdata", "prices.override.yaml")})
	if err != nil {
		t.Fatalf("catalog.New() error = %v", err)
	}

	client := NewClient(Options{
		APIKey:     "test-key",
		APIBaseURL: server.URL + "/api/v1",
		Now: func() time.Time {
			return time.Date(2026, 4, 17, 15, 4, 5, 0, time.UTC)
		},
	})

	syncService := service.NewCatalogSyncService(client, loader)
	snapshot, err := syncService.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if snapshot.Source != CacheSource {
		t.Fatalf("snapshot.Source = %q, want %q", snapshot.Source, CacheSource)
	}
	if len(snapshot.Entries) != 1 {
		t.Fatalf("len(snapshot.Entries) = %d, want 1", len(snapshot.Entries))
	}

	entry := snapshot.Entries[0]
	if entry.ModelID != "anthropic/claude-3.5-sonnet-20241022" || entry.LookupKey != "anthropic/claude-3.5-sonnet" {
		t.Fatalf("snapshot entry = %+v, want canonical/model alias split", entry)
	}
	if entry.InputUSDPer1M != 3 || entry.OutputUSDPer1M != 15 || entry.CacheReadUSDPer1M != 0.3 || entry.CacheWriteUSDPer1M != 3.75 || entry.ToolUSDPerInvocation != 0.01 {
		t.Fatalf("snapshot entry prices = %+v, want OpenRouter pricing normalized", entry)
	}

	cacheRef, err := domain.NewModelPricingRef(domain.ProviderOpenRouter, "anthropic/claude-3.5-sonnet-20241022", "anthropic/claude-3.5-sonnet")
	if err != nil {
		t.Fatalf("NewModelPricingRef() error = %v", err)
	}

	price, err := loader.LookupModelPrice(context.Background(), cacheRef, time.Now())
	if err != nil {
		t.Fatalf("LookupModelPrice() error = %v", err)
	}
	if price.InputUSDPer1M != 9.99 || price.OutputUSDPer1M != 19.99 {
		t.Fatalf("LookupModelPrice() = %+v, want override values to win", price)
	}

	cacheOnly, err := domain.NewModelPricingRef(domain.ProviderOpenRouter, "openrouter/anthropic/claude-sonnet-4", "openrouter/anthropic/claude-sonnet-4")
	if err != nil {
		t.Fatalf("NewModelPricingRef() cacheOnly error = %v", err)
	}

	_, err = loader.LookupModelPrice(context.Background(), cacheOnly, time.Now())
	if err == nil {
		t.Fatal("LookupModelPrice() cacheOnly error = nil, want synced cache miss because override fixture is narrow")
	}

	updatedSnapshot := loader.CacheSnapshot()
	if updatedSnapshot.Version != "sync-2026-04-17T15:04:05Z" || !updatedSnapshot.SyncedAt.Equal(time.Date(2026, 4, 17, 15, 4, 5, 0, time.UTC)) {
		t.Fatalf("CacheSnapshot() = %+v, want sync metadata refreshed", updatedSnapshot)
	}
	if len(updatedSnapshot.Entries) != 1 {
		t.Fatalf("len(CacheSnapshot().Entries) = %d, want 1", len(updatedSnapshot.Entries))
	}

	ref, err := domain.NewModelPricingRef(domain.ProviderOpenRouter, "anthropic/claude-3.5-sonnet-20241022", "anthropic/claude-3.5-sonnet")
	if err != nil {
		t.Fatalf("NewModelPricingRef() alias error = %v", err)
	}

	price, err = loader.LookupModelPrice(context.Background(), ref, time.Now())
	if err != nil {
		t.Fatalf("LookupModelPrice() alias error = %v", err)
	}
	if price.InputUSDPer1M != 9.99 || price.OutputUSDPer1M != 19.99 {
		t.Fatalf("LookupModelPrice() alias = %+v, want override precedence preserved", price)
	}
	if len(updatedSnapshot.Entries) != 1 {
		t.Fatalf("Cache snapshot entries = %d, want 1", len(updatedSnapshot.Entries))
	}
	if updatedSnapshot.Entries[0].InputUSDPer1M != 3 {
		t.Fatalf("Cache snapshot entry = %+v, want synced cache preserved separately from override", updatedSnapshot.Entries[0])
	}
	if got := loader.Warnings(); len(got) != 0 {
		t.Fatalf("Warnings() = %v, want none", got)
	}
	if !strings.HasPrefix(updatedSnapshot.Version, "sync-") {
		t.Fatalf("CacheSnapshot().Version = %q, want sync prefix", updatedSnapshot.Version)
	}
}

func TestOpenRouterMissingAPIKeyReturnsTypedWarningState(t *testing.T) {
	t.Parallel()

	client := NewClient(Options{})
	if client.Configured() {
		t.Fatal("Configured() = true, want false")
	}

	warning := client.WarningState()
	if warning == nil {
		t.Fatal("WarningState() = nil, want missing-key warning")
	}
	if warning.Code != WarningCodeMissingAPIKey {
		t.Fatalf("warning.Code = %q, want %q", warning.Code, WarningCodeMissingAPIKey)
	}
	if warning.SecretID != config.SecretOpenRouterAPIKey {
		t.Fatalf("warning.SecretID = %q, want %q", warning.SecretID, config.SecretOpenRouterAPIKey)
	}

	if _, err := client.FetchCatalog(context.Background()); err == nil {
		t.Fatal("FetchCatalog() error = nil, want typed warning")
	} else if typed, ok := err.(*WarningState); !ok || typed.Code != WarningCodeMissingAPIKey {
		t.Fatalf("FetchCatalog() error = %T %v, want missing-key WarningState", err, err)
	}

	if _, err := client.FetchUsageEntries(context.Background(), ports.OpenRouterActivityOptions{}); err == nil {
		t.Fatal("FetchUsageEntries() error = nil, want typed warning")
	} else if typed, ok := err.(*WarningState); !ok || typed.Code != WarningCodeMissingAPIKey {
		t.Fatalf("FetchUsageEntries() error = %T %v, want missing-key WarningState", err, err)
	}

	syncService := service.NewCatalogSyncService(client, nil)
	if syncService == nil {
		t.Fatal("NewCatalogSyncService() = nil, want service available despite missing key")
	}
}

func TestOpenRouterFetchUsageEntriesNormalizesActivity(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/activity" {
			t.Fatalf("path = %q, want /api/v1/activity", r.URL.Path)
		}
		if got := r.URL.Query().Get("date"); got != "2026-04-16" {
			t.Fatalf("date query = %q, want 2026-04-16", got)
		}
		if got := r.URL.Query().Get("api_key_hash"); got != "hash123" {
			t.Fatalf("api_key_hash query = %q, want hash123", got)
		}
		if got := r.URL.Query().Get("user_id"); got != "user_42" {
			t.Fatalf("user_id query = %q, want user_42", got)
		}

		_ = json.NewEncoder(w).Encode(activityResponse{Data: []activityPayload{{
			Date:               "2026-04-16",
			Model:              "openai/gpt-4.1",
			ModelPermaslug:     "openai/gpt-4.1-2025-04-14",
			EndpointID:         "endpoint_abc123",
			ProviderName:       "OpenAI",
			Usage:              1.25,
			BYOKUsageInference: 0.75,
			Requests:           6,
			PromptTokens:       1200,
			CompletionTokens:   340,
			ReasoningTokens:    25,
		}}})
	}))
	defer server.Close()

	client := NewClient(Options{APIKey: "test-key", APIBaseURL: server.URL + "/api/v1"})
	entries, err := client.FetchUsageEntries(context.Background(), ports.OpenRouterActivityOptions{
		Date:       time.Date(2026, 4, 16, 23, 59, 0, 0, time.FixedZone("UTC+9", 9*60*60)),
		APIKeyHash: "hash123",
		UserID:     "user_42",
	})
	if err != nil {
		t.Fatalf("FetchUsageEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}

	entry := entries[0]
	if entry.Source != domain.UsageSourceOpenRouter || entry.Provider != domain.ProviderOpenRouter || entry.BillingMode != domain.BillingModeOpenRouter {
		t.Fatalf("entry source/provider/billing = %+v, want OpenRouter normalized usage", entry)
	}
	if !entry.OccurredAt.Equal(time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("entry.OccurredAt = %s, want UTC day timestamp", entry.OccurredAt)
	}
	if entry.Tokens.InputTokens != 1200 || entry.Tokens.OutputTokens != 340 {
		t.Fatalf("entry.Tokens = %+v, want prompt/completion totals", entry.Tokens)
	}
	if entry.CostBreakdown.TotalUSD != 2.0 || entry.CostBreakdown.FlatUSD != 2.0 {
		t.Fatalf("entry.CostBreakdown = %+v, want total usage + BYOK preserved", entry.CostBreakdown)
	}
	if entry.PricingRef == nil {
		t.Fatal("entry.PricingRef = nil, want OpenRouter model reference")
	}
	if entry.PricingRef.ModelID != "openai/gpt-4.1-2025-04-14" || entry.PricingRef.PricingLookupKey != "openai/gpt-4.1" {
		t.Fatalf("entry.PricingRef = %+v, want permaslug/model mapping", entry.PricingRef)
	}
	if entry.ExternalID != "endpoint_abc123" {
		t.Fatalf("entry.ExternalID = %q, want endpoint id", entry.ExternalID)
	}
	if !strings.Contains(entry.SessionID, "2026-04-16") || !strings.Contains(entry.SessionID, "openai-gpt-4.1") {
		t.Fatalf("entry.SessionID = %q, want day/model granularity", entry.SessionID)
	}
	if entry.AgentName != "" || entry.ProjectName != "" {
		t.Fatalf("entry agent/project = %q/%q, want empty privacy-safe defaults", entry.AgentName, entry.ProjectName)
	}
}

func TestOpenRouterNormalizeUsageImportUsesPricingAndActualTotals(t *testing.T) {
	t.Parallel()

	client := NewClient(Options{APIKey: "test-key"})
	entry, err := client.NormalizeUsageImport(UsageImport{
		OccurredAt:       time.Date(2026, 4, 17, 10, 30, 0, 0, time.FixedZone("UTC+9", 9*60*60)),
		ExternalID:       "req_123",
		ProjectName:      "budget-app",
		Model:            "anthropic/claude-sonnet-4",
		ModelPermaslug:   "anthropic/claude-sonnet-4-20250514",
		PromptTokens:     1_000_000,
		CompletionTokens: 500_000,
		CacheReadTokens:  250_000,
		ToolInvocations:  2,
		Price: ports.ModelPrice{
			Provider:             domain.ProviderOpenRouter,
			ModelID:              "anthropic/claude-sonnet-4-20250514",
			LookupKey:            "anthropic/claude-sonnet-4",
			InputUSDPer1M:        3,
			OutputUSDPer1M:       15,
			CacheReadUSDPer1M:    0.3,
			ToolUSDPerInvocation: 0.01,
		},
		UsageUSD:     10.095,
		BYOKUsageUSD: 0.5,
	})
	if err != nil {
		t.Fatalf("NormalizeUsageImport() error = %v", err)
	}

	if entry.Provider != domain.ProviderOpenRouter || entry.Source != domain.UsageSourceOpenRouter {
		t.Fatalf("entry = %+v, want OpenRouter normalized usage entry", entry)
	}
	if !entry.OccurredAt.Equal(time.Date(2026, 4, 17, 1, 30, 0, 0, time.UTC)) {
		t.Fatalf("entry.OccurredAt = %s, want normalized UTC timestamp", entry.OccurredAt)
	}
	if entry.CostBreakdown.InputUSD != 3 || entry.CostBreakdown.OutputUSD != 7.5 || entry.CostBreakdown.CacheReadUSD != 0.075 || entry.CostBreakdown.ToolUSD != 0.02 {
		t.Fatalf("entry.CostBreakdown = %+v, want request pricing fields preserved", entry.CostBreakdown)
	}
	if entry.CostBreakdown.FlatUSD != 0 || math.Abs(entry.CostBreakdown.TotalUSD-10.595) > 1e-9 {
		t.Fatalf("entry.CostBreakdown = %+v, want actual total cost preserved", entry.CostBreakdown)
	}
	if entry.ExternalID != "req_123" || entry.ProjectName != "budget-app" {
		t.Fatalf("entry identifiers = %+v, want request/project normalization", entry)
	}
	if entry.PricingRef == nil || entry.PricingRef.ModelID != "anthropic/claude-sonnet-4-20250514" || entry.PricingRef.PricingLookupKey != "anthropic/claude-sonnet-4" {
		t.Fatalf("entry.PricingRef = %+v, want pricing reference normalized", entry.PricingRef)
	}
}
