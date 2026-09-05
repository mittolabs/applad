-- ---------------------------------------------------------------------------
-- Migration 043: Observe becomes Analytics.
--
-- Error monitoring is Bugslad's product, not Applad's, so the half of Observe
-- that is diagnostics leaves the platform: errors, logs, releases, replays,
-- alert rules and client-reported web vitals.
--
-- The half the platform measures about itself stays and moves under Analytics,
-- because a self-hoster must not need a second product to learn whether their
-- own cron fired or their own service is reachable: uptime monitors, cron
-- monitors and the request-latency snapshots written by the perf collector.
--
-- The tables are renamed rather than recreated so existing monitors, their
-- check history and the last 24h of latency survive the rename.
-- ---------------------------------------------------------------------------

-- ── Kept, renamed ────────────────────────────────────────────────────────────

ALTER TABLE IF EXISTS observe_uptime_monitors RENAME TO analytics_uptime_monitors;
ALTER TABLE IF EXISTS observe_uptime_checks   RENAME TO analytics_uptime_checks;
ALTER TABLE IF EXISTS observe_cron_monitors   RENAME TO analytics_cron_monitors;
ALTER TABLE IF EXISTS observe_cron_checkins   RENAME TO analytics_cron_checkins;
ALTER TABLE IF EXISTS observe_perf_snapshots  RENAME TO analytics_perf_snapshots;

-- A fresh install that never ran 003 has nothing to rename, so create what is
-- missing. Existing installs skip these entirely.

CREATE TABLE IF NOT EXISTS analytics_uptime_monitors (
    id             TEXT PRIMARY KEY,
    project_id     TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    url            TEXT NOT NULL,
    check_type     TEXT NOT NULL DEFAULT 'http', -- http | tcp | ping | keyword
    interval_secs  INT NOT NULL DEFAULT 60,
    keyword        TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'pending', -- pending | up | down | degraded | paused
    enabled        BOOLEAN NOT NULL DEFAULT TRUE,
    -- Nullable on purpose: a monitor that has never run has no uptime, and
    -- 100.0 is not a safe stand-in for "unknown" (see 021).
    uptime_pct     REAL,
    latency_ms     INT,
    last_checked   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS analytics_uptime_checks (
    id         TEXT PRIMARY KEY,
    monitor_id TEXT NOT NULL REFERENCES analytics_uptime_monitors(id) ON DELETE CASCADE,
    status     TEXT NOT NULL DEFAULT 'up',
    latency_ms INT NOT NULL DEFAULT 0,
    error_msg  TEXT NOT NULL DEFAULT '',
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS analytics_cron_monitors (
    id               TEXT PRIMARY KEY,
    project_id       TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name             TEXT NOT NULL,
    schedule         TEXT NOT NULL,   -- cron expression
    timezone         TEXT NOT NULL DEFAULT 'UTC',
    grace_period     INT NOT NULL DEFAULT 5,  -- minutes
    status           TEXT NOT NULL DEFAULT 'waiting', -- ok | missed | failed | running | waiting
    enabled          BOOLEAN NOT NULL DEFAULT TRUE,
    last_duration_ms INT,
    last_run_at      TIMESTAMPTZ,
    next_run_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS analytics_cron_checkins (
    id          TEXT PRIMARY KEY,
    monitor_id  TEXT NOT NULL REFERENCES analytics_cron_monitors(id) ON DELETE CASCADE,
    status      TEXT NOT NULL DEFAULT 'ok', -- ok | failed
    duration_ms INT,
    error_msg   TEXT NOT NULL DEFAULT '',
    checked_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS analytics_perf_snapshots (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    method      TEXT NOT NULL DEFAULT '',
    path        TEXT NOT NULL DEFAULT '',
    p50_ms      REAL NOT NULL DEFAULT 0,
    p75_ms      REAL NOT NULL DEFAULT 0,
    p95_ms      REAL NOT NULL DEFAULT 0,
    p99_ms      REAL NOT NULL DEFAULT 0,
    rps         REAL NOT NULL DEFAULT 0,
    error_pct   REAL NOT NULL DEFAULT 0,
    req_count   BIGINT NOT NULL DEFAULT 0,
    bucket_hour TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_uptime_project        ON analytics_uptime_monitors(project_id);
CREATE INDEX IF NOT EXISTS idx_uptime_checks_monitor ON analytics_uptime_checks(monitor_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_cron_project          ON analytics_cron_monitors(project_id);
CREATE INDEX IF NOT EXISTS idx_cron_checkins_monitor ON analytics_cron_checkins(monitor_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_perf_project_hour     ON analytics_perf_snapshots(project_id, bucket_hour DESC);

-- ── Removed: diagnostics, which is Bugslad's product ─────────────────────────
-- CASCADE takes the dependent activity/incident tables and their indexes with
-- the parents, so the order of these statements does not matter.

DROP TABLE IF EXISTS observe_error_activity   CASCADE;
DROP TABLE IF EXISTS observe_errors           CASCADE;
DROP TABLE IF EXISTS observe_logs             CASCADE;
DROP TABLE IF EXISTS observe_releases         CASCADE;
DROP TABLE IF EXISTS observe_replays          CASCADE;
DROP TABLE IF EXISTS observe_alert_incidents  CASCADE;
DROP TABLE IF EXISTS observe_alert_rules      CASCADE;
DROP TABLE IF EXISTS observe_web_vitals       CASCADE;
