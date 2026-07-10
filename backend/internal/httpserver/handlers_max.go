package httpserver

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/timemachine-auto/timemachine/internal/aiagent"
	"github.com/timemachine-auto/timemachine/internal/apperror"
	maxpkg "github.com/timemachine-auto/timemachine/internal/integrations/max"
	"github.com/timemachine-auto/timemachine/pkg/webutil"
)

func (h *handlers) MaxDeepLink(w http.ResponseWriter, r *http.Request) {
	webutil.WriteOK(w, map[string]string{"deeplink": h.d.Max.ChatDeepLink()})
}

// MaxWebhook accepts an inbound MAX event, authenticates it, persists the
// raw message, and returns HTTP 200 immediately so the messenger stops
// redelivering. Processing is async to keep the webhook fast; repeated
// events for the same message id are deduplicated in-memory (LRU bounded by
// recentInbound). For multi-instance deployments move the dedup table to DB.
func (h *handlers) MaxWebhook(w http.ResponseWriter, r *http.Request) {
	cfg := h.d.Cfg
	body, err := readAllWithLimit(r, 1<<20)
	if err != nil {
		webutil.WriteError(w, apperror.Invalid("body read failed"))
		return
	}
	if err := maxpkg.VerifyRequest(r, []byte(cfg.Max.WebhookSecret), body); err != nil {
		// In dev without a configured secret we still process taped webhooks
		// for local testing; in prod a missing/invalid signature is a hard 401.
		if cfg.IsProd() {
			webutil.WriteError(w, apperror.Unauthorized(err.Error()))
			return
		}
	}

	msg, err := maxpkg.ParseWebhook(body)
	if err != nil {
		webutil.WriteError(w, apperror.Invalid("malformed webhook"))
		return
	}
	if msg.Sender.ChatID == "" {
		webutil.WriteError(w, apperror.Invalid("chat_id missing"))
		return
	}

	// Idempotency: dedupe on message id (or text+chat when id absent). Once
	// accepted, we ack 200 to MAX within milliseconds.
	key := msg.MessageID
	if key == "" {
		key = msg.Sender.ChatID + "|" + msg.Text
	}
	if !h.recentInbound.mark(key) {
		webutil.WriteOK(w, map[string]string{"status": "duplicate"})
		return
	}

	// Persist the raw inbound event synchronously — it's a single fast INSERT.
	if err := h.d.MaxStore.RecordInbound(r.Context(), msg.Sender.ChatID, msg.Text, msg.Raw); err != nil {
		// DB outage should not crash the webhook; logging is enough in dev.
		h.d.Cfg.Log().Warn("max.inbound.persist failed",
			"chat_id", msg.Sender.ChatID, "err", err)
	}

	// Acknowledge MAX before running the (possibly slow) AI/LLM reply path.
	webutil.WriteOK(w, map[string]string{"status": "accepted"})

	// Async processing with a bounded timeout so a hung LLM cannot pile up.
	go h.processMaxMessage(msg)
}

func (h *handlers) processMaxMessage(msg maxpkg.InboundMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	chatID := msg.Sender.ChatID
	seat, _ := h.seatStore.load(chatID)
	if seat.Step == "" {
		seat = aiagent.Seat{Step: aiagent.StepStart}
	}

	ans := h.d.Agent.Handle(ctx, &seat, msg.Text)
	h.seatStore.save(chatID, seat)

	if ans.Lead != nil && ans.Done {
		if _, err := h.d.Leads.Create(ctx, leadCreateFromDraft(ans.Lead, chatID)); err != nil {
			h.d.Cfg.Log().Warn("max.lead.save failed",
				"chat_id", chatID, "err", err)
		}
	}

	text := ans.Text
	if ans.Disclaimer {
		text += "\n\nℹ️ Точный диагноз и цена — после осмотра автомобиля мастером."
	}
	buttons := maxButtons(ans)
	if len(buttons) > 0 {
		_ = h.d.Max.SendMessageWithButtons(ctx, chatID, text, buttons)
	} else {
		_ = h.d.Max.SendMessage(ctx, chatID, text)
	}
	_ = h.d.MaxStore.RecordOutbound(ctx, chatID, text)
}

// OpenAPI serves the bundled API spec.
func (h *handlers) OpenAPI(w http.ResponseWriter, r *http.Request) {
	b, err := openAPIDocument()
	if err != nil {
		webutil.WriteError(w, apperror.Internal("openapi not found", err))
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = w.Write(b)
}

func leadCreateFromDraft(d *aiagent.LeadDraft, chatID string) *leadCreateInput {
	return &leadCreateInput{
		Channel:   "max",
		Name:      d.Name,
		Phone:     d.Phone,
		MaxChatID: chatID,
		Answer: map[string]any{
			"service": strings.TrimSpace(d.Service),
			"symptom": strings.TrimSpace(d.Symptom),
		},
	}
}

// recentInbound is a small bounded set of recently seen inbound message ids
// to deduplicate MAX redeliveries. Size is conservative; for multi-instance
// deployments this should live in the database (referenced in REQUIRES_DECISION).
type recentSet struct {
	mu   sync.Mutex
	seen map[string]struct{}
	max  int
}

func newRecentSet(max int) *recentSet { return &recentSet{seen: make(map[string]struct{}), max: max} }

// mark records the key and returns false if it was already seen recently.
func (s *recentSet) mark(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.seen[key]; ok {
		return false
	}
	if len(s.seen) >= s.max {
		// crude evacuation of the oldest key by resetting when full;
		// adequate for in-process short TTL dedup, see REQUIRES_DECISION.
		for k := range s.seen {
			delete(s.seen, k)
			break
		}
	}
	s.seen[key] = struct{}{}
	return true
}
