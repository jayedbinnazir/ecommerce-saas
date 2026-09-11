-- 000002_stock: per-tenant stock levels per SKU + an append-only movement ledger.

CREATE TABLE stock_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,                        -- reference to a user-management tenant
    sku           CITEXT NOT NULL,                      -- matches a product-service variant sku
    on_hand       INTEGER NOT NULL DEFAULT 0,
    reserved      INTEGER NOT NULL DEFAULT 0,
    reorder_level INTEGER NOT NULL DEFAULT 0,
    location      TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT stock_items_tenant_sku_key   UNIQUE (tenant_id, sku),
    CONSTRAINT stock_items_on_hand_positive  CHECK (on_hand >= 0),
    CONSTRAINT stock_items_reserved_positive CHECK (reserved >= 0),
    CONSTRAINT stock_items_reorder_positive  CHECK (reorder_level >= 0),
    CONSTRAINT stock_items_reserved_lte_on_hand CHECK (reserved <= on_hand)
);

CREATE INDEX stock_items_tenant_id_idx ON stock_items (tenant_id);
-- fast "what needs reordering" scan
CREATE INDEX stock_items_low_stock_idx ON stock_items (tenant_id)
    WHERE reorder_level > 0 AND (on_hand - reserved) <= reorder_level;

CREATE TRIGGER stock_items_set_updated_at
    BEFORE UPDATE ON stock_items
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE stock_movements (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stock_item_id  UUID NOT NULL REFERENCES stock_items(id) ON DELETE CASCADE,
    tenant_id      UUID NOT NULL,
    sku            CITEXT NOT NULL,
    type           TEXT NOT NULL,
    quantity       INTEGER NOT NULL,          -- as requested; ADJUST may be negative
    on_hand_after  INTEGER NOT NULL,
    reserved_after INTEGER NOT NULL,
    reason         TEXT,
    reference      TEXT,                      -- external ref: order id, PO number...
    created_by     UUID,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT stock_movements_type_allowed
        CHECK (type IN ('RECEIVE', 'SHIP', 'ADJUST', 'RESERVE', 'RELEASE'))
);

CREATE INDEX stock_movements_item_idx      ON stock_movements (stock_item_id, created_at DESC);
CREATE INDEX stock_movements_tenant_sku_idx ON stock_movements (tenant_id, sku);
