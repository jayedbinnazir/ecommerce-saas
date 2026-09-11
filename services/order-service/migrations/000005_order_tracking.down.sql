ALTER TABLE orders
    DROP COLUMN IF EXISTS tracking_carrier,
    DROP COLUMN IF EXISTS tracking_number;
