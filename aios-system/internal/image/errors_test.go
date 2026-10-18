package image

import (
	"errors"
	"testing"
)

func TestErrorMessages(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{&ImageTooLargeError{Path: "x", Size: 30_000_000, Max: MaxImageSize}, "30000000"},
		{&UnsupportedFormatError{Path: "x.bin", Format: ".bin"}, ".bin"},
		{&ValidationFailure{Reason: "bad"}, "bad"},
	}
	for _, c := range cases {
		if !contains(c.err.Error(), c.want) {
			t.Fatalf("err %q missing %q", c.err.Error(), c.want)
		}
	}
}

func TestErrorSentinels(t *testing.T) {
	if !errors.Is(&ImageTooLargeError{}, ErrImageTooLarge) {
		t.Fatal("ImageTooLarge should match")
	}
	if !errors.Is(&UnsupportedFormatError{}, ErrUnsupportedFormat) {
		t.Fatal("UnsupportedFormat should match")
	}
	if !errors.Is(&ValidationFailure{}, ErrValidationFailure) {
		t.Fatal("ValidationFailure should match")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
