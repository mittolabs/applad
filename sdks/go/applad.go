// Package applad provides a Go server SDK for the Applad BaaS API.
package applad

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
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
func (c *Client) Databases() *DatabasesService { return &DatabasesService{client: c} }
func (c *Client) Storage() *StorageService     { return &StorageService{client: c} }
func (c *Client) Functions() *FunctionsService { return &FunctionsService{client: c} }
func (c *Client) Teams() *TeamsService         { return &TeamsService{client: c} }
func (c *Client) Workflows() *WorkflowsService { return &WorkflowsService{client: c} }
func (c *Client) Messaging() *MessagingService { return &MessagingService{client: c} }
func (c *Client) Deploy() *DeployService       { return &DeployService{client: c} }
func (c *Client) Flags() *FlagsService         { return &FlagsService{client: c} }
func (c *Client) Analytics() *AnalyticsService { return &AnalyticsService{client: c} }
func (c *Client) Search() *SearchService       { return &SearchService{client: c} }
func (c *Client) Vectors() *VectorsService     { return &VectorsService{client: c} }
func (c *Client) Edge() *EdgeService           { return &EdgeService{client: c} }
func (c *Client) Regions() *RegionsService     { return &RegionsService{client: c} }

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

// ListRowsFiltered narrows a content-enabled table to a publish state and/or a
// locale. Empty values are omitted.
func (s *DatabasesService) ListRowsFiltered(databaseID, tableID, status, locale string) (map[string]interface{}, error) {
	q := url.Values{}
	if status != "" {
		q.Set("status", status)
	}
	if locale != "" {
		q.Set("locale", locale)
	}
	path := fmt.Sprintf("/databases/%s/tables/%s/rows", databaseID, tableID)
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	return s.client.call("GET", path, nil)
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

// --- Content mode ---
// A table can act as an editorial collection: rows gain a draft/published
// workflow, a slug, a locale and version history. Same table, same rows API.

// EnableContentMode turns a table into an editorial collection.
func (s *DatabasesService) EnableContentMode(databaseID, tableID string) (map[string]interface{}, error) {
	return s.client.call("POST", fmt.Sprintf("/databases/%s/tables/%s/content", databaseID, tableID), nil)
}

// DisableContentMode hides the editorial tools again. Nothing is deleted.
func (s *DatabasesService) DisableContentMode(databaseID, tableID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", fmt.Sprintf("/databases/%s/tables/%s/content", databaseID, tableID), nil)
}

// PublishRow publishes an entry.
func (s *DatabasesService) PublishRow(databaseID, tableID, rowID string) (map[string]interface{}, error) {
	return s.client.call("POST", fmt.Sprintf("/databases/%s/tables/%s/rows/%s/publish", databaseID, tableID, rowID), nil)
}

// UnpublishRow returns an entry to draft.
func (s *DatabasesService) UnpublishRow(databaseID, tableID, rowID string) (map[string]interface{}, error) {
	return s.client.call("POST", fmt.Sprintf("/databases/%s/tables/%s/rows/%s/unpublish", databaseID, tableID, rowID), nil)
}

// ListRowVersions returns an entry's version snapshots, newest first.
func (s *DatabasesService) ListRowVersions(databaseID, tableID, rowID string) (map[string]interface{}, error) {
	return s.client.call("GET", fmt.Sprintf("/databases/%s/tables/%s/rows/%s/versions", databaseID, tableID, rowID), nil)
}

// ColumnOptions configures optional column-creation settings. The zero value
// creates a required=false, array=false, encrypted=false column with no
// default, size, bound, or enum constraint.
type ColumnOptions struct {
	Required bool
	Array    bool
	// Encrypted stores this column's values as opaque ciphertext at rest (see
	// field-level encryption docs). Cannot be combined with Array, and
	// requires the instance to have MASTER_ENCRYPTION_KEY configured.
	Encrypted bool
	Default   interface{}
	Size      int      // string columns
	Min, Max  *float64 // integer/double columns
	Elements  []string // enum columns
}

// CreateColumn creates a column of the given type ("string", "integer",
// "boolean", "double", "datetime", "email", "url", "enum", ...) on a table.
func (s *DatabasesService) CreateColumn(databaseID, tableID, columnType, key string, opts ColumnOptions) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"key":       key,
		"required":  opts.Required,
		"array":     opts.Array,
		"encrypted": opts.Encrypted,
	}
	if opts.Default != nil {
		body["default"] = opts.Default
	}
	if opts.Size > 0 {
		body["size"] = opts.Size
	}
	if opts.Min != nil {
		body["min"] = *opts.Min
	}
	if opts.Max != nil {
		body["max"] = *opts.Max
	}
	if len(opts.Elements) > 0 {
		body["elements"] = opts.Elements
	}
	return s.client.call("POST", fmt.Sprintf("/databases/%s/tables/%s/columns/%s", databaseID, tableID, columnType), body)
}

