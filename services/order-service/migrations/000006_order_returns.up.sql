-- 000006_order_returns: customer return requests + manager resolution.

CREATE TABLE order_returns (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL,
    customer_id     UUID NOT NULL,
    status          TEXT NOT NULL DEFAULT 'REQUESTED',
    reason          TEXT NOT NULL,
    resolution_note TEXT,
    refund_cents    BIGINT NOT NULL DEFAULT 0,   -- suggested at request, final on resolve
    restocked       BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at     TIMESTAMPTZ,
    CONSTRAINT order_returns_status_allowed CHECK (status IN ('REQUESTED', 'COMPLETED', 'REJECTED')),
    CONSTRAINT order_returns_refund_nonneg  CHECK (refund_cents >= 0)
);

CREATE INDEX order_returns_order_idx         ON order_returns (order_id);
CREATE INDEX order_returns_tenant_status_idx ON order_returns (tenant_id, status, created_at DESC);

CREATE TRIGGER order_returns_set_updated_at
    BEFORE UPDATE ON order_returns
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE order_return_items (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    return_id        UUID NOT NULL REFERENCES order_returns(id) ON DELETE CASCADE,
    sku              CITEXT NOT NULL,
    quantity         INTEGER NOT NULL,
    unit_price_cents BIGINT NOT NULL,
    product_name     TEXT NOT NULL,
    CONSTRAINT order_return_items_quantity_positive CHECK (quantity > 0)
);

CREATE INDEX order_return_items_return_idx ON order_return_items (return_id);
