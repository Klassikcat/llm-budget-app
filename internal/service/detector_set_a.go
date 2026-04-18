package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

const (
	contextAvalancheMinRatio        = 4.0
	contextAvalancheMinExcessTokens = int64(2000)
	missedCacheMinEntries           = 3
	missedCacheMinRepeatedTokens    = int64(2000)
	planningTaxMinReasoningTokens   = int64(1000)
	planningTaxMinReasoningRatio    = 2.0
	planningTaxMaxToolCalls         = int64(1)
)

func NewDetectorSetA() []ports.InsightDetector {
	return []ports.InsightDetector{
		ContextAvalancheDetector{},
		MissedPromptCachingDetector{},
		PlanningTaxDetector{},
	}
}

type ContextAvalancheDetector struct{}

func (ContextAvalancheDetector) Category() domain.DetectorCategory {
	return domain.DetectorContextAvalanche
}

func (d ContextAvalancheDetector) Detect(_ context.Context, period domain.MonthlyPeriod, sessions []domain.SessionSummary, usageEntries []domain.UsageEntry) ([]domain.Insight, error) {
	views := buildSessionInsightViews(sessions, usageEntries)
	insights := make([]domain.Insight, 0)
	for _, view := range views {
		productiveOutput := maxInt64(view.session.Tokens.OutputTokens, 1)
		ratio := float64(view.session.Tokens.InputTokens) / float64(productiveOutput)
		excessTokens := view.session.Tokens.InputTokens - view.session.Tokens.OutputTokens
		if ratio < contextAvalancheMinRatio || excessTokens < contextAvalancheMinExcessTokens {
			continue
		}

		avoidableCost := proportionalCost(view.session.CostBreakdown.InputUSD, excessTokens, view.session.Tokens.InputTokens)
		severity := severityFromRatioAndTokens(ratio, excessTokens)
		insight, err := buildInsight(view, period, d.Category(), severity, []domain.InsightCount{
			mustTask17InsightCount("usage_entry_count", int64(len(view.usageEntryIDs))),
		}, []domain.InsightMetric{
			mustTask17InsightMetric("input_tokens", domain.InsightMetricUnitTokens, float64(view.session.Tokens.InputTokens)),
			mustTask17InsightMetric("output_tokens", domain.InsightMetricUnitTokens, float64(view.session.Tokens.OutputTokens)),
			mustTask17InsightMetric("input_to_output_ratio", domain.InsightMetricUnitRatio, ratio),
			mustTask17InsightMetric("estimated_waste_tokens", domain.InsightMetricUnitTokens, float64(excessTokens)),
			mustTask17InsightMetric("estimated_waste_usd", domain.InsightMetricUnitUSD, avoidableCost),
		}, detectionSignature(
			d.Category(),
			view.session.SessionID,
			metricSignatureInt(view.session.Tokens.InputTokens),
			metricSignatureInt(view.session.Tokens.OutputTokens),
			metricSignatureFloat(ratio),
			metricSignatureInt(excessTokens),
		))
		if err != nil {
			return nil, err
		}
		insights = append(insights, insight)
	}

	return insights, nil
}

type MissedPromptCachingDetector struct{}

func (MissedPromptCachingDetector) Category() domain.DetectorCategory {
	return domain.DetectorMissedPromptCaching
}

func (d MissedPromptCachingDetector) Detect(_ context.Context, period domain.MonthlyPeriod, sessions []domain.SessionSummary, usageEntries []domain.UsageEntry) ([]domain.Insight, error) {
	views := buildSessionInsightViews(sessions, usageEntries)
	insights := make([]domain.Insight, 0)
	for _, view := range views {
		if len(view.entries) < missedCacheMinEntries || view.session.Tokens.CacheReadTokens > 0 {
			continue
		}

		repeatedInputTokens := repeatedInputTokens(view.entries)
		if repeatedInputTokens < missedCacheMinRepeatedTokens {
			continue
		}

		repeatedInputUSD := repeatedInputCost(view.entries)
		repeatedRatio := float64(repeatedInputTokens) / float64(maxInt64(view.session.Tokens.InputTokens, 1))
		severity := severityFromRatioAndTokens(repeatedRatio, repeatedInputTokens)
		insight, err := buildInsight(view, period, d.Category(), severity, []domain.InsightCount{
			mustTask17InsightCount("usage_entry_count", int64(len(view.usageEntryIDs))),
		}, []domain.InsightMetric{
			mustTask17InsightMetric("repeated_input_tokens", domain.InsightMetricUnitTokens, float64(repeatedInputTokens)),
			mustTask17InsightMetric("repeated_input_ratio", domain.InsightMetricUnitRatio, repeatedRatio),
			mustTask17InsightMetric("cache_read_tokens", domain.InsightMetricUnitTokens, float64(view.session.Tokens.CacheReadTokens)),
			mustTask17InsightMetric("cache_write_tokens", domain.InsightMetricUnitTokens, float64(view.session.Tokens.CacheWriteTokens)),
			mustTask17InsightMetric("estimated_waste_tokens", domain.InsightMetricUnitTokens, float64(repeatedInputTokens)),
			mustTask17InsightMetric("estimated_waste_usd", domain.InsightMetricUnitUSD, repeatedInputUSD),
		}, detectionSignature(
			d.Category(),
			view.session.SessionID,
			metricSignatureInt(int64(len(view.usageEntryIDs))),
			metricSignatureInt(repeatedInputTokens),
			metricSignatureFloat(repeatedRatio),
			metricSignatureInt(view.session.Tokens.CacheReadTokens),
		))
		if err != nil {
			return nil, err
		}
		insights = append(insights, insight)
	}

	return insights, nil
}

