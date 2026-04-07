-- Migration: migrations table for tracking data imports from external platforms
CREATE TABLE IF NOT EXISTS migrations (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) NOT NULL,
    source VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    config JSON,
    resources JSON,
    errors JSON,
    progress INT NOT NULL DEFAULT 0,
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_migrations_project (project_id),
    INDEX idx_migrations_status (status)
);
