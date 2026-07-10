package lead

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timemachine-auto/timemachine/internal/apperror"
)

type Lead struct {
	ID            int64   `json:"id"`
	ExternalRef   string  `json:"external_ref"`
	Channel       string  `json:"channel"`
	Name          string  `json:"name"`
	Phone         string  `json:"phone"`
	MaxChatID     string  `json:"max_chat_id"`
	Answer        any     `json:"answer"`
	Status        string  `json:"status"`
	AssignedTo    *int64  `json:"assigned_to,omitempty"`
	AppointmentID *int64  `json:"appointment_id,omitempty"`
	CreatedAt     string  `json:"created_at"`
	HandledAt     *string `json:"handled_at,omitempty"`
}

type Repo struct{ db *pgxpool.Pool }

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

// CreateInput is the payload that site forms and MAX agent push leads with.
type CreateInput struct {
	ExternalRef string         `json:"external_ref"`
	Channel     string         `json:"channel"`
	Name        string         `json:"name"`
	Phone       string         `json:"phone"`
	MaxChatID   string         `json:"max_chat_id"`
	Answer      map[string]any `json:"answer"`
}

func (in *CreateInput) Validate() error {
	if in.Channel != "site" && in.Channel != "max" {
		return apperror.Invalid("channel must be 'site' or 'max'")
	}
	if strings.TrimSpace(in.Name) == "" {
		return apperror.Invalid("name is required")
	}
	return nil
}

func (r *Repo) Create(ctx context.Context, in *CreateInput) (int64, error) {
	if err := in.Validate(); err != nil {
		return 0, err
	}
	payload, _ := json.Marshal(in.Answer)
	var id int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO leads (external_ref, channel, name, phone, max_chat_id, answer, status)
		VALUES ($1,$2,$3,$4,$5,$6,'raw') RETURNING id`,
		in.ExternalRef, in.Channel, in.Name, in.Phone, in.MaxChatID, payload,
	).Scan(&id)
	if err != nil {
		return 0, apperror.Internal("create lead", err)
	}
	return id, nil
}

func (r *Repo) List(ctx context.Context, status string, limit int) ([]Lead, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var clause string
	var args []any
	if status != "" {
		clause = "WHERE status = $1"
		args = append(args, status)
	}
	var total int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM leads "+clause, args...).Scan(&total); err != nil {
		return nil, 0, apperror.Internal("count leads", err)
	}
	n := len(args) + 1
	q := `SELECT id, external_ref, channel, name, phone, max_chat_id, answer,
		status, COALESCE(assigned_to,0), COALESCE(appointment_id,0),
		to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),
		COALESCE(to_char(handled_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),'')
		FROM leads ` + clause + ` ORDER BY created_at DESC LIMIT $` + itoa(n) + ` OFFSET $` + itoa(n+1)
	args = append(args, limit, 0)
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, apperror.Internal("list leads", err)
	}
	defer rows.Close()
	out := make([]Lead, 0)
	for rows.Next() {
		var l Lead
		var assigned, appt int64
		var handled string
		var answer []byte
		if err := rows.Scan(&l.ID, &l.ExternalRef, &l.Channel, &l.Name, &l.Phone,
			&l.MaxChatID, &answer, &l.Status, &assigned, &appt,
			&l.CreatedAt, &handled); err != nil {
			return nil, 0, apperror.Internal("scan lead", err)
		}
		if assigned > 0 {
			l.AssignedTo = &assigned
		}
		if appt > 0 {
			l.AppointmentID = &appt
		}
		if handled != "" {
			l.HandledAt = &handled
		}
		_ = json.Unmarshal(answer, &l.Answer)
		out = append(out, l)
	}
	return out, total, rows.Err()
}

func (r *Repo) SetStatus(ctx context.Context, id int64, status string) error {
	res, err := r.db.Exec(ctx, "UPDATE leads SET status=$2 WHERE id=$1", id, status)
	if err != nil {
		return apperror.Internal("update lead", err)
	}
	if res.RowsAffected() == 0 {
		return apperror.NotFound("lead not found")
	}
	return nil
}

func (r *Repo) Get(ctx context.Context, id int64) (*Lead, error) {
	var l Lead
	var answer []byte
	err := r.db.QueryRow(ctx, `SELECT id, external_ref, channel, name, phone,
		max_chat_id, answer, status FROM leads WHERE id=$1`, id).
		Scan(&l.ID, &l.ExternalRef, &l.Channel, &l.Name, &l.Phone,
			&l.MaxChatID, &answer, &l.Status)
	if err == pgx.ErrNoRows {
		return nil, apperror.NotFound("lead not found")
	}
	if err != nil {
		return nil, apperror.Internal("get lead", err)
	}
	_ = json.Unmarshal(answer, &l.Answer)
	return &l, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
