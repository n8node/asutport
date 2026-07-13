-- +goose Up
CREATE TYPE ticket_type AS ENUM (
    'onboarding',
    'typical',
    'defect',
    'warranty',
    'application',
    'cross_vendor'
);

CREATE TYPE ticket_priority AS ENUM ('emergency', 'degraded', 'question');

CREATE TYPE ticket_status AS ENUM (
    'open',
    'waiting_client',
    'waiting_platform',
    'waiting_vendor',
    'resolved',
    'closed'
);

CREATE TABLE tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_org_id UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    installation_id UUID,
    type ticket_type NOT NULL,
    priority ticket_priority NOT NULL DEFAULT 'question',
    status ticket_status NOT NULL DEFAULT 'open',
    ball_owner_org_id UUID REFERENCES organizations (id),
    subject TEXT NOT NULL DEFAULT '',
    created_by_user_id UUID REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tickets_client_org ON tickets (client_org_id);
CREATE INDEX idx_tickets_type_status ON tickets (type, status);
CREATE UNIQUE INDEX idx_tickets_onboarding_per_org ON tickets (client_org_id) WHERE type = 'onboarding';

ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS onboarding_ticket_id UUID REFERENCES tickets (id);

CREATE TABLE ticket_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id UUID NOT NULL REFERENCES tickets (id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    actor_user_id UUID REFERENCES users (id),
    actor_org_id UUID REFERENCES organizations (id),
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ticket_events_ticket_created ON ticket_events (ticket_id, created_at DESC);

CREATE TABLE ticket_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id UUID NOT NULL REFERENCES tickets (id) ON DELETE CASCADE,
    event_id UUID REFERENCES ticket_events (id),
    s3_key TEXT NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    uploaded_by_user_id UUID NOT NULL REFERENCES users (id),
    uploaded_by_org_id UUID NOT NULL REFERENCES organizations (id),
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ticket_attachments_ticket ON ticket_attachments (ticket_id);

-- +goose Down
DROP TABLE IF EXISTS ticket_attachments;
DROP TABLE IF EXISTS ticket_events;
ALTER TABLE organizations DROP COLUMN IF EXISTS onboarding_ticket_id;
DROP TABLE IF EXISTS tickets;
DROP TYPE IF EXISTS ticket_status;
DROP TYPE IF EXISTS ticket_priority;
DROP TYPE IF EXISTS ticket_type;
