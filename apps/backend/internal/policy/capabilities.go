package policy

// The capability registry is the versioned contract between core and anything
// that resolves policy. Keys are `domain.action` and are only added for actions
// that CREATE or CONSUME a resource, never for reads.
//
// Two rules keep this from metastasising through the codebase:
//
//  1. Adding a key requires a server-side enforcement site in the same change.
//     A key nothing enforces is a lie told to the UI.
//  2. Reads and exports are never gated. Enforcement degrades a plan, it does
//     not trap the data: whoever stops paying can still read and leave.
var capabilities = []Capability{
	{Key: "projects.create", Scope: ScopeOrg, Desc: "Create a project"},
	{Key: "members.invite", Scope: ScopeOrg, Desc: "Invite an organization member"},
	{Key: "databases.create", Scope: ScopeProject, Desc: "Create a database"},
	{Key: "functions.create", Scope: ScopeProject, Desc: "Create a function"},
	{Key: "storage.upload", Scope: ScopeProject, Desc: "Upload a file"},
	{Key: "deploy.run", Scope: ScopeProject, Desc: "Start a deployment"},
	{Key: "workflows.run", Scope: ScopeProject, Desc: "Execute a workflow"},
	{Key: "custom_domains.use", Scope: ScopeProject, Desc: "Attach a custom domain"},
	// A single write gate covering every mutating request on a project. Enforced
	// by middleware.EnforceWritable over POST/PUT/PATCH/DELETE, so a resolver can
	// put a whole workspace read-only without a check in every handler. Its
	// enforcement site is that middleware; the default resolver allows it, so a
	// build with no resolver (self-hosted) never blocks.
	{Key: "org.write", Scope: ScopeProject, Desc: "Perform any write in a workspace"},
}

// Capability describes one gateable action.
type Capability struct {
	Key   string    `json:"key"`
	Scope ScopeKind `json:"scope"`
	Desc  string    `json:"description"`
}

var byKey = func() map[string]Capability {
	m := make(map[string]Capability, len(capabilities))
	for _, c := range capabilities {
		m[c.Key] = c
	}
	return m
}()

// Capabilities returns the registry, for the UI and for contract tests.
func Capabilities() []Capability {
	out := make([]Capability, len(capabilities))
	copy(out, capabilities)
	return out
}

// Known reports whether a key is in the registry.
func Known(key string) bool {
	_, ok := byKey[key]
	return ok
}

// Lookup returns a capability by key.
func Lookup(key string) (Capability, bool) {
	c, ok := byKey[key]
	return c, ok
}
