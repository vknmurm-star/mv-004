package faq

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Item struct {
	ID          int64  `json:"id"`
	Question    string `json:"question"`
	Answer      string `json:"answer"`
	Category    string `json:"category"`
	SortOrder   int    `json:"sort_order"`
	IsPublished bool   `json:"is_published"`
}

type Repo struct{ db *pgxpool.Pool }

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

func (r *Repo) List(ctx context.Context, publishedOnly bool) ([]Item, error) {
	q := `SELECT id, question, answer, category, sort_order, is_published
		FROM faq_items`
	if publishedOnly {
		q += " WHERE is_published = true"
	}
	q += " ORDER BY sort_order, id"
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Item, 0)
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Question, &it.Answer, &it.Category, &it.SortOrder, &it.IsPublished); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

type Input struct {
	Question    string `json:"question"`
	Answer      string `json:"answer"`
	Category    string `json:"category"`
	SortOrder   int    `json:"sort_order"`
	IsPublished bool   `json:"is_published"`
}

func (r *Repo) Create(ctx context.Context, in *Input) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO faq_items (question, answer, category, sort_order, is_published)
		VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		in.Question, in.Answer, in.Category, in.SortOrder, in.IsPublished).Scan(&id)
	return id, err
}

func (r *Repo) Update(ctx context.Context, id int64, in *Input) error {
	_, err := r.db.Exec(ctx, `
		UPDATE faq_items SET question=$2, answer=$3, category=$4,
		sort_order=$5, is_published=$6 WHERE id=$1`,
		id, in.Question, in.Answer, in.Category, in.SortOrder, in.IsPublished)
	return err
}

func (r *Repo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, "DELETE FROM faq_items WHERE id=$1", id)
	return err
}
