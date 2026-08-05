package transfer

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"

	"github.com/mittolabs/applad/internal/auth"
	"github.com/mittolabs/applad/internal/databases"
	"github.com/mittolabs/applad/internal/storage"
	"github.com/mittolabs/applad/internal/uid"
)

// maxImportFileBytes bounds the size of a single imported file so one large
// object cannot exhaust the worker's memory. Larger files are skipped.
const maxImportFileBytes = 100 << 20 // 100 MiB

// appladDest writes normalized resources into an Applad project using the
// existing service layer, so all native validation, schema DDL, RLS wiring and
// storage drivers are reused. Source resource IDs are sanitized to Applad's ID
// rules and remembered so child resources (columns, rows, files) resolve to the
// destination IDs their parents were created with.
type appladDest struct {
	projectID string
	auth      *auth.Service
	dbs       *databases.Service
	stg       *storage.Service

	dbIDs     map[string]string   // source dbID -> dest dbID
	tableRefs map[string]tableRef // source "dbID/tableID" -> dest ids
	bucketIDs map[string]string   // source bucketID -> dest bucketID
}

type tableRef struct{ db, table string }

// NewAppladDestination builds an Applad destination bound to one project.
func NewAppladDestination(projectID string, a *auth.Service, d *databases.Service, s *storage.Service) Destination {
	return &appladDest{
		projectID: projectID,
		auth:      a,
		dbs:       d,
		stg:       s,
		dbIDs:     map[string]string{},
		tableRefs: map[string]tableRef{},
		bucketIDs: map[string]string{},
	}
}

func (d *appladDest) Name() string { return "applad" }
func (d *appladDest) Close() error { return nil }

func (d *appladDest) Import(ctx context.Context, res Resource) Result {
	switch r := res.(type) {
	case User:
		return d.importUser(ctx, r)
	case Database:
		return d.importDatabase(ctx, r)
	case Table:
		return d.importTable(ctx, r)
	case Column:
		return d.importColumn(ctx, r)
	case Index:
		return d.importIndex(ctx, r)
	case Row:
		return d.importRow(ctx, r)
	case Bucket:
		return d.importBucket(ctx, r)
	case File:
		return d.importFile(ctx, r)
	default:
		return Result{Status: StatusSkip, Message: fmt.Sprintf("unsupported resource kind %q", res.Kind())}
	}
}

func (d *appladDest) importUser(ctx context.Context, u User) Result {
	// A stable, source-derived ID makes the import idempotent (see ImportUser):
	// re-running updates the same row instead of duplicating email-less accounts.
	seed := u.ID
	if seed == "" {
		seed = u.Email + "|" + u.Phone
	}
	id, err := d.auth.ImportUser(ctx, d.projectID, auth.ImportedUser{
		ID:             safeID("u", seed),
		Email:          u.Email,
		Phone:          u.Phone,
		Name:           u.Name,
		PasswordHash:   u.PasswordHash,
		PasswordAlgo:   u.PasswordAlgo,
		PasswordParams: u.PasswordParams,
		EmailVerified:  u.EmailVerified,
		PhoneVerified:  u.PhoneVerified,
		Labels:         u.Labels,
		Prefs:          u.Prefs,
	})
	if err != nil {
		if isExists(err) {
			// Another account already owns this email; skip rather than fail.
			return Result{Status: StatusSkip, Message: "email already in use"}
		}
		return errResult(err)
	}
	return Result{DestID: id, Status: StatusDone}
}

func (d *appladDest) importDatabase(ctx context.Context, db Database) Result {
	destID := safeID("db", db.ID)
	_, err := d.dbs.CreateDatabase(ctx, d.projectID, destID, orName(db.Name, db.ID))
	if err != nil && !isExists(err) {
		return errResult(err)
	}
	d.dbIDs[db.ID] = destID
	return Result{DestID: destID, Status: StatusDone}
}

func (d *appladDest) importTable(ctx context.Context, t Table) Result {
	destDB, ok := d.dbIDs[t.DatabaseID]
	if !ok {
		return Result{Status: StatusError, Message: "parent database not imported: " + t.DatabaseID}
	}
	destTable := safeID("tbl", t.ID)
	_, err := d.dbs.CreateTable(ctx, d.projectID, destDB, destTable, orName(t.Name, t.ID), t.Permissions, t.RowSecurity)
	if err != nil && !isExists(err) {
		return errResult(err)
	}
	d.tableRefs[t.DatabaseID+"/"+t.ID] = tableRef{db: destDB, table: destTable}
	return Result{DestID: destTable, Status: StatusDone}
}

func (d *appladDest) importColumn(ctx context.Context, c Column) Result {
	ref, ok := d.tableRefs[c.DatabaseID+"/"+c.TableID]
	if !ok {
		return Result{Status: StatusError, Message: "parent table not imported: " + c.TableID}
	}
	_, err := d.dbs.CreateColumn(ctx, d.projectID, ref.table, c.Key, mapColumnType(c.Type),
		c.Required, c.Array, c.Default, c.Options, nil)
	if err != nil && !isExists(err) {
		return errResult(err)
	}
	return Result{DestID: c.Key, Status: StatusDone}
}

