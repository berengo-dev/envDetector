# Recursive Manifest Detection

## Purpose

Enable env-doctor to discover tools, environment files, and manifests from all subdirectories in a project, supporting monorepos and multi-package projects without hardcoding knowledge of specific stacks or frameworks.

## Requirements

### Requirement: Recursive Manifest Walk

The system MUST walk all subdirectories from the project root when detecting manifest files. The walk MUST NOT descend into `.git`, `node_modules`, `.dist`, `vendor`, `.venv`, `__pycache__`, `.turbo`, `build`, `.next`, `out`, `target` directories.

#### Scenario: Monorepo with frontend and backend

- GIVEN a project root with `frontend/package.json` and `backend/go.mod`
- WHEN `env-doctor init --auto` runs
- THEN tools from both `frontend/package.json` and `backend/go.mod` are detected
- AND each tool is annotated with its source subdirectory

#### Scenario: Single package project (backward compatibility)

- GIVEN a project with only `package.json` at the root
- WHEN `env-doctor init --auto` runs
- THEN behavior is identical to current root-only detection
- AND no subdirectory walk occurs if no manifests exist below root

#### Scenario: Skipped directories

- GIVEN a project with `node_modules/package.json` and `vendor/go.mod`
- WHEN detection runs
- THEN those files are NOT processed
- AND detection completes without errors

### Requirement: Recursive Env and File Scanning

The system MUST scan for `.env` and `.env.example` files in all subdirectories visited during the recursive walk. File scanning MUST detect common project files (`.gitignore`, `Makefile`, `Dockerfile`, manifest files) in subdirectories.

#### Scenario: Subdirectory .env detection

- GIVEN `frontend/.env.example` and `backend/.env` exist
- WHEN `env-doctor init --auto` runs
- THEN env variables from both files are aggregated
- AND each variable's source subdirectory is annotated

#### Scenario: Subdirectory project files

- GIVEN `frontend/Dockerfile` and `backend/Makefile` exist
- WHEN detection runs
- THEN both files are included in the `files` list
- AND each file is annotated with its subdirectory path

### Requirement: Tool Deduplication

The system MUST deduplicate tools detected across multiple subdirectories. For non-conflicting duplicates (same tool, same version), the system MUST store only one entry per tool using first-found-wins semantics.

#### Scenario: Duplicate tool with same version

- GIVEN eslint `^8.0.0` appears in `frontend/package.json` and `packages/shared/package.json`
- WHEN detection runs
- THEN eslint appears once in config with version `8.x`
- AND the first source encountered is annotated

### Requirement: Version Conflict Detection and Reporting

The system MUST detect when the same tool name appears with different version constraints across subdirectories. When conflicts are found, the system MUST store the highest version in the `tools` map and MUST emit a warning comment listing ALL conflicting sources and their versions.

#### Scenario: Version conflict between subdirectories

- GIVEN `frontend/package.json` requires eslint `^8.0.0` and `backend/package.json` requires eslint `^9.0.0`
- WHEN `env-doctor init --auto` runs
- THEN config stores `eslint: "9.x"` (highest version)
- AND a warning comment lists both sources with their versions

#### Scenario: Same version across subdirectories (no conflict)

- GIVEN both `frontend/package.json` and `backend/package.json` require eslint `^8.0.0`
- WHEN detection runs
- THEN eslint appears once in config with version `8.x`
- AND no conflict warning is emitted

#### Scenario: Three-way version conflict

- GIVEN eslint `^8.0.0`, `^9.0.0`, and `^7.0.0` across three subdirectories
- WHEN detection runs
- THEN config stores `eslint: "9.x"` (highest)
- AND warning lists all three conflicting sources with their versions
