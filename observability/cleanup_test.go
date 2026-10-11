package observability

import (
	"errors"
	"testing"
)

func TestCleanupFailureCount(t *testing.T) {
	before := CleanupFailureCount()
	CleanupFailure("test", nil)
	if CleanupFailureCount() != before {
		t.Fatal("nil cleanup changed counter")
	}
	CleanupFailure("test", errors.New("cleanup failed"))
	if CleanupFailureCount() != before+1 {
		t.Fatal("cleanup failure was not counted")
	}
}
