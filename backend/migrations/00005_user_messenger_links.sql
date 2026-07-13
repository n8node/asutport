-- +goose Up
CREATE TYPE messenger_provider AS ENUM ('telegram', 'max');

CREATE TABLE user_messenger_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider messenger_provider NOT NULL,
    external_user_id TEXT NOT NULL DEFAULT '',
    username TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT '',
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    notifications_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    linked_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, provider)
);

CREATE INDEX idx_user_messenger_links_user_id ON user_messenger_links (user_id);
CREATE INDEX idx_user_messenger_links_provider ON user_messenger_links (provider);

-- +goose Down
DROP TABLE IF EXISTS user_messenger_links;
DROP TYPE IF EXISTS messenger_provider;
