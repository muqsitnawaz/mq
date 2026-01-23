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

# Tree with preview (first few words of each section)
mq README.md '.tree("preview")'

# Tree of specific section
mq README.md '.section("API") | .tree'
```

Output for `.tree("preview")`:
```
README.md (413 lines)
├── # mq - jq for Markdown (1-413)
│        "Query markdown documents structurally..."
│   ├── ## Installation (34-39)
│   │        "go install github.com/muqsitnawaz/mq@latest"
│   ├── ## Usage (40-120)
│   │        "### Document Structure"
```

### Directory Overview

```bash
# See all .md files
mq docs/ .tree

# With section names
mq docs/ '.tree("expand")'

# With section names + previews (best for AI agents)
mq docs/ '.tree("full")'
```

Output for `.tree("full")`:
```
docs/ (7 files, 42 sections)
├── API.md (234 lines, 12 sections)
│   ├── # API Reference
│   │        "Complete reference for all REST endpoints..."
│   ├── ## Authentication
│   │        "All requests require Bearer token..."
│   └── ## Endpoints
│            "Base URL is https://api.example.com/v1..."
└── guides/
    └── setup.md (156 lines, 8 sections)
        ├── # Setup Guide
        │        "This guide walks you through installation..."
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
| `.tree("preview")` | Headings + first few words |
| `.tree("full")` | For directories: section names + previews |
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

## Agentic Indexing

Traditional document retrieval requires pre-building indexes:

```
Traditional: Documents → Build Index → Store in DB → Query Index → Results
mq:          Documents → Query → Structure IS the Index (in agent context)
```

When an LLM agent uses mq, the `.tree` output **is** the index. The agent holds it in context and reasons over it directly. No pre-computation, no vector database, no storage layer.

**Why this works**: The LLM already has semantic understanding. It can try synonyms, rephrase queries, and decide which sections to explore. You don't need a separate embedding layer when the agent itself is the reasoning engine.

```bash
# Agent gets the index on-demand
mq docs/ '.tree("full")'
# → docs/ (12 files, 2847 lines)
# → ├── auth.md (234 lines, 8 sections)
# → │   ├── ## Authentication
# → │   │        "OAuth 2.0 and API key authentication..."
# → │   └── ## OAuth Flow
# → │            "Step-by-step OAuth implementation..."

# Agent extracts what it needs
mq docs/auth.md '.section("OAuth Flow") | .text'
```

The agent's context window becomes the working index. Each query refines what the agent knows.

## Comparison: mq vs qmd vs PageIndex

Three approaches to markdown retrieval for AI agents:

| | **mq** | **[qmd](https://github.com/tobi/qmd)** | **[PageIndex](https://github.com/VectifyAI/PageIndex)** |
|--|--------|---------|---------------|
| **Technique** | AST parsing + query language | Vector embeddings + BM25 | Tree structure + LLM reasoning |
| **Index** | On-demand (agent context) | Pre-built (SQLite + vectors) | Pre-built (JSON tree) |
| **Retrieval** | Deterministic query | Similarity search | LLM traverses tree |
| **Dependencies** | Single Go binary | 3GB models, Bun, SQLite | Python, OpenAI API |
| **Cost per query** | Zero | Local compute | LLM API calls |
| **Output** | Exact content | Paths + scores | Node references |

**Core insight**: Markdown has explicit structure (`#` headings). All three tools parse the same headers - they differ in navigation:

- **qmd**: Converts text → vectors, searches by mathematical similarity
- **PageIndex**: Builds tree, uses LLM to reason which branch to follow
- **mq**: Exposes tree directly, lets the agent reason over it

When the consumer is an LLM agent, adding another embedding layer (qmd) or another LLM layer (PageIndex) is redundant. The agent navigates structure directly.

## Benchmark: Agent Performance

We benchmarked AI agent performance answering questions about the [LangChain](https://github.com/langchain-ai/langchain) monorepo (50+ markdown files). Agents were given identical questions and asked to find answers by reading documentation.

| Question | Mode | Input Tokens | Latency | Token Savings |
|----------|------|--------------|---------|---------------|
| Commit standards | without mq | 147,070 | 23s | - |
| | with mq | 166,501 | 25s | -13% |
| Package installation | without mq | 412,668 | 37s | - |
| | with mq | 108,225 | 19s | **74%** |
| Testing requirements | without mq | 244,271 | 24s | - |
| | with mq | 168,318 | 27s | 31% |
| CLI integration guide | without mq | 407,773 | 36s | - |
| | with mq | 545,708 | 56s | -34% |
| Documentation standards | without mq | 141,917 | 19s | - |
| | with mq | 108,618 | 22s | 23% |

### Summary

| Metric | Without mq | With mq | Improvement |
|--------|------------|---------|-------------|
| Total input tokens | 1,353,699 | 1,097,370 | **19% fewer** |
| Excluding outliers | 945,926 | 551,662 | **42% fewer** |
| Best case (q2) | 412,668 | 108,225 | **74% fewer** |

**Key findings:**
- Best case showed **74% token reduction** with 49% faster response
- 3 of 5 questions showed 23-74% token reduction
- When mq is used efficiently, it's both cheaper AND faster
- Token savings directly translate to cost savings on API calls

The "with mq" agent uses `.tree("full")` to see document structure with previews, then extracts specific sections with `.section() | .text`. The "without mq" agent reads entire files.

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
