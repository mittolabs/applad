package chat

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/model"
	"github.com/mittolabs/applad/internal/realtime"
	"github.com/mittolabs/applad/internal/uid"
)

// SendMessageInput carries one message send. EnvelopeType is "prekey" (first
// message of a new X3DH session) or "whisper" (an established Double Ratchet
// session) — "sender_key"/"sender_key_distribution" (group fan-out) arrive
// with group conversation support. Targets carries one ciphertext per
// recipient device: the client already resolved every conversation member's
// device list (via ListUserDevices) and encrypted once per device, including
// the sender's OWN other devices so multi-device sync sees sent messages.
type SendMessageInput struct {
	ClientMessageID string
	SenderDeviceID  string
	EnvelopeType    string
	Targets         []model.MessageTarget
}

// SendMessage relays an already-encrypted message to every target device: it
// persists the ciphertext, assigns the conversation's next sequence number,
// publishes a realtime event to live subscribers, and best-effort wakes
// offline devices with a content-free push notification. The server never
// sees plaintext — every argument here is either an opaque ciphertext blob
// or routing metadata.
func (s *Service) SendMessage(ctx context.Context, projectID, userID, conversationID string, in SendMessageInput) (*model.Message, error) {
	if in.ClientMessageID == "" {
		return nil, fmt.Errorf("chat: clientMessageId is required")
	}
	if in.EnvelopeType != "prekey" && in.EnvelopeType != "whisper" {
		return nil, fmt.Errorf("chat: envelopeType must be \"prekey\" or \"whisper\" (group envelopes arrive with group conversation support)")
	}
	if len(in.Targets) == 0 {
		return nil, fmt.Errorf("chat: at least one target device is required")
	}

	isMember, err := s.IsConversationMember(ctx, projectID, conversationID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrForbidden
	}
	senderOwned, err := s.deviceOwnedByUser(ctx, projectID, in.SenderDeviceID, userID)
	if err != nil {
		return nil, err
	}
	if !senderOwned {
		return nil, ErrForbidden
	}

	// Idempotency: a retried send (client timed out waiting for the response
	// but the server had already committed) returns the original message
	// rather than erroring on the dedupe constraint or double-delivering.
	if existing, err := s.getMessageByDedupeKey(ctx, conversationID, in.SenderDeviceID, in.ClientMessageID); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}

	targetIDs := make([]string, len(in.Targets))
	seenTarget := make(map[string]bool, len(in.Targets))
	for i, t := range in.Targets {
		if t.DeviceID == "" || t.Ciphertext == "" {
			return nil, fmt.Errorf("chat: every target requires a deviceId and ciphertext")
		}
		if seenTarget[t.DeviceID] {
			return nil, fmt.Errorf("chat: duplicate target device %q", t.DeviceID)
		}
		seenTarget[t.DeviceID] = true
		targetIDs[i] = t.DeviceID
	}
	valid, err := s.activeMemberDeviceSet(ctx, projectID, conversationID, targetIDs)
	if err != nil {
		return nil, err
	}
	for _, id := range targetIDs {
		if !valid[id] {
			return nil, fmt.Errorf("chat: target device %q is not an active device of a conversation member", id)
		}
	}

	id := uid.New("unique()")
	now := time.Now().UTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("chat: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Assign the conversation's next sequence number atomically, so two
	// concurrent sends into the same conversation never collide.
	var seq int64
	err = tx.QueryRowContext(ctx,
		`UPDATE chat_conversations SET next_seq = next_seq + 1, updated_at = $1 WHERE id = $2 AND project_id = $3 RETURNING next_seq`,
		now, conversationID, projectID,
	).Scan(&seq)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("chat: assign sequence: %w", err)
	}

	// ciphertext is NULL on the message row itself for a prekey/whisper send —
	// each recipient device's own copy lives in chat_message_deliveries below,
	// since Double Ratchet output differs per device.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO chat_messages (id, client_message_id, project_id, conversation_id, sender_user_id, sender_device_id, envelope_type, ciphertext, seq, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, $8, $9)`,
		id, in.ClientMessageID, projectID, conversationID, userID, in.SenderDeviceID, in.EnvelopeType, seq, now,
	); err != nil {
		return nil, fmt.Errorf("chat: insert message: %w", err)
	}

	for _, t := range in.Targets {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO chat_message_deliveries (id, message_id, project_id, recipient_device_id, ciphertext, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			uid.New("unique()"), id, projectID, t.DeviceID, t.Ciphertext, now,
		); err != nil {
			return nil, fmt.Errorf("chat: insert delivery: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO chat_message_receipts (id, message_id, project_id, recipient_device_id, status)
			 VALUES ($1, $2, $3, $4, 'sent')`,
			uid.New("unique()"), id, projectID, t.DeviceID,
		); err != nil {
			return nil, fmt.Errorf("chat: insert receipt: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("chat: commit: %w", err)
	}

	msg := &model.Message{
		ID: id, CreatedAt: now, ClientMessageID: in.ClientMessageID, ConversationID: conversationID,
		SenderUserID: userID, SenderDeviceID: in.SenderDeviceID, EnvelopeType: in.EnvelopeType, Seq: seq,
	}

	s.publishMessage(msg, in.Targets)
	s.wakeOfflineDevices(ctx, projectID, conversationID, id, in.SenderDeviceID, targetIDs)

	return msg, nil
}

// getMessageByDedupeKey returns the message already recorded for
// (conversationID, senderDeviceID, clientMessageID), if any.
func (s *Service) getMessageByDedupeKey(ctx context.Context, conversationID, senderDeviceID, clientMessageID string) (*model.Message, error) {
	var m model.Message
	err := s.db.QueryRowContext(ctx,
		`SELECT id, created_at, client_message_id, conversation_id, sender_user_id, sender_device_id, envelope_type, seq
		 FROM chat_messages WHERE conversation_id = $1 AND sender_device_id = $2 AND client_message_id = $3`,
		conversationID, senderDeviceID, clientMessageID,
	).Scan(&m.ID, &m.CreatedAt, &m.ClientMessageID, &m.ConversationID, &m.SenderUserID, &m.SenderDeviceID, &m.EnvelopeType, &m.Seq)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("chat: check dedupe: %w", err)
	}
	return &m, nil
}

// activeMemberDeviceSet returns the subset of deviceIDs that are active
// devices belonging to some (non-removed) member of conversationID.
func (s *Service) activeMemberDeviceSet(ctx context.Context, projectID, conversationID string, deviceIDs []string) (map[string]bool, error) {
	result := make(map[string]bool, len(deviceIDs))
	if len(deviceIDs) == 0 {
		return result, nil
	}
	placeholders := make([]string, len(deviceIDs))
	args := make([]interface{}, 0, len(deviceIDs)+2)
	args = append(args, projectID, conversationID)
	for i, id := range deviceIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+3)
		args = append(args, id)
	}
	query := fmt.Sprintf(
		`SELECT d.id FROM chat_devices d
		 JOIN chat_conversation_members m ON m.user_id = d.user_id
		 WHERE d.project_id = $1 AND m.conversation_id = $2 AND m.removed_at IS NULL
		   AND d.status = 'active' AND d.id IN (%s)`,
		strings.Join(placeholders, ","),
	)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("chat: validate target devices: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("chat: scan target device: %w", err)
		}
		result[id] = true
	}
	return result, rows.Err()
}

// messageEvent is the realtime payload for a new message: the message's
// metadata plus its per-device targets, so an online recipient device can
// pick out its own ciphertext without an extra REST round-trip. Every other
// subscriber in the conversation sees the same event (they're all already
// members, so no new information is disclosed beyond "a message was sent
// and to which devices") but can only decrypt the target meant for them.
type messageEvent struct {
	*model.Message
	Targets []model.MessageTarget `json:"targets,omitempty"`
}

func (s *Service) publishMessage(msg *model.Message, targets []model.MessageTarget) {
	if s.events == nil {
		return
	}
	s.events.Publish(realtime.Event{
		Type:      "chat.messages.create",
		Channel:   "chat." + msg.ConversationID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   messageEvent{Message: msg, Targets: targets},
	})
}

// wakeOfflineDevices best-effort pushes a content-free "new message"
// notification to every target device (other than the sender's own) that has
// a registered push token. There is no per-device "is this device currently
// connected over realtime" signal today — the realtime hub tracks WebSocket
// connections by user, not by chat device — so this fires unconditionally
// rather than only for devices believed offline: redundant for an
// already-connected device (harmless; the client dedupes by messageId), but
// a skipped push would be a permanently lost wake-up for a genuinely offline
// one. The notification carries no sender name or content preview, so even
// metadata reaching the push provider is minimized.
func (s *Service) wakeOfflineDevices(ctx context.Context, projectID, conversationID, messageID, senderDeviceID string, targetDeviceIDs []string) {
	if s.msg == nil {
		return
	}
	recipients := make([]string, 0, len(targetDeviceIDs))
	for _, id := range targetDeviceIDs {
		if id != senderDeviceID {
			recipients = append(recipients, id)
		}
	}
	if len(recipients) == 0 {
		return
	}
	tokens, err := s.pushTokensForDevices(ctx, projectID, recipients)
	if err != nil || len(tokens) == 0 {
		return
	}
	data := map[string]string{"conversationId": conversationID, "messageId": messageID}
	_ = s.msg.SendPushMulti(ctx, projectID, tokens, "New message", "", data)
}

func (s *Service) pushTokensForDevices(ctx context.Context, projectID string, deviceIDs []string) ([]string, error) {
	if len(deviceIDs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(deviceIDs))
	args := make([]interface{}, 0, len(deviceIDs)+1)
	args = append(args, projectID)
	for i, id := range deviceIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}
	query := fmt.Sprintf(
		`SELECT push_token FROM chat_devices WHERE project_id = $1 AND push_token IS NOT NULL AND id IN (%s)`,
		strings.Join(placeholders, ","),
	)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("chat: load push tokens: %w", err)
	}
	defer rows.Close()
	var tokens []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("chat: scan push token: %w", err)
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

// ListMessages returns messages after afterSeq (0 for full history),
// oldest-first, each decoded for the REQUESTING device specifically: a
// prekey/whisper message's ciphertext is that device's own row from
// chat_message_deliveries (empty if this device wasn't a target — e.g. it
// was linked after the message was sent). Used both for live catch-up after
// a reconnect and for a newly linked device's initial sync.
func (s *Service) ListMessages(ctx context.Context, projectID, userID, conversationID, deviceID string, afterSeq int64, limit int) ([]*model.Message, error) {
	isMember, err := s.IsConversationMember(ctx, projectID, conversationID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrForbidden
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT m.id, m.created_at, m.client_message_id, m.conversation_id, m.sender_user_id,
		        m.sender_device_id, m.envelope_type, m.seq, COALESCE(m.ciphertext, d.ciphertext, '')
		 FROM chat_messages m
		 LEFT JOIN chat_message_deliveries d ON d.message_id = m.id AND d.recipient_device_id = $1
		 WHERE m.conversation_id = $2 AND m.project_id = $3 AND m.seq > $4
		 ORDER BY m.seq ASC
		 LIMIT $5`,
		deviceID, conversationID, projectID, afterSeq, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("chat: list messages: %w", err)
	}
	defer rows.Close()

	messages := make([]*model.Message, 0)
	for rows.Next() {
		var m model.Message
		if err := rows.Scan(&m.ID, &m.CreatedAt, &m.ClientMessageID, &m.ConversationID, &m.SenderUserID,
			&m.SenderDeviceID, &m.EnvelopeType, &m.Seq, &m.Ciphertext); err != nil {
			return nil, fmt.Errorf("chat: scan message: %w", err)
		}
		messages = append(messages, &m)
	}
	return messages, rows.Err()
}

// AckMessage records a recipient device's delivery/read acknowledgement.
// deviceID must be one of the caller's own devices — a receipt records what
// THIS device has seen, not what its user has seen across all devices.
func (s *Service) AckMessage(ctx context.Context, projectID, userID, messageID, deviceID, status string) error {
	if status != "delivered" && status != "read" {
		return fmt.Errorf("chat: status must be \"delivered\" or \"read\"")
	}
	owned, err := s.deviceOwnedByUser(ctx, projectID, deviceID, userID)
	if err != nil {
		return err
	}
	if !owned {
		return ErrForbidden
	}

	var res sql.Result
	if status == "delivered" {
		res, err = s.db.ExecContext(ctx,
			`UPDATE chat_message_receipts SET status = 'delivered', delivered_at = NOW()
			 WHERE message_id = $1 AND project_id = $2 AND recipient_device_id = $3 AND status = 'sent'`,
			messageID, projectID, deviceID,
		)
	} else {
		res, err = s.db.ExecContext(ctx,
			`UPDATE chat_message_receipts SET status = 'read', read_at = NOW(),
			        delivered_at = COALESCE(delivered_at, NOW())
			 WHERE message_id = $1 AND project_id = $2 AND recipient_device_id = $3 AND status IN ('sent', 'delivered')`,
			messageID, projectID, deviceID,
		)
	}
	if err != nil {
		return fmt.Errorf("chat: ack message: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
