package retry

import (
	"errors"
	"testing"
)

func TestClassifier_Defaults(t *testing.T) {
	cases := []struct {
		err  error
		want ErrorClass
	}{
		{errors.New("connection timeout"), ClassTransient},
		{errors.New("rate limit exceeded"), ClassTransient},
		{errors.New("internal server error"), ClassTransient},
		{errors.New("service unavailable"), ClassTransient},
		{errors.New("permission denied"), ClassPermanent},
		{errors.New("invalid input"), ClassPermanent},
		{errors.New("not authorized"), ClassPermanent},
		{errors.New("syntax error in line 3"), ClassFixable},
		{errors.New("edit not unique: 3 matches"), ClassFixable},
		{errors.New("edit not found"), ClassFixable},
		{errors.New("merge conflict in file"), ClassFixable},
		{errors.New("totally unknown error"), ClassPermanent},
	}
	cls := NewClassifier()
	for _, tc := range cases {
		if got := cls.Classify(tc.err); got != tc.want {
			t.Fatalf("err=%q got=%s want=%s", tc.err.Error(), got, tc.want)
		}
	}
}

func TestClassifier_AddCustom(t *testing.T) {
	c := NewClassifier()
	c.Add(ClassTransient, `cloudflare`)
	if got := c.Classify(errors.New("CloudFlare 502")); got != ClassTransient {
		t.Fatalf("custom rule missed: %s", got)
	}
}

func TestClassifier_NilError(t *testing.T) {
	c := NewClassifier()
	if got := c.Classify(nil); got != ClassPermanent {
		t.Fatalf("nil err → %s, want permanent", got)
	}
}

func TestClassifier_FirstMatchWins(t *testing.T) {
	c := NewClassifier()
	// "not found" is fixable; the order in defaultPatterns
	// puts fixable before permanent.
	if got := c.Classify(errors.New("file not found")); got != ClassFixable {
		t.Fatalf("got %s, want fixable", got)
	}
}
