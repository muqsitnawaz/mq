# mq - jq for Markdown and Context files

A command-line tool that does one thing well: query markdown documents. Like `jq` for JSON, but for markdown files.

Perfect for AI agents working with context files like [AGENTS.md](https://agents.md) to extract only the information they need.

## Quick Start

```bash
# Filter headings by level and extract their text
mq README.md '.headings | filter(.level == 2) | .text'

# Get Python code from the API section
mq README.md '.section("API Reference") | .code("python")'

# Extract text content from all code blocks
mq README.md '.code | .text'

# Select headings at specific levels
mq README.md '.headings | select(.level <= 2)'

# Get all sections and extract their heading text
mq README.md '.sections | .heading | .text'

# Query multiple language code blocks
mq README.md '.code("go", "python")'

# Get document metadata and ownership
mq README.md '.metadata'
mq README.md '.owner'
mq README.md '.tags'
```

## Installation

```bash
go install github.com/muqsitnawaz/mq@latest
```

## Examples

### Filter and Extract

```bash
# Get text from H2 headings only
$ mq README.md '.headings | filter(.level == 2) | .text'
["Installation", "Quick Start", "Examples", "Query Language", "Library Usage", "Architecture", "License"]

# Select headings at specific levels
$ mq README.md '.headings | select(.level <= 2)'
Found 11 headings:
1. [H1] mq - jq for Markdown
2. [H2] Installation
3. [H2] Quick Start
...
```

### Advanced Section Queries

```bash
# Get all sections and their heading text
$ mq README.md '.sections | .heading | .text'
["mq - jq for Markdown", "Installation", "Quick Start", "Examples", ...]

# Extract Python code from API section
$ mq docs/api.md '.section("API Reference") | .code("python")'
Found 3 Python blocks in section

# Get code from section and extract text
$ mq README.md '.section("Examples") | .code | .text'
[
  "npm install mq",
  "import mq from 'mq'...",
  "const result = mq.query(doc, '.headings')"
]
```

### Multi-Language Code Extraction

```bash
# Get both Go and Python code blocks
$ mq tutorial.md '.code("go", "python")'
Found 8 code blocks (5 go, 3 python)

# Extract all code as text
$ mq README.md '.code | .text'
Returns array of all code block contents as strings
```

### Metadata and Frontmatter

```bash
# Get document owner
$ mq AGENTS.md '.owner'
"alice"

# Get document tags
$ mq doc.md '.tags'
["api", "documentation", "v2"]

# Get priority level
$ mq task.md '.priority'
"high"

# Get all sections owned by alice
$ mq doc.md '.sections' | jq 'select(.owner == "alice")'
```

### Complex Pipelines

```bash
# Filter, extract, and transform in one query
$ mq docs.md '.headings | filter(.level == 2) | .text' | jq 'map(ascii_downcase)'

# Get Python functions from API docs
$ mq api.md '.section("Endpoints") | .code("python") | .text' | grep "def "

# Extract all links from code sections
$ mq tutorial.md '.sections | select(.heading.text | contains("Code")) | .links'
```

## Why not grep?

`grep` searches text. `mq` understands markdown structure.

- **grep**: `grep -A 5 "## Installation"` - returns text lines
- **mq**: `mq doc.md '.section("Installation")'` - returns the full section

`mq` lets you:
- Query specific markdown elements (headings, code blocks, links)
- Navigate document structure (sections, subsections)
- Extract code by language, headings by level
- Work with frontmatter metadata

## Query Language

`mq` uses a simple, powerful query language inspired by `jq`:

### Selectors

| Selector | Description | Example |
|----------|-------------|---------|
| `.headings` | All headings | `mq doc.md '.headings'` |
| `.headings(n)` | Headings of level n | `mq doc.md '.headings(2)'` |
| `.sections` | All sections | `mq doc.md '.sections'` |
| `.section("name")` | Section by heading text | `mq doc.md '.section("API")'` |
| `.code` | All code blocks | `mq doc.md '.code'` |
| `.code("lang")` | Code blocks by language | `mq doc.md '.code("python")'` |
| `.code("l1", "l2")` | Multiple languages | `mq doc.md '.code("go", "python")'` |
| `.links` | All markdown links | `mq doc.md '.links'` |
| `.images` | All images | `mq doc.md '.images'` |
| `.tables` | All tables | `mq doc.md '.tables'` |
| `.lists` | All lists | `mq doc.md '.lists'` |
| `.metadata` | YAML frontmatter | `mq doc.md '.metadata'` |
| `.owner` | Document owner | `mq doc.md '.owner'` |
| `.tags` | Document tags | `mq doc.md '.tags'` |
| `.priority` | Document priority | `mq doc.md '.priority'` |

### Operations

| Operation | Description | Example |
|-----------|-------------|---------|
| `select()` | Filter based on condition | `.headings \| select(.level == 2)` |
| `filter()` | Alternative filter syntax | `.headings \| filter(.level <= 2)` |
| `.text` | Extract text content | `.headings \| .text` |
| `.heading` | Get heading from sections | `.sections \| .heading` |
| `\|` | Pipe operations | `.section("API") \| .code` |

### Advanced Queries

Combine selectors and operations for powerful queries:

```bash
# Filter headings and extract text
mq doc.md '.headings | filter(.level == 2) | .text'

# Get code from section with language filter
mq doc.md '.section("Examples") | .code("python") | .text'

# Extract heading text from all sections
mq doc.md '.sections | .heading | .text'

# Chain multiple filters
mq doc.md '.headings | select(.level <= 3) | filter(.text | contains("API"))'
```

### Frontmatter Support

Query documents with YAML frontmatter:

```markdown
---
owner: alice
tags: [api, documentation]
priority: high
---
# Document Title
```

```bash
# Get document metadata
mq doc.md '.metadata'

# Get specific field
mq doc.md '.owner'

# Filter by tags
mq doc.md '.tags'
```

## Library Usage

`mq` can also be used as a Go library for programmatic document processing:

### Installation

```bash
go get github.com/muqsitnawaz/mq
```

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/muqsitnawaz/mq/lib"
    "github.com/muqsitnawaz/mq/mql"
)

