package runtime

import (
	"strings"
	"testing"
)

// The build that broke: a pipeline naming `npm run build` produced a Dockerfile
// that copied the source and ran the build with nothing installed, because one
// command had to be both phases at once.
func TestBuildPhasesInstallBeforeSourceCopy(t *testing.T) {
	got := buildPhases(DeployConfig{
		InstallCommand: "npm ci",
		BuildCommand:   "npm run build",
	})

	install := strings.Index(got, "RUN npm ci")
	manifest := strings.Index(got, "COPY package.json")
	source := strings.Index(got, "COPY . .")
	build := strings.Index(got, "RUN npm run build")

	for name, idx := range map[string]int{
		"manifest copy": manifest, "install": install, "source copy": source, "build": build,
	} {
		if idx < 0 {
			t.Fatalf("%s missing from:\n%s", name, got)
		}
	}
	// The order is the whole point: dependencies install from the manifest
	// alone, so editing a source file does not reinstall them.
	if !(manifest < install && install < source && source < build) {
		t.Errorf("phases out of order:\n%s", got)
	}
}

func TestBuildPhasesWithoutAnInstall(t *testing.T) {
	// A pre-built upload has nothing to install and must not grow a layer
	// copying a package.json it does not have.
	got := buildPhases(DeployConfig{BuildCommand: "make"})
	if strings.Contains(got, "COPY package.json") {
		t.Errorf("manifest copied with no install step:\n%s", got)
	}
	if !strings.Contains(got, "COPY . .\nRUN make\n") {
		t.Errorf("unexpected phases:\n%s", got)
	}
}

func TestStartCommand(t *testing.T) {
	cases := map[string]string{
		"":                    `["node", "server.js"]`,
		"npm run start":       `["npm", "run", "start"]`,
		"next start -p 3000":  `["next", "start", "-p", "3000"]`,
		"node a.js && node b": `["/bin/sh", "-c", "node a.js && node b"]`,
	}
	for in, want := range cases {
		if got := startCommand(DeployConfig{StartCommand: in}); got != want {
			t.Errorf("startCommand(%q) = %s, want %s", in, got, want)
		}
	}
}

// The base image ships npm alone, so a pnpm project failed at the install it
// had just been given: "/bin/sh: pnpm: not found".
func TestBuildPhasesProvidesTheProjectsPackageManager(t *testing.T) {
	got := buildPhases(DeployConfig{
		InstallCommand: "pnpm install --frozen-lockfile",
		BuildCommand:   "pnpm build",
	})
	if !strings.Contains(got, "corepack enable") {
		t.Errorf("pnpm never installed:\n%s", got)
	}
	if strings.Index(got, "corepack enable") > strings.Index(got, "RUN pnpm install") {
		t.Errorf("package manager set up after it is used:\n%s", got)
	}

	// npm is already there; adding corepack would be a layer for nothing.
	npm := buildPhases(DeployConfig{InstallCommand: "npm ci", BuildCommand: "npm run build"})
	if strings.Contains(npm, "corepack") {
		t.Errorf("corepack added for npm:\n%s", npm)
	}
}
