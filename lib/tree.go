package mq

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// TreeOptions controls tree traversal bounds (directory mode) and the file
// tree's snippet/filter/asset rendering.
type TreeOptions struct {
	// Directory-mode traversal bounds.
	Depth int // Max directory depth to traverse (0 = unlimited)
	Limit int // Max children per directory (0 = unlimited)

	// File-tree snippet control (--trim).
	TrimN    int    // magnitude of the --trim value (0 = headings only)
	TrimUnit string // "L" lines, "P" paragraphs, "C" chars, "S" sentences, "W" words
	TrimTail bool   // keep the tail (last N) instead of the head
	TrimFull bool   // print the whole section body, no truncation (--full)
	More     string // remainder marker: "" follow TrimUnit, "off" to hide it

	// File-tree filtering.
	MaxLevel int             // --depth N: max heading level shown (0 = unlimited)
	Only     map[string]bool // whitelist tokens: "h1".."h6" and/or kinds
	Drop     map[string]bool // blacklist tokens (drop wins over only)

	// Asset index footer (rendered by the caller for HTML/PDF).
	Assets bool
}

// DefaultFileTreeOptions returns the adaptive default snippet settings used when
// no flags are given (and by the MQL .tree path).
func DefaultFileTreeOptions() TreeOptions {
	return TreeOptions{TrimN: 2, TrimUnit: "L"}
}

// TreeNode represents a node in the document structure tree.
type TreeNode struct {
	Type      string      // "section", "code", "table", "list", "link", "image", "frontmatter"
	Text      string      // Display text (heading text, language, etc.)
	Preview   string      // Section snippet (may contain newlines for multi-line trims)
	Remainder string      // "how much more" marker, e.g. "+3P" (empty if nothing hidden)
	Start     int         // Starting line number
	End       int         // Ending line number
	Level     int         // Heading level (1-6) for sections
	Page      int         // Page number (PDF only, 0 if not applicable)
	Meta      string      // Additional metadata (e.g., "3 blocks", "5 items")
	Children  []*TreeNode // Child nodes
}

// TreeResult represents the result of a .tree query.
type TreeResult struct {
	Path     string      // File path
	Format   Format      // Document format
	Lines    int         // Total line count
	Pages    int         // Total page count (PDF only, 0 if not applicable)
	Root     []*TreeNode // Top-level nodes
	Metadata []string    // Frontmatter field names
}

// BuildTree creates a tree representation of the document. opts controls snippet
// length (--trim), heading filtering (--depth/--only/--drop), and the remainder
// marker. Pass DefaultFileTreeOptions() for the adaptive default.
func (d *Document) BuildTree(opts TreeOptions) *TreeResult {
	result := &TreeResult{
		Path:   d.path,
		Format: d.format,
		Lines:  d.countLines(),
	}

	if d.format == FormatPDF {
		result.Pages = d.PageCount()
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
		node := d.buildSectionTree(section, opts)
		result.Root = append(result.Root, node)
	}

	// Apply heading-level filtering, reparenting kept descendants.
	result.Root = filterTreeLevels(result.Root, opts)

	return result
}

