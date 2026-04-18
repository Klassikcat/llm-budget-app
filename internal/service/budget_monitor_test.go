package service

import (
	"context"
	"testing"
	"time"

	"llm-budget-tracker/internal/config"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestMonthlyRollupAndForecast(t *testing.T) {
	period := mustMonthlyPeriod(t, 2026, time.March)
	usageRepo := &memoryUsageRepository{
		entries: []domain.UsageEntry{
			mustUsageEntry(t, "manual-openai", domain.UsageSourceManualAPI, domain.ProviderOpenAI, domain.BillingModeDirectAPI, time.Date(2026, time.March, 2, 10, 0, 0, 0, time.UTC), 12),
			mustUsageEntry(t, "openrouter", domain.UsageSourceOpenRouter, domain.ProviderOpenRouter, domain.BillingModeOpenRouter, time.Date(2026, time.March, 5, 11, 0, 0, 0, time.UTC), 18),
			mustUsageEntry(t, "cli-session", domain.UsageSourceCLISession, domain.ProviderClaude, domain.BillingModeBYOK, time.Date(2026, time.March, 10, 9, 0, 0, 0, time.UTC), 30),
		},
	}
	subscriptionRepo := &memorySubscriptionRepository{
		subscriptions: map[string]domain.Subscription{
			"sub-chatgpt": mustSubscription(t, domain.Subscription{
				SubscriptionID: "sub-chatgpt",
				Provider:       domain.ProviderOpenAI,
				PlanCode:       "chatgpt-plus",
				PlanName:       "ChatGPT Plus",
				RenewalDay:     15,
				StartsAt:       time.Date(2026, time.January, 15, 8, 0, 0, 0, time.UTC),
				FeeUSD:         30,
				IsActive:       true,
			}),
		},
	}
	budgetRepo := &memoryBudgetRepository{
		budgets: []domain.MonthlyBudget{
			mustBudget(t, domain.MonthlyBudget{
				BudgetID: "budget-total",
				Name:     "Monthly Total",
				Period:   period,
				LimitUSD: 100,
				Thresholds: []domain.BudgetThreshold{
					mustBudgetThreshold(t, domain.AlertSeverityWarning, 0.8),
				},
			}),
		},
	}
	forecastRepo := &memoryForecastRepository{}
	alertRepo := &memoryAlertRepository{}
	notifier := &memoryAlertNotifier{}

	service := NewBudgetMonitorService(config.DefaultSettings(), budgetRepo, usageRepo, subscriptionRepo, forecastRepo, alertRepo, notifier)

	result, err := service.MonitorPeriod(context.Background(), period)
	if err != nil {
		t.Fatalf("MonitorPeriod() error = %v", err)
	}

	if got, want := result.VariableSpendUSD, 60.0; got != want {
		t.Fatalf("VariableSpendUSD = %v, want %v", got, want)
	}
	if got, want := result.SubscriptionSpendUSD, 30.0; got != want {
		t.Fatalf("SubscriptionSpendUSD = %v, want %v", got, want)
	}
	if got, want := result.TotalSpendUSD, 90.0; got != want {
		t.Fatalf("TotalSpendUSD = %v, want %v", got, want)
	}

	if got := len(result.Forecasts); got != 1 {
		t.Fatalf("len(result.Forecasts) = %d, want 1", got)
	}
	forecast := result.Forecasts[0]
	if got, want := forecast.ActualSpendUSD, 90.0; got != want {
		t.Fatalf("forecast.ActualSpendUSD = %v, want %v", got, want)
	}
	if got, want := forecast.ForecastSpendUSD, 186.0; got != want {
		t.Fatalf("forecast.ForecastSpendUSD = %v, want %v", got, want)
	}
	if got, want := forecast.ObservedDayCount, 15; got != want {
		t.Fatalf("forecast.ObservedDayCount = %d, want %d", got, want)
	}

	if got := len(budgetRepo.states); got != 1 {
		t.Fatalf("persisted budget state count = %d, want 1", got)
	}
	if got := len(forecastRepo.forecasts); got != 1 {
		t.Fatalf("persisted forecast count = %d, want 1", got)
	}
	if got := len(alertRepo.alerts); got != 2 {
		t.Fatalf("persisted alert count = %d, want 2", got)
	}
	if got := len(notifier.alerts); got != 2 {
		t.Fatalf("notifier alert count = %d, want 2", got)
	}
}

func TestBudgetAlertDeduplication(t *testing.T) {
	period := mustMonthlyPeriod(t, 2026, time.March)
	usageRepo := &memoryUsageRepository{
		entries: []domain.UsageEntry{
			mustUsageEntry(t, "threshold-crossing", domain.UsageSourceManualAPI, domain.ProviderOpenAI, domain.BillingModeDirectAPI, time.Date(2026, time.March, 31, 12, 0, 0, 0, time.UTC), 81),
		},
	}
	subscriptionRepo := &memorySubscriptionRepository{}
	budgetRepo := &memoryBudgetRepository{
		budgets: []domain.MonthlyBudget{
			mustBudget(t, domain.MonthlyBudget{
				BudgetID: "budget-dedup",
				Name:     "Warning Budget",
				Period:   period,
				LimitUSD: 100,
				Thresholds: []domain.BudgetThreshold{
					mustBudgetThreshold(t, domain.AlertSeverityWarning, 0.8),
				},
			}),
		},
	}
	forecastRepo := &memoryForecastRepository{}
	alertRepo := &memoryAlertRepository{}
	notifier := &memoryAlertNotifier{}

	settings := config.DefaultSettings()
	settings.Notifications.ForecastWarnings = false

	service := NewBudgetMonitorService(settings, budgetRepo, usageRepo, subscriptionRepo, forecastRepo, alertRepo, notifier)

	first, err := service.MonitorPeriod(context.Background(), period)
	if err != nil {
		t.Fatalf("first MonitorPeriod() error = %v", err)
	}
	second, err := service.MonitorPeriod(context.Background(), period)
	if err != nil {
		t.Fatalf("second MonitorPeriod() error = %v", err)
	}

	if got := len(first.Alerts); got != 1 {
		t.Fatalf("len(first.Alerts) = %d, want 1", got)
	}
	if got := len(second.Alerts); got != 0 {
		t.Fatalf("len(second.Alerts) = %d, want 0", got)
	}
	if got := len(alertRepo.alerts); got != 1 {
		t.Fatalf("persisted alert history count = %d, want 1", got)
	}
	if got := len(notifier.alerts); got != 1 {
		t.Fatalf("notifier alert count = %d, want 1", got)
	}
	state, ok := budgetRepo.stateFor("budget-dedup", period)
	if !ok {
		t.Fatal("budget state missing after monitor run")
	}
	if got, want := state.TriggeredThresholdPercents, []float64{0.8}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("state.TriggeredThresholdPercents = %v, want %v", got, want)
	}
}

