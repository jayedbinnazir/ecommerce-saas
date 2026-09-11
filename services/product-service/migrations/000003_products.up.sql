-- 000003_products: products, their variants, images and category links.

CREATE TABLE products (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,                      -- reference to a user-management tenant
    name        TEXT NOT NULL,
    slug        CITEXT NOT NULL,
    description TEXT,
    status      TEXT NOT NULL DEFAULT 'DRAFT',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT products_tenant_slug_key UNIQUE (tenant_id, slug),
    CONSTRAINT products_slug_format CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    CONSTRAINT products_status_allowed CHECK (status IN ('DRAFT', 'ACTIVE', 'ARCHIVED'))
);

CREATE INDEX products_tenant_status_idx ON products (tenant_id, status);

CREATE TRIGGER products_set_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE product_variants (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id             UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    tenant_id              UUID NOT NULL,           -- denormalized so SKU is unique per tenant
    sku                    CITEXT NOT NULL,
    title                  TEXT,
    price_cents            BIGINT NOT NULL CHECK (price_cents >= 0),
    currency               TEXT NOT NULL DEFAULT 'USD',
    compare_at_price_cents BIGINT CHECK (compare_at_price_cents IS NULL OR compare_at_price_cents >= 0),
    weight_grams           INTEGER CHECK (weight_grams IS NULL OR weight_grams >= 0),
    barcode                TEXT,
    position               INTEGER NOT NULL DEFAULT 0,
    is_default             BOOLEAN NOT NULL DEFAULT false,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT product_variants_tenant_sku_key UNIQUE (tenant_id, sku)
);

CREATE INDEX product_variants_product_id_idx ON product_variants (product_id);

-- At most one default variant per product.
CREATE UNIQUE INDEX product_variants_one_default_per_product
    ON product_variants (product_id) WHERE is_default;

CREATE TRIGGER product_variants_set_updated_at
    BEFORE UPDATE ON product_variants
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE product_images (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id  UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    url         TEXT NOT NULL,               -- browser / CDN URL
    storage_key TEXT NOT NULL,               -- S3 object key: products/<tenant>/<product>/<image>.<ext>
    alt         TEXT,
    position    INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT product_images_storage_key_key UNIQUE (storage_key)
);

CREATE INDEX product_images_product_id_idx ON product_images (product_id);

CREATE TABLE product_categories (
    product_id  UUID NOT NULL REFERENCES products(id)   ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, category_id)
);

CREATE INDEX product_categories_category_id_idx ON product_categories (category_id);
