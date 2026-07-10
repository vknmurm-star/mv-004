package httpserver

import (
	"fmt"
	"log/slog"
	"os"
)

func pkgLogger(env string) *slog.Logger {
	level := slog.LevelInfo
	if env != "production" {
		level = slog.LevelDebug
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(h).With("svc", "timemachine", "env", env)
}

// quote is a tiny helper to avoid pulling fmt everywhere it's not already used.
func quote(s string) string { return fmt.Sprintf("%q", s) }
