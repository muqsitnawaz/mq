package mq

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// RgMatch represents a single ripgrep match.
type RgMatch struct {
	Path    string
	LineNum int
	Content string
}

// RgSearchOptions configures ripgrep search behavior.
type RgSearchOptions struct {
	// FileTypes to search (e.g., "jsonl", "md"). Empty means all supported types.
	FileTypes []string
	// MaxResults limits the number of results (0 = unlimited).
	MaxResults int
	// Workers is the number of parallel workers for processing matches.
	Workers int
	// CaseSensitive enables case-sensitive search.
	CaseSensitive bool
}

// DefaultRgSearchOptions returns sensible defaults.
func DefaultRgSearchOptions() RgSearchOptions {
	return RgSearchOptions{
		Workers: 8,
	}
}

// RgSearch performs a ripgrep-based search across files.
// It uses rg for fast text matching, then parses only the matched lines.
func RgSearch(query string, paths []string, opts RgSearchOptions) (*SearchResults, error) {
	if len(paths) == 0 {
		return &SearchResults{Query: query}, nil
	}

	// Build rg command
	args := buildRgArgs(query, paths, opts)

	cmd := exec.Command("rg", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			return nil, fmt.Errorf("ripgrep (rg) not found in PATH - install with: brew install ripgrep")
		}
		return nil, fmt.Errorf("failed to start rg: %w", err)
	}

	// Parse rg output and send to workers
	matches := make(chan RgMatch, 100)
	var parseWg sync.WaitGroup
	parseWg.Add(1)
	go func() {
		defer parseWg.Done()
		defer close(matches)

		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 1024*1024)
		scanner.Buffer(buf, 10*1024*1024)

		count := 0
		for scanner.Scan() {
			line := scanner.Text()
			match, ok := parseRgLine(line)
			if ok {
				matches <- match
				count++
				if opts.MaxResults > 0 && count >= opts.MaxResults {
					break
				}
			}
		}
	}()

	// Process matches with worker pool
	results := &SearchResults{Query: query}
	var resultsMu sync.Mutex
	var processWg sync.WaitGroup

	workers := opts.Workers
	if workers <= 0 {
		workers = 8
	}

	for i := 0; i < workers; i++ {
		processWg.Add(1)
		go func() {
			defer processWg.Done()
			for match := range matches {
				result := processMatch(match, query)
				if result != nil {
					resultsMu.Lock()
					results.Matches = append(results.Matches, result)
					resultsMu.Unlock()
				}
			}
		}()
	}

	parseWg.Wait()
	processWg.Wait()

	// Kill rg if we hit MaxResults early
	_ = cmd.Process.Kill()
	_ = cmd.Wait()

	return results, nil
}

// buildRgArgs constructs ripgrep arguments.
func buildRgArgs(query string, paths []string, opts RgSearchOptions) []string {
	args := []string{
		"--line-number",
		"--no-heading",
		"--color=never",
		"--with-filename",
	}

	if !opts.CaseSensitive {
		args = append(args, "--ignore-case")
	}

	// File type filters
	if len(opts.FileTypes) > 0 {
		for _, ft := range opts.FileTypes {
			switch ft {
			case "jsonl":
				args = append(args, "--type-add", "jsonl:*.jsonl", "-t", "jsonl")
			case "md", "markdown":
				args = append(args, "-t", "md")
			case "json":
				args = append(args, "-t", "json")
			case "yaml", "yml":
				args = append(args, "-t", "yaml")
			}
		}
	} else {
		// All supported types
		args = append(args,
			"--type-add", "jsonl:*.jsonl",
			"--type-add", "ndjson:*.ndjson",
		)
	}

	args = append(args, "--", query)
	args = append(args, paths...)

	return args
}

