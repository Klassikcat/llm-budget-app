package service

import (
	"context"
	"reflect"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestWasteSummaryService_QueryWasteSummary_REDScenarios(t *testing.T) {
	t.Parallel()

	period := mustWasteSummaryPeriod(t, 2026, time.April)
	clock := time.Date(2026, time.April, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		period    domain.MonthlyPeriod
		clock     time.Time
		entries   []domain.UsageEntry
		insights  []domain.Insight
		assertion func(*testing.T, domain.WasteSummary)
	}{
		{
			name:     "empty data",
			period:   period,
			clock:    clock,
			entries:  nil,
			insights: nil,
			assertion: func(t *testing.T, got domain.WasteSummary) {
				t.Helper()
				assertPeriodAndGeneratedAt(t, got, period, clock)
				assertDetectorOrder(t, got.ByDetector)
				assertDailyTrendDays(t, got.DailyTrend, 15)
			},
		},
		{
			name:   "single entry and single insight",
			period: period,
			clock:  clock,
			entries: []domain.UsageEntry{
				usageEntry("entry-1", time.Date(2026, time.April, 10, 8, 0, 0, 0, time.UTC), 12.5),
			},
			insights: []domain.Insight{
				insight("insight-1", domain.DetectorPlanningTax, domain.InsightSeverityHigh, time.Date(2026, time.April, 11, 8, 0, 0, 0, time.UTC), period, []string{"entry-1"}),
			},
			assertion: func(t *testing.T, got domain.WasteSummary) {
				t.Helper()
				assertPeriodAndGeneratedAt(t, got, period, clock)
				assertFloatEqual(t, "total waste", got.TotalWasteCostUSD, 12.5)
				assertFloatEqual(t, "total spend", got.TotalSpendCostUSD, 12.5)
				assertFloatEqual(t, "waste percent", got.WastePercent, 100)
				assertDetectorCostAndCount(t, got.ByDetector, domain.DetectorPlanningTax, 12.5, 1)
			},
		},
		{
			name:   "one entry referenced by two insights of different severities",
			period: period,
			clock:  clock,
			entries: []domain.UsageEntry{
				usageEntry("entry-1", time.Date(2026, time.April, 9, 9, 0, 0, 0, time.UTC), 20),
			},
			insights: []domain.Insight{
				insight("insight-medium", domain.DetectorRetryAmplification, domain.InsightSeverityMedium, time.Date(2026, time.April, 10, 9, 0, 0, 0, time.UTC), period, []string{"entry-1"}),
				insight("insight-high", domain.DetectorOverQualifiedModel, domain.InsightSeverityHigh, time.Date(2026, time.April, 11, 9, 0, 0, 0, time.UTC), period, []string{"entry-1"}),
			},
			assertion: func(t *testing.T, got domain.WasteSummary) {
				t.Helper()
				assertFloatEqual(t, "total waste", got.TotalWasteCostUSD, 20)
				assertDetectorCostAndCount(t, got.ByDetector, domain.DetectorOverQualifiedModel, 20, 1)
				assertDetectorCostAndCount(t, got.ByDetector, domain.DetectorRetryAmplification, 0, 1)
			},
		},
		{
			name:   "entry with no insight",
			period: period,
			clock:  clock,
			entries: []domain.UsageEntry{
				usageEntry("entry-1", time.Date(2026, time.April, 6, 14, 0, 0, 0, time.UTC), 7.25),
			},
			insights: nil,
			assertion: func(t *testing.T, got domain.WasteSummary) {
				t.Helper()
				assertFloatEqual(t, "total spend", got.TotalSpendCostUSD, 7.25)
				assertFloatEqual(t, "total waste", got.TotalWasteCostUSD, 0)
				assertFloatEqual(t, "waste percent", got.WastePercent, 0)
			},
		},
		{
			name:   "missing referenced entry id",
			period: period,
			clock:  clock,
			entries: []domain.UsageEntry{
				usageEntry("entry-1", time.Date(2026, time.April, 7, 10, 0, 0, 0, time.UTC), 5),
			},
			insights: []domain.Insight{
				insight("insight-1", domain.DetectorToolSchemaBloat, domain.InsightSeverityHigh, time.Date(2026, time.April, 8, 10, 0, 0, 0, time.UTC), period, []string{"missing-entry"}),
			},
			assertion: func(t *testing.T, got domain.WasteSummary) {
				t.Helper()
				assertFloatEqual(t, "total spend", got.TotalSpendCostUSD, 5)
				assertFloatEqual(t, "total waste", got.TotalWasteCostUSD, 0)
				assertDetectorCostAndCount(t, got.ByDetector, domain.DetectorToolSchemaBloat, 0, 1)
			},
		},
		{
			name:   "projection with elapsed_days equals zero",
			period: mustWasteSummaryPeriod(t, 2026, time.May),
			clock:  time.Date(2026, time.April, 30, 23, 59, 0, 0, time.UTC),
			entries: []domain.UsageEntry{
				usageEntry("entry-1", time.Date(2026, time.May, 1, 1, 0, 0, 0, time.UTC), 9),
			},
			insights: []domain.Insight{
				insight("insight-1", domain.DetectorContextAvalanche, domain.InsightSeverityHigh, time.Date(2026, time.May, 1, 2, 0, 0, 0, time.UTC), mustWasteSummaryPeriod(t, 2026, time.May), []string{"entry-1"}),
			},
			assertion: func(t *testing.T, got domain.WasteSummary) {
				t.Helper()
				assertFloatEqual(t, "projected waste", got.ProjectedMonthEndWasteUSD, 9)
			},
		},
		{
			name:   "projection with elapsed_days greater than zero",
			period: period,
			clock:  clock,
			entries: []domain.UsageEntry{
				usageEntry("entry-1", time.Date(2026, time.April, 2, 12, 0, 0, 0, time.UTC), 15),
			},
			insights: []domain.Insight{
				insight("insight-1", domain.DetectorContextAvalanche, domain.InsightSeverityHigh, time.Date(2026, time.April, 3, 12, 0, 0, 0, time.UTC), period, []string{"entry-1"}),
			},
			assertion: func(t *testing.T, got domain.WasteSummary) {
				t.Helper()
				assertFloatEqual(t, "projected waste", got.ProjectedMonthEndWasteUSD, 30)
			},
		},
		{
			name:    "waste percent with zero spend",
			period:  period,
			clock:   clock,
			entries: nil,
			insights: []domain.Insight{
				insight("insight-1", domain.DetectorZombieLoops, domain.InsightSeverityMedium, time.Date(2026, time.April, 4, 7, 0, 0, 0, time.UTC), period, []string{"missing-entry"}),
			},
			assertion: func(t *testing.T, got domain.WasteSummary) {
				t.Helper()
				assertPeriodAndGeneratedAt(t, got, period, clock)
				assertDetectorOrder(t, got.ByDetector)
				assertFloatEqual(t, "total spend", got.TotalSpendCostUSD, 0)
				assertFloatEqual(t, "waste percent", got.WastePercent, 0)
			},
		},
		{
			name:   "daily trend contiguous days including zeros",
			period: mustWasteSummaryPeriod(t, 2026, time.April),
			clock:  time.Date(2026, time.April, 3, 18, 0, 0, 0, time.UTC),
			entries: []domain.UsageEntry{
				usageEntry("entry-1", time.Date(2026, time.April, 1, 10, 0, 0, 0, time.UTC), 4),
				usageEntry("entry-2", time.Date(2026, time.April, 3, 10, 0, 0, 0, time.UTC), 6),
			},
			insights: []domain.Insight{
				insight("insight-1", domain.DetectorPlanningTax, domain.InsightSeverityHigh, time.Date(2026, time.April, 1, 11, 0, 0, 0, time.UTC), mustWasteSummaryPeriod(t, 2026, time.April), []string{"entry-1"}),
				insight("insight-2", domain.DetectorPlanningTax, domain.InsightSeverityHigh, time.Date(2026, time.April, 3, 11, 0, 0, 0, time.UTC), mustWasteSummaryPeriod(t, 2026, time.April), []string{"entry-2"}),
			},
			assertion: func(t *testing.T, got domain.WasteSummary) {
				t.Helper()
				want := []domain.WasteTrendPoint{
					{Day: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), WasteCostUSD: 4},
					{Day: time.Date(2026, time.April, 2, 0, 0, 0, 0, time.UTC), WasteCostUSD: 0},
					{Day: time.Date(2026, time.April, 3, 0, 0, 0, 0, time.UTC), WasteCostUSD: 6},
				}
				if !reflect.DeepEqual(got.DailyTrend, want) {
					t.Fatalf("daily trend mismatch\nwant: %#v\n got: %#v", want, got.DailyTrend)
				}
			},
		},
		{
			name:   "top causes with three categories",
			period: period,
			clock:  clock,
			entries: []domain.UsageEntry{
				usageEntry("entry-1", time.Date(2026, time.April, 1, 10, 0, 0, 0, time.UTC), 10),
				usageEntry("entry-2", time.Date(2026, time.April, 2, 10, 0, 0, 0, time.UTC), 8),
				usageEntry("entry-3", time.Date(2026, time.April, 3, 10, 0, 0, 0, time.UTC), 6),
			},
			insights: []domain.Insight{
				insight("insight-1", domain.DetectorContextAvalanche, domain.InsightSeverityHigh, time.Date(2026, time.April, 1, 11, 0, 0, 0, time.UTC), period, []string{"entry-1"}),
				insight("insight-2", domain.DetectorPlanningTax, domain.InsightSeverityHigh, time.Date(2026, time.April, 2, 11, 0, 0, 0, time.UTC), period, []string{"entry-2"}),
				insight("insight-3", domain.DetectorZombieLoops, domain.InsightSeverityHigh, time.Date(2026, time.April, 3, 11, 0, 0, 0, time.UTC), period, []string{"entry-3"}),
			},
			assertion: func(t *testing.T, got domain.WasteSummary) {
				t.Helper()
				want := []domain.WasteByDetector{
					{Category: domain.DetectorContextAvalanche, AttributedCostUSD: 10, InsightCount: 1},
					{Category: domain.DetectorPlanningTax, AttributedCostUSD: 8, InsightCount: 1},
					{Category: domain.DetectorZombieLoops, AttributedCostUSD: 6, InsightCount: 1},
				}
				if !reflect.DeepEqual(got.TopCauses, want) {
					t.Fatalf("top causes mismatch\nwant: %#v\n got: %#v", want, got.TopCauses)
				}
			},
		},
		{
			name:   "top causes with eight categories",
			period: period,
			clock:  clock,
			entries: []domain.UsageEntry{
				usageEntry("entry-1", time.Date(2026, time.April, 1, 9, 0, 0, 0, time.UTC), 8),
				usageEntry("entry-2", time.Date(2026, time.April, 2, 9, 0, 0, 0, time.UTC), 7),
				usageEntry("entry-3", time.Date(2026, time.April, 3, 9, 0, 0, 0, time.UTC), 6),
				usageEntry("entry-4", time.Date(2026, time.April, 4, 9, 0, 0, 0, time.UTC), 5),
				usageEntry("entry-5", time.Date(2026, time.April, 5, 9, 0, 0, 0, time.UTC), 4),
				usageEntry("entry-6", time.Date(2026, time.April, 6, 9, 0, 0, 0, time.UTC), 3),
				usageEntry("entry-7", time.Date(2026, time.April, 7, 9, 0, 0, 0, time.UTC), 2),
				usageEntry("entry-8", time.Date(2026, time.April, 8, 9, 0, 0, 0, time.UTC), 1),
			},
			insights: []domain.Insight{
				insight("insight-1", domain.DetectorContextAvalanche, domain.InsightSeverityHigh, time.Date(2026, time.April, 1, 10, 0, 0, 0, time.UTC), period, []string{"entry-1"}),
				insight("insight-2", domain.DetectorRepeatedFileReads, domain.InsightSeverityHigh, time.Date(2026, time.April, 2, 10, 0, 0, 0, time.UTC), period, []string{"entry-2"}),
				insight("insight-3", domain.DetectorRetryAmplification, domain.InsightSeverityHigh, time.Date(2026, time.April, 3, 10, 0, 0, 0, time.UTC), period, []string{"entry-3"}),
				insight("insight-4", domain.DetectorOverQualifiedModel, domain.InsightSeverityHigh, time.Date(2026, time.April, 4, 10, 0, 0, 0, time.UTC), period, []string{"entry-4"}),
				insight("insight-5", domain.DetectorToolSchemaBloat, domain.InsightSeverityHigh, time.Date(2026, time.April, 5, 10, 0, 0, 0, time.UTC), period, []string{"entry-5"}),
				insight("insight-6", domain.DetectorPlanningTax, domain.InsightSeverityHigh, time.Date(2026, time.April, 6, 10, 0, 0, 0, time.UTC), period, []string{"entry-6"}),
				insight("insight-7", domain.DetectorZombieLoops, domain.InsightSeverityHigh, time.Date(2026, time.April, 7, 10, 0, 0, 0, time.UTC), period, []string{"entry-7"}),
				insight("insight-8", domain.DetectorMissedPromptCaching, domain.InsightSeverityHigh, time.Date(2026, time.April, 8, 10, 0, 0, 0, time.UTC), period, []string{"entry-8"}),
			},
			assertion: func(t *testing.T, got domain.WasteSummary) {
				t.Helper()
				want := []domain.WasteByDetector{
					{Category: domain.DetectorContextAvalanche, AttributedCostUSD: 8, InsightCount: 1},
					{Category: domain.DetectorRepeatedFileReads, AttributedCostUSD: 7, InsightCount: 1},
					{Category: domain.DetectorRetryAmplification, AttributedCostUSD: 6, InsightCount: 1},
					{Category: domain.DetectorOverQualifiedModel, AttributedCostUSD: 5, InsightCount: 1},
					{Category: domain.DetectorToolSchemaBloat, AttributedCostUSD: 4, InsightCount: 1},
				}
				if !reflect.DeepEqual(got.TopCauses, want) {
					t.Fatalf("top causes mismatch\nwant: %#v\n got: %#v", want, got.TopCauses)
				}
			},
		},
		{
			name:   "category breakdown enum order",
			period: period,
			clock:  clock,
			entries: []domain.UsageEntry{
				usageEntry("entry-1", time.Date(2026, time.April, 12, 12, 0, 0, 0, time.UTC), 1),
			},
			insights: []domain.Insight{
				insight("insight-1", domain.DetectorMissedPromptCaching, domain.InsightSeverityHigh, time.Date(2026, time.April, 12, 13, 0, 0, 0, time.UTC), period, []string{"entry-1"}),
			},
			assertion: func(t *testing.T, got domain.WasteSummary) {
				t.Helper()
				assertDetectorOrder(t, got.ByDetector)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewWasteSummaryService(
				stubWasteUsageRepo{entries: tc.entries},
				stubInsightRepo{insights: tc.insights},
			)
			service.ClockForTest(func() time.Time { return tc.clock })

			got, err := service.QueryWasteSummary(context.Background(), tc.period)
			if err != nil {
				t.Fatalf("QueryWasteSummary() error = %v", err)
			}

			tc.assertion(t, got)
		})
	}
}

