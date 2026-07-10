package booking

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timemachine-auto/timemachine/internal/apperror"
)

type Appointment struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Phone       string     `json:"phone"`
	CarMake     string     `json:"car_make"`
	CarModel    string     `json:"car_model"`
	GovNumber   string     `json:"gov_number"`
	VIN         string     `json:"vin"`
	ServiceID   *int64     `json:"service_id,omitempty"`
	DesiredAt   *time.Time `json:"desired_at,omitempty"`
	Comment     string     `json:"comment"`
	Status      string     `json:"status"`
	ManagerNote string     `json:"manager_note,omitempty"`
	Source      string     `json:"source"`
	ExternalRef string     `json:"external_ref,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Input struct {
	Name      string     `json:"name"`
	Phone     string     `json:"phone"`
	CarMake   string     `json:"car_make"`
	CarModel  string     `json:"car_model"`
	GovNumber string     `json:"gov_number"`
	VIN       string     `json:"vin"`
	ServiceID *int64     `json:"service_id"`
	DesiredAt *time.Time `json:"desired_at"`
	Comment   string     `json:"comment"`
	Source    string     `json:"source"`
}

var phoneRE = regexp.MustCompile(`[^\d+]`)

func (in *Input) Validate() error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 120 {
		return apperror.Invalid("укажите имя (до 120 символов)")
	}
	clean := phoneRE.ReplaceAllString(in.Phone, "")
	if len(clean) < 10 || len(clean) > 15 {
		return apperror.Invalid("укажите корректный телефон")
	}
	if in.CarMake == "" || in.CarModel == "" {
		return apperror.Invalid("укажите марку и модель автомобиля")
	}
	if in.VIN != "" && len(in.VIN) != 17 {
		return apperror.Invalid("VIN должен быть 17 символов или пустой")
	}
	if in.Source == "" {
		in.Source = "site"
	}
	return nil
}

type Repo struct{ db *pgxpool.Pool }

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

func (r *Repo) Create(ctx context.Context, in *Input) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO appointments
		  (name, phone, car_make, car_model, gov_number, vin, service_id,
		   desired_at, comment, status, source)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'new',$10)
		RETURNING id`,
		in.Name, in.Phone, in.CarMake, in.CarModel, in.GovNumber, in.VIN,
		in.ServiceID, in.DesiredAt, in.Comment, in.Source,
	).Scan(&id)
	if err != nil {
		return 0, apperror.Internal("create appointment", err)
	}
	return id, nil
}

type ListParams struct {
	Status string
	Limit  int
	Offset int
}

func (r *Repo) List(ctx context.Context, p ListParams) ([]Appointment, int, error) {
	if p.Limit <= 0 || p.Limit > 200 {
		p.Limit = 100
	}
	var clause string
	var args []any
	if p.Status != "" {
		clause = "WHERE status = $1"
		args = append(args, p.Status)
	}
	var total int
	if err := r.db.QueryRow(ctx,
		"SELECT count(*) FROM appointments "+clause, args...).Scan(&total); err != nil {
		return nil, 0, apperror.Internal("count appointments", err)
	}
	n := len(args) + 1
	q := fmt.Sprintf(`SELECT id,name,phone,car_make,car_model,gov_number,vin,
		COALESCE(service_id,0), COALESCE(desired_at, now()), comment, status,
		manager_note, source, external_ref, created_at, updated_at
		FROM appointments %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		clause, n, n+1)
	args = append(args, p.Limit, p.Offset)
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, apperror.Internal("list appointments", err)
	}
	defer rows.Close()
	out := make([]Appointment, 0)
	for rows.Next() {
		var a Appointment
		var svcID int64
		var desired time.Time
		if err := rows.Scan(&a.ID, &a.Name, &a.Phone, &a.CarMake, &a.CarModel,
			&a.GovNumber, &a.VIN, &svcID, &desired, &a.Comment, &a.Status,
			&a.ManagerNote, &a.Source, &a.ExternalRef, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, 0, apperror.Internal("scan appointment", err)
		}
		if svcID > 0 {
			a.ServiceID = &svcID
		}
		a.DesiredAt = &desired
		out = append(out, a)
	}
	return out, total, rows.Err()
}

func (r *Repo) SetStatus(ctx context.Context, id int64, status, note string) error {
	res, err := r.db.Exec(ctx,
		"UPDATE appointments SET status=$2, manager_note=$3 WHERE id=$1",
		id, status, note)
	if err != nil {
		return apperror.Internal("update appointment", err)
	}
	if res.RowsAffected() == 0 {
		return apperror.NotFound("appointment not found")
	}
	return nil
}

func (r *Repo) Get(ctx context.Context, id int64) (*Appointment, error) {
	const q = `SELECT id,name,phone,car_make,car_model,gov_number,vin,
		COALESCE(service_id,0), COALESCE(desired_at, now()), comment, status,
		manager_note, source, external_ref, created_at, updated_at
		FROM appointments WHERE id=$1`
	var a Appointment
	var svcID int64
	var desired time.Time
	err := r.db.QueryRow(ctx, q, id).Scan(&a.ID, &a.Name, &a.Phone, &a.CarMake,
		&a.CarModel, &a.GovNumber, &a.VIN, &svcID, &desired, &a.Comment,
		&a.Status, &a.ManagerNote, &a.Source, &a.ExternalRef, &a.CreatedAt, &a.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, apperror.NotFound("appointment not found")
	}
	if err != nil {
		return nil, apperror.Internal("get appointment", err)
	}
	if svcID > 0 {
		a.ServiceID = &svcID
	}
	a.DesiredAt = &desired
	return &a, nil
}
