-- 000002_cart: one active cart per shopper per tenant, plus its line items.

CREATE TABLE carts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,                        -- reference to a user-management tenant
    customer_id UUID NOT NULL,                        -- the user id from the access token
    status      TEXT NOT NULL DEFAULT 'ACTIVE',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT carts_status_allowed CHECK (status IN ('ACTIVE', 'ORDERED', 'ABANDONED'))
);

CREATE INDEX carts_customer_idx ON carts (tenant_id, customer_id);
-- at most one ACTIVE cart per shopper
CREATE UNIQUE INDEX carts_one_active_per_customer
    ON carts (tenant_id, customer_id) WHERE status = 'ACTIVE';

CREATE TRIGGER carts_set_updated_at
    BEFORE UPDATE ON carts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE cart_items (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cart_id          UUID NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    product_id       UUID NOT NULL,                    -- product-service product id
    sku              CITEXT NOT NULL,                  -- product-service variant sku
    quantity         INTEGER NOT NULL,
    unit_price_cents BIGINT NOT NULL,                  -- snapshot at add time
    currency         TEXT NOT NULL DEFAULT 'USD',
    product_name     TEXT NOT NULL,                    -- snapshot
    variant_title    TEXT,                             -- snapshot
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT cart_items_cart_sku_key      UNIQUE (cart_id, sku),
    CONSTRAINT cart_items_quantity_positive CHECK (quantity > 0),
    CONSTRAINT cart_items_price_positive    CHECK (unit_price_cents >= 0)
);

CREATE INDEX cart_items_cart_id_idx ON cart_items (cart_id);

CREATE TRIGGER cart_items_set_updated_at
    BEFORE UPDATE ON cart_items
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
