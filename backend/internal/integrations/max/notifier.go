package max

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timemachine-auto/timemachine/internal/config"
	leadrepo "github.com/timemachine-auto/timemachine/internal/domain/lead"
)

// Notifier pushes new appointments/leads from the site out to the MAX admin
// chat (or logs them in dry-run). It is the single integration seam between
// the public website and the messenger side.
type Notifier struct {
	client Client
	chatID string
	log    *slog.Logger
}

func NewNotifier(client Client, adminChatID string, log *slog.Logger) *Notifier {
	if adminChatID == "" {
		log.Warn("MAX notify chat id not set; notifications will be logged only")
	}
	return &Notifier{client: client, chatID: adminChatID, log: log}
}

// NotifyAppointment informs the manager chat about a new site appointment.
// PII (name/phone) is delivered to MAX but NEVER logged in plaintext.
func (n *Notifier) NotifyAppointment(ctx context.Context, a AppointmentNotification) error {
	if n.chatID == "" {
		n.log.Info("max.notify.skipped (no chat id)", "appointment", a.ID)
		return nil
	}
	text := fmt.Sprintf("🛠 Новая заявка с сайта #%d\n👤 %s\n☎ %s\n🚗 %s %s (%s)\nКомментарий: %s",
		a.ID, a.Name, a.Phone, a.CarMake, a.CarModel, a.GovNumber, a.Comment)
	if err := n.client.SendMessage(ctx, n.chatID, text); err != nil {
		// Log only ids/hashes, never the message body itself.
		n.log.Error("max.notify.failed", "appointment_id", a.ID, "err", err)
		return err
	}
	return nil
}

// NotifyLead informs the manager chat about a new qualified MAX lead.
func (n *Notifier) NotifyLead(ctx context.Context, l *leadrepo.Lead) error {
	if n.chatID == "" {
		return nil
	}
	text := fmt.Sprintf("✨ Новый лид из MAX #%d\n👤 %s\nУслуга: %v",
		l.ID, l.Name, l.Answer)
	if err := n.client.SendMessage(ctx, n.chatID, text); err != nil {
		n.log.Error("max.notify.failed", "lead_id", l.ID, "err", err)
		return err
	}
	return nil
}

// AppointmentNotification is a small DTO to avoid importing booking types.
type AppointmentNotification struct {
	ID        int64
	Name      string
	Phone     string
	CarMake   string
	CarModel  string
	GovNumber string
	Comment   string
}

// Store persists an inbound MAX message and returns its id. Kept here so the
// webhook handler is thin.
type Store struct{ db *pgxpool.Pool }

func NewStore(db *pgxpool.Pool) *Store { return &Store{db: db} }

func (s *Store) RecordInbound(ctx context.Context, chatID, text string, payload map[string]any) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO max_messages (chat_id, direction, text, payload)
		VALUES ($1,'in',$2,$3)`, chatID, text, payload)
	return err
}

func (s *Store) RecordOutbound(ctx context.Context, chatID, text string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO max_messages (chat_id, direction, text)
		VALUES ($1,'out',$2)`, chatID, text)
	return err
}

// BuildClient is a factory used by the server wiring.
func BuildClient(cfg config.MaxConfig, log *slog.Logger) Client {
	return NewClient(cfg.BotHandle, cfg.DeepLinkPrefix, cfg.Enabled, log)
}
