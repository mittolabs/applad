package applad

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestServer creates a test HTTP server that records requests and responds with static data.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	client := New(srv.URL, "proj-1", "key-123")
	return srv, client
}

func jsonHandler(data interface{}, status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if data != nil {
			json.NewEncoder(w).Encode(data) //nolint:errcheck
		}
	}
}

// ── Client ────────────────────────────────────────────────────────────────────

func TestNew(t *testing.T) {
	c := New("http://localhost:8080/", "proj-1", "key-123")
	if c.endpoint != "http://localhost:8080" {
		t.Errorf("expected trailing slash stripped, got %s", c.endpoint)
	}
	if c.projectID != "proj-1" {
		t.Errorf("expected projectID=proj-1, got %s", c.projectID)
	}
}

func TestCallSendsHeaders(t *testing.T) {
	var gotProject, gotKey string
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotProject = r.Header.Get("X-Applad-Project")
		gotKey = r.Header.Get("X-Applad-Key")
		w.WriteHeader(200)
		w.Write([]byte("{}")) //nolint:errcheck
	})
	defer srv.Close()

	_, err := client.call("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotProject != "proj-1" {
		t.Errorf("expected project header proj-1, got %s", gotProject)
	}
	if gotKey != "key-123" {
		t.Errorf("expected key header key-123, got %s", gotKey)
	}
}

func TestCallBuildsURL(t *testing.T) {
	var gotPath string
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte("{}")) //nolint:errcheck
	})
	defer srv.Close()

	client.call("GET", "/databases", nil) //nolint:errcheck
	if gotPath != "/v1/databases" {
		t.Errorf("expected /v1/databases, got %s", gotPath)
	}
}

func TestCallError(t *testing.T) {
	srv, client := newTestServer(t, jsonHandler(map[string]string{"message": "not found"}, 404))
	defer srv.Close()

	_, err := client.call("GET", "/missing", nil)
	if err == nil {
		t.Fatal("expected error on 404")
	}
}

func TestCallNoContent(t *testing.T) {
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})
	defer srv.Close()

	res, err := client.call("DELETE", "/resource", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res != nil {
		t.Errorf("expected nil for 204, got %v", res)
	}
}

// ── Service accessors ─────────────────────────────────────────────────────────

func TestServiceAccessors(t *testing.T) {
	c := New("http://localhost:8080", "p", "k")
	if c.Users() == nil {
		t.Error("Users() returned nil")
	}
	if c.Databases() == nil {
		t.Error("Databases() returned nil")
	}
	if c.Storage() == nil {
		t.Error("Storage() returned nil")
	}
	if c.Functions() == nil {
		t.Error("Functions() returned nil")
	}
	if c.Teams() == nil {
		t.Error("Teams() returned nil")
	}
	if c.Workflows() == nil {
		t.Error("Workflows() returned nil")
	}
	if c.Messaging() == nil {
		t.Error("Messaging() returned nil")
	}
	if c.Deploy() == nil {
		t.Error("Deploy() returned nil")
	}
	if c.Flags() == nil {
		t.Error("Flags() returned nil")
	}
	if c.Analytics() == nil {
		t.Error("Analytics() returned nil")
	}
	if c.Search() == nil {
		t.Error("Search() returned nil")
	}
	if c.Vectors() == nil {
		t.Error("Vectors() returned nil")
	}
	if c.Edge() == nil {
		t.Error("Edge() returned nil")
	}
	if c.Regions() == nil {
		t.Error("Regions() returned nil")
	}
}

// ── Users ─────────────────────────────────────────────────────────────────────

func TestUsersListUsers(t *testing.T) {
	var gotMethod, gotPath string
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{"users":[],"total":0}`)) //nolint:errcheck
	})
	defer srv.Close()

	res, err := client.Users().ListUsers()
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != "GET" {
		t.Errorf("expected GET, got %s", gotMethod)
	}
	if gotPath != "/v1/users" {
		t.Errorf("expected /v1/users, got %s", gotPath)
	}
	if res["total"].(float64) != 0 {
		t.Errorf("expected total=0")
	}
}

func TestUsersCreateUser(t *testing.T) {
	srv, client := newTestServer(t, jsonHandler(map[string]string{"$id": "u1"}, 200))
	defer srv.Close()

	res, err := client.Users().CreateUser("a@b.com", "pass", "Test")
	if err != nil {
		t.Fatal(err)
	}
	if res["$id"] != "u1" {
		t.Errorf("expected $id=u1, got %v", res["$id"])
	}
}

// ── Databases ─────────────────────────────────────────────────────────────────

func TestDatabasesListDatabases(t *testing.T) {
	srv, client := newTestServer(t, jsonHandler(map[string]interface{}{"databases": []interface{}{}}, 200))
	defer srv.Close()

	res, err := client.Databases().ListDatabases()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res["databases"]; !ok {
		t.Error("expected databases key in response")
	}
}

func TestDatabasesCreateDatabase(t *testing.T) {
	var gotPath string
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{"$id":"db1","name":"mydb"}`)) //nolint:errcheck
	})
	defer srv.Close()

	res, err := client.Databases().CreateDatabase("mydb")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/databases" {
		t.Errorf("expected /v1/databases, got %s", gotPath)
	}
	if res["name"] != "mydb" {
		t.Errorf("expected name=mydb, got %v", res["name"])
	}
}

