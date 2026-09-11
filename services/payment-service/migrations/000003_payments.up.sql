-- 000003_payments: one payment per order (CARD via Stripe, or COD).

CREATE TABLE payments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,                    -- reference to a user-management tenant
    order_id      UUID NOT NULL,                    -- reference to an order-service order
    customer_id   UUID NOT NULL,                    -- the user id from the access token
    method        TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'PENDING',
    amount_cents  BIGINT NOT NULL,
    currency      TEXT NOT NULL,
    gateway_ref   TEXT,                             -- Stripe payment_intent id
    client_secret TEXT,                             -- Stripe client_secret (frontend only)
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    captured_at   TIMESTAMPTZ,
    refunded_at   TIMESTAMPTZ,
    CONSTRAINT payments_tenant_order_key UNIQUE (tenant_id, order_id),
    CONSTRAINT payments_method_allowed CHECK (method IN ('CARD', 'COD')),
    CONSTRAINT payments_status_allowed CHECK (status IN ('PENDING', 'CAPTURED', 'FAILED', 'REFUNDED')),
    CONSTRAINT payments_amount_positive CHECK (amount_cents > 0)
);

CREATE INDEX payments_tenant_customer_idx ON payments (tenant_id, customer_id, created_at DESC);
CREATE UNIQUE INDEX payments_gateway_ref_idx ON payments (gateway_ref) WHERE gateway_ref IS NOT NULL;

CREATE TRIGGER payments_set_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
