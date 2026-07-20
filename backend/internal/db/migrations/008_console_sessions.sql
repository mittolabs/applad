-- Tracked sign-in sessions for console (admin) users, powering the account
-- Sessions tab. Console auth is otherwise a stateless JWT; the JWT now carries a
-- session id (sid) so sessions can be listed and revoked.

CREATE TABLE IF NOT EXISTS console_sessions (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES console_users(id) ON DELETE CASCADE,
    user_agent   TEXT NOT NULL DEFAULT '',
    ip           TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_console_sessions_user ON console_sessions(user_id, created_at DESC);
