package mq

import (
	"bytes"
	"fmt"
	"os"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Parser parses markdown documents with frontmatter support.
type Parser struct {
	md goldmark.Markdown
}

// ParserOption configures the parser.
type ParserOption func(*Parser)

// NewParser creates a parser with frontmatter and table support.
func NewParser(opts ...ParserOption) *Parser {
	md := goldmark.New(
		goldmark.WithExtensions(
			meta.New(meta.WithStoresInDocument()),
			extension.Table,
			extension.TaskList,
			extension.Strikethrough,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	p := &Parser{md: md}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// WithExtensions adds custom extensions to the parser.
func WithExtensions(exts ...goldmark.Extender) ParserOption {
	return func(p *Parser) {
		p.md = goldmark.New(
			goldmark.WithExtensions(append([]goldmark.Extender{
				meta.New(meta.WithStoresInDocument()),
				extension.Table,
				extension.TaskList,
				extension.Strikethrough,
			}, exts...)...),
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
			),
		)
	}
}

// ParseFile parses a markdown file.
func (p *Parser) ParseFile(path string) (*Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return p.Parse(content, path)
}

// Parse parses markdown content.
func (p *Parser) Parse(source []byte, path string) (*Document, error) {
	reader := text.NewReader(source)
	ctx := parser.NewContext()
	node := p.md.Parser().Parse(reader, parser.WithContext(ctx))

	doc := &Document{
		source:          source,
		path:            path,
		format:          FormatMarkdown,
		root:            node,
		headingIndex:    make(map[string]*Heading),
		headingsByLevel: make(map[int][]*Heading),
		sectionIndex:    make(map[string]*Section),
		codeByLang:      make(map[string][]*CodeBlock),
		codeBlocks:      []*CodeBlock{},
		links:           []*Link{},
		images:          []*Image{},
		tables:          []*Table{},
		lists:           []*List{},
	}

	// Extract metadata from frontmatter
	metaData := meta.Get(ctx)
	if metaData != nil {
		doc.metadata = Metadata(metaData)
	}

	// Build indexes
	if err := p.buildIndexes(doc); err != nil {
		return nil, fmt.Errorf("building indexes: %w", err)
	}

	return doc, nil
}

// buildIndexes walks the AST and builds document indexes.
func (p *Parser) buildIndexes(doc *Document) error {
	var currentSection *Section
	var sectionStack []*Section
	var allSections []*Section

	// Pre-compute line starts for efficient line number lookups
	lineStarts := computeLineStarts(doc.source)

	err := ast.Walk(doc.root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Heading:
			heading := p.extractHeading(node, doc.source)
			// Get line number from AST node's byte offset
			if lines := node.Lines(); lines.Len() > 0 {
				heading.Line = getLineNumber(lineStarts, lines.At(0).Start)
			}

			// Add to heading indexes
			doc.headingIndex[heading.Text] = heading
			doc.headingsByLevel[heading.Level] = append(
				doc.headingsByLevel[heading.Level],
				heading,
			)

			// Create section
			section := &Section{
				Heading: heading,
				Start:   heading.Line,
				Content: []ast.Node{},
				source:  doc.source,
			}

			// Manage section hierarchy
			for len(sectionStack) > 0 && sectionStack[len(sectionStack)-1].Heading.Level >= heading.Level {
				// Close previous section at the line before this heading
				prev := sectionStack[len(sectionStack)-1]
				if heading.Line > 0 {
					prev.End = heading.Line - 1
				}
				// If heading.Line is 0, leave prev.End as 0 - it will be fixed in the final cleanup
				sectionStack = sectionStack[:len(sectionStack)-1]
			}

			// Set parent if exists
			if len(sectionStack) > 0 {
				parent := sectionStack[len(sectionStack)-1]
				section.Parent = parent
				parent.Children = append(parent.Children, section)
			}

			sectionStack = append(sectionStack, section)
			currentSection = section
			allSections = append(allSections, section)
			doc.sectionIndex[heading.Text] = section

		case *ast.FencedCodeBlock:
			cb := p.extractCodeBlock(node, doc.source)
			doc.codeBlocks = append(doc.codeBlocks, cb)
			if cb.Language != "" {
				doc.codeByLang[cb.Language] = append(
					doc.codeByLang[cb.Language],
					cb,
				)
			}
			if currentSection != nil {
				currentSection.Content = append(currentSection.Content, node)
				currentSection.AddCodeBlock(cb) // Store reference in section
			}

		case *ast.Link:
			link := p.extractLink(node, doc.source)
			doc.links = append(doc.links, link)
			if currentSection != nil {
				currentSection.Content = append(currentSection.Content, node)
			}

		case *ast.Image:
			image := p.extractImage(node, doc.source)
			doc.images = append(doc.images, image)
			if currentSection != nil {
				currentSection.Content = append(currentSection.Content, node)
			}

		case *east.Table:
			table := p.extractTable(node, doc.source)
			doc.tables = append(doc.tables, table)
			if currentSection != nil {
				currentSection.Content = append(currentSection.Content, node)
			}

		case *ast.List:
			list := p.extractList(node, doc.source)
			doc.lists = append(doc.lists, list)
			if currentSection != nil {
				currentSection.Content = append(currentSection.Content, node)
			}

		case *ast.Paragraph:
			if currentSection != nil {
				currentSection.Content = append(currentSection.Content, node)
			}

		default:
			// Add other nodes to current section
			if currentSection != nil {
				currentSection.Content = append(currentSection.Content, node)
			}
		}

		return ast.WalkContinue, nil
	})

	// Fix any sections with invalid End values (0 or negative)
	totalLines := len(lineStarts)
	for _, section := range allSections {
		if section.End <= 0 {
			section.End = totalLines
		}
	}

	return err
}

// computeLineStarts returns byte offsets where each line starts.
// lineStarts[i] is the byte offset where line i+1 starts (0-indexed internally).
func computeLineStarts(source []byte) []int {
	starts := []int{0} // Line 1 starts at byte 0
	for i, b := range source {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// getLineNumber returns the 1-based line number for a given byte offset.
func getLineNumber(lineStarts []int, offset int) int {
	// Binary search for the line containing this offset
	lo, hi := 0, len(lineStarts)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if lineStarts[mid] <= offset {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo + 1 // Convert to 1-based line number
}

// extractHeading extracts heading information from an AST node.
func (p *Parser) extractHeading(node *ast.Heading, source []byte) *Heading {
	var text string
	var buf bytes.Buffer

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*ast.Text); ok {
			buf.Write(t.Segment.Value(source))
		} else {
			ast.Walk(child, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
				if entering {
					if t, ok := n.(*ast.Text); ok {
						buf.Write(t.Segment.Value(source))
					}
				}
				return ast.WalkContinue, nil
			})
		}
	}
	text = buf.String()

	id := ""
	if v, ok := node.AttributeString("id"); ok {
		id = string(util.EscapeHTML(v.([]byte)))
	}

	return &Heading{
		Level: node.Level,
		Text:  text,
		ID:    id,
		Node:  node,
	}
}

// extractCodeBlock extracts code block information from an AST node.
func (p *Parser) extractCodeBlock(node *ast.FencedCodeBlock, source []byte) *CodeBlock {
	var language string
	if node.Info != nil {
		language = string(node.Info.Segment.Value(source))
	}

	var content bytes.Buffer
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		content.Write(line.Value(source))
	}

	code := content.String()
	return &CodeBlock{
		Language: language,
		Content:  code,
		Node:     node,
		Lines:    lines.Len(),
	}
}

