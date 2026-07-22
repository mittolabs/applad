package githubapp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"strings"
	"testing"
)

func testApp(t *testing.T) *App {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}
	app, err := New(Config{
		AppID:         "123456",
		Slug:          "applad",
		WebhookSecret: "s3cret",
		PrivateKey:    string(pem.EncodeToMemory(block)),
	})
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func TestNewWithoutCredentialsIsNotAnError(t *testing.T) {
	// A self-hosted instance has no app; that has to be distinguishable from
	// a broken one, because one is normal and the other needs fixing.
	if _, err := New(Config{}); err != ErrNotConfigured {
		t.Errorf("expected ErrNotConfigured, got %v", err)
	}
	if _, err := New(Config{AppID: "1", PrivateKey: "not a pem"}); err == nil ||
		err == ErrNotConfigured {
		t.Errorf("a bad key should be a real error, got %v", err)
	}
}

func TestPrivateKeySurvivesEscapedNewlines(t *testing.T) {
	// A PEM in a single-line env var arrives with literal backslash-n.
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}
	flattened := strings.ReplaceAll(string(pem.EncodeToMemory(block)), "\n", `\n`)

	if _, err := New(Config{AppID: "1", PrivateKey: flattened}); err != nil {
		t.Errorf("escaped PEM rejected: %v", err)
	}
}

func TestAppJWTIsSignedAndShortLived(t *testing.T) {
	app := testApp(t)
	token, err := app.appJWT()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(token, ".") != 2 {
		t.Errorf("not a JWT: %q", token)
	}
}

func TestVerifyWebhook(t *testing.T) {
	app := testApp(t)
	body := []byte(`{"action":"opened"}`)

	mac := hmac.New(sha256.New, []byte("s3cret"))
	mac.Write(body)
	good := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !app.VerifyWebhook(good, body) {
		t.Error("a correctly signed delivery was rejected")
	}
	if app.VerifyWebhook(good, []byte(`{"action":"closed"}`)) {
		t.Error("a tampered body was accepted")
	}
	if app.VerifyWebhook("", body) {
		t.Error("an unsigned delivery was accepted")
	}
	if app.VerifyWebhook("sha256=deadbeef", body) {
		t.Error("a wrong signature was accepted")
	}
}

func TestParseRepoURL(t *testing.T) {
	cases := map[string][2]string{
		"https://github.com/mittolabs/applad":        {"mittolabs", "applad"},
		"https://github.com/mittolabs/applad.git":    {"mittolabs", "applad"},
		"https://github.com/mittolabs/applad/":       {"mittolabs", "applad"},
		"git@github.com:mittolabs/applad.git":        {"mittolabs", "applad"},
		"http://github.com/mittolabs/applad":         {"mittolabs", "applad"},
		"https://github.com/mittolabs/applad/tree/x": {"mittolabs", "applad"},
	}
	for raw, want := range cases {
		owner, repo, ok := ParseRepoURL(raw)
		if !ok || owner != want[0] || repo != want[1] {
			t.Errorf("%s → %q/%q ok=%v, want %q/%q", raw, owner, repo, ok, want[0], want[1])
		}
	}

	for _, raw := range []string{
		"https://gitlab.com/group/proj",
		"https://example.com/x",
		"",
		"github.com/",
	} {
		if _, _, ok := ParseRepoURL(raw); ok {
			t.Errorf("%q was parsed as a GitHub repo", raw)
		}
	}
}

func TestCloneURLCarriesTheTokenAsAPassword(t *testing.T) {
	got := CloneURL("https://github.com/mittolabs/private.git", "ghs_abc123")
	want := "https://x-access-token:ghs_abc123@github.com/mittolabs/private.git"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Nothing to add, nothing to break.
	if got := CloneURL("https://github.com/a/b", ""); got != "https://github.com/a/b" {
		t.Errorf("a tokenless URL was rewritten: %q", got)
	}
	if got := CloneURL("git@github.com:a/b.git", "tok"); got != "git@github.com:a/b.git" {
		t.Errorf("an ssh URL was rewritten: %q", got)
	}
}
