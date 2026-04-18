CREATE TABLE IF NOT EXISTS monthly_budgets (
    budget_id TEXT NOT NULL,
    name TEXT,
    provider TEXT,
    project_hash TEXT,
    period_start_at TEXT NOT NULL,
    period_end_exclusive TEXT NOT NULL,
    limit_usd REAL NOT NULL,
    currency TEXT NOT NULL DEFAULT 'USD',
    thresholds_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (budget_id, period_start_at)
);

CREATE INDEX IF NOT EXISTS idx_monthly_budgets_period_provider
    ON monthly_budgets (period_start_at, provider);

CREATE TABLE IF NOT EXISTS budget_states (
    budget_id TEXT NOT NULL,
    period_start_at TEXT NOT NULL,
    period_end_exclusive TEXT NOT NULL,
    current_spend_usd REAL NOT NULL DEFAULT 0,
    forecast_spend_usd REAL NOT NULL DEFAULT 0,
    triggered_thresholds_json TEXT NOT NULL DEFAULT '[]',
    budget_overrun_active INTEGER NOT NULL DEFAULT 0,
    forecast_overrun_active INTEGER NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (budget_id, period_start_at),
    FOREIGN KEY (budget_id, period_start_at) REFERENCES monthly_budgets(budget_id, period_start_at) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS forecast_snapshots (
    forecast_id TEXT PRIMARY KEY,
    period_start_at TEXT NOT NULL,
    period_end_exclusive TEXT NOT NULL,
    generated_at TEXT NOT NULL,
    actual_spend_usd REAL NOT NULL DEFAULT 0,
    forecast_spend_usd REAL NOT NULL DEFAULT 0,
    budget_limit_usd REAL NOT NULL DEFAULT 0,
    projected_overrun_usd REAL NOT NULL DEFAULT 0,
    observed_day_count INTEGER NOT NULL DEFAULT 0,
    remaining_day_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_forecast_snapshots_period_start
    ON forecast_snapshots (period_start_at);

CREATE TABLE IF NOT EXISTS alert_events (
    alert_id TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    severity TEXT NOT NULL,
    triggered_at TEXT NOT NULL,
    period_start_at TEXT NOT NULL,
    period_end_exclusive TEXT NOT NULL,
    budget_id TEXT,
    forecast_id TEXT,
    insight_id TEXT,
    detector_category TEXT,
    current_spend_usd REAL NOT NULL DEFAULT 0,
    limit_usd REAL NOT NULL DEFAULT 0,
    threshold_percent REAL NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_alert_events_period_budget_kind
    ON alert_events (period_start_at, budget_id, kind, triggered_at);
