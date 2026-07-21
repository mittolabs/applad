package testlab

import (
	"archive/tar"
	"compress/gzip"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mittolabs/applad/internal/deploy"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/mittolabs/applad/internal/uid"
	"github.com/redis/go-redis/v9"
)

/*
 * The recording studio.
 *
 * A session is a real browser running in a container, streamed into the
 * console and driven from it. What makes it a studio rather than a remote
 * desktop is the recorder injected into the page: every interaction comes back
 * as a step with a durable selector, so clicking through the app writes the
 * test.
 */

//go:embed recorder.js
var recorderJS string

// Session is one live browser.
type Session struct {
	ID        string    `json:"$id"`
	ProjectID string    `json:"projectId"`
	Platform  string    `json:"platform"`
	Target    string    `json:"target"`
	Status    string    `json:"status"` // starting | ready | closed
	CreatedAt time.Time `json:"$createdAt"`

	containerID string
	cdp         *cdpClient
	mu          sync.Mutex
	steps       []Step
	// subscribers receive frames and steps; the console is normally the only one.
	subscribers map[chan []byte]struct{}
	closed      bool
}

// Studio owns live sessions. They are deliberately in memory: a session is
// worthless once its browser is gone, and what survives is the saved flow.
//
// The browser is started by the builds worker rather than here, because only
// that worker holds the Docker socket — giving the internet-facing API control
// of the host daemon to open a browser would be a poor trade. The API then
// talks to the browser directly over the shared network, which needs no
// Docker access at all.
type Studio struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	svc      *Service
	queue    *queue.Queue
	rdb      *redis.Client
}

func NewStudio(svc *Service, q *queue.Queue, rdb *redis.Client) *Studio {
	return &Studio{
		sessions: map[string]*Session{},
		svc:      svc,
		queue:    q,
		rdb:      rdb,
	}
}

// browserEndpointKey is where the worker leaves the DevTools address once the
// browser is up.
func browserEndpointKey(sessionID string) string { return "applad:studio:" + sessionID }

// requestBrowser asks the builds worker for a browser and waits for it.
func (s *Studio) requestBrowser(ctx context.Context, sessionID, target, image string) (string, error) {
	if s.queue == nil || s.rdb == nil {
		return "", fmt.Errorf("studio: not configured")
	}
	if err := s.queue.Push(ctx, "builds", queue.Job{
		ID:   sessionID,
		Type: "studio_start",
		Payload: map[string]interface{}{
			"sessionId": sessionID, "target": target, "image": image,
		},
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return "", fmt.Errorf("studio: request browser: %w", err)
	}

	// Starting a browser includes pulling its image the first time, so this
	// waits longer than a queue round trip would suggest.
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
		val, err := s.rdb.Get(ctx, browserEndpointKey(sessionID)).Result()
		if err == nil && val != "" {
			if strings.HasPrefix(val, "error:") {
				return "", fmt.Errorf("studio: %s", strings.TrimPrefix(val, "error:"))
			}
			return val, nil
		}
	}
	return "", fmt.Errorf("studio: browser did not start in time")
}

// Start launches a browser against a target and begins streaming.
func (s *Studio) Start(ctx context.Context, projectID, target, image string) (*Session, error) {
	sess := &Session{
		ID:          uid.New("unique()"),
		ProjectID:   projectID,
		Platform:    "web",
		Target:      target,
		Status:      "starting",
		CreatedAt:   time.Now().UTC(),
		subscribers: map[chan []byte]struct{}{},
	}

	wsURL, err := s.requestBrowser(ctx, sess.ID, target, image)
	if err != nil {
		return nil, err
	}
	sess.containerID = sess.ID

	cdp, err := newCDPClient(wsURL)
	if err != nil {
		s.releaseBrowser(sess.ID)
		return nil, fmt.Errorf("studio: connect to browser: %w", err)
	}
	sess.cdp = cdp

	// The recorder is added before navigation so it survives every page load,
	// including ones the user triggers by clicking a link.
	if err := cdp.setup(recorderJS, func(raw string) {
		var step Step
		if err := json.Unmarshal([]byte(raw), &step); err != nil {
			return
		}
		sess.addStep(step)
	}, func(frame []byte) {
		sess.broadcast(frame)
	}); err != nil {
		cdp.close()
		s.releaseBrowser(sess.ID)
		return nil, fmt.Errorf("studio: prepare browser: %w", err)
	}

	if err := cdp.navigate(target); err != nil {
		slog.Warn("studio: initial navigation failed", "target", target, "error", err)
	}
	// The opening step, so a saved flow starts where the recording did.
	sess.addStep(Step{Kind: StepGoto, Value: target, Description: "open " + target})
	sess.Status = "ready"

	s.mu.Lock()
	s.sessions[sess.ID] = sess
	s.mu.Unlock()

	return sess, nil
}

func (s *Studio) Get(id, projectID string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[id]
	if !ok || sess.ProjectID != projectID {
		return nil, false
	}
	return sess, true
}

// Stop closes a session and takes its browser with it.
func (s *Studio) Stop(id, projectID string) {
	sess, ok := s.Get(id, projectID)
	if !ok {
		return
	}
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()

	sess.mu.Lock()
	sess.closed = true
	sess.Status = "closed"
	for ch := range sess.subscribers {
		close(ch)
	}
	sess.subscribers = map[chan []byte]struct{}{}
	sess.mu.Unlock()

	if sess.cdp != nil {
		sess.cdp.close()
	}
	s.releaseBrowser(sess.ID)
}

