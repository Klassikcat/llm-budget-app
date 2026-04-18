package parsers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"

	_ "modernc.org/sqlite"
)

const openCodeParserName = "opencode"

type OpenCodeParser struct{}

type openCodeRow struct {
	MessageID       string
	SessionID       string
	MessageCreated  int64
	RawData         string
	SessionVersion  string
	SessionSlug     string
	SessionDir      string
	ProjectName     string
	ProjectWorktree string
	ToolCount       int64
}

type openCodeAuthEntry struct {
	Type string `json:"type"`
}

func NewOpenCodeParser() *OpenCodeParser {
	return &OpenCodeParser{}
}

func (p *OpenCodeParser) ParserName() string {
	return openCodeParserName
}

func (p *OpenCodeParser) Parse(ctx context.Context, input ports.ParseInput) (ports.ParseResult, error) {
	result, warnings, err := p.ParseDetailed(ctx, input)
	if err != nil {
		return ports.ParseResult{}, err
	}
	result.Warnings = openCodeWarningsToStrings(warnings)
	return result, nil
}

func (p *OpenCodeParser) ParseDetailed(ctx context.Context, input ports.ParseInput) (ports.ParseResult, []OpenCodeWarning, error) {
	result := ports.ParseResult{NextOffset: input.StartOffset}
	warnings := make([]OpenCodeWarning, 0, 4)

	dbPath, authPath, pathWarnings := resolveOpenCodePaths(input.Path)
	warnings = append(warnings, pathWarnings...)
	if dbPath == "" {
		result.Warnings = openCodeWarningsToStrings(warnings)
		return result, warnings, nil
	}

	if info, err := os.Stat(dbPath); err == nil {
		result.NextOffset = input.StartOffset + info.Size()
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		warnings = append(warnings, OpenCodeWarning{
			Code:   OpenCodeWarningDatabaseOpen,
			Path:   input.Path,
			Detail: err.Error(),
		})
		result.Warnings = openCodeWarningsToStrings(warnings)
		return result, warnings, nil
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		warnings = append(warnings, OpenCodeWarning{
			Code:   OpenCodeWarningDatabaseOpen,
			Path:   input.Path,
			Detail: err.Error(),
		})
		result.Warnings = openCodeWarningsToStrings(warnings)
		return result, warnings, nil
	}

	required := []string{"project", "session", "message"}
	for _, table := range required {
		exists, existsErr := openCodeTableExists(ctx, db, table)
		if existsErr != nil {
			warnings = append(warnings, OpenCodeWarning{
				Code:    OpenCodeWarningMissingTable,
				Path:    input.Path,
				Variant: table,
				Detail:  existsErr.Error(),
			})
			result.Warnings = openCodeWarningsToStrings(warnings)
			return result, warnings, nil
		}
		if !exists {
			warnings = append(warnings, OpenCodeWarning{
				Code:    OpenCodeWarningMissingTable,
				Path:    input.Path,
				Variant: table,
				Detail:  "required OpenCode table is missing",
			})
			result.Warnings = openCodeWarningsToStrings(warnings)
			return result, warnings, nil
		}
	}

	hasPartTable, err := openCodeTableExists(ctx, db, "part")
	if err != nil {
		warnings = append(warnings, OpenCodeWarning{
			Code:    OpenCodeWarningMissingTable,
			Path:    input.Path,
			Variant: "part",
			Detail:  err.Error(),
		})
		hasPartTable = false
	}
	if !hasPartTable {
		warnings = append(warnings, OpenCodeWarning{
			Code:    OpenCodeWarningSchemaDrift,
			Path:    input.Path,
			Variant: "part",
			Detail:  "optional part table is missing; tool-call counts will default to zero",
		})
	}

	authHints, authWarnings := loadOpenCodeBillingHints(authPath, input.Path)
	warnings = append(warnings, authWarnings...)

	rows, err := db.QueryContext(ctx, openCodeRowsQuery(hasPartTable))
	if err != nil {
		warnings = append(warnings, OpenCodeWarning{
			Code:   OpenCodeWarningDatabaseOpen,
			Path:   input.Path,
			Detail: err.Error(),
		})
		result.Warnings = openCodeWarningsToStrings(warnings)
		return result, warnings, nil
	}
	defer rows.Close()

	seenVersions := map[string]struct{}{}
	for rows.Next() {
		var row openCodeRow
		if err := rows.Scan(
			&row.MessageID,
			&row.SessionID,
			&row.MessageCreated,
			&row.RawData,
			&row.SessionVersion,
			&row.SessionSlug,
			&row.SessionDir,
			&row.ProjectName,
			&row.ProjectWorktree,
			&row.ToolCount,
		); err != nil {
			warnings = append(warnings, OpenCodeWarning{
				Code:   OpenCodeWarningDatabaseOpen,
				Path:   input.Path,
				Detail: err.Error(),
			})
			continue
		}

		if _, ok := seenVersions[row.SessionVersion]; !ok {
			seenVersions[row.SessionVersion] = struct{}{}
			if !strings.HasPrefix(strings.TrimSpace(row.SessionVersion), "1.") {
				warnings = append(warnings, OpenCodeWarning{
					Code:      OpenCodeWarningSchemaDrift,
					Path:      input.Path,
					SessionID: row.SessionID,
					Variant:   row.SessionVersion,
					Detail:    "session version was not covered by the discovered v1 fixtures; parsing continued using structured message fields",
				})
			}
		}

		event, rowWarnings := buildOpenCodeEvent(input.Path, row, authHints)
		warnings = append(warnings, rowWarnings...)
		if event != nil {
			result.Events = append(result.Events, *event)
		}
	}

	if err := rows.Err(); err != nil {
		warnings = append(warnings, OpenCodeWarning{
			Code:   OpenCodeWarningDatabaseOpen,
			Path:   input.Path,
			Detail: err.Error(),
		})
	}

	result.Warnings = openCodeWarningsToStrings(warnings)
	return result, warnings, nil
}

