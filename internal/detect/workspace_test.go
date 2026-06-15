package detect

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestParseWorkspaceHintsPackageJSONArray(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"name": "root",
		"workspaces": ["packages/*", "apps/*"]
	}`)
	mustMkdir(t, filepath.Join(dir, "packages", "client"))
	mustMkdir(t, filepath.Join(dir, "packages", "server"))
	mustMkdir(t, filepath.Join(dir, "apps", "web"))

	hints, err := parseWorkspaceHints(dir)
	if err != nil {
		t.Fatalf("parseWorkspaceHints failed: %v", err)
	}

	want := []string{
		filepath.Join(dir, "apps", "web"),
		filepath.Join(dir, "packages", "client"),
		filepath.Join(dir, "packages", "server"),
	}
	if !slices.Equal(hints, want) {
		t.Errorf("hints = %v, want %v", hints, want)
	}
}

func TestParseWorkspaceHintsPackageJSONObject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"name": "root",
		"workspaces": {"packages": ["packages/*"]}
	}`)
	mustMkdir(t, filepath.Join(dir, "packages", "a"))
	mustMkdir(t, filepath.Join(dir, "packages", "b"))

	hints, err := parseWorkspaceHints(dir)
	if err != nil {
		t.Fatalf("parseWorkspaceHints failed: %v", err)
	}

	want := []string{
		filepath.Join(dir, "packages", "a"),
		filepath.Join(dir, "packages", "b"),
	}
	if !slices.Equal(hints, want) {
		t.Errorf("hints = %v, want %v", hints, want)
	}
}

func TestParseWorkspaceHintsPnpmYAML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pnpm-workspace.yaml", "packages:\n  - 'packages/*'\n")
	mustMkdir(t, filepath.Join(dir, "packages", "a"))
	mustMkdir(t, filepath.Join(dir, "packages", "b"))

	hints, err := parseWorkspaceHints(dir)
	if err != nil {
		t.Fatalf("parseWorkspaceHints failed: %v", err)
	}

	want := []string{
		filepath.Join(dir, "packages", "a"),
		filepath.Join(dir, "packages", "b"),
	}
	if !slices.Equal(hints, want) {
		t.Errorf("hints = %v, want %v", hints, want)
	}
}

func TestParseWorkspaceHintsMalformedPnpmYAML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pnpm-workspace.yaml", "packages: [\n")
	mustMkdir(t, filepath.Join(dir, "packages", "a"))

	hints, err := parseWorkspaceHints(dir)
	if err != nil {
		t.Fatalf("parseWorkspaceHints returned error for malformed YAML: %v", err)
	}
	if len(hints) != 0 {
		t.Errorf("expected empty hints for malformed YAML, got %v", hints)
	}
}

func TestParseWorkspaceHintsPrefersNpmWhenPackageLockPresent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"workspaces": ["npm-pkgs/*"]
	}`)
	writeFile(t, dir, "pnpm-workspace.yaml", "packages:\n  - 'pnpm-pkgs/*'\n")
	writeFile(t, dir, "package-lock.json", "{}")
	mustMkdir(t, filepath.Join(dir, "npm-pkgs", "a"))
	mustMkdir(t, filepath.Join(dir, "pnpm-pkgs", "b"))

	hints, err := parseWorkspaceHints(dir)
	if err != nil {
		t.Fatalf("parseWorkspaceHints failed: %v", err)
	}

	want := []string{filepath.Join(dir, "npm-pkgs", "a")}
	if !slices.Equal(hints, want) {
		t.Errorf("hints = %v, want %v", hints, want)
	}
}

func TestParseWorkspaceHintsPrefersPnpmWhenPnpmLockPresent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"workspaces": ["npm-pkgs/*"]
	}`)
	writeFile(t, dir, "pnpm-workspace.yaml", "packages:\n  - 'pnpm-pkgs/*'\n")
	writeFile(t, dir, "pnpm-lock.yaml", "lockfileVersion: '6.0'\n")
	mustMkdir(t, filepath.Join(dir, "npm-pkgs", "a"))
	mustMkdir(t, filepath.Join(dir, "pnpm-pkgs", "b"))

	hints, err := parseWorkspaceHints(dir)
	if err != nil {
		t.Fatalf("parseWorkspaceHints failed: %v", err)
	}

	want := []string{filepath.Join(dir, "pnpm-pkgs", "b")}
	if !slices.Equal(hints, want) {
		t.Errorf("hints = %v, want %v", hints, want)
	}
}

