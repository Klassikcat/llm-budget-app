package tui

import (
	"fmt"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
)

func renderInsightsLogs(insights []domain.Insight, selection int, width int) string {
	if len(insights) == 0 {
		return strings.Join([]string{
			truncateLine("No insights found for this month.", width),
			helpStyle.Render(truncateLine("Press Esc to return or r to reload.", width)),
		}, "\n\n")
	}

	clampedSelection := clampInsightSelection(selection, len(insights))
	lines := []string{truncateLine(renderInsightCategoryCounts(insights), width)}

	for i, insight := range insights {
		line := fmt.Sprintf("%s  %-28s  %s  %s", insight.DetectedAt.Local().Format(time.DateTime), string(insight.Category), string(insight.Severity), insight.InsightID)
		if i == clampedSelection {
			lines = append(lines, focusStyle.Render(truncateLine("> "+line, width)))
			continue
		}
		lines = append(lines, truncateLine("  "+line, width))
	}

	lines = append(lines, mutedStyle.Render(truncateLine("Enter opens privacy-safe detail metadata. Prompt or response text is never shown here.", width)))
	return strings.Join(lines, "\n")
}

func renderInsightCategoryCounts(insights []domain.Insight) string {
	counts := map[domain.DetectorCategory]int{}
	for _, insight := range insights {
		counts[insight.Category]++
	}

	orderedCategories := []domain.DetectorCategory{
		domain.DetectorContextAvalanche,
		domain.DetectorRepeatedFileReads,
		domain.DetectorRetryAmplification,
		domain.DetectorOverQualifiedModel,
		domain.DetectorToolSchemaBloat,
		domain.DetectorPlanningTax,
		domain.DetectorZombieLoops,
		domain.DetectorMissedPromptCaching,
	}

	parts := make([]string, 0, len(orderedCategories))
	for _, category := range orderedCategories {
		count := counts[category]
		if count == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %d", category, count))
	}

	if len(parts) == 0 {
		return "Categories: none"
	}

	return "Categories: " + strings.Join(parts, " · ")
}

func clampInsightSelection(selection int, total int) int {
	if total <= 0 {
		return 0
	}
	if selection < 0 {
		return 0
	}
	if selection >= total {
		return total - 1
	}
	return selection
}
