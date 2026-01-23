package mq

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TreeNode represents a node in the document structure tree.
type TreeNode struct {
	Type     string      // "section", "code", "table", "list", "link", "image", "frontmatter"
	Text     string      // Display text (heading text, language, etc.)
	Start    int         // Starting line number
	End      int         // Ending line number
	Level    int         // Heading level (1-6) for sections
	Meta     string      // Additional metadata (e.g., "3 blocks", "5 items")
	Children []*TreeNode // Child nodes
}

// TreeResult represents the result of a .tree query.
type TreeResult struct {
	Path     string      // File path
	Lines    int         // Total line count
	Compact  bool        // Whether compact mode is enabled
	Root     []*TreeNode // Top-level nodes
	Metadata []string    // Frontmatter field names
}

// BuildTree creates a tree representation of the document.
func (d *Document) BuildTree(compact bool) *TreeResult {
	result := &TreeResult{
		Path:    d.path,
		Lines:   d.countLines(),
		Compact: compact,
	}

	// Add frontmatter if present
	if d.metadata != nil && len(d.metadata) > 0 {
		var fields []string
		for key := range d.metadata {
			fields = append(fields, key)
		}
		result.Metadata = fields
	}

	// Build section tree
	toc := d.GetTableOfContents()
	for _, section := range toc {
		node := d.buildSectionTree(section, compact)
		result.Root = append(result.Root, node)
	}

	return result
}

// buildSectionTree recursively builds tree nodes from sections.
func (d *Document) buildSectionTree(section *Section, compact bool) *TreeNode {
	node := &TreeNode{
		Type:  "section",
		Text:  section.Heading.Text,
		Start: section.Start,
		End:   section.End,
		Level: section.Heading.Level,
	}

	// Add child sections
	for _, child := range section.Children {
		childNode := d.buildSectionTree(child, compact)
		node.Children = append(node.Children, childNode)
	}

	// Add special elements (only in full mode)
	if !compact {
		// Code blocks in this section (not children)
		codeBlocks := section.codeBlocks
		if len(codeBlocks) > 0 {
			// Group by language
			langCounts := make(map[string]int)
			for _, cb := range codeBlocks {
				lang := cb.Language
				if lang == "" {
					lang = "plain"
				}
				langCounts[lang]++
			}
			for lang, count := range langCounts {
				meta := fmt.Sprintf("%d block", count)
				if count > 1 {
					meta = fmt.Sprintf("%d blocks", count)
				}
				node.Children = append(node.Children, &TreeNode{
					Type: "code",
					Text: lang,
					Meta: meta,
				})
			}
		}

		// Tables, lists, links, images would need to be tracked per-section
		// For now, we'll add them at the document level analysis
	}

	return node
}

// countLines counts the total lines in the document.
func (d *Document) countLines() int {
	return strings.Count(string(d.source), "\n") + 1
}

// String renders the tree as a string.
func (t *TreeResult) String() string {
	var buf strings.Builder

	// Header
	buf.WriteString(fmt.Sprintf("%s (%d lines)\n", t.Path, t.Lines))

	// Frontmatter
	if len(t.Metadata) > 0 {
		prefix := getPrefix(0, len(t.Root) > 0)
		buf.WriteString(fmt.Sprintf("%s[frontmatter: %s]\n", prefix, strings.Join(t.Metadata, ", ")))
	}

	// Render nodes
	for i, node := range t.Root {
		isLast := i == len(t.Root)-1
		t.renderNode(&buf, node, "", isLast)
	}

	return buf.String()
}

