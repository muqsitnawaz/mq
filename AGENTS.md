# Agent Instructions for mq

Instructions for AI coding agents contributing to this repository.

## Architecture

```
mq/
├── main.go              # CLI entry point, argument parsing, display
├── lib/                 # Core markdown processing
│   ├── parser.go        # Goldmark-based markdown parser
│   ├── document.go      # Document model with pre-computed indexes
│   ├── types.go         # Core types: Heading, Section, CodeBlock, etc.
│   ├── tree.go          # Tree rendering, directory tree, search
│   └── engine.go        # Fluent query builder API
└── mql/                 # Query language
    ├── lexer.go         # Tokenizer
    ├── parser.go        # Recursive descent parser
    ├── ast.go           # AST node definitions
    ├── compiler.go      # Compiles AST to executable plans
    └── engine.go        # MQL engine wrapper
```

## Key Patterns

### Adding a New Selector

1. Add case in `mql/compiler.go` in `VisitSelector()`:
```go
case "myselector":
    // Get arguments if any
    if len(args) > 0 {
        arg := args[0].(string)
    }
    // Return result from document
    return doc.MyMethod(), nil
```

2. Add method in `lib/document.go` if needed:
```go
func (d *Document) MyMethod() []*MyType {
    // Implementation
}
```

3. Add display case in `main.go` in `displayResult()`:
```go
case *mq.MyType:
    fmt.Print(v.String())
```

### Adding Directory Mode Support

Directory operations are handled in `main.go` `handleDirectory()`:
```go
if strings.HasPrefix(query, `.myquery("`) {
    // Parse argument
    // Call lib function
    // Print result
}
```

### Document Indexing

Documents pre-compute indexes at parse time (`lib/parser.go` `buildIndexes()`):
- `headingIndex` - map heading text to Heading
- `headingsByLevel` - map level to Heading slice
- `sectionIndex` - map title to Section
- `codeByLang` - map language to CodeBlock slice

Sections maintain parent/child hierarchy and store source reference for text extraction.

### Line Number Calculation

Line numbers are calculated from byte offsets using `computeLineStarts()` and `getLineNumber()` in `lib/parser.go`. This enables accurate line ranges for sections.

## Testing

Test data lives in `testdata/`:
- `plain.md` - No structure
- `deep-nested.md` - Deep heading hierarchy
- `code-heavy.md` - Many code blocks
- `links-images.md` - Links and images
- `tables-lists.md` - Tables and lists
- `frontmatter.md` - YAML frontmatter
- `outdated.md` - Deprecated content patterns

Run tests:
```bash
go test ./...
```

Manual testing:
```bash
go build . && ./mq testdata/code-heavy.md .tree
```

## Code Style

- No backwards compatibility hacks
- No emojis in code or output
- No tests unless explicitly requested
- Prefer refactoring over adding new types
- Use existing patterns from the codebase

## Common Tasks

### Add a new element type (e.g., blockquotes)

1. Add type in `lib/types.go`
2. Add extraction in `lib/parser.go` `buildIndexes()`
3. Add index fields in `lib/document.go`
4. Add selector in `mql/compiler.go`
5. Add display in `main.go`

### Modify tree output

Edit `lib/tree.go`:
- `TreeNode` struct for node data
- `renderNode()` for display format
- `buildSectionTree()` for what gets included

### Add search capability

Search is in `lib/tree.go`:
- `Search()` method on Document
- `SearchDir()` for directory search
- `SearchResults` type for output

## Dependencies

- [goldmark](https://github.com/yuin/goldmark) - Markdown parser
- [goldmark-meta](https://github.com/yuin/goldmark-meta) - YAML frontmatter

No external dependencies for the CLI binary.
