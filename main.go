package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/muqsitnawaz/mq/mql"
)

var version = "dev"

const (
	repo          = "muqsitnawaz/mq"
	releaseAPIURL = "https://api.github.com/repos/" + repo + "/releases/latest"
)

func main() {
	// Pull --flags out of the args, leaving positional [path, query].
	args, opts, assetsSet, err := parseFileTreeFlags(os.Args[1:])
	if err != nil {
		log.Fatalf("%v", err)
	}

	if len(args) >= 1 {
		switch args[0] {
		case "-h", "--help", "help":
			printUsage()
			os.Exit(0)
		case "-v", "--version", "version":
			fmt.Printf("mq %s\n", version)
			os.Exit(0)
		case "upgrade":
			if err := selfUpgrade(); err != nil {
				log.Fatalf("Upgrade failed: %v", err)
			}
			os.Exit(0)
		}
	}

	// Check for updates (non-blocking, silent on error)
	checkForUpdates()

	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	path := args[0]
	query := ""
	if len(args) >= 2 {
		query = args[1]
	}

	// Check if path is a directory
	info, err := os.Stat(path)
	if err != nil {
		log.Fatalf("Failed to stat path: %v", err)
	}

	if info.IsDir() {
		handleDirectory(path, query)
		return
	}

	// Load the document (auto-detect format)
	engine := mql.New()
	defer engine.Close()
	doc, err := engine.LoadDocument(path)
	if err != nil {
		log.Fatalf("Failed to load document: %v", err)
	}

	// No query: render the default tree with the flag-driven options, plus the
	// asset index footer for HTML/PDF.
	if query == "" {
		if !assetsSet {
			opts.Assets = doc.Format() == mq.FormatHTML || doc.Format() == mq.FormatPDF
		}
		result := doc.BuildTree(opts)
		displayResult(result)
		printAssets(doc, opts)
		return
	}

	// Execute the query
	result, err := engine.Query(doc, query)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	// Display results
	displayResult(result)
}

// parseFileTreeFlags extracts the file-tree flags from args, returning the
// remaining positional args, the assembled options, and whether the asset
// footer was explicitly toggled.
func parseFileTreeFlags(args []string) (positional []string, opts mq.TreeOptions, assetsSet bool, err error) {
	opts = mq.DefaultFileTreeOptions()

	next := func(i *int, flag string) (string, error) {
		if *i+1 >= len(args) {
			return "", fmt.Errorf("%s requires a value", flag)
		}
		*i++
		return args[*i], nil
	}

	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--trim":
			v, e := next(&i, "--trim")
			if e != nil {
				return nil, opts, false, e
			}
			if e := parseTrim(v, &opts); e != nil {
				return nil, opts, false, e
			}
		case strings.HasPrefix(a, "--trim="):
			if e := parseTrim(strings.TrimPrefix(a, "--trim="), &opts); e != nil {
				return nil, opts, false, e
			}
		case a == "--full":
			opts.TrimFull = true
		case a == "--bare":
			opts.TrimN = 0
			opts.TrimFull = false
			opts.Assets = false
			assetsSet = true
		case a == "--depth":
			v, e := next(&i, "--depth")
			if e != nil {
				return nil, opts, false, e
			}
			n, e := parseInt(v)
			if e != nil {
				return nil, opts, false, fmt.Errorf("--depth requires an integer")
			}
			opts.MaxLevel = n
		case a == "--only":
			v, e := next(&i, "--only")
			if e != nil {
				return nil, opts, false, e
			}
			opts.Only = parseTokenSet(v)
		case a == "--drop":
			v, e := next(&i, "--drop")
			if e != nil {
				return nil, opts, false, e
			}
			opts.Drop = parseTokenSet(v)
		case a == "--more":
			v, e := next(&i, "--more")
			if e != nil {
				return nil, opts, false, e
			}
			opts.More = v
		case a == "--assets":
			opts.Assets = true
			assetsSet = true
		case a == "--no-assets":
			opts.Assets = false
			assetsSet = true
		default:
			positional = append(positional, a)
		}
	}
	return positional, opts, assetsSet, nil
}