// extractLink extracts link information from an AST node.
func (p *Parser) extractLink(node *ast.Link, source []byte) *Link {
	var text bytes.Buffer
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*ast.Text); ok {
			text.Write(t.Segment.Value(source))
		}
	}

	return &Link{
		Text: text.String(),
		URL:  string(node.Destination),
		Node: node,
	}
}

// extractImage extracts image information from an AST node.
func (p *Parser) extractImage(node *ast.Image, source []byte) *Image {
	var altText bytes.Buffer
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*ast.Text); ok {
			altText.Write(t.Segment.Value(source))
		}
	}

	return &Image{
		AltText: altText.String(),
		URL:     string(node.Destination),
		Title:   string(node.Title),
		Node:    node,
	}
}

// extractTable extracts table information from an AST node.
func (p *Parser) extractTable(node *east.Table, source []byte) *Table {
	table := &Table{
		Node: node,
	}

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch row := child.(type) {
		case *east.TableHeader:
			// Extract headers
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				var text bytes.Buffer
				ast.Walk(cell, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
					if entering {
						if t, ok := n.(*ast.Text); ok {
							text.Write(t.Segment.Value(source))
						}
					}
					return ast.WalkContinue, nil
				})
				table.Headers = append(table.Headers, text.String())
			}

		case *east.TableRow:
			// Extract row data
			var rowData []string
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				var text bytes.Buffer
				ast.Walk(cell, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
					if entering {
						if t, ok := n.(*ast.Text); ok {
							text.Write(t.Segment.Value(source))
						}
					}
					return ast.WalkContinue, nil
				})
				rowData = append(rowData, text.String())
			}
			table.Rows = append(table.Rows, rowData)
		}
	}

	return table
}

// extractList extracts list information from an AST node.
func (p *Parser) extractList(node *ast.List, source []byte) *List {
	list := &List{
		Ordered: node.IsOrdered(),
		Node:    node,
	}

	for item := node.FirstChild(); item != nil; item = item.NextSibling() {
		if li, ok := item.(*ast.ListItem); ok {
			listItem := p.extractListItem(li, source)
			list.Items = append(list.Items, listItem)
		}
	}

	return list
}

// extractListItem extracts list item information.
func (p *Parser) extractListItem(node *ast.ListItem, source []byte) ListItem {
	item := ListItem{}

	// Check if it's a task list item
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if tl, ok := child.(*east.TaskCheckBox); ok {
			checked := tl.IsChecked
			item.Checked = &checked
			continue
		}

		// Extract text
		var text bytes.Buffer
		ast.Walk(child, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering {
				if t, ok := n.(*ast.Text); ok {
					text.Write(t.Segment.Value(source))
				}
			}
			return ast.WalkContinue, nil
		})
		if text.Len() > 0 {
			item.Text += text.String()
		}

		// Handle nested lists
		if list, ok := child.(*ast.List); ok {
			for subItem := list.FirstChild(); subItem != nil; subItem = subItem.NextSibling() {
				if li, ok := subItem.(*ast.ListItem); ok {
					item.Children = append(item.Children, p.extractListItem(li, source))
				}
			}
		}
	}

	return item
}
