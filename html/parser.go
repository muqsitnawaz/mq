// Package html provides HTML parsing for mq with Readability-style content extraction.
//
// The HTML parser converts web pages into mq's unified Document structure,
// enabling the same queries that work on Markdown to work on HTML.
//
// Key features:
//   - Readability-style main content extraction (strips nav, ads, footers)
//   - Structural element extraction (headings, links, tables, code blocks)
//   - Configurable extraction options
//
// Example:
//
//	parser := html.NewParser()
//	doc, _ := parser.ParseFile("page.html")
//
//	// Same queries as markdown
//	headings := doc.GetHeadings(2)
//	links := doc.GetLinks()
//	readable := doc.ReadableText()  // Main content for LLM
package html

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	mq "github.com/muqsitnawaz/mq/lib"
)

// Parser parses HTML documents into mq.Document.
type Parser struct {
	// Options
	extractReadable bool     // Use Readability algorithm for main content
	baseURL         *url.URL // Base URL for resolving relative links
	maxDepth        int      // Maximum DOM traversal depth (0 = unlimited)
}

// Option configures the parser.
type Option func(*Parser)

// NewParser creates a new HTML parser with default options.
func NewParser(opts ...Option) *Parser {
	p := &Parser{
		extractReadable: true,
		maxDepth:        0,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithReadability enables/disables Readability-style content extraction.
// When enabled (default), the parser identifies and extracts the main content
// area, stripping navigation, ads, and other non-content elements.
func WithReadability(enabled bool) Option {
	return func(p *Parser) {
		p.extractReadable = enabled
	}
}

// WithBaseURL sets the base URL for resolving relative links.
// This is useful when parsing HTML fetched from a URL.
func WithBaseURL(baseURL string) Option {
	return func(p *Parser) {
		if u, err := url.Parse(baseURL); err == nil {
			p.baseURL = u
		}
	}
}

// WithMaxDepth sets the maximum DOM traversal depth.
// Use 0 (default) for unlimited depth.
func WithMaxDepth(depth int) Option {
	return func(p *Parser) {
		p.maxDepth = depth
	}
}

// Format implements mq.Parser.
func (p *Parser) Format() mq.Format {
	return mq.FormatHTML
}

// ParseFile reads and parses an HTML file.
func (p *Parser) ParseFile(path string) (*mq.Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatHTML, Path: path, Err: err}
	}
	return p.Parse(content, path)
}

// Parse parses HTML content and returns an mq.Document.
func (p *Parser) Parse(content []byte, path string) (*mq.Document, error) {
	node, err := html.Parse(bytes.NewReader(content))
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatHTML, Path: path, Err: err}
	}

	ext := &extractor{
		parser: p,
		source: content,
		path:   path,
		root:   node,
		seen:   make(map[*html.Node]bool),
	}

	return ext.extract()
}

// ParseReader parses HTML from a reader.
func (p *Parser) ParseReader(r io.Reader, path string) (*mq.Document, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatHTML, Path: path, Err: err}
	}
	return p.Parse(content, path)
}

// extractor extracts structured content from HTML.
type extractor struct {
	parser *Parser
	source []byte
	path   string
	root   *html.Node
	seen   map[*html.Node]bool

	// Extracted elements
	title      string
	headings   []*mq.Heading
	links      []*mq.Link
	images     []*mq.Image
	tables     []*mq.Table
	lists      []*mq.List
	codeBlocks []*mq.CodeBlock
	sections   []*mq.Section
}

func (e *extractor) extract() (*mq.Document, error) {
	// Extract title from <title> tag
	e.title = e.extractTitle(e.root)

	// Find the main content area using Readability heuristics
	mainNode := e.root
	if e.parser.extractReadable {
		if found := e.findMainContent(e.root); found != nil {
			mainNode = found
		}
	}

	// Extract structural elements
	e.extractElements(mainNode, 0)

	// Build section hierarchy from headings
	e.buildSections()

	// Extract readable text
	readableText := e.extractReadableText(mainNode)

	return mq.NewDocument(
		e.source,
		e.path,
		mq.FormatHTML,
		e.title,
		e.headings,
		e.sections,
		e.codeBlocks,
		e.links,
		e.images,
		e.tables,
		e.lists,
		readableText,
	), nil
}

// extractTitle finds the <title> tag content.
func (e *extractor) extractTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.DataAtom == atom.Title {
		return strings.TrimSpace(e.getTextContent(n))
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title := e.extractTitle(c); title != "" {
			return title
		}
	}
	return ""
}

