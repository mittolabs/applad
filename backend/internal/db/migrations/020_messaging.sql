-- Messages: persistent log of every sent/draft message
CREATE TABLE IF NOT EXISTS messages (
    id          VARCHAR(32)  NOT NULL PRIMARY KEY,
    project_id  VARCHAR(32)  NOT NULL,
    type        VARCHAR(10)  NOT NULL, -- email | sms | push
    subject     VARCHAR(512) NOT NULL DEFAULT '',
    body        TEXT         NOT NULL DEFAULT '',
    recipients  TEXT         NOT NULL DEFAULT '',  -- JSON array of strings
    status      VARCHAR(20)  NOT NULL DEFAULT 'processing', -- processing | sent | failed | draft
    scheduled_at DATETIME    NULL,
    delivered_at DATETIME    NULL,
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_messages_project (project_id),
    INDEX idx_messages_status  (status),
    INDEX idx_messages_type    (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Topics: persistent pub/sub topics
CREATE TABLE IF NOT EXISTS msg_topics (
    id         VARCHAR(32)  NOT NULL PRIMARY KEY,
    project_id VARCHAR(32)  NOT NULL,
    name       VARCHAR(256) NOT NULL,
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_topic_project_name (project_id, name),
    INDEX idx_topics_project (project_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Topic subscribers
CREATE TABLE IF NOT EXISTS msg_topic_subscribers (
    id         BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    topic_id   VARCHAR(32)  NOT NULL,
    target     VARCHAR(512) NOT NULL,
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_subscriber (topic_id, target(255)),
    INDEX idx_subscriber_topic (topic_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
