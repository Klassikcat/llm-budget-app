package config

import (
	"errors"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestKeyringUnavailable(t *testing.T) {
	backend := failingKeyringBackend{err: errors.New("dbus secret service unavailable")}

	store, err := NewKeyringSecretStore("test-service", backend)
	if err != nil {
		t.Fatalf("NewKeyringSecretStore() error = %v", err)
	}

	err = store.Set(SecretOpenAIAPIKey, "secret")
	if err == nil {
		t.Fatal("Set() error = nil, want error")
	}
	if !IsSetupErrorCode(err, ErrorCodeKeyringUnavailable) {
		t.Fatalf("Set() error = %v, want keyring_unavailable", err)
	}

	var setupErr *SetupError
	if !errors.As(err, &setupErr) {
		t.Fatalf("Set() error = %v, want SetupError", err)
	}
	if setupErr.Message == "" {
		t.Fatal("SetupError message is empty")
	}
}

func TestKeyringNotFoundPassesThrough(t *testing.T) {
	store, err := NewKeyringSecretStore("test-service", failingKeyringBackend{err: keyring.ErrNotFound})
	if err != nil {
		t.Fatalf("NewKeyringSecretStore() error = %v", err)
	}

	_, err = store.Get(SecretAnthropicAPIKey)
	if !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
	if IsSetupErrorCode(err, ErrorCodeKeyringUnavailable) {
		t.Fatalf("Get() error = %v, did not want keyring_unavailable", err)
	}
}

type memoryKeyringBackend struct {
	values map[string]string
}

func (m *memoryKeyringBackend) Set(service, user, secret string) error {
	if m.values == nil {
		m.values = map[string]string{}
	}
	_ = service
	m.values[user] = secret
	return nil
}

func (m *memoryKeyringBackend) Get(service, user string) (string, error) {
	_ = service
	value, ok := m.values[user]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return value, nil
}

func (m *memoryKeyringBackend) Delete(service, user string) error {
	_ = service
	delete(m.values, user)
	return nil
}

type failingKeyringBackend struct {
	err error
}

func (f failingKeyringBackend) Set(service, user, secret string) error {
	_, _, _ = service, user, secret
	return f.err
}

func (f failingKeyringBackend) Get(service, user string) (string, error) {
	_, _ = service, user
	return "", f.err
}

func (f failingKeyringBackend) Delete(service, user string) error {
	_, _ = service, user
	return f.err
}
