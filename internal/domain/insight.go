package domain

import (
	"regexp"
	"strings"
	"time"
)

var insightKeyPattern = regexp.MustCompile(`^[a-z0-9]+(?:_[a-z0-9]+)*$`)
var safeHashPattern = regexp.MustCompile(`^[A-Za-z0-9:_\-/=.]+$`)

type DetectorCategory string

const (
	DetectorContextAvalanche    DetectorCategory = "context_avalanche"
	DetectorRepeatedFileReads   DetectorCategory = "repeated_file_reads"
	DetectorRetryAmplification  DetectorCategory = "retry_amplification"
	DetectorOverQualifiedModel  DetectorCategory = "over_qualified_model_choice"
	DetectorToolSchemaBloat     DetectorCategory = "tool_schema_bloat"
	DetectorPlanningTax         DetectorCategory = "planning_tax"
	DetectorZombieLoops         DetectorCategory = "zombie_loops"
	DetectorMissedPromptCaching DetectorCategory = "missed_prompt_caching"
)

func (c DetectorCategory) IsValid() bool {
	switch c {
	case DetectorContextAvalanche,
		DetectorRepeatedFileReads,
		DetectorRetryAmplification,
		DetectorOverQualifiedModel,
		DetectorToolSchemaBloat,
		DetectorPlanningTax,
		DetectorZombieLoops,
		DetectorMissedPromptCaching:
		return true
	default:
		return false
	}
}

type InsightSeverity string

const (
	InsightSeverityLow    InsightSeverity = "low"
	InsightSeverityMedium InsightSeverity = "medium"
	InsightSeverityHigh   InsightSeverity = "high"
)

func (s InsightSeverity) IsValid() bool {
	switch s {
	case InsightSeverityLow, InsightSeverityMedium, InsightSeverityHigh:
		return true
	default:
		return false
	}
}

type InsightHash struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

func NewInsightHash(kind, value string) (InsightHash, error) {
	kind = strings.TrimSpace(kind)
	if !insightKeyPattern.MatchString(kind) {
		return InsightHash{}, &ValidationError{
			Code:    ValidationCodeInvalidHash,
			Field:   "kind",
			Message: "hash kind must be a lowercase snake_case key",
		}
	}

	value = strings.TrimSpace(value)
	if value == "" || !safeHashPattern.MatchString(value) {
		return InsightHash{}, &ValidationError{
			Code:    ValidationCodeInvalidHash,
			Field:   "value",
			Message: "hash value must be a non-empty privacy-safe token or digest",
		}
	}

	return InsightHash{Kind: kind, Value: value}, nil
}

type InsightCount struct {
	Key   string `json:"key"`
	Value int64  `json:"value"`
}

func NewInsightCount(key string, value int64) (InsightCount, error) {
	key = strings.TrimSpace(key)
	if !insightKeyPattern.MatchString(key) {
		return InsightCount{}, &ValidationError{
			Code:    ValidationCodeInvalidMetric,
			Field:   "key",
			Message: "count key must be a lowercase snake_case key",
		}
	}

	if value < 0 {
		return InsightCount{}, &ValidationError{
			Code:    ValidationCodeNegativeTokens,
			Field:   "value",
			Message: "count values must be non-negative",
		}
	}

	return InsightCount{Key: key, Value: value}, nil
}

type InsightMetricUnit string

const (
	InsightMetricUnitTokens  InsightMetricUnit = "tokens"
	InsightMetricUnitUSD     InsightMetricUnit = "usd"
	InsightMetricUnitRatio   InsightMetricUnit = "ratio"
	InsightMetricUnitCount   InsightMetricUnit = "count"
	InsightMetricUnitSeconds InsightMetricUnit = "seconds"
)

func (u InsightMetricUnit) IsValid() bool {
	switch u {
	case InsightMetricUnitTokens, InsightMetricUnitUSD, InsightMetricUnitRatio, InsightMetricUnitCount, InsightMetricUnitSeconds:
		return true
	default:
		return false
	}
}