type stubWasteUsageRepo struct{ entries []domain.UsageEntry }

func (s stubWasteUsageRepo) UpsertUsageEntries(context.Context, []domain.UsageEntry) error {
	return nil
}
func (s stubWasteUsageRepo) ListUsageEntries(context.Context, ports.UsageFilter) ([]domain.UsageEntry, error) {
	return append([]domain.UsageEntry(nil), s.entries...), nil
}

type stubInsightRepo struct{ insights []domain.Insight }

func (s stubInsightRepo) UpsertInsights(context.Context, []domain.Insight) error { return nil }
func (s stubInsightRepo) ListInsights(context.Context, domain.MonthlyPeriod) ([]domain.Insight, error) {
	return append([]domain.Insight(nil), s.insights...), nil
}

func mustWasteSummaryPeriod(t *testing.T, year int, month time.Month) domain.MonthlyPeriod {
	t.Helper()

	period, err := domain.NewMonthlyPeriodFromParts(year, month)
	if err != nil {
		t.Fatalf("NewMonthlyPeriodFromParts() error = %v", err)
	}

	return period
}

func usageEntry(entryID string, occurredAt time.Time, totalCost float64) domain.UsageEntry {
	return domain.UsageEntry{
		EntryID:       entryID,
		Source:        domain.UsageSourceCLISession,
		Provider:      domain.ProviderOpenAI,
		BillingMode:   domain.BillingModeBYOK,
		OccurredAt:    occurredAt,
		CostBreakdown: domain.CostBreakdown{TotalUSD: totalCost},
	}
}

