ALTER TABLE orders
    DROP CONSTRAINT orders_payment_method_allowed,
    DROP CONSTRAINT orders_payment_status_allowed,
    DROP COLUMN payment_method,
    DROP COLUMN payment_status,
    DROP COLUMN payment_id,
    ADD COLUMN payment_ref TEXT;

ALTER TABLE orders DROP CONSTRAINT orders_status_allowed;
UPDATE orders SET status = 'PAID' WHERE status = 'CONFIRMED';
ALTER TABLE orders
    ADD CONSTRAINT orders_status_allowed
    CHECK (status IN ('PENDING_PAYMENT', 'PAID', 'FULFILLED', 'CANCELLED'));
