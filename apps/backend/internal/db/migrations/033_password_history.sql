-- Password history, so a project that turns on the passwordHistory policy can
-- refuse a password the user has already used.
--
-- One row per successful password set (account creation, password change, and
-- password recovery), holding only the bcrypt hash. The candidate on a change
-- or recovery is bcrypt-compared against the most recent N rows; nothing here is
-- reversible, and a leaked row is no more useful than the user's current hash.
--
-- Rows are scoped by project as well as user so the table honours the same
-- tenant boundary as every other auth table, and are removed with the user.
CREATE TABLE IF NOT EXISTS password_history (
    id            VARCHAR(36)  NOT NULL PRIMARY KEY,
    user_id       VARCHAR(36)  NOT NULL,
    project_id    VARCHAR(36)  NOT NULL,
    password_hash TEXT         NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- The reuse check reads the newest rows for one user, so index by user and time.
CREATE INDEX IF NOT EXISTS idx_password_history_user
    ON password_history (user_id, created_at DESC);
