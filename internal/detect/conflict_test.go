package detect

import (
	"path/filepath"
	"testing"

	"env-doctor/internal/config"
)

func TestHighestVersionPrefersHigherMajor(t *testing.T) {
	cases := []struct {
		versions []string
		want     string
	}{
		{[]string{"8.x", "9.x"}, "9.x"},
		{[]string{"9.x", "8.x"}, "9.x"},
		{[]string{"1.21", "1.22"}, "1.22"},
		{[]string{"18.x", "20.x", "19.x"}, "20.x"},
	}

	for _, c := range cases {
		got := highestVersion(c.versions)
		if got != c.want {
			t.Errorf("highestVersion(%v) = %q, want %q", c.versions, got, c.want)
		}
	}
}

func TestHighestVersionMajorEqualComparesMinor(t *testing.T) {
	cases := []struct {
		versions []string
		want     string
	}{
		{[]string{"1.20", "1.21"}, "1.21"},
		{[]string{"1.21.0", "1.21.5"}, "1.21.5"},
		{[]string{"9.0", "9.1"}, "9.1"},
	}

	for _, c := range cases {
		got := highestVersion(c.versions)
		if got != c.want {
			t.Errorf("highestVersion(%v) = %q, want %q", c.versions, got, c.want)
		}
	}
}

func TestHighestVersionHandlesExactSemver(t *testing.T) {
	got := highestVersion([]string{"^8.0.0", "^9.0.0", "~7.1.0"})
	want := "^9.0.0"
	if got != want {
		t.Errorf("highestVersion semver) = %q, want %q", got, want)
	}
}

func TestHighestVersionUnparsedFallback(t *testing.T) {
	cases := []struct {
		versions []string
		want     string
	}{
		{[]string{"latest", "9.x"}, "9.x"},
		{[]string{"alpha", "beta"}, "beta"},
		{[]string{"*", "8.x"}, "8.x"},
	}

	for _, c := range cases {
		got := highestVersion(c.versions)
		if got != c.want {
			t.Errorf("highestVersion(%v) = %q, want %q", c.versions, got, c.want)
		}
	}
}

func TestDetectConflictsNoConflictWhenSameVersion(t *testing.T) {
	d := Detected{
		Config: config.Config{Tools: map[string]string{"eslint": "8.x"}},
		ToolConflicts: map[string][]VersionEntry{
			"eslint": {
				{Version: "8.x", Source: "frontend/package.json"},
				{Version: "8.x", Source: "backend/package.json"},
			},
		},
	}

	detectConflicts(&d)

	if len(d.ToolConflicts["eslint"]) != 0 {
		t.Errorf("expected no conflict for same version, got %v", d.ToolConflicts["eslint"])
	}
	if d.Config.Tools["eslint"] != "8.x" {
		t.Errorf("Tools[eslint] = %q, want 8.x", d.Config.Tools["eslint"])
	}
}

func TestDetectConflictsTwoWayConflict(t *testing.T) {
	d := Detected{
		Config: config.Config{Tools: map[string]string{"eslint": "8.x"}},
		ToolConflicts: map[string][]VersionEntry{
			"eslint": {
				{Version: "8.x", Source: "frontend/package.json"},
				{Version: "9.x", Source: "backend/package.json"},
			},
		},
	}

	detectConflicts(&d)

	if len(d.ToolConflicts["eslint"]) != 2 {
		t.Errorf("expected 2 conflict entries, got %v", d.ToolConflicts["eslint"])
	}
	if d.Config.Tools["eslint"] != "9.x" {
		t.Errorf("Tools[eslint] = %q, want 9.x", d.Config.Tools["eslint"])
	}
}

func TestDetectConflictsThreeWayConflict(t *testing.T) {
	d := Detected{
		Config: config.Config{Tools: map[string]string{
			"eslint": "7.x",
		}},
		ToolConflicts: map[string][]VersionEntry{
			"eslint": {
				{Version: "8.x", Source: "frontend/package.json"},
				{Version: "9.x", Source: "backend/package.json"},
				{Version: "7.x", Source: "shared/package.json"},
			},
		},
	}

	detectConflicts(&d)

	if len(d.ToolConflicts["eslint"]) != 3 {
		t.Errorf("expected 3 conflict entries, got %v", d.ToolConflicts["eslint"])
	}
	if d.Config.Tools["eslint"] != "9.x" {
		t.Errorf("Tools[eslint] = %q, want 9.x", d.Config.Tools["eslint"])
	}
}