func buildOpenCodeEvent(path string, row openCodeRow, authHints map[string]domain.BillingMode) (*ports.SessionEvent, []OpenCodeWarning) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(row.RawData), &payload); err != nil {
		return nil, []OpenCodeWarning{{
			Code:      OpenCodeWarningInvalidJSON,
			Path:      path,
			SessionID: row.SessionID,
			RecordID:  row.MessageID,
			Detail:    err.Error(),
		}}
	}

	providerRaw := openCodeFirstString(payload, "providerID", "providerId", "provider")
	if providerRaw == "" {
		return nil, []OpenCodeWarning{{
			Code:      OpenCodeWarningMissingProvider,
			Path:      path,
			SessionID: row.SessionID,
			RecordID:  row.MessageID,
			Detail:    "assistant message is missing provider metadata",
		}}
	}

	provider, ok := openCodeProviderName(providerRaw)
	if !ok {
		return nil, []OpenCodeWarning{{
			Code:      OpenCodeWarningUnknownProvider,
			Path:      path,
			SessionID: row.SessionID,
			RecordID:  row.MessageID,
			Variant:   providerRaw,
			Detail:    "provider is not mapped into the shared provider enum",
		}}
	}

	tokensValue, ok := openCodeNestedMap(payload, []string{"tokens"})
	if !ok {
		return nil, []OpenCodeWarning{{
			Code:      OpenCodeWarningMissingTokens,
			Path:      path,
			SessionID: row.SessionID,
			RecordID:  row.MessageID,
			Detail:    "assistant message is missing the structured tokens object",
		}}
	}

	tokens, tokenWarnings, err := openCodeTokenUsage(tokensValue)
	if err != nil {
		return nil, []OpenCodeWarning{{
			Code:      OpenCodeWarningInvalidTokens,
			Path:      path,
			SessionID: row.SessionID,
			RecordID:  row.MessageID,
			Detail:    err.Error(),
		}}
	}

	costs, err := openCodeCostBreakdown(payload)
	if err != nil {
		return nil, []OpenCodeWarning{{
			Code:      OpenCodeWarningInvalidCost,
			Path:      path,
			SessionID: row.SessionID,
			RecordID:  row.MessageID,
			Detail:    err.Error(),
		}}
	}

	occurredAt, err := openCodeOccurredAt(payload, row.MessageCreated)
	if err != nil {
		return nil, []OpenCodeWarning{{
			Code:      OpenCodeWarningInvalidTimestamp,
			Path:      path,
			SessionID: row.SessionID,
			RecordID:  row.MessageID,
			Detail:    err.Error(),
		}}
	}

	modelID := openCodeFirstString(payload, "modelID", "modelId", "model")
	var pricingRef *domain.ModelPricingRef
	if modelID != "" {
		ref, refErr := domain.NewModelPricingRef(provider, modelID, modelID)
		if refErr != nil {
			tokenWarnings = append(tokenWarnings, OpenCodeWarning{
				Code:      OpenCodeWarningInvalidPricingRef,
				Path:      path,
				SessionID: row.SessionID,
				RecordID:  row.MessageID,
				Detail:    refErr.Error(),
			})
		} else {
			pricingRef = &ref
		}
	}

	agentName := strings.TrimSpace(openCodeFirstString(payload, "agent", "mode"))
	if agentName == "" {
		agentName = openCodeParserName
	}

	billingModeHint := openCodeBillingModeHint(providerRaw, authHints)
	projectName := openCodeProjectName(row.ProjectName, row.ProjectWorktree, row.SessionDir)

	privacySafeTags := map[string]string{
		"parser":                   openCodeParserName,
		"opencode_schema":          "sqlite/session+message+part",
		"opencode_session_version": strings.TrimSpace(row.SessionVersion),
		"opencode_session_slug":    strings.TrimSpace(row.SessionSlug),
	}
	if finish := strings.TrimSpace(openCodeFirstString(payload, "finish")); finish != "" {
		privacySafeTags["opencode_finish"] = finish
	}
	if variant := strings.TrimSpace(openCodeFirstString(payload, "variant")); variant != "" {
		privacySafeTags["opencode_variant"] = variant
	}
	if mode := strings.TrimSpace(openCodeFirstString(payload, "mode")); mode != "" {
		privacySafeTags["opencode_mode"] = mode
	}
	if reasoning := openCodeIntString(tokensValue, "reasoning"); reasoning != "" {
		privacySafeTags["opencode_reasoning_tokens"] = reasoning
	}
	if cacheWrite := openCodeNestedIntString(tokensValue, []string{"cache", "write"}); cacheWrite != "" {
		privacySafeTags["opencode_cache_write_tokens"] = cacheWrite
	}
	if costs.TotalUSD > 0 {
		privacySafeTags["opencode_cost_source"] = "message_total"
	}

	return &ports.SessionEvent{
		EntryID:          row.MessageID,
		ExternalID:       row.MessageID,
		SessionID:        row.SessionID,
		OccurredAt:       occurredAt,
		Source:           domain.UsageSourceCLISession,
		Provider:         provider,
		BillingModeHint:  billingModeHint,
		ProjectName:      projectName,
		AgentName:        agentName,
		PricingRef:       pricingRef,
		Tokens:           tokens,
		CostBreakdown:    costs,
		PrivacySafeTags:  privacySafeTags,
		ObservedToolCall: row.ToolCount,
	}, tokenWarnings
}

