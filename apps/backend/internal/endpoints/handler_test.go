package endpoints

import (
	"net/http"
	"testing"

	"github.com/mittolabs/applad/internal/workflows"
)

func TestFlattenHeaders_RedactsSecrets(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer supersecret")
	r.Header.Set("Cookie", "applad_session=abc")
	r.Header.Set("X-Applad-Key", "applad_key_live_xyz")
	r.Header.Set("X-Custom", "keepme")

	h := flattenHeaders(r)
	for _, k := range []string{"Authorization", "Cookie", "X-Applad-Key"} {
		if h[k] != "[redacted]" {
			t.Errorf("%s = %v, want [redacted]", k, h[k])
		}
	}
	if h["X-Custom"] != "keepme" {
		t.Errorf("X-Custom = %v, want keepme (non-sensitive headers stay)", h["X-Custom"])
	}
}

func TestRedactedRequest_RedactsSecretsKeepsRest(t *testing.T) {
	req := map[string]interface{}{
		"method": "POST", "path": "/signup",
		"body": map[string]interface{}{"email": "a@b.com", "password": "hunter2", "cardNumber": "4111111111111111"},
	}
	out := redactedRequest(req, nil)
	body, _ := out["body"].(map[string]interface{})
	if body["email"] != "a@b.com" {
		t.Fatalf("non-secret field should survive: %#v", body)
	}
	if body["password"] != "[redacted]" || body["cardNumber"] != "[redacted]" {
		t.Fatalf("secret fields not redacted: %#v", body)
	}
	if out["method"] != "POST" || out["path"] != "/signup" {
		t.Fatalf("redactedRequest dropped non-sensitive fields: %#v", out)
	}
}

// The author can name any field to redact, beyond the keyword defaults.
func TestRedactedRequest_AuthorNamedFields(t *testing.T) {
	req := map[string]interface{}{
		"body": map[string]interface{}{"ssn": "keep?", "note": "public", "internalId": "x"},
	}
	extra := map[string]bool{"internalid": true}
	body := redactedRequest(req, extra)["body"].(map[string]interface{})
	if body["internalId"] != "[redacted]" {
		t.Fatalf("author-named field not redacted: %#v", body)
	}
	if body["note"] != "public" {
		t.Fatalf("unnamed field should survive: %#v", body)
	}
}

func TestHandlerRedactSet(t *testing.T) {
	ep := &Endpoint{Nodes: []workflows.Node{
		{ID: "n1", Type: "endpoint_handler", Config: map[string]interface{}{"redactFields": []interface{}{"Foo", " bar "}}},
	}}
	set := handlerRedactSet(ep)
	if !set["foo"] || !set["bar"] {
		t.Fatalf("expected lowercased trimmed set, got %#v", set)
	}
}

func TestRedactValue_Nested(t *testing.T) {
	in := map[string]interface{}{
		"user": map[string]interface{}{
			"name":     "Ada",
			"token":    "abc",
			"payments": []interface{}{map[string]interface{}{"cvv": "123", "last4": "4242"}},
		},
	}
	out := redactValue(in, nil).(map[string]interface{})
	u := out["user"].(map[string]interface{})
	if u["name"] != "Ada" || u["token"] != "[redacted]" {
		t.Fatalf("nested redaction wrong: %#v", u)
	}
	pay := u["payments"].([]interface{})[0].(map[string]interface{})
	if pay["cvv"] != "[redacted]" || pay["last4"] != "4242" {
		t.Fatalf("array-of-object redaction wrong: %#v", pay)
	}
}

func TestRedactedLogs_RedactsHandlerBody(t *testing.T) {
	fullReq := map[string]interface{}{
		"method": "POST",
		"body":   map[string]interface{}{"password": "hunter2", "email": "a@b.com"},
	}
	logs := []workflows.StepLog{
		{NodeID: "n1", NodeType: "endpoint_handler", Output: fullReq},
		{NodeID: "n2", NodeType: "endpoint_data", Output: map[string]interface{}{"ok": true}},
	}
	out := redactedLogs(logs, nil)

	ho, _ := out[0].Output.(map[string]interface{})
	body, _ := ho["body"].(map[string]interface{})
	if body["password"] != "[redacted]" || body["email"] != "a@b.com" {
		t.Fatalf("stored handler trace not redacted correctly: %#v", body)
	}
	// The original logs (returned to the author's live test runner) are untouched.
	orig := logs[0].Output.(map[string]interface{})["body"].(map[string]interface{})
	if orig["password"] != "hunter2" {
		t.Fatalf("redactedLogs mutated the original trace; live test would lose data")
	}
}
