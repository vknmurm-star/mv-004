package middleware

import (
	"log/slog"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

type wrappedWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *wrappedWriter) WriteHeader(s int) {
	w.status = s
	w.ResponseWriter.WriteHeader(s)
}

func (w *wrappedWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// Logger emits a structured request log line per request.
func Logger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			reqID := uuid.NewString()
			ww := &wrappedWriter{ResponseWriter: w, status: http.StatusOK}
			ww.Header().Set("X-Request-ID", reqID)

			next.ServeHTTP(ww, r)

			path := r.URL.RequestURI()
			if !utf8.ValidString(path) {
				path = r.URL.Path
			}
			log.Info("http",
				"req_id", reqID,
				"method", r.Method,
				"path", path,
				"status", ww.status,
				"bytes", ww.bytes,
				"dur_ms", time.Since(start).Milliseconds(),
				"ip", clientIP(r),
			)
		})
	}
}

// Recover catches panics and converts them into 500s without crashing.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered",
						"err", rec,
						"path", r.URL.Path,
						"method", r.Method,
					)
					http.Error(w, `{"error":{"code":"internal","message":"internal error"}}`,
						http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
