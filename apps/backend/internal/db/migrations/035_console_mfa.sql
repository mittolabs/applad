-- Console-admin MFA. The console owner controls every project and all data on a
-- self-hosted instance, so the account that matters most had no second factor.
-- These columns give a console user the same TOTP enrolment the per-project auth
-- users already have.
--
-- mfa_secret and mfa_recovery are NULL until enrolment begins, and mfa_enabled
-- stays false until a first valid code is verified, so a half-finished enrolment
-- never locks anyone out. Disabling clears the secret and codes.
ALTER TABLE console_users
    ADD COLUMN IF NOT EXISTS mfa_enabled  BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS mfa_secret   TEXT,
    ADD COLUMN IF NOT EXISTS mfa_recovery TEXT;
