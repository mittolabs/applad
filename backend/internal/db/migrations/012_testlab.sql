-- Test runs.
--
-- Applad does not know what Jest, pytest or `flutter test` are, and should not
-- have to: a suite says how to run itself and where it leaves a JUnit XML
-- report, and that report is the interchange format. Nearly every framework
-- emits it natively or with a one-line reporter, so supporting a new stack is
-- configuration rather than code.

-- How to run a project's tests.
CREATE TABLE IF NOT EXISTS test_suites (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id   VARCHAR(36)  NOT NULL,
    name         VARCHAR(128) NOT NULL,
    source_type  VARCHAR(32)  NOT NULL DEFAULT 'upload',  -- 'git' | 'upload'
    source_url   VARCHAR(512),
    branch       VARCHAR(128),
    -- Base image the suite runs in, e.g. node:20-alpine. Inferred from the
    -- source when left empty.
    image        VARCHAR(256),
    setup_cmd    VARCHAR(512),   -- e.g. npm ci
    command      VARCHAR(512) NOT NULL,   -- e.g. npm test
    -- Where the JUnit XML lands, relative to the project root. Globs allowed,
    -- since many runners write one file per shard.
    report_path  VARCHAR(256) NOT NULL DEFAULT 'junit.xml',
    env_vars     JSONB,
    timeout_ms   INT          NOT NULL DEFAULT 900000,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ts_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ts_project ON test_suites (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON test_suites
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

-- One execution of a suite.
CREATE TABLE IF NOT EXISTS test_runs (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id   VARCHAR(36)  NOT NULL,
    suite_id     VARCHAR(36)  NOT NULL,
    status       VARCHAR(32)  NOT NULL DEFAULT 'queued',
                 -- queued | running | passed | failed | errored | cancelled
    -- Where the run happened. Containers today; an emulator or a browser is
    -- another target rather than another table.
    target       VARCHAR(64)  NOT NULL DEFAULT 'container',
    trigger_type VARCHAR(32)  NOT NULL DEFAULT 'manual',
    trigger_actor VARCHAR(128),
    commit_sha   VARCHAR(64),
    total        INT          NOT NULL DEFAULT 0,
    passed       INT          NOT NULL DEFAULT 0,
    failed       INT          NOT NULL DEFAULT 0,
    skipped      INT          NOT NULL DEFAULT 0,
    duration_ms  BIGINT       NOT NULL DEFAULT 0,
    log          TEXT,        -- build and run output, as a person would read it
    error        TEXT,        -- why the run itself failed, distinct from failing tests
    started_at   TIMESTAMPTZ,
    finished_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_tr_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT fk_tr_suite   FOREIGN KEY (suite_id)   REFERENCES test_suites(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_tr_project ON test_runs (project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tr_suite   ON test_runs (suite_id, created_at DESC);

-- One test case within a run. This is the addressable unit: history and
-- flakiness are read across runs by (suite_name, name), and the Spec module
-- will attach an example to a case through spec_ref.
CREATE TABLE IF NOT EXISTS test_cases (
    id              VARCHAR(36)  NOT NULL PRIMARY KEY,
    run_id          VARCHAR(36)  NOT NULL,
    project_id      VARCHAR(36)  NOT NULL,
    suite_name      VARCHAR(256) NOT NULL,   -- JUnit classname
    name            VARCHAR(512) NOT NULL,
    status          VARCHAR(32)  NOT NULL,   -- passed | failed | skipped | errored
    duration_ms     BIGINT       NOT NULL DEFAULT 0,
    failure_message TEXT,
    failure_details TEXT,
    -- Reserved: which specification example this case verifies, once specs
    -- exist. Kept here from the start so grounding needs no migration.
    spec_ref        VARCHAR(512),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_tc_run FOREIGN KEY (run_id) REFERENCES test_runs(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_tc_run ON test_cases (run_id);
CREATE INDEX IF NOT EXISTS idx_tc_history ON test_cases (project_id, suite_name, name, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tc_spec ON test_cases (spec_ref) WHERE spec_ref IS NOT NULL;
