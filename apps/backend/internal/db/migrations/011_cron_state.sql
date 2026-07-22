-- Scheduler state, one row per scheduled thing.
--
-- The cron worker used to ask "does the current minute match this expression?"
-- once a minute. Anything due while the worker was down was lost silently, and
-- two workers would both fire the same job. Tracking the next due time instead
-- makes a run something that is owed rather than something that must be
-- observed at the right instant.
--
-- It also answers "what is scheduled in this project, when did it last run,
-- when does it run next", which nothing could answer before.
CREATE TABLE IF NOT EXISTS cron_state (
    kind        VARCHAR(32)  NOT NULL,   -- 'workflow' | 'function' | 'deploy_target'
    entity_id   VARCHAR(36)  NOT NULL,
    project_id  VARCHAR(36)  NOT NULL,
    expression  VARCHAR(256) NOT NULL,
    -- Set when the expression cannot be parsed, so a broken schedule surfaces
    -- instead of just never running.
    parse_error VARCHAR(512),
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    missed_runs INT          NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (kind, entity_id)
);

CREATE INDEX IF NOT EXISTS idx_cron_state_project ON cron_state (project_id);
CREATE INDEX IF NOT EXISTS idx_cron_state_due ON cron_state (next_run_at)
    WHERE parse_error IS NULL;