// findMainContent implements Readability-style content detection.
//
// Strategy:
// 1. Look for semantic HTML5 elements: <main>, <article>
// 2. Look for role="main" attribute
// 3. Look for common content IDs/classes: content, main, article, post
// 4. Fall back to <body>
//
// This strips: navigation, sidebars, footers, ads, comments
func (e *extractor) findMainContent(n *html.Node) *html.Node {
	// Priority order for main content containers
	selectors := []struct {
		tag   atom.Atom
		attrs map[string][]string
	}{
		// Semantic HTML5
		{atom.Main, nil},
		{atom.Article, nil},

		// ARIA roles
		{0, map[string][]string{"role": {"main"}}},

		// Common IDs (exact match)
		{0, map[string][]string{"id": {"content", "main", "main-content", "article", "post", "entry"}}},

		// Common classes (contains match)
		{0, map[string][]string{"class": {"content", "main-content", "article", "post-content", "entry-content"}}},
	}

	for _, sel := range selectors {
		if found := e.findBySelector(n, sel.tag, sel.attrs); found != nil {
			return found
		}
	}

	// Fall back to body
	return e.findBySelector(n, atom.Body, nil)
}

func (e *extractor) findBySelector(n *html.Node, tag atom.Atom, attrs map[string][]string) *html.Node {
	if n.Type == html.ElementNode {
		tagMatch := tag == 0 || n.DataAtom == tag

		attrMatch := true
		if attrs != nil {
			attrMatch = false
			for attrName, values := range attrs {
				for _, attr := range n.Attr {
					if attr.Key == attrName {
						attrVal := strings.ToLower(attr.Val)
						for _, v := range values {
							if attrName == "class" {
								// Class: check if any class matches
								for _, cls := range strings.Fields(attrVal) {
									if cls == v || strings.Contains(cls, v) {
										attrMatch = true
										break
									}
								}
							} else {
								// ID and role: exact match
								if attrVal == v {
									attrMatch = true
								}
							}
							if attrMatch {
								break
							}
						}
					}
					if attrMatch {
						break
					}
				}
			}
		}

		if tagMatch && attrMatch && (tag != 0 || attrs != nil) {
			return n
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := e.findBySelector(c, tag, attrs); found != nil {
			return found
		}
	}
	return nil
}

// extractElements walks the DOM and extracts structural elements.
func (e *extractor) extractElements(n *html.Node, depth int) {
	if e.seen[n] {
		return
	}
	e.seen[n] = true

	// Check depth limit
	if e.parser.maxDepth > 0 && depth > e.parser.maxDepth {
		return
	}

	if n.Type == html.ElementNode {
		// Skip non-content elements
		if e.shouldSkip(n) {
			return
		}

		switch n.DataAtom {
		case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
			e.extractHeading(n)
		case atom.A:
			e.extractLink(n)
		case atom.Img:
			e.extractImage(n)
		case atom.Table:
			e.extractTable(n)
			return // Don't recurse into table
		case atom.Ul, atom.Ol:
			e.extractList(n)
			return // Don't recurse into list
		case atom.Pre:
			e.extractCodeBlock(n)
			return // Don't recurse into pre
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		e.extractElements(c, depth+1)
	}
}

// shouldSkip returns true for elements that should be excluded from extraction.
// This includes: scripts, styles, navigation, footers, ads, hidden elements.
func (e *extractor) shouldSkip(n *html.Node) bool {
	// Always skip these tags
	skipTags := map[atom.Atom]bool{
		atom.Script:   true,
		atom.Style:    true,
		atom.Noscript: true,
		atom.Iframe:   true,
		atom.Svg:      true,
		atom.Canvas:   true,
		atom.Video:    true,
		atom.Audio:    true,
		atom.Object:   true,
		atom.Embed:    true,
	}
	if skipTags[n.DataAtom] {
		return true
	}

	// Skip navigation and non-content sections when using Readability
	if e.parser.extractReadable {
		skipReadabilityTags := map[atom.Atom]bool{
			atom.Nav:    true,
			atom.Footer: true,
			atom.Aside:  true,
			atom.Header: true, // Usually site header, not content header
		}
		if skipReadabilityTags[n.DataAtom] {
			return true
		}
	}

	// Skip elements with ad/nav/comment classes
	skipPatterns := regexp.MustCompile(`(?i)^(ad|ads|advert|banner|comment|comments|sidebar|widget|social|share|related|recommended|newsletter|popup|modal|cookie|gdpr|nav|navigation|menu|footer|header)[-_]?`)

	for _, attr := range n.Attr {
		switch attr.Key {
		case "class", "id":
			if skipPatterns.MatchString(attr.Val) {
				return true
			}
		case "hidden", "aria-hidden":
			if attr.Val == "" || attr.Val == "true" {
				return true
			}
		case "style":
			if strings.Contains(attr.Val, "display:none") ||
				strings.Contains(attr.Val, "display: none") ||
				strings.Contains(attr.Val, "visibility:hidden") {
				return true
			}
		case "role":
			if attr.Val == "navigation" || attr.Val == "banner" ||
				attr.Val == "contentinfo" || attr.Val == "complementary" {
				return true
			}
		}
	}

	return false
}

func (e *extractor) extractHeading(n *html.Node) {
	// Map atom to level (atoms are NOT consecutive integers)
	var level int
	switch n.DataAtom {
	case atom.H1:
		level = 1
	case atom.H2:
		level = 2
	case atom.H3:
		level = 3
	case atom.H4:
		level = 4
	case atom.H5:
		level = 5
	case atom.H6:
		level = 6
	default:
		return
	}

	text := strings.TrimSpace(e.getTextContent(n))
	if text == "" {
		return
	}

	var id string
	for _, attr := range n.Attr {
		if attr.Key == "id" {
			id = attr.Val
			break
		}
	}

	e.headings = append(e.headings, &mq.Heading{
		Level: level,
		Text:  text,
		ID:    id,
	})
}

func (e *extractor) extractLink(n *html.Node) {
	var href string
	for _, attr := range n.Attr {
		if attr.Key == "href" {
			href = attr.Val
			break
		}
	}

	// Skip empty, javascript, and anchor-only links
	if href == "" || strings.HasPrefix(href, "javascript:") || href == "#" {
		return
	}

	text := strings.TrimSpace(e.getTextContent(n))
	if text == "" {
		return
	}

	// Resolve relative URLs
	if e.parser.baseURL != nil && !strings.HasPrefix(href, "http") && !strings.HasPrefix(href, "//") {
		if resolved, err := e.parser.baseURL.Parse(href); err == nil {
			href = resolved.String()
		}
	}

	e.links = append(e.links, &mq.Link{
		Text: text,
		URL:  href,
	})
}

func (e *extractor) extractImage(n *html.Node) {
	var src, alt, title string
	for _, attr := range n.Attr {
		switch attr.Key {
		case "src":
			src = attr.Val
		case "data-src": // Lazy loading
			if src == "" {
				src = attr.Val
			}
		case "alt":
			alt = attr.Val
		case "title":
			title = attr.Val
		}
	}

	if src == "" {
		return
	}

	// Skip tracking pixels and tiny images
	for _, attr := range n.Attr {
		if attr.Key == "width" || attr.Key == "height" {
			if attr.Val == "1" || attr.Val == "0" {
				return
			}
		}
	}

	// Resolve relative URLs
	if e.parser.baseURL != nil && !strings.HasPrefix(src, "http") && !strings.HasPrefix(src, "//") && !strings.HasPrefix(src, "data:") {
		if resolved, err := e.parser.baseURL.Parse(src); err == nil {
			src = resolved.String()
		}
	}

	e.images = append(e.images, &mq.Image{
		URL:     src,
		AltText: alt,
		Title:   title,
	})
}

func (e *extractor) extractTable(n *html.Node) {
	table := &mq.Table{}

	// Look for thead/tbody structure or direct rows
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}

		switch c.DataAtom {
		case atom.Thead:
			e.extractTableHeaders(c, table)
		case atom.Tbody:
			e.extractTableRows(c, table)
		case atom.Tr:
			// Direct row without thead/tbody
			e.extractTableRow(c, table)
		}
	}

	if len(table.Headers) > 0 || len(table.Rows) > 0 {
		e.tables = append(e.tables, table)
	}
}

