package mq

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TreeMode represents tree display modes.
type TreeMode string

const (
	TreeModeDefault TreeMode = ""        // Full structure with code blocks
	TreeModeCompact TreeMode = "compact" // Headings only
	TreeModePreview TreeMode = "preview" // Headings + first few words
	TreeModeFull    TreeMode = "full"    // Headings + first few words (for directories: expand + preview)
)

// TreeNode represents a node in the document structure tree.
type TreeNode struct {
	Type     string      // "section", "code", "table", "list", "link", "image", "frontmatter"
	Text     string      // Display text (heading text, language, etc.)
	Preview  string      // First few words of section content
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
	Mode     TreeMode    // Display mode
	Root     []*TreeNode // Top-level nodes
	Metadata []string    // Frontmatter field names
}

// BuildTree creates a tree representation of the document.
func (d *Document) BuildTree(mode TreeMode) *TreeResult {
	result := &TreeResult{
		Path:  d.path,
		Lines: d.countLines(),
		Mode:  mode,
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
		node := d.buildSectionTree(section, mode)
		result.Root = append(result.Root, node)
	}

	return result
}

// buildSectionTree recursively builds tree nodes from sections.
func (d *Document) buildSectionTree(section *Section, mode TreeMode) *TreeNode {
	node := &TreeNode{
		Type:  "section",
		Text:  section.Heading.Text,
		Start: section.Start,
		End:   section.End,
		Level: section.Heading.Level,
	}

	// Add preview text for preview/full modes
	if mode == TreeModePreview || mode == TreeModeFull {
		node.Preview = ExtractPreview(section.GetText(), 50)
	}

	// Add child sections
	for _, child := range section.Children {
		childNode := d.buildSectionTree(child, mode)
		node.Children = append(node.Children, childNode)
	}

	// Add special elements (only in default mode)
	if mode == TreeModeDefault {
		// Code blocks in this section (not children)
		codeBlocks := section.codeBlocks
		if len(codeBlocks) > 0 {
			for lang, count := range CountCodeByLanguage(codeBlocks) {
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

// ExtractPreview extracts the first few words from section content.
func ExtractPreview(text string, maxChars int) string {
	// Skip the heading line
	lines := strings.SplitN(text, "\n", 2)
	if len(lines) < 2 {
		return ""
	}
	content := strings.TrimSpace(lines[1])

	// Skip empty content
	if content == "" {
		return ""
	}

	// Clean up: remove code blocks, collapse whitespace
	// Simple approach: take first non-empty, non-code line
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		// Skip empty lines, code fences, list markers at start
		if line == "" || strings.HasPrefix(line, "```") || strings.HasPrefix(line, "---") {
			continue
		}
		// Skip pure link/image lines
		if strings.HasPrefix(line, "![") || (strings.HasPrefix(line, "[") && strings.Contains(line, "](")) {
			continue
		}

		// Clean up markdown formatting
		line = strings.ReplaceAll(line, "**", "")
		line = strings.ReplaceAll(line, "__", "")
		line = strings.ReplaceAll(line, "`", "")

		// Truncate to maxChars
		if len(line) > maxChars {
			// Try to break at word boundary
			truncated := line[:maxChars]
			if lastSpace := strings.LastIndex(truncated, " "); lastSpace > maxChars/2 {
				truncated = truncated[:lastSpace]
			}
			return truncated + "..."
		}
		return line
	}
	return ""
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

		// Render preview if available
		if node.Preview != "" {
			previewPrefix := prefix
			if isLast {
				previewPrefix += "    "
			} else {
				previewPrefix += "│   "
			}
			buf.WriteString(fmt.Sprintf("%s     \"%s\"\n", previewPrefix, node.Preview))
		}
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
	File    string   // File path
	Section string   // Section heading or record label
	Lines   string   // Line range (e.g., "34-89") or single line
	Match   string   // Snippet with match context
	Fields  []string // Key-value fields from the record (JSONL only)
}

// SearchResults holds all search matches.
type SearchResults struct {
	Query   string
	Matches []*SearchResult
}

type documentLoaderFunc func(path string) (*Document, error)

var traversalExtensions = map[string]struct{}{
	".md":       {},
	".markdown": {},
	".mdown":    {},
	".mkd":      {},
	".html":     {},
	".htm":      {},
	".xhtml":    {},
	".pdf":      {},
	".json":     {},
	".jsonl":    {},
	".ndjson":   {},
	".yaml":     {},
	".yml":      {},
}

func isTraversalFile(path string) bool {
	_, ok := traversalExtensions[strings.ToLower(filepath.Ext(path))]
	return ok
}

// Search finds sections containing the query term.
// For JSONL files, searches line by line for per-record granularity.
// For other formats, searches by section.
func (d *Document) Search(query string) *SearchResults {
	if d.format == FormatJSONL {
		return d.searchJSONL(query)
	}

	results := &SearchResults{Query: query}
	queryLower := strings.ToLower(query)
	hasSearchableSections := false

	for _, section := range d.GetSections() {
		text := section.GetText()
		if text == "" {
			continue
		}
		hasSearchableSections = true
		if strings.Contains(strings.ToLower(text), queryLower) {
			snippet := extractSnippet(text, query, 60)
			results.Matches = append(results.Matches, &SearchResult{
				File:    d.path,
				Section: section.Heading.Text,
				Lines:   fmt.Sprintf("%d-%d", section.Start, section.End),
				Match:   snippet,
			})
		}
	}

	// Non-markdown parsers may not populate section line ranges/source slices.
	// Fall back to readable text so directory search works across all formats.
	if !hasSearchableSections {
		text := d.ReadableText()
		if strings.Contains(strings.ToLower(text), queryLower) {
			section := d.Title()
			if section == "" {
				section = "Document"
			}
			results.Matches = append(results.Matches, &SearchResult{
				File:    d.path,
				Section: section,
				Lines:   "n/a",
				Match:   extractSnippet(text, query, 60),
			})
		}
	}

	return results
}

// searchJSONL searches a JSONL file line by line for per-record matches.
// Pure text search first, then JSON parse only on matching lines to extract
// record structure for display.
func (d *Document) searchJSONL(query string) *SearchResults {
	results := &SearchResults{Query: query}
	queryLower := strings.ToLower(query)

	scanner := bufio.NewScanner(bytes.NewReader(d.source))
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if !strings.Contains(strings.ToLower(trimmed), queryLower) {
			continue
		}

		// Only parse JSON for matching lines to extract structure
		label, fields := jsonlRecordInfo(trimmed)
		snippet := extractSnippet(trimmed, query, 80)

		results.Matches = append(results.Matches, &SearchResult{
			File:    d.path,
			Section: label,
			Lines:   fmt.Sprintf("%d", lineNum),
			Match:   snippet,
			Fields:  fields,
		})
	}

	return results
}

// jsonlRecordInfo extracts a label and key fields from a JSONL record.
// Returns a short label for the heading and a list of "key: value" strings
// showing the record's structure. Only parses JSON for matched lines.
func jsonlRecordInfo(line string) (string, []string) {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return "record", nil
	}

	var label string
	var fields []string

	// Claude session format detection
	typ, hasType := obj["type"].(string)
	if hasType {
		label = typ
		msg, hasMsg := obj["message"].(map[string]interface{})

		if hasMsg {
			role, _ := msg["role"].(string)
			if role != "" {
				label = typ + "/" + role
			}

			// Dig into content blocks for tool info
			if content, ok := msg["content"].([]interface{}); ok {
				for _, block := range content {
					m, ok := block.(map[string]interface{})
					if !ok {
						continue
					}
					blockType, _ := m["type"].(string)
					switch blockType {
					case "tool_use":
						name, _ := m["name"].(string)
						if name != "" {
							label = fmt.Sprintf("%s/tool_use: %s", typ, name)
						}
					case "tool_result":
						label = typ + "/tool_result"
					case "text":
						// Show a preview of the text content
						if text, ok := m["text"].(string); ok && len(text) > 0 {
							preview := text
							if len(preview) > 120 {
								preview = preview[:120] + "..."
							}
							fields = append(fields, "text: "+preview)
						}
					case "thinking":
						fields = append(fields, "thinking: (present)")
					}
				}
			}

			// String content (user messages)
			if content, ok := msg["content"].(string); ok && len(content) > 0 {
				preview := content
				if len(preview) > 120 {
					preview = preview[:120] + "..."
				}
				fields = append(fields, "content: "+preview)
			}
		}

		// Add timestamp if present
		if ts, ok := obj["timestamp"].(string); ok {
			fields = append(fields, "ts: "+ts)
		}

		return label, fields
	}

	// Generic JSONL: show top-level scalar fields
	label = "record"
	for _, key := range []string{"name", "id", "type", "role", "action", "event", "level", "status"} {
		if v, ok := obj[key].(string); ok {
			if label == "record" {
				label = key + ": " + v
			}
			fields = append(fields, key+": "+v)
		}
	}

	// Show remaining scalar fields (up to a few)
	shown := len(fields)
	for k, v := range obj {
		if shown >= 6 {
			break
		}
		switch val := v.(type) {
		case string:
			entry := k + ": " + val
			if len(val) > 80 {
				entry = k + ": " + val[:80] + "..."
			}
			// Avoid duplicating keys already shown
			dup := false
			for _, f := range fields {
				if strings.HasPrefix(f, k+": ") {
					dup = true
					break
				}
			}
			if !dup {
				fields = append(fields, entry)
				shown++
			}
		case float64:
			fields = append(fields, fmt.Sprintf("%s: %v", k, val))
			shown++
		case bool:
			fields = append(fields, fmt.Sprintf("%s: %v", k, val))
			shown++
		}
	}

	return label, fields
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
		buf.WriteString(fmt.Sprintf("  [line %s] %s\n", m.Lines, m.Section))
		for _, field := range m.Fields {
			buf.WriteString(fmt.Sprintf("    %s\n", field))
		}
		if m.Match != "" {
			buf.WriteString(fmt.Sprintf("    > %s\n", m.Match))
		}
	}

	return buf.String()
}

