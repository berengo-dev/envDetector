# Design: Fase 0 Bugfixes

## Context

Six bugs block a reliable public release of env-doctor. Bugs 0.1–0.4 silently produce wrong check results (file paths use CWD, semver `<` operator is invisible, version extractor grabs "2024" not "1.2.3", binary resolver misses `venv/bin/` and non-standard `*/bin/` dirs). Bug 0.5 breaks Go process hygiene with `os.Exit` inside Cobra `RunE`. Bug 0.6 leaves a compiled binary tracked by git.

## Goals / Non-Goals

**Goals**: Fix all 6 bugs with TDD red→green per bug. No behavioral changes outside the listed bugs. All existing tests must continue to pass.

**Non-Goals**: No new features, no UX overhaul, no refactoring beyond what each bug demands.

## Architecture Decisions

### 0.1 — `checkFile` ignores `workingDir`

**Approach**: `checkFile` currently calls `os.Stat(path)` with the raw path argument. Change to `os.Stat(filepath.Join(c.workingDir, path))`. This mirrors how `loadEnvFile` already resolves `.env` relative to `workingDir`.

```go
// Before
_, err := os.Stat(path)
// After
_, err := os.Stat(filepath.Join(c.workingDir, path))
```

**Test strategy** (red first): `TestCheckFileRelativeToConfigDir` — create `NewWithDir(tmpDir)`, register a file path relative to tmpDir, assert `StatusPass`. Before fix, `os.Stat` resolves against process CWD and fails.

**Files**: `internal/checker/checker.go` (modify line 246), `internal/checker/checker_test.go` (add test case).

**Risk**: Low. No other caller depends on `checkFile` resolving against CWD.

### 0.2 — Semver constraint regex missing `<`

**Approach**: The alternation in `semverConstraintRegex` has `<=` duplicated and `<` missing. Fix: `(\^|~|>=|<=|>|<|=)`.

```go
// Before: duplicated <=, no <
var semverConstraintRegex = regexp.MustCompile(`^(?:(\^|~|>=|<=|>|<=|=)\s*)?...`)
// After: each operator once, < included
var semverConstraintRegex = regexp.MustCompile(`^(?:(\^|~|>=|<=|>|<|=)\s*)?...`)
```

**Test strategy** (red first): `TestConvertSemverToWildcardLessThan` — `ConvertSemverToWildcard("<1.0.0")` returns `("1.x", true)`. Before fix, `<` is not matched and returns `("", false)`. Also verify `("<=2.0.0")` still produces `("2.x", true)` — regression guard for the dedup fix.

**Files**: `pkg/version/version.go` (fix regex line 27), `pkg/version/version_test.go` (add 2 test cases).

**Risk**: Low. Regex change only affects the alternation order; existing operators are unchanged.

### 0.3 — `versionRegex` grabs first number, not version

**Approach**: Replace `FindString` (first match) with `FindAllString` (all matches) and return the **last** match that contains at least one dot (`\d+\.\d+`). The regex stays the same; the logic changes.

The spec requires "last match with ≥1 dot" — this directly implements it. No keyword-aware heuristic needed; the copyright-year case `"Copyright 2024\nVersion 1.2.3"` works because `2024` has no dot and `1.2.3` does. The last dotted match wins in `"0.1 (dev) 1.0.0"` because `FindAllString` returns `["0.1", "1.0.0"]` and we pick the last.

```go
func Extract(raw string) string {
    all := versionRegex.FindAllString(raw, -1)
    var last string
    for _, m := range all {
        if strings.Contains(m, ".") {
            last = m
        }
    }
    return strings.TrimPrefix(last, "v")
}
```

**Test strategy** (red first): `TestExtractSkipsBareNumber` (input `"Copyright 2024\nVersion 1.2.3"` → `"1.2.3"`), `TestExtractBareNumberReturnsEmpty` (input `"Version 1"` → `""`), `TestExtractLastDottedWins` (input `"0.1 (dev) 1.0.0"` → `"1.0.0"`). Existing `TestExtract` cases must remain green.

**Files**: `pkg/version/version.go` (modify `Extract` function), `pkg/version/version_test.go` (add 3 cases, keep existing).

**Risk**: Low. Return type unchanged (`string`). Pure function.

### 0.4 — `resolveBinary` hardcoded to Node/.venv only

**Approach**: (a) Add `venv/bin/{name}` to `rootCandidates`. (b) Extend WalkDir to match any directory named `bin` (not just inside `.venv`), checking `*/bin/{name}`. Keep `node_modules` handler for `.bin` subdirectory skipping; no longer SkipDir on `.venv` since `bin` handler will catch its contents.

```go
// rootCandidates becomes:
rootCandidates := []string{
    filepath.Join(c.workingDir, "node_modules", ".bin", name),
    filepath.Join(c.workingDir, ".venv", "bin", name),
    filepath.Join(c.workingDir, "venv", "bin", name),  // NEW
}

// WalkDir: add case "bin" to catch any */bin/ directory
switch filepath.Base(path) {
case "node_modules":
    p := filepath.Join(path, ".bin", name)
    if _, err := os.Stat(p); err == nil {
        matches = append(matches, p)
    }
    return filepath.SkipDir
case "bin":
    p := filepath.Join(path, name)
    if _, err := os.Stat(p); err == nil {
        matches = append(matches, p)
    }
    // Don't SkipDir — deeper nesting may exist
}
```

