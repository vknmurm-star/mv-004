package max

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ParseWebhook normalize an arbitrary MAX webhook payload into our canonical
// InboundMessage. MAX's actual schema is owned by MAX and may evolve; this
// function reads defensively from common fields and falls back to raw map.
//
// The expected shape is loosely:
//
//	{
//	  "chat_id": "...",
//	  "from": { "user_id": "...", "name": "..." },
//	  "message": { "text": "..." }
//	}
func ParseWebhook(body []byte) (InboundMessage, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return InboundMessage{}, fmt.Errorf("invalid json: %w", err)
	}
	msg := InboundMessage{Raw: raw}

	if chatID, ok := raw["chat_id"].(string); ok {
		msg.Sender.ChatID = chatID
	}
	if mid, ok := raw["message_id"].(string); ok {
		msg.MessageID = mid
	} else if midf, ok := raw["message_id"].(float64); ok {
		msg.MessageID = fmt.Sprintf("%v", midf)
	}
	if from, ok := raw["from"].(map[string]any); ok {
		if uid, ok := from["user_id"].(string); ok {
			msg.Sender.UserID = uid
		}
		if name, ok := from["name"].(string); ok {
			msg.Sender.Name = name
		}
	}
	if messenger, ok := raw["message"].(map[string]any); ok {
		if text, ok := messenger["text"].(string); ok {
			msg.Text = text
		}
		if mid, ok := messenger["message_id"].(string); ok {
			if msg.MessageID == "" {
				msg.MessageID = mid
			}
		}
	}
	// Some APIs nest under "event" — try that as a fallback.
	if msg.Text == "" {
		if ev, ok := raw["event"].(map[string]any); ok {
			if text, ok := ev["text"].(string); ok {
				msg.Text = text
			}
		}
	}
	// NOTE: previously, when no recognized text field was found, the entire
	// raw JSON body was assigned as the user message. That made the agent
	// echo JSON back to customers in prod when MAX changed payload shape, and
	// is a footgun. Instead we leave Text empty and let the handler decide.
	return msg, nil
}

// SendMessage wrapper kept here for the AI agent to reuse asynchronously.
// The agent calls Client.SendMessage directly; nothing to add.

// DB-backed message store is in notifier.go (Store). To keep imports tidy we
// also expose a small helper to build a message Store from a pool.
func NewMessageStore(db *pgxpool.Pool) *Store { return NewStore(db) }

// DiagnosticPing returns whether the bot client is enabled (for /admin settings).
func DiagnosticPing(ctx context.Context, c Client) (string, error) {
	if _, ok := c.(*stubClient); ok {
		return "stub/dry-run", nil
	}
	_ = ctx
	return "live", nil
}
