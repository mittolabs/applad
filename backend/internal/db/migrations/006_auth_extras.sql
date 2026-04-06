ALTER TABLE users ADD COLUMN mfa_secret VARCHAR(64) AFTER oauth_id;
ALTER TABLE users ADD COLUMN mfa_enabled TINYINT(1) NOT NULL DEFAULT 0 AFTER mfa_secret;
ALTER TABLE users ADD COLUMN mfa_recovery JSON AFTER mfa_enabled;

CREATE TABLE IF NOT EXISTS auth_tokens (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    user_id    VARCHAR(36)  NOT NULL,
    project_id VARCHAR(36)  NOT NULL,
    type       VARCHAR(32)  NOT NULL,
    secret     VARCHAR(512) NOT NULL,
    expires_at DATETIME(3)  NOT NULL,
    created_at DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_at_user (user_id),
    INDEX idx_at_secret (secret),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
