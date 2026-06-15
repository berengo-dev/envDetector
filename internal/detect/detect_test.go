package detect

import (
	"os"
	"path/filepath"
	"testing"

	"env-doctor/internal/config"
)

func TestDetectPackageJSONManifest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"name": "demo",
		"engines": { "node": ">=20.0.0" },
		"dependencies": {
			"next": "16.2.7",
			"prisma": "^7.8.0"
		},
		"devDependencies": {
			"typescript": "^5",
			"eslint": "^9",
			"vitest": "^4.1.8"
		}
	}`)
	writeFile(t, dir, ".env.example", "DATABASE_URL=\nNEXT_PUBLIC_KEY=\n")
	writeFile(t, dir, "next.config.ts", "export default {}\n")
	writeFile(t, dir, "tsconfig.json", "{}\n")

	// Create local binaries so the extractor recognises them as tools.
	binDir := filepath.Join(dir, "node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	writeExecutable(t, binDir, "next")
	writeExecutable(t, binDir, "prisma")
	writeExecutable(t, binDir, "eslint")
	writeExecutable(t, binDir, "vitest")
	writeExecutable(t, binDir, "tsc")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	wantTools := map[string]string{
		"node":   "20.x",
		"next":   "16.x",
		"prisma": "7.x",
		"tsc":    "5.x",
		"eslint": "9.x",
		"vitest": "4.x",
	}
	assertTools(t, d.Config.Tools, wantTools)

	wantEnv := []string{"DATABASE_URL", "NEXT_PUBLIC_KEY"}
	assertSlice(t, d.Config.Env, wantEnv)

	wantFiles := []string{".env.example", "next.config.ts", "package.json", "tsconfig.json"}
	assertSlice(t, d.Config.Files, wantFiles)

	if len(d.Config.Ports) != 0 {
		t.Errorf("expected no auto-generated ports, got %v", d.Config.Ports)
	}
}

func TestDetectGoModManifest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module demo\n\ngo 1.21\n")
	writeFile(t, dir, "main.go", "package main\n")
	writeFile(t, dir, "Makefile", "build:\n")
	writeFile(t, dir, "Dockerfile", "FROM golang:1.21\n")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	wantTools := map[string]string{
		"go":     "1.21",
		"make":   "latest",
		"docker": "latest",
	}
	assertTools(t, d.Config.Tools, wantTools)

	wantFiles := []string{"Dockerfile", "Makefile", "go.mod", "main.go"}
	assertSlice(t, d.Config.Files, wantFiles)
}

func TestDetectMixedManifests(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"dependencies": { "react": "^18.2.0" }
	}`)
	writeFile(t, dir, "go.mod", "module demo\n\ngo 1.22\n")
	writeFile(t, dir, "README.md", "# Project\n")

	// Create local binary so react is recognised as a tool.
	binDir := filepath.Join(dir, "node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	writeExecutable(t, binDir, "react")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if _, ok := d.Config.Tools["react"]; !ok {
		t.Errorf("expected react tool from package.json")
	}
	if _, ok := d.Config.Tools["go"]; !ok {
		t.Errorf("expected go tool from go.mod")
	}
	if d.Config.Tools["react"] != "18.x" {
		t.Errorf("Tools[react] = %q, want 18.x", d.Config.Tools["react"])
	}
	if d.Config.Tools["go"] != "1.22" {
		t.Errorf("Tools[go] = %q, want 1.22", d.Config.Tools["go"])
	}
}

func TestDetectEnvExample(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DATABASE_URL=\nCLERK_SECRET_KEY=\nAPI_KEY=\n")
	writeFile(t, dir, "README.md", "# Project\n")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	wantEnv := []string{"API_KEY", "CLERK_SECRET_KEY", "DATABASE_URL"}
	assertSlice(t, d.Config.Env, wantEnv)

	wantFiles := []string{".env.example", "README.md"}
	assertSlice(t, d.Config.Files, wantFiles)
}

func TestDetectNoProject(t *testing.T) {
	dir := t.TempDir()
	_, err := Detect(dir)
	if err == nil {
		t.Fatal("expected error for empty project, got nil")
	}
}

