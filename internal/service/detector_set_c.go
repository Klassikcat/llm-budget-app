package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

const (
	metadataToolCallCountKey         = "mcp_tool_call_count"
	metadataObservedToolCallCountKey = "observed_tool_call_count"
	metadataToolSchemaBytesKey       = "tool_schema_bytes"
	metadataToolDefinitionCountKey   = "tool_definition_count"
	metadataToolSchemaOccurrencesKey = "tool_schema_occurrences"

	tokensPerEstimatedSchemaByte       = 4.0
	overqualifiedMaxOutputTokens       = 300
	overqualifiedMaxToolCalls          = 1
	overqualifiedMaxInputTokens        = 4000
	overqualifiedMaxDurationSeconds    = 120
	overqualifiedMinHintCount          = 3
	overqualifiedMinWasteUSD           = 0.01
	overqualifiedMinSavingsRatio       = 2.0
	toolSchemaBloatMinBytes            = 8192
	toolSchemaBloatMinEstimatedTokens  = 2000
	toolSchemaBloatMinInputTokenRatio  = 0.25
	toolSchemaBloatMinOccurrences      = 2
	toolSchemaBloatMinEstimatedWasteUS = 0.005
)

type OverQualifiedModelDetector struct {
	catalog ports.PriceCatalog
}

func NewOverQualifiedModelDetector(catalog ports.PriceCatalog) *OverQualifiedModelDetector {
	return &OverQualifiedModelDetector{catalog: catalog}
}

func (d *OverQualifiedModelDetector) Category() domain.DetectorCategory {
	return domain.DetectorOverQualifiedModel
}

func (d *OverQualifiedModelDetector) Detect(ctx context.Context, period domain.MonthlyPeriod, sessions []domain.SessionSummary, usageEntries []domain.UsageEntry) ([]domain.Insight, error) {
	if d == nil || d.catalog == nil {
		return nil, errPriceCatalogRequired
	}

	toolCallsBySession := sumToolCallsBySession(usageEntries)
	insights := make([]domain.Insight, 0)

	for _, session := range sessions {
		if session.PricingRef == nil {
			continue
		}

		currentPrice, err := d.catalog.LookupModelPrice(ctx, *session.PricingRef, session.EndedAt)
		if err != nil {
			continue
		}

		providerPrices, err := d.catalog.ListProviderPrices(ctx, session.PricingRef.Provider)
		if err != nil {
			return nil, err
		}

		toolCalls := toolCallsBySession[session.SessionID]
		complexityHints := overqualifiedHintCount(session, toolCalls)
		if complexityHints < overqualifiedMinHintCount {
			continue
		}

		currentEstimated, err := currentPrice.Calculate(session.Tokens, toolCalls)
		if err != nil {
			return nil, err
		}
		if currentEstimated.TotalUSD <= 0 {
			continue
		}

		candidate, candidateEstimated, ok, err := cheapestModelAlternative(session, toolCalls, providerPrices)
		if err != nil {
			return nil, err
		}
		if !ok || candidateEstimated.TotalUSD <= 0 {
			continue
		}

		wasteUSD := currentEstimated.TotalUSD - candidateEstimated.TotalUSD
		if wasteUSD < overqualifiedMinWasteUSD {
			continue
		}

		savingsRatio := currentEstimated.TotalUSD / candidateEstimated.TotalUSD
		if savingsRatio < overqualifiedMinSavingsRatio {
			continue
		}

		insight, err := buildOverqualifiedInsight(period, session, currentPrice, candidate, toolCalls, complexityHints, currentEstimated.TotalUSD, candidateEstimated.TotalUSD, wasteUSD, len(providerPrices))
		if err != nil {
			return nil, err
		}
		insights = append(insights, insight)
	}

	return insights, nil
}

type ToolSchemaBloatDetector struct {
	catalog ports.PriceCatalog
}

func NewToolSchemaBloatDetector(catalog ports.PriceCatalog) *ToolSchemaBloatDetector {
	return &ToolSchemaBloatDetector{catalog: catalog}
}

