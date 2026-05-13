package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"llm-budget-tracker/internal/adapters/openrouter"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestOpenRouterActivitySyncServiceSync(t *testing.T) {
	t.Parallel()

	sourceErr := errors.New("source unavailable")
	ingestionErr := errors.New("ingestion unavailable")
	validEntry := mustOpenRouterUsageEntry(t, "openrouter-entry-1")
	invalidEntry := validEntry
	invalidEntry.BillingMode = domain.BillingModeDirectAPI

	tests := []struct {
		name                string
		newService          func(*stubOpenRouterActivitySource, *stubIngestionService) *OpenRouterActivitySyncService
		source              *stubOpenRouterActivitySource
		ingestion           *stubIngestionService
		options             ports.OpenRouterActivityOptions
		wantErr             error
		wantErrCode         domain.ValidationCode
		wantResultEntries   int
		wantIngestedEntries int
		wantSourceCalls     int
		wantIngestionCalls  int
	}{
		{
			name: "success forwards normalized OpenRouter entries to ingestion",
			source: &stubOpenRouterActivitySource{
				entries: []domain.UsageEntry{validEntry},
			},
			ingestion:           &stubIngestionService{},
			options:             ports.OpenRouterActivityOptions{APIKeyHash: "synthetic-key-hash", UserID: "user_42"},
			wantResultEntries:   1,
			wantIngestedEntries: 1,
			wantSourceCalls:     1,
			wantIngestionCalls:  1,
		},
		{
			name:               "empty response skips ingestion",
			source:             &stubOpenRouterActivitySource{},
			ingestion:          &stubIngestionService{},
			wantSourceCalls:    1,
			wantIngestionCalls: 0,
		},
		{
			name: "nil service reports missing source",
			newService: func(_ *stubOpenRouterActivitySource, _ *stubIngestionService) *OpenRouterActivitySyncService {
				return nil
			},
			wantErr: errOpenRouterActivitySourceRequired,
		},
		{
			name: "missing source reports dependency error",
			newService: func(_ *stubOpenRouterActivitySource, ingestion *stubIngestionService) *OpenRouterActivitySyncService {
				return NewOpenRouterActivitySyncService(nil, ingestion)
			},
			ingestion: &stubIngestionService{},
			wantErr:   errOpenRouterActivitySourceRequired,
		},
		{
			name: "missing ingestion reports dependency error",
			newService: func(source *stubOpenRouterActivitySource, _ *stubIngestionService) *OpenRouterActivitySyncService {
				return NewOpenRouterActivitySyncService(source, nil)
			},
			source:          &stubOpenRouterActivitySource{},
			wantErr:         errIngestionServiceRequired,
			wantSourceCalls: 0,
		},
		{
			name: "source error is returned without ingestion",
			source: &stubOpenRouterActivitySource{
				err: sourceErr,
			},
			ingestion:          &stubIngestionService{},
			wantErr:            sourceErr,
			wantSourceCalls:    1,
			wantIngestionCalls: 0,
		},
		{
			name: "ingestion error is returned",
			source: &stubOpenRouterActivitySource{
				entries: []domain.UsageEntry{validEntry},
			},
			ingestion:           &stubIngestionService{err: ingestionErr},
			wantErr:             ingestionErr,
			wantSourceCalls:     1,
			wantIngestionCalls:  1,
			wantIngestedEntries: 1,
		},
		{
			name: "invalid domain entry error from ingestion is returned",
			source: &stubOpenRouterActivitySource{
				entries: []domain.UsageEntry{invalidEntry},
			},
			ingestion:           &stubIngestionService{validateEntries: true},
			wantErrCode:         domain.ValidationCodeInvalidBillingMode,
			wantSourceCalls:     1,
			wantIngestionCalls:  1,
			wantIngestedEntries: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			newService := tt.newService
			if newService == nil {
				newService = func(source *stubOpenRouterActivitySource, ingestion *stubIngestionService) *OpenRouterActivitySyncService {
					return NewOpenRouterActivitySyncService(source, ingestion)
				}
			}
			svc := newService(tt.source, tt.ingestion)

			result, err := svc.Sync(context.Background(), tt.options)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Sync() error = %v, want %v", err, tt.wantErr)
				}
			} else if tt.wantErrCode != "" {
				if !domain.IsValidationCode(err, tt.wantErrCode) {
					t.Fatalf("Sync() error = %v, want validation code %q", err, tt.wantErrCode)
				}
			} else if err != nil {
				t.Fatalf("Sync() error = %v", err)
			}

			if len(result.UsageEntries) != tt.wantResultEntries {
				t.Fatalf("len(result.UsageEntries) = %d, want %d", len(result.UsageEntries), tt.wantResultEntries)
			}
			if tt.source != nil {
				if tt.source.called != tt.wantSourceCalls {
					t.Fatalf("source.called = %d, want %d", tt.source.called, tt.wantSourceCalls)
				}
				if tt.source.called > 0 && tt.source.options.APIKeyHash != tt.options.APIKeyHash {
					t.Fatalf("source.options.APIKeyHash = %q, want %q", tt.source.options.APIKeyHash, tt.options.APIKeyHash)
				}
			}
			if tt.ingestion != nil {
				if tt.ingestion.usageCalls != tt.wantIngestionCalls {
					t.Fatalf("ingestion.usageCalls = %d, want %d", tt.ingestion.usageCalls, tt.wantIngestionCalls)
				}
				if len(tt.ingestion.entries) != tt.wantIngestedEntries {
					t.Fatalf("len(ingestion.entries) = %d, want %d", len(tt.ingestion.entries), tt.wantIngestedEntries)
				}
			}
			if tt.wantResultEntries > 0 {
				entry := result.UsageEntries[0]
				if entry.Source != domain.UsageSourceOpenRouter || entry.BillingMode != domain.BillingModeOpenRouter {
					t.Fatalf("entry source/billing = %q/%q, want %q/%q", entry.Source, entry.BillingMode, domain.UsageSourceOpenRouter, domain.BillingModeOpenRouter)
				}
				if tt.ingestion.entries[0].Source != domain.UsageSourceOpenRouter || tt.ingestion.entries[0].BillingMode != domain.BillingModeOpenRouter {
					t.Fatalf("ingested source/billing = %q/%q, want %q/%q", tt.ingestion.entries[0].Source, tt.ingestion.entries[0].BillingMode, domain.UsageSourceOpenRouter, domain.BillingModeOpenRouter)
				}
			}
		})
	}
}

