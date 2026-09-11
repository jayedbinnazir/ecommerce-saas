-- 000002_notifications: in-app notifications, one row per user per event.

CREATE TABLE notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL,                        -- reference to a user-management user
    tenant_id  UUID,                                 -- optional store context
    type       TEXT NOT NULL,                        -- "order.confirmed", "order.shipped", ...
    title      TEXT NOT NULL,
    body       TEXT NOT NULL,
    data       JSONB NOT NULL DEFAULT '{}'::jsonb,   -- ids / links the UI can use
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- the user's feed, newest first
CREATE INDEX notifications_user_idx ON notifications (user_id, created_at DESC);
-- fast unread count / unread filter
CREATE INDEX notifications_user_unread_idx ON notifications (user_id) WHERE read_at IS NULL;
