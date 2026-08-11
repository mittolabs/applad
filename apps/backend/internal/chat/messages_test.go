package chat

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/model"
)

func TestSendMessage_RejectsUnknownEnvelopeType(t *testing.T) {
	svc, _ := newMockService(t)
	_, err := svc.SendMessage(context.Background(), "proj1", "user1", "conv1", SendMessageInput{
		ClientMessageID: "c1", SenderDeviceID: "dev1", EnvelopeType: "sender_key",
		Targets: []model.MessageTarget{{DeviceID: "dev2", Ciphertext: "ct"}},
	})
	if err == nil {
		t.Fatal("expected error for a group-only envelope type in v1")
	}
}

func TestSendMessage_RejectsEmptyTargets(t *testing.T) {
	svc, _ := newMockService(t)
	_, err := svc.SendMessage(context.Background(), "proj1", "user1", "conv1", SendMessageInput{
		ClientMessageID: "c1", SenderDeviceID: "dev1", EnvelopeType: "whisper",
	})
	if err == nil {
		t.Fatal("expected error when no target devices are given")
	}
}

func TestSendMessage_ForbiddenWhenNotAMember(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT EXISTS\\(\\s*SELECT 1 FROM chat_conversation_members").
		WithArgs("conv1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	_, err := svc.SendMessage(context.Background(), "proj1", "user1", "conv1", SendMessageInput{
		ClientMessageID: "c1", SenderDeviceID: "dev1", EnvelopeType: "whisper",
		Targets: []model.MessageTarget{{DeviceID: "dev2", Ciphertext: "ct"}},
	})
	if err != ErrForbidden {
		t.Fatalf("got %v, want ErrForbidden", err)
	}
}

func TestSendMessage_ForbiddenWhenDeviceNotOwned(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT EXISTS\\(\\s*SELECT 1 FROM chat_conversation_members").
		WithArgs("conv1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT EXISTS\\(\\s*SELECT 1 FROM chat_devices").
		WithArgs("dev1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	_, err := svc.SendMessage(context.Background(), "proj1", "user1", "conv1", SendMessageInput{
		ClientMessageID: "c1", SenderDeviceID: "dev1", EnvelopeType: "whisper",
		Targets: []model.MessageTarget{{DeviceID: "dev2", Ciphertext: "ct"}},
	})
	if err != ErrForbidden {
		t.Fatalf("got %v, want ErrForbidden", err)
	}
}

