// Package browsershot photographs a page in an already-running Chromium.
//
// This is what is left of the browser machinery after the Test feature was
// removed: the deploy pipeline still wants a picture of a site once it is
// serving, and that never needed the recorder, the frame stream or the
// console/network capture the recording studio's CDP client carried.
package browsershot

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

/*
 * A picture of a deployed site.
 *
 * The console showed a globe icon labelled "Site preview", which is a promise
 * rather than a preview. A deploy already pays for a browser container, so the
 * same machinery can look at the site once it is live and keep what it saw.
 */

// Screenshot loads a page in an already-running browser and returns a PNG.
//
// The browser is the caller's: starting one costs seconds, and a deploy has
// just paid for a container start already.
func Screenshot(ctx context.Context, wsURL, pageURL string, width, height int) ([]byte, error) {
	client, err := dial(wsURL)
	if err != nil {
		return nil, fmt.Errorf("screenshot: connect: %w", err)
	}
	defer client.close()

	if _, err := client.send("Page.enable", map[string]interface{}{}); err != nil {
		return nil, err
	}
	// A fixed viewport keeps previews a consistent shape regardless of what
	// the browser image defaults to.
	client.send("Emulation.setDeviceMetricsOverride", map[string]interface{}{ //nolint:errcheck
		"width": width, "height": height, "deviceScaleFactor": 1, "mobile": false,
	})

	if _, err := client.send("Page.navigate", map[string]interface{}{"url": pageURL}); err != nil {
		return nil, fmt.Errorf("screenshot: navigate: %w", err)
	}

	// Wait for the page to settle rather than for a load event: a site whose
	// hero is a large image looks broken if caught too early.
	if err := waitForQuiet(ctx, client); err != nil {
		return nil, err
	}

	res, err := client.send("Page.captureScreenshot", map[string]interface{}{
		"format": "png", "captureBeyondViewport": false,
	})
	if err != nil {
		return nil, fmt.Errorf("screenshot: capture: %w", err)
	}

	var out struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(res, &out); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(out.Data)
}

// waitForQuiet polls until the document is complete, then gives images a
// moment to paint.
func waitForQuiet(ctx context.Context, client *cdpClient) error {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}

		res, err := client.send("Runtime.evaluate", map[string]interface{}{
			"expression":    "document.readyState",
			"returnByValue": true,
		})
		if err != nil {
			continue
		}
		var out struct {
			Result struct {
				Value string `json:"value"`
			} `json:"result"`
		}
		if json.Unmarshal(res, &out) == nil && out.Result.Value == "complete" {
			// Enough for images already in flight to land.
			time.Sleep(1500 * time.Millisecond)
			return nil
		}
	}
	return nil // capture whatever is on screen rather than failing the deploy
}

// ── A minimal Chrome DevTools Protocol client ────────────────────────────────

type cdpClient struct {
	conn *websocket.Conn

	mu      sync.Mutex
	nextID  int
	pending map[int]chan json.RawMessage

	writeMu sync.Mutex
	closed  bool
}

type cdpMessage struct {
	ID     int             `json:"id,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
}

func dial(wsURL string) (*cdpClient, error) {
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
		if json.Unmarshal(data, &msg) != nil || msg.ID == 0 {
			continue // an event; the screenshot path subscribes to none
		}
		c.mu.Lock()
		ch, ok := c.pending[msg.ID]
		delete(c.pending, msg.ID)
		c.mu.Unlock()
		if ok {
			ch <- msg.Result
		}
	}
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

func (c *cdpClient) close() {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	c.conn.Close() //nolint:errcheck
}
