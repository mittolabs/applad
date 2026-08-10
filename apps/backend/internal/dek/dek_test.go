package dek

import (
	"context"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	appladcrypto "github.com/mittolabs/applad/internal/crypto"
	"github.com/mittolabs/applad/internal/db"
)

func newMockDB(t *testing.T) (*db.DB, sqlmock.Sqlmock) {
	t.Helper()
	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	return &db.DB{DB: raw}, mock
}

const testMasterKeyHex = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"

func TestNewService_Disabled(t *testing.T) {
	database, _ := newMockDB(t)
	svc, err := NewService(database, "")
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if svc.Enabled() {
		t.Fatalf("expected disabled service with empty master key")
	}
	if _, _, err := svc.Unwrap(context.Background(), "proj1"); err != ErrDisabled {
		t.Fatalf("got %v, want ErrDisabled", err)
	}
	if err := svc.EnsureProjectKey(context.Background(), "proj1"); err != ErrDisabled {
		t.Fatalf("got %v, want ErrDisabled", err)
	}
}

func TestNewService_RejectsShortKey(t *testing.T) {
	database, _ := newMockDB(t)
	if _, err := NewService(database, "too-short"); err == nil {
		t.Fatalf("expected error for a too-short master key")
	}
}

func TestParseMasterKey_PrefersHexDecode(t *testing.T) {
	key, err := ParseMasterKey(testMasterKeyHex)
	if err != nil {
		t.Fatalf("ParseMasterKey: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("got key length %d, want 32", len(key))
	}
	if key[0] != 0x01 || key[1] != 0x02 {
		t.Fatalf("expected hex-decoded bytes, got %x", key[:2])
	}
}

func TestParseMasterKey_RawFallback(t *testing.T) {
	raw := strings.Repeat("z", 40) // not valid hex, long enough to use raw
	key, err := ParseMasterKey(raw)
	if err != nil {
		t.Fatalf("ParseMasterKey: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("got key length %d, want 32", len(key))
	}
}

func TestEnsureProjectKey_SkipsWhenActiveExists(t *testing.T) {
	database, mock := newMockDB(t)
	svc, err := NewService(database, testMasterKeyHex)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("proj1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	if err := svc.EnsureProjectKey(context.Background(), "proj1"); err != nil {
		t.Fatalf("EnsureProjectKey: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestEnsureProjectKey_CreatesWhenMissing(t *testing.T) {
	database, mock := newMockDB(t)
	svc, err := NewService(database, testMasterKeyHex)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("proj1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec("INSERT INTO project_encryption_keys").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := svc.EnsureProjectKey(context.Background(), "proj1"); err != nil {
		t.Fatalf("EnsureProjectKey: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUnwrap_RoundTripAndCache(t *testing.T) {
	database, mock := newMockDB(t)
	svc, err := NewService(database, testMasterKeyHex)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	rawDEK := make([]byte, 32)
	for i := range rawDEK {
		rawDEK[i] = byte(i + 1)
	}
	wrapped, err := appladcrypto.SealToken("dek", 1, svc.masterKey, rawDEK)
	if err != nil {
		t.Fatalf("SealToken: %v", err)
	}

	mock.ExpectQuery("SELECT key_version, wrapped_dek").
		WithArgs("proj1").
		WillReturnRows(sqlmock.NewRows([]string{"key_version", "wrapped_dek"}).AddRow(1, wrapped))

	key, version, err := svc.Unwrap(context.Background(), "proj1")
	if err != nil {
		t.Fatalf("Unwrap: %v", err)
	}
	if version != 1 {
		t.Fatalf("got version %d, want 1", version)
	}
	if string(key) != string(rawDEK) {
		t.Fatalf("unwrapped key does not match original DEK")
	}

	// Second call should hit the in-process cache, not issue another query.
	key2, version2, err := svc.Unwrap(context.Background(), "proj1")
	if err != nil {
		t.Fatalf("Unwrap (cached): %v", err)
	}
	if version2 != 1 || string(key2) != string(rawDEK) {
		t.Fatalf("cached unwrap mismatch")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations (cache should have avoided a second query): %v", err)
	}
}

func TestUnwrap_NoActiveKey(t *testing.T) {
	database, mock := newMockDB(t)
	svc, err := NewService(database, testMasterKeyHex)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	mock.ExpectQuery("SELECT key_version, wrapped_dek").
		WithArgs("proj1").
		WillReturnRows(sqlmock.NewRows([]string{"key_version", "wrapped_dek"}))

	if _, _, err := svc.Unwrap(context.Background(), "proj1"); err == nil {
		t.Fatalf("expected error when no active key exists")
	}
}

func TestRotateProjectKey_RetiresAndCreatesNew(t *testing.T) {
	database, mock := newMockDB(t)
	svc, err := NewService(database, testMasterKeyHex)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	mock.ExpectQuery("SELECT key_version FROM project_encryption_keys").
		WithArgs("proj1").
		WillReturnRows(sqlmock.NewRows([]string{"key_version"}).AddRow(1))
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE project_encryption_keys SET status = 'retired'").
		WithArgs("proj1", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO project_encryption_keys").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	newVersion, err := svc.RotateProjectKey(context.Background(), "proj1")
	if err != nil {
		t.Fatalf("RotateProjectKey: %v", err)
	}
	if newVersion != 2 {
		t.Fatalf("got new version %d, want 2", newVersion)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRewrapAll_ReencryptsUnderNewMasterKey(t *testing.T) {
	database, mock := newMockDB(t)

	oldMasterKey, err := ParseMasterKey(testMasterKeyHex)
	if err != nil {
		t.Fatalf("ParseMasterKey: %v", err)
	}
	newMasterKeyHex := "1112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f"
	svc, err := NewService(database, newMasterKeyHex)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	rawDEK := []byte("0123456789abcdef0123456789abcdef")[:32]
	wrappedOld, err := appladcrypto.SealToken("dek", 1, oldMasterKey, rawDEK)
	if err != nil {
		t.Fatalf("SealToken: %v", err)
	}

	mock.ExpectQuery("SELECT id, wrapped_dek FROM project_encryption_keys").
		WillReturnRows(sqlmock.NewRows([]string{"id", "wrapped_dek"}).AddRow("pek1", wrappedOld))
	mock.ExpectExec("UPDATE project_encryption_keys SET wrapped_dek").
		WillReturnResult(sqlmock.NewResult(0, 1))

	count, err := svc.RewrapAll(context.Background(), oldMasterKey)
	if err != nil {
		t.Fatalf("RewrapAll: %v", err)
	}
	if count != 1 {
		t.Fatalf("got count %d, want 1", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUnwrapVersion_UnknownKEKVersionFails(t *testing.T) {
	database, mock := newMockDB(t)
	svc, err := NewService(database, testMasterKeyHex)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	otherKey, err := ParseMasterKey("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatalf("ParseMasterKey: %v", err)
	}
	rawDEK := make([]byte, 32)
	// Wrapped under kek_version 2, but this service only knows kek_version 1
	// (its own masterKey) — simulates trying to unwrap a DEK that was wrapped
	// under a master key this instance was never rotated onto.
	wrapped, err := appladcrypto.SealToken("dek", 2, otherKey, rawDEK)
	if err != nil {
		t.Fatalf("SealToken: %v", err)
	}

	mock.ExpectQuery("SELECT wrapped_dek FROM project_encryption_keys").
		WithArgs("proj1", 2).
		WillReturnRows(sqlmock.NewRows([]string{"wrapped_dek"}).AddRow(wrapped))

	if _, err := svc.UnwrapVersion(context.Background(), "proj1", 2); err == nil {
		t.Fatalf("expected error unwrapping a DEK wrapped under an unknown master key version")
	}
}
