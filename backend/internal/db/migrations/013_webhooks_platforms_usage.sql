-- Outbound webhooks
CREATE TABLE IF NOT EXISTS webhooks (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    project_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    events JSON,
    secret VARCHAR(255) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    INDEX idx_webhooks_project (project_id)
);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    webhook_id VARCHAR(36) NOT NULL,
    event VARCHAR(255) NOT NULL,
    payload LONGTEXT,
    status_code INT NOT NULL DEFAULT 0,
    response TEXT,
    attempts INT NOT NULL DEFAULT 0,
    success BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL,
    INDEX idx_deliveries_webhook (webhook_id),
    INDEX idx_deliveries_created (created_at)
);

-- Platform registration
CREATE TABLE IF NOT EXISTS platforms (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    project_id VARCHAR(36) NOT NULL,
    type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    hostname VARCHAR(255),
    store_id VARCHAR(255),
    created_at DATETIME NOT NULL,
    INDEX idx_platforms_project (project_id)
);

-- Project config columns
ALTER TABLE projects ADD COLUMN IF NOT EXISTS auth_config JSON;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS services_config JSON;

-- Usage analytics metrics
CREATE TABLE IF NOT EXISTS usage_metrics (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    project_id VARCHAR(36) NOT NULL,
    metric VARCHAR(100) NOT NULL,
    value BIGINT NOT NULL DEFAULT 0,
    period VARCHAR(10) NOT NULL DEFAULT '1h',
    timestamp DATETIME NOT NULL,
    INDEX idx_usage_project_metric (project_id, metric),
    INDEX idx_usage_timestamp (timestamp)
);
