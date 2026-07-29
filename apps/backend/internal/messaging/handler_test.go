package messaging

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/db"
	mw "github.com/mittolabs/applad/internal/middleware"
)

// recipientList is the load-bearing decode fix: the SMS/push send bodies used
// to declare `to`/`token` as a plain string, so an array payload — which every
// server SDK sends, matching email — failed to unmarshal and the request 400'd
// before anything was sent. It must now accept both shapes.
func TestRecipientList_DecodesArrayAndString(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"array", `["+15551112222","+15553334444"]`, []string{"+15551112222", "+15553334444"}},
		{"single string", `"+15551112222"`, []string{"+15551112222"}},
		{"empty array", `[]`, nil},
		{"empty string", `""`, nil},
		{"null", `null`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var r recipientList
			if err := json.Unmarshal([]byte(tc.in), &r); err != nil {
				t.Fatalf("unmarshal %s: %v", tc.in, err)
			}
			if !reflect.DeepEqual([]string(r), tc.want) {
				t.Fatalf("got %v, want %v", []string(r), tc.want)
			}
		})
	}
}

// A push body may carry recipients under "to" (SDKs) or "token" (console);
// firstNonEmpty resolves whichever is populated, preferring "to".
func TestFirstNonEmpty_PrefersToThenToken(t *testing.T) {
	to := recipientList{"a"}
	token := recipientList{"b"}
	if got := firstNonEmpty(to, token); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("to should win: got %v", got)
	}
	if got := firstNonEmpty(nil, token); !reflect.DeepEqual(got, []string{"b"}) {
		t.Fatalf("token fallback: got %v", got)
	}
	if got := firstNonEmpty(nil, nil); got != nil {
		t.Fatalf("both empty should be nil: got %v", got)
	}
}

func newHandler(t *testing.T) (*Handler, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	return NewHandler(NewService(&db.DB{DB: mockDB}, Config{})), mock
}

func postJSON(h http.HandlerFunc, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req = req.WithContext(mw.ContextWithProject(context.Background(), "proj1"))
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

// The SMS create handler must accept an array `to` and persist every recipient.
// draft:true keeps the assertion on decode+persist without an inline send.
func TestCreateSMS_AcceptsArrayBody(t *testing.T) {
	h, mock := newHandler(t)
	defer mock.ExpectationsWereMet() //nolint:errcheck

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO messages`)).
		WithArgs(sqlmock.AnyArg(), "proj1", "sms", "", "hello",
			`["+15551112222","+15553334444"]`, "draft", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	rec := postJSON(h.createSMS, "/messages/sms",
		`{"to":["+15551112222","+15553334444"],"body":"hello","draft":true}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}
	var msg Message
	if err := json.Unmarshal(rec.Body.Bytes(), &msg); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !reflect.DeepEqual(msg.Recipients, []string{"+15551112222", "+15553334444"}) {
		t.Fatalf("recipients = %v, want both numbers", msg.Recipients)
	}
}

// The push create handler must accept an array `to` (SDK shape) plus an optional
// data map, and still accept `token` (console shape) as a fallback.
func TestCreatePush_AcceptsArrayBodyAndTokenFallback(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"sdk to key", `{"to":["dev1","dev2"],"title":"Hi","body":"there","data":{"k":"v"},"draft":true}`},
		{"console token key", `{"token":["dev1","dev2"],"title":"Hi","body":"there","draft":true}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, mock := newHandler(t)
			defer mock.ExpectationsWereMet() //nolint:errcheck

			mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO messages`)).
				WithArgs(sqlmock.AnyArg(), "proj1", "push", "Hi", "there",
					`["dev1","dev2"]`, "draft", sqlmock.AnyArg()).
				WillReturnResult(sqlmock.NewResult(1, 1))

			rec := postJSON(h.createPush, "/messages/push", tc.body)
			if rec.Code != http.StatusCreated {
				t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body.String())
			}
			var msg Message
			if err := json.Unmarshal(rec.Body.Bytes(), &msg); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if !reflect.DeepEqual(msg.Recipients, []string{"dev1", "dev2"}) {
				t.Fatalf("recipients = %v, want both tokens", msg.Recipients)
			}
		})
	}
}

// The legacy /sms endpoint (the one the server SDKs actually call) previously
// 400'd on an array body. It must now decode the array and proceed to send —
// here the send fails only because no provider is configured, which is a 500,
// proving the decode barrier is gone.
func TestSendSMSLegacy_ArrayBodyPassesDecode(t *testing.T) {
	h, mock := newHandler(t)
	defer mock.ExpectationsWereMet() //nolint:errcheck

	// SendSMSForProject first looks for a project provider; none configured.
	mock.ExpectQuery(regexp.QuoteMeta(`FROM msg_providers`)).
		WithArgs("proj1", "sms").
		WillReturnError(sql.ErrNoRows)

	rec := postJSON(h.sendSMSLegacy, "/sms",
		`{"to":["+15551112222"],"body":"hello"}`)

	if rec.Code == http.StatusBadRequest {
		t.Fatalf("array body should decode, got 400: %s", rec.Body.String())
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (no provider configured)", rec.Code)
	}
}
