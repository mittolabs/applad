package runtime

import (
	"strings"
	"testing"
)

// For a generated image the Dockerfile is Applad's, so narrating it tells the
// user about plumbing they never wrote. Their own command output is the part
// that belongs to them.
func TestDeployLogDropsImageBookkeeping(t *testing.T) {
	raw := `Step 1/5 : FROM nginx:alpine
 ---> 54f2a904c251
Step 2/5 : COPY applad-log.conf /etc/nginx/conf.d/applad-log.conf
 ---> Using cache
 ---> f483711f509d
Step 4/5 : RUN rm -f /usr/share/nginx/html/Dockerfile
 ---> Running in 921ce0116d5e
Removed intermediate container 921ce0116d5e
Step 5/5 : EXPOSE 80
Successfully built a3360c65b003
Successfully tagged applad-deploy-abc:latest`

	got := RenderDeployLog(raw)
	for _, noise := range []string{"Step 1/5", "applad-log.conf", "--->", "Successfully tagged", "nginx:alpine"} {
		if strings.Contains(got, noise) {
			t.Errorf("deploy log still mentions %q:\n%s", noise, got)
		}
	}
	if strings.TrimSpace(got) != "" {
		t.Errorf("a static site runs no commands of its own, so its log is empty:\n%s", got)
	}
}

// A project that builds has output worth reading, announced by the command
// that produced it.
func TestDeployLogKeepsTheProjectsOwnOutput(t *testing.T) {
	raw := `Step 1/6 : FROM node:20-alpine
 ---> abc123
Step 4/6 : RUN npm ci && npm run build
 ---> Running in def456
added 214 packages in 8s
vite v5.0.0 building for production...
✓ 41 modules transformed.
dist/index.html  0.46 kB
 ---> 789abc
Successfully built 789abc`

	got := RenderDeployLog(raw)
	if !strings.Contains(got, "$ npm ci && npm run build") {
		t.Errorf("the command should head its own output:\n%s", got)
	}
	for _, want := range []string{"added 214 packages", "41 modules transformed", "dist/index.html"} {
		if !strings.Contains(got, want) {
			t.Errorf("lost the project's output %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "--->") || strings.Contains(got, "Step 1/6") {
		t.Errorf("kept bookkeeping alongside it:\n%s", got)
	}
}

// The log from a real failed deploy: an uploaded source with no package.json
// at its root, which the console showed as one unbroken paragraph.
const npmFailureLog = `Step 1/7 : FROM node:20-alpine AS build
 ---> fb4cd12c85ee
Step 2/7 : WORKDIR /app
 ---> Running in 9fc57de6ec3a
 ---> Removed intermediate container 9fc57de6ec3a
 ---> 2d30dc1e2e95
Step 3/7 : COPY . .
 ---> eae1ba9fbdc7
Step 4/7 : RUN npm run build
 ---> Running in 4474d5ceef34
npm error code ENOENT
npm error syscall open
npm error path /app/package.json
npm error errno -2
npm error enoent Could not read package.json: Error: ENOENT: no such file or directory, open '/app/package.json'
npm error enoent This is related to npm not being able to find a file.
npm error enoent
npm error A complete log of this run can be found in: /root/.npm/_logs/2026-07-21T14_15_09_438Z-debug-0.log
 ---> Removed intermediate container 4474d5ceef34
The command '/bin/sh -c npm run build' returned a non-zero code: 254`

func TestSummariseBuildFailureNamesTheCommandAndItsLastWords(t *testing.T) {
	got := SummariseBuildFailure(npmFailureLog)

	head, rest, _ := strings.Cut(got, "\n")
	if head != "npm run build exited with code 254" {
		t.Errorf("headline = %q", head)
	}
	if !strings.Contains(rest, "Could not read package.json") {
		t.Errorf("summary dropped the reason:\n%s", got)
	}
	// The bookkeeping is what made it unreadable.
	for _, unwanted := range []string{"Step 1/7", "--->", "fb4cd12c85ee", "Removed intermediate container"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("summary still carries %q:\n%s", unwanted, got)
		}
	}
	// So is the tool telling you where to read more.
	if strings.Contains(got, "_logs/") {
		t.Errorf("summary kept the debug-log pointer:\n%s", got)
	}
	if lines := strings.Count(got, "\n") + 1; lines > 7 {
		t.Errorf("summary is %d lines, still a wall:\n%s", lines, got)
	}
}

func TestSummariseBuildFailureWithoutAFailedCommand(t *testing.T) {
	// An image that could not be assembled at all names no command.
	raw := "Step 1/3 : FROM node:20-alpine AS build\npull access denied for node:20-alpin, repository does not exist"
	got := SummariseBuildFailure(raw)
	if !strings.HasPrefix(got, "Build failed") {
		t.Errorf("expected a headline, got %q", got)
	}
	if !strings.Contains(got, "pull access denied") {
		t.Errorf("expected the daemon's reason, got %q", got)
	}
}

// BuildKit interleaves steps by number, so a project's own output arrives
// shuffled with layer bookkeeping. Feeding it to the Dockerfile renderer
// returned nothing, and the deploy log then claimed there had been no build.
const railpackLog = `#1 loading .
#1 DONE 0.0s
#3 docker-image://ghcr.io/railwayapp/railpack-runtime:mise-2026.7.7
#3 resolve ghcr.io/railwayapp/railpack-runtime:mise-2026.7.7 0.9s done
#3 DONE 0.9s
#12 pnpm install --frozen-lockfile
#12 0.421 Lockfile is up to date, resolution step is skipped
#12 2.310 Packages: +23
#12 DONE 3.1s
#13 npm run build
#13 0.131 > echo BUILDING-NOW
#13 0.134 BUILDING-NOW
#13 DONE 0.4s
#14 exporting to docker image format
#14 sending tarball
#14 DONE 8.0s`

func TestRenderRailpackLogKeepsTheProjectsOutput(t *testing.T) {
	got := RenderRailpackLog(railpackLog)

	for _, want := range []string{
		"$ pnpm install --frozen-lockfile",
		"Lockfile is up to date",
		"$ npm run build",
		"BUILDING-NOW",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q from:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"DONE", "sending tarball", "docker-image://", "resolve ghcr.io"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("kept bookkeeping %q:\n%s", unwanted, got)
		}
	}
}

func TestRenderBuildLogPicksTheRightRenderer(t *testing.T) {
	if RenderBuildLog(railpackLog) == "" {
		t.Error("a railpack build reported no output at all")
	}
	if !strings.Contains(RenderBuildLog(npmFailureLog), "npm error code ENOENT") {
		t.Error("a Docker build lost its output")
	}
}
