package parsers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

const openClawParserName = "openclaw"

type OpenClawParser struct{}

type OpenClawWarningCode string

const (
	OpenClawWarningDataSourceNotFound OpenClawWarningCode = "data_source_not_found"
	OpenClawWarningPathUnreadable     OpenClawWarningCode = "path_unreadable"
	OpenClawWarningUnsupportedPath    OpenClawWarningCode = "unsupported_path"
	OpenClawWarningMalformedJSON      OpenClawWarningCode = "malformed_json"
	OpenClawWarningUnsupportedSchema  OpenClawWarningCode = "unsupported_schema"
	OpenClawWarningMissingTimestamp   OpenClawWarningCode = "missing_timestamp"
	OpenClawWarningInvalidTimestamp   OpenClawWarningCode = "invalid_timestamp"
	OpenClawWarningMissingProvider    OpenClawWarningCode = "missing_provider"
	OpenClawWarningInvalidProvider    OpenClawWarningCode = "invalid_provider"
	OpenClawWarningMissingModel       OpenClawWarningCode = "missing_model"
	OpenClawWarningInvalidPricingRef  OpenClawWarningCode = "invalid_pricing_ref"
	OpenClawWarningMissingTokens      OpenClawWarningCode = "missing_tokens"
	OpenClawWarningInvalidTokens      OpenClawWarningCode = "invalid_tokens"
	OpenClawWarningInvalidCost        OpenClawWarningCode = "invalid_cost"
)

type OpenClawWarning struct {
	Code     OpenClawWarningCode
	Path     string
	Line     int
	RecordID string
	Variant  string
	Detail   string
}

func NewOpenClawParser() *OpenClawParser {
	return &OpenClawParser{}
}

func (p *OpenClawParser) ParserName() string {
	return openClawParserName
}

func (p *OpenClawParser) Parse(ctx context.Context, input ports.ParseInput) (ports.ParseResult, error) {
	result, warnings, err := p.ParseDetailed(ctx, input)
	if err != nil {
		return ports.ParseResult{}, err
	}
	result.Warnings = openClawWarningsToStrings(warnings)
	return result, nil
}

func (p *OpenClawParser) ParseDetailed(ctx context.Context, input ports.ParseInput) (ports.ParseResult, []OpenClawWarning, error) {
	_ = ctx

	result := ports.ParseResult{NextOffset: input.StartOffset}
	homeDir, _ := os.UserHomeDir()
	selectedPath, warnings := resolveOpenClawDataSource(input.Path, homeDir, os.Getenv("OPENCLAW_STATE_DIR"))
	if selectedPath == "" {
		result.Warnings = openClawWarningsToStrings(warnings)
		return result, warnings, nil
	}

	info, err := os.Stat(selectedPath)
	if err != nil {
		warnings = append(warnings, OpenClawWarning{Code: OpenClawWarningPathUnreadable, Path: input.Path, Detail: err.Error()})
		result.Warnings = openClawWarningsToStrings(warnings)
		return result, warnings, nil
	}

	if len(input.Content) > 0 || !info.IsDir() {
		content := input.Content
		if len(content) == 0 {
			content, err = os.ReadFile(selectedPath)
			if err != nil {
				warnings = append(warnings, OpenClawWarning{Code: OpenClawWarningPathUnreadable, Path: selectedPath, Detail: err.Error()})
				result.Warnings = openClawWarningsToStrings(warnings)
				return result, warnings, nil
			}
		}
		if len(input.Content) > 0 {
			result.NextOffset = input.StartOffset + int64(len(input.Content))
		} else {
			result.NextOffset = input.StartOffset + info.Size()
		}
		events, parseWarnings := parseOpenClawContent(selectedPath, content)
		result.Events = append(result.Events, events...)
		warnings = append(warnings, parseWarnings...)
		result.Warnings = openClawWarningsToStrings(warnings)
		return result, warnings, nil
	}

	files, walkWarnings := openClawUsageFiles(selectedPath)
	warnings = append(warnings, walkWarnings...)
	for _, filePath := range files {
		content, err := os.ReadFile(filePath)
		if err != nil {
			warnings = append(warnings, OpenClawWarning{Code: OpenClawWarningPathUnreadable, Path: filePath, Detail: err.Error()})
			continue
		}
		events, parseWarnings := parseOpenClawContent(filePath, content)
		result.Events = append(result.Events, events...)
		warnings = append(warnings, parseWarnings...)
	}
	result.Warnings = openClawWarningsToStrings(warnings)
	return result, warnings, nil
}

