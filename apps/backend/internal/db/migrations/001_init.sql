-- =============================================================================
-- Applad PostgreSQL Schema — consolidated migration
-- Replaces all MySQL migrations 001-020
-- Uses: real tables, PostgreSQL dialect, clean terminology (no collections/documents)
-- =============================================================================

-- ---------------------------------------------------------------------------
-- Reusable trigger function for updated_at
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION applad_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------------
-- CDC: fires pg_notify('applad_changes', ...) after INSERT/UPDATE/DELETE on
-- any user table. Reads project_id and database_id from session config set
-- by prepareDirectAccessTx (applad.project_id / applad.database_id).
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION applad_notify_change()
RETURNS TRIGGER AS $$
DECLARE
    project_id  TEXT := current_setting('applad.project_id',  true);
    database_id TEXT := current_setting('applad.database_id', true);
    payload     TEXT;
BEGIN
    IF project_id IS NULL OR project_id = '' THEN
        RETURN COALESCE(NEW, OLD);
    END IF;
    payload := json_build_object(
        'project_id',  project_id,
        'database_id', database_id,
        'schema',      TG_TABLE_SCHEMA,
        'table',       TG_TABLE_NAME,
        'action',      lower(TG_OP),
        'old',         CASE WHEN TG_OP = 'DELETE' THEN row_to_json(OLD) ELSE NULL END,
        'new',         CASE WHEN TG_OP <> 'DELETE' THEN row_to_json(NEW) ELSE NULL END,
        'timestamp',   to_char(NOW() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
    )::text;
    PERFORM pg_notify('applad_changes', payload);
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------------
-- Core: projects, api_keys
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS projects (
    id               VARCHAR(36)  NOT NULL PRIMARY KEY,
    org_id           VARCHAR(36),
    name             VARCHAR(128) NOT NULL,
    description      TEXT,
    auth_config      JSONB,
    services_config  JSONB,
    smtp_config      JSONB,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_proj_org ON projects (org_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS api_keys (
    id            VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id    VARCHAR(36)  NOT NULL,
    name          VARCHAR(128) NOT NULL,
    secret_hash   VARCHAR(256) NOT NULL,
    secret_prefix VARCHAR(16)  NOT NULL,
    scopes        JSONB,
    expires_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ak_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ak_project ON api_keys (project_id);

-- ---------------------------------------------------------------------------
-- Auth: users, sessions, auth_tokens
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id               VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id       VARCHAR(36)  NOT NULL,
    email            VARCHAR(256),
    phone            VARCHAR(32),
    name             VARCHAR(128),
    password_hash    VARCHAR(256),
    oauth_provider   VARCHAR(32),
    oauth_id         VARCHAR(256),
    mfa_secret       VARCHAR(64),
    mfa_enabled      BOOLEAN      NOT NULL DEFAULT FALSE,
    mfa_recovery     JSONB,
    email_verified   BOOLEAN      NOT NULL DEFAULT FALSE,
    phone_verified   BOOLEAN      NOT NULL DEFAULT FALSE,
    blocked          BOOLEAN      NOT NULL DEFAULT FALSE,
    status           SMALLINT     NOT NULL DEFAULT 1,
    labels           JSONB,
    prefs            JSONB,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_users_email_project UNIQUE (project_id, email)
);
CREATE INDEX IF NOT EXISTS idx_users_project ON users (project_id);
CREATE INDEX IF NOT EXISTS idx_users_oauth ON users (project_id, oauth_provider, oauth_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS sessions (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    user_id      VARCHAR(36)  NOT NULL,
    project_id   VARCHAR(36)  NOT NULL,
    ip           VARCHAR(64),
    user_agent   TEXT,
    expires_at   TIMESTAMPTZ  NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_sess_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_sess_user ON sessions (user_id);

CREATE TABLE IF NOT EXISTS auth_tokens (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    user_id    VARCHAR(36)  NOT NULL,
    project_id VARCHAR(36)  NOT NULL,
    type       VARCHAR(32)  NOT NULL,
    secret     VARCHAR(512) NOT NULL,
    expires_at TIMESTAMPTZ  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_at_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_at_user ON auth_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_at_secret ON auth_tokens (secret);

CREATE TABLE IF NOT EXISTS project_oauth_providers (
    id            VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id    VARCHAR(36)  NOT NULL,
    provider      VARCHAR(32)  NOT NULL,
    enabled       BOOLEAN      NOT NULL DEFAULT TRUE,
    client_id     VARCHAR(512) NOT NULL,
    client_secret VARCHAR(512) NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_project_provider UNIQUE (project_id, provider),
    CONSTRAINT fk_pop_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- ---------------------------------------------------------------------------
-- Teams & memberships
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS teams (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    name        VARCHAR(128) NOT NULL,
    prefs       JSONB,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_teams_project ON teams (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON teams
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS memberships (
    id            VARCHAR(36)  NOT NULL PRIMARY KEY,
    team_id       VARCHAR(36)  NOT NULL,
    user_id       VARCHAR(36),
    invited_email VARCHAR(256),
    roles         JSONB,
    invited       BOOLEAN      NOT NULL DEFAULT TRUE,
    joined        BOOLEAN      NOT NULL DEFAULT FALSE,
    secret        VARCHAR(512),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_mb_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE
);

-- ---------------------------------------------------------------------------
-- Organizations
-- ---------------------------------------------------------------------------
-- No plan/billing column: what an organization is entitled to is not a property
-- of the open core. A self-hosted install has no plans; the hosted product
-- derives an org's plan from its subscription in the commercial layer.
CREATE TABLE IF NOT EXISTS organizations (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    name         VARCHAR(128) NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS organization_members (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    org_id       VARCHAR(36)  NOT NULL,
    user_id      VARCHAR(36),
    email        VARCHAR(256) NOT NULL,
    name         VARCHAR(128),
    role         VARCHAR(32)  NOT NULL DEFAULT 'member',
    status       VARCHAR(32)  NOT NULL DEFAULT 'pending',
    invite_token VARCHAR(256),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_om_org FOREIGN KEY (org_id) REFERENCES organizations(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_om_org   ON organization_members (org_id);
CREATE INDEX IF NOT EXISTS idx_om_user  ON organization_members (user_id);
CREATE INDEX IF NOT EXISTS idx_om_email ON organization_members (email);

-- ---------------------------------------------------------------------------
-- Console users
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS console_users (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    email          VARCHAR(256) NOT NULL UNIQUE,
    name           VARCHAR(128),
    default_org_id VARCHAR(36),
    password_hash  VARCHAR(256) NOT NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON console_users
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

-- ---------------------------------------------------------------------------
-- Databases (metadata — maps to real PostgreSQL schemas per project+database)
-- Renamed from `_databases` → clean name `databases`
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS databases (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    name        VARCHAR(128) NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_db_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_databases_project ON databases (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON databases
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

-- ---------------------------------------------------------------------------
-- Tables, Columns, Indexes, Relationships (metadata for user-created real tables)
-- Renamed: collections→tables, attributes→columns, _indexes→indexes,
--          collection_relationships→table_relationships
-- The `documents` table is DELETED — user data lives in real PostgreSQL schemas
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS tables (
    id              VARCHAR(36)  NOT NULL PRIMARY KEY,
    database_id     VARCHAR(36)  NOT NULL,
    project_id      VARCHAR(36)  NOT NULL,
    name            VARCHAR(128) NOT NULL,
    row_security    BOOLEAN      NOT NULL DEFAULT FALSE,
    permissions     JSONB,
    enabled         BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_tbl_db FOREIGN KEY (database_id) REFERENCES databases(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_tables_db ON tables (database_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON tables
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS columns (
    id            VARCHAR(36)  NOT NULL PRIMARY KEY,
    table_id      VARCHAR(36)  NOT NULL,
    key_name      VARCHAR(128) NOT NULL,
    type          VARCHAR(32)  NOT NULL,
    required      BOOLEAN      NOT NULL DEFAULT FALSE,
    "array"       BOOLEAN      NOT NULL DEFAULT FALSE,
    default_value TEXT,
    options       JSONB,
    validation    JSONB        NOT NULL DEFAULT '{}',
    permissions   JSONB        NOT NULL DEFAULT '["read","write"]',
    status        VARCHAR(32)  NOT NULL DEFAULT 'available',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_col UNIQUE (table_id, key_name),
    CONSTRAINT fk_col_table FOREIGN KEY (table_id) REFERENCES tables(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS indexes (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    table_id   VARCHAR(36)  NOT NULL,
    key_name   VARCHAR(128) NOT NULL,
    type       VARCHAR(32)  NOT NULL,
    columns    JSONB        NOT NULL DEFAULT '[]',
    orders     JSONB        NOT NULL DEFAULT '[]',
    status     VARCHAR(32)  NOT NULL DEFAULT 'available',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_idx UNIQUE (table_id, key_name),
    CONSTRAINT fk_idx_table FOREIGN KEY (table_id) REFERENCES tables(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS table_relationships (
    id                 VARCHAR(36)  NOT NULL PRIMARY KEY,
    table_id           VARCHAR(36)  NOT NULL,
    related_table      VARCHAR(36)  NOT NULL,
    relationship_type  VARCHAR(32)  NOT NULL,
    two_way            BOOLEAN      NOT NULL DEFAULT FALSE,
    key_name           VARCHAR(128) NOT NULL,
    two_way_key        VARCHAR(128),
    on_delete          VARCHAR(32)  NOT NULL DEFAULT 'setNull',
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_rel_table   FOREIGN KEY (table_id)      REFERENCES tables(id) ON DELETE CASCADE,
    CONSTRAINT fk_rel_related FOREIGN KEY (related_table) REFERENCES tables(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_rel_table   ON table_relationships (table_id);
CREATE INDEX IF NOT EXISTS idx_rel_related ON table_relationships (related_table);

-- ---------------------------------------------------------------------------
-- Database permissions (RBAC)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS permissions (
    id            VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id    VARCHAR(36)  NOT NULL,
    resource_type VARCHAR(32)  NOT NULL,
    resource_id   VARCHAR(36)  NOT NULL,
    role          VARCHAR(128) NOT NULL,
    action        VARCHAR(32)  NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_perm UNIQUE (project_id, resource_type, resource_id, role, action),
    CONSTRAINT fk_perm_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_perm_project  ON permissions (project_id);
CREATE INDEX IF NOT EXISTS idx_perm_resource ON permissions (resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_perm_role     ON permissions (role);

-- ---------------------------------------------------------------------------
-- Storage: buckets, files
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS buckets (
    id                    VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id            VARCHAR(36)  NOT NULL,
    name                  VARCHAR(128) NOT NULL,
    permissions           JSONB,
    file_size_limit       BIGINT       NOT NULL DEFAULT 0,
    allowed_mime_types    JSONB,
    encryption            BOOLEAN      NOT NULL DEFAULT FALSE,
    antivirus             BOOLEAN      NOT NULL DEFAULT FALSE,
    file_security         BOOLEAN      NOT NULL DEFAULT FALSE,
    compression           VARCHAR(32)  NOT NULL DEFAULT 'none',
    image_transformations BOOLEAN      NOT NULL DEFAULT TRUE,
    enabled               BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_bucket_project ON buckets (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON buckets
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS files (
    id          VARCHAR(36)   NOT NULL PRIMARY KEY,
    bucket_id   VARCHAR(36)   NOT NULL,
    project_id  VARCHAR(36)   NOT NULL,
    name        VARCHAR(512)  NOT NULL,
    mime_type   VARCHAR(128)  NOT NULL DEFAULT 'application/octet-stream',
    size        BIGINT        NOT NULL,
    permissions JSONB,
    path        VARCHAR(1024) NOT NULL,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_file_bucket FOREIGN KEY (bucket_id) REFERENCES buckets(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_file_bucket ON files (bucket_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON files
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

-- ---------------------------------------------------------------------------
-- Deployments (v1 simple model)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS deployments (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    name        VARCHAR(255) NOT NULL,
    type        VARCHAR(50)  NOT NULL DEFAULT 'web',
    status      VARCHAR(50)  NOT NULL DEFAULT 'pending',
    config      JSONB,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_dep_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_deployments_project ON deployments (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON deployments
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

-- ---------------------------------------------------------------------------
-- Deploy targets, pipelines, releases, executions
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS deploy_targets (
    id                VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id        VARCHAR(36)  NOT NULL,
    environment_id    VARCHAR(36),
    git_connection_id VARCHAR(36),
    git_repo_url      VARCHAR(512),
    git_branch        VARCHAR(128),
    name              VARCHAR(128) NOT NULL,
    type              VARCHAR(32)  NOT NULL DEFAULT 'serverless',
    runtime           VARCHAR(64),
    entrypoint        VARCHAR(256),
    timeout_ms        INT          NOT NULL DEFAULT 30000,
    memory_mb         INT          NOT NULL DEFAULT 256,
    env_vars          JSONB,
    permissions       JSONB,
    cron              VARCHAR(128),
    domain            VARCHAR(256),
    ssl_enabled       BOOLEAN      NOT NULL DEFAULT FALSE,
    output_dir        VARCHAR(256),
    dockerfile        VARCHAR(256),
    registry_url      VARCHAR(512),
    tag_strategy      VARCHAR(32)  NOT NULL DEFAULT 'latest',
    build_type        VARCHAR(32),
    signing_config    JSONB,
    store_config      JSONB,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_dt_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_dt_project ON deploy_targets (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON deploy_targets
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS deploy_pipelines (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    target_id   VARCHAR(36)  NOT NULL,
    name        VARCHAR(128) NOT NULL,
    source_type VARCHAR(32)  NOT NULL DEFAULT 'upload',
    source_url  VARCHAR(512),
    branch      VARCHAR(128),
    build_cmd   VARCHAR(512),
    output_dir  VARCHAR(256),
    env_vars    JSONB,
    trigger_on  JSONB,
    cache_dirs  JSONB,
    agent_label VARCHAR(64),
    timeout_ms  INT          NOT NULL DEFAULT 600000,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_dp_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT fk_dp_target  FOREIGN KEY (target_id)  REFERENCES deploy_targets(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_dp_project ON deploy_pipelines (project_id);
CREATE INDEX IF NOT EXISTS idx_dp_target  ON deploy_pipelines (target_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON deploy_pipelines
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS deploy_releases (
    id            VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id    VARCHAR(36)  NOT NULL,
    pipeline_id   VARCHAR(36)  NOT NULL,
    target_id     VARCHAR(36)  NOT NULL,
    status        VARCHAR(32)  NOT NULL DEFAULT 'pending',
    trigger_type  VARCHAR(32)  NOT NULL DEFAULT 'manual',
    trigger_actor VARCHAR(128),
    commit_sha    VARCHAR(64),
    build_log     TEXT,
    deploy_log    TEXT,
    artifact_path VARCHAR(512),
    duration_ms   BIGINT       NOT NULL DEFAULT 0,
    error         TEXT,
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_dr_project  FOREIGN KEY (project_id)  REFERENCES projects(id)          ON DELETE CASCADE,
    CONSTRAINT fk_dr_pipeline FOREIGN KEY (pipeline_id) REFERENCES deploy_pipelines(id)  ON DELETE CASCADE,
    CONSTRAINT fk_dr_target   FOREIGN KEY (target_id)   REFERENCES deploy_targets(id)    ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_dr_project  ON deploy_releases (project_id);
CREATE INDEX IF NOT EXISTS idx_dr_pipeline ON deploy_releases (pipeline_id);

CREATE TABLE IF NOT EXISTS deploy_executions (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id     VARCHAR(36)  NOT NULL,
    target_id      VARCHAR(36)  NOT NULL,
    status         VARCHAR(32)  NOT NULL DEFAULT 'pending',
    status_code    INT          NOT NULL DEFAULT 0,
    request        JSONB,
    response       TEXT,
    stdout         TEXT,
    stderr         TEXT,
    duration_ms    BIGINT       NOT NULL DEFAULT 0,
    trigger_source VARCHAR(32)  NOT NULL DEFAULT 'http',
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_de_project FOREIGN KEY (project_id) REFERENCES projects(id)       ON DELETE CASCADE,
    CONSTRAINT fk_de_target  FOREIGN KEY (target_id)  REFERENCES deploy_targets(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_de_project ON deploy_executions (project_id);
CREATE INDEX IF NOT EXISTS idx_de_target  ON deploy_executions (target_id);

CREATE TABLE IF NOT EXISTS deploy_templates (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    name        VARCHAR(128) NOT NULL,
    description TEXT,
    category    VARCHAR(32)  NOT NULL,
    framework   VARCHAR(64),
    use_case    VARCHAR(64),
    repo_url    VARCHAR(512),
    branch      VARCHAR(128) NOT NULL DEFAULT 'main',
    build_cmd   VARCHAR(512),
    output_dir  VARCHAR(256),
    install_cmd VARCHAR(512),
    env_vars    JSONB,
    icon        VARCHAR(64),
    popularity  INT          NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS git_connections (
    id              VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id      VARCHAR(36)  NOT NULL,
    provider        VARCHAR(32)  NOT NULL DEFAULT 'github',
    installation_id VARCHAR(128),
    access_token    TEXT,
    refresh_token   TEXT,
    account_name    VARCHAR(128),
    account_type    VARCHAR(32),
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_gc_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_gc_project ON git_connections (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON git_connections
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS environments (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    name        VARCHAR(64)  NOT NULL,
    slug        VARCHAR(64)  NOT NULL,
    branch      VARCHAR(128),
    domain      VARCHAR(256),
    env_vars    JSONB,
    is_default  BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_env UNIQUE (project_id, slug),
    CONSTRAINT fk_env_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_env_project ON environments (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON environments
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS custom_domains (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id     VARCHAR(36)  NOT NULL,
    target_id      VARCHAR(36)  NOT NULL,
    domain         VARCHAR(256) NOT NULL UNIQUE,
    verification   VARCHAR(256),
    verified       BOOLEAN      NOT NULL DEFAULT FALSE,
    ssl_status     VARCHAR(32)  NOT NULL DEFAULT 'pending',
    ssl_expires_at TIMESTAMPTZ,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_cd_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cd_project ON custom_domains (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON custom_domains
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS build_agents (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id     VARCHAR(36)  NOT NULL,
    name           VARCHAR(128) NOT NULL,
    token          VARCHAR(256) NOT NULL UNIQUE,
    labels         JSONB,
    status         VARCHAR(32)  NOT NULL DEFAULT 'offline',
    last_heartbeat TIMESTAMPTZ,
    current_job_id VARCHAR(36),
    os             VARCHAR(32),
    arch           VARCHAR(32),
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ba_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ba_project ON build_agents (project_id);

CREATE TABLE IF NOT EXISTS registry_images (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    target_id   VARCHAR(36)  NOT NULL,
    repository  VARCHAR(256) NOT NULL,
    tag         VARCHAR(128) NOT NULL,
    digest      VARCHAR(128),
    size_bytes  BIGINT       NOT NULL DEFAULT 0,
    platform    VARCHAR(32),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ri_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ri_project ON registry_images (project_id);

-- ---------------------------------------------------------------------------
-- Workflows
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS workflows (
    id                VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id        VARCHAR(36)  NOT NULL,
    folder_id         VARCHAR(36),
    name              VARCHAR(128) NOT NULL,
    description       TEXT,
    status            VARCHAR(32)  NOT NULL DEFAULT 'draft',
    trigger_type      VARCHAR(32)  NOT NULL DEFAULT 'manual',
    trigger_config    JSONB,
    webhook_secret    VARCHAR(64),
    nodes             JSONB,
    edges             JSONB,
    tags              JSONB,
    error_workflow_id VARCHAR(36),
    retry_attempts    INT          NOT NULL DEFAULT 0,
    retry_delay_ms    INT          NOT NULL DEFAULT 1000,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_wf_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_wf_project ON workflows (project_id);
CREATE INDEX IF NOT EXISTS idx_wf_folder  ON workflows (folder_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON workflows
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS workflow_executions (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    workflow_id  VARCHAR(36)  NOT NULL,
    project_id   VARCHAR(36)  NOT NULL,
    status       VARCHAR(32)  NOT NULL DEFAULT 'pending',
    trigger_data JSONB,
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms  BIGINT       NOT NULL DEFAULT 0,
    error        TEXT,
    logs         JSONB,
    CONSTRAINT fk_wfe_workflow FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_wfe_workflow ON workflow_executions (workflow_id);
CREATE INDEX IF NOT EXISTS idx_wfe_project  ON workflow_executions (project_id);

CREATE TABLE IF NOT EXISTS workflow_folders (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    name        VARCHAR(128) NOT NULL,
    parent_id   VARCHAR(36),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_wff_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_wff_project ON workflow_folders (project_id);

CREATE TABLE IF NOT EXISTS workflow_versions (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    workflow_id    VARCHAR(36)  NOT NULL,
    version        INT          NOT NULL DEFAULT 1,
    name           VARCHAR(128) NOT NULL,
    description    TEXT,
    nodes          JSONB,
    edges          JSONB,
    trigger_type   VARCHAR(32),
    trigger_config JSONB,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by     VARCHAR(36),
    CONSTRAINT fk_wfv_workflow FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_wfv_workflow ON workflow_versions (workflow_id);

CREATE TABLE IF NOT EXISTS workflow_shares (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    workflow_id VARCHAR(36)  NOT NULL,
    user_id     VARCHAR(36)  NOT NULL,
    role        VARCHAR(32)  NOT NULL DEFAULT 'viewer',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_wfs UNIQUE (workflow_id, user_id),
    CONSTRAINT fk_wfs_workflow FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_wfs_workflow ON workflow_shares (workflow_id);

CREATE TABLE IF NOT EXISTS workflow_templates (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    name           VARCHAR(128) NOT NULL,
    description    TEXT,
    category       VARCHAR(64)  NOT NULL DEFAULT 'general',
    icon           VARCHAR(64),
    nodes          JSONB,
    edges          JSONB,
    trigger_type   VARCHAR(32)  NOT NULL DEFAULT 'manual',
    trigger_config JSONB,
    popularity     INT          NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS credentials (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    name        VARCHAR(128) NOT NULL,
    type        VARCHAR(64)  NOT NULL,
    data        TEXT         NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_cred_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cred_project ON credentials (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON credentials
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

-- ---------------------------------------------------------------------------
-- Functions
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS functions (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    name        VARCHAR(255) NOT NULL,
    runtime     VARCHAR(50)  NOT NULL,
    entrypoint  VARCHAR(255) NOT NULL DEFAULT 'main',
    timeout     INT          NOT NULL DEFAULT 15,
    env_vars    JSONB,
    source_type VARCHAR(20)  NOT NULL DEFAULT 'inline',
    source      TEXT,
    repository  TEXT,
    branch      VARCHAR(255),
    cron        VARCHAR(128),
    status      VARCHAR(50)  NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_functions_project ON functions (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON functions
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS function_executions (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    function_id VARCHAR(36)  NOT NULL,
    project_id  VARCHAR(36)  NOT NULL,
    status      VARCHAR(50)  NOT NULL DEFAULT 'pending',
    output      TEXT,
    errors      TEXT,
    duration    DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_executions_function ON function_executions (function_id);
CREATE INDEX IF NOT EXISTS idx_executions_project  ON function_executions (project_id);

-- ---------------------------------------------------------------------------
-- Webhooks
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS webhooks (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    name        VARCHAR(128) NOT NULL,
    url         VARCHAR(512) NOT NULL,
    events      JSONB        NOT NULL,
    secret      VARCHAR(256),
    enabled     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_wh_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_wh_project ON webhooks (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON webhooks
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    webhook_id  VARCHAR(36)  NOT NULL,
    event       VARCHAR(64)  NOT NULL,
    payload     JSONB,
    status_code INT,
    response    TEXT,
    attempts    INT          NOT NULL DEFAULT 0,
    success     BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_whd_webhook FOREIGN KEY (webhook_id) REFERENCES webhooks(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_whd_webhook  ON webhook_deliveries (webhook_id);
CREATE INDEX IF NOT EXISTS idx_whd_created  ON webhook_deliveries (created_at);

-- ---------------------------------------------------------------------------
-- Platforms
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS platforms (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    type        VARCHAR(32)  NOT NULL,
    name        VARCHAR(128) NOT NULL,
    hostname    VARCHAR(256),
    store_id    VARCHAR(256),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_plat_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_plat_project ON platforms (project_id);

-- ---------------------------------------------------------------------------
-- Usage metrics
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS usage_metrics (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    metric      VARCHAR(64)  NOT NULL,
    value       BIGINT       NOT NULL DEFAULT 0,
    period      VARCHAR(16)  NOT NULL DEFAULT 'hour',
    timestamp   TIMESTAMPTZ  NOT NULL,
    CONSTRAINT fk_um_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_um_project_metric ON usage_metrics (project_id, metric, timestamp);

CREATE TABLE IF NOT EXISTS user_logs (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    user_id     VARCHAR(36)  NOT NULL,
    event       VARCHAR(64)  NOT NULL,
    ip          VARCHAR(45),
    user_agent  VARCHAR(512),
    details     JSONB,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ul_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ul_user  ON user_logs (project_id, user_id);
CREATE INDEX IF NOT EXISTS idx_ul_event ON user_logs (project_id, event);

-- ---------------------------------------------------------------------------
-- Feature flags
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS feature_flags (
    id            VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id    VARCHAR(36)  NOT NULL,
    key_name      VARCHAR(128) NOT NULL,
    name          VARCHAR(256) NOT NULL,
    description   TEXT,
    type          VARCHAR(16)  NOT NULL DEFAULT 'boolean',
    default_value JSONB        NOT NULL,
    enabled       BOOLEAN      NOT NULL DEFAULT FALSE,
    tags          JSONB,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_ff_key UNIQUE (project_id, key_name),
    CONSTRAINT fk_ff_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ff_project ON feature_flags (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON feature_flags
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS flag_rules (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    flag_id     VARCHAR(36)  NOT NULL,
    priority    INT          NOT NULL DEFAULT 0,
    type        VARCHAR(32)  NOT NULL,
    conditions  JSONB        NOT NULL,
    value       JSONB        NOT NULL,
    rollout_pct INT          NOT NULL DEFAULT 100,
    enabled     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_fr_flag FOREIGN KEY (flag_id) REFERENCES feature_flags(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_fr_flag ON flag_rules (flag_id);

CREATE TABLE IF NOT EXISTS flag_overrides (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    flag_id     VARCHAR(36)  NOT NULL,
    target_type VARCHAR(32)  NOT NULL,
    target_id   VARCHAR(128) NOT NULL,
    value       JSONB        NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_fo UNIQUE (flag_id, target_type, target_id),
    CONSTRAINT fk_fo_flag FOREIGN KEY (flag_id) REFERENCES feature_flags(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_fo_flag ON flag_overrides (flag_id);

CREATE TABLE IF NOT EXISTS flag_evaluations (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    flag_id    VARCHAR(36)  NOT NULL,
    project_id VARCHAR(36)  NOT NULL,
    user_id    VARCHAR(128),
    value      JSONB        NOT NULL,
    rule_id    VARCHAR(36),
    timestamp  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_fev_flag FOREIGN KEY (flag_id) REFERENCES feature_flags(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_fev_flag     ON flag_evaluations (flag_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_fev_project  ON flag_evaluations (project_id, timestamp);

-- ---------------------------------------------------------------------------
-- Migrations (external platform imports)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS migrations (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id   VARCHAR(36)  NOT NULL,
    source       VARCHAR(50)  NOT NULL,
    status       VARCHAR(32)  NOT NULL DEFAULT 'pending',
    resources    JSONB,
    errors       JSONB,
    progress     INT          NOT NULL DEFAULT 0,
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_mig_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_mig_project ON migrations (project_id);

-- ---------------------------------------------------------------------------
-- Audit logs
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS audit_logs (
    id            VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id    VARCHAR(36)  NOT NULL,
    user_id       VARCHAR(36),
    action        VARCHAR(128) NOT NULL,
    resource_type VARCHAR(64)  NOT NULL,
    resource_id   VARCHAR(36),
    method        VARCHAR(16)  NOT NULL DEFAULT '',
    path          VARCHAR(512) NOT NULL DEFAULT '',
    status_code   SMALLINT     NOT NULL DEFAULT 0,
    ip_address    VARCHAR(64),
    user_agent    VARCHAR(512),
    metadata      JSONB,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_al_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_al_project  ON audit_logs (project_id);
CREATE INDEX IF NOT EXISTS idx_al_user     ON audit_logs (user_id);
CREATE INDEX IF NOT EXISTS idx_al_action   ON audit_logs (action);
CREATE INDEX IF NOT EXISTS idx_al_resource ON audit_logs (resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_al_created  ON audit_logs (created_at);

-- ---------------------------------------------------------------------------
-- Analytics
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS analytics_events (
    id          VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)   NOT NULL,
    user_id     VARCHAR(36),
    session_id  VARCHAR(36),
    event       VARCHAR(128)  NOT NULL,
    properties  JSONB,
    url         VARCHAR(2048),
    referrer    VARCHAR(2048),
    device_type VARCHAR(32),
    browser     VARCHAR(64),
    country     VARCHAR(8),
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ae_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ae_project ON analytics_events (project_id);
CREATE INDEX IF NOT EXISTS idx_ae_user    ON analytics_events (user_id);
CREATE INDEX IF NOT EXISTS idx_ae_session ON analytics_events (session_id);
CREATE INDEX IF NOT EXISTS idx_ae_event   ON analytics_events (project_id, event);
CREATE INDEX IF NOT EXISTS idx_ae_created ON analytics_events (project_id, created_at);

CREATE TABLE IF NOT EXISTS analytics_sessions (
    id          VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)   NOT NULL,
    user_id     VARCHAR(36),
    device_type VARCHAR(32),
    browser     VARCHAR(64),
    country     VARCHAR(8),
    entry_url   VARCHAR(2048),
    exit_url    VARCHAR(2048),
    page_views  INT           NOT NULL DEFAULT 0,
    duration_s  INT           NOT NULL DEFAULT 0,
    started_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    ended_at    TIMESTAMPTZ,
    CONSTRAINT fk_as_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_as_project ON analytics_sessions (project_id);
CREATE INDEX IF NOT EXISTS idx_as_user    ON analytics_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_as_started ON analytics_sessions (project_id, started_at);

CREATE TABLE IF NOT EXISTS analytics_funnels (
    id          VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)   NOT NULL,
    name        VARCHAR(128)  NOT NULL,
    steps       JSONB         NOT NULL,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_af_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_af_project ON analytics_funnels (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON analytics_funnels
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

-- ---------------------------------------------------------------------------
-- Search
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS search_indexes (
    id             VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id     VARCHAR(36)   NOT NULL,
    name           VARCHAR(128)  NOT NULL,
    fields         JSONB         NOT NULL,
    synonyms       JSONB,
    ranking_rules  JSONB,
    typo_tolerance BOOLEAN       NOT NULL DEFAULT TRUE,
    status         VARCHAR(32)   NOT NULL DEFAULT 'ready',
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_si_project_name UNIQUE (project_id, name),
    CONSTRAINT fk_si_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_si_project ON search_indexes (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON search_indexes
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS search_synonyms (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    index_id   VARCHAR(36)  NOT NULL,
    synonyms   JSONB        NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ss_index FOREIGN KEY (index_id) REFERENCES search_indexes(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ss_index ON search_synonyms (index_id);

CREATE TABLE IF NOT EXISTS search_documents (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    index_id   VARCHAR(36)  NOT NULL,
    project_id VARCHAR(36)  NOT NULL,
    doc_id     VARCHAR(36)  NOT NULL,
    content    TEXT         NOT NULL,
    metadata   JSONB,
    indexed_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_sd_index_doc UNIQUE (index_id, doc_id),
    CONSTRAINT fk_sd_index FOREIGN KEY (index_id) REFERENCES search_indexes(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_sd_index       ON search_documents (index_id);
-- Full-text search index (PostgreSQL GIN/tsvector, replaces MySQL FULLTEXT)
CREATE INDEX IF NOT EXISTS idx_sd_content_fts ON search_documents USING GIN (to_tsvector('english', content));

-- ---------------------------------------------------------------------------
-- Jobs / Queues
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS job_queues (
    id                   VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id           VARCHAR(36)  NOT NULL,
    name                 VARCHAR(128) NOT NULL,
    worker_url           VARCHAR(512),
    concurrency          INT          NOT NULL DEFAULT 10,
    retry_limit          INT          NOT NULL DEFAULT 3,
    retry_delay_s        INT          NOT NULL DEFAULT 60,
    dead_letter_queue_id VARCHAR(36),
    paused               BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_jq_project_name UNIQUE (project_id, name),
    CONSTRAINT fk_jq_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_jq_project ON job_queues (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON job_queues
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS jobs (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    queue_id     VARCHAR(36)  NOT NULL,
    project_id   VARCHAR(36)  NOT NULL,
    name         VARCHAR(128) NOT NULL,
    payload      JSONB,
    status       VARCHAR(32)  NOT NULL DEFAULT 'pending',
    priority     SMALLINT     NOT NULL DEFAULT 0,
    run_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    attempts     INT          NOT NULL DEFAULT 0,
    max_attempts INT          NOT NULL DEFAULT 3,
    last_error   TEXT,
    depends_on   JSONB,
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_job_queue   FOREIGN KEY (queue_id)   REFERENCES job_queues(id) ON DELETE CASCADE,
    CONSTRAINT fk_job_project FOREIGN KEY (project_id) REFERENCES projects(id)   ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_jobs_queue   ON jobs (queue_id, status, run_at);
CREATE INDEX IF NOT EXISTS idx_jobs_project ON jobs (project_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status  ON jobs (status, run_at);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON jobs
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

-- ---------------------------------------------------------------------------
-- Vectors
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS vector_indexes (
    id              VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id      VARCHAR(36)  NOT NULL,
    name            VARCHAR(128) NOT NULL,
    dimensions      INT          NOT NULL DEFAULT 1536,
    metric          VARCHAR(32)  NOT NULL DEFAULT 'cosine',
    embedding_field VARCHAR(128),
    model           VARCHAR(128) NOT NULL DEFAULT 'text-embedding-3-small',
    status          VARCHAR(32)  NOT NULL DEFAULT 'ready',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_vi_project_name UNIQUE (project_id, name),
    CONSTRAINT fk_vi_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_vi_project ON vector_indexes (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON vector_indexes
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS vector_embeddings (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    index_id   VARCHAR(36)  NOT NULL,
    project_id VARCHAR(36)  NOT NULL,
    doc_id     VARCHAR(36)  NOT NULL,
    vector     TEXT         NOT NULL,
    metadata   JSONB,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_ve_index_doc UNIQUE (index_id, doc_id),
    CONSTRAINT fk_ve_index   FOREIGN KEY (index_id)   REFERENCES vector_indexes(id) ON DELETE CASCADE,
    CONSTRAINT fk_ve_project FOREIGN KEY (project_id) REFERENCES projects(id)       ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ve_index   ON vector_embeddings (index_id);
CREATE INDEX IF NOT EXISTS idx_ve_project ON vector_embeddings (project_id);

-- ---------------------------------------------------------------------------
-- Content / CMS
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS content_types (
    id            VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id    VARCHAR(36)  NOT NULL,
    name          VARCHAR(128) NOT NULL,
    slug          VARCHAR(128) NOT NULL,
    fields        JSONB        NOT NULL,
    versioning    BOOLEAN      NOT NULL DEFAULT TRUE,
    localization  BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_ct_project_slug UNIQUE (project_id, slug),
    CONSTRAINT fk_ct_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ct_project ON content_types (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON content_types
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS content_entries (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    type_id      VARCHAR(36)  NOT NULL,
    project_id   VARCHAR(36)  NOT NULL,
    slug         VARCHAR(256),
    status       VARCHAR(32)  NOT NULL DEFAULT 'draft',
    locale       VARCHAR(16)  NOT NULL DEFAULT 'en',
    author_id    VARCHAR(36),
    published_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ce_type    FOREIGN KEY (type_id)    REFERENCES content_types(id) ON DELETE CASCADE,
    CONSTRAINT fk_ce_project FOREIGN KEY (project_id) REFERENCES projects(id)      ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ce_type    ON content_entries (type_id);
CREATE INDEX IF NOT EXISTS idx_ce_project ON content_entries (project_id);
CREATE INDEX IF NOT EXISTS idx_ce_slug    ON content_entries (project_id, slug);
CREATE INDEX IF NOT EXISTS idx_ce_status  ON content_entries (project_id, status);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON content_entries
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS content_versions (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    entry_id   VARCHAR(36)  NOT NULL,
    version    INT          NOT NULL DEFAULT 1,
    data       JSONB        NOT NULL,
    created_by VARCHAR(36),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_cv_entry_version UNIQUE (entry_id, version),
    CONSTRAINT fk_cv_entry FOREIGN KEY (entry_id) REFERENCES content_entries(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cv_entry ON content_versions (entry_id);

-- ---------------------------------------------------------------------------
-- Regions / Data residency
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS regions (
    id         VARCHAR(36)    NOT NULL PRIMARY KEY,
    name       VARCHAR(64)    NOT NULL,
    code       VARCHAR(16)    NOT NULL UNIQUE,
    location   VARCHAR(128)   NOT NULL,
    endpoint   VARCHAR(256)   NOT NULL DEFAULT '',
    latitude   NUMERIC(9,6),
    longitude  NUMERIC(9,6),
    status     VARCHAR(32)    NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS project_regions (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id     VARCHAR(36)  NOT NULL,
    region_id      VARCHAR(36)  NOT NULL,
    primary_region BOOLEAN      NOT NULL DEFAULT FALSE,
    gdpr           BOOLEAN      NOT NULL DEFAULT FALSE,
    hipaa          BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_pr_project_region UNIQUE (project_id, region_id),
    CONSTRAINT fk_pr_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    CONSTRAINT fk_pr_region  FOREIGN KEY (region_id)  REFERENCES regions(id)  ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_pr_project ON project_regions (project_id);

-- ---------------------------------------------------------------------------
-- Edge functions
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS edge_functions (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    name        VARCHAR(128) NOT NULL,
    slug        VARCHAR(128) NOT NULL,
    code        TEXT         NOT NULL DEFAULT '',
    runtime     VARCHAR(32)  NOT NULL DEFAULT 'js',
    regions     JSONB,
    env_vars    JSONB,
    status      VARCHAR(32)  NOT NULL DEFAULT 'draft',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_ef_project_slug UNIQUE (project_id, slug),
    CONSTRAINT fk_ef_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ef_project ON edge_functions (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON edge_functions
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS edge_deployments (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    function_id VARCHAR(36)  NOT NULL,
    project_id  VARCHAR(36)  NOT NULL,
    version     INT          NOT NULL DEFAULT 1,
    status      VARCHAR(32)  NOT NULL DEFAULT 'deploying',
    regions     JSONB,
    deployed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ed_function FOREIGN KEY (function_id) REFERENCES edge_functions(id) ON DELETE CASCADE,
    CONSTRAINT fk_ed_project  FOREIGN KEY (project_id)  REFERENCES projects(id)        ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ed_function ON edge_deployments (function_id);
CREATE INDEX IF NOT EXISTS idx_ed_project  ON edge_deployments (project_id);

-- ---------------------------------------------------------------------------
-- Messaging
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS messages (
    id           VARCHAR(32)  NOT NULL PRIMARY KEY,
    project_id   VARCHAR(32)  NOT NULL,
    type         VARCHAR(10)  NOT NULL,
    subject      VARCHAR(512) NOT NULL DEFAULT '',
    body         TEXT         NOT NULL DEFAULT '',
    recipients   TEXT         NOT NULL DEFAULT '',
    status       VARCHAR(20)  NOT NULL DEFAULT 'processing',
    scheduled_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_messages_project ON messages (project_id);
CREATE INDEX IF NOT EXISTS idx_messages_status  ON messages (status);
CREATE INDEX IF NOT EXISTS idx_messages_type    ON messages (type);

CREATE TABLE IF NOT EXISTS msg_topics (
    id          VARCHAR(32)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(32)  NOT NULL,
    name        VARCHAR(256) NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_topic_project_name UNIQUE (project_id, name)
);
CREATE INDEX IF NOT EXISTS idx_topics_project ON msg_topics (project_id);

CREATE TABLE IF NOT EXISTS msg_topic_subscribers (
    id         BIGSERIAL    NOT NULL PRIMARY KEY,
    topic_id   VARCHAR(32)  NOT NULL,
    target     VARCHAR(512) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_subscriber UNIQUE (topic_id, target)
);
CREATE INDEX IF NOT EXISTS idx_subscriber_topic ON msg_topic_subscribers (topic_id);

CREATE TABLE IF NOT EXISTS message_templates (
    id         VARCHAR(32)  NOT NULL PRIMARY KEY,
    project_id VARCHAR(32)  NOT NULL,
    name       VARCHAR(256) NOT NULL,
    type       VARCHAR(10)  NOT NULL DEFAULT 'email', -- email, sms, push
    subject    VARCHAR(512) NOT NULL DEFAULT '',
    body       TEXT         NOT NULL DEFAULT '',
    variables  TEXT         NOT NULL DEFAULT '[]',   -- JSON array of variable names
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_message_templates_project ON message_templates (project_id);

CREATE TABLE IF NOT EXISTS msg_providers (
    id          VARCHAR(32)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(32)  NOT NULL,
    name        VARCHAR(256) NOT NULL,
    type        VARCHAR(10)  NOT NULL,  -- email, sms, push
    provider    VARCHAR(32)  NOT NULL,  -- smtp, mailgun, sendgrid, resend, twilio, vonage, msg91, fcm, apns
    config      JSONB        NOT NULL DEFAULT '{}',
    enabled     BOOLEAN      NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_msg_providers_project ON msg_providers (project_id);

-- ---------------------------------------------------------------------------
-- Realtime change notification (used by Phase 6: LISTEN/NOTIFY)
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION applad_notify_change()
RETURNS TRIGGER AS $$
DECLARE
    payload JSONB;
BEGIN
    payload = jsonb_build_object(
        'project_id', COALESCE(NULLIF(current_setting('applad.project_id', true), ''), NULL),
        'user_id',   COALESCE(NULLIF(current_setting('applad.user_id', true), ''), NULL),
        'table',     TG_TABLE_NAME,
        'schema',    TG_TABLE_SCHEMA,
        'action',    TG_OP,
        'old',       CASE WHEN TG_OP IN ('UPDATE', 'DELETE') THEN row_to_json(OLD)::jsonb ELSE NULL END,
        'new',       CASE WHEN TG_OP IN ('INSERT', 'UPDATE') THEN row_to_json(NEW)::jsonb ELSE NULL END,
        'timestamp', NOW()
    );
    PERFORM pg_notify('applad_changes', payload::text);
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------------
-- Seed: regions
-- ---------------------------------------------------------------------------
INSERT INTO regions (id, name, code, location, endpoint, latitude, longitude) VALUES
    ('region-us-east-1',  'US East (N. Virginia)',  'us-east-1',    'Ashburn, VA, USA',    '',  38.9072,  -77.0369),
    ('region-us-west-2',  'US West (Oregon)',        'us-west-2',    'The Dalles, OR, USA', '',  45.5946, -121.1787),
    ('region-eu-west-1',  'EU West (Ireland)',       'eu-west-1',    'Dublin, Ireland',     '',  53.3498,   -6.2603),
    ('region-eu-central', 'EU Central (Frankfurt)', 'eu-central-1', 'Frankfurt, Germany',  '',  50.1109,    8.6821),
    ('region-ap-south-1', 'AP South (Mumbai)',       'ap-south-1',   'Mumbai, India',       '',  19.0760,   72.8777),
    ('region-ap-east-1',  'AP East (Singapore)',     'ap-east-1',    'Singapore',           '',   1.3521,  103.8198),
    ('region-sa-east-1',  'SA East (São Paulo)',     'sa-east-1',    'São Paulo, Brazil',   '', -23.5505,  -46.6333)
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Seed: workflow templates
-- ---------------------------------------------------------------------------
INSERT INTO workflow_templates (id, name, description, category, icon, trigger_type, nodes, edges) VALUES
('tpl_webhook_slack', 'Webhook to Slack', 'Forward incoming webhooks to a Slack channel', 'notifications', 'webhook', 'webhook',
 '[{"id":"t0","type":"trigger","label":"Webhook","config":{},"position":{"x":200,"y":250}},{"id":"n1","type":"slack","label":"Send to Slack","config":{"webhookUrl":"","message":"{{.trigger.body}}"},"position":{"x":500,"y":250}}]',
 '[{"id":"e0","source":"t0","target":"n1"}]'),
('tpl_cron_email', 'Scheduled Email Report', 'Send a recurring email report on a schedule', 'notifications', 'clock', 'cron',
 '[{"id":"t0","type":"trigger","label":"Schedule","config":{},"position":{"x":200,"y":250}},{"id":"n1","type":"http_request","label":"Fetch Data","config":{"url":"","method":"GET"},"position":{"x":480,"y":250}},{"id":"n2","type":"send_email","label":"Send Report","config":{"to":"","subject":"Daily Report","body":"{{.n1.body}}"},"position":{"x":760,"y":250}}]',
 '[{"id":"e0","source":"t0","target":"n1"},{"id":"e1","source":"n1","target":"n2"}]'),
('tpl_form_to_db', 'Form Submission Handler', 'Process form data and store results', 'data', 'file-text', 'webhook',
 '[{"id":"t0","type":"trigger","label":"Form Submit","config":{},"position":{"x":200,"y":250}},{"id":"n1","type":"edit_fields","label":"Transform","config":{"fields":"{}"},"position":{"x":480,"y":250}},{"id":"n2","type":"http_request","label":"Store","config":{"url":"","method":"POST"},"position":{"x":760,"y":250}}]',
 '[{"id":"e0","source":"t0","target":"n1"},{"id":"e1","source":"n1","target":"n2"}]'),
('tpl_if_branch', 'Conditional Routing', 'Route data based on conditions', 'flow', 'git-branch', 'manual',
 '[{"id":"t0","type":"trigger","label":"Start","config":{},"position":{"x":200,"y":250}},{"id":"n1","type":"if_condition","label":"Check Status","config":{"field":"trigger.status","operator":"eq","value":"active"},"position":{"x":480,"y":250}},{"id":"n2","type":"slack","label":"Notify Active","config":{"message":"Active!"},"position":{"x":760,"y":150}},{"id":"n3","type":"send_email","label":"Alert Inactive","config":{"subject":"Inactive alert"},"position":{"x":760,"y":350}}]',
 '[{"id":"e0","source":"t0","target":"n1"},{"id":"e1","source":"n1","target":"n2"},{"id":"e2","source":"n1","target":"n3"}]'),
('tpl_data_transform', 'Data Pipeline', 'Fetch, transform, and aggregate data', 'data', 'bar-chart', 'manual',
 '[{"id":"t0","type":"trigger","label":"Start","config":{},"position":{"x":200,"y":250}},{"id":"n1","type":"http_request","label":"Fetch","config":{"url":"","method":"GET"},"position":{"x":440,"y":250}},{"id":"n2","type":"filter","label":"Filter","config":{"field":"","operator":"not_empty"},"position":{"x":680,"y":250}},{"id":"n3","type":"aggregate","label":"Summarize","config":{"operation":"count"},"position":{"x":920,"y":250}}]',
 '[{"id":"e0","source":"t0","target":"n1"},{"id":"e1","source":"n1","target":"n2"},{"id":"e2","source":"n2","target":"n3"}]'),
('tpl_ai_summarize', 'AI Content Summarizer', 'Use AI to summarize text content', 'ai', 'brain', 'webhook',
 '[{"id":"t0","type":"trigger","label":"Webhook","config":{},"position":{"x":200,"y":250}},{"id":"n1","type":"ai_transform","label":"Summarize","config":{"model":"claude-sonnet-4-20250514","prompt":"Summarize this: {{.trigger.body.text}}"},"position":{"x":500,"y":250}},{"id":"n2","type":"http_request","label":"Store Result","config":{"url":"","method":"POST","body":"{{.n1.result}}"},"position":{"x":800,"y":250}}]',
 '[{"id":"e0","source":"t0","target":"n1"},{"id":"e1","source":"n1","target":"n2"}]'),
('tpl_multi_channel', 'Multi-Channel Notification', 'Send to Slack, Discord, and Email', 'notifications', 'megaphone', 'webhook',
 '[{"id":"t0","type":"trigger","label":"Webhook","config":{},"position":{"x":200,"y":300}},{"id":"n1","type":"slack","label":"Slack","config":{"message":"{{.trigger.body.message}}"},"position":{"x":500,"y":150}},{"id":"n2","type":"discord","label":"Discord","config":{"message":"{{.trigger.body.message}}"},"position":{"x":500,"y":300}},{"id":"n3","type":"send_email","label":"Email","config":{"subject":"Notification","body":"{{.trigger.body.message}}"},"position":{"x":500,"y":450}}]',
 '[{"id":"e0","source":"t0","target":"n1"},{"id":"e1","source":"t0","target":"n2"},{"id":"e2","source":"t0","target":"n3"}]')
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Seed: deploy templates
-- ---------------------------------------------------------------------------
INSERT INTO deploy_templates (id, name, description, category, framework, use_case, repo_url, branch, build_cmd, output_dir, install_cmd, icon) VALUES
('tpl_nextjs_starter', 'Next.js Starter', 'A minimal Next.js app with App Router', 'sites', 'nextjs', 'starter', 'https://github.com/vercel/next.js/tree/canary/examples/hello-world', 'main', 'npm run build', '.next', 'npm install', 'nextjs'),
('tpl_astro_blog', 'Astro Blog', 'A blog template built with Astro', 'sites', 'astro', 'blog', 'https://github.com/withastro/astro/tree/main/examples/blog', 'main', 'npm run build', 'dist', 'npm install', 'astro'),
('tpl_svelte_starter', 'SvelteKit Starter', 'A minimal SvelteKit application', 'sites', 'sveltekit', 'starter', '', 'main', 'npm run build', 'build', 'npm install', 'sveltekit'),
('tpl_nuxt_starter', 'Nuxt Starter', 'A Nuxt 3 starter template', 'sites', 'nuxt', 'starter', 'https://github.com/nuxt/starter', 'main', 'npm run build', '.output/public', 'npm install', 'nuxt'),
('tpl_react_vite', 'React + Vite', 'React with Vite for fast development', 'sites', 'react', 'starter', '', 'main', 'npm run build', 'dist', 'npm install', 'react'),
('tpl_vue_starter', 'Vue.js Starter', 'Vue 3 with Vite', 'sites', 'vue', 'starter', '', 'main', 'npm run build', 'dist', 'npm install', 'vue'),
('tpl_flutter_web', 'Flutter Web', 'Flutter application deployed as a web app', 'sites', 'flutter', 'starter', '', 'main', 'flutter build web --release', 'build/web', '', 'flutter'),
('tpl_static_html', 'Static HTML', 'Plain HTML/CSS/JS site', 'sites', 'static', 'starter', '', 'main', '', '.', '', 'html'),
('tpl_docker_node', 'Node.js API', 'Node.js REST API with Express', 'containers', 'nodejs', 'api', '', 'main', '', '', 'npm install', 'nodejs'),
('tpl_docker_go', 'Go API', 'Go REST API with Chi router', 'containers', 'go', 'api', '', 'main', 'go build -o /app/server ./cmd/api', '', '', 'go'),
('tpl_docker_python', 'Python API', 'Python FastAPI application', 'containers', 'python', 'api', '', 'main', '', '', 'pip install -r requirements.txt', 'python'),
('tpl_flutter_mobile', 'Flutter Mobile', 'Cross-platform Flutter app', 'mobile', 'flutter', 'starter', '', 'main', 'flutter build apk --release', 'build/app/outputs/flutter-apk', '', 'flutter'),
('tpl_flutter_desktop', 'Flutter Desktop', 'Cross-platform Flutter desktop app', 'desktop', 'flutter', 'starter', '', 'main', 'flutter build linux --release', 'build/linux/x64/release/bundle', '', 'flutter'),
('tpl_electron', 'Electron App', 'Desktop app with Electron', 'desktop', 'electron', 'starter', '', 'main', 'npm run build', 'dist', 'npm install', 'electron'),
('tpl_tauri', 'Tauri App', 'Lightweight desktop app with Tauri', 'desktop', 'tauri', 'starter', '', 'main', 'npm run tauri build', 'src-tauri/target/release', 'npm install', 'tauri')
ON CONFLICT (id) DO NOTHING;
