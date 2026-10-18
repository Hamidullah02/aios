// Package report is the report generator for aios.
//
// The package turns a Report into a string in one of
// three formats: JSON, Markdown, or HTML.
//
// The package provides four things:
//
//   - Report and Step: the typed data contracts. The
//     report describes one run of a plan. The step
//     describes one step in the run.
//   - Renderer: the renderer interface every concrete
//     renderer implements.
//   - JSONRenderer: the deterministic JSON renderer.
//     The renderer sorts the keys, indents by 2 spaces,
//     and produces a stable hash for a given report.
//   - MarkdownRenderer: the human-readable Markdown
//     renderer. The renderer builds a header, a steps
//     table, and an approvals list.
//   - HTMLRenderer: the HTML5 renderer. The renderer
//     loads the bundled report.html template and
//     renders the report.
//
// The package depends only on the Go standard library
// and the aios-core schemas for canonical-JSON encoding.
package report
