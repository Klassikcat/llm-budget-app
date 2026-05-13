package gui

import (
	"context"
	"fmt"
	"time"

	"llm-budget-tracker/internal/domain"
)

type wasteSummaryQuerier interface {
	QueryWasteSummary(ctx context.Context, period domain.MonthlyPeriod) (domain.WasteSummary, error)
}

type insightListerBinding interface {
	ListInsights(ctx context.Context, period domain.MonthlyPeriod) ([]domain.Insight, error)
}

type InsightsBinding struct {
	wasteSummary wasteSummaryQuerier
	insights     insightListerBinding
	ctx          context.Context
	clock        func() time.Time
}

func NewInsightsBinding(wasteSummary wasteSummaryQuerier, insights insightListerBinding) *InsightsBinding {
	return &InsightsBinding{
		wasteSummary: wasteSummary,
		insights:     insights,
		clock:        func() time.Time { return time.Now().UTC() },
	}
}

func (b *InsightsBinding) startup(ctx context.Context) {
	if b == nil {
		return
	}
	b.ctx = ctx
}

func (b *InsightsBinding) LoadWasteSummary(month string) (WasteSummaryResponse, error) {
	if b == nil || b.wasteSummary == nil {
		return WasteSummaryResponse{}, fmt.Errorf("waste summary service is not initialized")
	}

	period, err := resolveBindingPeriod(month, b.clock)
	if err != nil {
		return WasteSummaryResponse{}, err
	}

	summary, err := b.wasteSummary.QueryWasteSummary(b.context(), period)
	if err != nil {
		return WasteSummaryResponse{}, err
	}
	return toWasteSummaryResponse(summary), nil
}

func (b *InsightsBinding) LoadInsights(month string) (InsightListResponse, error) {
	if b == nil || b.insights == nil {
		return InsightListResponse{}, fmt.Errorf("insight repository is not initialized")
	}

	period, err := resolveBindingPeriod(month, b.clock)
	if err != nil {
		return InsightListResponse{}, err
	}

	items, err := b.insights.ListInsights(b.context(), period)
	if err != nil {
		return InsightListResponse{}, err
	}
	return toInsightListResponse(items), nil
}

func (b *InsightsBinding) context() context.Context {
	if b != nil && b.ctx != nil {
		return b.ctx
	}
	return context.Background()
}

type WasteSummaryResponse struct {
	Period                    DashboardPeriodResponse   `json:"period"`
	TotalWasteCostUSD         float64                   `json:"totalWasteCostUsd"`
	TotalSpendCostUSD         float64                   `json:"totalSpendCostUsd"`
	WastePercent              float64                   `json:"wastePercent"`
	WeeklyWasteCostUSD        float64                   `json:"weeklyWasteCostUsd"`
	MonthlyWasteCostUSD       float64                   `json:"monthlyWasteCostUsd"`
	ProjectedMonthEndWasteUSD float64                   `json:"projectedMonthEndWasteUsd"`
	ByDetector                []WasteByDetectorResponse `json:"byDetector"`
	TopCauses                 []WasteByDetectorResponse `json:"topCauses"`
	DailyTrend                []WasteTrendPointResponse `json:"dailyTrend"`
	GeneratedAt               string                    `json:"generatedAt"`
}

type WasteByDetectorResponse struct {
	Category          string  `json:"category"`
	AttributedCostUSD float64 `json:"attributedCostUsd"`
	InsightCount      int     `json:"insightCount"`
}

type WasteTrendPointResponse struct {
	Day          string  `json:"day"`
	WasteCostUSD float64 `json:"wasteCostUsd"`
}

type InsightListResponse struct {
	Items []InsightResponse `json:"items"`
	Empty bool              `json:"empty"`
}

type InsightResponse struct {
	InsightID  string                  `json:"insightId"`
	Category   string                  `json:"category"`
	Severity   string                  `json:"severity"`
	DetectedAt string                  `json:"detectedAt"`
	Period     DashboardPeriodResponse `json:"period"`
	Payload    InsightPayloadResponse  `json:"payload"`
}

