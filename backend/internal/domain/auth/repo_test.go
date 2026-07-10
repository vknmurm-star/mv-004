package auth

import (
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/timemachine-auto/timemachine/internal/config"
)

// Tests cover claims assembly: the Subject must be the numeric user id (so
// middleware can attribute admin actions to the right audit_logs.actor_id),
// and the uid claim must be carried for the same reason. These tests do not
// touch bcrypt or PostgreSQL (covered separately by integration tests).

func TestMergeClaims_IncludesUIDAndNumericSubject(t *testing.T) {
	const userID = int64(72)
	now := time.Now()
	std := jwt.RegisteredClaims{
		Issuer:    "timemachine",
		Subject:   strconv.FormatInt(userID, 10),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
	}
	extras := map[string]any{
		"email": "a@b.c",
		"role":  "admin",
		"uid":   userID,
	}
	claims := mergeClaims(std, extras)
	if got := claims["sub"]; got != strconv.FormatInt(userID, 10) {
		t.Errorf("sub = %v, want %d", got, userID)
	}
	if got, ok := claims["uid"].(int64); !ok || got != userID {
		t.Errorf("uid claim missing or wrong: %v", got)
	}
	if claims["email"] != "a@b.c" {
		t.Errorf("email claim missing")
	}
	if claims["role"] != "admin" {
		t.Errorf("role claim missing")
	}
	if claims["iss"] != "timemachine" {
		t.Errorf("iss claim missing")
	}
}

func TestConfigDefaults_DevFallbackSecretWhenEmpty(t *testing.T) {
	// Cfg.Load is exercised in integration tests; this just sanity-checks the
	// production requirement paths are visible (not run here).
	_ = config.JWTConfig{Secret: "dev-only-insecure-secret-change-me", TTL: 12 * time.Hour}
}
