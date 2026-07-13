-- +goose Up
-- +goose NO TRANSACTION
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'org_review_status') THEN
        CREATE TYPE org_review_status AS ENUM (
            'pending_email',
            'pending_review',
            'active',
            'rejected',
            'suspended'
        );
    END IF;
END $$;

ALTER TYPE org_type ADD VALUE IF NOT EXISTS 'vendor';

ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS legal_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS inn TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS website TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS contact_phone TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS review_comment TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS is_personal BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS review_status org_review_status NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reviewed_by UUID REFERENCES users (id);

CREATE INDEX IF NOT EXISTS idx_organizations_review_status ON organizations (review_status);
CREATE INDEX IF NOT EXISTS idx_organizations_type_status ON organizations (type, review_status);

-- +goose Down
ALTER TABLE organizations
    DROP COLUMN IF EXISTS reviewed_by,
    DROP COLUMN IF EXISTS reviewed_at,
    DROP COLUMN IF EXISTS review_status,
    DROP COLUMN IF EXISTS is_personal,
    DROP COLUMN IF EXISTS review_comment,
    DROP COLUMN IF EXISTS contact_phone,
    DROP COLUMN IF EXISTS website,
    DROP COLUMN IF EXISTS inn,
    DROP COLUMN IF EXISTS legal_name;

DROP TYPE IF EXISTS org_review_status;
