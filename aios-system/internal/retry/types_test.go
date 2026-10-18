package retry

import (
	"errors"
	"strings"
	"testing"
)

func TestRetryBudgetExceeded_Message(t *testing.T) {
	e := &RetryBudgetExceeded{TaskID: "t1", Tool: "edit", Used: 3, Limit: 3}
	if !strings.Contains(e.Error(), "t1") || !strings.Contains(e.Error(), "edit") {
		t.Fatalf("err = %q", e.Error())
	}
	if !errors.Is(e, ErrRetryBudgetExceeded) {
		t.Fatal("should match sentinel")
	}
}
