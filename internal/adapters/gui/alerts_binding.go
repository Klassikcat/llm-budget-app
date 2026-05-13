package gui

import (
	"context"
	"fmt"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

type alertListerBinding interface {
	ListAlerts(ctx context.Context, filter ports.AlertFilter) ([]domain.AlertEvent, error)
}

type AlertsBinding struct {
	alerts alertListerBinding
	ctx    context.Context
	clock  func() time.Time
}

func NewAlertsBinding(alerts alertListerBinding) *AlertsBinding {
	return &AlertsBinding{
		alerts: alerts,
		clock:  func() time.Time { return time.Now().UTC() },
	}
}

func (b *AlertsBinding) startup(ctx context.Context) {
	if b == nil {
		return
	}
	b.ctx = ctx
}

func (b *AlertsBinding) LoadAlerts(month string) (AlertListResponse, error) {
	if b == nil || b.alerts == nil {
		return AlertListResponse{}, fmt.Errorf("alert repository is not initialized")
	}

	period, err := resolveBindingPeriod(month, b.clock)
	if err != nil {
		return AlertListResponse{}, err
	}

	items, err := b.alerts.ListAlerts(b.context(), ports.AlertFilter{Period: &period})
	if err != nil {
		return AlertListResponse{}, err
	}
	return toAlertListResponse(items), nil
}

func (b *AlertsBinding) context() context.Context {
	if b != nil && b.ctx != nil {
		return b.ctx
	}
	return context.Background()
}

type AlertListResponse struct {
	Items []AlertResponse `json:"items"`
	Empty bool            `json:"empty"`
}

type AlertResponse struct {
	AlertID          string                  `json:"alertId"`
	Kind             string                  `json:"kind"`
	Severity         string                  `json:"severity"`
	TriggeredAt      string                  `json:"triggeredAt"`
	Period           DashboardPeriodResponse `json:"period"`
	BudgetID         string                  `json:"budgetId"`
	ForecastID       string                  `json:"forecastId"`
	InsightID        string                  `json:"insightId"`
	DetectorCategory string                  `json:"detectorCategory"`
	CurrentSpendUSD  float64                 `json:"currentSpendUsd"`
	LimitUSD         float64                 `json:"limitUsd"`
	ThresholdPercent float64                 `json:"thresholdPercent"`
}

func toAlertListResponse(alerts []domain.AlertEvent) AlertListResponse {
	items := make([]AlertResponse, 0, len(alerts))
	for _, alert := range alerts {
		items = append(items, AlertResponse{
			AlertID:          alert.AlertID,
			Kind:             string(alert.Kind),
			Severity:         string(alert.Severity),
			TriggeredAt:      formatDashboardTime(alert.TriggeredAt),
			Period:           toDashboardPeriodResponse(alert.Period),
			BudgetID:         alert.BudgetID,
			ForecastID:       alert.ForecastID,
			InsightID:        alert.InsightID,
			DetectorCategory: string(alert.DetectorCategory),
			CurrentSpendUSD:  alert.CurrentSpendUSD,
			LimitUSD:         alert.LimitUSD,
			ThresholdPercent: alert.ThresholdPercent,
		})
	}
	return AlertListResponse{Items: items, Empty: len(items) == 0}
}
