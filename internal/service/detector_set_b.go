package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

const (
	repeatedFileReadMinimumCount   = 4
	retryAmplificationMinimumFails = 3
	zombieLoopMinimumRunLength     = 5
)

type repeatedFileReadDetector struct{}

type retryAmplificationDetector struct{}

type zombieLoopDetector struct{}

func NewDetectorSetB() []ports.InsightDetector {
	return []ports.InsightDetector{
		repeatedFileReadDetector{},
		retryAmplificationDetector{},
		zombieLoopDetector{},
	}
}

func (repeatedFileReadDetector) Category() domain.DetectorCategory {
	return domain.DetectorRepeatedFileReads
}

func (repeatedFileReadDetector) Detect(_ context.Context, period domain.MonthlyPeriod, sessions []domain.SessionSummary, usageEntries []domain.UsageEntry) ([]domain.Insight, error) {
	groupedEntries := groupedUsageEntriesBySession(usageEntries)
	insights := make([]domain.Insight, 0)

	for _, session := range sessions {
		entries := groupedEntries[session.SessionID]
		if len(entries) == 0 {
			continue
		}

		readsByTarget := make(map[string][]domain.UsageEntry)
		for _, entry := range entries {
			if !isRepeatedFileReadCandidate(entry) {
				continue
			}
			targetHash := firstMetadataValue(entry.Metadata, "file_target_hash", "target_hash", "path_hash", "resource_hash")
			if targetHash == "" {
				continue
			}
			readsByTarget[targetHash] = append(readsByTarget[targetHash], entry)
		}

		if len(readsByTarget) == 0 {
			continue
		}

		wasteEntries := make([]domain.UsageEntry, 0)
		repeatedTargets := make([]string, 0)
		totalReadEvents := 0
		for targetHash, targetEntries := range readsByTarget {
			totalReadEvents += len(targetEntries)
			if len(targetEntries) < repeatedFileReadMinimumCount {
				continue
			}
			repeatedTargets = append(repeatedTargets, targetHash)
			wasteEntries = append(wasteEntries, targetEntries[2:]...)
		}

		if len(wasteEntries) < 2 || len(repeatedTargets) == 0 {
			continue
		}

		sort.Strings(repeatedTargets)
		payload, err := buildInsightPayload(session, wasteEntries, repeatedTargets,
			map[string]int64{
				"total_read_events":      int64(totalReadEvents),
				"unique_file_targets":    int64(len(readsByTarget)),
				"repeated_target_count":  int64(len(repeatedTargets)),
				"suspected_waste_events": int64(len(wasteEntries)),
			},
			map[string]float64{
				"suspected_waste_usd":     sumWasteUSD(wasteEntries),
				"suspected_waste_tokens":  float64(sumWasteTokens(wasteEntries)),
				"suspected_waste_seconds": approximateWasteSeconds(session, entries, wasteEntries),
				"repeated_read_ratio":     ratio(float64(len(wasteEntries)), float64(totalReadEvents)),
			},
		)
		if err != nil {
			return nil, err
		}

		insight, err := newDetectorInsight(period, domain.DetectorRepeatedFileReads, repeatedFileReadSeverity(wasteEntries), session, payload, repeatedTargets...)
		if err != nil {
			return nil, err
		}
		insights = append(insights, insight)
	}

	return insights, nil
}

func (retryAmplificationDetector) Category() domain.DetectorCategory {
	return domain.DetectorRetryAmplification
}

