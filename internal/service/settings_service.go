package service

import (
	"context"
	"strings"

	"llm-budget-tracker/internal/config"
	"llm-budget-tracker/internal/domain"
)

type settingsStore interface {
	Load() (config.Settings, error)
	Save(settings config.Settings) error
}

type SettingsService struct {
	store       settingsStore
	secretStore config.SecretStore
	defaults    config.Settings
}

func NewSettingsService(store settingsStore, secretStore config.SecretStore) *SettingsService {
	return &SettingsService{
		store:       store,
		secretStore: secretStore,
		defaults:    config.DefaultSettings(),
	}
}

func (s *SettingsService) Load(_ context.Context) (config.Settings, error) {
	if s == nil || s.store == nil {
		return config.Settings{}, errSettingsStoreRequired
	}

	return s.store.Load()
}

func (s *SettingsService) Save(_ context.Context, settings config.Settings) error {
	if s == nil || s.store == nil {
		return errSettingsStoreRequired
	}

	normalized, err := s.normalizeSettings(settings)
	if err != nil {
		return err
	}

	return s.store.Save(normalized)
}

func (s *SettingsService) SetSecret(_ context.Context, id config.SecretID, secret string) error {
	if s == nil || s.secretStore == nil {
		return errSecretStoreRequired
	}

	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return &domain.ValidationError{
			Code:    domain.ValidationCodeRequired,
			Field:   "secret",
			Message: "value is required",
		}
	}

	return s.secretStore.Set(id, trimmed)
}

func (s *SettingsService) DeleteSecret(_ context.Context, id config.SecretID) error {
	if s == nil || s.secretStore == nil {
		return errSecretStoreRequired
	}

	return s.secretStore.Delete(id)
}

func (s *SettingsService) normalizeSettings(settings config.Settings) (config.Settings, error) {
	return NormalizeSettings(settings, s.defaults)
}

func NormalizeSettings(settings config.Settings, defaults config.Settings) (config.Settings, error) {
	settings.SchemaVersion = defaults.SchemaVersion

	if err := validateBillingModeSetting("cli_billing_defaults.claude_code", settings.CLIBillingDefaults.ClaudeCode); err != nil {
		return config.Settings{}, err
	}
	if err := validateBillingModeSetting("cli_billing_defaults.codex", settings.CLIBillingDefaults.Codex); err != nil {
		return config.Settings{}, err
	}
	if err := validateBillingModeSetting("cli_billing_defaults.gemini_cli", settings.CLIBillingDefaults.GeminiCLI); err != nil {
		return config.Settings{}, err
	}
	if err := validateBillingModeSetting("cli_billing_defaults.opencode", settings.CLIBillingDefaults.OpenCode); err != nil {
		return config.Settings{}, err
	}

	var err error
	settings.SubscriptionDefaults.OpenAI, err = normalizeSubscriptionPlanSetting("subscription_defaults.openai", settings.SubscriptionDefaults.OpenAI)
	if err != nil {
		return config.Settings{}, err
	}
	settings.SubscriptionDefaults.Claude, err = normalizeSubscriptionPlanSetting("subscription_defaults.claude", settings.SubscriptionDefaults.Claude)
	if err != nil {
		return config.Settings{}, err
	}
	settings.SubscriptionDefaults.Gemini, err = normalizeSubscriptionPlanSetting("subscription_defaults.gemini", settings.SubscriptionDefaults.Gemini)
	if err != nil {
		return config.Settings{}, err
	}

	for field, value := range map[string]float64{
		"budgets.monthly_budget_usd":              settings.Budgets.MonthlyBudgetUSD,
		"budgets.monthly_subscription_budget_usd": settings.Budgets.MonthlySubscriptionBudgetUSD,
		"budgets.monthly_usage_budget_usd":        settings.Budgets.MonthlyUsageBudgetUSD,
	} {
		if value < 0 {
			return config.Settings{}, &domain.ValidationError{
				Code:    domain.ValidationCodeNegativeCost,
				Field:   field,
				Message: "value must be non-negative",
			}
		}
	}

	if err := validatePercentSetting("budgets.warning_threshold_percent", settings.Budgets.WarningThresholdPercent); err != nil {
		return config.Settings{}, err
	}
	if err := validatePercentSetting("budgets.critical_threshold_percent", settings.Budgets.CriticalThresholdPercent); err != nil {
		return config.Settings{}, err
	}
	if settings.Budgets.CriticalThresholdPercent < settings.Budgets.WarningThresholdPercent {
		return config.Settings{}, &domain.ValidationError{
			Code:    domain.ValidationCodeInvalidThreshold,
			Field:   "budgets.critical_threshold_percent",
			Message: "critical threshold must be greater than or equal to warning threshold",
		}
	}

	return settings, nil
}

func validateBillingModeSetting(field string, mode config.BillingMode) error {
	switch mode {
	case config.BillingModeSubscription, config.BillingModeBYOK:
		return nil
	default:
		return &domain.ValidationError{
			Code:    domain.ValidationCodeInvalidBillingMode,
			Field:   field,
			Message: "billing mode must be either subscription or byok",
		}
	}
}

func validatePercentSetting(field string, value int) error {
	if value < 1 || value > 100 {
		return &domain.ValidationError{
			Code:    domain.ValidationCodeInvalidThreshold,
			Field:   field,
			Message: "threshold percent must be between 1 and 100",
		}
	}

	return nil
}

func normalizeSubscriptionPlanSetting(field string, plan config.SubscriptionPlanSettings) (config.SubscriptionPlanSettings, error) {
	plan.PlanCode = strings.TrimSpace(plan.PlanCode)
	plan.PlanName = strings.TrimSpace(plan.PlanName)
	plan.SourceURL = strings.TrimSpace(plan.SourceURL)

	if plan.PlanCode == "" {
		return config.SubscriptionPlanSettings{}, &domain.ValidationError{
			Code:    domain.ValidationCodeRequired,
			Field:   field + ".plan_code",
			Message: "value is required",
		}
	}
	if plan.PlanName == "" {
		return config.SubscriptionPlanSettings{}, &domain.ValidationError{
			Code:    domain.ValidationCodeRequired,
			Field:   field + ".plan_name",
			Message: "value is required",
		}
	}
	if plan.FeeUSD < 0 {
		return config.SubscriptionPlanSettings{}, &domain.ValidationError{
			Code:    domain.ValidationCodeNegativeCost,
			Field:   field + ".fee_usd",
			Message: "value must be non-negative",
		}
	}
	if plan.RenewalDay < 1 || plan.RenewalDay > 31 {
		return config.SubscriptionPlanSettings{}, &domain.ValidationError{
			Code:    domain.ValidationCodeInvalidTimestamp,
			Field:   field + ".renewal_day",
			Message: "renewal day must be between 1 and 31",
		}
	}

	return plan, nil
}
