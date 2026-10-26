package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveModel_Precedence pins the order the model is
// chosen in: the flag beats the environment beats the default.
//
// The flag is the operator's instruction for this one command.
// The environment is their instruction for this shell. The
// default is ours. An ordering that put the default first
// would ignore both.
func TestResolveModel_Precedence(t *testing.T) {
	t.Setenv(EnvModel, "")

	if got := resolveModel(""); got != DefaultModel {
		t.Errorf("no override: got %q, want %q", got, DefaultModel)
	}
	t.Setenv(EnvModel, "env-model")
	if got := resolveModel(""); got != "env-model" {
		t.Errorf("env only: got %q, want env-model", got)
	}
	if got := resolveModel("flag-model"); got != "flag-model" {
		t.Errorf("flag wins: got %q, want flag-model", got)
	}
}

// TestResolveModel_DefaultIsFree pins that the default model
// costs nothing.
//
// A default that bills would mean a first run of this product
// silently spends the operator's money. That is not a tradeoff
// worth making for a model that is good enough.
func TestResolveModel_DefaultIsFree(t *testing.T) {
	if !strings.HasSuffix(DefaultModel, ":free") {
		t.Errorf("default model %q is not a free variant", DefaultModel)
	}
}

// TestBuildRouter_RegistersOpenRouter pins that the router can
// route the default model.
//
// The capability registry says which adapter a model id maps
// to, and the adapter map says which adapters exist. A model in
// one but not the other cannot be called, and the failure
// ("no adapter registered for provider") is a configuration
// error the operator cannot fix from the command line.
func TestBuildRouter_RegistersOpenRouter(t *testing.T) {
	dir := t.TempDir()
	router, _, _, err := buildRouter(dir)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := router.AdapterFor(DefaultModel)
	if err != nil {
		t.Fatalf("default model %q is not routable: %v", DefaultModel, err)
	}
	if adapter.Provider() != "openrouter" {
		t.Errorf("provider = %q, want openrouter", adapter.Provider())
	}
}

// TestBuildRouter_DefaultModelIsPriced pins that the default
// model has a price.
//
// The router treats a model with no price as a configuration
// error and refuses the call. An operator who has done
// nothing wrong would hit that on their first run.
func TestBuildRouter_DefaultModelIsPriced(t *testing.T) {
	dir := t.TempDir()
	_, prices, _, err := buildRouter(dir)
	if err != nil {
		t.Fatal(err)
	}
	cost, err := prices.Cost(DefaultModel, 1000, 1000)
	if err != nil {
		t.Fatalf("default model %q has no price: %v", DefaultModel, err)
	}
	if cost != 0 {
		t.Errorf("cost = %v, want 0 (the default model is free)", cost)
	}
}

