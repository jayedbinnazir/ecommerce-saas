-- 000002_mails: an audit log of every email the platform attempted to send.

CREATE TABLE mails (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    to_addr    TEXT NOT NULL,
    subject    TEXT NOT NULL,
    template   TEXT NOT NULL,
    status     TEXT NOT NULL,
    error      TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT mails_status_allowed CHECK (status IN ('SENT', 'FAILED'))
);

CREATE INDEX mails_to_addr_idx ON mails (to_addr, created_at DESC);