type InsightPayloadResponse struct {
	SessionIDs    []string                `json:"sessionIds"`
	UsageEntryIDs []string                `json:"usageEntryIds"`
	Hashes        []domain.InsightHash    `json:"hashes"`
	Counts        []domain.InsightCount   `json:"counts"`
	Metrics       []InsightMetricResponse `json:"metrics"`
}

type InsightMetricResponse struct {
	Key   string  `json:"key"`
	Unit  string  `json:"unit"`
	Value float64 `json:"value"`
}

func toWasteSummaryResponse(summary domain.WasteSummary) WasteSummaryResponse {
	return WasteSummaryResponse{
		Period:                    toDashboardPeriodResponse(summary.Period),
		TotalWasteCostUSD:         summary.TotalWasteCostUSD,
		TotalSpendCostUSD:         summary.TotalSpendCostUSD,
		WastePercent:              summary.WastePercent,
		WeeklyWasteCostUSD:        summary.WeeklyWasteCostUSD,
		MonthlyWasteCostUSD:       summary.MonthlyWasteCostUSD,
		ProjectedMonthEndWasteUSD: summary.ProjectedMonthEndWasteUSD,
		ByDetector:                toWasteByDetectorResponses(summary.ByDetector),
		TopCauses:                 toWasteByDetectorResponses(summary.TopCauses),
		DailyTrend:                toWasteTrendPointResponses(summary.DailyTrend),
		GeneratedAt:               formatDashboardTime(summary.GeneratedAt),
	}
}

func toInsightListResponse(insights []domain.Insight) InsightListResponse {
	items := make([]InsightResponse, 0, len(insights))
	for _, insight := range insights {
		items = append(items, toInsightResponse(insight))
	}
	return InsightListResponse{Items: items, Empty: len(items) == 0}
}

func toInsightResponse(insight domain.Insight) InsightResponse {
	return InsightResponse{
		InsightID:  insight.InsightID,
		Category:   string(insight.Category),
		Severity:   string(insight.Severity),
		DetectedAt: formatDashboardTime(insight.DetectedAt),
		Period:     toDashboardPeriodResponse(insight.Period),
		Payload:    toInsightPayloadResponse(insight.Payload),
	}
}

func toInsightPayloadResponse(payload domain.InsightPayload) InsightPayloadResponse {
	metrics := make([]InsightMetricResponse, 0, len(payload.Metrics))
	for _, metric := range payload.Metrics {
		metrics = append(metrics, InsightMetricResponse{Key: metric.Key, Unit: string(metric.Unit), Value: metric.Value})
	}
	return InsightPayloadResponse{
		SessionIDs:    append([]string(nil), payload.SessionIDs...),
		UsageEntryIDs: append([]string(nil), payload.UsageEntryIDs...),
		Hashes:        append([]domain.InsightHash(nil), payload.Hashes...),
		Counts:        append([]domain.InsightCount(nil), payload.Counts...),
		Metrics:       metrics,
	}
}

func toWasteByDetectorResponses(items []domain.WasteByDetector) []WasteByDetectorResponse {
	responses := make([]WasteByDetectorResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, WasteByDetectorResponse{Category: string(item.Category), AttributedCostUSD: item.AttributedCostUSD, InsightCount: item.InsightCount})
	}
	return responses
}

func toWasteTrendPointResponses(items []domain.WasteTrendPoint) []WasteTrendPointResponse {
	responses := make([]WasteTrendPointResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, WasteTrendPointResponse{Day: formatDashboardTime(item.Day), WasteCostUSD: item.WasteCostUSD})
	}
	return responses
}

func toDashboardPeriodResponse(period domain.MonthlyPeriod) DashboardPeriodResponse {
	return DashboardPeriodResponse{
		Month:        period.StartAt.Format(dashboardMonthLayout),
		StartAt:      formatDashboardTime(period.StartAt),
		EndExclusive: formatDashboardTime(period.EndExclusive),
		Currency:     "USD",
	}
}
