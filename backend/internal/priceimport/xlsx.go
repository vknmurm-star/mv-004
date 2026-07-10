package priceimport

import (
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

// ParseXLSX mirrors ParseCSV but reads an .xlsx workbook (first sheet).
func ParseXLSX(r io.Reader, cats categoryResolver) ([]ParsedRow, Summary, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, Summary{}, fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, Summary{}, fmt.Errorf("xlsx has no sheets")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, Summary{}, fmt.Errorf("read sheet: %w", err)
	}
	if len(rows) == 0 {
		return nil, Summary{}, fmt.Errorf("empty sheet")
	}

	idx := mapHeader(rows[0])
	if idx == nil {
		return nil, Summary{}, fmt.Errorf("missing required columns; expected: %v", Headers)
	}

	out := make([]ParsedRow, 0)
	summary := Summary{}
	for line, rec := range rows[1:] {
		line += 2 // header is line 1; data starts at line 2
		summary.Total++
		row, _ := buildRow(line, rec, idx, cats)
		if len(row.Errors) > 0 {
			row.Action = "error"
			summary.Errors++
		} else if !row.Payload.IsActive {
			row.Action = "skip"
			summary.Skipped++
		}
		out = append(out, row)
	}
	return out, summary, nil
}
