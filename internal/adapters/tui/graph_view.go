package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/NimbleMarkets/ntcharts/barchart"
	"github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

func getModelColor(modelName string) string {
	colors := []string{"86", "205", "42", "220", "170", "39", "214", "135", "208", "118"}
	hash := 0
	for _, c := range modelName {
		hash += int(c)
	}
	return colors[hash%len(colors)]
}

func renderGraphView(m *model, width int) string {
	switch m.graphTab {
	case graphTabModelTokenUsage:
		return renderModelTokenUsageGraph(m, width)
	case graphTabModelCost:
		return renderModelCostGraph(m, width)
	case graphTabDailyTokenTrend:
		return renderDailyTokenTrendGraph(m, width)
	case graphTabModelTokenBreakdown:
		return renderModelTokenBreakdownGraph(m, width)
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
					Style: lipgloss.NewStyle().Foreground(lipgloss.Color(getModelColor(usage.ModelName))),
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

func renderDailyTokenTrendGraph(m *model, width int) string {
	if len(m.graphData.DailyTokenTrends) == 0 {
		return mutedStyle.Render(truncateLine("No daily token trend available for this month.", width))
	}

	hasUsage := false
	for _, trend := range m.graphData.DailyTokenTrends {
		if len(trend.ModelBreakdown) > 0 {
			hasUsage = true
			break
		}
	}
	if !hasUsage {
		return mutedStyle.Render(truncateLine("No daily token trend available for this month.", width))
	}

	models := make(map[string]lipgloss.Style)
	modelNames := []string{}

	topN := 5
	if len(m.graphData.ModelTokenUsages) < topN {
		topN = len(m.graphData.ModelTokenUsages)
	}

	for i := 0; i < topN; i++ {
		modelName := m.graphData.ModelTokenUsages[i].ModelName
		models[modelName] = lipgloss.NewStyle().Foreground(lipgloss.Color(getModelColor(modelName)))
		modelNames = append(modelNames, modelName)
	}

	datasets := make(map[string][]timeserieslinechart.TimePoint)
	for _, modelName := range modelNames {
		datasets[modelName] = make([]timeserieslinechart.TimePoint, 0, len(m.graphData.DailyTokenTrends))
	}

	for _, trend := range m.graphData.DailyTokenTrends {
		dayUsage := make(map[string]int64)
		for _, breakdown := range trend.ModelBreakdown {
			dayUsage[breakdown.ModelName] = breakdown.TotalTokens
		}

		for _, modelName := range modelNames {
			val := float64(dayUsage[modelName])
			datasets[modelName] = append(datasets[modelName], timeserieslinechart.TimePoint{
				Time:  trend.Date,
				Value: val,
			})
		}
	}

	opts := []timeserieslinechart.Option{}
	for _, modelName := range modelNames {
		opts = append(opts, timeserieslinechart.WithDataSetTimeSeries(modelName, datasets[modelName]))
		opts = append(opts, timeserieslinechart.WithDataSetStyle(modelName, models[modelName]))
	}

	chartHeight := 15

	lc := timeserieslinechart.New(width, chartHeight, opts...)
	lc.DrawBrailleAll()

	var legendItems []string
	for _, modelName := range modelNames {
		style := models[modelName]
		legendItems = append(legendItems, style.Render("■ ")+modelName)
	}

	var legendRows []string
	currentRow := ""
	for i, item := range legendItems {
		if i > 0 && i%3 == 0 {
			legendRows = append(legendRows, currentRow)
			currentRow = item
		} else {
			if currentRow != "" {
				currentRow += "   " + item
			} else {
				currentRow = item
			}
		}
	}
	if currentRow != "" {
		legendRows = append(legendRows, currentRow)
	}

	legend := lipgloss.JoinVertical(lipgloss.Left, legendRows...)

	return lipgloss.JoinVertical(lipgloss.Left, lc.View(), "", legend)
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
					Style: lipgloss.NewStyle().Foreground(lipgloss.Color(getModelColor(cost.ModelName))),
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

func renderModelTokenBreakdownGraph(m *model, width int) string {
	if len(m.graphData.ModelTokenBreakdowns) == 0 {
		return mutedStyle.Render(truncateLine("No token breakdown data for this month.", width))
	}

	var sections []string

	barWidth := width - 4
	if barWidth > 60 {
		barWidth = 60
	}
	if barWidth < 20 {
		barWidth = 20
	}

	inputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	outputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	cacheReadStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	cacheWriteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	legendItemStyle := lipgloss.NewStyle().Width(28)

	for _, breakdown := range m.graphData.ModelTokenBreakdowns {
		total := breakdown.TotalTokens
		if total == 0 {
			continue
		}

		pcts := []float64{
			float64(breakdown.InputTokens) / float64(total),
			float64(breakdown.OutputTokens) / float64(total),
			float64(breakdown.CacheReadTokens) / float64(total),
			float64(breakdown.CacheWriteTokens) / float64(total),
		}

		blocks := make([]int, 4)
		totalBlocks := 0
		for i, p := range pcts {
			blocks[i] = int(p * float64(barWidth))
			totalBlocks += blocks[i]
		}

		type rem struct {
			idx int
			val float64
		}
		rems := make([]rem, 4)
		for i, p := range pcts {
			rems[i] = rem{idx: i, val: p*float64(barWidth) - float64(blocks[i])}
		}

		for i := 0; i < len(rems)-1; i++ {
			for j := i + 1; j < len(rems); j++ {
				if rems[i].val < rems[j].val {
					rems[i], rems[j] = rems[j], rems[i]
				}
			}
		}

		for i := 0; i < barWidth-totalBlocks; i++ {
			blocks[rems[i].idx]++
		}

		bar := ""
		if blocks[0] > 0 {
			bar += inputStyle.Render(strings.Repeat("█", blocks[0]))
		}
		if blocks[1] > 0 {
			bar += outputStyle.Render(strings.Repeat("█", blocks[1]))
		}
		if blocks[2] > 0 {
			bar += cacheReadStyle.Render(strings.Repeat("█", blocks[2]))
		}
		if blocks[3] > 0 {
			bar += cacheWriteStyle.Render(strings.Repeat("█", blocks[3]))
		}

		title := titleStyle.Render(breakdown.ModelName) + mutedStyle.Render(fmt.Sprintf(" • %s tokens", humanize.Comma(total)))

		legendInput := legendItemStyle.Render(fmt.Sprintf("%s Input: %s (%.1f%%)", inputStyle.Render("■"), humanize.Comma(breakdown.InputTokens), pcts[0]*100))
		legendOutput := legendItemStyle.Render(fmt.Sprintf("%s Output: %s (%.1f%%)", outputStyle.Render("■"), humanize.Comma(breakdown.OutputTokens), pcts[1]*100))
		legendCacheRead := legendItemStyle.Render(fmt.Sprintf("%s C.Read: %s (%.1f%%)", cacheReadStyle.Render("■"), humanize.Comma(breakdown.CacheReadTokens), pcts[2]*100))
		legendCacheWrite := legendItemStyle.Render(fmt.Sprintf("%s C.Write: %s (%.1f%%)", cacheWriteStyle.Render("■"), humanize.Comma(breakdown.CacheWriteTokens), pcts[3]*100))

		legend := legendInput + legendOutput + "\n" + legendCacheRead + legendCacheWrite

		section := title + "\n" + bar + "\n" + legend
		sections = append(sections, section)
	}

	if len(sections) == 0 {
		return mutedStyle.Render(truncateLine("No token breakdown data for this month.", width))
	}

	var joined []string
	for i, s := range sections {
		if i > 0 {
			joined = append(joined, "")
		}
		joined = append(joined, s)
	}

	return lipgloss.JoinVertical(lipgloss.Left, joined...)
}
