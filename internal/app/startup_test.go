package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"llm-budget-tracker/internal/adapters/parsers"
	"llm-budget-tracker/internal/adapters/sqlite"
	"llm-budget-tracker/internal/config"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
	"llm-budget-tracker/internal/service"
)

func TestSharedStartupGraph(t *testing.T) {
	t.Parallel()

	anchor := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)
	homeDir := t.TempDir()
	paths := testPaths(t)
	period := mustMonthlyPeriod(t, anchor.Year(), anchor.Month())
	writeSettings(t, paths, func(settings config.Settings) config.Settings {
		settings.Providers.OpenRouter.Enabled = false
		return settings
	})
	seedSharedStartupDatabase(t, paths.DatabaseFile, period, anchor)
	claudeRoot := writeClaudeStartupFixture(t, homeDir)

	watcherA := newStubWatcher()
	guiGraph, err := Start(context.Background(), Options{
		Paths:          paths,
		HomeDir:        homeDir,
		Notifier:       noopNotifier{},
		WatcherFactory: func() (service.FileWatcher, error) { return watcherA, nil },
		WatchTargets:   []service.WatchTarget{service.NewClaudeWatchTarget(claudeRoot, parsers.NewClaudeCodeParser())},
		Now:            func() time.Time { return anchor },
	})
	if err != nil {
		t.Fatalf("Start(gui) error = %v", err)
	}
	defer guiGraph.Close()

	watcherB := newStubWatcher()
	tuiGraph, err := Start(context.Background(), Options{
		Paths:          paths,
		HomeDir:        homeDir,
		Notifier:       noopNotifier{},
		WatcherFactory: func() (service.FileWatcher, error) { return watcherB, nil },
		WatchTargets:   []service.WatchTarget{service.NewClaudeWatchTarget(claudeRoot, parsers.NewClaudeCodeParser())},
		Now:            func() time.Time { return anchor },
	})
	if err != nil {
		t.Fatalf("Start(tui) error = %v", err)
	}
	defer tuiGraph.Close()

	guiSnapshot, err := guiGraph.DashboardQueryService.QueryDashboard(context.Background(), service.DashboardQuery{Period: period})
	if err != nil {
		t.Fatalf("gui QueryDashboard() error = %v", err)
	}
	tuiSnapshot, err := tuiGraph.DashboardQueryService.QueryDashboard(context.Background(), service.DashboardQuery{Period: period})
	if err != nil {
		t.Fatalf("tui QueryDashboard() error = %v", err)
	}

	assertFloatEquals(t, guiSnapshot.Totals.TotalSpendUSD, tuiSnapshot.Totals.TotalSpendUSD)
	assertFloatEquals(t, guiSnapshot.Totals.TotalSpendUSD, 26.5)
	assertFloatEquals(t, guiSnapshot.Totals.SubscriptionSpendUSD, 20)
	assertFloatEquals(t, guiSnapshot.Totals.VariableSpendUSD, 6.5)

	guiInsights, err := guiGraph.Store.ListInsights(context.Background(), period)
	if err != nil {
		t.Fatalf("gui ListInsights() error = %v", err)
	}
	tuiInsights, err := tuiGraph.Store.ListInsights(context.Background(), period)
	if err != nil {
		t.Fatalf("tui ListInsights() error = %v", err)
	}
	if len(guiInsights) == 0 {
		t.Fatal("len(gui insights) = 0, want detector output from startup refresh")
	}
	if len(guiInsights) != len(tuiInsights) {
		t.Fatalf("insight count mismatch: gui=%d tui=%d", len(guiInsights), len(tuiInsights))
	}

	guiAlerts, err := guiGraph.Store.ListAlerts(context.Background(), ports.AlertFilter{Period: &period})
	if err != nil {
		t.Fatalf("gui ListAlerts() error = %v", err)
	}
	tuiAlerts, err := tuiGraph.Store.ListAlerts(context.Background(), ports.AlertFilter{Period: &period})
	if err != nil {
		t.Fatalf("tui ListAlerts() error = %v", err)
	}
	if len(guiAlerts) == 0 {
		t.Fatal("len(gui alerts) = 0, want budget monitor output from startup refresh")
	}
	if len(guiAlerts) != len(tuiAlerts) {
		t.Fatalf("alert count mismatch: gui=%d tui=%d", len(guiAlerts), len(tuiAlerts))
	}

	if len(guiSnapshot.Budgets) != len(tuiSnapshot.Budgets) {
		t.Fatalf("budget summary count mismatch: gui=%d tui=%d", len(guiSnapshot.Budgets), len(tuiSnapshot.Budgets))
	}
	if len(guiSnapshot.Budgets) == 0 || !guiSnapshot.Budgets[0].BudgetOverrunActive {
		t.Fatalf("gui budget summaries = %+v, want active overrun state", guiSnapshot.Budgets)
	}
	if len(tuiSnapshot.Budgets) == 0 || !tuiSnapshot.Budgets[0].BudgetOverrunActive {
		t.Fatalf("tui budget summaries = %+v, want active overrun state", tuiSnapshot.Budgets)
	}

	if warnings := guiGraph.Warnings(); len(warnings) != 0 {
		t.Fatalf("gui warnings = %v, want none", warnings)
	}
	if warnings := tuiGraph.Warnings(); len(warnings) != 0 {
		t.Fatalf("tui warnings = %v, want none", warnings)
	}
}

