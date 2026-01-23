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
	repo           = "muqsitnawaz/mq"
	releaseAPIURL  = "https://api.github.com/repos/" + repo + "/releases/latest"
	yellow         = "\033[33m"
	reset          = "\033[0m"
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

	// Load the markdown file
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
	fmt.Printf("mq %s - Query markdown files efficiently\n\n", version)
	fmt.Println("Usage: mq <file|directory> [query]")
	fmt.Println("\nExamples:")
	fmt.Println("  mq README.md .tree                    # Document structure")
	fmt.Println("  mq README.md '.tree(\"full\")'          # Structure + previews")
	fmt.Println("  mq README.md '.section(\"API\") | .text' # Extract section")
	fmt.Println("  mq README.md '.search(\"auth\")'        # Search content")
	fmt.Println("  mq docs/ '.tree(\"full\")'              # Directory overview")
	fmt.Println("\nCommands:")
	fmt.Println("  upgrade         Upgrade to latest version")
	fmt.Println("\nFlags:")
	fmt.Println("  -h, --help      Show this help")
	fmt.Println("  -v, --version   Show version")
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

func handleDirectory(path string, query string) {
	// Directory mode supports .tree and .search queries
	if query == "" {
		query = ".tree"
	}

	// Handle tree queries
	if query == ".tree" || query == `.tree("compact")` {
		result, err := mq.BuildDirTree(path, mq.TreeModeDefault)
		if err != nil {
			log.Fatalf("Failed to build directory tree: %v", err)
		}
		fmt.Print(result.String())
		return
	}

	if query == `.tree("expand")` {
		// expand is now an alias for preview (shows section headings)
		result, err := mq.BuildDirTree(path, mq.TreeModePreview)
		if err != nil {
			log.Fatalf("Failed to build directory tree: %v", err)
		}
		fmt.Print(result.String())
		return
	}

	if query == `.tree("preview")` {
		result, err := mq.BuildDirTree(path, mq.TreeModePreview)
		if err != nil {
			log.Fatalf("Failed to build directory tree: %v", err)
		}
		fmt.Print(result.String())
		return
	}

	if query == `.tree("full")` {
		result, err := mq.BuildDirTree(path, mq.TreeModeFull)
		if err != nil {
			log.Fatalf("Failed to build directory tree: %v", err)
		}
		fmt.Print(result.String())
		return
	}

	// Handle search queries: .search("term")
	if strings.HasPrefix(query, `.search("`) && strings.HasSuffix(query, `")`) {
		searchTerm := query[9 : len(query)-2]
		result, err := mq.SearchDir(path, searchTerm)
		if err != nil {
			log.Fatalf("Search failed: %v", err)
		}
		fmt.Print(result.String())
		return
	}

	log.Fatalf("Directory mode supports: .tree, .tree(\"expand\"), .tree(\"preview\"), .tree(\"full\"), .search(\"term\")")
}

func showDocumentInfo(doc *mq.Document) {
	fmt.Printf("Document: %s\n", doc.Path())
	fmt.Println("=" + strings.Repeat("=", len(doc.Path())+9))

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

	// Show structure
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