// SearchDir searches all supported document files in a directory.
func SearchDir(dirPath string, query string) (*SearchResults, error) {
	parser := NewParser()
	return SearchDirWithLoader(dirPath, query, parser.ParseFile)
}

// SearchDirWithLoader searches all supported document files using a custom loader.
func SearchDirWithLoader(dirPath string, query string, load documentLoaderFunc) (*SearchResults, error) {
	results := &SearchResults{Query: query}

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if d.IsDir() || !isTraversalFile(path) {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		doc, err := load(path)
		if err != nil {
			return nil // Skip unparseable files
		}

		fileResults := doc.Search(query)
		results.Matches = append(results.Matches, fileResults.Matches...)
		return nil
	})

	return results, err
}

// DirHeading represents a heading with optional preview.
type DirHeading struct {
	Text    string // Heading text with level prefix (e.g., "## Installation")
	Preview string // First few words of content
}

// DirFileNode represents a file or directory in the directory tree.
type DirFileNode struct {
	Name        string         // File or directory name
	Path        string         // Full path
	IsDir       bool           // True if directory
	Format      Format         // Parsed document format
	Lines       int            // Line count (files only)
	Sections    int            // Section count (files only)
	Structure   string         // Format-aware structure label (e.g., sections, keys, records)
	Count       int            // Count of structure units for this format
	TopHeadings []*DirHeading  // Top-level headings for expand/full modes
	Children    []*DirFileNode // Child files/directories
}

