package config

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	ErrorCodeConfigDirUnavailable ErrorCode = "config_dir_unavailable"
	ErrorCodeSettingsIO           ErrorCode = "settings_io"
	ErrorCodeSettingsDecode       ErrorCode = "settings_decode"
	ErrorCodeKeyringUnavailable   ErrorCode = "keyring_unavailable"
)

type SetupError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *SetupError) Error() string {
	if e == nil {
		return "<nil>"
	}

	if e.Err == nil {
		return e.Message
	}

	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e *SetupError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

func IsSetupErrorCode(err error, code ErrorCode) bool {
	var setupErr *SetupError
	if !errors.As(err, &setupErr) {
		return false
	}

	return setupErr.Code == code
}
