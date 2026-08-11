// Package chat implements Applad's end-to-end encrypted messaging primitive:
// per-device identity/prekey bundles, direct (1:1) conversations, and
// ciphertext message relay. The server never holds plaintext content or any
// private key — every table it touches stores public keys, opaque
// ciphertext, and routing metadata only (see
// internal/db/migrations/042_chat.sql).
//
// v1 scope (this package, for now): device registration and prekey bundles,
// direct conversations, and Double Ratchet (prekey/whisper) message
// send/fetch/ack. Group conversations (Sender Keys), multi-device linking,
// and passphrase-wrapped identity backup land in later milestones on top of
// the same schema.
package chat

import (
	"context"
	"errors"
	"fmt"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/messaging"
	"github.com/mittolabs/applad/internal/realtime"
)

// Service handles chat device, conversation, and message operations.
type Service struct {
	db     *db.DB
	events realtime.EventPublisher
	// msg wakes offline devices with a content-free push notification when a
	// message arrives. Nil disables push wake-up (realtime delivery to
	// connected devices is unaffected).
	msg *messaging.Service
}

// NewService creates a new chat Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// SetEventPublisher wires realtime event publishing into the service.
func (s *Service) SetEventPublisher(pub realtime.EventPublisher) {
	s.events = pub
}

// SetMessagingService wires push notifications (waking offline devices) into
// the service.
func (s *Service) SetMessagingService(m *messaging.Service) {
	s.msg = m
}

// IsConversationMember reports whether userID is a (non-removed) member of
// conversationID within projectID. Implements
// realtime.ConversationMembershipVerifier, so a realtime subscription to
// chat.{conversationId} is authorized by the same membership check the REST
// endpoints use.
func (s *Service) IsConversationMember(ctx context.Context, projectID, conversationID, userID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM chat_conversation_members
			WHERE conversation_id = $1 AND project_id = $2 AND user_id = $3 AND removed_at IS NULL
		)`,
		conversationID, projectID, userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("chat: check membership: %w", err)
	}
	return exists, nil
}

// deviceOwnedByUser reports whether deviceID is an active device belonging to
// userID within projectID — used to stop a caller from acting through a
// device id they don't own (as sender, or when acknowledging a receipt).
func (s *Service) deviceOwnedByUser(ctx context.Context, projectID, deviceID, userID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM chat_devices
			WHERE id = $1 AND project_id = $2 AND user_id = $3 AND status = 'active'
		)`,
		deviceID, projectID, userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("chat: check device ownership: %w", err)
	}
	return exists, nil
}

// ErrNotFound is returned when a requested chat resource does not exist (or
// is outside the caller's project/membership — the two are indistinguishable
// on purpose, matching how the rest of the API avoids confirming a
// resource's existence to a caller who cannot see it).
var ErrNotFound = errors.New("chat: not found")

// ErrForbidden is returned when a caller is authenticated but not permitted
// to perform the requested action (e.g. sending through a device they don't
// own, or acting in a conversation they don't belong to).
var ErrForbidden = errors.New("chat: forbidden")
