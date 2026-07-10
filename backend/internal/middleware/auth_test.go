package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/timemachine-auto/timemachine/internal/config"
)

func mint(t *testing.T, cfg config.JWTConfig, userID int64, email, role string) string {
	t.Helper()
	std := jwt.RegisteredClaims{
		Issuer:    cfg.Issuer,
		Subject:   strconv.FormatInt(userID, 10),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	claims := mergeTestClaims(std, map[string]any{"email": email, "role": role, "uid": userID})
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(cfg.Secret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func mergeTestClaims(std jwt.RegisteredClaims, extras map[string]any) jwt.MapClaims {
	out := make(jwt.MapClaims, len(extras)+4)
	for k, v := range extras {
		out[k] = v
	}
	if std.Issuer != "" {
		out["iss"] = std.Issuer
	}
	if std.Subject != "" {
		out["sub"] = std.Subject
	}
	if std.ExpiresAt != nil {
		out["exp"] = std.ExpiresAt.Unix()
	}
	if std.IssuedAt != nil {
		out["iat"] = std.IssuedAt.Unix()
	}
	return out
}

func TestRequireAuth_StoresNumericUserID(t *testing.T) {
	cfg := config.JWTConfig{Secret: "test-secret-test-secret-test-32b", TTL: time.Hour, Issuer: "tm"}

	mw := RequireAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, _ := r.Context().Value(CtxUserID).(int64)
		role, _ := r.Context().Value(CtxRole).(string)
		email, _ := r.Context().Value(CtxEmail).(string)
		if uid != 72 {
			t.Errorf("CtxUserID = %d, want 72", uid)
		}
		if role != "admin" || email != "a@b.c" {
			t.Errorf("role/email = %q / %q", role, email)
		}
		w.WriteHeader(http.StatusOK)
	}))

	const userID = int64(72)
	tok := mint(t, cfg, userID, "a@b.c", "admin")

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestRequireAuth_RejectsBadSecret(t *testing.T) {
	cfg := config.JWTConfig{Secret: "good-secret-good-secret-32-bytes!", TTL: time.Hour}
	mw := RequireAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tok := mint(t, config.JWTConfig{Secret: "wrong-secret-xxxxxx-xxxxxx-xxxxxx-32!", TTL: time.Hour}, 1, "a@b.c", "admin")
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Errorf("expected non-200 for token minted with wrong secret, got %d", rec.Code)
	}
}

func TestRequireAuth_RejectsMissingToken(t *testing.T) {
	cfg := config.JWTConfig{Secret: "s", TTL: time.Hour}
	mw := RequireAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", rec.Code)
	}
}

func TestUserID_ReturnsZeroWhenAbsent(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = r.WithContext(context.Background())
	if got := UserID(r); got != 0 {
		t.Errorf("UserID without ctx = %d, want 0", got)
	}
}
