package office

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	mq "github.com/muqsitnawaz/mq/lib"
)

// DOCXParser parses Word .docx files using stdlib only (archive/zip + encoding/xml).
// DOCX is OOXML: a ZIP archive containing XML files.
type DOCXParser struct{}

// NewDOCXParser creates a new DOCX parser.
func NewDOCXParser() *DOCXParser {
	return &DOCXParser{}
}

// Format implements mq.FormatParser.
func (p *DOCXParser) Format() mq.Format {
	return mq.FormatDOCX
}

// ParseFile reads and parses a DOCX file.
func (p *DOCXParser) ParseFile(path string) (*mq.Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatDOCX, Path: path, Err: err}
	}
	return p.Parse(content, path)
}

// Parse parses DOCX content.
func (p *DOCXParser) Parse(content []byte, path string) (*mq.Document, error) {
	zr, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatDOCX, Path: path, Err: err}
	}

	// Find and read word/document.xml
	var docXML []byte
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return nil, &mq.ParseError{Format: mq.FormatDOCX, Path: path, Err: err}
			}
			docXML, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, &mq.ParseError{Format: mq.FormatDOCX, Path: path, Err: err}
			}
			break
		}
	}

	if docXML == nil {
		return nil, &mq.ParseError{Format: mq.FormatDOCX, Path: path, Err: fmt.Errorf("word/document.xml not found")}
	}

	return p.parseDocumentXML(docXML, content, path)
}

// OOXML types for minimal parsing
type docxBody struct {
	XMLName xml.Name      `xml:"body"`
	Items   []docxBodyItem `xml:",any"`
}

type docxDocument struct {
	XMLName xml.Name `xml:"document"`
	Body    docxBody `xml:"body"`
}

type docxBodyItem struct {
	XMLName xml.Name
	Inner   []byte `xml:",innerxml"`
}

type docxParagraph struct {
	Properties docxParagraphProperties `xml:"pPr"`
	Runs       []docxRun               `xml:"r"`
	Hyperlinks []docxHyperlink         `xml:"hyperlink"`
}

type docxParagraphProperties struct {
	Style docxStyle `xml:"pStyle"`
}

type docxStyle struct {
	Val string `xml:"val,attr"`
}

type docxRun struct {
	Text     []docxText `xml:"t"`
	RunProps docxRunProps `xml:"rPr"`
}

type docxRunProps struct {
	Bold *struct{} `xml:"b"`
}

type docxText struct {
	Content string `xml:",chardata"`
}

type docxHyperlink struct {
	Runs []docxRun `xml:"r"`
}

type docxTableRow struct {
	Cells []docxTableCell `xml:"tc"`
}

type docxTableCell struct {
	Paragraphs []docxParagraph `xml:"p"`
}

