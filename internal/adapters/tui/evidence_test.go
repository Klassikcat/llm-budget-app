package tui

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"llm-budget-tracker/internal/adapters/sqlite"
	catalogpkg "llm-budget-tracker/internal/catalog"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
	"llm-budget-tracker/internal/service"
)

const task15ExpectedVariableSpend = 0.1947

func TestTask15FullDashboardIntegration(t *testing.T) {
	store, period := seedTask15Store(t)
	defer store.Close()

	dashboardSnapshot, graphSnapshot := queryTask15Snapshots(t, store, period)
	assertTask15DashboardSnapshot(t, dashboardSnapshot)
	assertTask15GraphSnapshot(t, graphSnapshot)

	dashboardView := renderSnapshotForEvidence(dashboardSnapshot, period)
	for _, needle := range []string{"openrouter", "openclaw", "claude-code/acp", "$0.19", "$0.01", "$0.05", "$0.08", "anthropic/claude-3.5-sonnet", "gpt-4.1-openclaw"} {
		if !strings.Contains(dashboardView, needle) {
			t.Fatalf("dashboard view missing %q\n%s", needle, dashboardView)
		}
	}

	graphViews := renderTask15GraphViews(graphSnapshot, period)
	for _, needle := range []string{"claude-sonnet-4-standard", "claude-opus-4-acp", "anthropic/claude-3.5-sonnet", "gpt-4.1-openclaw", "opencode/qwen3-coder", "tokens", "$0.08", "$0.05", "$0.01"} {
		if !strings.Contains(graphViews, needle) {
			t.Fatalf("graph views missing %q\n%s", needle, graphViews)
		}
	}
}

