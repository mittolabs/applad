package runtime

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
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
 * A browser held open and driven over the DevTools protocol.
 *
 * It stays alive rather than building an image and waiting for it to exit, so
 * the caller can navigate it and take a picture of what it found.
 */

// BrowserImage is the headless Chromium image the deploy pipeline runs to
// photograph a site once it is serving.
func BrowserImage() string {
	if v := os.Getenv("APPLAD_BROWSER_IMAGE"); v != "" {
		return v
	}
	return browserImageName
}

const browserImageName = "applad-browser:1"

/*
 * Chromium binds its DevTools endpoint to loopback and refuses connections
 * from anywhere else — deliberate hardening that --remote-debugging-address
 * does not override in current builds. It also rejects HTTP requests whose
 * Host header is neither an IP nor localhost.
 *
 * So the browser image puts a forwarder in front: Chromium listens on
 * loopback, the forwarder accepts from the network and relays. Callers still
 * have to present Host: localhost, which is why every request below sets it.
 *
 * The forwarder used to be a plain `socat` relay, which meant DevTools was
 * reachable, unauthenticated, by anything on the shared deploy network — a
 * deployed customer container could drive the browser (navigate it, read the
 * page under test, read local files). The relay now checks a per-session
 * token (APPLAD_DEVTOOLS_TOKEN) before forwarding, so only the API, which
 * minted the token, can drive the browser. The browser stays on the deploy
 * network because it must reach the target app it records by container name.
 */
const browserDockerfile = `FROM golang:1.22-alpine AS build
WORKDIR /src
COPY devtools-proxy.go .
RUN go build -o /devtools-proxy devtools-proxy.go
FROM zenika/alpine-chrome:latest
COPY --from=build /devtools-proxy /usr/local/bin/devtools-proxy
USER chrome
ENTRYPOINT ["/bin/sh","-c","/usr/local/bin/devtools-proxy & exec chromium-browser --headless=new --remote-debugging-port=9223 --no-sandbox --disable-dev-shm-usage --disable-gpu --hide-scrollbars --window-size=1280,800 about:blank"]
`

// devToolsTokenHeader and devToolsTokenParam carry the session token to the
// forwarder. The header is used for plain HTTP calls the API controls
// directly (/json/list); the query param rides in the returned WebSocket URL,
// so the CDP client authenticates without any change to how it dials.
const (
	devToolsTokenHeader = "X-Applad-Devtools-Token"
	devToolsTokenParam  = "_applad_token"
)

// devtoolsProxySource is the token-checking forwarder baked into the browser
// image. It replaces `socat`: it accepts a connection, reads the HTTP request,
// rejects it with 403 unless it presents the expected token (constant-time),
// strips the token, forces Host: localhost, and then splices the connection
// through to Chromium on loopback — which transparently carries the WebSocket
// upgrade and every DevTools frame after it.
const devtoolsProxySource = `package main

import (
	"bufio"
	"crypto/subtle"
	"io"
	"net"
	"net/http"
	"os"
)

func main() {
	token := os.Getenv("APPLAD_DEVTOOLS_TOKEN")
	ln, err := net.Listen("tcp", ":9222")
	if err != nil {
		os.Exit(1)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handle(conn, token)
	}
}

func deny(c net.Conn) {
	io.WriteString(c, "HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	c.Close()
}

func handle(client net.Conn, token string) {
	br := bufio.NewReader(client)
	req, err := http.ReadRequest(br)
	if err != nil {
		client.Close()
		return
	}

	got := req.Header.Get("X-Applad-Devtools-Token")
	if got == "" {
		got = req.URL.Query().Get("_applad_token")
	}
	if token == "" || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
		deny(client)
		return
	}

	// Strip the token before Chromium sees the request, and present the
	// localhost Host it insists on.
	q := req.URL.Query()
	q.Del("_applad_token")
	req.URL.RawQuery = q.Encode()
	req.Header.Del("X-Applad-Devtools-Token")
	req.Host = "localhost"
	req.Header.Set("Host", "localhost")

	upstream, err := net.Dial("tcp", "127.0.0.1:9223")
	if err != nil {
		client.Close()
		return
	}

	if err := req.Write(upstream); err != nil {
		upstream.Close()
		client.Close()
		return
	}

	// Splice both directions. br may already hold bytes read past the request
	// header (the start of a WebSocket frame), so copy from it, not the raw
	// conn.
	go func() {
		io.Copy(upstream, br)
		upstream.Close()
	}()
	io.Copy(client, upstream)
	client.Close()
}
`

// ensureBrowserImage builds the browser image if it is not already present.
// Docker's layer cache makes this free after the first time.
func (d *DeployExecutor) ensureBrowserImage(ctx context.Context, image string) error {
	if image != browserImageName {
		// A caller supplied their own image and is responsible for it.
		return d.pullImage(ctx, image)
	}

	tarBuf := new(bytes.Buffer)
	tw := tar.NewWriter(tarBuf)
	addToTar(tw, "Dockerfile", []byte(browserDockerfile))
	addToTar(tw, "devtools-proxy.go", []byte(devtoolsProxySource))
	tw.Close()

	_, err := d.docker.BuildImage(ctx, image, tarBuf)
	return err
}

