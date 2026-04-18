CREATE TABLE IF NOT EXISTS pricing_catalog_cache (
    provider TEXT NOT NULL,
    model_key TEXT NOT NULL,
    price_version TEXT,
    currency TEXT NOT NULL DEFAULT 'USD',
    input_per_million_usd REAL NOT NULL DEFAULT 0,
    output_per_million_usd REAL NOT NULL DEFAULT 0,
    cached_at TEXT NOT NULL,
    expires_at TEXT,
    source TEXT NOT NULL,
    PRIMARY KEY (provider, model_key)
);

CREATE TABLE IF NOT EXISTS subscriptions (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    plan_code TEXT NOT NULL,
    plan_name TEXT NOT NULL,
    renewal_day INTEGER NOT NULL,
    amount_usd REAL NOT NULL,
    starts_at TEXT NOT NULL,
    ends_at TEXT,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_provider_active
    ON subscriptions (provider, is_active);

CREATE TABLE IF NOT EXISTS subscription_fees (
    subscription_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    plan_code TEXT NOT NULL,
    charged_at TEXT NOT NULL,
    period_start_at TEXT NOT NULL,
    period_end_exclusive TEXT NOT NULL,
    fee_usd REAL NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (subscription_id, period_start_at),
    FOREIGN KEY (subscription_id) REFERENCES subscriptions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_subscription_fees_period_start
    ON subscription_fees (period_start_at);

CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    source_type TEXT NOT NULL,
    provider TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    billing_mode TEXT NOT NULL,
    project_name TEXT,
    model_name TEXT,
    pricing_lookup_key TEXT,
    started_at TEXT NOT NULL,
    ended_at TEXT NOT NULL,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    input_cost_usd REAL NOT NULL DEFAULT 0,
    output_cost_usd REAL NOT NULL DEFAULT 0,
    cache_creation_cost_usd REAL NOT NULL DEFAULT 0,
    cache_read_cost_usd REAL NOT NULL DEFAULT 0,
    tool_cost_usd REAL NOT NULL DEFAULT 0,
    flat_cost_usd REAL NOT NULL DEFAULT 0,
    total_cost_usd REAL NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (session_id)
);

CREATE INDEX IF NOT EXISTS idx_sessions_started_at
    ON sessions (started_at);

CREATE TABLE IF NOT EXISTS usage_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id TEXT NOT NULL,
    session_key TEXT,
    provider TEXT NOT NULL,
    source_type TEXT NOT NULL,
    billing_mode TEXT NOT NULL,
    recorded_at TEXT NOT NULL,
    external_id TEXT,
    project_name TEXT,
    agent_name TEXT,
    model_name TEXT,
    pricing_lookup_key TEXT,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    input_cost_usd REAL NOT NULL DEFAULT 0,
    output_cost_usd REAL NOT NULL DEFAULT 0,
    cache_creation_cost_usd REAL NOT NULL DEFAULT 0,
    cache_read_cost_usd REAL NOT NULL DEFAULT 0,
    tool_cost_usd REAL NOT NULL DEFAULT 0,
    flat_cost_usd REAL NOT NULL DEFAULT 0,
    cost_usd REAL NOT NULL DEFAULT 0,
    metadata_json TEXT,
    currency TEXT NOT NULL DEFAULT 'USD',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (entry_id),
    FOREIGN KEY (session_key) REFERENCES sessions(session_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_usage_entries_recorded_at
    ON usage_entries (recorded_at);

CREATE INDEX IF NOT EXISTS idx_usage_entries_session_id
    ON usage_entries (session_key);

CREATE TABLE IF NOT EXISTS insights (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER,
    rule_key TEXT NOT NULL,
    severity TEXT NOT NULL,
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    detected_at TEXT NOT NULL,
    estimated_waste_usd REAL NOT NULL DEFAULT 0,
    estimated_waste_tokens INTEGER NOT NULL DEFAULT 0,
    dismissed_at TEXT,
    created_at TEXT NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_insights_detected_at
    ON insights (detected_at);

CREATE TABLE IF NOT EXISTS watcher_offsets (
    watcher_key TEXT PRIMARY KEY,
    source_path TEXT NOT NULL,
    file_identity TEXT,
    byte_offset INTEGER NOT NULL DEFAULT 0,
    last_marker TEXT,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_watcher_offsets_source_path
    ON watcher_offsets (source_path);

CREATE TABLE IF NOT EXISTS settings_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    schema_version INTEGER NOT NULL,
    captured_at TEXT NOT NULL,
    provider_anthropic_enabled INTEGER NOT NULL,
    provider_openai_enabled INTEGER NOT NULL,
    provider_gemini_enabled INTEGER NOT NULL,
    provider_openrouter_enabled INTEGER NOT NULL,
    default_claude_code_billing_mode TEXT NOT NULL,
    default_codex_billing_mode TEXT NOT NULL,
    default_gemini_cli_billing_mode TEXT NOT NULL,
    default_opencode_billing_mode TEXT NOT NULL,
    monthly_budget_usd REAL NOT NULL,
    monthly_subscription_budget_usd REAL NOT NULL,
    monthly_usage_budget_usd REAL NOT NULL,
    warning_threshold_percent INTEGER NOT NULL,
    critical_threshold_percent INTEGER NOT NULL,
    notifications_desktop_enabled INTEGER NOT NULL,
    notifications_tui_enabled INTEGER NOT NULL,
    notifications_budget_warnings INTEGER NOT NULL,
    notifications_forecast_warnings INTEGER NOT NULL,
    notifications_provider_sync_failure INTEGER NOT NULL
);