// DirTreeResult represents the result of a directory tree query.
type DirTreeResult struct {
	Path       string         // Directory path
	TotalFiles int            // Total supported files
	TotalLines int            // Total lines across all files
	Mode       TreeMode       // Display mode
	Root       []*DirFileNode // Top-level entries
}

// BuildDirTree creates a tree representation of supported document files in a directory.
func BuildDirTree(dirPath string, mode TreeMode) (*DirTreeResult, error) {
	parser := NewParser()
	return BuildDirTreeWithLoader(dirPath, mode, parser.ParseFile)
}

// BuildDirTreeWithLoader creates a tree representation using a custom loader.
func BuildDirTreeWithLoader(dirPath string, mode TreeMode, load documentLoaderFunc) (*DirTreeResult, error) {
	result := &DirTreeResult{
		Path: dirPath,
		Mode: mode,
	}

	root, err := buildDirNode(dirPath, mode, result, load)
	if err != nil {
		return nil, err
	}

	result.Root = root.Children
	return result, nil
}

// buildDirNode recursively builds directory tree nodes.
func buildDirNode(path string, mode TreeMode, result *DirTreeResult, load documentLoaderFunc) (*DirFileNode, error) {
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
		if isTraversalFile(path) {
			doc, err := load(path)
			if err != nil {
				// Skip files that can't be parsed
				node.Lines = -1
				return node, nil
			}

			node.Lines = doc.countLines()
			sections := doc.GetSections()
			node.Sections = len(sections)
			node.Format = doc.Format()
			node.Count, node.Structure = describeStructure(doc)

			result.TotalFiles++
			result.TotalLines += node.Lines

			// Get top-level headings for expand/full modes
			showHeadings := mode == TreeModeFull || mode == TreeModePreview
			if showHeadings {
				for _, section := range doc.GetTableOfContents() {
					h := section.Heading
					heading := &DirHeading{
						Text: formatTreeLabel(doc.Format(), h),
					}
					// Add preview for full mode
					if mode == TreeModeFull {
						heading.Preview = ExtractPreview(section.GetText(), 50)
					}
					node.TopHeadings = append(node.TopHeadings, heading)

					// Also add level 2 headings (direct children)
					for _, child := range section.Children {
						if child.Heading.Level <= 2 {
							childHeading := &DirHeading{
								Text: formatTreeLabel(doc.Format(), child.Heading),
							}
							if mode == TreeModeFull {
								childHeading.Preview = ExtractPreview(child.GetText(), 50)
							}
							node.TopHeadings = append(node.TopHeadings, childHeading)
						}
					}
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

		// For files, only include supported formats
		if !entry.IsDir() && !isTraversalFile(entry.Name()) {
			continue
		}

		child, err := buildDirNode(childPath, mode, result, load)
		if err != nil {
			continue // Skip entries that error
		}

		// Skip empty directories (no supported files)
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
		} else {
			label := node.Structure
			if label == "" {
				label = "sections"
			}

			switch {
			case node.Count == 0:
				buf.WriteString(fmt.Sprintf("%s%s%s (%d lines, no %s)\n", prefix, connector, node.Name, node.Lines, label))
			case node.Count == 1:
				buf.WriteString(fmt.Sprintf("%s%s%s (%d lines, 1 %s)\n", prefix, connector, node.Name, node.Lines, singularLabel(label)))
			default:
				buf.WriteString(fmt.Sprintf("%s%s%s (%d lines, %d %s)\n", prefix, connector, node.Name, node.Lines, node.Count, label))
			}
		}
	}

	// Calculate child prefix
	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	// Render top-level headings in expand/full modes
	showHeadings := t.Mode == TreeModeFull || t.Mode == TreeModePreview
	if showHeadings && len(node.TopHeadings) > 0 {
		for i, heading := range node.TopHeadings {
			hIsLast := i == len(node.TopHeadings)-1 && len(node.Children) == 0
			hConnector := "├── "
			if hIsLast {
				hConnector = "└── "
			}
			buf.WriteString(fmt.Sprintf("%s%s%s\n", childPrefix, hConnector, heading.Text))

			// Add preview in full mode
			if t.Mode == TreeModeFull && heading.Preview != "" {
				previewPrefix := childPrefix
				if hIsLast {
					previewPrefix += "    "
				} else {
					previewPrefix += "│   "
				}
				buf.WriteString(fmt.Sprintf("%s     \"%s\"\n", previewPrefix, heading.Preview))
			}
		}
	}

	// Render children
	for i, child := range node.Children {
		childIsLast := i == len(node.Children)-1
		t.renderNode(buf, child, childPrefix, childIsLast)
	}
}

func formatTreeLabel(format Format, h *Heading) string {
	switch format {
	case FormatMarkdown:
		return fmt.Sprintf("%s %s", strings.Repeat("#", h.Level), h.Text)
	case FormatHTML, FormatPDF:
		return fmt.Sprintf("H%d %s", h.Level, h.Text)
	case FormatJSON, FormatYAML:
		if h.Level <= 1 {
			return fmt.Sprintf("key %s", h.Text)
		}
		return fmt.Sprintf("subkey %s", h.Text)
	case FormatJSONL:
		return fmt.Sprintf("field %s", h.Text)
	default:
		return fmt.Sprintf("H%d %s", h.Level, h.Text)
	}
}

func describeStructure(doc *Document) (int, string) {
	switch doc.Format() {
	case FormatJSON, FormatYAML:
		return len(doc.GetSections()), "keys"
	case FormatJSONL:
		return countJSONLRecords(doc.Source()), "records"
	default:
		return len(doc.GetSections()), "sections"
	}
}

func countJSONLRecords(source []byte) int {
	if len(source) == 0 {
		return 0
	}

	count := 0
	inLine := false
	for _, b := range source {
		if b == '\n' {
			if inLine {
				count++
				inLine = false
			}
		} else if !inLine && b != ' ' && b != '\t' && b != '\r' {
			inLine = true
		}
	}
	if inLine {
		count++
	}
	return count
}

var pluralToSingular = map[string]string{
	"keys":     "key",
	"records":  "record",
	"sections": "section",
}

func singularLabel(label string) string {
	if s, ok := pluralToSingular[label]; ok {
		return s
	}
	return label
}
