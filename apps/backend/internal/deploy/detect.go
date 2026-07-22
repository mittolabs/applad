package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

/*
 * Framework detection.
 *
 * Two callers share this: the console asks before uploading (so the create
 * wizard can prefill), and the build worker asks after cloning or extracting
 * (so a pipeline with no build config still deploys correctly). Keeping one
 * implementation server-side stops the two from drifting.
 */

// DetectInput is a lightweight view of a source tree: every relative path,
// plus the contents of the few files worth parsing. The console sends this
// instead of the tree itself, which keeps the request a few KB.
type DetectInput struct {
	Files     []string          `json:"files"`
	Manifests map[string]string `json:"manifests"`
}

// Detection is the build configuration inferred from a source tree.
type Detection struct {
	Framework      string `json:"framework"`
	InstallCommand string `json:"installCommand"`
	BuildCommand   string `json:"buildCommand"`
	OutputDir      string `json:"outputDir"`
	StartCommand   string `json:"startCommand"`
	// ServeMode is "static" (build, then serve OutputDir with nginx) or
	// "node" (run a long-lived server). Frameworks differ here even when
	// their build command is identical.
	ServeMode      string `json:"serveMode"`
	PackageManager string `json:"packageManager"`
	// PackageManagerPin is package.json's own "packageManager" field, when it
	// has one. Without it, a build that shells out to corepack gets the
	// newest release of that tool — which is how an unchanged project starts
	// failing the day pnpm changes a default.
	PackageManagerPin string `json:"packageManagerPin"`
	NodeVersion       string `json:"nodeVersion"`
	// Reason states the signal that decided it, so the console can show
	// "Detected Next.js from package.json" rather than an unexplained guess.
	Reason string `json:"reason"`
}

// ManifestFiles are the files DetectDir reads in full; everything else is
// matched by path alone. The console should send the same set.
var ManifestFiles = []string{"package.json", ".nvmrc", "pubspec.yaml"}

// nodeFramework maps a dependency to its build configuration. Order matters:
// the first match wins, so meta-frameworks are listed before the UI libraries
// they depend on (a Next.js app also depends on react).
var nodeFrameworks = []struct {
	dep       string
	framework string
	build     string
	outputDir string
	serveMode string
	start     string
}{
	{"next", "nextjs", "npm run build", ".next", "node", "npm run start"},
	{"nuxt", "nuxt", "npm run build", ".output/public", "node", "node .output/server/index.mjs"},
	{"@sveltejs/kit", "sveltekit", "npm run build", "build", "static", ""},
	{"astro", "astro", "npm run build", "dist", "static", ""},
	{"gatsby", "gatsby", "npm run build", "public", "static", ""},
	{"@angular/core", "angular", "npm run build", "dist", "static", ""},
	{"vue", "vue", "npm run build", "dist", "static", ""},
	{"react-scripts", "react", "npm run build", "build", "static", ""},
	{"react", "react", "npm run build", "dist", "static", ""},
}

// Detect infers build configuration from a source tree description.
func Detect(in DetectInput) Detection {
	files := make(map[string]bool, len(in.Files))
	for _, f := range in.Files {
		files[strings.TrimPrefix(filepath.ToSlash(f), "./")] = true
	}

	// A repo that ships its own Dockerfile has already answered the question.
	if files["Dockerfile"] {
		return Detection{
			Framework: "docker",
			ServeMode: "node",
			Reason:    "Dockerfile in the project root",
		}
	}

	if pkg, ok := in.Manifests["package.json"]; ok {
		if d, matched := detectNode(pkg, in, files); matched {
			return d
		}
	}

	if files["pubspec.yaml"] && (files["web/index.html"] || hasPrefix(files, "web/")) {
		return Detection{
			Framework:      "flutter_web",
			InstallCommand: "flutter pub get",
			BuildCommand:   "flutter build web --release",
			OutputDir:      "build/web",
			ServeMode:      "static",
			Reason:         "pubspec.yaml with a web/ directory",
		}
	}

	// No build system and an entry page: serve exactly what was given.
	if files["index.html"] {
		return Detection{
			Framework: "static",
			OutputDir: ".",
			ServeMode: "static",
			Reason:    "index.html with no build system",
		}
	}

	return Detection{
		Framework: "static",
		OutputDir: ".",
		ServeMode: "static",
		Reason:    "no build system detected, serving files as-is",
	}
}

