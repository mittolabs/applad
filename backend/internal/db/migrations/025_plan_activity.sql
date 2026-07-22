-- Migration 025: what happened to an item
--
-- Comments say what people think; activity says what changed. Keeping them
-- apart is why a history tab is worth having: a status that moved three times
-- in a day is a fact about the work, and burying it in the discussion makes
-- both harder to read.

CREATE TABLE IF NOT EXISTS plan_activity (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    item_id    VARCHAR(36)  NOT NULL,
    actor_id   VARCHAR(36),
    -- The field that changed, in the API's own words, so the console can
    -- render it without a second vocabulary to keep in step.
    field      VARCHAR(64)  NOT NULL,
    old_value  TEXT         NOT NULL DEFAULT '',
    new_value  TEXT         NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_activity_item FOREIGN KEY (item_id) REFERENCES plan_items(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_activity_item ON plan_activity (item_id, created_at DESC);
