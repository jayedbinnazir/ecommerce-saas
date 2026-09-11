-- 000002_categories: per-tenant category tree.

CREATE TABLE categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL,                       -- reference to a user-management tenant
    parent_id  UUID REFERENCES categories(id) ON DELETE SET NULL,
    name       TEXT NOT NULL,
    slug       CITEXT NOT NULL,
    position   INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT categories_tenant_slug_key UNIQUE (tenant_id, slug),
    CONSTRAINT categories_slug_format CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    CONSTRAINT categories_not_self_parent CHECK (parent_id IS NULL OR parent_id <> id)
);

CREATE INDEX categories_tenant_id_idx ON categories (tenant_id);
CREATE INDEX categories_parent_id_idx ON categories (parent_id);

CREATE TRIGGER categories_set_updated_at
    BEFORE UPDATE ON categories
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
