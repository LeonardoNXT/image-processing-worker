package observability

import (
	"log/slog"
	"sync/atomic"
)

var cleanupFailures atomic.Uint64

func CleanupFailure(component string, err error) {
	if err == nil {
		return
	}
	total := cleanupFailures.Add(1)
	slog.Warn("cleanup failed", "component", component, "error", err, "cleanup_failures_total", total)
}

func CleanupFailureCount() uint64 {
	return cleanupFailures.Load()
}
