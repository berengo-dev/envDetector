# Proposal: Monorepo Detection

## Intent

env-doctor only scans root-level manifests, so monorepos with subdirectories (`backend/package.json`, `frontend/package.json`, `packages/*/package.json`) miss tools from subdirectories. This makes `env-doctor init --auto` and `env-doctor check` incomplete for monorepo workflows.

## Scope

### In Scope
- Recursive manifest scanning (WalkDir-based) across all subdirectories
- Workspace config parsing (`package.json` workspaces, `pnpm-workspace.yaml`) as optimization hints
- Subdirectory binary resolution in checker (`{subdir}/node_modules/.bin/`, `.venv/bin/`)
- Source annotation with subdirectory context (e.g., "Extracted from frontend/package.json")
- **Conflict detection**: when the same tool appears with different versions in different subdirectories, detect both and report both to the user
- `.env` and `.env.example` files in subdirectories
- Completely agnostic: no hardcoded stack knowledge, works with any manifest format

### Out of Scope
- Supporting `nx.json`, `lerna.json`, `turbo.json` parsing (future optimization)
- Per-subdirectory `.env-doctor.yaml` configs (single config remains)
- Automatic resolution of version conflicts (we detect and report; user decides)

## Capabilities

### New Capabilities
- `recursive-manifest-detection`: Walk subdirectories for manifest files, deduplicating tools across subdirectories with first-found-wins
- `workspace-aware-detection`: Parse npm/pnpm workspace configs to intelligently target subdirectory scanning
- `subdirectory-binary-resolution`: Resolve binaries in subdirectory `node_modules/.bin/`, `.venv/bin/` paths during checks

### Modified Capabilities
None — no existing specs to modify.

## Approach

**Hybrid WalkDir + Workspace Hints**
1. Parse root `package.json` workspaces / `pnpm-workspace.yaml` for target paths
2. Fall back: `filepath.WalkDir` with skip list (`.git`, `node_modules`, `.dist`, `vendor`, `.venv`, `__pycache__`, `.turbo`, `build`, `.next`)
3. Pass subdirectory paths to extractors (already subdir-compatible)
4. **Detect and report version conflicts**: if the same tool appears with different versions in different subdirectories, detect both, report both to the user, and include a warning comment in the YAML. Store the "highest version found" in the tool field but annotate all conflicting sources.
5. Update `Checker.resolveBinary` to search `{subdir}/node_modules/.bin/` before system PATH
6. **Detect `.env` files in subdirectories**: scan for `.env` and `.env.example` in all subdirectories and report them as separate checks

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/detect/detect.go` | Modified | Replace root-only `filepath.Glob` with recursive walk + workspace parsing |
| `internal/detect/scanners.go` | Modified | Wire BinaryScanner into Detect(); add subdirectory scan paths |
| `internal/checker/checker.go` | Modified | `resolveBinary` searches subdirectory bin paths |
| `internal/detect/extractors.go` | Unchanged | Already path-agnostic; receives subdirectory paths |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Performance: WalkDir on large repos | Medium | Skip list + workspace hints as primary path |
| Version conflicts: different subdirs want different versions | Medium | **Detect both, report both, let user choose**. Store highest version with warning comment. |
| False positives: non-project JSON files detected | Low | JSONExtractor already checks for dep fields |
| Config complexity: YAML comments for conflicts | Low | Comments are non-breaking; user can edit manually |

## Rollback Plan

Detection logic is self-contained in `detect.go`. Revert to root-only `filepath.Glob` by removing the WalkDir code path. Checker changes are additive — existing `resolveBinary` fallback to system PATH is preserved.

## Dependencies

None. Uses stdlib `filepath.WalkDir` and existing `encoding/json` for workspace parsing.

## Success Criteria

- [ ] `env-doctor init --auto` in a monorepo detects tools from ALL subdirectories
- [ ] Detected tools include source annotation with subdirectory path
- [ ] **Version conflicts are detected and reported to the user** (e.g., "eslint: frontend needs 8.x, backend needs 9.x")
- [ ] `env-doctor check` resolves binaries in subdirectory `node_modules/.bin/`
- [ ] `.env` and `.env.example` files in subdirectories are detected
- [ ] Non-monorepo projects produce identical output (no regression)
- [ ] Existing tests pass without modification
- [ ] Completely agnostic: works with any project structure and technology stack
