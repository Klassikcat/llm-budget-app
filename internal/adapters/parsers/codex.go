package parsers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

type CodexParser struct{}

type codexEnvelope struct {
	Timestamp string         `json:"timestamp"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload"`
}

type codexState struct {
	SessionID       string
	ProjectName     string
	PricingRef      *domain.ModelPricingRef
	BillingModeHint domain.BillingMode
}

func NewCodexParser() *CodexParser {
	return &CodexParser{}
}

func (p *CodexParser) ParserName() string {
	return "codex"
}

func (p *CodexParser) Parse(ctx context.Context, input ports.ParseInput) (ports.ParseResult, error) {
	result, warnings, err := p.ParseDetailed(ctx, input)
	if err != nil {
		return ports.ParseResult{}, err
	}
	result.Warnings = codexWarningsToStrings(warnings)
	return result, nil
}

func (p *CodexParser) ParseDetailed(ctx context.Context, input ports.ParseInput) (ports.ParseResult, []CodexWarning, error) {
	_ = ctx

	state := codexState{
		SessionID:       deriveCodexSessionID(input.Path),
		ProjectName:     "",
		PricingRef:      nil,
		BillingModeHint: domain.BillingModeUnknown,
	}

	lines := bytes.Split(input.Content, []byte("\n"))
	result := ports.ParseResult{NextOffset: input.StartOffset + int64(len(input.Content))}
	var warnings []CodexWarning

	for index, rawLine := range lines {
		lineNumber := index + 1
		trimmed := bytes.TrimSpace(rawLine)
		if len(trimmed) == 0 {
			continue
		}

		var envelope codexEnvelope
		if err := json.Unmarshal(trimmed, &envelope); err != nil {
			warnings = append(warnings, CodexWarning{
				Code:   CodexWarningMalformedJSON,
				Path:   input.Path,
				Line:   lineNumber,
				Detail: err.Error(),
			})
			continue
		}

		state = applyCodexMetadata(state, envelope.Payload)

		switch envelope.Type {
		case "session_start", "turn_context", "event_msg":
			continue
		case "response_item":
			event, eventWarnings := buildCodexEvent(input, lineNumber, envelope, state)
			warnings = append(warnings, eventWarnings...)
			if event != nil {
				result.Events = append(result.Events, *event)
			}
		default:
			warnings = append(warnings, CodexWarning{
				Code:    CodexWarningUnsupportedVariant,
				Path:    input.Path,
				Line:    lineNumber,
				Variant: envelope.Type,
				Detail:  "record variant is not yet normalized",
			})
		}
	}

	result.Warnings = codexWarningsToStrings(warnings)
	return result, warnings, nil
}

func buildCodexEvent(input ports.ParseInput, lineNumber int, envelope codexEnvelope, state codexState) (*ports.SessionEvent, []CodexWarning) {
	var warnings []CodexWarning

	occurredAt, timestampWarning := parseCodexTimestamp(input, lineNumber, envelope.Timestamp)
	if timestampWarning != nil {
		warnings = append(warnings, *timestampWarning)
		return nil, warnings
	}

	usageShape, usageMap, ok := extractCodexUsage(envelope.Payload)
	if !ok {
		return nil, warnings
	}

	tokens, err := newCodexTokenUsage(usageMap)
	if err != nil {
		warnings = append(warnings, CodexWarning{
			Code:    CodexWarningInvalidUsage,
			Path:    input.Path,
			Line:    lineNumber,
			Variant: usageShape,
			Detail:  err.Error(),
		})
		return nil, warnings
	}

	costs, err := newCodexCostBreakdown(envelope.Payload)
	if err != nil {
		warnings = append(warnings, CodexWarning{
			Code:    CodexWarningInvalidCost,
			Path:    input.Path,
			Line:    lineNumber,
			Variant: usageShape,
			Detail:  err.Error(),
		})
		return nil, warnings
	}

	pricingRef := state.PricingRef
	if candidate := deriveCodexPricingRef(envelope.Payload); candidate != nil {
		pricingRef = candidate
	}

	billingModeHint := state.BillingModeHint
	if candidate := codexBillingModeHint(envelope.Payload); candidate != domain.BillingModeUnknown {
		billingModeHint = candidate
	}

	sessionID := state.SessionID
	if candidate := firstCodexString(envelope.Payload, "session_id", "sessionId"); candidate != "" {
		sessionID = candidate
	}

	projectName := state.ProjectName
	if candidate := codexProjectName(envelope.Payload); candidate != "" {
		projectName = candidate
	}

	externalID := firstCodexString(envelope.Payload, "id")
	if externalID == "" {
		externalID = nestedCodexString(envelope.Payload, []string{"item", "id"}, []string{"message", "id"}, []string{"response", "id"})
	}
	entryID := fmt.Sprintf("%s:%d", sessionID, lineNumber)
	if externalID != "" {
		entryID = externalID
	}

	return &ports.SessionEvent{
		EntryID:         entryID,
		ExternalID:      externalID,
		SessionID:       sessionID,
		OccurredAt:      occurredAt,
		Source:          domain.UsageSourceCLISession,
		Provider:        domain.ProviderCodex,
		BillingModeHint: billingModeHint,
		ProjectName:     projectName,
		AgentName:       "codex",
		PricingRef:      pricingRef,
		Tokens:          tokens,
		CostBreakdown:   costs,
		PrivacySafeTags: map[string]string{
			"codex_record_type": envelope.Type,
			"codex_usage_shape": usageShape,
		},
		ObservedToolCall: countCodexToolCalls(envelope.Payload),
	}, warnings
}

