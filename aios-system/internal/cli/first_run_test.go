package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/sadi/aios-system/internal/policy"
	"github.com/sadi/aios-system/internal/tools"
)

// repoRoot is the directory holding the repository's root files.
//
// The tests that read install.sh and install.ps1 need to find
// them, and they live above the module directory: the git
// repository root is the PARENT of this module.
//
// The helper used to step up exactly two levels, which landed
// on the module root (aios-system/) and not on the repository
// root, so the installer tests failed with "cannot find the
// repository root" on a tree where the installers exist. The
// walk up is bounded, and it stops at the first directory that
// actually has install.sh in it.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// The test runs in internal/cli.
	for i := 0; i < 5; i++ {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
		if _, statErr := os.Stat(filepath.Join(dir, "install.sh")); statErr == nil {
			return dir
		}
	}
	t.Fatalf("cannot find the repository root (no install.sh above %s)", dir)
	return ""
}

// TestInstallers_PointAtTheRealRepo pins the download URL.
//
// The installers defaulted to a repo that was never the one the
// code was pushed to, so `curl ... | sh` downloaded a 404 and
// the operator's first experience of the product was a failure
// with no cause they could see.
//
// The Go module path is deliberately NOT the same string: the
// module is github.com/sadi/aios-system, and renaming it would
// touch every import in the tree for no benefit to an operator
// installing a binary. So this test pins the two separately: the
// URLs must name the GitHub repo, and `go install` must name the
// module.
func TestInstallers_PointAtTheRealRepo(t *testing.T) {
	root := repoRoot(t)

	const (
		githubRepo = "Hamidullah02/aios"
		goModule   = "github.com/sadi/aios-system"
	)

	for _, name := range []string{"install.sh", "install.ps1", ".github/workflows/release.yml", "README.md"} {
		path := filepath.Join(root, filepath.FromSlash(name))
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		text := string(body)

		// The old repo must not appear anywhere as a
		// download target.
		if strings.Contains(text, "raw.githubusercontent.com/sadi/aios-system") {
			t.Errorf("%s: still points the raw download at sadi/aios-system", name)
		}
		if strings.Contains(text, "github.com/sadi/aios-system/releases") {
			t.Errorf("%s: still points the release download at sadi/aios-system", name)
		}

		// The GitHub repo must appear where a download
		// happens.
		if name != "README.md" && !strings.Contains(text, githubRepo) {
			t.Errorf("%s: does not mention the release repo %q", name, githubRepo)
		}
		// The module path must appear where a `go install`
		// happens.
		if strings.Contains(text, "go install") && !strings.Contains(text, goModule) {
			t.Errorf("%s: `go install` does not use the module path %q", name, goModule)
		}
	}
}

// TestInstallers_AssetNamesMatchTheWorkflow pins that the
// installers ask for assets the release workflow actually
// builds.
//
// The installers and the workflow named the assets separately.
// A rename on one side and not the other produces an installer
// that downloads a 404 for every platform, and the failure looks
// like "the release has no build for your machine".
func TestInstallers_AssetNamesMatchTheWorkflow(t *testing.T) {
	root := repoRoot(t)

	workflow, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	shell, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	ps, err := os.ReadFile(filepath.Join(root, "install.ps1"))
	if err != nil {
		t.Fatal(err)
	}

	// The platforms the workflow builds.
	platforms := []string{
		"aios_linux_amd64",
		"aios_linux_arm64",
		"aios_darwin_amd64",
		"aios_darwin_arm64",
		"aios_windows_amd64",
	}
	for _, asset := range platforms {
		if !strings.Contains(string(workflow), asset) {
			t.Errorf("the workflow does not build %q", asset)
		}
	}

	// The installers must derive the asset name from the same
	// pattern the workflow uses.
	if !strings.Contains(string(shell), "aios_${os}_${arch}") {
		t.Error("install.sh does not use the aios_<os>_<arch> asset pattern")
	}
	if !strings.Contains(string(ps), "aios_windows_$cpu") {
		t.Error("install.ps1 does not use the aios_windows_<cpu> asset pattern")
	}
}

