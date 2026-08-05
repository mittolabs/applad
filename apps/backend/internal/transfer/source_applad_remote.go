package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mittolabs/applad/internal/netguard"
)

// wireResource is one line of the NDJSON export stream: a resource kind plus the
// JSON of the concrete resource struct. The export handler writes it; the remote
// source below reads it. Both ends are this same package, so the field-name
// encoding is stable.
type wireResource struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

// remoteAppladSource reads a project on ANOTHER Applad instance over its
// authenticated export endpoint (endpoint + project ID + API key). This is the
// cloud <-> self-hosted case. It streams NDJSON so a large project never has to
// be buffered whole on either side.
type remoteAppladSource struct {
	endpoint  string // e.g. https://api.applad.io
	projectID string
	apiKey    string
	http      *http.Client
}

// NewRemoteAppladSource builds a cross-instance Applad reader. The HTTP client
// has no overall timeout (a full export can stream for a while) but dials
// through netguard and is bounded by the caller's context / the job deadline.
func NewRemoteAppladSource(endpoint, projectID, apiKey string) (Source, error) {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" || projectID == "" || apiKey == "" {
		return nil, fmt.Errorf("applad remote: endpoint, sourceProjectId and sourceApiKey are required")
	}
	return &remoteAppladSource{
		endpoint:  endpoint,
		projectID: projectID,
		apiKey:    apiKey,
		http:      netguard.Client(0),
	}, nil
}

func (s *remoteAppladSource) Name() string { return "applad" }
func (s *remoteAppladSource) Close() error { return nil }

func (s *remoteAppladSource) request(ctx context.Context, query string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint+"/v1/migrations/export"+query, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Applad-Project", s.projectID)
	req.Header.Set("X-Applad-Key", s.apiKey)
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		resp.Body.Close()
		return nil, fmt.Errorf("applad remote: export %d: %s", resp.StatusCode, string(snippet))
	}
	return resp, nil
}

func (s *remoteAppladSource) Report(ctx context.Context, groups []Group) (map[Group]int, error) {
	resp, err := s.request(ctx, "?report=1&groups="+groupsCSV(groups))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Counts map[string]int `json:"counts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	res := map[Group]int{}
	for k, v := range out.Counts {
		res[Group(k)] = v
	}
	return res, nil
}

func (s *remoteAppladSource) Export(ctx context.Context, groups []Group, emit Emit) error {
	resp, err := s.request(ctx, "?groups="+groupsCSV(groups))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	batch := make([]Resource, 0, 200)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := emit(ctx, batch); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}
	for {
		var wr wireResource
		if err := dec.Decode(&wr); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("applad remote: decode stream: %w", err)
		}
		if wr.Kind == "error" {
			var msg struct {
				Message string `json:"message"`
			}
			_ = json.Unmarshal(wr.Data, &msg)
			return fmt.Errorf("applad remote: source error: %s", msg.Message)
		}
		r, err := decodeWireResource(wr)
		if err != nil {
			return err
		}
		if r == nil {
			continue
		}
		batch = append(batch, r)
		if len(batch) >= 200 {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	return flush()
}

// decodeWireResource turns one NDJSON line back into a concrete resource.
func decodeWireResource(wr wireResource) (Resource, error) {
	switch wr.Kind {
	case "user":
		var v User
		return v, json.Unmarshal(wr.Data, &v)
	case "team":
		var v Team
		return v, json.Unmarshal(wr.Data, &v)
	case "membership":
		var v Membership
		return v, json.Unmarshal(wr.Data, &v)
	case "database":
		var v Database
		return v, json.Unmarshal(wr.Data, &v)
	case "table":
		var v Table
		return v, json.Unmarshal(wr.Data, &v)
	case "column":
		var v Column
		return v, json.Unmarshal(wr.Data, &v)
	case "index":
		var v Index
		return v, json.Unmarshal(wr.Data, &v)
	case "row":
		var v Row
		return v, json.Unmarshal(wr.Data, &v)
	case "bucket":
		var v Bucket
		return v, json.Unmarshal(wr.Data, &v)
	case "file":
		var v File
		return v, json.Unmarshal(wr.Data, &v)
	case "function":
		var v Function
		return v, json.Unmarshal(wr.Data, &v)
	default:
		return nil, nil // unknown kind: ignore for forward-compatibility
	}
}

// groupsCSV joins groups for the export query string.
func groupsCSV(groups []Group) string {
	parts := make([]string, 0, len(groups))
	for _, g := range groups {
		parts = append(parts, string(g))
	}
	return strings.Join(parts, ",")
}

// parseGroups parses the export query's groups CSV, keeping only known groups.
func parseGroups(csv string) []Group {
	if strings.TrimSpace(csv) == "" {
		return nil
	}
	known := map[Group]bool{GroupAuth: true, GroupDatabases: true, GroupStorage: true, GroupFunctions: true}
	var out []Group
	for _, p := range strings.Split(csv, ",") {
		g := Group(strings.TrimSpace(p))
		if known[g] {
			out = append(out, g)
		}
	}
	return out
}
