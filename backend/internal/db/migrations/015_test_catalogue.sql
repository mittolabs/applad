-- Testing, modelled the way QA thinks about it.
--
-- The first cut conflated two different things under "suite": how to execute
-- tests (an image and a command) and which tests to run (a selection). Because
-- every recording got its own, "suite" came to mean "one test", so nothing
-- could be grouped, tagged or selected, and each recording appeared twice.
--
-- Now:
--   runner  — how a body of tests is executed
--   test    — one checkable behaviour, recorded or discovered by running
--   suite   — a named selection of tests, and when it should run
--   run     — a suite executed against a target, at a moment
--   case    — what one test did in one run

-- "How to execute" keeps the old table's shape under its real name.
ALTER TABLE test_suites RENAME TO test_runners;

-- The catalogue. Recorded flows create an entry directly; authored tests are
-- discovered the first time a run reports them, which is the only moment their
-- names are known.
CREATE TABLE IF NOT EXISTS tests (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    runner_id   VARCHAR(36)  NOT NULL,
    -- suite_name is the runner's own grouping (a file, a describe block), kept
    -- so a catalogue of hundreds stays navigable.
    suite_name  VARCHAR(256) NOT NULL DEFAULT '',
    name        VARCHAR(512) NOT NULL,
    source      VARCHAR(32)  NOT NULL DEFAULT 'discovered',  -- recorded | discovered
    flow_id     VARCHAR(36),
    tags        JSONB        NOT NULL DEFAULT '[]',
    -- A quarantined test still runs and still reports, but no longer decides
    -- whether the run passed. It is how a known-flaky test stops blocking a
    -- deploy without being deleted and forgotten.
    quarantined BOOLEAN      NOT NULL DEFAULT FALSE,
    last_status VARCHAR(32),
    last_run_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_t_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT fk_t_runner  FOREIGN KEY (runner_id)  REFERENCES test_runners(id) ON DELETE CASCADE
);

-- One behaviour is one row, however many times it runs.
CREATE UNIQUE INDEX IF NOT EXISTS idx_tests_identity
    ON tests (project_id, runner_id, suite_name, name);
CREATE INDEX IF NOT EXISTS idx_tests_project ON tests (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON tests
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

-- A selection, and when it runs. "smoke = @smoke, on every deploy" is the
-- point of this table.
CREATE TABLE IF NOT EXISTS test_suites (
    id            VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id    VARCHAR(36)  NOT NULL,
    name          VARCHAR(128) NOT NULL,
    -- Empty selects everything the runner has; otherwise a test must carry one
    -- of these tags.
    tags          JSONB        NOT NULL DEFAULT '[]',
    runner_id     VARCHAR(36),
    -- Where to run it. A run may override this, which is what makes testing a
    -- branch and testing main the same suite rather than two.
    default_target VARCHAR(512) NOT NULL DEFAULT '',
    run_on_deploy BOOLEAN      NOT NULL DEFAULT FALSE,
    cron          VARCHAR(256),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_tsu_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_tsu_project ON test_suites (project_id);
CREATE INDEX IF NOT EXISTS idx_tsu_deploy ON test_suites (project_id) WHERE run_on_deploy;

CREATE TRIGGER set_updated_at BEFORE UPDATE ON test_suites
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

-- Runs point at the runner that executed them, optionally the suite that
-- selected them, and record the target so a result is never ambiguous about
-- what it was testing.
ALTER TABLE test_runs RENAME COLUMN suite_id TO runner_id;
ALTER TABLE test_runs ADD COLUMN IF NOT EXISTS suite_id VARCHAR(36);
ALTER TABLE test_runs ADD COLUMN IF NOT EXISTS target_url VARCHAR(512) NOT NULL DEFAULT '';
ALTER TABLE test_runs ADD COLUMN IF NOT EXISTS flaky INT NOT NULL DEFAULT 0;
ALTER TABLE test_runs ADD COLUMN IF NOT EXISTS quarantined INT NOT NULL DEFAULT 0;

-- A result belongs to a catalogue entry, which is what makes history possible.
ALTER TABLE test_cases ADD COLUMN IF NOT EXISTS test_id VARCHAR(36);
-- Set when a test failed and then passed on retry within the same run: the
-- definition of flaky that does not require guessing.
ALTER TABLE test_cases ADD COLUMN IF NOT EXISTS flaky BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE test_cases ADD COLUMN IF NOT EXISTS retries INT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_tc_test ON test_cases (test_id, created_at DESC);

-- Recorded flows share one generated runner rather than one each, which is
-- what removed the duplicate listing.
ALTER TABLE test_flows RENAME COLUMN suite_id TO runner_id;
ALTER TABLE test_flows ADD COLUMN IF NOT EXISTS test_id VARCHAR(36);
