-- ═══════════════════════════════════════════════════════════════════════════
-- 014: Feature Flags — flags, rules, overrides, analytics
-- ═══════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS feature_flags (
    id              VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id      VARCHAR(36)  NOT NULL,
    key_name        VARCHAR(128) NOT NULL,
    name            VARCHAR(256) NOT NULL,
    description     TEXT,
    type            VARCHAR(16)  NOT NULL DEFAULT 'boolean',
    default_value   JSON         NOT NULL,
    enabled         TINYINT(1)   NOT NULL DEFAULT 0,
    tags            JSON,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_ff_key (project_id, key_name),
    INDEX idx_ff_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS flag_rules (
    id              VARCHAR(36)  NOT NULL PRIMARY KEY,
    flag_id         VARCHAR(36)  NOT NULL,
    priority        INT          NOT NULL DEFAULT 0,
    type            VARCHAR(32)  NOT NULL,
    conditions      JSON         NOT NULL,
    value           JSON         NOT NULL,
    rollout_pct     INT          NOT NULL DEFAULT 100,
    enabled         TINYINT(1)   NOT NULL DEFAULT 1,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_fr_flag (flag_id),
    FOREIGN KEY (flag_id) REFERENCES feature_flags(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS flag_overrides (
    id              VARCHAR(36)  NOT NULL PRIMARY KEY,
    flag_id         VARCHAR(36)  NOT NULL,
    target_type     VARCHAR(32)  NOT NULL,
    target_id       VARCHAR(128) NOT NULL,
    value           JSON         NOT NULL,
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_fo (flag_id, target_type, target_id),
    INDEX idx_fo_flag (flag_id),
    FOREIGN KEY (flag_id) REFERENCES feature_flags(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS flag_evaluations (
    id              VARCHAR(36)  NOT NULL PRIMARY KEY,
    flag_id         VARCHAR(36)  NOT NULL,
    project_id      VARCHAR(36)  NOT NULL,
    user_id         VARCHAR(128),
    value           JSON         NOT NULL,
    rule_id         VARCHAR(36),
    timestamp       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_fev_flag (flag_id, timestamp),
    INDEX idx_fev_project (project_id, timestamp),
    FOREIGN KEY (flag_id) REFERENCES feature_flags(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