func (s *DatabasesService) GetColumnPermissions(databaseID, tableID, key string) (map[string]interface{}, error) {
	return s.client.call("GET", fmt.Sprintf("/databases/%s/tables/%s/columns/%s/permissions", databaseID, tableID, key), nil)
}

func (s *DatabasesService) SetColumnPermissions(databaseID, tableID, key string, permissions []string) (map[string]interface{}, error) {
	return s.client.call("POST", fmt.Sprintf("/databases/%s/tables/%s/columns/%s/permissions", databaseID, tableID, key), map[string]interface{}{
		"permissions": permissions,
	})
}

// From returns a fluent QueryBuilder for the given table.
//
//	result, err := client.Databases().From("myDb", "posts").
//	    Equal("published", "true").
//	    OrderDesc("created_at").
//	    Limit(25).
//	    Get()
func (s *DatabasesService) From(databaseID, tableID string) *QueryBuilder {
	return &QueryBuilder{
		client:     s.client,
		databaseID: databaseID,
		tableID:    tableID,
	}
}

// QueryResult holds the response from a QueryBuilder.Get call.
type QueryResult struct {
	Total int                      `json:"total"`
	Rows  []map[string]interface{} `json:"rows"`
}

// QueryBuilder is a fluent builder for querying rows in an Applad table.
type QueryBuilder struct {
	client     *Client
	databaseID string
	tableID    string
	queries    []string
	selectCols string
	orderAttr  string
	orderType  string
	limitVal   int
	offsetVal  int
	cursor     string
}

func (q *QueryBuilder) scalar(method, field string, value interface{}) *QueryBuilder {
	q.queries = append(q.queries, fmt.Sprintf(`%s("%s","%v")`, method, field, value))
	return q
}

// Select specifies which columns to return.
func (q *QueryBuilder) Select(columns string) *QueryBuilder {
	q.selectCols = columns
	return q
}

// Equal matches rows where field equals value.
func (q *QueryBuilder) Equal(field string, value interface{}) *QueryBuilder {
	return q.scalar("equal", field, value)
}

// NotEqual matches rows where field does not equal value.
func (q *QueryBuilder) NotEqual(field string, value interface{}) *QueryBuilder {
	return q.scalar("notEqual", field, value)
}

// LessThan matches rows where field is less than value.
func (q *QueryBuilder) LessThan(field string, value interface{}) *QueryBuilder {
	return q.scalar("lessThan", field, value)
}

// LessThanOrEqual matches rows where field is less than or equal to value.
func (q *QueryBuilder) LessThanOrEqual(field string, value interface{}) *QueryBuilder {
	return q.scalar("lessThanEqual", field, value)
}

// GreaterThan matches rows where field is greater than value.
func (q *QueryBuilder) GreaterThan(field string, value interface{}) *QueryBuilder {
	return q.scalar("greaterThan", field, value)
}

// GreaterThanOrEqual matches rows where field is greater than or equal to value.
func (q *QueryBuilder) GreaterThanOrEqual(field string, value interface{}) *QueryBuilder {
	return q.scalar("greaterThanEqual", field, value)
}

// Contains matches rows where field contains value (case-insensitive).
func (q *QueryBuilder) Contains(field, value string) *QueryBuilder {
	return q.scalar("contains", field, value)
}

// StartsWith matches rows where field starts with value.
func (q *QueryBuilder) StartsWith(field, value string) *QueryBuilder {
	return q.scalar("startsWith", field, value)
}