func TestWriteTask15FullDashboardEvidence(t *testing.T) {
	path := os.Getenv("TASK15_FULL_DASHBOARD_EVIDENCE")
	if path == "" {
		t.Skip("TASK15_FULL_DASHBOARD_EVIDENCE not set")
	}

	store, period := seedTask15Store(t)
	defer store.Close()
	dashboardSnapshot, graphSnapshot := queryTask15Snapshots(t, store, period)
	content := strings.Join([]string{
		"Task 15 full dashboard integration evidence",
		fmt.Sprintf("monthly_total=%.4f", dashboardSnapshot.Totals.TotalSpendUSD),
		"",
		renderSnapshotForEvidence(dashboardSnapshot, period),
		"",
		"Graph views",
		renderTask15GraphViews(graphSnapshot, period),
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func TestTask15EmptyTempDBRendersSafely(t *testing.T) {
	store, period := emptyTask15Store(t)
	defer store.Close()

	dashboardSnapshot, graphSnapshot := queryTask15Snapshots(t, store, period)
	if !dashboardSnapshot.Empty {
		t.Fatalf("snapshot.Empty = false, want true: %#v", dashboardSnapshot)
	}
	if len(graphSnapshot.ModelTokenUsages) != 0 || len(graphSnapshot.ModelCosts) != 0 || len(graphSnapshot.ModelTokenBreakdowns) != 0 {
		t.Fatalf("graph snapshot should be empty: %#v", graphSnapshot)
	}

	dashboardView := renderSnapshotForEvidence(dashboardSnapshot, period)
	if !strings.Contains(dashboardView, "No spend, budgets, or sessions are available") {
		t.Fatalf("dashboard empty view missing empty state\n%s", dashboardView)
	}
	graphViews := renderTask15GraphViews(graphSnapshot, period)
	if !strings.Contains(graphViews, "No model token activity") || !strings.Contains(graphViews, "No model cost activity") {
		t.Fatalf("graph empty views missing safe empty messages\n%s", graphViews)
	}
}

func TestWriteTask15EmptyStateEvidence(t *testing.T) {
	path := os.Getenv("TASK15_EMPTY_STATE_EVIDENCE")
	if path == "" {
		t.Skip("TASK15_EMPTY_STATE_EVIDENCE not set")
	}

	store, period := emptyTask15Store(t)
	defer store.Close()
	dashboardSnapshot, graphSnapshot := queryTask15Snapshots(t, store, period)
	content := strings.Join([]string{
		"Task 15 empty temp DB evidence",
		fmt.Sprintf("empty=%t", dashboardSnapshot.Empty),
		"",
		renderSnapshotForEvidence(dashboardSnapshot, period),
		"",
		"Graph empty views",
		renderTask15GraphViews(graphSnapshot, period),
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

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

func seedTask15Store(t *testing.T) (*sqlite.Store, domain.MonthlyPeriod) {
	t.Helper()
	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: filepath.Join(t.TempDir(), "task15.sqlite3")})
	if err != nil {
		t.Fatalf("sqlite.Bootstrap() error = %v", err)
	}
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	seedTask15Usage(t, store, period)
	return store, period
}

func emptyTask15Store(t *testing.T) (*sqlite.Store, domain.MonthlyPeriod) {
	t.Helper()
	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: filepath.Join(t.TempDir(), "task15-empty.sqlite3")})
	if err != nil {
		t.Fatalf("sqlite.Bootstrap() error = %v", err)
	}
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	return store, period
}

func seedTask15Usage(t *testing.T, store *sqlite.Store, period domain.MonthlyPeriod) {
	t.Helper()
	fixtures := []struct {
		entryID     string
		sessionID   string
		source      domain.UsageSourceKind
		provider    domain.ProviderName
		billingMode domain.BillingMode
		occurredAt  time.Time
		startedAt   time.Time
		projectName string
		agentName   string
		modelID     string
		input       int64
		output      int64
		cacheRead   int64
		cacheWrite  int64
		cost        float64
		metadata    map[string]string
	}{
		{
			entryID: "task15-claude-standard", sessionID: "task15-session-claude-standard", source: domain.UsageSourceCLISession,
			provider: domain.ProviderClaude, billingMode: domain.BillingModeUnknown, occurredAt: period.StartAt.Add(2*24*time.Hour + 10*time.Hour), startedAt: period.StartAt.Add(2*24*time.Hour + 9*time.Hour + 40*time.Minute),
			projectName: "task15-claude-standard", agentName: "claude-code", modelID: "claude-sonnet-4-standard", input: 2100, output: 700, cacheRead: 300, cacheWrite: 100, cost: 0.0345,
			metadata: map[string]string{"claude_session_type": "standard", "project_hash": "task15-claude-standard"},
		},
		{
			entryID: "task15-claude-acp", sessionID: "task15-session-claude-acp", source: domain.UsageSourceCLISession,
			provider: domain.ProviderClaude, billingMode: domain.BillingModeUnknown, occurredAt: period.StartAt.Add(3*24*time.Hour + 11*time.Hour), startedAt: period.StartAt.Add(3*24*time.Hour + 10*time.Hour + 20*time.Minute),
			projectName: "task15-claude-acp", agentName: "claude-code", modelID: "claude-opus-4-acp", input: 3200, output: 900, cacheRead: 500, cacheWrite: 200, cost: 0.0789,
			metadata: map[string]string{"claude_session_type": "acp", "project_hash": "task15-claude-acp"},
		},
		{
			entryID: "task15-openrouter", sessionID: "task15-session-openrouter", source: domain.UsageSourceOpenRouter,
			provider: domain.ProviderOpenRouter, billingMode: domain.BillingModeOpenRouter, occurredAt: period.StartAt.Add(4*24*time.Hour + 12*time.Hour), startedAt: period.StartAt.Add(4*24*time.Hour + 11*time.Hour + 35*time.Minute),
			projectName: "task15-openrouter", agentName: "openrouter", modelID: "anthropic/claude-3.5-sonnet", input: 1200, output: 260, cacheRead: 40, cacheWrite: 0, cost: 0.0123,
			metadata: map[string]string{"project_hash": "task15-openrouter"},
		},
		{
			entryID: "task15-openclaw", sessionID: "task15-session-openclaw", source: domain.UsageSourceCLISession,
			provider: domain.ProviderOpenAI, billingMode: domain.BillingModeBYOK, occurredAt: period.StartAt.Add(5*24*time.Hour + 13*time.Hour), startedAt: period.StartAt.Add(5*24*time.Hour + 12*time.Hour + 25*time.Minute),
			projectName: "task15-openclaw", agentName: "openclaw", modelID: "gpt-4.1-openclaw", input: 1500, output: 500, cacheRead: 90, cacheWrite: 10, cost: 0.0456,
			metadata: map[string]string{"openclaw_record_shape": "jsonl_usage", "project_hash": "task15-openclaw"},
		},
		{
			entryID: "task15-opencode", sessionID: "task15-session-opencode", source: domain.UsageSourceCLISession,
			provider: domain.ProviderOpenCode, billingMode: domain.BillingModeBYOK, occurredAt: period.StartAt.Add(6*24*time.Hour + 14*time.Hour), startedAt: period.StartAt.Add(6*24*time.Hour + 13*time.Hour + 45*time.Minute),
			projectName: "task15-opencode", agentName: "opencode", modelID: "opencode/qwen3-coder", input: 1800, output: 450, cacheRead: 120, cacheWrite: 30, cost: 0.0234,
			metadata: map[string]string{"project_hash": "task15-opencode"},
		},
	}

	sessions := make([]domain.SessionSummary, 0, len(fixtures))
	entries := make([]domain.UsageEntry, 0, len(fixtures))
	for _, fixture := range fixtures {
		ref, err := domain.NewModelPricingRef(fixture.provider, fixture.modelID, fixture.modelID)
		if err != nil {
			t.Fatalf("NewModelPricingRef(%q) error = %v", fixture.modelID, err)
		}
		tokens, err := domain.NewTokenUsage(fixture.input, fixture.output, fixture.cacheRead, fixture.cacheWrite)
		if err != nil {
			t.Fatalf("NewTokenUsage(%q) error = %v", fixture.entryID, err)
		}
		costs, err := domain.NewCostBreakdown(fixture.cost, 0, 0, 0, 0, 0)
		if err != nil {
			t.Fatalf("NewCostBreakdown(%q) error = %v", fixture.entryID, err)
		}
		sessions = append(sessions, mustSessionSummary(t, domain.SessionSummary{
			SessionID:     fixture.sessionID,
			Source:        fixture.source,
			Provider:      fixture.provider,
			BillingMode:   fixture.billingMode,
			StartedAt:     fixture.startedAt,
			EndedAt:       fixture.occurredAt,
			ProjectName:   fixture.projectName,
			AgentName:     fixture.agentName,
			PricingRef:    &ref,
			Tokens:        tokens,
			CostBreakdown: costs,
		}))
		entries = append(entries, mustUsageEntry(t, domain.UsageEntry{
			EntryID:       fixture.entryID,
			Source:        fixture.source,
			Provider:      fixture.provider,
			BillingMode:   fixture.billingMode,
			OccurredAt:    fixture.occurredAt,
			SessionID:     fixture.sessionID,
			ProjectName:   fixture.projectName,
			AgentName:     fixture.agentName,
			Metadata:      fixture.metadata,
			PricingRef:    &ref,
			Tokens:        tokens,
			CostBreakdown: costs,
		}))
	}
	if err := store.UpsertSessions(context.Background(), sessions); err != nil {
		t.Fatalf("UpsertSessions() error = %v", err)
	}
	if err := store.UpsertUsageEntries(context.Background(), entries); err != nil {
		t.Fatalf("UpsertUsageEntries() error = %v", err)
	}
}

func queryTask15Snapshots(t *testing.T, store *sqlite.Store, period domain.MonthlyPeriod) (service.DashboardSnapshot, service.GraphSnapshot) {
	t.Helper()
	dashboardQuery := service.NewDashboardQueryService(store, store, store, store)
	dashboardSnapshot, err := dashboardQuery.QueryDashboard(context.Background(), service.DashboardQuery{Period: period, RecentSessionLimit: 8})
	if err != nil {
		t.Fatalf("QueryDashboard() error = %v", err)
	}
	graphSnapshot, err := service.NewGraphQueryService(store).QueryGraphs(context.Background(), service.GraphQuery{Period: period})
	if err != nil {
		t.Fatalf("QueryGraphs() error = %v", err)
	}
	return dashboardSnapshot, graphSnapshot
}

func assertTask15DashboardSnapshot(t *testing.T, snapshot service.DashboardSnapshot) {
	t.Helper()
	if !nearlyEqual(snapshot.Totals.TotalSpendUSD, task15ExpectedVariableSpend) {
		t.Fatalf("TotalSpendUSD = %.4f, want %.4f", snapshot.Totals.TotalSpendUSD, task15ExpectedVariableSpend)
	}
	if !nearlyEqual(snapshot.Totals.VariableSpendUSD, task15ExpectedVariableSpend) {
		t.Fatalf("VariableSpendUSD = %.4f, want %.4f", snapshot.Totals.VariableSpendUSD, task15ExpectedVariableSpend)
	}
	if snapshot.Empty {
		t.Fatal("snapshot.Empty = true, want false")
	}
	providers := map[domain.ProviderName]service.DashboardProviderSummary{}
	for _, summary := range snapshot.ProviderSummaries {
		providers[summary.Provider] = summary
	}
	for provider, want := range map[domain.ProviderName]float64{
		domain.ProviderClaude:     0.1134,
		domain.ProviderOpenRouter: 0.0123,
		domain.ProviderOpenAI:     0.0456,
		domain.ProviderOpenCode:   0.0234,
	} {
		if got := providers[provider].TotalSpendUSD; !nearlyEqual(got, want) {
			t.Fatalf("provider %s TotalSpendUSD = %.4f, want %.4f", provider, got, want)
		}
	}
	if len(snapshot.RecentSessions) != 5 {
		t.Fatalf("len(RecentSessions) = %d, want 5", len(snapshot.RecentSessions))
	}
}

func assertTask15GraphSnapshot(t *testing.T, snapshot service.GraphSnapshot) {
	t.Helper()
	if len(snapshot.ModelTokenUsages) != 5 {
		t.Fatalf("len(ModelTokenUsages) = %d, want 5", len(snapshot.ModelTokenUsages))
	}
	if len(snapshot.ModelCosts) != 5 {
		t.Fatalf("len(ModelCosts) = %d, want 5", len(snapshot.ModelCosts))
	}
	costs := map[string]float64{}
	tokens := map[string]int64{}
	for _, cost := range snapshot.ModelCosts {
		costs[cost.ModelName] = cost.TotalCostUSD
	}
	for _, usage := range snapshot.ModelTokenUsages {
		tokens[usage.ModelName] = usage.TotalTokens
	}
	for modelName, wantCost := range map[string]float64{
		"anthropic/claude-3.5-sonnet": 0.0123,
		"gpt-4.1-openclaw":            0.0456,
		"claude-opus-4-acp":           0.0789,
	} {
		if got := costs[modelName]; !nearlyEqual(got, wantCost) {
			t.Fatalf("graph cost for %s = %.4f, want %.4f", modelName, got, wantCost)
		}
		if tokens[modelName] == 0 {
			t.Fatalf("graph tokens for %s = 0, want non-zero", modelName)
		}
	}
}

func renderTask15GraphViews(snapshot service.GraphSnapshot, period domain.MonthlyPeriod) string {
	m := newModel(modelDependencies{loader: staticLoader{data: service.DashboardSnapshot{Period: period}}, graphs: staticGraphLoader{data: snapshot}}, period)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 36})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(model)
	updated, _ = m.Update(graphLoadedMsg{data: snapshot})
	m = updated.(model)

	views := make([]string, 0, 4)
	for _, tab := range []graphTab{graphTabModelTokenUsage, graphTabModelCost, graphTabDailyTokenTrend, graphTabModelTokenBreakdown} {
		m.graphTab = tab
		m.syncViewport()
		views = append(views, graphTabLabel(tab), m.View())
	}
	return strings.Join(views, "\n\n")
}

type staticGraphLoader struct {
	data service.GraphSnapshot
	err  error
}

func (s staticGraphLoader) QueryGraphs(context.Context, service.GraphQuery) (service.GraphSnapshot, error) {
	return s.data, s.err
}

func nearlyEqual(got, want float64) bool {
	return math.Abs(got-want) < 0.000001
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
