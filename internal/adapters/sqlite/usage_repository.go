package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

var _ ports.UsageEntryRepository = (*Store)(nil)

func (s *Store) UpsertUsageEntries(ctx context.Context, entries []domain.UsageEntry) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite store is not initialized")
	}
	if len(entries) == 0 {
		return nil
	}

	return s.WithTx(ctx, nil, func(tx *sql.Tx) error {
		for _, entry := range entries {
			validated, err := domain.NewUsageEntry(entry)
			if err != nil {
				return err
			}

			metadataJSON, err := marshalUsageMetadata(validated.Metadata)
			if err != nil {
				return err
			}

			now := time.Now().UTC().Format(time.RFC3339Nano)
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO usage_entries (
					entry_id,
					session_key,
					provider,
					source_type,
					billing_mode,
					recorded_at,
					external_id,
					project_name,
					agent_name,
					model_name,
					pricing_lookup_key,
					input_tokens,
					output_tokens,
					cache_creation_tokens,
					cache_read_tokens,
					input_cost_usd,
					output_cost_usd,
					cache_creation_cost_usd,
					cache_read_cost_usd,
					tool_cost_usd,
					flat_cost_usd,
					cost_usd,
					metadata_json,
					currency,
					created_at,
					updated_at
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'USD', ?, ?)
				ON CONFLICT(entry_id) DO UPDATE SET
					session_key=excluded.session_key,
					provider=excluded.provider,
					source_type=excluded.source_type,
					billing_mode=excluded.billing_mode,
					recorded_at=excluded.recorded_at,
					external_id=excluded.external_id,
					project_name=excluded.project_name,
					agent_name=excluded.agent_name,
					model_name=excluded.model_name,
					pricing_lookup_key=excluded.pricing_lookup_key,
					input_tokens=excluded.input_tokens,
					output_tokens=excluded.output_tokens,
					cache_creation_tokens=excluded.cache_creation_tokens,
					cache_read_tokens=excluded.cache_read_tokens,
					input_cost_usd=excluded.input_cost_usd,
					output_cost_usd=excluded.output_cost_usd,
					cache_creation_cost_usd=excluded.cache_creation_cost_usd,
					cache_read_cost_usd=excluded.cache_read_cost_usd,
					tool_cost_usd=excluded.tool_cost_usd,
					flat_cost_usd=excluded.flat_cost_usd,
					cost_usd=excluded.cost_usd,
					metadata_json=excluded.metadata_json,
					updated_at=excluded.updated_at
			`,
				validated.EntryID,
				nullIfBlank(validated.SessionID),
				validated.Provider.String(),
				string(validated.Source),
				string(validated.BillingMode),
				validated.OccurredAt.Format(time.RFC3339Nano),
				nullIfBlank(validated.ExternalID),
				nullIfBlank(validated.ProjectName),
				nullIfBlank(validated.AgentName),
				nullIfBlank(modelID(validated.PricingRef)),
				nullIfBlank(pricingLookupKey(validated.PricingRef)),
				validated.Tokens.InputTokens,
				validated.Tokens.OutputTokens,
				validated.Tokens.CacheWriteTokens,
				validated.Tokens.CacheReadTokens,
				validated.CostBreakdown.InputUSD,
				validated.CostBreakdown.OutputUSD,
				validated.CostBreakdown.CacheWriteUSD,
				validated.CostBreakdown.CacheReadUSD,
				validated.CostBreakdown.ToolUSD,
				validated.CostBreakdown.FlatUSD,
				validated.CostBreakdown.TotalUSD,
				metadataJSON,
				now,
				now,
			); err != nil {
				return fmt.Errorf("upsert usage entry %s: %w", validated.EntryID, err)
			}
		}

		return nil
	})
}

func (s *Store) ListUsageEntries(ctx context.Context, filter ports.UsageFilter) ([]domain.UsageEntry, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("sqlite store is not initialized")
	}

	query := `
		SELECT
			entry_id,
			session_key,
			provider,
			source_type,
			billing_mode,
			recorded_at,
			external_id,
			project_name,
			agent_name,
			model_name,
			pricing_lookup_key,
			input_tokens,
			output_tokens,
			cache_creation_tokens,
			cache_read_tokens,
			input_cost_usd,
			output_cost_usd,
			cache_creation_cost_usd,
			cache_read_cost_usd,
			tool_cost_usd,
			flat_cost_usd,
			cost_usd,
			metadata_json
		FROM usage_entries
		WHERE 1=1`
	args := make([]any, 0, 8)

	if filter.Period != nil {
		query += ` AND recorded_at >= ? AND recorded_at < ?`
		args = append(args, filter.Period.StartAt.Format(time.RFC3339Nano), filter.Period.EndExclusive.Format(time.RFC3339Nano))
	}
	if filter.Provider != "" {
		query += ` AND provider = ?`
		args = append(args, filter.Provider.String())
	}
	if strings.TrimSpace(filter.Project) != "" {
		query += ` AND project_name = ?`
		args = append(args, strings.TrimSpace(filter.Project))
	}
	if strings.TrimSpace(filter.Agent) != "" {
		query += ` AND agent_name = ?`
		args = append(args, strings.TrimSpace(filter.Agent))
	}
	if strings.TrimSpace(filter.SessionID) != "" {
		query += ` AND session_key = ?`
		args = append(args, strings.TrimSpace(filter.SessionID))
	}

	query += ` ORDER BY recorded_at ASC, entry_id ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list usage entries: %w", err)
	}
	defer rows.Close()

	entries := make([]domain.UsageEntry, 0)
	for rows.Next() {
		entry, err := scanUsageEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate usage entries: %w", err)
	}

	return entries, nil
}

func scanUsageEntry(scanner interface{ Scan(dest ...any) error }) (domain.UsageEntry, error) {
	var (
		entryID           string
		sessionKey        sql.NullString
		providerRaw       string
		sourceRaw         string
		billingModeRaw    string
		recordedAtRaw     string
		externalID        sql.NullString
		projectName       sql.NullString
		agentName         sql.NullString
		modelName         sql.NullString
		pricingLookupKey  sql.NullString
		inputTokens       int64
		outputTokens      int64
		cacheWriteTokens  int64
		cacheReadTokens   int64
		inputCostUSD      float64
		outputCostUSD     float64
		cacheWriteCostUSD float64
		cacheReadCostUSD  float64
		toolCostUSD       float64
		flatCostUSD       float64
		totalCostUSD      float64
		metadataJSON      sql.NullString
	)

	if err := scanner.Scan(
		&entryID,
		&sessionKey,
		&providerRaw,
		&sourceRaw,
		&billingModeRaw,
		&recordedAtRaw,
		&externalID,
		&projectName,
		&agentName,
		&modelName,
		&pricingLookupKey,
		&inputTokens,
		&outputTokens,
		&cacheWriteTokens,
		&cacheReadTokens,
		&inputCostUSD,
		&outputCostUSD,
		&cacheWriteCostUSD,
		&cacheReadCostUSD,
		&toolCostUSD,
		&flatCostUSD,
		&totalCostUSD,
		&metadataJSON,
	); err != nil {
		return domain.UsageEntry{}, fmt.Errorf("scan usage entry: %w", err)
	}

	provider, err := domain.NewProviderName(providerRaw)
	if err != nil {
		return domain.UsageEntry{}, err
	}
	source, err := domain.ParseUsageSourceKind(sourceRaw)
	if err != nil {
		return domain.UsageEntry{}, err
	}
	billingMode, err := domain.ParseBillingMode(billingModeRaw)
	if err != nil {
		return domain.UsageEntry{}, err
	}
	occurredAt, err := time.Parse(time.RFC3339Nano, recordedAtRaw)
	if err != nil {
		return domain.UsageEntry{}, fmt.Errorf("parse usage entry time: %w", err)
	}
	tokens, err := domain.NewTokenUsage(inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens)
	if err != nil {
		return domain.UsageEntry{}, err
	}
	costBreakdown, err := domain.NewCostBreakdown(inputCostUSD, outputCostUSD, cacheReadCostUSD, cacheWriteCostUSD, toolCostUSD, flatCostUSD)
	if err != nil {
		return domain.UsageEntry{}, err
	}
	if totalCostUSD != costBreakdown.TotalUSD {
		costBreakdown.TotalUSD = totalCostUSD
	}
	metadata, err := unmarshalUsageMetadata(metadataJSON)
	if err != nil {
		return domain.UsageEntry{}, err
	}

	var pricingRef *domain.ModelPricingRef
	if modelName.Valid {
		ref, err := domain.NewModelPricingRef(provider, modelName.String, pricingLookupKey.String)
		if err != nil {
			return domain.UsageEntry{}, err
		}
		pricingRef = &ref
	}

	return domain.NewUsageEntry(domain.UsageEntry{
		EntryID:       entryID,
		SessionID:     sessionKey.String,
		Source:        source,
		Provider:      provider,
		BillingMode:   billingMode,
		OccurredAt:    occurredAt,
		ExternalID:    externalID.String,
		ProjectName:   projectName.String,
		AgentName:     agentName.String,
		Metadata:      metadata,
		PricingRef:    pricingRef,
		Tokens:        tokens,
		CostBreakdown: costBreakdown,
	})
}

func marshalUsageMetadata(metadata map[string]string) (any, error) {
	if len(metadata) == 0 {
		return nil, nil
	}

	encoded, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal usage metadata: %w", err)
	}

	return string(encoded), nil
}

func unmarshalUsageMetadata(raw sql.NullString) (map[string]string, error) {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil, nil
	}

	metadata := make(map[string]string)
	if err := json.Unmarshal([]byte(raw.String), &metadata); err != nil {
		return nil, fmt.Errorf("unmarshal usage metadata: %w", err)
	}

	return metadata, nil
}

func modelID(ref *domain.ModelPricingRef) string {
	if ref == nil {
		return ""
	}

	return ref.ModelID
}

func pricingLookupKey(ref *domain.ModelPricingRef) string {
	if ref == nil {
		return ""
	}

	return ref.PricingLookupKey
}

func nullIfBlank(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return trimmed
}
