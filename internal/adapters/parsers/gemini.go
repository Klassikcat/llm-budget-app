package parsers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

const (
	geminiParserName = "gemini-cli"

	GeminiStateSupported   GeminiSupportState = "supported"
	GeminiStateInactive    GeminiSupportState = "inactive"
	GeminiStateUnsupported GeminiSupportState = "unsupported"
)

type GeminiSupportState string

type GeminiSupportStatus struct {
	Provider domain.ProviderName
	Path     string
	State    GeminiSupportState
	Message  string
}

func (s GeminiSupportStatus) Supported() bool {
	return s.State == GeminiStateSupported
}

type GeminiUnsupportedError struct {
	Status GeminiSupportStatus
	Err    error
}

func (e *GeminiUnsupportedError) Error() string {
	if e == nil {
		return "<nil>"
	}

	if e.Err == nil {
		return e.Status.Message
	}

	return fmt.Sprintf("%s: %v", e.Status.Message, e.Err)
}

func (e *GeminiUnsupportedError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

func GeminiStatusFromError(err error) (GeminiSupportStatus, bool) {
	var unsupportedErr *GeminiUnsupportedError
	if !errors.As(err, &unsupportedErr) {
		return GeminiSupportStatus{}, false
	}

	return unsupportedErr.Status, true
}

func IsGeminiState(err error, state GeminiSupportState) bool {
	status, ok := GeminiStatusFromError(err)
	if !ok {
		return false
	}

	return status.State == state
}

type GeminiCLIParser struct{}

func NewGeminiCLIParser() GeminiCLIParser {
	return GeminiCLIParser{}
}

func (GeminiCLIParser) ParserName() string {
	return geminiParserName
}

func (GeminiCLIParser) Parse(_ context.Context, input ports.ParseInput) (ports.ParseResult, error) {
	if len(input.Content) == 0 {
		return ports.ParseResult{NextOffset: input.StartOffset}, &GeminiUnsupportedError{
			Status: GeminiSupportStatus{
				Provider: domain.ProviderGemini,
				Path:     input.Path,
				State:    GeminiStateUnsupported,
				Message:  "Gemini parser supports only chat session JSON files with content",
			},
		}
	}

	trimmed := strings.TrimSpace(string(input.Content))
	if trimmed == "" {
		return ports.ParseResult{NextOffset: input.StartOffset}, &GeminiUnsupportedError{
			Status: GeminiSupportStatus{
				Provider: domain.ProviderGemini,
				Path:     input.Path,
				State:    GeminiStateUnsupported,
				Message:  "Gemini parser supports only chat session JSON files with content",
			},
		}
	}

	if strings.HasPrefix(trimmed, "[") {
		return ports.ParseResult{NextOffset: input.StartOffset}, &GeminiUnsupportedError{
			Status: GeminiSupportStatus{
				Provider: domain.ProviderGemini,
				Path:     input.Path,
				State:    GeminiStateUnsupported,
				Message:  "Gemini log arrays like logs.json are currently unsupported; use chat session JSON files instead",
			},
		}
	}

	var session geminiSessionFile
	if err := json.Unmarshal(input.Content, &session); err != nil {
		return ports.ParseResult{NextOffset: input.StartOffset}, fmt.Errorf("decode Gemini session JSON: %w", err)
	}

	if strings.TrimSpace(session.SessionID) == "" || len(session.Messages) == 0 {
		return ports.ParseResult{NextOffset: input.StartOffset}, &GeminiUnsupportedError{
			Status: GeminiSupportStatus{
				Provider: domain.ProviderGemini,
				Path:     input.Path,
				State:    GeminiStateUnsupported,
				Message:  "Gemini JSON file does not match the supported chat session shape",
			},
		}
	}

	events := make([]ports.SessionEvent, 0, len(session.Messages))
	warnings := make([]string, 0, 2)
	var sawUnmappedTokenDimensions bool

	for index, message := range session.Messages {
		if !strings.EqualFold(strings.TrimSpace(message.Type), "gemini") {
			continue
		}

		if strings.TrimSpace(message.ID) == "" {
			warnings = append(warnings, fmt.Sprintf("skipped Gemini message %d with missing id", index))
			continue
		}

		occurredAt, err := parseGeminiTimestamp(message.Timestamp)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipped Gemini message %q with invalid timestamp: %v", message.ID, err))
			continue
		}

		tokens, err := newGeminiTokenUsage(message.Tokens)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipped Gemini message %q with invalid tokens: %v", message.ID, err))
			continue
		}

		var pricingRef *domain.ModelPricingRef
		if strings.TrimSpace(message.Model) != "" {
			ref, err := domain.NewModelPricingRef(domain.ProviderGemini, message.Model, message.Model)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("Gemini message %q had invalid model reference: %v", message.ID, err))
			} else {
				pricingRef = &ref
			}
		} else {
			warnings = append(warnings, fmt.Sprintf("Gemini message %q is missing model metadata", message.ID))
		}

		costBreakdown, err := domain.NewCostBreakdown(0, 0, 0, 0, 0, 0)
		if err != nil {
			return ports.ParseResult{NextOffset: input.StartOffset}, fmt.Errorf("initialize Gemini cost breakdown: %w", err)
		}

		privacySafeTags := map[string]string{
			"parser":       geminiParserName,
			"project_hash": strings.TrimSpace(session.ProjectHash),
		}
		if message.Tokens != nil && message.Tokens.Thoughts > 0 {
			privacySafeTags["gemini_thought_tokens"] = fmt.Sprintf("%d", message.Tokens.Thoughts)
			sawUnmappedTokenDimensions = true
		}
		if message.Tokens != nil && message.Tokens.Tool > 0 {
			privacySafeTags["gemini_tool_tokens"] = fmt.Sprintf("%d", message.Tokens.Tool)
			sawUnmappedTokenDimensions = true
		}

		events = append(events, ports.SessionEvent{
			EntryID:         fmt.Sprintf("%s:%s", session.SessionID, message.ID),
			ExternalID:      message.ID,
			SessionID:       session.SessionID,
			OccurredAt:      occurredAt,
			Source:          domain.UsageSourceCLISession,
			Provider:        domain.ProviderGemini,
			BillingModeHint: domain.BillingModeUnknown,
			AgentName:       geminiParserName,
			PricingRef:      pricingRef,
			Tokens:          tokens,
			CostBreakdown:   costBreakdown,
			PrivacySafeTags: privacySafeTags,
		})
	}

	if sawUnmappedTokenDimensions {
		warnings = append(warnings, "Gemini messages included thought/tool token dimensions; the parser preserves them as privacy-safe tags and maps only input/output/cached tokens into the shared token fields")
	}

	return ports.ParseResult{
		Events:     events,
		NextOffset: input.StartOffset + int64(len(input.Content)),
		Warnings:   warnings,
	}, nil
}

