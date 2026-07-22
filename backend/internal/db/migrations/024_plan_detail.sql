-- Migration 024: what an item carries, and what contains it
--
-- Acceptance criteria live on the item rather than in the specification,
-- because they come first: they are the constraints the work is agreed
-- against, written when nobody has yet decided how the behaviour will be
-- expressed. A Rule in a specification is a criterion made executable, so a
-- criterion carries the reference to the rule it became — and one with no
-- reference is visibly still just an intention.
--
-- Kind separates a change from a defect. Not a taxonomy of ticket types: two
-- values with one real difference, which is whether the system already
-- claimed to do this. A defect is a promise already broken; a change is a
-- promise not yet made.

ALTER TABLE plan_items
    ADD COLUMN IF NOT EXISTS kind         VARCHAR(16)  NOT NULL DEFAULT 'change',
    ADD COLUMN IF NOT EXISTS milestone_id VARCHAR(36),
    ADD COLUMN IF NOT EXISTS target_date  DATE,
    ADD COLUMN IF NOT EXISTS estimate     NUMERIC(6,2);

CREATE TABLE IF NOT EXISTS plan_milestones (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    name        VARCHAR(256) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    -- What it is aimed at. A roadmap needs a time axis, and this is it.
    target_date DATE,
    -- Set when the milestone is called done. Progress is otherwise derived
    -- from the items in it rather than typed by anyone: a percentage nobody
    -- measured is the kind of number this project has spent the day removing.
    completed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ms_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ms_project ON plan_milestones (project_id, target_date);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON plan_milestones
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

-- Acceptance criteria: what has to be true for the item to be done.
CREATE TABLE IF NOT EXISTS plan_criteria (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    item_id    VARCHAR(36)  NOT NULL,
    text       TEXT         NOT NULL,
    -- The specification rule this criterion became, once somebody has written
    -- it. Empty means the constraint is agreed but not yet expressed as
    -- behaviour anything can check.
    spec_ref   VARCHAR(512) NOT NULL DEFAULT '',
    -- Hand-marked while a criterion has no rule. Once spec_ref is set the
    -- specification decides, and this stops being consulted.
    met        BOOLEAN      NOT NULL DEFAULT FALSE,
    position   INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_crit_item FOREIGN KEY (item_id) REFERENCES plan_items(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_crit_item ON plan_criteria (item_id, position);

-- Discussion. Without it the conversation moves elsewhere and the item
-- becomes a stub that records a decision without its reasons.
CREATE TABLE IF NOT EXISTS plan_comments (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    item_id    VARCHAR(36)  NOT NULL,
    author_id  VARCHAR(36),
    body       TEXT         NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_comment_item FOREIGN KEY (item_id) REFERENCES plan_items(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_comment_item ON plan_comments (item_id, created_at);
