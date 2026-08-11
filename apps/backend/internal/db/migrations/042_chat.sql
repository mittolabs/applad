-- E2E-encrypted chat: devices, prekey bundles, conversations, and messages.
-- The server stores and relays ciphertext, public keys, and routing metadata
-- only — no column here holds plaintext content or a private key. Table names
-- use a chat_ prefix to stay distinct from the existing messages/msg_topics/
-- msg_topic_subscribers tables, which are the unrelated email/SMS/push
-- messaging service (see internal/messaging).

-- Devices: one row per logical client install (phone app, browser tab,
-- desktop app) — NOT the same as a `sessions` row. A user has 1..N devices;
-- a device outlives many sessions. Public key material only.
CREATE TABLE IF NOT EXISTS chat_devices (
    id                VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id        VARCHAR(36)  NOT NULL,
    user_id           VARCHAR(36)  NOT NULL,
    name              VARCHAR(128) NOT NULL DEFAULT '',
    registration_id   INT          NOT NULL,
    identity_key      TEXT         NOT NULL,
    signed_prekey_id  INT          NOT NULL,
    signed_prekey     TEXT         NOT NULL,
    signed_prekey_sig TEXT         NOT NULL,
    push_token        TEXT,
    push_provider     VARCHAR(16),
    status            VARCHAR(16)  NOT NULL DEFAULT 'active',
    last_seen_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_cd_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cd_project ON chat_devices (project_id);
CREATE INDEX IF NOT EXISTS idx_cd_user ON chat_devices (user_id, status);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON chat_devices
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

-- One-time prekeys: consumed atomically (DELETE ... RETURNING) when another
-- device fetches a bundle to start an X3DH session.
CREATE TABLE IF NOT EXISTS chat_one_time_prekeys (
    id         VARCHAR(36) NOT NULL PRIMARY KEY,
    device_id  VARCHAR(36) NOT NULL,
    project_id VARCHAR(36) NOT NULL,
    key_id     INT         NOT NULL,
    public_key TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_cotp_device_key UNIQUE (device_id, key_id),
    CONSTRAINT fk_cotp_device FOREIGN KEY (device_id) REFERENCES chat_devices(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cotp_device ON chat_one_time_prekeys (device_id);

-- Conversations: direct (1:1) or group. next_seq is a per-conversation
-- monotonic counter, incremented atomically on each message send, so clients
-- can detect gaps and page a backfill by "after_seq".
CREATE TABLE IF NOT EXISTS chat_conversations (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id VARCHAR(36)  NOT NULL,
    type       VARCHAR(16)  NOT NULL DEFAULT 'direct',
    title      VARCHAR(256) NOT NULL DEFAULT '',
    created_by VARCHAR(36)  NOT NULL,
    next_seq   BIGINT       NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_cc_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cc_project ON chat_conversations (project_id);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON chat_conversations
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();

CREATE TABLE IF NOT EXISTS chat_conversation_members (
    id              VARCHAR(36) NOT NULL PRIMARY KEY,
    conversation_id VARCHAR(36) NOT NULL,
    project_id      VARCHAR(36) NOT NULL,
    user_id         VARCHAR(36) NOT NULL,
    role            VARCHAR(16) NOT NULL DEFAULT 'member',
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    removed_at      TIMESTAMPTZ,
    CONSTRAINT uq_ccm_conv_user UNIQUE (conversation_id, user_id),
    CONSTRAINT fk_ccm_conversation FOREIGN KEY (conversation_id) REFERENCES chat_conversations(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ccm_conversation ON chat_conversation_members (conversation_id);
CREATE INDEX IF NOT EXISTS idx_ccm_user ON chat_conversation_members (user_id);

-- Messages: ciphertext blobs and routing metadata only. A sender_key envelope
-- (shared across every recipient device) stores its ciphertext directly here;
-- a prekey/whisper envelope (Double Ratchet, per-device ciphertext) leaves
-- this NULL and stores one row per target device in chat_message_deliveries
-- instead, so a direct message's per-recipient ciphertext doesn't duplicate
-- this row's metadata.
CREATE TABLE IF NOT EXISTS chat_messages (
    id                 VARCHAR(36) NOT NULL PRIMARY KEY,
    client_message_id  VARCHAR(64) NOT NULL,
    project_id         VARCHAR(36) NOT NULL,
    conversation_id    VARCHAR(36) NOT NULL,
    sender_user_id     VARCHAR(36) NOT NULL,
    sender_device_id   VARCHAR(36) NOT NULL,
    envelope_type      VARCHAR(24) NOT NULL,
    ciphertext         TEXT,
    seq                BIGINT      NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_cm_dedupe UNIQUE (conversation_id, sender_device_id, client_message_id),
    CONSTRAINT fk_cm_conversation FOREIGN KEY (conversation_id) REFERENCES chat_conversations(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cm_conversation ON chat_messages (conversation_id, seq);
CREATE INDEX IF NOT EXISTS idx_cm_project ON chat_messages (project_id);

-- Per-recipient-device ciphertext for prekey/whisper (Double Ratchet) sends,
-- and per-recipient-device delivery/read receipts for every envelope type.
CREATE TABLE IF NOT EXISTS chat_message_deliveries (
    id                  VARCHAR(36) NOT NULL PRIMARY KEY,
    message_id          VARCHAR(36) NOT NULL,
    project_id          VARCHAR(36) NOT NULL,
    recipient_device_id VARCHAR(36) NOT NULL,
    ciphertext          TEXT        NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_cmd_message_device UNIQUE (message_id, recipient_device_id),
    CONSTRAINT fk_cmd_message FOREIGN KEY (message_id) REFERENCES chat_messages(id) ON DELETE CASCADE,
    CONSTRAINT fk_cmd_device FOREIGN KEY (recipient_device_id) REFERENCES chat_devices(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cmd_device ON chat_message_deliveries (recipient_device_id);

CREATE TABLE IF NOT EXISTS chat_message_receipts (
    id                  VARCHAR(36) NOT NULL PRIMARY KEY,
    message_id          VARCHAR(36) NOT NULL,
    project_id          VARCHAR(36) NOT NULL,
    recipient_device_id VARCHAR(36) NOT NULL,
    status              VARCHAR(16) NOT NULL DEFAULT 'sent',
    delivered_at        TIMESTAMPTZ,
    read_at             TIMESTAMPTZ,
    CONSTRAINT uq_cmr_message_device UNIQUE (message_id, recipient_device_id),
    CONSTRAINT fk_cmr_message FOREIGN KEY (message_id) REFERENCES chat_messages(id) ON DELETE CASCADE,
    CONSTRAINT fk_cmr_device FOREIGN KEY (recipient_device_id) REFERENCES chat_devices(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cmr_device ON chat_message_receipts (recipient_device_id, status);

-- Passphrase-wrapped identity backup (schema only — endpoints land with the
-- device-linking/backup milestone). One backup per user; salt/kdf params are
-- not secret, ciphertext is opaque, and the server never holds the passphrase
-- or the derived key.
CREATE TABLE IF NOT EXISTS chat_key_backups (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    project_id VARCHAR(36)  NOT NULL,
    user_id    VARCHAR(36)  NOT NULL,
    salt       TEXT         NOT NULL,
    kdf_algo   VARCHAR(32)  NOT NULL DEFAULT 'argon2id',
    kdf_params JSONB        NOT NULL DEFAULT '{}',
    ciphertext TEXT         NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_ckb_project_user UNIQUE (project_id, user_id),
    CONSTRAINT fk_ckb_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE TRIGGER set_updated_at BEFORE UPDATE ON chat_key_backups
    FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at();