func parseOpenClawContent(path string, content []byte) ([]ports.SessionEvent, []OpenClawWarning) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jsonl":
		return parseOpenClawJSONL(path, content)
	case ".json":
		return parseOpenClawJSON(path, content)
	case ".sqlite", ".db":
		return nil, []OpenClawWarning{{Code: OpenClawWarningUnsupportedSchema, Path: path, Detail: "sqlite OpenClaw usage schema is not supported by this parser yet"}}
	default:
		return nil, []OpenClawWarning{{Code: OpenClawWarningUnsupportedPath, Path: path, Detail: "openclaw data source is not a supported data file"}}
	}
}

func parseOpenClawJSONL(path string, content []byte) ([]ports.SessionEvent, []OpenClawWarning) {
	lines := bytes.Split(content, []byte("\n"))
	events := make([]ports.SessionEvent, 0, len(lines))
	warnings := make([]OpenClawWarning, 0)
	for index, rawLine := range lines {
		lineNumber := index + 1
		trimmed := bytes.TrimSpace(rawLine)
		if len(trimmed) == 0 {
			continue
		}

		var record map[string]any
		decoder := json.NewDecoder(bytes.NewReader(trimmed))
		decoder.UseNumber()
		if err := decoder.Decode(&record); err != nil {
			warnings = append(warnings, OpenClawWarning{Code: OpenClawWarningMalformedJSON, Path: path, Line: lineNumber, Detail: err.Error()})
			continue
		}

		event, recordWarnings := buildOpenClawEvent(path, lineNumber, "jsonl_usage", record)
		warnings = append(warnings, recordWarnings...)
		if event != nil {
			events = append(events, *event)
		}
	}
	return events, warnings
}

func parseOpenClawJSON(path string, content []byte) ([]ports.SessionEvent, []OpenClawWarning) {
	var root any
	decoder := json.NewDecoder(bytes.NewReader(bytes.TrimSpace(content)))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil {
		return nil, []OpenClawWarning{{Code: OpenClawWarningMalformedJSON, Path: path, Detail: err.Error()}}
	}

	records, ok := openClawJSONRecords(root)
	if !ok {
		return nil, []OpenClawWarning{{Code: OpenClawWarningUnsupportedSchema, Path: path, Detail: "JSON file does not contain usage records"}}
	}

	events := make([]ports.SessionEvent, 0, len(records))
	warnings := make([]OpenClawWarning, 0)
	for index, record := range records {
		event, recordWarnings := buildOpenClawEvent(path, index+1, "json_usage", record)
		warnings = append(warnings, recordWarnings...)
		if event != nil {
			events = append(events, *event)
		}
	}
	return events, warnings
}

func openClawJSONRecords(root any) ([]map[string]any, bool) {
	switch value := root.(type) {
	case []any:
		return openClawRecordsFromSlice(value)
	case map[string]any:
		for _, key := range []string{"records", "usage", "events", "messages"} {
			if rawRecords, ok := value[key].([]any); ok {
				return openClawRecordsFromSlice(rawRecords)
			}
		}
		return []map[string]any{value}, true
	default:
		return nil, false
	}
}

func openClawRecordsFromSlice(values []any) ([]map[string]any, bool) {
	records := make([]map[string]any, 0, len(values))
	for _, value := range values {
		record, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		records = append(records, record)
	}
	return records, true
}

