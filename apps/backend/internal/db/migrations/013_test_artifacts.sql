-- What a test run left behind: videos, screenshots, traces.
--
-- A failing assertion message tells you what broke; a recording tells you what
-- the user would have seen. For browser and device runs the recording is the
-- primary evidence, so it is stored per run and, where the runner names its
-- output after the test, attached to the case it belongs to.

ALTER TABLE test_suites
    ADD COLUMN IF NOT EXISTS artifacts_path VARCHAR(256) NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS test_artifacts (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    run_id       VARCHAR(36)  NOT NULL,
    project_id   VARCHAR(36)  NOT NULL,
    -- Set when the artifact could be traced to one test; run-level otherwise.
    case_id      VARCHAR(36),
    kind         VARCHAR(32)  NOT NULL,   -- video | screenshot | trace | report | other
    name         VARCHAR(512) NOT NULL,   -- path as the runner wrote it
    content_type VARCHAR(128) NOT NULL,
    size_bytes   BIGINT       NOT NULL DEFAULT 0,
    -- Location on the storage volume, relative to the artifacts root.
    storage_path VARCHAR(512) NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ta_run FOREIGN KEY (run_id) REFERENCES test_runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ta_run ON test_artifacts (run_id);
CREATE INDEX IF NOT EXISTS idx_ta_case ON test_artifacts (case_id) WHERE case_id IS NOT NULL;