func ProbeGeminiCLIPath(path string) GeminiSupportStatus {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return GeminiSupportStatus{
			Provider: domain.ProviderGemini,
			State:    GeminiStateUnsupported,
			Message:  "Gemini root path is empty",
		}
	}

	info, err := os.Stat(trimmedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return GeminiSupportStatus{
				Provider: domain.ProviderGemini,
				Path:     trimmedPath,
				State:    GeminiStateInactive,
				Message:  "Gemini CLI root directory is missing",
			}
		}

		return GeminiSupportStatus{
			Provider: domain.ProviderGemini,
			Path:     trimmedPath,
			State:    GeminiStateUnsupported,
			Message:  fmt.Sprintf("Gemini CLI root is not readable: %v", err),
		}
	}

	if !info.IsDir() {
		return classifyGeminiFilePath(trimmedPath)
	}

	hasSupportedSessions := false
	hasLogsJSON := false

	walkErr := filepath.WalkDir(trimmedPath, func(candidate string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		if entry.IsDir() {
			return nil
		}

		if isGeminiSupportedSessionPath(candidate) {
			hasSupportedSessions = true
			return fs.SkipAll
		}

		if strings.EqualFold(filepath.Base(candidate), "logs.json") {
			hasLogsJSON = true
		}

		return nil
	})
	if walkErr != nil {
		return GeminiSupportStatus{
			Provider: domain.ProviderGemini,
			Path:     trimmedPath,
			State:    GeminiStateUnsupported,
			Message:  fmt.Sprintf("Gemini CLI root scan failed: %v", walkErr),
		}
	}

	if hasSupportedSessions {
		return GeminiSupportStatus{
			Provider: domain.ProviderGemini,
			Path:     trimmedPath,
			State:    GeminiStateSupported,
			Message:  "Gemini CLI chat session JSON files were discovered",
		}
	}

	if hasLogsJSON {
		return GeminiSupportStatus{
			Provider: domain.ProviderGemini,
			Path:     trimmedPath,
			State:    GeminiStateUnsupported,
			Message:  "Gemini CLI root contains only unsupported logs.json-style files",
		}
	}

	return GeminiSupportStatus{
		Provider: domain.ProviderGemini,
		Path:     trimmedPath,
		State:    GeminiStateInactive,
		Message:  "Gemini CLI root exists but no supported chat session files were found",
	}
}

