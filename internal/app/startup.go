package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	adapterfsnotify "llm-budget-tracker/internal/adapters/fsnotify"
	"llm-budget-tracker/internal/adapters/openrouter"
	"llm-budget-tracker/internal/adapters/parsers"
	"llm-budget-tracker/internal/adapters/sqlite"
	catalogpkg "llm-budget-tracker/internal/catalog"
	"llm-budget-tracker/internal/config"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
	"llm-budget-tracker/internal/service"

	"github.com/zalando/go-keyring"
)

type Options struct {
	Paths               config.Paths
	PathResolverOptions config.PathResolverOptions
	DatabasePath        string
	HomeDir             string
	SecretStore         config.SecretStore
	Notifier            ports.AlertNotifier
	WatcherFactory      func() (service.FileWatcher, error)
	WatchTargets        []service.WatchTarget
	Now                 func() time.Time
}

type Graph struct {
	Paths                  config.Paths
	Settings               config.Settings
	SettingsStore          *config.SettingsStore
	SecretStore            config.SecretStore
	Store                  *sqlite.Store
	Catalog                *catalogpkg.Catalog
	DashboardQueryService  *service.DashboardQueryService
	SettingsService        *service.SettingsService
	SubscriptionService    *service.SubscriptionService
	ManualEntryService     *service.ManualAPIUsageEntryService
	MonthlyBudgetService   *service.MonthlyBudgetService
	InsightExecutorService *service.InsightExecutorService
	BudgetMonitorService   *service.BudgetMonitorService
	WatchCoordinator       *service.WatchCoordinator

	openRouterCatalogSync  *service.CatalogSyncService
	openRouterActivitySync *service.OpenRouterActivitySyncService
	rawNotifier            ports.AlertNotifier
	warningRecorder        *warningRecorder
	now                    func() time.Time
	cancel                 context.CancelFunc
	backgroundWG           sync.WaitGroup
	closeOnce              sync.Once
}

func Start(ctx context.Context, opts Options) (*Graph, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	paths, err := resolvePaths(opts)
	if err != nil {
		return nil, err
	}

	settingsStore := config.NewSettingsStore(paths)
	settings, err := settingsStore.Bootstrap()
	if err != nil {
		return nil, err
	}

	recorder := newWarningRecorder()
	secretStore := opts.SecretStore
	if secretStore == nil {
		resolvedSecretStore, secretErr := config.NewOSKeyringSecretStore()
		if secretErr != nil {
			recorder.Add(secretErr.Error())
		} else {
			secretStore = resolvedSecretStore
		}
	}

	startupCtx, cancel := context.WithCancel(ctx)

	store, err := sqlite.BootstrapFromPaths(startupCtx, paths, sqlite.Options{})
	if err != nil {
		cancel()
		return nil, err
	}

	catalog, err := catalogpkg.New(catalogpkg.Options{OverridePath: paths.PricesOverrideFile})
	if err != nil {
		_ = store.Close()
		cancel()
		return nil, err
	}
	for _, warning := range catalog.Warnings() {
		recorder.Add(warning.Error())
	}

	clock := opts.Now
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}

	settings, err = service.NormalizeSettings(settings, config.DefaultSettings())
	if err != nil {
		_ = store.Close()
		cancel()
		return nil, err
	}

	normalizer := service.NewSessionNormalizerService(store, store, store)
	queryService := service.NewDashboardQueryService(store, store, store, store)
	settingsService := service.NewSettingsService(settingsStore, secretStore)
	subscriptionService := service.NewSubscriptionService(store, store)
	manualEntryService := service.NewManualAPIUsageEntryService(catalog, store)
	monthlyBudgetService := service.NewMonthlyBudgetService(store)
	detectors := make([]ports.InsightDetector, 0, 8)
	detectors = append(detectors, service.NewDetectorSetA()...)
	detectors = append(detectors, service.NewDetectorSetB()...)
	detectors = append(detectors, service.NewOverQualifiedModelDetector(catalog))
	detectors = append(detectors, service.NewToolSchemaBloatDetector(catalog))
	insightExecutor := service.NewInsightExecutorService(detectors, store, store, store)
	safeNotifier := &warningNotifier{base: opts.Notifier, warnings: recorder}
	budgetMonitor := service.NewBudgetMonitorService(settings, store, store, store, store, store, safeNotifier)
	for _, warning := range syncConfiguredSubscriptions(startupCtx, subscriptionService, settings, clock().UTC()) {
		recorder.Add(warning)
	}

	graph := &Graph{
		Paths:                  paths,
		Settings:               settings,
		SettingsStore:          settingsStore,
		SecretStore:            secretStore,
		Store:                  store,
		Catalog:                catalog,
		DashboardQueryService:  queryService,
		SettingsService:        settingsService,
		SubscriptionService:    subscriptionService,
		ManualEntryService:     manualEntryService,
		MonthlyBudgetService:   monthlyBudgetService,
		InsightExecutorService: insightExecutor,
		BudgetMonitorService:   budgetMonitor,
		rawNotifier:            opts.Notifier,
		warningRecorder:        recorder,
		now:                    clock,
		cancel:                 cancel,
	}

	graph.openRouterCatalogSync, graph.openRouterActivitySync = buildOpenRouterServices(settings, secretStore, catalog, normalizer, recorder)

	if coordinator := buildWatchCoordinator(startupCtx, opts, settings, normalizer, store, recorder); coordinator != nil {
		graph.WatchCoordinator = coordinator
		graph.backgroundWG.Add(1)
		go func() {
			defer graph.backgroundWG.Done()
			for err := range coordinator.Errors() {
				if err == nil {
					continue
				}
				recorder.Add(err.Error())
			}
		}()
	}

	if err := graph.Refresh(startupCtx); err != nil {
		_ = graph.Close()
		return nil, err
	}

	return graph, nil
}

