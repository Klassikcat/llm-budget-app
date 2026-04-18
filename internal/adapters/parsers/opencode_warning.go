package parsers

import "fmt"

type OpenCodeWarningCode string

const (
	OpenCodeWarningEmptyPath         OpenCodeWarningCode = "empty_path"
	OpenCodeWarningPathUnreadable    OpenCodeWarningCode = "path_unreadable"
	OpenCodeWarningMissingDatabase   OpenCodeWarningCode = "missing_database"
	OpenCodeWarningDatabaseOpen      OpenCodeWarningCode = "database_open"
	OpenCodeWarningMissingTable      OpenCodeWarningCode = "missing_table"
	OpenCodeWarningSchemaDrift       OpenCodeWarningCode = "schema_drift"
	OpenCodeWarningAuthHints         OpenCodeWarningCode = "auth_hints"
	OpenCodeWarningInvalidJSON       OpenCodeWarningCode = "invalid_json"
	OpenCodeWarningMissingProvider   OpenCodeWarningCode = "missing_provider"
	OpenCodeWarningUnknownProvider   OpenCodeWarningCode = "unknown_provider"
	OpenCodeWarningMissingTokens     OpenCodeWarningCode = "missing_tokens"
	OpenCodeWarningInvalidTokens     OpenCodeWarningCode = "invalid_tokens"
	OpenCodeWarningInvalidCost       OpenCodeWarningCode = "invalid_cost"
	OpenCodeWarningInvalidTimestamp  OpenCodeWarningCode = "invalid_timestamp"
	OpenCodeWarningInvalidPricingRef OpenCodeWarningCode = "invalid_pricing_ref"
)

type OpenCodeWarning struct {
	Code      OpenCodeWarningCode
	Path      string
	SessionID string
	RecordID  string
	Variant   string
	Detail    string
}

func (w OpenCodeWarning) String() string {
	base := fmt.Sprintf("opencode warning [%s] path=%s", w.Code, w.Path)
	if w.SessionID != "" {
		base += fmt.Sprintf(" session=%s", w.SessionID)
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

func openCodeWarningsToStrings(warnings []OpenCodeWarning) []string {
	if len(warnings) == 0 {
		return nil
	}

	result := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		result = append(result, warning.String())
	}

	return result
}
