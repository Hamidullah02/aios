# aios-roles — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-roles module.

## 2. Scope

The scope of this document is the aios-roles module.
The module is a library.

## 3. Reference Documents

- aios-roles PRD. See `docs/modules/aios-roles/PRD.md`.
- aios-roles SRS. See `docs/modules/aios-roles/SRS.md`.
- aios-roles TRD. See `docs/modules/aios-roles/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Caller | The code that calls the resolver. |

## 5. User Flow

The aios-roles module has no user-facing flow.

## 6. System Flow

### 6.1 Resolve Flow

The resolve flow runs as follows.

1. The caller calls `resolver.resolve` with the
   name.
2. The resolver looks up the name in the
   registry.
3. When the name is found, the resolver returns
   the role.
4. When the name is not found and the fallback is
   set, the resolver returns the fallback.
5. When the name is not found and the fallback is
   not set, the resolver raises
   `UnknownRoleError`.

### 6.2 Load Flow

The load flow runs as follows.

1. The caller calls `registry.load` with the path.
2. The registry reads the YAML file.
3. The registry validates the roles through the
   schema.
4. The registry updates the internal state.

## 7. Error Flows

### 7.1 Unknown Role

The unknown role flow runs as follows.

1. The resolver does not find the name.
2. The resolver raises `UnknownRoleError`.
3. The caller catches the error.
4. The caller reports the error.

### 7.2 Invalid Role

The invalid role flow runs as follows.

1. The registry loads an invalid role.
2. The schema raises `InvalidRoleError`.
3. The caller catches the error.
4. The caller reports the error.

## 8. Sequence Diagrams

### 8.1 Resolve Sequence

The resolve sequence has the steps below.

1. The caller calls `resolver.resolve`.
2. The resolver looks up the name.
3. The resolver returns the role.