func (g *Graph) Refresh(ctx context.Context) error {
	if g == nil {
		return fmt.Errorf("startup graph is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if g.openRouterCatalogSync != nil {
		if _, err := g.openRouterCatalogSync.Sync(ctx); err != nil {
			g.warningRecorder.Add(fmt.Sprintf("OpenRouter catalog sync warning: %v", err))
		}
	}

	if g.openRouterActivitySync != nil {
		if _, err := g.openRouterActivitySync.Sync(ctx, ports.OpenRouterActivityOptions{}); err != nil {
			g.warningRecorder.Add(fmt.Sprintf("OpenRouter activity sync warning: %v", err))
		}
	}

	period, err := domain.NewMonthlyPeriod(g.now().UTC())
	if err != nil {
		return err
	}

	if g.InsightExecutorService != nil {
		if _, err := g.InsightExecutorService.Execute(ctx, period); err != nil {
			g.warningRecorder.Add(fmt.Sprintf("insight execution warning: %v", err))
		}
	}

	if g.BudgetMonitorService != nil {
		if _, err := g.BudgetMonitorService.MonitorPeriod(ctx, period); err != nil {
			g.warningRecorder.Add(fmt.Sprintf("budget monitoring warning: %v", err))
		}
	}

	return nil
}

func (g *Graph) RawNotifier() ports.AlertNotifier {
	if g == nil {
		return nil
	}
	return g.rawNotifier
}

func (g *Graph) Warnings() []string {
	if g == nil {
		return nil
	}

	merged := append([]string{}, g.warningRecorder.All()...)
	if g.WatchCoordinator != nil {
		merged = append(merged, g.WatchCoordinator.Warnings()...)
	}
	return dedupeWarnings(merged)
}

func (g *Graph) Close() error {
	if g == nil {
		return nil
	}

	var closeErr error
	g.closeOnce.Do(func() {
		if g.cancel != nil {
			g.cancel()
		}
		if g.WatchCoordinator != nil {
			if err := g.WatchCoordinator.Close(); err != nil && closeErr == nil {
				closeErr = err
			}
		}
		g.backgroundWG.Wait()
		if g.Store != nil {
			if err := g.Store.Close(); err != nil && closeErr == nil {
				closeErr = err
			}
		}
	})

	return closeErr
}

func resolvePaths(opts Options) (config.Paths, error) {
	if strings.TrimSpace(opts.Paths.ConfigDir) != "" || strings.TrimSpace(opts.Paths.DatabaseFile) != "" {
		paths := opts.Paths
		if strings.TrimSpace(opts.DatabasePath) != "" {
			paths.DatabaseFile = strings.TrimSpace(opts.DatabasePath)
		}
		return paths, nil
	}

	paths, err := config.ResolvePaths(opts.PathResolverOptions)
	if err != nil {
		return config.Paths{}, err
	}
	if strings.TrimSpace(opts.DatabasePath) != "" {
		paths.DatabaseFile = strings.TrimSpace(opts.DatabasePath)
	}
	return paths, nil
}

func buildOpenRouterServices(settings config.Settings, secretStore config.SecretStore, catalog *catalogpkg.Catalog, normalizer *service.SessionNormalizerService, warnings *warningRecorder) (*service.CatalogSyncService, *service.OpenRouterActivitySyncService) {
	if !settings.Providers.OpenRouter.Enabled {
		return nil, nil
	}

	apiKey := ""
	if secretStore != nil {
		loadedKey, err := secretStore.Get(config.SecretOpenRouterAPIKey)
		if err != nil {
			if !errors.Is(err, keyring.ErrNotFound) {
				warnings.Add(fmt.Sprintf("OpenRouter secret load warning: %v", err))
			}
		} else {
			apiKey = strings.TrimSpace(loadedKey)
		}
	}

	client := openrouter.NewClient(openrouter.Options{APIKey: apiKey})
	if warning := client.WarningState(); warning != nil {
		warnings.Add(warning.Error())
		return nil, nil
	}

	return service.NewCatalogSyncService(client, catalog), service.NewOpenRouterActivitySyncService(client, normalizer)
}

func syncConfiguredSubscriptions(ctx context.Context, subscriptions *service.SubscriptionService, settings config.Settings, now time.Time) []string {
	if subscriptions == nil {
		return nil
	}

	configs := []struct {
		subscriptionID string
		provider       domain.ProviderName
		plan           config.SubscriptionPlanSettings
	}{
		{subscriptionID: "settings-openai-subscription", provider: domain.ProviderOpenAI, plan: settings.SubscriptionDefaults.OpenAI},
		{subscriptionID: "settings-claude-subscription", provider: domain.ProviderClaude, plan: settings.SubscriptionDefaults.Claude},
		{subscriptionID: "settings-gemini-subscription", provider: domain.ProviderGemini, plan: settings.SubscriptionDefaults.Gemini},
	}

	warnings := make([]string, 0)
	for _, item := range configs {
		current, err := subscriptions.ListSubscriptions(ctx, ports.SubscriptionFilter{SubscriptionID: item.subscriptionID})
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("settings subscription sync warning for %s: %v", item.subscriptionID, err))
			continue
		}

		if !item.plan.Enabled {
			if len(current) == 0 || !current[0].IsActive {
				continue
			}
			if err := subscriptions.DisableSubscription(ctx, item.subscriptionID, now); err != nil {
				warnings = append(warnings, fmt.Sprintf("settings subscription disable warning for %s: %v", item.subscriptionID, err))
			}
			continue
		}

		subscription, err := buildConfiguredSubscription(item.subscriptionID, item.provider, item.plan, current, now)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("settings subscription build warning for %s: %v", item.subscriptionID, err))
			continue
		}
		if err := subscriptions.SaveSubscriptions(ctx, []domain.Subscription{subscription}); err != nil {
			warnings = append(warnings, fmt.Sprintf("settings subscription save warning for %s: %v", item.subscriptionID, err))
		}
	}

	return warnings
}

