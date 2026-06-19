# Verification Report: Phase 1 — Foundation

## Summary

Phase 1 of the `monorepo-detection` change implements the WalkDir-based recursive scanner and the multi-subdirectory data model. The implementation is located in `internal/detect/detect.go` (modified) and `internal/detect/monorepo.go` (new), with comprehensive tests in `internal/detect/monorepo_test.go`.

**Verdict**: `PASS WITH WARNINGS`

The Phase 1 foundation is solid: all tests pass, build/vet are clean, the data model matches the design, and the recursive scanner fulfills the core Phase 1 spec scenarios. Two minor warnings remain: a small wording deviation from the single-root spec (the code always walks, but output is identical), and the untracked `coverage.out` file is not ignored by the project's `.gitignore`.

## Spec Compliance

### Recursive Manifest Detection

| Requirement / Scenario | Status | Evidence |
|---|---|---|
| Walk all subdirectories; skip `.git`, `node_modules`, `.dist`, `vendor`, `.venv`, `__pycache__`, `.turbo`, `build`, `.next`, `out`, `target` | ✅ PASS | `defaultSkipList` in `monorepo.go:12-24`; `TestCollectSubdirsSkipsForbidden`; `TestDetectSkipsNestedGitAndNodeModules` |
| Monorepo with frontend and backend | ✅ PASS | `TestDetectSubdirectoryManifests` detects `eslint` from `frontend/package.json` and `go` from `backend/go.mod` |
| Single package project (backward compatibility) | ⚠️ PARTIAL | `TestDetectBackwardCompatibility` confirms identical output, but implementation always walks even when no subdir manifests exist (spec says "no subdirectory walk occurs") |
| Skipped directories ignored | ✅ PASS | `TestDetectSkipsNestedGitAndNodeModules` |
| Recursive env and file scanning | ✅ PASS | `TestDetectSubdirectoryEnvFiles`, `TestDetectSubdirectoryProjectFiles` |
| Tool deduplication (same version) | ✅ PASS | `TestDetectDeduplicatesSameVersionAcrossSubdirs` |
| Version conflict detection and reporting | ⏭️ NOT IN PHASE 1 | Reserved for Phase 3 per tasks/design |

### Workspace-Aware Detection

| Requirement | Status | Evidence |
|---|---|---|
| `package.json` workspaces parsing | ⏭️ NOT IN PHASE 1 | Reserved for Phase 2 |
| `pnpm-workspace.yaml` parsing | ⏭️ NOT IN PHASE 1 | Reserved for Phase 2 |
| Workspace hint fallback | ⏭️ NOT IN PHASE 1 | Reserved for Phase 2 |

### Subdirectory Binary Resolution

| Requirement | Status | Evidence |
|---|---|---|
| Subdirectory binary search | ⏭️ NOT IN PHASE 1 | Reserved for Phase 4 |
| Root-first resolution | ⏭️ NOT IN PHASE 1 | Reserved for Phase 4 |
| Deterministic subdirectory search | ⏭️ NOT IN PHASE 1 | Reserved for Phase 4 |

## Test Results

### Command Evidence

```bash
$ go test ./...
?       env-doctor/cmd/env-doctor       [no test files]
ok      env-doctor/internal/checker     (cached)
ok      env-doctor/internal/config      (cached)
ok      env-doctor/internal/detect      (cached)
?       env-doctor/internal/ui          [no test files]
ok      env-doctor/pkg/version          (cached)
```

```bash
$ go vet ./...
# no output (success)
```

```bash
$ go build ./...
# no output (success)
```

```bash
$ gofmt -l internal/detect/detect.go internal/detect/monorepo.go internal/detect/monorepo_test.go internal/detect/detect_test.go
# no output (success)
```

### TDD Compliance

