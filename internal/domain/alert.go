package domain

import (
	"strings"
	"time"
)

type AlertKind string

const (
	AlertKindBudgetThreshold AlertKind = "budget_threshold"
	AlertKindBudgetOverrun   AlertKind = "budget_overrun"
	AlertKindForecastOverrun AlertKind = "forecast_overrun"
	AlertKindInsightDetected AlertKind = "insight_detected"
)

type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

func (k AlertKind) IsValid() bool {
	switch k {
	case AlertKindBudgetThreshold, AlertKindBudgetOverrun, AlertKindForecastOverrun, AlertKindInsightDetected:
		return true
	default:
		return false
	}
}

func (s AlertSeverity) IsValid() bool {
	switch s {
	case AlertSeverityInfo, AlertSeverityWarning, AlertSeverityCritical:
		return true
	default:
		return false
	}
}

type AlertEvent struct {
	AlertID          string
	Kind             AlertKind
	Severity         AlertSeverity
	TriggeredAt      time.Time
	Period           MonthlyPeriod
	BudgetID         string
	ForecastID       string
	InsightID        string
	DetectorCategory DetectorCategory
	CurrentSpendUSD  float64
	LimitUSD         float64
	ThresholdPercent float64
}

func NewAlertEvent(event AlertEvent) (AlertEvent, error) {
	if strings.TrimSpace(event.AlertID) == "" {
		return AlertEvent{}, requiredError("alert_id")
	}

	if !event.Kind.IsValid() {
		return AlertEvent{}, &ValidationError{
			Code:    ValidationCodeInvalidAlertKind,
			Field:   "kind",
			Message: "alert kind must be one of budget_threshold, budget_overrun, forecast_overrun, or insight_detected",
		}
	}

	if !event.Severity.IsValid() {
		return AlertEvent{}, &ValidationError{
			Code:    ValidationCodeInvalidAlertLevel,
			Field:   "severity",
			Message: "severity must be one of info, warning, or critical",
		}
	}

	triggeredAt, err := NormalizeUTCTimestamp("triggered_at", event.TriggeredAt)
	if err != nil {
		return AlertEvent{}, err
	}
	event.TriggeredAt = triggeredAt

	for field, value := range map[string]float64{
		"current_spend_usd": event.CurrentSpendUSD,
		"limit_usd":         event.LimitUSD,
		"threshold_percent": event.ThresholdPercent,
	} {
		if value < 0 {
			return AlertEvent{}, &ValidationError{
				Code:    ValidationCodeNegativeCost,
				Field:   field,
				Message: "alert numeric values must be non-negative",
			}
		}
	}

	switch event.Kind {
	case AlertKindBudgetThreshold, AlertKindBudgetOverrun:
		if strings.TrimSpace(event.BudgetID) == "" {
			return AlertEvent{}, requiredError("budget_id")
		}
	case AlertKindForecastOverrun:
		if strings.TrimSpace(event.ForecastID) == "" {
			return AlertEvent{}, requiredError("forecast_id")
		}
	case AlertKindInsightDetected:
		if strings.TrimSpace(event.InsightID) == "" {
			return AlertEvent{}, requiredError("insight_id")
		}
		if !event.DetectorCategory.IsValid() {
			return AlertEvent{}, &ValidationError{
				Code:    ValidationCodeInvalidDetector,
				Field:   "detector_category",
				Message: "detector category must be a supported waste-pattern family",
			}
		}
	}

	if event.Kind == AlertKindBudgetOverrun && event.CurrentSpendUSD <= event.LimitUSD {
		return AlertEvent{}, &ValidationError{
			Code:    ValidationCodeInvalidThreshold,
			Field:   "current_spend_usd",
			Message: "budget overrun alerts require current spend greater than the budget limit",
		}
	}

	return event, nil
}
