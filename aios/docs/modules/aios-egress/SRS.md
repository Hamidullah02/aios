# aios-egress — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-egress module.

## 2. Scope

The scope of this document is the aios-egress module.
The module is a library and a service.

## 3. Reference Documents

- aios-egress PRD. See `docs/modules/aios-egress/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Proxy | A typed server that forwards traffic. |
| Allow list | A typed list of allowed hosts. |

## 5. Functional Requirements

### 5.1 Proxy Server

**REQ-EGS-PRX-001** The module shall provide the
`EgressProxy` class. The type shall accept the
allow list, the audit log, and the listen address.

**REQ-EGS-PRX-002** The proxy shall accept HTTP
requests on the listen address.

**REQ-EGS-PRX-003** The proxy shall forward the
request to the target host when the host is in the
allow list.

**REQ-EGS-PRX-004** The proxy shall return a 403
response when the host is not in the allow list.

### 5.2 Allow List

**REQ-EGS-ALW-001** The module shall provide the
`AllowList` class. The type shall accept the list
of allowed hosts.

**REQ-EGS-ALW-002** The allow list shall support
exact match. The allow list shall support glob
match.

**REQ-EGS-ALW-003** The allow list shall load the
hosts from a YAML file. The default path is
`./var/egress.yaml`.

### 5.3 Header Rewriting

**REQ-EGS-HDR-001** The proxy shall support the
removal of headers. The proxy shall support the
addition of headers.

**REQ-EGS-HDR-002** The proxy shall support the
rewriting of the `Authorization` header. The proxy
shall support the rewriting of the `User-Agent`
header.

### 5.4 Audit Log Integration

**REQ-EGS-AUD-001** The proxy shall write an audit
row for every request. The row type shall be
`egress_request`. The row shall include the URL, the
method, the status code, and the duration.

## 6. Non-Functional Requirements

**REQ-EGS-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-EGS-NFR-002** The module shall have 100 percent
coverage for the public functions.

## 7. Interface Requirements

**REQ-EGS-IFC-001** The module shall export the
following items from `aios.egress`.

- `EgressProxy`
- `AllowList`
- `HeaderRule`

## 8. Data Requirements

**REQ-EGS-DAT-001** The egress file shall be a YAML
file. The structure is below.

```yaml
allow:
  - api.openai.com
  - api.anthropic.com
  - "*.example.com"
deny:
  - "*.evil.com"
headers:
  remove:
    - X-Internal-Token
  add:
    X-Aios-Request: "true"
```

## 9. Error Handling

**REQ-EGS-ERR-001** The module shall raise
`HostNotAllowedError` when the host is not in the
allow list.

**REQ-EGS-ERR-002** The module shall raise
`UpstreamError` when the upstream returns an error.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The proxy forwards the traffic to allowed hosts.
5. The proxy denies the traffic to disallowed hosts.
6. The proxy rewrites the headers.
7. The proxy writes the audit row.
