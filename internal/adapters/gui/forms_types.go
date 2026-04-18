package gui

type FormError struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type MutationResponse struct {
	Success bool       `json:"success"`
	Error   *FormError `json:"error,omitempty"`
}

type SettingsFormInput struct {
	Providers            ProviderSettingsState     `json:"providers"`
	CLIBillingDefaults   CLIBillingDefaultsState   `json:"cliBillingDefaults"`
	SubscriptionDefaults SubscriptionDefaultsState `json:"subscriptionDefaults"`
	Budgets              BudgetSettingsState       `json:"budgets"`
	Notifications        NotificationSettingsState `json:"notifications"`
}

type SettingsFormResponse struct {
	Result   MutationResponse  `json:"result"`
	Settings SettingsFormState `json:"settings"`
}

type SettingsFormState struct {
	Providers            ProviderSettingsState     `json:"providers"`
	CLIBillingDefaults   CLIBillingDefaultsState   `json:"cliBillingDefaults"`
	SubscriptionDefaults SubscriptionDefaultsState `json:"subscriptionDefaults"`
	Budgets              BudgetSettingsState       `json:"budgets"`
	Notifications        NotificationSettingsState `json:"notifications"`
}

type SubscriptionDefaultsState struct {
	OpenAI SubscriptionPlanState `json:"openai"`
	Claude SubscriptionPlanState `json:"claude"`
	Gemini SubscriptionPlanState `json:"gemini"`
}

type SubscriptionPlanState struct {
	Enabled    bool    `json:"enabled"`
	PlanCode   string  `json:"planCode"`
	PlanName   string  `json:"planName"`
	FeeUSD     float64 `json:"feeUsd"`
	RenewalDay int     `json:"renewalDay"`
	SourceURL  string  `json:"sourceUrl"`
}

type ProviderSettingsState struct {
	AnthropicEnabled  bool `json:"anthropicEnabled"`
	OpenAIEnabled     bool `json:"openaiEnabled"`
	GeminiEnabled     bool `json:"geminiEnabled"`
	OpenRouterEnabled bool `json:"openRouterEnabled"`
}

type CLIBillingDefaultsState struct {
	ClaudeCode string `json:"claudeCode"`
	Codex      string `json:"codex"`
	GeminiCLI  string `json:"geminiCli"`
	OpenCode   string `json:"openCode"`
}

type BudgetSettingsState struct {
	MonthlyBudgetUSD             float64 `json:"monthlyBudgetUsd"`
	MonthlySubscriptionBudgetUSD float64 `json:"monthlySubscriptionBudgetUsd"`
	MonthlyUsageBudgetUSD        float64 `json:"monthlyUsageBudgetUsd"`
	WarningThresholdPercent      int     `json:"warningThresholdPercent"`
	CriticalThresholdPercent     int     `json:"criticalThresholdPercent"`
}

type NotificationSettingsState struct {
	DesktopEnabled      bool `json:"desktopEnabled"`
	TUIEnabled          bool `json:"tuiEnabled"`
	BudgetWarnings      bool `json:"budgetWarnings"`
	ForecastWarnings    bool `json:"forecastWarnings"`
	ProviderSyncFailure bool `json:"providerSyncFailure"`
}

type ProviderSecretInput struct {
	Provider   string `json:"provider"`
	SecretType string `json:"secretType"`
	Value      string `json:"value"`
}

type ProviderSecretDeleteInput struct {
	Provider   string `json:"provider"`
	SecretType string `json:"secretType"`
}

type SubscriptionFormInput struct {
	SubscriptionID string  `json:"subscriptionId"`
	Provider       string  `json:"provider"`
	PlanCode       string  `json:"planCode"`
	PlanName       string  `json:"planName"`
	RenewalDay     int     `json:"renewalDay"`
	StartsAt       string  `json:"startsAt"`
	EndsAt         string  `json:"endsAt"`
	FeeUSD         float64 `json:"feeUsd"`
	IsActive       bool    `json:"isActive"`
}

