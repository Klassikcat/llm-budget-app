package gui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/service"
)

type graphQuerier interface {
	QueryGraphs(ctx context.Context, query service.GraphQuery) (service.GraphSnapshot, error)
}

type GraphsBinding struct {
	queryService graphQuerier
	ctx          context.Context
	clock        func() time.Time
}

func NewGraphsBinding(queryService graphQuerier) *GraphsBinding {
	return &GraphsBinding{
		queryService: queryService,
		clock:        func() time.Time { return time.Now().UTC() },
	}
}

func (b *GraphsBinding) startup(ctx context.Context) {
	if b == nil {
		return
	}
	b.ctx = ctx
}

func (b *GraphsBinding) LoadGraphs(month string, timeRange string) (GraphResponse, error) {
	if b == nil || b.queryService == nil {
		return GraphResponse{}, fmt.Errorf("graph query service is not initialized")
	}

	period, err := resolveGraphBindingPeriod(month, timeRange, b.clock)
	if err != nil {
		return GraphResponse{}, err
	}

	snapshot, err := b.queryService.QueryGraphs(b.context(), service.GraphQuery{Period: period})
	if err != nil {
		return GraphResponse{}, err
	}

	return toGraphResponse(snapshot), nil
}

func (b *GraphsBinding) context() context.Context {
	if b != nil && b.ctx != nil {
		return b.ctx
	}
	return context.Background()
}

type GraphResponse struct {
	ModelTokenUsages     []ModelTokenUsageResponse     `json:"modelTokenUsages"`
	ModelCosts           []ModelCostResponse           `json:"modelCosts"`
	DailyTokenTrends     []DailyTokenTrendResponse     `json:"dailyTokenTrends"`
	ModelTokenBreakdowns []ModelTokenBreakdownResponse `json:"modelTokenBreakdowns"`
}

type ModelTokenUsageResponse struct {
	ModelName        string `json:"modelName"`
	TotalTokens      int64  `json:"totalTokens"`
	InputTokens      int64  `json:"inputTokens"`
	OutputTokens     int64  `json:"outputTokens"`
	CacheReadTokens  int64  `json:"cacheReadTokens"`
	CacheWriteTokens int64  `json:"cacheWriteTokens"`
}

type ModelCostResponse struct {
	ModelName    string  `json:"modelName"`
	TotalCostUSD float64 `json:"totalCostUsd"`
}

type DailyTokenTrendResponse struct {
	Date           string                     `json:"date"`
	ModelBreakdown []ModelDailyTokensResponse `json:"modelBreakdown"`
}

type ModelDailyTokensResponse struct {
	ModelName   string `json:"modelName"`
	TotalTokens int64  `json:"totalTokens"`
}

type ModelTokenBreakdownResponse struct {
	ModelName        string `json:"modelName"`
	InputTokens      int64  `json:"inputTokens"`
	OutputTokens     int64  `json:"outputTokens"`
	CacheReadTokens  int64  `json:"cacheReadTokens"`
	CacheWriteTokens int64  `json:"cacheWriteTokens"`
	TotalTokens      int64  `json:"totalTokens"`
}

func toGraphResponse(snapshot service.GraphSnapshot) GraphResponse {
	modelTokenUsages := make([]ModelTokenUsageResponse, 0, len(snapshot.ModelTokenUsages))
	for _, usage := range snapshot.ModelTokenUsages {
		modelTokenUsages = append(modelTokenUsages, ModelTokenUsageResponse{
			ModelName:        usage.ModelName,
			TotalTokens:      usage.TotalTokens,
			InputTokens:      usage.InputTokens,
			OutputTokens:     usage.OutputTokens,
			CacheReadTokens:  usage.CacheReadTokens,
			CacheWriteTokens: usage.CacheWriteTokens,
		})
	}

	modelCosts := make([]ModelCostResponse, 0, len(snapshot.ModelCosts))
	for _, cost := range snapshot.ModelCosts {
		modelCosts = append(modelCosts, ModelCostResponse{ModelName: cost.ModelName, TotalCostUSD: cost.TotalCostUSD})
	}

	dailyTokenTrends := make([]DailyTokenTrendResponse, 0, len(snapshot.DailyTokenTrends))
	for _, trend := range snapshot.DailyTokenTrends {
		breakdown := make([]ModelDailyTokensResponse, 0, len(trend.ModelBreakdown))
		for _, item := range trend.ModelBreakdown {
			breakdown = append(breakdown, ModelDailyTokensResponse{ModelName: item.ModelName, TotalTokens: item.TotalTokens})
		}
		dailyTokenTrends = append(dailyTokenTrends, DailyTokenTrendResponse{Date: formatDashboardTime(trend.Date), ModelBreakdown: breakdown})
	}

	modelTokenBreakdowns := make([]ModelTokenBreakdownResponse, 0, len(snapshot.ModelTokenBreakdowns))
	for _, breakdown := range snapshot.ModelTokenBreakdowns {
		modelTokenBreakdowns = append(modelTokenBreakdowns, ModelTokenBreakdownResponse{
			ModelName:        breakdown.ModelName,
			InputTokens:      breakdown.InputTokens,
			OutputTokens:     breakdown.OutputTokens,
			CacheReadTokens:  breakdown.CacheReadTokens,
			CacheWriteTokens: breakdown.CacheWriteTokens,
			TotalTokens:      breakdown.TotalTokens,
		})
	}

	return GraphResponse{
		ModelTokenUsages:     modelTokenUsages,
		ModelCosts:           modelCosts,
		DailyTokenTrends:     dailyTokenTrends,
		ModelTokenBreakdowns: modelTokenBreakdowns,
	}
}

func resolveGraphBindingPeriod(month string, timeRange string, clock func() time.Time) (domain.MonthlyPeriod, error) {
	trimmedRange := strings.TrimSpace(timeRange)
	if trimmedRange == "" || trimmedRange == "All" {
		return resolveBindingPeriod(month, clock)
	}

	anchor := time.Now().UTC()
	if clock != nil {
		anchor = clock().UTC()
	}

	var days int
	switch trimmedRange {
	case "7 days":
		days = 7
	case "30 days":
		days = 30
	default:
		return domain.MonthlyPeriod{}, fmt.Errorf("unsupported graph time range %q", trimmedRange)
	}

	endExclusive := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
	return domain.MonthlyPeriod{
		StartAt:      endExclusive.AddDate(0, 0, -days),
		EndExclusive: endExclusive,
	}, nil
}

func resolveBindingPeriod(month string, clock func() time.Time) (domain.MonthlyPeriod, error) {
	trimmed := strings.TrimSpace(month)
	if trimmed == "" {
		anchor := time.Now().UTC()
		if clock != nil {
			anchor = clock().UTC()
		}
		return domain.NewMonthlyPeriod(anchor)
	}

	parsed, err := time.Parse(dashboardMonthLayout, trimmed)
	if err != nil {
		return domain.MonthlyPeriod{}, fmt.Errorf("parse month %q: %w", trimmed, err)
	}
	return domain.NewMonthlyPeriod(parsed.UTC())
}