func TestOpenRouterActivitySyncServiceReturnsInvalidActivityDateError(t *testing.T) {
	t.Parallel()

	const syntheticAPIKey = "synthetic-openrouter-api-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/activity" {
			t.Fatalf("path = %q, want /api/v1/activity", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+syntheticAPIKey {
			t.Fatal("Authorization header did not contain synthetic bearer token")
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{
				"date":                 "04/16/2026",
				"model":                "openai/gpt-4.1",
				"model_permaslug":      "openai/gpt-4.1-2025-04-14",
				"endpoint_id":          "endpoint_invalid_date",
				"provider_name":        "OpenAI",
				"usage":                1.25,
				"byok_usage_inference": 0.75,
				"requests":             6,
				"prompt_tokens":        1200,
				"completion_tokens":    340,
			}},
		})
	}))
	defer server.Close()

	client := openrouter.NewClient(openrouter.Options{APIKey: syntheticAPIKey, APIBaseURL: server.URL + "/api/v1"})
	ingestion := &stubIngestionService{}
	svc := NewOpenRouterActivitySyncService(client, ingestion)

	_, err := svc.Sync(context.Background(), ports.OpenRouterActivityOptions{})
	if err == nil {
		t.Fatal("Sync() error = nil, want invalid activity date normalization error")
	}
	if !strings.Contains(err.Error(), "normalize OpenRouter activity 0") || !strings.Contains(err.Error(), "activity date must be YYYY-MM-DD") {
		t.Fatalf("Sync() error = %v, want invalid date normalization context", err)
	}
	if strings.Contains(err.Error(), syntheticAPIKey) {
		t.Fatal("Sync() error exposed synthetic API key")
	}
	if ingestion.usageCalls != 0 {
		t.Fatalf("ingestion.usageCalls = %d, want 0 after normalization error", ingestion.usageCalls)
	}
}

func TestOpenRouterActivitySyncServiceDuplicateEntryIDUpsertIsIdempotent(t *testing.T) {
	t.Parallel()

	entry := mustOpenRouterUsageEntry(t, "openrouter-duplicate-entry")
	source := &stubOpenRouterActivitySource{entries: []domain.UsageEntry{entry}}
	ingestion := newUpsertIngestionService()
	svc := NewOpenRouterActivitySyncService(source, ingestion)

	for i := 0; i < 2; i++ {
		result, err := svc.Sync(context.Background(), ports.OpenRouterActivityOptions{})
		if err != nil {
			t.Fatalf("Sync() run %d error = %v", i+1, err)
		}
		if len(result.UsageEntries) != 1 {
			t.Fatalf("Sync() run %d returned %d entries, want 1", i+1, len(result.UsageEntries))
		}
	}

	if source.called != 2 {
		t.Fatalf("source.called = %d, want 2", source.called)
	}
	if ingestion.usageCalls != 2 {
		t.Fatalf("ingestion.usageCalls = %d, want 2", ingestion.usageCalls)
	}
	if len(ingestion.entriesByID) != 1 {
		t.Fatalf("len(entriesByID) = %d, want 1 after duplicate entry ID upsert", len(ingestion.entriesByID))
	}
	if got := ingestion.entriesByID[entry.EntryID]; got.EntryID != entry.EntryID || got.CostBreakdown.TotalUSD != entry.CostBreakdown.TotalUSD {
		t.Fatalf("upserted entry = %+v, want original duplicate entry", got)
	}
}

