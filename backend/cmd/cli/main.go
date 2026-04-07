// Command applad is the CLI for the Applad BaaS platform.
// It mirrors everything the console does: scaffold projects, push schema migrations,
// invoke deploy targets, manage env vars, trigger pipelines, and tail logs.
//
// Usage:
//
//	applad [command] [flags]
//
// Commands:
//
//	login             Authenticate and store credentials
//	logout            Remove stored credentials
//	projects          Manage projects
//	databases         Manage databases and tables
//	storage           Manage buckets and files
//	auth              Manage users and sessions
//	functions         Manage and invoke functions
//	deploy            Manage deploy targets and pipelines
//	workflows         Manage and execute workflows
//	logs              Tail execution logs
//	schema            Push schema migrations
//	env               Manage project environment variables
//	mcp               Start the MCP server
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const version = "1.0.0"

// Config holds CLI authentication state persisted to ~/.applad/config.json
type Config struct {
	URL       string `json:"url"`
	ProjectID string `json:"projectId"`
	APIKey    string `json:"apiKey"`
	ConsoleToken string `json:"consoleToken"`
}

var (
	cfg        Config
	httpClient = &http.Client{Timeout: 30 * time.Second}
)

func main() {
	loadConfig()

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "version", "--version", "-v":
		fmt.Println("applad", version)

	case "login":
		runLogin(args)
	case "logout":
		runLogout()
	case "projects":
		runProjects(args)
	case "databases", "db":
		runDatabases(args)
	case "storage":
		runStorage(args)
	case "auth":
		runAuth(args)
	case "functions", "fn":
		runFunctions(args)
	case "deploy":
		runDeploy(args)
	case "workflows", "wf":
		runWorkflows(args)
	case "logs":
		runLogs(args)
	case "schema":
		runSchema(args)
	case "env":
		runEnv(args)
	case "whoami":
		runWhoami()
	case "help", "--help", "-h":
		printHelp()
	default:
		fatalf("unknown command: %s (run 'applad help')", cmd)
	}
}

// ── Login / Logout ────────────────────────────────────────────────────────────

func runLogin(args []string) {
	url := flagStr(args, "--url", getenv("APPLAD_URL", "http://localhost:80"))
	email := flagStr(args, "--email", "")
	password := flagStr(args, "--password", "")

	if email == "" {
		email = prompt("Email: ")
	}
	if password == "" {
		password = prompt("Password: ")
	}

	body := map[string]string{"email": email, "password": password}
	res, err := post(url+"/v1/console/login", "", body)
	must(err)

	token, _ := res["token"].(string)
	if token == "" {
		fatalf("login failed: %v", res)
	}
	cfg.URL = url
	cfg.ConsoleToken = token
	saveConfig()
	fmt.Println("Logged in successfully.")

	// Prompt to select or create a project
	fmt.Println("\nProjects:")
	projects, _ := apiList(url+"/v1/projects", token)
	for i, p := range projects {
		fmt.Printf("  [%d] %s (%s)\n", i+1, p["name"], p["$id"])
	}
	if len(projects) > 0 {
		choice := prompt("Select project number (or press Enter to skip): ")
		if n, err := strconv.Atoi(strings.TrimSpace(choice)); err == nil && n >= 1 && n <= len(projects) {
			cfg.ProjectID, _ = projects[n-1]["$id"].(string)
			fmt.Printf("Project set to: %s\n", cfg.ProjectID)
		}
	}
	saveConfig()
}

func runLogout() {
	cfg = Config{}
	saveConfig()
	fmt.Println("Logged out.")
}

func runWhoami() {
	if cfg.ConsoleToken == "" {
		fatalf("not logged in. Run 'applad login'")
	}
	res, err := get(cfg.URL+"/v1/console/me", cfg.ConsoleToken)
	must(err)
	prettyPrint(res)
}

// ── Projects ──────────────────────────────────────────────────────────────────

