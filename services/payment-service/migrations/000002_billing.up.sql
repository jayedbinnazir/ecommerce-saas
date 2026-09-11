-- 000002_billing: subscription plans + per-user subscriptions.
-- Taken over from user-management. Subscribing is stubbed (no card charge here).

CREATE TABLE plans (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code             TEXT NOT NULL,
    name             TEXT NOT NULL,
    billing_interval TEXT NOT NULL,
    price_cents      BIGINT NOT NULL,          -- charged each period
    currency         TEXT NOT NULL DEFAULT 'USD',
    active           BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT plans_code_key UNIQUE (code),
    CONSTRAINT plans_interval_allowed CHECK (billing_interval IN ('MONTH', 'YEAR'))
);

CREATE TRIGGER plans_set_updated_at
    BEFORE UPDATE ON plans
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE subscriptions (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID NOT NULL,                    -- reference to a user-management user
    plan_id              UUID NOT NULL REFERENCES plans(id) ON DELETE RESTRICT,
    status               TEXT NOT NULL,
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end   TIMESTAMPTZ NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT subscriptions_status_allowed CHECK (status IN ('ACTIVE', 'CANCELED', 'EXPIRED', 'PAST_DUE'))
);

CREATE INDEX subscriptions_user_id_idx ON subscriptions (user_id);

-- A user can hold only one ACTIVE subscription at a time.
CREATE UNIQUE INDEX subscriptions_one_active_per_user
    ON subscriptions (user_id) WHERE status = 'ACTIVE';

CREATE TRIGGER subscriptions_set_updated_at
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Two plans: pay monthly, or pay yearly for a lower per-month price.
INSERT INTO plans (code, name, billing_interval, price_cents, currency) VALUES
    ('starter-monthly', 'Starter — Monthly', 'MONTH',  3000, 'USD'),  -- $30 / month
    ('starter-yearly',  'Starter — Yearly',  'YEAR',  30000, 'USD');  -- $300 / year  ($25 / month)
