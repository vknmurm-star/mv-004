package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timemachine-auto/timemachine/internal/apperror"
)

type Repo struct{ db *pgxpool.Pool }

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

func (r *Repo) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, slug, COALESCE(parent_id,0), sort_order, created_at
		FROM service_categories ORDER BY sort_order, name`)
	if err != nil {
		return nil, apperror.Internal("list categories", err)
	}
	defer rows.Close()

	out := make([]Category, 0)
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.SortOrder, &c.CreatedAt); err != nil {
			return nil, apperror.Internal("scan category", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ServiceListParams filters the public/admin service list.
type ServiceListParams struct {
	Search     string
	Category   string // slug
	ActiveOnly bool
	Limit      int
	Offset     int
}

func (r *Repo) List(ctx context.Context, p ServiceListParams) ([]Service, int, error) {
	if p.Limit <= 0 || p.Limit > 500 {
		p.Limit = 500
	}
	var (
		where []string
		args  []any
		n     = 1
		add   = func(clause string, v any) { where = append(where, fmt.Sprintf(clause, n)); args = append(args, v); n++ }
	)
	if p.ActiveOnly {
		where = append(where, "s.is_active = true")
	}
	if p.Category != "" {
		add("c.slug = $%d", p.Category)
	}
	if q := strings.TrimSpace(p.Search); q != "" {
		add("s.name ILIKE $%d", "%"+q+"%")
	}
	clause := ""
	if len(where) > 0 {
		clause = "WHERE " + strings.Join(where, " AND ")
	}

	countQ := fmt.Sprintf("SELECT count(*) FROM services s JOIN service_categories c ON c.id=s.category_id %s", clause)
	var total int
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, apperror.Internal("count services", err)
	}

	query := fmt.Sprintf(`
		SELECT s.id, s.category_id, c.slug, c.name, s.name, s.description, s.long_description,
		       s.price_cents, s.currency, s.duration_minutes, s.is_from_price,
		       s.is_active, s.archived, s.sort_order, s.created_at, s.updated_at
		FROM services s
		JOIN service_categories c ON c.id = s.category_id
		%s ORDER BY s.sort_order, s.name
		LIMIT $%d OFFSET $%d`, clause, n, n+1)
	args = append(args, p.Limit, p.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, apperror.Internal("list services", err)
	}
	defer rows.Close()

	out := make([]Service, 0)
	for rows.Next() {
		var s Service
		if err := rows.Scan(&s.ID, &s.CategoryID, &s.CategorySlug, &s.CategoryName,
			&s.Name, &s.Description, &s.LongDescription, &s.PriceCents, &s.Currency,
			&s.DurationMin, &s.IsFromPrice, &s.IsActive, &s.Archived,
			&s.SortOrder, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, apperror.Internal("scan service", err)
		}
		s.PriceRUB = s.PriceCents / 100
		out = append(out, s)
	}
	return out, total, rows.Err()
}

func (r *Repo) Get(ctx context.Context, id int64) (*Service, error) {
	const q = `
		SELECT s.id, s.category_id, c.slug, c.name, s.name, s.description, s.long_description,
		       s.price_cents, s.currency, s.duration_minutes, s.is_from_price,
		       s.is_active, s.archived, s.sort_order, s.created_at, s.updated_at
		FROM services s
		JOIN service_categories c ON c.id = s.category_id
		WHERE s.id = $1`
	var s Service
	err := r.db.QueryRow(ctx, q, id).Scan(&s.ID, &s.CategoryID, &s.CategorySlug, &s.CategoryName,
		&s.Name, &s.Description, &s.LongDescription, &s.PriceCents, &s.Currency,
		&s.DurationMin, &s.IsFromPrice, &s.IsActive, &s.Archived,
		&s.SortOrder, &s.CreatedAt, &s.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, apperror.NotFound("service not found")
	}
	if err != nil {
		return nil, apperror.Internal("get service", err)
	}
	s.PriceRUB = s.PriceCents / 100
	return &s, nil
}

// ServiceInput is the validated payload for create/update.
type ServiceInput struct {
	CategoryID      int64  `json:"category_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	LongDescription string `json:"long_description"`
	PriceCents      int64  `json:"price_cents"`
	Currency        string `json:"currency"`
	DurationMin     int    `json:"duration_minutes"`
	IsFromPrice     bool   `json:"is_from_price"`
	IsActive        bool   `json:"is_active"`
	IsActivePtr     *bool  `json:"-"`
	SortOrder       int    `json:"sort_order"`
}