func TestGenerateEmitsConflictWarningComments(t *testing.T) {
	d := Detected{
		Config: config.Config{
			Version: "1",
			Tools:   map[string]string{"eslint": "9.x"},
		},
		ToolSources:  map[string]string{"eslint": "Extracted from manifest file"},
		ToolComments: map[string]string{"eslint": "From dependency manifest"},
		ToolConflicts: map[string][]VersionEntry{
			"eslint": {
				{Version: "8.x", Source: "frontend/package.json"},
				{Version: "9.x", Source: "backend/package.json"},
			},
		},
	}

	yaml, err := Generate(d)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !contains(yaml, "WARNING: Version conflicts detected") {
		t.Errorf("generated YAML missing conflict warning header")
	}
	if !contains(yaml, "frontend: 8.x") {
		t.Errorf("generated YAML missing frontend conflict version")
	}
	if !contains(yaml, "backend: 9.x") {
		t.Errorf("generated YAML missing backend conflict version")
	}
	if !contains(yaml, "Selected: 9.x") {
		t.Errorf("generated YAML missing selected highest version")
	}
	if !contains(yaml, `"eslint": "9.x"`) {
		t.Errorf("generated YAML missing resolved eslint version")
	}
}

func TestDetectConflictsIntegration(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "frontend"))
	mustMkdir(t, filepath.Join(dir, "backend"))

	writeFile(t, filepath.Join(dir, "frontend"), "package.json", `{
		"devDependencies": { "eslint": "^8.0.0" }
	}`)
	mustMkdir(t, filepath.Join(dir, "frontend", "node_modules", ".bin"))
	writeExecutable(t, filepath.Join(dir, "frontend", "node_modules", ".bin"), "eslint")

	writeFile(t, filepath.Join(dir, "backend"), "package.json", `{
		"devDependencies": { "eslint": "^9.0.0" }
	}`)
	mustMkdir(t, filepath.Join(dir, "backend", "node_modules", ".bin"))
	writeExecutable(t, filepath.Join(dir, "backend", "node_modules", ".bin"), "eslint")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if d.Config.Tools["eslint"] != "9.x" {
		t.Errorf("Tools[eslint] = %q, want 9.x", d.Config.Tools["eslint"])
	}
	if len(d.ToolConflicts["eslint"]) != 2 {
		t.Errorf("expected 2 conflict entries, got %v", d.ToolConflicts["eslint"])
	}

	yaml, err := Generate(d)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !contains(yaml, "WARNING: Version conflicts detected") {
		t.Errorf("generated YAML missing conflict warning")
	}
}

func TestDetectNoConflictWhenVersionsMatch(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "frontend"))
	mustMkdir(t, filepath.Join(dir, "backend"))

	writeFile(t, filepath.Join(dir, "frontend"), "package.json", `{
		"devDependencies": { "eslint": "^8.0.0" }
	}`)
	mustMkdir(t, filepath.Join(dir, "frontend", "node_modules", ".bin"))
	writeExecutable(t, filepath.Join(dir, "frontend", "node_modules", ".bin"), "eslint")

	writeFile(t, filepath.Join(dir, "backend"), "package.json", `{
		"devDependencies": { "eslint": "^8.0.0" }
	}`)
	mustMkdir(t, filepath.Join(dir, "backend", "node_modules", ".bin"))
	writeExecutable(t, filepath.Join(dir, "backend", "node_modules", ".bin"), "eslint")

	d, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if len(d.ToolConflicts["eslint"]) != 0 {
		t.Errorf("expected no conflict for matching versions, got %v", d.ToolConflicts["eslint"])
	}
	if d.Config.Tools["eslint"] != "8.x" {
		t.Errorf("Tools[eslint] = %q, want 8.x", d.Config.Tools["eslint"])
	}
}
