-- ═══════════════════════════════════════════════════════════════════════════
-- 015: P3 features — custom domains, build agents, migrations, CSV import
-- ═══════════════════════════════════════════════════════════════════════════

-- Custom domains for web deploy targets
CREATE TABLE IF NOT EXISTS custom_domains (
    id              VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id      VARCHAR(36)  NOT NULL,
    target_id       VARCHAR(36)  NOT NULL,
    domain          VARCHAR(256) NOT NULL UNIQUE,
    verification    VARCHAR(256),
    verified        TINYINT(1)   NOT NULL DEFAULT 0,
    ssl_status      VARCHAR(32)  NOT NULL DEFAULT 'pending',
    ssl_expires_at  DATETIME(3),
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_cd_project (project_id),
    INDEX idx_cd_target (target_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Build agents for mobile/container builds
CREATE TABLE IF NOT EXISTS build_agents (
    id              VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id      VARCHAR(36)  NOT NULL,
    name            VARCHAR(128) NOT NULL,
    token           VARCHAR(256) NOT NULL UNIQUE,
    labels          JSON,
    status          VARCHAR(32)  NOT NULL DEFAULT 'offline',
    last_heartbeat  DATETIME(3),
    current_job_id  VARCHAR(36),
    os              VARCHAR(32),
    arch            VARCHAR(32),
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_ba_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Container registry entries
CREATE TABLE IF NOT EXISTS registry_images (
    id              VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id      VARCHAR(36)  NOT NULL,
    target_id       VARCHAR(36)  NOT NULL,
    repository      VARCHAR(256) NOT NULL,
    tag             VARCHAR(128) NOT NULL,
    digest          VARCHAR(128),
    size_bytes      BIGINT       NOT NULL DEFAULT 0,
    platform        VARCHAR(32),
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_ri_project (project_id),
    INDEX idx_ri_target (target_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Migrations (import from external platforms)
CREATE TABLE IF NOT EXISTS migrations (
    id              VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id      VARCHAR(36)  NOT NULL,
    source          VARCHAR(32)  NOT NULL,
    status          VARCHAR(32)  NOT NULL DEFAULT 'pending',
    resources       JSON,
    errors          JSON,
    progress        INT          NOT NULL DEFAULT 0,
    started_at      DATETIME(3),
    completed_at    DATETIME(3),
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_mig_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Add web hosting fields to deploy targets
ALTER TABLE deploy_targets ADD COLUMN IF NOT EXISTS domain VARCHAR(256) AFTER cron;
ALTER TABLE deploy_targets ADD COLUMN IF NOT EXISTS ssl_enabled TINYINT(1) NOT NULL DEFAULT 0 AFTER domain;
ALTER TABLE deploy_targets ADD COLUMN IF NOT EXISTS output_dir VARCHAR(256) AFTER ssl_enabled;

-- Add container fields to deploy targets
ALTER TABLE deploy_targets ADD COLUMN IF NOT EXISTS dockerfile VARCHAR(256) AFTER output_dir;
ALTER TABLE deploy_targets ADD COLUMN IF NOT EXISTS registry_url VARCHAR(512) AFTER dockerfile;
ALTER TABLE deploy_targets ADD COLUMN IF NOT EXISTS tag_strategy VARCHAR(32) DEFAULT 'latest' AFTER registry_url;

-- Add mobile fields to deploy targets
ALTER TABLE deploy_targets ADD COLUMN IF NOT EXISTS build_type VARCHAR(32) AFTER tag_strategy;
ALTER TABLE deploy_targets ADD COLUMN IF NOT EXISTS signing_config JSON AFTER build_type;
ALTER TABLE deploy_targets ADD COLUMN IF NOT EXISTS store_config JSON AFTER signing_config;

-- Add agent label targeting to deploy pipelines
ALTER TABLE deploy_pipelines ADD COLUMN IF NOT EXISTS agent_label VARCHAR(64) AFTER cache_dirs;
