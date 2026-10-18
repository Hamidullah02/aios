package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/sadi/aios-system/internal/benchmark"
	"github.com/sadi/aios-system/internal/egress"
	"github.com/sadi/aios-system/internal/memory"
	"github.com/sadi/aios-system/internal/models"
	"github.com/sadi/aios-system/internal/persist"
	"github.com/sadi/aios-system/internal/planner"
	"github.com/sadi/aios-system/internal/search"
	"github.com/sadi/aios-system/internal/tools"
)

// yamlUnmarshalImpl is the one place the yaml
// package is imported; tests substitute a fake.
var yamlUnmarshalImpl = func(data []byte, v any) error {
	return yaml.Unmarshal(data, v)
}

// runSearch dispatches `search glob` and `search grep`
// from the command line. The two sub-actions use the
// same Tool implementations the agent would call.
func runSearch(ctx context.Context, args *Args, deps *Deps, stdout, stderr io.Writer) int {
	if args == nil || len(args.Positional) == 0 {
		fmt.Fprintln(stderr, "search: pass `glob` or `grep` as the first argument")
		return ExitUserError
	}
	ws, err := resolveWorkspace(args.Path)
	if err != nil {
		fmt.Fprintf(stderr, "search: %v\n", err)
		return ExitSystemError
	}
	var t tools.Tool
	switch args.Positional[0] {
	case "glob":
		t = search.GlobTool{}
	case "grep":
		t = search.GrepTool{}
	default:
		fmt.Fprintf(stderr, "search: unknown sub-command %q\n", args.Positional[0])
		return ExitUserError
	}
	res, err := t.Run(map[string]any{
		"pattern": args.Pattern,
		"query":   args.Query,
		"path":    ws,
	}, tools.Context{Actor: "cli", TaskID: "", Workspace: ws})
	printToolResult(stdout, res, err)
	if err != nil {
		return ExitSystemError
	}
	return ExitOK
}

// runMemory queries the memory store. The session
// scope is taken from the env var `AIOS_SESSION_ID`;
// without it the user scope is used.
func runMemory(ctx context.Context, args *Args, deps *Deps, stdout, stderr io.Writer) int {
	dataDir, err := memoryDataDir()
	if err != nil {
		fmt.Fprintf(stderr, "memory: %v\n", err)
		return ExitSystemError
	}
	idx, err := memory.NewIndex(memory.IndexConfig{Path: filepath.Join(dataDir, "index")})
	if err != nil {
		fmt.Fprintf(stderr, "memory: %v\n", err)
		return ExitSystemError
	}
	all := idx.All()
	limit := args.Limit
	if limit <= 0 {
		limit = 20
	}
	shown := 0
	for _, e := range all {
		if shown >= limit {
			break
		}
		if args.Query != "" && !stringContains(e.ContentHash, args.Query) {
			continue
		}
		fmt.Fprintf(stdout, "%s\t%s\n", e.ContentHash, e.Kind)
		shown++
	}
	return ExitOK
}

// runEgress starts the egress proxy.
func runEgress(ctx context.Context, args *Args, deps *Deps, stdout, stderr io.Writer) int {
	allow, headers, err := egress.LoadFile(args.ConfigPath)
	if err != nil {
		fmt.Fprintf(stderr, "egress: load %s: %v\n", args.ConfigPath, err)
		return ExitSystemError
	}
	proxy, err := egress.New(egress.Config{
		Listen:  args.Listen,
		Allow:   allow,
		Headers: headers,
	})
	if err != nil {
		fmt.Fprintf(stderr, "egress: %v\n", err)
		return ExitSystemError
	}
	fmt.Fprintf(stdout, "egress: listening on %s\n", args.Listen)
	if err := proxy.Serve(); err != nil && err != context.Canceled {
		fmt.Fprintf(stderr, "egress: serve: %v\n", err)
		return ExitSystemError
	}
	return ExitOK
}

// runBenchmark executes a benchmark dataset through
// the existing benchmark.CLI surface. The shim
// runner is a no-op for the smoke test.
func runBenchmark(ctx context.Context, args *Args, deps *Deps, stdout, stderr io.Writer) int {
	c := benchmark.CLI{
		DatasetPath: args.DatasetPath,
		OutputPath:  args.OutputPath,
		Quiet:       false,
	}
	return c.Run(ctx, shimRunner{})
}

// runModel lists configured models from the price
// table. The capability registry is a follow-up.
func runModel(ctx context.Context, args *Args, deps *Deps, stdout, stderr io.Writer) int {
	raw, err := readYAMLPrices(args.PricePath)
	if err != nil {
		fmt.Fprintf(stderr, "model: load %s: %v\n", args.PricePath, err)
		return ExitSystemError
	}
	for _, p := range raw {
		fmt.Fprintf(stdout, "%s\t$%.6f input / $%.6f output per 1k\n",
			p.Model, p.InputPer1K, p.OutputPer1K)
	}
	_ = models.LoadPriceTable // keep the import
	return ExitOK
}