func buildConfiguredSubscription(subscriptionID string, provider domain.ProviderName, plan config.SubscriptionPlanSettings, current []domain.Subscription, now time.Time) (domain.Subscription, error) {
	createdAt := time.Time{}
	startsAt := mostRecentRenewalAt(now, plan.RenewalDay)
	if len(current) > 0 {
		if !current[0].CreatedAt.IsZero() {
			createdAt = current[0].CreatedAt
		}
		if !current[0].StartsAt.IsZero() {
			startsAt = current[0].StartsAt
		}
	}

	subscription := domain.Subscription{
		SubscriptionID: subscriptionID,
		Provider:       provider,
		PlanCode:       plan.PlanCode,
		PlanName:       plan.PlanName,
		RenewalDay:     plan.RenewalDay,
		StartsAt:       startsAt,
		FeeUSD:         plan.FeeUSD,
		IsActive:       true,
		CreatedAt:      createdAt,
		UpdatedAt:      now,
	}

	return subscription, nil
}

func mostRecentRenewalAt(now time.Time, renewalDay int) time.Time {
	anchor := now.UTC()
	day := minInt(renewalDay, daysInMonth(anchor.Year(), anchor.Month()))
	candidate := time.Date(anchor.Year(), anchor.Month(), day, 0, 0, 0, 0, time.UTC)
	if candidate.After(anchor) {
		previousMonth := anchor.AddDate(0, -1, 0)
		day = minInt(renewalDay, daysInMonth(previousMonth.Year(), previousMonth.Month()))
		candidate = time.Date(previousMonth.Year(), previousMonth.Month(), day, 0, 0, 0, 0, time.UTC)
	}
	return candidate
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func buildWatchCoordinator(ctx context.Context, opts Options, settings config.Settings, normalizer *service.SessionNormalizerService, checkpoints ports.CheckpointRepository, warnings *warningRecorder) *service.WatchCoordinator {
	targets := opts.WatchTargets
	if len(targets) == 0 {
		targets = defaultWatchTargets(opts.HomeDir, settings, warnings)
	}
	if len(targets) == 0 {
		return nil
	}

	watcherFactory := opts.WatcherFactory
	if watcherFactory == nil {
		watcherFactory = func() (service.FileWatcher, error) {
			return adapterfsnotify.NewWatcher()
		}
	}

	watcher, err := watcherFactory()
	if err != nil {
		warnings.Add(fmt.Sprintf("file watcher startup warning: %v", err))
		return nil
	}

	coordinator, err := service.NewWatchCoordinator(normalizer, checkpoints, watcher, targets)
	if err != nil {
		warnings.Add(fmt.Sprintf("watch coordinator startup warning: %v", err))
		_ = watcher.Close()
		return nil
	}

	if err := coordinator.Start(ctx); err != nil {
		warnings.Add(fmt.Sprintf("watch coordinator start warning: %v", err))
		_ = coordinator.Close()
		return nil
	}

	return coordinator
}

func defaultWatchTargets(homeDir string, settings config.Settings, warnings *warningRecorder) []service.WatchTarget {
	resolvedHome := strings.TrimSpace(homeDir)
	if resolvedHome == "" {
		lookup, err := os.UserHomeDir()
		if err != nil {
			warnings.Add(fmt.Sprintf("watch target discovery warning: could not resolve user home directory: %v", err))
			return nil
		}
		resolvedHome = lookup
	}

	claudeParser := newBillingModeFallbackParser(parsers.NewClaudeCodeParser(), settings.CLIBillingDefaults.ClaudeCode)
	codexParser := parsers.NewCodexParser()
	geminiParser := newBillingModeFallbackParser(parsers.NewGeminiCLIParser(), settings.CLIBillingDefaults.GeminiCLI)
	openCodeParser := parsers.NewOpenCodeParser()

	targets := []service.WatchTarget{
		service.NewClaudeWatchTarget(filepath.Join(resolvedHome, ".config", "claude"), claudeParser),
		service.NewClaudeWatchTarget(filepath.Join(resolvedHome, ".claude"), claudeParser),
		service.NewCodexWatchTarget(filepath.Join(resolvedHome, ".codex"), codexParser),
		service.NewOpenCodeWatchTarget(filepath.Join(resolvedHome, ".local", "share", "opencode"), openCodeParser),
	}

	geminiRoot := filepath.Join(resolvedHome, ".gemini")
	geminiStatus := parsers.ProbeGeminiCLIPath(geminiRoot)
	if geminiStatus.Supported() {
		targets = append(targets, service.NewGeminiWatchTarget(geminiRoot, geminiParser))
	} else {
		warnings.Add(formatGeminiStatusWarning(geminiStatus))
	}

	return targets
}

type warningRecorder struct {
	mu       sync.Mutex
	warnings []string
}

func newWarningRecorder() *warningRecorder {
	return &warningRecorder{warnings: make([]string, 0, 8)}
}

func (r *warningRecorder) Add(message string) {
	if r == nil {
		return
	}
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.warnings = append(r.warnings, trimmed)
}

func (r *warningRecorder) All() []string {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	clone := make([]string, len(r.warnings))
	copy(clone, r.warnings)
	return clone
}

type warningNotifier struct {
	base     ports.AlertNotifier
	warnings *warningRecorder
}

func (n *warningNotifier) NotifyAlert(ctx context.Context, alert domain.AlertEvent) error {
	if n == nil || n.base == nil {
		return nil
	}
	if err := n.base.NotifyAlert(ctx, alert); err != nil {
		n.warnings.Add(fmt.Sprintf("alert notification failed for %s: %v", alert.AlertID, err))
	}
	return nil
}

func dedupeWarnings(warnings []string) []string {
	if len(warnings) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(warnings))
	result := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		trimmed := strings.TrimSpace(warning)
		if trimmed == "" {
			continue
		}
		if _, exists := set[trimmed]; exists {
			continue
		}
		set[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}

func formatGeminiStatusWarning(status parsers.GeminiSupportStatus) string {
	message := strings.TrimSpace(status.Message)
	if message == "" {
		message = "Gemini CLI support status is unavailable"
	}
	if strings.TrimSpace(status.Path) == "" {
		return message
	}
	return fmt.Sprintf("%s (%s)", message, status.Path)
}
