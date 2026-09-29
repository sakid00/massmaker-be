-- +goose Up
CREATE TABLE category_groups (
    slug                 TEXT PRIMARY KEY,
    label_id             TEXT NOT NULL,
    shortcut_order       INT NOT NULL DEFAULT 0,
    publication_status   publication_status NOT NULL DEFAULT 'draft',
    source               TEXT NOT NULL DEFAULT 'staff'
                         CHECK (source IN ('seed', 'staff', 'vendor_proposal')),
    proposed_by_maker_id UUID REFERENCES makers (id),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE categories
    ADD COLUMN group_slug TEXT REFERENCES category_groups (slug),
    ADD COLUMN publication_status publication_status NOT NULL DEFAULT 'draft',
    ADD COLUMN source TEXT NOT NULL DEFAULT 'staff'
        CHECK (source IN ('seed', 'staff', 'vendor_proposal')),
    ADD COLUMN proposed_by_maker_id UUID REFERENCES makers (id);

INSERT INTO category_groups (slug, label_id, shortcut_order, publication_status, source) VALUES
    ('print',    'Print',    1, 'published', 'seed'),
    ('fabric',   'Fabric',   2, 'published', 'seed'),
    ('services', 'Services', 3, 'published', 'seed');

UPDATE categories SET group_slug = 'print',    publication_status = 'published', source = 'seed' WHERE slug IN ('sticker', 'artprint', 'print-3d', 'pin');
UPDATE categories SET group_slug = 'fabric',   publication_status = 'published', source = 'seed' WHERE slug IN ('kaos', 'totebag', 'topi');
UPDATE categories SET group_slug = 'services', publication_status = 'published', source = 'seed' WHERE slug IN ('gantungan');

ALTER TABLE categories ALTER COLUMN group_slug SET NOT NULL;

CREATE INDEX idx_categories_group ON categories (group_slug);
CREATE INDEX idx_categories_status ON categories (publication_status);
CREATE UNIQUE INDEX idx_categories_label_lower ON categories (lower(label_id));
CREATE UNIQUE INDEX idx_category_groups_label_lower ON category_groups (lower(label_id));

-- +goose Down
DROP INDEX IF EXISTS idx_category_groups_label_lower;
DROP INDEX IF EXISTS idx_categories_label_lower;
DROP INDEX IF EXISTS idx_categories_status;
DROP INDEX IF EXISTS idx_categories_group;

ALTER TABLE categories
    DROP COLUMN IF EXISTS proposed_by_maker_id,
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS publication_status,
    DROP COLUMN IF EXISTS group_slug;

DROP TABLE IF EXISTS category_groups;
