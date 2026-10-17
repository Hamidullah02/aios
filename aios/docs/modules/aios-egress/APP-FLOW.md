# aios-egress — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-egress module.

## 2. Scope

The scope of this document is the aios-egress module.
The module is a service.

## 3. Reference Documents

- aios-egress PRD. See `docs/modules/aios-egress/PRD.md`.
- aios-egress SRS. See `docs/modules/aios-egress/SRS.md`.
- aios-egress TRD. See `docs/modules/aios-egress/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Client | The agent or tool that makes the request. |
| Upstream | The remote service that receives the request. |

## 5. User Flow

The aios-egress module has no user-facing flow. The
proxy is transparent to the user.

## 6. System Flow

### 6.1 Request Flow

The request flow runs as follows.

1. The client sends a request to the proxy.
2. The proxy parses the URL.
3. The proxy checks the allow list.
4. When the host is allowed, the proxy applies the
   header rules.
5. The proxy forwards the request to the upstream.
6. The upstream returns the response.
7. The proxy applies the header rules to the
   response.
8. The proxy returns the response to the client.
9. The proxy writes the audit row.

### 6.2 Deny Flow

The deny flow runs as follows.

1. The client sends a request to the proxy.
2. The proxy parses the URL.
3. The proxy checks the allow list.
4. When the host is not allowed, the proxy returns
   a 403 response.
5. The proxy writes the audit row.

## 7. Error Flows

### 7.1 Upstream Error

The upstream error flow runs as follows.

1. The upstream returns an error.
2. The proxy returns the error to the client.
3. The proxy writes the audit row.

### 7.2 Host Not Allowed

The host not allowed flow runs as follows.

1. The client sends a request to a denied host.
2. The proxy returns a 403 response.
3. The proxy writes the audit row.

## 8. Sequence Diagrams

### 8.1 Request Sequence

The request sequence has the steps below.

1. The client sends the request.
2. The proxy checks the allow list.
3. The proxy applies the header rules.
4. The proxy forwards the request.
5. The upstream returns the response.
6. The proxy applies the header rules.
7. The proxy writes the audit row.
8. The proxy returns the response.
