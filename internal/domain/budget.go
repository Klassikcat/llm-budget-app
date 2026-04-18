package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type BudgetThreshold struct {
	Severity AlertSeverity
	Percent  float64
}

func NewBudgetThreshold(severity AlertSeverity, percent float64) (BudgetThreshold, error) {
	if !severity.IsValid() {
		return BudgetThreshold{}, &ValidationError{
			Code:    ValidationCodeInvalidAlertLevel,
			Field:   "severity",
			Message: "severity must be one of info, warning, or critical",
		}
	}

	if percent <= 0 || percent >= 1 {
		return BudgetThreshold{}, &ValidationError{
			Code:    ValidationCodeInvalidThreshold,
			Field:   "percent",
			Message: "budget threshold percent must be greater than 0 and less than 1",
		}
	}

	return BudgetThreshold{Severity: severity, Percent: percent}, nil
}

type MonthlyBudget struct {
	BudgetID    string
	Name        string
	Period      MonthlyPeriod
	LimitUSD    float64
	Thresholds  []BudgetThreshold
	Currency    string
	Provider    ProviderName
	ProjectHash string
}

type BudgetStatus struct {
	TriggeredThresholds []BudgetThreshold
	RemainingUSD        float64
	IsOverrun           bool
}

func NewMonthlyBudget(budget MonthlyBudget) (MonthlyBudget, error) {
	if strings.TrimSpace(budget.BudgetID) == "" {
		return MonthlyBudget{}, requiredError("budget_id")
	}

	if budget.Period.StartAt.IsZero() || budget.Period.EndExclusive.IsZero() {
		return MonthlyBudget{}, requiredError("period")
	}

	if budget.LimitUSD <= 0 {
		return MonthlyBudget{}, &ValidationError{
			Code:    ValidationCodeNegativeCost,
			Field:   "limit_usd",
			Message: "budget limit must be greater than zero",
		}
	}

	if len(budget.Thresholds) == 0 {
		return MonthlyBudget{}, requiredError("thresholds")
	}

	thresholds := make([]BudgetThreshold, len(budget.Thresholds))
	copy(thresholds, budget.Thresholds)
	sort.Slice(thresholds, func(i, j int) bool {
		return thresholds[i].Percent < thresholds[j].Percent
	})

	for i, threshold := range thresholds {
		validated, err := NewBudgetThreshold(threshold.Severity, threshold.Percent)
		if err != nil {
			return MonthlyBudget{}, err
		}
		thresholds[i] = validated

		if i > 0 && thresholds[i-1].Percent == validated.Percent {
			return MonthlyBudget{}, &ValidationError{
				Code:    ValidationCodeInvalidThreshold,
				Field:   "thresholds",
				Message: "budget thresholds must use distinct percent values",
			}
		}
	}

	budget.Name = strings.TrimSpace(budget.Name)
	if budget.Currency == "" {
		budget.Currency = "USD"
	}
	budget.Currency = strings.ToUpper(strings.TrimSpace(budget.Currency))
	budget.ProjectHash = strings.TrimSpace(budget.ProjectHash)
	budget.Thresholds = thresholds

	if budget.Provider != "" {
		provider, err := NewProviderName(budget.Provider.String())
		if err != nil {
			return MonthlyBudget{}, err
		}
		budget.Provider = provider
	}

	return budget, nil
}

func (b MonthlyBudget) EvaluateSpend(spendUSD float64) (BudgetStatus, error) {
	if spendUSD < 0 {
		return BudgetStatus{}, &ValidationError{
			Code:    ValidationCodeNegativeCost,
			Field:   "spend_usd",
			Message: "spend must be non-negative",
		}
	}

	status := BudgetStatus{
		RemainingUSD: b.LimitUSD - spendUSD,
		IsOverrun:    spendUSD > b.LimitUSD,
	}

	for _, threshold := range b.Thresholds {
		if spendUSD >= b.LimitUSD*threshold.Percent {
			status.TriggeredThresholds = append(status.TriggeredThresholds, threshold)
		}
	}

	return status, nil
}