// buildSectionTree recursively builds tree nodes from sections.
func (d *Document) buildSectionTree(section *Section, opts TreeOptions) *TreeNode {
	preview, remainder := sectionPreview(section.GetText(), opts)
	node := &TreeNode{
		Type:      "section",
		Text:      section.Heading.Text,
		Start:     section.Start,
		End:       section.End,
		Level:     section.Heading.Level,
		Page:      section.Heading.Page,
		Preview:   preview,
		Remainder: remainder,
	}

	// Add child sections
	for _, child := range section.Children {
		childNode := d.buildSectionTree(child, opts)
		node.Children = append(node.Children, childNode)
	}

	// Code blocks in this section (not children)
	codeBlocks := section.codeBlocks
	if len(codeBlocks) > 0 && kindAllowed("code", opts) {
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

	return node
}

// filterTreeLevels drops section nodes whose heading level is filtered out by
// --depth/--only/--drop, splicing their (already-filtered) children up to the
// nearest kept ancestor so the spine stays connected.
func filterTreeLevels(nodes []*TreeNode, opts TreeOptions) []*TreeNode {
	if opts.MaxLevel == 0 && !hasLevelTokens(opts.Only) && !hasLevelTokens(opts.Drop) {
		return nodes
	}
	var out []*TreeNode
	for _, n := range nodes {
		n.Children = filterTreeLevels(n.Children, opts)
		if n.Type == "section" && !levelAllowed(n.Level, opts) {
			out = append(out, n.Children...)
			continue
		}
		out = append(out, n)
	}
	return out
}

func levelAllowed(level int, opts TreeOptions) bool {
	tok := fmt.Sprintf("h%d", level)
	if opts.Drop[tok] {
		return false
	}
	if opts.MaxLevel > 0 && level > opts.MaxLevel {
		return false
	}
	if hasLevelTokens(opts.Only) && !opts.Only[tok] {
		return false
	}
	return true
}

// kindAllowed reports whether an element kind (code/images/tables/...) should be
// shown, honoring --only/--drop. Level tokens in Only do not constrain kinds.
func kindAllowed(kind string, opts TreeOptions) bool {
	if opts.Drop[kind] {
		return false
	}
	if hasKindTokens(opts.Only) && !opts.Only[kind] {
		return false
	}
	return true
}

func hasLevelTokens(set map[string]bool) bool {
	for k := range set {
		if len(k) == 2 && k[0] == 'h' && k[1] >= '1' && k[1] <= '6' {
			return true
		}
	}
	return false
}

func hasKindTokens(set map[string]bool) bool {
	for k := range set {
		if !(len(k) == 2 && k[0] == 'h' && k[1] >= '1' && k[1] <= '6') {
			return true
		}
	}
	return false
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
// For binary formats (PDF), counts lines in the extracted readable text
// rather than in the raw source bytes.
func (d *Document) countLines() int {
	if d.format != FormatMarkdown && d.readableText != "" {
		return strings.Count(d.readableText, "\n") + 1
	}
	return strings.Count(string(d.source), "\n") + 1
}

// String renders the tree as a string.
func (t *TreeResult) String() string {
	var buf strings.Builder

	// Header
	if t.Format == FormatPDF && t.Pages > 0 {
		buf.WriteString(fmt.Sprintf("%s (%d pages)\n", t.Path, t.Pages))
	} else {
		buf.WriteString(fmt.Sprintf("%s (%d lines)\n", t.Path, t.Lines))
	}

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
		if t.Format == FormatPDF && node.Page > 0 {
			buf.WriteString(fmt.Sprintf("%s%s%s %s (p. %d)\n",
				prefix, connector, levelPrefix, node.Text, node.Page))
		} else {
			buf.WriteString(fmt.Sprintf("%s%s%s %s (%d-%d)\n",
				prefix, connector, levelPrefix, node.Text, node.Start, node.End))
		}

		// Render preview if available (may span multiple lines).
		if node.Preview != "" {
			previewPrefix := prefix
			if isLast {
				previewPrefix += "    "
			} else {
				previewPrefix += "│   "
			}
			plines := strings.Split(node.Preview, "\n")
			for i, pl := range plines {
				if i == 0 {
					buf.WriteString(fmt.Sprintf("%s     \"%s", previewPrefix, pl))
				} else {
					// align continuation under the opening quote's text
					buf.WriteString(fmt.Sprintf("%s      %s", previewPrefix, pl))
				}
				if i == len(plines)-1 {
					buf.WriteString("\"")
					if node.Remainder != "" {
						buf.WriteString(" …" + node.Remainder)
					}
				}
				buf.WriteString("\n")
			}
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
	Text    string   // Text projection of the matched content
	Raw     string   // Raw matched content safe to display in the terminal
}

// SearchResults holds all search matches.
type SearchResults struct {
	Query   string
	Matches []*SearchResult
}

// SearchTreeResult represents search results rendered as a grouped tree.
type SearchTreeResult struct {
	Query      string
	MatchCount int
	Files      []*SearchTreeFile
}

// SearchTreeFile groups matches for one file.
type SearchTreeFile struct {
	Path    string
	Matches []*SearchTreeMatch
}

// SearchTreeMatch represents one match in tree form.
type SearchTreeMatch struct {
	Label    string
	Lines    string
	Children []string
}

type documentLoaderFunc func(path string) (*Document, error)

// SearchExecutor provides the optimized directory-search hooks used by MQL.
type SearchExecutor interface {
	LoadDocumentBytes(path string, content []byte, info os.FileInfo) (*Document, error)
	SearchCache() *Cache
}

type searchCandidate struct {
	index int
	path  string
	info  os.FileInfo
}

type scannedSearchCandidate struct {
	index   int
	path    string
	info    os.FileInfo
	format  Format
	cached  *SearchResults
	raw     []byte
	matched bool
}

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
	".docx":     {},
	".xlsx":     {},
	".csv":      {},
	".tsv":      {},
	".pptx":     {},
}

func isTraversalFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := traversalExtensions[ext]; ok {
		return true
	}
	// Check extra detectors for formats not in the static map (code, office, etc.).
	// DetectFormat defaults to FormatMarkdown for unknown extensions, so we only
	// accept results from ExtraDetectors (which run before content sniffing).
	for _, detect := range ExtraDetectors {
		if _, ok := detect(path, nil); ok {
			return true
		}
	}
	return false
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
				Text:    text,
				Raw:     text,
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
				Text:    text,
				Raw:     d.RawText(),
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

		// Only parse JSON for matching lines to extract structure + text projection.
		label, fields, text := jsonlRecordProjection(trimmed)
		snippet := extractSnippet(trimmed, query, 80)

		results.Matches = append(results.Matches, &SearchResult{
			File:    d.path,
			Section: label,
			Lines:   fmt.Sprintf("%d", lineNum),
			Match:   snippet,
			Fields:  fields,
			Text:    text,
			Raw:     trimmed,
		})
	}

	return results
}