func buildOpenClawEvent(path string, lineNumber int, shape string, record map[string]any) (*ports.SessionEvent, []OpenClawWarning) {
	var warnings []OpenClawWarning
	recordID := firstOpenClawString(record, "request_id", "requestId", "id", "message_id", "messageId")

	occurredAt, timestampWarning := openClawOccurredAt(path, lineNumber, recordID, record)
	if timestampWarning != nil {
		return nil, []OpenClawWarning{*timestampWarning}
	}

	provider, providerWarning := openClawProvider(path, lineNumber, recordID, record)
	if providerWarning != nil {
		return nil, []OpenClawWarning{*providerWarning}
	}

	modelID := normalizeOpenClawModel(firstOpenClawString(record, "model", "model_id", "modelId"))
	if modelID == "" {
		modelID = normalizeOpenClawModel(nestedOpenClawString(record, []string{"message", "model"}, []string{"response", "model"}, []string{"usage", "model"}))
	}
	if modelID == "" {
		return nil, []OpenClawWarning{{Code: OpenClawWarningMissingModel, Path: path, Line: lineNumber, RecordID: recordID, Detail: "usage record is missing model metadata"}}
	}

	usageMap, ok := openClawUsageMap(record)
	if !ok {
		return nil, []OpenClawWarning{{Code: OpenClawWarningMissingTokens, Path: path, Line: lineNumber, RecordID: recordID, Detail: "usage record is missing structured token counters"}}
	}

	tokens, ok, err := openClawTokenUsage(usageMap)
	if err != nil {
		return nil, []OpenClawWarning{{Code: OpenClawWarningInvalidTokens, Path: path, Line: lineNumber, RecordID: recordID, Detail: err.Error()}}
	}
	if !ok {
		return nil, []OpenClawWarning{{Code: OpenClawWarningMissingTokens, Path: path, Line: lineNumber, RecordID: recordID, Detail: "usage record is missing input/output/cache token counters"}}
	}

	costs, err := openClawCostBreakdown(record)
	if err != nil {
		return nil, []OpenClawWarning{{Code: OpenClawWarningInvalidCost, Path: path, Line: lineNumber, RecordID: recordID, Detail: err.Error()}}
	}

	pricingRef, err := domain.NewModelPricingRef(provider, modelID, modelID)
	if err != nil {
		warnings = append(warnings, OpenClawWarning{Code: OpenClawWarningInvalidPricingRef, Path: path, Line: lineNumber, RecordID: recordID, Detail: err.Error()})
		pricingRef = domain.ModelPricingRef{Provider: provider, ModelID: modelID, PricingLookupKey: modelID}
	}

	sessionID := firstOpenClawString(record, "session_id", "sessionId")
	if sessionID == "" {
		sessionID = recordID
	}
	if sessionID == "" {
		sessionID = fmt.Sprintf("%s:%d", strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), lineNumber)
	}

	entryID := recordID
	if entryID == "" {
		entryID = fmt.Sprintf("%s:%d", sessionID, lineNumber)
	}

	projectName := firstOpenClawString(record, "project", "project_name", "projectName", "workspace", "workspace_name", "workspaceName")
	if projectName == "" {
		projectName = openClawProjectFromPath(firstOpenClawString(record, "cwd", "workdir", "workspace_path", "workspacePath"))
	}

	return &ports.SessionEvent{
		EntryID:         entryID,
		ExternalID:      recordID,
		SessionID:       sessionID,
		OccurredAt:      occurredAt,
		Source:          domain.UsageSourceCLISession,
		Provider:        provider,
		BillingModeHint: openClawBillingModeHint(record, provider),
		ProjectName:     projectName,
		AgentName:       openClawParserName,
		PricingRef:      &pricingRef,
		Tokens:          tokens,
		CostBreakdown:   costs,
		PrivacySafeTags: map[string]string{
			"parser":                openClawParserName,
			"openclaw_record_shape": shape,
		},
	}, warnings
}

