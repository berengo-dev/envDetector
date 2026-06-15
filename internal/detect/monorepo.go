package detect

import (
	"io/fs"
	"path/filepath"
	"slices"
	"sort"
)

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
