package office

import (
	"bytes"
	"fmt"
	"strings"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/xuri/excelize/v2"
)

// XLSXParser parses Excel .xlsx files.
type XLSXParser struct{}

// NewXLSXParser creates a new XLSX parser.
func NewXLSXParser() *XLSXParser {
	return &XLSXParser{}
}

// Format implements mq.FormatParser.
func (p *XLSXParser) Format() mq.Format {
	return mq.FormatXLSX
}

// ParseFile reads and parses an XLSX file.
func (p *XLSXParser) ParseFile(path string) (*mq.Document, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatXLSX, Path: path, Err: err}
	}
	defer f.Close()

	return p.parseWorkbook(f, nil, path)
}

// Parse parses XLSX content from bytes.
func (p *XLSXParser) Parse(content []byte, path string) (*mq.Document, error) {
	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatXLSX, Path: path, Err: err}
	}
	defer f.Close()

	return p.parseWorkbook(f, content, path)
}

func (p *XLSXParser) parseWorkbook(f *excelize.File, content []byte, path string) (*mq.Document, error) {
	sheets := f.GetSheetList()

	var headings []*mq.Heading
	var sections []*mq.Section
	var tables []*mq.Table
	var sb strings.Builder
	totalRows := 0

	for _, sheetName := range sheets {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			continue
		}

		// Sheet name as H1
		heading := &mq.Heading{
			Level: 1,
			Text:  sheetName,
			Line:  totalRows + 1,
		}
		headings = append(headings, heading)

		startLine := totalRows + 1

		if len(rows) > 0 {
			// First row as table headers
			headers := rows[0]
			var dataRows [][]string
			for _, row := range rows[1:] {
				dataRows = append(dataRows, row)
			}

			tables = append(tables, &mq.Table{
				Headers: headers,
				Rows:    dataRows,
			})

			// Column headers as H2
			for _, h := range headers {
				if strings.TrimSpace(h) != "" {
					headings = append(headings, &mq.Heading{
						Level: 2,
						Text:  h,
						Line:  totalRows + 1,
					})
				}
			}

			// Readable text
			sb.WriteString(fmt.Sprintf("## %s\n", sheetName))
			sb.WriteString(strings.Join(headers, " | "))
			sb.WriteByte('\n')
			limit := len(dataRows)
			if limit > 10 {
				limit = 10
			}
			for _, row := range dataRows[:limit] {
				sb.WriteString(strings.Join(row, " | "))
				sb.WriteByte('\n')
			}
			if len(dataRows) > 10 {
				sb.WriteString(fmt.Sprintf("... and %d more rows\n", len(dataRows)-10))
			}
			sb.WriteByte('\n')

			totalRows += len(rows)
		}

		endLine := totalRows
		if endLine < startLine {
			endLine = startLine
		}

		section := mq.NewSectionWithSource(heading, startLine, endLine, []byte(sb.String()))
		sections = append(sections, section)
	}

	title := fmt.Sprintf("XLSX (%d sheets, %d rows)", len(sheets), totalRows)

	return mq.NewDocument(
		content, path, mq.FormatXLSX, title,
		headings, sections, nil, nil, nil, tables, nil,
		sb.String(),
	), nil
}