func openClawOccurredAt(path string, lineNumber int, recordID string, record map[string]any) (time.Time, *OpenClawWarning) {
	raw := firstOpenClawString(record, "timestamp", "recorded_at", "recordedAt", "created_at", "createdAt")
	if raw == "" {
		return time.Time{}, &OpenClawWarning{Code: OpenClawWarningMissingTimestamp, Path: path, Line: lineNumber, RecordID: recordID, Detail: "usage record is missing timestamp metadata"}
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, &OpenClawWarning{Code: OpenClawWarningInvalidTimestamp, Path: path, Line: lineNumber, RecordID: recordID, Detail: err.Error()}
	}
	return parsed.UTC(), nil
}

func openClawProvider(path string, lineNumber int, recordID string, record map[string]any) (domain.ProviderName, *OpenClawWarning) {
	raw := firstOpenClawString(record, "provider", "provider_id", "providerId")
	if raw == "" {
		raw = nestedOpenClawString(record, []string{"message", "provider"}, []string{"response", "provider"}, []string{"usage", "provider"})
	}
	if raw == "" {
		raw = inferOpenClawProviderFromModel(firstOpenClawString(record, "model", "model_id", "modelId"))
	}
	if raw == "" {
		return "", &OpenClawWarning{Code: OpenClawWarningMissingProvider, Path: path, Line: lineNumber, RecordID: recordID, Detail: "usage record is missing provider metadata"}
	}

	provider, err := domain.NewProviderName(normalizeOpenClawProvider(raw))
	if err != nil {
		return "", &OpenClawWarning{Code: OpenClawWarningInvalidProvider, Path: path, Line: lineNumber, RecordID: recordID, Detail: err.Error()}
	}
	return provider, nil
}

func openClawUsageMap(record map[string]any) (map[string]any, bool) {
	for _, path := range [][]string{{"usage"}, {"tokens"}, {"message", "usage"}, {"response", "usage"}} {
		if usage, ok := nestedOpenClawMap(record, path); ok {
			return usage, true
		}
	}
	return record, openClawHasAnyTokenField(record)
}

func openClawTokenUsage(usage map[string]any) (domain.TokenUsage, bool, error) {
	input, hasInput, err := firstOpenClawInt(usage, "input_tokens", "prompt_tokens", "input", "prompt")
	if err != nil {
		return domain.TokenUsage{}, false, err
	}
	output, hasOutput, err := firstOpenClawInt(usage, "output_tokens", "completion_tokens", "output", "completion")
	if err != nil {
		return domain.TokenUsage{}, false, err
	}
	cacheRead, hasCacheRead, err := firstOpenClawInt(usage, "cache_read_input_tokens", "cached_tokens", "cache_read_tokens", "cache_read", "cached")
	if err != nil {
		return domain.TokenUsage{}, false, err
	}
	cacheWrite, hasCacheWrite, err := firstOpenClawInt(usage, "cache_creation_input_tokens", "cache_write_tokens", "cache_write", "cache_creation")
	if err != nil {
		return domain.TokenUsage{}, false, err
	}
	if cacheMap, ok := nestedOpenClawMap(usage, []string{"cache"}); ok {
		if value, ok, err := firstOpenClawInt(cacheMap, "read"); err != nil {
			return domain.TokenUsage{}, false, err
		} else if ok {
			cacheRead = value
			hasCacheRead = true
		}
		if value, ok, err := firstOpenClawInt(cacheMap, "write"); err != nil {
			return domain.TokenUsage{}, false, err
		} else if ok {
			cacheWrite = value
			hasCacheWrite = true
		}
	}
	if !hasInput && !hasOutput && !hasCacheRead && !hasCacheWrite {
		return domain.TokenUsage{}, false, nil
	}
	usageValue, err := domain.NewTokenUsage(input, output, cacheRead, cacheWrite)
	return usageValue, true, err
}

