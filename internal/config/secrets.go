package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

type SecretID string

const (
	SecretAnthropicAPIKey   SecretID = "provider.anthropic.api_key"
	SecretAnthropicAdminKey SecretID = "provider.anthropic.admin_key"
	SecretOpenAIAPIKey      SecretID = "provider.openai.api_key"
	SecretOpenAIAdminKey    SecretID = "provider.openai.admin_key"
	SecretOpenRouterAPIKey  SecretID = "provider.openrouter.api_key"
)

type SecretStore interface {
	Set(SecretID, string) error
	Get(SecretID) (string, error)
	Delete(SecretID) error
}

type KeyringBackend interface {
	Set(service, user, secret string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
}

type KeyringSecretStore struct {
	service string
	backend KeyringBackend
}

func NewKeyringSecretStore(service string, backend KeyringBackend) (*KeyringSecretStore, error) {
	service = strings.TrimSpace(service)
	if service == "" {
		service = AppDirectoryName
	}

	if backend == nil {
		return nil, &SetupError{
			Code:    ErrorCodeKeyringUnavailable,
			Message: "system keyring support is not available; install or unlock an OS keyring backend to store API keys",
		}
	}

	return &KeyringSecretStore{service: service, backend: backend}, nil
}

func NewOSKeyringSecretStore() (*KeyringSecretStore, error) {
	return NewKeyringSecretStore(AppDirectoryName, osKeyringBackend{})
}

func (s *KeyringSecretStore) Set(id SecretID, secret string) error {
	if err := s.backend.Set(s.service, string(id), secret); err != nil {
		return mapKeyringError(id, "store", err)
	}

	return nil
}

func (s *KeyringSecretStore) Get(id SecretID) (string, error) {
	secret, err := s.backend.Get(s.service, string(id))
	if err != nil {
		return "", mapKeyringError(id, "load", err)
	}

	return secret, nil
}

func (s *KeyringSecretStore) Delete(id SecretID) error {
	if err := s.backend.Delete(s.service, string(id)); err != nil {
		return mapKeyringError(id, "delete", err)
	}

	return nil
}

type osKeyringBackend struct{}

func (osKeyringBackend) Set(service, user, secret string) error {
	return keyring.Set(service, user, secret)
}

func (osKeyringBackend) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (osKeyringBackend) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

func mapKeyringError(id SecretID, action string, err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("%s secret %q: %w", action, id, err)
	}

	if errors.Is(err, keyring.ErrSetDataTooBig) {
		return fmt.Errorf("%s secret %q: %w", action, id, err)
	}

	return &SetupError{
		Code:    ErrorCodeKeyringUnavailable,
		Message: fmt.Sprintf("system keyring is unavailable; cannot %s secret %q without an OS keyring backend", action, id),
		Err:     err,
	}
}