func (d *ToolSchemaBloatDetector) Category() domain.DetectorCategory {
	return domain.DetectorToolSchemaBloat
}

func (d *ToolSchemaBloatDetector) Detect(ctx context.Context, period domain.MonthlyPeriod, sessions []domain.SessionSummary, usageEntries []domain.UsageEntry) ([]domain.Insight, error) {
	if d == nil || d.catalog == nil {
		return nil, errPriceCatalogRequired
	}

	metricsBySession := aggregateToolSchemaMetrics(usageEntries)
	insights := make([]domain.Insight, 0)

	for _, session := range sessions {
		if session.PricingRef == nil {
			continue
		}

		agg := metricsBySession[session.SessionID]
		if agg.schemaBytes < toolSchemaBloatMinBytes || agg.schemaOccurrences < toolSchemaBloatMinOccurrences {
			continue
		}

		price, err := d.catalog.LookupModelPrice(ctx, *session.PricingRef, session.EndedAt)
		if err != nil {
			continue
		}

		estimatedSchemaTokens := estimatedSchemaTokens(agg.schemaBytes)
		if estimatedSchemaTokens < toolSchemaBloatMinEstimatedTokens {
			continue
		}

		inputTokens := session.Tokens.InputTokens
		if inputTokens <= 0 {
			continue
		}

		schemaRatio := float64(estimatedSchemaTokens) / float64(inputTokens)
		if schemaRatio < toolSchemaBloatMinInputTokenRatio {
			continue
		}

		estimatedWasteUSD := float64(estimatedSchemaTokens) / 1_000_000.0 * price.InputUSDPer1M
		if estimatedWasteUSD < toolSchemaBloatMinEstimatedWasteUS {
			continue
		}

		insight, err := buildToolSchemaBloatInsight(period, session, agg, estimatedSchemaTokens, schemaRatio, estimatedWasteUSD)
		if err != nil {
			return nil, err
		}
		insights = append(insights, insight)
	}

	return insights, nil
}

type toolSchemaAggregate struct {
	schemaBytes       int64
	definitionCount   int64
	schemaOccurrences int64
	toolCalls         int64
	entryIDs          []string
}

func aggregateToolSchemaMetrics(entries []domain.UsageEntry) map[string]toolSchemaAggregate {
	bySession := make(map[string]toolSchemaAggregate)
	for _, entry := range entries {
		if strings.TrimSpace(entry.SessionID) == "" {
			continue
		}

		agg := bySession[entry.SessionID]
		agg.schemaBytes += metadataInt64(entry.Metadata, metadataToolSchemaBytesKey)
		agg.definitionCount += metadataInt64(entry.Metadata, metadataToolDefinitionCountKey)

		occurrences := metadataInt64(entry.Metadata, metadataToolSchemaOccurrencesKey)
		if occurrences == 0 && metadataInt64(entry.Metadata, metadataToolSchemaBytesKey) > 0 {
			occurrences = 1
		}
		agg.schemaOccurrences += occurrences
		agg.toolCalls += metadataToolCalls(entry.Metadata)
		agg.entryIDs = append(agg.entryIDs, entry.EntryID)
		bySession[entry.SessionID] = agg
	}

	for sessionID, agg := range bySession {
		sort.Strings(agg.entryIDs)
		bySession[sessionID] = agg
	}

	return bySession
}

func sumToolCallsBySession(entries []domain.UsageEntry) map[string]int64 {
	bySession := make(map[string]int64)
	for _, entry := range entries {
		if strings.TrimSpace(entry.SessionID) == "" {
			continue
		}
		bySession[entry.SessionID] += metadataToolCalls(entry.Metadata)
	}
	return bySession
}

func metadataToolCalls(metadata map[string]string) int64 {
	if count := metadataInt64(metadata, metadataToolCallCountKey); count > 0 {
		return count
	}
	return metadataInt64(metadata, metadataObservedToolCallCountKey)
}

