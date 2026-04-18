package tui

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"llm-budget-tracker/internal/adapters/sqlite"
	catalogpkg "llm-budget-tracker/internal/catalog"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
	"llm-budget-tracker/internal/service"
)

func TestSeedTask21FixtureDB(t *testing.T) {
	path := os.Getenv("TASK21_SEED_DB")
	if path == "" {
		t.Skip("TASK21_SEED_DB not set")
	}

	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: path})
	if err != nil {
		t.Fatalf("sqlite.Bootstrap() error = %v", err)
	}
	defer store.Close()

	period, err := domain.NewMonthlyPeriod(time.Now().UTC())
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	seedDashboardFixture(t, store, period)
}

func TestWriteTask21DashboardEvidence(t *testing.T) {
	path := os.Getenv("TASK21_DASHBOARD_EVIDENCE")
	if path == "" {
		t.Skip("TASK21_DASHBOARD_EVIDENCE not set")
	}

	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: t.TempDir() + "/dashboard.sqlite3"})
	if err != nil {
		t.Fatalf("sqlite.Bootstrap() error = %v", err)
	}
	defer store.Close()

	period, err := domain.NewMonthlyPeriod(time.Now().UTC())
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	seedDashboardFixture(t, store, period)

	query := service.NewDashboardQueryService(store, store, store, store)
	snapshot, err := query.QueryDashboard(context.Background(), service.DashboardQuery{Period: period, RecentSessionLimit: 8})
	if err != nil {
		t.Fatalf("QueryDashboard() error = %v", err)
	}

	if err := os.WriteFile(path, []byte(renderSnapshotForEvidence(snapshot, period)), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func TestWriteTask21EmptyStateEvidence(t *testing.T) {
	path := os.Getenv("TASK21_EMPTY_EVIDENCE")
	if path == "" {
		t.Skip("TASK21_EMPTY_EVIDENCE not set")
	}

	period, err := domain.NewMonthlyPeriod(time.Now().UTC())
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}

	if err := os.WriteFile(path, []byte(renderSnapshotForEvidence(service.DashboardSnapshot{Period: period, Empty: true}, period)), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func TestWriteTask22ManualEntryEvidence(t *testing.T) {
	path := os.Getenv("TASK22_MANUAL_ENTRY_EVIDENCE")
	if path == "" {
		t.Skip("TASK22_MANUAL_ENTRY_EVIDENCE not set")
	}

	store, period := seedTask22Store(t)
	defer store.Close()

	catalog, err := catalogpkg.New(catalogpkg.Options{})
	if err != nil {
		t.Fatalf("catalog.New() error = %v", err)
	}
	manualEntries := service.NewManualAPIUsageEntryService(catalog, store)
	subscriptions := service.NewSubscriptionService(store, store)
	query := service.NewDashboardQueryService(store, store, store, store)

	m := newModel(modelDependencies{loader: query, manualEntries: manualEntries, subscriptions: subscriptions, insights: store, alerts: store}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 32})
	m = updated.(model)
	updated, _ = m.Update(dashboardLoadedMsg{data: querySnapshot(t, query, period)})
	m = updated.(model)
	updated, _ = m.Update(alertsLoadedMsg{alerts: mustListAlerts(t, store, period)})
	m = updated.(model)
	updated, _ = m.Update(insightsLoadedMsg{insights: mustListInsights(t, store, period)})
	m = updated.(model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = updated.(model)
	setFieldValue(&m.manualForm.fields, "entry_id", "manual-evidence-1")
	setFieldValue(&m.manualForm.fields, "provider", "openai")
	setFieldValue(&m.manualForm.fields, "model_id", "gpt-4.1")
	setFieldValue(&m.manualForm.fields, "occurred_at", period.StartAt.Add(10*time.Hour).Format(time.RFC3339))
	setFieldValue(&m.manualForm.fields, "input_tokens", "1234")
	setFieldValue(&m.manualForm.fields, "output_tokens", "321")
	setFieldValue(&m.manualForm.fields, "cached_tokens", "100")
	setFieldValue(&m.manualForm.fields, "cache_write_tokens", "0")
	setFieldValue(&m.manualForm.fields, "project_name", "task-22")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(model)
	if cmd == nil {
		t.Fatal("expected manual entry submit command")
	}
	updated, next := m.Update(cmd())
	m = updated.(model)
	if next != nil {
		updated, _ = m.Update(next())
		m = updated.(model)
	}
	updated, _ = m.Update(dashboardLoadedMsg{data: querySnapshot(t, query, period)})
	m = updated.(model)

	entries, err := store.ListUsageEntries(context.Background(), ports.UsageFilter{Period: &period, Project: "task-22"})
	if err != nil {
		t.Fatalf("ListUsageEntries() error = %v", err)
	}
	content := m.View() + "\n\nSaved entries for task-22: " + strconv.Itoa(len(entries))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func TestWriteTask22ValidationEvidence(t *testing.T) {
	path := os.Getenv("TASK22_VALIDATION_EVIDENCE")
	if path == "" {
		t.Skip("TASK22_VALIDATION_EVIDENCE not set")
	}

	store, period := seedTask22Store(t)
	defer store.Close()
	query := service.NewDashboardQueryService(store, store, store, store)
	m := newModel(modelDependencies{loader: query, manualEntries: task22RejectingManualSaver{}, insights: store, alerts: store}, period)
	m.alerts = mustListAlerts(t, store, period)
	beforeEntries, err := store.ListUsageEntries(context.Background(), ports.UsageFilter{Period: &period})
	if err != nil {
		t.Fatalf("ListUsageEntries(before) error = %v", err)
	}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 32})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = updated.(model)
	setFieldValue(&m.manualForm.fields, "provider", "openrouter")
	setFieldValue(&m.manualForm.fields, "model_id", "unknown-model")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(model)
	if cmd == nil {
		t.Fatal("expected validation submit command")
	}
	updated, _ = m.Update(cmd())
	m = updated.(model)

	entries, err := store.ListUsageEntries(context.Background(), ports.UsageFilter{Period: &period})
	if err != nil {
		t.Fatalf("ListUsageEntries() error = %v", err)
	}
	content := m.View() + "\n\nPersisted entries before invalid submit: " + strconv.Itoa(len(beforeEntries)) + "\nPersisted entries after invalid submit: " + strconv.Itoa(len(entries))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func seedDashboardFixture(t *testing.T, store *sqlite.Store, period domain.MonthlyPeriod) {
	t.Helper()

	ref, err := domain.NewModelPricingRef(domain.ProviderOpenAI, "gpt-5-mini", "gpt-5-mini")
	if err != nil {
		t.Fatalf("NewModelPricingRef() error = %v", err)
	}
	tokens, err := domain.NewTokenUsage(1800, 420, 120, 0)
	if err != nil {
		t.Fatalf("NewTokenUsage() error = %v", err)
	}
	costs, err := domain.NewCostBreakdown(4.10, 1.20, 0.30, 0, 0, 0)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}
	now := period.StartAt.Add(16 * 24 * time.Hour)

	sessions := []domain.SessionSummary{
		mustSessionSummary(t, domain.SessionSummary{
			SessionID:     "task21-session-1",
			Source:        domain.UsageSourceCLISession,
			Provider:      domain.ProviderOpenAI,
			BillingMode:   domain.BillingModeBYOK,
			StartedAt:     now.Add(-35 * time.Minute),
			EndedAt:       now,
			ProjectName:   "alpha-project",
			AgentName:     "codex",
			PricingRef:    &ref,
			Tokens:        tokens,
			CostBreakdown: costs,
		}),
	}
	if err := store.UpsertSessions(context.Background(), sessions); err != nil {
		t.Fatalf("UpsertSessions() error = %v", err)
	}

	entries := []domain.UsageEntry{
		mustUsageEntry(t, domain.UsageEntry{
			EntryID:       "task21-usage-1",
			Source:        domain.UsageSourceCLISession,
			Provider:      domain.ProviderOpenAI,
			BillingMode:   domain.BillingModeBYOK,
			OccurredAt:    now,
			SessionID:     "task21-session-1",
			ProjectName:   "alpha-project",
			AgentName:     "codex",
			Metadata:      map[string]string{"project_hash": "project-alpha"},
			PricingRef:    &ref,
			Tokens:        tokens,
			CostBreakdown: costs,
		}),
	}
	if err := store.UpsertUsageEntries(context.Background(), entries); err != nil {
		t.Fatalf("UpsertUsageEntries() error = %v", err)
	}

	subscriptions := []domain.Subscription{
		mustSubscription(t, domain.Subscription{
			SubscriptionID: "task21-sub-1",
			Provider:       domain.ProviderClaude,
			PlanCode:       "claude-max",
			PlanName:       "Claude Max",
			RenewalDay:     2,
			StartsAt:       period.StartAt,
			FeeUSD:         100,
			IsActive:       true,
			CreatedAt:      period.StartAt,
			UpdatedAt:      now,
		}),
	}
	if err := store.UpsertSubscriptions(context.Background(), subscriptions); err != nil {
		t.Fatalf("UpsertSubscriptions() error = %v", err)
	}

	fees := []domain.SubscriptionFee{
		mustSubscriptionFee(t, domain.SubscriptionFee{
			SubscriptionID: "task21-sub-1",
			Provider:       domain.ProviderClaude,
			PlanCode:       "claude-max",
			ChargedAt:      period.StartAt.Add(24 * time.Hour),
			Period:         period,
			FeeUSD:         100,
		}),
	}
	if err := store.UpsertSubscriptionFees(context.Background(), fees); err != nil {
		t.Fatalf("UpsertSubscriptionFees() error = %v", err)
	}

	threshold, err := domain.NewBudgetThreshold(domain.AlertSeverityWarning, 0.8)
	if err != nil {
		t.Fatalf("NewBudgetThreshold() error = %v", err)
	}
	budgets := []domain.MonthlyBudget{
		mustBudget(t, domain.MonthlyBudget{
			BudgetID:   "task21-budget-openai",
			Name:       "OpenAI Usage",
			Period:     period,
			LimitUSD:   20,
			Thresholds: []domain.BudgetThreshold{threshold},
			Currency:   "USD",
			Provider:   domain.ProviderOpenAI,
		}),
	}
	if err := store.UpsertMonthlyBudgets(context.Background(), budgets); err != nil {
		t.Fatalf("UpsertMonthlyBudgets() error = %v", err)
	}

	forecasts := []domain.ForecastSnapshot{
		mustForecast(t, domain.ForecastSnapshot{
			ForecastID:        "task21-budget-openai:forecast",
			Period:            period,
			GeneratedAt:       now,
			ActualSpendUSD:    costs.TotalUSD,
			ForecastSpendUSD:  9.75,
			BudgetLimitUSD:    20,
			ObservedDayCount:  16,
			RemainingDayCount: 14,
		}),
	}
	if err := store.UpsertForecastSnapshots(context.Background(), forecasts); err != nil {
		t.Fatalf("UpsertForecastSnapshots() error = %v", err)
	}
}

func renderSnapshotForEvidence(snapshot service.DashboardSnapshot, period domain.MonthlyPeriod) string {
	m := newModel(modelDependencies{loader: staticLoader{data: snapshot}}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(model)
	updated, _ = m.Update(dashboardLoadedMsg{data: snapshot})
	m = updated.(model)
	return m.View()
}

func seedTask22Store(t *testing.T) (*sqlite.Store, domain.MonthlyPeriod) {
	t.Helper()
	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: filepath.Join(t.TempDir(), "task22.sqlite3")})
	if err != nil {
		t.Fatalf("sqlite.Bootstrap() error = %v", err)
	}
	period, err := domain.NewMonthlyPeriod(time.Now().UTC())
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	seedDashboardFixture(t, store, period)
	seedTask22InsightsAndAlerts(t, store, period)
	return store, period
}

