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
	Dockerfile     string            // Dockerfile content (for web type)
	InstallCommand string            // dependency install, run before the build
	BuildCommand   string            // build command to run inside the image
	StartCommand   string            // long-lived process, for ServeMode "node"
	Port           string            // port the app listens on (default "3000")
	Image          string            // image to pull (for container type)
	Env            map[string]string // environment variables
	SourceDir      string            // checked-out source to build from
	OutputDir      string            // directory of build artefacts to serve (static sites)
	Subdomain      string            // <sub>.applad.dev this app is served on
	ServeMode      string            // "static" (serve OutputDir) or "node" (run a server)
	NodeVersion    string            // major Node version for the build image
	// PackageManagerPin is package.json's own "packageManager" field, used so
	// the build runs the version the project chose rather than the newest.
	PackageManagerPin string
	// Platform forces the build architecture when a toolchain ships for only
	// one. Empty means the machine's own.
	Platform string

	// RailpackConfig describes the build for a project the builder has no
	// provider for — Flutter, so far.
	RailpackConfig string

	// LogSink, if set, receives each build output line as it is produced, so the
	// caller can stream progress live. It must be safe to call from the build
	// goroutine and must not block.
	LogSink func(string)
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
	if v, ok := raw["installCmd"].(string); ok {
		cfg.InstallCommand = v
	}
	if v, ok := raw["startCmd"].(string); ok {
		cfg.StartCommand = v
	}
	if v, ok := raw["packageManagerPin"].(string); ok {
		cfg.PackageManagerPin = v
	}
	if v, ok := raw["railpackConfig"].(string); ok {
		cfg.RailpackConfig = v
	}
	if v, ok := raw["platform"].(string); ok {
		cfg.Platform = v
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
	if v, ok := raw["serveMode"].(string); ok {
		cfg.ServeMode = v
	}
	if v, ok := raw["nodeVersion"].(string); ok {
		cfg.NodeVersion = v
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
func (d *DeployExecutor) DeployWeb(ctx context.Context, deploymentID, projectID string, cfg DeployConfig) (string, error) {
	imageName := fmt.Sprintf("applad-deploy-%s", deploymentID)

	// Does the checked-out source ship its own Dockerfile?
	repoDockerfile := false
	if cfg.SourceDir != "" {
		if _, err := os.Stat(filepath.Join(cfg.SourceDir, "Dockerfile")); err == nil {
			repoDockerfile = true
		}
	}

	// Every deployed site answers on 80, whatever it is built from.
	//
	// The ingress proxies <sub>.applad.dev to applad-site-<sub>:80, so a site
	// listening anywhere else is unreachable no matter how well it built.
	// PORT is passed to the container, and both Caddy and every framework
	// railpack generates for honour it.
	cfg.Port = "80"

	// Decide how to containerise the app:
	//   1. an explicit Dockerfile in the config wins
	//   2. otherwise the repo's own Dockerfile is used as-is — somebody who
	//      wrote one meant it, and second-guessing it is not our place
	//   3. otherwise railpack reads the source and builds it
	//
	// Only the first two produce a Dockerfile here. The third is a different
	// kind of build entirely and returns early.
	if cfg.Dockerfile == "" && !repoDockerfile {
		buildLog, err := d.BuildWithRailpack(ctx, RailpackBuild{
			SourceDir:  cfg.SourceDir,
			ImageName:  imageName,
			InstallCmd: cfg.InstallCommand,
			BuildCmd:   cfg.BuildCommand,
			StartCmd:   cfg.StartCommand,
			CacheKey:   cfg.Subdomain,
			Config:     cfg.RailpackConfig,
			Platform:   cfg.Platform,
			Env:        cfg.Env,
			Sink:       cfg.LogSink,
		})
		if err != nil {
			return buildLog, err
		}
		return buildLog, d.startContainer(ctx, deploymentID, projectID, imageName, cfg)
	}

	dockerfile := cfg.Dockerfile
	// Set when the generated image expects our nginx log config in the build
	// context.
	nginxConf := false
	switch {
	case dockerfile != "":
		// use as provided
	case repoDockerfile:
		// leave empty; the repo's Dockerfile is already in the build context
	case (cfg.BuildCommand != "" || cfg.InstallCommand != "") && cfg.ServeMode != "node":
		// Most frameworks build to a directory of static files. Build in Node,
		// then serve the output with nginx: the result is a ~50MB image with no
		// runtime, instead of dragging the whole toolchain into production.
		out := strings.Trim(cfg.OutputDir, "/ ")
		if out == "" || out == "." {
			out = "dist"
		}
		dockerfile = fmt.Sprintf(`FROM node:%s-alpine AS build
WORKDIR /app
%s
FROM nginx:alpine
COPY --from=build /app/%s/ /usr/share/nginx/html/
EXPOSE 80
`, nodeTag(cfg.NodeVersion), buildPhases(cfg), out)
		cfg.Port = "80"

	case cfg.BuildCommand != "" || cfg.InstallCommand != "":
		dockerfile = fmt.Sprintf(`FROM node:%s-alpine
WORKDIR /app
%s
EXPOSE %s
CMD %s
`, nodeTag(cfg.NodeVersion), buildPhases(cfg), cfg.Port, startCommand(cfg))
	default:
		root := strings.Trim(cfg.OutputDir, "/ ")
		if root == "" || root == "." {
			root = "."
		}
		// The generated Dockerfile sits in the build context root, so strip it
		// (and any VCS/config leftovers) back out of the served web root.
		// The log config is copied in as a file rather than echoed by a RUN.
		// Escaping nginx's own $variables through a shell inside a Dockerfile
		// is how the first attempt broke the build.
		nginxConf = true
		dockerfile = fmt.Sprintf(`FROM nginx:alpine
COPY applad-log.conf /etc/nginx/conf.d/applad-log.conf
COPY %s/ /usr/share/nginx/html/
RUN rm -f /usr/share/nginx/html/Dockerfile /usr/share/nginx/html/.gitignore \
    /usr/share/nginx/html/.env
EXPOSE 80
`, root)
		cfg.Port = "80"
	}

	if cfg.SourceDir == "" && dockerfile == "" {
		return "", fmt.Errorf("deploy: web deployment requires a source, dockerfile or buildCommand")
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
			return "", fmt.Errorf("deploy: build context: %w", err)
		}
	}
	if dockerfile != "" {
		addToTar(tw, "Dockerfile", []byte(dockerfile))
	}
	if nginxConf {
		addToTar(tw, "applad-log.conf", []byte(accessLogConf))
	}
	tw.Close()

	log.Printf("deploy: building image %s for deployment %s", imageName, deploymentID)
	buildLog, err := d.docker.BuildImageSink(ctx, imageName, tarBuf, cfg.LogSink)
	if err != nil {
		// Already a sentence about what failed — wrapping it again would only
		// prefix Go package names onto something a person has to read.
		return buildLog, err
	}

	// Start the container
	return buildLog, d.startContainer(ctx, deploymentID, projectID, imageName, cfg)
}

// buildPhases writes the install and build steps as separate layers.
//
// The manifest is copied and dependencies installed before the source is
// copied in, so a change to the source reuses the cached install rather than
// reinstalling every dependency on every deploy. It also fixes the reason a
// build could fail outright: a single conflated command meant naming a build
// step replaced the install that has to precede it, and `next build` ran in a
// tree with no node_modules.
func buildPhases(cfg DeployConfig) string {
	var b strings.Builder

	// Setup: the base image ships npm and nothing else, so a project using
	// pnpm or yarn needs its package manager put there first. Corepack is how
	// Node distributes them, and it reads the version from packageManager in
	// package.json, so the build uses what the project pinned.
	if pm := packageManagerOf(cfg.InstallCommand); pm != "" && pm != "npm" {
		b.WriteString("ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0\n")
		// A pinned version is used exactly; an unpinned one is pinned here
		// rather than resolved to "latest" at build time. Letting corepack
		// pick the newest release means a project that has not changed starts
		// failing the day that tool changes a default — which it did: pnpm 11
		// turned unapproved dependency build scripts into an error.
		if pin := strings.TrimSpace(cfg.PackageManagerPin); pin != "" {
			fmt.Fprintf(&b, "RUN corepack prepare %s --activate\n", pin)
		} else if fallback := defaultPackageManagerVersion[pm]; fallback != "" {
			fmt.Fprintf(&b, "RUN corepack prepare %s --activate\n", fallback)
		} else {
			b.WriteString("RUN corepack enable\n")
		}
		b.WriteString("RUN corepack enable\n")
	}

	if cfg.InstallCommand != "" {
		// Only the manifest and lockfile, so this layer survives source edits.
		// package.json is always present when there is something to install;
		// the rest are globs because a project has exactly one of them, and
		// Docker only requires that some source matches.
		b.WriteString("COPY package.json package-lock.json* pnpm-lock.yaml* yarn.lock* ./\n")
		fmt.Fprintf(&b, "RUN %s\n", cfg.InstallCommand)
	}

	b.WriteString("COPY . .\n")
	if cfg.BuildCommand != "" {
		fmt.Fprintf(&b, "RUN %s\n", cfg.BuildCommand)
	}
	return b.String()
}

// defaultPackageManagerVersion is what a project gets when it pins nothing.
//
// Chosen rather than inferred: "latest" is not a version, it is a moving
// target that turns somebody else's release into our outage.
var defaultPackageManagerVersion = map[string]string{
	"pnpm": "pnpm@10.18.3",
	"yarn": "yarn@4.10.3",
}

// packageManagerOf reads the tool out of an install command.
//
// Taken from the command rather than passed alongside it: the command is what
// actually runs, so anything derived from something else can disagree with it.
func packageManagerOf(installCmd string) string {
	fields := strings.Fields(strings.TrimSpace(installCmd))
	if len(fields) == 0 {
		return ""
	}
	switch fields[0] {
	case "pnpm", "yarn", "bun", "npm":
		return fields[0]
	}
	return ""
}

// startCommand is what the image runs when it is a server rather than a
// directory of files.
//
// Previously hardcoded to `node server.js`, which is not how any framework
// with a start script is launched — a Next.js app died at boot regardless of
// how well it built.
func startCommand(cfg DeployConfig) string {
	cmd := strings.TrimSpace(cfg.StartCommand)
	if cmd == "" {
		cmd = "node server.js"
	}
	// Exec form where we can, so the process gets signals directly; a command
	// with shell syntax in it has to go through a shell.
	if strings.ContainsAny(cmd, "&|;><$") {
		return fmt.Sprintf("[\"/bin/sh\", \"-c\", %q]", cmd)
	}
	parts := strings.Fields(cmd)
	quoted := make([]string, 0, len(parts))
	for _, p := range parts {
		quoted = append(quoted, fmt.Sprintf("%q", p))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// accessLogConf keeps nginx's combined format and appends the request time,
// which the default format omits — the reason every request reported 0ms.
const accessLogConf = `log_format applad '$remote_addr - $remote_user [$time_local] ` +
	`"$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" rt=$request_time';
access_log /dev/stdout applad;
`

// nodeTag picks the Node base image tag, defaulting when the project does not
// pin one. A project that declares engines.node or ships an .nvmrc usually
// means it.
// nodeTag picks the Node image, defaulting to the current LTS.
//
// It defaulted to 20, which corepack then outgrew: with no version pinned in
// package.json it fetches the newest pnpm, and pnpm 11 uses builtins Node 20
// does not have — so an unpinned project failed with ERR_UNKNOWN_BUILTIN_MODULE
// before installing anything. A project that states its own version still gets
// exactly that.
func nodeTag(version string) string {
	if version == "" {
		return "22"
	}
	return version
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
		// macOS tar writes an AppleDouble sidecar (._name) next to every file.
		// They are not source, and a test runner that globs *.test.js will try
		// to execute them and report a failure nobody wrote.
		if strings.HasPrefix(info.Name(), "._") || info.Name() == ".DS_Store" {
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
func (d *DeployExecutor) DeployContainer(ctx context.Context, deploymentID, projectID string, cfg DeployConfig) (string, error) {
	image := cfg.Image
	if image == "" {
		// Fall back to the applad-deploy image if no external image specified
		image = fmt.Sprintf("applad-deploy-%s", deploymentID)
	} else {
		// Pull the image
		log.Printf("deploy: pulling image %s for deployment %s", image, deploymentID)
		if err := d.pullImage(ctx, image); err != nil {
			return "", fmt.Errorf("deploy: image pull failed: %w", err)
		}
	}

	return "", d.startContainer(ctx, deploymentID, projectID, image, cfg)
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

	// A site that never answers is not a deploy that worked.
	//
	// This was logged and ignored, so a container that started and served
	// nothing at the advertised address was reported as a success, and the
	// console showed it as active while visitors got a placeholder.
	if err := d.waitForHealthy(ctx, containerName, cfg.Port); err != nil {
		return fmt.Errorf("the build succeeded but the site never answered on port %s: %w\n"+
			"Check that it listens on the port given in $PORT and binds 0.0.0.0 rather than localhost", cfg.Port, err)
	}

	log.Printf("deploy: container %s started for deployment %s", containerID[:12], deploymentID)
	return nil
}

// deployNetworkName is the user-defined network deployed apps join. Deployed
// apps must sit on the same network as the proxy, so the ingress can reach them
// by container name via Docker's embedded DNS. The default bridge has no name
// resolution, which makes apps unroutable.
func deployNetworkName() string {
	if network := os.Getenv("APPLAD_DEPLOY_NETWORK"); network != "" {
		return network
	}
	return "applad_default"
}

// deployContainerBody builds the Docker create request for a deployed app.
//
// It is a standalone function so the security-relevant HostConfig can be
// asserted in a test without a Docker daemon. Deployed apps run arbitrary
// customer code, so they carry the same baseline hardening functions get:
// capabilities dropped, no privilege escalation, and a process cap so a
// fork bomb cannot exhaust the host's PIDs. The rootfs is left writable on
// purpose — a web app routinely writes temp and cache files, and functions
// only get a read-only rootfs because they also get a writable /tmp tmpfs.
func deployContainerBody(name, image, port string, env []string, labels map[string]string, network string) map[string]interface{} {
	exposedPort := port + "/tcp"
	return map[string]interface{}{
		"Image":  image,
		"Env":    env,
		"Labels": labels,
		"ExposedPorts": map[string]interface{}{
			exposedPort: struct{}{},
		},
		"HostConfig": map[string]interface{}{
			// No PublishAllPorts: the app is reached by container name on the
			// deploy network through the edge proxy, so binding its port on the
			// host is unnecessary and would expose it without the proxy in front.
			"Memory":      int64(512 * 1024 * 1024), // 512MB for deployments
			"NanoCPUs":    int64(2e9),               // 2 CPUs
			"NetworkMode": network,
			// Baseline hardening for untrusted customer code. These do not
			// interfere with a normal web server binding its port.
			"CapDrop":     []string{"ALL"},
			"SecurityOpt": []string{"no-new-privileges"},
			"PidsLimit":   int64(512), // higher than a function: apps fork workers
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
}

// createDeployContainer creates a container with explicit port mapping
// instead of PublishAllPorts, so the deployment gets a predictable port.
func (d *DeployExecutor) createDeployContainer(ctx context.Context, name, image, port string, env []string, labels map[string]string) (string, error) {
	network := deployNetworkName()

	body := deployContainerBody(name, image, port, env, labels, network)

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

	// Long enough for a slow framework to boot; short enough that a site
	// which will never answer is reported rather than waited on.
	deadline := time.After(60 * time.Second)
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
			// 5xx is the app answering that it is broken, which is still the
			// app answering — a deploy is not the place to judge it. Anything
			// that connects at all clears this bar.
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

// ImageSize returns the size of a built image in bytes, so a deploy can report
// what it produced.
func (d *DeployExecutor) ImageSize(ctx context.Context, image string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/v1.44/images/%s/json", d.docker.baseURL, url.QueryEscape(image)), nil)
	if err != nil {
		return 0, err
	}
	resp, err := d.docker.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("image %s not found", image)
	}
	var info struct {
		Size int64 `json:"Size"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return 0, err
	}
	return info.Size, nil
}

// ContainerLogsByName reads a running container's output, given the name the
// deploy executor gave it.
func (d *DeployExecutor) ContainerLogsByName(ctx context.Context, name string) (string, error) {
	id := d.findContainerByName(ctx, name)
	if id == "" {
		return "", fmt.Errorf("no container named %s", name)
	}
	return d.docker.ContainerLogs(ctx, id)
}
