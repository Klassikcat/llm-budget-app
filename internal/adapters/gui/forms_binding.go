package gui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zalando/go-keyring"

	"llm-budget-tracker/internal/config"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
	"llm-budget-tracker/internal/service"
)

type settingsLoader interface {
	Load(ctx context.Context) (config.Settings, error)
	Save(ctx context.Context, settings config.Settings) error
	SetSecret(ctx context.Context, id config.SecretID, secret string) error
	DeleteSecret(ctx context.Context, id config.SecretID) error
}

type subscriptionSaver interface {
	SaveSubscriptions(ctx context.Context, subscriptions []domain.Subscription) error
	ListSubscriptions(ctx context.Context, filter ports.SubscriptionFilter) ([]domain.Subscription, error)
}

type manualEntrySaver interface {
	Save(ctx context.Context, cmd service.ManualAPIUsageEntryCommand) (domain.UsageEntry, error)
}

type budgetSaver interface {
	SaveBudgets(ctx context.Context, budgets []domain.MonthlyBudget) error
	ListBudgets(ctx context.Context, filter ports.BudgetFilter) ([]domain.MonthlyBudget, error)
}

type startupAware interface {
	startup(ctx context.Context)
}

type FormsBinding struct {
	settings      settingsLoader
	subscriptions subscriptionSaver
	manualEntries manualEntrySaver
	budgets       budgetSaver
	notifier      ports.AlertNotifier
	ctx           context.Context
	clock         func() time.Time
}

func NewFormsBinding(settings settingsLoader, subscriptions subscriptionSaver, manualEntries manualEntrySaver, budgets budgetSaver, notifier ports.AlertNotifier) *FormsBinding {
	return &FormsBinding{
		settings:      settings,
		subscriptions: subscriptions,
		manualEntries: manualEntries,
		budgets:       budgets,
		notifier:      notifier,
		clock:         func() time.Time { return time.Now().UTC() },
	}
}

func (b *FormsBinding) ListSubscriptionPresets() SubscriptionPresetsResponse {
	presets := service.ListSubscriptionPresets()
	items := make([]SubscriptionPresetState, 0, len(presets))
	for _, preset := range presets {
		items = append(items, SubscriptionPresetState{
			Key:        preset.Key,
			Provider:   preset.Provider.String(),
			PlanName:   preset.PlanName,
			RenewalDay: preset.DefaultRenewalDay,
			FeeUSD:     preset.DefaultFeeUSD,
		})
	}
	return SubscriptionPresetsResponse{Items: items}
}

func (b *FormsBinding) startup(ctx context.Context) {
	if b == nil {
		return
	}
	b.ctx = ctx
	if starter, ok := b.notifier.(startupAware); ok {
		starter.startup(ctx)
	}
}

func (b *FormsBinding) LoadSettings() SettingsFormResponse {
	if b == nil || b.settings == nil {
		return SettingsFormResponse{Result: failedMutationResult(errSettingsBindingUnavailable)}
	}

	settings, err := b.settings.Load(b.context())
	if err != nil {
		return SettingsFormResponse{Result: failedMutationResult(err)}
	}

	return SettingsFormResponse{
		Result:   successMutationResult(),
		Settings: toSettingsFormState(settings),
	}

}

