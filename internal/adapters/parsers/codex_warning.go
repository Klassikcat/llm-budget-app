package parsers

import "fmt"

type CodexWarningCode string

const (
	CodexWarningMalformedJSON      CodexWarningCode = "malformed_json"
	CodexWarningUnsupportedVariant CodexWarningCode = "unsupported_variant"
	CodexWarningMissingTimestamp   CodexWarningCode = "missing_timestamp"
	CodexWarningInvalidTimestamp   CodexWarningCode = "invalid_timestamp"
	CodexWarningInvalidUsage       CodexWarningCode = "invalid_usage"
	CodexWarningInvalidCost        CodexWarningCode = "invalid_cost"
)

type CodexWarning struct {
	Code    CodexWarningCode
	Path    string
	Line    int
	Variant string
	Detail  string
}

func (w CodexWarning) String() string {
	base := fmt.Sprintf("codex warning [%s] path=%s line=%d", w.Code, w.Path, w.Line)
	if w.Variant != "" {
		base += fmt.Sprintf(" variant=%s", w.Variant)
	}
	if w.Detail != "" {
		base += fmt.Sprintf(": %s", w.Detail)
	}
	return base
}

func codexWarningsToStrings(warnings []CodexWarning) []string {
	if len(warnings) == 0 {
		return nil
	}

	result := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		result = append(result, warning.String())
	}

	return result
}
