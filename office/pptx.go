package office

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	mq "github.com/muqsitnawaz/mq/lib"
)

// PPTXParser parses PowerPoint .pptx files using stdlib only.
// PPTX is OOXML: a ZIP archive with XML slides at ppt/slides/slide*.xml.
type PPTXParser struct{}

// NewPPTXParser creates a new PPTX parser.
func NewPPTXParser() *PPTXParser {
	return &PPTXParser{}
}

// Format implements mq.FormatParser.
func (p *PPTXParser) Format() mq.Format {
	return mq.FormatPPTX
}

// ParseFile reads and parses a PPTX file.
func (p *PPTXParser) ParseFile(path string) (*mq.Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatPPTX, Path: path, Err: err}
	}
	return p.Parse(content, path)
}

// Parse parses PPTX content.
func (p *PPTXParser) Parse(content []byte, path string) (*mq.Document, error) {
	zr, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatPPTX, Path: path, Err: err}
	}

	// Find slide files and sort them
	var slideFiles []*zip.File
	for _, f := range zr.File {
		dir := filepath.Dir(f.Name)
		base := filepath.Base(f.Name)
		if dir == "ppt/slides" && strings.HasPrefix(base, "slide") && strings.HasSuffix(base, ".xml") {
			slideFiles = append(slideFiles, f)
		}
	}
	sort.Slice(slideFiles, func(i, j int) bool {
		return slideFiles[i].Name < slideFiles[j].Name
	})

	var headings []*mq.Heading
	var sections []*mq.Section
	var sb strings.Builder
	title := ""

	for i, sf := range slideFiles {
		rc, err := sf.Open()
		if err != nil {
			continue
		}
		slideXML, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		slideTitle, slideText := extractSlideContent(slideXML)
		slideNum := i + 1

		if slideTitle == "" {
			slideTitle = fmt.Sprintf("Slide %d", slideNum)
		}

		if title == "" && i == 0 {
			title = slideTitle
		}

		heading := &mq.Heading{
			Level: 1,
			Text:  fmt.Sprintf("Slide %d: %s", slideNum, slideTitle),
			Line:  slideNum,
		}
		headings = append(headings, heading)

		section := mq.NewSectionWithSource(heading, slideNum, slideNum, []byte(slideText))
		sections = append(sections, section)

		sb.WriteString(fmt.Sprintf("## Slide %d: %s\n", slideNum, slideTitle))
		if slideText != "" {
			sb.WriteString(slideText)
			sb.WriteByte('\n')
		}
		sb.WriteByte('\n')
	}

	if title == "" {
		title = "PowerPoint Presentation"
	}
	docTitle := fmt.Sprintf("PPTX (%d slides): %s", len(slideFiles), title)

	return mq.NewDocument(
		content, path, mq.FormatPPTX, docTitle,
		headings, sections, nil, nil, nil, nil, nil,
		sb.String(),
	), nil
}

// extractSlideContent extracts the title and body text from a slide XML.
// It looks for <a:t> text elements within shape trees.
func extractSlideContent(slideXML []byte) (title string, body string) {
	decoder := xml.NewDecoder(bytes.NewReader(slideXML))

	var allTexts []string
	var inTitle bool
	var currentText strings.Builder

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "ph": // placeholder
				for _, attr := range t.Attr {
					if attr.Name.Local == "type" {
						if attr.Value == "title" || attr.Value == "ctrTitle" {
							inTitle = true
						}
					}
				}
			case "t": // text element
				var text string
				if err := decoder.DecodeElement(&text, &t); err == nil && strings.TrimSpace(text) != "" {
					currentText.WriteString(text)
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "p": // end of paragraph
				text := strings.TrimSpace(currentText.String())
				if text != "" {
					if inTitle && title == "" {
						title = text
					} else {
						allTexts = append(allTexts, text)
					}
				}
				currentText.Reset()
				inTitle = false
			case "sp": // end of shape
				inTitle = false
			}
		}
	}

	body = strings.Join(allTexts, "\n")
	return
}
