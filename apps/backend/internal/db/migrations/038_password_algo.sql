-- Multi-algorithm password support, the prerequisite for migrating users in from
-- other platforms without forcing a password reset.
--
-- A native Applad account is bcrypt. A user imported from Firebase, Appwrite,
-- Supabase, etc. arrives with a foreign hash we cannot reverse; we store the
-- hash together with the algorithm that produced it (and any per-user parameters
-- that algorithm needs, e.g. Firebase's salt/signer key). At the next successful
-- sign-in the login path verifies against the foreign algorithm and transparently
-- re-hashes to bcrypt, so imported accounts converge on the native scheme over
-- time and nothing but bcrypt is ever written by Applad itself.
--
-- Existing rows default to 'bcrypt' with NULL params, so behaviour is unchanged
-- for every account that predates this migration.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS password_algo   VARCHAR(32) NOT NULL DEFAULT 'bcrypt';

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS password_params JSONB;
