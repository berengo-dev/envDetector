package checker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBinaryRootFirst(t *testing.T) {
	dir := t.TempDir()
	rootBin := filepath.Join(dir, "node_modules", ".bin", "eslint")
	subBin := filepath.Join(dir, "frontend", "node_modules", ".bin", "eslint")
	if err := os.MkdirAll(filepath.Dir(rootBin), 0755); err != nil {
		t.Fatalf("mkdir root bin: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(subBin), 0755); err != nil {
		t.Fatalf("mkdir sub bin: %v", err)
	}
	if err := os.WriteFile(rootBin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write root bin: %v", err)
	}
	if err := os.WriteFile(subBin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write sub bin: %v", err)
	}

	c := &Checker{runner: &mockRunner{}, workingDir: dir}
	got := c.resolveBinary("eslint")
	if got != rootBin {
		t.Errorf("expected root binary %q, got %q", rootBin, got)
	}
}

func TestResolveBinarySubdirFallback(t *testing.T) {
	dir := t.TempDir()
	subBin := filepath.Join(dir, "frontend", "node_modules", ".bin", "eslint")
	if err := os.MkdirAll(filepath.Dir(subBin), 0755); err != nil {
		t.Fatalf("mkdir sub bin: %v", err)
	}
	if err := os.WriteFile(subBin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write sub bin: %v", err)
	}

	c := &Checker{runner: &mockRunner{}, workingDir: dir}
	got := c.resolveBinary("eslint")
	if got != subBin {
		t.Errorf("expected subdir binary %q, got %q", subBin, got)
	}
}

func TestResolveBinaryPATHFallback(t *testing.T) {
	dir := t.TempDir()
	c := &Checker{runner: &mockRunner{}, workingDir: dir}
	got := c.resolveBinary("eslint")
	if got != "eslint" {
		t.Errorf("expected PATH fallback %q, got %q", "eslint", got)
	}
}

func TestResolveBinaryDeterministicOrder(t *testing.T) {
	dir := t.TempDir()
	aBin := filepath.Join(dir, "a", "node_modules", ".bin", "eslint")
	bBin := filepath.Join(dir, "b", "node_modules", ".bin", "eslint")
	if err := os.MkdirAll(filepath.Dir(aBin), 0755); err != nil {
		t.Fatalf("mkdir a bin: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(bBin), 0755); err != nil {
		t.Fatalf("mkdir b bin: %v", err)
	}
	if err := os.WriteFile(aBin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write a bin: %v", err)
	}
	if err := os.WriteFile(bBin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write b bin: %v", err)
	}

	c := &Checker{runner: &mockRunner{}, workingDir: dir}
	got := c.resolveBinary("eslint")
	if got != aBin {
		t.Errorf("expected deterministic first %q, got %q", aBin, got)
	}
}

func TestResolveBinaryVenvRootFirst(t *testing.T) {
	dir := t.TempDir()
	rootBin := filepath.Join(dir, ".venv", "bin", "python")
	subBin := filepath.Join(dir, "backend", ".venv", "bin", "python")
	if err := os.MkdirAll(filepath.Dir(rootBin), 0755); err != nil {
		t.Fatalf("mkdir root venv: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(subBin), 0755); err != nil {
		t.Fatalf("mkdir sub venv: %v", err)
	}
	if err := os.WriteFile(rootBin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write root venv bin: %v", err)
	}
	if err := os.WriteFile(subBin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write sub venv bin: %v", err)
	}

	c := &Checker{runner: &mockRunner{}, workingDir: dir}
	got := c.resolveBinary("python")
	if got != rootBin {
		t.Errorf("expected root venv binary %q, got %q", rootBin, got)
	}
}

func TestResolveBinaryVenvSubdirFallback(t *testing.T) {
	dir := t.TempDir()
	subBin := filepath.Join(dir, "backend", ".venv", "bin", "python")
	if err := os.MkdirAll(filepath.Dir(subBin), 0755); err != nil {
		t.Fatalf("mkdir sub venv: %v", err)
	}
	if err := os.WriteFile(subBin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write sub venv bin: %v", err)
	}

	c := &Checker{runner: &mockRunner{}, workingDir: dir}
	got := c.resolveBinary("python")
	if got != subBin {
		t.Errorf("expected subdir venv binary %q, got %q", subBin, got)
	}
}

func TestResolveBinaryVenvNoDot(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "venv", "bin", "python")
	if err := os.MkdirAll(filepath.Dir(binPath), 0755); err != nil {
		t.Fatalf("mkdir venv: %v", err)
	}
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write venv bin: %v", err)
	}

	c := &Checker{runner: &mockRunner{}, workingDir: dir}
	got := c.resolveBinary("python")
	if got != binPath {
		t.Errorf("expected venv binary %q, got %q", binPath, got)
	}
}

func TestResolveBinaryNonStandardBinDir(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "tools", "bin", "go")
	if err := os.MkdirAll(filepath.Dir(binPath), 0755); err != nil {
		t.Fatalf("mkdir tools bin: %v", err)
	}
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write tools bin: %v", err)
	}

	c := &Checker{runner: &mockRunner{}, workingDir: dir}
	got := c.resolveBinary("go")
	if got != binPath {
		t.Errorf("expected tools/bin binary %q, got %q", binPath, got)
	}
}
