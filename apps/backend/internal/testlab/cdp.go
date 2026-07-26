package testlab

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

	onStep  func(string)
	onFrame func([]byte)

	writeMu sync.Mutex
	closed  bool
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
	c := &cdpClient{conn: conn, pending: map[int]chan json.RawMessage{}}
	go c.readLoop()
	return c, nil
}

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
func (c *cdpClient) setup(recorder string, onStep func(string), onFrame func([]byte)) error {
	c.onStep, c.onFrame = onStep, onFrame

	for _, m := range []string{"Page.enable", "Runtime.enable", "DOM.enable"} {
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

func (c *cdpClient) key(key, text string) error {
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

func (c *cdpClient) scroll(x, y, deltaY float64) error {
	_, err := c.send("Input.dispatchMouseEvent", map[string]interface{}{
		"type": "mouseWheel", "x": x, "y": y, "deltaX": 0, "deltaY": deltaY,
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

func (c *cdpClient) close() {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	c.conn.Close() //nolint:errcheck
}
