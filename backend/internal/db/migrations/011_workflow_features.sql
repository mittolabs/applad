-- ═══════════════════════════════════════════════════════════════════════════
-- 011: Workflow features — credentials, folders, tags, versions, sharing, templates
-- ═══════════════════════════════════════════════════════════════════════════

-- Encrypted credentials store
CREATE TABLE IF NOT EXISTS credentials (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    name        VARCHAR(128) NOT NULL,
    type        VARCHAR(64)  NOT NULL,
    data        TEXT         NOT NULL,
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_cred_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Workflow folders
CREATE TABLE IF NOT EXISTS workflow_folders (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(36)  NOT NULL,
    name        VARCHAR(128) NOT NULL,
    parent_id   VARCHAR(36),
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_wff_project (project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Add folder_id and tags to workflows
ALTER TABLE workflows ADD COLUMN folder_id VARCHAR(36) AFTER project_id;
ALTER TABLE workflows ADD COLUMN tags JSON AFTER edges;
ALTER TABLE workflows ADD COLUMN error_workflow_id VARCHAR(36) AFTER tags;
ALTER TABLE workflows ADD COLUMN retry_attempts INT NOT NULL DEFAULT 0 AFTER error_workflow_id;
ALTER TABLE workflows ADD COLUMN retry_delay_ms INT NOT NULL DEFAULT 1000 AFTER retry_attempts;
ALTER TABLE workflows ADD INDEX idx_wf_folder (folder_id);

-- Workflow version history
CREATE TABLE IF NOT EXISTS workflow_versions (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    workflow_id  VARCHAR(36)  NOT NULL,
    version      INT          NOT NULL DEFAULT 1,
    name         VARCHAR(128) NOT NULL,
    description  TEXT,
    nodes        JSON,
    edges        JSON,
    trigger_type VARCHAR(32),
    trigger_config JSON,
    created_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    created_by   VARCHAR(36),
    INDEX idx_wfv_workflow (workflow_id),
    FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Workflow sharing
CREATE TABLE IF NOT EXISTS workflow_shares (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    workflow_id  VARCHAR(36)  NOT NULL,
    user_id      VARCHAR(36)  NOT NULL,
    role         VARCHAR(32)  NOT NULL DEFAULT 'viewer',
    created_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_wfs_workflow (workflow_id),
    INDEX idx_wfs_user (user_id),
    FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
    UNIQUE KEY uk_wfs (workflow_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Workflow templates
CREATE TABLE IF NOT EXISTS workflow_templates (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY,
    name         VARCHAR(128) NOT NULL,
    description  TEXT,
    category     VARCHAR(64)  NOT NULL DEFAULT 'general',
    icon         VARCHAR(64),
    nodes        JSON,
    edges        JSON,
    trigger_type VARCHAR(32)  NOT NULL DEFAULT 'manual',
    trigger_config JSON,
    popularity   INT          NOT NULL DEFAULT 0,
    created_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Seed workflow templates
INSERT INTO workflow_templates (id, name, description, category, icon, trigger_type, nodes, edges) VALUES
('tpl_webhook_slack', 'Webhook to Slack', 'Forward incoming webhooks to a Slack channel', 'notifications', 'webhook',
 'webhook',
 '[{"id":"t0","type":"trigger","label":"Webhook","config":{},"position":{"x":200,"y":250}},{"id":"n1","type":"slack","label":"Send to Slack","config":{"webhookUrl":"","message":"{{.trigger.body}}"},"position":{"x":500,"y":250}}]',
 '[{"id":"e0","source":"t0","target":"n1"}]'),

('tpl_cron_email', 'Scheduled Email Report', 'Send a recurring email report on a schedule', 'notifications', 'clock',
 'cron',
 '[{"id":"t0","type":"trigger","label":"Schedule","config":{},"position":{"x":200,"y":250}},{"id":"n1","type":"http_request","label":"Fetch Data","config":{"url":"","method":"GET"},"position":{"x":480,"y":250}},{"id":"n2","type":"send_email","label":"Send Report","config":{"to":"","subject":"Daily Report","body":"{{.n1.body}}"},"position":{"x":760,"y":250}}]',
 '[{"id":"e0","source":"t0","target":"n1"},{"id":"e1","source":"n1","target":"n2"}]'),

('tpl_form_to_db', 'Form Submission Handler', 'Process form data and store results', 'data', 'file-text',
 'webhook',
 '[{"id":"t0","type":"trigger","label":"Form Submit","config":{},"position":{"x":200,"y":250}},{"id":"n1","type":"edit_fields","label":"Transform","config":{"fields":"{}"},"position":{"x":480,"y":250}},{"id":"n2","type":"http_request","label":"Store","config":{"url":"","method":"POST"},"position":{"x":760,"y":250}}]',
 '[{"id":"e0","source":"t0","target":"n1"},{"id":"e1","source":"n1","target":"n2"}]'),

('tpl_if_branch', 'Conditional Routing', 'Route data based on conditions', 'flow', 'git-branch',
 'manual',
 '[{"id":"t0","type":"trigger","label":"Start","config":{},"position":{"x":200,"y":250}},{"id":"n1","type":"if_condition","label":"Check Status","config":{"field":"trigger.status","operator":"eq","value":"active"},"position":{"x":480,"y":250}},{"id":"n2","type":"slack","label":"Notify Active","config":{"message":"Active!"},"position":{"x":760,"y":150}},{"id":"n3","type":"send_email","label":"Alert Inactive","config":{"subject":"Inactive alert"},"position":{"x":760,"y":350}}]',
 '[{"id":"e0","source":"t0","target":"n1"},{"id":"e1","source":"n1","target":"n2"},{"id":"e2","source":"n1","target":"n3"}]'),

('tpl_data_transform', 'Data Pipeline', 'Fetch, transform, and aggregate data', 'data', 'bar-chart',
 'manual',
 '[{"id":"t0","type":"trigger","label":"Start","config":{},"position":{"x":200,"y":250}},{"id":"n1","type":"http_request","label":"Fetch","config":{"url":"","method":"GET"},"position":{"x":440,"y":250}},{"id":"n2","type":"filter","label":"Filter","config":{"field":"","operator":"not_empty"},"position":{"x":680,"y":250}},{"id":"n3","type":"aggregate","label":"Summarize","config":{"operation":"count"},"position":{"x":920,"y":250}}]',
 '[{"id":"e0","source":"t0","target":"n1"},{"id":"e1","source":"n1","target":"n2"},{"id":"e2","source":"n2","target":"n3"}]'),

('tpl_error_handler', 'Error Handler Pattern', 'Workflow with try-catch error handling', 'flow', 'shield',
 'manual',
 '[{"id":"t0","type":"trigger","label":"Start","config":{},"position":{"x":200,"y":250}},{"id":"n1","type":"try_catch","label":"Try","config":{},"position":{"x":440,"y":250}},{"id":"n2","type":"http_request","label":"Risky Call","config":{"url":"","method":"GET"},"position":{"x":680,"y":180}},{"id":"n3","type":"slack","label":"Error Alert","config":{"message":"Error occurred"},"position":{"x":680,"y":340}}]',
 '[{"id":"e0","source":"t0","target":"n1"},{"id":"e1","source":"n1","target":"n2"},{"id":"e2","source":"n1","target":"n3"}]'),

('tpl_ai_summarize', 'AI Content Summarizer', 'Use AI to summarize text content', 'ai', 'brain',
 'webhook',
 '[{"id":"t0","type":"trigger","label":"Webhook","config":{},"position":{"x":200,"y":250}},{"id":"n1","type":"ai_transform","label":"Summarize","config":{"model":"claude-sonnet-4-20250514","prompt":"Summarize this: {{.trigger.body.text}}"},"position":{"x":500,"y":250}},{"id":"n2","type":"http_request","label":"Store Result","config":{"url":"","method":"POST","body":"{{.n1.result}}"},"position":{"x":800,"y":250}}]',
 '[{"id":"e0","source":"t0","target":"n1"},{"id":"e1","source":"n1","target":"n2"}]'),

('tpl_github_notify', 'GitHub Issue Notifier', 'Create GitHub issues and notify on Slack', 'integrations', 'github',
 'webhook',
 '[{"id":"t0","type":"trigger","label":"Webhook","config":{},"position":{"x":200,"y":250}},{"id":"n1","type":"github","label":"Create Issue","config":{"action":"create_issue","title":"{{.trigger.body.title}}","body":"{{.trigger.body.description}}"},"position":{"x":480,"y":250}},{"id":"n2","type":"slack","label":"Notify","config":{"message":"New issue: {{.trigger.body.title}}"},"position":{"x":760,"y":250}}]',
 '[{"id":"e0","source":"t0","target":"n1"},{"id":"e1","source":"n1","target":"n2"}]'),

('tpl_loop_batch', 'Batch Processor', 'Loop through items and process each', 'data', 'repeat',
 'manual',
 '[{"id":"t0","type":"trigger","label":"Start","config":{},"position":{"x":200,"y":250}},{"id":"n1","type":"http_request","label":"Fetch Items","config":{"url":"","method":"GET"},"position":{"x":440,"y":250}},{"id":"n2","type":"loop","label":"Loop","config":{"items":"n1.body.items","loopVariable":"item"},"position":{"x":680,"y":250}},{"id":"n3","type":"http_request","label":"Process","config":{"url":"","method":"POST","body":"{{.item}}"},"position":{"x":920,"y":250}}]',
 '[{"id":"e0","source":"t0","target":"n1"},{"id":"e1","source":"n1","target":"n2"},{"id":"e2","source":"n2","target":"n3"}]'),

('tpl_multi_channel', 'Multi-Channel Notification', 'Send to Slack, Discord, and Email', 'notifications', 'megaphone',
 'webhook',
 '[{"id":"t0","type":"trigger","label":"Webhook","config":{},"position":{"x":200,"y":300}},{"id":"n1","type":"slack","label":"Slack","config":{"message":"{{.trigger.body.message}}"},"position":{"x":500,"y":150}},{"id":"n2","type":"discord","label":"Discord","config":{"message":"{{.trigger.body.message}}"},"position":{"x":500,"y":300}},{"id":"n3","type":"send_email","label":"Email","config":{"subject":"Notification","body":"{{.trigger.body.message}}"},"position":{"x":500,"y":450}}]',
 '[{"id":"e0","source":"t0","target":"n1"},{"id":"e1","source":"t0","target":"n2"},{"id":"e2","source":"t0","target":"n3"}]');
