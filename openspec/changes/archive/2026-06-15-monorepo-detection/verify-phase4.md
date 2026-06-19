# Verification Report: Phase 4 — Subdirectory Binary Resolution

## Summary

| Item | Value |
|------|-------|
| Change | `monorepo-detection` |
| Phase | PR 4 — Subdirectory Binary Resolution |
| Mode | openspec |
| Strict TDD | active |
| Test runner | `go test ./...` |
| Final verdict | **PASS WITH WARNINGS** |

Phase 4 implementation is functionally complete and matches the subdirectory binary resolution spec and design. All new tests pass, build and static analysis succeed, and existing tests remain green. The only blocking concern under Strict TDD is the absence of an `apply-progress` artifact documenting the RED/GREEN/REFACTOR cycle evidence.

---

## Spec Compliance

### Requirement: Subdirectory Binary Search

| Scenario | Spec Statement | Evidence | Status |
|----------|---------------|----------|--------|
| Binary in frontend subdirectory `node_modules` | `frontend/node_modules/.bin/eslint` resolved and version check executed | `TestResolveBinarySubdirFallback` creates `frontend/node_modules/.bin/eslint` and asserts the exact path is returned | ✅ PASS |
| Binary in multiple subdirectories | First matching binary encountered in deterministic order is returned | `TestResolveBinaryDeterministicOrder` creates `a/.../eslint` and `b/.../eslint` and asserts `a/.../eslint` is returned after `sort.Strings` | ✅ PASS |
| No subdirectory binary found | Falls back to system PATH; original fallback preserved | `TestResolveBinaryPATHFallback` asserts return value `"eslint"` | ✅ PASS |

### Requirement: Backward Compatible Root-First Resolution

| Scenario | Spec Statement | Evidence | Status |
|----------|---------------|----------|--------|
| Root binary found | Root `node_modules/.bin/eslint` returned immediately; no subdir search | `TestResolveBinaryRootFirst` creates both root and subdir binaries and asserts root path is returned | ✅ PASS |

### Requirement: Backward Compatible Root-First Resolution (`.venv`)

The spec does not explicitly split `.venv` into its own requirement, but the subdirectory search requirement covers `{workDir}/**/.venv/bin/{tool}`. Tests verify root `.venv` precedence as well.

| Scenario | Evidence | Status |
|----------|----------|--------|
| Root `.venv/bin/python` wins over subdir `.venv/bin/python` | `TestResolveBinaryVenvRootFirst` | ✅ PASS |
| Subdir `.venv/bin/python` returned when no root binary | `TestResolveBinaryVenvSubdirFallback` | ✅ PASS |

### Deterministic Ordering Note

The prompt asks whether the resolver finds "the first matching binary (closest to root)." The spec and design require **deterministic** resolution, not proximity-to-root resolution. The implementation collects all matches and returns the lexicographically first path (`sort.Strings(matches)[0]`). This is the behavior documented in the design ("deterministic (alphabetical) order") and satisfies the spec scenario for consistent resolution across runs. No issue is raised.

---

## Test Results

### Full Test Suite

```text
$ go test ./...
?       env-doctor/cmd/env-doctor    [no test files]
ok      env-doctor/internal/checker  0.020s
ok      env-doctor/internal/config   (cached)
ok      env-doctor/internal/detect   (cached)
?       env-doctor/internal/ui       [no test files]
ok      env-doctor/pkg/version       (cached)
```

### Targeted Binary Resolution Tests

```text
$ go test -v ./internal/checker/... -run TestResolveBinary
=== RUN   TestResolveBinaryRootFirst
--- PASS: TestResolveBinaryRootFirst (0.00s)
=== RUN   TestResolveBinarySubdirFallback
--- PASS: TestResolveBinarySubdirFallback (0.00s)
=== RUN   TestResolveBinaryPATHFallback
--- PASS: TestResolveBinaryPATHFallback (0.00s)
=== RUN   TestResolveBinaryDeterministicOrder
--- PASS: TestResolveBinaryDeterministicOrder (0.00s)
=== RUN   TestResolveBinaryVenvRootFirst
--- PASS: TestResolveBinaryVenvRootFirst (0.00s)
=== RUN   TestResolveBinaryVenvSubdirFallback
--- PASS: TestResolveBinaryVenvSubdirFallback (0.00s)
PASS
ok      env-doctor/internal/checker  0.020s
```

