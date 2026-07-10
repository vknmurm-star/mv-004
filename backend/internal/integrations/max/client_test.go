package max

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func sign(t *testing.T, secret, body string) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyWebhook_ValidHMAC(t *testing.T) {
	body := `{"chat_id":"c1","message":{"text":"привет"}}`
	secret := "super-secret"
	sig := sign(t, secret, body)
	if err := VerifyWebhook([]byte(sig), []byte(secret), []byte(body)); err != nil {
		t.Errorf("valid HMAC rejected: %v", err)
	}
}

func TestVerifyWebhook_BadSignature(t *testing.T) {
	body := `{"x":1}`
	if err := VerifyWebhook([]byte("deadbeef"), []byte("secret"), []byte(body)); err == nil {
		t.Error("expected error for wrong signature, got nil")
	}
}

func TestVerifyWebhook_EmptySecret(t *testing.T) {
	if err := VerifyWebhook([]byte("x"), nil, []byte("body")); err == nil {
		t.Error("expected error when secret not configured")
	}
}

func TestVerifyWebhook_EmptySignature(t *testing.T) {
	if err := VerifyWebhook(nil, []byte("s"), []byte("body")); err == nil {
		t.Error("expected error when signature header missing")
	}
}

func TestVerifyWebhook_ConstantTimeSameHMAC(t *testing.T) {
	// Repeated correct signatures must verify (constant-time compare path).
	body := `{"msg":"x"}`
	secret := "k"
	for i := 0; i < 5; i++ {
		sig := sign(t, secret, body)
		if err := VerifyWebhook([]byte(sig), []byte(secret), []byte(body)); err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
	}
}

func TestVerifyRequest_HeaderNames(t *testing.T) {
	body := `{"chat_id":"c1","message":{"text":"привет"}}`
	secret := "s"
	sig := sign(t, secret, body)

	// Canonical header.
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set(WebhookHeader, sig)
	if err := VerifyRequest(r, []byte(secret), []byte(body)); err != nil {
		t.Errorf("canonical header rejected: %v", err)
	}

	// Legacy alias header.
	r2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r2.Header.Set(WebhookHeaderLegacy, sig)
	if err := VerifyRequest(r2, []byte(secret), []byte(body)); err != nil {
		t.Errorf("legacy header rejected: %v", err)
	}

	// No header at all.
	r3 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if err := VerifyRequest(r3, []byte(secret), []byte(body)); err == nil {
		t.Error("expected error when no signature header present")
	}
}

func TestParseWebhook_NormalizesFields(t *testing.T) {
	body := `{"chat_id":"c1","message_id":"mid-42","from":{"user_id":"u1","name":"Иван"},"message":{"text":"привет"}}`
	msg, err := ParseWebhook([]byte(body))
	if err != nil {
		t.Fatalf("ParseWebhook: %v", err)
	}
	if msg.Sender.ChatID != "c1" {
		t.Errorf("ChatID = %q, want c1", msg.Sender.ChatID)
	}
	if msg.MessageID != "mid-42" {
		t.Errorf("MessageID = %q, want mid-42", msg.MessageID)
	}
	if msg.Sender.UserID != "u1" || msg.Sender.Name != "Иван" {
		t.Errorf("Sender = %+v", msg.Sender)
	}
	if msg.Text != "привет" {
		t.Errorf("Text = %q, want привет", msg.Text)
	}
}

func TestParseWebhook_NestedEventFallback(t *testing.T) {
	body := `{"chat_id":"c2","event":{"text":"hi"}}`
	msg, err := ParseWebhook([]byte(body))
	if err != nil {
		t.Fatalf("ParseWebhook: %v", err)
	}
	if msg.Text != "hi" {
		t.Errorf("Text = %q, want hi", msg.Text)
	}
}

func TestParseWebhook_NoTextNoFallback(t *testing.T) {
	// Previously: full JSON body was echoed as user text. Now: empty text.
	body := `{"chat_id":"c3"}`
	msg, err := ParseWebhook([]byte(body))
	if err != nil {
		t.Fatalf("ParseWebhook: %v", err)
	}
	if msg.Text != "" {
		t.Errorf("Text = %q, want empty (no JSON-as-text echo)", msg.Text)
	}
}

func TestParseWebhook_NumericMessageID(t *testing.T) {
	body := `{"chat_id":"c","message_id":12345,"message":{"text":"x"}}`
	msg, err := ParseWebhook([]byte(body))
	if err != nil {
		t.Fatalf("ParseWebhook: %v", err)
	}
	if msg.MessageID != "12345" {
		t.Errorf("MessageID = %q, want 12345", msg.MessageID)
	}
}

func TestStubClient_DryRunNeverErrors(t *testing.T) {
	c := NewClient("bot", "https://max.ru/", false, testLogger())
	if err := c.SendMessage(t.Context(), "chat", "hello"); err != nil {
		t.Errorf("dry-run SendMessage returned error: %v", err)
	}
	if err := c.SendMessageWithButtons(t.Context(), "chat", "hi",
		[]Button{{Label: "A", Payload: "a"}}); err != nil {
		t.Errorf("dry-run SendMessageWithButtons returned error: %v", err)
	}
}

func TestDeepLink(t *testing.T) {
	c := NewClient("@bot", "https://max.ru/", false, testLogger())
	if got := c.ChatDeepLink(); got != "https://max.ru/bot" {
		t.Errorf("ChatDeepLink = %q, want https://max.ru/bot", got)
	}
}
