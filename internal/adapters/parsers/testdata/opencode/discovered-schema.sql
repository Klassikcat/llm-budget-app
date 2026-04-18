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
VALUES ('proj_alpha', '/workspace/redacted/alpha-project', 'git', '', '', '', 1771343300000, 1771343300000, 1771343300000, '[]', NULL);

INSERT INTO session (id, project_id, parent_id, slug, directory, title, version, share_url, summary_additions, summary_deletions, summary_files, summary_diffs, revert, permission, time_created, time_updated, time_compacting, time_archived, workspace_id)
VALUES
    ('ses_alpha_or', 'proj_alpha', NULL, 'silent-river', '/workspace/redacted/alpha-project', '[REDACTED SESSION TITLE]', '1.3.15', NULL, NULL, NULL, NULL, NULL, NULL, NULL, 1771343300000, 1771343310000, NULL, NULL, NULL),
    ('ses_alpha_oa', 'proj_alpha', NULL, 'quiet-mountain', '/workspace/redacted/alpha-project', '[REDACTED SESSION TITLE]', '1.3.15', NULL, NULL, NULL, NULL, NULL, NULL, NULL, 1771343320000, 1771343330000, NULL, NULL, NULL),
    ('ses_alpha_google', 'proj_alpha', NULL, 'bright-forest', '/workspace/redacted/alpha-project', '[REDACTED SESSION TITLE]', '1.3.15', NULL, NULL, NULL, NULL, NULL, NULL, NULL, 1771343340000, 1771343350000, NULL, NULL, NULL);

INSERT INTO message (id, session_id, time_created, time_updated, data)
VALUES
    ('msg_alpha_user', 'ses_alpha_or', 1771343300000, 1771343300000, '{"role":"user","agent":"explore","time":{"created":1771343300000}}'),
    ('msg_alpha_or_1', 'ses_alpha_or', 1771343301000, 1771343305000, '{"role":"assistant","agent":"explore","mode":"explore","providerID":"openrouter","modelID":"google/gemini-2.5-pro","tokens":{"input":1200,"output":210,"reasoning":64,"cache":{"read":300,"write":10},"total":1784},"cost":0.01234,"time":{"created":1771343301000,"completed":1771343305000},"finish":"tool-calls","variant":"default"}'),
    ('msg_alpha_oa_1', 'ses_alpha_oa', 1771343321000, 1771343328000, '{"role":"assistant","agent":"Sisyphus-Junior","mode":"Sisyphus-Junior","providerID":"openai","modelID":"gpt-5.2-codex","tokens":{"input":800,"output":120,"reasoning":0,"cache":{"read":64,"write":0}},"cost":0,"time":{"created":1771343321000,"completed":1771343328000},"finish":"completed"}'),
    ('msg_alpha_google_1', 'ses_alpha_google', 1771343341000, 1771343347000, '{"role":"assistant","agent":"atlas","mode":"atlas","providerID":"google","modelID":"gemini-2.0-flash","tokens":{"input":600,"output":90,"reasoning":32,"cache":{"read":44,"write":0},"total":766},"cost":0,"time":{"created":1771343341000,"completed":1771343347000},"finish":"completed"}');

INSERT INTO part (id, message_id, session_id, time_created, time_updated, data)
VALUES
    ('part_alpha_tool_1', 'msg_alpha_or_1', 'ses_alpha_or', 1771343302000, 1771343303000, '{"type":"tool","callID":"tool_001","state":{"status":"completed"}}'),
    ('part_alpha_text_1', 'msg_alpha_or_1', 'ses_alpha_or', 1771343304000, 1771343304000, '{"type":"text","text":"SENSITIVE_TRANSCRIPT_SENTINEL"}'),
    ('part_alpha_text_2', 'msg_alpha_oa_1', 'ses_alpha_oa', 1771343322000, 1771343322000, '{"type":"text","text":"SENSITIVE_TRANSCRIPT_SENTINEL"}');
