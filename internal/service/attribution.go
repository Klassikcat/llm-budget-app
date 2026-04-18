package service

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

type AttributionWarningCode string

const (
	AttributionWarningBillingModeMissing  AttributionWarningCode = "billing_mode_missing"
	AttributionWarningBillingModeConflict AttributionWarningCode = "billing_mode_conflict"
	AttributionWarningProjectConflict     AttributionWarningCode = "project_conflict"
	AttributionWarningAgentConflict       AttributionWarningCode = "agent_conflict"
	AttributionWarningProviderConflict    AttributionWarningCode = "provider_conflict"
	AttributionWarningModelConflict       AttributionWarningCode = "model_conflict"
)

type AttributionWarning struct {
	Code      AttributionWarningCode
	SessionID string
	Field     string
	Detail    string
}

func (w AttributionWarning) String() string {
	base := fmt.Sprintf("attribution warning [%s] session=%s", w.Code, strings.TrimSpace(w.SessionID))
	if field := strings.TrimSpace(w.Field); field != "" {
		base += " field=" + field
	}
	if detail := strings.TrimSpace(w.Detail); detail != "" {
		base += ": " + detail
	}
	return base
}

func canonicalSessionBillingMode(mode domain.BillingMode) domain.BillingMode {
	switch mode {
	case domain.BillingModeSubscription:
		return domain.BillingModeSubscription
	case domain.BillingModeBYOK, domain.BillingModeDirectAPI, domain.BillingModeOpenRouter:
		return domain.BillingModeBYOK
	default:
		return domain.BillingModeUnknown
	}
}

func resolveSessionBillingMode(sessionID string, counts map[domain.BillingMode]int) (domain.BillingMode, []AttributionWarning) {
	if len(counts) == 0 {
		return domain.BillingModeUnknown, []AttributionWarning{{
			Code:      AttributionWarningBillingModeMissing,
			SessionID: sessionID,
			Field:     "billing_mode",
			Detail:    "session did not include any authoritative subscription/BYOK hints; billing mode remains unknown",
		}}
	}

	if len(counts) > 1 {
		return domain.BillingModeUnknown, []AttributionWarning{{
			Code:      AttributionWarningBillingModeConflict,
			SessionID: sessionID,
			Field:     "billing_mode",
			Detail:    fmt.Sprintf("session included conflicting billing hints: %s", joinBillingModes(counts)),
		}}
	}

	for mode := range counts {
		return mode, nil
	}

	return domain.BillingModeUnknown, nil
}

func resolveStringAttribution(sessionID, field, lastValue string, values map[string]int) (string, []AttributionWarning) {
	trimmed := strings.TrimSpace(lastValue)
	if len(values) <= 1 {
		return trimmed, nil
	}

	code := AttributionWarningProjectConflict
	switch field {
	case "agent_name":
		code = AttributionWarningAgentConflict
	case "provider":
		code = AttributionWarningProviderConflict
	case "model":
		code = AttributionWarningModelConflict
	}

	return trimmed, []AttributionWarning{{
		Code:      code,
		SessionID: sessionID,
		Field:     field,
		Detail:    fmt.Sprintf("session included multiple %s values; using the latest observed value %q", field, trimmed),
	}}
}

func usageEntryMetadata(event ports.SessionEvent) map[string]string {
	metadata := make(map[string]string, len(event.PrivacySafeTags)+4)
	for key, value := range event.PrivacySafeTags {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		metadata[trimmedKey] = trimmedValue
	}

	if event.ObservedToolCall > 0 {
		count := strconv.FormatInt(event.ObservedToolCall, 10)
		metadata["mcp_tool_call_count"] = count
		metadata["observed_tool_call_count"] = count
	}

	if len(metadata) == 0 {
		return nil
	}

	return metadata
}

func joinBillingModes(counts map[domain.BillingMode]int) string {
	values := make([]string, 0, len(counts))
	for mode := range counts {
		values = append(values, string(mode))
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}