func applyCodexMetadata(state codexState, payload map[string]any) codexState {
	if payload == nil {
		return state
	}

	if candidate := firstCodexString(payload, "session_id", "sessionId"); candidate != "" {
		state.SessionID = candidate
	}
	if candidate := codexProjectName(payload); candidate != "" {
		state.ProjectName = candidate
	}
	if candidate := deriveCodexPricingRef(payload); candidate != nil {
		state.PricingRef = candidate
	}
	if candidate := codexBillingModeHint(payload); candidate != domain.BillingModeUnknown {
		state.BillingModeHint = candidate
	}

	return state
}

func parseCodexTimestamp(input ports.ParseInput, lineNumber int, raw string) (time.Time, *CodexWarning) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, &CodexWarning{
			Code:   CodexWarningMissingTimestamp,
			Path:   input.Path,
			Line:   lineNumber,
			Detail: "usage-bearing record is missing timestamp",
		}
	}

	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return time.Time{}, &CodexWarning{
			Code:   CodexWarningInvalidTimestamp,
			Path:   input.Path,
			Line:   lineNumber,
			Detail: err.Error(),
		}
	}

	return parsed.UTC(), nil
}

func extractCodexUsage(payload map[string]any) (string, map[string]any, bool) {
	for _, candidate := range []struct {
		shape string
		paths [][]string
	}{
		{shape: "payload.usage", paths: [][]string{{"usage"}}},
		{shape: "payload.item.usage", paths: [][]string{{"item", "usage"}}},
		{shape: "payload.message.usage", paths: [][]string{{"message", "usage"}}},
		{shape: "payload.response.usage", paths: [][]string{{"response", "usage"}}},
	} {
		for _, path := range candidate.paths {
			if usageMap, ok := nestedCodexMap(payload, path); ok {
				return candidate.shape, usageMap, true
			}
		}
	}

	return "", nil, false
}

func deriveCodexPricingRef(payload map[string]any) *domain.ModelPricingRef {
	modelID := firstCodexString(payload, "model")
	if modelID == "" {
		modelID = nestedCodexString(payload, []string{"item", "model"}, []string{"message", "model"}, []string{"response", "model"})
	}
	if modelID == "" {
		return nil
	}

	pricingRef, err := domain.NewModelPricingRef(domain.ProviderCodex, modelID, modelID)
	if err != nil {
		return nil
	}

	return &pricingRef
}

func codexProjectName(payload map[string]any) string {
	if project := firstCodexString(payload, "project_name", "projectName"); project != "" {
		return project
	}

	cwd := firstCodexString(payload, "cwd")
	if cwd == "" {
		cwd = nestedCodexString(payload, []string{"context", "cwd"})
	}
	if cwd == "" {
		return ""
	}

	base := filepath.Base(strings.TrimSpace(cwd))
	if base == "." || base == "/" {
		return ""
	}

	return base
}

func codexBillingModeHint(payload map[string]any) domain.BillingMode {
	for _, raw := range []string{
		firstCodexString(payload, "billing_mode", "billingMode", "auth_mode", "authMode", "auth_source", "authSource"),
		nestedCodexString(payload, []string{"context", "billing_mode"}, []string{"context", "auth_mode"}),
	} {
		switch normalizeCodexLabel(raw) {
		case "subscription", "chatgpt", "plus", "pro", "plan", "chatgpt_plan":
			return domain.BillingModeSubscription
		case "byok", "api_key", "apikey", "api-key":
			return domain.BillingModeBYOK
		case "direct_api", "directapi":
			return domain.BillingModeDirectAPI
		case "openrouter":
			return domain.BillingModeOpenRouter
		}
	}

	return domain.BillingModeUnknown
}

