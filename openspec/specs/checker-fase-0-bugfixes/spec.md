# Checker Fase 0 Bugfixes

## Purpose

Define corrected behavior for the checker, version parser, binary resolver, CLI exit handling, and repository hygiene.

## Requirements

### Requirement: checkFile MUST resolve relative paths against workingDir

`checkFile` SHALL resolve paths relative to the checker's `workingDir`, not the process CWD.

#### Scenario: Relative file found under config dir

- **WHEN** `NewWithDir("/project/config")` and `checkFile("data/config.json")`
- **THEN** system stats `/project/config/data/config.json` and passes if it exists

#### Scenario: File missing from config dir fails

- **WHEN** file does not exist under `workingDir`
- **THEN** result is `StatusFail` with "file not found"

### Requirement: Semver constraint regex MUST support `<` operator

The constraint regex SHALL match `<` (less-than) distinct from `<=`. Each operator SHALL appear exactly once in the alternation.

#### Scenario: `<` extracted as less-than operator

- **WHEN** `ConvertSemverToWildcard("<1.0.0")`
- **THEN** returns `("1.x", true)`

#### Scenario: `<=` extracted without ambiguity

- **WHEN** `ConvertSemverToWildcard("<=2.0.0")`
- **THEN** returns `("2.x", true)`; `<` and `<=` are distinct in the regex

### Requirement: Version extractor MUST return last match with ≥1 dot

`Extract` SHALL find the last match containing at least one dot (`\d+\.\d+`), filtering bare numbers like copyright years. When no dotted version exists, it SHALL return `""`.

#### Scenario: Copyright year skipped, version found

- **WHEN** input is `"Copyright 2024\nVersion 1.2.3"`
- **THEN** `Extract` returns `"1.2.3"`

#### Scenario: Bare number with no dot returns empty

- **WHEN** input is `"Version 1"`
- **THEN** `Extract` returns `""`

#### Scenario: Last dotted candidate wins among multiple

- **WHEN** input is `"0.1 (dev) 1.0.0"`
- **THEN** `Extract` returns `"1.0.0"`

### Requirement: resolveBinary MUST include venv/bin and walk `*/bin/`

`resolveBinary` SHALL check `venv/bin/{tool}` (no dot) at root level alongside `node_modules/.bin` and `.venv/bin`. The subdirectory walk SHALL match `*/bin/` directories beyond hardcoded `node_modules` and `.venv`.

#### Scenario: Python tool found in venv/bin (no dot prefix)

- **WHEN** `venv/bin/python` exists and `pyproject.toml` is detected
- **THEN** `resolveBinary("python")` returns `venv/bin/python`

#### Scenario: Tool found in non-standard subdirectory bin/

- **WHEN** a subdirectory `tools/bin/go` exists
- **THEN** `resolveBinary("go")` returns `tools/bin/go`

#### Scenario: Root candidates checked before subdirectory walk

- **WHEN** root `node_modules/.bin/tool` exists
- **THEN** it is returned immediately without subdirectory search

### Requirement: CLI MUST return error instead of calling os.Exit in RunE

The `check` command's `RunE` SHALL return a sentinel `ErrChecksFailed` when any check fails, instead of calling `os.Exit(1)`. `main()` SHALL inspect the error to set exit code 1.

#### Scenario: Failed checks propagate error

- **WHEN** any check returns `StatusFail`
- **THEN** `RunE` returns `ErrChecksFailed`; `main()` calls `os.Exit(1)` after Cobra

#### Scenario: All checks pass returns nil

- **WHEN** all checks pass
- **THEN** `RunE` returns `nil`; process exits 0

### Requirement: Compiled binary MUST NOT be tracked in git

The `env-doctor` binary SHALL be removed from the git index. `.gitignore` already excludes it; only the index needs cleanup.

#### Scenario: Binary absent from git index

- **WHEN** `git ls-files env-doctor` runs
- **THEN** the binary is not listed (local file may remain on disk)
