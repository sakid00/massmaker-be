-- +goose Up
CREATE TYPE order_status AS ENUM (
    'pending_review',
    'accepted',
    'declined',
    'cancelled'
);

CREATE TABLE orders (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_code       TEXT UNIQUE NOT NULL,
    maker_id             UUID NOT NULL REFERENCES makers (id),
    enmasse_user_id      UUID NOT NULL,
    enmasse_artist_id    UUID,
    artist_display_name  TEXT NOT NULL,
    maker_name           TEXT NOT NULL,
    maker_city           TEXT NOT NULL,
    status               order_status NOT NULL DEFAULT 'pending_review',
    project_name         TEXT NOT NULL,
    product_name         TEXT NOT NULL,
    category_slug        TEXT NOT NULL REFERENCES categories (slug),
    quantity             INT NOT NULL CHECK (quantity > 0),
    quantity_note        TEXT,
    design_count         INT NOT NULL DEFAULT 1 CHECK (design_count > 0),
    want_sample          BOOLEAN NOT NULL DEFAULT FALSE,
    sample_quantity      INT CHECK (sample_quantity IS NULL OR sample_quantity > 0),
    requested_ready_on   DATE NOT NULL,
    requested_ship_on    DATE NOT NULL,
    rush                 BOOLEAN NOT NULL DEFAULT FALSE,
    budget_min_idr       INT CHECK (budget_min_idr IS NULL OR budget_min_idr >= 0),
    budget_max_idr       INT CHECK (budget_max_idr IS NULL OR budget_max_idr >= 0),
    notes                TEXT,
    ship_to_province     TEXT NOT NULL,
    ship_to_city         TEXT NOT NULL,
    ship_to_district     TEXT NOT NULL,
    ship_to_address      TEXT,
    vendor_note          TEXT,
    status_changed_at    TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_artist ON orders (enmasse_user_id, created_at DESC);
CREATE INDEX idx_orders_maker ON orders (maker_id, status, created_at DESC);

CREATE TABLE order_attachments (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id          UUID NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    url               TEXT NOT NULL,
    content_type      TEXT NOT NULL,
    original_filename TEXT NOT NULL DEFAULT '',
    byte_size         INT NOT NULL CHECK (byte_size > 0),
    sort_order        INT NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_order_attachments_order ON order_attachments (order_id, sort_order);

-- +goose Down
DROP TABLE IF EXISTS order_attachments;
DROP TABLE IF EXISTS orders;
DROP TYPE IF EXISTS order_status;