func searchJSONLContent(path string, content []byte, query string) *SearchResults {
	doc := &Document{
		source: content,
		path:   path,
		format: FormatJSONL,
	}
	return doc.searchJSONL(query)
}

// jsonlRecordInfo extracts a label and key fields from a JSONL record.
// Returns a short label for the heading and a list of "key: value" strings
// showing the record's structure. Only parses JSON for matched lines.
func jsonlRecordProjection(line string) (string, []string, string) {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return "record", nil, line
	}

	label, fields := jsonlRecordInfoFromObject(obj)
	return label, fields, FlattenStructuredData(obj)
}

func jsonlRecordInfo(line string) (string, []string) {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return "record", nil
	}
	return jsonlRecordInfoFromObject(obj)
}

func jsonlRecordInfoFromObject(obj map[string]interface{}) (string, []string) {
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

// TextContent returns the best text representation of a single match.
func (r *SearchResult) TextContent() string {
	if r.Text != "" {
		return r.Text
	}
	if r.Raw != "" {
		return r.Raw
	}
	if len(r.Fields) > 0 {
		return strings.Join(r.Fields, "\n")
	}
	if r.Match != "" {
		return r.Match
	}
	return r.Section
}

// RawContent returns the raw payload for a single match when available.
func (r *SearchResult) RawContent() string {
	if r.Raw != "" {
		return r.Raw
	}
	return r.TextContent()
}

// Texts returns the best text representation of each match.
func (r *SearchResults) Texts() []string {
	results := make([]string, len(r.Matches))
	for i, match := range r.Matches {
		results[i] = match.TextContent()
	}
	return results
}

// RawTexts returns the raw payload of each match when available.
func (r *SearchResults) RawTexts() []string {
	results := make([]string, len(r.Matches))
	for i, match := range r.Matches {
		results[i] = match.RawContent()
	}
	return results
}

// BuildTree renders search results as a grouped tree.
func (r *SearchResults) BuildTree() *SearchTreeResult {
	tree := &SearchTreeResult{
		Query:      r.Query,
		MatchCount: len(r.Matches),
	}

	filesByPath := make(map[string]*SearchTreeFile)
	for _, match := range r.Matches {
		fileNode, ok := filesByPath[match.File]
		if !ok {
			fileNode = &SearchTreeFile{Path: match.File}
			filesByPath[match.File] = fileNode
			tree.Files = append(tree.Files, fileNode)
		}

		children := append([]string{}, match.Fields...)
		if len(children) == 0 && match.Match != "" {
			children = append(children, "match: "+match.Match)
		}

		fileNode.Matches = append(fileNode.Matches, &SearchTreeMatch{
			Label:    match.Section,
			Lines:    match.Lines,
			Children: children,
		})
	}

	return tree
}

// String renders search results as a tree grouped by file.
func (t *SearchTreeResult) String() string {
	if t.MatchCount == 0 {
		return fmt.Sprintf("No matches for %q\n", t.Query)
	}

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("Found %d matches for %q:\n\n", t.MatchCount, t.Query))

	for fileIdx, file := range t.Files {
		buf.WriteString(file.Path + "\n")
		for matchIdx, match := range file.Matches {
			matchLast := matchIdx == len(file.Matches)-1
			matchConnector := "├── "
			childPrefix := "│   "
			if matchLast {
				matchConnector = "└── "
				childPrefix = "    "
			}

			buf.WriteString(fmt.Sprintf("%s[line %s] %s\n", matchConnector, match.Lines, match.Label))
			for childIdx, child := range match.Children {
				childLast := childIdx == len(match.Children)-1
				childConnector := "├── "
				if childLast {
					childConnector = "└── "
				}
				buf.WriteString(fmt.Sprintf("%s%s%s\n", childPrefix, childConnector, child))
			}
		}
		if fileIdx < len(t.Files)-1 {
			buf.WriteString("\n")
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
// It uses a two-phase approach: first a cheap text scan to find files containing the
// query, then an expensive AST parse only on matching files for section context.
func SearchDirWithLoader(dirPath string, query string, load documentLoaderFunc) (*SearchResults, error) {
	results := &SearchResults{Query: query}
	queryLower := strings.ToLower(query)

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

		// Phase 1: cheap text scan — read raw bytes and check for match.
		// This avoids parsing every file into a full AST just to search it.
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip unreadable files
		}
		if !strings.Contains(strings.ToLower(string(raw)), queryLower) {
			return nil // No match — skip expensive parse
		}

		// Phase 2: file contains the query — parse into AST for section context.
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

// SearchDirWithExecutor searches a directory using an executor that can reuse
// already-read bytes and access the persistent cache.
func SearchDirWithExecutor(dirPath string, query string, executor SearchExecutor) (*SearchResults, error) {
	results := &SearchResults{Query: query}
	queryLower := strings.ToLower(query)
	cache := executor.SearchCache()
	var currentDirHash string

	if cache != nil {
		if cached, dirHash, ok := cache.LookupDirSearch(dirPath, query); ok {
			return cached, nil
		} else {
			currentDirHash = dirHash
		}
	}

	candidates, err := collectSearchCandidates(dirPath)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		if cache != nil {
			_ = cache.StoreDirSearch(dirPath, query, currentDirHash, results)
		}
		return results, nil
	}

	scanned := make([]scannedSearchCandidate, len(candidates))
	jobs := make(chan searchCandidate)
	outcomes := make(chan scannedSearchCandidate, len(candidates))

	var wg sync.WaitGroup
	workerCount := searchWorkerCount(len(candidates))
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for candidate := range jobs {
				outcomes <- scanSearchCandidate(candidate, query, queryLower, cache)
			}
		}()
	}

	go func() {
		for _, candidate := range candidates {
			jobs <- candidate
		}
		close(jobs)
		wg.Wait()
		close(outcomes)
	}()

	for outcome := range outcomes {
		scanned[outcome.index] = outcome
	}

	for _, outcome := range scanned {
		if outcome.cached != nil {
			results.Matches = append(results.Matches, outcome.cached.Matches...)
			continue
		}

		fileResults := &SearchResults{Query: query}
		if outcome.matched {
			switch outcome.format {
			case FormatJSONL:
				fileResults = searchJSONLContent(outcome.path, outcome.raw, query)
			default:
				doc, err := executor.LoadDocumentBytes(outcome.path, outcome.raw, outcome.info)
				if err != nil {
					continue
				}
				fileResults = doc.Search(query)
			}
			results.Matches = append(results.Matches, fileResults.Matches...)
		}

		if cache != nil {
			_ = cache.StoreFileSearch(outcome.path, query, outcome.info, fileResults)
		}
	}

	if cache != nil {
		_ = cache.StoreDirSearch(dirPath, query, currentDirHash, results)
	}

	return results, nil
}

