# aios-memory — Wireframes

## 1. Purpose

This document shows the wireframes for the
aios-memory module. The wireframes cover the
`aios memory` CLI. The CLI is a developer tool.
The output is grep-friendly JSON. The developer
can pipe to `jq`.

## 2. Scope

The scope of this document is the aios-memory
module.

## 3. Reference Documents

- aios-memory PRD. See `docs/modules/aios-memory/PRD.md`.
- aios-memory SRS. See `docs/modules/aios-memory/SRS.md`.
- aios-memory TRD. See `docs/modules/aios-memory/TRD.md`.
- aios-memory APP-FLOW. See `docs/modules/aios-memory/APP-FLOW.md`.
- aios-cli WIREFRAMES. See `docs/modules/aios-cli/WIREFRAMES.md`.

## 4. Wireframe Conventions

The wireframes use a monospace font. The output is
JSON. The developer can pipe to `jq` and filter.
The wireframes show the typical output for each
subcommand. The wireframes also show a sample
developer session.

## 5. `aios memory show`

The `aios memory show` subcommand prints the
current compacted view.

### 5.1 Output

```
$ aios memory show
{
  "view": {
    "artifacts": [
      {
        "kind": "task",
        "scope": "session",
        "session_id": "s_2026_01_15_abc",
        "task_id": "t_001",
        "timestamp": "2026-01-15T10:30:00Z",
        "tags": ["auth", "refactor"],
        "content_hash": "a1b2c3d4...",
        "content": "Refactor the authentication module to use JWT."
      },
      {
        "kind": "decision",
        "scope": "project",
        "session_id": "s_2026_01_10_xyz",
        "task_id": null,
        "timestamp": "2026-01-10T14:22:00Z",
        "tags": ["auth", "jwt"],
        "content_hash": "e5f6g7h8...",
        "content": "Use HS256 for JWT signing. Rotate keys every 90 days."
      }
    ],
    "total_tokens": 1024,
    "truncated": false,
    "pin_set_id": "p1n2s3t4...",
    "index_id": "1nd3x1d2..."
  }
}
```

### 5.2 With `--hash`

The `--hash` flag prints a single artifact.

```
$ aios memory show --hash a1b2c3d4
{
  "kind": "task",
  "scope": "session",
  "session_id": "s_2026_01_15_abc",
  "task_id": "t_001",
  "timestamp": "2026-01-15T10:30:00Z",
  "tags": ["auth", "refactor"],
  "content_hash": "a1b2c3d4...",
  "content": "Refactor the authentication module to use JWT."
}
```

## 6. `aios memory grep`

The `aios memory grep` subcommand returns the
matching artifacts.

### 6.1 Output

```
$ aios memory grep "jwt"
[
  {
    "kind": "decision",
    "scope": "project",
    "session_id": "s_2026_01_10_xyz",
    "task_id": null,
    "timestamp": "2026-01-10T14:22:00Z",
    "tags": ["auth", "jwt"],
    "content_hash": "e5f6g7h8...",
    "content": "Use HS256 for JWT signing. Rotate keys every 90 days."
  },
  {
    "kind": "observation",
    "scope": "project",
    "session_id": "s_2026_01_12_qwe",
    "task_id": null,
    "timestamp": "2026-01-12T09:15:00Z",
    "tags": ["jwt", "library"],
    "content_hash": "i9j0k1l2...",
    "content": "The github.com/golang-jwt/jwt library is the standard."
  }
]
```

## 7. `aios memory pin`

The `aios memory pin` subcommand pins an
artifact.

### 7.1 Output

```
$ aios memory pin e5f6g7h8
{
  "status": "ok",
  "pin_set_id": "p1n2s3t4...",
  "pinned_count": 3
}
```

## 8. `aios memory unpin`

The `aios memory unpin` subcommand unpins an
artifact.

### 8.1 Output

```
$ aios memory unpin e5f6g7h8
{
  "status": "ok",
  "pin_set_id": "p1n2s3t4...",
  "pinned_count": 2
}
```

## 9. `aios memory prune`

The `aios memory prune` subcommand marks an
artifact as pruned.

### 9.1 Output

```
$ aios memory prune a1b2c3d4
{
  "status": "ok",
  "pruned_count": 1
}
```

## 10. `aios memory diff`

The `aios memory diff` subcommand returns the
artifacts that differ between two sessions.

### 10.1 Output

