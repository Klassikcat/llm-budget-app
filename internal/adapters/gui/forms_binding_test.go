package gui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"llm-budget-tracker/internal/adapters/sqlite"
	catalogpkg "llm-budget-tracker/internal/catalog"
	"llm-budget-tracker/internal/config"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
	"llm-budget-tracker/internal/service"
)

func TestSettingsAndEntryForms(t *testing.T) {
	t.Parallel()

	binding, store, settingsFile, secrets := newTestFormsBinding(t, notifierStub{})
	defer store.Close()
	binding.startup(context.Background())

	settingsResponse := binding.SaveSettings(SettingsFormInput{
		Providers: ProviderSettingsState{
			AnthropicEnabled:  true,
			OpenAIEnabled:     true,
			GeminiEnabled:     false,
			OpenRouterEnabled: true,
		},
		CLIBillingDefaults: CLIBillingDefaultsState{
			ClaudeCode: "subscription",
			Codex:      "byok",
			GeminiCLI:  "subscription",
			OpenCode:   "byok",
		},
		SubscriptionDefaults: SubscriptionDefaultsState{
			OpenAI: SubscriptionPlanState{
				Enabled:    true,
				PlanCode:   "chatgpt-plus",
				PlanName:   "ChatGPT Plus",
				FeeUSD:     20,
				RenewalDay: 2,
				SourceURL:  "https://openai.com/ChatGPT/pricing",
			},
			Claude: SubscriptionPlanState{
				Enabled:    false,
				PlanCode:   "claude-pro",
				PlanName:   "Claude Pro",
				FeeUSD:     20,
				RenewalDay: 3,
				SourceURL:  "https://claude.com/pricing",
			},
			Gemini: SubscriptionPlanState{
				Enabled:    false,
				PlanCode:   "google-ai-pro",
				PlanName:   "Google AI Pro",
				FeeUSD:     19.99,
				RenewalDay: 4,
				SourceURL:  "https://gemini.google/us/subscriptions/",
			},
		},
		Budgets: BudgetSettingsState{
			MonthlyBudgetUSD:             320,
			MonthlySubscriptionBudgetUSD: 140,
			MonthlyUsageBudgetUSD:        180,
			WarningThresholdPercent:      75,
			CriticalThresholdPercent:     95,
		},
		Notifications: NotificationSettingsState{
			DesktopEnabled:      true,
			TUIEnabled:          false,
			BudgetWarnings:      true,
			ForecastWarnings:    true,
			ProviderSyncFailure: true,
		},
	})
	assertMutationSuccess(t, settingsResponse.Result)
	if !settingsResponse.Settings.Providers.OpenRouterEnabled || settingsResponse.Settings.Providers.GeminiEnabled {
		t.Fatalf("settings state = %+v, want openrouter enabled and gemini disabled", settingsResponse.Settings.Providers)
	}

	secretResponse := binding.SaveProviderSecret(ProviderSecretInput{
		Provider:   "openrouter",
		SecretType: "api_key",
		Value:      "secret-openrouter-key",
	})
	assertMutationSuccess(t, secretResponse)

	subscriptionResponse := binding.SaveSubscription(SubscriptionFormInput{
		SubscriptionID: "sub-openai-plus",
		Provider:       "openai",
		PlanCode:       "chatgpt-plus",
		PlanName:       "ChatGPT Plus",
		RenewalDay:     1,
		StartsAt:       "2026-04-01T00:00:00Z",
		FeeUSD:         20,
		IsActive:       true,
	})
	assertMutationSuccess(t, subscriptionResponse.Result)
	if subscriptionResponse.Subscription.SubscriptionID != "sub-openai-plus" {
		t.Fatalf("subscription id = %q, want sub-openai-plus", subscriptionResponse.Subscription.SubscriptionID)
	}

	manualEntryResponse := binding.SaveManualEntry(ManualEntryFormInput{
		Provider:     "openai",
		ModelID:      "gpt-4.1",
		OccurredAt:   "2026-04-18T10:15:00+09:00",
		InputTokens:  1200,
		OutputTokens: 300,
		CachedTokens: 100,
		ProjectName:  "llm-budget-tracker",
		Metadata: map[string]string{
			"environment": "test",
			"source":      "gui-form",
		},
	})
	assertMutationSuccess(t, manualEntryResponse.Result)
	if manualEntryResponse.Entry.TotalCostUSD <= 0 {
		t.Fatalf("manual entry total cost = %v, want positive value", manualEntryResponse.Entry.TotalCostUSD)
	}

	budgetResponse := binding.SaveBudget(BudgetFormInput{
		BudgetID:                 "budget-openai-april",
		Name:                     "OpenAI April Budget",
		Provider:                 "openai",
		PeriodMonth:              "2026-04",
		LimitUSD:                 60,
		WarningThresholdPercent:  80,
		CriticalThresholdPercent: 100,
		Currency:                 "usd",
	})
	assertMutationSuccess(t, budgetResponse.Result)
	if budgetResponse.Budget.WarningThresholdPercent != 80 || budgetResponse.Budget.CriticalThresholdPercent != 100 {
		t.Fatalf("budget thresholds = %+v, want warning=80 critical=100", budgetResponse.Budget)
	}

	loadedSettings := binding.LoadSettings()
	assertMutationSuccess(t, loadedSettings.Result)
	if got := loadedSettings.Settings.CLIBillingDefaults.Codex; got != "byok" {
		t.Fatalf("loaded codex billing default = %q, want byok", got)
	}
	if !loadedSettings.Settings.SubscriptionDefaults.OpenAI.Enabled {
		t.Fatalf("loaded openai subscription defaults = %+v, want enabled", loadedSettings.Settings.SubscriptionDefaults.OpenAI)
	}
	if got := loadedSettings.Settings.SubscriptionDefaults.Gemini.FeeUSD; got != 19.99 {
		t.Fatalf("loaded gemini subscription fee = %v, want 19.99", got)
	}

	rawSettings, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("ReadFile(settings) error = %v", err)
	}
	settingsContent := string(rawSettings)
	if strings.Contains(settingsContent, "secret-openrouter-key") {
		t.Fatal("settings file leaked secret value")
	}
	if strings.Contains(settingsContent, string(config.SecretOpenRouterAPIKey)) {
		t.Fatal("settings file leaked secret identifier")
	}
	if got := secrets.values[string(config.SecretOpenRouterAPIKey)]; got != "secret-openrouter-key" {
		t.Fatalf("keyring secret = %q, want secret-openrouter-key", got)
	}

	subscriptions, err := store.ListSubscriptions(context.Background(), ports.SubscriptionFilter{SubscriptionID: "sub-openai-plus"})
	if err != nil {
		t.Fatalf("ListSubscriptions() error = %v", err)
	}
	if len(subscriptions) != 1 || subscriptions[0].PlanCode != "chatgpt-plus" {
		t.Fatalf("subscriptions = %+v, want single chatgpt-plus subscription", subscriptions)
	}

	entries, err := store.ListUsageEntries(context.Background(), ports.UsageFilter{Project: "llm-budget-tracker"})
	if err != nil {
		t.Fatalf("ListUsageEntries() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Metadata["source"] != "gui-form" {
		t.Fatalf("entries = %+v, want one manual entry with source=gui-form", entries)
	}

	period, err := domain.NewMonthlyPeriodFromParts(2026, time.April)
	if err != nil {
		t.Fatalf("NewMonthlyPeriodFromParts() error = %v", err)
	}
	budgets, err := store.ListMonthlyBudgets(context.Background(), ports.BudgetFilter{Period: &period, Provider: domain.ProviderOpenAI})
	if err != nil {
		t.Fatalf("ListMonthlyBudgets() error = %v", err)
	}
	if len(budgets) != 1 || budgets[0].BudgetID != "budget-openai-april" {
		t.Fatalf("budgets = %+v, want one budget-openai-april budget", budgets)
	}
}