func collectSearchCandidates(dirPath string) ([]searchCandidate, error) {
	var candidates []searchCandidate
	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !isTraversalFile(path) {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		candidates = append(candidates, searchCandidate{
			index: len(candidates),
			path:  path,
			info:  info,
		})
		return nil
	})
	return candidates, err
}

func scanSearchCandidate(candidate searchCandidate, query string, queryLower string, cache *Cache) scannedSearchCandidate {
	outcome := scannedSearchCandidate{
		index: candidate.index,
		path:  candidate.path,
		info:  candidate.info,
	}

	if cache != nil {
		if cached := cache.LookupFileSearch(candidate.path, query, candidate.info); cached != nil {
			outcome.cached = cached
			return outcome
		}
	}

	raw, err := os.ReadFile(candidate.path)
	if err != nil {
		return outcome
	}
	if !strings.Contains(strings.ToLower(string(raw)), queryLower) {
		return outcome
	}

	outcome.raw = raw
	outcome.matched = true
	outcome.format = DetectFormat(candidate.path, raw)
	return outcome
}

func searchWorkerCount(total int) int {
	if total <= 1 {
		return 1
	}
	workers := runtime.GOMAXPROCS(0)
	if workers < 2 {
		workers = 2
	}
	if workers > 8 {
		workers = 8
	}
	if total < workers {
		return total
	}
	return workers
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
	Pages       int            // Page count (PDF only, 0 if not applicable)
	Sections    int            // Section count (files only)
	Structure   string         // Format-aware structure label (e.g., sections, keys, records)
	Count       int            // Count of structure units for this format
	TopHeadings []*DirHeading  // Top-level headings for expand/full modes
	Children    []*DirFileNode // Child files/directories
	Truncated   int            // Number of children not shown due to limit
	TotalFiles  int            // Total files in this subtree (for truncated dirs)
}

