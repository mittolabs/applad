package testlab

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

/*
 * A small Chrome DevTools Protocol client.
 *
 * Only what the studio needs: stream the page as frames, forward input, and
 * carry the recorder's findings back. Chromium is driven directly rather than
 * through Playwright because the recording is ours — Playwright is what the
 * saved flow compiles to, not what runs the session.
 */

type cdpClient struct {
	conn *websocket.Conn

	mu      sync.Mutex
	nextID  int
	pending map[int]chan json.RawMessage

	onStep    func(string)
	onFrame   func([]byte)
	onCapture func([]byte) // console/network/env timeline events

	// In-flight network requests, correlated across the request/response/finish
	// events into one complete row emitted when the request settles.
	netMu       sync.Mutex
	netInflight map[string]*netEntry

	writeMu sync.Mutex
	closed  bool
}

// netEntry accumulates a request as its CDP events arrive.
type netEntry struct {
	Method  string
	URL     string
	Type    string
	Status  int
	MIME    string
	Size    int64
	StartMs int64
}

type cdpMessage struct {
	ID     int             `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func newCDPClient(wsURL string) (*cdpClient, error) {
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	// Chromium refuses a DevTools connection whose Host is not localhost, and
	// the browser is reached by container name, so it is set explicitly.
	conn, _, err := dialer.Dial(wsURL, http.Header{"Host": []string{"localhost"}})
	if err != nil {
		return nil, err
	}
	c := &cdpClient{
		conn:        conn,
		pending:     map[int]chan json.RawMessage{},
		netInflight: map[string]*netEntry{},
	}
	go c.readLoop()
	return c, nil
}

// nowMs is the shared timeline clock: every capture event and (in Phase 2) every
// persisted frame is stamped with the server's wall clock, so the replay lines
// console, network, steps and video up against one another.
func nowMs() int64 { return time.Now().UnixMilli() }

func (c *cdpClient) readLoop() {
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			c.closed = true
			c.mu.Unlock()
			return
		}
		var msg cdpMessage
		if json.Unmarshal(data, &msg) != nil {
			continue
		}

		if msg.ID != 0 {
			c.mu.Lock()
			ch, ok := c.pending[msg.ID]
			delete(c.pending, msg.ID)
			c.mu.Unlock()
			if ok {
				ch <- msg.Result
			}
			continue
		}

		switch msg.Method {
		case "Page.screencastFrame":
			var p struct {
				Data      string `json:"data"`
				SessionID int    `json:"sessionId"`
				Metadata  struct {
					DeviceWidth  float64 `json:"deviceWidth"`
					DeviceHeight float64 `json:"deviceHeight"`
				} `json:"metadata"`
			}
			if json.Unmarshal(msg.Params, &p) != nil {
				continue
			}
			// Acknowledged without waiting for a reply. Waiting here would
			// deadlock: the reply can only be read by this loop, which would
			// be blocked waiting for it, and every later command would time
			// out from the first frame onwards.
			c.notify("Page.screencastFrameAck", map[string]interface{}{"sessionId": p.SessionID})
			if c.onFrame != nil {
				payload, _ := json.Marshal(map[string]interface{}{
					"type": "frame", "data": p.Data,
					"width": p.Metadata.DeviceWidth, "height": p.Metadata.DeviceHeight,
				})
				c.onFrame(payload)
			}

		case "Runtime.bindingCalled":
			var p struct {
				Name    string `json:"name"`
				Payload string `json:"payload"`
			}
			if json.Unmarshal(msg.Params, &p) != nil {
				continue
			}
			if p.Name == "__appladStep" && c.onStep != nil {
				c.onStep(p.Payload)
			}

		// ── Capture: the technical context a bug report needs ──────────────
		case "Runtime.consoleAPICalled":
			c.emitConsole(msg.Params)
		case "Runtime.exceptionThrown":
			c.emitException(msg.Params)
		case "Log.entryAdded":
			c.emitLogEntry(msg.Params)
		case "Network.requestWillBeSent":
			c.netStart(msg.Params)
		case "Network.responseReceived":
			c.netResponse(msg.Params)
		case "Network.loadingFinished":
			c.netFinish(msg.Params, false)
		case "Network.loadingFailed":
			c.netFinish(msg.Params, true)
		}
	}
}

// notify sends a command without waiting for its reply, for the cases where
// the reply carries nothing and waiting would be harmful.
func (c *cdpClient) notify(method string, params interface{}) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.nextID++
	id := c.nextID
	c.mu.Unlock()

	body, _ := json.Marshal(map[string]interface{}{"id": id, "method": method, "params": params})
	c.writeMu.Lock()
	c.conn.WriteMessage(websocket.TextMessage, body) //nolint:errcheck
	c.writeMu.Unlock()
}

func (c *cdpClient) send(method string, params interface{}) (json.RawMessage, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("cdp: connection closed")
	}
	c.nextID++
	id := c.nextID
	ch := make(chan json.RawMessage, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	body, _ := json.Marshal(map[string]interface{}{"id": id, "method": method, "params": params})

	c.writeMu.Lock()
	err := c.conn.WriteMessage(websocket.TextMessage, body)
	c.writeMu.Unlock()
	if err != nil {
		return nil, err
	}

	select {
	case res := <-ch:
		return res, nil
	case <-time.After(15 * time.Second):
		return nil, fmt.Errorf("cdp: %s timed out", method)
	}
}

// setup enables the domains the studio needs and installs the recorder so it
// runs on every document, including after a navigation.
func (c *cdpClient) setup(recorder string, onStep func(string), onFrame func([]byte), onCapture func([]byte)) error {
	c.onStep, c.onFrame, c.onCapture = onStep, onFrame, onCapture

	// Page/Runtime/DOM drive the recorder and screencast; Network and Log are the
	// capture domains — the console, exceptions and requests a bug report needs.
	for _, m := range []string{"Page.enable", "Runtime.enable", "DOM.enable", "Network.enable", "Log.enable"} {
		if _, err := c.send(m, map[string]interface{}{}); err != nil {
			return err
		}
	}

	// The binding is how the page hands a step back to us.
	if _, err := c.send("Runtime.addBinding", map[string]interface{}{"name": "__appladStep"}); err != nil {
		return err
	}
	if _, err := c.send("Page.addScriptToEvaluateOnNewDocument", map[string]interface{}{
		"source": recorder,
	}); err != nil {
		return err
	}

	// JPEG at a moderate quality: this is a live view for clicking through, not
	// the artifact. The recording that gets kept is the generated test.
	_, err := c.send("Page.startScreencast", map[string]interface{}{
		"format": "jpeg", "quality": 70, "maxWidth": 1280, "maxHeight": 800, "everyNthFrame": 1,
	})
	return err
}

// captureFrame takes a screenshot on demand, in the same envelope the screencast
// uses.
//
// The screencast only emits on repaint, so a page that has finished loading goes
// quiet. A console connecting after that would wait forever for a frame that is
// never coming — which is exactly what "Waiting for the first frame" was. This
// gives every new viewer something to show immediately, regardless of whether
// the page happens to be painting.
func (c *cdpClient) captureFrame() ([]byte, error) {
	res, err := c.send("Page.captureScreenshot", map[string]interface{}{
		"format": "jpeg", "quality": 70,
	})
	if err != nil {
		return nil, err
	}
	var out struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(res, &out); err != nil {
		return nil, err
	}
	if out.Data == "" {
		return nil, errNoFrame
	}
	return json.Marshal(map[string]interface{}{"type": "frame", "data": out.Data})
}

var errNoFrame = errors.New("cdp: empty screenshot")

func (c *cdpClient) navigate(url string) error {
	if _, err := c.send("Page.navigate", map[string]interface{}{"url": url}); err != nil {
		return err
	}
	// The recorder is installed per document; this covers the first one, which
	// may already have loaded before the script was registered.
	time.Sleep(400 * time.Millisecond)
	_, err := c.send("Runtime.evaluate", map[string]interface{}{"expression": recorderJS})
	return err
}

func (c *cdpClient) click(x, y float64) error {
	for _, kind := range []string{"mousePressed", "mouseReleased"} {
		if _, err := c.send("Input.dispatchMouseEvent", map[string]interface{}{
			"type": kind, "x": x, "y": y, "button": "left", "clickCount": 1,
		}); err != nil {
			return err
		}
	}
	return nil
}

// editingKeys are the non-printable keys that must carry a virtual key code, or
// Chromium ignores them entirely — which is why Backspace did nothing: you could
// type but not delete. Keys with a `text` (Enter → carriage return) are sent as
// keyDown so the char event fires and the default action runs — Enter submits a
// form only if that char event happens, which is why raw Enter did nothing. Keys
// without text (Backspace, arrows, …) are rawKeyDown, which is enough for the
// editing/navigation action.
var editingKeys = map[string]struct {
	vk   int
	code string
	text string
}{
	"Backspace":  {8, "Backspace", ""},
	"Tab":        {9, "Tab", ""},
	"Enter":      {13, "Enter", "\r"},
	"Escape":     {27, "Escape", ""},
	"Delete":     {46, "Delete", ""},
	"ArrowLeft":  {37, "ArrowLeft", ""},
	"ArrowUp":    {38, "ArrowUp", ""},
	"ArrowRight": {39, "ArrowRight", ""},
	"ArrowDown":  {40, "ArrowDown", ""},
	"Home":       {36, "Home", ""},
	"End":        {35, "End", ""},
}

func (c *cdpClient) key(key, text string) error {
	if sk, ok := editingKeys[key]; ok {
		base := map[string]interface{}{
			"key": key, "code": sk.code,
			"windowsVirtualKeyCode": sk.vk, "nativeVirtualKeyCode": sk.vk,
		}
		var down map[string]interface{}
		if sk.text != "" {
			// keyDown + text so the char event fires and Enter actually submits.
			down = clone(base, "type", "keyDown")
			down["text"] = sk.text
		} else {
			down = clone(base, "type", "rawKeyDown")
		}
		if _, err := c.send("Input.dispatchKeyEvent", down); err != nil {
			return err
		}
		_, err := c.send("Input.dispatchKeyEvent", clone(base, "type", "keyUp"))
		return err
	}

	params := map[string]interface{}{"type": "keyDown", "key": key}
	if text != "" {
		params["text"] = text
	}
	if _, err := c.send("Input.dispatchKeyEvent", params); err != nil {
		return err
	}
	_, err := c.send("Input.dispatchKeyEvent", map[string]interface{}{"type": "keyUp", "key": key})
	return err
}

func clone(m map[string]interface{}, k string, v interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m)+1)
	for kk, vv := range m {
		out[kk] = vv
	}
	out[k] = v
	return out
}

func (c *cdpClient) scroll(x, y, deltaX, deltaY float64) error {
	_, err := c.send("Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseWheel", "x": x, "y": y, "deltaX": deltaX, "deltaY": deltaY,
	})
	return err
}

// setViewport resizes the page to match what the console is showing and restarts
// the screencast at that size.
//
// Without this the browser is fixed at its launch size and the console scales
// the frames down to fit, so the view is small and imprecise and a frame shown
// larger than it was captured is blurry. Matching the viewport to the display
// makes the picture 1:1 and crisp, and makes a click land exactly where it looks
// like it should. Bounded so a maximised window cannot ask for an absurd canvas.
func (c *cdpClient) setViewport(width, height int) error {
	if width < 320 {
		width = 320
	}
	if width > 2560 {
		width = 2560
	}
	if height < 240 {
		height = 240
	}
	if height > 1600 {
		height = 1600
	}
	if _, err := c.send("Emulation.setDeviceMetricsOverride", map[string]interface{}{
		"width": width, "height": height, "deviceScaleFactor": 1, "mobile": false,
	}); err != nil {
		return err
	}
	// Restart the screencast so frames arrive at the new size.
	_, err := c.send("Page.startScreencast", map[string]interface{}{
		"format": "jpeg", "quality": 70, "maxWidth": width, "maxHeight": height, "everyNthFrame": 1,
	})
	return err
}

func (c *cdpClient) setAssertMode(on bool) error {
	_, err := c.send("Runtime.evaluate", map[string]interface{}{
		"expression": fmt.Sprintf("window.__appladAssertMode = %v", on),
	})
	return err
}

// ── Capture emitters ───────────────────────────────────────────────────────────
//
// Each turns a CDP event into a compact, timeline-stamped payload the console
// can show live and the session can persist. All share the server clock (nowMs)
// so console, network, steps and (later) video line up.

func (c *cdpClient) capture(v map[string]interface{}) {
	if c.onCapture == nil {
		return
	}
	v["ts"] = nowMs()
	if payload, err := json.Marshal(v); err == nil {
		c.onCapture(payload)
	}
}

func (c *cdpClient) emitConsole(params json.RawMessage) {
	var p struct {
		Type string `json:"type"`
		Args []struct {
			Value       json.RawMessage `json:"value"`
			Description string          `json:"description"`
		} `json:"args"`
		StackTrace struct {
			CallFrames []struct {
				URL        string `json:"url"`
				LineNumber int    `json:"lineNumber"`
			} `json:"callFrames"`
		} `json:"stackTrace"`
	}
	if json.Unmarshal(params, &p) != nil {
		return
	}
	parts := make([]string, 0, len(p.Args))
	for _, a := range p.Args {
		if len(a.Value) > 0 && a.Value[0] == '"' {
			var s string
			if json.Unmarshal(a.Value, &s) == nil {
				parts = append(parts, s)
				continue
			}
		}
		if a.Description != "" {
			parts = append(parts, a.Description)
		} else if len(a.Value) > 0 {
			parts = append(parts, string(a.Value))
		}
	}
	ev := map[string]interface{}{
		"type": "console", "level": consoleLevel(p.Type), "text": truncateStr(strings.Join(parts, " "), 2000),
	}
	if len(p.StackTrace.CallFrames) > 0 {
		ev["url"] = p.StackTrace.CallFrames[0].URL
		ev["line"] = p.StackTrace.CallFrames[0].LineNumber
	}
	c.capture(ev)
}

func (c *cdpClient) emitException(params json.RawMessage) {
	var p struct {
		ExceptionDetails struct {
			Text       string `json:"text"`
			URL        string `json:"url"`
			LineNumber int    `json:"lineNumber"`
			Exception  struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	if json.Unmarshal(params, &p) != nil {
		return
	}
	text := p.ExceptionDetails.Exception.Description
	if text == "" {
		text = p.ExceptionDetails.Text
	}
	c.capture(map[string]interface{}{
		"type": "console", "level": "error", "text": truncateStr(text, 2000),
		"url": p.ExceptionDetails.URL, "line": p.ExceptionDetails.LineNumber,
	})
}

func (c *cdpClient) emitLogEntry(params json.RawMessage) {
	var p struct {
		Entry struct {
			Level string `json:"level"`
			Text  string `json:"text"`
			URL   string `json:"url"`
		} `json:"entry"`
	}
	if json.Unmarshal(params, &p) != nil {
		return
	}
	c.capture(map[string]interface{}{
		"type": "console", "level": consoleLevel(p.Entry.Level),
		"text": truncateStr(p.Entry.Text, 2000), "url": p.Entry.URL,
	})
}

func (c *cdpClient) netStart(params json.RawMessage) {
	var p struct {
		RequestID string `json:"requestId"`
		Type      string `json:"type"`
		Request   struct {
			Method string `json:"method"`
			URL    string `json:"url"`
		} `json:"request"`
	}
	if json.Unmarshal(params, &p) != nil || p.RequestID == "" {
		return
	}
	c.netMu.Lock()
	c.netInflight[p.RequestID] = &netEntry{
		Method: p.Request.Method, URL: p.Request.URL, Type: p.Type, StartMs: nowMs(),
	}
	c.netMu.Unlock()
}

func (c *cdpClient) netResponse(params json.RawMessage) {
	var p struct {
		RequestID string `json:"requestId"`
		Response  struct {
			Status   int    `json:"status"`
			MIMEType string `json:"mimeType"`
		} `json:"response"`
	}
	if json.Unmarshal(params, &p) != nil {
		return
	}
	c.netMu.Lock()
	if e, ok := c.netInflight[p.RequestID]; ok {
		e.Status = p.Response.Status
		e.MIME = p.Response.MIMEType
	}
	c.netMu.Unlock()
}

func (c *cdpClient) netFinish(params json.RawMessage, failed bool) {
	var p struct {
		RequestID         string  `json:"requestId"`
		EncodedDataLength float64 `json:"encodedDataLength"`
	}
	if json.Unmarshal(params, &p) != nil {
		return
	}
	c.netMu.Lock()
	e := c.netInflight[p.RequestID]
	delete(c.netInflight, p.RequestID)
	c.netMu.Unlock()
	if e == nil {
		return
	}
	status := e.Status
	if failed {
		status = 0 // a failed request never got a status
	}
	c.capture(map[string]interface{}{
		"type": "network", "method": e.Method, "url": truncateStr(e.URL, 1000),
		"status": status, "mimeType": e.MIME, "resType": e.Type,
		"size": int64(p.EncodedDataLength), "durMs": nowMs() - e.StartMs, "failed": failed,
	})
}

// consoleLevel normalises CDP's console/log level names to info|warn|error.
func consoleLevel(t string) string {
	switch t {
	case "error", "assert":
		return "error"
	case "warning", "warn":
		return "warn"
	default:
		return "info"
	}
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func (c *cdpClient) close() {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	c.conn.Close() //nolint:errcheck
}