type ForecastSnapshot struct {
	ForecastID          string
	Period              MonthlyPeriod
	GeneratedAt         time.Time
	ActualSpendUSD      float64
	ForecastSpendUSD    float64
	BudgetLimitUSD      float64
	ProjectedOverrunUSD float64
	ObservedDayCount    int
	RemainingDayCount   int
}

func NewForecastSnapshot(snapshot ForecastSnapshot) (ForecastSnapshot, error) {
	if strings.TrimSpace(snapshot.ForecastID) == "" {
		return ForecastSnapshot{}, requiredError("forecast_id")
	}

	if snapshot.Period.StartAt.IsZero() || snapshot.Period.EndExclusive.IsZero() {
		return ForecastSnapshot{}, requiredError("period")
	}

	generatedAt, err := NormalizeUTCTimestamp("generated_at", snapshot.GeneratedAt)
	if err != nil {
		return ForecastSnapshot{}, err
	}
	snapshot.GeneratedAt = generatedAt

	for field, value := range map[string]float64{
		"actual_spend_usd":   snapshot.ActualSpendUSD,
		"forecast_spend_usd": snapshot.ForecastSpendUSD,
		"budget_limit_usd":   snapshot.BudgetLimitUSD,
	} {
		if value < 0 {
			return ForecastSnapshot{}, &ValidationError{
				Code:    ValidationCodeNegativeCost,
				Field:   field,
				Message: "forecast values must be non-negative",
			}
		}
	}

	if snapshot.ObservedDayCount < 0 || snapshot.RemainingDayCount < 0 {
		return ForecastSnapshot{}, &ValidationError{
			Code:    ValidationCodeInvalidThreshold,
			Field:   "day_count",
			Message: "forecast day counts must be non-negative",
		}
	}

	snapshot.ProjectedOverrunUSD = snapshot.ForecastSpendUSD - snapshot.BudgetLimitUSD
	if snapshot.ProjectedOverrunUSD < 0 {
		snapshot.ProjectedOverrunUSD = 0
	}

	return snapshot, nil
}

type BudgetState struct {
	BudgetID                   string
	Period                     MonthlyPeriod
	CurrentSpendUSD            float64
	ForecastSpendUSD           float64
	TriggeredThresholdPercents []float64
	BudgetOverrunActive        bool
	ForecastOverrunActive      bool
	UpdatedAt                  time.Time
}

func NewBudgetState(state BudgetState) (BudgetState, error) {
	if strings.TrimSpace(state.BudgetID) == "" {
		return BudgetState{}, requiredError("budget_id")
	}

	if state.Period.StartAt.IsZero() || state.Period.EndExclusive.IsZero() {
		return BudgetState{}, requiredError("period")
	}

	for field, value := range map[string]float64{
		"current_spend_usd":  state.CurrentSpendUSD,
		"forecast_spend_usd": state.ForecastSpendUSD,
	} {
		if value < 0 {
			return BudgetState{}, &ValidationError{
				Code:    ValidationCodeNegativeCost,
				Field:   field,
				Message: "budget state values must be non-negative",
			}
		}
	}

	updatedAt, err := NormalizeUTCTimestamp("updated_at", state.UpdatedAt)
	if err != nil {
		return BudgetState{}, err
	}
	state.UpdatedAt = updatedAt

	thresholds := make([]float64, 0, len(state.TriggeredThresholdPercents))
	seen := make(map[string]struct{}, len(state.TriggeredThresholdPercents))
	for _, percent := range state.TriggeredThresholdPercents {
		if percent <= 0 || percent >= 1 {
			return BudgetState{}, &ValidationError{
				Code:    ValidationCodeInvalidThreshold,
				Field:   "triggered_threshold_percents",
				Message: "triggered threshold percents must be greater than 0 and less than 1",
			}
		}
		key := fmt.Sprintf("%.6f", percent)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		thresholds = append(thresholds, percent)
	}
	sort.Float64s(thresholds)
	state.TriggeredThresholdPercents = thresholds

	return state, nil
}
