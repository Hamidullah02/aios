# aios-sandbox — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-sandbox module. The module runs untrusted code
in a safe environment. The module prevents the code
from accessing the host system.

## 2. Scope

The aios-sandbox module is in scope for this document.

- The os/exec wrapper.
- The resource limits.
- The path isolation.

The aios-sandbox module is not in scope for this
document.

- The egress proxy. The proxy is in aios-egress.
- The shell tool. The tool is in aios-tools.

## 3. Reference Documents

- Project design document, section 7.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Sandbox | An environment that limits the access of a process. |
| Runner | A component that starts a process in the sandbox. |
| Resource limit | A maximum value for CPU, memory, or time. |
| Path isolation | A restriction on the paths the process can access. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Bounded non-determinism**.

Wall clock and process scheduling are the only sources of non-determinism. The sandbox is content addressed at the file level.

The module provides the following determinism guarantees.

- Sandbox creation is reproducible given the same image and config.

- Resource limits are deterministic for a fixed clock.
## 6. User Personas

The user persona is a Go developer who needs to
run untrusted code.

## 7. User Stories

A developer wants to run a shell command. The
developer passes the command to the sandbox. The
sandbox runs the command. The sandbox returns the
result.

A developer wants to set resource limits. The
developer passes the limits to the sandbox. The
sandbox enforces the limits.

A developer wants to isolate the file system. The
developer passes the workspace to the sandbox. The
sandbox limits the access to the workspace.

## 8. Functional Requirements Summary

The aios-sandbox module provides the following
functions.

- The os/exec wrapper.
- The resource limits.
- The path isolation.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends only on aios-core and the Go
  standard library.
- The module has 100 percent test coverage for the
  public functions.

## 10. Success Criteria

- The runner runs a command.
- The resource limits are enforced.
- The path isolation is enforced.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The user interface for the sandbox.
- The remote access to the sandbox.
- The gVisor runtime. The runtime is a future
  enhancement.