// TestFirstRunPolicy_CoversEveryRegisteredTool pins that the
// first-run policy names every tool the agent can dispatch.
//
// The hand-written policy this replaced named "apply_patch",
// which is not a registered tool, and missed nine that are --
// so on a workspace that had never been configured the agent
// could not read a notebook, run the tests, or commit a change.
// The operator got "denied by policy" for a tool the product
// advertised in `aios tools`.
//
// The policy is generated from the tool set now, so this test
// checks the generation rather than a fixed list: every
// registered tool has a rule, and no rule names a tool that does
// not exist.
func TestFirstRunPolicy_CoversEveryRegisteredTool(t *testing.T) {
	toolSet := productionToolSet(nil, nil)
	policy := firstRunPolicyYAMLFor(toolSet)

	for name := range toolSet {
		if !strings.Contains(policy, "tool: "+name+"\n") {
			t.Errorf("the first-run policy has no rule for the registered tool %q", name)
		}
	}
	// No rule may name a tool that does not exist. The
	// hand-written list did: "apply_patch" was in it and is not
	// a registered tool.
	for _, name := range policyToolNames(policy) {
		if _, ok := toolSet[name]; !ok {
			t.Errorf("the first-run policy names %q, which is not a registered tool", name)
		}
	}
}

// TestFirstRunPolicy_GatesSubprocessTools pins that the tools
// that run a subprocess are gated behind an approval.
//
// A tier-2 tool runs a command. On a workspace that has never
// been configured, that command has to be something the operator
// saw and agreed to -- otherwise "write a file" turns into "run
// whatever the model felt like", which is the thing the policy
// exists to prevent.
func TestFirstRunPolicy_GatesSubprocessTools(t *testing.T) {
	toolSet := productionToolSet(nil, nil)
	policy := firstRunPolicyYAMLFor(toolSet)

	for name, tool := range toolSet {
		if tools.Tier(tool) < 2 {
			continue
		}
		// The rule for this tool must be an "ask", and it must
		// carry an approval block so the operator's browser
		// shows what is about to run.
		rule := policyRuleFor(policy, name)
		if rule == "" {
			t.Errorf("the first-run policy has no rule for the subprocess tool %q", name)
			continue
		}
		if !strings.Contains(rule, "action: ask") {
			t.Errorf("the rule for %q is not gated: %s", name, rule)
		}
		if !strings.Contains(rule, "approval:") {
			t.Errorf("the rule for %q has no approval block: %s", name, rule)
		}
	}
}

// TestFirstRunPolicy_ApprovalTemplatesResolve pins that every
// gated tool's approval template names an argument the tool
// actually has.
//
// This is the bug the generated policy fixed. The template used
// to name "{arg.command}" for every gated tool, but git_commit
// takes "message" and diagnostics takes "checker". A template
// naming an argument the call does not carry is a hard error in
// the policy gateway, so the rule failed with "approval.resource:
// template: arg.command: not in args" -- the operator's approval
// was dead and the tool could never run at all.
//
// The test loads the generated policy, decides a call for every
// tier-2 tool with its own arguments, and asserts the gateway
// does not error.
func TestFirstRunPolicy_ApprovalTemplatesResolve(t *testing.T) {
	toolSet := productionToolSet(nil, nil)
	policyText := firstRunPolicyYAMLFor(toolSet)

	// Load it the way the product does.
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(path, []byte(policyText), 0o644); err != nil {
		t.Fatal(err)
	}
	rules, err := policy.Load(path)
	if err != nil {
		t.Fatalf("the generated policy does not load: %v", err)
	}
	gw := policy.New(rules, nil)

	for name, tool := range toolSet {
		if tools.Tier(tool) < 2 {
			continue
		}
		// Build a call carrying the tool's own arguments.
		args := map[string]any{}
		for _, arg := range schemaArgNames(tool) {
			args[arg] = "x"
		}
		decision, err := gw.Decide(policy.NewCall(name, args, "actor", "task-1"))
		if err != nil {
			t.Errorf("the approval rule for %q does not resolve against its own arguments: %v", name, err)
			continue
		}
		if decision == nil || decision.Action != policy.ActionAsk {
			t.Errorf("the rule for %q is not gated: %+v", name, decision)
		}
	}
}