func runProjects(args []string) {
	if len(args) == 0 {
		args = []string{"list"}
	}
	switch args[0] {
	case "list", "ls":
		res, err := get(cfg.URL+"/v1/projects", cfg.ConsoleToken)
		must(err)
		printList(res, "projects")
	case "create":
		name := flagStr(args[1:], "--name", "")
		if name == "" {
			name = prompt("Project name: ")
		}
		res, err := post(cfg.URL+"/v1/projects", cfg.ConsoleToken, map[string]string{"name": name})
		must(err)
		prettyPrint(res)
		if id, ok := res["$id"].(string); ok {
			cfg.ProjectID = id
			saveConfig()
			fmt.Printf("Active project set to: %s\n", id)
		}
	case "use":
		if len(args) < 2 {
			fatalf("usage: applad projects use <projectId>")
		}
		cfg.ProjectID = args[1]
		saveConfig()
		fmt.Printf("Active project: %s\n", cfg.ProjectID)
	case "delete":
		id := argOrPrompt(args, 1, "Project ID: ")
		must(del(cfg.URL+"/v1/projects/"+id, cfg.ConsoleToken))
		fmt.Println("Deleted.")
	default:
		fatalf("unknown subcommand: applad projects %s", args[0])
	}
}

// ── Databases ─────────────────────────────────────────────────────────────────

func runDatabases(args []string) {
	requireProject()
	if len(args) == 0 {
		args = []string{"list"}
	}
	base := cfg.URL + "/v1/databases"
	switch args[0] {
	case "list", "ls":
		res, err := getWithProject(base)
		must(err)
		printList(res, "databases")
	case "create":
		name := flagStr(args[1:], "--name", argOrEmpty(args, 1))
		if name == "" {
			name = prompt("Database name: ")
		}
		res, err := postWithProject(base, map[string]string{"name": name})
		must(err)
		prettyPrint(res)
	case "tables":
		dbID := argOrPrompt(args, 1, "Database ID: ")
		res, err := getWithProject(base + "/" + dbID + "/collections")
		must(err)
		printList(res, "collections")
	case "create-table":
		dbID := argOrEmpty(args, 1)
		name := flagStr(args[1:], "--name", "")
		if name == "" {
			name = prompt("Table name: ")
		}
		res, err := postWithProject(base+"/"+dbID+"/collections", map[string]string{"name": name})
		must(err)
		prettyPrint(res)
	case "query":
		dbID := argOrEmpty(args, 1)
		tableID := argOrEmpty(args, 2)
		limit := flagStr(args, "--limit", "25")
		res, err := getWithProject(fmt.Sprintf("%s/%s/collections/%s/documents?limit=%s", base, dbID, tableID, limit))
		must(err)
		printList(res, "documents")
	default:
		fatalf("unknown subcommand: applad databases %s", args[0])
	}
}

// ── Storage ───────────────────────────────────────────────────────────────────

func runStorage(args []string) {
	requireProject()
	if len(args) == 0 {
		args = []string{"list"}
	}
	base := cfg.URL + "/v1/storage"
	switch args[0] {
	case "list", "ls":
		res, err := getWithProject(base + "/buckets")
		must(err)
		printList(res, "buckets")
	case "files":
		bucketID := argOrPrompt(args, 1, "Bucket ID: ")
		res, err := getWithProject(base + "/buckets/" + bucketID + "/files")
		must(err)
		printList(res, "files")
	case "upload":
		bucketID := argOrEmpty(args, 1)
		filePath := argOrPrompt(args, 2, "File path: ")
		fmt.Printf("Uploading %s to bucket %s...\n", filePath, bucketID)
		res, err := uploadFile(base+"/buckets/"+bucketID+"/files", filePath)
		must(err)
		prettyPrint(res)
	default:
		fatalf("unknown subcommand: applad storage %s", args[0])
	}
}

// ── Auth ──────────────────────────────────────────────────────────────────────

func runAuth(args []string) {
	requireProject()
	if len(args) == 0 {
		args = []string{"users"}
	}
	base := cfg.URL + "/v1/users"
	switch args[0] {
	case "users", "list":
		res, err := getWithProject(base)
		must(err)
		printList(res, "users")
	case "get":
		userID := argOrPrompt(args, 1, "User ID: ")
		res, err := getWithProject(base + "/" + userID)
		must(err)
		prettyPrint(res)
	case "create":
		email := flagStr(args[1:], "--email", "")
		if email == "" {
			email = prompt("Email: ")
		}
		res, err := postWithProject(base, map[string]string{"email": email})
		must(err)
		prettyPrint(res)
	case "delete":
		userID := argOrPrompt(args, 1, "User ID: ")
		must(delWithProject(base + "/" + userID))
		fmt.Println("Deleted.")
	default:
		fatalf("unknown subcommand: applad auth %s", args[0])
	}
}

// ── Functions ─────────────────────────────────────────────────────────────────

