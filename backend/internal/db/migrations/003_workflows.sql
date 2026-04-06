CREATE TABLE IF NOT EXISTS workflows (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id     VARCHAR(36)  NOT NULL,
    name           VARCHAR(128) NOT NULL,
    description    TEXT,
    status         VARCHAR(32)  NOT NULL DEFAULT 'draft',
    trigger_type   VARCHAR(32)  NOT NULL DEFAULT 'manual',
    trigger_config JSON,
    nodes          JSON,
    edges          JSON,
    created_at     DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_wf_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS workflow_executions (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    workflow_id    VARCHAR(36)  NOT NULL,
    project_id     VARCHAR(36)  NOT NULL,
    status         VARCHAR(32)  NOT NULL DEFAULT 'pending',
    trigger_data   JSON,
    started_at     DATETIME(3),
    completed_at   DATETIME(3),
    duration_ms    BIGINT       NOT NULL DEFAULT 0,
    error          TEXT,
    logs           JSON,
    INDEX idx_wfe_workflow (workflow_id),
    INDEX idx_wfe_project (project_id),
    FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
