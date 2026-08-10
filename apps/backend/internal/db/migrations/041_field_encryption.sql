-- Field-level encryption at rest: per-project data-encryption keys (DEKs),
-- enveloped by the server's MASTER_ENCRYPTION_KEY (see internal/dek), plus the
-- "encrypted" flag on arbitrary Databases columns (see internal/databases) and
-- the pre-existing buckets.encryption flag (see internal/storage), both of
-- which now encrypt under a project's own DEK instead of one global key.
--
-- A project may accumulate more than one row here over time: rotating a
-- project's DEK inserts a new active row and retires the old one rather than
-- replacing it in place, since existing ciphertext still needs the retired
-- key to decrypt (v1 does not re-encrypt existing data on rotation).

ALTER TABLE columns ADD COLUMN IF NOT EXISTS encrypted BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS project_encryption_keys (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id   VARCHAR(36)  NOT NULL,
    key_version  INT          NOT NULL DEFAULT 1,
    kek_version  INT          NOT NULL DEFAULT 1,
    wrapped_dek  TEXT         NOT NULL,
    status       VARCHAR(16)  NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_pek_project_version UNIQUE (project_id, key_version),
    CONSTRAINT fk_pek_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_pek_project_active ON project_encryption_keys (project_id) WHERE status = 'active';

CREATE TRIGGER set_updated_at BEFORE UPDATE ON project_encryption_keys
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();
