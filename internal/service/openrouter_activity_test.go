package service

import (
	"context"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestOpenRouterActivitySyncServiceSyncIngestsNormalizedEntries(t *testing.T) {
	t.Parallel()

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
		EntryID:       "openrouter-entry-1",
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

	source := &stubOpenRouterActivitySource{entries: []domain.UsageEntry{entry}}
	ingestion := &stubIngestionService{}
	service := NewOpenRouterActivitySyncService(source, ingestion)

	result, err := service.Sync(context.Background(), ports.OpenRouterActivityOptions{APIKeyHash: "hash123"})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if source.called != 1 {
		t.Fatalf("source.called = %d, want 1", source.called)
	}
	if len(result.UsageEntries) != 1 || len(ingestion.entries) != 1 {
		t.Fatalf("result/ingestion entries = %d/%d, want 1/1", len(result.UsageEntries), len(ingestion.entries))
	}
	if ingestion.entries[0].EntryID != "openrouter-entry-1" {
		t.Fatalf("ingested entry = %+v, want normalized usage entry forwarded", ingestion.entries[0])
	}
	if source.options.APIKeyHash != "hash123" {
		t.Fatalf("source.options = %+v, want options forwarded", source.options)
	}
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
	entries []domain.UsageEntry
}

func (s *stubIngestionService) IngestUsageEntries(_ context.Context, entries []domain.UsageEntry) error {
	s.entries = append(s.entries, entries...)
	return nil
}

func (s *stubIngestionService) IngestSubscriptionFees(_ context.Context, _ []domain.SubscriptionFee) error {
	return nil
}

func (s *stubIngestionService) IngestSessionEvents(_ context.Context, _ []ports.SessionEvent) error {
	return nil
}