func (e *extractor) extractTableHeaders(thead *html.Node, table *mq.Table) {
	for row := thead.FirstChild; row != nil; row = row.NextSibling {
		if row.DataAtom == atom.Tr {
			for cell := row.FirstChild; cell != nil; cell = cell.NextSibling {
				if cell.DataAtom == atom.Th || cell.DataAtom == atom.Td {
					table.Headers = append(table.Headers, strings.TrimSpace(e.getTextContent(cell)))
				}
			}
			break // Only first header row
		}
	}
}

func (e *extractor) extractTableRows(tbody *html.Node, table *mq.Table) {
	for row := tbody.FirstChild; row != nil; row = row.NextSibling {
		if row.DataAtom == atom.Tr {
			var rowData []string
			for cell := row.FirstChild; cell != nil; cell = cell.NextSibling {
				if cell.DataAtom == atom.Td || cell.DataAtom == atom.Th {
					rowData = append(rowData, strings.TrimSpace(e.getTextContent(cell)))
				}
			}
			if len(rowData) > 0 {
				table.Rows = append(table.Rows, rowData)
			}
		}
	}
}

func (e *extractor) extractTableRow(tr *html.Node, table *mq.Table) {
	var rowData []string
	isHeader := true

	for cell := tr.FirstChild; cell != nil; cell = cell.NextSibling {
		if cell.DataAtom == atom.Th {
			table.Headers = append(table.Headers, strings.TrimSpace(e.getTextContent(cell)))
		} else if cell.DataAtom == atom.Td {
			isHeader = false
			rowData = append(rowData, strings.TrimSpace(e.getTextContent(cell)))
		}
	}

	if !isHeader && len(rowData) > 0 {
		table.Rows = append(table.Rows, rowData)
	}
}