```
$ aios memory diff s_2026_01_15_abc s_2026_01_15_def
{
  "added": [
    {
      "kind": "decision",
      "scope": "session",
      "session_id": "s_2026_01_15_def",
      "task_id": null,
      "timestamp": "2026-01-15T11:45:00Z",
      "tags": ["auth", "redis"],
      "content_hash": "m3n4o5p6...",
      "content": "Cache the JWT validation in Redis for 5 minutes."
    }
  ],
  "removed": [],
  "changed": []
}
```

## 11. `aios memory gc`

The `aios memory gc` subcommand compacts the raw
layer.

### 11.1 Output

```
$ aios memory gc
{
  "status": "ok",
  "before": {
    "files": 1024,
    "size_bytes": 104857600
  },
  "after": {
    "files": 1,
    "size_bytes": 52428800
  }
}
```

## 12. `aios memory stats`

The `aios memory stats` subcommand prints the
memory statistics.

### 12.1 Output

```
$ aios memory stats
{
  "raw": {
    "files": 1024,
    "size_bytes": 104857600
  },
  "indexed": {
    "artifacts": 1024,
    "by_kind": {
      "task": 50,
      "file": 200,
      "decision": 100,
      "observation": 300,
      "tool_result": 250,
      "model_output": 124
    }
  },
  "pin_set": {
    "count": 3
  },
  "scopes": {
    "session": 50,
    "project": 500,
    "user": 474
  }
}
```

## 13. Developer Session

The following developer session shows how a
developer uses the `aios memory` CLI to inspect
the agent's memory, find a decision, pin it, and
diff two sessions.

### 13.1 Inspect

```
$ aios memory show | jq '.view.artifacts[] | select(.tags[] == "auth")'
{
  "kind": "decision",
  "scope": "project",
  "session_id": "s_2026_01_10_xyz",
  "task_id": null,
  "timestamp": "2026-01-10T14:22:00Z",
  "tags": ["auth", "jwt"],
  "content_hash": "e5f6g7h8...",
  "content": "Use HS256 for JWT signing. Rotate keys every 90 days."
}
```

### 13.2 Grep

```
$ aios memory grep "jwt" | jq '.[0].content_hash'
"e5f6g7h8..."
```

### 13.3 Pin

```
$ aios memory pin e5f6g7h8
{
  "status": "ok",
  "pin_set_id": "p1n2s3t4...",
  "pinned_count": 3
}
```

### 13.4 Diff

```
$ aios memory diff s_2026_01_15_abc s_2026_01_15_def | jq '.added[].content'
"Cache the JWT validation in Redis for 5 minutes."
```

## 14. Error States

### 14.1 Artifact Not Found

```
$ aios memory show --hash nonexistent
{
  "status": "error",
  "code": "ERR_NOT_FOUND",
  "message": "artifact not found"
}
```

### 14.2 Index Corrupt

```
$ aios memory show
{
  "status": "error",
  "code": "ERR_INDEX_CORRUPT",
  "message": "index file at var/memory/index.json failed to parse"
}
```

### 14.3 Budget Too Small

```
$ aios memory show --budget 0
{
  "status": "error",
  "code": "ERR_BUDGET_TOO_SMALL",
  "message": "budget is smaller than the smallest artifact"
}
```

## 15. Empty States

### 15.1 Empty Memory

```
$ aios memory show
{
  "view": {
    "artifacts": [],
    "total_tokens": 0,
    "truncated": false,
    "pin_set_id": "p1n2s3t4...",
    "index_id": "1nd3x1d2..."
  }
}
```

### 15.2 No Grep Matches

```
$ aios memory grep "nonexistent"
[]
```

## 16. Piping Examples

The output is grep-friendly JSON. The developer
can pipe to `jq` and filter.

### 16.1 All Decisions

```
$ aios memory show | jq '.view.artifacts[] | select(.kind == "decision")'
```

### 16.2 All Artifacts Tagged "auth"

```
$ aios memory show | jq '.view.artifacts[] | select(.tags[] == "auth")'
```

### 16.3 All Pinned Artifacts

```
$ aios memory show | jq '.view.artifacts[] | select(.content_hash == "e5f6g7h8" or .content_hash == "a1b2c3d4" or .content_hash == "i9j0k1l2")'
```

### 16.4 Token Count

```
$ aios memory show | jq '.view.total_tokens'
1024
```

### 16.5 Truncated

```
$ aios memory show | jq '.view.truncated'
false
```
