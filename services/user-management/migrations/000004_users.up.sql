-- 000004_users: platform users and their shipping/billing addresses.

CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT NOT NULL,
    email          CITEXT NOT NULL,
    password_hash  TEXT,                         -- NULL for OAuth-only accounts
    is_super_admin BOOLEAN NOT NULL DEFAULT false,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_email_key UNIQUE (email),
    CONSTRAINT users_name_not_blank CHECK (length(btrim(name)) > 0)
);

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE addresses (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label               TEXT,
    recipient_name      TEXT NOT NULL,
    phone               TEXT NOT NULL,
    address_line_1      TEXT NOT NULL,
    address_line_2      TEXT,
    city                TEXT NOT NULL,
    state               TEXT,
    postal_code         TEXT,
    country             TEXT NOT NULL,           -- ISO 3166-1 alpha-2
    is_default_shipping BOOLEAN NOT NULL DEFAULT false,
    is_default_billing  BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT addresses_country_iso2 CHECK (country ~ '^[A-Z]{2}$')
);

CREATE INDEX addresses_user_id_idx ON addresses (user_id);

-- At most one default shipping / billing address per user.
CREATE UNIQUE INDEX addresses_one_default_shipping_per_user
    ON addresses (user_id) WHERE is_default_shipping;
CREATE UNIQUE INDEX addresses_one_default_billing_per_user
    ON addresses (user_id) WHERE is_default_billing;

CREATE TRIGGER addresses_set_updated_at
    BEFORE UPDATE ON addresses
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
