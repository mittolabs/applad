-- A capture is the recording's technical context, kept alongside the flow.
--
-- The studio already stored what somebody did (test_flows.steps). A capture adds
-- what the browser did while they did it: the console, the network, the
-- environment, a video of the session, and — later — annotations and an AI
-- summary. One capture per saved flow, so a recording is one thing that happens
-- to carry its replay; the heavy, optional data lives here rather than bloating
-- the hot test_flows row.
--
-- The video and any large blobs are files under STORAGE_PATH/test-captures/<id>/;
-- this row holds the metadata and the timeline events (console, network) as
-- JSON, all sharing one server-clock timestamp so the replay lines them up.
CREATE TABLE IF NOT EXISTS test_captures (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    flow_id      VARCHAR(36),                          -- set when saved; a capture belongs to a flow
    project_id   VARCHAR(36)  NOT NULL,
    target       VARCHAR(512) NOT NULL DEFAULT '',
    duration_ms  BIGINT       NOT NULL DEFAULT 0,
    started_at   BIGINT       NOT NULL DEFAULT 0,       -- timeline origin (unix ms), so events are relative
    video_path   TEXT         NOT NULL DEFAULT '',      -- STORAGE_PATH/test-captures/<id>/video.mp4
    status       VARCHAR(16)  NOT NULL DEFAULT 'ready', -- encoding | ready | failed
    console      JSONB        NOT NULL DEFAULT '[]',
    network      JSONB        NOT NULL DEFAULT '[]',
    env          JSONB        NOT NULL DEFAULT '{}',
    steps        JSONB        NOT NULL DEFAULT '[]',    -- a copy at save time, timeline-aligned
    annotations  JSONB        NOT NULL DEFAULT '[]',
    ai_summary   TEXT         NOT NULL DEFAULT '',
    share_token  VARCHAR(64),                           -- null until shared
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_tc_flow FOREIGN KEY (flow_id) REFERENCES test_flows(id) ON DELETE CASCADE,
    CONSTRAINT fk_tc_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tc_flow ON test_captures (flow_id);
CREATE INDEX IF NOT EXISTS idx_tc_project ON test_captures (project_id, created_at DESC);
-- A share token is looked up directly by the read-only replay page.
CREATE UNIQUE INDEX IF NOT EXISTS idx_tc_share ON test_captures (share_token) WHERE share_token IS NOT NULL;

CREATE TRIGGER set_updated_at BEFORE UPDATE ON test_captures
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();