type memoryBudgetRepository struct {
	budgets []domain.MonthlyBudget
	states  map[string]domain.BudgetState
}

func (r *memoryBudgetRepository) UpsertMonthlyBudgets(_ context.Context, budgets []domain.MonthlyBudget) error {
	r.budgets = budgets
	return nil
}

func (r *memoryBudgetRepository) ListMonthlyBudgets(_ context.Context, filter ports.BudgetFilter) ([]domain.MonthlyBudget, error) {
	items := make([]domain.MonthlyBudget, 0, len(r.budgets))
	for _, budget := range r.budgets {
		if filter.Period != nil && !budget.Period.StartAt.Equal(filter.Period.StartAt) {
			continue
		}
		if filter.Provider != "" && budget.Provider != filter.Provider {
			continue
		}
		items = append(items, budget)
	}
	return items, nil
}

func (r *memoryBudgetRepository) UpsertBudgetStates(_ context.Context, states []domain.BudgetState) error {
	if r.states == nil {
		r.states = make(map[string]domain.BudgetState)
	}
	for _, state := range states {
		r.states[stateKey(state.BudgetID, state.Period)] = state
	}
	return nil
}

func (r *memoryBudgetRepository) GetBudgetState(_ context.Context, budgetID string, period domain.MonthlyPeriod) (domain.BudgetState, bool, error) {
	state, ok := r.stateFor(budgetID, period)
	return state, ok, nil
}

func (r *memoryBudgetRepository) stateFor(budgetID string, period domain.MonthlyPeriod) (domain.BudgetState, bool) {
	if r.states == nil {
		return domain.BudgetState{}, false
	}
	state, ok := r.states[stateKey(budgetID, period)]
	return state, ok
}

type memoryForecastRepository struct {
	forecasts map[string]domain.ForecastSnapshot
}

func (r *memoryForecastRepository) UpsertForecastSnapshots(_ context.Context, forecasts []domain.ForecastSnapshot) error {
	if r.forecasts == nil {
		r.forecasts = make(map[string]domain.ForecastSnapshot)
	}
	for _, forecast := range forecasts {
		r.forecasts[forecast.ForecastID] = forecast
	}
	return nil
}

func (r *memoryForecastRepository) ListForecastSnapshots(_ context.Context, period domain.MonthlyPeriod) ([]domain.ForecastSnapshot, error) {
	items := make([]domain.ForecastSnapshot, 0)
	for _, forecast := range r.forecasts {
		if forecast.Period.StartAt.Equal(period.StartAt) {
			items = append(items, forecast)
		}
	}
	return items, nil
}

type memoryAlertRepository struct {
	alerts map[string]domain.AlertEvent
}

func (r *memoryAlertRepository) UpsertAlerts(_ context.Context, alerts []domain.AlertEvent) error {
	if r.alerts == nil {
		r.alerts = make(map[string]domain.AlertEvent)
	}
	for _, alert := range alerts {
		r.alerts[alert.AlertID] = alert
	}
	return nil
}

func (r *memoryAlertRepository) ListAlerts(_ context.Context, filter ports.AlertFilter) ([]domain.AlertEvent, error) {
	items := make([]domain.AlertEvent, 0)
	for _, alert := range r.alerts {
		if filter.Period != nil && !alert.Period.StartAt.Equal(filter.Period.StartAt) {
			continue
		}
		if filter.BudgetID != "" && alert.BudgetID != filter.BudgetID {
			continue
		}
		if filter.Kind != "" && alert.Kind != filter.Kind {
			continue
		}
		items = append(items, alert)
	}
	return items, nil
}

type memoryAlertNotifier struct {
	alerts []domain.AlertEvent
}

func (n *memoryAlertNotifier) NotifyAlert(_ context.Context, alert domain.AlertEvent) error {
	n.alerts = append(n.alerts, alert)
	return nil
}

func mustBudget(t *testing.T, budget domain.MonthlyBudget) domain.MonthlyBudget {
	t.Helper()

	normalized, err := domain.NewMonthlyBudget(budget)
	if err != nil {
		t.Fatalf("NewMonthlyBudget() error = %v", err)
	}

	return normalized
}

func mustBudgetThreshold(t *testing.T, severity domain.AlertSeverity, percent float64) domain.BudgetThreshold {
	t.Helper()

	threshold, err := domain.NewBudgetThreshold(severity, percent)
	if err != nil {
		t.Fatalf("NewBudgetThreshold() error = %v", err)
	}

	return threshold
}

func stateKey(budgetID string, period domain.MonthlyPeriod) string {
	return budgetID + ":" + period.StartAt.Format(time.RFC3339Nano)
}
