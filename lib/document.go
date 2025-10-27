package mq

import (
	"bytes"
	"github.com/yuin/goldmark/ast"
	"strings"
	"sync"
)

// Document represents a parsed markdown document with pre-computed indexes.
type Document struct {
	source   []byte
	path     string
	root     ast.Node
	metadata Metadata

	// Frontmatter boundaries
	frontmatterRaw string // Raw YAML frontmatter text (including --- markers)
	bodyStart      int    // Byte offset where body content starts

	// Pre-computed indexes for O(1) lookups
	mu              sync.RWMutex
	headingIndex    map[string]*Heading     // by text
	headingsByLevel map[int][]*Heading     // by level
	sectionIndex    map[string]*Section     // by title
	codeBlocks      []*CodeBlock            // all code blocks
	codeByLang      map[string][]*CodeBlock // by language
	links           []*Link                 // all links
	images          []*Image                // all images
	tables          []*Table                // all tables
	lists           []*List                 // all lists
}

// Path returns the document's file path.
func (d *Document) Path() string {
	return d.path
}

// Source returns the raw markdown source.
func (d *Document) Source() []byte {
	return d.source
}

// AST returns the root AST node.
func (d *Document) AST() ast.Node {
	return d.root
}

// Metadata returns the document's frontmatter metadata.
func (d *Document) Metadata() Metadata {
	return d.metadata
}

// GetMetadataField retrieves a specific metadata field.
func (d *Document) GetMetadataField(key string) (interface{}, bool) {
	if d.metadata == nil {
		return nil, false
	}
	val, ok := d.metadata[key]
	return val, ok
}

// GetOwner returns the owner from metadata.
func (d *Document) GetOwner() (string, bool) {
	val, ok := d.GetMetadataField("owner")
	if !ok {
		return "", false
	}
	owner, ok := val.(string)
	return owner, ok
}

// CheckOwnership verifies if the document belongs to the given owner.
func (d *Document) CheckOwnership(owner string) bool {
	docOwner, ok := d.GetOwner()
	return ok && docOwner == owner
}

// GetTags returns tags from metadata.
func (d *Document) GetTags() []string {
	val, ok := d.GetMetadataField("tags")
	if !ok {
		return nil
	}

	// Handle different possible formats from YAML
	switch v := val.(type) {
	case []string:
		return v
	case []interface{}:
		tags := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				tags = append(tags, s)
			}
		}
		return tags
	default:
		return nil
	}
}

// GetPriority returns priority from metadata.
func (d *Document) GetPriority() (string, bool) {
	val, ok := d.GetMetadataField("priority")
	if !ok {
		return "", false
	}
	priority, ok := val.(string)
	return priority, ok
}

// GetHeadings returns headings, optionally filtered by level.
func (d *Document) GetHeadings(levels ...int) []*Heading {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(levels) == 0 {
		// Return all headings
		var all []*Heading
		for level := 1; level <= 6; level++ {
			all = append(all, d.headingsByLevel[level]...)
		}
		return all
	}

	// Return headings of specified levels
	var result []*Heading
	for _, level := range levels {
		if level >= 1 && level <= 6 {
			result = append(result, d.headingsByLevel[level]...)
		}
	}
	return result
}

// GetHeadingByText returns a heading by its exact text.
func (d *Document) GetHeadingByText(text string) (*Heading, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	heading, ok := d.headingIndex[text]
	return heading, ok
}

// GetSection returns a section by title.
func (d *Document) GetSection(title string) (*Section, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	section, ok := d.sectionIndex[title]
	return section, ok
}

// GetSections returns all sections.
func (d *Document) GetSections() []*Section {
	d.mu.RLock()
	defer d.mu.RUnlock()

	sections := make([]*Section, 0, len(d.sectionIndex))
	for _, section := range d.sectionIndex {
		sections = append(sections, section)
	}
	return sections
}

// GetCodeBlocks returns code blocks, optionally filtered by language.
func (d *Document) GetCodeBlocks(languages ...string) []*CodeBlock {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(languages) == 0 {
		return d.codeBlocks
	}

	var result []*CodeBlock
	for _, lang := range languages {
		result = append(result, d.codeByLang[lang]...)
	}
	return result
}

// GetLinks returns all links in the document.
func (d *Document) GetLinks() []*Link {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.links
}

// GetImages returns all images in the document.
func (d *Document) GetImages() []*Image {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.images
}

// GetTables returns all tables in the document.
func (d *Document) GetTables() []*Table {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.tables
}

// GetLists returns all lists in the document.
func (d *Document) GetLists(ordered *bool) []*List {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if ordered == nil {
		return d.lists
	}

	var result []*List
	for _, list := range d.lists {
		if list.Ordered == *ordered {
			result = append(result, list)
		}
	}
	return result
}

// GetTableOfContents returns the hierarchical structure of headings.
func (d *Document) GetTableOfContents() []*Section {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Return top-level sections
	var toc []*Section
	for _, section := range d.sectionIndex {
		if section.Parent == nil {
			toc = append(toc, section)
		}
	}
	return toc
}

// Walk traverses the document AST with a visitor function.
func (d *Document) Walk(visitor func(ast.Node, bool) (ast.WalkStatus, error)) error {
	return ast.Walk(d.root, visitor)
}

// GetSourceString returns the raw markdown source as a string.
func (d *Document) GetSourceString() string {
	return string(d.source)
}

// GetFrontmatterRaw returns the raw YAML frontmatter (including --- markers).
// Returns empty string if document has no frontmatter.
func (d *Document) GetFrontmatterRaw() string {
	return d.frontmatterRaw
}

// GetBody returns the document content after frontmatter.
// If there's no frontmatter, returns the entire document.
func (d *Document) GetBody() string {
	if d.bodyStart >= len(d.source) {
		return ""
	}
	return string(d.source[d.bodyStart:])
}

// GetTextContent returns plain text content without markdown syntax or frontmatter.
func (d *Document) GetTextContent() string {
	var buf bytes.Buffer

	// Walk the AST and extract text nodes
	ast.Walk(d.root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Text:
			buf.Write(node.Segment.Value(d.source))
		case *ast.String:
			buf.Write(node.Value)
		case *ast.Paragraph:
			// Add newline after paragraph if buffer has content
			if buf.Len() > 0 && !bytes.HasSuffix(buf.Bytes(), []byte("\n")) {
				buf.WriteString("\n")
			}
		}

		return ast.WalkContinue, nil
	})

	return buf.String()
}

// GetNestedField retrieves a nested metadata field using dot notation.
// Examples: "author", "config.theme", "nested.deep.value"
func (d *Document) GetNestedField(path string) (interface{}, bool) {
	if d.metadata == nil {
		return nil, false
	}

	// Split path by dots
	parts := strings.Split(path, ".")

	var current interface{} = d.metadata

	for _, part := range parts {
		// Handle map access
		if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[part]; exists {
				current = val
			} else {
				return nil, false
			}
		} else if m, ok := current.(Metadata); ok {
			if val, exists := m[part]; exists {
				current = val
			} else {
				return nil, false
			}
		} else if m, ok := current.(map[interface{}]interface{}); ok {
			// Handle YAML-parsed maps with interface{} keys
			found := false
			for k, v := range m {
				if strKey, ok := k.(string); ok && strKey == part {
					current = v
					found = true
					break
				}
			}
			if !found {
				return nil, false
			}
		} else {
			// Not a map, can't drill deeper
			return nil, false
		}
	}

	return current, true
}