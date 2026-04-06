CREATE TABLE IF NOT EXISTS functions (
    id          VARCHAR(36) NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    runtime     VARCHAR(50)  NOT NULL,
    entrypoint  VARCHAR(255) NOT NULL DEFAULT 'main',
    timeout     INT          NOT NULL DEFAULT 15,
    env_vars    JSON,
    source      LONGTEXT,
    status      VARCHAR(50)  NOT NULL DEFAULT 'active',
    created_at  DATETIME     NOT NULL,
    updated_at  DATETIME     NOT NULL,
    INDEX idx_functions_project (project_id)
);

CREATE TABLE IF NOT EXISTS function_executions (
    id          VARCHAR(36) NOT NULL PRIMARY KEY,
    function_id VARCHAR(36) NOT NULL,
    project_id  VARCHAR(36) NOT NULL,
    status      VARCHAR(50) NOT NULL DEFAULT 'pending',
    output      LONGTEXT,
    errors      LONGTEXT,
    duration    DOUBLE       NOT NULL DEFAULT 0,
    created_at  DATETIME     NOT NULL,
    INDEX idx_executions_function (function_id),
    INDEX idx_executions_project (project_id)
);
