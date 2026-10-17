# aios-image — Implementation Plan

## 1. Purpose

This document states the implementation plan for the
aios-image module.

## 2. Scope

The scope of this document is the aios-image module.
The module depends on aios-core, aios-tools, and
aios-models.

## 3. Reference Documents

- aios-image PRD. See `docs/modules/aios-image/PRD.md`.
- aios-image SRS. See `docs/modules/aios-image/SRS.md`.
- aios-image TRD. See `docs/modules/aios-image/TRD.md`.
- aios-image Application Flow. See
  `docs/modules/aios-image/APP-FLOW.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Phase | A unit of work in the implementation plan. |

## 5. Phases

### Phase 1: Project Skeleton

1. Create the project folders.
2. Create the `go.mod` file.
3. Create the `README.md` file.

The deliverable is the project skeleton.

### Phase 2: Image Schema

1. Create the file `internal/image/schema.py`.
2. Create the file `tests/test_schema.py`.
3. Implement the `Image` class.
4. Write the test cases.

The deliverable is the schema with tests.

### Phase 3: Read Tool

1. Create the file `internal/image/read.py`.
2. Create the file `tests/test_read.py`.
3. Implement the `ReadImageTool` class.
4. Implement the size check.
5. Write the test cases.

The deliverable is the read tool with tests.

### Phase 4: Attach API

1. Create the file `internal/image/attach.py`.
2. Create the file `tests/test_attach.py`.
3. Implement the `ImageAttachment` class.
4. Implement the provider format methods.
5. Implement the resize.
6. Write the test cases.

The deliverable is the attach API with tests.

### Phase 5: Validator

1. Create the file `internal/image/validator.py`.
2. Create the file `tests/test_validator.py`.
3. Implement the `StructuredOutputValidator` class.
4. Implement the struct + go-playground/validator (or hand-rolled tag validation) integration.
5. Write the test cases.

The deliverable is the validator with tests.

### Phase 6: Constrained Completion

1. Create the file `internal/image/constrain.py`.
2. Create the file `tests/test_constrain.py`.
3. Implement the `SchemaConstrainedCompletion`
   function.
4. Implement the retry loop.
5. Write the test cases.

The deliverable is the constrained completion with
tests.

### Phase 7: Quality Gates

1. Run the test suite with `go test`.
2. Run the coverage report with `go test -cover`.
3. Run the type check with `go vet ./internal/image`.
4. Run the linter with `gofmt + go vet check`.
5. Fix every failure.

The deliverable is a green test suite.

## 6. Deliverables

- The source files in `internal/image/`.
- The test files in `tests/`.
- The `go.mod` file.
- The `README.md` file.

## 7. Test Plan

### 7.1 Unit Tests

- `tests/test_schema.py` covers
  `internal/image/schema.py`.
- `tests/test_read.py` covers
  `internal/image/read.py`.
- `tests/test_attach.py` covers
  `internal/image/attach.py`.
- `tests/test_validator.py` covers
  `internal/image/validator.py`.
- `tests/test_constrain.py` covers
  `internal/image/constrain.py`.

### 7.2 Test Cases

The test cases include the following items.

- The schema validates.
- The read tool reads a PNG.
- The read tool reads a JPEG.
- The read tool rejects an oversized file.
- The attach API returns the Anthropic block.
- The attach API returns the OpenAI block.
- The attach API resizes the image.
- The validator validates the response.
- The validator supports structs.
- The constrained completion returns the
  validated object.
- The constrained completion retries on failure.

## 8. Merge Plan

### 8.1 Pre-Merge Checklist

1. The test suite has a 100 percent pass rate.
2. The coverage report shows 100 percent coverage
   for the public functions.
3. The `go vet` report shows no errors.
4. The `gofmt + go vet` report shows no errors.

### 8.2 Merge Steps

1. Create a branch named `merge/aios-image`.
2. Move the files from `modules/aios-image/` to
   the target locations in the main repository.
3. Run the test suite on the main repository.
4. Open a pull request.
5. Merge after the review is approved.

## 9. Risk Register

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The image is too large. | High | Medium | The tool enforces a maximum size. The attach API supports resize. |
| The structured output fails validation. | High | Medium | The function retries on failure. |
| The provider changes the image format. | Medium | Low | The attach API is versioned. |

## 10. Schedule

| Phase | Duration | Dependency |
| --- | --- | --- |
| Phase 1: Project Skeleton | 1 day | nil |
| Phase 2: Image Schema | 1 day | Phase 1 |
| Phase 3: Read Tool | 1 day | Phase 2 |
| Phase 4: Attach API | 2 days | Phase 2 |
| Phase 5: Validator | 2 days | Phase 1 |
| Phase 6: Constrained Completion | 2 days | Phase 4, Phase 5 |
| Phase 7: Quality Gates | 1 day | All phases |

The total duration is 10 days.
