# Tasks: Fase 0 Bugfixes

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low

## 1. [Bug 0.6] Untrack compiled env-doctor binary from git index
- [x] **RED**: `git ls-files --error-unmatch env-doctor` exits 0 (binary tracked)
- [x] **GREEN**: `git rm --cached env-doctor`
- [x] **VERIFY**: `git ls-files --error-unmatch env-doctor` exits non-zero
- Files: `.gitignore` (no edit — entry already present), git index
- Commit: `chore: untrack compiled env-doctor binary from git index`
- Notes: NOT standard TDD — pure git hygiene. `--cached` preserves the file on disk.

## 2. [Bug 0.2] Add `<` operator support to semver constraint regex
- [x] **RED**: Add `TestConvertSemverToWildcardLessThan` to `pkg/version/version_test.go`: `ConvertSemverToWildcard("<1.0.0")` expects `("1.x", true)`
- [x] **VERIFY RED**: `go test ./pkg/version/...` fails for new case
- [x] **GREEN**: Fix `semverConstraintRegex` in `pkg/version/version.go:27`: `(\^|~|>=|<=|>|<|=)` — remove `<=` dup, add `<`
- [x] **VERIFY GREEN**: `go test ./pkg/version/...` passes; `("<=2.0.0")` still returns `("2.x", true)`
- Files: `pkg/version/version.go`, `pkg/version/version_test.go`
- Commit: `fix(version): support strict less-than operator in semver constraints`
- Notes: Existing `>=`, `<=`, `^`, `~`, `=` cases must stay green.

## 3. [Bug 0.3] Fix version extractor — return last dotted match, skip bare numbers
- [x] **RED**: Add 3 tests to `pkg/version/version_test.go`: `TestExtractSkipsBareNumber` (`"Copyright 2024\nVersion 1.2.3"` → `"1.2.3"`), `TestExtractBareNumberReturnsEmpty` (`"Version 1"` → `""`), `TestExtractLastDottedWins` (`"0.1 (dev) 1.0.0"` → `"1.0.0"`)
- [x] **VERIFY RED**: `go test ./pkg/version/...` fails
- [x] **GREEN**: Rewrite `Extract()` in `pkg/version/version.go` — use `FindAllString`, iterate with `strings.Contains(m, ".")`, return last match with dot
- [x] **VERIFY GREEN**: `go test ./pkg/version/...` passes; all existing `TestExtract` cases green
- Files: `pkg/version/version.go`, `pkg/version/version_test.go`
- Commit: `fix(version): return last dotted version match, skip bare numbers like years`
- Notes: Pure function. No keyword heuristic per design decision. `strings.TrimPrefix(last, "v")` preserved.

## 4. [Bug 0.1] Fix checkFile resolution — use workingDir instead of CWD
- [x] **RED**: Add `TestCheckFileRelativeToConfigDir` to `internal/checker/checker_test.go`: `NewWithDir(tmpDir)` + relative path → `StatusPass` (before fix `os.Stat` uses process CWD → fails)
- [x] **VERIFY RED**: `go test ./internal/checker/...` fails
- [x] **GREEN**: Change `os.Stat(path)` to `os.Stat(filepath.Join(c.workingDir, path))` at `internal/checker/checker.go:246`
- [x] **VERIFY GREEN**: `go test ./internal/checker/...` passes
- Files: `internal/checker/checker.go`, `internal/checker/checker_test.go`
- Commit: `fix(checker): resolve checkFile paths relative to workingDir`
- Notes: One-line change. Mirrors `loadEnvFile` which already resolves relative to `workingDir`.

## 5. [Bug 0.5] Replace os.Exit in RunE with sentinel error
- [x] **RED**: Add checker integration test: create `Checker` with known-failing config, capture `RunE` error, assert `errors.Is(err, ErrChecksFailed)`. Before fix, `os.Exit(1)` terminates test.
- [x] **VERIFY RED**: Test process killed by `os.Exit(1)` (expected abort)
- [x] **GREEN**: Add `var ErrChecksFailed` sentinel in `internal/checker/checker.go`. In `cmd/env-doctor/main.go`: change `checkCmd.RunE` to `return checker.ErrChecksFailed`; set `SilenceUsage=true`, `SilenceErrors=true`; change `main()` to catch error, call `os.Exit(1)` only for non-check failures.
- [x] **VERIFY GREEN**: `go test ./internal/checker/...` passes; `go build ./...` succeeds
- Files: `internal/checker/checker.go` (add sentinel), `cmd/env-doctor/main.go` (RunE + main)
- Commit: `fix(cmd): return sentinel error instead of os.Exit inside RunE`
- Notes: SilenceUsage avoids duplicate usage text on correct invocation. Deferreds now run (`os.Exit` gone from RunE).

## 6. [Bug 0.4] Extend resolveBinary — detect venv/bin and walk all `*/bin/` dirs
- [x] **RED**: Add `TestResolveBinaryVenvNoDot` to `internal/checker/binary_test.go`: `venv/bin/python` at root resolves. Add `TestResolveBinaryNonStandardBinDir`: `tools/bin/go` resolves.
- [x] **VERIFY RED**: `go test ./internal/checker/...` fails (venv not in rootCandidates, WalkDir misses generic `bin/`)
- [x] **GREEN**: Add `venv/bin/{name}` to `rootCandidates`. In WalkDir, add `case "bin":` — check `filepath.Join(path, name)`, don't SkipDir. Keep `node_modules` handler for `.bin`. Remove standalone `.venv` SkipDir (bin handler catches it).
- [x] **VERIFY GREEN**: `go test ./internal/checker/...` passes; existing `node_modules/.bin` and `.venv/bin` tests must remain green
- Files: `internal/checker/checker.go` (lines 167–201), `internal/checker/binary_test.go`
- Commit: `fix(checker): expand binary resolution to venv/bin and generic */bin/ dirs`
- Notes: Most invasive. Run last with all context stable. Ensure no regression for `node_modules/.bin` and `.venv/bin` — both are covered by their explicit handlers.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~130 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low

### Estimation rationale
~97 lines of production changes (0.6: 0, 0.2: 2, 0.3: 8, 0.1: 1, 0.5: 10, 0.4: 8) + ~33 lines of test code = ~130. Each bug is 1–8 lines of source changes. No generated code, migrations, or new files inflating the diff.