func (b *FormsBinding) SaveSettings(input SettingsFormInput) SettingsFormResponse {
	if b == nil || b.settings == nil {
		return SettingsFormResponse{Result: failedMutationResult(errSettingsBindingUnavailable)}
	}

	settings := config.Settings{
		SchemaVersion: 1,
		Providers: config.ProviderSettings{
			Anthropic:  config.ProviderToggle{Enabled: input.Providers.AnthropicEnabled},
			OpenAI:     config.ProviderToggle{Enabled: input.Providers.OpenAIEnabled},
			Gemini:     config.ProviderToggle{Enabled: input.Providers.GeminiEnabled},
			OpenRouter: config.ProviderToggle{Enabled: input.Providers.OpenRouterEnabled},
		},
		CLIBillingDefaults: config.CLIBillingDefaults{
			ClaudeCode: config.BillingMode(strings.TrimSpace(input.CLIBillingDefaults.ClaudeCode)),
			Codex:      config.BillingMode(strings.TrimSpace(input.CLIBillingDefaults.Codex)),
			GeminiCLI:  config.BillingMode(strings.TrimSpace(input.CLIBillingDefaults.GeminiCLI)),
			OpenCode:   config.BillingMode(strings.TrimSpace(input.CLIBillingDefaults.OpenCode)),
		},
		SubscriptionDefaults: config.SubscriptionDefaults{
			OpenAI: toSubscriptionPlanSettings(input.SubscriptionDefaults.OpenAI, config.DefaultSettings().SubscriptionDefaults.OpenAI),
			Claude: toSubscriptionPlanSettings(input.SubscriptionDefaults.Claude, config.DefaultSettings().SubscriptionDefaults.Claude),
			Gemini: toSubscriptionPlanSettings(input.SubscriptionDefaults.Gemini, config.DefaultSettings().SubscriptionDefaults.Gemini),
		},
		Budgets: config.BudgetSettings{
			MonthlyBudgetUSD:             input.Budgets.MonthlyBudgetUSD,
			MonthlySubscriptionBudgetUSD: input.Budgets.MonthlySubscriptionBudgetUSD,
			MonthlyUsageBudgetUSD:        input.Budgets.MonthlyUsageBudgetUSD,
			WarningThresholdPercent:      input.Budgets.WarningThresholdPercent,
			CriticalThresholdPercent:     input.Budgets.CriticalThresholdPercent,
		},
		Notifications: config.NotificationPreferences{
			DesktopEnabled:      input.Notifications.DesktopEnabled,
			TUIEnabled:          input.Notifications.TUIEnabled,
			BudgetWarnings:      input.Notifications.BudgetWarnings,
			ForecastWarnings:    input.Notifications.ForecastWarnings,
			ProviderSyncFailure: input.Notifications.ProviderSyncFailure,
		},
	}

	if err := b.settings.Save(b.context(), settings); err != nil {
		return SettingsFormResponse{Result: failedMutationResult(err)}
	}
	persisted, err := b.settings.Load(b.context())
	if err != nil {
		return SettingsFormResponse{Result: failedMutationResult(err)}
	}

	return SettingsFormResponse{
		Result:   successMutationResult(),
		Settings: toSettingsFormState(persisted),
	}
}

func (b *FormsBinding) SaveProviderSecret(input ProviderSecretInput) MutationResponse {
	if b == nil || b.settings == nil {
		return failedMutationResult(errSettingsBindingUnavailable)
	}

	secretID, err := resolveSecretID(input.Provider, input.SecretType)
	if err != nil {
		return failedMutationResult(err)
	}

	if err := b.settings.SetSecret(b.context(), secretID, input.Value); err != nil {
		return failedMutationResult(err)
	}

	return successMutationResult()
}

func (b *FormsBinding) DeleteProviderSecret(input ProviderSecretDeleteInput) MutationResponse {
	if b == nil || b.settings == nil {
		return failedMutationResult(errSettingsBindingUnavailable)
	}

	secretID, err := resolveSecretID(input.Provider, input.SecretType)
	if err != nil {
		return failedMutationResult(err)
	}

	if err := b.settings.DeleteSecret(b.context(), secretID); err != nil {
		return failedMutationResult(err)
	}

	return successMutationResult()
}