func TestOpenRouterAutoSyncSkipFreshCheckpoint(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	checkpoints := newMemoryCheckpointRepository()
	if err := checkpoints.SaveCheckpoint(context.Background(), ports.IngestionCheckpoint{
		SourceID:  OpenRouterActivityAutoSyncCheckpointID,
		UpdatedAt: now.Add(-1 * time.Hour),
	}); err != nil {
		t.Fatalf("SaveCheckpoint() error = %v", err)
	}
	source := &stubOpenRouterActivitySource{entries: []domain.UsageEntry{mustOpenRouterUsageEntry(t, "fresh-skip-entry")}}
	ingestion := &stubIngestionService{}
	svc := NewOpenRouterActivitySyncService(source, ingestion)

	result, attempted, err := svc.AutoSync(context.Background(), checkpoints, now)
	if err != nil {
		t.Fatalf("AutoSync() error = %v", err)
	}
	if attempted {
		t.Fatal("AutoSync() attempted = true, want false for fresh checkpoint")
	}
	if source.called != 0 {
		t.Fatalf("source.called = %d, want 0", source.called)
	}
	if ingestion.usageCalls != 0 {
		t.Fatalf("ingestion.usageCalls = %d, want 0", ingestion.usageCalls)
	}
	if len(result.UsageEntries) != 0 {
		t.Fatalf("len(result.UsageEntries) = %d, want 0", len(result.UsageEntries))
	}
}

func TestOpenRouterAutoSyncRunStaleCheckpoint(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	checkpoints := newMemoryCheckpointRepository()
	if err := checkpoints.SaveCheckpoint(context.Background(), ports.IngestionCheckpoint{
		SourceID:  OpenRouterActivityAutoSyncCheckpointID,
		UpdatedAt: now.Add(-25 * time.Hour),
	}); err != nil {
		t.Fatalf("SaveCheckpoint() error = %v", err)
	}
	source := &stubOpenRouterActivitySource{entries: []domain.UsageEntry{mustOpenRouterUsageEntry(t, "stale-run-entry")}}
	ingestion := &stubIngestionService{}
	svc := NewOpenRouterActivitySyncService(source, ingestion)

	result, attempted, err := svc.AutoSync(context.Background(), checkpoints, now)
	if err != nil {
		t.Fatalf("AutoSync() error = %v", err)
	}
	if !attempted {
		t.Fatal("AutoSync() attempted = false, want true for stale checkpoint")
	}
	if source.called != 1 {
		t.Fatalf("source.called = %d, want 1", source.called)
	}
	if ingestion.usageCalls != 1 {
		t.Fatalf("ingestion.usageCalls = %d, want 1", ingestion.usageCalls)
	}
	if len(result.UsageEntries) != 1 {
		t.Fatalf("len(result.UsageEntries) = %d, want 1", len(result.UsageEntries))
	}
	checkpoint, err := checkpoints.LoadCheckpoint(context.Background(), OpenRouterActivityAutoSyncCheckpointID)
	if err != nil {
		t.Fatalf("LoadCheckpoint() error = %v", err)
	}
	if !checkpoint.UpdatedAt.Equal(now) {
		t.Fatalf("checkpoint.UpdatedAt = %v, want %v", checkpoint.UpdatedAt, now)
	}
}

func TestOpenRouterAutoSyncFailureDoesNotUpdateCheckpoint(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	stale := now.Add(-25 * time.Hour)
	checkpoints := newMemoryCheckpointRepository()
	if err := checkpoints.SaveCheckpoint(context.Background(), ports.IngestionCheckpoint{
		SourceID:  OpenRouterActivityAutoSyncCheckpointID,
		UpdatedAt: stale,
	}); err != nil {
		t.Fatalf("SaveCheckpoint() error = %v", err)
	}
	sourceErr := errors.New("synthetic OpenRouter activity failure")
	source := &stubOpenRouterActivitySource{err: sourceErr}
	ingestion := &stubIngestionService{}
	svc := NewOpenRouterActivitySyncService(source, ingestion)

	_, attempted, err := svc.AutoSync(context.Background(), checkpoints, now)
	if !errors.Is(err, sourceErr) {
		t.Fatalf("AutoSync() error = %v, want %v", err, sourceErr)
	}
	if !attempted {
		t.Fatal("AutoSync() attempted = false, want true for stale checkpoint failure")
	}
	if source.called != 1 {
		t.Fatalf("source.called = %d, want 1", source.called)
	}
	if ingestion.usageCalls != 0 {
		t.Fatalf("ingestion.usageCalls = %d, want 0", ingestion.usageCalls)
	}
	checkpoint, err := checkpoints.LoadCheckpoint(context.Background(), OpenRouterActivityAutoSyncCheckpointID)
	if err != nil {
		t.Fatalf("LoadCheckpoint() error = %v", err)
	}
	if !checkpoint.UpdatedAt.Equal(stale) {
		t.Fatalf("checkpoint.UpdatedAt = %v, want stale value %v", checkpoint.UpdatedAt, stale)
	}
}

