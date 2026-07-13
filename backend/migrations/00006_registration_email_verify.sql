-- +goose Up
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;

UPDATE users SET email_verified_at = COALESCE(email_verified_at, created_at);

CREATE TABLE registration_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    reg_id TEXT NOT NULL UNIQUE,
    account_type TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_registration_verifications_reg_id ON registration_verifications (reg_id);
CREATE INDEX idx_registration_verifications_user_id ON registration_verifications (user_id);

-- +goose Down
DROP TABLE IF EXISTS registration_verifications;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
