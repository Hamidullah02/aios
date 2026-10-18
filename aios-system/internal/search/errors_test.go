package search

import (
	"errors"
	"strings"
	"testing"
)

func TestErrors_Messages(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{&CheckerNotFoundError{Name: "x"}, `"x" not found`},
		{&GrepTimeoutError{Pattern: "p", Timeout: "1s"}, `timed out`},
		{&UnsupportedLanguageError{Path: "x.bin"}, `no grammar`},
	}
	for _, c := range cases {
		if !strings.Contains(c.err.Error(), c.want) {
			t.Fatalf("err %q missing %q", c.err.Error(), c.want)
		}
	}
}

func TestErrors_Is(t *testing.T) {
	if !errors.Is(&CheckerNotFoundError{Name: "x"}, ErrCheckerNotFound) {
		t.Fatal("CheckerNotFound should match sentinel")
	}
	if !errors.Is(&GrepTimeoutError{}, ErrGrepTimeout) {
		t.Fatal("GrepTimeout should match sentinel")
	}
	if !errors.Is(&UnsupportedLanguageError{}, ErrUnsupportedLanguage) {
		t.Fatal("UnsupportedLanguage should match sentinel")
	}
}
