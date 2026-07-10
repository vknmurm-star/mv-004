package priceimport

import (
	"bytes"
	"strings"
	"testing"
)

// stubResolver maps names to ids; satisfies categoryResolver.
type stubResolver struct {
	byName map[string]int64
}

func (s *stubResolver) Resolve(name string) (int64, error) {
	n := strings.TrimSpace(name)
	if id, ok := s.byName[strings.ToLower(n)]; ok {
		return id, nil
	}
	return 0, &resolveErr{name: n}
}

type resolveErr struct{ name string }

func (e *resolveErr) Error() string { return "категория не найдена: " + e.name }

func TestNormalizePrice(t *testing.T) {
	cases := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"1500", 150000, true},
		{"1 500 ₽", 150000, true},
		{"от 1500", 150000, true},
		{"1500.00", 150000, true},
		{"3 500.50", 350050, true},
		{"", 0, false},
		{"abc", 0, false},
		// "1.500.00" is rejected (multiple decimal points) — the parser
		// refuses ambiguous inputs rather than silently producing a wrong price.
		{"1.500.00", 0, false},
	}
	for _, c := range cases {
		got, err := normalizePrice(c.in)
		if c.ok && err != nil {
			t.Errorf("normalizePrice(%q): unexpected err %v", c.in, err)
			continue
		}
		if !c.ok && err == nil {
			t.Errorf("normalizePrice(%q): expected error, got %d", c.in, got)
			continue
		}
		if c.ok && got != c.want {
			t.Errorf("normalizePrice(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestMapHeader_MissingColumnRejected(t *testing.T) {
	if got := mapHeader([]string{"name", "price", "currency"}); got != nil {
		t.Fatalf("expected nil for missing columns, got %v", got)
	}
}

func TestMapHeader_DuplicateColumnsLastWins(t *testing.T) {
	// Known behavior (documented in REQUIRES_DECISION): duplicate columns
	// overwrite — the last index wins. This test pins current behavior.
	idx := mapHeader([]string{"name", "name", "category", "description", "price", "currency", "duration_minutes", "is_from_price", "is_active"})
	if idx == nil {
		t.Fatal("expected non-nil header map")
	}
	if idx["name"] != 1 {
		t.Errorf("duplicate name col: expected last index 1, got %d", idx["name"])
	}
}

func TestParseCSV_RequiredColumns(t *testing.T) {
	cats := &stubResolver{byName: map[string]int64{"тормозная система": 1}}
	const csvData = `category,name,description,price,currency,duration_minutes,is_from_price,is_active
Тормозная система,Замена колодок,Передние или задние,3500,RUB,45,да,1
Тормозная система,,Без названия,1000,RUB,30,нет,1
Неизвестная,Услуга,Описание,500,RUB,15,нет,1
`
	rows, sum, err := ParseCSV(strings.NewReader(csvData), cats)
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if sum.Total != 3 {
		t.Errorf("total = %d, want 3", sum.Total)
	}
	// row 1: valid create
	if rows[0].Action != "create" || rows[0].Payload.Name != "Замена колодок" {
		t.Errorf("row1: action=%s name=%q", rows[0].Action, rows[0].Payload.Name)
	}
	// row 2: empty name → error
	if rows[1].Action != "error" || len(rows[1].Errors) == 0 {
		t.Errorf("row2: expected error, got action=%s errors=%v", rows[1].Action, rows[1].Errors)
	}
	// row 3: unknown category → error
	if rows[2].Action != "error" {
		t.Errorf("row3: expected error, got %s", rows[2].Action)
	}
	if sum.Errors != 2 {
		t.Errorf("summary.Errors = %d, want 2", sum.Errors)
	}
}

func TestParseCSV_InactiveRowSkipped(t *testing.T) {
	cats := &stubResolver{byName: map[string]int64{"то": 1}}
	const csvData = `category,name,description,price,currency,duration_minutes,is_from_price,is_active
ТО,Замена масла,Работа,600,RUB,30,да,0
`
	rows, sum, err := ParseCSV(strings.NewReader(csvData), cats)
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if sum.Total != 1 || len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d (sum=%d)", len(rows), sum.Total)
	}
	if rows[0].Action != "skip" {
		t.Errorf("inactive row action = %s, want skip", rows[0].Action)
	}
}

func TestDetectFormat(t *testing.T) {
	if DetectFormat("prices.csv") != FormatCSV {
		t.Error("csv detection failed")
	}
	if DetectFormat("prices.xlsx") != FormatXLSX {
		t.Error("xlsx detection failed")
	}
	// Unknown extension falls back to csv.
	if DetectFormat("prices.unknown") != FormatCSV {
		t.Error("fallback to csv failed")
	}
}

func TestTemplate_HeaderMatchesParser(t *testing.T) {
	tpl := Template()
	if !bytes.Contains(tpl, []byte("category")) {
		t.Fatal("template missing 'category' header")
	}
	// Parser must accept the template we ship — invariant.
	rows, _, err := ParseCSV(bytes.NewReader(tpl),
		&stubResolver{byName: map[string]int64{
			"тормозная система":                         1,
			strings.ToLower("Тормозная система"):        1, // case-insensitive guard
			"техническое обслуживание":                  2,
			strings.ToLower("Техническое обслуживание"): 2,
		}})
	if err != nil {
		t.Fatalf("template not parseable: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("template should have 2 sample rows, got %d", len(rows))
	}
	for i, r := range rows {
		if r.Action == "error" {
			t.Errorf("template row %d errored: %v", i, r.Errors)
		}
	}
}