func openClawCostBreakdown(record map[string]any) (domain.CostBreakdown, error) {
	inputUSD, _, err := firstOpenClawFloat(record, "input_usd", "inputUSD")
	if err != nil {
		return domain.CostBreakdown{}, err
	}
	outputUSD, _, err := firstOpenClawFloat(record, "output_usd", "outputUSD")
	if err != nil {
		return domain.CostBreakdown{}, err
	}
	cacheReadUSD, _, err := firstOpenClawFloat(record, "cache_read_usd", "cacheReadUSD")
	if err != nil {
		return domain.CostBreakdown{}, err
	}
	cacheWriteUSD, _, err := firstOpenClawFloat(record, "cache_write_usd", "cacheWriteUSD", "cache_creation_usd", "cacheCreationUSD")
	if err != nil {
		return domain.CostBreakdown{}, err
	}
	toolUSD, _, err := firstOpenClawFloat(record, "tool_usd", "toolUSD")
	if err != nil {
		return domain.CostBreakdown{}, err
	}
	flatUSD, _, err := firstOpenClawFloat(record, "cost_usd", "costUSD", "total_usd", "totalUSD", "amount_usd", "amountUSD")
	if err != nil {
		return domain.CostBreakdown{}, err
	}
	if costMap, ok := nestedOpenClawMap(record, []string{"cost"}); ok {
		if value, ok, err := firstOpenClawFloat(costMap, "input", "input_usd", "inputUSD"); err != nil {
			return domain.CostBreakdown{}, err
		} else if ok {
			inputUSD = value
		}
		if value, ok, err := firstOpenClawFloat(costMap, "output", "output_usd", "outputUSD"); err != nil {
			return domain.CostBreakdown{}, err
		} else if ok {
			outputUSD = value
		}
		if value, ok, err := firstOpenClawFloat(costMap, "cache_read", "cacheRead", "cache_read_usd", "cacheReadUSD"); err != nil {
			return domain.CostBreakdown{}, err
		} else if ok {
			cacheReadUSD = value
		}
		if value, ok, err := firstOpenClawFloat(costMap, "cache_write", "cacheWrite", "cache_write_usd", "cacheWriteUSD"); err != nil {
			return domain.CostBreakdown{}, err
		} else if ok {
			cacheWriteUSD = value
		}
		if value, ok, err := firstOpenClawFloat(costMap, "tool", "tool_usd", "toolUSD"); err != nil {
			return domain.CostBreakdown{}, err
		} else if ok {
			toolUSD = value
		}
		if value, ok, err := firstOpenClawFloat(costMap, "flat", "total", "amount", "usd", "cost_usd", "costUSD"); err != nil {
			return domain.CostBreakdown{}, err
		} else if ok {
			flatUSD = value
		}
	} else if value, ok := record["cost"]; ok && value != nil {
		parsed, err := openClawFloat(value)
		if err != nil {
			return domain.CostBreakdown{}, fmt.Errorf("field %q: invalid numeric cost value", "cost")
		}
		flatUSD = parsed
	}
	return domain.NewCostBreakdown(inputUSD, outputUSD, cacheReadUSD, cacheWriteUSD, toolUSD, flatUSD)
}

func openClawBillingModeHint(record map[string]any, provider domain.ProviderName) domain.BillingMode {
	for _, raw := range []string{firstOpenClawString(record, "billing_mode", "billingMode", "auth_mode", "authMode", "auth_source", "authSource")} {
		switch normalizeCodexLabel(raw) {
		case "subscription", "plan", "oauth", "plus", "pro":
			return domain.BillingModeSubscription
		case "byok", "api", "api_key", "apikey", "api-key":
			return domain.BillingModeBYOK
		case "direct_api", "directapi":
			return domain.BillingModeDirectAPI
		case "openrouter":
			return domain.BillingModeOpenRouter
		}
	}
	if provider == domain.ProviderOpenRouter {
		return domain.BillingModeOpenRouter
	}
	return domain.BillingModeUnknown
}

