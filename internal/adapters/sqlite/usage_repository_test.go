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

func TestUsageEntryRepositoryRoundTripACPMetadata(t *testing.T) {
	store := mustBootstrapStore(t, filepath.Join(t.TempDir(), "usage-repo-acp.sqlite3"), Options{})
	defer store.Close()

	tokens, err := domain.NewTokenUsage(800, 200, 100, 25)
	if err != nil {
		t.Fatalf("NewTokenUsage() error = %v", err)
	}
	costs, err := domain.NewCostBreakdown(0.0024, 0.003, 0.00003, 0.00009375, 0, 0)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}

	entry, err := domain.NewUsageEntry(domain.UsageEntry{
		EntryID:       "claude:acp:metadata-roundtrip",
		Source:        domain.UsageSourceCLISession,
		Provider:      domain.ProviderAnthropic,
		BillingMode:   domain.BillingModeSubscription,
		OccurredAt:    time.Date(2026, 4, 18, 9, 15, 0, 0, time.UTC),
		ProjectName:   "acp-project",
		Metadata:      map[string]string{"claude_session_type": "acp"},
		Tokens:        tokens,
		CostBreakdown: costs,
	})
	if err != nil {
		t.Fatalf("NewUsageEntry() error = %v", err)
	}

	if err := store.UpsertUsageEntries(context.Background(), []domain.UsageEntry{entry}); err != nil {
		t.Fatalf("UpsertUsageEntries() error = %v", err)
	}

	entries, err := store.ListUsageEntries(context.Background(), ports.UsageFilter{Provider: domain.ProviderAnthropic, Project: "acp-project"})
	if err != nil {
		t.Fatalf("ListUsageEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListUsageEntries() len = %d, want 1", len(entries))
	}
	if got := entries[0].Metadata["claude_session_type"]; got != "acp" {
		t.Fatalf("Metadata[claude_session_type] = %q, want acp; metadata=%#v", got, entries[0].Metadata)
	}
}

func TestUsageEntryRepositoryOpenRouterActivityResyncIsIdempotentByEntryIDUpsert(t *testing.T) {
	store := mustBootstrapStore(t, filepath.Join(t.TempDir(), "usage-repo-openrouter-resync.sqlite3"), Options{})
	defer store.Close()

	entry := mustTask13SQLiteOpenRouterActivityEntry(t)
	session, err := domain.NewSessionSummary(domain.SessionSummary{
		SessionID:     entry.SessionID,
		Source:        entry.Source,
		Provider:      entry.Provider,
		BillingMode:   entry.BillingMode,
		StartedAt:     entry.OccurredAt,
		EndedAt:       entry.OccurredAt,
		ProjectName:   "openrouter-activity",
		AgentName:     "openrouter",
		PricingRef:    entry.PricingRef,
		Tokens:        entry.Tokens,
		CostBreakdown: entry.CostBreakdown,
	})
	if err != nil {
		t.Fatalf("NewSessionSummary() error = %v", err)
	}
	if err := store.UpsertSessions(context.Background(), []domain.SessionSummary{session}); err != nil {
		t.Fatalf("UpsertSessions() error = %v", err)
	}

	if err := store.UpsertUsageEntries(context.Background(), []domain.UsageEntry{entry}); err != nil {
		t.Fatalf("first UpsertUsageEntries() error = %v", err)
	}
	if err := store.UpsertUsageEntries(context.Background(), []domain.UsageEntry{entry}); err != nil {
		t.Fatalf("second UpsertUsageEntries() error = %v", err)
	}

	period, err := domain.NewMonthlyPeriod(entry.OccurredAt)
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	entries, err := store.ListUsageEntries(context.Background(), ports.UsageFilter{Period: &period, Provider: domain.ProviderOpenRouter})
	if err != nil {
		t.Fatalf("ListUsageEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListUsageEntries() len = %d, want 1 after duplicate OpenRouter activity sync", len(entries))
	}
	if got, want := entries[0].CostBreakdown.TotalUSD, entry.CostBreakdown.TotalUSD; got != want {
		t.Fatalf("CostBreakdown.TotalUSD = %v, want %v", got, want)
	}
}

func mustTask13SQLiteOpenRouterActivityEntry(t *testing.T) domain.UsageEntry {
	t.Helper()

	tokens, err := domain.NewTokenUsage(1200, 340, 0, 0)
	if err != nil {
		t.Fatalf("NewTokenUsage() error = %v", err)
	}
	costs, err := domain.NewCostBreakdown(2.0, 0, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}
	ref, err := domain.NewModelPricingRef(domain.ProviderOpenRouter, "openai/gpt-4.1-2025-04-14", "openai/gpt-4.1")
	if err != nil {
		t.Fatalf("NewModelPricingRef() error = %v", err)
	}
	entry, err := domain.NewUsageEntry(domain.UsageEntry{
		EntryID:       "openrouter-activity-task13-deterministic",
		Source:        domain.UsageSourceOpenRouter,
		Provider:      domain.ProviderOpenRouter,
		BillingMode:   domain.BillingModeOpenRouter,
		OccurredAt:    time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC),
		SessionID:     "openrouter:activity:2026-04-16:openai-gpt-4.1:endpoint_abc123",
		ExternalID:    "endpoint_abc123",
		PricingRef:    &ref,
		Tokens:        tokens,
		CostBreakdown: costs,
	})
	if err != nil {
		t.Fatalf("NewUsageEntry() error = %v", err)
	}
	return entry
}
