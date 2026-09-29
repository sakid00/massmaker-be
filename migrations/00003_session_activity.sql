-- +goose Up
ALTER TABLE events ADD COLUMN enmasse_user_id UUID;

CREATE INDEX idx_events_session ON events (anon_session_id);
CREATE INDEX idx_events_contact_dedup ON events (anon_session_id, maker_id, created_at)
    WHERE type = 'contact_click' AND counted = TRUE;

CREATE TABLE sessions (
    anon_session_id UUID PRIMARY KEY,
    enmasse_user_id UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_flush_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS sessions;
DROP INDEX IF EXISTS idx_events_contact_dedup;
DROP INDEX IF EXISTS idx_events_session;
ALTER TABLE events DROP COLUMN IF EXISTS enmasse_user_id;
