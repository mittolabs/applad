-- ---------------------------------------------------------------------------
-- Migration 046: registered OAuth redirect targets, per platform.
--
-- An OAuth sign-in could only redirect back to a relative path on the API's own
-- origin, so the callback could never reach a phone: a native app is handed its
-- session on a custom scheme (funnier://auth) or an app link, both of which
-- carry a scheme and a host and were therefore refused.
--
-- Refusing them wholesale was the safe thing to do without a registry — an
-- OAuth endpoint that redirects anywhere is an open redirect, and a good one,
-- because it is reached before the user is authenticated. This adds the
-- registry: a target is allowed because the project registered it against a
-- platform, not because it looked plausible.
-- ---------------------------------------------------------------------------

ALTER TABLE platforms
    ADD COLUMN IF NOT EXISTS redirect_uris JSONB NOT NULL DEFAULT '[]'::jsonb;