func (b *FormsBinding) SaveSubscription(input SubscriptionFormInput) SubscriptionMutationResponse {
	if b == nil || b.subscriptions == nil {
		return SubscriptionMutationResponse{Result: failedMutationResult(errSubscriptionBindingUnavailable)}
	}

	resolved, err := resolveSubscriptionInput(input)
	if err != nil {
		return SubscriptionMutationResponse{Result: failedMutationResult(err)}
	}

	startsAt, err := parseSubscriptionDateInput("starts_at", resolved.StartsAt)
	if err != nil {
		return SubscriptionMutationResponse{Result: failedMutationResult(err)}
	}

	endsAt, err := parseOptionalSubscriptionDateInput("ends_at", resolved.EndsAt)
	if err != nil {
		return SubscriptionMutationResponse{Result: failedMutationResult(err)}
	}

	planCode, err := domain.GenerateSubscriptionPlanCode(resolved.Provider, resolved.PlanName)
	if err != nil {
		return SubscriptionMutationResponse{Result: failedMutationResult(err)}
	}
	subscriptionID, err := domain.GenerateSubscriptionID(resolved.Provider, resolved.PlanName, startsAt)
	if err != nil {
		return SubscriptionMutationResponse{Result: failedMutationResult(err)}
	}

	createdAt := time.Time{}
	if existing, err := b.subscriptions.ListSubscriptions(b.context(), ports.SubscriptionFilter{SubscriptionID: subscriptionID}); err == nil && len(existing) > 0 {
		createdAt = existing[0].CreatedAt
	}

	subscription := domain.Subscription{
		SubscriptionID: subscriptionID,
		Provider:       resolved.Provider,
		PlanCode:       planCode,
		PlanName:       resolved.PlanName,
		RenewalDay:     resolved.RenewalDay,
		StartsAt:       startsAt,
		EndsAt:         endsAt,
		FeeUSD:         resolved.FeeUSD,
		IsActive:       resolved.IsActive,
		CreatedAt:      createdAt,
	}

	if err := b.subscriptions.SaveSubscriptions(b.context(), []domain.Subscription{subscription}); err != nil {
		return SubscriptionMutationResponse{Result: failedMutationResult(err)}
	}

	persisted, err := b.subscriptions.ListSubscriptions(b.context(), ports.SubscriptionFilter{SubscriptionID: subscription.SubscriptionID})
	if err != nil || len(persisted) == 0 {
		if err == nil {
			err = fmt.Errorf("subscription %q was not persisted", subscription.SubscriptionID)
		}
		return SubscriptionMutationResponse{Result: failedMutationResult(err)}
	}

	return SubscriptionMutationResponse{
		Result:       successMutationResult(),
		Subscription: toSubscriptionState(persisted[0]),
	}
}

type resolvedSubscriptionInput struct {
	PresetKey  string
	Provider   domain.ProviderName
	PlanName   string
	RenewalDay int
	StartsAt   string
	EndsAt     string
	FeeUSD     float64
	IsActive   bool
}

func resolveSubscriptionInput(input SubscriptionFormInput) (resolvedSubscriptionInput, error) {
	resolved := resolvedSubscriptionInput{
		PresetKey:  strings.TrimSpace(input.PresetKey),
		PlanName:   strings.TrimSpace(input.PlanName),
		RenewalDay: input.RenewalDay,
		StartsAt:   strings.TrimSpace(input.StartsAt),
		EndsAt:     strings.TrimSpace(input.EndsAt),
		FeeUSD:     input.FeeUSD,
		IsActive:   input.IsActive,
	}

	if resolved.PresetKey != "" {
		preset, err := service.ResolveSubscriptionPreset(resolved.PresetKey)
		if err != nil {
			return resolvedSubscriptionInput{}, err
		}
		resolved.Provider = preset.Provider
		resolved.PlanName = preset.PlanName
		if resolved.RenewalDay == 0 {
			resolved.RenewalDay = preset.DefaultRenewalDay
		}
		if resolved.FeeUSD == 0 {
			resolved.FeeUSD = preset.DefaultFeeUSD
		}
		return resolved, nil
	}

	provider, err := domain.NewProviderName(input.Provider)
	if err != nil {
		return resolvedSubscriptionInput{}, err
	}
	if resolved.PlanName == "" {
		return resolvedSubscriptionInput{}, &domain.ValidationError{Code: domain.ValidationCodeRequired, Field: "plan_name", Message: "value is required"}
	}
	resolved.Provider = provider
	return resolved, nil
}

func (b *FormsBinding) SaveManualEntry(input ManualEntryFormInput) ManualEntryMutationResponse {
	if b == nil || b.manualEntries == nil {
		return ManualEntryMutationResponse{Result: failedMutationResult(errManualEntryBindingUnavailable)}
	}

	occurredAt, err := parseTimestampInput("occurred_at", input.OccurredAt)
	if err != nil {
		return ManualEntryMutationResponse{Result: failedMutationResult(err)}
	}

	entry, err := b.manualEntries.Save(b.context(), service.ManualAPIUsageEntryCommand{
		Provider:         input.Provider,
		ModelID:          input.ModelID,
		OccurredAt:       occurredAt,
		InputTokens:      input.InputTokens,
		OutputTokens:     input.OutputTokens,
		CachedTokens:     input.CachedTokens,
		CacheWriteTokens: input.CacheWriteTokens,
		ProjectName:      input.ProjectName,
		Metadata:         input.Metadata,
	})
	if err != nil {
		return ManualEntryMutationResponse{Result: failedMutationResult(err)}
	}

	return ManualEntryMutationResponse{
		Result: successMutationResult(),
		Entry:  toManualEntryState(entry),
	}
}