func runFunctions(args []string) {
	requireProject()
	if len(args) == 0 {
		args = []string{"list"}
	}
	base := cfg.URL + "/v1/functions"
	switch args[0] {
	case "list", "ls":
		res, err := getWithProject(base)
		must(err)
		printList(res, "functions")
	case "invoke", "exec":
		fnID := argOrPrompt(args, 1, "Function ID: ")
		body := map[string]interface{}{}
		if data := flagStr(args, "--data", ""); data != "" {
			json.Unmarshal([]byte(data), &body) //nolint:errcheck
		}
		res, err := postWithProject(base+"/"+fnID+"/executions", body)
		must(err)
		prettyPrint(res)
	case "logs":
		fnID := argOrPrompt(args, 1, "Function ID: ")
		res, err := getWithProject(base + "/" + fnID + "/executions")
		must(err)
		printList(res, "executions")
	default:
		fatalf("unknown subcommand: applad functions %s", args[0])
	}
}

// ── Deploy ────────────────────────────────────────────────────────────────────

func runDeploy(args []string) {
	requireProject()
	if len(args) == 0 {
		args = []string{"list"}
	}
	base := cfg.URL + "/v1/deploy"
	switch args[0] {
	case "list", "ls":
		res, err := getWithProject(base + "/targets")
		must(err)
		printList(res, "targets")
	case "trigger":
		targetID := argOrPrompt(args, 1, "Target ID: ")
		res, err := postWithProject(base+"/targets/"+targetID+"/trigger", nil)
		must(err)
		prettyPrint(res)
	case "status":
		targetID := argOrPrompt(args, 1, "Target ID: ")
		res, err := getWithProject(base + "/targets/" + targetID)
		must(err)
		prettyPrint(res)
	default:
		fatalf("unknown subcommand: applad deploy %s", args[0])
	}
}

// ── Workflows ─────────────────────────────────────────────────────────────────

func runWorkflows(args []string) {
	requireProject()
	if len(args) == 0 {
		args = []string{"list"}
	}
	base := cfg.URL + "/v1/workflows"
	switch args[0] {
	case "list", "ls":
		res, err := getWithProject(base)
		must(err)
		printList(res, "workflows")
	case "execute", "run":
		wfID := argOrPrompt(args, 1, "Workflow ID: ")
		body := map[string]interface{}{}
		if data := flagStr(args, "--data", ""); data != "" {
			json.Unmarshal([]byte(data), &body) //nolint:errcheck
		}
		res, err := postWithProject(base+"/"+wfID+"/executions", body)
		must(err)
		prettyPrint(res)
	case "logs":
		wfID := argOrPrompt(args, 1, "Workflow ID: ")
		res, err := getWithProject(base + "/" + wfID + "/executions")
		must(err)
		printList(res, "executions")
	default:
		fatalf("unknown subcommand: applad workflows %s", args[0])
	}
}

// ── Logs ──────────────────────────────────────────────────────────────────────

