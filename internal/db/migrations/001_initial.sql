CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    start_time TEXT NOT NULL,
    end_time TEXT NOT NULL,
    location TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    meeting_link TEXT NOT NULL DEFAULT '',
    meeting_provider TEXT NOT NULL DEFAULT 'unknown',
    attendees_json TEXT NOT NULL DEFAULT '[]',
    organizer TEXT NOT NULL DEFAULT '',
    calendar_id TEXT NOT NULL DEFAULT 'primary',
    status TEXT NOT NULL DEFAULT 'confirmed',
    attendees_hash TEXT NOT NULL DEFAULT '',
    sync_generation INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_events_start ON events(start_time);
CREATE INDEX IF NOT EXISTS idx_events_sync_gen ON events(sync_generation);

CREATE TABLE IF NOT EXISTS sync_state (
    id INTEGER PRIMARY KEY,
    token TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS hook_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id TEXT NOT NULL,
    hook_name TEXT NOT NULL,
    hook_type TEXT NOT NULL,
    status TEXT NOT NULL,
    stdout TEXT NOT NULL DEFAULT '',
    stderr TEXT NOT NULL DEFAULT '',
    duration_ms INTEGER NOT NULL DEFAULT 0,
    executed_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_hook_exec_event ON hook_executions(event_id);
CREATE INDEX IF NOT EXISTS idx_hook_exec_dedup ON hook_executions(event_id, hook_name, hook_type);
