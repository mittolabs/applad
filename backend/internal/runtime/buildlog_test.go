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
