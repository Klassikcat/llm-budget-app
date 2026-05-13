package gui

import (
	"context"
	"math"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/service"
)

func TestGraphsBindingLoadGraphs(t *testing.T) {
	period := mustMonthlyPeriod(t, 2026, time.April)
	store := mustDashboardStore(t)
	defer store.Close()

	pricingRef := mustPricingRef(t, domain.ProviderOpenAI, "gpt-4.1")
	if err := store.UpsertUsageEntries(context.Background(), []domain.UsageEntry{
		mustUsageEntry(t, domain.UsageEntry{
			EntryID:       "usage-graph-1",
			Source:        domain.UsageSourceManualAPI,
			Provider:      domain.ProviderOpenAI,
			BillingMode:   domain.BillingModeDirectAPI,
			OccurredAt:    period.StartAt.Add(2 * time.Hour),
			PricingRef:    &pricingRef,
			Tokens:        mustTokenUsage(t, 100, 50, 10, 5),
			CostBreakdown: mustCostBreakdown(t, 1, 2, 0.1, 0.2, 0, 0),
		}),
	}); err != nil {
		t.Fatalf("UpsertUsageEntries() error = %v", err)
	}

	binding := NewGraphsBinding(service.NewGraphQueryService(store))
	binding.startup(context.Background())
	response, err := binding.LoadGraphs("2026-04", "All")
	if err != nil {
		t.Fatalf("LoadGraphs() error = %v", err)
	}

	if got := len(response.ModelTokenUsages); got != 1 {
		t.Fatalf("len(ModelTokenUsages) = %d, want 1", got)
	}
	if got := response.ModelTokenUsages[0]; got.ModelName != "gpt-4.1" || got.TotalTokens != 165 || got.CacheWriteTokens != 5 {
		t.Fatalf("ModelTokenUsages[0] = %+v, want gpt-4.1 with 165 total tokens", got)
	}
	if got := response.ModelCosts[0]; got.ModelName != "gpt-4.1" || math.Abs(got.TotalCostUSD-3.3) > 0.000001 {
		t.Fatalf("ModelCosts[0] = %+v, want gpt-4.1 cost 3.3", got)
	}
	if len(response.DailyTokenTrends) == 0 || response.DailyTokenTrends[0].Date == "" {
		t.Fatalf("DailyTokenTrends = %+v, want dated trend points", response.DailyTokenTrends)
	}
	if got := response.ModelTokenBreakdowns[0]; got.InputTokens != 100 || got.OutputTokens != 50 {
		t.Fatalf("ModelTokenBreakdowns[0] = %+v, want input 100 output 50", got)
	}
}

func TestGraphsBindingLoadGraphsUsesAdaptiveTimeRange(t *testing.T) {
	store := mustDashboardStore(t)
	defer store.Close()

	pricingRef := mustPricingRef(t, domain.ProviderOpenAI, "gpt-4.1")
	if err := store.UpsertUsageEntries(context.Background(), []domain.UsageEntry{
		mustUsageEntry(t, domain.UsageEntry{
			EntryID:       "usage-graph-in-range",
			Source:        domain.UsageSourceManualAPI,
			Provider:      domain.ProviderOpenAI,
			BillingMode:   domain.BillingModeDirectAPI,
			OccurredAt:    time.Date(2026, time.April, 9, 12, 0, 0, 0, time.UTC),
			PricingRef:    &pricingRef,
			Tokens:        mustTokenUsage(t, 100, 0, 0, 0),
			CostBreakdown: mustCostBreakdown(t, 1, 0, 0, 0, 0, 0),
		}),
		mustUsageEntry(t, domain.UsageEntry{
			EntryID:       "usage-graph-out-of-range",
			Source:        domain.UsageSourceManualAPI,
			Provider:      domain.ProviderOpenAI,
			BillingMode:   domain.BillingModeDirectAPI,
			OccurredAt:    time.Date(2026, time.March, 20, 12, 0, 0, 0, time.UTC),
			PricingRef:    &pricingRef,
			Tokens:        mustTokenUsage(t, 900, 0, 0, 0),
			CostBreakdown: mustCostBreakdown(t, 9, 0, 0, 0, 0, 0),
		}),
	}); err != nil {
		t.Fatalf("UpsertUsageEntries() error = %v", err)
	}

	binding := NewGraphsBinding(service.NewGraphQueryService(store))
	binding.clock = func() time.Time { return time.Date(2026, time.April, 10, 15, 0, 0, 0, time.UTC) }

	response, err := binding.LoadGraphs("", "7 days")
	if err != nil {
		t.Fatalf("LoadGraphs() error = %v", err)
	}

	if got, want := response.ModelTokenUsages[0].TotalTokens, int64(100); got != want {
		t.Fatalf("ModelTokenUsages[0].TotalTokens = %d, want %d", got, want)
	}
	if got, want := len(response.DailyTokenTrends), 7; got != want {
		t.Fatalf("len(DailyTokenTrends) = %d, want %d", got, want)
	}
}

func TestGraphsBindingRejectsInvalidMonth(t *testing.T) {
	binding := NewGraphsBinding(service.NewGraphQueryService(mustDashboardStore(t)))
	if _, err := binding.LoadGraphs("April 2026", "All"); err == nil {
		t.Fatal("LoadGraphs() error = nil, want invalid month error")
	}
}

func TestGraphsBindingRejectsInvalidTimeRange(t *testing.T) {
	binding := NewGraphsBinding(service.NewGraphQueryService(mustDashboardStore(t)))
	if _, err := binding.LoadGraphs("", "Last quarter"); err == nil {
		t.Fatal("LoadGraphs() error = nil, want invalid time range error")
	}
}
