-- +goose Up
CREATE TYPE publication_status AS ENUM ('draft', 'published', 'withdrawn');
CREATE TYPE moq_basis AS ENUM ('total', 'per_desain', 'per_warna', 'other');
CREATE TYPE traffic_source AS ENUM ('public', 'demo', 'staff_test');
CREATE TYPE event_type AS ENUM ('search', 'profile_view', 'contact_click');

CREATE TABLE staff_users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('content_owner', 'editor')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE categories (
    slug           TEXT PRIMARY KEY,
    label_id       TEXT NOT NULL,
    shortcut_order INT NOT NULL DEFAULT 0
);

CREATE TABLE category_synonyms (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_slug TEXT NOT NULL REFERENCES categories (slug),
    synonym       TEXT NOT NULL,
    UNIQUE (synonym)
);

CREATE TABLE makers (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name               TEXT NOT NULL,
    city               TEXT NOT NULL,
    publication_status publication_status NOT NULL DEFAULT 'draft',
    verification_date  DATE,
    moq_quantity       INT,
    moq_basis          moq_basis,
    moq_basis_note     TEXT,
    lead_time_estimate TEXT,
    price_min_idr      INT,
    price_max_idr      INT,
    price_note         TEXT,
    conditions         TEXT,
    is_demo_fixture    BOOLEAN NOT NULL DEFAULT FALSE,
    enmasse_vendor_id  UUID UNIQUE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE maker_capabilities (
    maker_id      UUID NOT NULL REFERENCES makers (id) ON DELETE CASCADE,
    category_slug TEXT NOT NULL REFERENCES categories (slug),
    PRIMARY KEY (maker_id, category_slug)
);

CREATE TABLE maker_contacts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    maker_id          UUID NOT NULL UNIQUE REFERENCES makers (id) ON DELETE CASCADE,
    channel           TEXT NOT NULL DEFAULT 'whatsapp',
    e164_digits       TEXT NOT NULL,
    publish_permitted BOOLEAN NOT NULL DEFAULT FALSE,
    permission_at     TIMESTAMPTZ
);

CREATE TABLE artists (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name      TEXT NOT NULL,
    enmasse_artist_id UUID UNIQUE
);

CREATE TABLE recommendations (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id              UUID NOT NULL REFERENCES artists (id),
    maker_id               UUID NOT NULL REFERENCES makers (id) ON DELETE CASCADE,
    enmasse_artist_id      UUID,
    product_made           TEXT NOT NULL,
    production_date_approx TEXT NOT NULL,
    body                   TEXT NOT NULL,
    quantity               TEXT,
    would_use_again        BOOLEAN,
    publication_status     publication_status NOT NULL DEFAULT 'draft',
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE events (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type                 event_type NOT NULL,
    traffic_source       traffic_source NOT NULL,
    anon_session_id      UUID NOT NULL,
    raw_query            TEXT,
    interpreted_category TEXT,
    matched_synonym      TEXT,
    result_count         INT,
    no_result_reason     TEXT,
    maker_id             UUID,
    counted              BOOLEAN NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_makers_published ON makers (publication_status) WHERE is_demo_fixture = FALSE;
CREATE INDEX idx_events_created ON events (created_at);

-- +goose Down
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS recommendations;
DROP TABLE IF EXISTS artists;
DROP TABLE IF EXISTS maker_contacts;
DROP TABLE IF EXISTS maker_capabilities;
DROP TABLE IF EXISTS makers;
DROP TABLE IF EXISTS category_synonyms;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS staff_users;
DROP TYPE IF EXISTS event_type;
DROP TYPE IF EXISTS traffic_source;
DROP TYPE IF EXISTS moq_basis;
DROP TYPE IF EXISTS publication_status;
