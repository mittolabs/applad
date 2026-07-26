package testlab

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mittolabs/applad/internal/uid"
)

/*
 * A capture is the recording's technical context, kept alongside the flow: the
 * console, the network, the environment, a frame-by-frame video, and (later)
 * annotations and an AI summary. The studio already stored what somebody did;
 * this stores what the browser did while they did it, so a saved recording is
 * also a replay you can scrub — the jam.dev loop.
 *
 * The capture format is deliberately front-door-agnostic: it is a stream of
 * timeline events plus frames, not tied to how they were collected. Today they
 * come from the server-side studio; a browser extension could produce the same
 * shape and feed the same replay without a rewrite.
 *
 * The video is stored as sampled frames on the storage volume (no ffmpeg
 * dependency); each frame carries its real timestamp so console, network, steps
 * and the picture line up. An mp4 export is a later enhancement.
 */

// CapturesDir is where a capture's frames live, on the storage volume the API
// serves. Mirrors ArtifactsDir.
func CapturesDir() string {
	base := os.Getenv("STORAGE_PATH")
	if base == "" {
		base = "/var/applad/storage"
	}
	return filepath.Join(base, "test-captures")
}

// FrameMark places one persisted frame on the timeline: its sequence number and
// the offset from the capture's start, in milliseconds.
type FrameMark struct {
	Seq int   `json:"seq"`
	Ms  int64 `json:"ms"`
}

// Capture is the persisted replay. The event streams and steps are kept as raw
// JSON: their shape is owned by the producers (cdp.go, recorder.js), and the
// replay renders them without the store needing to understand them.
type Capture struct {
	ID          string          `json:"$id"`
	FlowID      string          `json:"flowId,omitempty"`
	ProjectID   string          `json:"projectId"`
	Target      string          `json:"target"`
	DurationMs  int64           `json:"durationMs"`
	StartedAt   int64           `json:"startedAt"` // timeline origin (unix ms)
	Status      string          `json:"status"`
	Frames      []FrameMark     `json:"frames"`
	Console     json.RawMessage `json:"console"`
	Network     json.RawMessage `json:"network"`
	Env         json.RawMessage `json:"env"`
	Steps       json.RawMessage `json:"steps"`
	Annotations json.RawMessage `json:"annotations"`
	AISummary   string          `json:"aiSummary,omitempty"`
	Shared      bool            `json:"shared"`
	ShareToken  string          `json:"shareToken,omitempty"`
}

func rawOr(b []byte, fallback string) json.RawMessage {
	if len(b) == 0 {
		return json.RawMessage(fallback)
	}
	return json.RawMessage(b)
}

// SaveCapture writes a capture linked to a flow. The frames are already on disk
// under CapturesDir()/<id>; this row is the index and the timeline.
func (s *Service) SaveCapture(ctx context.Context, c Capture) (*Capture, error) {
	if c.ID == "" {
		c.ID = uid.New("")
	}
	if c.Status == "" {
		c.Status = "ready"
	}
	frames, _ := json.Marshal(c.Frames)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO test_captures
  (id, flow_id, project_id, target, duration_ms, started_at, video_path, status,
   console, network, env, steps, annotations)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (id) DO UPDATE SET
  flow_id=EXCLUDED.flow_id, duration_ms=EXCLUDED.duration_ms, status=EXCLUDED.status,
  console=EXCLUDED.console, network=EXCLUDED.network, env=EXCLUDED.env, steps=EXCLUDED.steps,
  updated_at=now()`,
		c.ID, nullStr(c.FlowID), c.ProjectID, c.Target, c.DurationMs, c.StartedAt,
		string(frames), c.Status,
		rawOr(c.Console, "[]"), rawOr(c.Network, "[]"), rawOr(c.Env, "{}"),
		rawOr(c.Steps, "[]"), json.RawMessage("[]"))
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Service) getCaptureWhere(ctx context.Context, where string, arg any) (*Capture, error) {
	var c Capture
	var flowID, videoPath sql.NullString
	var shareToken sql.NullString
	var console, network, env, steps, annotations []byte
	row := s.db.QueryRowContext(ctx, `
SELECT id, flow_id, project_id, target, duration_ms, started_at, video_path, status,
       console, network, env, steps, annotations, ai_summary, share_token
  FROM test_captures WHERE `+where, arg)
	if err := row.Scan(&c.ID, &flowID, &c.ProjectID, &c.Target, &c.DurationMs, &c.StartedAt,
		&videoPath, &c.Status, &console, &network, &env, &steps, &annotations,
		&c.AISummary, &shareToken); err != nil {
		return nil, err
	}
	c.FlowID = flowID.String
	c.Console, c.Network, c.Env = console, network, env
	c.Steps, c.Annotations = steps, annotations
	// video_path holds the frame manifest as JSON (frames live beside it on disk).
	if videoPath.Valid && videoPath.String != "" {
		_ = json.Unmarshal([]byte(videoPath.String), &c.Frames)
	}
	c.Shared = shareToken.Valid && shareToken.String != ""
	c.ShareToken = shareToken.String
	return &c, nil
}

func (s *Service) GetCapture(ctx context.Context, id, projectID string) (*Capture, error) {
	c, err := s.getCaptureWhere(ctx, "id = $1", id)
	if err != nil {
		return nil, err
	}
	if c.ProjectID != projectID {
		return nil, sql.ErrNoRows
	}
	return c, nil
}

// GetCaptureForFlow returns the capture attached to a flow, if any.
func (s *Service) GetCaptureForFlow(ctx context.Context, flowID, projectID string) (*Capture, error) {
	c, err := s.getCaptureWhere(ctx, "flow_id = $1 ORDER BY created_at DESC LIMIT 1", flowID)
	if err != nil {
		return nil, err
	}
	if c.ProjectID != projectID {
		return nil, sql.ErrNoRows
	}
	return c, nil
}

// GetCaptureByShare resolves a public share token to its capture. No project
// scoping: the token IS the authorisation, which is why it is unguessable.
func (s *Service) GetCaptureByShare(ctx context.Context, token string) (*Capture, error) {
	return s.getCaptureWhere(ctx, "share_token = $1", token)
}

func (s *Service) SetCaptureAnnotations(ctx context.Context, id, projectID string, annotations json.RawMessage) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE test_captures SET annotations=$3, updated_at=now() WHERE id=$1 AND project_id=$2`,
		id, projectID, rawOr(annotations, "[]"))
	return err
}

func (s *Service) SetCaptureShare(ctx context.Context, id, projectID, token string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE test_captures SET share_token=$3, updated_at=now() WHERE id=$1 AND project_id=$2`,
		id, projectID, nullStr(token))
	return err
}

func (s *Service) SetCaptureAISummary(ctx context.Context, id, projectID, summary string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE test_captures SET ai_summary=$3, updated_at=now() WHERE id=$1 AND project_id=$2`,
		id, projectID, summary)
	return err
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