func resolveOpenCodePaths(path string) (string, string, []OpenCodeWarning) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return "", "", []OpenCodeWarning{{
			Code:   OpenCodeWarningEmptyPath,
			Path:   path,
			Detail: "OpenCode parser requires a root, database, auth file, or log path",
		}}
	}

	info, err := os.Stat(trimmedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", []OpenCodeWarning{{
				Code:   OpenCodeWarningMissingDatabase,
				Path:   path,
				Detail: "OpenCode path does not exist",
			}}
		}
		return "", "", []OpenCodeWarning{{
			Code:   OpenCodeWarningPathUnreadable,
			Path:   path,
			Detail: err.Error(),
		}}
	}

	rootCandidates := make([]string, 0, 4)
	if info.IsDir() {
		rootCandidates = append(rootCandidates, trimmedPath)
	} else {
		rootCandidates = append(rootCandidates, filepath.Dir(trimmedPath))
		parent := filepath.Dir(filepath.Dir(trimmedPath))
		if parent != "" && parent != "." {
			rootCandidates = append(rootCandidates, parent)
		}
	}

	for _, candidate := range rootCandidates {
		dbPath := filepath.Join(candidate, "opencode.db")
		if fileExists(dbPath) {
			authPath := filepath.Join(candidate, "auth.json")
			if !fileExists(authPath) {
				authPath = ""
			}
			return dbPath, authPath, nil
		}
	}

	if !info.IsDir() && strings.EqualFold(filepath.Base(trimmedPath), "opencode.db") {
		authPath := filepath.Join(filepath.Dir(trimmedPath), "auth.json")
		if !fileExists(authPath) {
			authPath = ""
		}
		return trimmedPath, authPath, nil
	}

	return "", "", []OpenCodeWarning{{
		Code:   OpenCodeWarningMissingDatabase,
		Path:   path,
		Detail: "OpenCode database was not found beside the provided path",
	}}
}

