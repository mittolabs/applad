package runtime

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DeployConfig holds the configuration extracted from a deployment's config map.
type DeployConfig struct {
	Dockerfile   string            // Dockerfile content (for web type)
	BuildCommand string            // build command to run inside the image
	Port         string            // port the app listens on (default "3000")
	Image        string            // image to pull (for container type)
	Env          map[string]string // environment variables
	SourceDir    string            // checked-out source to build from
	OutputDir    string            // directory of build artefacts to serve (static sites)
	Subdomain    string            // <sub>.applad.dev this app is served on
}

// ParseDeployConfig extracts a DeployConfig from the raw config map.
func ParseDeployConfig(raw map[string]interface{}) DeployConfig {
	cfg := DeployConfig{Port: "3000"}

	if v, ok := raw["dockerfile"].(string); ok {
		cfg.Dockerfile = v
	}
	// The release worker sends "buildCmd"; accept both spellings.
	if v, ok := raw["buildCommand"].(string); ok {
		cfg.BuildCommand = v
	}
	if v, ok := raw["buildCmd"].(string); ok && v != "" {
		cfg.BuildCommand = v
	}
	if v, ok := raw["sourceDir"].(string); ok {
		cfg.SourceDir = v
	}
	if v, ok := raw["outputDir"].(string); ok {
		cfg.OutputDir = v
	}
	if v, ok := raw["subdomain"].(string); ok {
		cfg.Subdomain = v
	}
	if v, ok := raw["port"].(string); ok && v != "" {
		cfg.Port = v
	}
	if v, ok := raw["image"].(string); ok {
		cfg.Image = v
	}
	if v, ok := raw["env"].(map[string]interface{}); ok {
		cfg.Env = make(map[string]string, len(v))
		for k, val := range v {
			cfg.Env[k] = fmt.Sprintf("%v", val)
		}
	}
	if cfg.Env == nil {
		cfg.Env = map[string]string{}
	}
	return cfg
}

// DeployExecutor builds and manages deployment containers using the Docker API.
type DeployExecutor struct {
	docker     *Client
	mu         sync.RWMutex
	containers map[string]string // deploymentID -> containerID
}

// NewDeployExecutor creates a deployment executor.
func NewDeployExecutor() *DeployExecutor {
	return &DeployExecutor{
		docker:     NewClient(),
		containers: make(map[string]string),
	}
}

