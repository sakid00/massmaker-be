-- +goose Up
DROP TABLE IF EXISTS staff_users;

-- +goose Down
CREATE TABLE staff_users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('content_owner', 'editor')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
