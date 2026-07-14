-- +goose Up
ALTER TABLE tickets
    ADD COLUMN IF NOT EXISTS assigned_target_org_id UUID REFERENCES organizations (id);

CREATE INDEX IF NOT EXISTS idx_tickets_assigned_target ON tickets (assigned_target_org_id)
    WHERE assigned_target_org_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS fallback_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id UUID NOT NULL REFERENCES tickets (id) ON DELETE CASCADE,
    needed_role TEXT NOT NULL,
    missing_org_name TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_fallback_log_ticket_id ON fallback_log (ticket_id);

-- +goose Down
DROP TABLE IF EXISTS fallback_log;

ALTER TABLE tickets DROP COLUMN IF EXISTS assigned_target_org_id;
