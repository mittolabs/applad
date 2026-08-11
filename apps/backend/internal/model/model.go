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
	ID           string     `json:"$id"`
	ProjectID    string     `json:"projectId"`
	Name         string     `json:"name"`
	Secret       string     `json:"secret,omitempty"`       // only returned on creation
	SecretPrefix string     `json:"secretPrefix,omitempty"` // always returned as hint
	Scopes       []string   `json:"scopes"`
	ExpiresAt    *time.Time `json:"expire"`
	CreatedAt    time.Time  `json:"$createdAt"`
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
	// Secret is the session JWT, returned ONLY when a session is first created so
	// a non-browser client (mobile, a server) can authenticate by header instead
	// of relying on the Set-Cookie that only a browser honours. It is never
	// populated when listing or fetching a session, hence omitempty.
	Secret string `json:"secret,omitempty"`
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
	// Secret is the one-time invite token, returned ONLY when a membership is
	// created so the inviter can hand out a join link (a self-hosted instance
	// often has no SMTP to send one). It is never populated when listing
	// memberships, hence omitempty.
	Secret string `json:"secret,omitempty"`
}

// Database represents an Applad database.
type Database struct {
	ID        string    `json:"$id"`
	CreatedAt time.Time `json:"$createdAt"`
	UpdatedAt time.Time `json:"$updatedAt"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
}

// Table represents a database table.
type Table struct {
	ID          string    `json:"$id"`
	CreatedAt   time.Time `json:"$createdAt"`
	UpdatedAt   time.Time `json:"$updatedAt"`
	DatabaseID  string    `json:"databaseId"`
	Name        string    `json:"name"`
	Enabled     bool      `json:"enabled"`
	RowSecurity bool      `json:"rowSecurity"`
	// ContentEnabled turns on editorial behaviour: draft/publish, slug, locale
	// and version history on this table's rows.
	ContentEnabled bool     `json:"contentEnabled"`
	Permissions    []string `json:"$permissions"`
	Columns        []Column `json:"columns"`
	Indexes        []Index  `json:"indexes"`
}

// ColumnValidation holds application-level validation rules for a column.
// Rules are enforced when creating or updating rows via the API.
type ColumnValidation struct {
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	MinLength *int     `json:"minLength,omitempty"`
	MaxLength *int     `json:"maxLength,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Message   string   `json:"message,omitempty"` // custom error message shown on any violation
}