func TestSendMessage_IdempotentOnRetry(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT EXISTS\\(\\s*SELECT 1 FROM chat_conversation_members").
		WithArgs("conv1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT EXISTS\\(\\s*SELECT 1 FROM chat_devices").
		WithArgs("dev1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT id, created_at, client_message_id").
		WithArgs("conv1", "dev1", "c1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "client_message_id", "conversation_id", "sender_user_id", "sender_device_id", "envelope_type", "seq",
		}).AddRow("msg-existing", fixedTime(), "c1", "conv1", "user1", "dev1", "whisper", int64(5)))

	msg, err := svc.SendMessage(context.Background(), "proj1", "user1", "conv1", SendMessageInput{
		ClientMessageID: "c1", SenderDeviceID: "dev1", EnvelopeType: "whisper",
		Targets: []model.MessageTarget{{DeviceID: "dev2", Ciphertext: "ct"}},
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if msg.ID != "msg-existing" || msg.Seq != 5 {
		t.Fatalf("expected the original message to be returned on retry, got %+v", msg)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSendMessage_RejectsTargetDeviceOutsideConversation(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT EXISTS\\(\\s*SELECT 1 FROM chat_conversation_members").
		WithArgs("conv1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT EXISTS\\(\\s*SELECT 1 FROM chat_devices").
		WithArgs("dev1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT id, created_at, client_message_id").
		WithArgs("conv1", "dev1", "c1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "client_message_id", "conversation_id", "sender_user_id", "sender_device_id", "envelope_type", "seq",
		})) // no existing message
	mock.ExpectQuery("SELECT d.id FROM chat_devices d").
		WithArgs("proj1", "conv1", "dev-outsider").
		WillReturnRows(sqlmock.NewRows([]string{"id"})) // dev-outsider is not a member device

	_, err := svc.SendMessage(context.Background(), "proj1", "user1", "conv1", SendMessageInput{
		ClientMessageID: "c1", SenderDeviceID: "dev1", EnvelopeType: "whisper",
		Targets: []model.MessageTarget{{DeviceID: "dev-outsider", Ciphertext: "ct"}},
	})
	if err == nil {
		t.Fatal("expected error when a target device does not belong to a conversation member")
	}
}

func TestSendMessage_HappyPath(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT EXISTS\\(\\s*SELECT 1 FROM chat_conversation_members").
		WithArgs("conv1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT EXISTS\\(\\s*SELECT 1 FROM chat_devices").
		WithArgs("dev1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT id, created_at, client_message_id").
		WithArgs("conv1", "dev1", "c1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "client_message_id", "conversation_id", "sender_user_id", "sender_device_id", "envelope_type", "seq",
		}))
	mock.ExpectQuery("SELECT d.id FROM chat_devices d").
		WithArgs("proj1", "conv1", "dev2").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("dev2"))
	mock.ExpectBegin()
	mock.ExpectQuery("UPDATE chat_conversations SET next_seq").
		WithArgs(sqlmock.AnyArg(), "conv1", "proj1").
		WillReturnRows(sqlmock.NewRows([]string{"next_seq"}).AddRow(int64(1)))
	mock.ExpectExec("INSERT INTO chat_messages").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO chat_message_deliveries").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO chat_message_receipts").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	msg, err := svc.SendMessage(context.Background(), "proj1", "user1", "conv1", SendMessageInput{
		ClientMessageID: "c1", SenderDeviceID: "dev1", EnvelopeType: "whisper",
		Targets: []model.MessageTarget{{DeviceID: "dev2", Ciphertext: "ct-for-dev2"}},
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if msg.Seq != 1 || msg.ConversationID != "conv1" || msg.Ciphertext != "" {
		t.Fatalf("unexpected message: %+v", msg)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestListMessages_ForbiddenWhenNotAMember(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("conv1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	_, err := svc.ListMessages(context.Background(), "proj1", "user1", "conv1", "dev1", 0, 50)
	if err != ErrForbidden {
		t.Fatalf("got %v, want ErrForbidden", err)
	}
}

func TestListMessages_ReturnsRequestingDevicesOwnCiphertext(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("conv1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT m.id, m.created_at").
		WithArgs("dev2", "conv1", "proj1", int64(0), 100).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "client_message_id", "conversation_id", "sender_user_id", "sender_device_id", "envelope_type", "seq", "ciphertext",
		}).AddRow("msg1", fixedTime(), "c1", "conv1", "user1", "dev1", "whisper", int64(1), "ct-for-dev2"))

	messages, err := svc.ListMessages(context.Background(), "proj1", "user1", "conv1", "dev2", 0, 0)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(messages) != 1 || messages[0].Ciphertext != "ct-for-dev2" {
		t.Fatalf("unexpected messages: %+v", messages)
	}
}

func TestAckMessage_RequiresValidStatus(t *testing.T) {
	svc, _ := newMockService(t)
	err := svc.AckMessage(context.Background(), "proj1", "user1", "msg1", "dev1", "seen")
	if err == nil {
		t.Fatal("expected error for an invalid ack status")
	}
}

func TestAckMessage_ForbiddenWhenDeviceNotOwned(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("dev1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	err := svc.AckMessage(context.Background(), "proj1", "user1", "msg1", "dev1", "delivered")
	if err != ErrForbidden {
		t.Fatalf("got %v, want ErrForbidden", err)
	}
}

func TestAckMessage_NotFoundWhenNoMatchingReceipt(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("dev1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec("UPDATE chat_message_receipts SET status = 'delivered'").
		WithArgs("msg1", "proj1", "dev1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := svc.AckMessage(context.Background(), "proj1", "user1", "msg1", "dev1", "delivered")
	if err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestAckMessage_ReadTransition(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("dev1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec("UPDATE chat_message_receipts SET status = 'read'").
		WithArgs("msg1", "proj1", "dev1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.AckMessage(context.Background(), "proj1", "user1", "msg1", "dev1", "read"); err != nil {
		t.Fatalf("AckMessage: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
