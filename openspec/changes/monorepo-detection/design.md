# Design: Monorepo Detection

## Context

Current `Detect()` uses `filepath.Glob` at root only — monorepo subdirectories (`frontend/`, `backend/`, `packages/*/`) are invisible. Extractors and scanners are already path-agnostic (they receive full file paths) but are only called with the root directory. This design adds subdirectory iteration, workspace parsing, and conflict detection while preserving backward compatibility for single-root projects.

## Data Model

Extend `Detected` with multi-source tracking (specs: recursive-manifest-detection §Tool Deduplication, §Version Conflict Detection):

```go
type Detected struct {
    Config       config.Config            // unchanged — flat map for YAML output
    ToolComments map[string]string        // unchanged
    ToolSources  map[string]string        // unchanged — flat map (first-found-wins source)
    FileComments map[string]string        // unchanged

    // New — multi-subdirectory tracking
    ToolSubdirs   map[string][]string       // tool → all subdirectories where found
    ToolConflicts map[string][]VersionEntry // tool → conflicting versions+sources
    EnvSubdirs    map[string][]string       // env_var → subdirectories where .env found
    FileSubdirs   map[string][]string       // file path → subdirectories where found
}

type VersionEntry struct {
    Version string // e.g. "8.x" or "1.21"
    Source  string // subdirectory path + manifest, e.g. "frontend/package.json"
}
```

`ToolSources` keeps first-found semantics for backward-compat YAML generation. `ToolSubdirs` and `ToolConflicts` add the multi-source tracking needed for conflict warnings.

## Architecture Decisions

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Full WalkDir always | Simple, ignores workspace hints | **Hybrid** — workspace hints as optimization, fallback walk for completeness |
| Inline all logic in detect.go | No new file, but grows detect.go | **Create `internal/detect/monorepo.go`** for walk/workspace/conflict |
| `Detect()` signature changes | Would break callers | **Keep `Detect(dir string) (Detected, error)`** — wire subdir logic internally |
| Parse YAML manually for pnpm-workspace.yaml | Avoid new dep | **Use `gopkg.in/yaml.v3`** (already transitive via viper) |
| Conflict warnings only in UI | Transient, lost on re-init | **Emit YAML comments AND populate `ToolConflicts`** for both persistence and programmatic access |

## Algorithm

```
Detect(dir)
  │
  ├─ parseWorkspaceHints(dir)
  │   ├─ package.json workspaces → glob expansion
  │   ├─ pnpm-workspace.yaml → packages field
  │   └─ → []string (priority subdirs, may be empty)
  │
  ├─ collectSubdirs(dir, skipList)
  │   └─ WalkDir with skip list → []string (all scan targets)
  │
  ├─ Order: priority subdirs first, then remaining
  │
  ├─ for each subdir:
  │   ├─ Run extractors → tools + source path
  │   ├─ Run env scanner → env vars + source path
  │   ├─ Run file scanner → files + source path
  │   └─ Merge into Detected
  │
  ├─ detectConflicts(d)
  │   ├─ Group tools by name across subdirs
  │   ├─ All same version → dedup (first-found wins, no conflict)
  │   ├─ Different versions → populate ToolConflicts
  │   └─ Select highest by major.minor parse for Config.Tools
  │
  └─ Return Detected
```

**Conflict version selection**: Parse major version from each wildcard (e.g. `8.x` → `8`, `9.x` → `9`, `1.21` → `1`). Highest numeric major wins. If majors are equal, compare minors. This matches the semver precedent of major version being the breaking-change boundary.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/detect/monorepo.go` | **Create** | `collectSubdirs()`, `parseWorkspaceHints()`, `detectConflicts()`, `highestVersion()` |
| `internal/detect/detect.go` | Modify | Wire `WalkDir` iteration + workspace parsing + conflict detection into `Detect()` |
| `internal/detect/scanners.go` | Unchanged | Already path-agnostic; called per subdirectory |
| `internal/detect/extractors.go` | Unchanged | Already path-agnostic; receives subdir file paths |
| `internal/checker/checker.go` | Modify | `resolveBinary()` walks subdir `node_modules/.bin/` and `.venv/bin/` before PATH fallback |
| `openspec/changes/monorepo-detection/design.md` | Create | This document |

## New Functions

```go
// monorepo.go
func parseWorkspaceHints(dir string) ([]string, error)
func collectSubdirs(dir string, skipList []string) ([]string, error)
func detectConflicts(d *Detected)
func highestVersion(versions []string) string
```

## Testing Strategy

Strict TDD — tests before code. Test scaffolding uses `t.TempDir()` with fixture files and `writeExecutable()` (existing pattern).

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `collectSubdirs` with skip list | Temp dir tree with `node_modules/`, `.git/` → verify skipped |
| Unit | `parseWorkspaceHints` — glob expansion | `package.json` workspaces: `["packages/*"]` with `packages/a` and `packages/b` |
| Unit | `parseWorkspaceHints` — malformed YAML | Invalid `pnpm-workspace.yaml` → returns empty, no error |
| Unit | `detectConflicts` — same version across subdirs | 3 subdirs all with `eslint 9.x` → no conflicts |
| Unit | `detectConflicts` — version conflict | `eslint 8.x` + `eslint 9.x` → conflict entry, highest=`9.x` |
| Unit | `detectConflicts` — three-way conflict | `eslint 7.x`+`8.x`+`9.x` → conflicts for all 3, highest=`9.x` |
| Unit | `resolveBinary` — root first | Root `node_modules/.bin/eslint` wins over subdir |
| Unit | `resolveBinary` — subdir fallback | No root → subdir `node_modules/.bin/eslint` returned |
| Unit | `resolveBinary` — PATH fallback | No subdir binary → returns name (system PATH) |
| Integration | `Detect` on monorepo fixture | `frontend/package.json` + `backend/go.mod` → both tools detected |
| Integration | `Detect` on single-root fixture | Same input as existing tests → identical output (regression) |

## Performance

- **Skip list** prevents descent into `node_modules/`, `.git`, `.venv`, `vendor`, `build`, `.next`, `__pycache__`, `.turbo`, `out`, `target`
- **Workspace hints** reduce walk scope in npm/pnpm monorepos (walk only workspace dirs + remaining)
- No caching needed — CLI tool does one-shot detection
- WalkDir with skip list is typically <50ms on moderate repos; worst-case is flat project with many subdirs (still sub-second)

## Edge Cases

| Case | Behavior |
|------|----------|
| Malformed `pnpm-workspace.yaml` | Log warning, fall back to full WalkDir |
| Circular workspace refs | Glob-to-path expansion doesn't produce cycles; WalkDir handles symlink loops natively |
| Empty subdirectory | No manifests/env/files — skipped gracefully |
| Same tool + version in N subdirs | Deduplicated; first-found source annotated |
| Same tool, different versions | Conflict emitted; highest version stored in `Config.Tools` |
| Binary in multiple subdirs | Root-first then deterministic (alphabetical) order |
| Non-monorepo project | `collectSubdirs` returns only root → identical output |
| Both `package.json` workspaces AND `pnpm-workspace.yaml` | Prefer the one matching lockfile; union of both paths as priority |

## Dependencies

- `gopkg.in/yaml.v3` — already transitive via `github.com/spf13/viper`; add direct import in `go.mod`
- No other new dependencies. Uses stdlib `io/fs`, `path/filepath`, `encoding/json`.
