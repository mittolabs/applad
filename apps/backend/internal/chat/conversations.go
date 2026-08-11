package chat

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mittolabs/applad/internal/model"
	"github.com/mittolabs/applad/internal/uid"
)

// CreateConversation starts a direct (1:1) conversation between the caller
// and exactly one other user, reusing an existing one between the same pair
// if it already exists — so tapping "message this person" twice never
// spawns a duplicate thread. Group conversations (Sender Keys) are a later
// milestone; otherUserID is required and must name exactly one person.
func (s *Service) CreateConversation(ctx context.Context, projectID, userID, otherUserID, title string) (*model.Conversation, error) {
	if otherUserID == "" {
		return nil, fmt.Errorf("chat: otherUserId is required")
	}
	if otherUserID == userID {
		return nil, fmt.Errorf("chat: cannot start a conversation with yourself")
	}

	if existingID, err := s.findDirectConversation(ctx, projectID, userID, otherUserID); err != nil {
		return nil, err
	} else if existingID != "" {
		return s.getConversationRow(ctx, projectID, existingID)
	}

	id := uid.New("unique()")
	now := time.Now().UTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("chat: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO chat_conversations (id, project_id, type, title, created_by, created_at, updated_at)
		 VALUES ($1, $2, 'direct', $3, $4, $5, $5)`,
		id, projectID, title, userID, now,
	); err != nil {
		return nil, fmt.Errorf("chat: create conversation: %w", err)
	}

	for _, member := range []string{userID, otherUserID} {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO chat_conversation_members (id, conversation_id, project_id, user_id, role, joined_at)
			 VALUES ($1, $2, $3, $4, 'member', $5)`,
			uid.New("unique()"), id, projectID, member, now,
		); err != nil {
			return nil, fmt.Errorf("chat: add member: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("chat: commit: %w", err)
	}

	return &model.Conversation{ID: id, CreatedAt: now, UpdatedAt: now, Type: "direct", Title: title, CreatedBy: userID}, nil
}

// findDirectConversation returns the id of an existing direct conversation
// whose exactly-two members are userID and otherUserID, or "" if none exists.
func (s *Service) findDirectConversation(ctx context.Context, projectID, userID, otherUserID string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx,
		`SELECT c.id FROM chat_conversations c
		 WHERE c.project_id = $1 AND c.type = 'direct'
		   AND EXISTS (SELECT 1 FROM chat_conversation_members m WHERE m.conversation_id = c.id AND m.user_id = $2 AND m.removed_at IS NULL)
		   AND EXISTS (SELECT 1 FROM chat_conversation_members m WHERE m.conversation_id = c.id AND m.user_id = $3 AND m.removed_at IS NULL)
		   AND (SELECT COUNT(*) FROM chat_conversation_members m WHERE m.conversation_id = c.id AND m.removed_at IS NULL) = 2
		 LIMIT 1`,
		projectID, userID, otherUserID,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("chat: find direct conversation: %w", err)
	}
	return id, nil
}

func (s *Service) getConversationRow(ctx context.Context, projectID, conversationID string) (*model.Conversation, error) {
	var c model.Conversation
	err := s.db.QueryRowContext(ctx,
		`SELECT id, created_at, updated_at, type, title, created_by
		 FROM chat_conversations WHERE id = $1 AND project_id = $2`,
		conversationID, projectID,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt, &c.Type, &c.Title, &c.CreatedBy)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("chat: get conversation: %w", err)
	}
	return &c, nil
}

// ListConversations returns the caller's conversations, most recently active
// first — metadata only (title, type, timestamps), never message content.
func (s *Service) ListConversations(ctx context.Context, projectID, userID string) ([]*model.Conversation, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT c.id, c.created_at, c.updated_at, c.type, c.title, c.created_by
		 FROM chat_conversations c
		 JOIN chat_conversation_members m ON m.conversation_id = c.id
		 WHERE c.project_id = $1 AND m.user_id = $2 AND m.removed_at IS NULL
		 ORDER BY c.updated_at DESC`,
		projectID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("chat: list conversations: %w", err)
	}
	defer rows.Close()

	convs := make([]*model.Conversation, 0)
	for rows.Next() {
		var c model.Conversation
		if err := rows.Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt, &c.Type, &c.Title, &c.CreatedBy); err != nil {
			return nil, fmt.Errorf("chat: scan conversation: %w", err)
		}
		convs = append(convs, &c)
	}
	return convs, rows.Err()
}

// GetConversation returns one conversation and its members, if the caller
// belongs to it.
func (s *Service) GetConversation(ctx context.Context, projectID, userID, conversationID string) (*model.Conversation, []*model.ConversationMember, error) {
	isMember, err := s.IsConversationMember(ctx, projectID, conversationID, userID)
	if err != nil {
		return nil, nil, err
	}
	if !isMember {
		return nil, nil, ErrNotFound
	}

	conv, err := s.getConversationRow(ctx, projectID, conversationID)
	if err != nil {
		return nil, nil, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, conversation_id, user_id, role, joined_at, removed_at
		 FROM chat_conversation_members
		 WHERE conversation_id = $1 AND project_id = $2 AND removed_at IS NULL
		 ORDER BY joined_at ASC`,
		conversationID, projectID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("chat: list members: %w", err)
	}
	defer rows.Close()

	members := make([]*model.ConversationMember, 0)
	for rows.Next() {
		var m model.ConversationMember
		var removedAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.UserID, &m.Role, &m.JoinedAt, &removedAt); err != nil {
			return nil, nil, fmt.Errorf("chat: scan member: %w", err)
		}
		if removedAt.Valid {
			t := removedAt.Time
			m.RemovedAt = &t
		}
		members = append(members, &m)
	}
	return conv, members, rows.Err()
}
