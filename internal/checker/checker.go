// Package checker implements the environment health checks defined by the
// .env-doctor.yaml configuration.
package checker

import (
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"env-doctor/internal/config"
	"env-doctor/pkg/version"
	"github.com/joho/godotenv"
)

// Status represents the outcome of a single check.
type Status string

const (
	StatusPass Status = "PASS"
	StatusFail Status = "FAIL"
)

// Result describes the outcome of one health check.
type Result struct {
	Name     string
	Status   Status
	Expected string
	Actual   string
	Message  string
}

// CommandRunner abstracts running a binary so tests can inject fake output.
type CommandRunner interface {
	Run(name string, args ...string) (string, error)
}

type osRunner struct{}

func (osRunner) Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Checker runs all checks from a Config.
type Checker struct {
	runner     CommandRunner
	workingDir string
	envVars    map[string]string
}

// New returns a Checker that executes real binaries on the local system.
func New() *Checker {
	return NewWithDir(".")
}

// NewWithDir returns a Checker that operates in the given directory.
func NewWithDir(dir string) *Checker {
	c := &Checker{
		runner:     osRunner{},
		workingDir: dir,
		envVars:    make(map[string]string),
	}
	c.loadEnvFile()
	return c
}

// NewWithRunner returns a Checker that uses the provided command runner.
func NewWithRunner(r CommandRunner) *Checker {
	return &Checker{runner: r, workingDir: ".", envVars: make(map[string]string)}
}

func (c *Checker) loadEnvFile() {
	envPath := filepath.Join(c.workingDir, ".env")
	if _, err := os.Stat(envPath); err != nil {
		return
	}
	vars, err := godotenv.Read(envPath)
	if err != nil {
		return
	}
	c.envVars = vars
}

// Run executes every check in cfg and returns the ordered results.
func (c *Checker) Run(cfg config.Config) []Result {
	var results []Result

	for name, expected := range cfg.Tools {
		results = append(results, c.checkTool(name, expected))
	}
	for _, name := range cfg.Env {
		results = append(results, c.checkEnv(name))
	}
	for _, path := range cfg.Files {
		results = append(results, c.checkFile(path))
	}
	for port, expected := range cfg.Ports {
		results = append(results, c.checkPort(port, expected))
	}

	// Stable order makes output and tests deterministic.
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results
}

func (c *Checker) checkTool(name, expected string) Result {
	label := fmt.Sprintf("tool: %s", name)
	binPath := c.resolveBinary(name)

	out, err := c.runner.Run(binPath, "--version")
	if err != nil {
		out, err = c.runner.Run(binPath, "version")
	}
	if err != nil {
		return Result{
			Name:     label,
			Status:   StatusFail,
			Expected: expected,
			Actual:   "not found",
			Message:  fmt.Sprintf("could not run %s: %v", name, err),
		}
	}

	ok, actual, err := version.Match(out, expected)
	if err != nil {
		return Result{
			Name:     label,
			Status:   StatusFail,
			Expected: expected,
			Actual:   actual,
			Message:  fmt.Sprintf("version parse error: %v", err),
		}
	}
	if !ok {
		return Result{
			Name:     label,
			Status:   StatusFail,
			Expected: expected,
			Actual:   actual,
			Message:  fmt.Sprintf("expected %s, got %s", expected, actual),
		}
	}

	return Result{
		Name:     label,
		Status:   StatusPass,
		Expected: expected,
		Actual:   actual,
		Message:  "version OK",
	}
}

// resolveBinary looks for the binary in the root-level node_modules/.bin and
// .venv/bin directories first, then searches those directories in every
// subdirectory, and finally falls back to the system PATH. Matches are
// returned in deterministic lexicographic order so resolution is reproducible.
func (c *Checker) resolveBinary(name string) string {
	rootCandidates := []string{
		filepath.Join(c.workingDir, "node_modules", ".bin", name),
		filepath.Join(c.workingDir, ".venv", "bin", name),
	}
	for _, p := range rootCandidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	var matches []string
	_ = filepath.WalkDir(c.workingDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == c.workingDir {
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		switch filepath.Base(path) {
		case "node_modules":
			p := filepath.Join(path, ".bin", name)
			if _, err := os.Stat(p); err == nil {
				matches = append(matches, p)
			}
			return filepath.SkipDir
		case ".venv":
			p := filepath.Join(path, "bin", name)
			if _, err := os.Stat(p); err == nil {
				matches = append(matches, p)
			}
			return filepath.SkipDir
		}
		return nil
	})

	if len(matches) == 0 {
		return name
	}
	sort.Strings(matches)
	return matches[0]
}

func (c *Checker) checkEnv(name string) Result {
	label := fmt.Sprintf("env: %s", name)
	// Check system environment first.
	if _, ok := os.LookupEnv(name); ok {
		return Result{
			Name:     label,
			Status:   StatusPass,
			Expected: "set",
			Actual:   "set",
			Message:  "environment variable is set",
		}
	}
	// Fallback to the project's .env file.
	if val, ok := c.envVars[name]; ok && val != "" {
		return Result{
			Name:     label,
			Status:   StatusPass,
			Expected: "set",
			Actual:   "set (from .env)",
			Message:  "environment variable is set in .env file",
		}
	}
	return Result{
		Name:     label,
		Status:   StatusFail,
		Expected: "set",
		Actual:   "unset",
		Message:  "environment variable is not set",
	}
}

func (c *Checker) checkFile(path string) Result {
	label := fmt.Sprintf("file: %s", path)
	_, err := os.Stat(path)
	if err != nil {
		return Result{
			Name:     label,
			Status:   StatusFail,
			Expected: "exists",
			Actual:   "missing",
			Message:  fmt.Sprintf("file not found: %v", err),
		}
	}
	return Result{
		Name:     label,
		Status:   StatusPass,
		Expected: "exists",
		Actual:   "exists",
		Message:  "file exists",
	}
}

func (c *Checker) checkPort(port int, expected string) Result {
	label := fmt.Sprintf("port: %d", port)
	expected = strings.ToLower(strings.TrimSpace(expected))

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		if expected == "occupied" {
			return Result{
				Name:     label,
				Status:   StatusPass,
				Expected: expected,
				Actual:   "occupied",
				Message:  "port is listening",
			}
		}
		return Result{
			Name:     label,
			Status:   StatusFail,
			Expected: expected,
			Actual:   "occupied",
			Message:  "port is in use",
		}
	}
	_ = ln.Close()

	if expected == "free" {
		return Result{
			Name:     label,
			Status:   StatusPass,
			Expected: expected,
			Actual:   "free",
			Message:  "port is free",
		}
	}
	return Result{
		Name:     label,
		Status:   StatusFail,
		Expected: expected,
		Actual:   "free",
		Message:  "port is not listening",
	}
}
