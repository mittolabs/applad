-- Data migrations: importing a project's users, databases, storage and functions
-- from another platform (another Applad instance, Appwrite, Supabase, NHost,
-- Firebase) into this project. One data_migrations row per import job; one
-- data_migration_resources row per non-bulk resource (and per failed bulk row)
-- so a job's progress is inspectable and a failed job can resume idempotently.
--
-- Named data_migrations (not migrations) to avoid any confusion with the
-- schema_migrations table that tracks applied SQL migrations.
CREATE TABLE IF NOT EXISTS data_migrations (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id   VARCHAR(36)  NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_type  VARCHAR(32)  NOT NULL,                 -- applad|appwrite|supabase|nhost|firebase
    status       VARCHAR(24)  NOT NULL DEFAULT 'pending', -- pending|running|completed|failed|cancelled
    groups       JSONB        NOT NULL DEFAULT '[]',     -- selected resource groups
    options      JSONB        NOT NULL DEFAULT '{}',     -- target databaseId, bucket mapping, etc.
    -- Source credentials, AES-256-GCM encrypted with the credential vault key.
    -- Cleared (set NULL) once the job reaches a terminal state so secrets do not
    -- linger after they are no longer needed.
    credentials  TEXT,
    counts       JSONB        NOT NULL DEFAULT '{}',     -- {"auth":{"total":N,"done":N,"error":N}, ...}
    error        TEXT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    started_at   TIMESTAMPTZ,
    finished_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_data_migrations_project ON data_migrations (project_id, created_at DESC);

CREATE TABLE IF NOT EXISTS data_migration_resources (
    id            VARCHAR(36)  NOT NULL PRIMARY KEY,
    migration_id  VARCHAR(36)  NOT NULL REFERENCES data_migrations(id) ON DELETE CASCADE,
    grp           VARCHAR(24)  NOT NULL,   -- auth|databases|storage|functions
    resource_type VARCHAR(48)  NOT NULL,   -- user|team|table|row|bucket|file|function|...
    source_id     TEXT         NOT NULL,
    dest_id       TEXT,
    status        VARCHAR(16)  NOT NULL DEFAULT 'pending', -- pending|done|warning|error|skip
    message       TEXT,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    -- Idempotent resume: re-running a job upserts the same logical resource.
    CONSTRAINT uq_dmr_resource UNIQUE (migration_id, grp, resource_type, source_id)
);
CREATE INDEX IF NOT EXISTS idx_dmr_migration ON data_migration_resources (migration_id, status);
