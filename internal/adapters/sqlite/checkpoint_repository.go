package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"llm-budget-tracker/internal/ports"
)

var _ ports.CheckpointRepository = (*Store)(nil)

func (s *Store) LoadCheckpoint(ctx context.Context, sourceID string) (ports.IngestionCheckpoint, error) {
	if s == nil || s.db == nil {
		return ports.IngestionCheckpoint{}, fmt.Errorf("sqlite store is not initialized")
	}

	var (
		checkpoint ports.IngestionCheckpoint
		updatedAt  sql.NullString
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT watcher_key, source_path, file_identity, last_marker, byte_offset, updated_at
		FROM watcher_offsets
		WHERE watcher_key = ?
	`, sourceID).Scan(
		&checkpoint.SourceID,
		&checkpoint.Path,
		&checkpoint.FileIdentity,
		&checkpoint.LastMarker,
		&checkpoint.Offset,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return ports.IngestionCheckpoint{SourceID: sourceID}, nil
		}
		return ports.IngestionCheckpoint{}, err
	}

	if updatedAt.Valid {
		parsed, err := time.Parse(time.RFC3339Nano, updatedAt.String)
		if err != nil {
			return ports.IngestionCheckpoint{}, fmt.Errorf("parse checkpoint updated_at: %w", err)
		}
		checkpoint.UpdatedAt = parsed.UTC()
	}

	return checkpoint, nil
}

func (s *Store) SaveCheckpoint(ctx context.Context, checkpoint ports.IngestionCheckpoint) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite store is not initialized")
	}
	updatedAt := checkpoint.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO watcher_offsets (watcher_key, source_path, file_identity, byte_offset, last_marker, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(watcher_key) DO UPDATE SET
			source_path = excluded.source_path,
			file_identity = excluded.file_identity,
			byte_offset = excluded.byte_offset,
			last_marker = excluded.last_marker,
			updated_at = excluded.updated_at
	`, checkpoint.SourceID, checkpoint.Path, checkpoint.FileIdentity, checkpoint.Offset, checkpoint.LastMarker, updatedAt.Format(time.RFC3339Nano))
	return err
}