func TestDetectGenericDockerCompose(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "docker-compose.yml", "services:\n  db:\n")
	writeFile(t, dir, ".env.example", "API_KEY=\n")
	writeFile(t, dir, "README.md", "# Project\n")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if d.Config.Tools["docker"] != "latest" || d.Config.Tools["docker-compose"] != "latest" {
		t.Errorf("unexpected tools: %v", d.Config.Tools)
	}
	wantFiles := []string{".env.example", "README.md", "docker-compose.yml"}
	assertSlice(t, d.Config.Files, wantFiles)
	wantEnv := []string{"API_KEY"}
	assertSlice(t, d.Config.Env, wantEnv)
}

func TestDetectPythonManifest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "django>=4\npsycopg2\n")
	writeFile(t, dir, ".env", "SECRET_KEY=\n")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if d.Config.Tools["python"] != "3.x" {
		t.Errorf("Tools[python] = %q, want 3.x", d.Config.Tools["python"])
	}
	if d.Config.Tools["django"] != "latest" {
		t.Errorf("Tools[django] = %q, want latest", d.Config.Tools["django"])
	}
	if d.Config.Tools["psycopg2"] != "latest" {
		t.Errorf("Tools[psycopg2] = %q, want latest", d.Config.Tools["psycopg2"])
	}
	if len(d.Config.Ports) != 0 {
		t.Errorf("expected no auto-generated ports, got %v", d.Config.Ports)
	}
}

func TestGenerateNoPorts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"engines": { "node": ">=18.0.0" },
		"dependencies": { "next": "14.0.0" }
	}`)
	writeFile(t, dir, ".env.example", "DATABASE_URL=\n")

	// Create local binary so next is recognised as a tool.
	binDir := filepath.Join(dir, "node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	writeExecutable(t, binDir, "next")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	yaml, err := Generate(d)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !contains(yaml, "# Auto-generated by env-doctor init --auto") {
		t.Errorf("generated YAML missing auto-generated header")
	}
	if !contains(yaml, "# Add port checks manually:") {
		t.Errorf("generated YAML missing manual port placeholder")
	}
	if contains(yaml, "ports:") && !contains(yaml, "# ports:") {
		t.Errorf("generated YAML contains active ports section")
	}

	cfgPath := filepath.Join(dir, ".env-doctor.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("write generated yaml: %v", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load generated yaml: %v", err)
	}
	if cfg.Version != "1" {
		t.Errorf("Version = %q, want 1", cfg.Version)
	}
	if cfg.Tools["node"] != "18.x" {
		t.Errorf("Tools[node] = %q, want 18.x", d.Config.Tools["node"])
	}
	if cfg.Tools["next"] != "14.x" {
		t.Errorf("Tools[next] = %q, want 14.x", d.Config.Tools["next"])
	}
	if len(cfg.Ports) != 0 {
		t.Errorf("expected no ports in loaded config, got %v", cfg.Ports)
	}
}

func TestDetectMalformedPackageJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", "not json")

	_, err := Detect(dir)
	if err == nil {
		t.Fatal("expected error for malformed package.json, got nil")
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func writeExecutable(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write executable %s: %v", name, err)
	}
}

func assertTools(t *testing.T, got, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("Tools count = %d, want %d; got %v", len(got), len(want), got)
	}
	for name, expected := range want {
		if got[name] != expected {
			t.Errorf("Tools[%s] = %q, want %q", name, got[name], expected)
		}
	}
}

func assertSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("slice length = %d, want %d; got %v", len(got), len(want), got)
	}
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Errorf("slice[%d] = %q, want %q", i, safeGet(got, i), want[i])
		}
	}
}

func safeGet(s []string, i int) string {
	if i < 0 || i >= len(s) {
		return "<missing>"
	}
	return s[i]
}

func contains(s, substr string) bool {
	return len(substr) <= len(s) && (s == substr || len(s) > 0 && containsSub(s, substr))
}

func containsSub(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
