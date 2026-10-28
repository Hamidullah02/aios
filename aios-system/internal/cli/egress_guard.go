package cli

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/egress"
	"github.com/Hamidullah02/aios/aios-system/internal/sandbox"
)

// EgressGuard owns the running egress proxy and the network
// policy it installs.
//
// It exists because the proxy and the policy have to be created
// together and torn down together. The proxy needs the allow list
// and the audit log; the policy needs the proxy's bound address;
// and the policy has to be installed as the process-wide default
// runner before any tool can start a subprocess.
type EgressGuard struct {
	// Proxy is the running proxy. nil when the guard is in
	// deny-all mode with no proxy.
	Proxy *egress.Proxy
	// Policy is the policy installed on the default runner.
	Policy *sandbox.NetworkPolicy
	// Allow is the allow list the proxy enforces. nil in
	// deny-all mode.
	Allow *egress.AllowList
	// Mode is the mode the guard was started in.
	Mode string
	// Namespaces is the namespace policy installed on the
	// default runner. nil when namespace isolation is off.
	Namespaces *sandbox.NamespacePolicy

	previousRunner sandbox.Runner
	listener       net.Listener
	serveDone      chan struct{}
}

// Egress modes.
const (
	// EgressModeProxy routes child traffic through the proxy
	// and lets the allow list decide.
	EgressModeProxy = "proxy"
	// EgressModeDenyAll blocks child traffic entirely.
	EgressModeDenyAll = "deny"
	// EgressModeOff leaves the child's network untouched.
	// It is the mode used when the operator has not asked for
	// enforcement, and it is deliberately NOT the default for
	// a production run.
	EgressModeOff = "off"
)

// Isolation modes.
const (
	// IsolationModeAuto uses the namespaces when the platform
	// and the kernel support them, and falls back to the weaker
	// containment when they do not.
	IsolationModeAuto = "auto"
	// IsolationModeNamespaces requires the namespaces. A run
	// that cannot have them fails rather than running with less
	// isolation than the operator asked for.
	IsolationModeNamespaces = "namespaces"
	// IsolationModeNone disables namespace isolation. The
	// process-group and Job Object containment still apply.
	IsolationModeNone = "none"
)

// resolveNamespaces resolves the namespace policy for an
// isolation mode.
//
// The auto mode probes the kernel once and falls back to no
// namespaces when they are unavailable. The namespaces mode
// fails instead, because an operator who asked for namespaces
// asked for a boundary they would not otherwise have.
//
// The probe is the only way to know: the namespaces are a
// runtime kernel feature, not a compile-time one, and the same
// binary runs on a kernel that allows them and one that does
// not.
func resolveNamespaces(isolation string) (*sandbox.NamespacePolicy, error) {
	switch isolation {
	case IsolationModeNone:
		return nil, nil
	case IsolationModeNamespaces:
		policy := sandbox.DefaultNamespacePolicy()
		if !sandbox.NamespacesSupported() {
			return nil, &CliError{
				Kind:  "config",
				What:  "namespace isolation was requested but the platform does not support it",
				Cause: fmt.Errorf("the namespaces are a Linux feature and this kernel or container does not allow them"),
			}
		}
		return &policy, nil
	case IsolationModeAuto, "":
		if !sandbox.NamespacesSupported() {
			return nil, nil
		}
		policy := sandbox.DefaultNamespacePolicy()
		return &policy, nil
	default:
		return nil, &CliError{
			Kind:  "config",
			What:  fmt.Sprintf("unknown isolation mode %q", isolation),
			Cause: fmt.Errorf("expected %s, %s or %s", IsolationModeAuto, IsolationModeNamespaces, IsolationModeNone),
		}
	}
}

// StartEgressGuard starts the egress proxy for a run and installs
// the network policy on the process-wide default runner.
//
// The mode is one of EgressModeProxy, EgressModeDenyAll or
// EgressModeOff:
//
//   - proxy: the proxy enforces the allow list. A host that is
//     not on the list is unreachable from any subprocess.
//   - deny: no subprocess can reach the network at all.
//   - off: no network policy. The child gets whatever the
//     caller's environment had.
//
// The function returns a guard the caller must Close. A guard in
// off mode has no proxy and no policy; Close is still safe.
//
// The workspace is where the egress.yaml lives. A missing file
// means an empty allow list, which means the proxy denies
// everything -- the fail-closed default.
func StartEgressGuard(workspace string, auditLog *audit.Log, mode string) (*EgressGuard, error) {
	return StartGuard(workspace, auditLog, mode, IsolationModeAuto)
}