func (d *appladDest) importIndex(ctx context.Context, i Index) Result {
	ref, ok := d.tableRefs[i.DatabaseID+"/"+i.TableID]
	if !ok {
		return Result{Status: StatusError, Message: "parent table not imported: " + i.TableID}
	}
	_, err := d.dbs.CreateIndex(ctx, d.projectID, ref.table, i.Key, orName(i.Type, "key"), i.Columns, i.Orders)
	if err != nil && !isExists(err) {
		// An index that cannot be created (e.g. an unsupported type) is a warning,
		// not a hard failure: the data still imports.
		return Result{Status: StatusWarning, Message: err.Error()}
	}
	return Result{DestID: i.Key, Status: StatusDone}
}

func (d *appladDest) importRow(ctx context.Context, r Row) Result {
	ref, ok := d.tableRefs[r.DatabaseID+"/"+r.TableID]
	if !ok {
		return Result{Status: StatusError, Message: "parent table not imported: " + r.TableID}
	}
	rowID := safeID("row", r.ID)
	if r.ID == "" {
		rowID = uid.New("") // no source id to be idempotent on; generate a fresh one
	}
	_, err := d.dbs.CreateRow(ctx, d.projectID, ref.db, ref.table, rowID, r.Data, r.Permissions)
	if err != nil && !isExists(err) {
		return errResult(err)
	}
	return Result{DestID: rowID, Status: StatusDone}
}

func (d *appladDest) importBucket(ctx context.Context, b Bucket) Result {
	destID := safeID("bkt", b.ID)
	_, err := d.stg.CreateBucket(ctx, d.projectID, destID, orName(b.Name, b.ID), b.Permissions,
		b.FileSizeLimit, b.AllowedMimeTypes, "", b.Encryption, b.Antivirus, b.FileSecurity, false)
	if err != nil && !isExists(err) {
		return errResult(err)
	}
	d.bucketIDs[b.ID] = destID
	return Result{DestID: destID, Status: StatusDone}
}

func (d *appladDest) importFile(ctx context.Context, f File) Result {
	destBucket, ok := d.bucketIDs[f.BucketID]
	if !ok {
		return Result{Status: StatusError, Message: "parent bucket not imported: " + f.BucketID}
	}
	if int64(len(f.Content)) > maxImportFileBytes {
		return Result{Status: StatusSkip, Message: fmt.Sprintf("file exceeds %d MiB import limit", maxImportFileBytes>>20)}
	}
	fileID := safeID("file", f.ID)
	if f.ID == "" {
		fileID = uid.New("")
	}
	// Stream from the buffer with a hard byte cap rather than handing the whole
	// slice to the buffered path.
	_, err := d.stg.CreateFileStream(ctx, d.projectID, destBucket, fileID, orName(f.Name, f.ID),
		bytes.NewReader(f.Content), maxImportFileBytes, f.MimeType, f.Permissions)
	if err != nil && !isExists(err) {
		return errResult(err)
	}
	return Result{DestID: fileID, Status: StatusDone}
}

// --- helpers ------------------------------------------------------------------

var idAllowed = regexp.MustCompile(`[^A-Za-z0-9_]+`)

// sanitizeID coerces a foreign resource ID into Applad's identifier rules
// ([A-Za-z0-9_], since schema segments collapse '-' to '_') and bounds its
// length. Empty input yields "".
func sanitizeID(s string) string {
	s = idAllowed.ReplaceAllString(strings.TrimSpace(s), "_")
	s = strings.Trim(s, "_")
	if len(s) > 36 {
		s = s[:36]
	}
	return s
}

// safeID produces a stable, collision-resistant Applad ID from a foreign source
// ID. A source ID that is already a clean, short identifier passes through
// unchanged (readable, and stable across re-runs for idempotency). Anything that
// had to be rewritten (illegal characters, too long, empty) gets a deterministic
// hash suffix so two distinct source IDs can never collapse to the same dest ID
// and silently merge. The same source always yields the same result, so a
// resumed migration re-resolves to the identical destination row.
func safeID(prefix, source string) string {
	source = strings.TrimSpace(source)
	clean := sanitizeID(source)
	if clean == source && source != "" && len(source) <= 32 {
		return clean
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(source))
	suffix := fmt.Sprintf("%08x", h.Sum32())
	base := clean
	if max := 32 - len(suffix) - 1; len(base) > max {
		base = base[:max]
	}
	if base == "" {
		base = prefix
		if base == "" {
			base = "id"
		}
	}
	return base + "_" + suffix
}

func orName(name, fallback string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return fallback
}

func isExists(err error) bool {
	if err == nil {
		return false
	}
	e := strings.ToLower(err.Error())
	return strings.Contains(e, "exist") || strings.Contains(e, "duplicate") || strings.Contains(e, "already")
}

func errResult(err error) Result {
	return Result{Status: StatusError, Message: err.Error()}
}

// mapColumnType maps a source column type to the nearest Applad column type.
// Applad's own types pass through; unknowns fall back to string.
func mapColumnType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "string", "text", "varchar", "char":
		return "string"
	case "integer", "int", "int4", "int8", "bigint", "smallint", "serial":
		return "integer"
	case "double", "float", "float8", "numeric", "decimal", "real":
		return "double"
	case "boolean", "bool":
		return "boolean"
	case "datetime", "timestamp", "timestamptz", "date", "time":
		return "datetime"
	case "email":
		return "email"
	case "url":
		return "url"
	case "enum":
		return "enum"
	case "ip":
		return "ip"
	case "json", "jsonb", "object", "array":
		return "string"
	default:
		return "string"
	}
}
