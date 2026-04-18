package tui

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"llm-budget-tracker/internal/adapters/sqlite"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
	"llm-budget-tracker/internal/service"
)

func TestModelRendersDashboardSectionsAndNavigation(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	loader := staticLoader{data: service.DashboardSnapshot{
		Period: period,
		Totals: service.DashboardTotals{
			TotalSpendUSD:        123.45,
			VariableSpendUSD:     23.45,
			SubscriptionSpendUSD: 100,
		},
		ProviderSummaries: []service.DashboardProviderSummary{{
			Provider:             domain.ProviderOpenAI,
			TotalSpendUSD:        23.45,
			VariableSpendUSD:     23.45,
			SubscriptionSpendUSD: 0,
			SessionCount:         1,
			UsageEntryCount:      2,
		}},
		Budgets: []service.DashboardBudgetSummary{{
			BudgetID:        "budget-1",
			Name:            "Core Budget",
			Provider:        domain.ProviderOpenAI,
			LimitUSD:        50,
			CurrentSpendUSD: 23.45,
			RemainingUSD:    26.55,
		}},
		RecentSessions: []service.DashboardRecentSession{{
			SessionID:    "session-1",
			Provider:     domain.ProviderOpenAI,
			AgentName:    "codex",
			ProjectName:  "alpha",
			EndedAt:      time.Date(2026, 4, 17, 12, 45, 0, 0, time.UTC),
			TotalCostUSD: 23.45,
			TotalTokens:  1500,
			BillingMode:  domain.BillingModeBYOK,
			ModelID:      "gpt-5-mini",
		}},
	}}
	m := newModel(modelDependencies{loader: loader}, period)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(model)
	updated, _ = m.Update(dashboardLoadedMsg{data: loader.data})
	m = updated.(model)

	view := m.View()
	for _, needle := range []string{"Monthly Totals", "Provider Summary", "Budgets", "Recent Sessions", "Monthly total:"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("View() missing %q\n%s", needle, view)
		}
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)
	if !strings.Contains(m.View(), "[Provider Summary]") {
		t.Fatalf("View() did not move focus to provider section\n%s", m.View())
	}
}

func TestModelRendersExplicitEmptyState(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}}, period)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 70, Height: 18})
	m = updated.(model)
	updated, _ = m.Update(dashboardLoadedMsg{data: service.DashboardSnapshot{Period: period, Empty: true}})
	m = updated.(model)

	view := m.View()
	if !strings.Contains(view, "No spend, budgets, or sessions are available") {
		t.Fatalf("View() missing empty state message\n%s", view)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)
	if !strings.Contains(m.View(), "Navigation stays active") {
		t.Fatalf("View() lost empty state after navigation\n%s", m.View())
	}
}

func TestModelShowsAlertBannerAndInsightDrillDown(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	hash, err := domain.NewInsightHash("target_hash", "sha256:abc123")
	if err != nil {
		t.Fatalf("NewInsightHash() error = %v", err)
	}
	metric, err := domain.NewInsightMetric("estimated_waste_usd", domain.InsightMetricUnitUSD, 1.25)
	if err != nil {
		t.Fatalf("NewInsightMetric() error = %v", err)
	}
	payload, err := domain.NewInsightPayload([]string{"session-1"}, []string{"usage-1"}, []domain.InsightHash{hash}, nil, []domain.InsightMetric{metric})
	if err != nil {
		t.Fatalf("NewInsightPayload() error = %v", err)
	}
	insight, err := domain.NewInsight(domain.Insight{
		InsightID:  "insight-1",
		Category:   domain.DetectorToolSchemaBloat,
		Severity:   domain.InsightSeverityHigh,
		DetectedAt: period.StartAt.Add(24 * time.Hour),
		Period:     period,
		Payload:    payload,
	})
	if err != nil {
		t.Fatalf("NewInsight() error = %v", err)
	}
	alert, err := domain.NewAlertEvent(domain.AlertEvent{
		AlertID:          "alert-1",
		Kind:             domain.AlertKindInsightDetected,
		Severity:         domain.AlertSeverityWarning,
		TriggeredAt:      period.StartAt.Add(25 * time.Hour),
		Period:           period,
		InsightID:        insight.InsightID,
		DetectorCategory: insight.Category,
	})
	if err != nil {
		t.Fatalf("NewAlertEvent() error = %v", err)
	}

	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}}, period)
	m.alerts = []domain.AlertEvent{alert}
	m.insights = []domain.Insight{insight}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)

	view := m.View()
	for _, needle := range []string{"Alert status:", "Insight Detail", "sha256:abc123", "estimated_waste_usd"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("View() missing %q\n%s", needle, view)
		}
	}
}

