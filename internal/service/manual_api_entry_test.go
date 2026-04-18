package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"llm-budget-tracker/internal/adapters/sqlite"
	catalogpkg "llm-budget-tracker/internal/catalog"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestManualAPIUsageEntryServiceSave(t *testing.T) {
	t.Parallel()

	catalog, err := catalogpkg.New(catalogpkg.Options{})
	if err != nil {
		t.Fatalf("catalog.New() error = %v", err)
	}

	store := mustBootstrapManualUsageStore(t)
	defer store.Close()

	svc := NewManualAPIUsageEntryService(catalog, store)
	occurredAt := time.Date(2026, 4, 17, 15, 45, 0, 0, time.FixedZone("KST", 9*60*60))

	entry, err := svc.Save(context.Background(), ManualAPIUsageEntryCommand{
		Provider:     "openai",
		ModelID:      "gpt-4.1",
		OccurredAt:   occurredAt,
		InputTokens:  1500,
		OutputTokens: 250,
		CachedTokens: 500,
		ProjectName:  "llm-budget-tracker",
		Metadata: map[string]string{
			"environment": "production",
			"source":      "invoice-import",
		},
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if entry.Source != domain.UsageSourceManualAPI {
		t.Fatalf("entry.Source = %q, want %q", entry.Source, domain.UsageSourceManualAPI)
	}
	if entry.BillingMode != domain.BillingModeDirectAPI {
		t.Fatalf("entry.BillingMode = %q, want %q", entry.BillingMode, domain.BillingModeDirectAPI)
	}
	if entry.OccurredAt.Location() != time.UTC {
		t.Fatalf("entry.OccurredAt location = %v, want UTC", entry.OccurredAt.Location())
	}
	if entry.CostBreakdown.TotalUSD != 0.00525 {
		t.Fatalf("entry.CostBreakdown.TotalUSD = %v, want 0.00525", entry.CostBreakdown.TotalUSD)
	}
	if entry.Metadata["environment"] != "production" {
		t.Fatalf("entry.Metadata = %#v, want trimmed metadata", entry.Metadata)
	}

	entries, err := store.ListUsageEntries(context.Background(), ports.UsageFilter{Project: "llm-budget-tracker"})
	if err != nil {
		t.Fatalf("ListUsageEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListUsageEntries() len = %d, want 1", len(entries))
	}

	persisted := entries[0]
	if persisted.EntryID != entry.EntryID {
		t.Fatalf("persisted.EntryID = %q, want %q", persisted.EntryID, entry.EntryID)
	}
	if persisted.ProjectName != "llm-budget-tracker" {
		t.Fatalf("persisted.ProjectName = %q, want llm-budget-tracker", persisted.ProjectName)
	}
	if persisted.Metadata["source"] != "invoice-import" {
		t.Fatalf("persisted.Metadata = %#v, want metadata round-trip", persisted.Metadata)
	}
	if persisted.CostBreakdown.TotalUSD != entry.CostBreakdown.TotalUSD {
		t.Fatalf("persisted total = %v, want %v", persisted.CostBreakdown.TotalUSD, entry.CostBreakdown.TotalUSD)
	}

	period, err := domain.NewMonthlyPeriod(occurredAt)
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	periodEntries, err := store.ListUsageEntries(context.Background(), ports.UsageFilter{Period: &period, Provider: domain.ProviderOpenAI})
	if err != nil {
		t.Fatalf("ListUsageEntries(period) error = %v", err)
	}
	if len(periodEntries) != 1 {
		t.Fatalf("ListUsageEntries(period) len = %d, want 1", len(periodEntries))
	}
}

func TestManualEntryValidation(t *testing.T) {
	t.Parallel()

	catalog, err := catalogpkg.New(catalogpkg.Options{})
	if err != nil {
		t.Fatalf("catalog.New() error = %v", err)
	}

	store := mustBootstrapManualUsageStore(t)
	defer store.Close()

	svc := NewManualAPIUsageEntryService(catalog, store)
	base := ManualAPIUsageEntryCommand{
		Provider:     "anthropic",
		ModelID:      "claude-sonnet-4-0",
		OccurredAt:   time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC),
		InputTokens:  1000,
		OutputTokens: 200,
	}

	tests := []struct {
		name     string
		mutate   func(*ManualAPIUsageEntryCommand)
		wantCode domain.ValidationCode
	}{
		{
			name: "rejects negative tokens",
			mutate: func(cmd *ManualAPIUsageEntryCommand) {
				cmd.InputTokens = -1
			},
			wantCode: domain.ValidationCodeNegativeTokens,
		},
		{
			name: "rejects unknown models",
			mutate: func(cmd *ManualAPIUsageEntryCommand) {
				cmd.ModelID = "claude-mystery-9"
			},
			wantCode: domain.ValidationCodeUnknownModel,
		},
		{
			name: "rejects unsupported providers",
			mutate: func(cmd *ManualAPIUsageEntryCommand) {
				cmd.Provider = "openrouter"
			},
			wantCode: domain.ValidationCodeUnsupportedProvider,
		},
		{
			name: "rejects malformed metadata",
			mutate: func(cmd *ManualAPIUsageEntryCommand) {
				cmd.Metadata = map[string]string{"invoice": "   "}
			},
			wantCode: domain.ValidationCodeInvalidMetadata,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := base
			tt.mutate(&cmd)

			_, err := svc.Save(context.Background(), cmd)
			if err == nil {
				t.Fatal("Save() error = nil, want validation error")
			}
			if !domain.IsValidationCode(err, tt.wantCode) {
				t.Fatalf("Save() error = %v, want validation code %q", err, tt.wantCode)
			}

			entries, listErr := store.ListUsageEntries(context.Background(), ports.UsageFilter{})
			if listErr != nil {
				t.Fatalf("ListUsageEntries() error = %v", listErr)
			}
			if len(entries) != 0 {
				t.Fatalf("ListUsageEntries() len = %d, want 0 after rejected save", len(entries))
			}
		})
	}
}

