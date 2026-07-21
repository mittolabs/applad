package testlab

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

/*
 * A picture of a deployed site.
 *
 * The console showed a globe icon labelled "Site preview", which is a promise
 * rather than a preview. The studio already runs a real browser to record
 * against, so the same machinery can look at a site once it is live and keep
 * what it saw.
 */

// Screenshot loads a page in an already-running browser and returns a PNG.
//
// The browser is the caller's: starting one costs seconds, and a deploy has
// just paid for a container start already.
func Screenshot(ctx context.Context, wsURL, pageURL string, width, height int) ([]byte, error) {
	client, err := newCDPClient(wsURL)
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
