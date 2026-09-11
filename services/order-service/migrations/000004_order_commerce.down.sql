DROP TABLE IF EXISTS coupons;
DROP TABLE IF EXISTS shipping_rates;
DROP TABLE IF EXISTS order_config;

DROP INDEX IF EXISTS orders_idempotency_key;

ALTER TABLE orders
    DROP CONSTRAINT IF EXISTS orders_money_nonneg,
    DROP COLUMN IF EXISTS shipping_address,
    DROP COLUMN IF EXISTS billing_address,
    DROP COLUMN IF EXISTS discount_cents,
    DROP COLUMN IF EXISTS shipping_cents,
    DROP COLUMN IF EXISTS tax_cents,
    DROP COLUMN IF EXISTS grand_total_cents,
    DROP COLUMN IF EXISTS coupon_code,
    DROP COLUMN IF EXISTS idempotency_key;
