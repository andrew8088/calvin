ALTER TABLE sync_state ADD COLUMN calendar_id TEXT NOT NULL DEFAULT 'primary';

CREATE UNIQUE INDEX IF NOT EXISTS idx_sync_state_calendar ON sync_state(calendar_id);
