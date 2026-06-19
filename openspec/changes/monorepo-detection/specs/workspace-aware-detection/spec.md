# Workspace-Aware Detection

## Purpose

Optimize manifest scanning in monorepos by parsing workspace configuration files (`package.json` workspaces, `pnpm-workspace.yaml`) before falling back to a full recursive directory walk. Workspace configs serve as optimization hints and MUST NOT restrict what is ultimately scanned.

## Requirements

### Requirement: Package.json Workspaces Parsing

The system MUST parse the `workspaces` field from the root `package.json` when present. The `workspaces` field MAY contain an array of glob patterns (e.g., `["packages/*", "apps/*"]`) or an object with a `packages` array. The system MUST expand each glob relative to the project root to produce a list of scan target directories.

#### Scenario: package.json with workspaces array

- GIVEN root `package.json` contains `"workspaces": ["packages/*", "apps/*"]`
- WHEN detection runs
- THEN `packages/client`, `packages/server`, and `apps/web` are scanned first
- AND any remaining unscanned subdirectories are walked via full recursive walk

#### Scenario: package.json workspaces as object

- GIVEN root `package.json` contains `"workspaces": {"packages": ["packages/*"]}`
- WHEN detection runs
- THEN `packages/*` directories are scanned as priority targets

#### Scenario: No workspaces field

- GIVEN root `package.json` has no `workspaces` field
- WHEN detection runs
- THEN detection still performs full recursive WalkDir
- AND all manifests are still discovered
- NOTE: The WalkDir always runs; workspace hints are only for prioritizing specific paths

### Requirement: pnpm-workspace.yaml Parsing

The system MUST parse `pnpm-workspace.yaml` from the project root when present. The system MUST extract the `packages` field, which MAY contain an array of glob patterns. Globs MUST be expanded relative to the project root.

#### Scenario: pnpm-workspace.yaml present

- GIVEN root contains `pnpm-workspace.yaml` with `packages: ['packages/*']`
- WHEN detection runs
- THEN `packages/*` directories are scanned as priority targets
- AND the full recursive WalkDir still covers all directories (workspace hints do not replace WalkDir)

#### Scenario: Malformed pnpm-workspace.yaml

- GIVEN `pnpm-workspace.yaml` is present but not valid YAML
- WHEN detection runs
- THEN the system logs a warning
- AND falls back to full recursive WalkDir without error

### Requirement: Workspace Hint Integration

The system MUST always perform a full recursive WalkDir regardless of workspace configuration. Workspace configs serve as optimization hints — they are used to prioritize scanning of specific paths, but they MUST NOT replace or restrict the WalkDir. The skip list (`.git`, `node_modules`, etc.) MUST apply to all scans. The WalkDir ensures all manifests are discovered even when workspace configs are stale, incomplete, or absent.

#### Scenario: Workspace config is stale or incomplete

- GIVEN `pnpm-workspace.yaml` lists `['packages/*']` but `services/api/package.json` exists outside packages/
- WHEN detection runs
- THEN `services/api/package.json` is still discovered via the fallback WalkDir

#### Scenario: Both workspace configs present

- GIVEN root has both `package.json` workspaces and `pnpm-workspace.yaml`
- WHEN detection runs
- THEN the system SHOULD prefer the config matching the project's lock file
- AND all workspace paths from the preferred config are scanned as priority targets
- AND the fallback WalkDir covers remaining directories
