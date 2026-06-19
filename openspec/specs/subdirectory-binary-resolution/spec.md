# Subdirectory Binary Resolution

## Purpose

Enable `env-doctor check` to find tool binaries installed in subdirectory-level executable paths (`node_modules/.bin`, `.venv/bin`) during health checks, ensuring monorepo tools are correctly located even when installed per-subdirectory.

## Requirements

### Requirement: Subdirectory Binary Search

The system MUST search for tool binaries in subdirectory executable paths before falling back to the system PATH. The search MUST cover `{workDir}/**/node_modules/.bin/{tool}` and `{workDir}/**/.venv/bin/{tool}` patterns. The system MUST return the first matching binary found.

#### Scenario: Binary in frontend subdirectory node_modules

- GIVEN working directory contains `frontend/node_modules/.bin/eslint`
- WHEN `env-doctor check` runs for tool `eslint`
- THEN `frontend/node_modules/.bin/eslint` is resolved
- AND the version check executes against that binary

#### Scenario: Binary in multiple subdirectories

- GIVEN both `frontend/node_modules/.bin/eslint` and `backend/node_modules/.bin/eslint` exist
- WHEN `env-doctor check` resolves `eslint`
- THEN the first matching binary encountered in deterministic order is returned

#### Scenario: No subdirectory binary found

- GIVEN no `node_modules/.bin/eslint` or `.venv/bin/eslint` exists in any subdirectory
- WHEN `env-doctor check` resolves `eslint`
- THEN the system falls back to searching system PATH
- AND the original `resolveBinary` fallback behavior is preserved

### Requirement: Backward Compatible Root-First Resolution

The system MUST preserve the existing behavior of checking the working directory root `node_modules/.bin/{tool}` FIRST, before searching subdirectories.

#### Scenario: Root binary found

- GIVEN `node_modules/.bin/eslint` exists at the working directory root
- WHEN `env-doctor check` resolves `eslint`
- THEN the root binary is returned immediately
- AND no subdirectory search is performed

### Requirement: Deterministic Subdirectory Search

The system MUST search subdirectories in a deterministic order (traversal order of the recursive walk) to ensure reproducible behavior across runs.

#### Scenario: Consistent resolution across runs

- GIVEN multiple subdirectories contain the same binary
- WHEN `env-doctor check` runs twice in the same project
- THEN the same binary path is resolved both times