// parseTrim parses a --trim value like "4L", "3P", "200C", "-3L", "full".
func parseTrim(v string, opts *mq.TreeOptions) error {
	v = strings.TrimSpace(v)
	if v == "full" || v == "all" {
		opts.TrimFull = true
		return nil
	}
	tail := false
	if strings.HasPrefix(v, "-") {
		tail = true
		v = v[1:]
	}
	unit := "L"
	if v != "" {
		last := v[len(v)-1]
		if last < '0' || last > '9' {
			switch strings.ToUpper(string(last)) {
			case "L", "P", "C", "S", "W":
				unit = strings.ToUpper(string(last))
				v = v[:len(v)-1]
			default:
				return fmt.Errorf("unknown --trim unit %q (use L/P/C/S/W)", string(last))
			}
		}
	}
	n, e := parseInt(strings.TrimSpace(v))
	if e != nil {
		return fmt.Errorf("bad --trim value (want e.g. 4L, 3P, 200C, full)")
	}
	opts.TrimN = n
	opts.TrimUnit = unit
	opts.TrimTail = tail
	opts.TrimFull = false
	return nil
}

// parseTokenSet parses "h1,h2,images" into a lowercased set.
func parseTokenSet(v string) map[string]bool {
	set := map[string]bool{}
	for _, tok := range strings.Split(v, ",") {
		tok = strings.ToLower(strings.TrimSpace(tok))
		if tok != "" {
			set[tok] = true
		}
	}
	return set
}

// printAssets renders the asset index footer when enabled.
func printAssets(doc *mq.Document, opts mq.TreeOptions) {
	if !opts.Assets {
		return
	}
	fmt.Print(doc.BuildAssets().Render(opts))
}

