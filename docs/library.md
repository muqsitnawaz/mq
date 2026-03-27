# mq Library Reference

This guide covers using mq as a Go library for programmatic document querying.

## Installation

```bash
go get github.com/muqsitnawaz/mq
```

## Quick Start

```go
package main

import (
    "fmt"
    mq "github.com/muqsitnawaz/mq/lib"
)

func main() {
    engine := mq.New()
    doc, err := engine.LoadDocument("README.md")
    if err != nil {
        panic(err)
    }

    // Get all H1 and H2 headings
    headings := doc.GetHeadings(1, 2)
    for _, h := range headings {
        fmt.Printf("[H%d] %s\n", h.Level, h.Text)
    }

    // Get a specific section
    if section, ok := doc.GetSection("Installation"); ok {
        fmt.Println(section.Content)
    }
}
```

## Packages

mq exposes two packages:

| Package | Import | Purpose |
|---------|--------|---------|
| `lib` | `github.com/muqsitnawaz/mq/lib` | Core document engine and types |
| `mql` | `github.com/muqsitnawaz/mq/mql` | Query language parser and executor |

Use `lib` for direct API access. Use `mql` when you need string-based queries.

## Engine

The `Engine` is the main entry point for loading documents.

```go
import mq "github.com/muqsitnawaz/mq/lib"

// Create engine with defaults
engine := mq.New()

// Load from file (auto-detects format from extension)
doc, err := engine.LoadDocument("file.md")

// Parse from bytes
doc, err := engine.ParseDocument(content, "virtual.md")
```

### Supported Formats

| Format | Extensions | Auto-detected |
|--------|------------|---------------|
| Markdown | `.md` | Yes |
| HTML | `.html`, `.htm` | Yes |
| PDF | `.pdf` | Yes |
| JSON | `.json` | Yes |
| JSONL | `.jsonl`, `.ndjson` | Yes |
| YAML | `.yaml`, `.yml` | Yes |

## Document API

All documents expose the same interface regardless of source format.

### Basic Properties

```go
doc.Path()         // File path
doc.Format()       // mq.FormatMarkdown, mq.FormatHTML, etc.
doc.Title()        // Document title
doc.Source()       // Raw source bytes
doc.ReadableText() // Plain text content (for LLM context)
```

### Headings

```go
// Get all headings
headings := doc.GetHeadings()

// Get specific levels (H1 and H2 only)
headings := doc.GetHeadings(1, 2)

// Get heading by exact text
heading, ok := doc.GetHeadingByText("Installation")
```

Each `Heading` has:
```go
type Heading struct {
    Level int    // 1-6
    Text  string // Heading text
    Line  int    // Line number in source
}
```

### Sections

Sections are hierarchical content blocks under headings.

```go
// Get section by title
section, ok := doc.GetSection("API Reference")

// Get all sections
sections := doc.GetSections()

// Get table of contents (top-level sections only)
toc := doc.GetTableOfContents()
```

Each `Section` has:
```go
type Section struct {
    Heading  *Heading   // The section's heading
    Content  string     // Raw content
    Start    int        // Start line
    End      int        // End line
    Parent   *Section   // Parent section (nil for top-level)
    Children []*Section // Child sections
}
```

### Code Blocks

```go
// Get all code blocks
blocks := doc.GetCodeBlocks()

// Get by language
goBlocks := doc.GetCodeBlocks("go")
pyBlocks := doc.GetCodeBlocks("python")

// Get multiple languages
blocks := doc.GetCodeBlocks("go", "python", "javascript")
```

Each `CodeBlock` has:
```go
type CodeBlock struct {
    Language string // Language identifier
    Content  string // Code content
    Start    int    // Start line
    End      int    // End line
}
```

### Links and Images

```go
links := doc.GetLinks()
images := doc.GetImages()
```

```go
type Link struct {
    Text string // Link text
    URL  string // Target URL
    Line int    // Line number
}

type Image struct {
    AltText string // Alt text
    URL     string // Image URL
    Line    int    // Line number
}
```

### Tables

```go
tables := doc.GetTables()
```

```go
type Table struct {
    Headers []string   // Column headers
    Rows    [][]string // Row data
    Start   int        // Start line
    End     int        // End line
}
```

### Metadata (Frontmatter)

