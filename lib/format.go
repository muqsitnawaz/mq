// Package mq provides a unified query interface for structured documents.
//
// This file defines the core abstractions for multi-format support.
// The key insight: queries operate on STRUCTURE, not format.
//
// Architecture:
//
//	┌────────────┐  ┌────────────┐  ┌────────────┐
//	│  Markdown  │  │    HTML    │  │    PDF     │
//	│   Parser   │  │   Parser   │  │   Parser   │
//	└─────┬──────┘  └─────┬──────┘  └─────┬──────┘
//	      │               │               │
//	      └───────────────┼───────────────┘
//	                      ▼
//	        ┌─────────────────────────┐
//	        │   Unified Document      │
//	        │   (structural types)    │
//	        └───────────┬─────────────┘
//	                    ▼
//	        ┌─────────────────────────┐
//	        │    MQL Query Engine     │
//	        └─────────────────────────┘
package mq

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Format represents a document format.
type Format int

const (
	FormatUnknown Format = iota
	FormatMarkdown
	FormatHTML
	FormatPDF
	FormatJSON
	FormatJSONL
	FormatYAML
)

func (f Format) String() string {
	switch f {
	case FormatMarkdown:
		return "markdown"
	case FormatHTML:
		return "html"
	case FormatPDF:
		return "pdf"
	case FormatJSON:
		return "json"
	case FormatJSONL:
		return "jsonl"
	case FormatYAML:
		return "yaml"
	default:
		return "unknown"
	}
}

// FormatParser converts raw content into a unified Document structure.
// Each format implements this interface to produce the same structural types.
//
// The contract:
//   - Parse MUST extract structural elements (headings, sections, code, links, etc.)
//   - Parse MUST build pre-computed indexes for O(1) lookups
//   - Parse SHOULD strip non-content elements (scripts, styles, ads for HTML)
//   - Parse SHOULD preserve semantic meaning across formats
type FormatParser interface {
	// Parse converts raw content to a Document.
	// path is used for error messages and caching keys.
	Parse(content []byte, path string) (*Document, error)

	// ParseFile reads and parses a file.
	ParseFile(path string) (*Document, error)

	// Format returns the format this parser handles.
	Format() Format
}

// ParserRegistry manages format-specific parsers.
type ParserRegistry struct {
	parsers map[Format]FormatParser
}

// NewParserRegistry creates a registry with default parsers.
func NewParserRegistry() *ParserRegistry {
	return &ParserRegistry{
		parsers: make(map[Format]FormatParser),
	}
}

// Register adds a parser for a format.
func (r *ParserRegistry) Register(p FormatParser) {
	r.parsers[p.Format()] = p
}

// Get returns the parser for a format.
func (r *ParserRegistry) Get(f Format) (FormatParser, bool) {
	p, ok := r.parsers[f]
	return p, ok
}

// DetectFormat determines the format from file extension or content.
func DetectFormat(path string, content []byte) Format {
	// First try extension
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".markdown", ".mdown", ".mkd":
		return FormatMarkdown
	case ".html", ".htm", ".xhtml":
		return FormatHTML
	case ".pdf":
		return FormatPDF
	case ".json":
		return FormatJSON
	case ".jsonl", ".ndjson":
		return FormatJSONL
	case ".yaml", ".yml":
		return FormatYAML
	}

	// Fall back to content sniffing
	if len(content) > 0 {
		// Check for HTML
		trimmed := strings.TrimSpace(string(content[:min(len(content), 1024)]))
		if strings.HasPrefix(trimmed, "<!") ||
			strings.HasPrefix(trimmed, "<html") ||
			strings.HasPrefix(trimmed, "<HTML") {
			return FormatHTML
		}

		// Check for PDF magic bytes
		if len(content) >= 4 && string(content[:4]) == "%PDF" {
			return FormatPDF
		}

		// Check for JSON (starts with { or [)
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			return FormatJSON
		}

		// Check for YAML (starts with --- or key:)
		if strings.HasPrefix(trimmed, "---") {
			return FormatYAML
		}
	}

	// Default to markdown (most permissive)
	return FormatMarkdown
}

// ParseError wraps parsing errors with format context.
type ParseError struct {
	Format Format
	Path   string
	Err    error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse %s (%s): %v", e.Path, e.Format, e.Err)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}
