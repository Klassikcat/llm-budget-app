package ports

import (
	"context"
	"time"

	"llm-budget-tracker/internal/domain"
)

type UsageFilter struct {
	Period    *domain.MonthlyPeriod
	Provider  domain.ProviderName
	Project   string
	Agent     string
	SessionID string
}

type SessionFilter struct {
	Period    *domain.MonthlyPeriod
	Provider  domain.ProviderName
	Project   string
	Agent     string
	SessionID string
}

type BudgetFilter struct {
	Period   *domain.MonthlyPeriod
	Provider domain.ProviderName
}

type AlertFilter struct {
	Period   *domain.MonthlyPeriod
	BudgetID string
	Kind     domain.AlertKind
}

type SubscriptionFilter struct {
	Period         *domain.MonthlyPeriod
	Provider       domain.ProviderName
	SubscriptionID string
	PlanCode       string
	Active         *bool
}

type IngestionCheckpoint struct {
	SourceID     string
	Path         string
	FileIdentity string
	LastMarker   string
	Offset       int64
	UpdatedAt    time.Time
}

type UsageEntryRepository interface {
	UpsertUsageEntries(ctx context.Context, entries []domain.UsageEntry) error
	ListUsageEntries(ctx context.Context, filter UsageFilter) ([]domain.UsageEntry, error)
}

type SessionRepository interface {
	UpsertSessions(ctx context.Context, sessions []domain.SessionSummary) error
	ListSessions(ctx context.Context, filter SessionFilter) ([]domain.SessionSummary, error)
}

type SubscriptionRepository interface {
	UpsertSubscriptions(ctx context.Context, subscriptions []domain.Subscription) error
	ListSubscriptions(ctx context.Context, filter SubscriptionFilter) ([]domain.Subscription, error)
	DisableSubscription(ctx context.Context, subscriptionID string, disabledAt time.Time) error
	DeleteSubscription(ctx context.Context, subscriptionID string) error
	UpsertSubscriptionFees(ctx context.Context, fees []domain.SubscriptionFee) error
	ListSubscriptionFees(ctx context.Context, period domain.MonthlyPeriod) ([]domain.SubscriptionFee, error)
}

type BudgetRepository interface {
	UpsertMonthlyBudgets(ctx context.Context, budgets []domain.MonthlyBudget) error
	ListMonthlyBudgets(ctx context.Context, filter BudgetFilter) ([]domain.MonthlyBudget, error)
	UpsertBudgetStates(ctx context.Context, states []domain.BudgetState) error
	GetBudgetState(ctx context.Context, budgetID string, period domain.MonthlyPeriod) (domain.BudgetState, bool, error)
}

type ForecastRepository interface {
	UpsertForecastSnapshots(ctx context.Context, forecasts []domain.ForecastSnapshot) error
	ListForecastSnapshots(ctx context.Context, period domain.MonthlyPeriod) ([]domain.ForecastSnapshot, error)
}

type InsightRepository interface {
	UpsertInsights(ctx context.Context, insights []domain.Insight) error
	ListInsights(ctx context.Context, period domain.MonthlyPeriod) ([]domain.Insight, error)
}

type AlertRepository interface {
	UpsertAlerts(ctx context.Context, alerts []domain.AlertEvent) error
	ListAlerts(ctx context.Context, filter AlertFilter) ([]domain.AlertEvent, error)
}

type CheckpointRepository interface {
	LoadCheckpoint(ctx context.Context, sourceID string) (IngestionCheckpoint, error)
	SaveCheckpoint(ctx context.Context, checkpoint IngestionCheckpoint) error
}
