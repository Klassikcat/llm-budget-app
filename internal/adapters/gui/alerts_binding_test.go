package gui

import (
	"context"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
)

func TestAlertsBindingLoadAlerts(t *testing.T) {
	period := mustMonthlyPeriod(t, 2026, time.April)
	store := mustDashboardStore(t)
	defer store.Close()

	alert := mustAlertEvent(t, domain.AlertEvent{
		AlertID:          "alert-1",
		Kind:             domain.AlertKindBudgetThreshold,
		Severity:         domain.AlertSeverityWarning,
		TriggeredAt:      period.StartAt.Add(12 * time.Hour),
		Period:           period,
		BudgetID:         "budget-1",
		CurrentSpendUSD:  80,
		LimitUSD:         100,
		ThresholdPercent: 0.8,
	})
	if err := store.UpsertAlerts(context.Background(), []domain.AlertEvent{alert}); err != nil {
		t.Fatalf("UpsertAlerts() error = %v", err)
	}

	binding := NewAlertsBinding(store)
	binding.startup(context.Background())
	response, err := binding.LoadAlerts("2026-04")
	if err != nil {
		t.Fatalf("LoadAlerts() error = %v", err)
	}

	if response.Empty || len(response.Items) != 1 {
		t.Fatalf("LoadAlerts() = %+v, want one alert", response)
	}
	if got := response.Items[0]; got.AlertID != alert.AlertID || got.Kind != "budget_threshold" || got.Period.Month != "2026-04" || got.ThresholdPercent != 0.8 {
		t.Fatalf("LoadAlerts().Items[0] = %+v, want mapped threshold alert", got)
	}
}

func mustAlertEvent(t *testing.T, alert domain.AlertEvent) domain.AlertEvent {
	t.Helper()
	validated, err := domain.NewAlertEvent(alert)
	if err != nil {
		t.Fatalf("NewAlertEvent() error = %v", err)
	}
	return validated
}