func insight(insightID string, category domain.DetectorCategory, severity domain.InsightSeverity, detectedAt time.Time, period domain.MonthlyPeriod, usageEntryIDs []string) domain.Insight {
	return domain.Insight{
		InsightID:  insightID,
		Category:   category,
		Severity:   severity,
		DetectedAt: detectedAt,
		Period:     period,
		Payload: domain.InsightPayload{
			UsageEntryIDs: append([]string(nil), usageEntryIDs...),
		},
	}
}

func assertPeriodAndGeneratedAt(t *testing.T, got domain.WasteSummary, wantPeriod domain.MonthlyPeriod, wantGeneratedAt time.Time) {
	t.Helper()

	if got.Period != wantPeriod {
		t.Fatalf("period mismatch\nwant: %#v\n got: %#v", wantPeriod, got.Period)
	}

	if !got.GeneratedAt.Equal(wantGeneratedAt) {
		t.Fatalf("generated_at mismatch\nwant: %s\n got: %s", wantGeneratedAt, got.GeneratedAt)
	}
}

func assertDetectorOrder(t *testing.T, got []domain.WasteByDetector) {
	t.Helper()

	want := []domain.DetectorCategory{
		domain.DetectorContextAvalanche,
		domain.DetectorRepeatedFileReads,
		domain.DetectorRetryAmplification,
		domain.DetectorOverQualifiedModel,
		domain.DetectorToolSchemaBloat,
		domain.DetectorPlanningTax,
		domain.DetectorZombieLoops,
		domain.DetectorMissedPromptCaching,
	}

	if len(got) != len(want) {
		t.Fatalf("detector breakdown length mismatch\nwant: %d\n got: %d", len(want), len(got))
	}

	for index, category := range want {
		if got[index].Category != category {
			t.Fatalf("detector breakdown order mismatch at index %d\nwant: %s\n got: %s", index, category, got[index].Category)
		}
	}
}

func assertDailyTrendDays(t *testing.T, got []domain.WasteTrendPoint, wantDays int) {
	t.Helper()

	if len(got) != wantDays {
		t.Fatalf("daily trend length mismatch\nwant: %d\n got: %d", wantDays, len(got))
	}
}

func assertFloatEqual(t *testing.T, label string, got, want float64) {
	t.Helper()

	if got != want {
		t.Fatalf("%s mismatch\nwant: %v\n got: %v", label, want, got)
	}
}

func assertDetectorCostAndCount(t *testing.T, got []domain.WasteByDetector, category domain.DetectorCategory, wantCost float64, wantCount int) {
	t.Helper()

	for _, detector := range got {
		if detector.Category != category {
			continue
		}

		if detector.AttributedCostUSD != wantCost {
			t.Fatalf("detector %s cost mismatch\nwant: %v\n got: %v", category, wantCost, detector.AttributedCostUSD)
		}
		if detector.InsightCount != wantCount {
			t.Fatalf("detector %s insight count mismatch\nwant: %d\n got: %d", category, wantCount, detector.InsightCount)
		}
		return
	}

	t.Fatalf("detector %s not found in breakdown: %#v", category, got)
}
