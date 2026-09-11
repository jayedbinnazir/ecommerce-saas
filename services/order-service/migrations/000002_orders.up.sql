-- 000002_orders: immutable checkout snapshots + their line items.

CREATE TABLE orders (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,                       -- reference to a user-management tenant
    customer_id    UUID NOT NULL,                       -- the user id from the access token
    status         TEXT NOT NULL DEFAULT 'PENDING_PAYMENT',
    currency       TEXT NOT NULL,
    subtotal_cents BIGINT NOT NULL,
    item_count     INTEGER NOT NULL,
    payment_ref    TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    paid_at        TIMESTAMPTZ,
    fulfilled_at   TIMESTAMPTZ,
    cancelled_at   TIMESTAMPTZ,
    CONSTRAINT orders_status_allowed
        CHECK (status IN ('PENDING_PAYMENT', 'PAID', 'FULFILLED', 'CANCELLED')),
    CONSTRAINT orders_subtotal_positive CHECK (subtotal_cents >= 0),
    CONSTRAINT orders_item_count_positive CHECK (item_count > 0)
);

-- customer's own order history, newest first
CREATE INDEX orders_tenant_customer_idx ON orders (tenant_id, customer_id, created_at DESC);
-- tenant-wide management view, filterable by status
CREATE INDEX orders_tenant_status_idx ON orders (tenant_id, status, created_at DESC);

CREATE TRIGGER orders_set_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE order_items (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id         UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id       UUID NOT NULL,                     -- product-service product id
    sku              CITEXT NOT NULL,                   -- product-service variant sku
    quantity         INTEGER NOT NULL,
    unit_price_cents BIGINT NOT NULL,                   -- snapshot at checkout
    currency         TEXT NOT NULL,
    product_name     TEXT NOT NULL,                     -- snapshot
    variant_title    TEXT,                              -- snapshot
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT order_items_quantity_positive CHECK (quantity > 0),
    CONSTRAINT order_items_price_positive    CHECK (unit_price_cents >= 0)
);

CREATE INDEX order_items_order_id_idx ON order_items (order_id);
