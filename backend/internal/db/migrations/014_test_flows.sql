-- Recorded flows.
--
-- A flow is what somebody did, stored as steps rather than as code: tap this,
-- type that, expect this to be visible. Keeping it as data is what lets one
-- studio serve every platform — the same step list compiles to a Playwright
-- spec for the web and a Maestro flow for a device, and a new platform is a
-- compiler rather than a second product.
CREATE TABLE IF NOT EXISTS test_flows (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    name        VARCHAR(256) NOT NULL,
    platform    VARCHAR(32)  NOT NULL DEFAULT 'web',   -- web | android | ios
    -- What was being exercised: a URL for the web, a build reference for a
    -- device. Recorded so a flow can be replayed against another branch.
    target      VARCHAR(512) NOT NULL,
    -- Ordered steps. Each carries a kind, a way of finding its element, and a
    -- value where the kind needs one.
    steps       JSONB        NOT NULL DEFAULT '[]',
    -- The suite created when this flow was saved, so a recording and the thing
    -- that runs it stay connected.
    suite_id    VARCHAR(36),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_tf_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tf_project ON test_flows (project_id, created_at DESC);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON test_flows
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();
