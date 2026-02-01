// Package pdf provides PDF parsing for mq with structure inference.
//
// Unlike HTML and Markdown, PDFs don't have explicit semantic structure.
// This parser infers structure from visual cues:
//   - Headings: Larger/bolder text, followed by smaller text
//   - Sections: Content grouped under headings
//   - Tables: Aligned text in grid patterns
//   - Lists: Lines starting with bullets or numbers
//   - Links: PDF annotation objects
//
// Example:
//
//	parser := pdf.NewParser()
//	doc, _ := parser.ParseFile("document.pdf")
//
//	// Same queries as markdown and HTML
//	headings := doc.GetHeadings()
//	readable := doc.ReadableText()
package pdf

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	mq "github.com/muqsitnawaz/mq/lib"
)

// Parser parses PDF documents into mq.Document.
type Parser struct {
	// Options
	inferHeadings   bool    // Infer headings from font size changes
	inferTables     bool    // Detect tables from aligned text
	headingMinRatio float64 // Min font size ratio to consider heading (e.g., 1.2 = 20% larger)
}

// Option configures the parser.
type Option func(*Parser)

// NewParser creates a new PDF parser with default options.
func NewParser(opts ...Option) *Parser {
	p := &Parser{
		inferHeadings:   true,
		inferTables:     true,
		headingMinRatio: 1.15, // 15% larger = heading
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithHeadingInference enables/disables heading detection from font sizes.
func WithHeadingInference(enabled bool) Option {
	return func(p *Parser) {
		p.inferHeadings = enabled
	}
}

// WithTableDetection enables/disables table detection.
func WithTableDetection(enabled bool) Option {
	return func(p *Parser) {
		p.inferTables = enabled
	}
}

// WithHeadingRatio sets the minimum font size ratio for heading detection.
// A ratio of 1.2 means text must be at least 20% larger than body text.
func WithHeadingRatio(ratio float64) Option {
	return func(p *Parser) {
		p.headingMinRatio = ratio
	}
}

// Format implements mq.FormatParser.
func (p *Parser) Format() mq.Format {
	return mq.FormatPDF
}

// ParseFile reads and parses a PDF file.
func (p *Parser) ParseFile(path string) (*mq.Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatPDF, Path: path, Err: err}
	}
	return p.Parse(content, path)
}

// Parse parses PDF content and returns an mq.Document.
//
// NOTE: This is a design stub. Full implementation would use pdfcpu or
// similar library for PDF parsing. The key insight is that PDF structure
// must be INFERRED from visual cues, unlike HTML/Markdown which have
// explicit structure.
func (p *Parser) Parse(content []byte, path string) (*mq.Document, error) {
	ext := &extractor{
		parser: p,
		source: content,
		path:   path,
	}

	return ext.extract()
}

// textRun represents a chunk of text with position and style.
// In a full implementation, this would be populated from PDF content streams.
type textRun struct {
	text     string
	page     int
	x, y     float64 // Position
	fontSize float64
	fontName string
	isBold   bool
	isItalic bool
}

// extractor extracts content from PDF.
type extractor struct {
	parser *Parser
	source []byte
	path   string

	// Raw extracted data
	textRuns []textRun
	title    string
}

// structureResult holds the JSON output from extract_structure.py
type structureResult struct {
	Title        string  `json:"title"`
	BodyFontSize float64 `json:"body_font_size"`
	Headings     []struct {
		Level    int     `json:"level"`
		Text     string  `json:"text"`
		Page     int     `json:"page"`
		FontSize float64 `json:"font_size"`
	} `json:"headings"`
	Tables []struct {
		Page    int      `json:"page"`
		Rows    int      `json:"rows"`
		Cols    int      `json:"cols"`
		Headers []string `json:"headers"`
	} `json:"tables"`
	PageCount int `json:"page_count"`
}

func (e *extractor) extract() (*mq.Document, error) {
	// Extract text content using pdftotext (fast, reliable)
	text := e.extractBasicText()

	// Try to extract structure using PyMuPDF (headings, tables)
	var headings []*mq.Heading
	var sections []*mq.Section
	var tables []*mq.Table

	structure := e.extractStructure()
	if structure != nil {
		e.title = structure.Title

		// Convert headings
		for _, h := range structure.Headings {
			headings = append(headings, &mq.Heading{
				Level: h.Level,
				Text:  h.Text,
			})
		}

		// Build sections from headings
		sections = e.buildSections(headings)

		// Convert tables
		for _, t := range structure.Tables {
			tables = append(tables, &mq.Table{
				Headers: t.Headers,
				Rows:    nil, // We don't extract full table data yet
			})
		}
	}

	return mq.NewDocument(
		e.source,
		e.path,
		mq.FormatPDF,
		e.title,
		headings,
		sections,
		nil, // codeBlocks - would be detected from monospace
		nil, // links - would be extracted from annotations
		nil, // images - would be extracted from PDF
		tables,
		nil, // lists - would be detected from bullets
		text,
	), nil
}

