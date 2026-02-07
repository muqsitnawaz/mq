package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
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
	yellow        = "\033[33m"
	reset         = "\033[0m"
)

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
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

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	path := os.Args[1]
	query := ""
	if len(os.Args) >= 3 {
		query = os.Args[2]
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
	doc, err := engine.LoadDocument(path)
	if err != nil {
		log.Fatalf("Failed to load document: %v", err)
	}

	// If no query provided, show document info
	if query == "" {
		showDocumentInfo(doc)
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

func printUsage() {
	fmt.Printf("mq %s - Query structured documents without reading entire contents\n\n", version)
	fmt.Println("Usage: mq <file|directory> [query]")
	fmt.Println("\nWorkflow:")
	fmt.Println("  1. See structure:  mq <path> '.tree(\"full\")'")
	fmt.Println("  2. Extract content: mq <file> '.section(\"Name\") | .text'")
	fmt.Println("")
	fmt.Println("  Scope matters: tree a specific file or subdir, not the entire repo.")
	fmt.Println("")
	fmt.Println("Tree modes:")
	fmt.Println("  .tree              Structure with line ranges")
	fmt.Println("  .tree(\"preview\")   Structure + content preview")
	fmt.Println("  .tree(\"full\")      Structure + previews (best for directories)")
	fmt.Println("")
	fmt.Println("Selectors:")
	fmt.Println("  .section(\"Name\")   Get section by heading")
	fmt.Println("  .search(\"term\")    Find sections containing term")
	fmt.Println("  .code(\"lang\")      Get code blocks by language")
	fmt.Println("  .headings          Get all headings")
	fmt.Println("  .links             Get all links")
	fmt.Println("")
	fmt.Println("Pipes:")
	fmt.Println("  | .text            Extract raw content")
	fmt.Println("  | .tree            Show structure of selection")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  mq docs/ '.tree(\"full\")'                    # See all docs structure")
	fmt.Println("  mq README.md '.section(\"Install\") | .text'  # Get install instructions")
	fmt.Println("  mq src/ '.search(\"auth\")'                   # Find auth-related sections")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  upgrade            Upgrade to latest version")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  -h, --help         Show this help")
	fmt.Println("  -v, --version      Show version")
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
		fmt.Fprintf(os.Stderr, "%sA new version is available: %s (current: %s). Run 'mq upgrade' to update.%s\n\n", yellow, release.TagName, version, reset)
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

	fmt.Printf("Upgraded to %s\n", release.TagName)
	return nil
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
	// Directory mode supports .tree and .search queries
	if query == "" {
		query = ".tree"
	}

	method, arg, ok := parseMethodCall(query)
	if !ok {
		log.Fatalf("Invalid query format. Supported: .tree, .tree(\"mode\"), .search(\"term\")")
	}

	switch method {
	case "tree":
		mode := mq.TreeModeDefault
		switch arg {
		case "", "compact":
			mode = mq.TreeModeDefault
		case "expand", "preview":
			mode = mq.TreeModePreview
		case "full":
			mode = mq.TreeModeFull
		default:
			log.Fatalf("Unknown tree mode: %q. Use: compact, preview, full", arg)
		}
		result, err := mql.BuildDirTree(path, mode)
		if err != nil {
			log.Fatalf("Failed to build directory tree: %v", err)
		}
		fmt.Print(result.String())

	case "search":
		if arg == "" {
			log.Fatalf("Search requires a term: .search(\"term\")")
		}
		result, err := mql.SearchDir(path, arg)
		if err != nil {
			log.Fatalf("Search failed: %v", err)
		}
		fmt.Print(result.String())

	default:
		log.Fatalf("Unknown method: .%s. Supported: .tree, .search", method)
	}
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
		for i, s := range v {
			fmt.Printf("%d. %s\n", i+1, s)
		}

	case *mq.TreeResult:
		fmt.Print(v.String())

	case *mq.SearchResults:
		fmt.Print(v.String())

	default:
		fmt.Printf("Result type: %T\n", result)
		fmt.Printf("Result: %+v\n", result)
	}
}
