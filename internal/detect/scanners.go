package detect

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joho/godotenv"
)

// BinaryScanner finds executables in local directories.
type BinaryScanner interface {
	// Scan looks for binaries in common local directories under dir.
	Scan(dir string) []string
}

// DefaultBinaryScanner is the default BinaryScanner implementation.
type DefaultBinaryScanner struct{}

// Scan returns the names of binaries found in local directories such as
// node_modules/.bin, virtual environments, and any */bin/ directory.
func (s *DefaultBinaryScanner) Scan(dir string) []string {
	seen := make(map[string]struct{})

	paths := []string{
		filepath.Join(dir, "node_modules", ".bin"),
		filepath.Join(dir, ".venv", "bin"),
		filepath.Join(dir, "venv", "bin"),
	}

	// Add any */bin/ directory at the project root.
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				binPath := filepath.Join(dir, entry.Name(), "bin")
				if info, err := os.Stat(binPath); err == nil && info.IsDir() {
					paths = append(paths, binPath)
				}
			}
		}
	}

	for _, p := range paths {
		entries, err := os.ReadDir(p)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			// Skip shell wrapper scripts on Windows and hidden files.
			if strings.HasPrefix(name, ".") {
				continue
			}
			// On Unix, require an executable bit. On Windows, os.ReadDir cannot
			// report mode bits reliably, so accept the file as-is.
			if info, err := entry.Info(); err == nil {
				if info.Mode()&0o111 == 0 && info.Mode()&os.ModeSymlink == 0 {
					continue
				}
			}
			seen[name] = struct{}{}
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// EnvScanner finds environment variables from .env files.
type EnvScanner interface {
	// Scan returns environment variable names found in .env.example or .env.
	Scan(dir string) []string
}

// DefaultEnvScanner is the default EnvScanner implementation.
type DefaultEnvScanner struct{}

// Scan reads .env.example first, then falls back to .env, and returns the
// sorted list of variable names.
func (s *DefaultEnvScanner) Scan(dir string) []string {
	for _, name := range []string{".env.example", ".env"} {
		path := filepath.Join(dir, name)
		vars, err := godotenv.Read(path)
		if err != nil {
			continue
		}
		names := make([]string, 0, len(vars))
		for n := range vars {
			names = append(names, n)
		}
		sort.Strings(names)
		return names
	}
	return nil
}

// FileScanner finds common project files.
type FileScanner interface {
	// Scan returns a curated list of project files present under dir.
	Scan(dir string) []string
}

// DefaultFileScanner is the default FileScanner implementation.
type DefaultFileScanner struct{}

// Scan returns common project files and recognizable config files present in
// the given directory.
func (s *DefaultFileScanner) Scan(dir string) []string {
	seen := make(map[string]struct{})

	common := []string{
		".env",
		".env.example",
		".gitignore",
		"README.md",
		"README",
		"LICENSE",
		"Makefile",
		"Dockerfile",
		"docker-compose.yml",
		"docker-compose.yaml",
		"package.json",
		"go.mod",
		"main.go",
		"requirements.txt",
		"pyproject.toml",
	}
	for _, name := range common {
		if fileExists(filepath.Join(dir, name)) {
			seen[name] = struct{}{}
		}
	}

	// Recognizable config/config-like files.
	patterns := []string{
		"*.config.*",
		"tsconfig*.json",
		"jsconfig*.json",
		"*.lock",
	}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			continue
		}
		for _, m := range matches {
			base := filepath.Base(m)
			if base == "package-lock.json" || base == "yarn.lock" || base == "pnpm-lock.yaml" {
				// Treat lock files as project files.
				seen[base] = struct{}{}
				continue
			}
			seen[base] = struct{}{}
		}
	}

	files := make([]string, 0, len(seen))
	for f := range seen {
		files = append(files, f)
	}
	sort.Strings(files)
	return files
}

// DefaultScanners returns the default scanner implementations.
func DefaultScanners() (BinaryScanner, EnvScanner, FileScanner) {
	return &DefaultBinaryScanner{}, &DefaultEnvScanner{}, &DefaultFileScanner{}
}