func openCodeRowsQuery(hasPartTable bool) string {
	toolCount := "0 as tool_count"
	if hasPartTable {
		toolCount = `coalesce((
			select count(*)
			from part
			where part.message_id = message.id
			  and json_extract(part.data, '$.type') = 'tool'
		), 0) as tool_count`
	}

	return fmt.Sprintf(`
		select
			message.id,
			message.session_id,
			message.time_created,
			message.data,
			session.version,
			session.slug,
			session.directory,
			coalesce(project.name, ''),
			project.worktree,
			%s
		from message
		join session on session.id = message.session_id
		join project on project.id = session.project_id
		where json_extract(message.data, '$.role') = 'assistant'
		order by message.time_created, message.id
	`, toolCount)
}

func openCodeTableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	var exists int
	err := db.QueryRowContext(ctx, `select count(*) from sqlite_master where type = 'table' and name = ?`, table).Scan(&exists)
	return exists > 0, err
}

func loadOpenCodeBillingHints(authPath, sourcePath string) (map[string]domain.BillingMode, []OpenCodeWarning) {
	if strings.TrimSpace(authPath) == "" {
		return nil, nil
	}

	content, err := os.ReadFile(authPath)
	if err != nil {
		return nil, []OpenCodeWarning{{
			Code:   OpenCodeWarningAuthHints,
			Path:   sourcePath,
			Detail: err.Error(),
		}}
	}

	raw := map[string]openCodeAuthEntry{}
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, []OpenCodeWarning{{
			Code:   OpenCodeWarningAuthHints,
			Path:   sourcePath,
			Detail: err.Error(),
		}}
	}

	hints := make(map[string]domain.BillingMode, len(raw))
	for provider, entry := range raw {
		switch normalizeCodexLabel(entry.Type) {
		case "api":
			if normalizeCodexLabel(provider) == "openrouter" {
				hints[normalizeCodexLabel(provider)] = domain.BillingModeOpenRouter
			} else {
				hints[normalizeCodexLabel(provider)] = domain.BillingModeBYOK
			}
		case "oauth":
			if normalizeCodexLabel(provider) == "openrouter" {
				hints[normalizeCodexLabel(provider)] = domain.BillingModeOpenRouter
			} else {
				hints[normalizeCodexLabel(provider)] = domain.BillingModeSubscription
			}
		}
	}

	return hints, nil
}

func openCodeProviderName(raw string) (domain.ProviderName, bool) {
	switch normalizeCodexLabel(raw) {
	case "openai":
		return domain.ProviderOpenAI, true
	case "anthropic":
		return domain.ProviderAnthropic, true
	case "openrouter":
		return domain.ProviderOpenRouter, true
	case "google", "gemini":
		return domain.ProviderGemini, true
	case "github_copilot", "codex":
		return domain.ProviderCodex, true
	case "opencode":
		return domain.ProviderOpenCode, true
	default:
		return "", false
	}
}

func openCodeBillingModeHint(providerRaw string, authHints map[string]domain.BillingMode) domain.BillingMode {
	normalized := normalizeCodexLabel(providerRaw)
	if hint, ok := authHints[normalized]; ok {
		return hint
	}
	if normalized == "openrouter" {
		return domain.BillingModeOpenRouter
	}
	return domain.BillingModeUnknown
}

func openCodeProjectName(projectName, projectWorktree, sessionDir string) string {
	if trimmed := strings.TrimSpace(projectName); trimmed != "" {
		return trimmed
	}
	for _, candidate := range []string{projectWorktree, sessionDir} {
		base := filepath.Base(strings.TrimSpace(candidate))
		if base != "" && base != "." && base != "/" {
			return base
		}
	}
	return ""
}

