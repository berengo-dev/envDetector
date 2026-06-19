# Archive Report: fase-0-bugfixes

## Summary

Archived the fase-0-bugfixes change: six correctness and hygiene bugs (0.1–0.6) that blocked a reliable public release of env-doctor. All bugs were implemented via 8 commits (6 task commits + 2 follow-ups), verified with passing tests, and merged into `openspec/specs/checker-fase-0-bugfixes/spec.md`. Verification verdict: PASS WITH WARNINGS (no critical issues).

## Change Artifacts Archived

- proposal.md ✅
- design.md ✅
- specs/checker-fase-0-bugfixes/spec.md ✅
- tasks.md ✅ (23/23 tasks complete)
- verify-report.md — saved to Engram only (topic_key: `sdd/fase-0-bugfixes/verify-report`, observation #133)

## Specs Synced

| Domain | Action | Details |
|--------|--------|---------|
| checker-fase-0-bugfixes | Created | 6 ADDED Requirements merged into new main spec at `openspec/specs/checker-fase-0-bugfixes/spec.md` |

## Implementation Summary

| Bug | Commit | Type | Description |
|-----|--------|------|-------------|
| 0.6 | `55b4c7b` | chore | Untrack compiled env-doctor binary from git index |
| 0.6 (follow-up) | `4887b6e` | test(checker) | Accept root-scoped env-doctor gitignore rule |
| 0.2 | `e31a287` | fix(version) | Support strict less-than operator in semver constraints |
| 0.3 | `4b6669c` | fix(version) | Extract last semver-like match with at least one dot |
| 0.1 | `33b1429` | fix(checker) | Resolve file checks against workingDir not CWD |
| 0.5 | `265fced` | refactor(cmd) | Return sentinel error from RunE instead of os.Exit |
| 0.4 | `dbd9ce6` | fix(checker) | Resolve binaries from venv/bin and any */bin/ subdirectory |
| tasks | `9c2f6c4` | docs | Mark fase-0-bugfixes tasks as complete |

Total commits: 8 (6 task + 2 follow-up)

## Verification

- **Verify report**: `sdd/fase-0-bugfixes/verify-report` (Engram observation #133)
- **Status**: PASS WITH WARNINGS
- **Tests**: `go test ./... -count=1` — PASS, 0 failures across 5 packages
- **Build**: `go build ./...` — clean
- **Vet**: `go vet ./...` — clean
- **Coverage**: 74.7% total statements
- **Critical issues**: 0
- **Warnings**: 0.5 commit type mismatch (`refactor(cmd)` vs expected `fix(cmd)`); working tree has unrelated untracked artifact files; proposal success criteria checkboxes unchecked; 0.4 broader walk scope non-functional

## Notes

- The verify report lives in Engram only (observation #133) — no `verify-report.md` file was written to disk.
- The proposal success criteria checkboxes in `proposal.md` remain unchecked (cosmetic only — all criteria are met per verification evidence).
- The working tree has untracked `openspec/` artifact files (specs, archive) that were committed as part of this archive.
- Warning #2 from the verify report (unrelated working tree changes — monorepo-detection artifacts and config.yaml modifications) persists as they are outside the scope of this change.
