package config

type BillingMode string

const (
	BillingModeSubscription BillingMode = "subscription"
	BillingModeBYOK         BillingMode = "byok"
)

type Settings struct {
	SchemaVersion        int                     `json:"schema_version"`
	Providers            ProviderSettings        `json:"providers"`
	CLIBillingDefaults   CLIBillingDefaults      `json:"cli_billing_defaults"`
	SubscriptionDefaults SubscriptionDefaults    `json:"subscription_defaults"`
	Budgets              BudgetSettings          `json:"budgets"`
	Notifications        NotificationPreferences `json:"notifications"`
}

type SubscriptionDefaults struct {
	OpenAI SubscriptionPlanSettings `json:"openai"`
	Claude SubscriptionPlanSettings `json:"claude"`
	Gemini SubscriptionPlanSettings `json:"gemini"`
}

type SubscriptionPlanSettings struct {
	Enabled    bool    `json:"enabled"`
	PlanCode   string  `json:"plan_code"`
	PlanName   string  `json:"plan_name"`
	FeeUSD     float64 `json:"fee_usd"`
	RenewalDay int     `json:"renewal_day"`
	SourceURL  string  `json:"source_url"`
}

type ProviderSettings struct {
	Anthropic  ProviderToggle `json:"anthropic"`
	OpenAI     ProviderToggle `json:"openai"`
	Gemini     ProviderToggle `json:"gemini"`
	OpenRouter ProviderToggle `json:"openrouter"`
}

type ProviderToggle struct {
	Enabled bool `json:"enabled"`
}

type CLIBillingDefaults struct {
	ClaudeCode BillingMode `json:"claude_code"`
	Codex      BillingMode `json:"codex"`
	GeminiCLI  BillingMode `json:"gemini_cli"`
	OpenCode   BillingMode `json:"opencode"`
}

type BudgetSettings struct {
	MonthlyBudgetUSD             float64 `json:"monthly_budget_usd"`
	MonthlySubscriptionBudgetUSD float64 `json:"monthly_subscription_budget_usd"`
	MonthlyUsageBudgetUSD        float64 `json:"monthly_usage_budget_usd"`
	WarningThresholdPercent      int     `json:"warning_threshold_percent"`
	CriticalThresholdPercent     int     `json:"critical_threshold_percent"`
}

type NotificationPreferences struct {
	DesktopEnabled      bool `json:"desktop_enabled"`
	TUIEnabled          bool `json:"tui_enabled"`
	BudgetWarnings      bool `json:"budget_warnings"`
	ForecastWarnings    bool `json:"forecast_warnings"`
	ProviderSyncFailure bool `json:"provider_sync_failure"`
}

func DefaultSettings() Settings {
	return Settings{
		SchemaVersion: 1,
		Providers: ProviderSettings{
			Anthropic:  ProviderToggle{Enabled: true},
			OpenAI:     ProviderToggle{Enabled: true},
			Gemini:     ProviderToggle{Enabled: true},
			OpenRouter: ProviderToggle{Enabled: true},
		},
		CLIBillingDefaults: CLIBillingDefaults{
			ClaudeCode: BillingModeSubscription,
			Codex:      BillingModeSubscription,
			GeminiCLI:  BillingModeSubscription,
			OpenCode:   BillingModeBYOK,
		},
		SubscriptionDefaults: SubscriptionDefaults{
			OpenAI: SubscriptionPlanSettings{
				Enabled:    false,
				PlanCode:   "chatgpt-plus",
				PlanName:   "ChatGPT Plus",
				FeeUSD:     20,
				RenewalDay: 1,
				SourceURL:  "https://openai.com/ChatGPT/pricing",
			},
			Claude: SubscriptionPlanSettings{
				Enabled:    false,
				PlanCode:   "claude-pro",
				PlanName:   "Claude Pro",
				FeeUSD:     20,
				RenewalDay: 1,
				SourceURL:  "https://claude.com/pricing",
			},
			Gemini: SubscriptionPlanSettings{
				Enabled:    false,
				PlanCode:   "google-ai-pro",
				PlanName:   "Google AI Pro",
				FeeUSD:     19.99,
				RenewalDay: 1,
				SourceURL:  "https://gemini.google/us/subscriptions/",
			},
		},
		Budgets: BudgetSettings{
			MonthlyBudgetUSD:             250,
			MonthlySubscriptionBudgetUSD: 100,
			MonthlyUsageBudgetUSD:        150,
			WarningThresholdPercent:      80,
			CriticalThresholdPercent:     100,
		},
		Notifications: NotificationPreferences{
			DesktopEnabled:      true,
			TUIEnabled:          true,
			BudgetWarnings:      true,
			ForecastWarnings:    true,
			ProviderSyncFailure: true,
		},
	}
}
