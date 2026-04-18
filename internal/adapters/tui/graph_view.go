package tui

import (
	"fmt"
	"unicode/utf8"

	"github.com/NimbleMarkets/ntcharts/barchart"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

func renderGraphView(m *model, width int) string {
	switch m.graphTab {
	case graphTabModelTokenUsage:
		return renderModelTokenUsageGraph(m, width)
	case graphTabModelCost:
		return renderModelCostGraph(m, width)
	default:
		return renderGraphPlaceholder(m, width)
	}
}

func renderModelTokenUsageGraph(m *model, width int) string {
	if len(m.graphData.ModelTokenUsages) == 0 {
		return mutedStyle.Render(truncateLine("No model token activity for this month.", width))
	}

	var totalTokens int64
	for _, usage := range m.graphData.ModelTokenUsages {
		totalTokens += usage.TotalTokens
	}

	data := make([]barchart.BarData, 0, len(m.graphData.ModelTokenUsages))
	for i, usage := range m.graphData.ModelTokenUsages {
		percent := 0.0
		if totalTokens > 0 {
			percent = float64(usage.TotalTokens) / float64(totalTokens) * 100
		}
		rank := i + 1

		label := fmt.Sprintf("#%d  %s   %s tokens (%.1f%%)", rank, usage.ModelName, humanize.Comma(usage.TotalTokens), percent)
		maxLabelWidth := width / 2
		if maxLabelWidth < 20 {
			maxLabelWidth = 20
		}
		if utf8.RuneCountInString(label) > maxLabelWidth {
			tokensPart := fmt.Sprintf("   %s tokens (%.1f%%)", humanize.Comma(usage.TotalTokens), percent)
			rankPart := fmt.Sprintf("#%d  ", rank)
			allowedModelLen := maxLabelWidth - utf8.RuneCountInString(tokensPart) - utf8.RuneCountInString(rankPart)
			if allowedModelLen > 3 {
				runes := []rune(usage.ModelName)
				if len(runes) > allowedModelLen-1 {
					label = rankPart + string(runes[:allowedModelLen-1]) + "…" + tokensPart
				}
			} else {
				runes := []rune(label)
				label = string(runes[:maxLabelWidth-1]) + "…"
			}
		}

		data = append(data, barchart.BarData{
			Label: label,
			Values: []barchart.BarValue{
				{
					Value: float64(usage.TotalTokens),
					Style: lipgloss.NewStyle().Foreground(lipgloss.Color("86")),
				},
			},
		})
	}

	height := len(data)*2 - 1
	if height < 1 {
		height = 1
	}

	bc := barchart.New(
		width,
		height,
		barchart.WithDataSet(data),
		barchart.WithHorizontalBars(),
		barchart.WithNoAutoBarWidth(),
		barchart.WithBarWidth(1),
		barchart.WithBarGap(1),
	)
	bc.Draw()

	return bc.View()
}

func renderModelCostGraph(m *model, width int) string {
	if len(m.graphData.ModelCosts) == 0 {
		return mutedStyle.Render(truncateLine("No model cost activity for this month.", width))
	}

	var totalCost float64
	for _, cost := range m.graphData.ModelCosts {
		totalCost += cost.TotalCostUSD
	}

	data := make([]barchart.BarData, 0, len(m.graphData.ModelCosts))
	for i, cost := range m.graphData.ModelCosts {
		percent := 0.0
		if totalCost > 0 {
			percent = cost.TotalCostUSD / totalCost * 100
		}
		rank := i + 1

		label := fmt.Sprintf("#%d  %s   %s (%.1f%%)", rank, cost.ModelName, formatUSD(cost.TotalCostUSD), percent)
		maxLabelWidth := width / 2
		if maxLabelWidth < 20 {
			maxLabelWidth = 20
		}
		if utf8.RuneCountInString(label) > maxLabelWidth {
			costPart := fmt.Sprintf("   %s (%.1f%%)", formatUSD(cost.TotalCostUSD), percent)
			rankPart := fmt.Sprintf("#%d  ", rank)
			allowedModelLen := maxLabelWidth - utf8.RuneCountInString(costPart) - utf8.RuneCountInString(rankPart)
			if allowedModelLen > 3 {
				runes := []rune(cost.ModelName)
				if len(runes) > allowedModelLen-1 {
					label = rankPart + string(runes[:allowedModelLen-1]) + "…" + costPart
				}
			} else {
				runes := []rune(label)
				label = string(runes[:maxLabelWidth-1]) + "…"
			}
		}

		data = append(data, barchart.BarData{
			Label: label,
			Values: []barchart.BarValue{
				{
					Value: cost.TotalCostUSD,
					Style: lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
				},
			},
		})
	}

	height := len(data)*2 - 1
	if height < 1 {
		height = 1
	}

	bc := barchart.New(
		width,
		height,
		barchart.WithDataSet(data),
		barchart.WithHorizontalBars(),
		barchart.WithNoAutoBarWidth(),
		barchart.WithBarWidth(1),
		barchart.WithBarGap(1),
	)
	bc.Draw()

	return bc.View()
}