func seedTask22InsightsAndAlerts(t *testing.T, store *sqlite.Store, period domain.MonthlyPeriod) {
	t.Helper()
	hash, err := domain.NewInsightHash("target_hash", "sha256:task22")
	if err != nil {
		t.Fatalf("NewInsightHash() error = %v", err)
	}
	count, err := domain.NewInsightCount("tool_definition_count", 3)
	if err != nil {
		t.Fatalf("NewInsightCount() error = %v", err)
	}
	metric, err := domain.NewInsightMetric("estimated_waste_usd", domain.InsightMetricUnitUSD, 1.42)
	if err != nil {
		t.Fatalf("NewInsightMetric() error = %v", err)
	}
	payload, err := domain.NewInsightPayload([]string{"task21-session-1"}, []string{"task21-usage-1"}, []domain.InsightHash{hash}, []domain.InsightCount{count}, []domain.InsightMetric{metric})
	if err != nil {
		t.Fatalf("NewInsightPayload() error = %v", err)
	}
	insight, err := domain.NewInsight(domain.Insight{
		InsightID:  "task22-insight-1",
		Category:   domain.DetectorToolSchemaBloat,
		Severity:   domain.InsightSeverityMedium,
		DetectedAt: period.StartAt.Add(12 * time.Hour),
		Period:     period,
		Payload:    payload,
	})
	if err != nil {
		t.Fatalf("NewInsight() error = %v", err)
	}
	if err := store.UpsertInsights(context.Background(), []domain.Insight{insight}); err != nil {
		t.Fatalf("UpsertInsights() error = %v", err)
	}
	alert, err := domain.NewAlertEvent(domain.AlertEvent{
		AlertID:          "task22-alert-1",
		Kind:             domain.AlertKindInsightDetected,
		Severity:         domain.AlertSeverityWarning,
		TriggeredAt:      period.StartAt.Add(13 * time.Hour),
		Period:           period,
		InsightID:        insight.InsightID,
		DetectorCategory: insight.Category,
	})
	if err != nil {
		t.Fatalf("NewAlertEvent() error = %v", err)
	}
	if err := store.UpsertAlerts(context.Background(), []domain.AlertEvent{alert}); err != nil {
		t.Fatalf("UpsertAlerts() error = %v", err)
	}
}