func TestDatabasesCreateTable(t *testing.T) {
	var gotPath string
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{"$id":"t1"}`)) //nolint:errcheck
	})
	defer srv.Close()

	_, err := client.Databases().CreateTable("db1", "users")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/databases/db1/tables" {
		t.Errorf("expected /v1/databases/db1/tables, got %s", gotPath)
	}
}

func TestDatabasesCreateColumn(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody) //nolint:errcheck
		w.WriteHeader(200)
		w.Write([]byte(`{"key":"ssn"}`)) //nolint:errcheck
	})
	defer srv.Close()

	_, err := client.Databases().CreateColumn("db1", "t1", "string", "ssn", ColumnOptions{Encrypted: true})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/databases/db1/tables/t1/columns/string" {
		t.Errorf("expected .../columns/string, got %s", gotPath)
	}
	if gotBody["encrypted"] != true {
		t.Errorf("expected encrypted=true in request body, got %v", gotBody["encrypted"])
	}
	if gotBody["required"] != false || gotBody["array"] != false {
		t.Errorf("expected required/array to default false, got %v", gotBody)
	}
}

func TestDatabasesCreateColumn_DefaultsEncryptedFalse(t *testing.T) {
	var gotBody map[string]interface{}
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody) //nolint:errcheck
		w.WriteHeader(200)
		w.Write([]byte(`{"key":"name"}`)) //nolint:errcheck
	})
	defer srv.Close()

	_, err := client.Databases().CreateColumn("db1", "t1", "string", "name", ColumnOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["encrypted"] != false {
		t.Errorf("expected encrypted=false by default, got %v", gotBody["encrypted"])
	}
}

func TestDatabasesRowCRUD(t *testing.T) {
	srv, client := newTestServer(t, jsonHandler(map[string]interface{}{"$id": "r1", "data": map[string]interface{}{"name": "test"}}, 200))
	defer srv.Close()

	res, err := client.Databases().CreateRow("db1", "t1", map[string]interface{}{"name": "test"})
	if err != nil {
		t.Fatal(err)
	}
	if res["$id"] != "r1" {
		t.Errorf("expected $id=r1")
	}
}

// ── Storage ───────────────────────────────────────────────────────────────────

func TestStorageListBuckets(t *testing.T) {
	srv, client := newTestServer(t, jsonHandler(map[string]interface{}{"buckets": []interface{}{}}, 200))
	defer srv.Close()

	res, err := client.Storage().ListBuckets()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res["buckets"]; !ok {
		t.Error("expected buckets key")
	}
}

// ── Functions ─────────────────────────────────────────────────────────────────

func TestFunctionsList(t *testing.T) {
	srv, client := newTestServer(t, jsonHandler(map[string]interface{}{"functions": []interface{}{}}, 200))
	defer srv.Close()

	_, err := client.Functions().List()
	if err != nil {
		t.Fatal(err)
	}
}

func TestFunctionsExecute(t *testing.T) {
	var gotPath string
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{"output":"hello"}`)) //nolint:errcheck
	})
	defer srv.Close()

	_, err := client.Functions().Execute("fn1", map[string]interface{}{"input": "world"})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/functions/fn1/executions" {
		t.Errorf("expected /v1/functions/fn1/executions, got %s", gotPath)
	}
}

// ── Workflows ─────────────────────────────────────────────────────────────────

