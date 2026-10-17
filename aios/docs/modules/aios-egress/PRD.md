# aios-egress — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-egress module. The module is the egress proxy.
The proxy controls the network traffic from the
agent.

## 2. Scope

The aios-egress module is in scope for this document.

- The proxy server.
- The host allow list.
- The header rewriting.
- The audit log integration.

The aios-egress module is not in scope for this
document.

- The model adapter. The adapter is in aios-models.
- The shell tool. The tool is in aios-tools.

## 3. Reference Documents

- Project design document, section 7.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Egress | The network traffic that leaves the host. |
| Proxy | A server that forwards network traffic. |
| Allow list | A list of approved hosts. |
| Header rewriting | The replacement of headers in the request or the response. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Bounded non-determinism**.

Network is the only source of non-determinism. The proxy is a pure function of the policy and the request.

The module provides the following determinism guarantees.

- Proxy decisions are pure functions of the policy and the request.

- Policy is content addressed.
## 6. User Personas

The user persona is a Go developer who needs to
control the network traffic.

## 7. User Stories

A developer wants to allow a specific host. The
developer adds the host to the allow list. The proxy
forwards the traffic to the host.

A developer wants to deny a specific host. The
developer omits the host from the allow list. The
proxy denies the traffic.

A developer wants to rewrite a header. The developer
adds a rule to the proxy. The proxy rewrites the
header.

## 8. Functional Requirements Summary

The aios-egress module provides the following
functions.

- The proxy server.
- The host allow list.
- The header rewriting.
- The audit log integration.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core, net/http (stdlib), and the
  Go standard library.
- The module has 100 percent test coverage for the
  public functions.

## 10. Success Criteria

- The proxy forwards the traffic to allowed hosts.
- The proxy denies the traffic to disallowed hosts.
- The proxy rewrites the headers.
- The proxy writes the audit row.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The user interface for the proxy.
- The remote access to the proxy.
- The TLS termination.
