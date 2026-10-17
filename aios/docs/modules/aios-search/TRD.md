# aios-search — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-search module.

## 2. Scope

The scope of this document is the aios-search module.
The module is a Go package.

## 3. Reference Documents

- aios-search PRD. See `docs/modules/aios-search/PRD.md`.
- aios-search SRS. See `docs/modules/aios-search/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-tools TRD. See `docs/modules/aios-tools/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| ripgrep | A fast content search tool. |
| tree-sitter | A parsing library for source code. |
| os/exec (`rg`) | A direct invocation of the `rg` binary. |

## 5. Architecture

The aios-search module has four components.

- `internal/search/glob` — the Glob tool.
- `internal/search/grep` — the Grep tool.
- `aios.search.code_search` — the CodeSearch tool.
- `aios.search.diagnostics` — the Diagnostics
  tool.

The four components depend on aios-core and
aios-tools. The components depend on each other in
this order. The Glob tool has no dependencies. The
Grep tool has no dependencies. The CodeSearch tool
uses tree-sitter. The Diagnostics tool uses the
os/exec wrapper from aios-sandbox.

## 6. Components

### 6.1 Glob Tool

The Glob tool uses the `path/filepath (stdlib)` module. The tool
walks the workspace. The tool matches the paths
against the glob pattern. The tool returns the
matching paths.

The tool uses the `.aiosignore` file. The tool
loads the file. The tool skips the paths that
match any pattern in the file.

### 6.2 Grep Tool

The Grep tool uses `ripgrepy`. The tool calls
ripgrep with the pattern, the path, and the
options. The tool parses the JSON output. The tool
returns the matches.

The tool supports the `content`, `files_with_matches`,
and `count` modes. The tool supports the `-n` flag
for line numbers.

### 6.3 CodeSearch Tool

The CodeSearch tool uses tree-sitter. The tool
loads the language grammar for the file extension.
The tool parses the file. The tool walks the AST.
The tool finds the symbol that matches the name
and the kind.

The tool supports Go, JavaScript, TypeScript,
and Rust. The tool uses the `tree-sitter-language-pack`
for the grammars.

### 6.4 Diagnostics Tool

The Diagnostics tool uses the os/exec wrapper
from aios-sandbox. The tool runs the checker. The
tool parses the checker output. The tool returns
the diagnostics.

The tool supports `go vet`, `gofmt + go vet`, `pylint`, and
`eslint`. The tool uses a per-checker parser. The
parser reads the JSON output when available.

## 7. Data Structures

### 7.1 Match

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `path` | string | yes | The path of the file. |
| `line` | int | yes | The line number. |
| `column` | int | no | The column number. |
| `text` | string | yes | The matching line. |
| `context_before` | list of string | no | The lines before. |
| `context_after` | list of string | no | The lines after. |

### 7.2 Diagnostic

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `path` | string | yes | The path of the file. |
| `line` | int | yes | The line number. |
| `column` | int | no | The column number. |
| `severity` | string | yes | The severity. |
| `code` | string | no | The error code. |
| `message` | string | yes | The message. |

## 8. Interfaces

### 8.1 GrepTool

```go
type GrepTool:
    """The grep tool."""

    func run(self, args: dict, context: ToolContext) -> ToolResult:
        """Run the grep tool.

        Args:
            args: The arguments of the call.
            context: The context for the call.

        Returns:
            The result of the call.
        """
```

## 9. Algorithms

### 9.1 Grep Algorithm

The grep algorithm works as follows.

1. Validate the arguments.
2. Load the `.aiosignore` patterns.
3. Build the ripgrep command.
4. Run ripgrep through the os/exec wrapper.
5. Parse the JSON output.
6. Return the matches.

### 9.2 Code Search Algorithm

The code search algorithm works as follows.

1. Walk the workspace for the source files.
2. For each file, load the language grammar.
3. Parse the file.
4. Walk the AST.
5. Find the symbol that matches the name and the
   kind.
6. Return the locations.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- ripgrepy
- tree-sitter
- tree-sitter-language-pack
- aios-core
- aios-tools
- aios-sandbox

### 10.2 Module Dependencies

The aios-search module depends on the following
internal modules.

- aios-core
- aios-tools
- aios-sandbox

The aios-search module is depended on by the
following internal modules.

- aios-runtime
- aios-cli

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/search`. The module is
registered in the tool bus.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The Grep tool is slow on large repos. | Medium | Medium | The tool uses ripgrep. The tool respects `.aiosignore`. |
| The CodeSearch tool has no grammar. | Medium | Medium | The tool raises `UnsupportedLanguageError`. The tool lists the supported languages. |
| The Diagnostics tool misses a checker. | Low | Medium | The tool supports a custom checker. The tool is extensible. |
