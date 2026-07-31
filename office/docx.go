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
type DOCXParser struct{}

func NewDOCXParser() *DOCXParser { return &DOCXParser{} }

func (p *DOCXParser) Format() mq.Format { return mq.FormatDOCX }

func (p *DOCXParser) ParseFile(path string) (*mq.Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatDOCX, Path: path, Err: err}
	}
	return p.Parse(content, path)
}

func (p *DOCXParser) Parse(content []byte, path string) (*mq.Document, error) {
	zr, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatDOCX, Path: path, Err: err}
	}

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

// parseDocumentXML walks the XML token stream to extract paragraphs and tables.
// We use raw token walking because OOXML uses namespaced elements (w:p, w:r, etc.)
// and Go's xml.Decoder.DecodeElement requires exact namespace matching in struct tags.
func (p *DOCXParser) parseDocumentXML(docXML, content []byte, path string) (*mq.Document, error) {
	decoder := xml.NewDecoder(bytes.NewReader(docXML))

	var headings []*mq.Heading
	var sections []*mq.Section
	var tables []*mq.Table
	var sb strings.Builder
	lineNum := 0
	title := ""
	var currentSection *mq.Section

	// State for tracking where we are in the XML tree
	type state int
	const (
		stBody state = iota
		stParagraph
		stParagraphProps
		stRun
		stText
		stTable
		stTableRow
		stTableCell
	)

	var stack []state
	push := func(s state) { stack = append(stack, s) }
	pop := func() {
		if len(stack) > 0 {
			stack = stack[:len(stack)-1]
		}
	}
	current := func() state {
		if len(stack) == 0 {
			return stBody
		}
		return stack[len(stack)-1]
	}

	var paraStyle string     // style of current paragraph
	var paraTexts []string   // text runs in current paragraph
	var tableRows [][]string // rows of current table
	var rowCells []string    // cells of current table row
	var cellText []string    // text in current cell
	inTable := false

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			local := t.Name.Local
			switch local {
			case "p":
				if inTable {
					// paragraph inside table cell -- just collect text
				} else {
					push(stParagraph)
					paraStyle = ""
					paraTexts = nil
				}
			case "pPr":
				push(stParagraphProps)
			case "pStyle":
				for _, attr := range t.Attr {
					if attr.Name.Local == "val" {
						paraStyle = attr.Value
					}
				}
			case "r":
				push(stRun)
			case "t":
				push(stText)
			case "tbl":
				inTable = true
				tableRows = nil
			case "tr":
				rowCells = nil
			case "tc":
				cellText = nil
			}

		case xml.EndElement:
			local := t.Name.Local
			switch local {
			case "p":
				if inTable {
					// end of paragraph inside table cell
				} else if current() == stParagraph {
					pop()
					text := strings.Join(paraTexts, "")
					if text == "" {
						break
					}

					lineNum++
					headingLevel := docxStyleToLevel(paraStyle)

					if headingLevel > 0 {
						if currentSection != nil {
							currentSection.End = lineNum - 1
							if currentSection.End < currentSection.Start {
								currentSection.End = currentSection.Start
							}
						}
						heading := &mq.Heading{Level: headingLevel, Text: text, Line: lineNum}
						headings = append(headings, heading)
						if title == "" && headingLevel == 1 {
							title = text
						}
						currentSection = mq.NewSectionWithSource(heading, lineNum, lineNum, []byte(sb.String()))
						sections = append(sections, currentSection)
						sb.WriteString(fmt.Sprintf("%s %s\n", strings.Repeat("#", headingLevel), text))
					} else {
						sb.WriteString(text)
						sb.WriteByte('\n')
					}
				}
			case "pPr":
				if current() == stParagraphProps {
					pop()
				}
			case "r":
				if current() == stRun {
					pop()
				}
			case "t":
				if current() == stText {
					pop()
				}
			case "tbl":
				inTable = false
				if len(tableRows) > 0 {
					var headers []string
					var dataRows [][]string
					headers = tableRows[0]
					if len(tableRows) > 1 {
						dataRows = tableRows[1:]
					}
					tables = append(tables, &mq.Table{Headers: headers, Rows: dataRows})
					lineNum++
				}
			case "tr":
				tableRows = append(tableRows, rowCells)
			case "tc":
				cellContent := strings.Join(cellText, " ")
				rowCells = append(rowCells, cellContent)
			}

		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text == "" {
				break
			}
			if inTable {
				cellText = append(cellText, text)
			} else if current() == stText {
				paraTexts = append(paraTexts, text)
			}
		}
	}

	if currentSection != nil {
		currentSection.End = lineNum
	}

	if title == "" {
		title = "Word Document"
	}

	return mq.NewDocument(
		content, path, mq.FormatDOCX, title,
		headings, sections, nil, nil, nil, tables, nil,
		sb.String(),
	), nil
}

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
