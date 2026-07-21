package runtime

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

/*
 * Building an image from source without writing a Dockerfile.
 *
 * Applad used to generate one. That meant owning, per language, which base
 * image, which package manager, at which version, in what order, with which
 * runtime libraries — and getting any of it wrong produced a build that
 * failed late or, worse, an image that started and served nothing. A Next.js
 * app went out serving a placeholder because the generated Dockerfile copied
 * a build cache into nginx.
 *
 * Railpack does that job: it reads the source, produces a plan, and builds it
 * with BuildKit. Cache mounts for the package store and framework caches
 * survive between builds, which a generated Dockerfile could not express.
 *
 * What stays ours is the plan's overrides — the pipeline's install, build and
 * start commands — and where the build runs. BuildKit is addressed rather
 * than assumed local, so the same call builds on this machine or on a server
 * somebody added.
 */

// buildTimeout bounds a single image build. Long enough for a cold cache to
// fetch a language toolchain and every dependency; short enough that a build
// which will never finish is reported as one.
const buildTimeout = 25 * time.Minute

// RailpackBuild describes one image build from a source tree.
type RailpackBuild struct {
	SourceDir string
	ImageName string

	// Overrides. Empty means "let the plan decide", which is the point: a
	// command nobody wrote down is inferred rather than replaced by a guess.
	InstallCmd string
	BuildCmd   string
	StartCmd   string

	// CacheKey scopes BuildKit's caches. Per target, so two projects that
	// happen to share a lockfile do not share a node_modules cache, and so a
	// rebuild of the same site hits everything it populated last time.
	CacheKey string

	// Env is passed to the build, for build-time configuration a framework
	// reads (NEXT_PUBLIC_*, VITE_*, and the like).
	Env map[string]string
}

// BuildWithRailpack builds an image and returns the build output, whether or
// not it succeeded — a failed build's log is the only thing that explains it.
func (d *DeployExecutor) BuildWithRailpack(ctx context.Context, b RailpackBuild) (string, error) {
	if _, err := exec.LookPath("railpack"); err != nil {
		return "", fmt.Errorf("railpack is not installed in this worker: %w", err)
	}

	args := []string{"build", b.SourceDir, "--name", b.ImageName, "--progress", "plain"}
	if b.BuildCmd != "" {
		args = append(args, "--build-cmd", b.BuildCmd)
	}
	if b.StartCmd != "" {
		args = append(args, "--start-cmd", b.StartCmd)
	}
	if b.CacheKey != "" {
		args = append(args, "--cache-key", b.CacheKey)
	}
	// A web app that cannot be started is not a deployable image. Failing at
	// build time names the problem; failing later shows a running container
	// that answers nothing.
	args = append(args, "--error-missing-start")

	// A build that stops making progress must fail rather than hang. Without
	// this the worker sat on a finished-but-unloadable image until the queue
	// reaper killed the job, and the release said "deploying" for ever.
	ctx, cancel := context.WithTimeout(ctx, buildTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "railpack", args...)
	cmd.Env = append(os.Environ(), railpackEnv(b)...)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	log := out.String()
	if ctx.Err() == context.DeadlineExceeded {
		return log, fmt.Errorf("the build did not finish within %s and was stopped", buildTimeout)
	}
	if err != nil {
		return log, fmt.Errorf("%s", SummariseBuildFailure(log))
	}
	return log, nil
}

// railpackEnv carries the overrides railpack takes as environment rather than
// flags, plus the project's own build-time variables.
func railpackEnv(b RailpackBuild) []string {
	var env []string
	if b.InstallCmd != "" {
		env = append(env, "RAILPACK_INSTALL_CMD="+b.InstallCmd)
	}
	for k, v := range b.Env {
		// Railpack's own namespace is reserved; a project variable that
		// collided with it would silently rewrite the build plan.
		if strings.HasPrefix(strings.ToUpper(k), "RAILPACK_") {
			continue
		}
		env = append(env, k+"="+v)
	}
	return env
}

// RailpackPlan asks what railpack would do with a source tree, without
// building it.
//
// The console shows this before a deploy runs. Applad used to answer with a
// detector of its own, which meant two implementations disagreeing: the
// console prefilled a build command from one while the worker built from the
// other, and the install step fell down the gap between them.
func (d *DeployExecutor) RailpackPlan(ctx context.Context, sourceDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "railpack", "plan", sourceDir)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("railpack plan: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return out.String(), nil
}
