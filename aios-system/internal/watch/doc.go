// Package watch is the operator visibility layer for aios.
//
// The package has three components:
//
//   - StreamChannel: wraps the model router. Yields chunks to every
//     subscriber and to a Go-channel iterator. Closes on EOF or error.
//
//   - FileWatcherTool: a tools.Tool that polls a directory for
//     create/modify/delete events, filtering through the .aiosignore
//     patterns from the aios-search package. Writes a "file_watch"
//     audit row per call.
//
//   - TaskTool: a tools.Tool that spawns a sub-agent (one of
//     general/explore/plan/code) through the runtime.Worker interface.
//     Supports parallel calls. Writes a "subagent_task" audit row.
//
// The package depends on aios-core, aios-models, aios-runtime,
// aios-tools, and aios-search. The watcher has no third-party
// dependencies; it uses os.Stat polling (Bounded non-determinism,
// REQ-WTC-IFC-001).
package watch