func printUsage() {
	fmt.Printf("mq %s - Structure-aware queries for documents and directories\n\n", version)
	fmt.Println("Usage: mq <file|directory> [query]")
	fmt.Println("")
	fmt.Println("Formats: Markdown, HTML, PDF, JSON, YAML, Code (Go/Python/TS/Rust/...),")
	fmt.Println("         CSV, XLSX, DOCX, PPTX")
	fmt.Println("")
	fmt.Println("WHEN TO USE MQ")
	fmt.Println("")
	fmt.Println("  Before reading files in a directory you haven't seen:")
	fmt.Println("    mq docs/ '.tree | depth(1)'     See every file, its size, and what it covers")
	fmt.Println("                                     Then decide which files to read fully")
	fmt.Println("")
	fmt.Println("  When a file is too large to read whole (200+ lines):")
	fmt.Println("    mq GUIDE.md .tree                See all sections with line ranges")
	fmt.Println("    mq GUIDE.md '.section(\"Auth\") | .text'   Extract just what you need")
	fmt.Println("")
	fmt.Println("  When searching for a topic across many documents:")
	fmt.Println("    mq research/ '.search(\"pricing\")'  Section-level matches across all files")
	fmt.Println("")
	fmt.Println("  When a file is small (<100 lines), just read it directly.")
	fmt.Println("")
	fmt.Println("WORKFLOW")
	fmt.Println("")
	fmt.Println("  1. Map:     mq <dir>  '.tree | depth(1)'              What's here?")
	fmt.Println("  2. Narrow:  mq <file> .tree                           What sections?")
	fmt.Println("  3. Extract: mq <file> '.section(\"Name\") | .text'      Get the content")
	fmt.Println("")
	fmt.Println("  Each step returns a compact result. You build understanding incrementally")
	fmt.Println("  instead of dumping entire files into context.")
	fmt.Println("")
	fmt.Println("SCALE")
	fmt.Println("")
	fmt.Println("  Warm cache (after first parse):")
	fmt.Println("    58 PDFs, 148MB   .tree in 0.96s    .search in ~2s")
	fmt.Println("    123 PDFs, 365MB  .tree in 2.97s    .search in ~4s")
	fmt.Println("    30K-line markdown dir  .tree in <1s")
	fmt.Println("")
	fmt.Println("  Cold parse is slow for PDFs (minutes). But it's one-time — the cache")
	fmt.Println("  persists across sessions. Every subsequent query is sub-second to seconds.")
	fmt.Println("")
	fmt.Println("SELECTORS")
	fmt.Println("")
	fmt.Println("  .tree                Show document/directory structure")
	fmt.Println("  .section(\"Name\")     Full section by heading (any level)")
	fmt.Println("  .h1 ... .h6          List headings at that level")
	fmt.Println("  .h2(\"Name\")          Full section under matching H2")
	fmt.Println("  .search(\"term\")      Find sections containing term")
	fmt.Println("  .code(\"lang\")        Get code blocks by language")
	fmt.Println("  .headings            List all headings")
	fmt.Println("  .links               List all links")
	fmt.Println("")
	fmt.Println("  Matching: exact -> case-insensitive -> prefix -> contains.")
	fmt.Println("  .section(\"Auth\") matches \"Authentication\".")
	fmt.Println("  .h3(\"Chapter 1\") matches \"Chapter 1: What Is an Agent?\".")
	fmt.Println("")
	fmt.Println("PIPES (chain after selectors)")
	fmt.Println("")
	fmt.Println("  | .text              Extract text content")
	fmt.Println("  | .raw               Extract source text")
	fmt.Println("  | .tree              Show structure of selection")
	fmt.Println("  | .nth(N)            Pick the Nth result (0-based)")
	fmt.Println("  | depth(N)           Limit directory traversal depth")
	fmt.Println("  | limit(N)           Max entries per directory")
	fmt.Println("")
	fmt.Println("EXAMPLES")
	fmt.Println("")
	fmt.Println("  # Triage a directory before reading anything")
	fmt.Println("  mq agents/ '.tree | depth(1)'")
	fmt.Println("")
	fmt.Println("  # List all H2 headings, then read one")
	fmt.Println("  mq GUIDE.md .h2")
	fmt.Println("  mq GUIDE.md '.h2(\"Testing\") | .text'")
	fmt.Println("")
	fmt.Println("  # Extract a section (any level) by partial name")
	fmt.Println("  mq GUIDE.md '.section(\"Auth\") | .text'")
	fmt.Println("")
	fmt.Println("  # Search across all docs for a topic")
	fmt.Println("  mq docs/ '.search(\"auth\")'")
	fmt.Println("")
	fmt.Println("  # Bounded traversal for very large directories")
	fmt.Println("  mq corpus/ '.tree | depth(2) | limit(50)'")
	fmt.Println("")
	fmt.Println("  # Works on any supported format")
	fmt.Println("  mq report.pdf '.h2(\"Results\") | .text'")
	fmt.Println("  mq page.html .tree")
	fmt.Println("  mq src/ .tree                          # Code: functions, classes, structs")
	fmt.Println("  mq data.xlsx '.section(\"Sheet1\") | .text'")
	fmt.Println("  mq report.docx .headings")
	fmt.Println("")
	fmt.Println("COMMANDS")
	fmt.Println("")
	fmt.Println("  upgrade              Upgrade to latest version")
	fmt.Println("")
	fmt.Println("FLAGS")
	fmt.Println("")
	fmt.Println("  -h, --help           Show this help")
	fmt.Println("  -v, --version        Show version")
	fmt.Println("")
	fmt.Println("  Tree snippet (mq <file>):")
	fmt.Println("    --trim <N><unit>   Snippet size; unit = L lines, P paragraphs, C chars,")
	fmt.Println("                       S sentences, W words (e.g. 4L, 3P, 200C). Default: 2L.")
	fmt.Println("                       Negative N keeps the tail (--trim -3L). --trim 0 = none.")
	fmt.Println("    --full             Print whole section bodies (alias: --trim full)")
	fmt.Println("    --bare             Headings only: no snippets, no asset index")
	fmt.Println("    --more off         Hide the '…+N' remainder marker")
	fmt.Println("    --depth <N>        Max heading level shown (--depth 2 = just h1/h2)")
	fmt.Println("    --only <list>      Show only these: levels h1..h6 and/or kinds")
	fmt.Println("                       (images,figures,links,tables,code,svg,media)")
	fmt.Println("    --drop <list>      Hide these (drop wins over only)")
	fmt.Println("    --assets           Force the asset index footer (default on for HTML/PDF)")
	fmt.Println("    --no-assets        Suppress the asset index footer")
}