type SubscriptionMutationResponse struct {
	Result       MutationResponse  `json:"result"`
	Subscription SubscriptionState `json:"subscription"`
}

type SubscriptionState struct {
	SubscriptionID string  `json:"subscriptionId"`
	Provider       string  `json:"provider"`
	PlanCode       string  `json:"planCode"`
	PlanName       string  `json:"planName"`
	RenewalDay     int     `json:"renewalDay"`
	StartsAt       string  `json:"startsAt"`
	EndsAt         string  `json:"endsAt,omitempty"`
	FeeUSD         float64 `json:"feeUsd"`
	IsActive       bool    `json:"isActive"`
}

type ManualEntryFormInput struct {
	Provider         string            `json:"provider"`
	ModelID          string            `json:"modelId"`
	OccurredAt       string            `json:"occurredAt"`
	InputTokens      int64             `json:"inputTokens"`
	OutputTokens     int64             `json:"outputTokens"`
	CachedTokens     int64             `json:"cachedTokens"`
	CacheWriteTokens int64             `json:"cacheWriteTokens"`
	ProjectName      string            `json:"projectName"`
	Metadata         map[string]string `json:"metadata"`
}

type ManualEntryMutationResponse struct {
	Result MutationResponse `json:"result"`
	Entry  ManualEntryState `json:"entry"`
}

type ManualEntryState struct {
	EntryID          string            `json:"entryId"`
	Provider         string            `json:"provider"`
	ModelID          string            `json:"modelId"`
	OccurredAt       string            `json:"occurredAt"`
	ProjectName      string            `json:"projectName"`
	InputTokens      int64             `json:"inputTokens"`
	OutputTokens     int64             `json:"outputTokens"`
	CachedTokens     int64             `json:"cachedTokens"`
	CacheWriteTokens int64             `json:"cacheWriteTokens"`
	TotalCostUSD     float64           `json:"totalCostUsd"`
	Metadata         map[string]string `json:"metadata"`
}

type BudgetFormInput struct {
	BudgetID                 string  `json:"budgetId"`
	Name                     string  `json:"name"`
	Provider                 string  `json:"provider"`
	ProjectHash              string  `json:"projectHash"`
	PeriodMonth              string  `json:"periodMonth"`
	LimitUSD                 float64 `json:"limitUsd"`
	WarningThresholdPercent  int     `json:"warningThresholdPercent"`
	CriticalThresholdPercent int     `json:"criticalThresholdPercent"`
	Currency                 string  `json:"currency"`
}

type BudgetMutationResponse struct {
	Result MutationResponse `json:"result"`
	Budget BudgetState      `json:"budget"`
}

type BudgetState struct {
	BudgetID                 string  `json:"budgetId"`
	Name                     string  `json:"name"`
	Provider                 string  `json:"provider,omitempty"`
	ProjectHash              string  `json:"projectHash,omitempty"`
	PeriodMonth              string  `json:"periodMonth"`
	LimitUSD                 float64 `json:"limitUsd"`
	WarningThresholdPercent  int     `json:"warningThresholdPercent"`
	CriticalThresholdPercent int     `json:"criticalThresholdPercent,omitempty"`
	Currency                 string  `json:"currency"`
}

type AlertNotificationInput struct {
	AlertID          string  `json:"alertId"`
	Kind             string  `json:"kind"`
	Severity         string  `json:"severity"`
	TriggeredAt      string  `json:"triggeredAt"`
	PeriodMonth      string  `json:"periodMonth"`
	BudgetID         string  `json:"budgetId"`
	ForecastID       string  `json:"forecastId"`
	InsightID        string  `json:"insightId"`
	DetectorCategory string  `json:"detectorCategory"`
	CurrentSpendUSD  float64 `json:"currentSpendUsd"`
	LimitUSD         float64 `json:"limitUsd"`
	ThresholdPercent float64 `json:"thresholdPercent"`
}

type NotificationDispatchResponse struct {
	Result     MutationResponse `json:"result"`
	Dispatched bool             `json:"dispatched"`
}
