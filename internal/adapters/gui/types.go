package gui

import (
	"time"

	"llm-budget-tracker/internal/service"
)

type DashboardResponse struct {
	Period            DashboardPeriodResponse            `json:"period"`
	Totals            DashboardTotalsResponse            `json:"totals"`
	ProviderSummaries []DashboardProviderSummaryResponse `json:"providerSummaries"`
	Budgets           []DashboardBudgetResponse          `json:"budgets"`
	RecentSessions    []DashboardRecentSessionResponse   `json:"recentSessions"`
	Empty             bool                               `json:"empty"`
}

type DashboardPeriodResponse struct {
	Month        string `json:"month"`
	StartAt      string `json:"startAt"`
	EndExclusive string `json:"endExclusive"`
	Currency     string `json:"currency"`
}

type DashboardTotalsResponse struct {
	VariableSpendUSD     float64 `json:"variableSpendUsd"`
	SubscriptionSpendUSD float64 `json:"subscriptionSpendUsd"`
	TotalSpendUSD        float64 `json:"totalSpendUsd"`
	Currency             string  `json:"currency"`
}

type DashboardProviderSummaryResponse struct {
	Provider             string  `json:"provider"`
	VariableSpendUSD     float64 `json:"variableSpendUsd"`
	SubscriptionSpendUSD float64 `json:"subscriptionSpendUsd"`
	TotalSpendUSD        float64 `json:"totalSpendUsd"`
	UsageEntryCount      int     `json:"usageEntryCount"`
	SessionCount         int     `json:"sessionCount"`
	Currency             string  `json:"currency"`
}

type DashboardBudgetResponse struct {
	BudgetID                   string    `json:"budgetId"`
	Name                       string    `json:"name"`
	Provider                   string    `json:"provider"`
	ProjectHash                string    `json:"projectHash"`
	LimitUSD                   float64   `json:"limitUsd"`
	CurrentSpendUSD            float64   `json:"currentSpendUsd"`
	RemainingUSD               float64   `json:"remainingUsd"`
	TriggeredThresholdPercents []float64 `json:"triggeredThresholdPercents"`
	BudgetOverrunActive        bool      `json:"budgetOverrunActive"`
	Currency                   string    `json:"currency"`
}

type DashboardRecentSessionResponse struct {
	SessionID      string  `json:"sessionId"`
	Provider       string  `json:"provider"`
	BillingMode    string  `json:"billingMode"`
	ProjectName    string  `json:"projectName"`
	AgentName      string  `json:"agentName"`
	ModelID        string  `json:"modelId"`
	StartedAt      string  `json:"startedAt"`
	EndedAt        string  `json:"endedAt"`
	DurationSecond int64   `json:"durationSeconds"`
	TotalCostUSD   float64 `json:"totalCostUsd"`
	TotalTokens    int64   `json:"totalTokens"`
	Currency       string  `json:"currency"`
}

func toDashboardResponse(snapshot service.DashboardSnapshot) DashboardResponse {
	providerSummaries := make([]DashboardProviderSummaryResponse, 0, len(snapshot.ProviderSummaries))
	for _, summary := range snapshot.ProviderSummaries {
		providerSummaries = append(providerSummaries, DashboardProviderSummaryResponse{
			Provider:             summary.Provider.String(),
			VariableSpendUSD:     summary.VariableSpendUSD,
			SubscriptionSpendUSD: summary.SubscriptionSpendUSD,
			TotalSpendUSD:        summary.TotalSpendUSD,
			UsageEntryCount:      summary.UsageEntryCount,
			SessionCount:         summary.SessionCount,
			Currency:             "USD",
		})
	}

	budgets := make([]DashboardBudgetResponse, 0, len(snapshot.Budgets))
	for _, budget := range snapshot.Budgets {
		triggered := make([]float64, len(budget.TriggeredThresholdPercents))
		copy(triggered, budget.TriggeredThresholdPercents)
		budgets = append(budgets, DashboardBudgetResponse{
			BudgetID:                   budget.BudgetID,
			Name:                       budget.Name,
			Provider:                   budget.Provider.String(),
			ProjectHash:                budget.ProjectHash,
			LimitUSD:                   budget.LimitUSD,
			CurrentSpendUSD:            budget.CurrentSpendUSD,
			RemainingUSD:               budget.RemainingUSD,
			TriggeredThresholdPercents: triggered,
			BudgetOverrunActive:        budget.BudgetOverrunActive,
			Currency:                   budget.Currency,
		})
	}

	recentSessions := make([]DashboardRecentSessionResponse, 0, len(snapshot.RecentSessions))
	for _, session := range snapshot.RecentSessions {
		recentSessions = append(recentSessions, DashboardRecentSessionResponse{
			SessionID:      session.SessionID,
			Provider:       session.Provider.String(),
			BillingMode:    string(session.BillingMode),
			ProjectName:    session.ProjectName,
			AgentName:      session.AgentName,
			ModelID:        session.ModelID,
			StartedAt:      formatDashboardTime(session.StartedAt),
			EndedAt:        formatDashboardTime(session.EndedAt),
			DurationSecond: session.DurationSecond,
			TotalCostUSD:   session.TotalCostUSD,
			TotalTokens:    session.TotalTokens,
			Currency:       "USD",
		})
	}

	return DashboardResponse{
		Period: DashboardPeriodResponse{
			Month:        snapshot.Period.StartAt.Format(dashboardMonthLayout),
			StartAt:      formatDashboardTime(snapshot.Period.StartAt),
			EndExclusive: formatDashboardTime(snapshot.Period.EndExclusive),
			Currency:     "USD",
		},
		Totals: DashboardTotalsResponse{
			VariableSpendUSD:     snapshot.Totals.VariableSpendUSD,
			SubscriptionSpendUSD: snapshot.Totals.SubscriptionSpendUSD,
			TotalSpendUSD:        snapshot.Totals.TotalSpendUSD,
			Currency:             "USD",
		},
		ProviderSummaries: providerSummaries,
		Budgets:           budgets,
		RecentSessions:    recentSessions,
		Empty:             snapshot.Empty,
	}
}

func formatDashboardTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
