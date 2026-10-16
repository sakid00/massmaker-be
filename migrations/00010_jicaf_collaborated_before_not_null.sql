-- +goose Up
-- Migration 00009 originally added jicaf_collaborated_before as nullable.
-- The file was later edited to NOT NULL DEFAULT FALSE, but goose had already
-- applied version 9, so existing DBs kept NULLs and broke bool scans.
UPDATE makers
SET jicaf_collaborated_before = FALSE
WHERE jicaf_collaborated_before IS NULL;

ALTER TABLE makers
    ALTER COLUMN jicaf_collaborated_before SET DEFAULT FALSE,
    ALTER COLUMN jicaf_collaborated_before SET NOT NULL;

-- +goose Down
ALTER TABLE makers
    ALTER COLUMN jicaf_collaborated_before DROP NOT NULL,
    ALTER COLUMN jicaf_collaborated_before DROP DEFAULT;
