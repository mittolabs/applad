-- Phone OTP is a pre-authentication flow: at first sign-in there is no user row
-- yet, so the code cannot live in auth_tokens (whose user_id is FK-constrained
-- to users.id, which is why the previous phone-OTP insert failed at runtime). A
-- dedicated table keyed by phone, with an attempt counter that bounds brute
-- force of the 6-digit code.
CREATE TABLE IF NOT EXISTS phone_otps (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id VARCHAR(36)  NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    phone      VARCHAR(32)  NOT NULL,
    code       VARCHAR(12)  NOT NULL,
    attempts   INT          NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_phone_otps_lookup ON phone_otps (project_id, phone);
