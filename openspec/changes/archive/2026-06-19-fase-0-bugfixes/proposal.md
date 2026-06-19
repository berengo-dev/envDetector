# Proposal: Fase 0 Bugfixes

## Intent

Six confirmed correctness/hygiene bugs block a reliable public release. Fase 0 fixes them before any other improvement phase — bugs 0.1–0.4 silently produce wrong check results, 0.5 breaks Go process hygiene, 0.6 pollutes the repo with a committed binary.

## Scope

### In Scope
- **0.1**: `checkFile` ignores `workingDir` — `checker.go:246`
- **0.2**: semver constraint regex broken (`<=` dup, `<` missing) — `version.go:27`
- **0.3**: `versionRegex` grabs first number, not version — `version.go:13`
- **0.4**: `resolveBinary` hardcoded to Node/.venv only — `checker.go:167–201`
- **0.5**: `os.Exit(1)` inside Cobra `RunE` — `main.go:72`
- **0.6**: compiled `env-doctor` binary tracked by git — `.gitignore` + `git rm --cached`

### Out of Scope
Any Fase 1+ improvements (error messages, progress bars, new scanners, refactors beyond these 6 bugs).

## Capabilities

### New Capabilities
None — bugfixes only.

### Modified Capabilities
None — no spec-level behavior changes.

## Approach

| Bug | File:Line | Fix Strategy | Test (TDD red→green) | Risk |
|-----|-----------|-------------|----------------------|------|
| 0.1 | `checker.go:246` | `os.Stat(filepath.Join(c.workingDir, path))` | `NewWithDir` + relative path → FAIL before fix | Low |
| 0.2 | `version.go:27` | Fix alternation: `(\^|~|>=|<=|>|<|=)` | `ConvertSemverToWildcard("<1.0.0")` → `("",false)` before fix | Low |
| 0.3 | `version.go:13` | Require ≥1 dot: `\d+\.\d+(?:\.\d+)?` + find last match | `Extract("Copyright 2024\nVersion 1.2.3")` → `"2024"` before fix | Low |
| 0.4 | `checker.go:167–201` | Add `venv/bin` to rootCandidates; extend WalkDir to `*/bin/` | `resolveBinary("python")` with `venv/bin/python` → PATH fallback before fix | Low |
| 0.5 | `main.go:72` | Return `ErrChecksFailed` sentinel; `main()` checks `errors.Is` to suppress duplicate stderr | Integration: failing check exits 1 via error, not `os.Exit` | Low |
| 0.6 | `.gitignore:2` + index | `git rm --cached env-doctor` (`.gitignore` entry already exists) | `git ls-files --error-unmatch env-doctor` succeeds before fix → fails after | Low |

**0.3 strategy note**: Default is "last match with ≥1 dot." Keyword-aware alternative (prefer near "version"/"v") is more robust but adds complexity. Default chosen; user may override at design phase.

## Change Strategy

- **Estimated delta**: ~130 lines (well under 400-line budget)
- **Chained PRs**: Not needed. Single PR, 6 reviewable commits.
- **Dependencies**: 0.2 ↔ 0.3 tightly coupled (same file `version.go`). Others independent.

## Affected Areas

| File | Bugs | Δ |
|------|------|---|
| `internal/checker/checker.go` | 0.1, 0.4 | ~8 lines |
| `internal/checker/checker_test.go` | 0.1 | ~15 lines |
| `internal/checker/binary_test.go` | 0.4 | ~18 lines |
| `pkg/version/version.go` | 0.2, 0.3 | ~8 lines |
| `pkg/version/version_test.go` | 0.2, 0.3 | ~15 lines |
| `cmd/env-doctor/main.go` | 0.5 | ~10 lines |
| `.gitignore` + git index | 0.6 | 0 code lines |

## Conventions Detected

- Change names: `kebab-case`, English artifacts (from `monorepo-detection`)
- Proposal structure: Intent → Scope → Capabilities → Approach → Risks → Rollback → Success Criteria
- Specs: `openspec/specs/<domain>/spec.md`, Given/When/Then scenarios, RFC 2119 keywords
- Archive naming: `YYYY-MM-DD-<change-name>/`
- Strict TDD: active (`strict_tdd: true` in `config.yaml`)

## Risks & Open Questions

- **0.3 single-component regression**: Requiring ≥1 dot filters out bare `Version 1`-style output. In practice, no tool outputs versions without a dot. No existing test covers this either.
- **0.4 `*/bin/` breadth**: Scanning all `*/bin/` dirs could surface unexpected binaries. Mitigated: this only affects resolution order; PATH fallback remains.
- **0.5 UX**: Currently `os.Exit(1)` immediately terminates — returning error may allow Cobra to print usage (undesirable). Mitigated with `cmd.SilenceUsage = true`.

## Rollback Plan

Each bug is isolated to its file. Revert individual commits. No schema, config, or API changes.

## Dependencies

None. All stdlib (`filepath`, `errors`, `os`, `regexp`).

## Success Criteria

- [ ] 0.1: File checks resolve relative to config directory
- [ ] 0.2: `<1.0.0` constraint converts to `1.x` wildcard
- [ ] 0.3: `Extract("Copyright 2024\nVersion 1.2.3")` returns `1.2.3`
- [ ] 0.4: Binary in `venv/bin/` (no dot) is resolved
- [ ] 0.5: Failed checks return error without skipping deferreds
- [ ] 0.6: `env-doctor` binary no longer tracked by git
- [ ] All existing tests pass; 6 new test cases (one per bug)
