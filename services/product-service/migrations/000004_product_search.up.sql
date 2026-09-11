-- 000004_product_search: full-text search over product name + description.

ALTER TABLE products
    ADD COLUMN search tsvector
    GENERATED ALWAYS AS (
        to_tsvector('simple', coalesce(name, '') || ' ' || coalesce(description, ''))
    ) STORED;

CREATE INDEX products_search_idx ON products USING GIN (search);
