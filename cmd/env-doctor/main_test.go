package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"env-doctor/internal/checker"
)

func TestCheckCmdReturnsErrChecksFailed(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".env-doctor.yaml")
	content := `version: "1"
files:
  - definitely-missing-file.txt
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	oldCfgFile := cfgFile
	defer func() { cfgFile = oldCfgFile }()
	cfgFile = cfgPath

	err := checkCmd.RunE(checkCmd, []string{})
	if !errors.Is(err, checker.ErrChecksFailed) {
		t.Fatalf("expected ErrChecksFailed, got %v", err)
	}
}
