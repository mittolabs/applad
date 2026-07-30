package realtime

import (
	"context"
	"strings"
	"time"
)

// Channel authorization for realtime subscriptions.
//
// A WebSocket connection carries a projectID (from the X-Applad-Project header
// or ?project= query, set by middleware.ProjectContext) and, when the caller
// presented a valid credential, an authenticated identity. A client may then
// ask to subscribe to arbitrary channel strings. Historically the hub honoured
// any such request, so anyone who learned another project's ID could subscribe
// to `databases.{thatProject}...` and receive its row changes — full rows,
// live, cross-tenant, with no credential. authorizeSubscribe closes that:
//
//  1. Project scoping: a channel that names a project must name THIS
//     connection's project. A deploy channel is tied to the project by
//     verifying the release belongs to it.
//  2. Authentication: row-data and resource channels require an authenticated
//     connection, not merely a project header (which is not a secret).
//  3. Per-row read filtering: for a table-scoped database channel, the
//     connection is authorized against the table's permissions, and — when the
//     table uses document (row) security — each event is delivered only if the
//     row's own _permissions grant the subscriber's roles read access.
//
// Broad-access connections (server API keys, console administrators) are
// project-scoped like everyone else but skip per-row filtering: they already
// hold project-wide read access through the REST API.

// ReadAuthorizer resolves, at subscribe time, whether a connection may read a
// database table's rows over realtime, and how per-row events must be filtered.
// It is implemented by the databases service, reusing the same table-level and
// document-security permission logic the REST row endpoints enforce.
type ReadAuthorizer interface {
	AuthorizeTableRead(ctx context.Context, projectID, databaseID, tableName, userID string) (TableReadDecision, error)
}

// TableReadDecision describes a connection's read access to one table.
type TableReadDecision struct {
	// AllowAll: a table-level read grant covers the caller; deliver every row.
	AllowAll bool
	// RowFiltered: the table uses document security and the caller has no
	// table-level grant; deliver only rows whose own _permissions grant read to
	// one of Roles.
	RowFiltered bool
	// Roles is the caller's fully-resolved role set (any, users, user:<id>,
	// team:<id>...), used to match a row's _permissions when RowFiltered.
	Roles []string
}

// ReleaseVerifier reports whether a deploy release belongs to a project. It is
// implemented by the deploy service and lets a deploy-log channel be tied to
// the subscribing connection's project.
type ReleaseVerifier interface {
	ReleaseBelongsToProject(ctx context.Context, releaseID, projectID string) (bool, error)
}

type channelKind int

const (
	kindUnknown channelKind = iota
	// kindDatabaseData: databases.{project}.{db}[.{table}] — live row data.
	kindDatabaseData
	// kindProjectData: projects.{project}.{service}.{resource} — resource data
	// (storage files, project-wide row feed).
	kindProjectData
	// kindDeploy: deploy.{releaseID} — build/deploy log stream.
	kindDeploy
)

type parsedChannel struct {
	kind       channelKind
	projectID  string
	databaseID string
	tableName  string
	releaseID  string
	service    string // kindProjectData: the {service} segment, e.g. "databases"
	resource   string // kindProjectData: the {resource} segment, e.g. "rows"
}

// parseChannel classifies a channel string and extracts its scoping segments.
// The formats are produced in hub.go (publishDatabaseNotification) and
// events.go (PublishResourceEvent).
func parseChannel(channel string) parsedChannel {
	parts := strings.Split(channel, ".")
	if len(parts) == 0 {
		return parsedChannel{kind: kindUnknown}
	}
	switch parts[0] {
	case "databases":
		// databases.{project}.{db}            (database-scoped, broad)
		// databases.{project}.{db}.{table}    (table-scoped, specific)
		pc := parsedChannel{kind: kindDatabaseData}
		if len(parts) >= 2 {
			pc.projectID = parts[1]
		}
		if len(parts) >= 3 {
			pc.databaseID = parts[2]
		}
		if len(parts) >= 4 {
			// A table name is an identifier; join defensively in case it ever
			// contains a dot so the project/db segments stay correct.
			pc.tableName = strings.Join(parts[3:], ".")
		}
		return pc
	case "projects":
		// projects.{project}.{service}.{resource}
		pc := parsedChannel{kind: kindProjectData}
		if len(parts) >= 2 {
			pc.projectID = parts[1]
		}
		if len(parts) >= 3 {
			pc.service = parts[2]
		}
		if len(parts) >= 4 {
			pc.resource = strings.Join(parts[3:], ".")
		}
		return pc
	case "deploy":
		// deploy.{releaseID}
		pc := parsedChannel{kind: kindDeploy}
		if len(parts) >= 2 {
			pc.releaseID = strings.Join(parts[1:], ".")
		}
		return pc
	}
	return parsedChannel{kind: kindUnknown}
}

// authDecision is the outcome of authorizing one subscribe request.
type authDecision struct {
	allowed bool
	code    string
	reason  string
	filter  *rowFilter // non-nil only for per-row filtered table channels
}