func normalizeCodexLabel(raw string) string {
	replacer := strings.NewReplacer("-", "_", " ", "_", ".", "_")
	return replacer.Replace(strings.ToLower(strings.TrimSpace(raw)))
}

func newCodexTokenUsage(usage map[string]any) (domain.TokenUsage, error) {
	inputTokens := firstCodexInt(usage, "input_tokens", "prompt_tokens")
	outputTokens := firstCodexInt(usage, "output_tokens", "completion_tokens")
	cacheReadTokens := firstCodexInt(usage, "cache_read_input_tokens", "cached_tokens")
	cacheWriteTokens := firstCodexInt(usage, "cache_creation_input_tokens", "cache_write_tokens")
	return domain.NewTokenUsage(inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens)
}

func newCodexCostBreakdown(payload map[string]any) (domain.CostBreakdown, error) {
	inputUSD := firstCodexFloat(payload, "input_usd")
	outputUSD := firstCodexFloat(payload, "output_usd")
	cacheReadUSD := firstCodexFloat(payload, "cache_read_usd")
	cacheWriteUSD := firstCodexFloat(payload, "cache_creation_usd", "cache_write_usd")
	toolUSD := firstCodexFloat(payload, "tool_usd")
	flatUSD := 0.0

	totalUSD := firstCodexFloat(payload, "total_usd", "cost_usd", "costUSD")
	if totalUSD > 0 && inputUSD == 0 && outputUSD == 0 && cacheReadUSD == 0 && cacheWriteUSD == 0 && toolUSD == 0 {
		flatUSD = totalUSD
	}

	return domain.NewCostBreakdown(inputUSD, outputUSD, cacheReadUSD, cacheWriteUSD, toolUSD, flatUSD)
}

func deriveCodexSessionID(path string) string {
	base := filepath.Base(strings.TrimSpace(path))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "codex-session"
	}
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func countCodexToolCalls(payload map[string]any) int64 {
	var total int64
	for _, path := range [][]string{{"content"}, {"item", "content"}, {"message", "content"}, {"response", "content"}} {
		items, ok := nestedCodexSlice(payload, path)
		if !ok {
			continue
		}
		for _, item := range items {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			switch normalizeCodexLabel(firstCodexString(entry, "type")) {
			case "tool_call", "function_call", "custom_tool_call", "mcp_tool_call":
				total++
			}
		}
	}
	return total
}

func firstCodexString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := m[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			trimmed := strings.TrimSpace(typed)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func firstCodexInt(m map[string]any, keys ...string) int64 {
	for _, key := range keys {
		value, ok := m[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return int64(typed)
		case int64:
			return typed
		case int:
			return int64(typed)
		case json.Number:
			parsed, err := typed.Int64()
			if err == nil {
				return parsed
			}
		case string:
			var parsed int64
			if _, err := fmt.Sscan(strings.TrimSpace(typed), &parsed); err == nil {
				return parsed
			}
		}
	}
	return 0
}

func firstCodexFloat(m map[string]any, keys ...string) float64 {
	for _, key := range keys {
		value, ok := m[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return typed
		case float32:
			return float64(typed)
		case int64:
			return float64(typed)
		case int:
			return float64(typed)
		case json.Number:
			parsed, err := typed.Float64()
			if err == nil {
				return parsed
			}
		case string:
			var parsed float64
			if _, err := fmt.Sscan(strings.TrimSpace(typed), &parsed); err == nil {
				return parsed
			}
		}
	}
	return 0
}

func nestedCodexString(root map[string]any, paths ...[]string) string {
	for _, path := range paths {
		current := any(root)
		ok := true
		for _, segment := range path {
			mapValue, isMap := current.(map[string]any)
			if !isMap {
				ok = false
				break
			}
			current, ok = mapValue[segment]
			if !ok {
				break
			}
		}
		if !ok {
			continue
		}
		if typed, isString := current.(string); isString {
			trimmed := strings.TrimSpace(typed)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func nestedCodexMap(root map[string]any, path []string) (map[string]any, bool) {
	current := any(root)
	for _, segment := range path {
		mapValue, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = mapValue[segment]
		if !ok {
			return nil, false
		}
	}
	mapValue, ok := current.(map[string]any)
	return mapValue, ok
}

func nestedCodexSlice(root map[string]any, path []string) ([]any, bool) {
	current := any(root)
	for _, segment := range path {
		mapValue, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = mapValue[segment]
		if !ok {
			return nil, false
		}
	}
	sliceValue, ok := current.([]any)
	return sliceValue, ok
}
