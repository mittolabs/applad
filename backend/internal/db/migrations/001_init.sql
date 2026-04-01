CREATE TABLE IF NOT EXISTS projects (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    name         VARCHAR(128) NOT NULL,
    description  TEXT,
    created_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS api_keys (
    id           VARCHAR(36)   NOT NULL PRIMARY KEY,
    project_id   VARCHAR(36)   NOT NULL,
    name         VARCHAR(128)  NOT NULL,
    secret_hash  VARCHAR(256)  NOT NULL,
    secret_prefix VARCHAR(16)  NOT NULL,
    scopes       JSON,
    expires_at   DATETIME(3),
    created_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_ak_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS users (
    id               VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id       VARCHAR(36)  NOT NULL,
    email            VARCHAR(256),
    phone            VARCHAR(32),
    name             VARCHAR(128),
    password_hash    VARCHAR(256),
    email_verified   TINYINT(1)   NOT NULL DEFAULT 0,
    phone_verified   TINYINT(1)   NOT NULL DEFAULT 0,
    status           TINYINT(1)   NOT NULL DEFAULT 1,
    labels           JSON,
    prefs            JSON,
    created_at       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_users_project (project_id),
    UNIQUE KEY uq_users_email_project (project_id, email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS sessions (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    user_id      VARCHAR(36)  NOT NULL,
    project_id   VARCHAR(36)  NOT NULL,
    ip           VARCHAR(64),
    user_agent   TEXT,
    expires_at   DATETIME(3)  NOT NULL,
    created_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_sess_user (user_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS teams (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id   VARCHAR(36)  NOT NULL,
    name         VARCHAR(128) NOT NULL,
    prefs        JSON,
    created_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_teams_project (project_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS memberships (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    team_id        VARCHAR(36)  NOT NULL,
    user_id        VARCHAR(36),
    invited_email  VARCHAR(256),
    roles          JSON,
    invited        TINYINT(1)   NOT NULL DEFAULT 1,
    joined         TINYINT(1)   NOT NULL DEFAULT 0,
    secret         VARCHAR(512),
    created_at     DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `_databases` (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id   VARCHAR(36)  NOT NULL,
    name         VARCHAR(128) NOT NULL,
    created_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_db_project (project_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS collections (
    id                VARCHAR(36)  NOT NULL PRIMARY KEY,
    database_id       VARCHAR(36)  NOT NULL,
    project_id        VARCHAR(36)  NOT NULL,
    name              VARCHAR(128) NOT NULL,
    document_security TINYINT(1)   NOT NULL DEFAULT 0,
    permissions       JSON,
    enabled           TINYINT(1)   NOT NULL DEFAULT 1,
    created_at        DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at        DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_coll_db (database_id),
    FOREIGN KEY (database_id) REFERENCES `_databases`(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS attributes (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    collection_id  VARCHAR(36)  NOT NULL,
    `key`          VARCHAR(128) NOT NULL,
    type           VARCHAR(32)  NOT NULL,
    required       TINYINT(1)   NOT NULL DEFAULT 0,
    array          TINYINT(1)   NOT NULL DEFAULT 0,
    default_value  TEXT,
    options        JSON,
    status         VARCHAR(32)  NOT NULL DEFAULT 'available',
    created_at     DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_attr (collection_id, `key`),
    FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `_indexes` (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    collection_id  VARCHAR(36)  NOT NULL,
    `key`          VARCHAR(128) NOT NULL,
    type           VARCHAR(32)  NOT NULL,
    attributes     JSON         NOT NULL,
    status         VARCHAR(32)  NOT NULL DEFAULT 'available',
    created_at     DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_idx (collection_id, `key`),
    FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS documents (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    collection_id  VARCHAR(36)  NOT NULL,
    database_id    VARCHAR(36)  NOT NULL,
    project_id     VARCHAR(36)  NOT NULL,
    data           JSON         NOT NULL,
    permissions    JSON,
    created_at     DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_doc_coll (collection_id),
    FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS buckets (
    id                    VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id            VARCHAR(36)  NOT NULL,
    name                  VARCHAR(128) NOT NULL,
    permissions           JSON,
    file_size_limit       BIGINT       NOT NULL DEFAULT 0,
    allowed_mime_types    JSON,
    encryption            TINYINT(1)   NOT NULL DEFAULT 0,
    antivirus             TINYINT(1)   NOT NULL DEFAULT 0,
    compression           VARCHAR(32)  NOT NULL DEFAULT 'none',
    image_transformations TINYINT(1)   NOT NULL DEFAULT 1,
    enabled               TINYINT(1)   NOT NULL DEFAULT 1,
    created_at            DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at            DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_bucket_project (project_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS files (
    id           VARCHAR(36)   NOT NULL PRIMARY KEY,
    bucket_id    VARCHAR(36)   NOT NULL,
    project_id   VARCHAR(36)   NOT NULL,
    name         VARCHAR(512)  NOT NULL,
    mime_type    VARCHAR(128)  NOT NULL DEFAULT 'application/octet-stream',
    size         BIGINT        NOT NULL,
    permissions  JSON,
    path         VARCHAR(1024) NOT NULL,
    created_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_file_bucket (bucket_id),
    FOREIGN KEY (bucket_id) REFERENCES buckets(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS schema_migrations (
    version     VARCHAR(32) NOT NULL PRIMARY KEY,
    applied_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
