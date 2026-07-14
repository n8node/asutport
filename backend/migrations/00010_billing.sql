-- +goose Up
CREATE TYPE plan_org_type AS ENUM ('client', 'manufacturer', 'partner');

CREATE TYPE subscription_status AS ENUM ('active', 'past_due', 'cancelled', 'trialing');

CREATE TYPE payment_type AS ENUM ('subscription', 'service', 'overage');

CREATE TYPE payment_status AS ENUM ('pending', 'paid', 'cancelled');

CREATE TABLE plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_type plan_org_type NOT NULL,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    price_monthly_rub INT NOT NULL DEFAULT 0,
    ticket_quota INT,
    overage_price_rub INT NOT NULL DEFAULT 0,
    sla_matrix JSONB NOT NULL DEFAULT '{}'::jsonb,
    features JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_public BOOLEAN NOT NULL DEFAULT TRUE,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_type, slug)
);

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL UNIQUE REFERENCES organizations (id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES plans (id),
    status subscription_status NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMPTZ NOT NULL DEFAULT now(),
    current_period_end TIMESTAMPTZ NOT NULL,
    cancel_at_period_end BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES subscriptions (id) ON DELETE SET NULL,
    ticket_id UUID REFERENCES tickets (id) ON DELETE SET NULL,
    type payment_type NOT NULL,
    amount_kopecks INT NOT NULL,
    status payment_status NOT NULL DEFAULT 'pending',
    invoice_s3_key TEXT,
    note TEXT NOT NULL DEFAULT '',
    recorded_by UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_subscriptions_plan_id ON subscriptions (plan_id);
CREATE INDEX idx_payments_org_id ON payments (org_id);
CREATE INDEX idx_payments_status ON payments (status);

INSERT INTO plans (org_type, name, slug, price_monthly_rub, ticket_quota, overage_price_rub, sla_matrix, sort_order) VALUES
    ('client', 'Бесплатный', 'free', 0, 3, 5000, '{"emergency":30,"degraded":480,"question":2880}'::jsonb, 10),
    ('client', 'Входной', 'entry', 25000, 10, 3500, '{"emergency":30,"degraded":120,"question":480}'::jsonb, 20),
    ('client', 'Priority', 'priority', 60000, 30, 2500, '{"emergency":15,"degraded":60,"question":120}'::jsonb, 30),
    ('manufacturer', 'Базовый', 'basic', 110000, NULL, 0, '{}'::jsonb, 10),
    ('manufacturer', 'Расширенный', 'extended', 300000, NULL, 0, '{}'::jsonb, 20),
    ('partner', 'Канал поддержки', 'channel', 30000, NULL, 0, '{}'::jsonb, 10);

-- +goose Down
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS plans;
DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS payment_type;
DROP TYPE IF EXISTS subscription_status;
DROP TYPE IF EXISTS plan_org_type;
