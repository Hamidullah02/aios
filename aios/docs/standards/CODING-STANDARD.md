# Coding Standard

## Purpose

This document states the coding standard for all source code
in the project. The standard covers Go code, test code,
and configuration files.

The coding standard is the project equivalent of ASD-STE100
for prose. STE is a controlled vocabulary for technical
documentation. The coding standard is a controlled style for
program code.

## Scope

The standard applies to all source code in:

- `internal/`
- `tests/`
- `benchmark/`
- `docs/examples/`
- Build and configuration scripts

The standard does not apply to:

- Generated code
- Third-party code in `site-packages`
- Vendor SDK code

## Language

The project uses Go 1.23 as the primary language.
Type hints are required on all public functions and
methods.

## File Layout

### Module File

A module file has the parts in this order.

1. The module docstring. The docstring states the purpose
   of the module in one or two sentences.
2. The `from __future__ import annotations` line. This
   line allows forward references in type hints.
3. The standard library imports.
4. The third-party imports.
5. The local imports.
6. The module-level constants.
7. The module-level functions and classes.

### Test File

A test file has the parts in this order.

1. The test module docstring.
2. The standard library imports.
3. The third-party imports.
4. The local imports.
5. The fixtures.
6. The test functions, grouped by feature.

## Naming

The project uses these rules for identifiers.

### Variables and Functions

- Use snake_case.
- Use a name that describes the value or the action.
- Do not use abbreviations except for standard ones.
  Standard abbreviations: id, max, min, url, http, api.
- Do not use single-letter names except for loop counters
  and for mathematical formulas.

### Classes

- Use PascalCase.
- Use a noun or a noun phrase.
- The type name does not include the type of the class.
  Example: `Orchestrator` is preferred over `OrchestratorClass`.

### Constants

- Use UPPER_SNAKE_CASE.
- A constant is a module-level value that does not change.

### Private Members

- Prefix with a single underscore.
- Example: `_internal_state`, `_validate_input`.

### Type Variables

- Use PascalCase.
- Use a short name that describes the role.
  Example: `T` for a generic type, `R` for a return type.

## Type Hints

- Type hints are required on public functions and methods.
- Type hints are optional on private functions and methods.
- Use `from __future__ import annotations` to allow forward
  references.
- Use the standard collection types from `typing` for
  Go 1.23 compatibility: `list`, `dict`, `tuple`, `set`.
- Use `*T]` for a value that can be `nil`.
- Use `Union[A, B]` for a value that can be one of two
  types. Use `A | B` only when the project supports
  Go 1.23 and later.

## Functions

- A function does one thing. The name describes the thing.
- A function has a maximum of 50 lines of code.
- A function has a maximum of 5 parameters. Use a parameter
  object when more parameters are needed.
- A function does not use global variables.
- A function does not have side effects unless the name
  describes the side effect. Example: `set_state` is
  allowed. `get_state` is not allowed to change state.

## Classes

- A type has one responsibility.
- A type has a maximum of 500 lines of code.
- A type has a maximum of 20 public methods.
- Use composition over inheritance.
- A type does not inherit from more than one parent class.

## Comments

- A comment explains why, not what.
- A comment is one line or two lines. Longer explanations
  go in a docstring.
- A comment uses the imperative form or the simple
  present tense.
- A comment does not use the future tense.
- A comment does not repeat the code in plain English.

## Docstrings

- A module has a docstring.
- A public type has a docstring.
- A public function or method has a docstring.
- A private function or method has a docstring only when
  the purpose is not clear from the name.
- Use the project docstring format:

```
"""One-line summary.

Optional longer description. The longer description
states the purpose, the inputs, the outputs, and the
errors.

Args:
    param_name: Description.

Returns:
    Description of the return value.
"""
```

## Errors

- Use specific exception types. Do not raise `Exception`.
- The project defines a base exception type per package.
  Example: `aios.policy.PolicyError`.
- The error message starts with a lowercase letter and
  ends with a period.
- The error message includes the value that caused the
  error when the value is safe to include.

## Tests

- A test file mirrors the source file it tests.
  Example: `tests/test_orchestrator.py` tests
  `internal/runtime/orchestrator.py`.
- A test function name starts with `test_`.
- A test function name describes the behavior it tests.
- A test uses one assertion or a small number of related
  assertions.
- A test does not depend on the order of execution.
- A test does not depend on the state of another test.
- A test uses fixtures for shared setup.

## Imports

- Import the module, not the function. Example:
  `from aios.policy import PolicyGateway`.
- Do not use `from module import *`.
- Group imports in this order: standard library,
  third-party, local. Separate the groups with a blank
  line.
- Sort each group alphabetically.

## Formatting

- Use 4 spaces for each level of indentation.
- Use a maximum line length of 100 characters.
- Use one blank line between functions in a class.
- Use two blank lines between top-level definitions.

## Logging

- Use the standard `logging` module.
- The logger name is the module name.
- The log level is `INFO` for normal events, `WARNING`
  for recoverable errors, `ERROR` for non-recoverable
  errors.
- The log message starts with a lowercase letter.

## Audit and Trace Records

- The project writes audit and trace records through the
  audit log module. The audit log is the only place that
  writes these records.
- A function does not write an audit row directly.
  The function calls a writer from the audit log module.
- The audit row payload is a JSON-serializable dict.
  The payload does not include secrets.

## Review Checklist

The project reviews every change against this checklist.

- [ ] The code follows the file layout.
- [ ] The naming rules are applied.
- [ ] The type hints are present on public functions and
      methods.
- [ ] Each function does one thing.
- [ ] Each type has one responsibility.
- [ ] Comments explain why, not what.
- [ ] Docstrings follow the project format.
- [ ] Specific exception types are used.
- [ ] The test file mirrors the source file.
- [ ] The test function names start with `test_`.
- [ ] The tests are independent.
- [ ] The imports are sorted and grouped.
- [ ] The formatting rules are applied.
- [ ] The audit records go through the audit log module.
