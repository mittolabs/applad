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
	"path"
	"strings"
	"time"
)

/*
 * Running a project's own test suite.
 *
 * The shape is deliberately the same as a web deploy: build an image from the
 * source, run it, read what came out. The difference is what we want back —
 * not a long-lived container but an exit code, the console output, and the
 * JUnit report the suite left behind, which is copied out of the stopped
 * container rather than mounted, so nothing needs to be shared with the host.
 */

// TestRunConfig describes one execution of a suite.
type TestRunConfig struct {
	SourceDir  string // checked-out or extracted project
	Image      string // base image, e.g. node:20-alpine
	SetupCmd   string // e.g. npm ci
	Command    string // e.g. npm test
	ReportPath string // where the JUnit XML lands, relative to the project
	// ArtifactsPath is a directory the suite writes evidence into — videos,
	// screenshots, traces. Copied out whole when set.
	ArtifactsPath string
	Env           map[string]string
	TimeoutMs     int
	// OnLine, when set, receives the suite's output as it is produced, so a
	// run can be watched rather than waited for.
	OnLine func(string)
}

// TestRunResult is what a finished run produced.
type TestRunResult struct {
	ExitCode int
	Log      string
	// Report is the JUnit XML, empty when the suite wrote none. A suite can
	// fail without producing one — a setup command that dies, say — which is
	// why the exit code and log are reported separately.
	Report []byte
	// Artifacts are the files found under ArtifactsPath, keyed by their path
	// relative to that directory.
	Artifacts map[string][]byte
}

// RunTests builds the suite's image, runs it to completion, and returns the
// output together with the report.
func (d *DeployExecutor) RunTests(ctx context.Context, runID string, cfg TestRunConfig) (*TestRunResult, error) {
	if cfg.SourceDir == "" {
		return nil, fmt.Errorf("testrun: no source to test")
	}
	if strings.TrimSpace(cfg.Command) == "" {
		return nil, fmt.Errorf("testrun: no test command configured")
	}

	image := cfg.Image
	if image == "" {
		image = "node:20-alpine"
	}
	if cfg.ReportPath == "" {
		cfg.ReportPath = "junit.xml"
	}

	// The suite's own commands are the image's build steps up to the test run,
	// so dependency installation is cached between runs by Docker's layer
	// cache while the test command itself always executes.
	dockerfile := fmt.Sprintf("FROM %s\nWORKDIR /app\nCOPY . .\n", image)
	if strings.TrimSpace(cfg.SetupCmd) != "" {
		dockerfile += fmt.Sprintf("RUN %s\n", cfg.SetupCmd)
	}
	// A failing suite must still produce an image to read the report from, so
	// the exit code is captured to a file rather than failing the build.
	dockerfile += fmt.Sprintf("CMD sh -c '%s; echo $? > /app/.applad-exit'\n", escapeSingleQuotes(cfg.Command))

	tarBuf := new(bytes.Buffer)
	tw := tar.NewWriter(tarBuf)
	if err := addDirToTar(tw, cfg.SourceDir, true); err != nil {
		tw.Close()
		return nil, fmt.Errorf("testrun: build context: %w", err)
	}
	addToTar(tw, "Dockerfile", []byte(dockerfile))
	tw.Close()

	imageName := fmt.Sprintf("applad-test-%s", runID)
	log.Printf("testrun: building image %s", imageName)
	if err := d.docker.BuildImage(ctx, imageName, tarBuf); err != nil {
		return nil, fmt.Errorf("testrun: image build failed: %w", err)
	}
	defer d.docker.RemoveImage(context.Background(), imageName) //nolint:errcheck

	var env []string
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	containerName := fmt.Sprintf("applad-test-%s", runID)
	containerID, err := d.createTestContainer(ctx, containerName, imageName, env)
	if err != nil {
		return nil, fmt.Errorf("testrun: create container: %w", err)
	}
	defer d.docker.RemoveContainer(context.Background(), containerID) //nolint:errcheck

	if err := d.docker.StartContainer(ctx, containerID); err != nil {
		return nil, fmt.Errorf("testrun: start container: %w", err)
	}

	if cfg.OnLine != nil {
		// Follows in the background; it ends by itself when the container does.
		go d.docker.FollowContainerLogs(ctx, containerID, cfg.OnLine) //nolint:errcheck
	}

	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	exitCode, waitErr := d.docker.WaitContainer(waitCtx, containerID)
	if waitErr != nil {
		// A suite that never finishes is a failure with a reason, not a
		// mystery: stop it and report the timeout with whatever it logged.
		d.docker.StopContainer(context.Background(), containerID) //nolint:errcheck
		out, _ := d.docker.ContainerLogs(context.Background(), containerID)
		return &TestRunResult{ExitCode: -1, Log: out}, fmt.Errorf("testrun: timed out after %s", timeout)
	}

	out, _ := d.docker.ContainerLogs(ctx, containerID)
	report, _ := d.copyFromContainer(ctx, containerID, path.Join("/app", cfg.ReportPath))

	var artifacts map[string][]byte
	if strings.TrimSpace(cfg.ArtifactsPath) != "" {
		// Best effort: a suite that recorded nothing is not a failed run.
		artifacts, _ = d.copyDirFromContainer(ctx, containerID, path.Join("/app", cfg.ArtifactsPath))
	}

	return &TestRunResult{ExitCode: exitCode, Log: out, Report: report, Artifacts: artifacts}, nil
}