func allow(f *rowFilter) authDecision { return authDecision{allowed: true, filter: f} }

func deny(code, reason string) authDecision {
	return authDecision{allowed: false, code: code, reason: reason}
}

// authorizeSubscribe decides whether client c may subscribe to channel, and how
// events on it must be filtered. It may perform bounded database lookups (table
// permission resolution, release ownership) and so must be called WITHOUT the
// hub lock held.
func (h *Hub) authorizeSubscribe(c *Client, channel string) authDecision {
	pc := parseChannel(channel)
	switch pc.kind {
	case kindDatabaseData:
		if pc.projectID == "" || pc.projectID != c.projectID {
			return deny("realtime_forbidden_project", "Channel belongs to a different project.")
		}
		if !c.authenticated {
			return deny("realtime_unauthenticated", "Authentication is required to subscribe to data channels.")
		}
		// A table-less database channel (databases.{proj}.{db}, or the even
		// broader databases.{proj}) is an aggregate feed: it carries EVERY
		// table's row changes with no per-row filtering possible, so it must
		// not be readable by an ordinary end-user session. Only broad-access
		// connections (server API keys, console administrators) — which already
		// hold project-wide read access through the REST API — may subscribe.
		// An end-user session reads a specific table channel instead, which is
		// per-row filtered below.
		if pc.tableName == "" {
			if !c.broadAccess {
				return deny("realtime_forbidden_read", "The aggregate database feed requires a server API key.")
			}
			return allow(nil)
		}
		// Server API keys and console administrators hold project-wide read
		// access already; deliver everything within their own project without
		// per-row filtering. Connections with no read authorizer wired fall
		// back to project+auth scoping only.
		if c.broadAccess || h.readAuth == nil {
			return allow(nil)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		dec, err := h.readAuth.AuthorizeTableRead(ctx, pc.projectID, pc.databaseID, pc.tableName, c.userID)
		if err != nil {
			// Fail closed: an unresolved permission is a denied one.
			return deny("realtime_forbidden_read", "You do not have permission to read this table.")
		}
		if dec.AllowAll {
			return allow(nil)
		}
		if dec.RowFiltered {
			return allow(newRowFilter(dec.Roles))
		}
		return deny("realtime_forbidden_read", "You do not have permission to read this table.")

	case kindProjectData:
		if pc.projectID == "" || pc.projectID != c.projectID {
			return deny("realtime_forbidden_project", "Channel belongs to a different project.")
		}
		if !c.authenticated {
			return deny("realtime_unauthenticated", "Authentication is required to subscribe to data channels.")
		}
		// projects.{proj}.databases.rows is the project-wide row feed: it
		// aggregates every table's row changes across the whole project with no
		// per-row filtering, so — like the table-less database channel — it must
		// require broad access rather than a plain end-user session. Other
		// resource feeds (storage files, etc.) stay open to authenticated
		// sessions.
		if pc.service == "databases" && pc.resource == "rows" && !c.broadAccess {
			return deny("realtime_forbidden_read", "The project-wide row feed requires a server API key.")
		}
		return allow(nil)

	case kindDeploy:
		if !c.authenticated {
			return deny("realtime_unauthenticated", "Authentication is required to subscribe to deploy channels.")
		}
		if h.releaseVerifier != nil && pc.releaseID != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			ok, err := h.releaseVerifier.ReleaseBelongsToProject(ctx, pc.releaseID, c.projectID)
			if err != nil || !ok {
				return deny("realtime_forbidden_project", "Release belongs to a different project.")
			}
		}
		return allow(nil)

	default:
		// Unknown channel shapes fail closed rather than subscribing to
		// something no publisher scopes.
		return deny("realtime_unknown_channel", "Unknown channel.")
	}
}

// rowFilter admits a database change event only when the changed row's own
// _permissions grant read to one of the subscriber's roles. It is applied on
// the delivery path for document-security tables and uses only the event
// payload — no database access — so broadcasting stays fast.
type rowFilter struct {
	roles map[string]bool
}

func newRowFilter(roles []string) *rowFilter {
	m := make(map[string]bool, len(roles))
	for _, r := range roles {
		if r != "" {
			m[r] = true
		}
	}
	return &rowFilter{roles: m}
}

// allows reports whether the event's row is readable by this filter's roles.
// Fails closed: a payload without a usable _permissions.read list is not
// delivered, matching row-security semantics (no grant means no read).
func (f *rowFilter) allows(ev Event) bool {
	msg, ok := ev.Payload.(map[string]interface{})
	if !ok {
		return false
	}
	row, _ := msg["new"].(map[string]interface{})
	if row == nil {
		row, _ = msg["old"].(map[string]interface{})
	}
	if row == nil {
		return false
	}
	perms, ok := row["_permissions"].(map[string]interface{})
	if !ok {
		return false
	}
	reads, ok := perms["read"].([]interface{})
	if !ok {
		return false
	}
	for _, r := range reads {
		role, _ := r.(string)
		role = strings.Trim(role, `"'`)
		if role == "any" || f.roles[role] {
			return true
		}
	}
	return false
}
