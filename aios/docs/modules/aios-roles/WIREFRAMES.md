# aios-roles — Wireframes

## 1. Purpose

This document describes the wireframes for the
aios-roles module. The module has one user-facing
surface: the roles.yaml template printed by
`aios init`.

## 2. Scope

The scope of this document is the roles.yaml
template.

## 3. Reference Documents

- aios-roles PRD. See `docs/modules/aios-roles/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Wireframe | A visual layout for a user-facing surface. |

## 5. Screen List

The module has one screen: the roles.yaml template.

## 6. Wireframes

### 6.1 roles.yaml Template

```yaml
roles:
  planner:
    system_prompt: |
      You are a planner. Break the goal into a
      sequence of tasks. Use the tools to inspect
      the codebase first.
    tools: [read_file, search_files, run_shell]
    model_tier: strong
    output_format: json
    temperature: 0
    max_tokens: 8000

  coder:
    system_prompt: |
      You are a coder. Write the code that
      satisfies the task. Use the edit and write
      tools.
    tools:
      - read_file
      - write_file
      - edit
      - run_shell
    model_tier: strong
    output_format: code
    temperature: 0

  editor:
    system_prompt: |
      You are an editor. Apply small edits to the
      code. Use the edit tool.
    tools: [read_file, edit]
    model_tier: medium
    output_format: code
    temperature: 0

  explorer:
    system_prompt: |
      You are an explorer. Find the relevant files
      and read them. Do not edit files.
    tools: [read_file, search_files, grep]
    model_tier: weak
    output_format: text
    temperature: 0

  reviewer:
    system_prompt: |
      You are a reviewer. Check the code for
      bugs. Suggest fixes.
    tools: [read_file, search_files, grep]
    model_tier: strong
    output_format: text
    temperature: 0

  verifier:
    system_prompt: |
      You are a verifier. Check the result against
      the goal. Return pass, fail, or
      inconclusive.
    tools: [read_file, search_files, run_tests]
    model_tier: strong
    output_format: json
    temperature: 0
```

## 7. Interaction Notes

- The CLI creates the roles.yaml file on
  `aios init`.
- The operator can edit the file to add new
  roles or change the default roles.
- The CLI reloads the file when the file mtime
  changes.
