package parsers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"llm-budget-tracker/internal/ports"
)

func TestOpenClawDiscoveryParserName(t *testing.T) {
	parser := NewOpenClawParser()

	if got := parser.ParserName(); got != "openclaw" {
		t.Fatalf("ParserName() = %q, want openclaw", got)
	}
}

func TestOpenClawDiscoveryPathNotFoundWarnsWithoutFatalError(t *testing.T) {
	parser := NewOpenClawParser()
	missingPath := filepath.Join(t.TempDir(), "missing-openclaw")

	result, err := parser.Parse(context.Background(), ports.ParseInput{Path: missingPath, StartOffset: 7})
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(result.Events) != 0 {
		t.Fatalf("len(result.Events) = %d, want 0", len(result.Events))
	}
	if result.NextOffset != 7 {
		t.Fatalf("result.NextOffset = %d, want 7", result.NextOffset)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("len(result.Warnings) = %d, want 1: %v", len(result.Warnings), result.Warnings)
	}
	if !strings.Contains(result.Warnings[0], "openclaw data source not found") {
		t.Fatalf("warning = %q, want not-found warning", result.Warnings[0])
	}
}

func TestOpenClawDiscoveryValidCandidateFromEnvironment(t *testing.T) {
	stateRoot := t.TempDir()

	selected, warnings := resolveOpenClawDataSource("", "", stateRoot)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if selected != stateRoot {
		t.Fatalf("selected = %q, want %q", selected, stateRoot)
	}

	t.Setenv("OPENCLAW_STATE_DIR", stateRoot)
	parser := NewOpenClawParser()
	result, err := parser.Parse(context.Background(), ports.ParseInput{StartOffset: 11})
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(result.Events) != 0 {
		t.Fatalf("len(result.Events) = %d, want 0", len(result.Events))
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("result.Warnings = %v, want none", result.Warnings)
	}
	if result.NextOffset != 11 {
		t.Fatalf("result.NextOffset = %d, want 11", result.NextOffset)
	}
}

func TestOpenClawDiscoveryManualPathOverridePrecedence(t *testing.T) {
	envRoot := t.TempDir()
	manualRoot := t.TempDir()

	selected, warnings := resolveOpenClawDataSource(manualRoot, filepath.Join(t.TempDir(), "home"), envRoot)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if selected != manualRoot {
		t.Fatalf("selected = %q, want manual path %q", selected, manualRoot)
	}
}

func TestOpenClawDiscoveryManualSupportedFileOverridePrecedence(t *testing.T) {
	envRoot := t.TempDir()
	manualFile := filepath.Join(t.TempDir(), "session.jsonl")
	if err := writeTestFile(manualFile); err != nil {
		t.Fatalf("writeTestFile() error = %v", err)
	}

	selected, warnings := resolveOpenClawDataSource(manualFile, filepath.Join(t.TempDir(), "home"), envRoot)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if selected != manualFile {
		t.Fatalf("selected = %q, want manual file %q", selected, manualFile)
	}
}

func TestOpenClawDiscoveryPlatformCandidateListBehavior(t *testing.T) {
	tests := []struct {
		name string
		home string
	}{
		{name: "macOS", home: filepath.Join(string(filepath.Separator), "Users", "alice")},
		{name: "Linux", home: filepath.Join(string(filepath.Separator), "home", "alice")},
		{name: "Windows", home: `C:\Users\alice`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidates := openClawCandidateStateDirs(test.home, "")
			if len(candidates) != 1 {
				t.Fatalf("len(candidates) = %d, want 1: %v", len(candidates), candidates)
			}
			want := filepath.Join(test.home, ".openclaw")
			if candidates[0] != want {
				t.Fatalf("candidate = %q, want %q", candidates[0], want)
			}
		})
	}

	envRoot := filepath.Join(t.TempDir(), "env-openclaw")
	candidates := openClawCandidateStateDirs(filepath.Join(t.TempDir(), "home"), envRoot)
	if len(candidates) != 2 {
		t.Fatalf("len(candidates) = %d, want 2: %v", len(candidates), candidates)
	}
	if candidates[0] != envRoot {
		t.Fatalf("first candidate = %q, want env override %q", candidates[0], envRoot)
	}
}

func writeTestFile(path string) error {
	return os.WriteFile(path, []byte("{}\n"), 0o644)
}