func (p *DOCXParser) parseDocumentXML(docXML, content []byte, path string) (*mq.Document, error) {
	decoder := xml.NewDecoder(bytes.NewReader(docXML))

	var headings []*mq.Heading
	var sections []*mq.Section
	var tables []*mq.Table
	var links []*mq.Link
	var sb strings.Builder
	lineNum := 0
	title := ""

	var currentSection *mq.Section
	sectionStart := 1

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		localName := se.Name.Local

		switch localName {
		case "p": // paragraph
			var para docxParagraph
			if err := decoder.DecodeElement(&para, &se); err != nil {
				continue
			}

			text := extractParagraphText(&para)
			if text == "" {
				continue
			}

			lineNum++
			style := para.Properties.Style.Val

			// Detect headings by style name
			headingLevel := docxStyleToLevel(style)
			if headingLevel > 0 {
				// Close previous section
				if currentSection != nil {
					currentSection.End = lineNum - 1
					if currentSection.End < currentSection.Start {
						currentSection.End = currentSection.Start
					}
				}

				heading := &mq.Heading{
					Level: headingLevel,
					Text:  text,
					Line:  lineNum,
				}
				headings = append(headings, heading)

				if title == "" && headingLevel == 1 {
					title = text
				}

				sectionStart = lineNum
				currentSection = mq.NewSectionWithSource(heading, sectionStart, lineNum, []byte(sb.String()))
				sections = append(sections, currentSection)

				sb.WriteString(fmt.Sprintf("%s %s\n", strings.Repeat("#", headingLevel), text))
			} else {
				sb.WriteString(text)
				sb.WriteByte('\n')
			}

			// Extract hyperlinks
			for _, hl := range para.Hyperlinks {
				linkText := extractRunsText(hl.Runs)
				if linkText != "" {
					links = append(links, &mq.Link{Text: linkText})
				}
			}

		case "tbl": // table
			table := p.parseTable(decoder, &se)
			if table != nil {
				tables = append(tables, table)
				lineNum++
				// Add table to readable text
				if len(table.Headers) > 0 {
					sb.WriteString(strings.Join(table.Headers, " | "))
					sb.WriteByte('\n')
				}
				for _, row := range table.Rows {
					sb.WriteString(strings.Join(row, " | "))
					sb.WriteByte('\n')
				}
			}

		default:
			decoder.Skip()
		}
	}

	// Close last section
	if currentSection != nil {
		currentSection.End = lineNum
	}

	if title == "" {
		title = "Word Document"
	}

	return mq.NewDocument(
		content, path, mq.FormatDOCX, title,
		headings, sections, nil, links, nil, tables, nil,
		sb.String(),
	), nil
}

func (p *DOCXParser) parseTable(decoder *xml.Decoder, start *xml.StartElement) *mq.Table {
	// Read the raw XML for this table element
	var rawTable struct {
		Rows []docxTableRow `xml:"tr"`
	}
	if err := decoder.DecodeElement(&rawTable, start); err != nil {
		return nil
	}

	if len(rawTable.Rows) == 0 {
		return nil
	}

	// First row as headers
	var headers []string
	for _, cell := range rawTable.Rows[0].Cells {
		var cellText []string
		for _, para := range cell.Paragraphs {
			t := extractParagraphText(&para)
			if t != "" {
				cellText = append(cellText, t)
			}
		}
		headers = append(headers, strings.Join(cellText, " "))
	}

	var rows [][]string
	for _, row := range rawTable.Rows[1:] {
		var cells []string
		for _, cell := range row.Cells {
			var cellText []string
			for _, para := range cell.Paragraphs {
				t := extractParagraphText(&para)
				if t != "" {
					cellText = append(cellText, t)
				}
			}
			cells = append(cells, strings.Join(cellText, " "))
		}
		rows = append(rows, cells)
	}

	return &mq.Table{
		Headers: headers,
		Rows:    rows,
	}
}

func extractParagraphText(para *docxParagraph) string {
	var parts []string
	for _, run := range para.Runs {
		parts = append(parts, extractRunText(run))
	}
	for _, hl := range para.Hyperlinks {
		parts = append(parts, extractRunsText(hl.Runs))
	}
	return strings.Join(parts, "")
}

func extractRunText(run docxRun) string {
	var parts []string
	for _, t := range run.Text {
		parts = append(parts, t.Content)
	}
	return strings.Join(parts, "")
}

func extractRunsText(runs []docxRun) string {
	var parts []string
	for _, run := range runs {
		parts = append(parts, extractRunText(run))
	}
	return strings.Join(parts, "")
}

// docxStyleToLevel maps Word paragraph style names to heading levels.
func docxStyleToLevel(style string) int {
	style = strings.ToLower(style)
	switch {
	case style == "heading1" || style == "heading 1" || style == "title":
		return 1
	case style == "heading2" || style == "heading 2" || style == "subtitle":
		return 2
	case style == "heading3" || style == "heading 3":
		return 3
	case style == "heading4" || style == "heading 4":
		return 4
	case style == "heading5" || style == "heading 5":
		return 5
	case style == "heading6" || style == "heading 6":
		return 6
	default:
		return 0
	}
}
