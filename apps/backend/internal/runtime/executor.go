package runtime

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExecRequest is the input to execute a function.
type ExecRequest struct {
	FunctionID string
	ProjectID  string
	Runtime    string
	Entrypoint string
	Source     string // inline source code
	SourceDir  string // path to a cloned git repo directory (takes priority over Source)
	Dockerfile string // custom Dockerfile (takes priority over Source+Runtime)
	EnvVars    map[string]string
	Timeout    int    // seconds
	Payload    string // JSON payload to send to the function
}

// ExecResult is the output of a function execution.
type ExecResult struct {
	Output   string
	Errors   string
	Duration float64 // seconds
	ExitCode int
}

// Executor builds and runs function containers.
type Executor struct {
	docker *Client
	pool   *Pool
}

// NewExecutor creates a function executor.
func NewExecutor() *Executor {
	docker := NewClient()
	return &Executor{
		docker: docker,
		pool:   NewPool(docker),
	}
}

// Build builds a container image for a function. Call this when the function
// source changes. The image is tagged as applad-fn-{functionID}.
func (e *Executor) Build(ctx context.Context, req ExecRequest) (string, error) {
	imageName := fmt.Sprintf("applad-fn-%s", req.FunctionID)

	// Generate Dockerfile
	var dockerfile string
	if req.Dockerfile != "" {
		dockerfile = req.Dockerfile
	} else {
		dockerfile = GenerateDockerfile(req.Runtime, req.Entrypoint, req.Source)
	}
	if dockerfile == "" {
		return "", fmt.Errorf("runtime: unsupported runtime %q and no Dockerfile provided", req.Runtime)
	}

	// Create tar archive with Dockerfile + source
	var tarBuf *bytes.Buffer
	if req.SourceDir != "" {
		var err error
		tarBuf, err = buildTarContextFromDir(dockerfile, req.SourceDir)
		if err != nil {
			return "", fmt.Errorf("runtime: tar from directory: %w", err)
		}
	} else {
		tarBuf = buildTarContext(dockerfile, req.Source, req.Runtime, req.Entrypoint)
	}

	if _, err := e.docker.BuildImage(ctx, imageName, tarBuf); err != nil {
		return "", err
	}

	// Destroy any existing warm containers for this function (image changed)
	e.pool.DestroyFunction(ctx, req.FunctionID)

	return imageName, nil
}

// Execute runs a function and returns the result. Uses warm containers
// from the pool when available.
func (e *Executor) Execute(ctx context.Context, req ExecRequest) (*ExecResult, error) {
	imageName := fmt.Sprintf("applad-fn-%s", req.FunctionID)

	env := funcEnv(req)

	// Get or create a container
	container, err := e.pool.GetOrCreate(ctx, req.FunctionID, imageName, env)
	if err != nil {
		return nil, fmt.Errorf("runtime: get container: %w", err)
	}

	// Set timeout
	timeout := time.Duration(req.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	// Invoke the function via HTTP
	addr := "http://localhost:" + container.Port
	if len(container.Port) > 5 {
		addr = "http://" + container.Port
	}

	payload := req.Payload
	if payload == "" {
		payload = "{}"
	}

	httpReq, err := http.NewRequestWithContext(execCtx, "POST", addr+"/", strings.NewReader(payload))
	if err != nil {
		e.pool.Release(container.ID)
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Applad-Function-ID", req.FunctionID)

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(httpReq)
	duration := time.Since(start).Seconds()

	if err != nil {
		// Timeout or connection error — destroy the container (it's in a bad state)
		e.pool.Destroy(execCtx, container.ID)
		return &ExecResult{
			Output:   "",
			Errors:   fmt.Sprintf("execution failed: %v", err),
			Duration: duration,
			ExitCode: 1,
		}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max

	// Release container back to pool for reuse
	e.pool.Release(container.ID)

	result := &ExecResult{
		Output:   string(body),
		Duration: duration,
		ExitCode: 0,
	}

	if resp.StatusCode >= 400 {
		result.Errors = fmt.Sprintf("function returned status %d", resp.StatusCode)
		result.ExitCode = 1
	}

	return result, nil
}

// funcEnv builds the container environment for a function: the function's own
// configured variables as KEY=VALUE, followed by the Applad-provided identifiers.
// These become the container's Docker Config.Env, so a runtime reads them the
// normal way (process.env, os.environ, os.Getenv, …).
func funcEnv(req ExecRequest) []string {
	env := make([]string, 0, len(req.EnvVars)+2)
	for k, v := range req.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	env = append(env, fmt.Sprintf("APPLAD_FUNCTION_ID=%s", req.FunctionID))
	env = append(env, fmt.Sprintf("APPLAD_PROJECT_ID=%s", req.ProjectID))
	return env
}

// Cleanup removes the image and all containers for a function.
func (e *Executor) Cleanup(ctx context.Context, functionID string) {
	e.pool.DestroyFunction(ctx, functionID)
	e.docker.RemoveImage(ctx, fmt.Sprintf("applad-fn-%s", functionID))
}

// buildTarContext creates a tar archive containing the Dockerfile and source files.
func buildTarContext(dockerfile, source, runtime, entrypoint string) *bytes.Buffer {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	// Add Dockerfile
	addToTar(tw, "Dockerfile", []byte(dockerfile))

	// Add source file with appropriate name
	if source != "" {
		filename := sourceFilename(runtime, entrypoint)
		addToTar(tw, filename, []byte(source))
	}

	tw.Close()
	return buf
}

// buildTarContextFromDir creates a tar archive from a directory (git clone output).
// The Dockerfile is prepended; all files in the directory are included at their
// relative paths so the Dockerfile's COPY . . picks them up.
func buildTarContextFromDir(dockerfile, srcDir string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	addToTar(tw, "Dockerfile", []byte(dockerfile))

	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip .git directory
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(srcDir, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, _ := d.Info()
		tw.WriteHeader(&tar.Header{
			Name: rel,
			Size: int64(len(data)),
			Mode: int64(info.Mode()),
		})
		tw.Write(data)
		return nil
	})
	if err != nil {
		return nil, err
	}
	tw.Close()
	return buf, nil
}

func addToTar(tw *tar.Writer, name string, data []byte) {
	tw.WriteHeader(&tar.Header{
		Name: name,
		Size: int64(len(data)),
		Mode: 0644,
	})
	tw.Write(data)
}

func sourceFilename(runtime, entrypoint string) string {
	switch {
	case strings.HasPrefix(runtime, "node"):
		return "index.js"
	case strings.HasPrefix(runtime, "python"):
		return "main.py"
	case strings.HasPrefix(runtime, "go"):
		return "main.go"
	case strings.HasPrefix(runtime, "dart"):
		return "main.dart"
	case strings.HasPrefix(runtime, "bun"):
		return "index.ts"
	default:
		return "handler"
	}
}
