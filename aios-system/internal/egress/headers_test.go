package egress

import (
	"net/http"
	"testing"
)

func TestHeaderRules_RemoveThenAdd(t *testing.T) {
	r := NewHeaderRules(
		[]HeaderRule{{Name: "X-Secret"}},
		[]HeaderRule{{Name: "X-Aios", Value: "1"}},
	)
	h := http.Header{}
	h.Set("X-Secret", "leak")
	h.Set("X-Other", "keep")
	r.ApplyToRequest(h)
	if got := h.Get("X-Secret"); got != "" {
		t.Errorf("X-Secret should be removed, got %q", got)
	}
	if got := h.Get("X-Other"); got != "keep" {
		t.Errorf("X-Other should be kept, got %q", got)
	}
	if got := h.Get("X-Aios"); got != "1" {
		t.Errorf("X-Aios should be added, got %q", got)
	}
}

func TestHeaderRules_ReplacesHeaderOnRemoveThenAdd(t *testing.T) {
	r := NewHeaderRules(
		[]HeaderRule{{Name: "Authorization"}},
		[]HeaderRule{{Name: "Authorization", Value: "Bearer proxy-token"}},
	)
	h := http.Header{}
	h.Set("Authorization", "Bearer original")
	r.ApplyToRequest(h)
	if got := h.Get("Authorization"); got != "Bearer proxy-token" {
		t.Fatalf("Authorization should be replaced, got %q", got)
	}
}

func TestHeaderRules_NilSafe(t *testing.T) {
	var r *HeaderRules
	h := http.Header{}
	h.Set("X-A", "1")
	r.ApplyToRequest(h) // must not panic
	if h.Get("X-A") != "1" {
		t.Fatal("nil rules should not mutate")
	}
	if r.Hash() != "" {
		t.Fatal("nil hash should be empty")
	}
}

func TestHeaderRules_HashStable(t *testing.T) {
	r1 := NewHeaderRules(
		[]HeaderRule{{Name: "X-A"}},
		[]HeaderRule{{Name: "X-B", Value: "1"}},
	)
	r2 := NewHeaderRules(
		[]HeaderRule{{Name: "X-A"}},
		[]HeaderRule{{Name: "X-B", Value: "1"}},
	)
	if r1.Hash() != r2.Hash() {
		t.Fatal("same rules should hash equally")
	}
	r3 := NewHeaderRules(
		[]HeaderRule{{Name: "X-A"}},
		[]HeaderRule{{Name: "X-B", Value: "2"}},
	)
	if r1.Hash() == r3.Hash() {
		t.Fatal("different values should hash differently")
	}
}

func TestHeaderRules_ApplyToResponse(t *testing.T) {
	r := NewHeaderRules(
		[]HeaderRule{{Name: "Server"}},
		[]HeaderRule{{Name: "Via", Value: "aios"}},
	)
	h := http.Header{}
	h.Set("Server", "nginx")
	h.Set("Content-Type", "application/json")
	r.ApplyToResponse(h)
	if h.Get("Server") != "" {
		t.Error("Server should be removed from response")
	}
	if h.Get("Via") != "aios" {
		t.Error("Via should be added to response")
	}
	if h.Get("Content-Type") != "application/json" {
		t.Error("Content-Type should be untouched")
	}
}

func TestValidateHeaderName(t *testing.T) {
	cases := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{"X-Aios", "X-Aios", false},
		{"  X-Aios  ", "", true},
		{"", "", true},
		{"Authorization", "Authorization", false},
	}
	for _, c := range cases {
		got, err := validateHeaderName(c.name)
		if (err != nil) != c.wantErr {
			t.Errorf("validateHeaderName(%q) err=%v, wantErr=%v", c.name, err, c.wantErr)
		}
		if !c.wantErr && got != c.want {
			t.Errorf("validateHeaderName(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}