func (in *ServiceInput) Validate() error {
	if in.CategoryID <= 0 {
		return apperror.Invalid("category_id is required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return apperror.Invalid("name is required")
	}
	if in.PriceCents < 0 {
		return apperror.Invalid("price must be >= 0")
	}
	if in.Currency == "" {
		in.Currency = "RUB"
	}
	if in.Currency != "RUB" && in.Currency != "USD" && in.Currency != "EUR" {
		return apperror.Invalid("currency must be RUB, USD or EUR")
	}
	if in.DurationMin <= 0 {
		in.DurationMin = 60
	}
	return nil
}

func (r *Repo) Create(ctx context.Context, in *ServiceInput) (int64, error) {
	const q = `
		INSERT INTO services (category_id, name, description, long_description, price_cents,
		    currency, duration_minutes, is_from_price, is_active, sort_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id`
	var id int64
	err := r.db.QueryRow(ctx, q,
		in.CategoryID, in.Name, in.Description, in.LongDescription, in.PriceCents,
		in.Currency, in.DurationMin, in.IsFromPrice, in.IsActive, in.SortOrder,
	).Scan(&id)
	if err != nil {
		return 0, apperror.Internal("create service", err)
	}
	return id, nil
}

func (r *Repo) Update(ctx context.Context, id int64, in *ServiceInput) error {
	res, err := r.db.Exec(ctx, `
		UPDATE services SET
		    category_id=$2, name=$3, description=$4, long_description=$5,
		    price_cents=$6, currency=$7, duration_minutes=$8, is_from_price=$9,
		    is_active=$10, sort_order=$11
		WHERE id=$1`,
		id, in.CategoryID, in.Name, in.Description, in.LongDescription, in.PriceCents,
		in.Currency, in.DurationMin, in.IsFromPrice, in.IsActive, in.SortOrder)
	if err != nil {
		return apperror.Internal("update service", err)
	}
	if res.RowsAffected() == 0 {
		return apperror.NotFound("service not found")
	}
	return nil
}

func (r *Repo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.Exec(ctx, `UPDATE services SET archived=true, is_active=false WHERE id=$1`, id)
	if err != nil {
		return apperror.Internal("archive service", err)
	}
	if res.RowsAffected() == 0 {
		return apperror.NotFound("service not found")
	}
	return nil
}

// UpsertByNameInTx makes import idempotent by keying on (category_id, name)
// via the unique index ux_services_category_name. Returns "create" or "update".
func (r *Repo) UpsertByNameInTx(ctx context.Context, tx pgx.Tx, in *ServiceInput) (int64, string, error) {
	var id int64
	var inserted bool
	err := tx.QueryRow(ctx, `
		INSERT INTO services (category_id, name, description, long_description, price_cents,
		    currency, duration_minutes, is_from_price, is_active, sort_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (category_id, name) WHERE archived = false
		DO UPDATE SET
		    description      = EXCLUDED.description,
		    long_description = EXCLUDED.long_description,
		    price_cents      = EXCLUDED.price_cents,
		    currency         = EXCLUDED.currency,
		    duration_minutes = EXCLUDED.duration_minutes,
		    is_from_price    = EXCLUDED.is_from_price,
		    is_active        = EXCLUDED.is_active,
		    sort_order       = EXCLUDED.sort_order,
		    archived         = false,
		    updated_at       = now()
		RETURNING id, (xmax = 0) AS inserted`,
		in.CategoryID, in.Name, in.Description, in.LongDescription, in.PriceCents,
		in.Currency, in.DurationMin, in.IsFromPrice, in.IsActive, in.SortOrder,
	).Scan(&id, &inserted)
	if err != nil {
		return 0, "", err
	}
	if inserted {
		return id, "create", nil
	}
	return id, "update", nil
}
