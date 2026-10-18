package git

import "testing"

func TestParseLog_Empty(t *testing.T) {
	if got := parseLog(""); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestParseLog_One(t *testing.T) {
	out := "abc123|Alice|2024-01-01T00:00:00Z|hello world\x1e"
	rows := parseLog(out)
	if len(rows) != 1 {
		t.Fatalf("rows = %d", len(rows))
	}
	if rows[0].Hash != "abc123" || rows[0].Author != "Alice" ||
		rows[0].Subject != "hello world" {
		t.Fatalf("row = %+v", rows[0])
	}
}

func TestParseLog_Multiple(t *testing.T) {
	out := "aaa|Alice|2024-01-01T00:00:00Z|first\x1ebbb|Bob|2024-01-02T00:00:00Z|second\x1e"
	rows := parseLog(out)
	if len(rows) != 2 {
		t.Fatalf("rows = %d", len(rows))
	}
	if rows[1].Hash != "bbb" || rows[1].Author != "Bob" {
		t.Fatalf("row 1 = %+v", rows[1])
	}
}

func TestParseLog_TrailingNewline(t *testing.T) {
	out := "abc|Alice|2024-01-01T00:00:00Z|hello\x1e\n"
	rows := parseLog(out)
	if len(rows) != 1 || rows[0].Hash != "abc" {
		t.Fatalf("rows = %v", rows)
	}
}

func TestParseWorktreeList_Empty(t *testing.T) {
	if got := parseWorktreeList(""); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestParseWorktreeList_One(t *testing.T) {
	out := "worktree /tmp/wt1\nHEAD abc123\nbranch refs/heads/main\n\n"
	rows := parseWorktreeList(out)
	if len(rows) != 1 {
		t.Fatalf("rows = %d", len(rows))
	}
	if rows[0].Path != "/tmp/wt1" || rows[0].Commit != "abc123" || rows[0].Branch != "refs/heads/main" {
		t.Fatalf("row = %+v", rows[0])
	}
}

func TestParseWorktreeList_Two(t *testing.T) {
	out := "worktree /tmp/a\nHEAD 111\nbranch refs/heads/main\n\nworktree /tmp/b\nHEAD 222\nbranch refs/heads/feat\n\n"
	rows := parseWorktreeList(out)
	if len(rows) != 2 {
		t.Fatalf("rows = %d", len(rows))
	}
	if rows[0].Path != "/tmp/a" || rows[1].Branch != "refs/heads/feat" {
		t.Fatalf("rows = %+v", rows)
	}
}
