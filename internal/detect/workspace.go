package detect

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// parseWorkspaceHints returns directories declared by workspace configuration
// files in dir. It parses npm-style package.json workspaces and
// pnpm-workspace.yaml, expands glob patterns, and returns an ordered list of
// absolute directory paths that should be scanned before the fallback WalkDir.
//
// If both package.json workspaces and pnpm-workspace.yaml are present, the
// config matching the project's lock file is preferred. When no lock file is
// present, the workspace paths from both configs are combined.
func parseWorkspaceHints(dir string) ([]string, error) {
	npmHints, err := parsePackageJSONWorkspaces(dir)
	if err != nil {
		return nil, err
	}

	pnpmHints, err := parsePnpmWorkspaceYAML(dir)
	if err != nil {
		return nil, err
	}

	switch {
	case len(npmHints) > 0 && len(pnpmHints) == 0:
		return npmHints, nil
	case len(npmHints) == 0 && len(pnpmHints) > 0:
		return pnpmHints, nil
	case len(npmHints) == 0 && len(pnpmHints) == 0:
		return nil, nil
	}

	// Both configs present. Prefer the one matching the lock file.
	hasPackageLock := fileExists(filepath.Join(dir, "package-lock.json"))
	hasPnpmLock := fileExists(filepath.Join(dir, "pnpm-lock.yaml"))

	if hasPackageLock && !hasPnpmLock {
		return npmHints, nil
	}
	if hasPnpmLock && !hasPackageLock {
		return pnpmHints, nil
	}

	// No decisive lock file (none or both): union both sets of paths.
	return unionPaths(npmHints, pnpmHints), nil
}

// parsePackageJSONWorkspaces extracts workspace directory patterns from the
// root package.json. It supports both the array form and the object form
// ({"packages": [...]}).
func parsePackageJSONWorkspaces(dir string) ([]string, error) {
	path := filepath.Join(dir, "package.json")
	if !fileExists(path) {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read package.json: %w", err)
	}

	var pkg struct {
		Workspaces any `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}

	var patterns []string
	switch v := pkg.Workspaces.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				patterns = append(patterns, s)
			}
		}
	case map[string]any:
		if arr, ok := v["packages"].([]any); ok {
			for _, item := range arr {
				if s, ok := item.(string); ok {
					patterns = append(patterns, s)
				}
			}
		}
	}

	return expandGlobs(dir, patterns)
}

// parsePnpmWorkspaceYAML extracts workspace directory patterns from
// pnpm-workspace.yaml. Malformed YAML is logged as a warning and returns an
// empty list so detection can fall back to WalkDir.
func parsePnpmWorkspaceYAML(dir string) ([]string, error) {
	path := filepath.Join(dir, "pnpm-workspace.yaml")
	if !fileExists(path) {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pnpm-workspace.yaml: %w", err)
	}

	var ws struct {
		Packages []string `yaml:"packages"`
	}
	if err := yaml.Unmarshal(data, &ws); err != nil {
		log.Printf("warning: malformed %s, falling back to WalkDir: %v", path, err)
		return nil, nil
	}

	return expandGlobs(dir, ws.Packages)
}

// expandGlobs expands a list of glob patterns relative to dir and returns the
// matching directories in sorted order. Files and duplicate directories are
// ignored.
func expandGlobs(dir string, patterns []string) ([]string, error) {
	seen := make(map[string]struct{})
	var dirs []string

	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			info, err := os.Stat(m)
			if err != nil || !info.IsDir() {
				continue
			}
			if _, ok := seen[m]; ok {
				continue
			}
			seen[m] = struct{}{}
			dirs = append(dirs, m)
		}
	}

	sort.Strings(dirs)
	return dirs, nil
}

// unionPaths returns the sorted union of two absolute path lists, removing
// duplicates.
func unionPaths(a, b []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, p := range a {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	for _, p := range b {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
