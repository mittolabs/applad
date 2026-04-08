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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const dockerSocket = "/var/run/docker.sock"

// Client talks to the Docker Engine API via Unix socket.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a Docker API client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", dockerSocket)
				},
			},
			Timeout: 5 * time.Minute,
		},
	}
}

// --- Image operations ---

// BuildImage builds a Docker image from a tar context (Dockerfile + source).
// Returns the image ID.
func (c *Client) BuildImage(ctx context.Context, imageName string, tarContext io.Reader) error {
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("http://docker/v1.43/build?t=%s&rm=true&forcerm=true", imageName),
		tarContext)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-tar")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("docker build: %w", err)
	}
	defer resp.Body.Close()

	// Read build output to completion
	output, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("docker build failed (%d): %s", resp.StatusCode, string(output))
	}

	// Check for error in build stream
	if bytes.Contains(output, []byte(`"error"`)) {
		return fmt.Errorf("docker build error: %s", string(output))
	}

	return nil
}

// RemoveImage removes a Docker image.
func (c *Client) RemoveImage(ctx context.Context, imageName string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE",
		fmt.Sprintf("http://docker/v1.43/images/%s?force=true", imageName), nil)
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
			"Memory":         int64(256 * 1024 * 1024), // 256MB
			"MemorySwap":     int64(256 * 1024 * 1024), // no swap
			"NanoCPUs":       int64(1e9),                // 1 CPU
			"PidsLimit":      int64(256),                // limit process count
			"NetworkMode":    "bridge",
			"ReadonlyRootfs": true,
			"SecurityOpt":    []string{"no-new-privileges"},
			"CapDrop":        []string{"ALL"},
			"Tmpfs": map[string]string{
				"/tmp": "rw,noexec,nosuid,size=64m",
			},
		},
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("http://docker/v1.43/containers/create?name=%s", name),
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
		fmt.Sprintf("http://docker/v1.43/containers/%s/start", containerID), nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotModified {
		return fmt.Errorf("docker start failed: %d", resp.StatusCode)
	}
	return nil
}

// StopContainer stops a running container.
func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("http://docker/v1.43/containers/%s/stop?t=5", containerID), nil)
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
		fmt.Sprintf("http://docker/v1.43/containers/%s?force=true&v=true", containerID), nil)
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
		fmt.Sprintf("http://docker/v1.43/containers/%s/json", containerID), nil)
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
func (c *Client) ContainerLogs(ctx context.Context, containerID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("http://docker/v1.43/containers/%s/logs?stdout=true&stderr=true", containerID), nil)
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
		fmt.Sprintf("http://docker/v1.43/containers/%s/wait", containerID), nil)
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