// DeployWeb builds a Docker image from the deployment config and starts a container.
// This is used for "web" type deployments where the user provides a Dockerfile or
// build command. The image is tagged as applad-deploy-{deploymentID}.
func (d *DeployExecutor) DeployWeb(ctx context.Context, deploymentID, projectID string, cfg DeployConfig) error {
	imageName := fmt.Sprintf("applad-deploy-%s", deploymentID)

	// Does the checked-out source ship its own Dockerfile?
	repoDockerfile := false
	if cfg.SourceDir != "" {
		if _, err := os.Stat(filepath.Join(cfg.SourceDir, "Dockerfile")); err == nil {
			repoDockerfile = true
		}
	}

	// Decide how to containerise the app:
	//   1. an explicit Dockerfile in the config wins
	//   2. otherwise the repo's own Dockerfile is used as-is
	//   3. otherwise a build command implies a Node app
	//   4. otherwise it's a static site — serve it with nginx
	dockerfile := cfg.Dockerfile
	switch {
	case dockerfile != "":
		// use as provided
	case repoDockerfile:
		// leave empty; the repo's Dockerfile is already in the build context
	case cfg.BuildCommand != "":
		dockerfile = fmt.Sprintf(`FROM node:20-alpine
WORKDIR /app
COPY . .
RUN %s
EXPOSE %s
CMD ["node", "server.js"]
`, cfg.BuildCommand, cfg.Port)
	default:
		root := strings.Trim(cfg.OutputDir, "/ ")
		if root == "" || root == "." {
			root = "."
		}
		// The generated Dockerfile sits in the build context root, so strip it
		// (and any VCS/config leftovers) back out of the served web root.
		dockerfile = fmt.Sprintf(`FROM nginx:alpine
COPY %s/ /usr/share/nginx/html/
RUN rm -f /usr/share/nginx/html/Dockerfile /usr/share/nginx/html/.gitignore \
    /usr/share/nginx/html/.env
EXPOSE 80
`, root)
		cfg.Port = "80"
	}

	if cfg.SourceDir == "" && dockerfile == "" {
		return fmt.Errorf("deploy: web deployment requires a source, dockerfile or buildCommand")
	}

	// Build a tar context from the checked-out source plus the Dockerfile.
	// Without the source the image would be built from an empty context and
	// every COPY would silently produce nothing.
	tarBuf := new(bytes.Buffer)
	tw := tar.NewWriter(tarBuf)
	if cfg.SourceDir != "" {
		skipDockerfile := dockerfile != ""
		if err := addDirToTar(tw, cfg.SourceDir, skipDockerfile); err != nil {
			tw.Close()
			return fmt.Errorf("deploy: build context: %w", err)
		}
	}
	if dockerfile != "" {
		addToTar(tw, "Dockerfile", []byte(dockerfile))
	}
	tw.Close()

	log.Printf("deploy: building image %s for deployment %s", imageName, deploymentID)
	if err := d.docker.BuildImage(ctx, imageName, tarBuf); err != nil {
		return fmt.Errorf("deploy: image build failed: %w", err)
	}

	// Start the container
	return d.startContainer(ctx, deploymentID, projectID, imageName, cfg)
}

// addDirToTar walks a source directory into a Docker build context. Version
// control and dependency directories are skipped: they bloat the context and
// are never needed to build. Set skipDockerfile when a generated Dockerfile
// will be written separately, so the two don't collide.
func addDirToTar(tw *tar.Writer, dir string, skipDockerfile bool) error {
	skipDirs := map[string]bool{".git": true, "node_modules": true, ".next": true, "vendor/bundle": true}

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil || rel == "." {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil // skip symlinks, sockets, devices
		}
		if skipDockerfile && rel == "Dockerfile" {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: filepath.ToSlash(rel),
			Mode: int64(info.Mode().Perm()),
			Size: int64(len(data)),
		}); err != nil {
			return err
		}
		_, err = tw.Write(data)
		return err
	})
}

// DeployContainer pulls (or uses) the specified image and starts a container.
// This is used for "container" type deployments where the user specifies a
// pre-built image to run.
func (d *DeployExecutor) DeployContainer(ctx context.Context, deploymentID, projectID string, cfg DeployConfig) error {
	image := cfg.Image
	if image == "" {
		// Fall back to the applad-deploy image if no external image specified
		image = fmt.Sprintf("applad-deploy-%s", deploymentID)
	} else {
		// Pull the image
		log.Printf("deploy: pulling image %s for deployment %s", image, deploymentID)
		if err := d.pullImage(ctx, image); err != nil {
			return fmt.Errorf("deploy: image pull failed: %w", err)
		}
	}

	return d.startContainer(ctx, deploymentID, projectID, image, cfg)
}

// StopDeployment stops and removes the container for a deployment.
func (d *DeployExecutor) StopDeployment(ctx context.Context, deploymentID string) error {
	d.mu.Lock()
	containerID, ok := d.containers[deploymentID]
	if ok {
		delete(d.containers, deploymentID)
	}
	d.mu.Unlock()

	if !ok {
		// Try to find the container by name convention
		containerID = d.findContainerByName(ctx, fmt.Sprintf("applad-deploy-%s", deploymentID))
		if containerID == "" {
			log.Printf("deploy: no container found for deployment %s", deploymentID)
			return nil
		}
	}

	log.Printf("deploy: stopping container %s for deployment %s", containerID[:12], deploymentID)
	if err := d.docker.StopContainer(ctx, containerID); err != nil {
		log.Printf("deploy: stop warning: %v", err)
	}
	if err := d.docker.RemoveContainer(ctx, containerID); err != nil {
		return fmt.Errorf("deploy: remove container: %w", err)
	}
	return nil
}

