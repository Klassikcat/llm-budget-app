package parsers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

const (
	ClaudeCodeParserName = "claude_code"
	claudeCodeAgentName  = "claude-code"

	claudeLocationCurrent = "current"
	claudeLocationLegacy  = "legacy"
	claudeLocationUnknown = "unknown"
)

type ClaudeCodeParser struct{}

type claudeCodeRecord struct {
	CWD       string             `json:"cwd"`
	SessionID string             `json:"sessionId"`
	Timestamp string             `json:"timestamp"`
	Version   string             `json:"version"`
	RequestID string             `json:"requestId"`
	CostUSD   *float64           `json:"costUSD"`
	Message   *claudeCodeMessage `json:"message"`
}

type claudeCodeMessage struct {
	ID      string                   `json:"id"`
	Model   string                   `json:"model"`
	Usage   *claudeCodeUsage         `json:"usage"`
	Content []claudeCodeContentBlock `json:"content"`
}

type claudeCodeUsage struct {
	InputTokens         *int64 `json:"input_tokens"`
	OutputTokens        *int64 `json:"output_tokens"`
	CacheCreationTokens *int64 `json:"cache_creation_input_tokens"`
	CacheReadTokens     *int64 `json:"cache_read_input_tokens"`
	Speed               string `json:"speed"`
}

type claudeCodeContentBlock struct {
	Type string `json:"type"`
}

type claudeLineDisposition int

const (
	claudeLineAccepted claudeLineDisposition = iota
	claudeLineSkipped
	claudeLinePartial
)

func NewClaudeCodeParser() *ClaudeCodeParser {
	return &ClaudeCodeParser{}
}

func (p *ClaudeCodeParser) ParserName() string {
	return ClaudeCodeParserName
}

func (p *ClaudeCodeParser) Parse(_ context.Context, input ports.ParseInput) (ports.ParseResult, error) {
	contentLength := int64(len(input.Content))
	result := ports.ParseResult{
		Events:     []ports.SessionEvent{},
		NextOffset: contentLength,
		Warnings:   []string{},
	}

	startOffset := input.StartOffset
	if startOffset < 0 {
		result.Warnings = append(result.Warnings, "claude parser: negative start offset; restarting from beginning")
		startOffset = 0
	}

	if startOffset > contentLength {
		result.Warnings = append(result.Warnings, "claude parser: start offset exceeds content length; restarting from beginning")
		startOffset = 0
	}

	seen := make(map[string]struct{})
	lineStart := startOffset

	for idx := startOffset; idx < contentLength; idx++ {
		if input.Content[idx] != '\n' {
			continue
		}

		line := trimTrailingCarriageReturn(input.Content[lineStart:idx])
		lineStart = idx + 1

		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		disposition, event, dedupeKey, warnings := parseClaudeCodeLine(input.Path, line, false)
		result.Warnings = append(result.Warnings, warnings...)

		switch disposition {
		case claudeLineAccepted:
			if _, exists := seen[dedupeKey]; exists {
				result.Warnings = append(result.Warnings, fmt.Sprintf("claude parser: skipped duplicate row at byte offset %d", lineStart-1))
				continue
			}
			seen[dedupeKey] = struct{}{}
			result.Events = append(result.Events, event)
		case claudeLineSkipped:
			continue
		case claudeLinePartial:
			result.NextOffset = lineStart - 1
			return result, nil
		}
	}

	if lineStart >= contentLength {
		return result, nil
	}

	tail := trimTrailingCarriageReturn(input.Content[lineStart:])
	if len(bytes.TrimSpace(tail)) == 0 {
		return result, nil
	}

	disposition, event, dedupeKey, warnings := parseClaudeCodeLine(input.Path, tail, true)
	result.Warnings = append(result.Warnings, warnings...)

	switch disposition {
	case claudeLineAccepted:
		if _, exists := seen[dedupeKey]; exists {
			result.Warnings = append(result.Warnings, fmt.Sprintf("claude parser: skipped duplicate row at byte offset %d", lineStart))
			return result, nil
		}
		seen[dedupeKey] = struct{}{}
		result.Events = append(result.Events, event)
	case claudeLineSkipped:
		return result, nil
	case claudeLinePartial:
		result.Warnings = append(result.Warnings, fmt.Sprintf("claude parser: skipped partial trailing line at byte offset %d", lineStart))
		result.NextOffset = lineStart
	}

	return result, nil
}

