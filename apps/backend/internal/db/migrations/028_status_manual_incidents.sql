-- Operator-reported incidents on the public status page.
--
-- The status checker auto-opens an incident when a component it probes goes
-- unhealthy (origin='auto') and resolves it on recovery. Operators also post
-- their own from the backoffice operator plane (origin='manual') — planned
-- maintenance, a partial outage the probes can't see, a third-party dependency.
--
-- The reconciler only ever resolves what it opened (origin='auto'), so a manual
-- incident is never auto-closed out from under an operator; an operator resolves
-- it explicitly. `message` carries the human detail shown under the title on
-- status.applad.io.
ALTER TABLE status_incidents ADD COLUMN IF NOT EXISTS origin  TEXT NOT NULL DEFAULT 'auto';
ALTER TABLE status_incidents ADD COLUMN IF NOT EXISTS message TEXT NOT NULL DEFAULT '';