func metadataInt64(metadata map[string]string, key string) int64 {
	if len(metadata) == 0 {
		return 0
	}
	value, err := strconv.ParseInt(strings.TrimSpace(metadata[key]), 10, 64)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func cheapestModelAlternative(session domain.SessionSummary, toolCalls int64, prices []ports.ModelPrice) (ports.ModelPrice, domain.CostBreakdown, bool, error) {
	var (
		bestPrice ports.ModelPrice
		bestCost  domain.CostBreakdown
		bestFound bool
	)

	for _, candidate := range prices {
		if strings.EqualFold(candidate.ModelID, session.PricingRef.ModelID) {
			continue
		}

		candidateCost, err := candidate.Calculate(session.Tokens, toolCalls)
		if err != nil {
			return ports.ModelPrice{}, domain.CostBreakdown{}, false, err
		}
		if candidateCost.TotalUSD <= 0 {
			continue
		}
		if !bestFound || candidateCost.TotalUSD < bestCost.TotalUSD {
			bestFound = true
			bestPrice = candidate
			bestCost = candidateCost
		}
	}

	return bestPrice, bestCost, bestFound, nil
}

func overqualifiedHintCount(session domain.SessionSummary, toolCalls int64) int64 {
	var hints int64
	if session.Tokens.OutputTokens <= overqualifiedMaxOutputTokens {
		hints++
	}
	if toolCalls <= overqualifiedMaxToolCalls {
		hints++
	}
	if session.Tokens.InputTokens <= overqualifiedMaxInputTokens {
		hints++
	}
	if session.Duration() <= time.Duration(overqualifiedMaxDurationSeconds)*time.Second {
		hints++
	}
	return hints
}

func buildOverqualifiedInsight(period domain.MonthlyPeriod, session domain.SessionSummary, current ports.ModelPrice, candidate ports.ModelPrice, toolCalls, hintCount int64, currentEstimatedUSD, baselineUSD, wasteUSD float64, candidateCount int) (domain.Insight, error) {
	hashes := mustTask19InsightHashes(
		mustTask19InsightHash("provider", session.Provider.String()),
		mustTask19InsightHash("current_model", current.ModelID),
		mustTask19InsightHash("recommended_model", candidate.ModelID),
	)
	counts := mustTask19InsightCounts(
		mustTask19InsightCount("trivial_task_hints", hintCount),
		mustTask19InsightCount("observed_tool_calls", toolCalls),
		mustTask19InsightCount("candidate_model_count", int64(candidateCount)),
	)
	metrics := mustTask19InsightMetrics(
		mustTask19InsightMetric("input_tokens", domain.InsightMetricUnitTokens, float64(session.Tokens.InputTokens)),
		mustTask19InsightMetric("output_tokens", domain.InsightMetricUnitTokens, float64(session.Tokens.OutputTokens)),
		mustTask19InsightMetric("session_duration_seconds", domain.InsightMetricUnitSeconds, session.Duration().Seconds()),
		mustTask19InsightMetric("current_estimated_cost_usd", domain.InsightMetricUnitUSD, currentEstimatedUSD),
		mustTask19InsightMetric("baseline_estimated_cost_usd", domain.InsightMetricUnitUSD, baselineUSD),
		mustTask19InsightMetric("estimated_waste_usd", domain.InsightMetricUnitUSD, wasteUSD),
		mustTask19InsightMetric("savings_ratio", domain.InsightMetricUnitRatio, currentEstimatedUSD/baselineUSD),
	)

	payload, err := domain.NewInsightPayload([]string{session.SessionID}, nil, hashes, counts, metrics)
	if err != nil {
		return domain.Insight{}, err
	}

	insight, err := domain.NewInsight(domain.Insight{
		InsightID:  stableDetectorInsightID(domain.DetectorOverQualifiedModel, session.SessionID, current.ModelID, candidate.ModelID),
		Category:   domain.DetectorOverQualifiedModel,
		Severity:   detectorSeverityFromWaste(wasteUSD),
		DetectedAt: session.EndedAt,
		Period:     period,
		Payload:    payload,
	})
	if err != nil {
		return domain.Insight{}, err
	}

	return insight, nil
}

func buildToolSchemaBloatInsight(period domain.MonthlyPeriod, session domain.SessionSummary, agg toolSchemaAggregate, estimatedTokens int64, schemaRatio, wasteUSD float64) (domain.Insight, error) {
	hashes := mustTask19InsightHashes(
		mustTask19InsightHash("provider", session.Provider.String()),
		mustTask19InsightHash("current_model", session.PricingRef.ModelID),
	)
	counts := mustTask19InsightCounts(
		mustTask19InsightCount("tool_schema_bytes", agg.schemaBytes),
		mustTask19InsightCount("tool_definition_count", agg.definitionCount),
		mustTask19InsightCount("schema_occurrences", agg.schemaOccurrences),
		mustTask19InsightCount("observed_tool_calls", agg.toolCalls),
	)
	metrics := mustTask19InsightMetrics(
		mustTask19InsightMetric("input_tokens", domain.InsightMetricUnitTokens, float64(session.Tokens.InputTokens)),
		mustTask19InsightMetric("estimated_schema_tokens", domain.InsightMetricUnitTokens, float64(estimatedTokens)),
		mustTask19InsightMetric("schema_input_ratio", domain.InsightMetricUnitRatio, schemaRatio),
		mustTask19InsightMetric("estimated_waste_usd", domain.InsightMetricUnitUSD, wasteUSD),
	)

	payload, err := domain.NewInsightPayload([]string{session.SessionID}, agg.entryIDs, hashes, counts, metrics)
	if err != nil {
		return domain.Insight{}, err
	}

	insight, err := domain.NewInsight(domain.Insight{
		InsightID:  stableDetectorInsightID(domain.DetectorToolSchemaBloat, session.SessionID, session.PricingRef.ModelID, fmt.Sprintf("%d", agg.schemaBytes)),
		Category:   domain.DetectorToolSchemaBloat,
		Severity:   detectorSeverityFromWaste(wasteUSD),
		DetectedAt: session.EndedAt,
		Period:     period,
		Payload:    payload,
	})
	if err != nil {
		return domain.Insight{}, err
	}

	return insight, nil
}

func estimatedSchemaTokens(schemaBytes int64) int64 {
	return int64(math.Ceil(float64(schemaBytes) / tokensPerEstimatedSchemaByte))
}

func stableDetectorInsightID(category domain.DetectorCategory, parts ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(append([]string{string(category)}, parts...), "|")))
	return string(category) + ":" + hex.EncodeToString(hash[:8])
}

