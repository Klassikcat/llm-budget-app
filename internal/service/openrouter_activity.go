package service

import (
	"context"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

type OpenRouterActivitySyncResult struct {
	UsageEntries []domain.UsageEntry
}

type OpenRouterActivitySyncService struct {
	source    ports.OpenRouterActivitySource
	ingestion ports.IngestionService
}

func NewOpenRouterActivitySyncService(source ports.OpenRouterActivitySource, ingestion ports.IngestionService) *OpenRouterActivitySyncService {
	return &OpenRouterActivitySyncService{source: source, ingestion: ingestion}
}

func (s *OpenRouterActivitySyncService) Sync(ctx context.Context, options ports.OpenRouterActivityOptions) (OpenRouterActivitySyncResult, error) {
	if s == nil || s.source == nil {
		return OpenRouterActivitySyncResult{}, errOpenRouterActivitySourceRequired
	}

	if s.ingestion == nil {
		return OpenRouterActivitySyncResult{}, errIngestionServiceRequired
	}

	entries, err := s.source.FetchUsageEntries(ctx, options)
	if err != nil {
		return OpenRouterActivitySyncResult{}, err
	}

	if len(entries) == 0 {
		return OpenRouterActivitySyncResult{}, nil
	}

	if err := s.ingestion.IngestUsageEntries(ctx, entries); err != nil {
		return OpenRouterActivitySyncResult{}, err
	}

	return OpenRouterActivitySyncResult{UsageEntries: entries}, nil
}