// extractStructure uses PyMuPDF to extract headings and tables.
func (e *extractor) extractStructure() *structureResult {
	// Check if content looks like a PDF
	if len(e.source) < 4 || string(e.source[:4]) != "%PDF" {
		return nil
	}

	// Find the Python script relative to this Go file
	scriptPath := e.findPythonScript()
	if scriptPath == "" {
		return nil
	}

	// Run the Python script
	cmd := exec.Command("python3", scriptPath, "-")
	cmd.Stdin = bytes.NewReader(e.source)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// PyMuPDF not installed or script failed
		return nil
	}

	// Parse JSON output
	var result structureResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil
	}

	return &result
}

// findPythonScript locates extract_structure.py.
func (e *extractor) findPythonScript() string {
	// Try multiple locations:
	// 1. Same directory as the executable
	// 2. pdf/ subdirectory relative to executable
	// 3. Relative to current working directory
	// 4. Source location (for development)

	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)

		// Check same directory as executable
		scriptPath := filepath.Join(execDir, "extract_structure.py")
		if _, err := os.Stat(scriptPath); err == nil {
			return scriptPath
		}

		// Check pdf/ subdirectory
		scriptPath = filepath.Join(execDir, "pdf", "extract_structure.py")
		if _, err := os.Stat(scriptPath); err == nil {
			return scriptPath
		}
	}

	// Check relative to current working directory
	cwd, err := os.Getwd()
	if err == nil {
		scriptPath := filepath.Join(cwd, "pdf", "extract_structure.py")
		if _, err := os.Stat(scriptPath); err == nil {
			return scriptPath
		}
	}

	// Try source location (for development)
	_, thisFile, _, ok := runtime.Caller(0)
	if ok {
		dir := filepath.Dir(thisFile)
		scriptPath := filepath.Join(dir, "extract_structure.py")
		if _, err := os.Stat(scriptPath); err == nil {
			return scriptPath
		}
	}

	return ""
}

// buildSections creates section hierarchy from headings.
func (e *extractor) buildSections(headings []*mq.Heading) []*mq.Section {
	if len(headings) == 0 {
		return nil
	}

	var sections []*mq.Section
	var stack []*mq.Section

	for _, h := range headings {
		s := &mq.Section{Heading: h}

		// Pop stack until we find a parent with lower level
		for len(stack) > 0 && stack[len(stack)-1].Heading.Level >= h.Level {
			stack = stack[:len(stack)-1]
		}

		// If stack has items, the top is our parent
		if len(stack) > 0 {
			parent := stack[len(stack)-1]
			s.Parent = parent
			parent.Children = append(parent.Children, s)
		}

		stack = append(stack, s)
		sections = append(sections, s)
	}

	return sections
}

// extractBasicText extracts text content from PDF using pdftotext (poppler).
func (e *extractor) extractBasicText() string {
	// Check if content looks like a PDF
	if len(e.source) < 4 || string(e.source[:4]) != "%PDF" {
		return ""
	}

	// Use pdftotext CLI (poppler-utils)
	// -layout preserves the original physical layout
	// -nopgbrk removes page break characters
	// - (stdin) and - (stdout)
	cmd := exec.Command("pdftotext", "-layout", "-nopgbrk", "-", "-")
	cmd.Stdin = bytes.NewReader(e.source)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// pdftotext not installed or failed
		return ""
	}

	return stdout.String()
}

// Ensure Parser implements mq.FormatParser
var _ mq.FormatParser = (*Parser)(nil)

// ParsePDF is a convenience function for quick parsing.
func ParsePDF(content []byte, path string) (*mq.Document, error) {
	return NewParser().Parse(content, path)
}

// ParsePDFFile is a convenience function for quick file parsing.
func ParsePDFFile(path string) (*mq.Document, error) {
	return NewParser().ParseFile(path)
}

// Implementation Notes:
//
// This parser uses two external tools:
//
// 1. pdftotext (poppler-utils) for text extraction
//    - Fast and reliable text extraction
//    - Preserves layout with -layout flag
//    - Handles complex PDFs well
//
// 2. PyMuPDF (via extract_structure.py) for structure inference
//    - Extracts font sizes and positions
//    - Infers headings from font size ratios
//    - Detects tables using PyMuPDF's find_tables()
//
// PDF parsing is fundamentally different from HTML/Markdown because PDFs
// describe APPEARANCE, not STRUCTURE. We infer structure from visual cues:
//
// - Headings: Text with font size > body_size * 1.15
// - Tables: Detected via PyMuPDF's table finder
// - Title: First large heading on page 1
//
// Requirements:
//   - pdftotext: brew install poppler (macOS) or apt install poppler-utils
//   - PyMuPDF: pip install pymupdf
