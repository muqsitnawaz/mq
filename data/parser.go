// Package data provides parsers for structured data formats (JSON, JSONL, YAML).
//
// These formats don't have document structure like headings/sections, but they
// have data structure (keys, arrays, nested objects). The parser exposes this
// structure through mq's unified interface:
//
//   - Headings: Top-level keys become H1, nested keys become H2+
//   - Sections: Each top-level key/array element is a section
//   - Tables: Arrays of objects with consistent keys become tables
//   - ReadableText: Pretty-printed or summarized content
//
// JSONL files are treated as arrays where each line is an element.
//
// Example:
//
//	parser := data.NewJSONLParser()
//	doc, _ := parser.ParseFile("messages.jsonl")
//
//	// Query like any other document
//	headings := doc.GetHeadings()  // Top-level keys from first object
//	tables := doc.GetTables()      // If array of uniform objects
package data

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	mq "github.com/muqsitnawaz/mq/lib"
	"gopkg.in/yaml.v3"
)

// JSONParser parses JSON files.
type JSONParser struct {
	prettyPrint bool
}

// JSONOption configures the JSON parser.
type JSONOption func(*JSONParser)

// NewJSONParser creates a new JSON parser.
func NewJSONParser(opts ...JSONOption) *JSONParser {
	p := &JSONParser{
		prettyPrint: true,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Format implements mq.FormatParser.
func (p *JSONParser) Format() mq.Format {
	return mq.FormatJSON
}

// ParseFile reads and parses a JSON file.
func (p *JSONParser) ParseFile(path string) (*mq.Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatJSON, Path: path, Err: err}
	}
	return p.Parse(content, path)
}

// Parse parses JSON content.
func (p *JSONParser) Parse(content []byte, path string) (*mq.Document, error) {
	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, &mq.ParseError{Format: mq.FormatJSON, Path: path, Err: err}
	}

	return p.buildDocument(content, path, data, mq.FormatJSON)
}

// JSONLParser parses JSONL (JSON Lines) files.
type JSONLParser struct {
	maxLines int // Maximum lines to parse (0 = unlimited)
}

// JSONLOption configures the JSONL parser.
type JSONLOption func(*JSONLParser)

// NewJSONLParser creates a new JSONL parser.
func NewJSONLParser(opts ...JSONLOption) *JSONLParser {
	p := &JSONLParser{
		maxLines: 0, // unlimited
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithMaxLines limits the number of lines parsed.
func WithMaxLines(n int) JSONLOption {
	return func(p *JSONLParser) {
		p.maxLines = n
	}
}

// Format implements mq.FormatParser.
func (p *JSONLParser) Format() mq.Format {
	return mq.FormatJSONL
}

// ParseFile reads and parses a JSONL file.
func (p *JSONLParser) ParseFile(path string) (*mq.Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatJSONL, Path: path, Err: err}
	}
	return p.Parse(content, path)
}

// Parse parses JSONL content.
func (p *JSONLParser) Parse(content []byte, path string) (*mq.Document, error) {
	var items []interface{}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	// Increase buffer size for large lines
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	lineNum := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var item interface{}
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			// Skip malformed lines
			continue
		}

		items = append(items, item)
		lineNum++

		if p.maxLines > 0 && lineNum >= p.maxLines {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, &mq.ParseError{Format: mq.FormatJSONL, Path: path, Err: err}
	}

	// Build document from array of items
	jsonParser := &JSONParser{prettyPrint: true}
	return jsonParser.buildDocument(content, path, items, mq.FormatJSONL)
}

// YAMLParser parses YAML files.
type YAMLParser struct{}

// NewYAMLParser creates a new YAML parser.
func NewYAMLParser() *YAMLParser {
	return &YAMLParser{}
}

// Format implements mq.FormatParser.
func (p *YAMLParser) Format() mq.Format {
	return mq.FormatYAML
}

// ParseFile reads and parses a YAML file.
func (p *YAMLParser) ParseFile(path string) (*mq.Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatYAML, Path: path, Err: err}
	}
	return p.Parse(content, path)
}

// Parse parses YAML content.
func (p *YAMLParser) Parse(content []byte, path string) (*mq.Document, error) {
	var data interface{}
	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, &mq.ParseError{Format: mq.FormatYAML, Path: path, Err: err}
	}

	jsonParser := &JSONParser{prettyPrint: true}
	return jsonParser.buildDocument(content, path, data, mq.FormatYAML)
}