func runLogs(args []string) {
	requireProject()
	resource := flagStr(args, "--resource", "audit")
	fmt.Printf("Tailing %s logs (Ctrl+C to stop)...\n", resource)

	// Poll audit logs with long-polling simulation
	seen := map[string]bool{}
	for {
		res, err := getWithProject(cfg.URL + "/v1/audit?limit=20")
		if err == nil {
			if logs, ok := res["logs"].([]interface{}); ok {
				for _, l := range logs {
					entry, ok := l.(map[string]interface{})
					if !ok {
						continue
					}
					id, _ := entry["$id"].(string)
					if seen[id] {
						continue
					}
					seen[id] = true
					ts, _ := entry["$createdAt"].(string)
					action, _ := entry["action"].(string)
					path, _ := entry["path"].(string)
					status, _ := entry["statusCode"].(float64)
					fmt.Printf("[%s] %s %s → %d\n", ts, action, path, int(status))
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
}

// ── Schema ────────────────────────────────────────────────────────────────────
//
// Schema file format (schema.json):
//
//	{
//	  "databases": [{
//	    "name": "mydb",
//	    "tables": [{
//	      "name": "users",
//	      "attributes": [
//	        {"key": "name",  "type": "string",  "required": true},
//	        {"key": "email", "type": "email",   "required": true},
//	        {"key": "age",   "type": "integer", "required": false}
//	      ]
//	    }]
//	  }]
//	}

type schemaFile struct {
	Databases []schemaDatabase `json:"databases"`
}
type schemaDatabase struct {
	Name   string        `json:"name"`
	Tables []schemaTable `json:"tables"`
}
type schemaTable struct {
	Name       string            `json:"name"`
	Attributes []schemaAttribute `json:"attributes"`
}
type schemaAttribute struct {
	Key      string `json:"key"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

func runSchema(args []string) {
	requireProject()
	if len(args) == 0 {
		fatalf("usage: applad schema push --file <schema.json>")
	}
	switch args[0] {
	case "push":
		file := flagStr(args[1:], "--file", "schema.json")
		data, err := os.ReadFile(file)
		must(err)
		var schema schemaFile
		if err := json.Unmarshal(data, &schema); err != nil {
			fatalf("invalid JSON in %s: %v", file, err)
		}
		fmt.Printf("Pushing schema from %s...\n", file)
		base := cfg.URL + "/v1/databases"
		for _, sdb := range schema.Databases {
			// Create or find database
			dbRes, err := postWithProject(base, map[string]string{"name": sdb.Name})
			if err != nil {
				// Try to find existing
				listRes, lerr := getWithProject(base)
				must(lerr)
				dbRes = findByName(listRes, "databases", sdb.Name)
				if dbRes == nil {
					fatalf("could not create or find database %q", sdb.Name)
				}
			}
			dbID, _ := dbRes["$id"].(string)
			fmt.Printf("  database %q (%s)\n", sdb.Name, dbID)

			for _, st := range sdb.Tables {
				// Create or find table
				tableBase := base + "/" + dbID + "/collections"
				tRes, terr := postWithProject(tableBase, map[string]string{"name": st.Name})
				if terr != nil {
					listRes2, _ := getWithProject(tableBase)
					tRes = findByName(listRes2, "collections", st.Name)
				}
				tableID, _ := tRes["$id"].(string)
				fmt.Printf("    table %q (%s)\n", st.Name, tableID)

				// Create attributes
				attrBase := tableBase + "/" + tableID + "/attributes"
				for _, a := range st.Attributes {
					_, aerr := postWithProject(attrBase, map[string]interface{}{
						"key":      a.Key,
						"type":     a.Type,
						"required": a.Required,
					})
					if aerr != nil {
						fmt.Printf("      attribute %q: %v (may already exist)\n", a.Key, aerr)
					} else {
						fmt.Printf("      attribute %q (%s)\n", a.Key, a.Type)
					}
				}
			}
		}
		fmt.Println("Schema push complete.")
	case "dump":
		// Dump current schema to stdout as a schema.json skeleton
		res, err := getWithProject(cfg.URL + "/v1/databases")
		must(err)
		prettyPrint(res)
	default:
		fatalf("unknown subcommand: applad schema %s", args[0])
	}
}

// findByName finds an item by name in a list response map.
func findByName(res map[string]interface{}, key, name string) map[string]interface{} {
	if res == nil {
		return nil
	}
	items, ok := res[key].([]interface{})
	if !ok {
		return nil
	}
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if n, _ := m["name"].(string); n == name {
			return m
		}
	}
	return nil
}

// ── Env ───────────────────────────────────────────────────────────────────────

func runEnv(args []string) {
	requireProject()
	if len(args) == 0 {
		args = []string{"list"}
	}
	base := cfg.URL + "/v1/credentials"
	switch args[0] {
	case "list", "ls":
		res, err := getWithProject(base)
		must(err)
		printList(res, "credentials")
	case "set":
		name := argOrPrompt(args, 1, "Name: ")
		value := flagStr(args[1:], "--value", "")
		if value == "" {
			value = prompt("Value: ")
		}
		res, err := postWithProject(base, map[string]string{"name": name, "value": value})
		must(err)
		prettyPrint(res)
	case "delete", "rm":
		id := argOrPrompt(args, 1, "Credential ID: ")
		must(delWithProject(base + "/" + id))
		fmt.Println("Deleted.")
	default:
		fatalf("unknown subcommand: applad env %s", args[0])
	}
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

func get(url, token string) (map[string]interface{}, error) {
	return doRequest(http.MethodGet, url, token, "", nil)
}

func getWithProject(url string) (map[string]interface{}, error) {
	return doRequest(http.MethodGet, url, cfg.ConsoleToken, cfg.ProjectID, nil)
}

func post(url, token string, body interface{}) (map[string]interface{}, error) {
	return doRequest(http.MethodPost, url, token, "", body)
}

func postWithProject(url string, body interface{}) (map[string]interface{}, error) {
	return doRequest(http.MethodPost, url, cfg.ConsoleToken, cfg.ProjectID, body)
}

func del(url, token string) error {
	_, err := doRequest(http.MethodDelete, url, token, "", nil)
	return err
}

func delWithProject(url string) error {
	_, err := doRequest(http.MethodDelete, url, cfg.ConsoleToken, cfg.ProjectID, nil)
	return err
}

func doRequest(method, url, token, projectID string, body interface{}) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if projectID != "" {
		req.Header.Set("X-Applad-Project", projectID)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return map[string]interface{}{}, nil
	}
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if resp.StatusCode >= 400 {
		msg, _ := out["message"].(string)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, msg)
	}
	return out, nil
}

func apiList(url, token string) ([]map[string]interface{}, error) {
	res, err := get(url, token)
	if err != nil {
		return nil, err
	}
	for _, key := range []string{"projects", "users", "data"} {
		if v, ok := res[key]; ok {
			if arr, ok := v.([]interface{}); ok {
				out := make([]map[string]interface{}, 0, len(arr))
				for _, item := range arr {
					if m, ok := item.(map[string]interface{}); ok {
						out = append(out, m)
					}
				}
				return out, nil
			}
		}
	}
	return nil, nil
}

func uploadFile(url, filePath string) (map[string]interface{}, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, err
	}
	mw.Close()

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+cfg.ConsoleToken)
	if cfg.ProjectID != "" {
		req.Header.Set("X-Applad-Project", cfg.ProjectID)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		msg, _ := out["message"].(string)
		return nil, fmt.Errorf("upload error %d: %s", resp.StatusCode, msg)
	}
	return out, nil
}

// ── Config helpers ────────────────────────────────────────────────────────────

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".applad", "config.json")
}

func loadConfig() {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return
	}
	json.Unmarshal(data, &cfg) //nolint:errcheck
	if cfg.URL == "" {
		cfg.URL = getenv("APPLAD_URL", "http://localhost:80")
	}
	if cfg.ProjectID == "" {
		cfg.ProjectID = os.Getenv("APPLAD_PROJECT")
	}
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("APPLAD_KEY")
	}
}

func saveConfig() {
	dir := filepath.Dir(configPath())
	os.MkdirAll(dir, 0700) //nolint:errcheck
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(configPath(), data, 0600) //nolint:errcheck
}

// ── Print helpers ─────────────────────────────────────────────────────────────

func prettyPrint(v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

func printList(res map[string]interface{}, key string) {
	if items, ok := res[key].([]interface{}); ok {
		fmt.Printf("Total: %v\n", res["total"])
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				id, _ := m["$id"].(string)
				name, _ := m["name"].(string)
				status, _ := m["status"].(string)
				line := id
				if name != "" {
					line += " | " + name
				}
				if status != "" {
					line += " | " + status
				}
				fmt.Println(" -", line)
			}
		}
	} else {
		prettyPrint(res)
	}
}

func printHelp() {
	fmt.Print(`applad CLI ` + version + `

Usage: applad <command> [subcommand] [flags]

Commands:
  login             Authenticate (--url, --email, --password)
  logout            Remove stored credentials
  whoami            Show current user

  projects          list | create | use | delete
  databases         list | create | tables | query | create-table
  storage           list | files | upload
  auth              users | get | create | delete
  functions         list | invoke | logs
  deploy            list | trigger | status
  workflows         list | execute | logs
  logs              Tail audit logs (--resource)
  schema            push --file <file.json>
  env               list | set | delete

Global flags:
  APPLAD_URL        API base URL (default: http://localhost:80)
  APPLAD_PROJECT    Project ID
  APPLAD_KEY        API key
`)
}

// ── Utility helpers ───────────────────────────────────────────────────────────

func requireProject() {
	if cfg.ProjectID == "" {
		fatalf("no project selected. Run 'applad projects use <id>'")
	}
}

func must(err error) {
	if err != nil {
		fatalf("%v", err)
	}
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

func prompt(label string) string {
	fmt.Print(label)
	var s string
	fmt.Scanln(&s) //nolint:errcheck
	return strings.TrimSpace(s)
}

func flagStr(args []string, flag, def string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, flag+"=") {
			return strings.TrimPrefix(a, flag+"=")
		}
	}
	return def
}

func argOrEmpty(args []string, pos int) string {
	if pos < len(args) && !strings.HasPrefix(args[pos], "-") {
		return args[pos]
	}
	return ""
}

func argOrPrompt(args []string, pos int, label string) string {
	if v := argOrEmpty(args, pos); v != "" {
		return v
	}
	return prompt(label)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
