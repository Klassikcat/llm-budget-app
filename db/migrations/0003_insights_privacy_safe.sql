DROP TABLE IF EXISTS insights;

CREATE TABLE IF NOT EXISTS insights (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    insight_id TEXT NOT NULL,
    category TEXT NOT NULL,
    severity TEXT NOT NULL,
    detected_at TEXT NOT NULL,
    period_start_at TEXT NOT NULL,
    period_end_exclusive TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (insight_id)
);

CREATE INDEX IF NOT EXISTS idx_insights_period_start_category
    ON insights (period_start_at, category);

CREATE INDEX IF NOT EXISTS idx_insights_detected_at
    ON insights (detected_at);
