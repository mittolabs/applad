package chat

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestCreateConversation_RejectsSelfConversation(t *testing.T) {
	svc, _ := newMockService(t)
	_, err := svc.CreateConversation(context.Background(), "proj1", "user1", "user1", "")
	if err == nil {
		t.Fatal("expected error when starting a conversation with yourself")
	}
}

func TestCreateConversation_ReusesExistingDirectConversation(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT c.id FROM chat_conversations").
		WithArgs("proj1", "user1", "user2").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("conv-existing"))
	mock.ExpectQuery("SELECT id, created_at, updated_at, type, title, created_by").
		WithArgs("conv-existing", "proj1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "type", "title", "created_by"}).
			AddRow("conv-existing", fixedTime(), fixedTime(), "direct", "", "user1"))

	conv, err := svc.CreateConversation(context.Background(), "proj1", "user1", "user2", "")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if conv.ID != "conv-existing" {
		t.Fatalf("expected the existing conversation to be reused, got %+v", conv)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateConversation_CreatesNewWhenNoneExists(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT c.id FROM chat_conversations").
		WithArgs("proj1", "user1", "user2").
		WillReturnRows(sqlmock.NewRows([]string{"id"})) // none found
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO chat_conversations").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO chat_conversation_members").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO chat_conversation_members").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	conv, err := svc.CreateConversation(context.Background(), "proj1", "user1", "user2", "")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if conv.Type != "direct" || conv.CreatedBy != "user1" {
		t.Fatalf("unexpected conversation: %+v", conv)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetConversation_NotFoundWhenNotAMember(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("conv1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	_, _, err := svc.GetConversation(context.Background(), "proj1", "user1", "conv1")
	if err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}