func (b *FormsBinding) SaveBudget(input BudgetFormInput) BudgetMutationResponse {
	if b == nil || b.budgets == nil {
		return BudgetMutationResponse{Result: failedMutationResult(errBudgetBindingUnavailable)}
	}

	period, err := parseMonthInput(strings.TrimSpace(input.PeriodMonth))
	if err != nil {
		return BudgetMutationResponse{Result: failedMutationResult(err)}
	}

	thresholds, err := budgetThresholdsFromInput(input.WarningThresholdPercent, input.CriticalThresholdPercent)
	if err != nil {
		return BudgetMutationResponse{Result: failedMutationResult(err)}
	}

	provider, err := parseOptionalProvider(input.Provider)
	if err != nil {
		return BudgetMutationResponse{Result: failedMutationResult(err)}
	}

	budget := domain.MonthlyBudget{
		BudgetID:    strings.TrimSpace(input.BudgetID),
		Name:        input.Name,
		Period:      period,
		LimitUSD:    input.LimitUSD,
		Thresholds:  thresholds,
		Currency:    input.Currency,
		Provider:    provider,
		ProjectHash: input.ProjectHash,
	}

	if err := b.budgets.SaveBudgets(b.context(), []domain.MonthlyBudget{budget}); err != nil {
		return BudgetMutationResponse{Result: failedMutationResult(err)}
	}

	persisted, err := b.budgets.ListBudgets(b.context(), ports.BudgetFilter{Period: &period, Provider: provider})
	if err != nil {
		return BudgetMutationResponse{Result: failedMutationResult(err)}
	}

	for _, candidate := range persisted {
		if candidate.BudgetID == budget.BudgetID {
			return BudgetMutationResponse{Result: successMutationResult(), Budget: toBudgetState(candidate)}
		}
	}

	return BudgetMutationResponse{Result: failedMutationResult(fmt.Errorf("budget %q was not persisted", budget.BudgetID))}
}

func (b *FormsBinding) DispatchAlertNotification(input AlertNotificationInput) NotificationDispatchResponse {
	if b == nil || b.notifier == nil {
		return NotificationDispatchResponse{Result: failedMutationResult(errNotifierBindingUnavailable)}
	}

	period, err := parseMonthInput(strings.TrimSpace(input.PeriodMonth))
	if err != nil {
		return NotificationDispatchResponse{Result: failedMutationResult(err)}
	}

	severity := domain.AlertSeverity(strings.TrimSpace(input.Severity))
	if !severity.IsValid() {
		return NotificationDispatchResponse{Result: failedMutationResult(&domain.ValidationError{
			Code:    domain.ValidationCodeInvalidAlertLevel,
			Field:   "severity",
			Message: "severity must be one of info, warning, or critical",
		})}
	}

	kind := domain.AlertKind(strings.TrimSpace(input.Kind))
	if !kind.IsValid() {
		return NotificationDispatchResponse{Result: failedMutationResult(&domain.ValidationError{
			Code:    domain.ValidationCodeInvalidAlertKind,
			Field:   "kind",
			Message: "alert kind must be one of budget_threshold, budget_overrun, forecast_overrun, or insight_detected",
		})}
	}

	triggeredAt, err := parseTimestampInput("triggered_at", input.TriggeredAt)
	if err != nil {
		return NotificationDispatchResponse{Result: failedMutationResult(err)}
	}

	alert, err := domain.NewAlertEvent(domain.AlertEvent{
		AlertID:          strings.TrimSpace(input.AlertID),
		Kind:             kind,
		Severity:         severity,
		TriggeredAt:      triggeredAt,
		Period:           period,
		BudgetID:         strings.TrimSpace(input.BudgetID),
		ForecastID:       strings.TrimSpace(input.ForecastID),
		InsightID:        strings.TrimSpace(input.InsightID),
		DetectorCategory: domain.DetectorCategory(strings.TrimSpace(input.DetectorCategory)),
		CurrentSpendUSD:  input.CurrentSpendUSD,
		LimitUSD:         input.LimitUSD,
		ThresholdPercent: input.ThresholdPercent,
	})
	if err != nil {
		return NotificationDispatchResponse{Result: failedMutationResult(err)}
	}

	if err := b.notifier.NotifyAlert(b.context(), alert); err != nil {
		return NotificationDispatchResponse{Result: failedMutationResult(err)}
	}

	return NotificationDispatchResponse{Result: successMutationResult(), Dispatched: true}
}

