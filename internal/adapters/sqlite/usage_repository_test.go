package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestUsageEntryRepositoryRoundTrip(t *testing.T) {
	store := mustBootstrapStore(t, filepath.Join(t.TempDir(), "usage-repo.sqlite3"), Options{})
	defer store.Close()

	pricingRef, err := domain.NewModelPricingRef(domain.ProviderAnthropic, "claude-sonnet-4-0", "anthropic/claude-sonnet-4-0")
	if err != nil {
		t.Fatalf("NewModelPricingRef() error = %v", err)
	}
	tokens, err := domain.NewTokenUsage(1200, 400, 300, 50)
	if err != nil {
		t.Fatalf("NewTokenUsage() error = %v", err)
	}
	costs, err := domain.NewCostBreakdown(0.0036, 0.006, 0.00009, 0.0001875, 0, 0)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}

	entry, err := domain.NewUsageEntry(domain.UsageEntry{
		EntryID:       "manual:anthropic:test",
		Source:        domain.UsageSourceManualAPI,
		Provider:      domain.ProviderAnthropic,
		BillingMode:   domain.BillingModeDirectAPI,
		OccurredAt:    time.Date(2026, 4, 17, 8, 30, 0, 0, time.UTC),
		ProjectName:   "alpha-project",
		Metadata:      map[string]string{"invoice_id": "inv-123"},
		PricingRef:    &pricingRef,
		Tokens:        tokens,
		CostBreakdown: costs,
	})
	if err != nil {
		t.Fatalf("NewUsageEntry() error = %v", err)
	}

	if err := store.UpsertUsageEntries(context.Background(), []domain.UsageEntry{entry}); err != nil {
		t.Fatalf("UpsertUsageEntries() error = %v", err)
	}

	entries, err := store.ListUsageEntries(context.Background(), ports.UsageFilter{Provider: domain.ProviderAnthropic, Project: "alpha-project"})
	if err != nil {
		t.Fatalf("ListUsageEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListUsageEntries() len = %d, want 1", len(entries))
	}

	got := entries[0]
	if got.EntryID != entry.EntryID {
		t.Fatalf("got.EntryID = %q, want %q", got.EntryID, entry.EntryID)
	}
	if got.Metadata["invoice_id"] != "inv-123" {
		t.Fatalf("got.Metadata = %#v, want invoice metadata", got.Metadata)
	}
	if got.CostBreakdown.CacheWriteUSD != entry.CostBreakdown.CacheWriteUSD {
		t.Fatalf("got.CacheWriteUSD = %v, want %v", got.CostBreakdown.CacheWriteUSD, entry.CostBreakdown.CacheWriteUSD)
	}

	period, err := domain.NewMonthlyPeriod(entry.OccurredAt)
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	filtered, err := store.ListUsageEntries(context.Background(), ports.UsageFilter{Period: &period, Project: "alpha-project"})
	if err != nil {
		t.Fatalf("ListUsageEntries(filtered) error = %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("ListUsageEntries(filtered) len = %d, want 1", len(filtered))
	}
}
