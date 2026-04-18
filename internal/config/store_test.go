package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsBootstrapCreatesExpectedConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	paths := Paths{
		ConfigDir:          filepath.Join(tempDir, "config", AppDirectoryName),
		DataDir:            filepath.Join(tempDir, "data", AppDirectoryName),
		SettingsFile:       filepath.Join(tempDir, "config", AppDirectoryName, SettingsFileName),
		PricesOverrideFile: filepath.Join(tempDir, "config", AppDirectoryName, PricesOverrideFileName),
		DatabaseFile:       filepath.Join(tempDir, "data", AppDirectoryName, DatabaseFileName),
	}

	store := NewSettingsStore(paths)
	settings, err := store.Bootstrap()
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	if settings.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", settings.SchemaVersion)
	}
	if got := settings.SubscriptionDefaults.OpenAI.FeeUSD; got != 20 {
		t.Fatalf("OpenAI default subscription fee = %v, want 20", got)
	}
	if got := settings.SubscriptionDefaults.Gemini.FeeUSD; got != 19.99 {
		t.Fatalf("Gemini default subscription fee = %v, want 19.99", got)
	}

	if _, err := os.Stat(paths.SettingsFile); err != nil {
		t.Fatalf("settings file stat error = %v", err)
	}

	entries, err := os.ReadDir(paths.ConfigDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != SettingsFileName {
		t.Fatalf("config dir entries = %v, want only %q", dirNames(entries), SettingsFileName)
	}

	raw, err := os.ReadFile(paths.SettingsFile)
	if err != nil {
		t.Fatalf("ReadFile(settings) error = %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "subscription_defaults") {
		t.Fatalf("settings file content = %s, want subscription_defaults section", content)
	}
}

func TestSecretsAreNeverWrittenIntoSettingsFile(t *testing.T) {
	tempDir := t.TempDir()
	paths := Paths{
		ConfigDir:          filepath.Join(tempDir, "config", AppDirectoryName),
		DataDir:            filepath.Join(tempDir, "data", AppDirectoryName),
		SettingsFile:       filepath.Join(tempDir, "config", AppDirectoryName, SettingsFileName),
		PricesOverrideFile: filepath.Join(tempDir, "config", AppDirectoryName, PricesOverrideFileName),
		DatabaseFile:       filepath.Join(tempDir, "data", AppDirectoryName, DatabaseFileName),
	}

	store := NewSettingsStore(paths)
	if _, err := store.Bootstrap(); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	backend := &memoryKeyringBackend{}
	secrets, err := NewKeyringSecretStore("test-service", backend)
	if err != nil {
		t.Fatalf("NewKeyringSecretStore() error = %v", err)
	}

	const secretValue = "super-secret-openrouter-key"
	if err := secrets.Set(SecretOpenRouterAPIKey, secretValue); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	raw, err := os.ReadFile(paths.SettingsFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	content := string(raw)
	if strings.Contains(content, secretValue) {
		t.Fatalf("settings file unexpectedly contains secret value: %s", content)
	}
	if strings.Contains(content, string(SecretOpenRouterAPIKey)) {
		t.Fatalf("settings file unexpectedly contains secret identifier: %s", content)
	}

	if got := backend.values[string(SecretOpenRouterAPIKey)]; got != secretValue {
		t.Fatalf("keyring backend stored %q, want %q", got, secretValue)
	}
}

func dirNames(entries []os.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	return names
}