// TestBuildRouter_WorkspacePricesDoNotOverrideDefaults pins
// that a workspace's own prices win over ours.
//
// The operator's file is more specific than the built-in
// default, so a workspace that prices the default model
// differently keeps its number.
func TestBuildRouter_WorkspacePricesDoNotOverrideDefaults(t *testing.T) {
	dir := t.TempDir()
	body := "prices:\n  " + DefaultModel + ":\n    input_per_1k: 0.5\n    output_per_1k: 1.5\n"
	if err := os.WriteFile(filepath.Join(dir, "prices.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	_, prices, _, err := buildRouter(dir)
	if err != nil {
		t.Fatal(err)
	}
	cost, err := prices.Cost(DefaultModel, 1000, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if cost != 0.5+1.5 {
		t.Errorf("cost = %v, want %v (the workspace price must win)", cost, 2.0)
	}
}

// TestBuildRouter_WorkspaceCapabilityWins pins that a
// workspace that names the default model with a different
// provider keeps that provider.
//
// The capability registry is the routing table. An operator
// who routes a model through a different provider means it.
func TestBuildRouter_WorkspaceCapabilityWins(t *testing.T) {
	dir := t.TempDir()
	body := "capabilities:\n  " + DefaultModel + ":\n    provider: openai\n    context_window: 128000\n"
	if err := os.WriteFile(filepath.Join(dir, "capabilities.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	router, _, _, err := buildRouter(dir)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := router.AdapterFor(DefaultModel)
	if err != nil {
		t.Fatal(err)
	}
	if adapter.Provider() != "openai" {
		t.Errorf("provider = %q, want openai (the workspace capability must win)", adapter.Provider())
	}
}

// TestBuildRouter_UnknownModelStillFails pins that we did not
// make the registry accept everything.
//
// A registry that resolved every id would hide the operator's
// typos behind a confusing provider error instead of a clear
// "unknown model id".
func TestBuildRouter_UnknownModelStillFails(t *testing.T) {
	dir := t.TempDir()
	router, _, _, err := buildRouter(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := router.AdapterFor("no-such-model"); err == nil {
		t.Error("expected an error for an unknown model")
	}
}

// TestBuildRouter_KeepsWorkspacePrices pins that the
// operator's prices.yaml is actually read.
//
// The empty check used to probe the table with a sentinel model
// that no table prices, so it answered "empty" for every
// table -- and the caller then replaced the operator's file
// with the built-in list. Every price the operator set was
// silently discarded.
func TestBuildRouter_KeepsWorkspacePrices(t *testing.T) {
	dir := t.TempDir()
	body := "prices:\n  my-model:\n    input_per_1k: 2\n    output_per_1k: 3\n"
	if err := os.WriteFile(filepath.Join(dir, "prices.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	_, prices, _, err := buildRouter(dir)
	if err != nil {
		t.Fatal(err)
	}
	cost, err := prices.Cost("my-model", 1000, 1000)
	if err != nil {
		t.Fatalf("the workspace price was dropped: %v", err)
	}
	if cost != 5 {
		t.Errorf("cost = %v, want 5", cost)
	}
}

// TestBuildRouter_FallsBackWhenPricesAreEmpty pins that a
// missing or unusable prices file still yields a usable table.
func TestBuildRouter_FallsBackWhenPricesAreEmpty(t *testing.T) {
	dir := t.TempDir()
	// A list-shaped file: the loader cannot read it, so the
	// table comes back empty.
	body := "- model: gpt-4o\n  input_per_1k: 0.005\n"
	if err := os.WriteFile(filepath.Join(dir, "prices.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	_, prices, _, err := buildRouter(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prices.Cost("gpt-4o", 1000, 1000); err != nil {
		t.Fatalf("the fallback price table is missing gpt-4o: %v", err)
	}
}

// TestParseDotEnvLine covers the .env parsing rules.
//
// The cases are the ones an operator's file actually contains:
// a bare key, a quoted value, a comment, a blank line, an
// export prefix, and a trailing comment.
func TestParseDotEnvLine(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		wantKey string
		wantVal string
		wantOK  bool
	}{
		{"bare", "KEY=value", "KEY", "value", true},
		{"double quoted", `KEY="hello world"`, "KEY", "hello world", true},
		{"single quoted", "KEY='hello world'", "KEY", "hello world", true},
		{"comment", "# KEY=value", "", "", false},
		{"blank", "", "", "", false},
		{"no equals", "KEY", "", "", false},
		{"empty key", "=value", "", "", false},
		{"export prefix", "export KEY=value", "KEY", "value", true},
		{"trailing comment", "KEY=value # note", "KEY", "value", true},
		{"hash inside quotes", `KEY="a # b"`, "KEY", "a # b", true},
		{"value with equals", "KEY=a=b", "KEY", "a=b", true},
		// A UTF-8 BOM at the start of the line. Windows tools
		// write one; without the strip the key is read as
		// "\ufeffKEY" and set under a name nothing reads.
		{"utf-8 BOM", "\ufeffKEY=value", "KEY", "value", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			key, val, ok := parseDotEnvLine(c.line)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if !c.wantOK {
				return
			}
			if key != c.wantKey {
				t.Errorf("key = %q, want %q", key, c.wantKey)
			}
			if val != c.wantVal {
				t.Errorf("value = %q, want %q", val, c.wantVal)
			}
		})
	}
}

// TestLoadDotEnv_AppliesFile pins that a .env file's variables
// reach the process environment.
//
// This is the whole reason the loader exists: an operator
// pasted the key into a file, and the tool has to read it.
func TestLoadDotEnv_AppliesFile(t *testing.T) {
	dir := t.TempDir()
	body := "# the key\nOPENROUTER_API_KEY=\"sk-or-test\"\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENROUTER_API_KEY", "")

	applied := applyDotEnvFile(filepath.Join(dir, ".env"))
	if !applied {
		t.Fatal("the file should have been applied")
	}
	if got := os.Getenv("OPENROUTER_API_KEY"); got != "sk-or-test" {
		t.Errorf("OPENROUTER_API_KEY = %q, want sk-or-test", got)
	}
}

// TestLoadDotEnv_UTF8BOMFile pins that a .env written by a
// Windows tool is read.
//
// Notepad's "UTF-8" encoding, PowerShell's
// Set-Content -Encoding utf8, and the redirect operator in older
// shells all prepend a UTF-8 BOM. Without the strip, the first
// key in the file is set under the name "\ufeffOPENROUTER_API_KEY"
// -- a variable nothing will ever read. The operator sees their
// key sitting in the file and is told no key is configured.
//
// This is a file-level test on purpose: the line-level test
// proves the parser, and this proves the whole path from a
// byte-identical-to-Notepad file to the environment variable
// the provider adapter reads.
func TestLoadDotEnv_UTF8BOMFile(t *testing.T) {
	dir := t.TempDir()
	// Exactly what Notepad writes: BOM, then the key.
	body := "\ufeffOPENROUTER_API_KEY=sk-or-test\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENROUTER_API_KEY", "")

	applied := applyDotEnvFile(filepath.Join(dir, ".env"))
	if !applied {
		t.Fatal("the file should have been applied")
	}
	if got := os.Getenv("OPENROUTER_API_KEY"); got != "sk-or-test" {
		t.Errorf("OPENROUTER_API_KEY = %q, want sk-or-test", got)
	}
}

// TestLoadDotEnv_ShellWins pins that the environment beats the
// file.
//
// An operator who exported a key for one command means that
// key. A .env left lying around from another project must not
// override it.
func TestLoadDotEnv_ShellWins(t *testing.T) {
	dir := t.TempDir()
	body := "OPENROUTER_API_KEY=\"from-file\"\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENROUTER_API_KEY", "from-shell")

	applyDotEnvFile(filepath.Join(dir, ".env"))
	if got := os.Getenv("OPENROUTER_API_KEY"); got != "from-shell" {
		t.Errorf("OPENROUTER_API_KEY = %q, want from-shell", got)
	}
}

// TestLoadDotEnv_MissingFileIsNotAnError pins that a
// workspace with no .env is the normal case, not a failure.
func TestLoadDotEnv_MissingFileIsNotAnError(t *testing.T) {
	if applyDotEnvFile(filepath.Join(t.TempDir(), ".env")) {
		t.Error("a missing file must not report as applied")
	}
}

// TestLoadDotEnv_MalformedLineIsSkipped pins that one bad line
// does not stop the rest of the file from loading.
//
// A secret file is edited by hand. One stray line must not
// stop the tool from starting.
func TestLoadDotEnv_MalformedLineIsSkipped(t *testing.T) {
	dir := t.TempDir()
	body := "this is not a key\nOPENROUTER_API_KEY=sk-or-test\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENROUTER_API_KEY", "")

	applyDotEnvFile(filepath.Join(dir, ".env"))
	if got := os.Getenv("OPENROUTER_API_KEY"); got != "sk-or-test" {
		t.Errorf("OPENROUTER_API_KEY = %q, want sk-or-test", got)
	}
}