func querySnapshot(t *testing.T, query *service.DashboardQueryService, period domain.MonthlyPeriod) service.DashboardSnapshot {
	t.Helper()
	snapshot, err := query.QueryDashboard(context.Background(), service.DashboardQuery{Period: period, RecentSessionLimit: 8})
	if err != nil {
		t.Fatalf("QueryDashboard() error = %v", err)
	}
	return snapshot
}

func mustListAlerts(t *testing.T, alerts ports.AlertRepository, period domain.MonthlyPeriod) []domain.AlertEvent {
	t.Helper()
	items, err := alerts.ListAlerts(context.Background(), ports.AlertFilter{Period: &period})
	if err != nil {
		t.Fatalf("ListAlerts() error = %v", err)
	}
	return items
}

func mustListInsights(t *testing.T, insights ports.InsightRepository, period domain.MonthlyPeriod) []domain.Insight {
	t.Helper()
	items, err := insights.ListInsights(context.Background(), period)
	if err != nil {
		t.Fatalf("ListInsights() error = %v", err)
	}
	return items
}

func setFieldValue(fields *[]formField, key, value string) {
	for i := range *fields {
		if (*fields)[i].key == key {
			(*fields)[i].input.SetValue(value)
			return
		}
	}
}

type task22RejectingManualSaver struct{}

