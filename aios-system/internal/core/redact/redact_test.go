package redact

import (
	"strings"
	"testing"
)

func TestRedactString_NoSecrets(t *testing.T) {
	in := "this is a normal log line"
	if got := RedactString(in); got != in {
		t.Fatalf("got %q want %q", got, in)
	}
}

func TestRedactString_Empty(t *testing.T) {
	if got := RedactString(""); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestRedactString_APIKey(t *testing.T) {
	in := `log: api_key=abcdefghijklmnop1234 end`
	out := RedactString(in)
	if strings.Contains(out, "abcdefghijklmnop1234") {
		t.Fatalf("secret leaked: %s", out)
	}
	if !strings.Contains(out, PlaceholderPrefix+"api_key:") {
		t.Fatalf("no api_key placeholder in: %s", out)
	}
}

func TestRedactString_JWT(t *testing.T) {
	in := `bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1MSJ9.7s5V4qQkbcP7VQrRmYj8`
	out := RedactString(in)
	if strings.Contains(out, "eyJhbGciOiJIUzI1NiJ9") {
		t.Fatalf("JWT leaked: %s", out)
	}
	if !strings.Contains(out, PlaceholderPrefix+"jwt:") {
		t.Fatalf("no jwt placeholder in: %s", out)
	}
}

func TestRedactString_AWSKey(t *testing.T) {
	in := `aws key: AKIAIOSFODNN7EXAMPLE`
	out := RedactString(in)
	if strings.Contains(out, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatalf("AWS key leaked: %s", out)
	}
	if !strings.Contains(out, PlaceholderPrefix+"aws_key:") {
		t.Fatalf("no aws_key placeholder in: %s", out)
	}
}

func TestRedactString_GitHub(t *testing.T) {
	in := `token: ghp_1234567890abcdefghijklmnopqrstuvwxyz`
	out := RedactString(in)
	if strings.Contains(out, "ghp_1234567890") {
		t.Fatalf("GitHub token leaked: %s", out)
	}
	if !strings.Contains(out, PlaceholderPrefix+"github_token:") {
		t.Fatalf("no github_token placeholder in: %s", out)
	}
}

func TestRedactString_PEM(t *testing.T) {
	in := `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA
-----END RSA PRIVATE KEY-----`
	out := RedactString(in)
	if strings.Contains(out, "MIIEowIBAAKCAQEA") {
		t.Fatalf("PEM body leaked: %s", out)
	}
	if !strings.Contains(out, PlaceholderPrefix+"pem:") {
		t.Fatalf("no pem placeholder in: %s", out)
	}
}

func TestRedactString_Stable(t *testing.T) {
	in := `api_key=abcdefghijklmnop1234 end`
	a := RedactString(in)
	b := RedactString(in)
	if a != b {
		t.Fatalf("not stable: %q vs %q", a, b)
	}
}

func TestRedactString_StablePlaceholder_DifferentKinds(t *testing.T) {
	same := "abcdefghijklmnop1234"
	p1 := StablePlaceholder(KindAPIKey, same)
	p2 := StablePlaceholder(KindJWT, same)
	if p1 == p2 {
		t.Fatal("placeholder collision across kinds")
	}
}

func TestRedactValue_Nested(t *testing.T) {
	v := map[string]any{
		"step":  "edit",
		"args":  map[string]any{"creds": "api_key=abcdefghijklmnop1234"},
		"trace": []any{"a", "AKIAIOSFODNN7EXAMPLE", "b"},
	}
	out := RedactValue(v, "approval_id")
	m := out.(map[string]any)
	args := m["args"].(map[string]any)
	if !strings.Contains(args["creds"].(string), "REDACTED") {
		t.Fatalf("nested creds not redacted: %v", args)
	}
	trace := m["trace"].([]any)
	if !strings.Contains(trace[1].(string), "REDACTED") {
		t.Fatalf("nested aws not redacted: %v", trace)
	}
}

func TestRedactValue_ApprovalIDPreserved(t *testing.T) {
	v := map[string]any{
		"approval_id": "550e8400-e29b-41d4-a716-446655440000",
		"note":        "greet",
	}
	out := RedactValue(v, "approval_id")
	m := out.(map[string]any)
	if m["approval_id"] != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("approval_id not preserved: %v", m)
	}
}