func TestStartupGracefulDegradation(t *testing.T) {
	t.Parallel()

	anchor := time.Date(2026, time.March, 20, 9, 0, 0, 0, time.UTC)
	paths := testPaths(t)
	period := mustMonthlyPeriod(t, anchor.Year(), anchor.Month())
	writeSettings(t, paths, func(settings config.Settings) config.Settings {
		settings.Providers.OpenRouter.Enabled = true
		settings.Notifications.BudgetWarnings = true
		settings.Notifications.ForecastWarnings = true
		return settings
	})
	seedDegradationDatabase(t, paths.DatabaseFile, period, anchor)

	watcher := newStubWatcher()
	notifier := failingNotifier{}
	graph, err := Start(context.Background(), Options{
		Paths:          paths,
		SecretStore:    emptySecretStore{},
		Notifier:       notifier,
		WatcherFactory: func() (service.FileWatcher, error) { return watcher, nil },
		WatchTargets:   []service.WatchTarget{service.NewGeminiWatchTarget(filepath.Join(t.TempDir(), ".gemini"), parsers.NewGeminiCLIParser())},
		Now:            func() time.Time { return anchor },
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	snapshot, err := graph.DashboardQueryService.QueryDashboard(context.Background(), service.DashboardQuery{Period: period})
	if err != nil {
		graph.Close()
		t.Fatalf("QueryDashboard() error = %v", err)
	}
	assertFloatEquals(t, snapshot.Totals.TotalSpendUSD, 12)

	alerts, err := graph.Store.ListAlerts(context.Background(), ports.AlertFilter{Period: &period})
	if err != nil {
		graph.Close()
		t.Fatalf("ListAlerts() error = %v", err)
	}
	if len(alerts) == 0 {
		graph.Close()
		t.Fatal("len(alerts) = 0, want persisted alert state despite notifier failure")
	}

	warnings := graph.Warnings()
	assertContainsWarning(t, warnings, "OpenRouter sync is disabled until provider.openrouter.api_key is configured")
	assertContainsWarning(t, warnings, "watch target gemini-cli unavailable")
	assertContainsWarning(t, warnings, "alert notification failed")

	if err := graph.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := graph.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if watcher.closeCount() != 1 {
		t.Fatalf("watcher close count = %d, want 1", watcher.closeCount())
	}
}

func TestStartupSyncsEnabledConfiguredSubscriptions(t *testing.T) {
	t.Parallel()

	anchor := time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC)
	paths := testPaths(t)
	writeSettings(t, paths, func(settings config.Settings) config.Settings {
		settings.Providers.OpenRouter.Enabled = false
		settings.SubscriptionDefaults.OpenAI.Enabled = true
		settings.SubscriptionDefaults.OpenAI.RenewalDay = 5
		settings.SubscriptionDefaults.Claude.Enabled = false
		settings.SubscriptionDefaults.Gemini.Enabled = false
		return settings
	})

	graph, err := Start(context.Background(), Options{
		Paths:          paths,
		HomeDir:        t.TempDir(),
		Notifier:       noopNotifier{},
		SecretStore:    emptySecretStore{},
		WatcherFactory: func() (service.FileWatcher, error) { return newStubWatcher(), nil },
		Now:            func() time.Time { return anchor },
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer graph.Close()
	if !graph.Settings.SubscriptionDefaults.OpenAI.Enabled {
		t.Fatalf("graph settings openai subscription defaults = %+v, want enabled", graph.Settings.SubscriptionDefaults.OpenAI)
	}

	subscriptions, err := graph.Store.ListSubscriptions(context.Background(), ports.SubscriptionFilter{SubscriptionID: "settings-openai-subscription"})
	if err != nil {
		t.Fatalf("ListSubscriptions() error = %v", err)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("len(subscriptions) = %d, want 1", len(subscriptions))
	}
	if got := subscriptions[0].PlanCode; got != "chatgpt-plus" {
		t.Fatalf("PlanCode = %q, want chatgpt-plus", got)
	}
	if got := subscriptions[0].FeeUSD; got != 20 {
		t.Fatalf("FeeUSD = %v, want 20", got)
	}
	if got := subscriptions[0].RenewalDay; got != 5 {
		t.Fatalf("RenewalDay = %d, want 5", got)
	}
	if subscriptions[0].StartsAt.After(anchor) {
		t.Fatalf("StartsAt = %v, want not after anchor %v", subscriptions[0].StartsAt, anchor)
	}

	active := true
	claudeSubscriptions, err := graph.Store.ListSubscriptions(context.Background(), ports.SubscriptionFilter{SubscriptionID: "settings-claude-subscription", Active: &active})
	if err != nil {
		t.Fatalf("ListSubscriptions(claude) error = %v", err)
	}
	if len(claudeSubscriptions) != 0 {
		t.Fatalf("claude subscriptions = %+v, want none when disabled", claudeSubscriptions)
	}

	period, err := domain.NewMonthlyPeriod(anchor)
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	snapshot, err := graph.DashboardQueryService.QueryDashboard(context.Background(), service.DashboardQuery{Period: period})
	if err != nil {
		t.Fatalf("QueryDashboard() error = %v", err)
	}
	assertFloatEquals(t, snapshot.Totals.SubscriptionSpendUSD, 20)
}

func TestStartupBootstrapOnlySkipsRefreshAndWatchers(t *testing.T) {
	t.Parallel()

	anchor := time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC)
	paths := testPaths(t)
	writeSettings(t, paths, func(settings config.Settings) config.Settings {
		settings.Providers.OpenRouter.Enabled = true
		settings.Notifications.BudgetWarnings = true
		settings.Notifications.ForecastWarnings = true
		settings.SubscriptionDefaults.OpenAI.Enabled = true
		settings.SubscriptionDefaults.OpenAI.RenewalDay = 5
		return settings
	})

	watcherCalls := 0
	graph, err := Start(context.Background(), Options{
		Paths:         paths,
		BootstrapOnly: true,
		HomeDir:       t.TempDir(),
		Notifier:      noopNotifier{},
		SecretStore:   emptySecretStore{},
		WatcherFactory: func() (service.FileWatcher, error) {
			watcherCalls++
			return newStubWatcher(), nil
		},
		WatchTargets: []service.WatchTarget{service.NewClaudeWatchTarget(filepath.Join(t.TempDir(), ".claude"), parsers.NewClaudeCodeParser())},
		Now:          func() time.Time { return anchor },
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer graph.Close()

	if watcherCalls != 0 {
		t.Fatalf("watcherCalls = %d, want 0", watcherCalls)
	}
	if graph.WatchCoordinator != nil {
		t.Fatal("WatchCoordinator != nil, want nil in bootstrap-only mode")
	}
	if warnings := graph.Warnings(); len(warnings) != 0 {
		t.Fatalf("Warnings() = %v, want none in bootstrap-only mode", warnings)
	}

	period := mustMonthlyPeriod(t, anchor.Year(), anchor.Month())
	insights, err := graph.Store.ListInsights(context.Background(), period)
	if err != nil {
		t.Fatalf("ListInsights() error = %v", err)
	}
	if len(insights) != 0 {
		t.Fatalf("len(insights) = %d, want 0 when refresh is skipped", len(insights))
	}

	alerts, err := graph.Store.ListAlerts(context.Background(), ports.AlertFilter{Period: &period})
	if err != nil {
		t.Fatalf("ListAlerts() error = %v", err)
	}
	if len(alerts) != 0 {
		t.Fatalf("len(alerts) = %d, want 0 when refresh is skipped", len(alerts))
	}

	subscriptions, err := graph.Store.ListSubscriptions(context.Background(), ports.SubscriptionFilter{SubscriptionID: "settings-openai-subscription"})
	if err != nil {
		t.Fatalf("ListSubscriptions() error = %v", err)
	}
	if len(subscriptions) != 0 {
		t.Fatalf("len(subscriptions) = %d, want 0 when configured subscription sync is skipped", len(subscriptions))
	}
}

type stubWatcher struct {
	events chan service.FileWatchEvent
	errors chan error

	mu         sync.Mutex
	additions  []string
	closeCalls int
}

func newStubWatcher() *stubWatcher {
	return &stubWatcher{
		events: make(chan service.FileWatchEvent),
		errors: make(chan error),
	}
}

func (w *stubWatcher) Add(name string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.additions = append(w.additions, name)
	return nil
}

func (w *stubWatcher) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closeCalls == 0 {
		close(w.events)
		close(w.errors)
	}
	w.closeCalls++
	return nil
}

func (w *stubWatcher) Events() <-chan service.FileWatchEvent { return w.events }

func (w *stubWatcher) Errors() <-chan error { return w.errors }

func (w *stubWatcher) closeCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closeCalls
}

type noopNotifier struct{}

func (noopNotifier) NotifyAlert(context.Context, domain.AlertEvent) error { return nil }

type failingNotifier struct{}

func (failingNotifier) NotifyAlert(context.Context, domain.AlertEvent) error {
	return fmt.Errorf("notifier transport unavailable")
}

type emptySecretStore struct{}

func (emptySecretStore) Set(config.SecretID, string) error { return nil }

func (emptySecretStore) Get(config.SecretID) (string, error) { return "", nil }

func (emptySecretStore) Delete(config.SecretID) error { return nil }

func testPaths(t *testing.T) config.Paths {
	t.Helper()
	root := t.TempDir()
	configDir := filepath.Join(root, "config", config.AppDirectoryName)
	dataDir := filepath.Join(root, "data", config.AppDirectoryName)
	return config.Paths{
		ConfigDir:          configDir,
		DataDir:            dataDir,
		SettingsFile:       filepath.Join(configDir, config.SettingsFileName),
		PricesOverrideFile: filepath.Join(configDir, config.PricesOverrideFileName),
		DatabaseFile:       filepath.Join(dataDir, config.DatabaseFileName),
	}
}

func writeSettings(t *testing.T, paths config.Paths, mutate func(config.Settings) config.Settings) {
	t.Helper()
	store := config.NewSettingsStore(paths)
	settings, err := store.Bootstrap()
	if err != nil {
		t.Fatalf("Bootstrap(settings) error = %v", err)
	}
	if mutate != nil {
		settings = mutate(settings)
	}
	if err := store.Save(settings); err != nil {
		t.Fatalf("Save(settings) error = %v", err)
	}
}

func seedSharedStartupDatabase(t *testing.T, dbPath string, period domain.MonthlyPeriod, anchor time.Time) {
	t.Helper()
	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: dbPath})
	if err != nil {
		t.Fatalf("sqlite.Bootstrap() error = %v", err)
	}
	defer store.Close()

	manualRef := mustPricingRef(t, domain.ProviderOpenAI, "gpt-4.1")
	manualCost, err := domain.NewCostBreakdown(2.5, 1.5, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}
	manualTokens := mustTokenUsage(t, 1500, 800, 0, 0)
	manualEntry := mustUsageEntry(t, domain.UsageEntry{
		EntryID:       "manual-shared-1",
		Source:        domain.UsageSourceManualAPI,
		Provider:      domain.ProviderOpenAI,
		BillingMode:   domain.BillingModeDirectAPI,
		OccurredAt:    anchor.Add(-48 * time.Hour),
		ProjectName:   "acme-app",
		Metadata:      map[string]string{"project_hash": "proj-acme"},
		PricingRef:    &manualRef,
		Tokens:        manualTokens,
		CostBreakdown: manualCost,
	})
	if err := store.UpsertUsageEntries(context.Background(), []domain.UsageEntry{manualEntry}); err != nil {
		t.Fatalf("UpsertUsageEntries(manual) error = %v", err)
	}

	subscription := mustSubscription(t, domain.Subscription{
		SubscriptionID: "sub-claude-max",
		Provider:       domain.ProviderClaude,
		PlanCode:       "claude-max",
		PlanName:       "Claude Max",
		RenewalDay:     3,
		StartsAt:       time.Date(period.StartAt.Year(), period.StartAt.Month(), 3, 8, 0, 0, 0, time.UTC),
		FeeUSD:         20,
		IsActive:       true,
		CreatedAt:      anchor.Add(-10 * 24 * time.Hour),
		UpdatedAt:      anchor.Add(-10 * 24 * time.Hour),
	})
	if err := store.UpsertSubscriptions(context.Background(), []domain.Subscription{subscription}); err != nil {
		t.Fatalf("UpsertSubscriptions() error = %v", err)
	}

	budget := mustMonthlyBudget(t, domain.MonthlyBudget{
		BudgetID: "budget-global-shared",
		Name:     "Global Budget",
		Period:   period,
		LimitUSD: 5,
		Thresholds: []domain.BudgetThreshold{
			mustBudgetThreshold(t, domain.AlertSeverityWarning, 0.8),
		},
		Currency: "USD",
	})
	if err := store.UpsertMonthlyBudgets(context.Background(), []domain.MonthlyBudget{budget}); err != nil {
		t.Fatalf("UpsertMonthlyBudgets() error = %v", err)
	}
}

