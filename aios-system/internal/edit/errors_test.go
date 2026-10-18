package edit

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
		{&EditNotUniqueError{Path: "x", Count: 3}, "3 matches"},
		{&EditNotFoundError{Path: "x"}, "not found"},
		{&EditEmptyError{Path: "x"}, "empty"},
		{&MultiEditError{Path: "x", Index: 2, Reason: errors.New("boom")}, "edit 2 failed"},
		{&PatchParseError{Line: 4, Msg: "bad"}, "line 4: bad"},
		{&PatchParseError{Msg: "no files"}, "no files"},
		{&PatchContextMismatch{Path: "x", Hunk: 1, Line: 5, Expect: "a", Got: "b"}, "context mismatch"},
	}
	for _, c := range cases {
		if !strings.Contains(c.err.Error(), c.want) {
			t.Fatalf("err %q missing %q", c.err.Error(), c.want)
		}
	}
}

func TestErrorSentinels(t *testing.T) {
	if !errors.Is(&EditNotUniqueError{}, ErrEditNotUnique) {
		t.Fatal("EditNotUnique should match")
	}
	if !errors.Is(&EditNotFoundError{}, ErrEditNotFound) {
		t.Fatal("EditNotFound should match")
	}
	if !errors.Is(&EditEmptyError{}, ErrEditEmpty) {
		t.Fatal("EditEmpty should match")
	}
	if !errors.Is(&PatchParseError{}, ErrPatchParse) {
		t.Fatal("PatchParse should match")
	}
	if !errors.Is(&PatchContextMismatch{}, ErrPatchContext) {
		t.Fatal("PatchContext should match")
	}
}

func TestMultiEditUnwrap(t *testing.T) {
	inner := &EditNotFoundError{Path: "x"}
	m := &MultiEditError{Path: "x", Index: 1, Reason: inner}
	if !errors.Is(m, ErrEditNotFound) {
		t.Fatal("MultiEdit should unwrap to inner sentinel")
	}
}