// StartBrowser launches a browser and returns its container and the DevTools
// endpoint to drive it through.
func (d *DeployExecutor) StartBrowser(ctx context.Context, sessionID, image string) (containerID, wsURL string, err error) {
	name := fmt.Sprintf("applad-browser-%s", sessionID)

	// Remove a container left behind by a session that ended badly, so a
	// retry is not blocked by its own debris.
	if old := d.findContainerByName(ctx, name); old != "" {
		d.docker.RemoveContainer(context.Background(), old) //nolint:errcheck
	}

	if err := d.ensureBrowserImage(ctx, image); err != nil {
		return "", "", fmt.Errorf("browser: prepare image %s: %w", image, err)
	}

	// A per-session secret the forwarder demands before it will relay DevTools.
	// It is handed to the browser as an env var and echoed back to the API in
	// the endpoint URL, so an unauthenticated peer on the deploy network cannot
	// drive the browser even though it can reach the port.
	token := newDevToolsToken()
	body := browserBody(image, token)

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

	endpoint, err := d.waitForDevTools(ctx, name, token)
	if err != nil {
		d.StopBrowser(context.Background(), created.ID)
		return "", "", err
	}
	return created.ID, endpoint, nil
}

// newDevToolsToken mints the per-session secret for the DevTools forwarder.
func newDevToolsToken() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand does not fail in practice; a zero token would be a
		// worse outcome than refusing to start, so make it obviously unusable.
		return ""
	}
	return hex.EncodeToString(b)
}

// browserBody builds the Docker create request for a browser.
// Split out so the token wiring and hardening can be asserted without a daemon.
func browserBody(image, token string) map[string]interface{} {
	return map[string]interface{}{
		"Image": image,
		"Env":   []string{"APPLAD_DEVTOOLS_TOKEN=" + token},
		"Labels": map[string]string{
			"applad.managed": "true",
			"applad.type":    "browser",
		},
		"ExposedPorts": map[string]interface{}{"9222/tcp": struct{}{}},
		// On the deploy network, so a session can record against an app
		// deployed here by name.
		"NetworkingConfig": map[string]interface{}{
			"EndpointsConfig": map[string]interface{}{
				deployNetworkName(): map[string]interface{}{},
			},
		},
		"HostConfig": map[string]interface{}{
			"Memory":      int64(1536 * 1024 * 1024),
			"NanoCPUs":    int64(2e9),
			"NetworkMode": deployNetworkName(),
			"SecurityOpt": []string{"no-new-privileges"},
			// Chromium runs with --no-sandbox, so it needs no Linux capabilities;
			// drop them all and bound process count to match the function/deploy
			// containers rather than running the browser at full privilege.
			"CapDrop":   []string{"ALL"},
			"PidsLimit": int64(512),
			// A browser that outlives its session would hold a gigabyte for
			// nothing, so it is never restarted.
			"RestartPolicy": map[string]interface{}{"Name": "no"},
		},
	}
}

// waitForDevTools polls the browser until it reports a debuggable page.
// Chromium accepts connections a moment before it has a target to attach to,
// so asking for the endpoint is the only reliable readiness check.
func (d *DeployExecutor) waitForDevTools(ctx context.Context, host, token string) (string, error) {
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
		// Authenticate to the forwarder that guards the port.
		req.Header.Set(devToolsTokenHeader, token)

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
						// reach it by container name on our shared network, and
						// carry the token so the forwarder relays the socket.
						return withDevToolsToken(rewriteDevToolsHost(t.WebSocketDebuggerURL, host), token), nil
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

// withDevToolsToken appends the forwarder token to a DevTools WebSocket URL as
// a query parameter, so the CDP client authenticates simply by dialing it.
func withDevToolsToken(wsURL, token string) string {
	if token == "" {
		return wsURL
	}
	sep := "?"
	if strings.Contains(wsURL, "?") {
		sep = "&"
	}
	return wsURL + sep + devToolsTokenParam + "=" + token
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

// ReapStaleBrowsers removes browsers older than maxAge.
//
// A caller stops the browser it started, but that only holds while the caller
// is alive: a worker killed mid-capture leaves a Chromium running and nothing
// would ever clean it up. Several were found up for more than a day. This is
// the backstop that does not depend on anyone remembering — a capture takes
// seconds, so a container older than maxAge is abandoned by definition.
func (d *DeployExecutor) ReapStaleBrowsers(ctx context.Context, maxAge time.Duration) (int, error) {
	filter := url.QueryEscape(`{"name":["applad-browser-"]}`)
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
			slog.Warn("runtime: could not reap browser", "container", c.ID, "error", err)
			continue
		}
		slog.Info("runtime: reaped stale browser", "names", c.Names)
		reaped++
	}
	return reaped, nil
}
