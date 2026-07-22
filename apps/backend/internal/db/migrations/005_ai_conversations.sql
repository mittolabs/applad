-- AI conversation history (per console user, not per project)
CREATE TABLE IF NOT EXISTS ai_conversations (
    id          VARCHAR(36) PRIMARY KEY,
    user_id     VARCHAR(36) NOT NULL,
    title       TEXT        NOT NULL DEFAULT 'New conversation',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ai_messages (
    id              VARCHAR(36)  PRIMARY KEY,
    conversation_id VARCHAR(36)  NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    role            VARCHAR(16)  NOT NULL CHECK (role IN ('user', 'assistant')),
    content         TEXT         NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_conversations_user_id ON ai_conversations(user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_messages_conversation ON ai_messages(conversation_id, created_at ASC);