// policyToolNames returns the tool names a policy names.
func policyToolNames(policy string) []string {
	var out []string
	for _, m := range regexp.MustCompile(`tool:\s*([a-z_]+)`).FindAllStringSubmatch(policy, -1) {
		out = append(out, m[1])
	}
	return out
}

// policyRuleFor returns the rule block that names a tool.
func policyRuleFor(policy, tool string) string {
	blocks := strings.Split(policy, "  - id: ")
	for _, b := range blocks[1:] {
		if strings.Contains(b, "tool: "+tool+"\n") {
			return b
		}
	}
	return ""
}

// TestFirstRunPolicy_AllowsFileWriting pins that the first-run
// policy lets the agent write files.
//
// The `aios init` default allows only read-only tools, which is
// right for a tool an expert configures and wrong for the first
// run of a tool whose job is to do what the operator asked. An
// operator who said "write the word hello into a file called
// hi.txt" got a run that failed with "write denied by policy:
// no rule matched; default deny" -- the tool refusing the exact
// thing it was asked to do, with nothing in the message about
// how to fix it.
//
// The first-run policy is a different file from the init
// default, so this test is the only thing keeping the two from
// being made the same by accident.
func TestFirstRunPolicy_AllowsFileWriting(t *testing.T) {
	dir := t.TempDir()
	workspace, stateDir, created, err := ensureWorkspaceReady(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) == 0 {
		t.Fatal("the first run created no files")
	}
	// The workspace is the operator's own directory, so a file
	// the agent writes lands next to them.
	if filepath.Base(workspace) != filepath.Base(dir) {
		t.Errorf("workspace = %q, want the operator's directory %q", workspace, dir)
	}
	// The tool's own files live in var/, not in the operator's
	// project folder.
	if filepath.Base(stateDir) != "var" {
		t.Errorf("state dir = %q, want a var/ subdirectory", stateDir)
	}

	body, err := os.ReadFile(filepath.Join(stateDir, "policy.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	policy := string(body)
	for _, tool := range []string{"write_file", "edit", "apply_patch"} {
		if !strings.Contains(policy, "tool: "+tool) {
			t.Errorf("the first-run policy does not allow %q", tool)
		}
	}
	// Shell access must still be gated. It is the part that can
	// do anything the process can do.
	if !strings.Contains(policy, "tool: run_shell") {
		t.Error("the first-run policy does not mention run_shell at all")
	}
	if !strings.Contains(policy, "action: ask") {
		t.Error("the first-run policy does not gate run_shell behind an approval")
	}
}

// TestFirstRun_IsIdempotent pins that a second run does not
// overwrite what the operator has changed.
//
// An operator who tunes their policy and restarts the tool must
// find their policy intact. A bootstrap that rewrites the files
// every time would silently undo their work.
func TestFirstRun_IsIdempotent(t *testing.T) {
	dir := t.TempDir()
	if _, _, _, err := ensureWorkspaceReady(dir); err != nil {
		t.Fatal(err)
	}

	// The operator edits the policy.
	policyPath := filepath.Join(dir, "var", "policy.yaml")
	edited := "default: deny\nrules: []\n"
	if err := os.WriteFile(policyPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, created, err := ensureWorkspaceReady(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 0 {
		t.Errorf("the second run created %v; it should have created nothing", created)
	}

	got, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != edited {
		t.Errorf("the operator's policy was overwritten:\n%s", got)
	}
}

// TestFirstRun_EgressAllowsTheModelProvider pins that the
// first-run egress list includes the model provider.
//
// Without it the tool cannot call the model at all, which is the
// same as not working -- and the failure is a network error
// rather than a clear message about the allow list.
func TestFirstRun_EgressAllowsTheModelProvider(t *testing.T) {
	dir := t.TempDir()
	_, stateDir, _, err := ensureWorkspaceReady(dir)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(stateDir, "egress.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "openrouter.ai") {
		t.Errorf("the first-run egress list does not allow the model provider:\n%s", body)
	}
}
