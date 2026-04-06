CREATE TABLE IF NOT EXISTS organizations (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    name        VARCHAR(128) NOT NULL,
    billing_plan VARCHAR(32) NOT NULL DEFAULT 'free',
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS organization_members (
    id              VARCHAR(36)  NOT NULL PRIMARY KEY,
    org_id          VARCHAR(36)  NOT NULL,
    user_id         VARCHAR(36),
    email           VARCHAR(256) NOT NULL,
    name            VARCHAR(128),
    role            VARCHAR(32)  NOT NULL DEFAULT 'member',
    status          VARCHAR(32)  NOT NULL DEFAULT 'pending',
    invite_token    VARCHAR(256),
    created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_om_org (org_id),
    INDEX idx_om_user (user_id),
    INDEX idx_om_email (email),
    FOREIGN KEY (org_id) REFERENCES organizations(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Link projects to organizations
ALTER TABLE projects ADD COLUMN org_id VARCHAR(36) AFTER id;
ALTER TABLE projects ADD INDEX idx_proj_org (org_id);

-- Link console users to their default org
ALTER TABLE console_users ADD COLUMN default_org_id VARCHAR(36) AFTER name;
