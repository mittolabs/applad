-- ---------------------------------------------------------------------------
-- Credentials Vault: extend credentials table + add per-credential access log
-- ---------------------------------------------------------------------------

-- key_version tracks which encryption key was used:
--   0 = JWT_SECRET (legacy, backward-compat)
--   1 = CREDENTIALS_ENCRYPTION_KEY (dedicated key)
ALTER TABLE credentials ADD COLUMN IF NOT EXISTS key_version  INTEGER     NOT NULL DEFAULT 0;
ALTER TABLE credentials ADD COLUMN IF NOT EXISTS expires_at   TIMESTAMPTZ;
ALTER TABLE credentials ADD COLUMN IF NOT EXISTS protected    BOOLEAN     NOT NULL DEFAULT FALSE;
ALTER TABLE credentials ADD COLUMN IF NOT EXISTS description  TEXT;

-- Per-credential access log — who read/wrote/deleted what and when.
CREATE TABLE IF NOT EXISTS credential_accesses (
    id            VARCHAR(36)  NOT NULL PRIMARY KEY,
    credential_id VARCHAR(36)  NOT NULL,
    project_id    VARCHAR(36)  NOT NULL,
    action        VARCHAR(32)  NOT NULL, -- 'create' | 'read' | 'update' | 'delete' | 'rotate'
    actor_id      VARCHAR(256),          -- user_id or api key id
    actor_type    VARCHAR(32),           -- 'user' | 'api_key'
    ip            VARCHAR(64),
    user_agent    TEXT,
    accessed_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ca_credential FOREIGN KEY (credential_id)
        REFERENCES credentials(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ca_credential  ON credential_accesses (credential_id);
CREATE INDEX IF NOT EXISTS idx_ca_project     ON credential_accesses (project_id);
CREATE INDEX IF NOT EXISTS idx_ca_accessed_at ON credential_accesses (accessed_at DESC);