func (b *FormsBinding) context() context.Context {
	if b != nil && b.ctx != nil {
		return b.ctx
	}
	return context.Background()
}

func resolveSecretID(providerRaw, secretTypeRaw string) (config.SecretID, error) {
	provider := strings.TrimSpace(providerRaw)
	secretType := strings.TrimSpace(secretTypeRaw)

	switch provider {
	case domain.ProviderAnthropic.String():
		switch secretType {
		case "api_key":
			return config.SecretAnthropicAPIKey, nil
		case "admin_key":
			return config.SecretAnthropicAdminKey, nil
		}
	case domain.ProviderOpenAI.String():
		switch secretType {
		case "api_key":
			return config.SecretOpenAIAPIKey, nil
		case "admin_key":
			return config.SecretOpenAIAdminKey, nil
		}
	case domain.ProviderOpenRouter.String():
		if secretType == "api_key" {
			return config.SecretOpenRouterAPIKey, nil
		}
	}

	return "", &domain.ValidationError{
		Code:    domain.ValidationCodeUnsupportedProvider,
		Field:   "provider",
		Message: fmt.Sprintf("unsupported provider secret combination %q/%q", provider, secretType),
	}
}

func parseOptionalProvider(providerRaw string) (domain.ProviderName, error) {
	trimmed := strings.TrimSpace(providerRaw)
	if trimmed == "" {
		return "", nil
	}

	return domain.NewProviderName(trimmed)
}

func parseTimestampInput(field, raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, &domain.ValidationError{Code: domain.ValidationCodeRequired, Field: field, Message: "value is required"}
	}

	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			if layout == "2006-01-02" {
				parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
			}
			return parsed.UTC(), nil
		}
	}

	return time.Time{}, &domain.ValidationError{Code: domain.ValidationCodeInvalidTimestamp, Field: field, Message: "timestamp must be RFC3339 or YYYY-MM-DD"}
}

func parseOptionalTimestampInput(field, raw string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	parsed, err := parseTimestampInput(field, raw)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func parseSubscriptionDateInput(field, raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, &domain.ValidationError{Code: domain.ValidationCodeRequired, Field: field, Message: "value is required"}
	}

	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return time.Time{}, &domain.ValidationError{Code: domain.ValidationCodeInvalidTimestamp, Field: field, Message: "date must use YYYY-MM-DD"}
	}

	return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC), nil
}

