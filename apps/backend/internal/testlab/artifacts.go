package testlab

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/uid"
)

/*
 * Evidence from a run.
 *
 * An assertion message says what broke. A recording says what the user would
 * have seen, which for a browser or device run is usually the faster way to
 * understand a failure. Files are written to the shared storage volume rather
 * than the database, and served back through the API.
 */

// Artifact is one file a run produced.
type Artifact struct {
	ID          string    `json:"$id"`
	RunID       string    `json:"runId"`
	CaseID      string    `json:"caseId,omitempty"`
	Kind        string    `json:"kind"`
	Name        string    `json:"name"`
	ContentType string    `json:"contentType"`
	SizeBytes   int64     `json:"sizeBytes"`
	CreatedAt   time.Time `json:"$createdAt"`
}

// ArtifactsDir is where run evidence lives on the storage volume, shared
// between the API (which serves it) and the worker (which writes it).
func ArtifactsDir() string {
	base := os.Getenv("STORAGE_PATH")
	if base == "" {
		base = "/var/applad/storage"
	}
	return filepath.Join(base, "test-artifacts")
}

// classify guesses what a file is from its name, which is all the runners
// give us. Kind drives how the console presents it: a video gets a player, a
// screenshot an image, everything else a download.
func classify(name string) (kind, contentType string) {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".webm":
		return "video", "video/webm"
	case ".mp4":
		return "video", "video/mp4"
	case ".png":
		return "screenshot", "image/png"
	case ".jpg", ".jpeg":
		return "screenshot", "image/jpeg"
	case ".zip":
		return "trace", "application/zip"
	case ".xml":
		return "report", "application/xml"
	case ".json":
		return "report", "application/json"
	case ".txt", ".log":
		return "other", "text/plain"
	default:
		return "other", "application/octet-stream"
	}
}

// nonWord is used to compare a runner's directory naming against a test name.
var nonWord = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	return strings.Trim(nonWord.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

// matchesCase decides whether an artifact path belongs to a test.
//
// Runners shorten long directory names: Playwright writes
// "a-visitor-lands-on-the-hom-21f4f-itor-lands-on-the-home-page" for a test
// called "a visitor lands on the home page", eliding the middle and inserting
// a hash. Looking for the whole slug therefore matches only short names, so
// each end is checked instead.
func matchesCase(haystack, testSlug string) bool {
	if testSlug == "" {
		return false
	}
	if strings.Contains(haystack, testSlug) {
		return true
	}
	// Long enough to be unlikely to collide, short enough to survive elision.
	const edge = 20
	if len(testSlug) <= edge {
		return false
	}
	return strings.Contains(haystack, testSlug[:edge]) ||
		strings.Contains(haystack, testSlug[len(testSlug)-edge:])
}

// StoreArtifacts writes a run's files to disk and records them.
//
// Runners that record per test name their output directory after the test, so
// where a path contains a case's name the artifact is attached to that case.
// Anything else stays attached to the run, which is still the right home for
// a combined report or a trace covering the whole suite.
func (s *Service) StoreArtifacts(ctx context.Context, runID, projectID string, files map[string][]byte) error {
	if len(files) == 0 {
		return nil
	}

	cases, _, err := s.ListCases(ctx, runID, projectID)
	if err != nil {
		return err
	}
	// Longest names first, so a specific test wins over one whose name is a
	// prefix of it.
	type indexed struct{ id, slug string }
	var index []indexed
	for _, c := range cases {
		index = append(index, indexed{id: c.ID, slug: slug(c.Name)})
	}
	for i := range index {
		for j := i + 1; j < len(index); j++ {
			if len(index[j].slug) > len(index[i].slug) {
				index[i], index[j] = index[j], index[i]
			}
		}
	}

	runDir := filepath.Join(ArtifactsDir(), runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return fmt.Errorf("testlab: artifacts dir: %w", err)
	}

	for name, data := range files {
		// Refuse paths that would escape the run's directory.
		clean := filepath.Clean("/" + name)
		dest := filepath.Join(runDir, clean)
		if !strings.HasPrefix(dest, filepath.Clean(runDir)+string(os.PathSeparator)) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			continue
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			continue
		}

		kind, contentType := classify(name)
		var caseID string
		haystack := slug(name)
		for _, c := range index {
			if matchesCase(haystack, c.slug) {
				caseID = c.id
				break
			}
		}

		s.db.ExecContext(ctx, //nolint:errcheck
			`INSERT INTO test_artifacts (id, run_id, project_id, case_id, kind, name, content_type, size_bytes, storage_path, created_at)
			 VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,$9,$10)`,
			uid.New("unique()"), runID, projectID, caseID, kind, name, contentType,
			int64(len(data)), filepath.Join(runID, clean), time.Now().UTC())
	}
	return nil
}

// ListArtifacts returns a run's evidence, videos first: on a red run the
// recording is what people open.
func (s *Service) ListArtifacts(ctx context.Context, runID, projectID string) ([]*Artifact, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, run_id, COALESCE(case_id,''), kind, name, content_type, size_bytes, created_at
		   FROM test_artifacts WHERE run_id = $1 AND project_id = $2
		  ORDER BY CASE kind WHEN 'video' THEN 0 WHEN 'screenshot' THEN 1 WHEN 'trace' THEN 2 ELSE 3 END, name`,
		runID, projectID)
	if err != nil {
		return nil, fmt.Errorf("testlab: list artifacts: %w", err)
	}
	defer rows.Close()

	var out []*Artifact
	for rows.Next() {
		var a Artifact
		if err := rows.Scan(&a.ID, &a.RunID, &a.CaseID, &a.Kind, &a.Name,
			&a.ContentType, &a.SizeBytes, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, nil
}

// OpenArtifact resolves an artifact to a file on disk for serving.
func (s *Service) OpenArtifact(ctx context.Context, id, projectID string) (path, contentType, name string, err error) {
	var storagePath string
	err = s.db.QueryRowContext(ctx,
		`SELECT storage_path, content_type, name FROM test_artifacts WHERE id = $1 AND project_id = $2`,
		id, projectID).Scan(&storagePath, &contentType, &name)
	if err != nil {
		return "", "", "", fmt.Errorf("testlab: artifact not found")
	}
	return filepath.Join(ArtifactsDir(), storagePath), contentType, name, nil
}

// DeleteRunArtifacts removes a run's files from disk.
func DeleteRunArtifacts(runID string) {
	os.RemoveAll(filepath.Join(ArtifactsDir(), runID)) //nolint:errcheck
}
