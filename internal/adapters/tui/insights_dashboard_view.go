package tui

import (
	"fmt"
	"strings"

	"github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"llm-budget-tracker/internal/domain"
)

func renderInsightsDashboard(summary domain.WasteSummary, width int) string {
	if summary.TotalSpendCostUSD == 0 && summary.TotalWasteCostUSD == 0 && len(summary.ByDetector) == 0 {
		return renderEmptyDashboard(summary.Period, width)
	}

	var sections []string

	// Row 1: W1, W2
	w1 := renderWasteHeadlineCard(summary, width)
	w2 := renderWastePercentCard(summary, width)
	sections = append(sections, w1, w2)

	// Row 2: W3, W4
	w3 := renderWasteProjectionCard(summary, width)
	w4 := renderWasteWeeklyCard(summary, width)
	sections = append(sections, w3, w4)

	// Row 3: W5, W6
	w5 := renderTopCausesBarList(summary, width)
	w6 := renderDailyWasteTrend(summary, width)
	sections = append(sections, w5, w6)

	return strings.Join(sections, "\n\n")
}

func renderEmptyDashboard(period domain.MonthlyPeriod, width int) string {
	lines := []string{
		titleStyle.Render("This Month Waste"),
		truncateLine("No waste data available for this month.", width),
		truncateLine("Add usage entries and run insights to populate the dashboard.", width),
	}
	return strings.Join(lines, "\n")
}

func renderWasteHeadlineCard(summary domain.WasteSummary, width int) string {
	lines := []string{
		titleStyle.Render("This Month Waste"),
		fmt.Sprintf("%s", formatUSD(summary.TotalWasteCostUSD)),
	}
	return strings.Join(lines, "\n")
}

func renderWastePercentCard(summary domain.WasteSummary, width int) string {
	lines := []string{
		titleStyle.Render("Waste % of Total Spend"),
		fmt.Sprintf("%.1f%%", summary.WastePercent),
	}
	return strings.Join(lines, "\n")
}

func renderWasteProjectionCard(summary domain.WasteSummary, width int) string {
	lines := []string{
		titleStyle.Render("Projected Month-End Waste"),
		fmt.Sprintf("%s %s", formatUSD(summary.ProjectedMonthEndWasteUSD), mutedStyle.Render("proj.")),
	}
	return strings.Join(lines, "\n")
}

func renderWasteWeeklyCard(summary domain.WasteSummary, width int) string {
	lines := []string{
		titleStyle.Render("This Week Waste"),
		fmt.Sprintf("%s", formatUSD(summary.WeeklyWasteCostUSD)),
	}
	return strings.Join(lines, "\n")
}

func renderTopCausesBarList(summary domain.WasteSummary, width int) string {
	lines := []string{titleStyle.Render("Top Waste Causes")}

	if len(summary.TopCauses) == 0 {
		lines = append(lines, mutedStyle.Render("No waste causes found."))
		return strings.Join(lines, "\n")
	}

	maxCost := 0.0
	for _, cause := range summary.TopCauses {
		if cause.AttributedCostUSD > maxCost {
			maxCost = cause.AttributedCostUSD
		}
	}

	maxBarWidth := 20
	for _, cause := range summary.TopCauses {
		barWidth := 0
		if maxCost > 0 {
			barWidth = int((cause.AttributedCostUSD / maxCost) * float64(maxBarWidth))
		}
		if barWidth == 0 && cause.AttributedCostUSD > 0 {
			barWidth = 1
		}

		bar := strings.Repeat("█", barWidth)
		line := fmt.Sprintf("%-20s %8s  %s", bar, formatUSD(cause.AttributedCostUSD), cause.Category)
		lines = append(lines, truncateLine(line, width))
	}

	return strings.Join(lines, "\n")
}

func renderDailyWasteTrend(summary domain.WasteSummary, width int) string {
	lines := []string{titleStyle.Render("Daily Waste Trend (30-day)")}

	if width < 60 {
		lines = append(lines, mutedStyle.Render("Trend requires ≥60 cols"))
		return strings.Join(lines, "\n")
	}

	if len(summary.DailyTrend) == 0 {
		lines = append(lines, mutedStyle.Render("No trend data available."))
		return strings.Join(lines, "\n")
	}

	var points []timeserieslinechart.TimePoint
	for _, pt := range summary.DailyTrend {
		points = append(points, timeserieslinechart.TimePoint{
			Time:  pt.Day,
			Value: pt.WasteCostUSD,
		})
	}

	chartHeight := 10
	lc := timeserieslinechart.New(width, chartHeight, timeserieslinechart.WithDataSetTimeSeries("Waste", points))
	lc.DrawBrailleAll()

	lines = append(lines, lc.View())
	return strings.Join(lines, "\n")
}
