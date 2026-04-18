package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

type fileSystem interface {
	MkdirAll(path string, perm fs.FileMode) error
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm fs.FileMode) error
}

type osFileSystem struct{}

func (osFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (osFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (osFileSystem) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}

type SettingsStore struct {
	paths    Paths
	defaults Settings
	fs       fileSystem
}

func NewSettingsStore(paths Paths) *SettingsStore {
	return &SettingsStore{
		paths:    paths,
		defaults: DefaultSettings(),
		fs:       osFileSystem{},
	}
}

func (s *SettingsStore) Bootstrap() (Settings, error) {
	if s == nil {
		return Settings{}, &SetupError{
			Code:    ErrorCodeSettingsIO,
			Message: "settings store is not initialized",
		}
	}

	if err := s.ensureConfigDir(); err != nil {
		return Settings{}, err
	}

	raw, err := s.fs.ReadFile(s.paths.SettingsFile)
	if err == nil {
		settings, err := s.decode(raw)
		if err != nil {
			return Settings{}, err
		}
		if err := s.persistMergedSettingsIfNeeded(settings, raw); err != nil {
			return Settings{}, err
		}
		return settings, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return Settings{}, &SetupError{
			Code:    ErrorCodeSettingsIO,
			Message: fmt.Sprintf("failed to read settings file %q", s.paths.SettingsFile),
			Err:     err,
		}
	}

	settings := s.defaults
	if err := s.Save(settings); err != nil {
		return Settings{}, err
	}

	return settings, nil
}

func (s *SettingsStore) Load() (Settings, error) {
	if s == nil {
		return Settings{}, &SetupError{
			Code:    ErrorCodeSettingsIO,
			Message: "settings store is not initialized",
		}
	}

	raw, err := s.fs.ReadFile(s.paths.SettingsFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Settings{}, fs.ErrNotExist
		}

		return Settings{}, &SetupError{
			Code:    ErrorCodeSettingsIO,
			Message: fmt.Sprintf("failed to read settings file %q", s.paths.SettingsFile),
			Err:     err,
		}
	}

	return s.decode(raw)
}

func (s *SettingsStore) Save(settings Settings) error {
	if s == nil {
		return &SetupError{
			Code:    ErrorCodeSettingsIO,
			Message: "settings store is not initialized",
		}
	}

	if err := s.ensureConfigDir(); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return &SetupError{
			Code:    ErrorCodeSettingsDecode,
			Message: "failed to encode settings",
			Err:     err,
		}
	}
	raw = append(raw, '\n')

	if err := s.fs.WriteFile(s.paths.SettingsFile, raw, 0o600); err != nil {
		return &SetupError{
			Code:    ErrorCodeSettingsIO,
			Message: fmt.Sprintf("failed to write settings file %q", s.paths.SettingsFile),
			Err:     err,
		}
	}

	return nil
}

func (s *SettingsStore) ensureConfigDir() error {
	if err := s.fs.MkdirAll(s.paths.ConfigDir, 0o755); err != nil {
		return &SetupError{
			Code:    ErrorCodeSettingsIO,
			Message: fmt.Sprintf("failed to create config directory %q", s.paths.ConfigDir),
			Err:     err,
		}
	}

	return nil
}

func (s *SettingsStore) decode(raw []byte) (Settings, error) {
	settings := s.defaults
	if err := json.Unmarshal(raw, &settings); err != nil {
		return Settings{}, &SetupError{
			Code:    ErrorCodeSettingsDecode,
			Message: fmt.Sprintf("failed to parse settings file %q", s.paths.SettingsFile),
			Err:     err,
		}
	}

	return settings, nil
}

func (s *SettingsStore) persistMergedSettingsIfNeeded(settings Settings, existing []byte) error {
	encoded, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return &SetupError{
			Code:    ErrorCodeSettingsDecode,
			Message: "failed to encode settings",
			Err:     err,
		}
	}
	encoded = append(encoded, '\n')
	if bytes.Equal(existing, encoded) {
		return nil
	}

	return s.Save(settings)
}
