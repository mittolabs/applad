package chat

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestRegisterDevice_InsertsDeviceAndPrekeys(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO chat_devices").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO chat_one_time_prekeys").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO chat_one_time_prekeys").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	device, err := svc.RegisterDevice(context.Background(), "proj1", "user1", RegisterDeviceInput{
		DeviceID: "dev1", Name: "iPhone", RegistrationID: 42,
		IdentityKey: "idkey", SignedPrekeyID: 1, SignedPrekey: "spk", SignedPrekeySig: "sig",
		OneTimePrekeys: []OneTimePrekeyInput{{KeyID: 1, PublicKey: "opk1"}, {KeyID: 2, PublicKey: "opk2"}},
	})
	if err != nil {
		t.Fatalf("RegisterDevice: %v", err)
	}
	if device.ID != "dev1" || device.UserID != "user1" || device.Status != "active" {
		t.Fatalf("unexpected device: %+v", device)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRegisterDevice_RequiresKeyMaterial(t *testing.T) {
	svc, _ := newMockService(t)
	_, err := svc.RegisterDevice(context.Background(), "proj1", "user1", RegisterDeviceInput{DeviceID: "dev1"})
	if err == nil {
		t.Fatal("expected error when key material is missing")
	}
}

func TestRevokeDevice_NotFoundWhenNoRowsAffected(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectExec("UPDATE chat_devices SET status = 'revoked'").
		WithArgs("dev1", "proj1", "user1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := svc.RevokeDevice(context.Background(), "proj1", "user1", "dev1")
	if err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestGetPrekeyBundle_ConsumesOneOneTimePrekey(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT registration_id, identity_key").
		WithArgs("dev1", "proj1").
		WillReturnRows(sqlmock.NewRows([]string{"registration_id", "identity_key", "signed_prekey_id", "signed_prekey", "signed_prekey_sig"}).
			AddRow(42, "idkey", 1, "spk", "sig"))
	mock.ExpectQuery("DELETE FROM chat_one_time_prekeys").
		WithArgs("dev1").
		WillReturnRows(sqlmock.NewRows([]string{"key_id", "public_key"}).AddRow(7, "opk-public"))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM chat_one_time_prekeys").
		WithArgs("dev1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))

	bundle, err := svc.GetPrekeyBundle(context.Background(), "proj1", "dev1")
	if err != nil {
		t.Fatalf("GetPrekeyBundle: %v", err)
	}
	if bundle.OneTimePrekeyID == nil || *bundle.OneTimePrekeyID != 7 || bundle.OneTimePrekey != "opk-public" {
		t.Fatalf("expected a consumed one-time prekey, got %+v", bundle)
	}
	if bundle.PrekeysRemaining != 4 {
		t.Fatalf("got remaining %d, want 4", bundle.PrekeysRemaining)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetPrekeyBundle_NoOneTimePrekeysLeft(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT registration_id, identity_key").
		WithArgs("dev1", "proj1").
		WillReturnRows(sqlmock.NewRows([]string{"registration_id", "identity_key", "signed_prekey_id", "signed_prekey", "signed_prekey_sig"}).
			AddRow(42, "idkey", 1, "spk", "sig"))
	mock.ExpectQuery("DELETE FROM chat_one_time_prekeys").
		WithArgs("dev1").
		WillReturnRows(sqlmock.NewRows([]string{"key_id", "public_key"})) // empty: pool exhausted
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM chat_one_time_prekeys").
		WithArgs("dev1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	bundle, err := svc.GetPrekeyBundle(context.Background(), "proj1", "dev1")
	if err != nil {
		t.Fatalf("GetPrekeyBundle: %v", err)
	}
	if bundle.OneTimePrekeyID != nil {
		t.Fatalf("expected no one-time prekey, got %+v", bundle)
	}
	// X3DH can still proceed without an OPK (Signal spec treats it as optional).
	if bundle.IdentityKey != "idkey" {
		t.Fatalf("expected identity key regardless of OPK availability, got %+v", bundle)
	}
}

func TestGetPrekeyBundle_DeviceNotFound(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT registration_id, identity_key").
		WithArgs("dev1", "proj1").
		WillReturnRows(sqlmock.NewRows([]string{"registration_id", "identity_key", "signed_prekey_id", "signed_prekey", "signed_prekey_sig"}))

	_, err := svc.GetPrekeyBundle(context.Background(), "proj1", "dev1")
	if err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestTopUpPrekeys_ForbiddenWhenNotOwned(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("dev1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	err := svc.TopUpPrekeys(context.Background(), "proj1", "user1", "dev1", []OneTimePrekeyInput{{KeyID: 1, PublicKey: "x"}})
	if err != ErrForbidden {
		t.Fatalf("got %v, want ErrForbidden", err)
	}
}

func TestListUserDevices_ReturnsMinimalRefsOnly(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT id, registration_id, status FROM chat_devices").
		WithArgs("proj1", "user2").
		WillReturnRows(sqlmock.NewRows([]string{"id", "registration_id", "status"}).
			AddRow("dev1", 1, "active").
			AddRow("dev2", 2, "active"))

	refs, err := svc.ListUserDevices(context.Background(), "proj1", "user2")
	if err != nil {
		t.Fatalf("ListUserDevices: %v", err)
	}
	if len(refs) != 2 || refs[0].ID != "dev1" || refs[1].ID != "dev2" {
		t.Fatalf("unexpected device refs: %+v", refs)
	}
}
