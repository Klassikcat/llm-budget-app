package gui

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"llm-budget-tracker/internal/adapters/sqlite"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/service"
)

func TestDashboardBindings(t *testing.T) {
	period := mustMonthlyPeriod(t, 2026, time.January)
	store := mustDashboardStore(t)
	defer store.Close()

	seedDashboardData(t, store, period)
	binding := newTestDashboardBinding(period, store)

	response, err := binding.LoadDashboard("2026-01")
	if err != nil {
		t.Fatalf("LoadDashboard() error = %v", err)
	}

	if response.Empty {
		t.Fatal("LoadDashboard() empty = true, want false")
	}
	if response.Period.Month != "2026-01" {
		t.Fatalf("Period.Month = %q, want 2026-01", response.Period.Month)
	}
	if response.Totals.VariableSpendUSD != 13 {
		t.Fatalf("VariableSpendUSD = %v, want 13", response.Totals.VariableSpendUSD)
	}
	if response.Totals.SubscriptionSpendUSD != 20 {
		t.Fatalf("SubscriptionSpendUSD = %v, want 20", response.Totals.SubscriptionSpendUSD)
	}
	if response.Totals.TotalSpendUSD != 33 {
		t.Fatalf("TotalSpendUSD = %v, want 33", response.Totals.TotalSpendUSD)
	}

	if len(response.ProviderSummaries) != 2 {
		t.Fatalf("len(ProviderSummaries) = %d, want 2", len(response.ProviderSummaries))
	}
	if got := response.ProviderSummaries[0]; got.Provider != "openai" || got.TotalSpendUSD != 25 || got.SessionCount != 1 {
		t.Fatalf("ProviderSummaries[0] = %+v, want openai total 25 sessionCount 1", got)
	}
	if got := response.ProviderSummaries[1]; got.Provider != "openrouter" || got.TotalSpendUSD != 8 || got.UsageEntryCount != 1 {
		t.Fatalf("ProviderSummaries[1] = %+v, want openrouter total 8 usageEntryCount 1", got)
	}

	if len(response.Budgets) != 2 {
		t.Fatalf("len(Budgets) = %d, want 2", len(response.Budgets))
	}
	if got := response.Budgets[0]; got.BudgetID != "budget-global" || got.CurrentSpendUSD != 33 || got.RemainingUSD != 67 {
		t.Fatalf("Budgets[0] = %+v, want global spend 33 remaining 67", got)
	}
	if got := response.Budgets[1]; got.BudgetID != "budget-openrouter" || got.CurrentSpendUSD != 8 || len(got.TriggeredThresholdPercents) != 1 || got.TriggeredThresholdPercents[0] != 0.8 {
		t.Fatalf("Budgets[1] = %+v, want openrouter spend 8 threshold 0.8", got)
	}

	if len(response.RecentSessions) != 2 {
		t.Fatalf("len(RecentSessions) = %d, want 2", len(response.RecentSessions))
	}
	if got := response.RecentSessions[0]; got.SessionID != "session-openrouter-late" || got.Provider != "openrouter" || got.TotalCostUSD != 8 {
		t.Fatalf("RecentSessions[0] = %+v, want latest openrouter session", got)
	}
	if got := response.RecentSessions[1]; got.SessionID != "session-openai-early" || got.Provider != "openai" || got.TotalTokens != 1500 {
		t.Fatalf("RecentSessions[1] = %+v, want openai session with 1500 total tokens", got)
	}
}

func TestDashboardEmptyState(t *testing.T) {
	store := mustDashboardStore(t)
	defer store.Close()

	period := mustMonthlyPeriod(t, 2026, time.January)
	binding := newTestDashboardBinding(period, store)

	response, err := binding.LoadDashboard("2026-01")
	if err != nil {
		t.Fatalf("LoadDashboard() error = %v", err)
	}

	if !response.Empty {
		t.Fatal("LoadDashboard() empty = false, want true")
	}
	if response.ProviderSummaries == nil {
		t.Fatal("ProviderSummaries = nil, want empty slice")
	}
	if response.Budgets == nil {
		t.Fatal("Budgets = nil, want empty slice")
	}
	if response.RecentSessions == nil {
		t.Fatal("RecentSessions = nil, want empty slice")
	}
	if response.Totals.TotalSpendUSD != 0 || response.Totals.VariableSpendUSD != 0 || response.Totals.SubscriptionSpendUSD != 0 {
		t.Fatalf("Totals = %+v, want all zero", response.Totals)
	}
	if len(response.ProviderSummaries) != 0 || len(response.Budgets) != 0 || len(response.RecentSessions) != 0 {
		t.Fatalf("response lengths = (%d, %d, %d), want all 0", len(response.ProviderSummaries), len(response.Budgets), len(response.RecentSessions))
	}
}

