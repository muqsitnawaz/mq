package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/muqsitnawaz/mq/mql"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: mq <markdown-file|directory> [query]")
		fmt.Println("\nBasic Examples:")
		fmt.Println("  mq README.md                                    # Show document info")
		fmt.Println("  mq README.md '.headings'                        # Get all headings")
		fmt.Println("  mq README.md '.headings(2)'                    # Get H2 headings only")
		fmt.Println("  mq README.md '.code(\"python\")'                 # Get Python code blocks")
		fmt.Println("  mq README.md '.section(\"Installation\")'        # Get Installation section")
		fmt.Println("\nTree Examples:")
		fmt.Println("  mq README.md .tree                              # Show document structure")
		fmt.Println("  mq README.md '.tree(\"compact\")'                # Headings only")
		fmt.Println("  mq README.md '.tree(\"preview\")'                # Headings + first few words")
		fmt.Println("  mq docs/ .tree                                  # Show all .md files in directory")
		fmt.Println("  mq docs/ '.tree(\"expand\")'                     # Show files with section names")
		fmt.Println("  mq docs/ '.tree(\"full\")'                       # Section names + previews")
		fmt.Println("\nAdvanced Examples:")
		fmt.Println("  mq README.md '.section(\"API\") | .code(\"curl\")'  # Get curl examples in API section")
		fmt.Println("  mq README.md '.section(\"API\") | .text'          # Get section content")
		fmt.Println("  mq README.md '.section(\"API\") | .tree'          # Tree of specific section")
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
