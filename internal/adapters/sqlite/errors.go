package sqlite

import (
	"errors"
	"fmt"
	"strings"
)

const (
	sqliteBusyCode   = 5
	sqliteLockedCode = 6
)

type BusyTimeoutError struct {
	Attempts int
	LastErr  error
}

func (e *BusyTimeoutError) Error() string {
	if e == nil {
		return "<nil>"
	}

	if e.LastErr == nil {
		return fmt.Sprintf("sqlite busy timeout after %d attempts", e.Attempts)
	}

	return fmt.Sprintf("sqlite busy timeout after %d attempts: %v", e.Attempts, e.LastErr)
}

func (e *BusyTimeoutError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.LastErr
}

func IsBusyTimeout(err error) bool {
	var busyErr *BusyTimeoutError
	return errors.As(err, &busyErr)
}

func isBusyError(err error) bool {
	if err == nil {
		return false
	}

	type codeCarrier interface {
		Code() int
	}

	var carrier codeCarrier
	if errors.As(err, &carrier) {
		switch carrier.Code() {
		case sqliteBusyCode, sqliteLockedCode:
			return true
		}
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") || strings.Contains(message, "database is busy")
}
