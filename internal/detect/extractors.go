package detect

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"env-doctor/pkg/version"
)

// ManifestExtractor knows how to parse a specific file type but remains
// agnostic about the technology it represents.
type ManifestExtractor interface {
	// FilePatterns returns glob patterns this extractor handles.
	FilePatterns() []string

	// Extract reads a manifest file and returns tools/versions found.
	// The returned map maps tool_name -> version_constraint.
	Extract(path string) (map[string]string, error)
}

// DefaultExtractors returns the built-in manifest extractors in priority order.
func DefaultExtractors() []ManifestExtractor {
	return []ManifestExtractor{
		&JSONExtractor{},
		&GoModExtractor{},
		&PythonExtractor{},
		&GenericExtractor{},
	}
}

// JSONExtractor parses JSON manifest files such as package.json.
type JSONExtractor struct{}

func (e *JSONExtractor) FilePatterns() []string {
	return []string{"package.json", "*.json"}
}

func (e *JSONExtractor) Extract(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}

	// Only treat this JSON file as a manifest when it contains at least one
	// common dependency field. package.json is always processed.
	if filepath.Base(path) != "package.json" {
		hasManifestShape := false
		for _, key := range []string{"dependencies", "devDependencies", "peerDependencies", "engines"} {
			if _, ok := doc[key]; ok {
				hasManifestShape = true
				break
			}
		}
		if !hasManifestShape {
			return nil, nil
		}
	}

	// Build a set of dependency names for quick lookup.
	allDeps := make(map[string]struct{})
	collectDeps := func(key string) {
		m, ok := doc[key].(map[string]any)
		if !ok {
			return
		}
		for name := range m {
			allDeps[name] = struct{}{}
		}
	}
	collectDeps("dependencies")
	collectDeps("devDependencies")
	collectDeps("peerDependencies")

	// Collect all declared deps and their versions.
	depVersions := make(map[string]string)
	extractDeps := func(key string) {
		m, ok := doc[key].(map[string]any)
		if !ok {
			return
		}
		for name, val := range m {
			v, ok := val.(string)
			if !ok {
				continue
			}
			wildcard, ok := version.ConvertSemverToWildcard(v)
			if !ok {
				continue
			}
			// Dependency declarations without an operator prefix are exact
			// versions. Convert them to major.x for friendlier matching.
			if !hasOperatorPrefix(v) && strings.Count(wildcard, ".") == 1 && !strings.HasSuffix(wildcard, ".x") {
				parts := strings.Split(wildcard, ".")
				wildcard = parts[0] + ".x"
			}
			depVersions[name] = wildcard
		}
	}
	extractDeps("dependencies")
	extractDeps("devDependencies")
	extractDeps("peerDependencies")

	// Filter: only keep deps that actually have a binary in node_modules/.bin.
	// This avoids listing pure libraries (e.g. react, date-fns) as tools.
	binDir := filepath.Join(filepath.Dir(path), "node_modules", ".bin")
	tools := make(map[string]string)
	for name, ver := range depVersions {
		// npm packages can expose binaries with the package name or a
		// different name. Check the most common mapping.
		binName := npmBinName(name)
		if fileExists(filepath.Join(binDir, binName)) {
			// Store under the actual binary name, not the package name.
			tools[binName] = ver
			continue
		}
		// Some packages expose binaries with different names than the
		// package name. Keep the package name if it has a bin.
		if fileExists(filepath.Join(binDir, name)) {
			tools[name] = ver
		}
	}

	// The npm package "typescript" provides the "tsc" binary.
	if _, ok := allDeps["typescript"]; ok {
		if v, ok := depVersions["typescript"]; ok {
			tools["tsc"] = v
		} else {
			tools["tsc"] = "latest"
		}
	}

	if engines, ok := doc["engines"].(map[string]any); ok {
		for name, val := range engines {
			v, ok := val.(string)
			if !ok {
				continue
			}
			if wildcard, ok := version.ConvertSemverToWildcard(v); ok {
				tools[name] = wildcard
			}
		}
	}

	return tools, nil
}

// npmBinName converts an npm package name to its most likely binary name.
// For scoped packages like "@scope/name" it returns "name".
func npmBinName(pkg string) string {
	if strings.HasPrefix(pkg, "@") {
		parts := strings.Split(pkg, "/")
		if len(parts) == 2 {
			return parts[1]
		}
	}
	return pkg
}

