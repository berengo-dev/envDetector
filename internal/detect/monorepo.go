package detect

import (
	"io/fs"
	"path/filepath"
	"slices"
	"sort"
)

// buildScanTargets returns the ordered list of directories to scan for a
// project rooted at dir. The returned slice always starts with dir itself,
// followed by workspace-hinted directories (from package.json and
// pnpm-workspace.yaml), and finally any remaining subdirectories discovered
// by a recursive WalkDir. The skipList is applied to all sources.
func buildScanTargets(dir string, skipList []string) ([]string, error) {
	hints, err := parseWorkspaceHints(dir)
	if err != nil {
		return nil, err
	}

	allSubdirs, err := collectSubdirs(dir, skipList)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	scanTargets := []string{dir}
	seen[dir] = struct{}{}

	// Add workspace hints first, filtering through the skip list.
	for _, h := range hints {
		if _, ok := seen[h]; ok {
			continue
		}
		if slices.Contains(skipList, filepath.Base(h)) {
			continue
		}
		scanTargets = append(scanTargets, h)
		seen[h] = struct{}{}
	}

	// Append remaining subdirectories discovered by WalkDir.
	for _, s := range allSubdirs {
		if _, ok := seen[s]; ok {
			continue
		}
		scanTargets = append(scanTargets, s)
		seen[s] = struct{}{}
	}

	return scanTargets, nil
}

// defaultSkipList contains directory names that should never be descended into
// during recursive project scanning.
var defaultSkipList = []string{
	".git",
	"node_modules",
	".dist",
	"vendor",
	".venv",
	"__pycache__",
	".turbo",
	"build",
	".next",
	"out",
	"target",
}

// collectSubdirs returns all directories under dir (excluding dir itself),
// skipping any directory whose base name appears in skipList.
func collectSubdirs(dir string, skipList []string) ([]string, error) {
	var subdirs []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Ignore permission or symlink errors and keep walking.
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		if rel == "." {
			return nil
		}

		if slices.Contains(skipList, filepath.Base(path)) {
			return fs.SkipDir
		}

		subdirs = append(subdirs, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(subdirs)
	return subdirs, nil
}