func main() {
    // Create MQL engine
    engine := mql.New()

    // Load document
    doc, err := engine.LoadDocument("README.md")
    if err != nil {
        panic(err)
    }

    // Execute advanced queries
    result, err := engine.Query(doc, `.headings | filter(.level == 2) | .text`)
    if err != nil {
        panic(err)
    }
    fmt.Println(result) // ["Installation", "Quick Start", ...]

    // Query with multiple operations
    result, err = engine.Query(doc, `.section("API") | .code("python") | .text`)

    // Use direct API for complex operations
    section, _ := doc.GetSection("API Reference")
    pythonCode := section.GetCodeBlocks("python")
    for _, block := range pythonCode {
        fmt.Printf("Python code: %d lines\n", len(block.Content))
    }
}
```

### Fluent API

```go
// Chain operations fluently
result, err := engine.From(doc).
    Section("API Reference").
    Code("python").
    Take(5).
    Execute()

// With filters
result, err := engine.From(doc).
    Headings().
    Filter(func(h *mq.Heading) bool {
        return h.Level <= 2
    }).
    Execute()
```

### Direct Document API

```go
// Load and parse document
engine := mq.New()
doc, err := engine.LoadDocument("doc.md")

// Direct access methods
headings := doc.GetHeadings()           // All headings
section, _ := doc.GetSection("Intro")   // Specific section
codeBlocks := doc.GetCodeBlocks("go")   // Go code blocks
links := doc.GetLinks()                 // All links
tables := doc.GetTables()               // All tables

// Metadata access
if owner, ok := doc.GetOwner(); ok {
    fmt.Printf("Owner: %s\n", owner)
}
tags := doc.GetTags()
```

## Architecture

`mq` is built with performance and extensibility in mind:

### Core Components

- **`lib/`** - Core markdown processing engine
  - Document parsing with [goldmark](https://github.com/yuin/goldmark)
  - Pre-computed indexes for O(1) lookups
  - Type-safe direct API

- **`mql/`** - Query language implementation
  - Lexer and parser for query syntax
  - AST-based query compilation
  - Query optimization and caching

### Features

- **Pre-computed indexes**: Documents are indexed at parse time for fast queries
- **Lazy evaluation**: Results are computed on-demand
- **Zero-allocation design**: Reuses buffers where possible
- **Extensible**: Support for custom goldmark extensions
- **Type-safe**: Strongly typed API with compile-time checks

## Advanced Features

### Ownership-Based Access Control

Control document access based on frontmatter metadata:

```go
// Check document ownership
if doc.CheckOwnership("alice") {
    // Process document
}

// Query with ownership filter
result := engine.From(doc).
    WhereOwner("alice").
    Sections().
    Execute()
```

### Custom Extensions

Add custom goldmark extensions:

```go
parser := mq.NewParser(
    mq.WithExtensions(myCustomExtension),
)
engine := mq.New(mq.WithParser(parser))
```

## Performance

- **Fast parsing**: ~50ms for 10MB documents
- **O(1) lookups**: Pre-computed indexes for sections and headings
- **Memory efficient**: Streaming parser with minimal allocations
- **Query caching**: Compiled queries are cached for reuse

## Development

```bash
# Run tests
go test ./...

# Build CLI
go build -o mq .

# Install locally
go install .
```

## Acknowledgments

Built on [goldmark](https://github.com/yuin/goldmark)'s excellent extensible AST. Inspired by tools like [mdq](https://docs.rs/mdq/latest/mdq/) in the Rust ecosystem.

## License

MIT