// createTestContainer makes a one-shot container for a suite.
//
// It deliberately does not reuse CreateContainer: that one is tuned for the
// function sandbox, where a read-only root filesystem and 256MB are correct.
// A test suite has to write its report, compilers need room, and runners fork
// freely — so this keeps the hardening that still applies (no new privileges,
// dropped capabilities, no swap) and relaxes what would simply break it.
func (d *DeployExecutor) createTestContainer(ctx context.Context, name, image string, env []string) (string, error) {
	body := map[string]interface{}{
		"Image": image,
		"Env":   env,
		"Labels": map[string]string{
			"applad.managed": "true",
			"applad.type":    "test",
		},
		// Joined to the deploy network so a browser test can reach the app it
		// is testing by name, the same way the ingress does.
		"NetworkingConfig": map[string]interface{}{
			"EndpointsConfig": map[string]interface{}{
				deployNetwork(): map[string]interface{}{},
			},
		},
		"HostConfig": map[string]interface{}{
			"Memory":      int64(2 * 1024 * 1024 * 1024),
			"MemorySwap":  int64(2 * 1024 * 1024 * 1024),
			"NanoCPUs":    int64(2e9),
			"PidsLimit":   int64(2048),
			"NetworkMode": deployNetwork(),
			"SecurityOpt": []string{"no-new-privileges"},
			// Browsers need their sandbox syscalls; dropping every capability
			// makes Chromium refuse to start.
			"CapAdd": []string{"SYS_ADMIN"},
			// One shot: a suite that exits is finished, however it exited.
			"RestartPolicy": map[string]interface{}{"Name": "no"},
		},
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf(d.docker.baseURL+"/v1.44/containers/create?name=%s", name), bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.docker.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("testrun: create container: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("testrun: create container failed (%d): %s", resp.StatusCode, string(respBody))
	}
	var result struct {
		ID string `json:"Id"`
	}
	json.Unmarshal(respBody, &result) //nolint:errcheck
	return result.ID, nil
}

// copyDirFromContainer copies a directory out of a stopped container. Docker
// streams it as a tar, so this unpacks it into a map keyed by the path
// relative to the directory itself.
func (d *DeployExecutor) copyDirFromContainer(ctx context.Context, containerID, dirPath string) (map[string][]byte, error) {
	endpoint := fmt.Sprintf("%s/v1.44/containers/%s/archive?path=%s",
		d.docker.baseURL, containerID, url.QueryEscape(dirPath))
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := d.docker.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("testrun: nothing at %s", dirPath)
	}

	base := path.Base(dirPath)
	out := map[string][]byte{}
	tr := tar.NewReader(resp.Body)
	var total int64
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return out, err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		// Entries arrive prefixed with the directory's own name.
		rel := strings.TrimPrefix(strings.TrimPrefix(hdr.Name, base), "/")
		if rel == "" {
			continue
		}
		// A whole run's evidence is capped so a runaway recording cannot fill
		// the volume.
		if total > 512<<20 {
			break
		}
		data, err := io.ReadAll(io.LimitReader(tr, 128<<20))
		if err != nil {
			return out, err
		}
		total += int64(len(data))
		out[rel] = data
	}
	return out, nil
}

// deployNetwork is the Docker network deployed apps and tests share, so a test
// can address the app it exercises.
func deployNetwork() string {
	if n := os.Getenv("APPLAD_DEPLOY_NETWORK"); n != "" {
		return n
	}
	return "applad_default"
}

// copyFromContainer reads a single file out of a stopped container. Docker
// returns a tar stream, so the first regular entry is the file.
func (d *DeployExecutor) copyFromContainer(ctx context.Context, containerID, filePath string) ([]byte, error) {
	endpoint := fmt.Sprintf("%s/v1.44/containers/%s/archive?path=%s",
		d.docker.baseURL, containerID, url.QueryEscape(filePath))
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := d.docker.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("testrun: no report at %s", filePath)
	}

	tr := tar.NewReader(resp.Body)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("testrun: report %s was empty", filePath)
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag == tar.TypeReg {
			return io.ReadAll(io.LimitReader(tr, 32<<20))
		}
	}
}

// escapeSingleQuotes makes a command safe to embed in a single-quoted sh -c
// string, so a test command containing quotes does not break the Dockerfile.
func escapeSingleQuotes(s string) string {
	return strings.ReplaceAll(s, "'", `'\''`)
}
