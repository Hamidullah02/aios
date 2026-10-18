package egress

import "testing"

func TestAllowList_Exact(t *testing.T) {
	a := NewAllowList([]string{"api.openai.com"}, nil)
	if !a.Allows("api.openai.com") {
		t.Fatal("exact should match")
	}
	if a.Allows("evil.com") {
		t.Fatal("non-list should not match")
	}
}

func TestAllowList_Glob(t *testing.T) {
	a := NewAllowList([]string{"*.example.com"}, nil)
	if !a.Allows("api.example.com") {
		t.Fatal("glob should match sub")
	}
	if a.Allows("example.com") {
		t.Fatal("apex should not match")
	}
}

func TestAllowList_CaseInsensitive(t *testing.T) {
	a := NewAllowList([]string{"api.openai.com"}, nil)
	if !a.Allows("API.OPENAI.COM") {
		t.Fatal("case-insensitive host match")
	}
}

func TestAllowList_EmptyDenies(t *testing.T) {
	a := NewAllowList(nil, nil)
	if a.Allows("anything.com") {
		t.Fatal("empty allow list should deny")
	}
}

func TestAllowList_AllowWinsGlob(t *testing.T) {
	a := NewAllowList([]string{"*"}, nil)
	if !a.Allows("anywhere.com") {
		t.Fatal("wildcard allow should match any host")
	}
}

func TestAllowList_DenyWins(t *testing.T) {
	a := NewAllowList([]string{"*"}, []string{"*.evil.com"})
	if !a.Allows("good.com") {
		t.Fatal("good.com should be allowed")
	}
	if a.Allows("api.evil.com") {
		t.Fatal("api.evil.com should be denied")
	}
}

func TestAllowList_StripPort(t *testing.T) {
	a := NewAllowList([]string{"api.openai.com"}, nil)
	if !a.Allows("api.openai.com:443") {
		t.Fatal("port should be stripped for comparison")
	}
}

func TestAllowList_StripIPv6(t *testing.T) {
	a := NewAllowList([]string{"[::1]"}, nil)
	if !a.Allows("[::1]:8080") {
		t.Fatal("ipv6 with port should match")
	}
}

func TestAllowList_NilSafe(t *testing.T) {
	var a *AllowList
	if a.Allows("x") {
		t.Fatal("nil list should deny")
	}
	if a.Hash() != "" {
		t.Fatal("nil hash should be empty")
	}
}

func TestAllowList_HashStable(t *testing.T) {
	a := NewAllowList([]string{"a", "b"}, []string{"c"})
	b := NewAllowList([]string{"a", "b"}, []string{"c"})
	if a.Hash() != b.Hash() {
		t.Fatal("same rules should hash equally")
	}
	c := NewAllowList([]string{"b", "a"}, []string{"c"})
	if a.Hash() != c.Hash() {
		t.Fatal("hash should be order-insensitive for the underlying set")
	}
}

func TestNormaliseHost(t *testing.T) {
	cases := map[string]string{
		"API.EXAMPLE.com":     "api.example.com",
		"  api.example.com":   "api.example.com",
		"api.example.com:443": "api.example.com",
		"[::1]:8080":          "::1",
		"example.com":         "example.com",
		"":                    "",
	}
	for in, want := range cases {
		if got := normaliseHost(in); got != want {
			t.Errorf("normaliseHost(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExtractHost(t *testing.T) {
	host, err := extractHost("https://api.example.com:443/v1/chat")
	if err != nil {
		t.Fatal(err)
	}
	if host != "api.example.com:443" {
		t.Fatalf("host: %q", host)
	}
	// empty URL
	if _, err := extractHost(""); err == nil {
		t.Fatal("expected error for empty URL")
	}
}