// renderNode recursively renders a tree node.
func (t *TreeResult) renderNode(buf *strings.Builder, node *TreeNode, prefix string, isLast bool) {
	// Determine connector
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	// Render this node
	switch node.Type {
	case "section":
		levelPrefix := strings.Repeat("#", node.Level)
		buf.WriteString(fmt.Sprintf("%s%s%s %s (%d-%d)\n",
			prefix, connector, levelPrefix, node.Text, node.Start, node.End))
	case "code":
		buf.WriteString(fmt.Sprintf("%s%s[code: %s, %s]\n",
			prefix, connector, node.Text, node.Meta))
	case "table":
		buf.WriteString(fmt.Sprintf("%s%s[table: %s]\n",
			prefix, connector, node.Meta))
	case "list":
		buf.WriteString(fmt.Sprintf("%s%s[list: %s]\n",
			prefix, connector, node.Meta))
	case "link":
		buf.WriteString(fmt.Sprintf("%s%s[link: %s]\n",
			prefix, connector, node.Meta))
	case "image":
		buf.WriteString(fmt.Sprintf("%s%s[image: %s]\n",
			prefix, connector, node.Meta))
	}

	// Calculate child prefix
	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	// Render children
	for i, child := range node.Children {
		childIsLast := i == len(node.Children)-1
		t.renderNode(buf, child, childPrefix, childIsLast)
	}
}

// getPrefix returns the appropriate prefix for tree rendering.
func getPrefix(depth int, hasMore bool) string {
	if depth == 0 {
		if hasMore {
			return "├── "
		}
		return "└── "
	}
	return strings.Repeat("│   ", depth)
}

// SearchResult represents a search match with section context.
type SearchResult struct {
	File    string // File path
	Section string // Section heading
	Lines   string // Line range (e.g., "34-89")
	Match   string // Snippet with match context
}

// SearchResults holds all search matches.
type SearchResults struct {
	Query   string
	Matches []*SearchResult
}

// Search finds sections containing the query term.
func (d *Document) Search(query string) *SearchResults {
	results := &SearchResults{Query: query}
	query = strings.ToLower(query)

	for _, section := range d.GetSections() {
		text := section.GetText()
		if strings.Contains(strings.ToLower(text), query) {
			// Find a snippet around the match
			snippet := extractSnippet(text, query, 60)
			results.Matches = append(results.Matches, &SearchResult{
				File:    d.path,
				Section: section.Heading.Text,
				Lines:   fmt.Sprintf("%d-%d", section.Start, section.End),
				Match:   snippet,
			})
		}
	}

	return results
}

// extractSnippet extracts text around the first match.
func extractSnippet(text, query string, contextLen int) string {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, strings.ToLower(query))
	if idx < 0 {
		return ""
	}

	start := idx - contextLen
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + contextLen
	if end > len(text) {
		end = len(text)
	}

	snippet := text[start:end]
	// Clean up whitespace
	snippet = strings.ReplaceAll(snippet, "\n", " ")
	snippet = strings.Join(strings.Fields(snippet), " ")

	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(text) {
		snippet = snippet + "..."
	}

	return snippet
}

// String renders search results.
func (r *SearchResults) String() string {
	if len(r.Matches) == 0 {
		return fmt.Sprintf("No matches for %q\n", r.Query)
	}

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("Found %d matches for %q:\n\n", len(r.Matches), r.Query))

	currentFile := ""
	for _, m := range r.Matches {
		if m.File != currentFile {
			if currentFile != "" {
				buf.WriteString("\n")
			}
			buf.WriteString(fmt.Sprintf("%s:\n", m.File))
			currentFile = m.File
		}
		buf.WriteString(fmt.Sprintf("  ## %s (lines %s)\n", m.Section, m.Lines))
		if m.Match != "" {
			buf.WriteString(fmt.Sprintf("     %q\n", m.Match))
		}
	}

	return buf.String()
}

// SearchDir searches all markdown files in a directory.
func SearchDir(dirPath string, query string) (*SearchResults, error) {
	results := &SearchResults{Query: query}
	parser := NewParser()

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		doc, err := parser.ParseFile(path)
		if err != nil {
			return nil // Skip unparseable files
		}

		fileResults := doc.Search(query)
		results.Matches = append(results.Matches, fileResults.Matches...)
		return nil
	})

	return results, err
}

// DirFileNode represents a file or directory in the directory tree.
type DirFileNode struct {
	Name       string         // File or directory name
	Path       string         // Full path
	IsDir      bool           // True if directory
	Lines      int            // Line count (files only)
	Sections   int            // Section count (files only)
	TopHeadings []string      // Top-level headings for expand mode
	Children   []*DirFileNode // Child files/directories
}