type PlanningTaxDetector struct{}

func (PlanningTaxDetector) Category() domain.DetectorCategory {
	return domain.DetectorPlanningTax
}

func (d PlanningTaxDetector) Detect(_ context.Context, period domain.MonthlyPeriod, sessions []domain.SessionSummary, usageEntries []domain.UsageEntry) ([]domain.Insight, error) {
	views := buildSessionInsightViews(sessions, usageEntries)
	insights := make([]domain.Insight, 0)
	for _, view := range views {
		if view.reasoningTokens < planningTaxMinReasoningTokens || view.toolCalls > planningTaxMaxToolCalls {
			continue
		}

		actionTokens := maxInt64(view.session.Tokens.OutputTokens, 1)
		ratio := float64(view.reasoningTokens) / float64(actionTokens)
		if ratio < planningTaxMinReasoningRatio {
			continue
		}

		excessTokens := view.reasoningTokens - view.session.Tokens.OutputTokens
		avoidableCost := proportionalCost(view.session.CostBreakdown.InputUSD, view.reasoningTokens, maxInt64(view.session.Tokens.InputTokens, 1))
		severity := severityFromRatioAndTokens(ratio, excessTokens)
		insight, err := buildInsight(view, period, d.Category(), severity, []domain.InsightCount{
			mustTask17InsightCount("tool_call_count", view.toolCalls),
			mustTask17InsightCount("usage_entry_count", int64(len(view.usageEntryIDs))),
		}, []domain.InsightMetric{
			mustTask17InsightMetric("reasoning_tokens", domain.InsightMetricUnitTokens, float64(view.reasoningTokens)),
			mustTask17InsightMetric("output_tokens", domain.InsightMetricUnitTokens, float64(view.session.Tokens.OutputTokens)),
			mustTask17InsightMetric("reasoning_to_output_ratio", domain.InsightMetricUnitRatio, ratio),
			mustTask17InsightMetric("estimated_waste_tokens", domain.InsightMetricUnitTokens, float64(maxInt64(excessTokens, 0))),
			mustTask17InsightMetric("estimated_waste_usd", domain.InsightMetricUnitUSD, avoidableCost),
		}, detectionSignature(
			d.Category(),
			view.session.SessionID,
			metricSignatureInt(view.reasoningTokens),
			metricSignatureInt(view.session.Tokens.OutputTokens),
			metricSignatureInt(view.toolCalls),
			metricSignatureFloat(ratio),
		))
		if err != nil {
			return nil, err
		}
		insights = append(insights, insight)
	}

	return insights, nil
}

type sessionInsightView struct {
	session         domain.SessionSummary
	entries         []domain.UsageEntry
	usageEntryIDs   []string
	reasoningTokens int64
	toolCalls       int64
	projectHash     string
	agentHash       string
	pricingHash     string
}

func buildSessionInsightViews(sessions []domain.SessionSummary, usageEntries []domain.UsageEntry) []sessionInsightView {
	entriesBySession := make(map[string][]domain.UsageEntry, len(sessions))
	for _, entry := range usageEntries {
		if sessionID := strings.TrimSpace(entry.SessionID); sessionID != "" {
			entriesBySession[sessionID] = append(entriesBySession[sessionID], entry)
		}
	}

	views := make([]sessionInsightView, 0, len(sessions))
	for _, session := range sessions {
		entries := append([]domain.UsageEntry(nil), entriesBySession[session.SessionID]...)
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].OccurredAt.Equal(entries[j].OccurredAt) {
				return entries[i].EntryID < entries[j].EntryID
			}
			return entries[i].OccurredAt.Before(entries[j].OccurredAt)
		})

		view := sessionInsightView{
			session:         session,
			entries:         entries,
			usageEntryIDs:   make([]string, 0, len(entries)),
			reasoningTokens: 0,
			toolCalls:       0,
			projectHash:     privacySafeHash(session.ProjectName),
			agentHash:       privacySafeHash(session.AgentName),
			pricingHash:     pricingRefHash(session.PricingRef),
		}
		for _, entry := range entries {
			view.usageEntryIDs = append(view.usageEntryIDs, entry.EntryID)
			view.reasoningTokens += reasoningTokensFromMetadata(entry.Metadata)
			view.toolCalls += toolCallsFromMetadata(entry.Metadata)
		}
		views = append(views, view)
	}

	return views
}

