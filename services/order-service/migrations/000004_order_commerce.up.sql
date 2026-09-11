-- 000004_order_commerce: real checkout — delivery address, shipping, tax,
-- discounts, and a per-tenant config for each. Plus checkout idempotency.

-- ---- order: addresses, money breakdown, idempotency ----
ALTER TABLE orders
    ADD COLUMN shipping_address JSONB,
    ADD COLUMN billing_address  JSONB,
    ADD COLUMN discount_cents    BIGINT  NOT NULL DEFAULT 0,
    ADD COLUMN shipping_cents    BIGINT  NOT NULL DEFAULT 0,
    ADD COLUMN tax_cents         BIGINT  NOT NULL DEFAULT 0,
    ADD COLUMN grand_total_cents BIGINT  NOT NULL DEFAULT 0,
    ADD COLUMN coupon_code       CITEXT,
    ADD COLUMN idempotency_key   TEXT,
    ADD CONSTRAINT orders_money_nonneg
        CHECK (discount_cents >= 0 AND shipping_cents >= 0 AND tax_cents >= 0 AND grand_total_cents >= 0);

UPDATE orders SET grand_total_cents = subtotal_cents WHERE grand_total_cents = 0;

-- one checkout per (tenant, customer, key): a retried request returns the same order
CREATE UNIQUE INDEX orders_idempotency_key
    ON orders (tenant_id, customer_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- ---- per-tenant tax config ----
CREATE TABLE order_config (
    tenant_id     UUID PRIMARY KEY,
    tax_rate_bps  INTEGER NOT NULL DEFAULT 0,     -- basis points: 875 = 8.75%
    tax_inclusive BOOLEAN NOT NULL DEFAULT false, -- true = catalog prices already include tax
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT order_config_tax_rate_range CHECK (tax_rate_bps BETWEEN 0 AND 10000)
);

CREATE TRIGGER order_config_set_updated_at
    BEFORE UPDATE ON order_config
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---- per-tenant shipping rates ----
CREATE TABLE shipping_rates (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    name           TEXT NOT NULL,
    amount_cents   BIGINT NOT NULL,
    free_over_cents BIGINT,                       -- free when (subtotal - discount) >= this; NULL = never
    active         BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT shipping_rates_amount_nonneg CHECK (amount_cents >= 0),
    CONSTRAINT shipping_rates_free_over_nonneg CHECK (free_over_cents IS NULL OR free_over_cents >= 0)
);

CREATE INDEX shipping_rates_tenant_idx ON shipping_rates (tenant_id);

CREATE TRIGGER shipping_rates_set_updated_at
    BEFORE UPDATE ON shipping_rates
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---- per-tenant coupons ----
CREATE TABLE coupons (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    code            CITEXT NOT NULL,
    kind            TEXT NOT NULL,                -- PERCENT | FIXED
    value           BIGINT NOT NULL,             -- PERCENT: whole percent 1-100; FIXED: cents
    min_order_cents BIGINT NOT NULL DEFAULT 0,
    max_redemptions INTEGER,                      -- NULL = unlimited
    redeemed_count  INTEGER NOT NULL DEFAULT 0,
    starts_at       TIMESTAMPTZ,
    ends_at         TIMESTAMPTZ,
    active          BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT coupons_tenant_code_key UNIQUE (tenant_id, code),
    CONSTRAINT coupons_kind_allowed CHECK (kind IN ('PERCENT', 'FIXED')),
    CONSTRAINT coupons_value_positive CHECK (value > 0),
    CONSTRAINT coupons_percent_max CHECK (kind <> 'PERCENT' OR value <= 100)
);

CREATE INDEX coupons_tenant_idx ON coupons (tenant_id);

CREATE TRIGGER coupons_set_updated_at
    BEFORE UPDATE ON coupons
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