func detectNode(pkgJSON string, in DetectInput, files map[string]bool) (Detection, bool) {
	var pkg struct {
		Scripts      map[string]string `json:"scripts"`
		Dependencies map[string]string `json:"dependencies"`
		DevDeps      map[string]string `json:"devDependencies"`
		Engines      struct {
			Node string `json:"node"`
		} `json:"engines"`
		// The version the project pinned, e.g. "pnpm@9.1.0". Corepack reads
		// this too; carrying it means the build can pin something sensible
		// when it is absent instead of fetching whatever is newest.
		PackageManager string `json:"packageManager"`
	}
	if err := json.Unmarshal([]byte(pkgJSON), &pkg); err != nil {
		return Detection{}, false
	}

	deps := map[string]bool{}
	for name := range pkg.Dependencies {
		deps[name] = true
	}
	for name := range pkg.DevDeps {
		deps[name] = true
	}

	pm, install := packageManager(files)
	nodeVersion := strings.TrimSpace(in.Manifests[".nvmrc"])
	if nodeVersion == "" {
		nodeVersion = normalizeNodeVersion(pkg.Engines.Node)
	} else {
		nodeVersion = normalizeNodeVersion(nodeVersion)
	}

	for _, fw := range nodeFrameworks {
		if !deps[fw.dep] {
			continue
		}
		d := Detection{
			Framework:         fw.framework,
			InstallCommand:    install,
			BuildCommand:      scriptOr(pkg.Scripts, "build", fw.build, pm),
			OutputDir:         fw.outputDir,
			ServeMode:         fw.serveMode,
			StartCommand:      fw.start,
			PackageManager:    pm,
			PackageManagerPin: pkg.PackageManager,
			NodeVersion:       nodeVersion,
			Reason:            fmt.Sprintf("%q in package.json dependencies", fw.dep),
		}
		// Vite writes to dist regardless of the UI library on top of it.
		if deps["vite"] && (fw.framework == "react" || fw.framework == "vue") {
			d.OutputDir = "dist"
			d.Reason += " with vite"
		}
		if d.ServeMode == "node" && d.StartCommand != "" {
			d.StartCommand = withPackageManager(d.StartCommand, pm)
		}
		return d, true
	}

	// A build script with no recognisable framework: still buildable. Serving
	// a guessed output directory would be worse than running the app.
	if _, ok := pkg.Scripts["build"]; ok {
		return Detection{
			Framework:         "node",
			InstallCommand:    install,
			BuildCommand:      withPackageManager("npm run build", pm),
			OutputDir:         firstExisting(files, "dist", "build", "public", "."),
			ServeMode:         "static",
			PackageManager:    pm,
			PackageManagerPin: pkg.PackageManager,
			NodeVersion:       nodeVersion,
			Reason:            "build script in package.json",
		}, true
	}

	if _, ok := pkg.Scripts["start"]; ok {
		return Detection{
			Framework:         "node",
			InstallCommand:    install,
			StartCommand:      withPackageManager("npm run start", pm),
			ServeMode:         "node",
			PackageManager:    pm,
			PackageManagerPin: pkg.PackageManager,
			NodeVersion:       nodeVersion,
			Reason:            "start script in package.json",
		}, true
	}

	return Detection{}, false
}

// packageManager picks the manager from the lockfile that is present. Running
// npm against a pnpm lockfile is one of the most common build failures.
func packageManager(files map[string]bool) (name, install string) {
	switch {
	case files["bun.lockb"] || files["bun.lock"]:
		return "bun", "bun install --frozen-lockfile"
	case files["pnpm-lock.yaml"]:
		return "pnpm", "pnpm install --frozen-lockfile"
	case files["yarn.lock"]:
		return "yarn", "yarn install --frozen-lockfile"
	case files["package-lock.json"]:
		return "npm", "npm ci"
	default:
		return "npm", "npm install"
	}
}

// withPackageManager rewrites an npm-flavoured command for the detected
// manager, so "npm run build" becomes "pnpm run build" and so on.
func withPackageManager(cmd, pm string) string {
	if pm == "" || pm == "npm" {
		return cmd
	}
	return strings.Replace(cmd, "npm ", pm+" ", 1)
}

// scriptOr prefers the project's own build script when it defines one.
func scriptOr(scripts map[string]string, name, fallback, pm string) string {
	if _, ok := scripts[name]; ok {
		return withPackageManager("npm run "+name, pm)
	}
	return withPackageManager(fallback, pm)
}

// normalizeNodeVersion reduces a semver range to the major version usable as
// a Docker tag: ">=18.17.0" and "20.x" both become "20"-style tags.
func normalizeNodeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	v = strings.TrimLeft(v, "^~>=v ")
	major := v
	if i := strings.IndexAny(v, ".x -"); i > 0 {
		major = v[:i]
	}
	for _, r := range major {
		if r < '0' || r > '9' {
			return ""
		}
	}
	if major == "" {
		return ""
	}
	return major
}

func firstExisting(files map[string]bool, candidates ...string) string {
	for _, c := range candidates {
		if files[c] || hasPrefix(files, c+"/") {
			return c
		}
	}
	return candidates[len(candidates)-1]
}

func hasPrefix(files map[string]bool, prefix string) bool {
	for f := range files {
		if strings.HasPrefix(f, prefix) {
			return true
		}
	}
	return false
}

// DetectDir runs detection against a directory on disk. Used by the build
// worker once a repository is cloned or an upload extracted.
func DetectDir(dir string) Detection {
	in := DetectInput{Manifests: map[string]string{}}

	skip := map[string]bool{".git": true, "node_modules": true, ".next": true}
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error { //nolint:errcheck
		if err != nil {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil || rel == "." {
			return nil
		}
		if info.IsDir() {
			if skip[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		in.Files = append(in.Files, filepath.ToSlash(rel))
		return nil
	})

	for _, name := range ManifestFiles {
		// Only root manifests decide the build; nested ones belong to
		// packages inside the project.
		if data, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			in.Manifests[name] = string(data)
		}
	}

	sort.Strings(in.Files)
	return Detect(in)
}
