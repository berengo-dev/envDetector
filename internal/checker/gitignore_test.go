package checker

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitignoreIgnoresCompiledBinary(t *testing.T) {
	// Locate repo root from the test file location (internal/checker/).
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	f, err := os.Open(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatalf("open .gitignore: %v", err)
	}
	defer f.Close()

	found := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "env-doctor" {
			found = true
			break
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}

	if !found {
		t.Errorf(".gitignore does not contain an entry for the compiled env-doctor binary")
	}
}
