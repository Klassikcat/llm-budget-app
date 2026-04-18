package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

type rawCatalogDocument struct {
	SchemaVersion  string            `json:"schema_version"`
	CatalogVersion string            `json:"catalog_version"`
	Source         string            `json:"source"`
	Provider       string            `json:"provider"`
	SyncedAt       string            `json:"synced_at,omitempty"`
	Entries        []rawCatalogEntry `json:"entries"`
}

type rawCatalogEntry struct {
	Provider             string   `json:"provider,omitempty"`
	ModelID              string   `json:"model_id"`
	LookupKey            string   `json:"lookup_key,omitempty"`
	InputUSDPer1M        *float64 `json:"input_usd_per_1m"`
	OutputUSDPer1M       *float64 `json:"output_usd_per_1m"`
	CacheReadUSDPer1M    *float64 `json:"cache_read_usd_per_1m,omitempty"`
	CacheWriteUSDPer1M   *float64 `json:"cache_write_usd_per_1m,omitempty"`
	ToolUSDPerInvocation *float64 `json:"tool_usd_per_invocation,omitempty"`
	CachedAt             string   `json:"cached_at,omitempty"`
	ExpiresAt            string   `json:"expires_at,omitempty"`
}

func parseCatalogDocument(data []byte, sourceName string) (catalogDocument, error) {
	raw, err := decodeRawCatalogDocument(data)
	if err != nil {
		return catalogDocument{}, fmt.Errorf("decode catalog %s: %w", sourceName, err)
	}

	if strings.TrimSpace(raw.SchemaVersion) != priceCatalogSchemaV1 {
		return catalogDocument{}, fmt.Errorf("catalog %s has unsupported schema_version %q", sourceName, raw.SchemaVersion)
	}
	if strings.TrimSpace(raw.CatalogVersion) == "" {
		return catalogDocument{}, fmt.Errorf("catalog %s missing catalog_version", sourceName)
	}
	if strings.TrimSpace(raw.Source) == "" {
		return catalogDocument{}, fmt.Errorf("catalog %s missing source", sourceName)
	}

	provider, err := normalizeOptionalProvider(raw.Provider)
	if err != nil {
		return catalogDocument{}, fmt.Errorf("catalog %s provider: %w", sourceName, err)
	}

	entries := make([]catalogEntry, 0, len(raw.Entries))
	for i, rawEntry := range raw.Entries {
		entry, err := newCatalogEntry(rawEntry, provider)
		if err != nil {
			return catalogDocument{}, fmt.Errorf("catalog %s entry %d: %w", sourceName, i, err)
		}
		entries = append(entries, entry)
	}

	syncedAt, err := parseOptionalTime(raw.SyncedAt, "synced_at")
	if err != nil {
		return catalogDocument{}, fmt.Errorf("catalog %s: %w", sourceName, err)
	}

	return catalogDocument{
		SchemaVersion:  priceCatalogSchemaV1,
		CatalogVersion: strings.TrimSpace(raw.CatalogVersion),
		Source:         strings.TrimSpace(raw.Source),
		Provider:       provider,
		Entries:        entries,
		SyncedAt:       syncedAt,
	}, nil
}

func decodeRawCatalogDocument(data []byte) (rawCatalogDocument, error) {
	var raw rawCatalogDocument
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return rawCatalogDocument{}, fmt.Errorf("catalog payload is empty")
	}

	if trimmed[0] == '{' || trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &raw); err != nil {
			return rawCatalogDocument{}, err
		}
		return raw, nil
	}

	parsed, err := parseYAMLDocument(trimmed)
	if err != nil {
		return rawCatalogDocument{}, err
	}

	marshaled, err := json.Marshal(parsed)
	if err != nil {
		return rawCatalogDocument{}, err
	}
	if err := json.Unmarshal(marshaled, &raw); err != nil {
		return rawCatalogDocument{}, err
	}

	return raw, nil
}

