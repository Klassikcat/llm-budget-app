package tui

import (
	"fmt"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
)

func TestInsightsLogs_WithData(t *testing.T) {
	out := renderInsightsLogs(testInsightsLogsFixture(t), 1, 100)
	assertGolden(t, "insights_logs_withdata.golden", out)
}

func TestInsightsLogs_Empty(t *testing.T) {
	out := renderInsightsLogs(nil, 0, 100)
	assertGolden(t, "insights_logs_empty.golden", out)
}

func TestInsightsLogs_SelectionOutOfRange(t *testing.T) {
	out := renderInsightsLogs(testInsightsLogsFixture(t), 99, 100)
	assertGolden(t, "insights_logs_selection_out_of_range.golden", out)
}

func TestInsightTabsChrome(t *testing.T) {
	assertGolden(t, "insight_tabs_dashboard.golden", renderInsightTabs(insightTabDashboard, 100))
	assertGolden(t, "insight_tabs_logs.golden", renderInsightTabs(insightTabLogs, 100))
}

func testInsightsLogsFixture(t *testing.T) []domain.Insight {
	t.Helper()

	period, err := domain.NewMonthlyPeriod(time.Date(2026, time.April, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}

	categories := []domain.DetectorCategory{
		domain.DetectorContextAvalanche,
		domain.DetectorContextAvalanche,
		domain.DetectorPlanningTax,
		domain.DetectorToolSchemaBloat,
		domain.DetectorMissedPromptCaching,
	}
	severities := []domain.InsightSeverity{
		domain.InsightSeverityHigh,
		domain.InsightSeverityMedium,
		domain.InsightSeverityHigh,
		domain.InsightSeverityLow,
		domain.InsightSeverityMedium,
	}

	insights := make([]domain.Insight, 0, len(categories))
	for i := range categories {
		payload, err := domain.NewInsightPayload([]string{fmt.Sprintf("session-%d", i)}, []string{fmt.Sprintf("usage-%d", i)}, nil, nil, nil)
		if err != nil {
			t.Fatalf("NewInsightPayload() error = %v", err)
		}

		insight, err := domain.NewInsight(domain.Insight{
			InsightID:  fmt.Sprintf("insight-%02d", i),
			Category:   categories[i],
			Severity:   severities[i],
			DetectedAt: period.StartAt.Add(time.Duration(i+1) * time.Hour),
			Period:     period,
			Payload:    payload,
		})
		if err != nil {
			t.Fatalf("NewInsight() error = %v", err)
		}
		insights = append(insights, insight)
	}

	return insights
}