func parseOptionalSubscriptionDateInput(field, raw string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	parsed, err := parseSubscriptionDateInput(field, raw)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func parseMonthInput(raw string) (domain.MonthlyPeriod, error) {
	if strings.TrimSpace(raw) == "" {
		return domain.MonthlyPeriod{}, &domain.ValidationError{Code: domain.ValidationCodeRequired, Field: "period_month", Message: "value is required"}
	}

	parsed, err := time.Parse(dashboardMonthLayout, raw)
	if err != nil {
		return domain.MonthlyPeriod{}, &domain.ValidationError{Code: domain.ValidationCodeInvalidMonth, Field: "period_month", Message: "month must use YYYY-MM format"}
	}

	return domain.NewMonthlyPeriod(parsed.UTC())
}

func budgetThresholdsFromInput(warningPercent, criticalPercent int) ([]domain.BudgetThreshold, error) {
	if warningPercent < 1 || warningPercent >= 100 {
		return nil, &domain.ValidationError{Code: domain.ValidationCodeInvalidThreshold, Field: "warning_threshold_percent", Message: "warning threshold percent must be between 1 and 99"}
	}
	if criticalPercent < 1 || criticalPercent > 100 {
		return nil, &domain.ValidationError{Code: domain.ValidationCodeInvalidThreshold, Field: "critical_threshold_percent", Message: "threshold percent must be between 1 and 100"}
	}
	if criticalPercent < warningPercent {
		return nil, &domain.ValidationError{Code: domain.ValidationCodeInvalidThreshold, Field: "critical_threshold_percent", Message: "critical threshold must be greater than or equal to warning threshold"}
	}

	thresholds := make([]domain.BudgetThreshold, 0, 2)
	warning, err := domain.NewBudgetThreshold(domain.AlertSeverityWarning, float64(warningPercent)/100)
	if err != nil {
		return nil, err
	}
	thresholds = append(thresholds, warning)

	if criticalPercent != warningPercent && criticalPercent < 100 {
		critical, err := domain.NewBudgetThreshold(domain.AlertSeverityCritical, float64(criticalPercent)/100)
		if err != nil {
			return nil, err
		}
		thresholds = append(thresholds, critical)
	}

	return thresholds, nil
}

func successMutationResult() MutationResponse {
	return MutationResponse{Success: true}
}

func failedMutationResult(err error) MutationResponse {
	if err == nil {
		return MutationResponse{Success: false}
	}

	return MutationResponse{
		Success: false,
		Error:   toFormError(err),
	}
}

func toFormError(err error) *FormError {
	if err == nil {
		return nil
	}

	var validationErr *domain.ValidationError
	if errors.As(err, &validationErr) {
		return &FormError{Code: string(validationErr.Code), Field: validationErr.Field, Message: validationErr.Message}
	}

	var setupErr *config.SetupError
	if errors.As(err, &setupErr) {
		return &FormError{Code: string(setupErr.Code), Message: setupErr.Message}
	}

	if errors.Is(err, keyring.ErrNotFound) {
		return &FormError{Code: "secret_not_found", Message: err.Error()}
	}

	return &FormError{Code: "internal_error", Message: err.Error()}
}

func toSettingsFormState(settings config.Settings) SettingsFormState {
	return SettingsFormState{
		Providers: ProviderSettingsState{
			AnthropicEnabled:  settings.Providers.Anthropic.Enabled,
			OpenAIEnabled:     settings.Providers.OpenAI.Enabled,
			GeminiEnabled:     settings.Providers.Gemini.Enabled,
			OpenRouterEnabled: settings.Providers.OpenRouter.Enabled,
		},
		CLIBillingDefaults: CLIBillingDefaultsState{
			ClaudeCode: string(settings.CLIBillingDefaults.ClaudeCode),
			Codex:      string(settings.CLIBillingDefaults.Codex),
			GeminiCLI:  string(settings.CLIBillingDefaults.GeminiCLI),
			OpenCode:   string(settings.CLIBillingDefaults.OpenCode),
		},
		SubscriptionDefaults: SubscriptionDefaultsState{
			OpenAI: SubscriptionPlanState{
				Enabled:    settings.SubscriptionDefaults.OpenAI.Enabled,
				PlanCode:   settings.SubscriptionDefaults.OpenAI.PlanCode,
				PlanName:   settings.SubscriptionDefaults.OpenAI.PlanName,
				FeeUSD:     settings.SubscriptionDefaults.OpenAI.FeeUSD,
				RenewalDay: settings.SubscriptionDefaults.OpenAI.RenewalDay,
				SourceURL:  settings.SubscriptionDefaults.OpenAI.SourceURL,
			},
			Claude: SubscriptionPlanState{
				Enabled:    settings.SubscriptionDefaults.Claude.Enabled,
				PlanCode:   settings.SubscriptionDefaults.Claude.PlanCode,
				PlanName:   settings.SubscriptionDefaults.Claude.PlanName,
				FeeUSD:     settings.SubscriptionDefaults.Claude.FeeUSD,
				RenewalDay: settings.SubscriptionDefaults.Claude.RenewalDay,
				SourceURL:  settings.SubscriptionDefaults.Claude.SourceURL,
			},
			Gemini: SubscriptionPlanState{
				Enabled:    settings.SubscriptionDefaults.Gemini.Enabled,
				PlanCode:   settings.SubscriptionDefaults.Gemini.PlanCode,
				PlanName:   settings.SubscriptionDefaults.Gemini.PlanName,
				FeeUSD:     settings.SubscriptionDefaults.Gemini.FeeUSD,
				RenewalDay: settings.SubscriptionDefaults.Gemini.RenewalDay,
				SourceURL:  settings.SubscriptionDefaults.Gemini.SourceURL,
			},
		},
		Budgets: BudgetSettingsState{
			MonthlyBudgetUSD:             settings.Budgets.MonthlyBudgetUSD,
			MonthlySubscriptionBudgetUSD: settings.Budgets.MonthlySubscriptionBudgetUSD,
			MonthlyUsageBudgetUSD:        settings.Budgets.MonthlyUsageBudgetUSD,
			WarningThresholdPercent:      settings.Budgets.WarningThresholdPercent,
			CriticalThresholdPercent:     settings.Budgets.CriticalThresholdPercent,
		},
		Notifications: NotificationSettingsState{
			DesktopEnabled:      settings.Notifications.DesktopEnabled,
			TUIEnabled:          settings.Notifications.TUIEnabled,
			BudgetWarnings:      settings.Notifications.BudgetWarnings,
			ForecastWarnings:    settings.Notifications.ForecastWarnings,
			ProviderSyncFailure: settings.Notifications.ProviderSyncFailure,
		},
	}
}

func toSubscriptionPlanSettings(input SubscriptionPlanState, fallback config.SubscriptionPlanSettings) config.SubscriptionPlanSettings {
	if strings.TrimSpace(input.PlanCode) == "" && strings.TrimSpace(input.PlanName) == "" && strings.TrimSpace(input.SourceURL) == "" && input.FeeUSD == 0 && input.RenewalDay == 0 && !input.Enabled {
		return fallback
	}

	return config.SubscriptionPlanSettings{
		Enabled:    input.Enabled,
		PlanCode:   input.PlanCode,
		PlanName:   input.PlanName,
		FeeUSD:     input.FeeUSD,
		RenewalDay: input.RenewalDay,
		SourceURL:  input.SourceURL,
	}
}

func toSubscriptionState(subscription domain.Subscription) SubscriptionState {
	state := SubscriptionState{
		Provider:   subscription.Provider.String(),
		PlanName:   subscription.PlanName,
		RenewalDay: subscription.RenewalDay,
		StartsAt:   formatSubscriptionDate(subscription.StartsAt),
		FeeUSD:     subscription.FeeUSD,
		IsActive:   subscription.IsActive,
	}
	if subscription.EndsAt != nil {
		state.EndsAt = formatSubscriptionDate(*subscription.EndsAt)
	}
	return state
}

func formatSubscriptionDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format("2006-01-02")
}

