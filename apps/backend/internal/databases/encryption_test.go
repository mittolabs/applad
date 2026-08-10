package databases

import (
	"context"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	appladcrypto "github.com/mittolabs/applad/internal/crypto"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/dek"
)

// ── toSQLType ────────────────────────────────────────────────────────────────

func TestToSQLType_EncryptedForcesText(t *testing.T) {
	cases := []struct {
		columnType string
		options    map[string]interface{}
	}{
		{"integer", nil},
		{"boolean", nil},
		{"datetime", nil},
		{"string", map[string]interface{}{"size": 50}},
	}
	for _, c := range cases {
		got := toSQLType(c.columnType, c.options, true)
		if got != "TEXT" {
			t.Errorf("toSQLType(%q, encrypted=true) = %q, want TEXT", c.columnType, got)
		}
	}
	// Unencrypted behavior is unchanged.
	if got := toSQLType("integer", nil, false); got != "BIGINT" {
		t.Errorf("toSQLType(integer, encrypted=false) = %q, want BIGINT", got)
	}
}

// ── rejectEncryptedFilterOrSort ──────────────────────────────────────────────

func TestRejectEncryptedFilterOrSort(t *testing.T) {
	encCols := map[string]bool{"ssn": true}

	if err := rejectEncryptedFilterOrSort(nil, "", nil); err != nil {
		t.Errorf("no encrypted columns at all: expected nil, got %v", err)
	}
	if err := rejectEncryptedFilterOrSort(
		[]Query{{Field: "name", Method: "equal", Values: "x"}}, "", encCols); err != nil {
		t.Errorf("filtering a non-encrypted field: expected nil, got %v", err)
	}
	if err := rejectEncryptedFilterOrSort(
		[]Query{{Field: "ssn", Method: "equal", Values: "x"}}, "", encCols); err == nil {
		t.Error("expected error filtering an encrypted field with 'equal'")
	}
	if err := rejectEncryptedFilterOrSort(
		[]Query{{Field: "ssn", Method: "contains", Values: "x"}}, "", encCols); err == nil {
		t.Error("expected error filtering an encrypted field with 'contains'")
	}
	// isNull/isNotNull are exempt: NULL-ness survives encryption.
	if err := rejectEncryptedFilterOrSort(
		[]Query{{Field: "ssn", Method: "isNull"}}, "", encCols); err != nil {
		t.Errorf("isNull on an encrypted field should be allowed, got %v", err)
	}
	if err := rejectEncryptedFilterOrSort(nil, "ssn", encCols); err == nil {
		t.Error("expected error ordering by an encrypted field")
	}
	if err := rejectEncryptedFilterOrSort(nil, "name", encCols); err != nil {
		t.Errorf("ordering by a non-encrypted field: expected nil, got %v", err)
	}
}

// ── encryptRowData / decryptFieldValue round trip ───────────────────────────

func testMasterKeyHex() string {
	return strings.Repeat("11", 32) // 64 hex chars -> 32 bytes
}

func newServiceWithDEK(t *testing.T) (*Service, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { mockDB.Close() })
	database := &db.DB{DB: mockDB}
	svc := NewService(database)
	dekSvc, err := dek.NewService(database, testMasterKeyHex())
	if err != nil {
		t.Fatalf("dek.NewService: %v", err)
	}
	svc.SetDEKService(dekSvc)
	return svc, mock
}

// sealedTestDEK returns a wrapped-DEK token as project_encryption_keys would
// store it, plus the raw key it wraps, for mocking dek.Service's queries.
func sealedTestDEK(t *testing.T) (wrapped string, raw []byte) {
	t.Helper()
	masterKey, err := dek.ParseMasterKey(testMasterKeyHex())
	if err != nil {
		t.Fatalf("ParseMasterKey: %v", err)
	}
	raw = make([]byte, 32)
	for i := range raw {
		raw[i] = byte(i + 1)
	}
	wrapped, err = appladcrypto.SealToken("dek", 1, masterKey, raw)
	if err != nil {
		t.Fatalf("SealToken: %v", err)
	}
	return wrapped, raw
}

