-- +goose Up
CREATE TYPE consent_type AS ENUM (
    'activity_tracking',
    'marketing',
    'whatsapp_publication',
    'recommendation_publication',
    'privacy_policy',
    'user_agreement',
    'terms'
);

CREATE TYPE consent_source AS ENUM (
    'signup',
    'onboarding',
    'activity_banner',
    'footer'
);

CREATE TABLE consents (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID,
    anon_session_id  UUID NOT NULL,
    consent_type     consent_type NOT NULL,
    version          TEXT NOT NULL DEFAULT '1.0',
    granted          BOOLEAN NOT NULL,
    granted_at       TIMESTAMPTZ,
    withdrawn_at     TIMESTAMPTZ,
    source           consent_source NOT NULL,
    ip_hash          TEXT NOT NULL DEFAULT '',
    user_agent_hash  TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (anon_session_id, consent_type)
);

CREATE UNIQUE INDEX consents_user_type ON consents (user_id, consent_type)
    WHERE user_id IS NOT NULL;

CREATE INDEX consents_user_id ON consents (user_id)
    WHERE user_id IS NOT NULL;

ALTER TABLE makers ADD COLUMN enmasse_user_id UUID;

-- +goose Down
ALTER TABLE makers DROP COLUMN IF EXISTS enmasse_user_id;
DROP TABLE IF EXISTS consents;
DROP TYPE IF EXISTS consent_source;
DROP TYPE IF EXISTS consent_type;
