-- 000002_roles: the fixed role catalog.

CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT roles_name_key UNIQUE (name),
    CONSTRAINT roles_name_allowed CHECK (name IN ('SUPER_ADMIN', 'ADMIN', 'MANAGER', 'CUSTOMER'))
);

CREATE TRIGGER roles_set_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Seed the four roles the application expects to exist.
INSERT INTO roles (name, description) VALUES
    ('SUPER_ADMIN', 'Platform operator'),
    ('ADMIN',       'Tenant owner / administrator'),
    ('MANAGER',     'Staff member assigned by a tenant admin'),
    ('CUSTOMER',    'Shopper');