func checkForUpdates() {
	if version == "dev" {
		return
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(releaseAPIURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(version, "v")

	if latest != current && latest > current {
		fmt.Fprintf(os.Stderr, "A new version is available: %s (current: %s). Run 'mq upgrade' to update.\n\n", release.TagName, version)
	}
}

func selfUpgrade() error {
	fmt.Println("Checking for updates...")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(releaseAPIURL)
	if err != nil {
		return fmt.Errorf("failed to check releases: %w", err)
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to parse release: %w", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(version, "v")

	if latest == current {
		fmt.Printf("Already at latest version (%s)\n", version)
		return nil
	}

	// Find the right asset
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}

	assetName := fmt.Sprintf("mq_%s_%s.%s", goos, goarch, ext)
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no binary available for %s/%s", goos, goarch)
	}

	fmt.Printf("Downloading %s...\n", release.TagName)

	// Download to temp file
	resp, err = client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	tmpDir, err := os.MkdirTemp("", "mq-upgrade")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, assetName)
	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	f.Close()

	// Extract binary
	binaryPath := filepath.Join(tmpDir, "mq")
	if goos == "windows" {
		binaryPath += ".exe"
	}

	if ext == "zip" {
		if err := extractZip(archivePath, tmpDir); err != nil {
			return err
		}
	} else {
		if err := extractTarGz(archivePath, tmpDir); err != nil {
			return err
		}
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return err
	}

	// Replace current binary
	if err := os.Rename(binaryPath, execPath); err != nil {
		// Try copy if rename fails (cross-device)
		src, err := os.Open(binaryPath)
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.OpenFile(execPath, os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return err
		}
	}

	fmt.Printf("Upgraded to %s\n\n", release.TagName)

	// Show changelog for this version
	showChangelog(release.TagName)

	return nil
}

//go:embed CHANGELOG.md
var changelogContent string

func showChangelog(tag string) {
	ver := strings.TrimPrefix(tag, "v")
	// Find the section for this version in the changelog
	marker := fmt.Sprintf("## [%s]", ver)
	idx := strings.Index(changelogContent, marker)
	if idx < 0 {
		return
	}

	// Find the next ## section (end of this version's entry)
	rest := changelogContent[idx:]
	nextSection := strings.Index(rest[3:], "\n## ")
	if nextSection > 0 {
		rest = rest[:nextSection+3]
	}

	fmt.Println("What's new:")
	fmt.Println(strings.TrimSpace(rest))
}

func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if header.Typeflag == tar.TypeReg {
			outPath := filepath.Join(destDir, header.Name)
			outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}

func extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outPath := filepath.Join(destDir, f.Name)
		outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// parseMethodCall parses queries like .method("arg"), .method('arg'), or .method(arg)
// Returns the method name, argument value (with quotes stripped), and whether parsing succeeded.
// This handles different shell quoting behaviors across Windows CMD, PowerShell, and Unix shells.
func parseMethodCall(query string) (method string, arg string, ok bool) {
	if !strings.HasPrefix(query, ".") {
		return "", "", false
	}

	query = query[1:]

	method, rest, found := strings.Cut(query, "(")
	if !found {
		return method, "", true
	}

	// Must end with closing paren
	if !strings.HasSuffix(rest, ")") {
		return "", "", false
	}

	arg = strings.TrimSuffix(rest, ")")

	// Strip quotes from arg if present (handle ", ', or no quotes)
	if len(arg) >= 2 {
		if (arg[0] == '"' && arg[len(arg)-1] == '"') ||
			(arg[0] == '\'' && arg[len(arg)-1] == '\'') {
			arg = arg[1 : len(arg)-1]
		}
	}

	return method, arg, true
}

func handleDirectory(path string, query string) {
	result, err := runDirectoryQuery(path, query)
	if err != nil {
		log.Fatal(err)
	}
	displayResult(result)
}