// EndsWith matches rows where field ends with value.
func (q *QueryBuilder) EndsWith(field, value string) *QueryBuilder {
	return q.scalar("endsWith", field, value)
}

// Search performs a full-text search on field for value.
func (q *QueryBuilder) Search(field, value string) *QueryBuilder {
	return q.scalar("search", field, value)
}

// IsNull matches rows where field is NULL.
func (q *QueryBuilder) IsNull(field string) *QueryBuilder {
	q.queries = append(q.queries, fmt.Sprintf(`isNull("%s")`, field))
	return q
}

// IsNotNull matches rows where field is NOT NULL.
func (q *QueryBuilder) IsNotNull(field string) *QueryBuilder {
	q.queries = append(q.queries, fmt.Sprintf(`isNotNull("%s")`, field))
	return q
}

// Between matches rows where field is between min and max (inclusive).
func (q *QueryBuilder) Between(field string, min, max interface{}) *QueryBuilder {
	q.queries = append(q.queries, fmt.Sprintf(`between("%s","%v","%v")`, field, min, max))
	return q
}

// OrderAsc orders results by field ascending.
func (q *QueryBuilder) OrderAsc(field string) *QueryBuilder {
	q.orderAttr = field
	q.orderType = "ASC"
	return q
}

// OrderDesc orders results by field descending.
func (q *QueryBuilder) OrderDesc(field string) *QueryBuilder {
	q.orderAttr = field
	q.orderType = "DESC"
	return q
}

// Limit sets the maximum number of rows to return.
func (q *QueryBuilder) Limit(n int) *QueryBuilder {
	q.limitVal = n
	return q
}

// Offset sets the number of rows to skip.
func (q *QueryBuilder) Offset(n int) *QueryBuilder {
	q.offsetVal = n
	return q
}

// CursorAfter sets cursor-based pagination — pass the last seen row ID.
func (q *QueryBuilder) CursorAfter(rowID string) *QueryBuilder {
	q.cursor = rowID
	return q
}