// StopByName stops and removes a container by its name. Used to tear down a
// deployed app when its target is deleted, where the deployment ID that
// created the container is no longer known.
func (d *DeployExecutor) StopByName(ctx context.Context, name string) error {
	containerID := d.findContainerByName(ctx, name)
	if containerID == "" {
		return nil // nothing running under that name
	}
	if err := d.docker.StopContainer(ctx, containerID); err != nil {
		log.Printf("deploy: stop %s warning: %v", name, err)
	}
	if err := d.docker.RemoveContainer(ctx, containerID); err != nil {
		return fmt.Errorf("deploy: remove container %s: %w", name, err)
	}
	log.Printf("deploy: removed container %s", name)
	return nil
}

// startContainer creates and starts a container for the given deployment.
func (d *DeployExecutor) startContainer(ctx context.Context, deploymentID, projectID, imageName string, cfg DeployConfig) error {
	// Stop any existing container for this deployment
	d.StopDeployment(ctx, deploymentID)

	// Apps are addressed by subdomain: the ingress proxies
	// <sub>.applad.dev -> applad-site-<sub>, so the container name IS the route.
	// Without a subdomain fall back to the deployment id (not routable).
	containerName := fmt.Sprintf("applad-deploy-%s", deploymentID)
	if cfg.Subdomain != "" {
		containerName = fmt.Sprintf("applad-site-%s", cfg.Subdomain)
	}

	// Build environment variables
	var env []string
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	env = append(env, fmt.Sprintf("PORT=%s", cfg.Port))
	env = append(env, fmt.Sprintf("APPLAD_DEPLOYMENT_ID=%s", deploymentID))
	env = append(env, fmt.Sprintf("APPLAD_PROJECT_ID=%s", projectID))

	// Create container with the specified port
	containerID, err := d.createDeployContainer(ctx, containerName, imageName, cfg.Port, env, map[string]string{
		"applad.deployment": deploymentID,
		"applad.project":    projectID,
		"applad.managed":    "true",
		"applad.type":       "deployment",
	})
	if err != nil {
		return fmt.Errorf("deploy: create container: %w", err)
	}

	if err := d.docker.StartContainer(ctx, containerID); err != nil {
		d.docker.RemoveContainer(ctx, containerID)
		return fmt.Errorf("deploy: start container: %w", err)
	}

	// Store the mapping
	d.mu.Lock()
	d.containers[deploymentID] = containerID
	d.mu.Unlock()

	// Wait briefly for the container to become healthy
	if err := d.waitForHealthy(ctx, containerName, cfg.Port); err != nil {
		log.Printf("deploy: container started but health check failed (non-fatal): %v", err)
	}

	log.Printf("deploy: container %s started for deployment %s", containerID[:12], deploymentID)
	return nil
}