// StartGuard starts the egress proxy and installs the network
// and namespace policies on the process-wide default runner.
//
// The isolation mode is one of IsolationModeAuto,
// IsolationModeNamespaces or IsolationModeNone:
//
//   - auto: use the namespaces when the platform and the kernel
//     support them, and fall back to the weaker containment
//     when they do not.
//   - namespaces: require the namespaces. A run that cannot have
//     them fails rather than running with less isolation than
//     the operator asked for.
//   - none: no namespace isolation. The process-group and Job
//     Object containment still apply.
//
// The auto mode is the default because a kernel without user
// namespaces, a container that forbids CLONE_NEWUSER and a
// hardened CI runner all refuse the namespaces, and refusing to
// run at all would be a worse answer than running with less
// containment.
func StartGuard(workspace string, auditLog *audit.Log, mode, isolation string) (*EgressGuard, error) {
	switch mode {
	case EgressModeProxy, EgressModeDenyAll, EgressModeOff:
	default:
		return nil, &CliError{
			Kind:  "config",
			What:  fmt.Sprintf("unknown egress mode %q", mode),
			Cause: fmt.Errorf("expected %s, %s or %s", EgressModeProxy, EgressModeDenyAll, EgressModeOff),
		}
	}

	if mode == EgressModeOff {
		return &EgressGuard{Mode: mode}, nil
	}

	// The namespace policy is resolved before the runner is
	// built, because the runner carries it.
	namespaces, nsErr := resolveNamespaces(isolation)
	if nsErr != nil {
		return nil, nsErr
	}

	allow, headers, err := loadEgressPolicy(workspace)
	if err != nil {
		return nil, err
	}

	if mode == EgressModeDenyAll {
		policy := &sandbox.NetworkPolicy{Mode: sandbox.NetworkModeDenyAll}
		if err := policy.Validate(); err != nil {
			return nil, &CliError{Kind: "config", What: "network policy invalid", Cause: err}
		}
		runner := sandbox.NewSubprocessRunnerWithNetwork(policy)
		runner.Namespaces = namespaces
		previous := sandbox.SetDefaultRunner(runner)
		return &EgressGuard{
			Mode:           mode,
			Policy:         policy,
			Namespaces:     namespaces,
			previousRunner: previous,
		}, nil
	}

	proxy, err := egress.New(egress.Config{
		Allow:   allow,
		Headers: headers,
		Audit:   auditLog,
		Listen:  "127.0.0.1:0",
	})
	if err != nil {
		return nil, &CliError{Kind: "config", What: "egress proxy failed", Cause: err}
	}

	// Bind before serving so the policy can name the port the
	// child will actually reach.
	listener, err := proxy.Listen()
	if err != nil {
		return nil, &CliError{Kind: "config", What: "egress proxy bind failed", Cause: err}
	}

	guard := &EgressGuard{
		Proxy:     proxy,
		Allow:     allow,
		Mode:      mode,
		listener:  listener,
		serveDone: make(chan struct{}),
	}

	go func() {
		defer close(guard.serveDone)
		// Serve returns http.ErrServerClosed on a graceful
		// shutdown. Any other error means the proxy died, and
		// the guard's policy would then point children at a
		// dead address -- which fails closed, so the error is
		// logged rather than escalated.
		if err := proxy.ServeListener(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "aios: egress proxy stopped: %v\n", err)
		}
	}()

	policy := &sandbox.NetworkPolicy{
		Mode:     sandbox.NetworkModeProxy,
		ProxyURL: proxy.URL(),
	}
	if err := policy.Validate(); err != nil {
		_ = guard.Close()
		return nil, &CliError{Kind: "config", What: "network policy invalid", Cause: err}
	}
	guard.Policy = policy
	runner := sandbox.NewSubprocessRunnerWithNetwork(policy)
	runner.Namespaces = namespaces
	guard.Namespaces = namespaces
	guard.previousRunner = sandbox.SetDefaultRunner(runner)
	return guard, nil
}

// loadEgressPolicy reads the egress policy for a workspace.
//
// A missing file is not an error: the loader returns an empty
// allow list, which denies everything. That is the fail-closed
// behaviour the safety story depends on -- a workspace without an
// egress policy gets a proxy that refuses every host, not an
// unrestricted child.
func loadEgressPolicy(workspace string) (*egress.AllowList, *egress.HeaderRules, error) {
	if workspace == "" {
		workspace = "./var"
	}
	path := filepath.Join(workspace, "egress.yaml")
	allow, headers, err := egress.LoadFile(path)
	if err != nil {
		return nil, nil, &CliError{
			Kind:  "config",
			What:  fmt.Sprintf("egress load failed: %s", path),
			Cause: err,
		}
	}
	return allow, headers, nil
}

// Close stops the proxy and restores the previous default runner.
//
// It is safe to call more than once, and safe on a guard that was
// never started. The caller should always defer it.
func (g *EgressGuard) Close() error {
	if g == nil {
		return nil
	}
	if g.previousRunner != nil {
		sandbox.SetDefaultRunner(g.previousRunner)
		g.previousRunner = nil
	}
	if g.Proxy == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := g.Proxy.Shutdown(ctx)
	if g.serveDone != nil {
		<-g.serveDone
	}
	return err
}

// Describe returns a one-line summary of what the guard enforces.
//
// It is printed at the start of a run so the operator can see the
// network scope in force without reading the config.
func (g *EgressGuard) Describe() string {
	if g == nil {
		return "egress: off"
	}
	var scope string
	switch g.Mode {
	case EgressModeDenyAll:
		scope = "egress: deny all (subprocesses cannot reach the network)"
	case EgressModeProxy:
		scope = fmt.Sprintf("egress: proxy at %s, allow list has %d host rule(s)", g.Proxy.URL(), g.allowRuleCount())
	default:
		scope = "egress: off (no network policy in force)"
	}
	return scope + "; " + g.describeIsolation()
}

// describeIsolation reports the process isolation in force.
func (g *EgressGuard) describeIsolation() string {
	if g == nil || g.Namespaces == nil {
		return "isolation: process group / job object only"
	}
	return "isolation: user, mount and pid namespaces"
}

// allowRuleCount is the number of allow patterns in the list.
func (g *EgressGuard) allowRuleCount() int {
	if g.Allow == nil {
		return 0
	}
	return g.Allow.AllowCount()
}