func (retryAmplificationDetector) Detect(_ context.Context, period domain.MonthlyPeriod, sessions []domain.SessionSummary, usageEntries []domain.UsageEntry) ([]domain.Insight, error) {
	groupedEntries := groupedUsageEntriesBySession(usageEntries)
	insights := make([]domain.Insight, 0)

	for _, session := range sessions {
		entries := groupedEntries[session.SessionID]
		if len(entries) == 0 {
			continue
		}

		operationGroups := make(map[string][]domain.UsageEntry)
		for _, entry := range entries {
			signature := retrySignature(entry)
			if signature == "" {
				continue
			}
			operationGroups[signature] = append(operationGroups[signature], entry)
		}

		var dominantSignature string
		var dominantError string
		var wasteEntries []domain.UsageEntry
		var failedAttempts int
		for signature, group := range operationGroups {
			failureGroups := make(map[string][]domain.UsageEntry)
			for _, entry := range group {
				if classifyExecutionOutcome(entry.Metadata) != outcomeFailure {
					continue
				}
				errorHash := firstMetadataValue(entry.Metadata, "error_hash", "error_code", "status_code", "failure_hash")
				if errorHash == "" {
					errorHash = "failure"
				}
				failureGroups[errorHash] = append(failureGroups[errorHash], entry)
			}

			for errorHash, failures := range failureGroups {
				if len(failures) < retryAmplificationMinimumFails {
					continue
				}
				candidateWaste := failures[1:]
				if len(candidateWaste) <= len(wasteEntries) {
					continue
				}
				dominantSignature = signature
				dominantError = errorHash
				failedAttempts = len(failures)
				wasteEntries = candidateWaste
			}
		}

		if len(wasteEntries) < 2 {
			continue
		}

		payload, err := buildInsightPayload(session, wasteEntries, []string{dominantSignature, dominantError},
			map[string]int64{
				"failed_attempts":        int64(failedAttempts),
				"retry_attempts":         int64(failedAttempts - 1),
				"suspected_waste_events": int64(len(wasteEntries)),
			},
			map[string]float64{
				"suspected_waste_usd":     sumWasteUSD(wasteEntries),
				"suspected_waste_tokens":  float64(sumWasteTokens(wasteEntries)),
				"suspected_waste_seconds": approximateWasteSeconds(session, entries, wasteEntries),
				"retry_failure_ratio":     ratio(float64(failedAttempts), float64(len(operationGroups[dominantSignature]))),
			},
		)
		if err != nil {
			return nil, err
		}

		insight, err := newDetectorInsight(period, domain.DetectorRetryAmplification, retryAmplificationSeverity(wasteEntries), session, payload, dominantSignature, dominantError)
		if err != nil {
			return nil, err
		}
		insights = append(insights, insight)
	}

	return insights, nil
}

func (zombieLoopDetector) Category() domain.DetectorCategory {
	return domain.DetectorZombieLoops
}

func (zombieLoopDetector) Detect(_ context.Context, period domain.MonthlyPeriod, sessions []domain.SessionSummary, usageEntries []domain.UsageEntry) ([]domain.Insight, error) {
	groupedEntries := groupedUsageEntriesBySession(usageEntries)
	insights := make([]domain.Insight, 0)

	for _, session := range sessions {
		entries := groupedEntries[session.SessionID]
		if len(entries) < zombieLoopMinimumRunLength {
			continue
		}

		bestRun := detectZombieLoopRun(entries)
		if len(bestRun.entries) == 0 {
			continue
		}

		payload, err := buildInsightPayload(session, bestRun.entries, []string{bestRun.signature},
			map[string]int64{
				"loop_length":               int64(bestRun.length),
				"stalled_iterations":        int64(len(bestRun.entries)),
				"distinct_progress_markers": int64(bestRun.progressMarkers),
			},
			map[string]float64{
				"suspected_waste_usd":     sumWasteUSD(bestRun.entries),
				"suspected_waste_tokens":  float64(sumWasteTokens(bestRun.entries)),
				"suspected_waste_seconds": approximateWasteSeconds(session, entries, bestRun.entries),
				"stalled_loop_ratio":      ratio(float64(len(bestRun.entries)), float64(bestRun.length)),
			},
		)
		if err != nil {
			return nil, err
		}

		insight, err := newDetectorInsight(period, domain.DetectorZombieLoops, zombieLoopSeverity(bestRun.entries), session, payload, bestRun.signature)
		if err != nil {
			return nil, err
		}
		insights = append(insights, insight)
	}

	return insights, nil
}

type zombieRun struct {
	signature       string
	length          int
	progressMarkers int
	entries         []domain.UsageEntry
}

type executionOutcome int

const (
	outcomeUnknown executionOutcome = iota
	outcomeSuccess
	outcomeFailure
)

func groupedUsageEntriesBySession(entries []domain.UsageEntry) map[string][]domain.UsageEntry {
	grouped := make(map[string][]domain.UsageEntry)
	for _, entry := range entries {
		grouped[entry.SessionID] = append(grouped[entry.SessionID], entry)
	}
	for sessionID := range grouped {
		sort.Slice(grouped[sessionID], func(i, j int) bool {
			left := grouped[sessionID][i]
			right := grouped[sessionID][j]
			if left.OccurredAt.Equal(right.OccurredAt) {
				return left.EntryID < right.EntryID
			}
			return left.OccurredAt.Before(right.OccurredAt)
		})
	}
	return grouped
}