func TestEncryptRowData_SkipsWhenNoEncryptedColumns(t *testing.T) {
	svc, _ := newServiceWithDEK(t)
	data := map[string]interface{}{"name": "plain"}
	if err := svc.encryptRowData(context.Background(), "proj1", data, nil); err != nil {
		t.Fatalf("encryptRowData: %v", err)
	}
	if data["name"] != "plain" {
		t.Fatalf("expected untouched value, got %v", data["name"])
	}
}

func TestEncryptRowData_LeavesNilValuesAsNull(t *testing.T) {
	svc, _ := newServiceWithDEK(t)
	data := map[string]interface{}{"ssn": nil}
	encCols := map[string]bool{"ssn": true}
	if err := svc.encryptRowData(context.Background(), "proj1", data, encCols); err != nil {
		t.Fatalf("encryptRowData: %v", err)
	}
	if data["ssn"] != nil {
		t.Fatalf("expected nil to stay nil (SQL NULL), got %v", data["ssn"])
	}
}

func TestEncryptRowData_DecryptFieldValue_RoundTrip(t *testing.T) {
	svc, mock := newServiceWithDEK(t)
	wrapped, _ := sealedTestDEK(t)

	mock.ExpectQuery("SELECT key_version, wrapped_dek").
		WithArgs("proj1").
		WillReturnRows(sqlmock.NewRows([]string{"key_version", "wrapped_dek"}).AddRow(1, wrapped))

	data := map[string]interface{}{"ssn": "123-45-6789", "name": "plain"}
	encCols := map[string]bool{"ssn": true}
	if err := svc.encryptRowData(context.Background(), "proj1", data, encCols); err != nil {
		t.Fatalf("encryptRowData: %v", err)
	}
	if data["name"] != "plain" {
		t.Fatalf("non-encrypted field should be untouched, got %v", data["name"])
	}
	ciphertext, ok := data["ssn"].(string)
	if !ok || ciphertext == "123-45-6789" {
		t.Fatalf("expected ssn to become opaque ciphertext, got %v", data["ssn"])
	}
	if !strings.HasPrefix(ciphertext, "fe1:") {
		t.Fatalf("expected a versioned 'fe1:' token, got %q", ciphertext)
	}

	// Decrypting reads the version back out of the token itself, so it asks
	// for key_version 1 specifically (a fresh query — UnwrapVersion's cache is
	// separate from Unwrap's).
	mock.ExpectQuery("SELECT wrapped_dek FROM project_encryption_keys").
		WithArgs("proj1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"wrapped_dek"}).AddRow(wrapped))

	decrypted, err := svc.decryptFieldValue(context.Background(), "proj1", ciphertext)
	if err != nil {
		t.Fatalf("decryptFieldValue: %v", err)
	}
	if decrypted != "123-45-6789" {
		t.Fatalf("got %v, want original plaintext", decrypted)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestEncryptRowData_WithoutDEKServiceFails(t *testing.T) {
	svc := NewService(&db.DB{})
	data := map[string]interface{}{"ssn": "123-45-6789"}
	encCols := map[string]bool{"ssn": true}
	err := svc.encryptRowData(context.Background(), "proj1", data, encCols)
	if err != dek.ErrDisabled {
		t.Fatalf("got %v, want dek.ErrDisabled", err)
	}
}

func TestDecryptFieldValue_NilPassesThrough(t *testing.T) {
	svc, _ := newServiceWithDEK(t)
	got, err := svc.decryptFieldValue(context.Background(), "proj1", nil)
	if err != nil || got != nil {
		t.Fatalf("got (%v, %v), want (nil, nil)", got, err)
	}
}

// ── CreateColumn validation ──────────────────────────────────────────────────

func TestCreateColumn_RejectsArrayAndEncrypted(t *testing.T) {
	svc, mock := newServiceWithDEK(t)
	_, err := svc.CreateColumn(context.Background(), "proj1", "t1", "tags", "string", false, true, true, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("expected 'not supported' error for array+encrypted, got %v", err)
	}
	// Rejected before any DB access — no lookupProjectTable query.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateColumn_EncryptedWithoutDEKServiceFails(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()
	svc := NewService(&db.DB{DB: mockDB})

	expectLookupTable(mock, "t1", "db1", "proj1", "users")

	_, err = svc.CreateColumn(context.Background(), "proj1", "t1", "ssn", "string", false, false, true, nil, nil, nil)
	if err != dek.ErrDisabled {
		t.Fatalf("got %v, want dek.ErrDisabled", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
