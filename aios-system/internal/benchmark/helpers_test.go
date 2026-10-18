package benchmark

import "time"

// fakeRunner is a deterministic Runner for tests.
// The Responses map returns the desired Result for a
// given task id; the Default is returned for any other
// task. The Sleep field adds a per-call delay so
// timeout tests work.
type fakeRunner struct {
	Responses map[string]Result
	Default   Result
	Sleep     time.Duration
	Calls     int
}

func (f *fakeRunner) Run(task WedgeTask) Result {
	f.Calls++
	if f.Sleep > 0 {
		time.Sleep(f.Sleep)
	}
	if r, ok := f.Responses[task.ID]; ok {
		return r
	}
	return f.Default
}
