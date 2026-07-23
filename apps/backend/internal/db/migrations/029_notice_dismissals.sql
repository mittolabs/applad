-- A dismissed banner stays dismissed.
--
-- Dismissal is per USER, not per organization: a notice is shown to everyone in
-- an org, and one person clearing it from their own view must not clear it from
-- everyone else's. Keyed by the notice id supplied by whatever provides
-- entitlements, so core needs to know nothing about where notices come from.
CREATE TABLE IF NOT EXISTS notice_dismissals (
    user_id      VARCHAR(36)  NOT NULL,
    notice_id    TEXT         NOT NULL,
    dismissed_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, notice_id)
);
CREATE INDEX IF NOT EXISTS idx_notice_dismissals_user ON notice_dismissals (user_id);
