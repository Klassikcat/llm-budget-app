package tui

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
)

var update = flag.Bool("update", false, "update .golden files")

func TestInsightsDashboard_FullData(t *testing.T) {
	summary := domain.WasteSummary{
		Period:                    domain.MonthlyPeriod{StartAt: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), EndExclusive: time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)},
		TotalWasteCostUSD:         12.34,
		TotalSpendCostUSD:         100.0,
		WastePercent:              12.34,
		WeeklyWasteCostUSD:        5.0,
		MonthlyWasteCostUSD:       12.34,
		ProjectedMonthEndWasteUSD: 45.67,
		ByDetector: []domain.WasteByDetector{
			{Category: domain.DetectorContextAvalanche, AttributedCostUSD: 5.0, InsightCount: 2},
			{Category: domain.DetectorMissedPromptCaching, AttributedCostUSD: 4.0, InsightCount: 1},
			{Category: domain.DetectorToolSchemaBloat, AttributedCostUSD: 2.0, InsightCount: 1},
			{Category: domain.DetectorOverQualifiedModel, AttributedCostUSD: 1.34, InsightCount: 1},
		},
		TopCauses: []domain.WasteByDetector{
			{Category: domain.DetectorContextAvalanche, AttributedCostUSD: 5.0, InsightCount: 2},
			{Category: domain.DetectorMissedPromptCaching, AttributedCostUSD: 4.0, InsightCount: 1},
			{Category: domain.DetectorToolSchemaBloat, AttributedCostUSD: 2.0, InsightCount: 1},
			{Category: domain.DetectorOverQualifiedModel, AttributedCostUSD: 1.34, InsightCount: 1},
		},
		DailyTrend: []domain.WasteTrendPoint{
			{Day: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), WasteCostUSD: 1.0},
			{Day: time.Date(2026, time.April, 2, 0, 0, 0, 0, time.UTC), WasteCostUSD: 2.0},
			{Day: time.Date(2026, time.April, 3, 0, 0, 0, 0, time.UTC), WasteCostUSD: 0.0},
			{Day: time.Date(2026, time.April, 4, 0, 0, 0, 0, time.UTC), WasteCostUSD: 5.0},
			{Day: time.Date(2026, time.April, 5, 0, 0, 0, 0, time.UTC), WasteCostUSD: 4.34},
		},
	}

	out := renderInsightsDashboard(summary, 80)
	assertGolden(t, "insights_dashboard_full.golden", out)
}

func TestInsightsDashboard_Empty(t *testing.T) {
	summary := domain.WasteSummary{
		Period: domain.MonthlyPeriod{StartAt: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), EndExclusive: time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)},
	}

	out := renderInsightsDashboard(summary, 80)
	assertGolden(t, "insights_dashboard_empty.golden", out)
}

func TestInsightsDashboard_SingleCategory(t *testing.T) {
	summary := domain.WasteSummary{
		Period:                    domain.MonthlyPeriod{StartAt: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), EndExclusive: time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)},
		TotalWasteCostUSD:         10.0,
		TotalSpendCostUSD:         100.0,
		WastePercent:              10.0,
		WeeklyWasteCostUSD:        10.0,
		MonthlyWasteCostUSD:       10.0,
		ProjectedMonthEndWasteUSD: 30.0,
		ByDetector: []domain.WasteByDetector{
			{Category: domain.DetectorContextAvalanche, AttributedCostUSD: 10.0, InsightCount: 1},
		},
		TopCauses: []domain.WasteByDetector{
			{Category: domain.DetectorContextAvalanche, AttributedCostUSD: 10.0, InsightCount: 1},
		},
		DailyTrend: []domain.WasteTrendPoint{
			{Day: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), WasteCostUSD: 10.0},
		},
	}

	out := renderInsightsDashboard(summary, 80)
	assertGolden(t, "insights_dashboard_single.golden", out)
}

func TestInsightsDashboard_NarrowTerminal(t *testing.T) {
	summary := domain.WasteSummary{
		Period:                    domain.MonthlyPeriod{StartAt: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), EndExclusive: time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)},
		TotalWasteCostUSD:         10.0,
		TotalSpendCostUSD:         100.0,
		WastePercent:              10.0,
		WeeklyWasteCostUSD:        10.0,
		MonthlyWasteCostUSD:       10.0,
		ProjectedMonthEndWasteUSD: 30.0,
		ByDetector: []domain.WasteByDetector{
			{Category: domain.DetectorContextAvalanche, AttributedCostUSD: 10.0, InsightCount: 1},
		},
		TopCauses: []domain.WasteByDetector{
			{Category: domain.DetectorContextAvalanche, AttributedCostUSD: 10.0, InsightCount: 1},
		},
		DailyTrend: []domain.WasteTrendPoint{
			{Day: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), WasteCostUSD: 10.0},
		},
	}

	out := renderInsightsDashboard(summary, 50) // < 60
	assertGolden(t, "insights_dashboard_narrow.golden", out)
}

func TestInsightsDashboard_ZeroSpend(t *testing.T) {
	summary := domain.WasteSummary{
		Period:                    domain.MonthlyPeriod{StartAt: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), EndExclusive: time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)},
		TotalWasteCostUSD:         0.0,
		TotalSpendCostUSD:         0.0,
		WastePercent:              0.0,
		WeeklyWasteCostUSD:        0.0,
		MonthlyWasteCostUSD:       0.0,
		ProjectedMonthEndWasteUSD: 0.0,
		ByDetector: []domain.WasteByDetector{
			{Category: domain.DetectorContextAvalanche, AttributedCostUSD: 0.0, InsightCount: 1},
		},
		TopCauses: []domain.WasteByDetector{
			{Category: domain.DetectorContextAvalanche, AttributedCostUSD: 0.0, InsightCount: 1},
		},
		DailyTrend: []domain.WasteTrendPoint{
			{Day: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), WasteCostUSD: 0.0},
		},
	}

	out := renderInsightsDashboard(summary, 80)
	assertGolden(t, "insights_dashboard_zerospend.golden", out)
}

func assertGolden(t *testing.T, filename string, actual string) {
	t.Helper()
	path := filepath.Join("testdata", filename)

	if *update {
		os.MkdirAll("testdata", 0755)
		err := os.WriteFile(path, []byte(actual), 0644)
		if err != nil {
			t.Fatalf("failed to update golden file: %v", err)
		}
	}

	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}

	if string(expected) != actual {
		t.Errorf("output does not match golden file %s\nExpected:\n%s\nActual:\n%s", filename, string(expected), actual)
	}
}
