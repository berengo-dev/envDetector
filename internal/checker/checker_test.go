package checker

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"env-doctor/internal/config"
)

type mockRunner struct {
	outputs map[string]mockResult
}

type mockResult struct {
	out string
	err error
}

func (m *mockRunner) Run(name string, args ...string) (string, error) {
	key := name + " " + strings.Join(args, " ")
	r, ok := m.outputs[key]
	if !ok {
		return "", fmt.Errorf("unknown command: %s", key)
	}
	return r.out, r.err
}

func TestCheckToolPass(t *testing.T) {
	mr := &mockRunner{
		outputs: map[string]mockResult{
			"go --version": {out: "go version go1.21.5 linux/amd64"},
		},
	}
	c := NewWithRunner(mr)
	cfg := config.Config{Tools: map[string]string{"go": "1.21"}}
	results := c.Run(cfg)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusPass {
		t.Errorf("expected PASS, got %s: %s", results[0].Status, results[0].Message)
	}
}

func TestCheckToolFail(t *testing.T) {
	mr := &mockRunner{
		outputs: map[string]mockResult{
			"node --version": {out: "v18.17.0"},
		},
	}
	c := NewWithRunner(mr)
	cfg := config.Config{Tools: map[string]string{"node": "20.x"}}
	results := c.Run(cfg)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusFail {
		t.Errorf("expected FAIL, got %s: %s", results[0].Status, results[0].Message)
	}
}

func TestCheckToolFallbackToVersion(t *testing.T) {
	mr := &mockRunner{
		outputs: map[string]mockResult{
			"docker --version": {err: fmt.Errorf("exit status 1")},
			"docker version":   {out: "Docker version 24.0.7"},
		},
	}
	c := NewWithRunner(mr)
	cfg := config.Config{Tools: map[string]string{"docker": "24.x"}}
	results := c.Run(cfg)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusPass {
		t.Errorf("expected PASS, got %s: %s", results[0].Status, results[0].Message)
	}
}

func TestCheckEnv(t *testing.T) {
	t.Setenv("ENV_DOCTOR_TEST_VAR", "value")
	c := New()
	cfg := config.Config{Env: []string{"ENV_DOCTOR_TEST_VAR", "ENV_DOCTOR_MISSING_VAR"}}
	results := c.Run(cfg)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	byName := make(map[string]Result, len(results))
	for _, r := range results {
		byName[r.Name] = r
	}

	if r, ok := byName["env: ENV_DOCTOR_TEST_VAR"]; !ok || r.Status != StatusPass {
		t.Errorf("expected set var to PASS, got %+v", r)
	}
	if r, ok := byName["env: ENV_DOCTOR_MISSING_VAR"]; !ok || r.Status != StatusFail {
		t.Errorf("expected missing var to FAIL, got %+v", r)
	}
}

func TestCheckFile(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(existing, []byte("ok"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	c := New()
	cfg := config.Config{Files: []string{existing, filepath.Join(dir, "missing.txt")}}
	results := c.Run(cfg)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Status != StatusPass {
		t.Errorf("expected existing file to PASS, got %s", results[0].Status)
	}
	if results[1].Status != StatusFail {
		t.Errorf("expected missing file to FAIL, got %s", results[1].Status)
	}
}

func TestCheckFileRelativeToConfigDir(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "config.json")
	if err := os.WriteFile(existing, []byte("{}"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	c := NewWithDir(dir)
	cfg := config.Config{Files: []string{"config.json"}}
	results := c.Run(cfg)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusPass {
		t.Errorf("expected relative file under workingDir to PASS, got %s: %s", results[0].Status, results[0].Message)
	}
}

func TestCheckPortFree(t *testing.T) {
	// Bind to a random port and then release it so the check sees it as free.
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	c := New()
	results := c.Run(config.Config{Ports: map[int]string{port: "free"}})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusPass {
		t.Errorf("expected free port to PASS, got %s: %s", results[0].Status, results[0].Message)
	}
}

func TestCheckPortOccupied(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	c := New()
	results := c.Run(config.Config{Ports: map[int]string{port: "occupied"}})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusPass {
		t.Errorf("expected occupied port to PASS, got %s: %s", results[0].Status, results[0].Message)
	}
}

func TestClassifyPortError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		want    string
		message string
	}{
		{
			name:    "nil means free",
			err:     nil,
			want:    "free",
			message: "port is free",
		},
		{
			name:    "EADDRINUSE means occupied",
			err:     &net.OpError{Op: "listen", Net: "tcp", Err: syscall.EADDRINUSE},
			want:    "occupied",
			message: "port is occupied",
		},
		{
			name:    "EACCES means permission denied",
			err:     &net.OpError{Op: "listen", Net: "tcp", Err: syscall.EACCES},
			want:    "permission denied",
			message: "port check failed: permission denied",
		},
		{
			name:    "EADDRNOTAVAIL means address not available",
			err:     &net.OpError{Op: "listen", Net: "tcp", Err: syscall.EADDRNOTAVAIL},
			want:    "address not available",
			message: "port check failed: address not available",
		},
		{
			name:    "unknown OpError means failed to check",
			err:     &net.OpError{Op: "listen", Net: "tcp", Err: errors.New("weird")},
			want:    "failed to check",
			message: "port check failed: weird",
		},
		{
			name:    "non-OpError means failed to check",
			err:     errors.New("boom"),
			want:    "failed to check",
			message: "port check failed: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, message := classifyPortError(tt.err)
			if actual != tt.want {
				t.Errorf("classifyPortError(%v) actual = %q, want %q", tt.err, actual, tt.want)
			}
			if message != tt.message {
				t.Errorf("classifyPortError(%v) message = %q, want %q", tt.err, message, tt.message)
			}
		})
	}
}