// buildDocument creates an mq.Document from parsed data.
func (p *JSONParser) buildDocument(source []byte, path string, data interface{}, format mq.Format) (*mq.Document, error) {
	var headings []*mq.Heading
	var sections []*mq.Section
	var tables []*mq.Table
	var title string

	switch v := data.(type) {
	case map[string]interface{}:
		// Object: keys become headings
		title = inferTitle(v)
		headings, sections = extractObjectStructure(v, 1)

	case []interface{}:
		// Array: check if it's a table (array of uniform objects)
		if len(v) > 0 {
			if table := tryExtractTable(v); table != nil {
				tables = append(tables, table)
				title = fmt.Sprintf("Array (%d items)", len(v))
			} else {
				// Not a table, create sections for each item
				title = fmt.Sprintf("Array (%d items)", len(v))
				for i, item := range v {
					if i >= 100 { // Limit sections for large arrays
						break
					}
					h := &mq.Heading{
						Level: 1,
						Text:  fmt.Sprintf("Item %d", i+1),
					}
					headings = append(headings, h)

					s := &mq.Section{Heading: h}
					if obj, ok := item.(map[string]interface{}); ok {
						childHeadings, childSections := extractObjectStructure(obj, 2)
						headings = append(headings, childHeadings...)
						s.Children = childSections
					}
					sections = append(sections, s)
				}
			}
		}

	default:
		title = "Value"
	}

	// Generate readable text
	readableText := generateReadableText(data)

	return mq.NewDocument(
		source,
		path,
		format,
		title,
		headings,
		sections,
		nil, // codeBlocks
		nil, // links
		nil, // images
		tables,
		nil, // lists
		readableText,
	), nil
}

// extractObjectStructure extracts headings and sections from an object.
func extractObjectStructure(obj map[string]interface{}, level int) ([]*mq.Heading, []*mq.Section) {
	var headings []*mq.Heading
	var sections []*mq.Section

	// Sort keys for consistent output
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := obj[key]

		h := &mq.Heading{
			Level: level,
			Text:  key,
		}
		headings = append(headings, h)

		s := &mq.Section{Heading: h}

		// Recurse into nested objects
		if nested, ok := value.(map[string]interface{}); ok && level < 4 {
			childHeadings, childSections := extractObjectStructure(nested, level+1)
			headings = append(headings, childHeadings...)
			s.Children = childSections
		}

		sections = append(sections, s)
	}

	return headings, sections
}

// inferTitle tries to find a suitable title from object keys.
func inferTitle(obj map[string]interface{}) string {
	// Common title keys
	titleKeys := []string{"title", "name", "id", "type", "role"}
	for _, key := range titleKeys {
		if v, ok := obj[key]; ok {
			if s, ok := v.(string); ok && len(s) < 100 {
				return s
			}
		}
	}
	return fmt.Sprintf("Object (%d keys)", len(obj))
}

// tryExtractTable checks if array is a table (uniform objects).
func tryExtractTable(arr []interface{}) *mq.Table {
	if len(arr) == 0 {
		return nil
	}

	// Check if first item is an object
	firstObj, ok := arr[0].(map[string]interface{})
	if !ok {
		return nil
	}

	// Get keys from first object
	keys := make([]string, 0, len(firstObj))
	for k := range firstObj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) == 0 || len(keys) > 20 { // Too many columns = not a good table
		return nil
	}

	// Check if all items have the same keys
	for _, item := range arr {
		obj, ok := item.(map[string]interface{})
		if !ok {
			return nil
		}
		if len(obj) != len(keys) {
			return nil // Different number of keys
		}
		for _, k := range keys {
			if _, exists := obj[k]; !exists {
				return nil // Missing key
			}
		}
	}

	// Build table
	table := &mq.Table{
		Headers: keys,
		Rows:    make([][]string, 0, len(arr)),
	}

	for _, item := range arr {
		obj := item.(map[string]interface{})
		row := make([]string, len(keys))
		for i, k := range keys {
			row[i] = formatValue(obj[k])
		}
		table.Rows = append(table.Rows, row)
	}

	return table
}

// formatValue converts a value to string for table display.
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		if len(val) > 50 {
			return val[:47] + "..."
		}
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%.2f", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case nil:
		return "null"
	case map[string]interface{}:
		return fmt.Sprintf("{%d keys}", len(val))
	case []interface{}:
		return fmt.Sprintf("[%d items]", len(val))
	default:
		return fmt.Sprintf("%v", val)
	}
}

// generateReadableText creates a readable text summary.
func generateReadableText(data interface{}) string {
	// Pretty-print with limited depth
	formatted, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", data)
	}

	text := string(formatted)

	// Truncate if too long
	if len(text) > 50000 {
		return text[:50000] + "\n... (truncated)"
	}

	return text
}

// Ensure parsers implement mq.FormatParser
var (
	_ mq.FormatParser = (*JSONParser)(nil)
	_ mq.FormatParser = (*JSONLParser)(nil)
	_ mq.FormatParser = (*YAMLParser)(nil)
)
