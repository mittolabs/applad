-- Migration 021: an unchecked monitor has no uptime
--
-- uptime_pct defaulted to 100.0 NOT NULL and status to 'up', so a monitor was
-- born claiming perfect availability for a URL nobody had fetched yet. The
-- console then rendered that as "100.00%" in green — a fabricated SLA figure,
-- which is the worst version of a UI filling a gap with the happy value.
--
-- Unmeasured is now NULL, and rows that have never been checked are corrected
-- rather than left asserting a number they never earned.

ALTER TABLE observe_uptime_monitors
    ALTER COLUMN uptime_pct DROP DEFAULT,
    ALTER COLUMN uptime_pct DROP NOT NULL,
    ALTER COLUMN latency_ms DROP DEFAULT,
    ALTER COLUMN latency_ms DROP NOT NULL,
    ALTER COLUMN status SET DEFAULT 'pending';

UPDATE observe_uptime_monitors
   SET uptime_pct = NULL,
       latency_ms = NULL,
       status     = 'pending'
 WHERE last_checked IS NULL;
