// Package transfer implements Applad's data-migration engine: importing a
// project's users, databases, storage and functions from another platform
// (another Applad instance, Appwrite, Supabase, NHost, Firebase) into an Applad
// project. It follows an adapter pattern: a Source reads a platform and emits a
// normalized Resource stream, an orchestrator (Transfer) hands each Resource to
// a Destination, and the Destination writes it into Applad. Sources and
// Destinations never know about each other; they only share the Resource model.
package transfer

import "context"

// Group is a family of resources migrated together and selectable independently.
type Group string

const (
	GroupAuth      Group = "auth"
	GroupDatabases Group = "databases"
	GroupStorage   Group = "storage"
	GroupFunctions Group = "functions"
)

// Status is the outcome of importing a single resource.
const (
	StatusPending = "pending"
	StatusDone    = "done"
	StatusWarning = "warning"
	StatusError   = "error"
	StatusSkip    = "skip"
)

// Resource is one normalized item flowing from a Source to a Destination.
// Concrete types implement it; a Destination type-switches to import each.
type Resource interface {
	Group() Group
	Kind() string
	SourceID() string
}

// Emit is how a Source hands resources to the orchestrator. Resources within a
// single call and across calls must be emitted in dependency order (a parent
// before its children: database before table before column/index before row;
// bucket before file; user before team before membership).
type Emit func(ctx context.Context, res []Resource) error

// Result is what a Destination returns for one imported resource.
type Result struct {
	DestID  string
	Status  string // StatusDone | StatusWarning | StatusError | StatusSkip
	Message string
}

// --- Auth group ---------------------------------------------------------------

type User struct {
	ID             string
	Email          string
	Phone          string
	Name           string
	EmailVerified  bool
	PhoneVerified  bool
	PasswordHash   string         // empty for OAuth-only accounts
	PasswordAlgo   string         // "" defaults to bcrypt
	PasswordParams map[string]any // algorithm-specific verify params
	Labels         []string
	Prefs          map[string]any
}

func (User) Group() Group       { return GroupAuth }
func (User) Kind() string       { return "user" }
func (u User) SourceID() string { return u.ID }

type Team struct {
	ID   string
	Name string
}

func (Team) Group() Group       { return GroupAuth }
func (Team) Kind() string       { return "team" }
func (t Team) SourceID() string { return t.ID }

type Membership struct {
	ID     string
	TeamID string
	UserID string
	Roles  []string
}

func (Membership) Group() Group       { return GroupAuth }
func (Membership) Kind() string       { return "membership" }
func (m Membership) SourceID() string { return m.ID }

// --- Databases group ----------------------------------------------------------

type Database struct {
	ID   string
	Name string
}

func (Database) Group() Group       { return GroupDatabases }
func (Database) Kind() string       { return "database" }
func (d Database) SourceID() string { return d.ID }

type Table struct {
	DatabaseID  string
	ID          string
	Name        string
	Permissions []string
	RowSecurity bool
}

func (Table) Group() Group       { return GroupDatabases }
func (Table) Kind() string       { return "table" }
func (t Table) SourceID() string { return t.DatabaseID + "/" + t.ID }

type Column struct {
	DatabaseID string
	TableID    string
	Key        string
	Type       string // string|integer|double|boolean|datetime|email|url|enum|json ...
	Required   bool
	Array      bool
	Default    any
	Options    map[string]any
}

func (Column) Group() Group       { return GroupDatabases }
func (Column) Kind() string       { return "column" }
func (c Column) SourceID() string { return c.TableID + "/" + c.Key }

type Index struct {
	DatabaseID string
	TableID    string
	Key        string
	Type       string // key|unique|fulltext
	Columns    []string
	Orders     []string
}

func (Index) Group() Group       { return GroupDatabases }
func (Index) Kind() string       { return "index" }
func (i Index) SourceID() string { return i.TableID + "/" + i.Key }

type Row struct {
	DatabaseID  string
	TableID     string
	ID          string
	Data        map[string]any
	Permissions []string
}

func (Row) Group() Group       { return GroupDatabases }
func (Row) Kind() string       { return "row" }
func (r Row) SourceID() string { return r.TableID + "/" + r.ID }

// --- Storage group ------------------------------------------------------------

type Bucket struct {
	ID               string
	Name             string
	Permissions      []string
	FileSizeLimit    int64
	AllowedMimeTypes []string
	FileSecurity     bool
	Encryption       bool
	Antivirus        bool
}

func (Bucket) Group() Group       { return GroupStorage }
func (Bucket) Kind() string       { return "bucket" }
func (b Bucket) SourceID() string { return b.ID }

type File struct {
	BucketID    string
	ID          string
	Name        string
	MimeType    string
	Content     []byte
	Permissions []string
}

func (File) Group() Group       { return GroupStorage }
func (File) Kind() string       { return "file" }
func (f File) SourceID() string { return f.BucketID + "/" + f.ID }

// --- Functions group ----------------------------------------------------------

type Function struct {
	ID      string
	Name    string
	Runtime string
	Entry   string
	Source  []byte // packaged source, if the platform exposes it
	Env     map[string]string
}

func (Function) Group() Group       { return GroupFunctions }
func (Function) Kind() string       { return "function" }
func (f Function) SourceID() string { return f.ID }