type InsightMetric struct {
	Key   string            `json:"key"`
	Unit  InsightMetricUnit `json:"unit"`
	Value float64           `json:"value"`
}

func NewInsightMetric(key string, unit InsightMetricUnit, value float64) (InsightMetric, error) {
	key = strings.TrimSpace(key)
	if !insightKeyPattern.MatchString(key) {
		return InsightMetric{}, &ValidationError{
			Code:    ValidationCodeInvalidMetric,
			Field:   "key",
			Message: "metric key must be a lowercase snake_case key",
		}
	}

	if !unit.IsValid() {
		return InsightMetric{}, &ValidationError{
			Code:    ValidationCodeInvalidMetric,
			Field:   "unit",
			Message: "metric unit must be one of tokens, usd, ratio, count, or seconds",
		}
	}

	if value < 0 {
		return InsightMetric{}, &ValidationError{
			Code:    ValidationCodeInvalidMetric,
			Field:   "value",
			Message: "metric value must be non-negative",
		}
	}

	return InsightMetric{Key: key, Unit: unit, Value: value}, nil
}

type InsightPayload struct {
	SessionIDs    []string        `json:"session_ids,omitempty"`
	UsageEntryIDs []string        `json:"usage_entry_ids,omitempty"`
	Hashes        []InsightHash   `json:"hashes,omitempty"`
	Counts        []InsightCount  `json:"counts,omitempty"`
	Metrics       []InsightMetric `json:"metrics,omitempty"`
}

func NewInsightPayload(sessionIDs, usageEntryIDs []string, hashes []InsightHash, counts []InsightCount, metrics []InsightMetric) (InsightPayload, error) {
	ids := append([]string{}, sessionIDs...)
	ids = append(ids, usageEntryIDs...)
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			return InsightPayload{}, &ValidationError{
				Code:    ValidationCodeRequired,
				Field:   "id",
				Message: "privacy-safe payload identifiers must be non-empty",
			}
		}
	}

	return InsightPayload{
		SessionIDs:    append([]string(nil), sessionIDs...),
		UsageEntryIDs: append([]string(nil), usageEntryIDs...),
		Hashes:        append([]InsightHash(nil), hashes...),
		Counts:        append([]InsightCount(nil), counts...),
		Metrics:       append([]InsightMetric(nil), metrics...),
	}, nil
}

type Insight struct {
	InsightID  string
	Category   DetectorCategory
	Severity   InsightSeverity
	DetectedAt time.Time
	Period     MonthlyPeriod
	Payload    InsightPayload
}

func NewInsight(insight Insight) (Insight, error) {
	if strings.TrimSpace(insight.InsightID) == "" {
		return Insight{}, requiredError("insight_id")
	}

	if !insight.Category.IsValid() {
		return Insight{}, &ValidationError{
			Code:    ValidationCodeInvalidDetector,
			Field:   "category",
			Message: "category must be one of the supported detector families",
		}
	}

	if !insight.Severity.IsValid() {
		return Insight{}, &ValidationError{
			Code:    ValidationCodeInvalidAlertLevel,
			Field:   "severity",
			Message: "severity must be one of low, medium, or high",
		}
	}

	detectedAt, err := NormalizeUTCTimestamp("detected_at", insight.DetectedAt)
	if err != nil {
		return Insight{}, err
	}
	insight.DetectedAt = detectedAt

	if insight.Period.StartAt.IsZero() || insight.Period.EndExclusive.IsZero() {
		period, err := NewMonthlyPeriod(insight.DetectedAt)
		if err != nil {
			return Insight{}, err
		}
		insight.Period = period
	}

	if !insight.Period.Contains(insight.DetectedAt) {
		return Insight{}, &ValidationError{
			Code:    ValidationCodeInvalidTimeRange,
			Field:   "detected_at",
			Message: "detected_at must fall within the insight monthly period",
		}
	}

	return insight, nil
}
