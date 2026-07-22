-- ---------------------------------------------------------------------------
-- Migration 003: Observe — errors, logs, performance, releases, replays,
--                          uptime, crons, and alerts.
-- ---------------------------------------------------------------------------

-- ── Errors ──────────────────────────────────────────────────────────────────
-- One row per unique error fingerprint (title + type + project).
-- Individual occurrences update the counters and last_seen.

CREATE TABLE IF NOT EXISTS observe_errors (
    id            TEXT PRIMARY KEY,
    project_id    TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title         TEXT NOT NULL,
    error_type    TEXT NOT NULL DEFAULT '',
    level         TEXT NOT NULL DEFAULT 'error', -- fatal | error | warning | info
    status        TEXT NOT NULL DEFAULT 'unresolved', -- unresolved | resolved | ignored
    fingerprint   TEXT NOT NULL,
    stack_trace   TEXT NOT NULL DEFAULT '',
    breadcrumbs   JSONB NOT NULL DEFAULT '[]',
    user_context  JSONB NOT NULL DEFAULT '{}',
    request_ctx   JSONB NOT NULL DEFAULT '{}',
    runtime_ctx   JSONB NOT NULL DEFAULT '{}',
    tags          JSONB NOT NULL DEFAULT '{}',
    environment   TEXT NOT NULL DEFAULT 'production',
    release       TEXT NOT NULL DEFAULT '',
    count         BIGINT NOT NULL DEFAULT 1,
    affected_users BIGINT NOT NULL DEFAULT 0,
    priority      TEXT NOT NULL DEFAULT '',  -- P1 | P2 | P3 | P4
    assignee      TEXT NOT NULL DEFAULT '',
    first_seen    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, fingerprint)
);

CREATE INDEX IF NOT EXISTS idx_observe_errors_project ON observe_errors(project_id);
CREATE INDEX IF NOT EXISTS idx_observe_errors_status  ON observe_errors(project_id, status);
CREATE INDEX IF NOT EXISTS idx_observe_errors_level   ON observe_errors(project_id, level);
CREATE INDEX IF NOT EXISTS idx_observe_errors_seen    ON observe_errors(project_id, last_seen DESC);