// Get executes the query and returns a QueryResult.
func (q *QueryBuilder) Get() (*QueryResult, error) {
	params := url.Values{}
	if q.selectCols != "" {
		params.Set("select", q.selectCols)
	}
	if q.orderAttr != "" {
		params.Set("orderAttr", q.orderAttr)
		params.Set("orderType", q.orderType)
	}
	if q.limitVal > 0 {
		params.Set("limit", fmt.Sprintf("%d", q.limitVal))
	}
	if q.offsetVal > 0 {
		params.Set("offset", fmt.Sprintf("%d", q.offsetVal))
	}
	if q.cursor != "" {
		params.Set("cursorAfter", q.cursor)
	}
	for _, qry := range q.queries {
		params.Add("queries[]", qry)
	}

	path := fmt.Sprintf("/databases/%s/tables/%s/rows", q.databaseID, q.tableID)
	if qs := params.Encode(); qs != "" {
		path += "?" + qs
	}

	raw, err := q.client.call("GET", path, nil)
	if err != nil {
		return nil, err
	}

	result := &QueryResult{}
	if t, ok := raw["total"]; ok {
		switch v := t.(type) {
		case float64:
			result.Total = int(v)
		case int:
			result.Total = v
		}
	}
	if rows, ok := raw["rows"].([]interface{}); ok {
		for _, r := range rows {
			if row, ok := r.(map[string]interface{}); ok {
				result.Rows = append(result.Rows, row)
			}
		}
	}
	return result, nil
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

func (s *FunctionsService) Create(name, runtime string, opts ...map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"name":       name,
		"runtime":    runtime,
		"entrypoint": "index.handler",
		"timeout":    15,
		"vars":       map[string]string{},
		"source":     "",
		"cron":       "",
	}
	if len(opts) > 0 {
		for k, v := range opts[0] {
			body[k] = v
		}
	}
	return s.client.call("POST", "/functions", body)
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

func (s *TeamsService) GetMembership(teamID, membershipID string) (map[string]interface{}, error) {
	return s.client.call("GET", fmt.Sprintf("/teams/%s/memberships/%s", teamID, membershipID), nil)
}

func (s *TeamsService) UpdateMembership(teamID, membershipID string, roles []string) (map[string]interface{}, error) {
	return s.client.call("PATCH", fmt.Sprintf("/teams/%s/memberships/%s", teamID, membershipID), map[string]interface{}{
		"roles": roles,
	})
}

func (s *TeamsService) DeleteMembership(teamID, membershipID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", fmt.Sprintf("/teams/%s/memberships/%s", teamID, membershipID), nil)
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

func (s *MessagingService) SendSMS(to []string, body string) (map[string]interface{}, error) {
	return s.client.call("POST", "/messaging/sms", map[string]interface{}{
		"to":   to,
		"body": body,
	})
}

func (s *MessagingService) SendPush(to []string, title, body string, data map[string]string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"to":    to,
		"title": title,
		"body":  body,
	}
	if len(data) > 0 {
		payload["data"] = data
	}
	return s.client.call("POST", "/messaging/push", payload)
}

func (s *MessagingService) CreateTemplate(name, typ, subject, body string, variables []string) (map[string]interface{}, error) {
	if variables == nil {
		variables = []string{}
	}
	return s.client.call("POST", "/messaging/templates", map[string]interface{}{
		"templateId": "unique()",
		"name":       name,
		"type":       typ,
		"subject":    subject,
		"body":       body,
		"variables":  variables,
	})
}

func (s *MessagingService) ListTemplates() (map[string]interface{}, error) {
	return s.client.call("GET", "/messaging/templates", nil)
}

func (s *MessagingService) GetTemplate(templateID string) (map[string]interface{}, error) {
	return s.client.call("GET", "/messaging/templates/"+templateID, nil)
}

func (s *MessagingService) UpdateTemplate(templateID, name, typ, subject, body string, variables []string) (map[string]interface{}, error) {
	if variables == nil {
		variables = []string{}
	}
	return s.client.call("PUT", "/messaging/templates/"+templateID, map[string]interface{}{
		"name":      name,
		"type":      typ,
		"subject":   subject,
		"body":      body,
		"variables": variables,
	})
}

func (s *MessagingService) DeleteTemplate(templateID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", "/messaging/templates/"+templateID, nil)
}

func (s *MessagingService) SendTemplate(templateID string, to []string, variables map[string]string) (map[string]interface{}, error) {
	if variables == nil {
		variables = map[string]string{}
	}
	return s.client.call("POST", "/messaging/templates/"+templateID+"/send", map[string]interface{}{
		"to":        to,
		"variables": variables,
	})
}

// ---------------------------------------------------------------------------
// Deploy
// ---------------------------------------------------------------------------

type DeployService struct{ client *Client }

// Deploy targets are the deployable unit. The API mounts them under
// /deploy/targets (there is no flat /deploy resource); triggering a deploy
// runs the target as an execution.

func (s *DeployService) Create(name, deployType string, config map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"name": name,
		"type": deployType,
	}
	for k, v := range config {
		body[k] = v
	}
	return s.client.call("POST", "/deploy/targets", body)
}

func (s *DeployService) List() (map[string]interface{}, error) {
	return s.client.call("GET", "/deploy/targets", nil)
}

func (s *DeployService) Get(targetID string) (map[string]interface{}, error) {
	return s.client.call("GET", "/deploy/targets/"+targetID, nil)
}

func (s *DeployService) Update(targetID string, data map[string]interface{}) (map[string]interface{}, error) {
	return s.client.call("PUT", "/deploy/targets/"+targetID, data)
}

func (s *DeployService) Delete(targetID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", "/deploy/targets/"+targetID, nil)
}

