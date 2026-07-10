package review

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Review struct {
	ID          int64  `json:"id"`
	Author      string `json:"author"`
	Rating      int    `json:"rating"`
	Body        string `json:"body"`
	CarInfo     string `json:"car_info"`
	IsPublished bool   `json:"is_published"`
	CreatedAt   string `json:"created_at"`
}

type Repo struct{ db *pgxpool.Pool }

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

func (r *Repo) List(ctx context.Context, publishedOnly bool) ([]Review, error) {
	q := `SELECT id, author, rating, body, car_info, is_published,
		to_char(created_at,'YYYY-MM-DD') FROM reviews`
	if publishedOnly {
		q += " WHERE is_published = true"
	}
	q += " ORDER BY created_at DESC LIMIT 200"
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Review, 0)
	for rows.Next() {
		var rv Review
		if err := rows.Scan(&rv.ID, &rv.Author, &rv.Rating, &rv.Body, &rv.CarInfo, &rv.IsPublished, &rv.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, rv)
	}
	return out, rows.Err()
}

type Input struct {
	Author  string `json:"author"`
	Rating  int    `json:"rating"`
	Body    string `json:"body"`
	CarInfo string `json:"car_info"`
}

func (r *Repo) Create(ctx context.Context, in *Input) (int64, error) {
	if in.Rating < 1 || in.Rating > 5 {
		in.Rating = 5
	}
	var id int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO reviews (author, rating, body, car_info, is_published)
		VALUES ($1,$2,$3,$4,false) RETURNING id`,
		in.Author, in.Rating, in.Body, in.CarInfo).Scan(&id)
	return id, err
}

func (r *Repo) SetPublished(ctx context.Context, id int64, published bool) error {
	_, err := r.db.Exec(ctx, "UPDATE reviews SET is_published=$2 WHERE id=$1", id, published)
	return err
}
