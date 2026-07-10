package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/cors"

	"github.com/timemachine-auto/timemachine/internal/config"
)

// CORS restricts cross-origin origins to the configured allow-list.
func CORS(allowed []string) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   allowed,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Requested-With"},
		ExposedHeaders:   []string{"Content-Disposition", " Retry-After"},
		AllowCredentials: true,
		MaxAge:           int(5 * time.Minute / time.Second),
	})
}

// SecurityHeaders adds baseline browser security headers.
func SecurityHeaders(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			if cfg.IsProd() {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
				h.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")
			}
			next.ServeHTTP(w, r)
		})
	}
}
