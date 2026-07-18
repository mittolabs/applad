-- Self-monitoring for the public status page (status.applad.io).
-- Applad probes its own components and stores the results here; the public
-- GET /v1/status endpoint aggregates them. Not tied to any project.

CREATE TABLE IF NOT EXISTS status_checks (
    id         TEXT PRIMARY KEY,
    component  TEXT NOT NULL,                 -- api | postgres | redis | storage | workers
    status     TEXT NOT NULL,                 -- operational | degraded | down
    latency_ms INT  NOT NULL DEFAULT 0,
    error_msg  TEXT NOT NULL DEFAULT '',
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_status_checks_comp ON status_checks(component, checked_at DESC);

CREATE TABLE IF NOT EXISTS status_incidents (
    id          TEXT PRIMARY KEY,
    component   TEXT NOT NULL,
    title       TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'investigating', -- investigating | resolved
    severity    TEXT NOT NULL DEFAULT 'major',         -- minor | major | critical
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_status_incidents_started ON status_incidents(started_at DESC);