-- Activity feed for each error (resolve/ignore/comment events)
CREATE TABLE IF NOT EXISTS observe_error_activity (
    id         TEXT PRIMARY KEY,
    error_id   TEXT NOT NULL REFERENCES observe_errors(id) ON DELETE CASCADE,
    type       TEXT NOT NULL, -- note | resolved | ignored | assigned | unresolved
    user_name  TEXT NOT NULL DEFAULT 'System',
    text       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Logs ────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS observe_logs (
    id         TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    level      TEXT NOT NULL DEFAULT 'info', -- debug | info | warn | error | fatal
    message    TEXT NOT NULL,
    source     TEXT NOT NULL DEFAULT '',
    environment TEXT NOT NULL DEFAULT '',
    release    TEXT NOT NULL DEFAULT '',
    meta       JSONB NOT NULL DEFAULT '{}',
    trace_id   TEXT NOT NULL DEFAULT '',
    span_id    TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_observe_logs_project ON observe_logs(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_observe_logs_level   ON observe_logs(project_id, level);

-- ── Performance ──────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS observe_perf_snapshots (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    method      TEXT NOT NULL DEFAULT 'GET',
    path        TEXT NOT NULL DEFAULT '',
    p50_ms      REAL NOT NULL DEFAULT 0,
    p75_ms      REAL NOT NULL DEFAULT 0,
    p95_ms      REAL NOT NULL DEFAULT 0,
    p99_ms      REAL NOT NULL DEFAULT 0,
    rps         REAL NOT NULL DEFAULT 0,
    error_pct   REAL NOT NULL DEFAULT 0,
    req_count   BIGINT NOT NULL DEFAULT 0,
    bucket_hour TIMESTAMPTZ NOT NULL DEFAULT DATE_TRUNC('hour', NOW())
);

CREATE INDEX IF NOT EXISTS idx_perf_project_hour ON observe_perf_snapshots(project_id, bucket_hour DESC);

-- Web Vitals per page/session
CREATE TABLE IF NOT EXISTS observe_web_vitals (
    id         TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    page_url   TEXT NOT NULL DEFAULT '',
    lcp_ms     REAL,
    fid_ms     REAL,
    cls_score  REAL,
    ttfb_ms    REAL,
    fcp_ms     REAL,
    inp_ms     REAL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_vitals_project ON observe_web_vitals(project_id, created_at DESC);

-- ── Releases ─────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS observe_releases (
    id                      TEXT PRIMARY KEY,
    project_id              TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    version                 TEXT NOT NULL,
    environment             TEXT NOT NULL DEFAULT 'production',
    commit_count            INT NOT NULL DEFAULT 0,
    commits                 JSONB NOT NULL DEFAULT '[]',
    crash_free_sessions_pct REAL NOT NULL DEFAULT 100.0,
    new_issues              INT NOT NULL DEFAULT 0,
    regressed_issues        INT NOT NULL DEFAULT 0,
    fixed_issues            INT NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deployed_at             TIMESTAMPTZ,
    UNIQUE(project_id, version, environment)
);

CREATE INDEX IF NOT EXISTS idx_observe_releases_project ON observe_releases(project_id, created_at DESC);

-- ── Session Replays ───────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS observe_replays (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    session_id      TEXT NOT NULL DEFAULT '',
    user_id         TEXT NOT NULL DEFAULT '',
    user_name       TEXT NOT NULL DEFAULT 'Anonymous',
    url             TEXT NOT NULL DEFAULT '',
    browser         TEXT NOT NULL DEFAULT '',
    os              TEXT NOT NULL DEFAULT '',
    country         TEXT NOT NULL DEFAULT '',
    duration_secs   INT NOT NULL DEFAULT 0,
    error_count     INT NOT NULL DEFAULT 0,
    has_rage_click  BOOLEAN NOT NULL DEFAULT FALSE,
    has_dead_click  BOOLEAN NOT NULL DEFAULT FALSE,
    events          JSONB NOT NULL DEFAULT '[]',
    network         JSONB NOT NULL DEFAULT '[]',
    console         JSONB NOT NULL DEFAULT '[]',
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_observe_replays_project ON observe_replays(project_id, started_at DESC);

-- ── Uptime Monitors ───────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS observe_uptime_monitors (
    id             TEXT PRIMARY KEY,
    project_id     TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    url            TEXT NOT NULL,
    check_type     TEXT NOT NULL DEFAULT 'http', -- http | tcp | ping | keyword
    interval_secs  INT NOT NULL DEFAULT 60,
    keyword        TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'up',   -- up | down | degraded | paused
    enabled        BOOLEAN NOT NULL DEFAULT TRUE,
    uptime_pct     REAL NOT NULL DEFAULT 100.0,
    latency_ms     INT NOT NULL DEFAULT 0,
    last_checked   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_uptime_project ON observe_uptime_monitors(project_id);

CREATE TABLE IF NOT EXISTS observe_uptime_checks (
    id         TEXT PRIMARY KEY,
    monitor_id TEXT NOT NULL REFERENCES observe_uptime_monitors(id) ON DELETE CASCADE,
    status     TEXT NOT NULL DEFAULT 'up',
    latency_ms INT NOT NULL DEFAULT 0,
    error_msg  TEXT NOT NULL DEFAULT '',
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_uptime_checks_monitor ON observe_uptime_checks(monitor_id, checked_at DESC);

-- ── Cron Monitors ─────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS observe_cron_monitors (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    schedule        TEXT NOT NULL,   -- cron expression
    timezone        TEXT NOT NULL DEFAULT 'UTC',
    grace_period    INT NOT NULL DEFAULT 5,  -- minutes
    status          TEXT NOT NULL DEFAULT 'waiting', -- ok | missed | failed | running | waiting
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    last_duration_ms INT,
    last_run_at     TIMESTAMPTZ,
    next_run_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cron_project ON observe_cron_monitors(project_id);

CREATE TABLE IF NOT EXISTS observe_cron_checkins (
    id         TEXT PRIMARY KEY,
    monitor_id TEXT NOT NULL REFERENCES observe_cron_monitors(id) ON DELETE CASCADE,
    status     TEXT NOT NULL DEFAULT 'ok', -- ok | failed
    duration_ms INT,
    error_msg  TEXT NOT NULL DEFAULT '',
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cron_checkins_monitor ON observe_cron_checkins(monitor_id, checked_at DESC);

-- ── Alert Rules ───────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS observe_alert_rules (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    metric      TEXT NOT NULL,    -- error_rate | p95_latency | uptime | ...
    operator    TEXT NOT NULL,    -- gt | lt | gte | lte
    threshold   REAL NOT NULL,
    time_window TEXT NOT NULL DEFAULT '5m',
    severity    TEXT NOT NULL DEFAULT 'warning', -- info | warning | critical
    channel     TEXT NOT NULL DEFAULT 'email',   -- email | slack | webhook | pagerduty
    channel_cfg JSONB NOT NULL DEFAULT '{}',
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    last_fired  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alert_rules_project ON observe_alert_rules(project_id);

CREATE TABLE IF NOT EXISTS observe_alert_incidents (
    id          TEXT PRIMARY KEY,
    rule_id     TEXT NOT NULL REFERENCES observe_alert_rules(id) ON DELETE CASCADE,
    project_id  TEXT NOT NULL,
    rule_name   TEXT NOT NULL,
    severity    TEXT NOT NULL,
    value       REAL NOT NULL,
    fired_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_alert_incidents_project ON observe_alert_incidents(project_id, fired_at DESC);