func TestWorkflowsList(t *testing.T) {
	srv, client := newTestServer(t, jsonHandler(map[string]interface{}{"workflows": []interface{}{}}, 200))
	defer srv.Close()

	_, err := client.Workflows().List()
	if err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowsExecute(t *testing.T) {
	var gotPath string
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{"executionId":"ex1"}`)) //nolint:errcheck
	})
	defer srv.Close()

	_, err := client.Workflows().Execute("wf1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/workflows/wf1/execute" {
		t.Errorf("expected /v1/workflows/wf1/execute, got %s", gotPath)
	}
}

// ── Teams ─────────────────────────────────────────────────────────────────────

func TestTeamsList(t *testing.T) {
	srv, client := newTestServer(t, jsonHandler(map[string]interface{}{"teams": []interface{}{}}, 200))
	defer srv.Close()

	_, err := client.Teams().List()
	if err != nil {
		t.Fatal(err)
	}
}

// ── Messaging ─────────────────────────────────────────────────────────────────

func TestMessagingSendEmail(t *testing.T) {
	var gotPath string
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{"success":true}`)) //nolint:errcheck
	})
	defer srv.Close()

	_, err := client.Messaging().SendEmail([]string{"a@b.com"}, "Test", "<p>Hi</p>")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/messaging/email" {
		t.Errorf("expected /v1/messaging/email, got %s", gotPath)
	}
}

// ── Deploy ────────────────────────────────────────────────────────────────────

func TestDeployList(t *testing.T) {
	srv, client := newTestServer(t, jsonHandler(map[string]interface{}{"deployments": []interface{}{}}, 200))
	defer srv.Close()

	_, err := client.Deploy().List()
	if err != nil {
		t.Fatal(err)
	}
}

// ── Flags ─────────────────────────────────────────────────────────────────────

func TestFlagsList(t *testing.T) {
	srv, client := newTestServer(t, jsonHandler(map[string]interface{}{"flags": []interface{}{}}, 200))
	defer srv.Close()

	_, err := client.Flags().List()
	if err != nil {
		t.Fatal(err)
	}
}

func TestFlagsCreate(t *testing.T) {
	var gotPath string
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{"key":"dark-mode"}`)) //nolint:errcheck
	})
	defer srv.Close()

	_, err := client.Flags().Create("dark-mode", "Dark Mode", nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/flags" {
		t.Errorf("expected /v1/flags, got %s", gotPath)
	}
}

func TestFlagsToggle(t *testing.T) {
	var gotPath string
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{"enabled":false}`)) //nolint:errcheck
	})
	defer srv.Close()

	_, err := client.Flags().Toggle("dark-mode", false)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/flags/dark-mode/toggle" {
		t.Errorf("expected /v1/flags/dark-mode/toggle, got %s", gotPath)
	}
}

func TestFlagsEvaluateFlag(t *testing.T) {
	srv, client := newTestServer(t, jsonHandler(map[string]interface{}{"value": true}, 200))
	defer srv.Close()

	res, err := client.Flags().EvaluateFlag("dark-mode", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res["value"] != true {
		t.Errorf("expected value=true")
	}
}

// ── Analytics ────────────────────────────────────────────────────────────────

func TestAnalyticsTrackEvent(t *testing.T) {
	var gotPath string
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{"success":true}`)) //nolint:errcheck
	})
	defer srv.Close()

	_, err := client.Analytics().TrackEvent("signup", map[string]interface{}{"source": "web"})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/analytics/events" {
		t.Errorf("expected /v1/analytics/events, got %s", gotPath)
	}
}

// ── Search ───────────────────────────────────────────────────────────────────

func TestSearchQuery(t *testing.T) {
	var gotPath string
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{"documents":[]}`)) //nolint:errcheck
	})
	defer srv.Close()

	_, err := client.Search().Query("idx1", "hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/search/indexes/idx1/search" {
		t.Errorf("expected /v1/search/indexes/idx1/search, got %s", gotPath)
	}
}

// ── Vectors ──────────────────────────────────────────────────────────────────

func TestVectorsQuery(t *testing.T) {
	var gotPath string
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{"matches":[]}`)) //nolint:errcheck
	})
	defer srv.Close()

	_, err := client.Vectors().Query("vec1", []float64{0.1, 0.2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/vectors/indexes/vec1/query" {
		t.Errorf("expected /v1/vectors/indexes/vec1/query, got %s", gotPath)
	}
}

// ── Edge ─────────────────────────────────────────────────────────────────────

func TestEdgeInvoke(t *testing.T) {
	var gotPath string
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
	})
	defer srv.Close()

	_, err := client.Edge().Invoke("edge1", map[string]interface{}{"name": "test"})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/edge/functions/edge1/invoke" {
		t.Errorf("expected /v1/edge/functions/edge1/invoke, got %s", gotPath)
	}
}

// ── Regions ──────────────────────────────────────────────────────────────────

func TestRegionsSetActive(t *testing.T) {
	var gotPath string
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{"regionId":"fra1"}`)) //nolint:errcheck
	})
	defer srv.Close()

	_, err := client.Regions().SetActive("fra1")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/regions/active" {
		t.Errorf("expected /v1/regions/active, got %s", gotPath)
	}
}