func newTestDashboardBinding(anchorPeriod domain.MonthlyPeriod, store *sqlite.Store) *DashboardBinding {
	queryService := service.NewDashboardQueryService(store, store, store, store)
	clock := anchorPeriod.StartAt.Add(14 * 24 * time.Hour).UTC()
	queryService.ClockForTest(func() time.Time { return clock })

	binding := NewDashboardBinding(queryService)
	binding.clock = func() time.Time { return clock }
	binding.startup(context.Background())
	return binding
}

func mustDashboardStore(t *testing.T) *sqlite.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dashboard.sqlite3")
	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: path})
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	return store
}

func seedDashboardData(t *testing.T, store *sqlite.Store, period domain.MonthlyPeriod) {
	t.Helper()
	ctx := context.Background()

	openAIRef := mustPricingRef(t, domain.ProviderOpenAI, "gpt-4o-mini")
	openRouterRef := mustPricingRef(t, domain.ProviderOpenRouter, "openrouter/anthropic/claude-3.5-sonnet")

	if err := store.UpsertSessions(ctx, []domain.SessionSummary{
		mustSessionSummary(t, domain.SessionSummary{
			SessionID:     "session-openai-early",
			Source:        domain.UsageSourceCLISession,
			Provider:      domain.ProviderOpenAI,
			BillingMode:   domain.BillingModeBYOK,
			StartedAt:     period.StartAt.Add(1 * time.Hour),
			EndedAt:       period.StartAt.Add(2 * time.Hour),
			ProjectName:   "workspace-a",
			AgentName:     "codex",
			PricingRef:    &openAIRef,
			Tokens:        mustTokenUsage(t, 1000, 500, 0, 0),
			CostBreakdown: mustCostBreakdown(t, 2, 3, 0, 0, 0, 0),
		}),
		mustSessionSummary(t, domain.SessionSummary{
			SessionID:     "session-openrouter-late",
			Source:        domain.UsageSourceCLISession,
			Provider:      domain.ProviderOpenRouter,
			BillingMode:   domain.BillingModeOpenRouter,
			StartedAt:     period.StartAt.Add(9*24*time.Hour + 30*time.Minute),
			EndedAt:       period.StartAt.Add(9*24*time.Hour + 2*time.Hour),
			ProjectName:   "workspace-b",
			AgentName:     "opencode",
			PricingRef:    &openRouterRef,
			Tokens:        mustTokenUsage(t, 1200, 300, 100, 0),
			CostBreakdown: mustCostBreakdown(t, 4, 3, 1, 0, 0, 0),
		}),
	}); err != nil {
		t.Fatalf("UpsertSessions() error = %v", err)
	}

	if err := store.UpsertUsageEntries(ctx, []domain.UsageEntry{
		mustUsageEntry(t, domain.UsageEntry{
			EntryID:       "usage-openai",
			SessionID:     "session-openai-early",
			Source:        domain.UsageSourceCLISession,
			Provider:      domain.ProviderOpenAI,
			BillingMode:   domain.BillingModeBYOK,
			OccurredAt:    period.StartAt.Add(2 * time.Hour),
			ProjectName:   "workspace-a",
			AgentName:     "codex",
			PricingRef:    &openAIRef,
			Tokens:        mustTokenUsage(t, 1000, 500, 0, 0),
			CostBreakdown: mustCostBreakdown(t, 2, 3, 0, 0, 0, 0),
		}),
		mustUsageEntry(t, domain.UsageEntry{
			EntryID:       "usage-openrouter",
			SessionID:     "session-openrouter-late",
			Source:        domain.UsageSourceOpenRouter,
			Provider:      domain.ProviderOpenRouter,
			BillingMode:   domain.BillingModeOpenRouter,
			OccurredAt:    period.StartAt.Add(9 * 24 * time.Hour),
			ProjectName:   "workspace-b",
			AgentName:     "opencode",
			PricingRef:    &openRouterRef,
			Metadata:      map[string]string{"project_hash": "hash-openrouter"},
			Tokens:        mustTokenUsage(t, 1200, 300, 100, 0),
			CostBreakdown: mustCostBreakdown(t, 4, 3, 1, 0, 0, 0),
		}),
	}); err != nil {
		t.Fatalf("UpsertUsageEntries() error = %v", err)
	}

	if err := store.UpsertSubscriptions(ctx, []domain.Subscription{
		mustSubscription(t, domain.Subscription{
			SubscriptionID: "sub-openai",
			Provider:       domain.ProviderOpenAI,
			PlanCode:       "chatgpt-plus",
			PlanName:       "ChatGPT Plus",
			RenewalDay:     1,
			StartsAt:       period.StartAt,
			FeeUSD:         20,
			IsActive:       true,
			CreatedAt:      period.StartAt,
			UpdatedAt:      period.StartAt,
		}),
	}); err != nil {
		t.Fatalf("UpsertSubscriptions() error = %v", err)
	}

	if err := store.UpsertMonthlyBudgets(ctx, []domain.MonthlyBudget{
		mustMonthlyBudget(t, domain.MonthlyBudget{
			BudgetID: "budget-global",
			Name:     "Global Budget",
			Period:   period,
			LimitUSD: 100,
			Thresholds: []domain.BudgetThreshold{
				mustBudgetThreshold(t, domain.AlertSeverityWarning, 0.8),
			},
			Currency: "USD",
		}),
		mustMonthlyBudget(t, domain.MonthlyBudget{
			BudgetID:    "budget-openrouter",
			Name:        "OpenRouter Budget",
			Provider:    domain.ProviderOpenRouter,
			ProjectHash: "hash-openrouter",
			Period:      period,
			LimitUSD:    10,
			Thresholds: []domain.BudgetThreshold{
				mustBudgetThreshold(t, domain.AlertSeverityWarning, 0.8),
			},
			Currency: "USD",
		}),
	}); err != nil {
		t.Fatalf("UpsertMonthlyBudgets() error = %v", err)
	}
}

