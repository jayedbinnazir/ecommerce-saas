-- 000006_memberships: user <-> tenant <-> role, plus per-member permission grants.

CREATE TABLE memberships (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles(id)   ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT memberships_tenant_user_key UNIQUE (tenant_id, user_id)
);

CREATE INDEX memberships_user_id_idx ON memberships (user_id);
CREATE INDEX memberships_role_id_idx ON memberships (role_id);

CREATE TRIGGER memberships_set_updated_at
    BEFORE UPDATE ON memberships
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE membership_permissions (
    membership_id UUID NOT NULL REFERENCES memberships(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (membership_id, permission_id)
);

CREATE INDEX membership_permissions_permission_id_idx ON membership_permissions (permission_id);
