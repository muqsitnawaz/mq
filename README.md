# mq - jq for Markdown

Query markdown documents structurally. Like `jq` for JSON, but for `.md` files.

```bash
# See document structure
mq README.md .tree

# Extract a section
mq README.md '.section("Installation") | .text'

# Search across all docs
mq docs/ '.search("authentication")'
```

## Installation

```bash
go install github.com/muqsitnawaz/mq@latest
```

## Usage

### Document Structure

```bash
# Full tree with code blocks, tables, etc.
mq README.md .tree

# Compact tree (headings only)
mq README.md '.tree("compact")'

# Tree of specific section
mq README.md '.section("API") | .tree'
```

Output:
```
README.md (413 lines)
├── # mq - jq for Markdown (1-413)
│   ├── ## Installation (34-39)
│   │   └── [code: bash, 1 block]
│   ├── ## Usage (40-120)
│   │   ├── ### Document Structure (42-60)
│   │   └── ### Extracting Content (61-90)
```

### Directory Overview

```bash
# See all .md files
mq docs/ .tree

# With top-level headings
mq docs/ '.tree("expand")'
```

Output:
```
docs/ (7 files, 1245 lines total)
├── API.md (234 lines, 12 sections)
├── README.md (89 lines, 5 sections)
└── guides/
    └── setup.md (156 lines, 8 sections)
```

### Search

```bash
# Search in single file
mq README.md '.search("installation")'

# Search across directory
mq docs/ '.search("OAuth")'
```

Output:
```
Found 3 matches for "OAuth":

docs/auth.md:
  ## Authentication (lines 34-89)
     "...OAuth 2.0 authentication flow..."
  ## OAuth Flow (lines 45-67)
     "#### OAuth Flow implementation..."
```

### Extracting Content

```bash
# Get section metadata
mq doc.md '.section("API")'

# Get section content (raw markdown)
mq doc.md '.section("API") | .text'

# Get code blocks by language
mq doc.md '.code("python")'
mq doc.md '.section("Examples") | .code("go")'

# Get all links
mq doc.md .links

# Get frontmatter
mq doc.md .metadata
mq doc.md .owner
mq doc.md .tags
```

## Query Language

### Selectors

| Selector | Description |
|----------|-------------|
| `.tree` | Document structure |
| `.tree("compact")` | Headings only |
| `.search("term")` | Find sections containing term |
| `.section("name")` | Section by heading |
| `.sections` | All sections |
| `.headings` | All headings |
| `.headings(2)` | H2 headings only |
| `.code` | All code blocks |
| `.code("lang")` | Code by language |
| `.links` | All links |
| `.images` | All images |
| `.tables` | All tables |
| `.metadata` | YAML frontmatter |
| `.owner` | Document owner |
| `.tags` | Document tags |

### Operations

| Operation | Description |
|-----------|-------------|
| `.text` | Extract content |
| `\| .tree` | Pipe to tree |
| `filter(.level == 2)` | Filter by condition |
| `select(.level <= 2)` | Alternative filter |

### Examples

```bash
# Filter headings
mq doc.md '.headings | filter(.level == 2) | .text'

# Get Python code from section
mq doc.md '.section("Examples") | .code("python")'

# Chain operations
mq doc.md '.section("API") | .tree'
```

## mq vs qmd

[qmd](https://github.com/tobi/qmd) is semantic search. `mq` is structural extraction. They complement each other.

| | **mq** | **qmd** |
|--|--------|---------|
| **Purpose** | Extract section X from file | Find files about topic Y |
| **Query** | `.section("Auth")` | `"how to authenticate"` |
| **Output** | Actual content | File paths + scores |
| **Deps** | Single binary | Bun, SQLite, 1.6GB models |

```bash
# Find files with qmd
qmd query "authentication"
# → docs/auth.md (0.92)

# Extract content with mq
mq docs/auth.md '.section("OAuth") | .text'
```

## Benchmark: Agent Performance

We benchmarked AI agent performance answering questions about the [LangChain](https://github.com/langchain-ai/langchain) monorepo (50+ markdown files). Agents were given identical questions and asked to find answers by reading documentation.

| Question | Mode | Input Tokens | Latency | Token Savings |
|----------|------|--------------|---------|---------------|
| Commit standards | without mq | 192,667 | 24s | - |
| | with mq | 140,822 | 28s | 27% |
| Package installation | without mq | 383,244 | 39s | - |
| | with mq | 190,591 | 30s | 50% |
| Testing requirements | without mq | 236,929 | 25s | - |
| | with mq | 146,334 | 33s | 38% |
| CLI integration guide | without mq | 337,225 | 31s | - |
| | with mq | 516,897 | 85s | -53% |
| Documentation standards | without mq | 187,768 | 21s | - |
| | with mq | 92,334 | 21s | 51% |

### Summary

| Metric | Without mq | With mq | Improvement |
|--------|------------|---------|-------------|
| Total input tokens | 1,337,833 | 1,086,978 | **19% fewer** |
| Total latency | 140s | 197s | 41% slower |
| Excluding outlier (q4) | | | |
| - Input tokens | 1,000,608 | 570,081 | **43% fewer** |
| - Latency | 109s | 112s | ~same |

**Key findings:**
- 4 of 5 questions showed 27-51% token reduction
- Latency is similar when mq is used efficiently (q2 was actually 23% faster)
- One outlier (q4) where the agent made excessive mq queries, hurting both metrics
- Token savings directly translate to cost savings on API calls

The "with mq" agent uses `.tree` to see document structure, `.search()` to find relevant sections, and `.section() | .text` to extract only what's needed. The "without mq" agent reads entire files.

Run the benchmark yourself:
```bash
./scripts/bench.sh
```

## Library Usage

```go
import (
    "github.com/muqsitnawaz/mq/mql"
)

engine := mql.New()
doc, _ := engine.LoadDocument("README.md")

// Query
result, _ := engine.Query(doc, `.section("API") | .code("go")`)

// Direct API
section, _ := doc.GetSection("API")
code := section.GetCodeBlocks("go")
```

## License

MIT