func TestManualEntryValidationPreservesAlertBanner(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	alert, err := domain.NewAlertEvent(domain.AlertEvent{
		AlertID:          "alert-threshold",
		Kind:             domain.AlertKindBudgetThreshold,
		Severity:         domain.AlertSeverityCritical,
		TriggeredAt:      period.StartAt.Add(3 * time.Hour),
		Period:           period,
		BudgetID:         "budget-1",
		CurrentSpendUSD:  92,
		LimitUSD:         100,
		ThresholdPercent: 0.9,
	})
	if err != nil {
		t.Fatalf("NewAlertEvent() error = %v", err)
	}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}, manualEntries: rejectingManualSaver{}}, period)
	m.alerts = []domain.AlertEvent{alert}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = updated.(model)
	setFieldValue(&m.manualForm.fields, "provider", "openrouter")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(model)
	if cmd == nil {
		t.Fatal("expected submit command")
	}
	updated, _ = m.Update(cmd())
	m = updated.(model)

	view := m.View()
	for _, needle := range []string{"Alert status:", "budget budget-1 crossed 90%", "manual API entries support only openai and anthropic", "Fix the highlighted fields"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("View() missing %q\n%s", needle, view)
		}
	}
	if m.mode != viewManualEntryForm {
		t.Fatalf("mode = %v, want manual form after validation failure", m.mode)
	}
}

func TestSubscriptionFormSubmitsThroughSharedService(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	manager := &captureSubscriptionManager{}
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period, Empty: true}}, subscriptions: manager}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(model)
	setFieldValue(&m.subscriptionForm.fields, "subscription_id", "sub-test")
	setFieldValue(&m.subscriptionForm.fields, "provider", "openai")
	setFieldValue(&m.subscriptionForm.fields, "plan_code", "chatgpt-plus")
	setFieldValue(&m.subscriptionForm.fields, "plan_name", "ChatGPT Plus")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(model)
	if cmd == nil {
		t.Fatal("expected submit command")
	}
	updated, _ = m.Update(cmd())
	m = updated.(model)

	if len(manager.saved) != 1 {
		t.Fatalf("saved subscriptions = %d, want 1", len(manager.saved))
	}
	if got := manager.saved[0].SubscriptionID; got != "sub-test" {
		t.Fatalf("SubscriptionID = %q, want sub-test", got)
	}
	if m.mode != viewDashboard {
		t.Fatalf("mode = %v, want dashboard after successful save", m.mode)
	}
}

func TestSubscriptionFormDefaultsToCurrentBillingDate(t *testing.T) {
	fixedNow := time.Date(2026, time.April, 19, 14, 30, 0, 0, time.UTC)
	form := newSubscriptionFormAt(fixedNow)
	values := collectFormValues(form.fields)

	if got, want := values["renewal_day"], "5"; got != want {
		t.Fatalf("renewal_day default = %q, want %q", got, want)
	}

	expectedStartsAt := time.Date(2026, time.April, 5, 14, 30, 0, 0, time.UTC).Format(time.RFC3339)
	if got := values["starts_at"]; got != expectedStartsAt {
		t.Fatalf("starts_at default = %q, want %q", got, expectedStartsAt)
	}
}

func TestSubscriptionFormDefaultsContributeToCurrentMonthDashboardTotal(t *testing.T) {
	fixedNow := time.Date(2026, time.April, 19, 14, 30, 0, 0, time.UTC)
	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: filepath.Join(t.TempDir(), "subscription-defaults.sqlite3")})
	if err != nil {
		t.Fatalf("sqlite.Bootstrap() error = %v", err)
	}
	defer store.Close()

	period, err := domain.NewMonthlyPeriod(fixedNow)
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}

	form := newSubscriptionFormAt(fixedNow)
	manager := service.NewSubscriptionService(store, store)
	subscription, ok := form.parseSubscription(manager)
	if !ok {
		t.Fatalf("parseSubscription() errors = %#v", form.errors)
	}

	if err := manager.SaveSubscriptions(context.Background(), []domain.Subscription{subscription}); err != nil {
		t.Fatalf("SaveSubscriptions() error = %v", err)
	}

	query := service.NewDashboardQueryService(store, store, store, store)
	snapshot, err := query.QueryDashboard(context.Background(), service.DashboardQuery{Period: period, RecentSessionLimit: 8})
	if err != nil {
		t.Fatalf("QueryDashboard() error = %v", err)
	}

	if got, want := snapshot.Totals.SubscriptionSpendUSD, 20.0; got != want {
		t.Fatalf("SubscriptionSpendUSD = %v, want %v", got, want)
	}
	if got, want := snapshot.Totals.TotalSpendUSD, 20.0; got != want {
		t.Fatalf("TotalSpendUSD = %v, want %v", got, want)
	}
}

type staticLoader struct {
	data service.DashboardSnapshot
	err  error
}

func (s staticLoader) QueryDashboard(context.Context, service.DashboardQuery) (service.DashboardSnapshot, error) {
	return s.data, s.err
}

type rejectingManualSaver struct{}

func (rejectingManualSaver) Save(context.Context, service.ManualAPIUsageEntryCommand) (domain.UsageEntry, error) {
	return domain.UsageEntry{}, &domain.ValidationError{Code: domain.ValidationCodeUnsupportedProvider, Field: "provider", Message: "manual API entries support only openai and anthropic"}
}

type captureSubscriptionManager struct {
	saved []domain.Subscription
}

func (c *captureSubscriptionManager) SaveSubscriptions(_ context.Context, subscriptions []domain.Subscription) error {
	c.saved = append(c.saved, subscriptions...)
	return nil
}

func (c *captureSubscriptionManager) ListSubscriptions(context.Context, ports.SubscriptionFilter) ([]domain.Subscription, error) {
	return nil, nil
}
