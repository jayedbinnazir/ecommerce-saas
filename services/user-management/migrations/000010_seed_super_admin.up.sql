-- 000010_seed_super_admin: add users.phone, then bootstrap the platform
-- super-admin account.
--
-- The seed is idempotent — it does nothing if a user with this email already
-- exists. The password hash is bcrypt (cost 10), produced by pgcrypto's crypt();
-- the auth service verifies with golang.org/x/crypto/bcrypt, which accepts $2a$
-- hashes. Change the password after first login.

ALTER TABLE users ADD COLUMN IF NOT EXISTS phone TEXT;

INSERT INTO users (name, email, phone, password_hash, is_super_admin)
VALUES (
    'Jayed Bin Nazir',
    'jayed.official1998@gmail.com',
    '01521323469',
    crypt('super@admin1998', gen_salt('bf', 10)),
    true
)
ON CONFLICT (email) DO NOTHING;