```go
// Get all metadata
meta := doc.Metadata()

// Get specific field
val, ok := doc.GetMetadataField("author")

// Convenience methods
owner, ok := doc.GetOwner()
tags := doc.GetTags()
priority, ok := doc.GetPriority()

// Check ownership
if doc.CheckOwnership("alice") {
    // document belongs to alice
}
```

## MQL String Queries

For dynamic or user-provided queries, use the `mql` package.

```go
import "github.com/muqsitnawaz/mq/mql"

engine := mql.New()
doc, _ := engine.LoadDocument("doc.md")

// Execute query string
result, err := engine.Query(doc, `.section("API") | .code("go")`)
```

### Query Syntax

| Query | Description |
|-------|-------------|
| `.headings` | All headings |
| `.headings(2)` | H2 headings only |
| `.section("Name")` | Section by heading text |
| `.sections` | All sections |
| `.code` | All code blocks |
| `.code("go")` | Go code blocks |
| `.links` | All links |
| `.images` | All images |
| `.tables` | All tables |
| `.nth(N)` | Nth item from current results |
| `.metadata` | Frontmatter metadata |
| `.text` | Extract text content |
| `.tree` | Document structure |

### Pipes

Chain operations with `|`:

```go
// Get text from a section
`.section("API") | .text`

// Get code from a section
`.section("Examples") | .code("python")`

// Get structure of a section
`.section("Reference") | .tree`
```

### Filters

Filter results with conditions:

```go
// H2 headings only
`.headings | filter(.level == 2)`

// Sections with "auth" in title
`.sections | filter(.title contains "auth")`
```

## Utility Functions

The `lib` package includes functional utilities for working with slices.

```go
import mq "github.com/muqsitnawaz/mq/lib"

// Filter
h2s := mq.Filter(headings, func(h *mq.Heading) bool {
    return h.Level == 2
})

// Map
texts := mq.Map(headings, func(h *mq.Heading) string {
    return h.Text
})

// Find
found, ok := mq.Find(sections, func(s *mq.Section) bool {
    return strings.Contains(s.Heading.Text, "API")
})

// Chain operations
result := mq.NewChain(headings).
    Filter(func(h *mq.Heading) bool { return h.Level <= 2 }).
    Take(5).
    Result()
```

Available functions:
- `Filter`, `Map`, `FlatMap`, `Reduce`
- `Take`, `Skip`, `Unique`, `UniqueBy`
- `GroupBy`, `SortBy`, `Find`, `Any`, `All`

## Directory Operations

Query entire directories of documents.

```go
import mq "github.com/muqsitnawaz/mq/lib"

// Build directory tree
tree, err := mq.BuildDirTree("./docs")
fmt.Print(tree.String())

// With depth/limit bounds
tree, err = mq.BuildDirTreeWithOptions("./docs", mq.TreeOptions{Depth: 2, Limit: 50})
fmt.Print(tree.String())

// Search across directory
results, err := mq.SearchDir("./docs", "authentication")
fmt.Print(results.String())
```

## Example: Building a RAG Pipeline

```go
package main

import (
    "fmt"
    "strings"

    mq "github.com/muqsitnawaz/mq/lib"
)

func main() {
    engine := mq.New()

    // Load document
    doc, _ := engine.LoadDocument("docs/api.md")

    // Step 1: Get structure for context
    headings := doc.GetHeadings(1, 2)
    fmt.Println("Available sections:")
    for _, h := range headings {
        fmt.Printf("  - %s\n", h.Text)
    }

    // Step 2: User asks about authentication
    query := "authentication"

    // Step 3: Find relevant sections
    sections := doc.GetSections()
    relevant := mq.Filter(sections, func(s *mq.Section) bool {
        return strings.Contains(
            strings.ToLower(s.Heading.Text),
            strings.ToLower(query),
        )
    })

    // Step 4: Extract content for LLM context
    for _, section := range relevant {
        fmt.Printf("\n## %s\n", section.Heading.Text)
        fmt.Println(section.Content)
    }
}
```

## Error Handling

```go
doc, err := engine.LoadDocument("file.md")
if err != nil {
    // File not found, parse error, unsupported format
    log.Fatal(err)
}

section, ok := doc.GetSection("Missing")
if !ok {
    // Section doesn't exist
}
```

## Thread Safety

Document methods are safe for concurrent reads. The internal indexes are protected by a read-write mutex.

```go
// Safe to call from multiple goroutines
go func() { doc.GetHeadings() }()
go func() { doc.GetSection("API") }()
```
