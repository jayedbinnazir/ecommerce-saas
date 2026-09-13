-- Subscription renewal support: dedupe key for retried/duplicate renewal
-- requests, mirroring orders.idempotency_key.
ALTER TABLE subscriptions ADD COLUMN last_renewal_key TEXT;