func newCatalogEntry(raw rawCatalogEntry, defaultProvider domain.ProviderName) (catalogEntry, error) {
	provider, err := normalizeEntryProvider(raw.Provider, defaultProvider)
	if err != nil {
		return catalogEntry{}, err
	}

	modelID := strings.TrimSpace(raw.ModelID)
	if modelID == "" {
		return catalogEntry{}, fmt.Errorf("model_id is required")
	}

	lookupKey := strings.TrimSpace(raw.LookupKey)
	if lookupKey == "" {
		lookupKey = modelID
	}

	inputPrice, err := requiredPrice(raw.InputUSDPer1M, "input_usd_per_1m")
	if err != nil {
		return catalogEntry{}, err
	}
	outputPrice, err := requiredPrice(raw.OutputUSDPer1M, "output_usd_per_1m")
	if err != nil {
		return catalogEntry{}, err
	}

	cacheReadPrice, err := optionalPrice(raw.CacheReadUSDPer1M, "cache_read_usd_per_1m")
	if err != nil {
		return catalogEntry{}, err
	}
	cacheWritePrice, err := optionalPrice(raw.CacheWriteUSDPer1M, "cache_write_usd_per_1m")
	if err != nil {
		return catalogEntry{}, err
	}
	toolPrice, err := optionalPrice(raw.ToolUSDPerInvocation, "tool_usd_per_invocation")
	if err != nil {
		return catalogEntry{}, err
	}

	cachedAt, err := parseOptionalTime(raw.CachedAt, "cached_at")
	if err != nil {
		return catalogEntry{}, err
	}
	expiresAt, err := parseOptionalTime(raw.ExpiresAt, "expires_at")
	if err != nil {
		return catalogEntry{}, err
	}
	if !cachedAt.IsZero() && !expiresAt.IsZero() && expiresAt.Before(cachedAt) {
		return catalogEntry{}, fmt.Errorf("expires_at must be at or after cached_at")
	}

	price := ports.ModelPrice{
		Provider:             provider,
		ModelID:              modelID,
		LookupKey:            lookupKey,
		InputUSDPer1M:        inputPrice,
		OutputUSDPer1M:       outputPrice,
		CacheReadUSDPer1M:    cacheReadPrice,
		CacheWriteUSDPer1M:   cacheWritePrice,
		ToolUSDPerInvocation: toolPrice,
	}

	if _, err := price.Calculate(domain.TokenUsage{}, 0); err != nil {
		return catalogEntry{}, err
	}

	return catalogEntry{
		ModelPrice: price,
		CachedAt:   cachedAt,
		ExpiresAt:  expiresAt,
	}, nil
}

func normalizeOptionalProvider(raw string) (domain.ProviderName, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	return domain.NewProviderName(raw)
}

func normalizeEntryProvider(raw string, defaultProvider domain.ProviderName) (domain.ProviderName, error) {
	if strings.TrimSpace(raw) == "" {
		if defaultProvider == "" {
			return "", fmt.Errorf("provider is required")
		}
		return defaultProvider, nil
	}
	return domain.NewProviderName(raw)
}

func requiredPrice(value *float64, field string) (float64, error) {
	if value == nil {
		return 0, fmt.Errorf("%s is required", field)
	}
	return optionalPrice(value, field)
}

func optionalPrice(value *float64, field string) (float64, error) {
	if value == nil {
		return 0, nil
	}
	if *value < 0 {
		return 0, fmt.Errorf("%s must be non-negative", field)
	}
	return *value, nil
}

func parseOptionalTime(raw, field string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be RFC3339: %w", field, err)
	}
	return parsed.UTC(), nil
}

func parseYAMLDocument(data []byte) (map[string]any, error) {
	lines := strings.Split(string(data), "\n")
	root := map[string]any{}
	entries := make([]map[string]any, 0)
	inEntries := false
	var current map[string]any

	flushCurrent := func() {
		if current != nil {
			entries = append(entries, current)
			current = nil
		}
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(stripInlineComment(line))
		if trimmed == "" {
			continue
		}

		if !inEntries {
			if trimmed == "entries:" {
				inEntries = true
				continue
			}
			key, value, err := splitYAMLKeyValue(trimmed)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", i+1, err)
			}
			root[key] = parseScalarValue(value)
			continue
		}

		if strings.HasPrefix(trimmed, "- ") {
			flushCurrent()
			current = map[string]any{}
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			if rest == "" {
				continue
			}
			key, value, err := splitYAMLKeyValue(rest)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", i+1, err)
			}
			current[key] = parseScalarValue(value)
			continue
		}

		if current == nil {
			return nil, fmt.Errorf("line %d: expected list entry beginning with '- '", i+1)
		}

		key, value, err := splitYAMLKeyValue(trimmed)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		current[key] = parseScalarValue(value)
	}

	flushCurrent()
	if inEntries {
		root["entries"] = entries
	}

	return root, nil
}

func splitYAMLKeyValue(line string) (string, string, error) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("expected key: value pair")
	}
	key := strings.TrimSpace(parts[0])
	if key == "" {
		return "", "", fmt.Errorf("missing key before ':'")
	}
	return key, strings.TrimSpace(parts[1]), nil
}

func parseScalarValue(raw string) any {
	value := strings.TrimSpace(raw)
	value = strings.Trim(value, `"'`)
	if value == "" {
		return ""
	}
	if number, err := strconv.ParseFloat(value, 64); err == nil {
		return number
	}
	return value
}

func stripInlineComment(line string) string {
	inSingleQuote := false
	inDoubleQuote := false
	for i, r := range line {
		switch r {
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
		case '#':
			if !inSingleQuote && !inDoubleQuote {
				return line[:i]
			}
		}
	}
	return line
}