func TestParseWorkspaceHintsNoConfig(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "packages", "a"))

	hints, err := parseWorkspaceHints(dir)
	if err != nil {
		t.Fatalf("parseWorkspaceHints failed: %v", err)
	}
	if len(hints) != 0 {
		t.Errorf("expected empty hints with no workspace config, got %v", hints)
	}
}

func TestParseWorkspaceHintsGlobExpansionIgnoresFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"workspaces": ["packages/*"]
	}`)
	mustMkdir(t, filepath.Join(dir, "packages", "a"))
	writeFile(t, filepath.Join(dir, "packages"), "README.md", "# packages\n")

	hints, err := parseWorkspaceHints(dir)
	if err != nil {
		t.Fatalf("parseWorkspaceHints failed: %v", err)
	}

	want := []string{filepath.Join(dir, "packages", "a")}
	if !slices.Equal(hints, want) {
		t.Errorf("hints = %v, want %v", hints, want)
	}
}

func TestDetectScansWorkspaceDirsFirst(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"workspaces": ["packages/*"]
	}`)

	// Workspace dir is named so it sorts AFTER the non-workspace dir.
	// Without workspace prioritization, the non-workspace eslint ^9 would win.
	mustMkdir(t, filepath.Join(dir, "packages", "z"))
	writeFile(t, filepath.Join(dir, "packages", "z"), "package.json", `{
		"devDependencies": { "eslint": "^8.0.0" }
	}`)
	mustMkdir(t, filepath.Join(dir, "packages", "z", "node_modules", ".bin"))
	writeExecutable(t, filepath.Join(dir, "packages", "z", "node_modules", ".bin"), "eslint")

	mustMkdir(t, filepath.Join(dir, "apps", "a"))
	writeFile(t, filepath.Join(dir, "apps", "a"), "package.json", `{
		"devDependencies": { "eslint": "^9.0.0" }
	}`)
	mustMkdir(t, filepath.Join(dir, "apps", "a", "node_modules", ".bin"))
	writeExecutable(t, filepath.Join(dir, "apps", "a", "node_modules", ".bin"), "eslint")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if d.Config.Tools["eslint"] != "8.x" {
		t.Errorf("Tools[eslint] = %q, want 8.x (workspace dir scanned first)", d.Config.Tools["eslint"])
	}

	subdirs := d.ToolSubdirs["eslint"]
	if len(subdirs) != 2 {
		t.Fatalf("expected eslint in 2 subdirs, got %v", subdirs)
	}
	if subdirs[0] != filepath.Join("packages", "z") {
		t.Errorf("expected workspace dir first, got %v", subdirs)
	}
}

func TestDetectStaleWorkspaceConfigStillDiscoversOutsidePaths(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pnpm-workspace.yaml", "packages:\n  - 'packages/*'\n")

	mustMkdir(t, filepath.Join(dir, "packages", "a"))
	writeFile(t, filepath.Join(dir, "packages", "a"), "package.json", `{
		"devDependencies": { "eslint": "^8.0.0" }
	}`)
	mustMkdir(t, filepath.Join(dir, "packages", "a", "node_modules", ".bin"))
	writeExecutable(t, filepath.Join(dir, "packages", "a", "node_modules", ".bin"), "eslint")

	mustMkdir(t, filepath.Join(dir, "services", "api"))
	writeFile(t, filepath.Join(dir, "services", "api"), "go.mod", "module api\n\ngo 1.22\n")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if d.Config.Tools["eslint"] != "8.x" {
		t.Errorf("Tools[eslint] = %q, want 8.x", d.Config.Tools["eslint"])
	}
	if d.Config.Tools["go"] != "1.22" {
		t.Errorf("Tools[go] = %q, want 1.22", d.Config.Tools["go"])
	}
}

func TestDetectWalkDirOnlyWhenNoWorkspaceConfig(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "backend"))
	writeFile(t, filepath.Join(dir, "backend"), "go.mod", "module backend\n\ngo 1.21\n")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if d.Config.Tools["go"] != "1.21" {
		t.Errorf("Tools[go] = %q, want 1.21", d.Config.Tools["go"])
	}
}
