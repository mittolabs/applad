-- ═══════════════════════════════════════════════════════════════════════════
-- 012: P0/P1 features — RBAC, webhooks, platforms, usage, user logs, deploy model
-- ═══════════════════════════════════════════════════════════════════════════

-- ── Granular RBAC permissions ──
CREATE TABLE IF NOT EXISTS permissions (
    id            VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id    VARCHAR(36)  NOT NULL,
    resource_type VARCHAR(32)  NOT NULL,
    resource_id   VARCHAR(36)  NOT NULL,
    role          VARCHAR(128) NOT NULL,
    action        VARCHAR(32)  NOT NULL,
    created_at    DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_perm_project (project_id),
    INDEX idx_perm_resource (resource_type, resource_id),
    INDEX idx_perm_role (role),
    UNIQUE KEY uk_perm (project_id, resource_type, resource_id, role, action),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Add permissions JSON to collections and documents
ALTER TABLE collections ADD COLUMN IF NOT EXISTS permissions JSON AFTER project_id;
ALTER TABLE collections ADD COLUMN IF NOT EXISTS document_security TINYINT(1) NOT NULL DEFAULT 0 AFTER permissions;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS permissions JSON AFTER collection_id;

-- ── Outbound webhooks ──
CREATE TABLE IF NOT EXISTS webhooks (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    name        VARCHAR(128) NOT NULL,
    url         VARCHAR(512) NOT NULL,
    events      JSON         NOT NULL,
    secret      VARCHAR(256),
    enabled     TINYINT(1)   NOT NULL DEFAULT 1,
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_wh_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    webhook_id  VARCHAR(36)  NOT NULL,
    event       VARCHAR(64)  NOT NULL,
    payload     JSON,
    status_code INT,
    response    TEXT,
    attempts    INT          NOT NULL DEFAULT 0,
    success     TINYINT(1)   NOT NULL DEFAULT 0,
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_whd_webhook (webhook_id),
    FOREIGN KEY (webhook_id) REFERENCES webhooks(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── Platform registration ──
CREATE TABLE IF NOT EXISTS platforms (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    type        VARCHAR(32)  NOT NULL,
    name        VARCHAR(128) NOT NULL,
    hostname    VARCHAR(256),
    store_id    VARCHAR(256),
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_plat_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── Project configuration ──
ALTER TABLE projects ADD COLUMN IF NOT EXISTS auth_config JSON AFTER description;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS services_config JSON AFTER auth_config;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS smtp_config JSON AFTER services_config;

-- ── Usage metrics (time-series) ──
CREATE TABLE IF NOT EXISTS usage_metrics (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    metric      VARCHAR(64)  NOT NULL,
    value       BIGINT       NOT NULL DEFAULT 0,
    period      VARCHAR(16)  NOT NULL DEFAULT 'hour',
    timestamp   DATETIME(3)  NOT NULL,
    INDEX idx_um_project_metric (project_id, metric, timestamp),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── User activity logs ──
CREATE TABLE IF NOT EXISTS user_logs (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    user_id     VARCHAR(36)  NOT NULL,
    event       VARCHAR(64)  NOT NULL,
    ip          VARCHAR(45),
    user_agent  VARCHAR(512),
    details     JSON,
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_ul_user (project_id, user_id),
    INDEX idx_ul_event (project_id, event),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Add blocked status to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS blocked TINYINT(1) NOT NULL DEFAULT 0 AFTER status;
ALTER TABLE users ADD COLUMN IF NOT EXISTS labels JSON AFTER blocked;

-- ── Deploy: Target+Pipeline+Release model ──
CREATE TABLE IF NOT EXISTS deploy_targets (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    name        VARCHAR(128) NOT NULL,
    type        VARCHAR(32)  NOT NULL DEFAULT 'serverless',
    runtime     VARCHAR(64),
    entrypoint  VARCHAR(256),
    timeout_ms  INT          NOT NULL DEFAULT 30000,
    memory_mb   INT          NOT NULL DEFAULT 256,
    env_vars    JSON,
    permissions JSON,
    cron        VARCHAR(128),
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_dt_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

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
    env_vars    JSON,
    trigger_on  JSON,
    cache_dirs  JSON,
    timeout_ms  INT          NOT NULL DEFAULT 600000,
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_dp_project (project_id),
    INDEX idx_dp_target (target_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (target_id) REFERENCES deploy_targets(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS deploy_releases (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id   VARCHAR(36)  NOT NULL,
    pipeline_id  VARCHAR(36)  NOT NULL,
    target_id    VARCHAR(36)  NOT NULL,
    status       VARCHAR(32)  NOT NULL DEFAULT 'pending',
    trigger_type VARCHAR(32)  NOT NULL DEFAULT 'manual',
    trigger_actor VARCHAR(128),
    commit_sha   VARCHAR(64),
    build_log    LONGTEXT,
    deploy_log   LONGTEXT,
    artifact_path VARCHAR(512),
    duration_ms  BIGINT       NOT NULL DEFAULT 0,
    error        TEXT,
    started_at   DATETIME(3),
    completed_at DATETIME(3),
    created_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_dr_project (project_id),
    INDEX idx_dr_pipeline (pipeline_id),
    INDEX idx_dr_target (target_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (pipeline_id) REFERENCES deploy_pipelines(id) ON DELETE CASCADE,
    FOREIGN KEY (target_id) REFERENCES deploy_targets(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── Serverless executions (proper model) ──
CREATE TABLE IF NOT EXISTS deploy_executions (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    target_id   VARCHAR(36)  NOT NULL,
    status      VARCHAR(32)  NOT NULL DEFAULT 'pending',
    status_code INT          NOT NULL DEFAULT 0,
    request     JSON,
    response    TEXT,
    stdout      TEXT,
    stderr      TEXT,
    duration_ms BIGINT       NOT NULL DEFAULT 0,
    trigger     VARCHAR(32)  NOT NULL DEFAULT 'http',
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_de_project (project_id),
    INDEX idx_de_target (target_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (target_id) REFERENCES deploy_targets(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