// Deploy triggers a deploy of the target and returns the created execution.
func (s *DeployService) Deploy(targetID string, options map[string]interface{}) (map[string]interface{}, error) {
	if options == nil {
		options = map[string]interface{}{}
	}
	return s.client.call("POST", "/deploy/targets/"+targetID+"/executions", options)
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

// ---------------------------------------------------------------------------
// Analytics
// ---------------------------------------------------------------------------

type AnalyticsService struct{ client *Client }

func (s *AnalyticsService) TrackEvent(event string, properties map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{"event": event}
	if properties != nil {
		body["properties"] = properties
	}
	return s.client.call("POST", "/analytics/events", body)
}

func (s *AnalyticsService) ListEvents(params map[string]string) (map[string]interface{}, error) {
	path := "/analytics/events"
	if len(params) > 0 {
		query := make([]string, 0, len(params))
		for key, value := range params {
			query = append(query, fmt.Sprintf("%s=%s", key, value))
		}
		path += "?" + strings.Join(query, "&")
	}
	return s.client.call("GET", path, nil)
}

func (s *AnalyticsService) GetStats(params map[string]string) (map[string]interface{}, error) {
	path := "/analytics/stats"
	if len(params) > 0 {
		query := make([]string, 0, len(params))
		for key, value := range params {
			query = append(query, fmt.Sprintf("%s=%s", key, value))
		}
		path += "?" + strings.Join(query, "&")
	}
	return s.client.call("GET", path, nil)
}

func (s *AnalyticsService) GetRealtimeCount() (map[string]interface{}, error) {
	return s.client.call("GET", "/analytics/realtime", nil)
}

// GetOverview returns events, active users, request latency and average uptime
// for the last 24 hours.
func (s *AnalyticsService) GetOverview() (map[string]interface{}, error) {
	return s.client.call("GET", "/analytics/overview", nil)
}

// GetPerformance returns per-route request latency measured by the platform.
func (s *AnalyticsService) GetPerformance() (map[string]interface{}, error) {
	return s.client.call("GET", "/analytics/performance", nil)
}

// ── Uptime monitors ──────────────────────────────────────────────────────────

func (s *AnalyticsService) ListMonitors() (map[string]interface{}, error) {
	return s.client.call("GET", "/analytics/uptime", nil)
}

func (s *AnalyticsService) CreateMonitor(name, monitorURL string, opts map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{"name": name, "url": monitorURL}
	for k, v := range opts {
		body[k] = v
	}
	return s.client.call("POST", "/analytics/uptime", body)
}

func (s *AnalyticsService) DeleteMonitor(monitorID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", "/analytics/uptime/"+monitorID, nil)
}

// ── Cron monitors ────────────────────────────────────────────────────────────

func (s *AnalyticsService) ListCronMonitors() (map[string]interface{}, error) {
	return s.client.call("GET", "/analytics/crons", nil)
}

func (s *AnalyticsService) CreateCronMonitor(name, schedule string, opts map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{"name": name, "schedule": schedule}
	for k, v := range opts {
		body[k] = v
	}
	return s.client.call("POST", "/analytics/crons", body)
}

func (s *AnalyticsService) DeleteCronMonitor(monitorID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", "/analytics/crons/"+monitorID, nil)
}

// CronCheckin reports a run of a scheduled job. A monitor that hears nothing
// within its grace period is marked missed.
func (s *AnalyticsService) CronCheckin(monitorID string, opts map[string]interface{}) (map[string]interface{}, error) {
	return s.client.call("POST", fmt.Sprintf("/analytics/crons/%s/checkin", monitorID), opts)
}

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

type SearchService struct{ client *Client }

func (s *SearchService) CreateIndex(indexID string, opts map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{"indexId": indexID}
	for key, value := range opts {
		body[key] = value
	}
	return s.client.call("POST", "/search/indexes", body)
}

func (s *SearchService) ListIndexes() (map[string]interface{}, error) {
	return s.client.call("GET", "/search/indexes", nil)
}

func (s *SearchService) GetIndex(indexID string) (map[string]interface{}, error) {
	return s.client.call("GET", "/search/indexes/"+indexID, nil)
}

func (s *SearchService) DeleteIndex(indexID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", "/search/indexes/"+indexID, nil)
}

func (s *SearchService) IndexDocument(indexID, documentID string, data map[string]interface{}) (map[string]interface{}, error) {
	return s.client.call("POST", fmt.Sprintf("/search/indexes/%s/documents", indexID), map[string]interface{}{
		"documentId": documentID,
		"data":       data,
	})
}

func (s *SearchService) Query(indexID, query string, opts map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{"query": query}
	for key, value := range opts {
		body[key] = value
	}
	return s.client.call("POST", fmt.Sprintf("/search/indexes/%s/search", indexID), body)
}

func (s *SearchService) DeleteDocument(indexID, documentID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", fmt.Sprintf("/search/indexes/%s/documents/%s", indexID, documentID), nil)
}

// ---------------------------------------------------------------------------
// Vectors
// ---------------------------------------------------------------------------

type VectorsService struct{ client *Client }

func (s *VectorsService) CreateIndex(indexID string, dimensions int, opts map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"indexId":    indexID,
		"dimensions": dimensions,
	}
	for key, value := range opts {
		body[key] = value
	}
	return s.client.call("POST", "/vectors/indexes", body)
}