// DirTreeResult represents the result of a directory tree query.
type DirTreeResult struct {
	Path       string         // Directory path
	TotalFiles int            // Total .md files
	TotalLines int            // Total lines across all files
	Expand     bool           // Whether to show top-level headings
	Root       []*DirFileNode // Top-level entries
}

// BuildDirTree creates a tree representation of markdown files in a directory.
func BuildDirTree(dirPath string, expand bool) (*DirTreeResult, error) {
	result := &DirTreeResult{
		Path:   dirPath,
		Expand: expand,
	}

	parser := NewParser()
	root, err := buildDirNode(dirPath, parser, expand, result)
	if err != nil {
		return nil, err
	}

	result.Root = root.Children
	return result, nil
}

// buildDirNode recursively builds directory tree nodes.
func buildDirNode(path string, parser *Parser, expand bool, result *DirTreeResult) (*DirFileNode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	node := &DirFileNode{
		Name:  info.Name(),
		Path:  path,
		IsDir: info.IsDir(),
	}

	if !info.IsDir() {
		// It's a file - parse it
		if strings.HasSuffix(strings.ToLower(path), ".md") {
			doc, err := parser.ParseFile(path)
			if err != nil {
				// Skip files that can't be parsed
				node.Lines = -1
				return node, nil
			}

			node.Lines = doc.countLines()
			sections := doc.GetSections()
			node.Sections = len(sections)

			result.TotalFiles++
			result.TotalLines += node.Lines

			// Get top-level headings for expand mode
			if expand {
				for _, h := range doc.GetHeadings(1, 2) {
					node.TopHeadings = append(node.TopHeadings, fmt.Sprintf("%s %s", strings.Repeat("#", h.Level), h.Text))
				}
			}
		}
		return node, nil
	}

	// It's a directory - read entries
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	// Sort: directories first, then files, both alphabetically
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		// Skip hidden files/directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		childPath := filepath.Join(path, entry.Name())

		// For files, only include .md files
		if !entry.IsDir() && !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}

		child, err := buildDirNode(childPath, parser, expand, result)
		if err != nil {
			continue // Skip entries that error
		}

		// Skip empty directories (no .md files)
		if child.IsDir && len(child.Children) == 0 {
			continue
		}

		node.Children = append(node.Children, child)
	}

	return node, nil
}

// String renders the directory tree as a string.
func (t *DirTreeResult) String() string {
	var buf strings.Builder

	// Header
	buf.WriteString(fmt.Sprintf("%s (%d files, %d lines total)\n", t.Path, t.TotalFiles, t.TotalLines))

	// Render nodes
	for i, node := range t.Root {
		isLast := i == len(t.Root)-1
		t.renderNode(&buf, node, "", isLast)
	}

	return buf.String()
}

// renderNode recursively renders a directory tree node.
func (t *DirTreeResult) renderNode(buf *strings.Builder, node *DirFileNode, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	if node.IsDir {
		buf.WriteString(fmt.Sprintf("%s%s%s/\n", prefix, connector, node.Name))
	} else {
		if node.Lines < 0 {
			buf.WriteString(fmt.Sprintf("%s%s%s (parse error)\n", prefix, connector, node.Name))
		} else if node.Sections == 0 {
			buf.WriteString(fmt.Sprintf("%s%s%s (%d lines, no sections)\n", prefix, connector, node.Name, node.Lines))
		} else {
			buf.WriteString(fmt.Sprintf("%s%s%s (%d lines, %d sections)\n", prefix, connector, node.Name, node.Lines, node.Sections))
		}
	}

	// Calculate child prefix
	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	// Render top-level headings in expand mode
	if t.Expand && len(node.TopHeadings) > 0 {
		for i, heading := range node.TopHeadings {
			hIsLast := i == len(node.TopHeadings)-1 && len(node.Children) == 0
			hConnector := "├── "
			if hIsLast {
				hConnector = "└── "
			}
			buf.WriteString(fmt.Sprintf("%s%s%s\n", childPrefix, hConnector, heading))
		}
	}

	// Render children
	for i, child := range node.Children {
		childIsLast := i == len(node.Children)-1
		t.renderNode(buf, child, childPrefix, childIsLast)
	}
}