// DirTreeResult represents the result of a directory tree query.
type DirTreeResult struct {
	Path          string         // Directory path
	TotalFiles    int            // Total supported files
	TotalLines    int            // Total lines across all files
	Options       TreeOptions    // Depth/limit options used
	Root          []*DirFileNode // Top-level entries
	RootTruncated int            // Number of top-level entries not shown due to limit
}

// BuildDirTree creates a tree representation of supported document files in a directory.
func BuildDirTree(dirPath string) (*DirTreeResult, error) {
	parser := NewParser()
	return BuildDirTreeWithLoader(dirPath, TreeOptions{}, parser.ParseFile)
}

// BuildDirTreeWithOptions creates a tree with depth/limit bounds.
func BuildDirTreeWithOptions(dirPath string, opts TreeOptions) (*DirTreeResult, error) {
	parser := NewParser()
	return BuildDirTreeWithLoader(dirPath, opts, parser.ParseFile)
}

// BuildDirTreeWithLoader creates a tree representation using a custom loader.
func BuildDirTreeWithLoader(dirPath string, opts TreeOptions, load documentLoaderFunc) (*DirTreeResult, error) {
	result := &DirTreeResult{
		Path:    dirPath,
		Options: opts,
	}

	root, err := buildDirNode(dirPath, opts, 0, result, load)
	if err != nil {
		return nil, err
	}

	result.Root = root.Children
	result.RootTruncated = root.Truncated
	return result, nil
}

// buildDirNode recursively builds directory tree nodes.
func buildDirNode(path string, opts TreeOptions, currentDepth int, result *DirTreeResult, load documentLoaderFunc) (*DirFileNode, error) {
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

			if doc.Format() == FormatPDF {
				node.Pages = doc.PageCount()
			}

			result.TotalFiles++
			result.TotalLines += node.Lines

			// Show top-level headings with previews
			for _, section := range doc.GetTableOfContents() {
				h := section.Heading
				heading := &DirHeading{
					Text:    formatTreeLabel(doc.Format(), h),
					Preview: ExtractPreview(section.GetText(), 50),
				}
				node.TopHeadings = append(node.TopHeadings, heading)

				// Also add level 2 headings (direct children)
				for _, child := range section.Children {
					if child.Heading.Level <= 2 {
						childHeading := &DirHeading{
							Text:    formatTreeLabel(doc.Format(), child.Heading),
							Preview: ExtractPreview(child.GetText(), 50),
						}
						node.TopHeadings = append(node.TopHeadings, childHeading)
					}
				}
			}
		}
		return node, nil
	}

	// Check if we've hit depth limit
	if opts.Depth > 0 && currentDepth >= opts.Depth {
		// Count files in this subtree without expanding
		fileCount := countFilesInDir(path)
		node.TotalFiles = fileCount
		node.Truncated = -1 // Special value meaning "depth limit reached"
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

	// Filter to valid entries first
	var validEntries []os.DirEntry
	for _, entry := range entries {
		// Skip hidden files/directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		// For files, only include supported formats
		if !entry.IsDir() && !isTraversalFile(entry.Name()) {
			continue
		}
		validEntries = append(validEntries, entry)
	}

	// Apply limit
	limit := len(validEntries)
	if opts.Limit > 0 && limit > opts.Limit {
		node.Truncated = limit - opts.Limit
		limit = opts.Limit
	}

	for i := 0; i < limit; i++ {
		entry := validEntries[i]
		childPath := filepath.Join(path, entry.Name())

		child, err := buildDirNode(childPath, opts, currentDepth+1, result, load)
		if err != nil {
			continue // Skip entries that error
		}

		// Skip empty directories (no supported files) unless truncated by depth
		if child.IsDir && len(child.Children) == 0 && child.Truncated == 0 {
			continue
		}

		node.Children = append(node.Children, child)
	}

	return node, nil
}