func openCodeOccurredAt(payload map[string]any, fallbackMS int64) (time.Time, error) {
	if timeMap, ok := openCodeNestedMap(payload, []string{"time"}); ok {
		for _, path := range [][]string{{"completed"}, {"created"}} {
			if raw, ok := openCodeNestedValue(timeMap, path...); ok {
				if parsed, ok := openCodeMilliseconds(raw); ok {
					return time.UnixMilli(parsed).UTC(), nil
				}
			}
		}
	}
	if fallbackMS > 0 {
		return time.UnixMilli(fallbackMS).UTC(), nil
	}
	return time.Time{}, fmt.Errorf("no usable millisecond timestamp was found")
}

func openCodeTokenUsage(tokens map[string]any) (domain.TokenUsage, []OpenCodeWarning, error) {
	input := openCodeInt(tokens["input"])
	output := openCodeInt(tokens["output"])
	reasoning := openCodeInt(tokens["reasoning"])

	var cacheRead int64
	var cacheWrite int64
	var warnings []OpenCodeWarning
	if cacheMap, ok := openCodeNestedMap(tokens, []string{"cache"}); ok {
		cacheRead = openCodeInt(cacheMap["read"])
		cacheWrite = openCodeInt(cacheMap["write"])
	} else if rawCache, ok := tokens["cache"]; ok {
		cacheRead = openCodeInt(rawCache)
		warnings = append(warnings, OpenCodeWarning{
			Code:   OpenCodeWarningSchemaDrift,
			Detail: "tokens.cache used a scalar value; parser treated it as cache read tokens",
		})
	}

	usage, err := domain.NewTokenUsage(input+reasoning, output, cacheRead, cacheWrite)
	if err != nil {
		return domain.TokenUsage{}, warnings, err
	}
	return usage, warnings, nil
}

func openCodeCostBreakdown(payload map[string]any) (domain.CostBreakdown, error) {
	rawCost, ok := payload["cost"]
	if !ok || rawCost == nil {
		return domain.NewCostBreakdown(0, 0, 0, 0, 0, 0)
	}

	if costMap, ok := rawCost.(map[string]any); ok {
		return domain.NewCostBreakdown(
			openCodeFloat(costMap["input"]),
			openCodeFloat(costMap["output"]),
			openCodeFloat(costMap["cache_read"]),
			openCodeFloat(costMap["cache_write"]),
			openCodeFloat(costMap["tool"]),
			openCodeFloat(firstOpenCodeValue(costMap, "flat", "total", "amount")),
		)
	}

	return domain.NewCostBreakdown(0, 0, 0, 0, 0, openCodeFloat(rawCost))
}

func openCodeFirstString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			if normalized := strings.TrimSpace(fmt.Sprint(value)); normalized != "" && normalized != "<nil>" {
				return normalized
			}
		}
	}
	return ""
}

func openCodeNestedMap(payload map[string]any, path []string) (map[string]any, bool) {
	current := any(payload)
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

func openCodeNestedValue(payload map[string]any, path ...string) (any, bool) {
	current := any(payload)
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
	return current, true
}

func openCodeMilliseconds(raw any) (int64, bool) {
	value := openCodeInt(raw)
	return value, value > 0
}

func openCodeInt(raw any) int64 {
	switch value := raw.(type) {
	case float64:
		return int64(value)
	case float32:
		return int64(value)
	case int:
		return int64(value)
	case int64:
		return value
	case int32:
		return int64(value)
	case json.Number:
		parsed, _ := value.Int64()
		return parsed
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return 0
		}
		parsed, _ := time.ParseDuration(trimmed)
		if parsed > 0 {
			return int64(parsed)
		}
		var intValue int64
		fmt.Sscan(trimmed, &intValue)
		return intValue
	default:
		return 0
	}
}

func openCodeFloat(raw any) float64 {
	switch value := raw.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case json.Number:
		parsed, _ := value.Float64()
		return parsed
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return 0
		}
		var floatValue float64
		fmt.Sscan(trimmed, &floatValue)
		return floatValue
	default:
		return 0
	}
}

func openCodeIntString(payload map[string]any, key string) string {
	if value, ok := payload[key]; ok {
		return fmt.Sprintf("%d", openCodeInt(value))
	}
	return ""
}

func openCodeNestedIntString(payload map[string]any, path []string) string {
	value, ok := openCodeNestedValue(payload, path...)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d", openCodeInt(value))
}

func firstOpenCodeValue(payload map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			return value
		}
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
