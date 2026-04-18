package service

import (
	"context"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestDashboardQueryServiceLoadAggregatesSummaries(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}

	usageRef, err := domain.NewModelPricingRef(domain.ProviderOpenAI, "gpt-5-mini", "gpt-5-mini")
	if err != nil {
		t.Fatalf("NewModelPricingRef() error = %v", err)
	}
	tokens, err := domain.NewTokenUsage(1200, 300, 40, 0)
	if err != nil {
		t.Fatalf("NewTokenUsage() error = %v", err)
	}
	costs, err := domain.NewCostBreakdown(3.10, 1.10, 0.20, 0, 0, 0)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}
	entry, err := domain.NewUsageEntry(domain.UsageEntry{
		EntryID:       "usage-1",
		Source:        domain.UsageSourceCLISession,
		Provider:      domain.ProviderOpenAI,
		BillingMode:   domain.BillingModeBYOK,
		OccurredAt:    time.Date(2026, 4, 17, 12, 30, 0, 0, time.UTC),
		SessionID:     "session-1",
		ProjectName:   "alpha",
		AgentName:     "codex",
		Metadata:      map[string]string{"project_hash": "project-alpha"},
		PricingRef:    &usageRef,
		Tokens:        tokens,
		CostBreakdown: costs,
	})
	if err != nil {
		t.Fatalf("NewUsageEntry() error = %v", err)
	}

	session, err := domain.NewSessionSummary(domain.SessionSummary{
		SessionID:     "session-1",
		Source:        domain.UsageSourceCLISession,
		Provider:      domain.ProviderOpenAI,
		BillingMode:   domain.BillingModeBYOK,
		StartedAt:     time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC),
		EndedAt:       time.Date(2026, 4, 17, 12, 45, 0, 0, time.UTC),
		ProjectName:   "alpha",
		AgentName:     "codex",
		PricingRef:    &usageRef,
		Tokens:        tokens,
		CostBreakdown: costs,
	})
	if err != nil {
		t.Fatalf("NewSessionSummary() error = %v", err)
	}

	fee, err := domain.NewSubscriptionFee(domain.SubscriptionFee{
		SubscriptionID: "sub-1",
		Provider:       domain.ProviderClaude,
		PlanCode:       "claude-max",
		ChargedAt:      time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC),
		Period:         period,
		FeeUSD:         100,
	})
	if err != nil {
		t.Fatalf("NewSubscriptionFee() error = %v", err)
	}

	threshold, err := domain.NewBudgetThreshold(domain.AlertSeverityWarning, 0.8)
	if err != nil {
		t.Fatalf("NewBudgetThreshold() error = %v", err)
	}
	budget, err := domain.NewMonthlyBudget(domain.MonthlyBudget{
		BudgetID:   "budget-openai",
		Name:       "OpenAI Usage",
		Period:     period,
		LimitUSD:   10,
		Thresholds: []domain.BudgetThreshold{threshold},
		Currency:   "USD",
		Provider:   domain.ProviderOpenAI,
	})
	if err != nil {
		t.Fatalf("NewMonthlyBudget() error = %v", err)
	}
	svc := NewDashboardQueryService(
		stubUsageRepo{entries: []domain.UsageEntry{entry}},
		stubSessionRepo{sessions: []domain.SessionSummary{session}},
		stubBudgetRepo{budgets: []domain.MonthlyBudget{budget}},
		stubSubscriptionRepo{fees: []domain.SubscriptionFee{fee}},
	)

	data, err := svc.QueryDashboard(context.Background(), DashboardQuery{Period: period, RecentSessionLimit: 5})
	if err != nil {
		t.Fatalf("QueryDashboard() error = %v", err)
	}

	if data.Empty {
		t.Fatal("QueryDashboard() marked non-empty dashboard as empty")
	}
	if got, want := data.Totals.TotalSpendUSD, 104.4; got != want {
		t.Fatalf("Totals.TotalSpendUSD = %v, want %v", got, want)
	}
	if len(data.ProviderSummaries) != 2 {
		t.Fatalf("len(ProviderSummaries) = %d, want 2", len(data.ProviderSummaries))
	}
	if len(data.Budgets) != 1 {
		t.Fatalf("len(Budgets) = %d, want 1", len(data.Budgets))
	}
	if got := data.Budgets[0].CurrentSpendUSD; got != 4.4 {
		t.Fatalf("Budgets[0].CurrentSpendUSD = %v, want 4.4", got)
	}
	if len(data.RecentSessions) != 1 || data.RecentSessions[0].SessionID != "session-1" {
		t.Fatalf("RecentSessions = %+v, want session-1", data.RecentSessions)
	}
}

type stubUsageRepo struct{ entries []domain.UsageEntry }

func (s stubUsageRepo) UpsertUsageEntries(context.Context, []domain.UsageEntry) error { return nil }
func (s stubUsageRepo) ListUsageEntries(context.Context, ports.UsageFilter) ([]domain.UsageEntry, error) {
	return append([]domain.UsageEntry(nil), s.entries...), nil
}

type stubSessionRepo struct{ sessions []domain.SessionSummary }

func (s stubSessionRepo) UpsertSessions(context.Context, []domain.SessionSummary) error { return nil }
func (s stubSessionRepo) ListSessions(context.Context, ports.SessionFilter) ([]domain.SessionSummary, error) {
	return append([]domain.SessionSummary(nil), s.sessions...), nil
}

type stubSubscriptionRepo struct{ fees []domain.SubscriptionFee }

func (s stubSubscriptionRepo) UpsertSubscriptions(context.Context, []domain.Subscription) error {
	return nil
}
func (s stubSubscriptionRepo) ListSubscriptions(context.Context, ports.SubscriptionFilter) ([]domain.Subscription, error) {
	return nil, nil
}
func (s stubSubscriptionRepo) DisableSubscription(context.Context, string, time.Time) error {
	return nil
}
func (s stubSubscriptionRepo) UpsertSubscriptionFees(context.Context, []domain.SubscriptionFee) error {
	return nil
}
func (s stubSubscriptionRepo) ListSubscriptionFees(context.Context, domain.MonthlyPeriod) ([]domain.SubscriptionFee, error) {
	return append([]domain.SubscriptionFee(nil), s.fees...), nil
}

type stubBudgetRepo struct{ budgets []domain.MonthlyBudget }

func (s stubBudgetRepo) UpsertMonthlyBudgets(context.Context, []domain.MonthlyBudget) error {
	return nil
}
func (s stubBudgetRepo) ListMonthlyBudgets(context.Context, ports.BudgetFilter) ([]domain.MonthlyBudget, error) {
	return append([]domain.MonthlyBudget(nil), s.budgets...), nil
}
func (s stubBudgetRepo) UpsertBudgetStates(context.Context, []domain.BudgetState) error { return nil }
func (s stubBudgetRepo) GetBudgetState(context.Context, string, domain.MonthlyPeriod) (domain.BudgetState, bool, error) {
	return domain.BudgetState{}, false, nil
}
