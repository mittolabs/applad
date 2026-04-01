package model

import (
	"encoding/json"
	"time"
)

// Project represents an Applad project.
type Project struct {
	ID          string    `json:"$id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"$createdAt"`
	UpdatedAt   time.Time `json:"$updatedAt"`
}

// APIKey represents a project API key.
type APIKey struct {
	ID        string     `json:"$id"`
	ProjectID string     `json:"projectId"`
	Name      string     `json:"name"`
	Secret    string     `json:"secret,omitempty"` // only returned on creation
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expire"`
	CreatedAt time.Time  `json:"$createdAt"`
}

// User represents an Applad account.
type User struct {
	ID            string                 `json:"$id"`
	CreatedAt     time.Time              `json:"$createdAt"`
	UpdatedAt     time.Time              `json:"$updatedAt"`
	Name          string                 `json:"name"`
	Email         string                 `json:"email"`
	Phone         string                 `json:"phone"`
	EmailVerified bool                   `json:"emailVerification"`
	PhoneVerified bool                   `json:"phoneVerification"`
	Status        bool                   `json:"status"`
	Labels        []string               `json:"labels"`
	Prefs         map[string]interface{} `json:"prefs"`
	AccessedAt    time.Time              `json:"accessedAt"`
}

// Session represents an authenticated session.
type Session struct {
	ID        string    `json:"$id"`
	CreatedAt time.Time `json:"$createdAt"`
	UserID    string    `json:"userId"`
	Expire    time.Time `json:"expire"`
	Provider  string    `json:"provider"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"userAgent"`
	Current   bool      `json:"current"`
}

// Team represents a team of users.
type Team struct {
	ID        string                 `json:"$id"`
	CreatedAt time.Time              `json:"$createdAt"`
	UpdatedAt time.Time              `json:"$updatedAt"`
	Name      string                 `json:"name"`
	Total     int                    `json:"total"`
	Prefs     map[string]interface{} `json:"prefs"`
}

// Membership represents a user's membership in a team.
type Membership struct {
	ID        string    `json:"$id"`
	CreatedAt time.Time `json:"$createdAt"`
	TeamID    string    `json:"teamId"`
	TeamName  string    `json:"teamName"`
	UserID    string    `json:"userId"`
	UserName  string    `json:"userName"`
	UserEmail string    `json:"userEmail"`
	Roles     []string  `json:"roles"`
	Invited   bool      `json:"invited"`
	Joined    bool      `json:"joined"`
	Confirm   bool      `json:"confirm"`
}

// Database represents an Applad database.
type Database struct {
	ID        string    `json:"$id"`
	CreatedAt time.Time `json:"$createdAt"`
	UpdatedAt time.Time `json:"$updatedAt"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
}

// Collection represents a document collection.
type Collection struct {
	ID               string      `json:"$id"`
	CreatedAt        time.Time   `json:"$createdAt"`
	UpdatedAt        time.Time   `json:"$updatedAt"`
	DatabaseID       string      `json:"databaseId"`
	Name             string      `json:"name"`
	Enabled          bool        `json:"enabled"`
	DocumentSecurity bool        `json:"documentSecurity"`
	Permissions      []string    `json:"$permissions"`
	Attributes       []Attribute `json:"attributes"`
	Indexes          []Index     `json:"indexes"`
}

// Attribute represents a collection attribute.
type Attribute struct {
	Key      string      `json:"key"`
	Type     string      `json:"type"`
	Status   string      `json:"status"`
	Required bool        `json:"required"`
	Array    bool        `json:"array"`
	Default  interface{} `json:"default"`
}

// Index represents a collection index.
type Index struct {
	Key        string   `json:"key"`
	Type       string   `json:"type"`
	Status     string   `json:"status"`
	Attributes []string `json:"attributes"`
	Orders     []string `json:"orders"`
}

// Document represents a database document.
type Document struct {
	ID           string                 `json:"$id"`
	CollectionID string                 `json:"$collectionId"`
	DatabaseID   string                 `json:"$databaseId"`
	CreatedAt    time.Time              `json:"$createdAt"`
	UpdatedAt    time.Time              `json:"$updatedAt"`
	Permissions  []string               `json:"$permissions"`
	Data         map[string]interface{} `json:"-"`
}

// MarshalJSON merges document Data fields into the top-level JSON object.
func (d Document) MarshalJSON() ([]byte, error) {
	type Alias Document
	base, err := json.Marshal(struct {
		Alias
	}{Alias: Alias(d)})
	if err != nil {
		return nil, err
	}

	if len(d.Data) == 0 {
		return base, nil
	}

	// Merge data fields into base object
	var baseMap map[string]interface{}
	if err := json.Unmarshal(base, &baseMap); err != nil {
		return nil, err
	}
	for k, v := range d.Data {
		baseMap[k] = v
	}
	return json.Marshal(baseMap)
}

// Bucket represents a storage bucket.
type Bucket struct {
	ID                    string     `json:"$id"`
	CreatedAt             time.Time  `json:"$createdAt"`
	UpdatedAt             time.Time  `json:"$updatedAt"`
	Name                  string     `json:"name"`
	Enabled               bool       `json:"enabled"`
	Permissions           []string   `json:"$permissions"`
	FileSizeLimit         int64      `json:"maximumFileSize"`
	AllowedFileExtensions []string   `json:"allowedFileExtensions"`
	Compression           string     `json:"compression"`
	Encryption            bool       `json:"encryption"`
	Antivirus             bool       `json:"antivirus"`
}

// File represents a stored file's metadata.
type File struct {
	ID           string    `json:"$id"`
	BucketID     string    `json:"bucketId"`
	CreatedAt    time.Time `json:"$createdAt"`
	UpdatedAt    time.Time `json:"$updatedAt"`
	Name         string    `json:"name"`
	Signature    string    `json:"signature"`
	MimeType     string    `json:"mimeType"`
	SizeOriginal int64     `json:"sizeOriginal"`
	Permissions  []string  `json:"$permissions"`
}

// AppwriteError is the standard error response shape.
type AppwriteError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Type    string `json:"type"`
	Version string `json:"version"`
}
