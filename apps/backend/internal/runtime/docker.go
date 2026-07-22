// Package runtime provides a container-based function execution engine.
// It talks to the Docker Engine API via Unix socket to build images,
// manage a warm container pool, and route HTTP invocations to containers.
//
// Functions are OCI containers that listen on a port. The convention:
//   - Container exposes an HTTP server on port 3000
//   - POST / with JSON body → function receives the invocation
//   - Response body = function output
//
// For simple functions (source code + runtime), Applad auto-generates
// a Dockerfile from built-in templates. For advanced functions, users
// provide their own Dockerfile.
package runtime

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client talks to the Docker Engine API.
// By default it connects via the Unix socket at /var/run/docker.sock.
// Set DOCKER_HOST to override: unix:///path/to/docker.sock or tcp://host:port.
type Client struct {
	httpClient *http.Client
	baseURL    string // e.g. "http://docker" (unix) or "http://host:2375" (tcp)
}

// NewClient creates a Docker API client, honouring DOCKER_HOST.
func NewClient() *Client {
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		dockerHost = "unix:///var/run/docker.sock"
	}

	var (
		transport *http.Transport
		baseURL   string
	)

	if strings.HasPrefix(dockerHost, "unix://") {
		socketPath := strings.TrimPrefix(dockerHost, "unix://")
		transport = &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		}
		baseURL = "http://docker"
	} else {
		// tcp:// or plain host:port
		addr := strings.TrimPrefix(dockerHost, "tcp://")
		transport = &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "tcp", addr)
			},
		}
		baseURL = "http://" + addr
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   5 * time.Minute,
		},
		baseURL: baseURL,
	}
}

// --- Image operations ---

// BuildImage builds a Docker image from a tar context (Dockerfile + source).
//
// The build output is returned whether or not the build succeeded. Discarding
// it on success meant a deploy that worked left no record of what it did, so
// there was nothing to compare against when the next one behaved differently.
func (c *Client) BuildImage(ctx context.Context, imageName string, tarContext io.Reader) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf(c.baseURL+"/v1.44/build?t=%s&rm=true&forcerm=true", imageName),
		tarContext)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-tar")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("docker build: %w", err)
	}
	defer resp.Body.Close()

	// Read build output to completion
	output, _ := io.ReadAll(resp.Body)
	log := buildStreamText(output)

	if resp.StatusCode != http.StatusOK {
		return log, fmt.Errorf("docker build failed (%d): %s", resp.StatusCode, log)
	}

	// Check for error in build stream. The log is returned alongside, so the
	// error carries only the explanation rather than a copy of the stream.
	if bytes.Contains(output, []byte(`"error"`)) {
		return log, errors.New(SummariseBuildFailure(log))
	}

	return log, nil
}

// RemoveImage removes a Docker image.
func (c *Client) RemoveImage(ctx context.Context, imageName string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE",
		fmt.Sprintf(c.baseURL+"/v1.44/images/%s?force=true", imageName), nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// --- Container operations ---

// ContainerConfig holds config for creating a container.
type ContainerConfig struct {
	Image   string
	Env     []string
	Labels  map[string]string
	Timeout int // seconds
}

// CreateContainer creates and starts a container, returning its ID.
// Containers are hardened by default: no privilege escalation, read-only rootfs
// (with /tmp writable), no capabilities, isolated PID namespace, and restricted
// syscalls via the default seccomp profile.
func (c *Client) CreateContainer(ctx context.Context, name string, cfg ContainerConfig) (string, error) {
	body := map[string]interface{}{
		"Image":  cfg.Image,
		"Env":    cfg.Env,
		"Labels": cfg.Labels,
		"ExposedPorts": map[string]interface{}{
			"3000/tcp": struct{}{},
		},
		"HostConfig": map[string]interface{}{
			"PublishAllPorts": true,
			"Memory":          int64(256 * 1024 * 1024), // 256MB
			"MemorySwap":      int64(256 * 1024 * 1024), // no swap
			"NanoCPUs":        int64(1e9),               // 1 CPU
			"PidsLimit":       int64(256),               // limit process count
			"NetworkMode":     "bridge",
			"ReadonlyRootfs":  true,
			"SecurityOpt":     []string{"no-new-privileges"},
			"CapDrop":         []string{"ALL"},
			"Tmpfs": map[string]string{
				"/tmp": "rw,noexec,nosuid,size=64m",
			},
		},
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf(c.baseURL+"/v1.44/containers/create?name=%s", name),
		bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("docker create: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("docker create failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID string `json:"Id"`
	}
	json.Unmarshal(respBody, &result)
	return result.ID, nil
}

// StartContainer starts an existing container.
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf(c.baseURL+"/v1.44/containers/%s/start", containerID), nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	// Docker says why in the body. Reporting only the status code turned
	// "the image has no /bin/bash" into "docker start failed: 400", which is
	// a number where the answer was.
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotModified {
		return fmt.Errorf("the container would not start: %s", dockerMessage(body, resp.StatusCode))
	}
	return nil
}

// dockerMessage pulls the human sentence out of a Docker error body.
func dockerMessage(body []byte, status int) string {
	var out struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &out)
	msg := strings.TrimSpace(out.Message)
	if msg == "" {
		msg = strings.TrimSpace(string(body))
	}
	if msg == "" {
		return fmt.Sprintf("HTTP %d", status)
	}
	if len(msg) > 300 {
		msg = msg[:300]
	}
	return msg
}

