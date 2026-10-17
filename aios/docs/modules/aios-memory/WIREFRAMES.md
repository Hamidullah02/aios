# aios-memory — Wireframes

## 1. Purpose

This document describes the wireframes for the
aios-memory module. The module has one user-facing
surface: the AGENTS.md template printed by
`aios init`.

## 2. Scope

The scope of this document is the AGENTS.md
template.

## 3. Reference Documents

- aios-memory PRD. See `docs/modules/aios-memory/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Wireframe | A visual layout for a user-facing surface. |

## 5. Screen List

The module has one screen: the AGENTS.md template.

## 6. Wireframes

### 6.1 AGENTS.md Template

```
# AGENTS.md

## Project

A short description of the project. The agent reads
this section to understand the project.

## Commands

- Build:  `<build-command>`
- Test:   `<test-command>`
- Lint:   `<lint-command>`
- Type:   `<type-check-command>`

## Conventions

- Use snake_case for functions and variables.
- Use PascalCase for classes.
- Use type hints on public functions.
- Use struct + go-playground/validator (or hand-rolled tag validation) for typed data.

## Architecture

- `internal/` holds the source code.
- `tests/` holds the test code.
- `docs/` holds the project documents.

## Notes

- The project uses Go 1.23.
- The project uses goroutines + channels (stdlib) + golang.org/x/sync/errgroup for async I/O.
- The project uses struct + go-playground/validator (or hand-rolled tag validation) for typed data.
```

### 6.2 .aiosignore Template

```
# Build artifacts
build/
dist/
*.egg-info/

# Virtual environments
.venv/
venv/

# Test artifacts
.go test_cache/
.coverage
htmlcov/

# Editor
.vscode/
.idea/

# OS
.DS_Store
Thumbs.db
```

## 7. Interaction Notes

- The CLI creates the AGENTS.md file on `aios init`.
- The CLI creates the .aiosignore file on
  `aios init`.
- The CLI does not create the CONVENTIONS.md file.
  The operator creates the file.
