CREATE TABLE IF NOT EXISTS project_oauth_providers (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id     VARCHAR(36)  NOT NULL,
    provider       VARCHAR(32)  NOT NULL,
    enabled        TINYINT(1)   NOT NULL DEFAULT 1,
    client_id      VARCHAR(512) NOT NULL,
    client_secret  VARCHAR(512) NOT NULL,
    created_at     DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_project_provider (project_id, provider),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
