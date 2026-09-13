-- 000007_auth: external identities (Google/Facebook) and refresh tokens.

CREATE TABLE auth_identities (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         TEXT NOT NULL,
    provider_subject TEXT NOT NULL,           -- the provider's stable user id
    provider_email   TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT auth_identities_provider_subject_key UNIQUE (provider, provider_subject),
    CONSTRAINT auth_identities_provider_allowed CHECK (provider IN ('google', 'facebook'))
);

CREATE INDEX auth_identities_user_id_idx ON auth_identities (user_id);

CREATE TRIGGER auth_identities_set_updated_at
    BEFORE UPDATE ON auth_identities
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id UUID NOT NULL,
    family_id  UUID NOT NULL,
    token_hash TEXT NOT NULL,                 -- SHA-256 of the opaque token
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    user_agent TEXT,
    ip_address TEXT,
    CONSTRAINT refresh_tokens_token_hash_key UNIQUE (token_hash)
);

CREATE INDEX refresh_tokens_user_id_idx    ON refresh_tokens (user_id);
CREATE INDEX refresh_tokens_session_id_idx ON refresh_tokens (session_id);
CREATE INDEX refresh_tokens_family_id_idx  ON refresh_tokens (family_id);
