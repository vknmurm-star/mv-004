package max

// This file defines the integration surface for the MAX messenger Bot API.
//
// MAX (https://max.ru) exposes a Bot API similar in shape to the common
// messenger bot APIs. The exact field names/endpoints are owned by MAX and
// must be filled from the official Bot API docs once a bot token is issued.
//
// To keep the project real but not hardcoded to a moving external API, we
// define small stable interfaces and a stub implementation that:
//   - never performs network calls unless MAX_ENABLED=true,
//   - logs everything it would have sent (dry-run),
//   - is trivially replaceable by a real HTTP client without touching callers.
//
// Replace stubClient with the real implementation keyed on cfg.AI/Max.
//
// MaxEvent describes an inbound webhook event from MAX.
// Webhook signing: MAX provides a secret used to validate the request
// (signature header or shared secret). We verify it before processing.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// Sender describes the author of an inbound message.
type Sender struct {
	ChatID string `json:"chat_id"`
	UserID string `json:"user_id"`
	Name   string `json:"name"`
}

// InboundMessage is the canonical shape our handlers work with, regardless of
// the raw webhook payload (which is normalized to this struct).
type InboundMessage struct {
	Text   string `json:"text"`
	Sender Sender `json:"sender"`
	// MessageID is populated from a top-level "message_id" field when present;
	// it is the webhook idempotency key.
	MessageID string `json:"message_id,omitempty"`
	// Raw is kept for extensibility (attachments, buttons, etc).
	Raw map[string]any `json:"raw,omitempty"`
}

// Client is the outbound MAX API surface used by the app.
type Client interface {
	// SendMessage delivers text to a chat. In dry-run it only logs.
	SendMessage(ctx context.Context, chatID, text string) error
	// SendMessageWithButtons sends text + quick-reply buttons (label → payload).
	SendMessageWithButtons(ctx context.Context, chatID, text string, buttons []Button) error
	// ChatDeepLink builds a public link to start the bot in MAX.
	ChatDeepLink() string
}

// Button is a quick-reply / inline button shown in MAX.
type Button struct {
	Label   string `json:"label"`
	Payload string `json:"payload"`
}

// stubClient implements Client without touching the network.
type stubClient struct {
	handle  string
	prefix  string
	enabled bool
	log     *slog.Logger
}

func NewClient(handle, deepLinkPrefix string, enabled bool, log *slog.Logger) Client {
	if deepLinkPrefix == "" {
		deepLinkPrefix = "https://max.ru/"
	}
	return &stubClient{handle: handle, prefix: deepLinkPrefix, enabled: enabled, log: log}
}

func (c *stubClient) SendMessage(ctx context.Context, chatID, text string) error {
	if c.enabled {
		// Real implementation would POST to `${apiBase}/messages/sendText`
		// with Authorization: Bearer <MAX_API_TOKEN>. Left as a clearly-marked
		// extension point so we never hardcode an unstable external contract.
	}
	c.log.Info("max.send (dry-run)", "chat_id", chatID, "text", truncate(text, 200), "enabled", c.enabled)
	return nil
}

func (c *stubClient) SendMessageWithButtons(ctx context.Context, chatID string, text string, buttons []Button) error {
	labels := make([]string, 0, len(buttons))
	for _, b := range buttons {
		labels = append(labels, b.Label)
	}
	c.log.Info("max.send.buttons (dry-run)", "chat_id", chatID, "text", truncate(text, 200), "buttons", strings.Join(labels, "|"), "enabled", c.enabled)
	return nil
}

func (c *stubClient) ChatDeepLink() string {
	return strings.TrimRight(c.prefix, "/") + "/" + strings.TrimPrefix(c.handle, "@")
}

// WebhookHeader is the HTTP header carrying the MAX webhook signature.
// MAX's Bot API uses "X-Max-Bot-Api-Secret" per the platform integration guide.
// The previous implementation read "X-Max-Signature", which is kept as a
// fallback header name so local test harnesses that send either are accepted.
const (
	WebhookHeader       = "X-Max-Bot-Api-Secret"
	WebhookHeaderLegacy = "X-Max-Signature"
)

// VerifyWebhook confirms an incoming MAX request is authentic against a shared
// secret using HMAC-SHA256 over the raw request body, the same scheme the
// MAX Bot API signs outbound webhooks with. `signatureHex` is the hex-encoded
// HMAC the messenger sent; the comparison is constant-time.
//
// Note: the previous implementation compared hex(secret) == signature, which
// is not HMAC and accepts any attacker who learns the secret's hex form. This
// fixes that by recomputing the MAC server-side.
func VerifyWebhook(signatureHex, secret, body []byte) error {
	if len(secret) == 0 {
		return fmt.Errorf("webhook secret not configured")
	}
	if len(signatureHex) == 0 {
		return fmt.Errorf("missing signature")
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expected := make([]byte, hex.EncodedLen(mac.Size()))
	hex.Encode(expected, mac.Sum(nil))
	// length must match before constant-time compare (which itself requires
	// equal lengths to return 1).
	if len(signatureHex) != len(expected) {
		return fmt.Errorf("invalid webhook signature")
	}
	if !hmac.Equal(signatureHex, expected) {
		return fmt.Errorf("invalid webhook signature")
	}
	return nil
}

// VerifyRequest extracts the signature header from an *http.Request, accepting
// the canonical header first then the legacy alias, and verifies it.
func VerifyRequest(r *http.Request, secret, body []byte) error {
	sig := r.Header.Get(WebhookHeader)
	if sig == "" {
		sig = r.Header.Get(WebhookHeaderLegacy)
	}
	return VerifyWebhook([]byte(sig), secret, body)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
