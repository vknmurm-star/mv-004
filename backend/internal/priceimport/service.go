package priceimport

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timemachine-auto/timemachine/internal/domain/service"
)

// FileFormat detects import file format from a filename.
type FileFormat string

const (
	FormatCSV  FileFormat = "csv"
	FormatXLSX FileFormat = "xlsx"
)

func DetectFormat(name string) FileFormat {
	low := strings.ToLower(name)
	switch {
	case strings.HasSuffix(low, ".xlsx"):
		return FormatXLSX
	default:
		return FormatCSV
	}
}

// Service orchestrates upload → preview → apply for price imports.
type Service struct {
	db   *pgxpool.Pool
	repo *service.Repo
}

func NewService(db *pgxpool.Pool, repo *service.Repo) *Service {
	return &Service{db: db, repo: repo}
}

// categoryLookup satisfying resolver interface.
type catLookup struct{ db *pgxpool.Pool }

func (c *catLookup) Resolve(name string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var id int64
	n := strings.TrimSpace(name)
	err := c.db.QueryRow(ctx, `
		SELECT id FROM service_categories
		WHERE slug = $1 OR lower(name) = lower($1)
		LIMIT 1`, n).Scan(&id)
	if err == pgx.ErrNoRows {
		return 0, fmt.Errorf("категория не найдена: %q", n)
	}
	return id, err
}

// Parse reads and validates a file into rows without persisting anything.
func (s *Service) Parse(r io.Reader, format FileFormat, fileName string) ([]ParsedRow, Summary, error) {
	var rows []ParsedRow
	var sum Summary
	var err error
	cats := &catLookup{db: s.db}
	switch format {
	case FormatXLSX:
		rows, sum, err = ParseXLSX(r, cats)
	default:
		rows, sum, err = ParseCSV(r, cats)
	}
	if err != nil {
		return nil, Summary{}, err
	}
	return rows, sum, nil
}

// Job holds a created import job + its parsed rows for preview/apply.
type Job struct {
	ID        int64
	Status    string
	FileName  string
	Rows      []ParsedRow `json:"-"`
	Summary   Summary
	CreatedAt time.Time
}

// CreateJob persists a job shell with parsed rows.
func (s *Service) CreateJob(ctx context.Context, fileName string, createdBy int64, rows []ParsedRow, sum Summary) (int64, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO price_import_jobs (status, file_name, total_rows, added, updated, skipped, created_by)
		VALUES ('validated', $1, $2, $3, $4, $5, $6) RETURNING id`,
		fileName, sum.Total, sum.Added, sum.Updated, sum.Skipped, createdBy).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert job: %w", err)
	}

	batch := make([][]any, 0, len(rows))
	for _, r := range rows {
		payload, _ := json.Marshal(r.Payload)
		errs, _ := json.Marshal(r.Errors)
		batch = append(batch, []any{id, r.RowNumber, r.Action, payload, errs})
	}
	if _, err := tx.CopyFrom(ctx,
		pgx.Identifier{"price_import_rows"},
		[]string{"job_id", "row_number", "action", "payload", "errors"},
		pgx.CopyFromRows(batch)); err != nil {
		return 0, fmt.Errorf("copy rows: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return id, nil
}

// Apply executes a validated job inside one transaction so the import is
// atomic. It is idempotent: it locks the job row and only proceeds when the
// status is still 'validated'; a repeated request returns the stored summary
// without re-applying changes.
func (s *Service) Apply(ctx context.Context, jobID int64) (Summary, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Summary{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var status string
	if err := tx.QueryRow(ctx,
		`SELECT status FROM price_import_jobs WHERE id=$1 FOR UPDATE`,
		jobID).Scan(&status); err != nil {
		return Summary{}, fmt.Errorf("lock job: %w", err)
	}
	if status == "applied" {
		// Already applied — return the stored summary instead of redoing work.
		var sum Summary
		if err := tx.QueryRow(ctx,
			`SELECT total_rows, added, updated, skipped FROM price_import_jobs WHERE id=$1`,
			jobID).Scan(&sum.Total, &sum.Added, &sum.Updated, &sum.Skipped); err != nil {
			return Summary{}, fmt.Errorf("load applied summary: %w", err)
		}
		return sum, tx.Commit(ctx)
	}
	if status != "validated" {
		return Summary{}, fmt.Errorf("job is not in validated state: %s", status)
	}

	rows, err := tx.Query(ctx, `
		SELECT row_number, action, payload, errors FROM price_import_rows
		WHERE job_id=$1 ORDER BY row_number`, jobID)
	if err != nil {
		return Summary{}, fmt.Errorf("load rows: %w", err)
	}
	defer rows.Close()

	type rowItem struct {
		RowNumber int
		Action    string
		Payload   []byte
	}
	items := make([]rowItem, 0)
	for rows.Next() {
		var it rowItem
		if err := rows.Scan(&it.RowNumber, &it.Action, &it.Payload, nil); err != nil {
			return Summary{}, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return Summary{}, err
	}

	var sum Summary
	sum.Total = len(items)
	for _, it := range items {
		var in service.ServiceInput
		if err := json.Unmarshal(it.Payload, &in); err != nil {
			sum.Errors++
			continue
		}
		if it.Action == "error" || it.Action == "skip" {
			if it.Action == "skip" {
				sum.Skipped++
			} else {
				sum.Errors++
			}
			continue
		}
		_, action, err := s.repo.UpsertByNameInTx(ctx, tx, &in)
		if err != nil {
			sum.Errors++
			continue
		}
		if action == "create" {
			sum.Added++
		} else {
			sum.Updated++
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE price_import_jobs
		SET status='applied', applied_at=now(), added=$2, updated=$3, skipped=$4
		WHERE id=$1 AND status='validated'`, jobID, sum.Added, sum.Updated, sum.Skipped); err != nil {
		return sum, fmt.Errorf("update job: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return sum, fmt.Errorf("commit: %w", err)
	}
	return sum, nil
}

// Template returns a CSV template so managers know the exact column order.
func Template() []byte {
	var buf bytes.Buffer
	bufw := csv.NewWriter(&buf)
	_ = bufw.Write(Headers)
	_ = bufw.Write([]string{"Тормозная система", "Замена колодок (оси)", "Передние или задние, работа за ось", "3500", "RUB", "45", "да", "1"})
	_ = bufw.Write([]string{"Техническое обслуживание", "Замена масла ДВС", "Работа + промывка, без расходников", "600", "RUB", "30", "да", "1"})
	bufw.Flush()
	return buf.Bytes()
}

// Export builds a CSV of all active services for download.
func (s *Service) Export(ctx context.Context) ([]byte, error) {
	list, _, err := s.repo.List(ctx, service.ServiceListParams{Limit: 1000})
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write(Headers)
	for _, sv := range list {
		from := "нет"
		if sv.IsFromPrice {
			from = "да"
		}
		active := "0"
		if sv.IsActive {
			active = "1"
		}
		_ = w.Write([]string{
			sv.CategoryName, sv.Name, sv.Description,
			fmt.Sprintf("%d.%02d", sv.PriceRUB, sv.PriceCents%100),
			sv.Currency, fmt.Sprintf("%d", sv.DurationMin), from, active,
		})
	}
	w.Flush()
	return buf.Bytes(), nil
}
