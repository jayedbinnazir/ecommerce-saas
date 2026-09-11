-- 000004_payment_partial_refund: track how much of a payment has been refunded
-- (returns can refund only part of an order).

ALTER TABLE payments
    ADD COLUMN refunded_cents BIGINT NOT NULL DEFAULT 0,
    ADD CONSTRAINT payments_refunded_within_amount
        CHECK (refunded_cents >= 0 AND refunded_cents <= amount_cents);
