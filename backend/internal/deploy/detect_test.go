package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func pkg(deps string, extra string) string {
	return `{"dependencies":{` + deps + `}` + extra + `}`
}

func TestDetectFrameworks(t *testing.T) {
	tests := []struct {
		name      string
		in        DetectInput
		framework string
		outputDir string
		serveMode string
	}{
		{
			name: "next.js",
			in: DetectInput{
				Files:     []string{"package.json", "next.config.js", "package-lock.json"},
				Manifests: map[string]string{"package.json": pkg(`"next":"14.0.0","react":"18.0.0"`, "")},
			},
			framework: "nextjs", outputDir: ".next", serveMode: "node",
		},
		{
			name: "astro beats its react dependency",
			in: DetectInput{
				Files:     []string{"package.json"},
				Manifests: map[string]string{"package.json": pkg(`"astro":"4.0.0","react":"18.0.0"`, "")},
			},
			framework: "astro", outputDir: "dist", serveMode: "static",
		},
		{
			name: "react on vite outputs dist",
			in: DetectInput{
				Files:     []string{"package.json"},
				Manifests: map[string]string{"package.json": pkg(`"react":"18.0.0","vite":"5.0.0"`, "")},
			},
			framework: "react", outputDir: "dist", serveMode: "static",
		},
		{
			name: "create-react-app outputs build",
			in: DetectInput{
				Files:     []string{"package.json"},
				Manifests: map[string]string{"package.json": pkg(`"react":"18.0.0","react-scripts":"5.0.0"`, "")},
			},
			framework: "react", outputDir: "build", serveMode: "static",
		},
		{
			name: "flutter web",
			in: DetectInput{
				Files:     []string{"pubspec.yaml", "web/index.html", "lib/main.dart"},
				Manifests: map[string]string{"pubspec.yaml": "name: app"},
			},
			framework: "flutter_web", outputDir: "build/web", serveMode: "static",
		},
		{
			name:      "plain static site",
			in:        DetectInput{Files: []string{"index.html", "about.html", "assets/css/style.css"}},
			framework: "static", outputDir: ".", serveMode: "static",
		},
		{
			name:      "repo dockerfile wins",
			in:        DetectInput{Files: []string{"Dockerfile", "package.json"}},
			framework: "docker",
		},
		{
			name:      "unknown tree falls back to static",
			in:        DetectInput{Files: []string{"README.md", "main.c"}},
			framework: "static", outputDir: ".", serveMode: "static",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Detect(tt.in)
			if got.Framework != tt.framework {
				t.Errorf("framework = %q, want %q (reason: %s)", got.Framework, tt.framework, got.Reason)
			}
			if tt.outputDir != "" && got.OutputDir != tt.outputDir {
				t.Errorf("outputDir = %q, want %q", got.OutputDir, tt.outputDir)
			}
			if tt.serveMode != "" && got.ServeMode != tt.serveMode {
				t.Errorf("serveMode = %q, want %q", got.ServeMode, tt.serveMode)
			}
		})
	}
}

func TestDetectPackageManager(t *testing.T) {
	tests := []struct {
		lockfile string
		pm       string
		install  string
	}{
		{"pnpm-lock.yaml", "pnpm", "pnpm install --frozen-lockfile"},
		{"yarn.lock", "yarn", "yarn install --frozen-lockfile"},
		{"bun.lockb", "bun", "bun install --frozen-lockfile"},
		{"package-lock.json", "npm", "npm ci"},
		{"", "npm", "npm install"},
	}

	for _, tt := range tests {
		t.Run(tt.lockfile, func(t *testing.T) {
			files := []string{"package.json"}
			if tt.lockfile != "" {
				files = append(files, tt.lockfile)
			}
			got := Detect(DetectInput{
				Files:     files,
				Manifests: map[string]string{"package.json": pkg(`"astro":"4.0.0"`, "")},
			})
			if got.PackageManager != tt.pm {
				t.Errorf("packageManager = %q, want %q", got.PackageManager, tt.pm)
			}
			if got.InstallCommand != tt.install {
				t.Errorf("installCommand = %q, want %q", got.InstallCommand, tt.install)
			}
		})
	}
}

// The build command must follow the detected manager: running npm against a
// pnpm lockfile is a common real-world failure.
func TestDetectBuildCommandFollowsPackageManager(t *testing.T) {
	got := Detect(DetectInput{
		Files:     []string{"package.json", "pnpm-lock.yaml"},
		Manifests: map[string]string{"package.json": pkg(`"astro":"4.0.0"`, `,"scripts":{"build":"astro build"}`)},
	})
	if got.BuildCommand != "pnpm run build" {
		t.Errorf("buildCommand = %q, want %q", got.BuildCommand, "pnpm run build")
	}
}

func TestNormalizeNodeVersion(t *testing.T) {
	tests := map[string]string{
		">=18.17.0": "18",
		"20.x":      "20",
		"^22.0.0":   "22",
		"v20":       "20",
		"latest":    "",
		"":          "",
	}
	for in, want := range tests {
		if got := normalizeNodeVersion(in); got != want {
			t.Errorf("normalizeNodeVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDetectDirReadsFromDisk(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", pkg(`"astro":"4.0.0"`, ""))
	writeFile(t, dir, "pnpm-lock.yaml", "lockfileVersion: 9")
	writeFile(t, dir, ".nvmrc", "20\n")
	// node_modules must not be walked, and a nested package.json must not win.
	writeFile(t, filepath.Join(dir, "node_modules", "left-pad"), "package.json", pkg(`"next":"14.0.0"`, ""))

	got := DetectDir(dir)
	if got.Framework != "astro" {
		t.Errorf("framework = %q, want astro (reason: %s)", got.Framework, got.Reason)
	}
	if got.PackageManager != "pnpm" {
		t.Errorf("packageManager = %q, want pnpm", got.PackageManager)
	}
	if got.NodeVersion != "20" {
		t.Errorf("nodeVersion = %q, want 20", got.NodeVersion)
	}
}

func writeFile(t *testing.T, dir, name, contents string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