func isRepeatedFileReadCandidate(entry domain.UsageEntry) bool {
	metadata := entry.Metadata
	if metadata == nil {
		return false
	}
	if targetKind := firstMetadataValue(metadata, "target_kind", "resource_kind"); targetKind != "" && !strings.Contains(strings.ToLower(targetKind), "file") {
		return false
	}

	operation := strings.ToLower(firstMetadataValue(metadata, "tool_name", "operation", "action", "tool_kind"))
	if operation == "" {
		return false
	}
	if strings.Contains(operation, "write") || strings.Contains(operation, "edit") || strings.Contains(operation, "patch") {
		return false
	}

	return strings.Contains(operation, "read") || strings.Contains(operation, "open") || strings.Contains(operation, "view")
}

func retrySignature(entry domain.UsageEntry) string {
	if signature := firstMetadataValue(entry.Metadata, "retry_key", "operation_hash", "tool_call_signature", "action_signature"); signature != "" {
		return signature
	}

	toolName := firstMetadataValue(entry.Metadata, "tool_name", "operation", "action")
	targetHash := firstMetadataValue(entry.Metadata, "target_hash", "file_target_hash", "path_hash", "resource_hash")
	if toolName == "" && targetHash == "" {
		return ""
	}

	return strings.TrimSpace(toolName + ":" + targetHash)
}

func detectZombieLoopRun(entries []domain.UsageEntry) zombieRun {
	best := zombieRun{}
	for start := 0; start < len(entries); start++ {
		signature := zombieLoopSignature(entries[start])
		if signature == "" {
			continue
		}

		progressSet := make(map[string]struct{})
		hasSuccess := false
		end := start
		for end < len(entries) && zombieLoopSignature(entries[end]) == signature {
			if progress := firstMetadataValue(entries[end].Metadata, "progress_marker", "progress_hash", "checkpoint_hash", "step_hash", "state_hash"); progress != "" {
				progressSet[progress] = struct{}{}
			}
			if classifyExecutionOutcome(entries[end].Metadata) == outcomeSuccess {
				hasSuccess = true
			}
			end++
		}

		runLength := end - start
		if runLength < zombieLoopMinimumRunLength || hasSuccess || len(progressSet) > 1 {
			start = end - 1
			continue
		}

		wasteEntries := append([]domain.UsageEntry(nil), entries[start+3:end]...)
		if len(wasteEntries) < 2 {
			start = end - 1
			continue
		}

		candidate := zombieRun{
			signature:       signature,
			length:          runLength,
			progressMarkers: len(progressSet),
			entries:         wasteEntries,
		}
		if len(candidate.entries) > len(best.entries) {
			best = candidate
		}
		start = end - 1
	}

	return best
}

func zombieLoopSignature(entry domain.UsageEntry) string {
	if signature := firstMetadataValue(entry.Metadata, "loop_signature", "action_signature", "operation_hash", "tool_call_signature", "retry_key"); signature != "" {
		return signature
	}
	return retrySignature(entry)
}

