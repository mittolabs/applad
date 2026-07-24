package runtime

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
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

// StudioBrowserImage is the image sessions run.
func StudioBrowserImage() string {
	if v := os.Getenv("APPLAD_BROWSER_IMAGE"); v != "" {
		return v
	}
	return studioImageName
}

const studioImageName = "applad-studio-browser:1"

/*
 * Chromium binds its DevTools endpoint to loopback and refuses connections
 * from anywhere else — deliberate hardening that --remote-debugging-address
 * does not override in current builds. It also rejects HTTP requests whose
 * Host header is neither an IP nor localhost.
 *
 * So the browser image puts a forwarder in front: Chromium listens on
 * loopback, socat accepts from the network and relays. Callers still have to
 * present Host: localhost, which is why every request below sets it.
 */
const studioDockerfile = `FROM zenika/alpine-chrome:latest
USER root
RUN apk add --no-cache socat
USER chrome
ENTRYPOINT ["/bin/sh","-c","socat TCP-LISTEN:9222,fork,reuseaddr TCP:127.0.0.1:9223 & exec chromium-browser --headless=new --remote-debugging-port=9223 --no-sandbox --disable-dev-shm-usage --disable-gpu --hide-scrollbars --window-size=1280,800 about:blank"]
`

// ensureBrowserImage builds the studio's browser image if it is not already
// present. Docker's layer cache makes this free after the first session.
func (d *DeployExecutor) ensureBrowserImage(ctx context.Context, image string) error {
	if image != studioImageName {
		// A caller supplied their own image and is responsible for it.
		return d.pullImage(ctx, image)
	}

	tarBuf := new(bytes.Buffer)
	tw := tar.NewWriter(tarBuf)
	addToTar(tw, "Dockerfile", []byte(studioDockerfile))
	tw.Close()

	_, err := d.docker.BuildImage(ctx, image, tarBuf)
	return err
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

	if err := d.ensureBrowserImage(ctx, image); err != nil {
		return "", "", fmt.Errorf("browser: prepare image %s: %w", image, err)
	}

	body := map[string]interface{}{
		"Image": image,
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

		req, reqErr := http.NewRequestWithContext(ctx, "GET",
			fmt.Sprintf("http://%s:9222/json/list", host), nil)
		if reqErr != nil {
			return "", reqErr
		}
		// Chromium rejects a DevTools request from any other host.
		req.Host = "localhost"

		resp, err := client.Do(req)
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

// StopBrowserByName ends a session's browser given the session's container
// name, which is what the worker has when the request comes from elsewhere.
func (d *DeployExecutor) StopBrowserByName(ctx context.Context, name string) error {
	id := d.findContainerByName(ctx, name)
	if id == "" {
		return nil
	}
	return d.StopBrowser(ctx, id)
}

// StopBrowser ends a session's browser.
func (d *DeployExecutor) StopBrowser(ctx context.Context, containerID string) error {
	if containerID == "" {
		return nil
	}
	d.docker.StopContainer(ctx, containerID) //nolint:errcheck
	return d.docker.RemoveContainer(ctx, containerID)
}

// ReapStaleBrowsers removes studio browsers older than maxAge.
//
// The API reaps sessions it knows about, but that state is in memory: an API
// restart forgets every live session while their containers keep running, and
// nothing would ever clean them up. Several were found up for more than a day,
// each holding a Chromium. This is the backstop that does not depend on anyone
// remembering — a recording is interactive and short-lived, so a studio
// container older than maxAge is abandoned by definition.
func (d *DeployExecutor) ReapStaleBrowsers(ctx context.Context, maxAge time.Duration) (int, error) {
	filter := url.QueryEscape(`{"name":["applad-studio-"]}`)
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf(d.docker.baseURL+"/v1.44/containers/json?all=true&filters=%s", filter), nil)
	if err != nil {
		return 0, err
	}
	resp, err := d.docker.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var containers []struct {
		ID      string   `json:"Id"`
		Names   []string `json:"Names"`
		Created int64    `json:"Created"` // unix seconds
	}
	if err := json.Unmarshal(body, &containers); err != nil {
		return 0, err
	}

	cutoff := time.Now().Add(-maxAge).Unix()
	reaped := 0
	for _, c := range containers {
		if c.Created > cutoff {
			continue
		}
		if err := d.StopBrowser(ctx, c.ID); err != nil {
			slog.Warn("runtime: could not reap studio browser", "container", c.ID, "error", err)
			continue
		}
		slog.Info("runtime: reaped stale studio browser", "names", c.Names)
		reaped++
	}
	return reaped, nil
}
