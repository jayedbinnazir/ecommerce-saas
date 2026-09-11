-- 000005_reviews: customer product reviews (one per customer per product).

CREATE TABLE product_reviews (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    product_id  UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL,                 -- user-management user id
    rating      SMALLINT NOT NULL,
    title       TEXT,
    body        TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT product_reviews_rating_range CHECK (rating BETWEEN 1 AND 5),
    CONSTRAINT product_reviews_one_per_customer UNIQUE (product_id, customer_id)
);

CREATE INDEX product_reviews_product_idx ON product_reviews (product_id, created_at DESC);

CREATE TRIGGER product_reviews_set_updated_at
    BEFORE UPDATE ON product_reviews
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
