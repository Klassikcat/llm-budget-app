package service

import (
	"context"
	"fmt"
	"slices"
	"time"

	"llm-budget-tracker/internal/config"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

type BudgetMonitorResult struct {
	VariableSpendUSD     float64
	SubscriptionSpendUSD float64
	TotalSpendUSD        float64
	Forecasts            []domain.ForecastSnapshot
	Alerts               []domain.AlertEvent
	Budgets              []domain.MonthlyBudget
	States               []domain.BudgetState
}

type BudgetMonitorService struct {
	settings         config.Settings
	budgetRepo       ports.BudgetRepository
	usageRepo        ports.UsageEntryRepository
	subscriptionRepo ports.SubscriptionRepository
	forecastRepo     ports.ForecastRepository
	alertRepo        ports.AlertRepository
	notifier         ports.AlertNotifier
	clock            func() time.Time
}

func NewBudgetMonitorService(settings config.Settings, budgetRepo ports.BudgetRepository, usageRepo ports.UsageEntryRepository, subscriptionRepo ports.SubscriptionRepository, forecastRepo ports.ForecastRepository, alertRepo ports.AlertRepository, notifier ports.AlertNotifier) *BudgetMonitorService {
	return &BudgetMonitorService{
		settings:         settings,
		budgetRepo:       budgetRepo,
		usageRepo:        usageRepo,
		subscriptionRepo: subscriptionRepo,
		forecastRepo:     forecastRepo,
		alertRepo:        alertRepo,
		notifier:         notifier,
		clock:            func() time.Time { return time.Now().UTC() },
	}
}

func (s *BudgetMonitorService) MonitorPeriod(ctx context.Context, period domain.MonthlyPeriod) (BudgetMonitorResult, error) {
	if s == nil || s.budgetRepo == nil {
		return BudgetMonitorResult{}, errBudgetRepositoryRequired
	}
	if s.usageRepo == nil {
		return BudgetMonitorResult{}, errUsageEntryRepositoryRequired
	}
	if s.subscriptionRepo == nil {
		return BudgetMonitorResult{}, errSubscriptionRepoRequired
	}
	if s.forecastRepo == nil {
		return BudgetMonitorResult{}, errForecastRepositoryRequired
	}
	if s.alertRepo == nil {
		return BudgetMonitorResult{}, errAlertRepositoryRequired
	}

	budgets, err := s.budgetRepo.ListMonthlyBudgets(ctx, ports.BudgetFilter{Period: &period})
	if err != nil {
		return BudgetMonitorResult{}, err
	}

	rollupService := NewSubscriptionService(s.subscriptionRepo, s.usageRepo)
	rollup, err := rollupService.RollupMonthlySpend(ctx, period)
	if err != nil {
		return BudgetMonitorResult{}, err
	}

	forecasts := make([]domain.ForecastSnapshot, 0, len(budgets))
	alerts := make([]domain.AlertEvent, 0)
	states := make([]domain.BudgetState, 0, len(budgets))

	for _, budget := range budgets {
		budgetUsageEntries, budgetSubscriptionFees := filterBudgetInputs(budget, rollup.UsageEntries, rollup.SubscriptionFees)
		actualSpend := sumUsageSpend(budgetUsageEntries) + sumSubscriptionSpend(budgetSubscriptionFees)
		asOf := forecastAsOf(period, budgetUsageEntries, budgetSubscriptionFees, s.clock)
		forecastSpend := projectMonthlySpend(period, asOf, actualSpend)

		forecast, err := newForecastSnapshot(budget, period, asOf, actualSpend, forecastSpend)
		if err != nil {
			return BudgetMonitorResult{}, err
		}
		forecasts = append(forecasts, forecast)

		status, err := budget.EvaluateSpend(actualSpend)
		if err != nil {
			return BudgetMonitorResult{}, err
		}

		state, err := newBudgetState(budget, forecast, status, asOf)
		if err != nil {
			return BudgetMonitorResult{}, err
		}
		states = append(states, state)

		previousState, found, err := s.budgetRepo.GetBudgetState(ctx, budget.BudgetID, period)
		if err != nil {
			return BudgetMonitorResult{}, err
		}

		triggeredAlerts, err := buildBudgetAlerts(budget, status, forecast, previousState, found, asOf)
		if err != nil {
			return BudgetMonitorResult{}, err
		}
		alerts = append(alerts, triggeredAlerts...)
	}

	if len(forecasts) > 0 {
		if err := s.forecastRepo.UpsertForecastSnapshots(ctx, forecasts); err != nil {
			return BudgetMonitorResult{}, err
		}
	}

	if len(states) > 0 {
		if err := s.budgetRepo.UpsertBudgetStates(ctx, states); err != nil {
			return BudgetMonitorResult{}, err
		}
	}

	if len(alerts) > 0 {
		if err := s.alertRepo.UpsertAlerts(ctx, alerts); err != nil {
			return BudgetMonitorResult{}, err
		}
		if shouldDispatchNotifications(s.settings) && s.notifier != nil {
			for _, alert := range alerts {
				if !shouldDispatchAlert(s.settings, alert) {
					continue
				}
				if err := s.notifier.NotifyAlert(ctx, alert); err != nil {
					return BudgetMonitorResult{}, err
				}
			}
		}
	}

	return BudgetMonitorResult{
		VariableSpendUSD:     rollup.VariableSpendUSD,
		SubscriptionSpendUSD: rollup.SubscriptionSpendUSD,
		TotalSpendUSD:        rollup.TotalSpendUSD,
		Forecasts:            forecasts,
		Alerts:               alerts,
		Budgets:              budgets,
		States:               states,
	}, nil
}

func shouldDispatchNotifications(settings config.Settings) bool {
	return settings.Notifications.BudgetWarnings || settings.Notifications.ForecastWarnings
}

func shouldDispatchAlert(settings config.Settings, alert domain.AlertEvent) bool {
	switch alert.Kind {
	case domain.AlertKindForecastOverrun:
		return settings.Notifications.ForecastWarnings
	default:
		return settings.Notifications.BudgetWarnings
	}
}

func sumUsageSpend(entries []domain.UsageEntry) float64 {
	total := 0.0
	for _, entry := range entries {
		total += entry.CostBreakdown.TotalUSD
	}
	return total
}

func sumVariableUsageSpend(entries []domain.UsageEntry) float64 {
	total := 0.0
	for _, entry := range entries {
		if entry.Source == domain.UsageSourceSubscription {
			continue
		}
		total += entry.CostBreakdown.TotalUSD
	}
	return total
}

func sumSubscriptionSpend(fees []domain.SubscriptionFee) float64 {
	total := 0.0
	for _, fee := range fees {
		total += fee.FeeUSD
	}
	return total
}

func filterBudgetInputs(budget domain.MonthlyBudget, usageEntries []domain.UsageEntry, subscriptionFees []domain.SubscriptionFee) ([]domain.UsageEntry, []domain.SubscriptionFee) {
	filteredEntries := make([]domain.UsageEntry, 0, len(usageEntries))
	for _, entry := range usageEntries {
		if budget.Provider != "" && entry.Provider != budget.Provider {
			continue
		}
		if budget.ProjectHash != "" && entry.Metadata["project_hash"] != budget.ProjectHash {
			continue
		}
		filteredEntries = append(filteredEntries, entry)
	}

	filteredFees := make([]domain.SubscriptionFee, 0, len(subscriptionFees))
	for _, fee := range subscriptionFees {
		if budget.Provider != "" && fee.Provider != budget.Provider {
			continue
		}
		if budget.ProjectHash != "" {
			continue
		}
		filteredFees = append(filteredFees, fee)
	}

	return filteredEntries, filteredFees
}

func forecastAsOf(period domain.MonthlyPeriod, usageEntries []domain.UsageEntry, subscriptionFees []domain.SubscriptionFee, clock func() time.Time) time.Time {
	asOf := period.StartAt
	for _, entry := range usageEntries {
		if entry.OccurredAt.After(asOf) {
			asOf = entry.OccurredAt
		}
	}
	for _, fee := range subscriptionFees {
		if fee.ChargedAt.After(asOf) {
			asOf = fee.ChargedAt
		}
	}

	if asOf.Equal(period.StartAt) && clock != nil {
		current := clock().UTC()
		if period.Contains(current) {
			return period.StartAt
		}
	}
	if !period.Contains(asOf) {
		if asOf.Before(period.StartAt) {
			return period.StartAt
		}
		return period.EndExclusive.Add(-time.Nanosecond)
	}
	return asOf
}

func projectMonthlySpend(period domain.MonthlyPeriod, asOf time.Time, currentSpend float64) float64 {
	if currentSpend == 0 {
		return 0
	}
	if !period.Contains(asOf) {
		return currentSpend
	}

	observedDays := int(asOf.Sub(period.StartAt).Hours()/24) + 1
	if observedDays < 1 {
		observedDays = 1
	}

	totalDays := int(period.EndExclusive.Sub(period.StartAt).Hours() / 24)
	if totalDays < observedDays {
		return currentSpend
	}

	return currentSpend * float64(totalDays) / float64(observedDays)
}

func newForecastSnapshot(budget domain.MonthlyBudget, period domain.MonthlyPeriod, generatedAt time.Time, actualSpend, forecastSpend float64) (domain.ForecastSnapshot, error) {
	observedDays := 1
	if period.Contains(generatedAt) {
		observedDays = int(generatedAt.Sub(period.StartAt).Hours()/24) + 1
	}
	if observedDays < 1 {
		observedDays = 1
	}
	remainingDays := int(period.EndExclusive.Sub(period.StartAt).Hours()/24) - observedDays
	if remainingDays < 0 {
		remainingDays = 0
	}

	return domain.NewForecastSnapshot(domain.ForecastSnapshot{
		ForecastID:        fmt.Sprintf("%s:forecast", budget.BudgetID),
		Period:            budget.Period,
		GeneratedAt:       generatedAt,
		ActualSpendUSD:    actualSpend,
		ForecastSpendUSD:  forecastSpend,
		BudgetLimitUSD:    budget.LimitUSD,
		ObservedDayCount:  observedDays,
		RemainingDayCount: remainingDays,
	})
}

func newBudgetState(budget domain.MonthlyBudget, forecast domain.ForecastSnapshot, status domain.BudgetStatus, updatedAt time.Time) (domain.BudgetState, error) {
	percents := make([]float64, 0, len(status.TriggeredThresholds))
	for _, threshold := range status.TriggeredThresholds {
		percents = append(percents, threshold.Percent)
	}

	return domain.NewBudgetState(domain.BudgetState{
		BudgetID:                   budget.BudgetID,
		Period:                     budget.Period,
		CurrentSpendUSD:            forecast.ActualSpendUSD,
		ForecastSpendUSD:           forecast.ForecastSpendUSD,
		TriggeredThresholdPercents: percents,
		BudgetOverrunActive:        status.IsOverrun,
		ForecastOverrunActive:      forecast.ProjectedOverrunUSD > 0,
		UpdatedAt:                  updatedAt,
	})
}

func buildBudgetAlerts(budget domain.MonthlyBudget, status domain.BudgetStatus, forecast domain.ForecastSnapshot, previous domain.BudgetState, hasPrevious bool, triggeredAt time.Time) ([]domain.AlertEvent, error) {
	alerts := make([]domain.AlertEvent, 0, len(status.TriggeredThresholds)+2)
	previousThresholds := map[float64]struct{}{}
	if hasPrevious {
		for _, percent := range previous.TriggeredThresholdPercents {
			previousThresholds[percent] = struct{}{}
		}
	}

	for _, threshold := range status.TriggeredThresholds {
		if _, seen := previousThresholds[threshold.Percent]; seen {
			continue
		}
		alert, err := domain.NewAlertEvent(domain.AlertEvent{
			AlertID:          fmt.Sprintf("%s:threshold:%0.2f:%s", budget.BudgetID, threshold.Percent, triggeredAt.Format(time.RFC3339Nano)),
			Kind:             domain.AlertKindBudgetThreshold,
			Severity:         threshold.Severity,
			TriggeredAt:      triggeredAt,
			Period:           budget.Period,
			BudgetID:         budget.BudgetID,
			CurrentSpendUSD:  forecast.ActualSpendUSD,
			LimitUSD:         budget.LimitUSD,
			ThresholdPercent: threshold.Percent,
		})
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	if status.IsOverrun && (!hasPrevious || !previous.BudgetOverrunActive) {
		alert, err := domain.NewAlertEvent(domain.AlertEvent{
			AlertID:         fmt.Sprintf("%s:overrun:%s", budget.BudgetID, triggeredAt.Format(time.RFC3339Nano)),
			Kind:            domain.AlertKindBudgetOverrun,
			Severity:        domain.AlertSeverityCritical,
			TriggeredAt:     triggeredAt,
			Period:          budget.Period,
			BudgetID:        budget.BudgetID,
			CurrentSpendUSD: forecast.ActualSpendUSD,
			LimitUSD:        budget.LimitUSD,
		})
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	if forecast.ProjectedOverrunUSD > 0 && (!hasPrevious || !previous.ForecastOverrunActive) {
		alert, err := domain.NewAlertEvent(domain.AlertEvent{
			AlertID:         fmt.Sprintf("%s:forecast:%s", budget.BudgetID, triggeredAt.Format(time.RFC3339Nano)),
			Kind:            domain.AlertKindForecastOverrun,
			Severity:        domain.AlertSeverityWarning,
			TriggeredAt:     triggeredAt,
			Period:          budget.Period,
			ForecastID:      forecast.ForecastID,
			CurrentSpendUSD: forecast.ForecastSpendUSD,
			LimitUSD:        budget.LimitUSD,
		})
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	slices.SortFunc(alerts, func(a, b domain.AlertEvent) int {
		if a.TriggeredAt.Before(b.TriggeredAt) {
			return -1
		}
		if a.TriggeredAt.After(b.TriggeredAt) {
			return 1
		}
		if a.AlertID < b.AlertID {
			return -1
		}
		if a.AlertID > b.AlertID {
			return 1
		}
		return 0
	})

	return alerts, nil
}