func buildInsight(view sessionInsightView, period domain.MonthlyPeriod, category domain.DetectorCategory, severity domain.InsightSeverity, counts []domain.InsightCount, metrics []domain.InsightMetric, signature string) (domain.Insight, error) {
	hashes := make([]domain.InsightHash, 0, 4)
	if view.projectHash != "" {
		hashes = append(hashes, mustTask17InsightHash("project_hash", view.projectHash))
	}
	if view.agentHash != "" {
		hashes = append(hashes, mustTask17InsightHash("agent_hash", view.agentHash))
	}
	if view.pricingHash != "" {
		hashes = append(hashes, mustTask17InsightHash("pricing_ref_hash", view.pricingHash))
	}
	hashes = append(hashes, mustTask17InsightHash("metrics_hash", privacySafeHash(signature)))

	payload, err := domain.NewInsightPayload([]string{view.session.SessionID}, view.usageEntryIDs, hashes, counts, metrics)
	if err != nil {
		return domain.Insight{}, err
	}

	insightID := fmt.Sprintf("insight:%s:%s:%s", category, view.session.SessionID, signatureDigest(signature))
	return domain.NewInsight(domain.Insight{
		InsightID:  insightID,
		Category:   category,
		Severity:   severity,
		DetectedAt: detectionTime(view.session, period),
		Period:     period,
		Payload:    payload,
	})
}

func detectionTime(session domain.SessionSummary, period domain.MonthlyPeriod) time.Time {
	if period.Contains(session.EndedAt) {
		return session.EndedAt
	}
	if period.Contains(session.StartedAt) {
		return session.StartedAt
	}
	return period.StartAt
}

func repeatedInputTokens(entries []domain.UsageEntry) int64 {
	if len(entries) < 2 {
		return 0
	}
	var total int64
	for _, entry := range entries[1:] {
		total += entry.Tokens.InputTokens
	}
	return total
}

func repeatedInputCost(entries []domain.UsageEntry) float64 {
	if len(entries) < 2 {
		return 0
	}
	var total float64
	for _, entry := range entries[1:] {
		total += entry.CostBreakdown.InputUSD
	}
	return total
}

func reasoningTokensFromMetadata(metadata map[string]string) int64 {
	return sumMetadataInts(metadata, "opencode_reasoning_tokens", "gemini_thought_tokens")
}

func toolCallsFromMetadata(metadata map[string]string) int64 {
	return sumMetadataInts(metadata, "observed_tool_call_count", "mcp_tool_call_count")
}

func sumMetadataInts(metadata map[string]string, keys ...string) int64 {
	var total int64
	for _, key := range keys {
		value := strings.TrimSpace(metadata[key])
		if value == "" {
			continue
		}
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed <= 0 {
			continue
		}
		total += parsed
	}
	return total
}

func severityFromRatioAndTokens(ratio float64, tokens int64) domain.InsightSeverity {
	switch {
	case ratio >= 8 || tokens >= 8000:
		return domain.InsightSeverityHigh
	case ratio >= 5 || tokens >= 4000:
		return domain.InsightSeverityMedium
	default:
		return domain.InsightSeverityLow
	}
}

func proportionalCost(totalCost float64, partialTokens, totalTokens int64) float64 {
	if totalCost <= 0 || partialTokens <= 0 || totalTokens <= 0 {
		return 0
	}
	if partialTokens >= totalTokens {
		return totalCost
	}
	return totalCost * (float64(partialTokens) / float64(totalTokens))
}

func privacySafeHash(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func pricingRefHash(ref *domain.ModelPricingRef) string {
	if ref == nil {
		return ""
	}
	return privacySafeHash(ref.Provider.String() + ":" + ref.PricingLookupKey)
}

func detectionSignature(category domain.DetectorCategory, sessionID string, metrics ...string) string {
	parts := []string{string(category), strings.TrimSpace(sessionID)}
	parts = append(parts, metrics...)
	return strings.Join(parts, "|")
}

func signatureDigest(signature string) string {
	sum := sha256.Sum256([]byte(signature))
	return hex.EncodeToString(sum[:12])
}

func metricSignatureInt(value int64) string {
	return strconv.FormatInt(value, 10)
}

func metricSignatureFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 6, 64)
}

func mustTask17InsightHash(kind, value string) domain.InsightHash {
	hash, err := domain.NewInsightHash(kind, value)
	if err != nil {
		panic(err)
	}
	return hash
}

func mustTask17InsightCount(key string, value int64) domain.InsightCount {
	count, err := domain.NewInsightCount(key, value)
	if err != nil {
		panic(err)
	}
	return count
}

func mustTask17InsightMetric(key string, unit domain.InsightMetricUnit, value float64) domain.InsightMetric {
	metric, err := domain.NewInsightMetric(key, unit, value)
	if err != nil {
		panic(err)
	}
	return metric
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