func parseClaudeCodeLine(path string, line []byte, allowPartial bool) (claudeLineDisposition, ports.SessionEvent, string, []string) {
	var record claudeCodeRecord
	if err := json.Unmarshal(line, &record); err != nil {
		if allowPartial {
			return claudeLinePartial, ports.SessionEvent{}, "", nil
		}
		return claudeLineSkipped, ports.SessionEvent{}, "", []string{"claude parser: skipped invalid JSONL row"}
	}

	if record.Message == nil || record.Message.Usage == nil || record.Message.Usage.InputTokens == nil || record.Message.Usage.OutputTokens == nil {
		return claudeLineSkipped, ports.SessionEvent{}, "", []string{"claude parser: skipped row without required usage fields"}
	}

	occurredAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(record.Timestamp))
	if err != nil {
		return claudeLineSkipped, ports.SessionEvent{}, "", []string{"claude parser: skipped row with invalid timestamp"}
	}

	tokens, err := domain.NewTokenUsage(
		*record.Message.Usage.InputTokens,
		*record.Message.Usage.OutputTokens,
		int64Value(record.Message.Usage.CacheReadTokens),
		int64Value(record.Message.Usage.CacheCreationTokens),
	)
	if err != nil {
		return claudeLineSkipped, ports.SessionEvent{}, "", []string{fmt.Sprintf("claude parser: skipped row with invalid token usage: %v", err)}
	}

	costs, err := domain.NewCostBreakdown(0, 0, 0, 0, 0, float64Value(record.CostUSD))
	if err != nil {
		return claudeLineSkipped, ports.SessionEvent{}, "", []string{fmt.Sprintf("claude parser: skipped row with invalid cost usage: %v", err)}
	}

	var pricingRef *domain.ModelPricingRef
	modelID := strings.TrimSpace(record.Message.Model)
	if modelID != "" {
		ref, err := domain.NewModelPricingRef(domain.ProviderClaude, modelID, modelID)
		if err == nil {
			pricingRef = &ref
		}
	}

	sessionID := strings.TrimSpace(record.SessionID)
	if sessionID == "" {
		sessionID = claudeSessionIDFromPath(path)
	}

	projectName := claudeProjectName(path, record.CWD)
	location := claudeLocationKind(path)
	privacySafeTags := map[string]string{"location": location}
	if speed := strings.TrimSpace(record.Message.Usage.Speed); speed != "" {
		privacySafeTags["speed"] = speed
	}
	if version := strings.TrimSpace(record.Version); version != "" {
		privacySafeTags["version"] = version
	}

	dedupeKey := claudeDedupeKey(record, line)
	externalID := firstNonEmpty(strings.TrimSpace(record.RequestID), strings.TrimSpace(record.Message.ID), dedupeKey)

	event := ports.SessionEvent{
		EntryID:          stableClaudeEntryID(path, dedupeKey, occurredAt),
		ExternalID:       externalID,
		SessionID:        sessionID,
		OccurredAt:       occurredAt.UTC(),
		Source:           domain.UsageSourceCLISession,
		Provider:         domain.ProviderClaude,
		BillingModeHint:  domain.BillingModeUnknown,
		ProjectName:      projectName,
		AgentName:        claudeCodeAgentName,
		PricingRef:       pricingRef,
		Tokens:           tokens,
		CostBreakdown:    costs,
		PrivacySafeTags:  privacySafeTags,
		ObservedToolCall: countClaudeToolCalls(record.Message.Content),
	}

	return claudeLineAccepted, event, dedupeKey, nil
}

func trimTrailingCarriageReturn(line []byte) []byte {
	return bytes.TrimSuffix(line, []byte{'\r'})
}

func claudeLocationKind(path string) string {
	normalized := filepath.ToSlash(path)
	switch {
	case strings.Contains(normalized, "/.config/claude/"):
		return claudeLocationCurrent
	case strings.Contains(normalized, "/.claude/"):
		return claudeLocationLegacy
	default:
		return claudeLocationUnknown
	}
}

func claudeProjectName(path, cwd string) string {
	trimmedCWD := strings.TrimSpace(cwd)
	if trimmedCWD != "" {
		base := filepath.Base(filepath.Clean(trimmedCWD))
		if base != "." && base != string(filepath.Separator) && base != "" {
			return base
		}
	}

	normalized := strings.Split(filepath.ToSlash(path), "/")
	for idx, segment := range normalized {
		if segment == "projects" && idx+1 < len(normalized) {
			project := strings.TrimSpace(normalized[idx+1])
			if project != "" {
				return project
			}
		}
	}

	return "unknown"
}

func claudeSessionIDFromPath(path string) string {
	base := strings.TrimSpace(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	if base == "" {
		return "unknown"
	}
	return base
}

func claudeDedupeKey(record claudeCodeRecord, line []byte) string {
	messageID := strings.TrimSpace(record.Message.ID)
	requestID := strings.TrimSpace(record.RequestID)

	switch {
	case messageID != "" && requestID != "":
		return "message_request:" + messageID + ":" + requestID
	case messageID != "":
		return "message:" + messageID
	case requestID != "":
		return "request:" + requestID
	default:
		sum := sha256.Sum256(bytes.TrimSpace(line))
		return "line:" + hex.EncodeToString(sum[:])
	}
}

func stableClaudeEntryID(path, dedupeKey string, occurredAt time.Time) string {
	sum := sha256.Sum256([]byte(path + "\n" + dedupeKey + "\n" + occurredAt.UTC().Format(time.RFC3339Nano)))
	return "claude-" + hex.EncodeToString(sum[:])
}

func countClaudeToolCalls(content []claudeCodeContentBlock) int64 {
	var total int64
	for _, block := range content {
		if strings.EqualFold(strings.TrimSpace(block.Type), "tool_use") {
			total++
		}
	}
	return total
}

func int64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func float64Value(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

var _ ports.SessionParser = (*ClaudeCodeParser)(nil)
