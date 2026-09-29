-- +goose Up
ALTER TABLE makers
    ADD COLUMN hours_open TEXT,
    ADD COLUMN hours_close TEXT,
    ADD COLUMN service_area TEXT;

CREATE TABLE maker_media (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    maker_id   UUID NOT NULL REFERENCES makers (id) ON DELETE CASCADE,
    url        TEXT NOT NULL,
    alt        TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_maker_media_maker ON maker_media (maker_id, sort_order);

-- +goose Down
DROP TABLE IF EXISTS maker_media;
ALTER TABLE makers
    DROP COLUMN IF EXISTS service_area,
    DROP COLUMN IF EXISTS hours_close,
    DROP COLUMN IF EXISTS hours_open;
