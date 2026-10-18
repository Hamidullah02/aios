package git

import "testing"

func TestParseStatus_Empty(t *testing.T) {
	st := parseStatus("")
	if st.Branch != "HEAD" || len(st.Staged) != 0 || len(st.Modified) != 0 || len(st.Untracked) != 0 {
		t.Fatalf("got %+v", st)
	}
}

func TestParseStatus_BranchHeader(t *testing.T) {
	st := parseStatus("## main\x00")
	if st.Branch != "main" {
		t.Fatalf("branch = %q", st.Branch)
	}
}

func TestParseStatus_BranchWithTracking(t *testing.T) {
	st := parseStatus("## main...origin/main [ahead 3]\x00")
	if st.Branch != "main" {
		t.Fatalf("branch = %q", st.Branch)
	}
}

func TestParseStatus_ModifiedNotStaged(t *testing.T) {
	// " M file" → modified=true, staged=false
	st := parseStatus("## main\x00 M foo.go\x00")
	if len(st.Staged) != 0 || len(st.Modified) != 1 || st.Modified[0] != "foo.go" {
		t.Fatalf("got %+v", st)
	}
}

func TestParseStatus_Staged(t *testing.T) {
	// "M  file" → staged=true
	st := parseStatus("## main\x00M  foo.go\x00")
	if len(st.Staged) != 1 || st.Staged[0] != "foo.go" {
		t.Fatalf("got %+v", st)
	}
}

func TestParseStatus_StagedAndModified(t *testing.T) {
	// "MM file" → both
	st := parseStatus("## main\x00MM foo.go\x00")
	if len(st.Staged) != 1 || len(st.Modified) != 1 {
		t.Fatalf("got %+v", st)
	}
}

func TestParseStatus_Untracked(t *testing.T) {
	// "?? file"
	st := parseStatus("## main\x00?? new.txt\x00")
	if len(st.Untracked) != 1 || st.Untracked[0] != "new.txt" {
		t.Fatalf("got %+v", st)
	}
}

func TestParseStatus_Deleted(t *testing.T) {
	// "D  file" → staged
	st := parseStatus("## main\x00D  gone.txt\x00")
	if len(st.Staged) != 1 || st.Staged[0] != "gone.txt" {
		t.Fatalf("got %+v", st)
	}
	// " D file" → modified (deleted from worktree)
	st = parseStatus("## main\x00 D gone.txt\x00")
	if len(st.Modified) != 1 || st.Modified[0] != "gone.txt" {
		t.Fatalf("got %+v", st)
	}
}

func TestXIsStaged(t *testing.T) {
	for _, c := range []struct {
		b    byte
		want bool
	}{
		{' ', false}, {'?', false}, {'M', true}, {'A', true}, {'D', true}, {'R', true}, {'C', true},
	} {
		if got := xIsStaged(c.b); got != c.want {
			t.Fatalf("xIsStaged(%q) = %v, want %v", c.b, got, c.want)
		}
	}
}

func TestYIsModified(t *testing.T) {
	for _, c := range []struct {
		b    byte
		want bool
	}{
		{' ', false}, {'?', false}, {'M', true}, {'D', true}, {'A', false}, {'U', false},
	} {
		if got := yIsModified(c.b); got != c.want {
			t.Fatalf("yIsModified(%q) = %v, want %v", c.b, got, c.want)
		}
	}
}

func TestSplitNonEmpty(t *testing.T) {
	got := splitNonEmpty("a\n\nb\n\n", "\n")
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got %v", got)
	}
}
