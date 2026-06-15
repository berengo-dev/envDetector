package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".env-doctor.yaml")
	content := `
version: "1"
tools:
  go: "1.21"
  node: "20.x"
env:
  - DATABASE_URL
  - REDIS_URL
files:
  - .env
ports:
  3000: occupied
  5432: free
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Version != "1" {
		t.Errorf("Version = %q, want %q", cfg.Version, "1")
	}

	if got, want := cfg.Tools["go"], "1.21"; got != want {
		t.Errorf("Tools[go] = %q, want %q", got, want)
	}
	if got, want := cfg.Tools["node"], "20.x"; got != want {
		t.Errorf("Tools[node] = %q, want %q", got, want)
	}

	if len(cfg.Env) != 2 || cfg.Env[0] != "DATABASE_URL" || cfg.Env[1] != "REDIS_URL" {
		t.Errorf("Env = %v, want [DATABASE_URL REDIS_URL]", cfg.Env)
	}

	if len(cfg.Files) != 1 || cfg.Files[0] != ".env" {
		t.Errorf("Files = %v, want [.env]", cfg.Files)
	}

	if cfg.Ports[3000] != "occupied" {
		t.Errorf("Ports[3000] = %q, want occupied", cfg.Ports[3000])
	}
	if cfg.Ports[5432] != "free" {
		t.Errorf("Ports[5432] = %q, want free", cfg.Ports[5432])
	}
}
