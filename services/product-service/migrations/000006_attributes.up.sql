-- 000006_attributes: per-category attribute definitions.
--
-- An attribute belongs to a category (every product in that category inherits it)
-- and has one of two roles:
--   VARIANT — the customer selects it; each combination is a product_variant with
--             its own SKU and price (e.g. "RAM: 8GB / 12GB / 16GB").
--   SPEC    — display only; shown on the product page (e.g. "Chipset").
--
-- Chosen values for VARIANT attributes come from product_attribute_values.
-- A product's SPEC values live in product_specs (one row per attribute).
-- product_variant_options ties one variant to one value per VARIANT attribute.

CREATE TABLE product_attributes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name        TEXT   NOT NULL,                 -- human label, "RAM"
    code        CITEXT NOT NULL,                 -- machine key, "ram"
    role        TEXT   NOT NULL DEFAULT 'SPEC',
    position    INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT product_attributes_role_allowed  CHECK (role IN ('VARIANT', 'SPEC')),
    CONSTRAINT product_attributes_code_format   CHECK (code ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    CONSTRAINT product_attributes_category_code_key UNIQUE (category_id, code)
);

CREATE INDEX product_attributes_category_idx ON product_attributes (category_id);

CREATE TRIGGER product_attributes_set_updated_at
    BEFORE UPDATE ON product_attributes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE product_attribute_values (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attribute_id UUID NOT NULL REFERENCES product_attributes(id) ON DELETE CASCADE,
    value        TEXT NOT NULL,                  -- "12GB"
    position     INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT product_attribute_values_unique       UNIQUE (attribute_id, value),
    -- lets product_variant_options reference (attribute_id, value_id) as a pair
    CONSTRAINT product_attribute_values_id_attr_key  UNIQUE (attribute_id, id)
);

CREATE INDEX product_attribute_values_attribute_idx ON product_attribute_values (attribute_id);

-- One product's display-spec values ("Chipset" = "Snapdragon 8 Gen 3").
CREATE TABLE product_specs (
    product_id   UUID NOT NULL REFERENCES products(id)           ON DELETE CASCADE,
    attribute_id UUID NOT NULL REFERENCES product_attributes(id) ON DELETE CASCADE,
    value        TEXT NOT NULL,
    PRIMARY KEY (product_id, attribute_id)
);

-- Ties a variant to its chosen value for one VARIANT attribute.
CREATE TABLE product_variant_options (
    variant_id         UUID NOT NULL REFERENCES product_variants(id)   ON DELETE CASCADE,
    attribute_id       UUID NOT NULL REFERENCES product_attributes(id) ON DELETE CASCADE,
    attribute_value_id UUID NOT NULL,
    PRIMARY KEY (variant_id, attribute_id),
    CONSTRAINT product_variant_options_value_fk
        FOREIGN KEY (attribute_id, attribute_value_id)
        REFERENCES product_attribute_values (attribute_id, id) ON DELETE CASCADE
);

CREATE INDEX product_variant_options_variant_idx ON product_variant_options (variant_id);