func mustMonthlyPeriod(t *testing.T, year int, month time.Month) domain.MonthlyPeriod {
	t.Helper()
	period, err := domain.NewMonthlyPeriodFromParts(year, month)
	if err != nil {
		t.Fatalf("NewMonthlyPeriodFromParts() error = %v", err)
	}
	return period
}

func mustUsageEntry(t *testing.T, entry domain.UsageEntry) domain.UsageEntry {
	t.Helper()
	validated, err := domain.NewUsageEntry(entry)
	if err != nil {
		t.Fatalf("NewUsageEntry() error = %v", err)
	}
	return validated
}

func mustSessionSummary(t *testing.T, summary domain.SessionSummary) domain.SessionSummary {
	t.Helper()
	validated, err := domain.NewSessionSummary(summary)
	if err != nil {
		t.Fatalf("NewSessionSummary() error = %v", err)
	}
	return validated
}

func mustSubscription(t *testing.T, subscription domain.Subscription) domain.Subscription {
	t.Helper()
	validated, err := domain.NewSubscription(subscription)
	if err != nil {
		t.Fatalf("NewSubscription() error = %v", err)
	}
	return validated
}

func mustMonthlyBudget(t *testing.T, budget domain.MonthlyBudget) domain.MonthlyBudget {
	t.Helper()
	validated, err := domain.NewMonthlyBudget(budget)
	if err != nil {
		t.Fatalf("NewMonthlyBudget() error = %v", err)
	}
	return validated
}

func mustBudgetThreshold(t *testing.T, severity domain.AlertSeverity, percent float64) domain.BudgetThreshold {
	t.Helper()
	threshold, err := domain.NewBudgetThreshold(severity, percent)
	if err != nil {
		t.Fatalf("NewBudgetThreshold() error = %v", err)
	}
	return threshold
}

func mustPricingRef(t *testing.T, provider domain.ProviderName, modelID string) domain.ModelPricingRef {
	t.Helper()
	ref, err := domain.NewModelPricingRef(provider, modelID, modelID)
	if err != nil {
		t.Fatalf("NewModelPricingRef() error = %v", err)
	}
	return ref
}

func mustTokenUsage(t *testing.T, input, output, cacheRead, cacheWrite int64) domain.TokenUsage {
	t.Helper()
	tokens, err := domain.NewTokenUsage(input, output, cacheRead, cacheWrite)
	if err != nil {
		t.Fatalf("NewTokenUsage() error = %v", err)
	}
	return tokens
}

func mustCostBreakdown(t *testing.T, input, output, cacheRead, cacheWrite, tool, flat float64) domain.CostBreakdown {
	t.Helper()
	breakdown, err := domain.NewCostBreakdown(input, output, cacheRead, cacheWrite, tool, flat)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}
	return breakdown
}
