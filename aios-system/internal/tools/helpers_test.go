package tools

import "github.com/sadi/aios-system/internal/sandbox"

// recordingRunner records the RunOptions it was given.
type recordingRunner struct {
	onRun func(opts sandbox.RunOptions)
}

func (r *recordingRunner) Run(opts sandbox.RunOptions) (sandbox.SandboxResult, error) {
	if r.onRun != nil {
		r.onRun(opts)
	}
	return sandbox.SandboxResult{ExitCode: 0, Stdout: ""}, nil
}
