ALTER TABLE users ADD COLUMN oauth_provider VARCHAR(32) AFTER phone;
ALTER TABLE users ADD COLUMN oauth_id VARCHAR(256) AFTER oauth_provider;
ALTER TABLE users ADD INDEX idx_users_oauth (project_id, oauth_provider, oauth_id);