// Column represents a table column.
type Column struct {
	Key         string                 `json:"key"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	Required    bool                   `json:"required"`
	Array       bool                   `json:"array"`
	Default     interface{}            `json:"default"`
	Options     map[string]interface{} `json:"options,omitempty"`
	Validation  *ColumnValidation      `json:"validation,omitempty"`
	Permissions []string               `json:"$permissions"`
	// Encrypted marks this column for field-level encryption at rest: values
	// are stored as opaque ciphertext (see internal/dek), decrypted only for
	// authorized reads, and cannot be filtered, sorted, or searched on.
	// Mutually exclusive with Array (see internal/databases.CreateColumn).
	Encrypted bool `json:"encrypted"`
}

// Index represents a table index.
type Index struct {
	Key     string   `json:"key"`
	Type    string   `json:"type"`
	Status  string   `json:"status"`
	Columns []string `json:"columns"`
	Orders  []string `json:"orders"`
}

// Row represents a database row.
type Row struct {
	ID          string                 `json:"$id"`
	TableID     string                 `json:"$tableId"`
	DatabaseID  string                 `json:"$databaseId"`
	CreatedAt   time.Time              `json:"$createdAt"`
	UpdatedAt   time.Time              `json:"$updatedAt"`
	Permissions []string               `json:"$permissions"`
	Data        map[string]interface{} `json:"-"`
}

// MarshalJSON merges row Data fields into the top-level JSON object.
func (d Row) MarshalJSON() ([]byte, error) {
	type Alias Row
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
	ID                    string    `json:"$id"`
	CreatedAt             time.Time `json:"$createdAt"`
	UpdatedAt             time.Time `json:"$updatedAt"`
	Name                  string    `json:"name"`
	Enabled               bool      `json:"enabled"`
	Permissions           []string  `json:"$permissions"`
	FileSizeLimit         int64     `json:"maximumFileSize"`
	AllowedFileExtensions []string  `json:"allowedFileExtensions"`
	Compression           string    `json:"compression"`
	Encryption            bool      `json:"encryption"`
	Antivirus             bool      `json:"antivirus"`
	FileSecurity          bool      `json:"fileSecurity"`
	ImageTransformations  bool      `json:"imageTransformations"`
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

// ── Chat ─────────────────────────────────────────────────────────────────────
//
// E2E-encrypted chat: the server stores and relays ciphertext, public keys,
// and routing metadata only. None of these structs ever carry plaintext
// content or private key material — see internal/chat.

// Device is a chat client install (a phone app, a browser tab, a desktop
// app) identified by its public identity/prekey bundle. Not the same as a
// Session: a device outlives many logins, and a user may hold several at
// once (multi-device). The push token is write-only and never echoed back.
type Device struct {
	ID              string     `json:"$id"`
	CreatedAt       time.Time  `json:"$createdAt"`
	UpdatedAt       time.Time  `json:"$updatedAt"`
	UserID          string     `json:"userId"`
	Name            string     `json:"name"`
	RegistrationID  int        `json:"registrationId"`
	IdentityKey     string     `json:"identityKey"`
	SignedPrekeyID  int        `json:"signedPrekeyId"`
	SignedPrekey    string     `json:"signedPrekey"`
	SignedPrekeySig string     `json:"signedPrekeySig"`
	PushProvider    string     `json:"pushProvider,omitempty"`
	Status          string     `json:"status"`
	LastSeenAt      *time.Time `json:"lastSeenAt,omitempty"`
}

// DeviceRef is the minimal, public information about another user's device:
// enough to know it exists and target it, not enough to start a session with
// it (that needs a PrekeyBundle, fetched per-device).
type DeviceRef struct {
	ID             string `json:"$id"`
	RegistrationID int    `json:"registrationId"`
	Status         string `json:"status"`
}

// PrekeyBundle is what another device fetches to start an X3DH session with
// this device — public keys only. OneTimePrekeyID/OneTimePrekey are present
// only if the pool wasn't empty; the key is consumed atomically on fetch.
type PrekeyBundle struct {
	DeviceID         string `json:"deviceId"`
	RegistrationID   int    `json:"registrationId"`
	IdentityKey      string `json:"identityKey"`
	SignedPrekeyID   int    `json:"signedPrekeyId"`
	SignedPrekey     string `json:"signedPrekey"`
	SignedPrekeySig  string `json:"signedPrekeySig"`
	OneTimePrekeyID  *int   `json:"oneTimePrekeyId,omitempty"`
	OneTimePrekey    string `json:"oneTimePrekey,omitempty"`
	PrekeysRemaining int    `json:"prekeysRemaining"`
}

// Conversation is a direct (1:1) or group chat. Only metadata is visible to
// the server — message content is end-to-end encrypted and never touches
// this struct. Title stays in the clear (matches WhatsApp's own model).
type Conversation struct {
	ID        string    `json:"$id"`
	CreatedAt time.Time `json:"$createdAt"`
	UpdatedAt time.Time `json:"$updatedAt"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	CreatedBy string    `json:"createdBy"`
}

// ConversationMember is one user's membership in a Conversation.
type ConversationMember struct {
	ID             string     `json:"$id"`
	ConversationID string     `json:"conversationId"`
	UserID         string     `json:"userId"`
	Role           string     `json:"role"`
	JoinedAt       time.Time  `json:"joinedAt"`
	RemovedAt      *time.Time `json:"removedAt,omitempty"`
}

// MessageTarget is one recipient device's ciphertext for a prekey/whisper
// (Double Ratchet, per-device) send — the client already resolved the
// recipient's device list and encrypted once per device.
type MessageTarget struct {
	DeviceID   string `json:"deviceId"`
	Ciphertext string `json:"ciphertext"`
}

// Message is an opaque ciphertext envelope routed through a conversation.
// Ciphertext is set directly for a sender_key (shared-across-recipients)
// envelope; a prekey/whisper envelope leaves it empty here and carries
// per-recipient-device ciphertext in Targets instead.
type Message struct {
	ID              string          `json:"$id"`
	CreatedAt       time.Time       `json:"$createdAt"`
	ClientMessageID string          `json:"clientMessageId"`
	ConversationID  string          `json:"conversationId"`
	SenderUserID    string          `json:"senderUserId"`
	SenderDeviceID  string          `json:"senderDeviceId"`
	EnvelopeType    string          `json:"envelopeType"`
	Ciphertext      string          `json:"ciphertext,omitempty"`
	Seq             int64           `json:"seq"`
	Targets         []MessageTarget `json:"targets,omitempty"`
}

// MessageReceipt tracks one recipient device's delivery/read status for a message.
type MessageReceipt struct {
	MessageID         string     `json:"messageId"`
	RecipientDeviceID string     `json:"deviceId"`
	Status            string     `json:"status"`
	DeliveredAt       *time.Time `json:"deliveredAt,omitempty"`
	ReadAt            *time.Time `json:"readAt,omitempty"`
}
