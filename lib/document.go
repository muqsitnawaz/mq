package mq

import (
	"strings"
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
	pageCount    int    // Total page count (PDF only, 0 if not applicable)

	// Pre-computed indexes for O(1) lookups
	mu              sync.RWMutex
	headingIndex    map[string]*Heading     // by text
	headingsByLevel map[int][]*Heading      // by level
	sectionIndex    map[string]*Section     // by title
	sectionList     []*Section              // ordered list of all sections
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

	// Build section index and ordered list, including nested children.
	// This allows .section("child") to find sections at any depth.
	doc.sectionList = sections
	var indexSections func([]*Section)
	indexSections = func(ss []*Section) {
		for _, s := range ss {
			if s.Heading != nil {
				if _, exists := doc.sectionIndex[s.Heading.Text]; !exists {
					doc.sectionIndex[s.Heading.Text] = s
				}
			}
			if len(s.Children) > 0 {
				indexSections(s.Children)
			}
		}
	}
	indexSections(sections)

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

// RawText returns the original textual source when it is safe to display.
// Binary formats like PDF fall back to extracted readable text.
func (d *Document) RawText() string {
	switch d.format {
	case FormatPDF, FormatDOCX, FormatXLSX, FormatPPTX:
		return d.readableText
	}
	return string(d.source)
}

// PageCount returns the document page count for paginated formats like PDF.
// Falls back to the highest heading page if no explicit count was stored.
func (d *Document) PageCount() int {
	if d.pageCount > 0 {
		return d.pageCount
	}

	maxPage := 0
	for _, s := range d.sectionList {
		if s.Heading != nil && s.Heading.Page > maxPage {
			maxPage = s.Heading.Page
		}
	}
	return maxPage
}

// SetPageCount stores the total page count for paginated formats like PDF.
func (d *Document) SetPageCount(pageCount int) {
	d.pageCount = pageCount
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

// GetSection returns a section by title, searching all heading levels.
// Matching: exact -> case-insensitive exact -> case-insensitive prefix
// -> case-insensitive contains. First match in document order wins.
func (d *Document) GetSection(title string) (*Section, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return matchSection(d.sectionIndex, d.sectionList, title, 0)
}

// GetSectionByLevel finds a section at a specific heading level by title.
// Same matching ladder as GetSection but restricted to the given level.
func (d *Document) GetSectionByLevel(level int, title string) (*Section, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return matchSection(d.sectionIndex, d.sectionList, title, level)
}

// GetSectionsByLevel returns all sections at the given heading level.
func (d *Document) GetSectionsByLevel(level int) []*Section {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var result []*Section
	for _, s := range d.sectionList {
		if s.Heading != nil && s.Heading.Level == level {
			result = append(result, s)
		}
	}
	return result
}

// matchSection implements the 4-tier matching ladder.
// If level is 0, all levels are searched. Otherwise only headings at that level.
func matchSection(index map[string]*Section, ordered []*Section, title string, level int) (*Section, bool) {
	titleLower := strings.ToLower(title)

	// Build candidate list in document order, filtered by level
	candidates := ordered
	if level > 0 {
		candidates = nil
		for _, s := range ordered {
			if s.Heading != nil && s.Heading.Level == level {
				candidates = append(candidates, s)
			}
		}
	}

	// 1. Exact match (fast path via index when unfiltered)
	if level == 0 {
		if s, ok := index[title]; ok {
			return s, true
		}
	} else {
		for _, s := range candidates {
			if s.Heading.Text == title {
				return s, true
			}
		}
	}

	// 2. Case-insensitive exact
	for _, s := range candidates {
		if strings.ToLower(s.Heading.Text) == titleLower {
			return s, true
		}
	}

	// 3. Case-insensitive prefix
	for _, s := range candidates {
		if strings.HasPrefix(strings.ToLower(s.Heading.Text), titleLower) {
			return s, true
		}
	}

	// 4. Case-insensitive contains
	for _, s := range candidates {
		if strings.Contains(strings.ToLower(s.Heading.Text), titleLower) {
			return s, true
		}
	}

	return nil, false
}

// GetSections returns all sections in document order.
func (d *Document) GetSections() []*Section {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.sectionList) > 0 {
		return d.sectionList
	}

	// Fallback for documents that didn't populate sectionList
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

// GetTableOfContents returns the hierarchical structure of headings in document order.
func (d *Document) GetTableOfContents() []*Section {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Use ordered list if available, otherwise fall back to map
	source := d.sectionList
	if len(source) == 0 {
		source = make([]*Section, 0, len(d.sectionIndex))
		for _, section := range d.sectionIndex {
			source = append(source, section)
		}
	}

	// Return top-level sections in order
	var toc []*Section
	for _, section := range source {
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
