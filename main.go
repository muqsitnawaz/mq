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
		fmt.Println("Usage: mq <markdown-file> [query]")
		fmt.Println("\nBasic Examples:")
		fmt.Println("  mq README.md                                    # Show document info")
		fmt.Println("  mq README.md '.headings'                        # Get all headings")
		fmt.Println("  mq README.md '.headings(2)'                    # Get H2 headings only")
		fmt.Println("  mq README.md '.code(\"python\")'                 # Get Python code blocks")
		fmt.Println("  mq README.md '.section(\"Installation\")'        # Get Installation section")
		fmt.Println("\nAdvanced Examples:")
		fmt.Println("  mq README.md '.section(\"API\").code(\"curl\")'    # Get curl examples in API section")
		fmt.Println("  mq README.md '.code | select(.language == \"python\" and .lines > 10)'  # Filter code blocks")
		fmt.Println("  mq README.md '.headings | select(.level <= 2)'  # Filter by heading level")
		fmt.Println("  mq README.md '.context(\"authentication flow\")'  # Find relevant sections")
		fmt.Println("\nTransformation Examples:")
		fmt.Println("  mq README.md '.headings | map(.text)'           # Extract heading text")
		fmt.Println("  mq README.md '.code | map({lang: .language})'   # Transform to JSON")
		fmt.Println("  mq README.md '.section(\"API\") | .summary(100)' # Summarize section")
		fmt.Println("\nStructural Examples:")
		fmt.Println("  mq README.md '.toc'                             # Get table of contents")
		fmt.Println("  mq README.md '.depth(2)'                        # Get elements at depth 2")
		fmt.Println("  mq README.md '.section(\"API\").children'        # Get subsections")
		fmt.Println("\nOutput Format Examples:")
		fmt.Println("  mq README.md '.section(\"Examples\") | .json'    # Output as JSON")
		fmt.Println("  mq README.md '.section(\"API\") | .yaml'         # Output as YAML")
		fmt.Println("  mq README.md '.code | .chunk(4000)'            # Split into chunks")
		os.Exit(1)
	}

	// Load the markdown file
	engine := mql.New()
	doc, err := engine.LoadDocument(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to load document: %v", err)
	}

	// If no query provided, show document info
	if len(os.Args) < 3 {
		showDocumentInfo(doc)
		return
	}

	// Execute the query
	query := os.Args[2]
	result, err := engine.Query(doc, query)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	// Display results
	displayResult(result)
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

	default:
		fmt.Printf("Result type: %T\n", result)
		fmt.Printf("Result: %+v\n", result)
	}
}
