package knowledge

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timemachine-auto/timemachine/internal/apperror"
)

type Item struct {
	ID        int64    `json:"id"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Tags      []string `json:"tags"`
	Source    string   `json:"source"`
	UpdatedAt string   `json:"updated_at"`
}

type Repo struct{ db *pgxpool.Pool }

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

func (r *Repo) List(ctx context.Context) ([]Item, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, title, body, COALESCE(tags,'{}'), source, updated_at
		FROM knowledge_items ORDER BY title`)
	if err != nil {
		return nil, apperror.Internal("list knowledge", err)
	}
	defer rows.Close()
	out := make([]Item, 0)
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Title, &it.Body, &it.Tags, &it.Source, &it.UpdatedAt); err != nil {
			return nil, apperror.Internal("scan knowledge", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// Search returns KB items whose body or title match any token (case-insensitive).
func (r *Repo) Search(ctx context.Context, query string, limit int) ([]Item, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}
	tokens := strings.Fields(q)
	conds := make([]string, 0, len(tokens))
	args := make([]any, 0, len(tokens))
	for i, tok := range tokens {
		conds = append(conds, fmtILike(i+1))
		args = append(args, "%"+tok+"%")
	}
	args = append(args, limit)
	querySQL := `SELECT id, title, body, COALESCE(tags,'{}'), source, updated_at
		FROM knowledge_items
		WHERE ` + strings.Join(conds, " AND ") + `
		ORDER BY updated_at DESC LIMIT $` + intToStr(len(args))
	rows, err := r.db.Query(ctx, querySQL, args...)
	if err != nil {
		return nil, apperror.Internal("search knowledge", err)
	}
	defer rows.Close()
	out := make([]Item, 0)
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Title, &it.Body, &it.Tags, &it.Source, &it.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func fmtILike(n int) string {
	return "(title ILIKE $" + intToStr(n) + " OR body ILIKE $" + intToStr(n) + ")"
}

func intToStr(n int) string {
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

type Input struct {
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Tags   []string `json:"tags"`
	Source string   `json:"source"`
}

func (in *Input) Validate() error {
	if strings.TrimSpace(in.Title) == "" {
		return apperror.Invalid("title обязателен")
	}
	if strings.TrimSpace(in.Body) == "" {
		return apperror.Invalid("body обязателен")
	}
	return nil
}

func (r *Repo) Upsert(ctx context.Context, id int64, in *Input, userID int64) (int64, error) {
	if err := in.Validate(); err != nil {
		return 0, err
	}
	if id == 0 {
		var newID int64
		err := r.db.QueryRow(ctx, `
			INSERT INTO knowledge_items (title, body, tags, source, updated_by)
			VALUES ($1,$2,$3,$4,$5) RETURNING id`,
			in.Title, in.Body, in.Tags, in.Source, userID).Scan(&newID)
		return newID, err
	}
	_, err := r.db.Exec(ctx, `
		UPDATE knowledge_items SET title=$2, body=$3, tags=$4, source=$5,
		updated_by=$6, updated_at=now() WHERE id=$1`,
		id, in.Title, in.Body, in.Tags, in.Source, userID)
	return id, err
}

func (r *Repo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, "DELETE FROM knowledge_items WHERE id=$1", id)
	return err
}
