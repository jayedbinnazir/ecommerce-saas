ALTER TABLE payments
    DROP CONSTRAINT IF EXISTS payments_refunded_within_amount,
    DROP COLUMN IF EXISTS refunded_cents;
