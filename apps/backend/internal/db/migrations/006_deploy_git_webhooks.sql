-- Migration 006: Git webhook triggers and preview deployments
--
-- Adds:
--   git_connections.webhook_secret   — HMAC-SHA256 key for verifying inbound push/PR events
--   deploy_releases.is_preview       — marks an ephemeral per-PR preview deploy
--   deploy_releases.preview_url      — generated URL for the preview deploy
--   deploy_releases.pr_number        — PR number that triggered the preview
--   deploy_releases.pr_branch        — source branch of the PR

ALTER TABLE git_connections
    ADD COLUMN IF NOT EXISTS webhook_secret VARCHAR(128);

ALTER TABLE deploy_releases
    ADD COLUMN IF NOT EXISTS is_preview  BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS preview_url VARCHAR(512),
    ADD COLUMN IF NOT EXISTS pr_number   INT,
    ADD COLUMN IF NOT EXISTS pr_branch   VARCHAR(128);

CREATE INDEX IF NOT EXISTS idx_dr_preview ON deploy_releases (target_id, is_preview)
    WHERE is_preview = TRUE;
