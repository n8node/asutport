-- +goose Up
CREATE TYPE production_criticality AS ENUM ('continuous', 'batch', 'auxiliary');

CREATE TYPE product_kind AS ENUM ('plc', 'scada', 'hmi', 'drive', 'sensor', 'network', 'other');

CREATE TYPE supply_verify_status AS ENUM ('client_claim', 'manufacturer_verified', 'partner_verified');

CREATE TABLE installations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_org_id UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    name TEXT NOT NULL DEFAULT '',
    site_address TEXT NOT NULL DEFAULT '',
    criticality production_criticality NOT NULL DEFAULT 'batch',
    snapshot_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    emergency_contact_name TEXT NOT NULL DEFAULT '',
    emergency_contact_phone TEXT NOT NULL DEFAULT '',
    environment JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_installations_client_org ON installations (client_org_id);

CREATE TABLE installation_products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    installation_id UUID NOT NULL REFERENCES installations (id) ON DELETE CASCADE,
    manufacturer_name TEXT NOT NULL DEFAULT '',
    product_name TEXT NOT NULL DEFAULT '',
    kind product_kind NOT NULL DEFAULT 'other',
    version TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_installation_products_installation ON installation_products (installation_id);

CREATE TABLE supply_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    installation_product_id UUID NOT NULL REFERENCES installation_products (id) ON DELETE CASCADE,
    serial_or_license TEXT NOT NULL DEFAULT '',
    supplier_name TEXT NOT NULL DEFAULT '',
    integrator_name TEXT NOT NULL DEFAULT '',
    purchase_date DATE,
    warranty_until DATE,
    contract_ref TEXT NOT NULL DEFAULT '',
    verify_status supply_verify_status NOT NULL DEFAULT 'client_claim',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (installation_product_id, serial_or_license)
);

CREATE INDEX idx_supply_records_product ON supply_records (installation_product_id);

ALTER TABLE tickets
    ADD COLUMN IF NOT EXISTS sla_reaction_deadline TIMESTAMPTZ;

-- +goose Down
ALTER TABLE tickets DROP COLUMN IF EXISTS sla_reaction_deadline;

DROP TABLE IF EXISTS supply_records;
DROP TABLE IF EXISTS installation_products;
DROP TABLE IF EXISTS installations;

DROP TYPE IF EXISTS supply_verify_status;
DROP TYPE IF EXISTS product_kind;
DROP TYPE IF EXISTS production_criticality;
