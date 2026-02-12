CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cognito_sub     TEXT NOT NULL UNIQUE,
    email           TEXT NOT NULL,
    nickname        TEXT NOT NULL DEFAULT '',
    profile_image_url TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_cognito_sub ON users (cognito_sub);