func TestOpenClawParserParsesSyntheticUsageRecords(t *testing.T) {
	fixturePath := filepath.Join("testdata", "openclaw", "usage-valid.jsonl")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", fixturePath, err)
	}

	parser := NewOpenClawParser()
	result, warnings, err := parser.ParseDetailed(context.Background(), ports.ParseInput{
		SourceID:    "openclaw-fixture",
		Path:        fixturePath,
		Content:     content,
		StartOffset: 5,
	})
	if err != nil {
		t.Fatalf("ParseDetailed() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if len(result.Events) != 2 {
		t.Fatalf("len(result.Events) = %d, want 2", len(result.Events))
	}
	if result.NextOffset != int64(5+len(content)) {
		t.Fatalf("result.NextOffset = %d, want %d", result.NextOffset, 5+len(content))
	}

	first := result.Events[0]
	if first.Provider != "openai" {
		t.Fatalf("first.Provider = %q, want openai", first.Provider)
	}
	if first.SessionID != "oc_sess_1" || first.ExternalID != "oc_req_1" || first.EntryID != "oc_req_1" {
		t.Fatalf("first identifiers = entry %q external %q session %q, want request/session ids", first.EntryID, first.ExternalID, first.SessionID)
	}
	if first.ProjectName != "budget-app" {
		t.Fatalf("first.ProjectName = %q, want budget-app", first.ProjectName)
	}
	if first.PricingRef == nil || first.PricingRef.Provider != "openai" || first.PricingRef.ModelID != "gpt-5-mini" {
		t.Fatalf("first.PricingRef = %#v, want normalized openai/gpt-5-mini", first.PricingRef)
	}
	if first.Tokens.InputTokens != 1000 || first.Tokens.OutputTokens != 200 || first.Tokens.CacheReadTokens != 150 || first.Tokens.CacheWriteTokens != 25 {
		t.Fatalf("first.Tokens = %+v, want normalized token counters", first.Tokens)
	}
	if first.CostBreakdown.TotalUSD != 0.045 {
		t.Fatalf("first.CostBreakdown.TotalUSD = %v, want 0.045", first.CostBreakdown.TotalUSD)
	}
	if got := first.PrivacySafeTags["openclaw_record_shape"]; got != "jsonl_usage" {
		t.Fatalf("first openclaw_record_shape = %q, want jsonl_usage", got)
	}

	second := result.Events[1]
	if second.Provider != "anthropic" {
		t.Fatalf("second.Provider = %q, want anthropic", second.Provider)
	}
	if second.SessionID != "oc_req_2" || second.ExternalID != "oc_req_2" {
		t.Fatalf("second identifiers = external %q session %q, want request fallback", second.ExternalID, second.SessionID)
	}
	if second.ProjectName != "planning-space" {
		t.Fatalf("second.ProjectName = %q, want planning-space", second.ProjectName)
	}
	if second.PricingRef == nil || second.PricingRef.ModelID != "claude-3-7-sonnet-20250219" {
		t.Fatalf("second.PricingRef = %#v, want normalized model", second.PricingRef)
	}
	if second.Tokens.InputTokens != 300 || second.Tokens.OutputTokens != 75 || second.Tokens.CacheReadTokens != 40 || second.Tokens.CacheWriteTokens != 0 {
		t.Fatalf("second.Tokens = %+v, want normalized token counters", second.Tokens)
	}
	if second.CostBreakdown.TotalUSD != 0 {
		t.Fatalf("second.CostBreakdown.TotalUSD = %v, want zero missing-cost fallback", second.CostBreakdown.TotalUSD)
	}
	assertOpenClawNoSentinelLeak(t, result.Events, result.Warnings)
}

func TestOpenClawParserSkipsMalformedRecordWithPrivacySafeWarning(t *testing.T) {
	fixturePath := filepath.Join("testdata", "openclaw", "usage-malformed.jsonl")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", fixturePath, err)
	}

	parser := NewOpenClawParser()
	result, warnings, err := parser.ParseDetailed(context.Background(), ports.ParseInput{
		SourceID: "openclaw-malformed-fixture",
		Path:     fixturePath,
		Content:  content,
	})
	if err != nil {
		t.Fatalf("ParseDetailed() error = %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("len(result.Events) = %d, want 1", len(result.Events))
	}
	if len(warnings) != 1 {
		t.Fatalf("len(warnings) = %d, want 1", len(warnings))
	}
	if warnings[0].Code != OpenClawWarningMalformedJSON {
		t.Fatalf("warnings[0].Code = %q, want %q", warnings[0].Code, OpenClawWarningMalformedJSON)
	}
	if warnings[0].Line != 2 {
		t.Fatalf("warnings[0].Line = %d, want 2", warnings[0].Line)
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != warnings[0].String() {
		t.Fatalf("result.Warnings = %v, want typed warning string", result.Warnings)
	}
	if strings.Contains(result.Warnings[0], "SENSITIVE_TRANSCRIPT_SENTINEL") {
		t.Fatalf("warning leaked sentinel transcript content: %q", result.Warnings[0])
	}
	if result.Events[0].Provider != "gemini" || result.Events[0].CostBreakdown.TotalUSD != 0.003 {
		t.Fatalf("event = %+v, want valid record preserved", result.Events[0])
	}
	assertOpenClawNoSentinelLeak(t, result.Events, result.Warnings)
}

func TestOpenClawParserSkipsRecordsMissingTokens(t *testing.T) {
	fixturePath := filepath.Join("testdata", "openclaw", "usage-missing-tokens.jsonl")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", fixturePath, err)
	}

	parser := NewOpenClawParser()
	result, warnings, err := parser.ParseDetailed(context.Background(), ports.ParseInput{
		SourceID: "openclaw-missing-tokens-fixture",
		Path:     fixturePath,
		Content:  content,
	})
	if err != nil {
		t.Fatalf("ParseDetailed() error = %v", err)
	}
	if len(result.Events) != 0 {
		t.Fatalf("len(result.Events) = %d, want 0", len(result.Events))
	}
	if len(warnings) != 1 {
		t.Fatalf("len(warnings) = %d, want 1", len(warnings))
	}
	if warnings[0].Code != OpenClawWarningMissingTokens {
		t.Fatalf("warnings[0].Code = %q, want %q", warnings[0].Code, OpenClawWarningMissingTokens)
	}
	if strings.Contains(warnings[0].String(), "SENSITIVE_TRANSCRIPT_SENTINEL") {
		t.Fatalf("warning leaked sentinel transcript content: %q", warnings[0].String())
	}
}

