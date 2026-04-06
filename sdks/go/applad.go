// Package applad provides a Go server SDK for the Applad BaaS API.
package applad

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// Client is the main entry point for the Applad server SDK.
type Client struct {
	endpoint   string
	projectID  string
	apiKey     string
	httpClient *http.Client
}

// New creates a new Applad server client.
func New(endpoint, projectID, apiKey string) *Client {
	return &Client{
		endpoint:  strings.TrimRight(endpoint, "/"),
		projectID: projectID,
		apiKey:    apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// call makes an authenticated API request and decodes the JSON response.
func (c *Client) call(method, path string, body interface{}) (map[string]interface{}, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("applad: marshal body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.endpoint+"/v1"+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("applad: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Applad-Project", c.projectID)
	req.Header.Set("X-Applad-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("applad: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("applad: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("applad: %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("applad: decode response: %w", err)
	}
	return result, nil
}

// upload sends a multipart form request for file uploads.
func (c *Client) upload(path string, fields map[string]string, fileName string, fileData io.Reader) (map[string]interface{}, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return nil, fmt.Errorf("applad: write field %s: %w", k, err)
		}
	}

	part, err := w.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("applad: create form file: %w", err)
	}
	if _, err := io.Copy(part, fileData); err != nil {
		return nil, fmt.Errorf("applad: copy file data: %w", err)
	}
	w.Close()

	req, err := http.NewRequest("POST", c.endpoint+"/v1"+path, &buf)
	if err != nil {
		return nil, fmt.Errorf("applad: create request: %w", err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-Applad-Project", c.projectID)
	req.Header.Set("X-Applad-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("applad: POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("applad: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("applad: POST %s returned %d: %s", path, resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("applad: decode response: %w", err)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Service accessors
// ---------------------------------------------------------------------------

func (c *Client) Users() *UsersService         { return &UsersService{client: c} }
func (c *Client) Databases() *DatabasesService  { return &DatabasesService{client: c} }
func (c *Client) Storage() *StorageService      { return &StorageService{client: c} }
func (c *Client) Functions() *FunctionsService   { return &FunctionsService{client: c} }
func (c *Client) Teams() *TeamsService           { return &TeamsService{client: c} }
func (c *Client) Workflows() *WorkflowsService   { return &WorkflowsService{client: c} }
func (c *Client) Messaging() *MessagingService   { return &MessagingService{client: c} }
func (c *Client) Deploy() *DeployService         { return &DeployService{client: c} }
func (c *Client) Flags() *FlagsService           { return &FlagsService{client: c} }

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

type UsersService struct{ client *Client }

func (s *UsersService) CreateUser(email, password string, name string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"userId":   "unique()",
		"email":    email,
		"password": password,
	}
	if name != "" {
		body["name"] = name
	}
	return s.client.call("POST", "/users", body)
}

func (s *UsersService) GetUser(userID string) (map[string]interface{}, error) {
	return s.client.call("GET", "/users/"+userID, nil)
}

func (s *UsersService) ListUsers() (map[string]interface{}, error) {
	return s.client.call("GET", "/users", nil)
}

func (s *UsersService) DeleteUser(userID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", "/users/"+userID, nil)
}

// ---------------------------------------------------------------------------
// Databases
// ---------------------------------------------------------------------------

type DatabasesService struct{ client *Client }

func (s *DatabasesService) CreateDatabase(name string) (map[string]interface{}, error) {
	return s.client.call("POST", "/databases", map[string]interface{}{
		"name":       name,
		"databaseId": "unique()",
	})
}

func (s *DatabasesService) ListDatabases() (map[string]interface{}, error) {
	return s.client.call("GET", "/databases", nil)
}

func (s *DatabasesService) GetDatabase(databaseID string) (map[string]interface{}, error) {
	return s.client.call("GET", "/databases/"+databaseID, nil)
}

func (s *DatabasesService) DeleteDatabase(databaseID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", "/databases/"+databaseID, nil)
}

func (s *DatabasesService) CreateTable(databaseID, name string) (map[string]interface{}, error) {
	return s.client.call("POST", fmt.Sprintf("/databases/%s/tables", databaseID), map[string]interface{}{
		"name":             name,
		"tableId":          "unique()",
		"permissions":      []string{},
		"documentSecurity": false,
	})
}

func (s *DatabasesService) ListTables(databaseID string) (map[string]interface{}, error) {
	return s.client.call("GET", fmt.Sprintf("/databases/%s/tables", databaseID), nil)
}

func (s *DatabasesService) CreateRow(databaseID, tableID string, data map[string]interface{}) (map[string]interface{}, error) {
	return s.client.call("POST", fmt.Sprintf("/databases/%s/tables/%s/rows", databaseID, tableID), map[string]interface{}{
		"rowId":       "unique()",
		"data":        data,
		"permissions": []string{},
	})
}

func (s *DatabasesService) ListRows(databaseID, tableID string) (map[string]interface{}, error) {
	return s.client.call("GET", fmt.Sprintf("/databases/%s/tables/%s/rows", databaseID, tableID), nil)
}

func (s *DatabasesService) GetRow(databaseID, tableID, rowID string) (map[string]interface{}, error) {
	return s.client.call("GET", fmt.Sprintf("/databases/%s/tables/%s/rows/%s", databaseID, tableID, rowID), nil)
}

func (s *DatabasesService) UpdateRow(databaseID, tableID, rowID string, data map[string]interface{}) (map[string]interface{}, error) {
	return s.client.call("PATCH", fmt.Sprintf("/databases/%s/tables/%s/rows/%s", databaseID, tableID, rowID), map[string]interface{}{
		"data": data,
	})
}

func (s *DatabasesService) DeleteRow(databaseID, tableID, rowID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", fmt.Sprintf("/databases/%s/tables/%s/rows/%s", databaseID, tableID, rowID), nil)
}

// ---------------------------------------------------------------------------
// Storage
// ---------------------------------------------------------------------------

type StorageService struct{ client *Client }

func (s *StorageService) CreateBucket(name string) (map[string]interface{}, error) {
	return s.client.call("POST", "/storage/buckets", map[string]interface{}{
		"name":                  name,
		"bucketId":              "unique()",
		"permissions":           []string{},
		"allowedFileExtensions": []string{},
		"encryption":            false,
		"antivirus":             false,
	})
}

func (s *StorageService) ListBuckets() (map[string]interface{}, error) {
	return s.client.call("GET", "/storage/buckets", nil)
}

// CreateFile uploads a file to a bucket using multipart form encoding.
// fileData is an io.Reader providing the file contents.
func (s *StorageService) CreateFile(bucketID, fileName string, fileData io.Reader) (map[string]interface{}, error) {
	return s.client.upload(
		fmt.Sprintf("/storage/buckets/%s/files", bucketID),
		map[string]string{"fileId": "unique()"},
		fileName,
		fileData,
	)
}

func (s *StorageService) ListFiles(bucketID string) (map[string]interface{}, error) {
	return s.client.call("GET", fmt.Sprintf("/storage/buckets/%s/files", bucketID), nil)
}

func (s *StorageService) GetFile(bucketID, fileID string) (map[string]interface{}, error) {
	return s.client.call("GET", fmt.Sprintf("/storage/buckets/%s/files/%s", bucketID, fileID), nil)
}

func (s *StorageService) DeleteFile(bucketID, fileID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", fmt.Sprintf("/storage/buckets/%s/files/%s", bucketID, fileID), nil)
}

// ---------------------------------------------------------------------------
// Functions
// ---------------------------------------------------------------------------

type FunctionsService struct{ client *Client }

func (s *FunctionsService) Create(name, runtime string) (map[string]interface{}, error) {
	return s.client.call("POST", "/functions", map[string]interface{}{
		"name":       name,
		"runtime":    runtime,
		"entrypoint": "index.handler",
		"timeout":    15,
		"vars":       map[string]string{},
		"source":     "",
	})
}

func (s *FunctionsService) List() (map[string]interface{}, error) {
	return s.client.call("GET", "/functions", nil)
}

func (s *FunctionsService) Get(functionID string) (map[string]interface{}, error) {
	return s.client.call("GET", "/functions/"+functionID, nil)
}

func (s *FunctionsService) Delete(functionID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", "/functions/"+functionID, nil)
}

func (s *FunctionsService) Execute(functionID string, data map[string]interface{}) (map[string]interface{}, error) {
	if data == nil {
		data = map[string]interface{}{}
	}
	return s.client.call("POST", fmt.Sprintf("/functions/%s/executions", functionID), map[string]interface{}{
		"data": data,
	})
}

func (s *FunctionsService) ListExecutions(functionID string) (map[string]interface{}, error) {
	return s.client.call("GET", fmt.Sprintf("/functions/%s/executions", functionID), nil)
}

// ---------------------------------------------------------------------------
// Workflows
// ---------------------------------------------------------------------------

type WorkflowsService struct{ client *Client }

func (s *WorkflowsService) Create(name string) (map[string]interface{}, error) {
	return s.client.call("POST", "/workflows", map[string]interface{}{
		"name":          name,
		"description":   "",
		"triggerType":   "manual",
		"triggerConfig": map[string]interface{}{},
		"nodes":         []interface{}{},
		"edges":         []interface{}{},
	})
}

func (s *WorkflowsService) List() (map[string]interface{}, error) {
	return s.client.call("GET", "/workflows", nil)
}

func (s *WorkflowsService) Get(workflowID string) (map[string]interface{}, error) {
	return s.client.call("GET", "/workflows/"+workflowID, nil)
}

func (s *WorkflowsService) Delete(workflowID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", "/workflows/"+workflowID, nil)
}

func (s *WorkflowsService) Execute(workflowID string, triggerData map[string]interface{}) (map[string]interface{}, error) {
	if triggerData == nil {
		triggerData = map[string]interface{}{}
	}
	return s.client.call("POST", fmt.Sprintf("/workflows/%s/execute", workflowID), map[string]interface{}{
		"triggerData": triggerData,
	})
}

func (s *WorkflowsService) ListExecutions(workflowID string) (map[string]interface{}, error) {
	return s.client.call("GET", fmt.Sprintf("/workflows/%s/executions", workflowID), nil)
}

// ---------------------------------------------------------------------------
// Teams
// ---------------------------------------------------------------------------

type TeamsService struct{ client *Client }

func (s *TeamsService) Create(name string) (map[string]interface{}, error) {
	return s.client.call("POST", "/teams", map[string]interface{}{
		"name":   name,
		"teamId": "unique()",
	})
}

func (s *TeamsService) List() (map[string]interface{}, error) {
	return s.client.call("GET", "/teams", nil)
}

func (s *TeamsService) Get(teamID string) (map[string]interface{}, error) {
	return s.client.call("GET", "/teams/"+teamID, nil)
}

func (s *TeamsService) Delete(teamID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", "/teams/"+teamID, nil)
}

func (s *TeamsService) CreateMembership(teamID, email string, roles []string) (map[string]interface{}, error) {
	return s.client.call("POST", fmt.Sprintf("/teams/%s/memberships", teamID), map[string]interface{}{
		"email": email,
		"roles": roles,
	})
}

func (s *TeamsService) ListMemberships(teamID string) (map[string]interface{}, error) {
	return s.client.call("GET", fmt.Sprintf("/teams/%s/memberships", teamID), nil)
}

// ---------------------------------------------------------------------------
// Messaging
// ---------------------------------------------------------------------------

type MessagingService struct{ client *Client }

func (s *MessagingService) SendEmail(to []string, subject, html string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"to":      to,
		"subject": subject,
	}
	if html != "" {
		body["html"] = html
	}
	return s.client.call("POST", "/messaging/email", body)
}

func (s *MessagingService) SendSMS(to []string, content string) (map[string]interface{}, error) {
	return s.client.call("POST", "/messaging/sms", map[string]interface{}{
		"to":      to,
		"content": content,
	})
}

func (s *MessagingService) SendPush(to []string, title, body string) (map[string]interface{}, error) {
	return s.client.call("POST", "/messaging/push", map[string]interface{}{
		"to":    to,
		"title": title,
		"body":  body,
	})
}

// ---------------------------------------------------------------------------
// Deploy
// ---------------------------------------------------------------------------

type DeployService struct{ client *Client }

func (s *DeployService) Create(name, deployType string, config map[string]interface{}) (map[string]interface{}, error) {
	if config == nil {
		config = map[string]interface{}{}
	}
	return s.client.call("POST", "/deploy", map[string]interface{}{
		"name":   name,
		"type":   deployType,
		"config": config,
	})
}

func (s *DeployService) List() (map[string]interface{}, error) {
	return s.client.call("GET", "/deploy", nil)
}

func (s *DeployService) Get(deploymentID string) (map[string]interface{}, error) {
	return s.client.call("GET", "/deploy/"+deploymentID, nil)
}

func (s *DeployService) Delete(deploymentID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", "/deploy/"+deploymentID, nil)
}

// ---------------------------------------------------------------------------
// Flags
// ---------------------------------------------------------------------------

type FlagsService struct{ client *Client }

// --- CRUD ---

func (s *FlagsService) List() (map[string]interface{}, error) {
	return s.client.call("GET", "/flags", nil)
}

func (s *FlagsService) Create(key, name string, opts map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"key":  key,
		"name": name,
	}
	for k, v := range opts {
		body[k] = v
	}
	return s.client.call("POST", "/flags", body)
}

func (s *FlagsService) Get(key string) (map[string]interface{}, error) {
	return s.client.call("GET", "/flags/"+key, nil)
}

func (s *FlagsService) Update(key string, opts map[string]interface{}) (map[string]interface{}, error) {
	return s.client.call("PUT", "/flags/"+key, opts)
}

func (s *FlagsService) Delete(key string) (map[string]interface{}, error) {
	return s.client.call("DELETE", "/flags/"+key, nil)
}

func (s *FlagsService) Toggle(key string, enabled bool) (map[string]interface{}, error) {
	return s.client.call("PATCH", fmt.Sprintf("/flags/%s/toggle", key), map[string]interface{}{
		"enabled": enabled,
	})
}

// --- Evaluation ---

func (s *FlagsService) GetFlag(key string) (map[string]interface{}, error) {
	return s.client.call("GET", fmt.Sprintf("/flags/evaluate/%s", key), nil)
}

func (s *FlagsService) GetAllFlags(context map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{}
	if context != nil {
		body["context"] = context
	}
	return s.client.call("POST", "/flags/evaluate/all", body)
}

func (s *FlagsService) EvaluateFlag(key string, context map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"key": key,
	}
	if context != nil {
		body["context"] = context
	}
	return s.client.call("POST", "/flags/evaluate", body)
}
