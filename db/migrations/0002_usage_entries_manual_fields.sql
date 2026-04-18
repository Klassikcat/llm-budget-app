CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_entries_entry_id
    ON usage_entries (entry_id);

CREATE INDEX IF NOT EXISTS idx_usage_entries_provider_project
    ON usage_entries (provider, project_name, recorded_at);

CREATE INDEX IF NOT EXISTS idx_sessions_provider_project
    ON sessions (provider, project_name, started_at);