// parseRgLine parses "path:line:content" format.
func parseRgLine(line string) (RgMatch, bool) {
	// Find first colon (after path)
	firstColon := strings.Index(line, ":")
	if firstColon == -1 {
		return RgMatch{}, false
	}

	// Find second colon (after line number)
	rest := line[firstColon+1:]
	secondColon := strings.Index(rest, ":")
	if secondColon == -1 {
		return RgMatch{}, false
	}

	path := line[:firstColon]
	lineNumStr := rest[:secondColon]
	content := rest[secondColon+1:]

	lineNum, err := strconv.Atoi(lineNumStr)
	if err != nil {
		return RgMatch{}, false
	}

	return RgMatch{
		Path:    path,
		LineNum: lineNum,
		Content: content,
	}, true
}

// processMatch converts RgMatch to SearchResult based on file type.
func processMatch(match RgMatch, query string) *SearchResult {
	ext := strings.ToLower(filepath.Ext(match.Path))

	switch ext {
	case ".jsonl", ".ndjson":
		return processJSONLMatch(match, query)
	case ".md", ".markdown", ".mdown", ".mkd":
		return processMarkdownMatch(match, query)
	default:
		return &SearchResult{
			File:    match.Path,
			Section: "line",
			Lines:   strconv.Itoa(match.LineNum),
			Match:   rgSnippet(match.Content, query, 80),
		}
	}
}

// processJSONLMatch parses JSON and extracts structure.
func processJSONLMatch(match RgMatch, query string) *SearchResult {
	label, fields := rgExtractJSONLInfo(match.Content)
	return &SearchResult{
		File:    match.Path,
		Section: label,
		Lines:   strconv.Itoa(match.LineNum),
		Match:   rgSnippet(match.Content, query, 80),
		Fields:  fields,
	}
}

// rgExtractJSONLInfo extracts label and fields from JSON line.
func rgExtractJSONLInfo(line string) (string, []string) {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return "record", nil
	}

	var label string
	var fields []string

	// Rush message format
	if typ, ok := obj["type"].(string); ok {
		label = typ
		if role, ok := obj["role"].(string); ok {
			fields = append(fields, fmt.Sprintf("role: %s", role))
		}
		if name, ok := obj["name"].(string); ok {
			fields = append(fields, fmt.Sprintf("name: %s", name))
		}
	}

	// Log format
	if level, ok := obj["level"].(string); ok {
		label = fmt.Sprintf("level: %s", level)
		if agent, ok := obj["agent"].(string); ok {
			fields = append(fields, fmt.Sprintf("agent: %s", agent))
		}
		if msg, ok := obj["message"].(string); ok {
			if len(msg) > 50 {
				msg = msg[:50] + "..."
			}
			fields = append(fields, fmt.Sprintf("message: %s", msg))
		}
	}

	if label == "" {
		label = "record"
	}

	return label, fields
}

// processMarkdownMatch extracts context from markdown.
func processMarkdownMatch(match RgMatch, query string) *SearchResult {
	return &SearchResult{
		File:    match.Path,
		Section: fmt.Sprintf("line %d", match.LineNum),
		Lines:   strconv.Itoa(match.LineNum),
		Match:   rgSnippet(match.Content, query, 80),
	}
}

// RgSearchDir searches a directory using ripgrep.
// This is much faster than SearchDir for large directories.
func RgSearchDir(dirPath string, query string) (*SearchResults, error) {
	return RgSearch(query, []string{dirPath}, DefaultRgSearchOptions())
}

// RgSearchDirWithOptions searches a directory with custom options.
func RgSearchDirWithOptions(dirPath string, query string, opts RgSearchOptions) (*SearchResults, error) {
	return RgSearch(query, []string{dirPath}, opts)
}

// rgSnippet extracts snippet around query match.
func rgSnippet(content, query string, maxLen int) string {
	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(query)

	idx := strings.Index(lowerContent, lowerQuery)
	if idx == -1 {
		if len(content) > maxLen {
			return content[:maxLen] + "..."
		}
		return content
	}

	start := idx - maxLen/2
	if start < 0 {
		start = 0
	}
	end := start + maxLen
	if end > len(content) {
		end = len(content)
		start = end - maxLen
		if start < 0 {
			start = 0
		}
	}

	snippet := content[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}

	return snippet
}
