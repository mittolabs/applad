-- Content mode: fold the standalone CMS into Databases. A table can opt into
-- editorial behaviour (draft/publish, per-locale entries, slugs, versions)
-- instead of Content being a separate product with its own storage.

ALTER TABLE tables ADD COLUMN IF NOT EXISTS content_enabled BOOLEAN NOT NULL DEFAULT FALSE;

-- Version snapshots for rows in content-enabled tables.
CREATE TABLE IF NOT EXISTS row_versions (
    id         TEXT PRIMARY KEY,
    table_id   TEXT NOT NULL,
    row_id     TEXT NOT NULL,
    version    INT  NOT NULL,
    data       JSONB NOT NULL DEFAULT '{}',
    author_id  TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_row_versions_row ON row_versions (table_id, row_id, version DESC);
