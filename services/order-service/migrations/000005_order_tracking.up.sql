-- 000005_order_tracking: carrier + tracking number captured at fulfilment.

ALTER TABLE orders
    ADD COLUMN tracking_carrier TEXT,
    ADD COLUMN tracking_number  TEXT;
