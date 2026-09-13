-- 000003_permissions: the permission catalog and the role-level grant table.
-- membership_permissions (per-member grants) is created later, with memberships.

CREATE TABLE permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT permissions_name_key UNIQUE (name)
);

CREATE TRIGGER permissions_set_updated_at
    BEFORE UPDATE ON permissions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id)       ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX role_permissions_permission_id_idx ON role_permissions (permission_id);

-- A starter set of permissions a store admin can hand to managers.
INSERT INTO permissions (name, description) VALUES
    ('product:read',  'View products'),
    ('product:write', 'Create and edit products'),
    ('order:read',    'View orders'),
    ('order:write',   'Manage orders'),
    ('member:read',   'View tenant members'),
    ('member:write',  'Manage tenant members');