func runDirectoryQuery(path string, query string) (interface{}, error) {
	// Directory mode: default to tree (always includes previews now)
	if query == "" {
		query = ".tree"
	}

	parts := splitPipeline(query)
	method, arg, ok := parseMethodCall(parts[0])
	if !ok {
		return nil, fmt.Errorf("Invalid query format. Supported: .tree, .search(\"term\")")
	}

	switch method {
	case "tree":
		opts := mq.TreeOptions{}
		for _, part := range parts[1:] {
			modMethod, modArg, ok := parseMethodCall(part)
			if !ok {
				modMethod, modArg, ok = parseFunctionCall(part)
				if !ok {
					return nil, fmt.Errorf("Invalid modifier: %s", part)
				}
			}

			switch modMethod {
			case "depth":
				if n, err := parseInt(modArg); err == nil {
					opts.Depth = n
				} else {
					return nil, fmt.Errorf("depth() requires an integer argument")
				}
			case "limit":
				if n, err := parseInt(modArg); err == nil {
					opts.Limit = n
				} else {
					return nil, fmt.Errorf("limit() requires an integer argument")
				}
			default:
				return nil, fmt.Errorf("Unknown modifier: %s. Supported: depth(N), limit(N)", modMethod)
			}
		}

		if arg != "" {
			fmt.Fprintf(os.Stderr, "Warning: .tree(%q) is deprecated — .tree now adapts automatically. Ignoring argument.\n", arg)
		}
		return mql.BuildDirTreeWithOptions(path, opts)

	case "search":
		if arg == "" {
			return nil, fmt.Errorf("Search requires a term: .search(\"term\")")
		}
		result, err := mql.SearchDir(path, arg)
		if err != nil {
			return nil, fmt.Errorf("Search failed: %w", err)
		}
		return applySearchPipeline(result, parts[1:])

	default:
		return nil, fmt.Errorf("Unknown method: .%s. Supported: .tree, .search", method)
	}
}

func applySearchPipeline(current interface{}, parts []string) (interface{}, error) {
	for _, part := range parts {
		method, arg, ok := parseMethodCall(part)
		if !ok {
			return nil, fmt.Errorf("Invalid selector: %s", part)
		}

		switch method {
		case "text":
			switch v := current.(type) {
			case *mq.SearchResults:
				current = v.Texts()
			case []*mq.SearchResult:
				current = (&mq.SearchResults{Matches: v}).Texts()
			case *mq.SearchResult:
				current = v.TextContent()
			default:
				return nil, fmt.Errorf("Cannot apply .text to %T", current)
			}
		case "raw":
			switch v := current.(type) {
			case *mq.SearchResults:
				current = v.RawTexts()
			case []*mq.SearchResult:
				current = (&mq.SearchResults{Matches: v}).RawTexts()
			case *mq.SearchResult:
				current = v.RawContent()
			default:
				return nil, fmt.Errorf("Cannot apply .raw to %T", current)
			}
		case "tree":
			switch v := current.(type) {
			case *mq.SearchResults:
				current = v.BuildTree()
			case []*mq.SearchResult:
				current = (&mq.SearchResults{Matches: v}).BuildTree()
			case *mq.SearchResult:
				current = (&mq.SearchResults{Matches: []*mq.SearchResult{v}}).BuildTree()
			default:
				return nil, fmt.Errorf("Cannot apply .tree to %T", current)
			}
		case "length":
			switch v := current.(type) {
			case *mq.SearchResults:
				current = len(v.Matches)
			case []*mq.SearchResult:
				current = len(v)
			case []string:
				current = len(v)
			default:
				return nil, fmt.Errorf("Cannot apply .length to %T", current)
			}
		case "nth":
			index, err := parseInt(arg)
			if err != nil {
				return nil, fmt.Errorf(".nth() requires an integer argument")
			}
			switch v := current.(type) {
			case *mq.SearchResults:
				if index < 0 || index >= len(v.Matches) {
					return nil, fmt.Errorf("index out of range: %d", index)
				}
				current = v.Matches[index]
			case []*mq.SearchResult:
				if index < 0 || index >= len(v) {
					return nil, fmt.Errorf("index out of range: %d", index)
				}
				current = v[index]
			case []string:
				if index < 0 || index >= len(v) {
					return nil, fmt.Errorf("index out of range: %d", index)
				}
				current = v[index]
			default:
				return nil, fmt.Errorf("Cannot apply .nth to %T", current)
			}
		default:
			return nil, fmt.Errorf("Unknown selector: .%s. Supported after directory .search: .text, .raw, .tree, .length, .nth(index)", method)
		}
	}

	return current, nil
}

