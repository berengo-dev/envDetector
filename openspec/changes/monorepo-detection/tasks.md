# Tasks: Monorepo Detection

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~470 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1: WalkDir → PR 2: Workspace Hints → PR 3: Conflicts → PR 4: Binary Resolution |
| Delivery strategy | ask-always |
| Chain strategy | pending |

```text
Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High
```

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | WalkDir + data model + subdir iteration for env/files | PR 1 | Base; ~150 lines; tests included |
| 2 | Workspace hint parsing (npm/pnpm) | PR 2 | Depends on PR 1; ~120 lines |
| 3 | Conflict detection + YAML warnings | PR 3 | Depends on PR 2; ~100 lines |
| 4 | Checker subdir binary resolution | PR 4 | Independent of PR 1-3; ~80 lines |

## Phase 1: Foundation — Data Model + WalkDir

- [x] 1.1 Add `Detected.ToolSubdirs`, `ToolConflicts`, `VersionEntry`, `EnvSubdirs`, `FileSubdirs` to `internal/detect/detect.go`
- [x] 1.2 Create `collectSubdirs(dir, skipList)` in `internal/detect/monorepo.go` with WalkDir + skip list (`.git`, `node_modules`, `.venv`, `vendor`, `build`, `.next`, `__pycache__`, `.turbo`, `out`, `target`, `.dist`)
- [x] 1.3 Write `collectSubdirs` tests: skips `node_modules/`, `.git/`, single-root returns only root
- [x] 1.4 Modify `Detect()` to iterate subdirectories using `collectSubdirs` for env + file scanning (tools extraction still root-only in this PR)
- [x] 1.5 Add subdirectory `.env` / `.env.example` detection in env scanner; annotate source with subdir path
- [x] 1.6 Write integration tests: monorepo fixture detects env/files from subdirs; single-root regression

## Phase 2: Workspace-Aware Detection

- [x] 2.1 Create `parseWorkspaceHints(dir)` — parse `package.json` workspaces (array + object) and `pnpm-workspace.yaml`
- [x] 2.2 Write tests: glob expansion, malformed YAML → empty fallback, both configs union
- [x] 2.3 Modify `Detect()`: use workspace hints as priority subdirs, fallback WalkDir for remaining
- [x] 2.4 Write integration test: stale workspace config still discovers paths outside it

## Phase 3: Version Conflict Detection

- [x] 3.1 Create `detectConflicts(d *Detected)` + `highestVersion(versions)` in `internal/detect/monorepo.go`
- [x] 3.2 Write tests: same version no conflict, two-way conflict, three-way conflict, highest version selection
- [x] 3.3 Wire `detectConflicts` into `Detect()` after all subdirs processed
- [x] 3.4 Modify `Generate()` to emit YAML warning comments for tools in `ToolConflicts`
- [x] 3.5 Write integration test: conflict detection with real manifests

## Phase 4: Subdirectory Binary Resolution

- [ ] 4.1 Extend `Checker.resolveBinary()` in `internal/checker/checker.go` — walk `{workDir}/**/node_modules/.bin/` and `{workDir}/**/.venv/bin/` before PATH fallback
- [ ] 4.2 Write tests: root binary found (early return), subdir binary found, no binary → PATH fallback
- [ ] 4.3 Ensure deterministic ordering via sorted WalkDir traversal
