-- Migration 023: Plan — the work, before it is behaviour
--
-- A plan item is a decision to do something. What it must then *do* is a
-- specification, and whether it does it is a test; those are separate objects
-- with their own lifecycles, and an item points at them rather than
-- containing them. Keeping intent and behaviour apart is what lets a spec
-- outlive the item that prompted it — the item is done and gone, the
-- behaviour it asked for is permanent.

CREATE TABLE IF NOT EXISTS plan_items (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id   VARCHAR(36)  NOT NULL,
    parent_id    VARCHAR(36),
    title        VARCHAR(512) NOT NULL,
    body         TEXT         NOT NULL DEFAULT '',
    -- todo | in_progress | blocked | done | cancelled
    status       VARCHAR(32)  NOT NULL DEFAULT 'todo',
    -- low | medium | high | urgent
    priority     VARCHAR(16)  NOT NULL DEFAULT 'medium',
    assignee_id  VARCHAR(36),
    labels       JSONB        NOT NULL DEFAULT '[]'::jsonb,
    -- Where the item sits in a hand-ordered list. Sparse, so an item can be
    -- moved between two others without renumbering the rest.
    rank         BIGINT       NOT NULL DEFAULT 0,
    created_by   VARCHAR(36),
    closed_at    TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_plan_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT fk_plan_parent  FOREIGN KEY (parent_id)  REFERENCES plan_items(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_plan_project ON plan_items (project_id, status);
CREATE INDEX IF NOT EXISTS idx_plan_rank    ON plan_items (project_id, rank);
CREATE INDEX IF NOT EXISTS idx_plan_parent  ON plan_items (parent_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON plan_items
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

-- What an item points at: a specification, a test, a deploy, a repository.
--
-- A table rather than a column per kind, because the kinds are not known yet
-- — Spec is still being designed — and a link that arrives later should not
-- need a migration to be recordable.
CREATE TABLE IF NOT EXISTS plan_item_links (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    item_id    VARCHAR(36)  NOT NULL,
    -- spec | test | deploy | repo | url
    kind       VARCHAR(32)  NOT NULL,
    -- What it points at, in the terms of that kind: a .feature path, a test
    -- id, a release id, a URL.
    ref        VARCHAR(512) NOT NULL,
    label      VARCHAR(256) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_link_item FOREIGN KEY (item_id) REFERENCES plan_items(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_plan_link_item ON plan_item_links (item_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_plan_link_unique ON plan_item_links (item_id, kind, ref);
