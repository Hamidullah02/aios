package schemas

import "testing"

func TestBoundTo_Match(t *testing.T) {
	cases := []struct {
		name     string
		b        *BoundTo
		task     string
		tool     string
		actor    string
		resource string
		want     bool
	}{
		{"nil-wildcard", nil, "t1", "edit", "alice", "/x", true},
		{"empty-wildcard", &BoundTo{}, "t1", "edit", "alice", "/x", true},
		{"all-match", &BoundTo{Task: "t1", Tool: "edit", Actor: "alice", Resource: "/x"}, "t1", "edit", "alice", "/x", true},
		{"task-mismatch", &BoundTo{Task: "t1"}, "t2", "edit", "alice", "/x", false},
		{"tool-mismatch", &BoundTo{Tool: "edit"}, "t1", "del", "alice", "/x", false},
		{"actor-mismatch", &BoundTo{Actor: "alice"}, "t1", "edit", "bob", "/x", false},
		{"resource-mismatch", &BoundTo{Resource: "/x"}, "t1", "edit", "alice", "/y", false},
		{"partial-wildcard", &BoundTo{Tool: "edit"}, "any", "edit", "any", "any", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.b.Match(c.task, c.tool, c.actor, c.resource); got != c.want {
				t.Fatalf("Match = %v, want %v", got, c.want)
			}
		})
	}
}

func TestBoundTo_RoundTrip(t *testing.T) {
	original := &BoundTo{Task: "t1", Tool: "edit"}
	data, err := original.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var got BoundTo
	if err := got.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if got.Task != original.Task || got.Tool != original.Tool {
		t.Fatalf("roundtrip mismatch: %+v vs %+v", got, *original)
	}
}

func TestBoundTo_StrictUnmarshal_RejectsUnknown(t *testing.T) {
	data := []byte(`{"task":"t1","scope":"s1"}`)
	var b BoundTo
	if err := b.UnmarshalJSON(data); err == nil {
		t.Fatal("expected error for unknown field")
	}
}
