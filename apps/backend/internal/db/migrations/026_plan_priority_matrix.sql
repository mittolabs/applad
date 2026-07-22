-- Migration 026: priority derived, not guessed
--
-- A single priority field inflates. Nothing costs anything to call urgent, so
-- everything becomes urgent and the word stops sorting anything. It also
-- conflates two facts that move independently: how much this matters, and how
-- soon it is needed.
--
-- Impact is a property of the problem; urgency is a property of the calendar.
-- High impact never falls below high because a deadline is soft — you may
-- defer it, you may not demote it. That asymmetry is what a grid expresses and
-- a single dropdown cannot.
--
-- The grid is per kind, because a defect's high × high deserves a different
-- verdict from a change's, and it resolves onto the scale Plan already speaks
-- rather than introducing a second vocabulary.
--
-- Items that predate this keep the priority they have and are marked as set
-- directly. Their impact and urgency stay null rather than fabricating answers
-- nobody ever gave.

ALTER TABLE plan_items
    ADD COLUMN IF NOT EXISTS priority_impact  SMALLINT,
    ADD COLUMN IF NOT EXISTS priority_urgency SMALLINT,
    -- A hand-set priority has to be visibly distinct from a derived one, or an
    -- override cannot be told from stale data.
    ADD COLUMN IF NOT EXISTS priority_is_manual BOOLEAN NOT NULL DEFAULT TRUE;

CREATE TABLE IF NOT EXISTS plan_priority_matrix (
    id         VARCHAR(36) NOT NULL PRIMARY KEY,
    project_id VARCHAR(36) NOT NULL,
    kind       VARCHAR(16) NOT NULL,
    impact     SMALLINT    NOT NULL,   -- 1 low, 2 medium, 3 high
    urgency    SMALLINT    NOT NULL,
    priority   VARCHAR(16) NOT NULL,   -- low | medium | high | urgent
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_matrix_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT uq_matrix_cell UNIQUE (project_id, kind, impact, urgency)
);
CREATE INDEX IF NOT EXISTS idx_matrix_project ON plan_priority_matrix (project_id, kind);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON plan_priority_matrix
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();
