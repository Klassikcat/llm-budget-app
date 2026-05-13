# OpenClaw Discovery Notes

## Purpose

This document records the OpenClaw storage paths and formats that future `llm-budget-app` parsers should consider. It is intentionally conservative. Confirmed facts come from the OpenClaw repository and are separated from hypotheses or platform conventions that still need fixture confirmation.

Authoritative source repository: https://github.com/openclaw/openclaw

## Privacy Boundary

OpenClaw session transcript files are JSONL and may contain prompt text, model responses, tool inputs, raw provider payloads, or other user content. Future parsers must read only token, cost, timestamp, model, provider, agent, and session metadata needed for budgeting. They must not store prompt or response text.

Do not inspect a user's live OpenClaw directories during development unless the user has explicitly provided a fixture. Use synthetic fixtures or source guided samples instead.

## Confirmed Repository Evidence

| Fact | Evidence |
| --- | --- |
| Default state directory is `~/.openclaw` through Node.js home directory style resolution | `src/config/paths.ts`, constants `NEW_STATE_DIRNAME = ".openclaw"` and state path resolution with `os.homedir()` |
| State directory can be overridden | `src/config/paths.ts`, `OPENCLAW_STATE_DIR` |
| Config file path can be overridden | `src/config/paths.ts`, `OPENCLAW_CONFIG_PATH` |
| Main config file is `openclaw.json` | `src/config/paths.ts`, config filename constant |
| Legacy state and config names exist | `src/config/paths.ts`, `.clawdbot` and `clawdbot.json` |
| Dev and profile state directories exist | `src/config/paths.ts`, `~/.openclaw-dev` and `~/.openclaw-<name>` style variants |
| Session index and transcript paths exist | `src/config/sessions/paths.ts` |
| Plugin state SQLite path exists | `src/plugin-state/plugin-state-store.paths.ts` |
| Task run SQLite path exists | `src/tasks/task-registry.paths.ts` |
| Memory vector search uses SQLite and sqlite-vec | `packages/memory-host-sdk/src/host/sqlite-vec.ts` |
| Cron jobs and cron run JSONL paths exist | `src/cron/store.ts` |
| Exec approval file and socket paths exist | `src/infra/exec-approvals.ts` |
| Log JSONL paths are part of the state layout | `src/infra/backup-volatile-filter.ts` and related logging path code |
| No LevelDB usage was found in the research pass | OpenClaw source search summarized in Task 4.1 research notes |

## Candidate Paths By Platform

These are candidate locations for discovery. The shared confirmed fact is that OpenClaw uses a home directory based state directory named `.openclaw`. The macOS Application Support path was an initial hypothesis and is not confirmed by the repository evidence.

| Platform | Confirmed or candidate root | Status | Notes |
| --- | --- | --- | --- |
| macOS | `~/.openclaw` | Confirmed | Main OpenClaw state directory from `src/config/paths.ts` using `os.homedir()` |
| Linux | `~/.openclaw` | Confirmed | Same home directory based resolution as macOS |
| Windows | `C:\Users\<user>\.openclaw` | Confirmed convention | Same `os.homedir()` based state root, rendered as the typical Windows home directory path |
| macOS | `~/Library/Application Support/openclaw` | Unverified hypothesis | Not confirmed. Current research points to `~/.openclaw` instead |
| Windows | `%LOCALAPPDATA%\OpenClaw\deps\portable-git` | Confirmed non usage data | Portable Git dependency path only. Do not parse for LLM usage data |
| Any | `$OPENCLAW_STATE_DIR` | Confirmed override | Highest priority when set |
| Any | `$OPENCLAW_CONFIG_PATH` | Confirmed config override | Treat as a direct config file path, not a state root |
| Any | `$OPENCLAW_HOME` | Unverified candidate | Include only as a hypothesis until source confirmation finds it |
| Any | `~/.openclaw-dev` | Confirmed variant | Dev mode state directory |
| Any | `~/.openclaw-<name>` | Confirmed variant | Profile mode state directory |
| Any | `~/.clawdbot` | Confirmed legacy variant | Legacy state directory before the OpenClaw rename |

## Key Candidate Files