func TestSettingsAndEntryFormsValidationError(t *testing.T) {
	t.Parallel()

	binding, store, _, _ := newTestFormsBinding(t, notifierStub{})
	defer store.Close()
	binding.startup(context.Background())

	response := binding.SaveManualEntry(ManualEntryFormInput{
		Provider:     "openai",
		ModelID:      "unknown-model",
		OccurredAt:   "2026-04-18T10:15:00Z",
		InputTokens:  100,
		OutputTokens: 50,
	})

	if response.Result.Success {
		t.Fatal("SaveManualEntry() success = true, want false")
	}
	if response.Result.Error == nil || response.Result.Error.Code != string(domain.ValidationCodeUnknownModel) {
		t.Fatalf("SaveManualEntry() error = %+v, want unknown_model", response.Result.Error)
	}

	entries, err := store.ListUsageEntries(context.Background(), ports.UsageFilter{})
	if err != nil {
		t.Fatalf("ListUsageEntries() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("ListUsageEntries() len = %d, want 0 after validation failure", len(entries))
	}
}

func TestNotificationFailureSurface(t *testing.T) {
	t.Parallel()

	binding, store, _, _ := newTestFormsBinding(t, notifierStub{err: errors.New("desktop notification backend unavailable")})
	defer store.Close()
	binding.startup(context.Background())

	response := binding.DispatchAlertNotification(AlertNotificationInput{
		AlertID:          "alert-budget-1",
		Kind:             string(domain.AlertKindBudgetOverrun),
		Severity:         string(domain.AlertSeverityCritical),
		TriggeredAt:      "2026-04-18T11:00:00Z",
		PeriodMonth:      "2026-04",
		BudgetID:         "budget-openai-april",
		CurrentSpendUSD:  81,
		LimitUSD:         60,
		ThresholdPercent: 1,
	})

	if response.Result.Success {
		t.Fatal("DispatchAlertNotification() success = true, want false")
	}
	if response.Result.Error == nil || !strings.Contains(response.Result.Error.Message, "desktop notification backend unavailable") {
		t.Fatalf("DispatchAlertNotification() error = %+v, want notifier failure message", response.Result.Error)
	}
	if response.Dispatched {
		t.Fatal("DispatchAlertNotification() dispatched = true, want false")
	}

	manualEntryResponse := binding.SaveManualEntry(ManualEntryFormInput{
		Provider:     "anthropic",
		ModelID:      "claude-sonnet-4-0",
		OccurredAt:   "2026-04-18T12:00:00Z",
		InputTokens:  1000,
		OutputTokens: 150,
		ProjectName:  "still-usable",
	})
	assertMutationSuccess(t, manualEntryResponse.Result)
}

func TestWailsAlertNotifierEmitsDesktopEvent(t *testing.T) {
	t.Parallel()

	notifier := NewWailsAlertNotifier()
	ctx := context.Background()
	notifier.startup(ctx)

	var (
		eventName string
		payload   DesktopNotificationPayload
	)
	notifier.emit = func(_ context.Context, name string, data ...interface{}) {
		eventName = name
		payload = data[0].(DesktopNotificationPayload)
	}

	period, err := domain.NewMonthlyPeriodFromParts(2026, time.April)
	if err != nil {
		t.Fatalf("NewMonthlyPeriodFromParts() error = %v", err)
	}
	alert, err := domain.NewAlertEvent(domain.AlertEvent{
		AlertID:          "alert-threshold-1",
		Kind:             domain.AlertKindBudgetThreshold,
		Severity:         domain.AlertSeverityWarning,
		TriggeredAt:      time.Date(2026, time.April, 18, 10, 0, 0, 0, time.UTC),
		Period:           period,
		BudgetID:         "budget-openai-april",
		CurrentSpendUSD:  48,
		LimitUSD:         60,
		ThresholdPercent: 0.8,
	})
	if err != nil {
		t.Fatalf("NewAlertEvent() error = %v", err)
	}

	if err := notifier.NotifyAlert(nil, alert); err != nil {
		t.Fatalf("NotifyAlert() error = %v", err)
	}
	if eventName != desktopNotificationEvent {
		t.Fatalf("eventName = %q, want %q", eventName, desktopNotificationEvent)
	}
	if payload.ID != alert.AlertID || !strings.Contains(payload.Body, "80%") {
		t.Fatalf("payload = %+v, want alert id and threshold body", payload)
	}
}

func newTestFormsBinding(t *testing.T, notifier ports.AlertNotifier) (*FormsBinding, *sqlite.Store, string, *memorySecretStore) {
	t.Helper()

	root := t.TempDir()
	paths := config.Paths{
		ConfigDir:          filepath.Join(root, "config"),
		DataDir:            filepath.Join(root, "data"),
		SettingsFile:       filepath.Join(root, "config", config.SettingsFileName),
		PricesOverrideFile: filepath.Join(root, "config", config.PricesOverrideFileName),
		DatabaseFile:       filepath.Join(root, "data", config.DatabaseFileName),
	}

	settingsStore := config.NewSettingsStore(paths)
	if _, err := settingsStore.Bootstrap(); err != nil {
		t.Fatalf("Bootstrap(settings) error = %v", err)
	}

	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: paths.DatabaseFile})
	if err != nil {
		t.Fatalf("sqlite.Bootstrap() error = %v", err)
	}

	catalog, err := catalogpkg.New(catalogpkg.Options{})
	if err != nil {
		t.Fatalf("catalog.New() error = %v", err)
	}

	secrets := &memorySecretStore{values: make(map[string]string)}
	settingsService := service.NewSettingsService(settingsStore, secrets)
	subscriptionService := service.NewSubscriptionService(store, store)
	manualEntryService := service.NewManualAPIUsageEntryService(catalog, store)
	budgetService := service.NewMonthlyBudgetService(store)

	return NewFormsBinding(settingsService, subscriptionService, manualEntryService, budgetService, notifier), store, paths.SettingsFile, secrets
}

func assertMutationSuccess(t *testing.T, result MutationResponse) {
	t.Helper()
	if !result.Success {
		t.Fatalf("mutation success = false, error = %+v", result.Error)
	}
}

type memorySecretStore struct {
	values map[string]string
}

func (s *memorySecretStore) Set(id config.SecretID, secret string) error {
	s.values[string(id)] = secret
	return nil
}

func (s *memorySecretStore) Get(id config.SecretID) (string, error) {
	secret, ok := s.values[string(id)]
	if !ok {
		return "", errors.New("secret not found")
	}
	return secret, nil
}

func (s *memorySecretStore) Delete(id config.SecretID) error {
	delete(s.values, string(id))
	return nil
}

type notifierStub struct {
	err error
}

func (n notifierStub) NotifyAlert(context.Context, domain.AlertEvent) error {
	return n.err
}
