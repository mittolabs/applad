-- Per-project OAuth config outgrew its two-column shape. Providers such as
-- Microsoft (tenant id) and Apple (key id, team id) carry auxiliary,
-- non-secret identifiers the console was collecting and then silently dropping,
-- because only client_id and client_secret had a home. `extra` is that home: a
-- JSON object of provider-specific, non-secret fields.
--
-- The client secret itself keeps living in client_secret, but is now written as
-- an AES-256-GCM token (see internal/credentials, keyed by
-- CREDENTIALS_ENCRYPTION_KEY) instead of plaintext, so a database dump no longer
-- discloses it. Apple has no static secret at all: its .p8 private key is stored
-- there encrypted and a short-lived ES256 JWT is signed from it at token
-- exchange time. Existing plaintext rows still decrypt (the reader treats an
-- unmarked value as legacy plaintext), so this migration needs no data rewrite.
ALTER TABLE project_oauth_providers
    ADD COLUMN IF NOT EXISTS extra JSONB;
