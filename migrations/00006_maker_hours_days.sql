-- +goose Up
ALTER TABLE makers
    ADD COLUMN hours_days TEXT[] NOT NULL DEFAULT '{}';

UPDATE makers
SET hours_days = ARRAY['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun']
WHERE hours_open IS NOT NULL AND btrim(hours_open) <> ''
  AND hours_close IS NOT NULL AND btrim(hours_close) <> '';

-- +goose Down
ALTER TABLE makers DROP COLUMN IF EXISTS hours_days;