// releaseBrowser asks the worker to take the browser down.
func (s *Studio) releaseBrowser(sessionID string) {
	ctx := context.Background()
	if s.rdb != nil {
		s.rdb.Del(ctx, browserEndpointKey(sessionID)) //nolint:errcheck
	}
	if s.queue == nil {
		return
	}
	s.queue.Push(ctx, "builds", queue.Job{ //nolint:errcheck
		ID:        sessionID + "-stop",
		Type:      "studio_stop",
		Payload:   map[string]interface{}{"sessionId": sessionID},
		CreatedAt: time.Now().UTC(),
	})
}

// Steps returns what has been recorded so far.
func (sess *Session) Steps() []Step {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	out := make([]Step, len(sess.steps))
	copy(out, sess.steps)
	return out
}

// DeleteStep drops one recorded step, so a stray click does not have to end
// the recording.
func (sess *Session) DeleteStep(index int) {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if index < 0 || index >= len(sess.steps) {
		return
	}
	sess.steps = append(sess.steps[:index], sess.steps[index+1:]...)
}

func (sess *Session) addStep(step Step) {
	sess.mu.Lock()
	sess.steps = append(sess.steps, step)
	n := len(sess.steps)
	sess.mu.Unlock()

	payload, _ := json.Marshal(map[string]interface{}{
		"type": "step", "index": n - 1, "step": step,
	})
	sess.broadcast(payload)
}

func (sess *Session) broadcast(msg []byte) {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	for ch := range sess.subscribers {
		// A slow console must not stall the browser: drop rather than block.
		select {
		case ch <- msg:
		default:
		}
	}
}

func (sess *Session) subscribe() chan []byte {
	ch := make(chan []byte, 64)
	sess.mu.Lock()
	sess.subscribers[ch] = struct{}{}
	sess.mu.Unlock()
	return ch
}

func (sess *Session) unsubscribe(ch chan []byte) {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if _, ok := sess.subscribers[ch]; ok {
		delete(sess.subscribers, ch)
		close(ch)
	}
}

// ── The console's connection ──

var studioUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Stream connects the console to a session: frames and steps out, input in.
func (s *Studio) Stream(w http.ResponseWriter, r *http.Request, sess *Session) {
	conn, err := studioUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ch := sess.subscribe()
	defer sess.unsubscribe(ch)

	// Existing steps first, so a reconnecting console is not left blank.
	if payload, err := json.Marshal(map[string]interface{}{
		"type": "steps", "steps": sess.Steps(),
	}); err == nil {
		conn.WriteMessage(websocket.TextMessage, payload) //nolint:errcheck
	}

	go func() {
		for msg := range ch {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var in struct {
			Type       string  `json:"type"`
			X          float64 `json:"x"`
			Y          float64 `json:"y"`
			Button     string  `json:"button"`
			Text       string  `json:"text"`
			Key        string  `json:"key"`
			DeltaY     float64 `json:"deltaY"`
			AssertMode bool    `json:"assertMode"`
			Index      int     `json:"index"`
		}
		if json.Unmarshal(data, &in) != nil {
			continue
		}

		switch in.Type {
		case "click":
			sess.cdp.click(in.X, in.Y) //nolint:errcheck
		case "key":
			sess.cdp.key(in.Key, in.Text) //nolint:errcheck
		case "scroll":
			sess.cdp.scroll(in.X, in.Y, in.DeltaY) //nolint:errcheck
		case "assertMode":
			// Toggling is a page-side flag: in assert mode a click marks
			// something to check instead of doing it.
			sess.cdp.setAssertMode(in.AssertMode) //nolint:errcheck
		case "deleteStep":
			sess.DeleteStep(in.Index)
			if payload, err := json.Marshal(map[string]interface{}{
				"type": "steps", "steps": sess.Steps(),
			}); err == nil {
				sess.broadcast(payload)
			}
		}
	}
}

// browserTestImage is what a generated suite runs in: Playwright's own image,
// which ships the browsers so nothing has to be downloaded mid-run.
func browserTestImage() string {
	if v := os.Getenv("APPLAD_PLAYWRIGHT_IMAGE"); v != "" {
		return v
	}
	return "mcr.microsoft.com/playwright:v1.49.0-noble"
}

// writeGeneratedProject lays every recording down as one runnable Playwright
// project at the path the runner reads uploaded sources from, so a saved flow
// needs no upload step of its own.
func writeGeneratedProject(runnerID string, specs map[string]string) error {
	dir, err := os.MkdirTemp("", "applad-flows-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	if err := os.MkdirAll(filepath.Join(dir, "tests"), 0o755); err != nil {
		return err
	}

	files := map[string]string{
		"package.json": `{"name":"applad-recorded","private":true}`,
		"playwright.config.js": `// Generated by Applad from recorded flows.
//
// Retries are on: a test that fails and then passes is reported as flaky
// rather than failing the run, which is what stops one unreliable test from
// blocking every deploy.
module.exports = {
  testDir: './tests',
  outputDir: './test-results',
  timeout: 30000,
  retries: 1,
  reporter: [['junit', { outputFile: 'junit.xml' }]],
  use: {
    baseURL: process.env.BASE_URL,
    video: 'on',
    screenshot: 'only-on-failure',
    viewport: { width: 1280, height: 800 },
  },
};
`,
	}
	for name, spec := range specs {
		files[filepath.Join("tests", name)] = spec
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(deploy.SourceDir(), 0o755); err != nil {
		return err
	}
	return tarGzDir(dir, deploy.SourceArchivePath(runnerID))
}

// tarGzDir packages a directory as the gzipped tar the runner extracts.
func tarGzDir(dir, dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: filepath.ToSlash(rel), Mode: 0o644, Size: int64(len(data)),
		}); err != nil {
			return err
		}
		_, err = tw.Write(data)
		return err
	})
}