| Candidate path | Format | Status | Evidence | Parser relevance |
| --- | --- | --- | --- | --- |
| `~/.openclaw/openclaw.json` | JSON5 style config | Confirmed | `src/config/paths.ts` | Agent, model, channel, logging, and session settings may help map usage metadata |
| `~/.openclaw/agents/<agentId>/sessions/sessions.json` | JSON | Confirmed | `src/config/sessions/paths.ts` | Session index and updated timestamps |
| `~/.openclaw/agents/<agentId>/sessions/<sessionId>.jsonl` | JSONL | Confirmed | `src/config/sessions/paths.ts`, `src/config/sessions/artifacts.ts` | Main transcript candidate for token and cost metadata |
| `~/.openclaw/plugin-state/state.sqlite` | SQLite | Confirmed | `src/plugin-state/plugin-state-store.paths.ts` | Plugin state. Inspect schema before using |
| `~/.openclaw/tasks/runs.sqlite` | SQLite | Confirmed | `src/tasks/task-registry.paths.ts` | Task run registry and delivery state |
| `~/.openclaw/logs/*.jsonl` | JSONL | Confirmed | `src/infra/backup-volatile-filter.ts`, logging path code | Logs such as `cache-trace.jsonl`, `anthropic-payload.jsonl`, `raw-stream.jsonl`, and `config-audit.jsonl` may contain metadata, but can also contain sensitive payloads |
| `~/.openclaw/cron/jobs.json` | JSON | Confirmed | `src/cron/store.ts` | Cron job definitions |
| `~/.openclaw/cron/runs/<jobId>.jsonl` | JSONL | Confirmed | `src/cron/store.ts` | Cron run history, possibly linked to automated model calls |
| `~/.openclaw/exec-approvals.json` | JSON | Confirmed | `src/infra/exec-approvals.ts` | Not a usage source, but part of state layout |
| `~/.openclaw/exec-approvals.sock` | Unix socket | Confirmed | `src/infra/exec-approvals.ts` | Not a usage source. Never parse as data |

## Storage Formats

| Format | Confirmed locations | Notes |
| --- | --- | --- |
| JSON and JSON5 | `openclaw.json`, `sessions.json`, `cron/jobs.json`, `exec-approvals.json` | Config appears JSON5 like. Other state files are plain JSON candidates |
| JSONL | `agents/<agentId>/sessions/<sessionId>.jsonl`, `logs/*.jsonl`, `cron/runs/<jobId>.jsonl`, `config-audit.jsonl` | One JSON object per line. Treat transcript and raw provider logs as sensitive |
| SQLite | `plugin-state/state.sqlite`, `tasks/runs.sqlite`, memory sqlite-vec storage | Inspect schema at implementation time before extracting usage metadata |
| LevelDB | None found | Research found no LevelDB usage in the OpenClaw source tree |

## Schema Hints

| File | Current hint | Confidence |
| --- | --- | --- |
| `sessions.json` | Likely a record keyed by session key with fields such as `sessionId`, `updatedAt`, and `channel` | Confirmed source path, partial schema hint |
| `sessions/<sessionId>.jsonl` | One JSON object per line for transcript events | Confirmed format, exact line schema unknown |
| `cron/jobs.json` | Expected shape resembles `{ version: 1, jobs: CronJob[] }` | Source guided hint |
| `exec-approvals.json` | Expected shape resembles `{ version: 1, agents: Record<string, AgentPolicy> }` | Source guided hint |
| `tasks/runs.sqlite` | Research identified `task_runs`, `task_delivery_state`, and `task_registry_records` tables | Source guided hint |
| `plugin-state/state.sqlite` | Uses SQLite schema versioning and WAL mode | Source guided hint |

## Unknowns And Caveats

1. The exact session transcript JSONL line schema is still unverified. Implementation should inspect `src/agents/session-write-lock.ts`, source fixtures, or synthetic captures before parsing.
2. The exact fields that carry token counts, model identifiers, cached token counts, and cost are unknown from this discovery note alone.
3. `$OPENCLAW_HOME` is listed as an unverified candidate only. Do not treat it as a confirmed OpenClaw source until the implementation pass confirms it in source.
4. The macOS Application Support path is not confirmed and should not be scanned by default based on current evidence.
5. Logs such as `anthropic-payload.jsonl` and `raw-stream.jsonl` can contain private request or response bodies. Prefer session metadata and explicit usage fields over raw payload logs.
6. SQLite database schemas should be read from source or fixtures before any query is written. Do not assume table stability across OpenClaw releases.
7. Legacy `.clawdbot` paths may exist for older installs, but parser support should be deliberate and tested separately.

## Downstream Parser Recommendations

1. Discovery order should start with `$OPENCLAW_STATE_DIR`, then dev or profile variants when configured, then the default home directory state path for the current platform.
2. Treat `$OPENCLAW_CONFIG_PATH` as a direct config file override that can help locate agent and session settings, not as a replacement state directory.
3. Prefer `sessions.json` for session discovery and session JSONL files for usage records. Avoid raw provider logs unless no safer metadata source exists.
4. Extract only privacy safe budgeting data. Good candidates are timestamps, model IDs, provider names, token counts, cache token counts, costs, session IDs, and agent IDs.
5. Add fixtures that cover macOS, Linux, Windows home directory roots, `$OPENCLAW_STATE_DIR`, dev mode, and profile mode before enabling automatic parsing.