func TestOpenClawParserSkipsMalformedNumericFields(t *testing.T) {
	content := []byte(strings.Join([]string{
		`{"timestamp":"2026-04-17T12:00:00Z","request_id":"bad-token","provider":"openai","model":"gpt-5-mini","usage":{"input_tokens":"not-a-number","output_tokens":20},"cost_usd":"0.01","prompt":"SENSITIVE_TRANSCRIPT_SENTINEL"}`,
		`{"timestamp":"2026-04-17T12:01:00Z","request_id":"bad-cost","provider":"openai","model":"gpt-5-mini","usage":{"input_tokens":10,"output_tokens":20},"cost_usd":"not-a-cost","response":"SENSITIVE_TRANSCRIPT_SENTINEL"}`,
	}, "\n"))

	parser := NewOpenClawParser()
	result, warnings, err := parser.ParseDetailed(context.Background(), ports.ParseInput{
		SourceID: "openclaw-malformed-numeric-fixture",
		Path:     filepath.Join("testdata", "openclaw", "usage-valid.jsonl"),
		Content:  content,
	})
	if err != nil {
		t.Fatalf("ParseDetailed() error = %v", err)
	}
	if len(result.Events) != 0 {
		t.Fatalf("len(result.Events) = %d, want 0", len(result.Events))
	}
	if len(warnings) != 2 {
		t.Fatalf("len(warnings) = %d, want 2: %v", len(warnings), warnings)
	}
	if warnings[0].Code != OpenClawWarningInvalidTokens {
		t.Fatalf("warnings[0].Code = %q, want %q", warnings[0].Code, OpenClawWarningInvalidTokens)
	}
	if warnings[1].Code != OpenClawWarningInvalidCost {
		t.Fatalf("warnings[1].Code = %q, want %q", warnings[1].Code, OpenClawWarningInvalidCost)
	}
	if !strings.Contains(warnings[0].Detail, "input_tokens") || !strings.Contains(warnings[1].Detail, "cost_usd") {
		t.Fatalf("warning details = %q / %q, want privacy-safe field names", warnings[0].Detail, warnings[1].Detail)
	}
	for _, warning := range warnings {
		if strings.Contains(warning.String(), "not-a-number") || strings.Contains(warning.String(), "not-a-cost") || strings.Contains(warning.String(), "SENSITIVE_TRANSCRIPT_SENTINEL") {
			t.Fatalf("warning leaked raw malformed value or transcript content: %q", warning.String())
		}
	}
}

func assertOpenClawNoSentinelLeak(t *testing.T, events []ports.SessionEvent, warnings []string) {
	t.Helper()

	const sentinel = "SENSITIVE_TRANSCRIPT_SENTINEL"
	for _, warning := range warnings {
		if strings.Contains(warning, sentinel) {
			t.Fatalf("warning leaked sentinel transcript content: %q", warning)
		}
	}
	for _, event := range events {
		for _, value := range []string{event.EntryID, event.ExternalID, event.SessionID, event.ProjectName, event.AgentName} {
			if strings.Contains(value, sentinel) {
				t.Fatalf("event leaked sentinel transcript content: %+v", event)
			}
		}
		for key, value := range event.PrivacySafeTags {
			if strings.Contains(key, sentinel) || strings.Contains(value, sentinel) {
				t.Fatalf("PrivacySafeTags leaked sentinel transcript content: %v", event.PrivacySafeTags)
			}
		}
	}
}