func TestManualAPIUsageEntryServiceSaveUsesExplicitEntryIDForUpsert(t *testing.T) {
	t.Parallel()

	catalog, err := catalogpkg.New(catalogpkg.Options{})
	if err != nil {
		t.Fatalf("catalog.New() error = %v", err)
	}

	store := mustBootstrapManualUsageStore(t)
	defer store.Close()

	svc := NewManualAPIUsageEntryService(catalog, store)
	cmd := ManualAPIUsageEntryCommand{
		EntryID:      "manual-explicit-id",
		Provider:     "openai",
		ModelID:      "gpt-4.1",
		OccurredAt:   time.Date(2026, 4, 18, 9, 0, 0, 0, time.UTC),
		InputTokens:  100,
		OutputTokens: 20,
	}

	entry, err := svc.Save(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if entry.EntryID != cmd.EntryID {
		t.Fatalf("entry.EntryID = %q, want %q", entry.EntryID, cmd.EntryID)
	}

	cmd.OutputTokens = 40
	updated, err := svc.Save(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Save(update) error = %v", err)
	}
	if updated.EntryID != cmd.EntryID {
		t.Fatalf("updated.EntryID = %q, want %q", updated.EntryID, cmd.EntryID)
	}

	entries, err := store.ListUsageEntries(context.Background(), ports.UsageFilter{})
	if err != nil {
		t.Fatalf("ListUsageEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListUsageEntries() len = %d, want 1 after upsert", len(entries))
	}
	if entries[0].Tokens.OutputTokens != 40 {
		t.Fatalf("entries[0].Tokens.OutputTokens = %d, want 40", entries[0].Tokens.OutputTokens)
	}
}

func mustBootstrapManualUsageStore(t *testing.T) *sqlite.Store {
	t.Helper()

	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: filepath.Join(t.TempDir(), "manual-usage.sqlite3")})
	if err != nil {
		t.Fatalf("sqlite.Bootstrap() error = %v", err)
	}

	return store
}
