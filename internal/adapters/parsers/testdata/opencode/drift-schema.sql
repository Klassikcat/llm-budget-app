CREATE TABLE project (
    id TEXT PRIMARY KEY,
    worktree TEXT NOT NULL,
    vcs TEXT,
    name TEXT,
    icon_url TEXT,
    icon_color TEXT,
    time_created INTEGER NOT NULL,
    time_updated INTEGER NOT NULL,
    time_initialized INTEGER,
    sandboxes TEXT NOT NULL,
    commands TEXT
);

CREATE TABLE session (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    parent_id TEXT,
    slug TEXT NOT NULL,
    directory TEXT NOT NULL,
    title TEXT NOT NULL,
    version TEXT NOT NULL,
    share_url TEXT,
    summary_additions INTEGER,
    summary_deletions INTEGER,
    summary_files INTEGER,
    summary_diffs TEXT,
    revert TEXT,
    permission TEXT,
    time_created INTEGER NOT NULL,
    time_updated INTEGER NOT NULL,
    time_compacting INTEGER,
    time_archived INTEGER,
    workspace_id TEXT,
    FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE
);

CREATE TABLE message (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    time_created INTEGER NOT NULL,
    time_updated INTEGER NOT NULL,
    data TEXT NOT NULL,
    FOREIGN KEY (session_id) REFERENCES session(id) ON DELETE CASCADE
);

CREATE TABLE part (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    time_created INTEGER NOT NULL,
    time_updated INTEGER NOT NULL,
    data TEXT NOT NULL,
    FOREIGN KEY (message_id) REFERENCES message(id) ON DELETE CASCADE
);

INSERT INTO project (id, worktree, vcs, name, icon_url, icon_color, time_created, time_updated, time_initialized, sandboxes, commands)
VALUES ('proj_drift', '/workspace/redacted/drift-project', 'git', '', '', '', 1771344300000, 1771344300000, 1771344300000, '[]', NULL);

INSERT INTO session (id, project_id, parent_id, slug, directory, title, version, share_url, summary_additions, summary_deletions, summary_files, summary_diffs, revert, permission, time_created, time_updated, time_compacting, time_archived, workspace_id)
VALUES ('ses_drift_1', 'proj_drift', NULL, 'curious-cloud', '/workspace/redacted/drift-project', '[REDACTED SESSION TITLE]', '2.0.0', NULL, NULL, NULL, NULL, NULL, NULL, NULL, 1771344300000, 1771344310000, NULL, NULL, NULL);

INSERT INTO message (id, session_id, time_created, time_updated, data)
VALUES
    ('msg_drift_supported', 'ses_drift_1', 1771344301000, 1771344305000, '{"role":"assistant","agent":"atlas","mode":"atlas","providerID":"google","modelID":"gemini-2.0-flash","tokens":{"input":500,"output":70,"reasoning":18,"cache":88},"cost":{"total":0.0042},"time":{"created":1771344301000,"completed":1771344305000},"finish":"completed","variant":"preview"}'),
    ('msg_drift_missing_provider', 'ses_drift_1', 1771344306000, 1771344309000, '{"role":"assistant","agent":"atlas","tokens":{"input":120,"output":30},"cost":0.001,"time":{"created":1771344306000,"completed":1771344309000}}');