**Test strategy** (red first): `TestResolveBinaryVenvNoDot` (`venv/bin/python` at root), `TestResolveBinaryNonStandardBinDir` (e.g. `tools/bin/go`). Both fail before the fix because `venv` (no dot) is not in rootCandidates and the WalkDir doesn't match general `bin` directories.

**Files**: `internal/checker/checker.go` (modify lines 167–201), `internal/checker/binary_test.go` (add 2 test cases).

**Risk**: Medium. WalkDir behavior changes — must ensure no regression for existing `.venv/bin` and `node_modules/.bin` resolution. Mitigated by keeping `node_modules` handler and relying on `bin` handler to catch `.venv/bin` (previously handled by explicit `.venv` case).

### 0.5 — `os.Exit(1)` inside Cobra `RunE`

**Approach**: Replace `os.Exit(1)` with `return ErrChecksFailed`. Add sentinel error in `checker` package. In `main()`, use `errors.Is` to distinguish check failures from other errors.

```go
// checker/checker.go — new sentinel
var ErrChecksFailed = errors.New("checker: one or more checks failed")

// cmd/env-doctor/main.go — RunE returns error instead of os.Exit
RunE: func(cmd *cobra.Command, args []string) error {
    // ...
    for _, r := range results {
        if r.Status == checker.StatusFail {
            return checker.ErrChecksFailed
        }
    }
    return nil
},

// cmd/env-doctor/main.go — main handles the sentinel
func main() {
    if err := rootCmd.Execute(); err != nil {
        if !errors.Is(err, checker.ErrChecksFailed) {
            fmt.Fprintln(os.Stderr, "Error:", err)
        }
        os.Exit(1)
    }
}
```

**SilenceUsage**: Yes — Cobra should not print usage on check failure (the user ran the command correctly).

**SilenceErrors**: Yes — Cobra should not print the error; `main()` controls what reaches stderr. Check failures are already rendered by `ui.Render`, so duplicate stderr is noise. Non-check errors (config load, flags) still print via `main()`.

**Test strategy** (red first): Integration-level test: create `Checker` with a known-failing config, call via RunE capture, assert `errors.Is(err, ErrChecksFailed)`. Before fix, `os.Exit(1)` terminates the test process.

**Files**: `internal/checker/checker.go` (add `ErrChecksFailed`), `cmd/env-doctor/main.go` (modify `checkCmd.RunE` and `main()`). No test file for `main.go` — test via `checker` package with mock runner.

**Risk**: Low. Exit code is still 1. The only behavioral change: deferred functions now run before exit (because `os.Exit` is gone from RunE).

### 0.6 — Compiled binary tracked by git

**Approach**: `git rm --cached env-doctor` removes the binary from the index. `.gitignore` already excludes it (line 2). No Go code changes.

**Test strategy** (red→green): Before fix, `git ls-files --error-unmatch env-doctor` exits 0 (file tracked). After `git rm --cached`, same command exits with error (file no longer tracked).

**Files**: `.gitignore` (already correct — verify), git index (remove `env-doctor`).

**Risk**: Low. `--cached` preserves the local binary. The binary will be excluded from future commits.

## Cross-Cutting Concerns

**Recommended order**: 0.6 → 0.2 → 0.3 → 0.1 → 0.5 → 0.4

| Step | Bug | Rationale |
|------|-----|-----------|
| 1 | 0.6 | Zero code changes, cleans repo hygiene immediately |
| 2 | 0.2 | Quick regex fix in `version.go`, independent |
| 3 | 0.3 | Same file as 0.2 — batch version.go changes minimize churn |
| 4 | 0.1 | Straightforward one-liner in `checker.go` |
| 5 | 0.5 | Touches main exit path — stabilize before riskiest change |
| 6 | 0.4 | Most invasive (WalkDir logic) — do last with all context stable |

**Conventions**: One commit per bug, conventional commit messages (`fix: 0.x ...`). Each commit runs `go test ./...` green. Final validation: `go test ./...` + `go build ./...` + `go vet ./...`.

## Closed Decisions

| # | Decision | Chosen | Justification |
|---|----------|--------|---------------|
| 0.3 strategy | Last-match / Keyword / Hybrid | **A — Last match with ≥1 dot** | Directly implements spec requirement. Keeps `Extract` as pure regex utility. Copyright year case is correctly handled by dot filter. Keyword-awareness can be added later without contract change. |
| 0.4 scan scope | Full walk / Conservative / Whitelist | **A — Full `*/bin/` walk** | Spec scenario requires `tools/bin/go` to be found (non-standard dir). Conservative B misses it. Whitelist C adds maintenance burden for no benefit (PATH fallback is already the safety net). |
| 0.5 SilenceUsage | Yes / No | **Yes** | User ran command correctly; no reason to print usage on tool mismatch. |
| 0.5 SilenceErrors | Yes / No | **Yes** | Prevents duplicate stderr: `ui.Render` already prints results; Cobra printing the error adds noise. `main()` controls the exit path cleanly. |
| 0.5 ErrChecksFailed location | checker / cmd / new pkg | **`checker` package** | Semantically belongs with the check result types. `errors.Is` works across packages. No new package needed for one sentinel. |

## Migration

No data migration. No feature flags. Individual commits per bug with `git revert` as rollback.

## Open Questions

None — all design-time decisions are closed.
