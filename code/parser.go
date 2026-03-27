package code

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"

	mq "github.com/muqsitnawaz/mq/lib"
)

// Parser extracts structural declarations from source code files
// using tree-sitter for AST parsing.
type Parser struct{}

// NewParser creates a new code parser.
func NewParser() *Parser {
	return &Parser{}
}

// Format implements mq.FormatParser.
func (p *Parser) Format() mq.Format {
	return mq.FormatCode
}

// ParseFile reads and parses a source code file.
func (p *Parser) ParseFile(path string) (*mq.Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatCode, Path: path, Err: err}
	}
	return p.Parse(content, path)
}

// Parse parses source code content and extracts structural declarations.
func (p *Parser) Parse(content []byte, path string) (*mq.Document, error) {
	bt, err := grammars.ParseFile(filepath.Base(path), content)
	if err != nil {
		return nil, &mq.ParseError{Format: mq.FormatCode, Path: path, Err: err}
	}
	defer bt.Release()

	lang := bt.Language()
	langName := lang.Name
	source := string(content)

	var headings []*mq.Heading
	var sections []*mq.Section

	root := bt.RootNode()
	p.extractDeclarations(root, lang, source, langName, content, &headings, &sections, 0)

	// One code block for the entire file
	var codeBlocks []*mq.CodeBlock
	codeBlocks = append(codeBlocks, &mq.CodeBlock{
		Language: langName,
		Content:  source,
	})

	title := fmt.Sprintf("%s: %s", langName, filepath.Base(path))

	return mq.NewDocument(
		content,
		path,
		mq.FormatCode,
		title,
		headings,
		sections,
		codeBlocks,
		nil, // links
		nil, // images
		nil, // tables
		nil, // lists
		source,
	), nil
}

// extractDeclarations walks tree-sitter nodes and extracts declaration-level
// headings and sections.
func (p *Parser) extractDeclarations(
	node *ts.Node,
	lang *ts.Language,
	source string,
	langName string,
	raw []byte,
	headings *[]*mq.Heading,
	sections *[]*mq.Section,
	depth int,
) {
	if node == nil || depth > 3 {
		return
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil || !child.IsNamed() {
			continue
		}

		nodeType := child.Type(lang)
		level := getDeclarationLevel(langName, nodeType)

		if level == -1 {
			// For unknown languages, treat named root children with children as declarations
			if depth == 0 && langName != "" && declarationTypes[langName] == nil && child.NamedChildCount() > 0 {
				level = levelFunc
			} else {
				continue
			}
		}

		if level == levelUnwrap {
			// Unwrap containers like export_statement, decorated_definition
			p.extractDeclarations(child, lang, source, langName, raw, headings, sections, depth+1)
			continue
		}

		line := int(child.StartPoint().Row) + 1
		signature := firstLine(child.Text(raw), 120)

		heading := &mq.Heading{
			Level: level,
			Text:  signature,
			Line:  line,
		}
		*headings = append(*headings, heading)

		endLine := int(child.EndPoint().Row) + 1
		section := mq.NewSectionWithSource(heading, line, endLine, raw)
		*sections = append(*sections, section)

		// For type-level declarations (classes, structs, impl blocks),
		// extract nested methods as children
		if level == levelType {
			var childHeadings []*mq.Heading
			var childSections []*mq.Section
			p.extractDeclarations(child, lang, source, langName, raw, &childHeadings, &childSections, depth+1)
			for _, ch := range childHeadings {
				*headings = append(*headings, ch)
			}
			for _, cs := range childSections {
				section.Children = append(section.Children, cs)
				cs.Parent = section
			}
		}
	}
}

// firstLine returns the first line of text, truncated to maxLen.
func firstLine(text string, maxLen int) string {
	if idx := strings.Index(text, "\n"); idx != -1 {
		text = text[:idx]
	}
	text = strings.TrimSpace(text)
	if len(text) > maxLen {
		text = text[:maxLen] + "..."
	}
	return text
}