func toManualEntryState(entry domain.UsageEntry) ManualEntryState {
	metadata := make(map[string]string, len(entry.Metadata))
	for key, value := range entry.Metadata {
		metadata[key] = value
	}

	modelID := ""
	if entry.PricingRef != nil {
		modelID = entry.PricingRef.ModelID
	}

	return ManualEntryState{
		EntryID:          entry.EntryID,
		Provider:         entry.Provider.String(),
		ModelID:          modelID,
		OccurredAt:       formatDashboardTime(entry.OccurredAt),
		ProjectName:      entry.ProjectName,
		InputTokens:      entry.Tokens.InputTokens,
		OutputTokens:     entry.Tokens.OutputTokens,
		CachedTokens:     entry.Tokens.CacheReadTokens,
		CacheWriteTokens: entry.Tokens.CacheWriteTokens,
		TotalCostUSD:     entry.CostBreakdown.TotalUSD,
		Metadata:         metadata,
	}
}

func toBudgetState(budget domain.MonthlyBudget) BudgetState {
	state := BudgetState{
		BudgetID:    budget.BudgetID,
		Name:        budget.Name,
		Provider:    budget.Provider.String(),
		ProjectHash: budget.ProjectHash,
		PeriodMonth: budget.Period.StartAt.Format(dashboardMonthLayout),
		LimitUSD:    budget.LimitUSD,
		Currency:    budget.Currency,
	}
	for _, threshold := range budget.Thresholds {
		percent := int(threshold.Percent * 100)
		switch threshold.Severity {
		case domain.AlertSeverityWarning:
			state.WarningThresholdPercent = percent
		case domain.AlertSeverityCritical:
			state.CriticalThresholdPercent = percent
		}
	}
	if state.CriticalThresholdPercent == 0 {
		state.CriticalThresholdPercent = 100
	}
	return state
}

var (
	errSettingsBindingUnavailable     = errors.New("forms binding requires settings services")
	errSubscriptionBindingUnavailable = errors.New("forms binding requires subscription services")
	errManualEntryBindingUnavailable  = errors.New("forms binding requires manual entry services")
	errBudgetBindingUnavailable       = errors.New("forms binding requires budget services")
	errNotifierBindingUnavailable     = errors.New("forms binding requires a notifier")
)
