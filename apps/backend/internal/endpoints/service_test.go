package endpoints

import (
	"reflect"
	"testing"
)

func TestMatchPath(t *testing.T) {
	cases := []struct {
		name     string
		template string
		actual   string
		want     map[string]string
		score    int
		ok       bool
	}{
		{"exact", "/health", "/health", map[string]string{}, 1, true},
		{"root", "/", "/", map[string]string{}, 0, true},
		{"single param", "/users/{id}", "/users/42", map[string]string{"id": "42"}, 1, true},
		{"two params", "/orgs/{org}/repos/{repo}", "/orgs/acme/repos/api",
			map[string]string{"org": "acme", "repo": "api"}, 2, true},
		{"length mismatch", "/users/{id}", "/users/42/posts", nil, 0, false},
		{"literal mismatch", "/users/{id}", "/teams/42", nil, 0, false},
		{"trailing normalised by caller", "/a/b", "/a/b", map[string]string{}, 2, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			params, score, ok := matchPath(c.template, c.actual)
			if ok != c.ok {
				t.Fatalf("ok = %v, want %v", ok, c.ok)
			}
			if !ok {
				return
			}
			if score != c.score {
				t.Errorf("score = %d, want %d", score, c.score)
			}
			if !reflect.DeepEqual(params, c.want) {
				t.Errorf("params = %#v, want %#v", params, c.want)
			}
		})
	}
}

// An exact route must out-rank a parameterised one that also matches, via the
// specificity score MatchPublished uses to break ties.
func TestMatchPath_ExactBeatsParam(t *testing.T) {
	_, exactScore, ok1 := matchPath("/users/me", "/users/me")
	_, paramScore, ok2 := matchPath("/users/{id}", "/users/me")
	if !ok1 || !ok2 {
		t.Fatal("both templates should match /users/me")
	}
	if exactScore <= paramScore {
		t.Fatalf("exact score %d must exceed param score %d", exactScore, paramScore)
	}
}

func TestNormalizePath(t *testing.T) {
	cases := map[string]string{
		"":             "/",
		"/":            "/",
		"users":        "/users",
		"/users/":      "/users",
		"/users/{id}/": "/users/{id}",
		"a/b/c":        "/a/b/c",
	}
	for in, want := range cases {
		if got := normalizePath(in); got != want {
			t.Errorf("normalizePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeMethod(t *testing.T) {
	cases := map[string]string{"": "GET", "get": "GET", " post ": "POST", "Delete": "DELETE"}
	for in, want := range cases {
		if got := normalizeMethod(in); got != want {
			t.Errorf("normalizeMethod(%q) = %q, want %q", in, got, want)
		}
	}
}
