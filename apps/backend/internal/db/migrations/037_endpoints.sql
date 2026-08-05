-- ---------------------------------------------------------------------------
-- Endpoints
--
-- A visual, block-based way to define a REST API endpoint: request in, a graph
-- of nodes in the middle, response out. The graph is stored in the same shape
-- as a workflow (nodes + edges JSONB) and runs on the hardened workflow node
-- executor, synchronously in the API process. Own table, shared executor.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS endpoints (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id   VARCHAR(36)  NOT NULL,
    method       VARCHAR(10)  NOT NULL DEFAULT 'GET',
    path         VARCHAR(512) NOT NULL,
    name         VARCHAR(128) NOT NULL DEFAULT '',
    description  TEXT,
    -- who may CALL the endpoint: 'public' | 'session' | 'api_key' | 'either'.
    -- Distinct from what the nodes may TOUCH (the per-node apply-rules toggle).
    auth         VARCHAR(16)  NOT NULL DEFAULT 'public',
    input_schema JSONB,
    nodes        JSONB,
    edges        JSONB,
    status       VARCHAR(16)  NOT NULL DEFAULT 'draft',
    version      INT          NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ep_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- One published endpoint per (project, method, path). A draft may collide with a
-- published one only if they differ, so scope uniqueness to the routable pair.
CREATE UNIQUE INDEX IF NOT EXISTS idx_ep_project_method_path
    ON endpoints (project_id, method, path);
CREATE INDEX IF NOT EXISTS idx_ep_project ON endpoints (project_id);
CREATE INDEX IF NOT EXISTS idx_ep_routing ON endpoints (project_id, status);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON endpoints
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS endpoint_executions (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    endpoint_id VARCHAR(36)  NOT NULL,
    project_id  VARCHAR(36)  NOT NULL,
    status      VARCHAR(16)  NOT NULL DEFAULT 'ok',
    method      VARCHAR(10),
    path        VARCHAR(512),
    status_code INT          NOT NULL DEFAULT 0,
    request     JSONB,
    response    JSONB,
    logs        JSONB,
    error       TEXT,
    duration_ms BIGINT       NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_epe_endpoint FOREIGN KEY (endpoint_id) REFERENCES endpoints(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_epe_endpoint ON endpoint_executions (endpoint_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_epe_project  ON endpoint_executions (project_id);
