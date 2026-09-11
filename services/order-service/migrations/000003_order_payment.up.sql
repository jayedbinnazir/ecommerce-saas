-- 000003_order_payment: payment-service integration.
--   * order status PAID becomes CONFIRMED (money now lives in payment-service)
--   * payment_method / payment_status / payment_id track the payment-service record

ALTER TABLE orders DROP CONSTRAINT orders_status_allowed;
UPDATE orders SET status = 'CONFIRMED' WHERE status = 'PAID';
ALTER TABLE orders
    ADD CONSTRAINT orders_status_allowed
    CHECK (status IN ('PENDING_PAYMENT', 'CONFIRMED', 'FULFILLED', 'CANCELLED'));

ALTER TABLE orders
    DROP COLUMN payment_ref,
    ADD COLUMN payment_method TEXT,
    ADD COLUMN payment_status TEXT NOT NULL DEFAULT 'PENDING',
    ADD COLUMN payment_id     UUID,
    ADD CONSTRAINT orders_payment_method_allowed
        CHECK (payment_method IS NULL OR payment_method IN ('CARD', 'COD')),
    ADD CONSTRAINT orders_payment_status_allowed
        CHECK (payment_status IN ('PENDING', 'PAID', 'REFUNDED', 'FAILED'));
