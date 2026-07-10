package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/timemachine-auto/timemachine/internal/apperror"
	"github.com/timemachine-auto/timemachine/pkg/webutil"
)

// RateLimiter is a per-IP token bucket limiter used by form/API endpoints.
type RateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*entry
	r         rate.Limit
	burst     int
	ttl       time.Duration
	lastSweep time.Time
}

type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter builds a limiter allowing `perSec` events per second with
// the given burst capacity. Idle buckets are swept periodically.
func NewRateLimiter(perSec float64, burst int) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*entry),
		r:       rate.Limit(perSec),
		burst:   burst,
		ttl:     10 * time.Minute,
	}
}

// Limit is an http middleware that enforces the limiter by client IP.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if ip == "" {
			ip = "unknown"
		}
		if !rl.allow(ip) {
			w.Header().Set("Retry-After", "60")
			webutil.WriteError(w, apperror.RateLimited("too many requests, please slow down"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	e, ok := rl.buckets[key]
	if !ok {
		e = &entry{limiter: rate.NewLimiter(rl.r, rl.burst)}
		rl.buckets[key] = e
	}
	e.lastSeen = time.Now()
	if time.Since(rl.lastSweep) > time.Minute {
		rl.sweep()
	}
	rl.mu.Unlock()
	return e.limiter.Allow()
}

func (rl *RateLimiter) sweep() {
	now := time.Now()
	for k, e := range rl.buckets {
		if now.Sub(e.lastSeen) > rl.ttl {
			delete(rl.buckets, k)
		}
	}
	rl.lastSweep = now
}

// clientIP returns the IP for rate-limit bucketing.
//
// chi.middleware.RealIP (mounted in the router) already sets r.RemoteAddr from
// the X-Forwarded-For header when configured with trusted proxies, so here we
// trust that value instead of re-reading X-Forwarded-For — which would let
// any client spoof IPs to evade per-IP limits. Behind an untrusted path,
// configure RealIP explicitly or remove it; tests assert RemoteAddr behaviour.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