### Coverage

```text
$ go test -cover ./...
ok  env-doctor/internal/checker  0.021s  coverage: 85.2% of statements
```

Per-function coverage for changed logic:

| File | Function | Coverage |
|------|----------|----------|
| `internal/checker/checker.go` | `resolveBinary` | 96.2% |
| `internal/checker/checker.go` | package total | 85.2% |

The only uncovered branch in `resolveBinary` is the `if err != nil` guard inside `filepath.WalkDir` (line 179), which is defensive error handling. No functional gap.

### Static Analysis & Build

| Check | Command | Result |
|-------|---------|--------|
| Static analysis | `go vet ./...` | ✅ No issues |
| Build | `go build ./...` | ✅ Success |
| Formatting | `gofmt -l internal/checker/checker.go internal/checker/binary_test.go` | ✅ No issues |

### Quality Tools Available

- `go vet`: available and clean
- `go build`: available and clean
- `gofmt`: available and clean
- `golangci-lint` / `staticcheck`: not installed in this environment

---

## Issues Found

### CRITICAL

| # | Issue | Why | Suggested Fix |
|---|-------|-----|---------------|
| 1 | **Missing TDD Cycle Evidence artifact** | Strict TDD Mode is active, but no `apply-progress` artifact with a RED/GREEN/REFACTOR table was found in the change directory. The Strict TDD verify module requires this table to confirm the code was actually built test-first. | The apply phase should produce `openspec/changes/monorepo-detection/apply-progress.md` (or equivalent) documenting RED (test written), GREEN (test passed), TRIANGULATE, SAFETY NET, and REFACTOR for each Phase 4 task. If this artifact exists elsewhere, update the convention or link it from the change directory. |

### WARNING

| # | Issue | Why | Suggested Fix |
|---|-------|-----|---------------|
| 1 | None | — | — |

### SUGGESTION

| # | Issue | Why | Suggested Fix |
|---|-------|-----|---------------|
| 1 | None | — | — |

---

## Design Coherence

| Design Decision | Implementation | Status |
|-----------------|----------------|--------|
| `resolveBinary()` walks `{workDir}/**/node_modules/.bin/` and `{workDir}/**/.venv/bin/` before PATH fallback | `resolveBinary` iterates root candidates first, then `filepath.WalkDir` looks for `node_modules` and `.venv` directories and checks `.bin/{name}` / `bin/{name}` respectively, finally falling back to `name` | ✅ Compliant |
| Root-first check preserved | Root `node_modules/.bin/{name}` and `.venv/bin/{name}` are checked before `WalkDir` begins | ✅ Compliant |
| Deterministic (alphabetical) order | All matches collected, `sort.Strings(matches)`, return `matches[0]` | ✅ Compliant |
| Skip descent into matched `node_modules` / `.venv` | Returns `filepath.SkipDir` after checking the immediate binary subdir | ✅ Compliant |

---

## Task Completion

| Task | Status |
|------|--------|
| 4.1 Extend `Checker.resolveBinary()` — walk `{workDir}/**/node_modules/.bin/` and `{workDir}/**/.venv/bin/` before PATH fallback | ✅ Complete |
| 4.2 Write tests: root binary found (early return), subdir binary found, no binary → PATH fallback | ✅ Complete |
| 4.3 Ensure deterministic ordering via sorted WalkDir traversal | ✅ Complete |

---

## Recommendations

1. **Create the missing TDD Cycle Evidence artifact** before archiving. Under Strict TDD, the verification phase must be able to cross-reference apply-phase RED/GREEN evidence with actual test execution. Without it, future audits cannot confirm test-first development was followed.
2. **Consider adding a test for deeply nested `.venv` / `node_modules`** (e.g., `packages/a/node_modules/.bin/eslint`) to strengthen the deterministic-order claim across varying depths. Not required for spec compliance.
3. **Keep the current `filepath.SkipDir` behavior**; it prevents expensive descent into large dependency directories and is consistent with the design's skip-list philosophy.

---

## Next Step

Proceed to **`sdd-archive`** once the missing TDD Cycle Evidence artifact is supplied or the Strict TDD deviation is explicitly accepted by the orchestrator. Functionally, the implementation is ready for archive.