func (e *extractor) extractList(n *html.Node) {
	list := &mq.List{
		Ordered: n.DataAtom == atom.Ol,
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.DataAtom == atom.Li {
			item := e.extractListItem(c)
			list.Items = append(list.Items, item)
		}
	}

	if len(list.Items) > 0 {
		e.lists = append(e.lists, list)
	}
}

func (e *extractor) extractListItem(li *html.Node) mq.ListItem {
	item := mq.ListItem{}

	// Check for checkbox input (task list)
	for c := li.FirstChild; c != nil; c = c.NextSibling {
		if c.DataAtom == atom.Input {
			for _, attr := range c.Attr {
				if attr.Key == "type" && attr.Val == "checkbox" {
					checked := false
					for _, a := range c.Attr {
						if a.Key == "checked" {
							checked = true
							break
						}
					}
					item.Checked = &checked
					break
				}
			}
		}
	}

	item.Text = strings.TrimSpace(e.getTextContent(li))

	// Extract nested lists
	for c := li.FirstChild; c != nil; c = c.NextSibling {
		if c.DataAtom == atom.Ul || c.DataAtom == atom.Ol {
			for gc := c.FirstChild; gc != nil; gc = gc.NextSibling {
				if gc.DataAtom == atom.Li {
					item.Children = append(item.Children, e.extractListItem(gc))
				}
			}
		}
	}

	return item
}

func (e *extractor) extractCodeBlock(pre *html.Node) {
	var language string
	var codeNode *html.Node = pre

	// Look for <code> inside <pre>
	for c := pre.FirstChild; c != nil; c = c.NextSibling {
		if c.DataAtom == atom.Code {
			codeNode = c
			// Extract language from class
			for _, attr := range c.Attr {
				if attr.Key == "class" {
					language = e.detectLanguageFromClass(attr.Val)
					break
				}
			}
			break
		}
	}

	// Also check pre's class for language
	if language == "" {
		for _, attr := range pre.Attr {
			if attr.Key == "class" {
				language = e.detectLanguageFromClass(attr.Val)
				break
			}
		}
	}

	content := e.getTextContent(codeNode)
	if strings.TrimSpace(content) == "" {
		return
	}

	e.codeBlocks = append(e.codeBlocks, &mq.CodeBlock{
		Language: language,
		Content:  content,
		Lines:    strings.Count(content, "\n") + 1,
	})
}

// detectLanguageFromClass extracts programming language from CSS classes.
// Common patterns:
//   - language-python, lang-python
//   - highlight-python
//   - python (as standalone class)
func (e *extractor) detectLanguageFromClass(class string) string {
	classes := strings.Fields(class)
	for _, c := range classes {
		// Common prefixes
		for _, prefix := range []string{"language-", "lang-", "highlight-", "brush:"} {
			if strings.HasPrefix(c, prefix) {
				return strings.TrimPrefix(c, prefix)
			}
		}
	}

	// Known language names as standalone classes
	knownLangs := map[string]bool{
		"python": true, "javascript": true, "js": true, "typescript": true, "ts": true,
		"go": true, "golang": true, "rust": true, "java": true, "kotlin": true,
		"c": true, "cpp": true, "csharp": true, "ruby": true, "php": true,
		"swift": true, "bash": true, "shell": true, "sql": true, "html": true,
		"css": true, "json": true, "yaml": true, "xml": true, "markdown": true,
	}

	for _, c := range classes {
		if knownLangs[strings.ToLower(c)] {
			return strings.ToLower(c)
		}
	}

	return ""
}