// createDeployContainer creates a container with explicit port mapping
// instead of PublishAllPorts, so the deployment gets a predictable port.
func (d *DeployExecutor) createDeployContainer(ctx context.Context, name, image, port string, env []string, labels map[string]string) (string, error) {
	exposedPort := port + "/tcp"

	// Deployed apps must sit on the same user-defined network as the proxy, so
	// the ingress can reach them by container name via Docker's embedded DNS.
	// The default bridge has no name resolution, which makes apps unroutable.
	network := os.Getenv("APPLAD_DEPLOY_NETWORK")
	if network == "" {
		network = "applad_default"
	}

	body := map[string]interface{}{
		"Image":  image,
		"Env":    env,
		"Labels": labels,
		"ExposedPorts": map[string]interface{}{
			exposedPort: struct{}{},
		},
		"HostConfig": map[string]interface{}{
			"PublishAllPorts": true,
			"Memory":          int64(512 * 1024 * 1024), // 512MB for deployments
			"NanoCPUs":        int64(2e9),               // 2 CPUs
			"NetworkMode":     network,
			// Deployed apps are long-lived: they must come back after a host or
			// Docker daemon restart, not just after a crash.
			"RestartPolicy": map[string]interface{}{
				"Name": "unless-stopped",
			},
		},
		"NetworkingConfig": map[string]interface{}{
			"EndpointsConfig": map[string]interface{}{
				network: map[string]interface{}{"Aliases": []string{name}},
			},
		},
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf(d.docker.baseURL+"/v1.44/containers/create?name=%s", name),
		bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.docker.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("docker create: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// If a container with this name already exists, remove it and retry
	if resp.StatusCode == http.StatusConflict {
		old := d.findContainerByName(ctx, name)
		if old != "" {
			d.docker.StopContainer(ctx, old)
			d.docker.RemoveContainer(ctx, old)
		}
		// Retry
		return d.createDeployContainer(ctx, name, image, port, env, labels)
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("docker create failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID string `json:"Id"`
	}
	json.Unmarshal(respBody, &result)
	return result.ID, nil
}

// pullImage pulls a Docker image from a registry.
func (d *DeployExecutor) pullImage(ctx context.Context, image string) error {
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf(d.docker.baseURL+"/v1.44/images/create?fromImage=%s", image), nil)
	if err != nil {
		return err
	}

	resp, err := d.docker.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("docker pull: %w", err)
	}
	defer resp.Body.Close()

	// Read the pull output to completion (streaming JSON)
	output, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("docker pull failed (%d): %s", resp.StatusCode, string(output))
	}

	// Check for error in pull stream
	if bytes.Contains(output, []byte(`"error"`)) {
		return fmt.Errorf("docker pull error: %s", string(output))
	}

	return nil
}

// findContainerByName looks up a container ID by name using the Docker API.
func (d *DeployExecutor) findContainerByName(ctx context.Context, name string) string {
	// The filter must be URL-encoded: sent raw, Docker parses it as an empty
	// filter set and the lookup silently matches nothing.
	filter := url.QueryEscape(fmt.Sprintf(`{"name":["%s"]}`, name))
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf(d.docker.baseURL+"/v1.44/containers/json?all=true&filters=%s", filter), nil)
	if err != nil {
		return ""
	}

	resp, err := d.docker.httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var containers []struct {
		ID    string   `json:"Id"`
		Names []string `json:"Names"`
	}
	json.Unmarshal(body, &containers)

	// Match by exact name (Docker prepends /)
	target := "/" + name
	for _, c := range containers {
		for _, n := range c.Names {
			if n == target || strings.TrimPrefix(n, "/") == name {
				return c.ID
			}
		}
	}
	return ""
}

// waitForHealthy polls the container until it responds to an HTTP request.
func (d *DeployExecutor) waitForHealthy(ctx context.Context, containerName, port string) error {
	// Probe the container over the shared Docker network by its name alias.
	// Probing localhost:<published port> only works from the Docker host; from
	// inside this worker container "localhost" is the worker itself, so the
	// check timed out on every single deploy.
	addr := fmt.Sprintf("http://%s:%s/", containerName, port)

	deadline := time.After(30 * time.Second)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()

	client := &http.Client{Timeout: 2 * time.Second}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("timeout waiting for %s to respond on port %s", containerName, port)
		case <-tick.C:
			resp, err := client.Get(addr)
			if err != nil {
				continue
			}
			resp.Body.Close()
			return nil
		}
	}
}

// GetContainerID returns the container ID for a deployment, if tracked.
func (d *DeployExecutor) GetContainerID(deploymentID string) (string, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	id, ok := d.containers[deploymentID]
	return id, ok
}
