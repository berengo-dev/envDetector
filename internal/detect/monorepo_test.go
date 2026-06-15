package detect

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestCollectSubdirsSkipsForbidden(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "node_modules"))
	mustMkdir(t, filepath.Join(dir, ".git"))
	mustMkdir(t, filepath.Join(dir, "vendor"))
	mustMkdir(t, filepath.Join(dir, ".venv"))
	mustMkdir(t, filepath.Join(dir, "__pycache__"))
	mustMkdir(t, filepath.Join(dir, ".turbo"))
	mustMkdir(t, filepath.Join(dir, "build"))
	mustMkdir(t, filepath.Join(dir, ".next"))
	mustMkdir(t, filepath.Join(dir, "out"))
	mustMkdir(t, filepath.Join(dir, "target"))
	mustMkdir(t, filepath.Join(dir, ".dist"))

	// Nested project directories should still be discovered.
	mustMkdir(t, filepath.Join(dir, "frontend"))
	mustMkdir(t, filepath.Join(dir, "backend"))

	subdirs, err := collectSubdirs(dir, defaultSkipList)
	if err != nil {
		t.Fatalf("collectSubdirs failed: %v", err)
	}

	want := []string{filepath.Join(dir, "backend"), filepath.Join(dir, "frontend")}
	if !slices.Equal(subdirs, want) {
		t.Errorf("subdirs = %v, want %v", subdirs, want)
	}
}

func TestCollectSubdirsSingleRoot(t *testing.T) {
	dir := t.TempDir()

	subdirs, err := collectSubdirs(dir, defaultSkipList)
	if err != nil {
		t.Fatalf("collectSubdirs failed: %v", err)
	}

	if len(subdirs) != 0 {
		t.Errorf("expected no subdirectories, got %v", subdirs)
	}
}

func TestDetectSubdirectoryManifests(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "frontend"))
	mustMkdir(t, filepath.Join(dir, "backend"))

	writeFile(t, filepath.Join(dir, "frontend"), "package.json", `{
		"engines": { "node": ">=20.0.0" },
		"devDependencies": { "eslint": "^9.0.0" }
	}`)
	mustMkdir(t, filepath.Join(dir, "frontend", "node_modules", ".bin"))
	writeExecutable(t, filepath.Join(dir, "frontend", "node_modules", ".bin"), "eslint")

	writeFile(t, filepath.Join(dir, "backend"), "go.mod", "module backend\n\ngo 1.21\n")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if d.Config.Tools["eslint"] != "9.x" {
		t.Errorf("Tools[eslint] = %q, want 9.x", d.Config.Tools["eslint"])
	}
	if d.Config.Tools["go"] != "1.21" {
		t.Errorf("Tools[go] = %q, want 1.21", d.Config.Tools["go"])
	}
	if d.Config.Tools["node"] != "20.x" {
		t.Errorf("Tools[node] = %q, want 20.x", d.Config.Tools["node"])
	}

	if !slices.Contains(d.ToolSubdirs["eslint"], "frontend") {
		t.Errorf("expected eslint tracked in frontend, got %v", d.ToolSubdirs["eslint"])
	}
	if !slices.Contains(d.ToolSubdirs["go"], "backend") {
		t.Errorf("expected go tracked in backend, got %v", d.ToolSubdirs["go"])
	}
}

func TestDetectAggregatesToolsFromMultipleSubdirectories(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "frontend"))
	mustMkdir(t, filepath.Join(dir, "backend"))

	writeFile(t, filepath.Join(dir, "frontend"), "package.json", `{
		"devDependencies": { "eslint": "^8.0.0" }
	}`)
	mustMkdir(t, filepath.Join(dir, "frontend", "node_modules", ".bin"))
	writeExecutable(t, filepath.Join(dir, "frontend", "node_modules", ".bin"), "eslint")

	writeFile(t, filepath.Join(dir, "backend"), "package.json", `{
		"devDependencies": { "prettier": "^3.0.0" }
	}`)
	mustMkdir(t, filepath.Join(dir, "backend", "node_modules", ".bin"))
	writeExecutable(t, filepath.Join(dir, "backend", "node_modules", ".bin"), "prettier")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if d.Config.Tools["eslint"] != "8.x" {
		t.Errorf("Tools[eslint] = %q, want 8.x", d.Config.Tools["eslint"])
	}
	if d.Config.Tools["prettier"] != "3.x" {
		t.Errorf("Tools[prettier] = %q, want 3.x", d.Config.Tools["prettier"])
	}

	if !slices.Contains(d.ToolSubdirs["eslint"], "frontend") {
		t.Errorf("expected eslint tracked in frontend, got %v", d.ToolSubdirs["eslint"])
	}
	if !slices.Contains(d.ToolSubdirs["prettier"], "backend") {
		t.Errorf("expected prettier tracked in backend, got %v", d.ToolSubdirs["prettier"])
	}
}

