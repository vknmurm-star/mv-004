package max

import (
	"log/slog"
	"os"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// Use testing's t.Context() when available (Go 1.24+); keeps tests deadline-aware.
func init() { testing.Init() }
