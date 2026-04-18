package domain

import "time"

func NormalizeUTCTimestamp(field string, value time.Time) (time.Time, error) {
	if value.IsZero() {
		return time.Time{}, &ValidationError{
			Code:    ValidationCodeInvalidTimestamp,
			Field:   field,
			Message: "timestamp must be set",
		}
	}

	return value.UTC(), nil
}
