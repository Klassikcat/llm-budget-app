package gui

import (
	"context"
	"fmt"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"llm-budget-tracker/internal/domain"
)

const desktopNotificationEvent = "llmbudget:desktop-notification"

type DesktopNotificationPayload struct {
	ID       string         `json:"id"`
	Title    string         `json:"title"`
	Subtitle string         `json:"subtitle,omitempty"`
	Body     string         `json:"body"`
	Kind     string         `json:"kind"`
	Severity string         `json:"severity"`
	Data     map[string]any `json:"data,omitempty"`
}

type WailsAlertNotifier struct {
	ctx  context.Context
	emit func(context.Context, string, ...interface{})
}

func NewWailsAlertNotifier() *WailsAlertNotifier {
	return &WailsAlertNotifier{emit: runtime.EventsEmit}
}

func (n *WailsAlertNotifier) startup(ctx context.Context) {
	if n == nil {
		return
	}
	n.ctx = ctx
}

func (n *WailsAlertNotifier) NotifyAlert(ctx context.Context, alert domain.AlertEvent) error {
	if n == nil {
		return fmt.Errorf("wails notifier is not initialized")
	}
	if ctx == nil {
		ctx = n.ctx
	}
	if ctx == nil {
		return fmt.Errorf("wails notifier requires startup context")
	}
	if n.emit == nil {
		return fmt.Errorf("wails notifier emit function is not configured")
	}

	n.emit(ctx, desktopNotificationEvent, DesktopNotificationPayload{
		ID:       alert.AlertID,
		Title:    notificationTitle(alert),
		Subtitle: notificationSubtitle(alert),
		Body:     notificationBody(alert),
		Kind:     string(alert.Kind),
		Severity: string(alert.Severity),
		Data: map[string]any{
			"budgetId":   alert.BudgetID,
			"forecastId": alert.ForecastID,
			"insightId":  alert.InsightID,
		},
	})

	return nil
}

func notificationTitle(alert domain.AlertEvent) string {
	switch alert.Kind {
	case domain.AlertKindBudgetThreshold:
		return fmt.Sprintf("Budget threshold reached (%s)", strings.ToUpper(string(alert.Severity)))
	case domain.AlertKindBudgetOverrun:
		return "Budget exceeded"
	case domain.AlertKindForecastOverrun:
		return "Forecast overrun risk"
	case domain.AlertKindInsightDetected:
		return "New budget insight"
	default:
		return "LLM Budget Tracker alert"
	}
}

func notificationSubtitle(alert domain.AlertEvent) string {
	if strings.TrimSpace(alert.BudgetID) != "" {
		return fmt.Sprintf("Budget: %s", alert.BudgetID)
	}
	if strings.TrimSpace(alert.ForecastID) != "" {
		return fmt.Sprintf("Forecast: %s", alert.ForecastID)
	}
	if strings.TrimSpace(alert.InsightID) != "" {
		return fmt.Sprintf("Insight: %s", alert.InsightID)
	}
	return ""
}

func notificationBody(alert domain.AlertEvent) string {
	switch alert.Kind {
	case domain.AlertKindBudgetThreshold:
		return fmt.Sprintf("Spend $%.2f of $%.2f crossed the %.0f%% threshold.", alert.CurrentSpendUSD, alert.LimitUSD, alert.ThresholdPercent*100)
	case domain.AlertKindBudgetOverrun:
		return fmt.Sprintf("Spend is now $%.2f against a $%.2f limit.", alert.CurrentSpendUSD, alert.LimitUSD)
	case domain.AlertKindForecastOverrun:
		return fmt.Sprintf("Forecast projects $%.2f against a $%.2f limit.", alert.CurrentSpendUSD, alert.LimitUSD)
	case domain.AlertKindInsightDetected:
		return "A new privacy-safe insight requires review."
	default:
		return "A new budget alert is available."
	}
}