// StopContainer stops a running container.
func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf(c.baseURL+"/v1.44/containers/%s/stop?t=5", containerID), nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// RemoveContainer removes a container.
func (c *Client) RemoveContainer(ctx context.Context, containerID string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE",
		fmt.Sprintf(c.baseURL+"/v1.44/containers/%s?force=true&v=true", containerID), nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// GetContainerPort returns the host-mapped port for a container's internal port.
func (c *Client) GetContainerPort(ctx context.Context, containerID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf(c.baseURL+"/v1.44/containers/%s/json", containerID), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var info struct {
		NetworkSettings struct {
			Ports map[string][]struct {
				HostPort string `json:"HostPort"`
			} `json:"Ports"`
			Networks map[string]struct {
				IPAddress string `json:"IPAddress"`
			} `json:"Networks"`
		} `json:"NetworkSettings"`
	}
	json.Unmarshal(body, &info)

	// Try mapped port first
	if bindings, ok := info.NetworkSettings.Ports["3000/tcp"]; ok && len(bindings) > 0 {
		return bindings[0].HostPort, nil
	}

	// Fall back to container IP on bridge network
	for _, net := range info.NetworkSettings.Networks {
		if net.IPAddress != "" {
			return net.IPAddress + ":3000", nil
		}
	}

	return "", fmt.Errorf("no port mapping found for container %s", containerID)
}

// ContainerLogs returns the combined stdout/stderr logs of a container.
// FollowContainerLogs streams a running container's output line by line until
// it exits. Waiting for a container to finish before showing anything means
// staring at nothing for minutes while dependencies install; this is what
// makes a run watchable while it happens.
func (c *Client) FollowContainerLogs(ctx context.Context, containerID string, onLine func(string)) error {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf(c.baseURL+"/v1.44/containers/%s/logs?stdout=true&stderr=true&follow=true", containerID), nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		// Each frame carries an 8-byte header: stream id, three reserved
		// bytes, then a big-endian length.
		var header [8]byte
		if _, err := io.ReadFull(reader, header[:]); err != nil {
			return nil // the container exited, or the context was cancelled
		}
		size := binary.BigEndian.Uint32(header[4:8])
		if size == 0 {
			continue
		}
		if size > 1<<20 {
			size = 1 << 20
		}
		payload := make([]byte, size)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return nil
		}
		for _, line := range strings.Split(strings.TrimRight(string(payload), "\n"), "\n") {
			if line != "" {
				onLine(line)
			}
		}
	}
}

func (c *Client) ContainerLogs(ctx context.Context, containerID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf(c.baseURL+"/v1.44/containers/%s/logs?stdout=true&stderr=true", containerID), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	// Docker log stream has 8-byte header per frame; strip them for readability
	return stripDockerLogHeaders(body), nil
}

// WaitContainer waits for a container to be in a "not-running" state.
func (c *Client) WaitContainer(ctx context.Context, containerID string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf(c.baseURL+"/v1.44/containers/%s/wait", containerID), nil)
	if err != nil {
		return -1, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	var result struct {
		StatusCode int `json:"StatusCode"`
	}
	json.Unmarshal(mustReadAll(resp.Body), &result)
	return result.StatusCode, nil
}

func mustReadAll(r io.Reader) []byte {
	b, _ := io.ReadAll(r)
	return b
}

func stripDockerLogHeaders(data []byte) string {
	var result strings.Builder
	for len(data) >= 8 {
		// Each frame: [stream_type(1), 0, 0, 0, size(4 big-endian), payload(size)]
		size := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])
		data = data[8:]
		if size > len(data) {
			size = len(data)
		}
		result.Write(data[:size])
		data = data[size:]
	}
	if result.Len() == 0 {
		return string(data) // fallback if no headers
	}
	return result.String()
}

// buildStreamText renders Docker's newline-delimited JSON build output as the
// plain build log a person would see in a terminal. Storing the raw stream
// made failures unreadable in the console.
func buildStreamText(output []byte) string {
	var b strings.Builder
	for _, line := range bytes.Split(output, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var msg struct {
			Stream string `json:"stream"`
			Error  string `json:"error"`
			Status string `json:"status"`
		}
		if json.Unmarshal(line, &msg) != nil {
			continue
		}
		switch {
		case msg.Error != "":
			b.WriteString(msg.Error)
			b.WriteByte('\n')
		case msg.Stream != "":
			// Progress noise (layer pulls) carries a status, not a stream.
			b.WriteString(stripANSI(msg.Stream))
		}
	}
	return strings.TrimSpace(b.String())
}

// stripANSI removes terminal colour codes so logs read cleanly in the console.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