func detectorSeverityFromWaste(wasteUSD float64) domain.InsightSeverity {
	switch {
	case wasteUSD >= 0.05:
		return domain.InsightSeverityHigh
	case wasteUSD >= 0.02:
		return domain.InsightSeverityMedium
	default:
		return domain.InsightSeverityLow
	}
}

func mustTask19InsightHash(kind, value string) domain.InsightHash {
	hash, err := domain.NewInsightHash(kind, value)
	if err != nil {
		panic(err)
	}
	return hash
}

func mustTask19InsightHashes(values ...domain.InsightHash) []domain.InsightHash {
	return append([]domain.InsightHash(nil), values...)
}

func mustTask19InsightCount(key string, value int64) domain.InsightCount {
	count, err := domain.NewInsightCount(key, value)
	if err != nil {
		panic(err)
	}
	return count
}

func mustTask19InsightCounts(values ...domain.InsightCount) []domain.InsightCount {
	return append([]domain.InsightCount(nil), values...)
}

func mustTask19InsightMetric(key string, unit domain.InsightMetricUnit, value float64) domain.InsightMetric {
	metric, err := domain.NewInsightMetric(key, unit, value)
	if err != nil {
		panic(err)
	}
	return metric
}

func mustTask19InsightMetrics(values ...domain.InsightMetric) []domain.InsightMetric {
	return append([]domain.InsightMetric(nil), values...)
}