// splitPipeline splits a query on | while respecting parentheses.
func splitPipeline(query string) []string {
	var parts []string
	var current strings.Builder
	depth := 0

	for _, ch := range query {
		switch ch {
		case '(':
			depth++
			current.WriteRune(ch)
		case ')':
			depth--
			current.WriteRune(ch)
		case '|':
			if depth == 0 {
				if s := strings.TrimSpace(current.String()); s != "" {
					parts = append(parts, s)
				}
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if s := strings.TrimSpace(current.String()); s != "" {
		parts = append(parts, s)
	}

	return parts
}

// parseFunctionCall parses function calls like depth(3) or limit(50).
func parseFunctionCall(s string) (name string, arg string, ok bool) {
	name, rest, found := strings.Cut(s, "(")
	if !found {
		return "", "", false
	}
	if !strings.HasSuffix(rest, ")") {
		return "", "", false
	}
	arg = strings.TrimSuffix(rest, ")")
	return strings.TrimSpace(name), strings.TrimSpace(arg), true
}

// parseInt parses an integer from a string.
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func showDocumentInfo(doc *mq.Document) {
	fmt.Printf("Document: %s\n", doc.Path())
	fmt.Printf("Format: %s\n", doc.Format())
	fmt.Println(strings.Repeat("=", len(doc.Path())+10))

	// Show metadata
	if meta := doc.Metadata(); meta != nil {
		fmt.Println("\nMetadata:")
		if owner, ok := doc.GetOwner(); ok {
			fmt.Printf("  Owner: %s\n", owner)
		}
		if tags := doc.GetTags(); len(tags) > 0 {
			fmt.Printf("  Tags: %v\n", tags)
		}
		if priority, ok := doc.GetPriority(); ok {
			fmt.Printf("  Priority: %s\n", priority)
		}
	}

	// For data formats (JSON, JSONL, YAML), show data-specific info
	format := doc.Format()
	if format == mq.FormatJSON || format == mq.FormatJSONL || format == mq.FormatYAML {
		showDataInfo(doc)
		return
	}

	// Show structure for document formats
	fmt.Println("\nStructure:")
	headings := doc.GetHeadings()
	fmt.Printf("  Headings: %d\n", len(headings))

	sections := doc.GetSections()
	fmt.Printf("  Sections: %d\n", len(sections))

	codeBlocks := doc.GetCodeBlocks()
	fmt.Printf("  Code blocks: %d\n", len(codeBlocks))

	// Show code languages
	if len(codeBlocks) > 0 {
		langs := make(map[string]int)
		for _, block := range codeBlocks {
			if block.Language != "" {
				langs[block.Language]++
			}
		}
		if len(langs) > 0 {
			fmt.Println("    Languages:")
			for lang, count := range langs {
				fmt.Printf("      - %s: %d\n", lang, count)
			}
		}
	}

	tables := doc.GetTables()
	if len(tables) > 0 {
		fmt.Printf("  Tables: %d\n", len(tables))
	}

	links := doc.GetLinks()
	if len(links) > 0 {
		fmt.Printf("  Links: %d\n", len(links))
	}

	images := doc.GetImages()
	if len(images) > 0 {
		fmt.Printf("  Images: %d\n", len(images))
	}

	// Show table of contents
	fmt.Println("\nTable of Contents:")
	for _, heading := range headings {
		indent := strings.Repeat("  ", heading.Level-1)
		fmt.Printf("%s- %s\n", indent, heading.Text)
	}
}

func showDataInfo(doc *mq.Document) {
	title := doc.Title()
	if title != "" {
		fmt.Printf("\nTitle: %s\n", title)
	}

	// Show top-level keys (H1 headings only)
	headings := doc.GetHeadings(1)
	tables := doc.GetTables()

	if len(tables) > 0 {
		// It's tabular data (array of uniform objects)
		fmt.Println("\nData Type: Table (array of uniform objects)")
		for _, table := range tables {
			fmt.Printf("  Columns: %d\n", len(table.Headers))
			fmt.Printf("  Rows: %d\n", len(table.Rows))
			fmt.Printf("  Headers: %v\n", table.Headers)

			// Show sample rows
			if len(table.Rows) > 0 {
				fmt.Println("\nSample (first 3 rows):")
				for i, row := range table.Rows {
					if i >= 3 {
						fmt.Printf("  ... and %d more rows\n", len(table.Rows)-3)
						break
					}
					fmt.Printf("  %d. %v\n", i+1, row)
				}
			}
		}
	} else if len(headings) > 0 {
		// It's structured data (object with keys)
		fmt.Println("\nData Type: Object")
		fmt.Printf("  Top-level keys: %d\n", len(headings))
		fmt.Println("\nKeys:")
		for i, h := range headings {
			if i >= 20 {
				fmt.Printf("  ... and %d more keys\n", len(headings)-20)
				break
			}
			fmt.Printf("  - %s\n", h.Text)
		}
	}

	// Show preview of readable text
	text := doc.ReadableText()
	if len(text) > 0 {
		fmt.Println("\nPreview:")
		preview := text
		if len(preview) > 500 {
			preview = preview[:500] + "\n..."
		}
		// Indent the preview
		lines := strings.Split(preview, "\n")
		for i, line := range lines {
			if i >= 15 {
				fmt.Println("  ...")
				break
			}
			fmt.Printf("  %s\n", line)
		}
	}
}

func displayResult(result interface{}) {
	switch v := result.(type) {
	case []*mq.Heading:
		fmt.Printf("Found %d headings:\n", len(v))
		for i, h := range v {
			fmt.Printf("%d. [H%d] %s\n", i+1, h.Level, h.Text)
		}

	case *mq.Section:
		fmt.Printf("Section: %s\n", v.Heading.Text)
		fmt.Printf("Lines: %d-%d\n", v.Start, v.End)
		if len(v.Children) > 0 {
			fmt.Printf("Children: %d\n", len(v.Children))
			for _, child := range v.Children {
				fmt.Printf("  - %s\n", child.Heading.Text)
			}
		}

	case []*mq.Section:
		fmt.Printf("Found %d sections:\n", len(v))
		for i, s := range v {
			fmt.Printf("%d. %s (lines %d-%d)\n", i+1, s.Heading.Text, s.Start, s.End)
		}

	case []*mq.CodeBlock:
		fmt.Printf("Found %d code blocks:\n", len(v))
		for i, cb := range v {
			lang := cb.Language
			if lang == "" {
				lang = "plain"
			}
			fmt.Printf("\n%d. [%s] %d lines\n", i+1, lang, cb.GetLines())
			fmt.Println("---")
			fmt.Println(cb.Content)
			fmt.Println("---")
		}

	case []*mq.Link:
		fmt.Printf("Found %d links:\n", len(v))
		for i, link := range v {
			fmt.Printf("%d. %s -> %s\n", i+1, link.Text, link.URL)
		}

	case []*mq.Image:
		fmt.Printf("Found %d images:\n", len(v))
		for i, img := range v {
			fmt.Printf("%d. %s: %s\n", i+1, img.AltText, img.URL)
		}

	case []*mq.Table:
		fmt.Printf("Found %d tables:\n", len(v))
		for i, table := range v {
			fmt.Printf("\n%d. Table with %d columns and %d rows\n", i+1, len(table.Headers), len(table.Rows))
			fmt.Printf("Headers: %v\n", table.Headers)
		}

	case mq.Metadata:
		fmt.Println("Metadata:")
		for key, value := range v {
			fmt.Printf("  %s: %v\n", key, value)
		}

	case string:
		fmt.Println(v)

	case []string:
		multiline := false
		for _, s := range v {
			if strings.Contains(s, "\n") {
				multiline = true
				break
			}
		}
		for i, s := range v {
			if multiline {
				fmt.Printf("%d.\n%s\n", i+1, s)
				if i < len(v)-1 {
					fmt.Println()
				}
				continue
			}
			fmt.Printf("%d. %s\n", i+1, s)
		}

	case *mq.TreeResult:
		fmt.Print(v.String())

	case *mq.DirTreeResult:
		fmt.Print(v.String())

	case *mq.SearchTreeResult:
		fmt.Print(v.String())

	case *mq.SearchResults:
		fmt.Print(v.String())

	case *mq.SearchResult:
		fmt.Println(v.RawContent())

	default:
		fmt.Printf("%+v\n", result)
	}
}