// buildSections creates a section hierarchy from headings.
func (e *extractor) buildSections() {
	if len(e.headings) == 0 {
		return
	}

	var stack []*mq.Section

	for _, h := range e.headings {
		section := &mq.Section{
			Heading: h,
		}

		// Pop sections of equal or higher level (lower number = higher level)
		for len(stack) > 0 {
			top := stack[len(stack)-1]
			if top.Heading.Level >= h.Level {
				stack = stack[:len(stack)-1]
			} else {
				break
			}
		}

		// Set parent relationship
		if len(stack) > 0 {
			parent := stack[len(stack)-1]
			section.Parent = parent
			parent.Children = append(parent.Children, section)
		}

		stack = append(stack, section)
		e.sections = append(e.sections, section)
	}
}

// getTextContent extracts text from a node and its descendants.
func (e *extractor) getTextContent(n *html.Node) string {
	var buf strings.Builder
	e.collectText(n, &buf)
	return buf.String()
}

func (e *extractor) collectText(n *html.Node, buf *strings.Builder) {
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
		return
	}

	if n.Type == html.ElementNode && e.shouldSkip(n) {
		return
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		e.collectText(c, buf)
	}
}

// extractReadableText gets clean text content suitable for LLM context.
func (e *extractor) extractReadableText(n *html.Node) string {
	var buf strings.Builder
	e.collectReadableText(n, &buf, 0)
	return cleanText(buf.String())
}

func (e *extractor) collectReadableText(n *html.Node, buf *strings.Builder, depth int) {
	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		if text != "" {
			buf.WriteString(text)
			buf.WriteString(" ")
		}
		return
	}

	if n.Type == html.ElementNode {
		if e.shouldSkip(n) {
			return
		}

		// Add newlines for block elements
		isBlock := isBlockElement(n.DataAtom)
		if isBlock && buf.Len() > 0 {
			buf.WriteString("\n")
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		e.collectReadableText(c, buf, depth+1)
	}

	if n.Type == html.ElementNode && isBlockElement(n.DataAtom) {
		buf.WriteString("\n")
	}
}

func isBlockElement(a atom.Atom) bool {
	blocks := map[atom.Atom]bool{
		atom.P: true, atom.Div: true, atom.Article: true, atom.Section: true,
		atom.H1: true, atom.H2: true, atom.H3: true, atom.H4: true, atom.H5: true, atom.H6: true,
		atom.Ul: true, atom.Ol: true, atom.Li: true,
		atom.Blockquote: true, atom.Pre: true,
		atom.Table: true, atom.Tr: true,
		atom.Header: true, atom.Footer: true, atom.Main: true, atom.Aside: true, atom.Nav: true,
		atom.Figure: true, atom.Figcaption: true,
		atom.Br: true, atom.Hr: true,
	}
	return blocks[a]
}

func cleanText(s string) string {
	// Normalize whitespace
	s = regexp.MustCompile(`[ \t]+`).ReplaceAllString(s, " ")
	// Normalize newlines
	s = regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")
	// Trim lines
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// Ensure Parser implements mq.FormatParser
var _ mq.FormatParser = (*Parser)(nil)

// ParseHTML is a convenience function for quick parsing.
func ParseHTML(content []byte, path string) (*mq.Document, error) {
	return NewParser().Parse(content, path)
}

// ParseHTMLFile is a convenience function for quick file parsing.
func ParseHTMLFile(path string) (*mq.Document, error) {
	return NewParser().ParseFile(path)
}

// ParseHTMLWithOptions parses with custom options.
func ParseHTMLWithOptions(content []byte, path string, opts ...Option) (*mq.Document, error) {
	return NewParser(opts...).Parse(content, path)
}

// Example usage in documentation:
//
//	// Basic usage
//	doc, err := html.ParseHTMLFile("page.html")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get main content for LLM (strips nav, ads, etc.)
//	readable := doc.ReadableText()
//	fmt.Printf("Content: %d chars\n", len(readable))
//
//	// Query structure (same as markdown)
//	for _, h := range doc.GetHeadings(2) {
//	    fmt.Printf("H2: %s\n", h.Text)
//	}
//
//	// With options
//	doc, err = html.ParseHTMLWithOptions(
//	    content,
//	    "https://example.com/page",
//	    html.WithBaseURL("https://example.com"),
//	    html.WithReadability(true),
//	)

func init() {
	// Document the size reduction this parser achieves
	_ = `
	Size Reduction Example (LinkedIn profile):
	- Raw HTML:     214,182 tokens (exceeds 200k limit)
	- ReadableText:   5,000 tokens (fits comfortably)
	- Structural:     2,000 tokens (headings + links only)

	This is why mq exists: transform raw content into queryable structure.
	`
}

// Error types for HTML parsing
type ErrInvalidHTML struct {
	msg string
}

func (e ErrInvalidHTML) Error() string {
	return fmt.Sprintf("invalid HTML: %s", e.msg)
}