| Check | Result | Details |
|---|---|---|
| TDD Evidence reported | ✅ | Found in Engram `sdd/monorepo-detection/apply-progress` |
| All tasks have tests | ✅ | 6/6 Phase 1 tasks have test coverage |
| RED confirmed (tests exist) | ✅ | 9/9 new test functions exist in `monorepo_test.go` |
| GREEN confirmed (tests pass) | ✅ | All tests pass on `go test ./...` |
| Triangulation adequate | ✅ | Multiple cases per behavior (skip list, aggregation, env/files, dedup, backward compat) |
| Safety Net for modified files | ✅ | Apply report states 9/9 baseline tests passed before modifications |

### Test Layer Distribution

| Layer | Tests | Files | Tools |
|---|---|---|---|
| Unit | 9 | 1 | `testing` (stdlib) |
| Integration | 9 | 1 | `testing` (stdlib) |
| E2E | 0 | 0 | — |
| **Total** | **9 new** | **1** | |

*Note: The 9 new tests in `monorepo_test.go` mix unit-style (`TestCollectSubdirs*`) and integration-style (`TestDetect*`) assertions.*

### Changed File Coverage

| File | Line % | Branch % | Uncovered Lines | Rating |
|---|---|---|---|---|
| `internal/detect/detect.go` | 91.9% | n/a | L54-56 (collectSubdirs error), L68-69 (Glob error), L83 (non-primary manifest error continue), L110-111 (env dedup), L130-131 (file dedup), L162-164 (empty tools branch), L174-176 (group spacing), L191-193 (empty env), L201-203 / L209-226 (empty files / found-files branch), L274-296 (groupBySource fallbacks), L318 / L338-339 / L348 (extractor type fallbacks) | ✅ Excellent |
| `internal/detect/monorepo.go` | 84.2% | n/a | L32-35 (WalkDir error), L41-43 (Rel error), L55-57 (top-level error return) | ✅ Excellent |

**Average changed file coverage**: ~88%

Coverage is well above the 80% threshold. Uncovered blocks are defensive error handling and less-common output branches, which is acceptable for a foundation PR.

### Assertion Quality

**Assertion quality**: ✅ All assertions verify real behavior

No tautologies, ghost loops, type-only assertions, or smoke-test-only cases were found. Assertions check concrete tool versions, env variable names, file paths, and subdirectory tracking.

## Issues Found

### WARNING

1. **Spec wording deviation — single-root walk**
   - The spec states: "no subdirectory walk occurs if no manifests exist below root".
   - The implementation always calls `collectSubdirs` and walks, but `TestDetectBackwardCompatibility` proves the output is identical to root-only detection.
   - **Impact**: Functional behavior matches the backward-compatibility requirement; only the literal "no walk" guarantee is not met. This is not blocking for Phase 1.

2. **`coverage.out` not ignored**
   - Running `go test -coverprofile=coverage.out` produced an untracked `coverage.out` in the repo root. The existing `.gitignore` does not exclude coverage artifacts.
   - **Impact**: Low risk of accidental commit. Add `*.out` or `coverage.out` to `.gitignore`.

### SUGGESTION

3. **Add test for empty-tools / empty-env / empty-files branches in `Generate`**
   - `Generate` has uncovered branches for projects with no tools, env, or files. While unlikely in practice, a regression test would harden the output format.

4. **Add test for `collectSubdirs` error path**
   - The error branches in `collectSubdirs` (WalkDir error, `filepath.Rel` error, top-level error return) are not exercised by tests. These are defensive paths, but coverage could be improved with a permission-denied or broken-symlink fixture.

## Recommendations

1. Fix the `.gitignore` to exclude `coverage.out` (or `*.out`) before the next PR.
2. Consider adding one regression test for `Generate` with empty `Config.Tools` / `Config.Env` / `Config.Files` to lock the output format.
3. Proceed to **sdd-apply Phase 2** (Workspace-Aware Detection) once the warnings above are addressed or explicitly accepted.

## Next Step

**Recommended next phase**: `sdd-apply Phase 2 — Workspace-Aware Detection`

Phase 1 is functionally complete and verified. The foundation data model and recursive WalkDir scanner are ready to support workspace hints (`package.json` workspaces, `pnpm-workspace.yaml`) in the next PR.
