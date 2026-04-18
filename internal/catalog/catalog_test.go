package catalog

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestEmbeddedCatalogsLoadSuccessfully(t *testing.T) {
	t.Parallel()

	loaded, err := New(Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ref, err := domain.NewModelPricingRef(domain.ProviderOpenAI, "gpt-4.1", "openai/gpt-4.1")
	if err != nil {
		t.Fatalf("NewModelPricingRef() error = %v", err)
	}

	price, err := loaded.LookupModelPrice(context.Background(), ref, time.Now())
	if err != nil {
		t.Fatalf("LookupModelPrice() error = %v", err)
	}

	if price.InputUSDPer1M <= 0 || price.OutputUSDPer1M <= 0 {
		t.Fatalf("LookupModelPrice() = %+v, want required prices loaded", price)
	}
	if len(loaded.Warnings()) != 0 {
		t.Fatalf("Warnings() = %v, want none", loaded.Warnings())
	}
	if got := loaded.CacheSnapshot(); got.Source != openRouterCacheSource || got.Version == "" {
		t.Fatalf("CacheSnapshot() = %+v, want placeholder openrouter cache metadata", got)
	}
}

func TestOverridePrecedence(t *testing.T) {
	t.Parallel()

	overridePath := filepath.Join("testdata", "prices.override.yaml")
	loaded, err := New(Options{OverridePath: overridePath})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ref, err := domain.NewModelPricingRef(domain.ProviderOpenAI, "gpt-4.1", "openai/gpt-4.1")
	if err != nil {
		t.Fatalf("NewModelPricingRef() error = %v", err)
	}

	price, err := loaded.LookupModelPrice(context.Background(), ref, time.Now())
	if err != nil {
		t.Fatalf("LookupModelPrice() error = %v", err)
	}

	if price.InputUSDPer1M != 9.99 || price.OutputUSDPer1M != 19.99 {
		t.Fatalf("LookupModelPrice() = %+v, want override values", price)
	}
	if len(loaded.Warnings()) != 0 {
		t.Fatalf("Warnings() = %v, want none", loaded.Warnings())
	}
}

func TestMalformedOverrideFallsBackCleanly(t *testing.T) {
	t.Parallel()

	overridePath := filepath.Join("testdata", "prices.malformed.yaml")
	loaded, err := New(Options{OverridePath: overridePath})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if len(loaded.Warnings()) != 1 {
		t.Fatalf("Warnings() len = %d, want 1", len(loaded.Warnings()))
	}

	ref, err := domain.NewModelPricingRef(domain.ProviderOpenAI, "gpt-4.1", "openai/gpt-4.1")
	if err != nil {
		t.Fatalf("NewModelPricingRef() error = %v", err)
	}

	price, err := loaded.LookupModelPrice(context.Background(), ref, time.Now())
	if err != nil {
		t.Fatalf("LookupModelPrice() error = %v", err)
	}

	if price.InputUSDPer1M != 2.0 || price.OutputUSDPer1M != 8.0 {
		t.Fatalf("LookupModelPrice() = %+v, want embedded fallback", price)
	}
}

func TestReplaceCatalogUsesOpenRouterCacheWhereApplicable(t *testing.T) {
	t.Parallel()

	loaded, err := New(Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	syncedAt := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	err = loaded.ReplaceCatalog(context.Background(), ports.CatalogSnapshot{
		Source:   openRouterCacheSource,
		Version:  "sync-2026-04-17T12:00:00Z",
		SyncedAt: syncedAt,
		Entries: []ports.ModelPrice{{
			Provider:       domain.ProviderOpenRouter,
			ModelID:        "openrouter/anthropic/claude-sonnet-4",
			LookupKey:      "openrouter/anthropic/claude-sonnet-4",
			InputUSDPer1M:  3.5,
			OutputUSDPer1M: 17.5,
		}},
	})
	if err != nil {
		t.Fatalf("ReplaceCatalog() error = %v", err)
	}

	ref, err := domain.NewModelPricingRef(domain.ProviderOpenRouter, "openrouter/anthropic/claude-sonnet-4", "openrouter/anthropic/claude-sonnet-4")
	if err != nil {
		t.Fatalf("NewModelPricingRef() error = %v", err)
	}

	price, err := loaded.LookupModelPrice(context.Background(), ref, time.Now())
	if err != nil {
		t.Fatalf("LookupModelPrice() error = %v", err)
	}

	if price.InputUSDPer1M != 3.5 || price.OutputUSDPer1M != 17.5 {
		t.Fatalf("LookupModelPrice() = %+v, want cached OpenRouter price", price)
	}

	snapshot := loaded.CacheSnapshot()
	if snapshot.Source != openRouterCacheSource || snapshot.Version != "sync-2026-04-17T12:00:00Z" || !snapshot.SyncedAt.Equal(syncedAt) {
		t.Fatalf("CacheSnapshot() = %+v, want updated cache metadata", snapshot)
	}
}

func TestReplaceCatalogValidatesRequiredPricingAttributes(t *testing.T) {
	t.Parallel()

	loaded, err := New(Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = loaded.ReplaceCatalog(context.Background(), ports.CatalogSnapshot{
		Source:  openRouterCacheSource,
		Version: "sync-invalid",
		Entries: []ports.ModelPrice{{
			Provider:       domain.ProviderOpenRouter,
			ModelID:        "broken-model",
			LookupKey:      "broken-model",
			InputUSDPer1M:  1,
			OutputUSDPer1M: -1,
		}},
	})
	if err == nil {
		t.Fatal("ReplaceCatalog() error = nil, want validation failure")
	}
	if !strings.Contains(err.Error(), "must be non-negative") {
		t.Fatalf("ReplaceCatalog() error = %v, want pricing validation", err)
	}
	if errors.Is(err, errModelPriceNotFound) {
		t.Fatalf("ReplaceCatalog() error = %v, want validation error not lookup error", err)
	}
}

func TestListProviderPricesReturnsDeduplicatedProviderModels(t *testing.T) {
	t.Parallel()

	loaded, err := New(Options{OverridePath: filepath.Join("testdata", "prices.override.yaml")})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	prices, err := loaded.ListProviderPrices(context.Background(), domain.ProviderOpenAI)
	if err != nil {
		t.Fatalf("ListProviderPrices() error = %v", err)
	}

	if len(prices) != 3 {
		t.Fatalf("len(prices) = %d, want 3 deduplicated OpenAI models", len(prices))
	}

	byModel := make(map[string]ports.ModelPrice, len(prices))
	for _, price := range prices {
		byModel[price.ModelID] = price
	}

	if got := byModel["gpt-4.1"].InputUSDPer1M; got != 9.99 {
		t.Fatalf("override model input price = %v, want 9.99", got)
	}
	if got := byModel["gpt-4.1-mini"].InputUSDPer1M; got != 0.4 {
		t.Fatalf("embedded model input price = %v, want 0.4", got)
	}
	if got := byModel["gpt-4.1-nano"].InputUSDPer1M; got != 0.1 {
		t.Fatalf("embedded nano input price = %v, want 0.1", got)
	}

	openRouterPrices, err := loaded.ListProviderPrices(context.Background(), domain.ProviderOpenRouter)
	if err != nil {
		t.Fatalf("ListProviderPrices(openrouter) error = %v", err)
	}
	if len(openRouterPrices) != 0 {
		t.Fatalf("len(openRouterPrices) = %d, want 0 before cache sync", len(openRouterPrices))
	}
}