// runSession lists persisted session files.
func runSession(ctx context.Context, args *Args, deps *Deps, stdout, stderr io.Writer) int {
	dir := args.SessionsPath
	if dir == "" {
		dir = filepath.Join(defaultAiosHome(), "sessions")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(stderr, "session: read %s: %v\n", dir, err)
		return ExitSystemError
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fmt.Fprintln(stdout, e.Name())
	}
	return ExitOK
}

// runPlan validates a plan YAML and re-encodes it
// through the planner package. The encoder is the
// canonical JSON the planner uses for prompts.
func runPlan(ctx context.Context, args *Args, deps *Deps, stdout, stderr io.Writer) int {
	plan, err := LoadPlan(args.PlanPath)
	if err != nil {
		fmt.Fprintf(stderr, "plan: %v\n", err)
		return ExitUserError
	}
	if err := plan.Validate(); err != nil {
		fmt.Fprintf(stderr, "plan: %v\n", err)
		return ExitUserError
	}
	enc, err := planner.EncodePlan(plan)
	if err != nil {
		fmt.Fprintf(stderr, "plan: encode: %v\n", err)
		return ExitSystemError
	}
	fmt.Fprintln(stdout, enc)
	return ExitOK
}

// --- helpers ---

// resolveWorkspace picks the workspace for tool calls
// made directly from the CLI. The flag is preferred
// over the env var, which is preferred over the CWD.
func resolveWorkspace(p string) (string, error) {
	if p != "" && p != "." {
		abs, err := filepath.Abs(p)
		if err != nil {
			return "", err
		}
		return abs, nil
	}
	ws, err := tools.Workspace()
	if err != nil {
		return "", err
	}
	return ws, nil
}

// printToolResult writes a tool's Result and any
// error to the right stream.
func printToolResult(out io.Writer, res tools.Result, err error) {
	if !res.OK {
		fmt.Fprintf(out, "ok=false summary=%s\n", res.Summary)
	}
	for _, v := range res.Output {
		switch t := v.(type) {
		case string:
			fmt.Fprintln(out, t)
		case []string:
			for _, s := range t {
				fmt.Fprintln(out, s)
			}
		case []any:
			for _, e := range t {
				fmt.Fprintln(out, fmt.Sprint(e))
			}
		default:
			b, _ := json.Marshal(v)
			fmt.Fprintln(out, string(b))
		}
	}
}

// memoryDataDir returns the on-disk root for the
// memory store.
func memoryDataDir() (string, error) {
	if d := os.Getenv("AIOS_DATA_DIR"); d != "" {
		return d, nil
	}
	home := os.Getenv("AIOS_HOME")
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		home = h
	}
	return filepath.Join(home, "memory"), nil
}

// defaultAiosHome returns the aios home directory.
func defaultAiosHome() string {
	if d := os.Getenv("AIOS_HOME"); d != "" {
		return d
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return ".aios"
	}
	return filepath.Join(h, ".aios")
}

// shimRunner is a no-op benchmark runner. The
// signature is dictated by the benchmark.Runner
// interface (Run(WedgeTask) Result).
type shimRunner struct{}

func (shimRunner) Run(_ benchmark.WedgeTask) benchmark.Result {
	return benchmark.Result{Success: true}
}

// Force the import set to stay live even if a
// follow-up removes a call. The persist package is
// imported for a future session-show sub-action.
var _ = persist.NewSessionCheckpoint

// readYAMLPrices reads the price YAML directly so the
// CLI can list models without depending on the
// PriceTable's private maps. The shape is the same
// the doc set describes for `var/prices.yaml`.
func readYAMLPrices(path string) ([]models.Price, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return readPriceYAMLBytes(raw)
}

// readPriceYAML parses a price YAML. The function
// lives next to readYAMLPrices; the split is a
// trivial testability seam.
func readPriceYAML(path string) ([]models.Price, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return readPriceYAMLBytes(raw)
}

// readPriceYAMLBytes parses a price YAML payload.
func readPriceYAMLBytes(raw []byte) ([]models.Price, error) {
	var wrapper struct {
		Prices []models.Price `yaml:"prices"`
	}
	if err := yamlUnmarshal(raw, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Prices, nil
}

// yamlUnmarshal is a thin wrapper that keeps the
// import of gopkg.in/yaml.v3 local to the file.
// Returning the same signature as yaml.Unmarshal
// means callers do not need to know the wrapper
// exists.
func yamlUnmarshal(data []byte, v any) error {
	return yamlUnmarshalImpl(data, v)
}

// stringContains is a tiny inline test to keep
// imports minimal. It avoids `strings.Contains` for
// the one call site in the memory subcommand.
func stringContains(s, sub string) bool {
	if sub == "" {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
