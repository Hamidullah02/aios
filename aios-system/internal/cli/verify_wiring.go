package cli

import (
	"context"
	"fmt"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/models"
	"github.com/Hamidullah02/aios/aios-system/internal/sandbox"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
	"github.com/Hamidullah02/aios/aios-system/internal/verify"
)

// wireVerifyTools constructs the verify-package tools
// (SelfCheckTool, RunTestsTool) and returns a map keyed
// by the canonical tool names the bus uses:
//
//   - "run_tests":  verify.RunTestsTool. Always wired.
//   - "self_check": verify.SelfCheckTool. Only wired
//     when a router is available (the Judge adapter
//     needs one).
//
// The function is split out of wireAgentLoop because
// it is conditional: SelfCheckTool needs a router, and
// only the `fix` subcommand builds one today (the
// planner turn). A `run` subcommand that wants to use
// self_check would need a router too; that is a v0.36.0
// concern (move router construction into
// wireAgentLoop).
//
// RunTestsTool is always wired because it has no model
// dependency — it is a pure subprocess runner plus a
// parser. The policy gateway can still deny it
// (default-deny on a fresh workspace), and the user
// can grant it via a policy rule.
//
// The audit log is wired into both tools so they
// write the doc-set's required rows
// (REQ-VRF-SCK-006, REQ-VRF-RNT-006).
func wireVerifyTools(router *models.Router, auditLog *audit.Log) map[string]tools.Tool {
	out := make(map[string]tools.Tool)
	// RunTestsTool: always wired. The default runner
	// is go-test because the doc set's most common
	// case is a Go project; the tool accepts an
	// explicit `runner` arg to override.
	//
	// The sandbox is the LAZY default runner, not a fresh
	// one. A fresh runner has no network policy, and the
	// egress guard installs its policy on the process-wide
	// default runner AFTER the tool set is built. Capturing
	// the default here would capture the policy-less runner
	// that existed before the guard started, so a test run
	// would download its dependencies straight past the
	// allow list.
	runTests := &verify.RunTestsTool{
		Sandbox:       sandbox.LazyDefaultRunner{},
		DefaultRunner: verify.RunnerGoTest,
		Audit:         auditLog,
	}
	out["run_tests"] = runTests
	// SelfCheckTool: only build when a router is
	// available. The Judge adapter picks a model id
	// (the same one the planner uses) and calls
	// router.Complete.
	if router != nil {
		selfCheck, err := verify.NewSelfCheckTool(verify.SelfCheckConfig{
			Judge: routerJudge{router: router, model: resolveModel("")},
			Audit: auditLog,
		})
		if err != nil {
			// SelfCheckTool init failed: this is a
			// system error (a misconfigured router or
			// a broken default prompt). We skip the
			// tool; the bus will return
			// ToolNotFoundError if a plan asks for
			// it. The CLI does not fail here because
			// the user may not have any self_check
			// tasks in the plan.
			_ = err
		} else {
			out["self_check"] = selfCheck
		}
	}
	return out
}

// routerJudge adapts a *models.Router to the
// verify.Judge interface. The adapter picks a model
// id (configurable; the resolved default) and calls
// router.Complete, returning the response text.
//
// The adapter lives here (not in verify) so verify
// does not import aios-models. The doc set's verify
// TRD §8.1 names ModelRouter as the dependency; we
// satisfy it through the Judge interface exactly as
// the verify package's doc.go describes.
type routerJudge struct {
	router *models.Router
	model  string
}

// wireSelfCheckOnly returns the SelfCheckTool alone
// (no RunTestsTool). The function exists for the
// deferred-wiring case: RunTestsTool is built in
// wireAgentLoop (no model needed), but SelfCheckTool
// needs a router that is only available after
// wirePlanner. The CLI calls this function once the
// router is built and registers the tool into the bus
// via the bus's Register method.
//
// The function returns nil if the router is nil or
// if SelfCheckTool construction fails; the caller
// then skips the registration and the bus returns
// ToolNotFoundError if a plan asks for self_check.
func wireSelfCheckOnly(router *models.Router, auditLog *audit.Log) tools.Tool {
	if router == nil {
		return nil
	}
	selfCheck, err := verify.NewSelfCheckTool(verify.SelfCheckConfig{
		Judge: routerJudge{router: router, model: resolveModel("")},
		Audit: auditLog,
	})
	if err != nil {
		return nil
	}
	return selfCheck
}

// Judge calls router.Complete and returns the response
// text. The request uses temperature 0 (determinism
// stance; PRD §5) and no system prompt (the self-check
// prompt is fully self-contained in the verify
// package's default template).
func (j routerJudge) Judge(ctx context.Context, prompt string) (string, error) {
	if j.router == nil {
		return "", fmt.Errorf("verify: no router wired")
	}
	resp, err := j.router.Complete(ctx, models.Request{
		Model:  j.model,
		Prompt: prompt,
		Options: models.Options{
			Temperature: 0,
		},
	})
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}
