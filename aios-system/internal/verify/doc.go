// Package verify is the self-evaluation layer for
// aios.
//
// The package provides four things:
//
//   - Result: the typed result schemas the
//     SelfCheck tool and the RunTests tool
//     produce. VerificationResult is the
//     model-judgement outcome; TestResult is
//     the test-runner outcome.
//   - Prompt: the verification prompt builder
//     (text/template). The default prompt is
//     the doc-set REQ-VRF-DAT-003 template;
//     callers may override it.
//   - SelfCheckTool: the tools.Tool that asks
//     the model to verify the work and parses
//     the response into a VerificationResult.
//   - RunTestsTool: the tools.Tool that runs
//     the test suite and parses the output
//     into a TestResult. Supports the
//     per-runner parsers the doc set names
//     (go test, jest, mocha, cargo, npm).
//
// The package depends on aios-core (for the
// audit log and the canonical-JSON
// encoders), aios-tools (for the Tool
// interface), aios-sandbox (for the
// subprocess runner), and the Go standard
// library. No new third-party dependencies.
//
// Per the doc set the package also depends on
// aios-models. The implementation does not
// import aios-models: the SelfCheck tool's
// "model call" is the call site's problem
// through a Judge interface. The Judge
// interface is the same shape as the
// aios-fanout Summarizer interface; a real
// aios-models Router satisfies it.
package verify