func mustOpenRouterUsageEntry(t *testing.T, entryID string) domain.UsageEntry {
	t.Helper()

	tokens, err := domain.NewTokenUsage(10, 5, 0, 0)
	if err != nil {
		t.Fatalf("NewTokenUsage() error = %v", err)
	}
	breakdown, err := domain.NewCostBreakdown(0.1, 0.2, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}
	ref, err := domain.NewModelPricingRef(domain.ProviderOpenRouter, "openai/gpt-4.1-2025-04-14", "openai/gpt-4.1")
	if err != nil {
		t.Fatalf("NewModelPricingRef() error = %v", err)
	}
	entry, err := domain.NewUsageEntry(domain.UsageEntry{
		EntryID:       entryID,
		Source:        domain.UsageSourceOpenRouter,
		Provider:      domain.ProviderOpenRouter,
		BillingMode:   domain.BillingModeOpenRouter,
		OccurredAt:    time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC),
		SessionID:     "openrouter:activity:2026-04-17:openai-gpt-4.1:endpoint_1",
		ExternalID:    "endpoint_1",
		PricingRef:    &ref,
		Tokens:        tokens,
		CostBreakdown: breakdown,
	})
	if err != nil {
		t.Fatalf("NewUsageEntry() error = %v", err)
	}

	return entry
}

type stubOpenRouterActivitySource struct {
	entries []domain.UsageEntry
	err     error
	called  int
	options ports.OpenRouterActivityOptions
}

func (s *stubOpenRouterActivitySource) FetchUsageEntries(_ context.Context, options ports.OpenRouterActivityOptions) ([]domain.UsageEntry, error) {
	s.called++
	s.options = options
	return s.entries, s.err
}

type stubIngestionService struct {
	entries         []domain.UsageEntry
	err             error
	usageCalls      int
	validateEntries bool
}

func (s *stubIngestionService) IngestUsageEntries(_ context.Context, entries []domain.UsageEntry) error {
	s.usageCalls++
	s.entries = append(s.entries, entries...)
	if s.err != nil {
		return s.err
	}
	if s.validateEntries {
		for _, entry := range entries {
			if _, err := domain.NewUsageEntry(entry); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *stubIngestionService) IngestSubscriptionFees(_ context.Context, _ []domain.SubscriptionFee) error {
	return nil
}

func (s *stubIngestionService) IngestSessionEvents(_ context.Context, _ []ports.SessionEvent) error {
	return nil
}

type upsertIngestionService struct {
	entriesByID map[string]domain.UsageEntry
	usageCalls  int
}

func newUpsertIngestionService() *upsertIngestionService {
	return &upsertIngestionService{entriesByID: make(map[string]domain.UsageEntry)}
}

func (s *upsertIngestionService) IngestUsageEntries(_ context.Context, entries []domain.UsageEntry) error {
	s.usageCalls++
	for _, entry := range entries {
		s.entriesByID[entry.EntryID] = entry
	}
	return nil
}

func (s *upsertIngestionService) IngestSubscriptionFees(_ context.Context, _ []domain.SubscriptionFee) error {
	return nil
}

func (s *upsertIngestionService) IngestSessionEvents(_ context.Context, _ []ports.SessionEvent) error {
	return nil
}

type memoryCheckpointRepository struct {
	checkpoints map[string]ports.IngestionCheckpoint
}

func newMemoryCheckpointRepository() *memoryCheckpointRepository {
	return &memoryCheckpointRepository{checkpoints: make(map[string]ports.IngestionCheckpoint)}
}

func (r *memoryCheckpointRepository) LoadCheckpoint(_ context.Context, sourceID string) (ports.IngestionCheckpoint, error) {
	checkpoint, ok := r.checkpoints[sourceID]
	if !ok {
		return ports.IngestionCheckpoint{SourceID: sourceID}, nil
	}
	return checkpoint, nil
}

func (r *memoryCheckpointRepository) SaveCheckpoint(_ context.Context, checkpoint ports.IngestionCheckpoint) error {
	r.checkpoints[checkpoint.SourceID] = checkpoint
	return nil
}