func openClawHasAnyTokenField(record map[string]any) bool {
	for _, key := range []string{"input_tokens", "prompt_tokens", "output_tokens", "completion_tokens", "cache_read_input_tokens", "cached_tokens", "cache_creation_input_tokens"} {
		if _, ok := record[key]; ok {
			return true
		}
	}
	return false
}

func openClawUsageFiles(root string) ([]string, []OpenClawWarning) {
	var files []string
	var warnings []OpenClawWarning
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			warnings = append(warnings, OpenClawWarning{Code: OpenClawWarningPathUnreadable, Path: path, Detail: err.Error()})
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".jsonl" || ext == ".json" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		warnings = append(warnings, OpenClawWarning{Code: OpenClawWarningPathUnreadable, Path: root, Detail: err.Error()})
	}
	sort.Strings(files)
	return files, warnings
}

func resolveOpenClawDataSource(path, homeDir, stateDirOverride string) (string, []OpenClawWarning) {
	if trimmedPath := strings.TrimSpace(path); trimmedPath != "" {
		return resolveOpenClawExplicitPath(trimmedPath, path)
	}

	for _, candidate := range openClawCandidateStateDirs(homeDir, stateDirOverride) {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return "", []OpenClawWarning{{
				Code:   OpenClawWarningPathUnreadable,
				Path:   candidate,
				Detail: err.Error(),
			}}
		}
	}

	return "", []OpenClawWarning{{
		Code:   OpenClawWarningDataSourceNotFound,
		Path:   strings.Join(openClawCandidateStateDirs(homeDir, stateDirOverride), ","),
		Detail: "openclaw data source not found",
	}}
}

func resolveOpenClawExplicitPath(trimmedPath, originalPath string) (string, []OpenClawWarning) {
	info, err := os.Stat(trimmedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", []OpenClawWarning{{
				Code:   OpenClawWarningDataSourceNotFound,
				Path:   originalPath,
				Detail: "openclaw data source not found",
			}}
		}
		return "", []OpenClawWarning{{
			Code:   OpenClawWarningPathUnreadable,
			Path:   originalPath,
			Detail: err.Error(),
		}}
	}

	if info.IsDir() || openClawSupportedDataFile(trimmedPath) {
		return trimmedPath, nil
	}

	return "", []OpenClawWarning{{
		Code:   OpenClawWarningUnsupportedPath,
		Path:   originalPath,
		Detail: "openclaw data source is not a directory or supported data file",
	}}
}

func openClawCandidateStateDirs(homeDir, stateDirOverride string) []string {
	candidates := make([]string, 0, 2)
	if trimmedOverride := strings.TrimSpace(stateDirOverride); trimmedOverride != "" {
		candidates = append(candidates, trimmedOverride)
	}
	if trimmedHome := strings.TrimSpace(homeDir); trimmedHome != "" {
		candidates = append(candidates, filepath.Join(trimmedHome, ".openclaw"))
	}
	return candidates
}

func openClawSupportedDataFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json", ".jsonl", ".sqlite", ".db":
		return true
	default:
		return false
	}
}