// GoModExtractor parses go.mod files.
type GoModExtractor struct{}

func (e *GoModExtractor) FilePatterns() []string {
	return []string{"go.mod"}
}

var goVersionRe = regexp.MustCompile(`(?m)^go\s+(\S+)`)

func (e *GoModExtractor) Extract(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	tools := make(map[string]string)
	if m := goVersionRe.FindStringSubmatch(string(data)); m != nil {
		v := version.Extract(m[1])
		parts := strings.Split(v, ".")
		switch len(parts) {
		case 1:
			tools["go"] = parts[0] + ".x"
		default:
			tools["go"] = parts[0] + "." + parts[1]
		}
	}

	dir := filepath.Dir(path)
	if fileExists(filepath.Join(dir, "Makefile")) {
		tools["make"] = "latest"
	}
	if fileExists(filepath.Join(dir, "Dockerfile")) {
		tools["docker"] = "latest"
	}

	return tools, nil
}

// PythonExtractor parses Python manifest files.
type PythonExtractor struct{}

func (e *PythonExtractor) FilePatterns() []string {
	return []string{"requirements.txt", "pyproject.toml"}
}

var pyprojectPythonRe = regexp.MustCompile(`(?m)^requires-python\s*=\s*["']([^"']+)["']`)

func (e *PythonExtractor) Extract(path string) (map[string]string, error) {
	tools := make(map[string]string)
	dir := filepath.Dir(path)

	switch filepath.Base(path) {
	case "requirements.txt":
		tools["python"] = "3.x"
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			name := strings.TrimSpace(requirementName(line))
			if name != "" && hasVenvExecutable(dir, name) {
				tools[name] = "latest"
			}
		}

	case "pyproject.toml":
		tools["python"] = "3.x"
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if m := pyprojectPythonRe.FindStringSubmatch(string(data)); m != nil {
			if wildcard, ok := version.ConvertSemverToWildcard(m[1]); ok {
				tools["python"] = wildcard
			}
		}
		for _, dep := range extractPyprojectDeps(string(data)) {
			tools[dep] = "latest"
		}
	}

	// Prefer a local virtual environment binary over the system python when
	// one is present.
	for _, venv := range []string{".venv", "venv"} {
		if fileExists(filepath.Join(dir, venv, "bin", "python")) {
			tools["python"] = "latest"
			break
		}
	}

	return tools, nil
}

func hasVenvExecutable(dir, name string) bool {
	for _, venv := range []string{".venv", "venv"} {
		if fileExists(filepath.Join(dir, venv, "bin", name)) {
			return true
		}
	}
	return false
}

func hasOperatorPrefix(c string) bool {
	c = strings.TrimSpace(strings.ToLower(c))
	for _, op := range []string{"^", "~", ">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(c, op) {
			return true
		}
	}
	return false
}

func requirementName(line string) string {
	// Strip extras, version specifiers, and markers.
	// Examples:
	//   django>=4
	//   requests[security]>=2.0
	//   package==1.0; python_version>="3.8"
	if idx := strings.IndexAny(line, "[<>=!~;"); idx != -1 {
		line = line[:idx]
	}
	return strings.TrimSpace(line)
}

func extractPyprojectDeps(content string) []string {
	var deps []string
	inDeps := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[project.dependencies]") ||
			strings.HasPrefix(trimmed, "[tool.poetry.dependencies]") {
			inDeps = true
			continue
		}
		if inDeps {
			if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
				break
			}
			if strings.HasPrefix(trimmed, "#") || trimmed == "" {
				continue
			}
			name := strings.TrimSpace(requirementName(trimmed))
			if name != "" {
				deps = append(deps, name)
			}
		}
	}
	return deps
}

// GenericExtractor handles common project files that are not tied to a
// specific stack.
type GenericExtractor struct{}

func (e *GenericExtractor) FilePatterns() []string {
	return []string{".env", ".env.example", "docker-compose.yml", "docker-compose.yaml"}
}

func (e *GenericExtractor) Extract(path string) (map[string]string, error) {
	tools := make(map[string]string)
	switch filepath.Base(path) {
	case "docker-compose.yml", "docker-compose.yaml":
		tools["docker"] = "latest"
		tools["docker-compose"] = "latest"
	}
	return tools, nil
}
