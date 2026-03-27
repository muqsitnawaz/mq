// Package office provides parsers for office document formats (CSV, XLSX, DOCX, PPTX).
package office

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	mq "github.com/muqsitnawaz/mq/lib"
)

// CSVParser parses CSV and TSV files.
type CSVParser struct{}

// NewCSVParser creates a new CSV parser.
func NewCSVParser() *CSVParser {
	return &CSVParser{}
}

// Format implements mq.FormatParser.
func (p *CSVParser) Format() mq.Format {
	return mq.FormatCSV
}

// ParseFile reads and parses a CSV file.
func (p *CSVParser) ParseFile(path string) (*mq.Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatCSV, Path: path, Err: err}
	}
	return p.Parse(content, path)
}

// Parse parses CSV content.
func (p *CSVParser) Parse(content []byte, path string) (*mq.Document, error) {
	reader := csv.NewReader(strings.NewReader(string(content)))

	// Detect TSV
	if strings.HasSuffix(strings.ToLower(path), ".tsv") {
		reader.Comma = '\t'
	}

	// Read all records
	var records [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, &mq.ParseError{Format: mq.FormatCSV, Path: path, Err: err}
		}
		records = append(records, record)
	}

	if len(records) == 0 {
		return mq.NewDocument(content, path, mq.FormatCSV, "CSV (empty)", nil, nil, nil, nil, nil, nil, nil, ""), nil
	}

	// First row as headers
	headers := records[0]
	dataRows := records[1:]

	title := fmt.Sprintf("CSV (%d rows x %d columns)", len(dataRows), len(headers))

	// Build a table
	var tables []*mq.Table
	tableRows := make([][]string, len(dataRows))
	for i, row := range dataRows {
		tableRows[i] = row
	}
	tables = append(tables, &mq.Table{
		Headers: headers,
		Rows:    tableRows,
	})

	// Build readable text: header line + first few rows
	var sb strings.Builder
	sb.WriteString(strings.Join(headers, " | "))
	sb.WriteByte('\n')
	limit := len(dataRows)
	if limit > 20 {
		limit = 20
	}
	for _, row := range dataRows[:limit] {
		sb.WriteString(strings.Join(row, " | "))
		sb.WriteByte('\n')
	}
	if len(dataRows) > 20 {
		sb.WriteString(fmt.Sprintf("... and %d more rows\n", len(dataRows)-20))
	}

	// Column headers as H2 headings
	var headings []*mq.Heading
	for i, h := range headers {
		headings = append(headings, &mq.Heading{
			Level: 2,
			Text:  h,
			Line:  1,
			ID:    fmt.Sprintf("col-%d", i),
		})
	}

	return mq.NewDocument(
		content, path, mq.FormatCSV, title,
		headings, nil, nil, nil, nil, tables, nil,
		sb.String(),
	), nil
}
