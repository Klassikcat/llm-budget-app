package service

import (
	"context"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

type OpenRouterActivitySyncResult struct {
	UsageEntries []domain.UsageEntry
}

const OpenRouterActivityAutoSyncCheckpointID = "openrouter:activity:auto-sync"

const OpenRouterActivityAutoSyncInterval = 24 * time.Hour

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

func (s *OpenRouterActivitySyncService) AutoSync(ctx context.Context, checkpoints ports.CheckpointRepository, now time.Time) (OpenRouterActivitySyncResult, bool, error) {
	if checkpoints == nil {
		return OpenRouterActivitySyncResult{}, false, errCheckpointRepositoryRequired
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()

	checkpoint, err := checkpoints.LoadCheckpoint(ctx, OpenRouterActivityAutoSyncCheckpointID)
	if err != nil {
		return OpenRouterActivitySyncResult{}, false, err
	}
	if !checkpoint.UpdatedAt.IsZero() && now.Sub(checkpoint.UpdatedAt.UTC()) < OpenRouterActivityAutoSyncInterval {
		return OpenRouterActivitySyncResult{}, false, nil
	}

	result, err := s.Sync(ctx, ports.OpenRouterActivityOptions{})
	if err != nil {
		return OpenRouterActivitySyncResult{}, true, err
	}

	if err := checkpoints.SaveCheckpoint(ctx, ports.IngestionCheckpoint{
		SourceID:   OpenRouterActivityAutoSyncCheckpointID,
		LastMarker: "success",
		UpdatedAt:  now,
	}); err != nil {
		return OpenRouterActivitySyncResult{}, true, err
	}

	return result, true, nil
}