func seedDegradationDatabase(t *testing.T, dbPath string, period domain.MonthlyPeriod, anchor time.Time) {
	t.Helper()
	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: dbPath})
	if err != nil {
		t.Fatalf("sqlite.Bootstrap() error = %v", err)
	}
	defer store.Close()

	ref := mustPricingRef(t, domain.ProviderOpenAI, "gpt-4.1")
	costs := mustCostBreakdown(t, 7, 5, 0, 0, 0, 0)
	entry := mustUsageEntry(t, domain.UsageEntry{
		EntryID:       "manual-degradation-1",
		Source:        domain.UsageSourceManualAPI,
		Provider:      domain.ProviderOpenAI,
		BillingMode:   domain.BillingModeDirectAPI,
		OccurredAt:    anchor.Add(-24 * time.Hour),
		ProjectName:   "degradation-app",
		Metadata:      map[string]string{"project_hash": "proj-degrade"},
		PricingRef:    &ref,
		Tokens:        mustTokenUsage(t, 3000, 1200, 0, 0),
		CostBreakdown: costs,
	})
	if err := store.UpsertUsageEntries(context.Background(), []domain.UsageEntry{entry}); err != nil {
		t.Fatalf("UpsertUsageEntries() error = %v", err)
	}

	budget := mustMonthlyBudget(t, domain.MonthlyBudget{
		BudgetID: "budget-global-degradation",
		Name:     "Global Budget",
		Period:   period,
		LimitUSD: 10,
		Thresholds: []domain.BudgetThreshold{
			mustBudgetThreshold(t, domain.AlertSeverityWarning, 0.8),
		},
		Currency: "USD",
	})
	if err := store.UpsertMonthlyBudgets(context.Background(), []domain.MonthlyBudget{budget}); err != nil {
		t.Fatalf("UpsertMonthlyBudgets() error = %v", err)
	}
}

