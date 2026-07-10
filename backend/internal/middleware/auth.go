package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/timemachine-auto/timemachine/internal/apperror"
	"github.com/timemachine-auto/timemachine/internal/config"
	"github.com/timemachine-auto/timemachine/pkg/webutil"
)

type ctxKey string

const (
	CtxUserID ctxKey = "uid"   // int64
	CtxRole   ctxKey = "role"  // string
	CtxEmail  ctxKey = "email" // string
)

type Claims struct {
	jwt.RegisteredClaims
	Email string     `json:"email"`
	Role  string     `json:"role"`
	UID   jsonNumber `json:"uid"`
}

// jsonNumber lets us accept "1" or 1 from JWT claims into int64 robustly.
type jsonNumber int64

func (n *jsonNumber) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*n = jsonNumber(v)
	return nil
}

// RequireAuth protects admin endpoints. It validates a Bearer JWT and stores
// the authenticated subject (numeric user id) in the request context.
func RequireAuth(cfg config.JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				webutil.WriteError(w, apperror.Unauthorized("missing bearer token"))
				return
			}
			tokenStr := strings.TrimPrefix(h, "Bearer ")

			claims := &Claims{}
			_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, apperror.Unauthorized("unexpected signing method")
				}
				return []byte(cfg.Secret), nil
			})
			if err != nil || claims.Subject == "" {
				webutil.WriteError(w, apperror.Unauthorized("invalid or expired token"))
				return
			}

			uid := int64(claims.UID)
			if uid == 0 {
				// legacy token without uid: fall back to subject if numeric,
				// otherwise reject so audit stays consistent.
				if n, perr := strconv.ParseInt(claims.Subject, 10, 64); perr == nil {
					uid = n
				} else {
					webutil.WriteError(w, apperror.Unauthorized("token lacks user id"))
					return
				}
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, CtxUserID, uid)
			ctx = context.WithValue(ctx, CtxRole, claims.Role)
			ctx = context.WithValue(ctx, CtxEmail, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole narrows access to a set of roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(CtxRole).(string)
			if _, ok := allowed[role]; !ok {
				webutil.WriteError(w, apperror.Forbidden("role not permitted"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserID returns the authenticated user id from context, or 0 if absent.
// Formerly returned a (random uuid) string; now returns int64 to match the
// audit_logs.actor_id FK so admin actions are correctly attributed.
func UserID(r *http.Request) int64 {
	v, _ := r.Context().Value(CtxUserID).(int64)
	return v
}