func classifyExecutionOutcome(metadata map[string]string) executionOutcome {
	status := strings.ToLower(firstMetadataValue(metadata, "status", "result", "outcome", "tool_result"))
	switch {
	case status == "":
		return outcomeUnknown
	case containsAny(status, "success", "ok", "completed", "resolved"):
		return outcomeSuccess
	case containsAny(status, "fail", "error", "timeout", "denied", "unreachable", "invalid"):
		return outcomeFailure
	default:
		return outcomeUnknown
	}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func firstMetadataValue(metadata map[string]string, keys ...string) string {
	for _, key := range keys {
		if metadata == nil {
			return ""
		}
		if value := strings.TrimSpace(metadata[key]); value != "" {
			return value
		}
	}
	return ""
}

func buildInsightPayload(session domain.SessionSummary, wasteEntries []domain.UsageEntry, hashValues []string, counts map[string]int64, metrics map[string]float64) (domain.InsightPayload, error) {
	hashes := make([]domain.InsightHash, 0, len(hashValues))
	for _, value := range hashValues {
		if strings.TrimSpace(value) == "" {
			continue
		}
		hash, err := domain.NewInsightHash("privacy_safe_ref", value)
		if err != nil {
			return domain.InsightPayload{}, err
		}
		hashes = append(hashes, hash)
	}

	countKeys := sortedInt64Keys(counts)
	countValues := make([]domain.InsightCount, 0, len(countKeys))
	for _, key := range countKeys {
		count, err := domain.NewInsightCount(key, counts[key])
		if err != nil {
			return domain.InsightPayload{}, err
		}
		countValues = append(countValues, count)
	}

	metricKeys := sortedFloat64Keys(metrics)
	metricValues := make([]domain.InsightMetric, 0, len(metricKeys))
	for _, key := range metricKeys {
		unit := domain.InsightMetricUnitCount
		switch {
		case strings.HasSuffix(key, "_usd"):
			unit = domain.InsightMetricUnitUSD
		case strings.HasSuffix(key, "_tokens"):
			unit = domain.InsightMetricUnitTokens
		case strings.HasSuffix(key, "_seconds"):
			unit = domain.InsightMetricUnitSeconds
		case strings.HasSuffix(key, "_ratio"):
			unit = domain.InsightMetricUnitRatio
		}
		metric, err := domain.NewInsightMetric(key, unit, metrics[key])
		if err != nil {
			return domain.InsightPayload{}, err
		}
		metricValues = append(metricValues, metric)
	}

	usageEntryIDs := make([]string, 0, len(wasteEntries))
	for _, entry := range wasteEntries {
		usageEntryIDs = append(usageEntryIDs, entry.EntryID)
	}

	return domain.NewInsightPayload([]string{session.SessionID}, usageEntryIDs, hashes, countValues, metricValues)
}

func newDetectorInsight(period domain.MonthlyPeriod, category domain.DetectorCategory, severity domain.InsightSeverity, session domain.SessionSummary, payload domain.InsightPayload, hashValues ...string) (domain.Insight, error) {
	basis := []string{string(category), session.SessionID, period.StartAt.Format("2006-01"), session.StartedAt.Format("2006-01-02T15:04:05Z07:00")}
	basis = append(basis, hashValues...)
	sort.Strings(basis[4:])
	sum := sha256.Sum256([]byte(strings.Join(basis, "|")))
	return domain.NewInsight(domain.Insight{
		InsightID:  fmt.Sprintf("%s-%s", category, hex.EncodeToString(sum[:8])),
		Category:   category,
		Severity:   severity,
		DetectedAt: session.EndedAt,
		Period:     period,
		Payload:    payload,
	})
}

func repeatedFileReadSeverity(wasteEntries []domain.UsageEntry) domain.InsightSeverity {
	switch {
	case len(wasteEntries) >= 5:
		return domain.InsightSeverityHigh
	case len(wasteEntries) >= 3:
		return domain.InsightSeverityMedium
	default:
		return domain.InsightSeverityLow
	}
}

func retryAmplificationSeverity(wasteEntries []domain.UsageEntry) domain.InsightSeverity {
	switch {
	case len(wasteEntries) >= 4:
		return domain.InsightSeverityHigh
	case len(wasteEntries) >= 3:
		return domain.InsightSeverityMedium
	default:
		return domain.InsightSeverityLow
	}
}

func zombieLoopSeverity(wasteEntries []domain.UsageEntry) domain.InsightSeverity {
	switch {
	case len(wasteEntries) >= 5:
		return domain.InsightSeverityHigh
	case len(wasteEntries) >= 3:
		return domain.InsightSeverityMedium
	default:
		return domain.InsightSeverityLow
	}
}

func sumWasteUSD(entries []domain.UsageEntry) float64 {
	var total float64
	for _, entry := range entries {
		total += entry.CostBreakdown.TotalUSD
	}
	return total
}

func sumWasteTokens(entries []domain.UsageEntry) int64 {
	var total int64
	for _, entry := range entries {
		total += entry.Tokens.TotalTokens
	}
	return total
}

func approximateWasteSeconds(session domain.SessionSummary, allEntries []domain.UsageEntry, wasteEntries []domain.UsageEntry) float64 {
	if len(allEntries) == 0 || len(wasteEntries) == 0 {
		return 0
	}
	seconds := session.Duration().Seconds()
	if seconds <= 0 {
		return 0
	}
	return seconds * (float64(len(wasteEntries)) / float64(len(allEntries)))
}

func ratio(numerator, denominator float64) float64 {
	if denominator <= 0 {
		return 0
	}
	value, _ := strconv.ParseFloat(fmt.Sprintf("%.4f", numerator/denominator), 64)
	return value
}

func sortedInt64Keys(values map[string]int64) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedFloat64Keys(values map[string]float64) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
