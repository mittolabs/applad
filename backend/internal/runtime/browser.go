package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

/*
 * A browser held open for a recording session.
 *
 * Unlike a test run, which builds an image and waits for it to exit, this one
 * stays alive and is driven over the DevTools protocol for as long as somebody
 * is clicking through the app.
 */

// StudioBrowserImage is the image sessions run. Chromium with the DevTools
// protocol exposed is all that is required; the studio drives it directly.
func StudioBrowserImage() string {
	if v := os.Getenv("APPLAD_BROWSER_IMAGE"); v != "" {
		return v
	}
	return "zenika/alpine-chrome:latest"
}

// StartBrowser launches a browser and returns its container and the DevTools
// endpoint to drive it through.
func (d *DeployExecutor) StartBrowser(ctx context.Context, sessionID, image string) (containerID, wsURL string, err error) {
	name := fmt.Sprintf("applad-studio-%s", sessionID)

	// Remove a container left behind by a session that ended badly, so a
	// retry is not blocked by its own debris.
	if old := d.findContainerByName(ctx, name); old != "" {
		d.docker.RemoveContainer(context.Background(), old) //nolint:errcheck
	}

	if err := d.pullImage(ctx, image); err != nil {
		return "", "", fmt.Errorf("browser: pull %s: %w", image, err)
	}

	body := map[string]interface{}{
		"Image": image,
		"Cmd": []string{
			"--headless=new",
			"--remote-debugging-address=0.0.0.0",
			"--remote-debugging-port=9222",
			// Sandboxing is off because the container is the sandbox, and
			// Chromium's own needs privileges we would rather not grant.
			"--no-sandbox",
			"--disable-dev-shm-usage",
			"--disable-gpu",
			"--hide-scrollbars",
			"--window-size=1280,800",
			"about:blank",
		},
		"Labels": map[string]string{
			"applad.managed": "true",
			"applad.type":    "studio",
		},
		"ExposedPorts": map[string]interface{}{"9222/tcp": struct{}{}},
		// On the deploy network, so a session can record against an app
		// deployed here by name.
		"NetworkingConfig": map[string]interface{}{
			"EndpointsConfig": map[string]interface{}{
				deployNetwork(): map[string]interface{}{},
			},
		},
		"HostConfig": map[string]interface{}{
			"Memory":      int64(1536 * 1024 * 1024),
			"NanoCPUs":    int64(2e9),
			"NetworkMode": deployNetwork(),
			"SecurityOpt": []string{"no-new-privileges"},
			// A browser that outlives its session would hold a gigabyte for
			// nothing, so it is never restarted.
			"RestartPolicy": map[string]interface{}{"Name": "no"},
		},
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf(d.docker.baseURL+"/v1.44/containers/create?name=%s", name), bytes.NewReader(data))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.docker.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("browser: create: %w", err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("browser: create failed (%d): %s", resp.StatusCode, string(respBody))
	}
	var created struct {
		ID string `json:"Id"`
	}
	json.Unmarshal(respBody, &created) //nolint:errcheck

	if err := d.docker.StartContainer(ctx, created.ID); err != nil {
		d.docker.RemoveContainer(context.Background(), created.ID) //nolint:errcheck
		return "", "", fmt.Errorf("browser: start: %w", err)
	}

	endpoint, err := d.waitForDevTools(ctx, name)
	if err != nil {
		d.StopBrowser(context.Background(), created.ID)
		return "", "", err
	}
	return created.ID, endpoint, nil
}

// waitForDevTools polls the browser until it reports a debuggable page.
// Chromium accepts connections a moment before it has a target to attach to,
// so asking for the endpoint is the only reliable readiness check.
func (d *DeployExecutor) waitForDevTools(ctx context.Context, host string) (string, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		resp, err := client.Get(fmt.Sprintf("http://%s:9222/json/list", host))
		if err == nil {
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			var targets []struct {
				Type                 string `json:"type"`
				WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
			}
			if json.Unmarshal(data, &targets) == nil {
				for _, t := range targets {
					if t.Type == "page" && t.WebSocketDebuggerURL != "" {
						// The URL Chromium reports is addressed to itself;
						// reach it by container name on our shared network.
						return rewriteDevToolsHost(t.WebSocketDebuggerURL, host), nil
					}
				}
			}
		}
		time.Sleep(400 * time.Millisecond)
	}
	return "", fmt.Errorf("browser: devtools did not become ready")
}

func rewriteDevToolsHost(wsURL, host string) string {
	const prefix = "ws://"
	if !strings.HasPrefix(wsURL, prefix) {
		return wsURL
	}
	rest := wsURL[len(prefix):]
	if i := strings.Index(rest, "/"); i >= 0 {
		return fmt.Sprintf("%s%s:9222%s", prefix, host, rest[i:])
	}
	return wsURL
}

// StopBrowser ends a session's browser.
func (d *DeployExecutor) StopBrowser(ctx context.Context, containerID string) error {
	if containerID == "" {
		return nil
	}
	d.docker.StopContainer(ctx, containerID) //nolint:errcheck
	return d.docker.RemoveContainer(ctx, containerID)
}
