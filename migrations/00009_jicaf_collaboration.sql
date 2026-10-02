-- +goose Up
ALTER TABLE makers
    ADD COLUMN jicaf_collaborated_before BOOLEAN,
    ADD COLUMN jicaf_collaboration_artist TEXT,
    ADD COLUMN jicaf_collaboration_year INT;

-- +goose Down
ALTER TABLE makers
    DROP COLUMN IF EXISTS jicaf_collaboration_year,
    DROP COLUMN IF EXISTS jicaf_collaboration_artist,
    DROP COLUMN IF EXISTS jicaf_collaborated_before;
