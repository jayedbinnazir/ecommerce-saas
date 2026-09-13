-- 000009_drop_billing: billing (plans + subscriptions) moved to payment-service.
-- user-management now checks entitlement over HTTP; these tables are dead here.

DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS plans;