func firstOpenClawString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := m[key]
		if !ok {
			continue
		}
		if typed, ok := value.(string); ok {
			trimmed := strings.TrimSpace(typed)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func firstOpenClawInt(m map[string]any, keys ...string) (int64, bool, error) {
	for _, key := range keys {
		value, ok := m[key]
		if !ok || value == nil {
			continue
		}
		parsed, err := openClawInt(value)
		if err != nil {
			return 0, true, fmt.Errorf("field %q: invalid integer token value", key)
		}
		return parsed, true, nil
	}
	return 0, false, nil
}

func firstOpenClawFloat(m map[string]any, keys ...string) (float64, bool, error) {
	for _, key := range keys {
		value, ok := m[key]
		if !ok || value == nil {
			continue
		}
		parsed, err := openClawFloat(value)
		if err != nil {
			return 0, true, fmt.Errorf("field %q: invalid numeric cost value", key)
		}
		return parsed, true, nil
	}
	return 0, false, nil
}

func openClawInt(raw any) (int64, error) {
	switch value := raw.(type) {
	case json.Number:
		return value.Int64()
	case float64:
		if math.Trunc(value) != value {
			return 0, fmt.Errorf("not an integer")
		}
		return int64(value), nil
	case int64:
		return value, nil
	case int:
		return int64(value), nil
	case string:
		return strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	default:
		return 0, fmt.Errorf("unsupported numeric type")
	}
}

func openClawFloat(raw any) (float64, error) {
	switch value := raw.(type) {
	case json.Number:
		return value.Float64()
	case float64:
		return value, nil
	case float32:
		return float64(value), nil
	case int64:
		return float64(value), nil
	case int:
		return float64(value), nil
	case string:
		return strconv.ParseFloat(strings.TrimSpace(value), 64)
	default:
		return 0, fmt.Errorf("unsupported numeric type")
	}
}

func nestedOpenClawMap(root map[string]any, path []string) (map[string]any, bool) {
	current := any(root)
	for _, segment := range path {
		mapping, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := mapping[segment]
		if !ok {
			return nil, false
		}
		current = next
	}
	mapping, ok := current.(map[string]any)
	return mapping, ok
}

func nestedOpenClawString(root map[string]any, paths ...[]string) string {
	for _, path := range paths {
		current := any(root)
		ok := true
		for _, segment := range path {
			mapping, isMap := current.(map[string]any)
			if !isMap {
				ok = false
				break
			}
			current, ok = mapping[segment]
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

func normalizeOpenClawProvider(raw string) string {
	normalized := normalizeCodexLabel(raw)
	switch normalized {
	case "google":
		return "gemini"
	case "open_router":
		return "openrouter"
	case "github_copilot":
		return "codex"
	default:
		return strings.ReplaceAll(normalized, "_", "-")
	}
}

func normalizeOpenClawModel(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	trimmed = strings.ReplaceAll(trimmed, " ", "-")
	trimmed = strings.ReplaceAll(trimmed, "_", "-")
	return trimmed
}

func inferOpenClawProviderFromModel(model string) string {
	normalized := normalizeOpenClawModel(model)
	switch {
	case strings.HasPrefix(normalized, "claude"):
		return "anthropic"
	case strings.HasPrefix(normalized, "gpt"), strings.HasPrefix(normalized, "o1"), strings.HasPrefix(normalized, "o3"), strings.HasPrefix(normalized, "o4"):
		return "openai"
	case strings.HasPrefix(normalized, "gemini"):
		return "gemini"
	case strings.Contains(normalized, "/"):
		return strings.Split(normalized, "/")[0]
	default:
		return ""
	}
}

func openClawProjectFromPath(path string) string {
	base := filepath.Base(strings.TrimSpace(path))
	if base == "" || base == "." || base == "/" {
		return ""
	}
	return base
}

func (w OpenClawWarning) String() string {
	base := fmt.Sprintf("openclaw warning [%s] path=%s", w.Code, w.Path)
	if w.Line > 0 {
		base += fmt.Sprintf(" line=%d", w.Line)
	}
	if w.RecordID != "" {
		base += fmt.Sprintf(" record=%s", w.RecordID)
	}
	if w.Variant != "" {
		base += fmt.Sprintf(" variant=%s", w.Variant)
	}
	if w.Detail != "" {
		base += fmt.Sprintf(": %s", w.Detail)
	}
	return base
}

func openClawWarningsToStrings(warnings []OpenClawWarning) []string {
	if len(warnings) == 0 {
		return nil
	}

	result := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		result = append(result, warning.String())
	}
	return result
}

var _ ports.SessionParser = (*OpenClawParser)(nil)
