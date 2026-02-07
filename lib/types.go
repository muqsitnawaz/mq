package mq

import (
	"slices"
	"strings"

	"github.com/yuin/goldmark/ast"
)

// Metadata represents YAML frontmatter in a markdown document.
type Metadata map[string]interface{}

// Heading represents a markdown heading with metadata.
type Heading struct {
	Level int      // 1-6 for H1-H6
	Text  string   // The heading text
	ID    string   // Auto-generated or explicit ID for anchoring
	Node  ast.Node // Reference to the AST node
	Line  int      // Line number in the document
}

// Section represents a document section defined by a heading.
type Section struct {
	Heading  *Heading   // The heading that starts this section
	Content  []ast.Node // All nodes in this section
	Parent   *Section   // Parent section (if nested)
	Children []*Section // Child sections
	Start    int        // Starting line number
	End      int        // Ending line number
	source   []byte     // Reference to document source for text extraction

	// Store references to extracted elements for this section
	codeBlocks []*CodeBlock // Code blocks in this section (not children)
}

// GetText extracts the raw markdown content from the section using line numbers.
func (s *Section) GetText() string {
	if s.source == nil || s.Start == 0 || s.End == 0 {
		return ""
	}

	lines := strings.Split(string(s.source), "\n")
	if s.Start > len(lines) {
		return ""
	}

	end := s.End
	if end > len(lines) {
		end = len(lines)
	}

	// Extract lines (1-indexed to 0-indexed)
	sectionLines := lines[s.Start-1 : end]
	return strings.Join(sectionLines, "\n")
}

// GetCodeBlocks returns all code blocks in this section and its children.
func (s *Section) GetCodeBlocks(languages ...string) []*CodeBlock {
	var blocks []*CodeBlock

	// Return stored code blocks for this section
	for _, cb := range s.codeBlocks {
		if len(languages) == 0 || slices.Contains(languages, cb.Language) {
			blocks = append(blocks, cb)
		}
	}

	// Recursively check child sections
	for _, child := range s.Children {
		childBlocks := child.GetCodeBlocks(languages...)
		blocks = append(blocks, childBlocks...)
	}

	return blocks
}

// AddCodeBlock adds a code block to this section.
func (s *Section) AddCodeBlock(cb *CodeBlock) {
	s.codeBlocks = append(s.codeBlocks, cb)
}

// CodeBlock represents a fenced code block.
type CodeBlock struct {
	Language string   // Programming language identifier
	Content  string   // The code content
	Node     ast.Node // Reference to the AST node
	Lines    int      // Number of lines in the code block
}

// GetLines returns the number of lines in the code block.
func (c *CodeBlock) GetLines() int {
	if c.Lines == 0 {
		c.Lines = strings.Count(c.Content, "\n") + 1
	}
	return c.Lines
}

// Link represents a markdown link.
type Link struct {
	Text string // Display text
	URL  string // Target URL
	Node ast.Node
}

// Image represents a markdown image.
type Image struct {
	AltText string // Alternative text
	URL     string // Image URL
	Title   string // Optional title
	Node    ast.Node
}

// Table represents a markdown table.
type Table struct {
	Headers []string
	Rows    [][]string
	Node    ast.Node
}

// List represents a markdown list.
type List struct {
	Ordered bool       // true for numbered lists
	Items   []ListItem // List items
	Node    ast.Node
}

// ListItem represents an item in a list.
type ListItem struct {
	Text     string
	Checked  *bool // For task lists (nil if not a task item)
	Children []ListItem
}

// helper functions

func extractText(node ast.Node, buf *strings.Builder) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.Text:
		// Can't extract text without source bytes, just skip
		// This should ideally be called with source bytes
		return
	case *ast.String:
		buf.Write(n.Value)
	}

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		extractText(child, buf)
	}
}
