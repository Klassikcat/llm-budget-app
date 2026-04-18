package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func (s *Store) WithTx(ctx context.Context, opts *sql.TxOptions, fn func(*sql.Tx) error) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite store is not initialized")
	}
	if fn == nil {
		return fmt.Errorf("transaction callback is required")
	}

	var lastErr error
	for attempt := 1; attempt <= s.retry.MaxAttempts; attempt++ {
		tx, err := s.db.BeginTx(ctx, opts)
		if err != nil {
			if retryable, waitErr := s.maybeRetry(ctx, attempt, err); waitErr != nil {
				return waitErr
			} else if retryable {
				lastErr = err
				continue
			}

			return err
		}

		err = fn(tx)
		if err != nil {
			_ = tx.Rollback()
			if retryable, waitErr := s.maybeRetry(ctx, attempt, err); waitErr != nil {
				return waitErr
			} else if retryable {
				lastErr = err
				continue
			}

			return err
		}

		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			if retryable, waitErr := s.maybeRetry(ctx, attempt, err); waitErr != nil {
				return waitErr
			} else if retryable {
				lastErr = err
				continue
			}

			return err
		}

		return nil
	}

	return &BusyTimeoutError{Attempts: s.retry.MaxAttempts, LastErr: lastErr}
}

func (s *Store) maybeRetry(ctx context.Context, attempt int, err error) (bool, error) {
	if !isBusyError(err) {
		return false, nil
	}
	if attempt >= s.retry.MaxAttempts {
		return false, &BusyTimeoutError{Attempts: attempt, LastErr: err}
	}

	timer := time.NewTimer(s.retry.Backoff)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-timer.C:
		return true, nil
	}
}
