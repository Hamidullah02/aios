// Package schemas contains the typed data contracts for cross-module records.
//
// Every record type implements Schema (Validate) and the standard
// json.Marshaler / json.Unmarshaler interfaces. The encoder is canonical:
// object keys are sorted at every level so that two equal values produce
// the same bytes. The decoder is strict: it rejects unknown fields.
//
// The records here are:
//   - TaskPlan, TaskSpec, Budget (planning inputs)
//   - AgentSpec (agent configuration)
//   - WorkflowSpec, WorkflowStep, WorkflowTransition (workflow graphs)
//   - ScheduleSpec, ScheduleTrigger, ScheduleLimits (recurring jobs)
//   - DecisionTrace, Alternative (per-step reasoning)
//   - Envelope (cross-module carrier)
//   - ApprovalPayload (pending approval body)
//   - ModelCapability (model advertisement)
//   - ToolCall, ToolResult (tool I/O)
package schemas