// countFilesInDir counts supported files in a directory (non-recursive for speed).
func countFilesInDir(path string) int {
	count := 0
	filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && isTraversalFile(p) && !strings.HasPrefix(d.Name(), ".") {
			count++
		}
		return nil
	})
	return count
}

// String renders the directory tree as a string.
func (t *DirTreeResult) String() string {
	var buf strings.Builder

	// Header
	buf.WriteString(fmt.Sprintf("%s (%d files, %d lines total)\n", t.Path, t.TotalFiles, t.TotalLines))

	// Render nodes
	for i, node := range t.Root {
		isLast := i == len(t.Root)-1 && t.RootTruncated == 0
		t.renderNode(&buf, node, "", isLast)
	}

	// Show root truncation hint
	if t.RootTruncated > 0 {
		buf.WriteString(fmt.Sprintf("└── ... (%d more)\n", t.RootTruncated))
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
		// Check for truncation states
		if node.Truncated == -1 {
			// Depth limit reached - show file count
			buf.WriteString(fmt.Sprintf("%s%s%s/ (%d files, depth limit)\n", prefix, connector, node.Name, node.TotalFiles))
			return
		}
		buf.WriteString(fmt.Sprintf("%s%s%s/\n", prefix, connector, node.Name))
	} else {
		if node.Lines < 0 {
			buf.WriteString(fmt.Sprintf("%s%s%s (parse error)\n", prefix, connector, node.Name))
		} else {
			label := node.Structure
			if label == "" {
				label = "sections"
			}

			// Use "pages" instead of "lines" for PDFs
			sizeLabel := fmt.Sprintf("%d lines", node.Lines)
			if node.Format == FormatPDF && node.Pages > 0 {
				sizeLabel = fmt.Sprintf("%d pages", node.Pages)
			}

			switch {
			case node.Count == 0:
				buf.WriteString(fmt.Sprintf("%s%s%s (%s, no %s)\n", prefix, connector, node.Name, sizeLabel, label))
			case node.Count == 1:
				buf.WriteString(fmt.Sprintf("%s%s%s (%s, 1 %s)\n", prefix, connector, node.Name, sizeLabel, singularLabel(label)))
			default:
				buf.WriteString(fmt.Sprintf("%s%s%s (%s, %d %s)\n", prefix, connector, node.Name, sizeLabel, node.Count, label))
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

	// Render top-level headings with previews
	if len(node.TopHeadings) > 0 {
		for i, heading := range node.TopHeadings {
			hIsLast := i == len(node.TopHeadings)-1 && len(node.Children) == 0 && node.Truncated == 0
			hConnector := "├── "
			if hIsLast {
				hConnector = "└── "
			}
			buf.WriteString(fmt.Sprintf("%s%s%s\n", childPrefix, hConnector, heading.Text))

			if heading.Preview != "" {
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
		childIsLast := i == len(node.Children)-1 && node.Truncated == 0
		t.renderNode(buf, child, childPrefix, childIsLast)
	}

	// Show truncation hint
	if node.Truncated > 0 {
		buf.WriteString(fmt.Sprintf("%s└── ... (%d more)\n", childPrefix, node.Truncated))
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
