package gui

import (
	"context"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/service"
)

func TestInsightsBindingLoadWasteSummaryAndInsights(t *testing.T) {
	period := mustMonthlyPeriod(t, 2026, time.April)
	store := mustDashboardStore(t)
	defer store.Close()

	pricingRef := mustPricingRef(t, domain.ProviderOpenAI, "gpt-4.1")
	entry := mustUsageEntry(t, domain.UsageEntry{
		EntryID:       "usage-waste-1",
		Source:        domain.UsageSourceManualAPI,
		Provider:      domain.ProviderOpenAI,
		BillingMode:   domain.BillingModeDirectAPI,
		OccurredAt:    period.StartAt.Add(2 * time.Hour),
		PricingRef:    &pricingRef,
		Tokens:        mustTokenUsage(t, 100, 50, 0, 0),
		CostBreakdown: mustCostBreakdown(t, 1, 2, 0, 0, 0, 0),
	})
	if err := store.UpsertUsageEntries(context.Background(), []domain.UsageEntry{entry}); err != nil {
		t.Fatalf("UpsertUsageEntries() error = %v", err)
	}

	payload, err := domain.NewInsightPayload(nil, []string{entry.EntryID}, nil, []domain.InsightCount{{Key: "retry_count", Value: 2}}, []domain.InsightMetric{{Key: "waste_cost", Unit: domain.InsightMetricUnitUSD, Value: 3}})
	if err != nil {
		t.Fatalf("NewInsightPayload() error = %v", err)
	}
	insight := mustInsight(t, domain.Insight{
		InsightID:  "insight-1",
		Category:   domain.DetectorRetryAmplification,
		Severity:   domain.InsightSeverityHigh,
		DetectedAt: period.StartAt.Add(3 * time.Hour),
		Period:     period,
		Payload:    payload,
	})
	if err := store.UpsertInsights(context.Background(), []domain.Insight{insight}); err != nil {
		t.Fatalf("UpsertInsights() error = %v", err)
	}

	wasteService := service.NewWasteSummaryService(store, store)
	wasteService.ClockForTest(func() time.Time { return period.StartAt.AddDate(0, 0, 10) })
	binding := NewInsightsBinding(wasteService, store)
	binding.startup(context.Background())

	summary, err := binding.LoadWasteSummary("2026-04")
	if err != nil {
		t.Fatalf("LoadWasteSummary() error = %v", err)
	}
	if summary.TotalWasteCostUSD != 3 || summary.TotalSpendCostUSD != 3 {
		t.Fatalf("LoadWasteSummary() = %+v, want waste and spend 3", summary)
	}
	if len(summary.ByDetector) == 0 || len(summary.DailyTrend) == 0 || summary.GeneratedAt == "" {
		t.Fatalf("LoadWasteSummary() = %+v, want detector, trend, and generated time", summary)
	}

	insights, err := binding.LoadInsights("2026-04")
	if err != nil {
		t.Fatalf("LoadInsights() error = %v", err)
	}
	if insights.Empty || len(insights.Items) != 1 {
		t.Fatalf("LoadInsights() = %+v, want one item", insights)
	}
	if got := insights.Items[0]; got.InsightID != insight.InsightID || got.Payload.UsageEntryIDs[0] != entry.EntryID || got.Payload.Metrics[0].Unit != "usd" {
		t.Fatalf("LoadInsights().Items[0] = %+v, want mapped insight payload", got)
	}
}

func mustInsight(t *testing.T, insight domain.Insight) domain.Insight {
	t.Helper()
	validated, err := domain.NewInsight(insight)
	if err != nil {
		t.Fatalf("NewInsight() error = %v", err)
	}
	return validated
}
