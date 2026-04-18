package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsBootstrapBackfillsNewDefaultFieldsIntoExistingFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := Paths{
		ConfigDir:          filepath.Join(tempDir, "config", AppDirectoryName),
		DataDir:            filepath.Join(tempDir, "data", AppDirectoryName),
		SettingsFile:       filepath.Join(tempDir, "config", AppDirectoryName, SettingsFileName),
		PricesOverrideFile: filepath.Join(tempDir, "config", AppDirectoryName, PricesOverrideFileName),
		DatabaseFile:       filepath.Join(tempDir, "data", AppDirectoryName, DatabaseFileName),
	}
	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	legacySettings := `{
  "schema_version": 1,
  "providers": {
    "anthropic": {"enabled": true},
    "openai": {"enabled": true},
    "gemini": {"enabled": true},
    "openrouter": {"enabled": true}
  },
  "cli_billing_defaults": {
    "claude_code": "subscription",
    "codex": "subscription",
    "gemini_cli": "subscription",
    "opencode": "byok"
  },
  "budgets": {
    "monthly_budget_usd": 250,
    "monthly_subscription_budget_usd": 100,
    "monthly_usage_budget_usd": 150,
    "warning_threshold_percent": 80,
    "critical_threshold_percent": 100
  },
  "notifications": {
    "desktop_enabled": true,
    "tui_enabled": true,
    "budget_warnings": true,
    "forecast_warnings": true,
    "provider_sync_failure": true
  }
}
`
	if err := os.WriteFile(paths.SettingsFile, []byte(legacySettings), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := NewSettingsStore(paths)
	settings, err := store.Bootstrap()
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if got := settings.SubscriptionDefaults.OpenAI.PlanCode; got != "chatgpt-plus" {
		t.Fatalf("OpenAI plan code = %q, want chatgpt-plus", got)
	}

	raw, err := os.ReadFile(paths.SettingsFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "subscription_defaults") {
		t.Fatalf("settings file content = %s, want subscription_defaults section", content)
	}
	if !strings.Contains(content, "chatgpt-plus") {
		t.Fatalf("settings file content = %s, want openai subscription defaults persisted", content)
	}
}