func (task22RejectingManualSaver) Save(context.Context, service.ManualAPIUsageEntryCommand) (domain.UsageEntry, error) {
	return domain.UsageEntry{}, &domain.ValidationError{Code: domain.ValidationCodeUnsupportedProvider, Field: "provider", Message: "manual API entries support only openai and anthropic"}
}

func mustUsageEntry(t *testing.T, entry domain.UsageEntry) domain.UsageEntry {
	t.Helper()
	validated, err := domain.NewUsageEntry(entry)
	if err != nil {
		t.Fatalf("NewUsageEntry() error = %v", err)
	}
	return validated
}

func mustSessionSummary(t *testing.T, session domain.SessionSummary) domain.SessionSummary {
	t.Helper()
	validated, err := domain.NewSessionSummary(session)
	if err != nil {
		t.Fatalf("NewSessionSummary() error = %v", err)
	}
	return validated
}

func mustSubscriptionFee(t *testing.T, fee domain.SubscriptionFee) domain.SubscriptionFee {
	t.Helper()
	validated, err := domain.NewSubscriptionFee(fee)
	if err != nil {
		t.Fatalf("NewSubscriptionFee() error = %v", err)
	}
	return validated
}

func mustBudget(t *testing.T, budget domain.MonthlyBudget) domain.MonthlyBudget {
	t.Helper()
	validated, err := domain.NewMonthlyBudget(budget)
	if err != nil {
		t.Fatalf("NewMonthlyBudget() error = %v", err)
	}
	return validated
}

func mustForecast(t *testing.T, forecast domain.ForecastSnapshot) domain.ForecastSnapshot {
	t.Helper()
	validated, err := domain.NewForecastSnapshot(forecast)
	if err != nil {
		t.Fatalf("NewForecastSnapshot() error = %v", err)
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
