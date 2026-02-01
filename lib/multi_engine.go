package mq

import (
	"fmt"
	"os"
)

// MultiFormatEngine is an engine that automatically detects and parses
// multiple document formats (Markdown, HTML, PDF).
//
// Usage:
//
//	engine := mq.NewMultiFormatEngine()
//
//	// Auto-detect format from extension
//	doc, _ := engine.Load("page.html")      // Uses HTML parser
//	doc, _ := engine.Load("document.pdf")   // Uses PDF parser
//	doc, _ := engine.Load("README.md")      // Uses Markdown parser
//
//	// Same queries work on all formats
//	headings := doc.GetHeadings(2)
//	readable := doc.ReadableText()
type MultiFormatEngine struct {
	registry *ParserRegistry

	// Default parser for unknown formats
	defaultFormat Format
}

// MultiEngineOption configures the multi-format engine.
type MultiEngineOption func(*MultiFormatEngine)

// NewMultiFormatEngine creates an engine with all format parsers registered.
//
// By default, this registers:
//   - Markdown parser (default for unknown formats)
//   - HTML parser (with Readability enabled)
//   - PDF parser (with structure inference)
//
// To register custom parsers or override defaults, use WithParser option.
func NewMultiFormatEngine(opts ...MultiEngineOption) *MultiFormatEngine {
	e := &MultiFormatEngine{
		registry:      NewParserRegistry(),
		defaultFormat: FormatMarkdown,
	}

	// Register default Markdown parser
	e.registry.Register(&markdownParserAdapter{parser: NewParser()})

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// WithFormatParser registers a custom parser for a format.
func WithFormatParser(p FormatParser) MultiEngineOption {
	return func(e *MultiFormatEngine) {
		e.registry.Register(p)
	}
}

// WithDefaultFormat sets the format to use when detection fails.
func WithDefaultFormat(f Format) MultiEngineOption {
	return func(e *MultiFormatEngine) {
		e.defaultFormat = f
	}
}

// Load reads a file and parses it using the appropriate parser.
// Format is auto-detected from the file extension.
func (e *MultiFormatEngine) Load(path string) (*Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	return e.Parse(content, path)
}

// Parse parses content using the appropriate parser.
// Format is auto-detected from path extension and content.
func (e *MultiFormatEngine) Parse(content []byte, path string) (*Document, error) {
	format := DetectFormat(path, content)

	parser, ok := e.registry.Get(format)
	if !ok {
		// Fall back to default
		parser, ok = e.registry.Get(e.defaultFormat)
		if !ok {
			return nil, fmt.Errorf("no parser for format: %s", format)
		}
	}

	return parser.Parse(content, path)
}

// ParseWithFormat parses content using a specific parser.
func (e *MultiFormatEngine) ParseWithFormat(content []byte, path string, format Format) (*Document, error) {
	parser, ok := e.registry.Get(format)
	if !ok {
		return nil, fmt.Errorf("no parser registered for format: %s", format)
	}

	return parser.Parse(content, path)
}

// RegisterParser adds a parser for a format.
func (e *MultiFormatEngine) RegisterParser(p FormatParser) {
	e.registry.Register(p)
}

// HasParser checks if a parser is registered for a format.
func (e *MultiFormatEngine) HasParser(f Format) bool {
	_, ok := e.registry.Get(f)
	return ok
}

// From creates a fluent query builder for a document.
func (e *MultiFormatEngine) From(doc *Document) *QueryBuilder {
	return &QueryBuilder{
		engine: &Engine{parser: NewParser()},
		doc:    doc,
	}
}

// markdownParserAdapter wraps the existing Parser to implement the FormatParser interface.
type markdownParserAdapter struct {
	parser *Parser
}

func (a *markdownParserAdapter) Parse(content []byte, path string) (*Document, error) {
	return a.parser.Parse(content, path)
}

func (a *markdownParserAdapter) ParseFile(path string) (*Document, error) {
	return a.parser.ParseFile(path)
}

func (a *markdownParserAdapter) Format() Format {
	return FormatMarkdown
}

// Convenience functions for common use cases

// LoadMarkdown loads and parses a Markdown file.
func LoadMarkdown(path string) (*Document, error) {
	return NewParser().ParseFile(path)
}

// LoadAny loads a file with auto-format detection.
// This is the simplest way to parse any supported format.
func LoadAny(path string) (*Document, error) {
	return NewMultiFormatEngine().Load(path)
}

// ParseAny parses content with auto-format detection.
func ParseAny(content []byte, path string) (*Document, error) {
	return NewMultiFormatEngine().Parse(content, path)
}

// Example usage demonstrating the multi-format capability:
//
//	// Create engine once, reuse for all documents
//	engine := mq.NewMultiFormatEngine()
//
//	// Load different formats - same API
//	mdDoc, _ := engine.Load("README.md")
//	htmlDoc, _ := engine.Load("page.html")
//	pdfDoc, _ := engine.Load("report.pdf")
//
//	// Same queries work on all
//	for _, doc := range []*mq.Document{mdDoc, htmlDoc, pdfDoc} {
//	    fmt.Printf("Format: %s\n", doc.Format())
//	    fmt.Printf("Title: %s\n", doc.Title())
//	    fmt.Printf("Headings: %d\n", len(doc.GetHeadings()))
//	    fmt.Printf("Readable text: %d chars\n", len(doc.ReadableText()))
//	}
//
// This is the core value proposition of mq:
// Parse once, query uniformly across all document formats.