func writeClaudeStartupFixture(t *testing.T, homeDir string) string {
	t.Helper()
	root := filepath.Join(homeDir, ".config", "claude")
	sessionDir := filepath.Join(root, "projects", "acme-app", "sessions")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", sessionDir, err)
	}

	row := `{"cwd":"/workspace/acme-app","sessionId":"claude-startup-session","timestamp":"2026-03-10T10:00:00Z","requestId":"req-startup-1","costUSD":2.5,"message":{"id":"msg-startup-1","model":"claude-sonnet-4","usage":{"input_tokens":5000,"output_tokens":200,"cache_creation_input_tokens":0,"cache_read_input_tokens":0},"content":[{"type":"tool_use"}]}}` + "\n"
	logPath := filepath.Join(sessionDir, "startup-session.jsonl")
	if err := os.WriteFile(logPath, []byte(row), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", logPath, err)
	}

	return root
}

func mustMonthlyPeriod(t *testing.T, year int, month time.Month) domain.MonthlyPeriod {
	t.Helper()
	period, err := domain.NewMonthlyPeriodFromParts(year, month)
	if err != nil {
		t.Fatalf("NewMonthlyPeriodFromParts() error = %v", err)
	}
	return period
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

func mustUsageEntry(t *testing.T, entry domain.UsageEntry) domain.UsageEntry {
	t.Helper()
	validated, err := domain.NewUsageEntry(entry)
	if err != nil {
		t.Fatalf("NewUsageEntry() error = %v", err)
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

func assertFloatEquals(t *testing.T, got, want float64) {
	t.Helper()
	if diff := got - want; diff > 0.0000001 || diff < -0.0000001 {
		t.Fatalf("float mismatch: got %.6f want %.6f", got, want)
	}
}

func assertContainsWarning(t *testing.T, warnings []string, needle string) {
	t.Helper()
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return
		}
	}
	t.Fatalf("warnings %v do not contain %q", warnings, needle)
}