func (s *VectorsService) ListIndexes() (map[string]interface{}, error) {
	return s.client.call("GET", "/vectors/indexes", nil)
}

func (s *VectorsService) GetIndex(indexID string) (map[string]interface{}, error) {
	return s.client.call("GET", "/vectors/indexes/"+indexID, nil)
}

func (s *VectorsService) DeleteIndex(indexID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", "/vectors/indexes/"+indexID, nil)
}

func (s *VectorsService) Upsert(indexID string, vectors []map[string]interface{}) (map[string]interface{}, error) {
	return s.client.call("POST", fmt.Sprintf("/vectors/indexes/%s/vectors", indexID), map[string]interface{}{
		"vectors": vectors,
	})
}

func (s *VectorsService) Query(indexID string, vector []float64, opts map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{"vector": vector}
	for key, value := range opts {
		body[key] = value
	}
	return s.client.call("POST", fmt.Sprintf("/vectors/indexes/%s/query", indexID), body)
}

func (s *VectorsService) DeleteVectors(indexID string, ids []string) (map[string]interface{}, error) {
	return s.client.call("POST", fmt.Sprintf("/vectors/indexes/%s/delete", indexID), map[string]interface{}{
		"ids": ids,
	})
}

// ---------------------------------------------------------------------------
// Edge
// ---------------------------------------------------------------------------

type EdgeService struct{ client *Client }

func (s *EdgeService) Create(name, code string, opts map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"name": name,
		"code": code,
	}
	for key, value := range opts {
		body[key] = value
	}
	return s.client.call("POST", "/edge/functions", body)
}

func (s *EdgeService) List() (map[string]interface{}, error) {
	return s.client.call("GET", "/edge/functions", nil)
}

func (s *EdgeService) Get(functionID string) (map[string]interface{}, error) {
	return s.client.call("GET", "/edge/functions/"+functionID, nil)
}

func (s *EdgeService) Update(functionID string, opts map[string]interface{}) (map[string]interface{}, error) {
	return s.client.call("PUT", "/edge/functions/"+functionID, opts)
}

func (s *EdgeService) Delete(functionID string) (map[string]interface{}, error) {
	return s.client.call("DELETE", "/edge/functions/"+functionID, nil)
}

func (s *EdgeService) Invoke(functionID string, data map[string]interface{}) (map[string]interface{}, error) {
	if data == nil {
		data = map[string]interface{}{}
	}
	return s.client.call("POST", fmt.Sprintf("/edge/functions/%s/invoke", functionID), data)
}

func (s *EdgeService) ListExecutions(functionID string) (map[string]interface{}, error) {
	return s.client.call("GET", fmt.Sprintf("/edge/functions/%s/executions", functionID), nil)
}

// ---------------------------------------------------------------------------
// Regions
// ---------------------------------------------------------------------------

type RegionsService struct{ client *Client }

func (s *RegionsService) List() (map[string]interface{}, error) {
	return s.client.call("GET", "/regions", nil)
}

func (s *RegionsService) Get(regionID string) (map[string]interface{}, error) {
	return s.client.call("GET", "/regions/"+regionID, nil)
}

func (s *RegionsService) GetActive() (map[string]interface{}, error) {
	return s.client.call("GET", "/regions/active", nil)
}

func (s *RegionsService) SetActive(regionID string) (map[string]interface{}, error) {
	return s.client.call("PUT", "/regions/active", map[string]interface{}{"regionId": regionID})
}

func (s *RegionsService) GetHealth(regionID string) (map[string]interface{}, error) {
	return s.client.call("GET", fmt.Sprintf("/regions/%s/health", regionID), nil)
}