func classifyGeminiFilePath(path string) GeminiSupportStatus {
	if isGeminiSupportedSessionPath(path) {
		return GeminiSupportStatus{
			Provider: domain.ProviderGemini,
			Path:     path,
			State:    GeminiStateSupported,
			Message:  "Gemini chat session JSON file is supported",
		}
	}

	if strings.EqualFold(filepath.Base(path), "logs.json") {
		return GeminiSupportStatus{
			Provider: domain.ProviderGemini,
			Path:     path,
			State:    GeminiStateUnsupported,
			Message:  "Gemini logs.json files are not a supported parser input",
		}
	}

	return GeminiSupportStatus{
		Provider: domain.ProviderGemini,
		Path:     path,
		State:    GeminiStateUnsupported,
		Message:  "Gemini file path is not a supported chat session JSON file",
	}
}

func isGeminiSupportedSessionPath(path string) bool {
	cleaned := filepath.Clean(path)
	if !strings.EqualFold(filepath.Ext(cleaned), ".json") {
		return false
	}

	if !strings.HasPrefix(strings.ToLower(filepath.Base(cleaned)), "session-") {
		return false
	}

	return filepath.Base(filepath.Dir(cleaned)) == "chats"
}

func parseGeminiTimestamp(raw string) (time.Time, error) {
	return domain.NormalizeUTCTimestamp("gemini_timestamp", mustParseGeminiTime(raw))
}

func mustParseGeminiTime(raw string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, strings.TrimSpace(raw))
	return parsed
}

func newGeminiTokenUsage(tokens *geminiTokens) (domain.TokenUsage, error) {
	if tokens == nil {
		return domain.NewTokenUsage(0, 0, 0, 0)
	}

	return domain.NewTokenUsage(tokens.Input, tokens.Output, tokens.Cached, 0)
}

type geminiSessionFile struct {
	SessionID   string          `json:"sessionId"`
	ProjectHash string          `json:"projectHash"`
	StartTime   string          `json:"startTime"`
	LastUpdated string          `json:"lastUpdated"`
	Messages    []geminiMessage `json:"messages"`
}

type geminiMessage struct {
	ID        string        `json:"id"`
	Timestamp string        `json:"timestamp"`
	Type      string        `json:"type"`
	Model     string        `json:"model"`
	Tokens    *geminiTokens `json:"tokens"`
}

type geminiTokens struct {
	Input    int64 `json:"input"`
	Output   int64 `json:"output"`
	Cached   int64 `json:"cached"`
	Thoughts int64 `json:"thoughts"`
	Tool     int64 `json:"tool"`
	Total    int64 `json:"total"`
}
