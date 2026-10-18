package git

import (
	"errors"
	"strings"
	"testing"
)

func TestErrorMessages(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{&NotARepositoryError{Path: "x"}, "not a repository"},
		{&GitCommandError{Command: []string{"git", "foo"}, Stderr: "boom", Exit: 2}, "exit 2"},
		{&EmptyCommitError{Reason: "no files"}, "no files"},
	}
	for _, c := range cases {
		if !strings.Contains(c.err.Error(), c.want) {
			t.Fatalf("err %q missing %q", c.err.Error(), c.want)
		}
	}
}

func TestErrorSentinels(t *testing.T) {
	if !errors.Is(&NotARepositoryError{}, ErrNotARepository) {
		t.Fatal("NotARepository should match")
	}
	if !errors.Is(&GitCommandError{}, ErrGitCommand) {
		t.Fatal("GitCommand should match")
	}
	if !errors.Is(&EmptyCommitError{}, ErrEmptyCommit) {
		t.Fatal("EmptyCommit should match")
	}
}
