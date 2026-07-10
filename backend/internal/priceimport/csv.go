package priceimport

import (
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/timemachine-auto/timemachine/internal/domain/service"
)

// Column order for both CSV and XLSX. The template and parser use the same
// canonical header set so import/export are symmetric.
var (
	Headers = []string{
		"category", "name", "description", "price", "currency",
		"duration_minutes", "is_from_price", "is_active",
	}
)

// RowError is a per-row validation problem shown in the preview UI.
type RowError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ParsedRow is a validated import row ready to be persisted.
type ParsedRow struct {
	RowNumber int                  `json:"row_number"`
	Action    string               `json:"action"` // create | update | skip | error
	Payload   service.ServiceInput `json:"payload"`
	Errors    []RowError           `json:"errors"`
}

// Summary aggregates import results.
type Summary struct {
	Total   int `json:"total"`
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
	Errors  int `json:"errors"`
}

var priceRE = regexp.MustCompile(`[^\d.]`)

// normalizePrice parses human prices like "1 500 ₽", "от 1500", "1500.00".
func normalizePrice(s string) (int64, error) {
	s = priceRE.ReplaceAllString(strings.TrimSpace(s), "")
	if s == "" {
		return 0, fmt.Errorf("empty price")
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", s)
	}
	return int64(f * 100), nil
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "да", "yes", "1", "true", "y", "от":
		return true
	}
	return false
}

// categoryResolver maps a category name/slug to its id.
type categoryResolver interface {
	Resolve(name string) (int64, error)
}

// ParseCSV parses the import file into rows. It is pure (no side effects),
// so it can be reused for preview and apply steps.
func ParseCSV(r io.Reader, cats categoryResolver) ([]ParsedRow, Summary, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err != nil {
		return nil, Summary{}, fmt.Errorf("read header: %w", err)
	}
	idx := mapHeader(header)
	if idx == nil {
		return nil, Summary{}, fmt.Errorf("missing required columns; expected: %s",
			strings.Join(Headers, ", "))
	}

	rows := make([]ParsedRow, 0)
	summary := Summary{}
	line := 1

	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, Summary{}, fmt.Errorf("read row %d: %w", line, err)
		}
		line++
		summary.Total++

		for i := range rec {
			rec[i] = strings.TrimSpace(rec[i])
		}

		row, _ := buildRow(line, rec, idx, cats)
		if len(row.Errors) > 0 {
			row.Action = "error"
			summary.Errors++
		} else if !row.Payload.IsActive {
			row.Action = "skip"
			summary.Skipped++
		}
		rows = append(rows, row)
	}
	return rows, summary, nil
}

func mapHeader(h []string) map[string]int {
	idx := make(map[string]int, len(h))
	want := make(map[string]struct{}, len(Headers))
	for _, hh := range Headers {
		want[strings.ToLower(hh)] = struct{}{}
	}
	for i, c := range h {
		k := strings.ToLower(strings.TrimSpace(c))
		if _, ok := want[k]; ok {
			idx[k] = i
		}
	}
	for k := range want {
		if _, ok := idx[k]; !ok {
			return nil
		}
	}
	return idx
}

func cell(rec []string, idx map[string]int, key string) string {
	i, ok := idx[key]
	if !ok || i >= len(rec) {
		return ""
	}
	return rec[i]
}

func buildRow(line int, rec []string, idx map[string]int, cats categoryResolver) (ParsedRow, error) {
	row := ParsedRow{RowNumber: line, Action: "create"}
	in := service.ServiceInput{}

	catName := strings.TrimSpace(cell(rec, idx, "category"))
	if catName == "" {
		row.Errors = append(row.Errors, RowError{Field: "category", Message: "обязательное поле"})
	} else {
		id, err := cats.Resolve(catName)
		if err != nil {
			row.Errors = append(row.Errors, RowError{Field: "category", Message: err.Error()})
		} else {
			in.CategoryID = id
		}
	}

	in.Name = strings.TrimSpace(cell(rec, idx, "name"))
	if in.Name == "" {
		row.Errors = append(row.Errors, RowError{Field: "name", Message: "обязательное поле"})
	}
	in.Description = cell(rec, idx, "description")
	in.LongDescription = ""

	priceCents, perr := normalizePrice(cell(rec, idx, "price"))
	if perr != nil {
		row.Errors = append(row.Errors, RowError{Field: "price", Message: "неверный формат цены"})
	} else {
		in.PriceCents = priceCents
	}

	in.Currency = strings.ToUpper(strings.TrimSpace(cell(rec, idx, "currency")))
	if in.Currency == "" {
		in.Currency = "RUB"
	}
	if in.Currency != "RUB" && in.Currency != "USD" && in.Currency != "EUR" {
		row.Errors = append(row.Errors, RowError{Field: "currency", Message: "допустимо: RUB, USD, EUR"})
	}

	if dur := strings.TrimSpace(cell(rec, idx, "duration_minutes")); dur != "" {
		d, err := strconv.Atoi(dur)
		if err != nil || d < 0 {
			row.Errors = append(row.Errors, RowError{Field: "duration_minutes", Message: "должно быть числом минут"})
		} else {
			in.DurationMin = d
		}
	} else {
		in.DurationMin = 60
	}

	in.IsFromPrice = parseBool(cell(rec, idx, "is_from_price"))
	active := cell(rec, idx, "is_active")
	switch strings.ToLower(active) {
	case "", "да", "yes", "1", "true":
		in.IsActive = true
	case "нет", "no", "0", "false":
		in.IsActive = false
	default:
		in.IsActive = true
	}

	row.Payload = in
	_ = in.Validate()
	return row, nil
}
