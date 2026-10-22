package runtime

// recordTaskVars publishes a completed task's structured
// output into the plan's vars bag so a later task can
// reference it.
//
// The bag is keyed two ways:
//
//   - "<taskID>" holds the task's Data map, so
//     `$vars.t1.hash` reaches Data["hash"].
//   - "last" holds the same map, for the common case where
//     a plan chains two tasks and the second only needs the
//     first's most recent output.
//
// "last" is a convenience with a sharp edge: it is
// overwritten by every task that produces output. In a
// long plan a reference to `$vars.last` therefore means
// "whatever ran most recently", not "the one I meant". A
// plan that needs a specific task's output must name the
// task. The convenience is kept because the two-task chain
// is by far the most common shape and naming the first task
// adds no information there.
//
// An empty task id or an empty Data map is a no-op: there
// is nothing a later task could usefully reference, and
// writing an empty map would make `$vars.t1` resolve to an
// empty string rather than fail to resolve.
//
// The function is a method on *TaskContext rather than a
// free function so the nil check stays in one place. The
// orchestrator always passes a non-nil context (it creates
// one, then hydrates it from the checkpoint).
func (c *TaskContext) recordTaskVars(taskID string, data map[string]any) bool {
	if c == nil {
		return false
	}
	if c.Vars == nil {
		c.Vars = map[string]any{}
	}
	if taskID == "" || len(data) == 0 {
		return false
	}
	c.Vars[taskID] = data
	c.Vars["last"] = data
	return true
}
