-- +goose Up
CREATE TABLE maker_portfolio_works (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    maker_id    UUID NOT NULL REFERENCES makers (id) ON DELETE CASCADE,
    title       TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    sort_order  INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_maker_portfolio_works_maker ON maker_portfolio_works (maker_id, sort_order);

ALTER TABLE maker_media
    ADD COLUMN work_id UUID REFERENCES maker_portfolio_works (id) ON DELETE CASCADE;

-- One work per existing flat media row (title empty, description = alt).
-- +goose StatementBegin
DO $$
DECLARE
    r RECORD;
    wid UUID;
BEGIN
    FOR r IN
        SELECT id, maker_id, alt, sort_order
        FROM maker_media
        WHERE work_id IS NULL
        ORDER BY maker_id, sort_order, id
    LOOP
        wid := gen_random_uuid();
        INSERT INTO maker_portfolio_works (id, maker_id, title, description, sort_order)
        VALUES (wid, r.maker_id, '', r.alt, r.sort_order);
        UPDATE maker_media SET work_id = wid WHERE id = r.id;
    END LOOP;
END $$;
-- +goose StatementEnd

ALTER TABLE maker_media
    ALTER COLUMN work_id SET NOT NULL;

CREATE INDEX idx_maker_media_work ON maker_media (work_id, sort_order);

-- +goose Down
DROP INDEX IF EXISTS idx_maker_media_work;
ALTER TABLE maker_media DROP COLUMN IF EXISTS work_id;
DROP TABLE IF EXISTS maker_portfolio_works;