func TestDetectSubdirectoryEnvFiles(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "frontend"))
	mustMkdir(t, filepath.Join(dir, "backend"))

	writeFile(t, filepath.Join(dir, "frontend"), ".env.example", "NEXT_PUBLIC_KEY=\n")
	writeFile(t, filepath.Join(dir, "backend"), ".env", "DATABASE_URL=\n")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if !slices.Contains(d.Config.Env, "NEXT_PUBLIC_KEY") {
		t.Errorf("expected NEXT_PUBLIC_KEY in env, got %v", d.Config.Env)
	}
	if !slices.Contains(d.Config.Env, "DATABASE_URL") {
		t.Errorf("expected DATABASE_URL in env, got %v", d.Config.Env)
	}

	if !slices.Contains(d.EnvSubdirs["NEXT_PUBLIC_KEY"], "frontend") {
		t.Errorf("expected NEXT_PUBLIC_KEY tracked in frontend, got %v", d.EnvSubdirs["NEXT_PUBLIC_KEY"])
	}
	if !slices.Contains(d.EnvSubdirs["DATABASE_URL"], "backend") {
		t.Errorf("expected DATABASE_URL tracked in backend, got %v", d.EnvSubdirs["DATABASE_URL"])
	}
}

func TestDetectSubdirectoryProjectFiles(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "frontend"))
	mustMkdir(t, filepath.Join(dir, "backend"))

	writeFile(t, filepath.Join(dir, "frontend"), "Dockerfile", "FROM node:20\n")
	writeFile(t, filepath.Join(dir, "backend"), "Makefile", "build:\n")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if !slices.Contains(d.Config.Files, filepath.Join("frontend", "Dockerfile")) {
		t.Errorf("expected frontend/Dockerfile in files, got %v", d.Config.Files)
	}
	if !slices.Contains(d.Config.Files, filepath.Join("backend", "Makefile")) {
		t.Errorf("expected backend/Makefile in files, got %v", d.Config.Files)
	}
}

func TestDetectSkipsNestedGitAndNodeModules(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, ".git"))
	mustMkdir(t, filepath.Join(dir, "node_modules"))

	writeFile(t, filepath.Join(dir, ".git"), "package.json", `{"dependencies":{"ignored":"^1.0.0"}}`)
	writeFile(t, filepath.Join(dir, "node_modules"), "go.mod", "module ignored\n\ngo 1.21\n")

	mustMkdir(t, filepath.Join(dir, "src"))
	writeFile(t, filepath.Join(dir, "src"), "go.mod", "module src\n\ngo 1.22\n")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if _, ok := d.Config.Tools["ignored"]; ok {
		t.Errorf("expected ignored tool from .git to be skipped")
	}
	if d.Config.Tools["go"] != "1.22" {
		t.Errorf("Tools[go] = %q, want 1.22", d.Config.Tools["go"])
	}
}

func TestDetectDeduplicatesSameVersionAcrossSubdirs(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "frontend"))
	mustMkdir(t, filepath.Join(dir, "backend"))

	writeFile(t, filepath.Join(dir, "frontend"), "package.json", `{
		"devDependencies": { "eslint": "^8.0.0" }
	}`)
	mustMkdir(t, filepath.Join(dir, "frontend", "node_modules", ".bin"))
	writeExecutable(t, filepath.Join(dir, "frontend", "node_modules", ".bin"), "eslint")

	writeFile(t, filepath.Join(dir, "backend"), "package.json", `{
		"devDependencies": { "eslint": "^8.0.0" }
	}`)
	mustMkdir(t, filepath.Join(dir, "backend", "node_modules", ".bin"))
	writeExecutable(t, filepath.Join(dir, "backend", "node_modules", ".bin"), "eslint")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if d.Config.Tools["eslint"] != "8.x" {
		t.Errorf("Tools[eslint] = %q, want 8.x", d.Config.Tools["eslint"])
	}
	if len(d.ToolSubdirs["eslint"]) != 2 {
		t.Errorf("expected eslint in 2 subdirs, got %v", d.ToolSubdirs["eslint"])
	}
}

func TestDetectBackwardCompatibility(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"engines": { "node": ">=20.0.0" },
		"devDependencies": { "eslint": "^9.0.0" }
	}`)
	writeFile(t, dir, ".env.example", "DATABASE_URL=\n")
	writeFile(t, dir, "README.md", "# Project\n")

	binDir := filepath.Join(dir, "node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	writeExecutable(t, binDir, "eslint")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	wantTools := map[string]string{
		"node":   "20.x",
		"eslint": "9.x",
	}
	assertTools(t, d.Config.Tools, wantTools)

	wantEnv := []string{"DATABASE_URL"}
	assertSlice(t, d.Config.Env, wantEnv)

	wantFiles := []string{".env.example", "README.md", "package.json"}
	assertSlice(t, d.Config.Files, wantFiles)
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}
