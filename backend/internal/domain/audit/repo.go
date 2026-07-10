package audit

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Entry struct {
	ID        int64  `json:"id"`
	ActorID   *int64 `json:"actor_id,omitempty"`
	Action    string `json:"action"`
	Entity    string `json:"entity"`
	EntityID  string `json:"entity_id"`
	Diff      any    `json:"diff"`
	CreatedAt string `json:"created_at"`
}

type Repo struct{ db *pgxpool.Pool }

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

func (r *Repo) Record(ctx context.Context, actorID int64, action, entity, entityID string, diff any) error {
	b, _ := json.Marshal(diff)
	_, err := r.db.Exec(ctx, `
		INSERT INTO audit_logs (actor_id, action, entity, entity_id, diff)
		VALUES ($1,$2,$3,$4,$5)`,
		nullableID(actorID), action, entity, entityID, b)
	return err
}

func nullableID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

func (r *Repo) List(ctx context.Context, limit int, offset int) ([]Entry, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, COALESCE(actor_id,0), action, entity, entity_id, diff,
		to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
		FROM audit_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Entry, 0)
	for rows.Next() {
		var e Entry
		var actor int64
		var diff []byte
		if err := rows.Scan(&e.ID, &actor, &e.Action, &e.Entity, &e.EntityID, &diff, &e.CreatedAt); err != nil {
			return nil, err
		}
		if actor > 0 {
			e.ActorID = &actor
		}
		_ = json.Unmarshal(diff, &e.Diff)
		out = append(out, e)
	}
	return out, rows.Err()
}
