package mq

import (
	"sync"

	"github.com/yuin/goldmark/ast"
)

// Document represents a parsed document with pre-computed indexes.
// Documents can be created from multiple formats (Markdown, HTML, PDF)
// but expose the same structural interface for querying.
//
// The structural types (Heading, Section, CodeBlock, etc.) are format-agnostic.
// This allows the same MQL queries to work on any document regardless of source format.
type Document struct {
	source   []byte
	path     string
	format   Format
	metadata Metadata

	// Markdown-specific: AST from goldmark (nil for other formats)
	root ast.Node

	// Format-agnostic content
	title        string // Document title (HTML: <title>, PDF: metadata, MD: first H1)
	readableText string // Main content as plain text (for LLM context)

	// Pre-computed indexes for O(1) lookups
	mu              sync.RWMutex
	headingIndex    map[string]*Heading     // by text
	headingsByLevel map[int][]*Heading      // by level
	sectionIndex    map[string]*Section     // by title
	codeBlocks      []*CodeBlock            // all code blocks
	codeByLang      map[string][]*CodeBlock // by language
	links           []*Link                 // all links
	images          []*Image                // all images
	tables          []*Table                // all tables
	lists           []*List                 // all lists
}

// NewDocument creates a Document from pre-extracted structural elements.
// This is the constructor used by HTML and PDF parsers.
//
// The parser is responsible for:
//   - Extracting structural elements from the source format
//   - Building the section hierarchy (parent/children relationships)
//   - Determining the readable text content
func NewDocument(
	source []byte,
	path string,
	format Format,
	title string,
	headings []*Heading,
	sections []*Section,
	codeBlocks []*CodeBlock,
	links []*Link,
	images []*Image,
	tables []*Table,
	lists []*List,
	readableText string,
) *Document {
	doc := &Document{
		source:          source,
		path:            path,
		format:          format,
		title:           title,
		readableText:    readableText,
		headingIndex:    make(map[string]*Heading),
		headingsByLevel: make(map[int][]*Heading),
		sectionIndex:    make(map[string]*Section),
		codeBlocks:      codeBlocks,
		codeByLang:      make(map[string][]*CodeBlock),
		links:           links,
		images:          images,
		tables:          tables,
		lists:           lists,
	}

	// Build heading indexes
	for _, h := range headings {
		doc.headingIndex[h.Text] = h
		doc.headingsByLevel[h.Level] = append(doc.headingsByLevel[h.Level], h)
	}

	// Build section index
	for _, s := range sections {
		if s.Heading != nil {
			doc.sectionIndex[s.Heading.Text] = s
		}
	}

	// Build code block language index
	for _, cb := range codeBlocks {
		if cb.Language != "" {
			doc.codeByLang[cb.Language] = append(doc.codeByLang[cb.Language], cb)
		}
	}

	return doc
}

// NewHTMLDocument is a convenience constructor for HTML documents.
// Deprecated: Use NewDocument with FormatHTML instead.
func NewHTMLDocument(
	source []byte,
	path string,
	title string,
	headings []*Heading,
	sections []*Section,
	codeBlocks []*CodeBlock,
	links []*Link,
	images []*Image,
	tables []*Table,
	lists []*List,
	readableText string,
) *Document {
	return NewDocument(source, path, FormatHTML, title, headings, sections, codeBlocks, links, images, tables, lists, readableText)
}

// Path returns the document's file path.
func (d *Document) Path() string {
	return d.path
}

// Format returns the document's source format.
func (d *Document) Format() Format {
	return d.format
}

// Source returns the raw source content.
func (d *Document) Source() []byte {
	return d.source
}

// Title returns the document title.
// For HTML: <title> tag
// For PDF: document metadata
// For Markdown: first H1 heading or empty
func (d *Document) Title() string {
	if d.title != "" {
		return d.title
	}
	// Fall back to first H1 for markdown
	if headings := d.headingsByLevel[1]; len(headings) > 0 {
		return headings[0].Text
	}
	return ""
}

// ReadableText returns the main content as plain text.
// This is the content suitable for LLM context - stripped of
// navigation, ads, scripts, and other non-content elements.
//
// For Markdown: full text content
// For HTML: Readability-extracted main content
// For PDF: extracted text content
func (d *Document) ReadableText() string {
	return d.readableText
}

// AST returns the root AST node (Markdown only).
// Returns nil for HTML and PDF documents.
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
