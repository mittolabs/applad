package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRow_MarshalJSON_MergesData(t *testing.T) {
	row := Row{
		ID:           "doc1",
		TableID:      "tbl1",
		DatabaseID:   "db1",
		CreatedAt:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Permissions:  []string{"read(\"any\")"},
		Data: map[string]interface{}{
			"title": "Hello",
			"count": float64(42),
		},
	}

	b, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	// Check that data fields are at the top level
	if result["title"] != "Hello" {
		t.Fatalf("expected title='Hello', got %v", result["title"])
	}
	if result["count"] != float64(42) {
		t.Fatalf("expected count=42, got %v", result["count"])
	}

	// Check that standard fields are present
	if result["$id"] != "doc1" {
		t.Fatalf("expected $id='doc1', got %v", result["$id"])
	}
	if result["$tableId"] != "tbl1" {
		t.Fatalf("expected $tableId='tbl1', got %v", result["$tableId"])
	}
}

func TestRow_MarshalJSON_EmptyData(t *testing.T) {
	row := Row{
		ID:          "doc2",
		Permissions: []string{},
	}

	b, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(b, &result)
	if result["$id"] != "doc2" {
		t.Fatalf("expected $id='doc2', got %v", result["$id"])
	}
}

func TestUser_JSONTags(t *testing.T) {
	u := User{
		ID:            "u1",
		Name:          "Alice",
		Email:         "alice@example.com",
		EmailVerified: true,
		Status:        true,
		Labels:        []string{"admin"},
		Prefs:         map[string]interface{}{"theme": "dark"},
		CreatedAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	b, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(b, &result)

	// Appwrite-style field names
	if result["$id"] != "u1" {
		t.Fatalf("expected $id, got %v", result["$id"])
	}
	if result["emailVerification"] != true {
		t.Fatalf("expected emailVerification=true, got %v", result["emailVerification"])
	}
}

func TestProject_JSONTags(t *testing.T) {
	p := Project{
		ID:   "p1",
		Name: "My Project",
	}

	b, _ := json.Marshal(p)
	var result map[string]interface{}
	json.Unmarshal(b, &result)

	if result["$id"] != "p1" {
		t.Fatalf("expected $id='p1', got %v", result["$id"])
	}
}

func TestAPIKey_SecretOmitEmpty(t *testing.T) {
	k := APIKey{
		ID:     "k1",
		Name:   "test",
		Scopes: []string{},
	}

	b, _ := json.Marshal(k)
	var result map[string]interface{}
	json.Unmarshal(b, &result)

	if _, ok := result["secret"]; ok {
		t.Fatal("secret should be omitted when empty")
	}

	k.Secret = "applad_key_abc123"
	b, _ = json.Marshal(k)
	json.Unmarshal(b, &result)
	if result["secret"] != "applad_key_abc123" {
		t.Fatal("secret should be present when set")
	}
}
