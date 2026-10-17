# aios-egress — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-egress module.

## 2. Scope

The scope of this document is the aios-egress module.
The module is a Go package and a service.

## 3. Reference Documents

- aios-egress PRD. See `docs/modules/aios-egress/PRD.md`.
- aios-egress SRS. See `docs/modules/aios-egress/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| net/http (stdlib) | The Go standard-library HTTP client. |
| net/http (stdlib) | The Go standard-library HTTP server. |

## 5. Architecture

The aios-egress module has three components.

- `internal/egress/proxy` — the proxy server.
- `aios.egress.allow` — the allow list.
- `aios.egress.headers` — the header rules.

The three components depend on aios-core. The
components depend on each other in this order. The
allow list has no dependencies. The header rules
have no dependencies. The proxy depends on the allow
list, the header rules, and the audit log.

## 6. Components

### 6.1 Allow List

The allow list component loads the YAML file. The
component builds a list of allowed hosts. The
component supports exact match and glob match. The
component uses the `fnmatch` module for glob match.

### 6.2 Header Rules

The header rules component loads the YAML file. The
component builds a list of header rules. The
component supports removal and addition. The
component rewrites the headers in the request and
the response.

### 6.3 Proxy

The proxy component uses `aiohttp`. The component
listens on the configured address. The component
checks the allow list. The component applies the
header rules. The component forwards the request
through `net/http (stdlib)`. The component writes the audit
row.

## 7. Data Structures

### 7.1 EgressRequest

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | The unique identifier of the request. |
| `method` | string | yes | The HTTP method. |
| `url` | string | yes | The target URL. |
| `headers` | dict | yes | The request headers. |
| `body` | bytes | no | The request body. |

### 7.2 EgressResponse

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | The id of the request. |
| `status` | int | yes | The HTTP status code. |
| `headers` | dict | yes | The response headers. |
| `body` | bytes | yes | The response body. |
| `duration_ms` | int | yes | The duration in milliseconds. |

## 8. Interfaces

### 8.1 EgressProxy

```go
type EgressProxy:
    """The egress proxy."""

    def __init__(
        self,
        allow_list: AllowList,
        header_rules: HeaderRules,
        audit_log: AuditLog,
        listen: str = "127.0.0.1:8080",
    ) -> nil:
        """Create the egress proxy."""

    func start(self) -> nil:
        """Start the proxy."""

    func stop(self) -> nil:
        """Stop the proxy."""
```

## 9. Algorithms

### 9.1 Allow Check

The allow check algorithm works as follows.

1. Parse the URL to extract the host.
2. Match the host against the allow list.
3. When the host matches, return true.
4. When the host does not match, return false.

### 9.2 Header Rewriting

The header rewriting algorithm works as follows.

1. For each remove rule, remove the header.
2. For each add rule, add the header.
3. Return the modified headers.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- aiohttp
- net/http (stdlib)
- gopkg.in/yaml.v3
- aios-core

### 10.2 Module Dependencies

The aios-egress module depends on the following
internal modules.

- aios-core

The aios-egress module is depended on by the following
internal modules.

- aios-cli
- aios-models (when called through the proxy)

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/egress`. The module starts a
server on the listen address. The listen address is
configurable through the `AIOS_EGRESS_LISTEN`
environment variable.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The proxy leaks the API key. | High | Low | The proxy removes the `Authorization` header when the allow list is bypassed. The proxy logs the header. |
| The proxy is slow. | Medium | Low | The proxy uses `aiohttp`. The proxy uses a connection pool. |
| The proxy has a bypass. | High | Low | The proxy checks the allow list on every request. The proxy supports the deny list. |
