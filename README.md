# mq - Agentic Querying for Structured Documents

[![CI](https://github.com/muqsitnawaz/mq/actions/workflows/ci.yml/badge.svg)](https://github.com/muqsitnawaz/mq/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/muqsitnawaz/mq)](https://github.com/muqsitnawaz/mq/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/muqsitnawaz/mq)](https://goreportcard.com/report/github.com/muqsitnawaz/mq)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

AI agents waste tokens reading entire files. mq lets them query structure first, then extract only what they need. The agent's context window becomes the working index.

**Results:**
- **83% fewer tokens** for markdown when scoped correctly
- **50x more PDFs** searchable (800 vs 16 in 200k context) via structure-first approach

**The philosophy**: Don't outsource reasoning to embeddings and rerankers. Expose structure, let the agent reason.

[Install](#installation) | [Agent Skill](#agent-skill) | [Usage](#usage) | [Query Language](#query-language)

## Supported Formats

| Format | Extensions | Structure Extraction |
|--------|------------|---------------------|
| Markdown | `.md` | Headings, sections, code blocks, links, tables |
| HTML | `.html`, `.htm` | Headings, readable content (Readability algorithm) |
| PDF | `.pdf` | Headings (font-size inference), tables, text |
| JSON | `.json` | Top-level keys as headings, nested structure |
| JSONL | `.jsonl`, `.ndjson` | Uniform objects as tables, mixed as items |
| YAML | `.yaml`, `.yml` | Keys as headings, nested structure |

### Directory Tree Labels

When browsing directories, mq uses format-aware labels:

```bash
$ mq project/ .tree
project/ (5 files, 14 lines total)
├── config.json (12 lines, 3 keys)
├── config.yaml (15 lines, 4 keys)
├── README.md (80 lines, 5 sections)
├── events.jsonl (100 lines, 98 records)
└── index.html (45 lines, 3 sections)
```

Expanded trees show format-specific heading labels:

```bash
$ mq project/ '.tree("expand")'
project/ (5 files)
├── config.json (12 lines, 3 keys)
│   ├── key name
│   └── key database
├── README.md (80 lines, 5 sections)
│   ├── # Overview
│   └── ## Install
├── events.jsonl (100 lines, 98 records)
└── index.html (45 lines, 3 sections)
    └── H1 Welcome
```

| Format | Count Label | Heading Label |
|--------|-------------|---------------|
| Markdown | sections | `# Heading` |
| HTML/PDF | sections | `H1 Heading` |
| JSON/YAML | keys | `key name` / `subkey field` |
| JSONL | records | `field name` |

### Works With

<p>
  <img src="assets/claude.png" alt="Claude" height="40">
  <img src="assets/cursor.png" alt="Cursor" height="40">
  <img src="assets/opencode.png" alt="OpenCode" height="40">
  <img src="assets/chatgpt.png" alt="ChatGPT" height="40">
  <img src="assets/gemini.png" alt="Gemini" height="40">
  <img src="assets/vscode.png" alt="VS Code" height="40">
</p>

Any AI agent or coding assistant that can execute shell commands.

### Why mq?

| | mq | [qmd](https://github.com/tobi/qmd) | [PageIndex](https://github.com/VectifyAI/PageIndex) |
|--|:--:|:--:|:--:|
| Zero external API calls | **Yes** | No | No |
| No pre-built index | **Yes** | No | No |
| Single binary, no deps | **Yes** | No | No |
| Deterministic output | **Yes** | No | No |

<details>
<summary>See full comparison</summary>

- **vs [qmd](https://github.com/tobi/qmd)**: No 3GB models to download, no SQLite database, no embedding step
- **vs [PageIndex](https://github.com/VectifyAI/PageIndex)**: No OpenAI API costs, no pre-processing, works offline
- **vs both**: Agent reasons in its own context - no external computation
</details>

```bash
# Markdown - structure and extraction
mq docs/ '.tree("full")'
mq docs/auth.md '.section("OAuth Flow") | .text'

# HTML - readable content from web pages
mq page.html '.headings'
mq page.html '.text'

# PDF - extract structure from papers
mq paper.pdf '.headings'
mq paper.pdf '.tables'

# JSON/YAML - query data files
mq config.json '.headings'      # Top-level keys
mq data.yaml '.text'            # Readable representation

# JSONL - query ML datasets and logs
mq users.jsonl '.tables'        # Uniform objects as tables
```

## Why This Works

Traditional retrieval adds external API hops. mq keeps everything in the agent's context:

```
┌─────────────────────────────────────────────────────────────────────────┐
│  Traditional RAG                                                        │
│                                                                         │
│  Agent → Embedding API → Vector DB → Reranker API → back to Agent       │
│            (hop 1)         (hop 2)      (hop 3)        (hop 4)          │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│  mq                                                                     │
│                                                                         │
│  Agent ←→ mq (local binary)                                             │
│    ↓                                                                    │
│  Agent reasons over structure in its own context                        │
│                                                                         │
│  No external APIs. No round trips. One context.                         │
└─────────────────────────────────────────────────────────────────────────┘
```

mq is an **interface**, not an answer engine. It extracts structure into the agent's context, where the agent can reason over it directly.

**The insight**: Agents like Claude Code and Codex are already LLMs with reasoning capability. Adding embedding APIs and rerankers just adds latency and cost. The agent can find what it needs - it just needs to **see** the structure.

## Benchmark: Up to 83% Token Reduction

We benchmarked agents answering questions about the [LangChain](https://github.com/langchain-ai/langchain) monorepo (50+ markdown files):

| Metric | Without mq | With mq | Improvement |
|--------|------------|---------|-------------|
| Best case (scoped) | 147,070 | 24,000* | **83% fewer** |
| Typical case | 412,668 | 108,225 | **74% fewer** |
| Naive (tree entire repo) | 147,070 | 166,501 | -13% (worse) |

*When agent narrows down to specific file before running `.tree("full")`

### The Scoping Insight

Running `.tree("full")` on an entire repo is expensive. For 50 files, the tree output alone is ~22,000 characters before extracting any content.

```
Naive:   .tree("full") on /repo           → 22K chars just for tree
Scoped:  .tree("full") on /repo/docs/auth.md → 500 chars, then extract
```

**The fix**: Agents should explore directory structure first (ls, glob), identify the likely subdirectory, then run `.tree("full")` only on that target.

<details>
<summary>Full benchmark results</summary>

| Question | Mode | Chars Read | Savings |
|----------|------|------------|---------|
| Commit standards | without mq | 9,115 | - |
| | with mq (naive) | 12,877 | -41% |
| | with mq (scoped) | 2,144 | **76%** |
| Package installation | without mq | 10,407 | - |
| | with mq | 3,200 | **74%** |

Run it yourself: `./scripts/bench.sh`
</details>

## Comparison: mq vs qmd vs PageIndex

Benchmarked on LangChain monorepo (36 markdown files, 1,804 lines). [Full logs](benchmark/tool_comparison.md).

| Metric | **mq** | **[qmd](https://github.com/tobi/qmd)** | **[PageIndex](https://github.com/VectifyAI/PageIndex)** |
|--------|--------|---------|---------------|
| **Setup time** | 0 | 29s + 3.1GB models | 6s/file (API) |
| **Query latency** | **3-22ms** | 154ms (BM25) / 74s (semantic) | 6.3s |
| **Cost per query** | $0 | $0 (local) | ~$0.01-0.10 |
| **Dependencies** | Single binary | Bun, SQLite, node-llama-cpp | Python, OpenAI API |
| **Pre-indexing** | No | Yes (embed step) | Yes (tree generation) |
| **Works offline** | Yes | Yes (after model download) | No |

### Latency Comparison (same query: "commit standards")

```
mq:        22ms   ████
qmd BM25: 154ms   ███████████████████████████
qmd semantic: 74s ████████████████████████████████████████████████████████ (CPU, no GPU)
PageIndex: 6.3s   ████████████████████████████████████████████
```

**Core insight**: qmd and PageIndex compute results for you. mq doesn't - it exposes structure so the agent reasons to results itself:

- **qmd**: System computes similarity scores → returns ranked files
- **PageIndex**: System's LLM reasons over tree → returns relevant nodes
- **mq**: Exposes structure → agent reasons → agent finds what it needs

When the consumer is an LLM, it already has reasoning capability. mq leverages that instead of adding redundant computation layers.

### Why Markdown is Different

PageIndex uses heavy LLM processing because **PDF structure isn't deterministic** - you need an LLM to detect TOC pages, extract hierarchy, map page indices, and verify correctness.

But **markdown structure IS deterministic**. Headings, code blocks, lists - these can be parsed with an AST. No LLM needed to understand structure, only to reason over it.

This is mq's advantage: zero-cost structure extraction for formats where structure is explicit.

## Roadmap: Vision Support

For non-deterministic formats (PDFs, images, scanned documents), we're exploring a sub-agent architecture:

```
Main Agent (Opus/Sonnet)
    └── spawns Explorer Sub-Agent (Haiku with vision)
            └── examines PDF/image
            └── returns structured summary to main context
```

**The insight**: Vision-capable models (even Haiku) can do OCR. Instead of pre-processing documents with a separate service, reuse the agent infrastructure:

- **No pre-processing step** - explore on demand
- **Cheaper models for exploration** - Haiku has vision but costs less
- **Disposable context** - sub-agent's work doesn't pollute main context
- **Unified interface** - same query patterns for markdown and vision

This extends the mq philosophy: let agents reason over structure, but use sub-agents to extract structure from non-deterministic formats.

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/muqsitnawaz/mq/main/install.sh | bash
```

Or with Go (works on Windows too):

```bash
go install github.com/muqsitnawaz/mq@latest
```

### Agent Skill

Install the mq skill for Claude Code, Cursor, Codex, and other agents:

```bash
npx skills add muqsitnawaz/mq
```

See [skills.sh](https://skills.sh) for more.

Skills aren't always loaded into context. Add this line to your `CLAUDE.md` for optimal performance:

```markdown
Use `mq` to query markdown files. Narrow down to a specific file/subdir first, then run `mq <path> '.tree("full")'` to see structure before reading.
```

## Usage

### See Structure

```bash
# Document tree
mq README.md .tree

# With content previews
mq README.md '.tree("preview")'

# Directory overview
mq docs/ .tree

# Directory with sections + previews (best for agents)
mq docs/ '.tree("full")'
```

### Search

```bash
# Search in file
mq README.md '.search("OAuth")'

# Search across directory
mq docs/ '.search("authentication")'
```

### Extract Content

```bash
# Get section content
mq doc.md '.section("API") | .text'

# Get code blocks
mq doc.md '.code("python")'
mq doc.md '.section("Examples") | .code("go")'

# Get links, metadata
mq doc.md .links
mq doc.md .metadata
```

## Query Language

mq uses a jq-inspired query syntax with piping and selectors. If you're familiar with jq, see [docs/syntax.md](docs/syntax.md) for differences and design rationale.

### Selectors

| Selector | Description |
|----------|-------------|
| `.tree` | Document structure |
| `.tree("compact")` | Headings only |
| `.tree("preview")` | Headings + content preview |
| `.tree("full")` | Sections + previews (directories) |
| `.search("term")` | Find sections containing term |
| `.section("name")` | Section by heading |
| `.sections` | All sections |
| `.headings` | All headings |
| `.headings(2)` | H2 headings only |
| `.code` / `.code("lang")` | Code blocks |
| `.links` / `.images` / `.tables` | Other elements |
| `.metadata` / `.owner` / `.tags` | Frontmatter |

### Operations

| Operation | Description |
|-----------|-------------|
| `.text` | Extract raw content |
| `\| .tree` | Pipe to tree view |
| `filter(.level == 2)` | Filter results |

### Examples

```bash
mq doc.md '.headings | filter(.level == 2) | .text'
mq doc.md '.section("Examples") | .code("python")'
mq doc.md '.section("API") | .tree'
```

## Architecture

mq is built on a **Structural AST Pattern**: different formats are parsed into a common structural representation.

```
┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│ Markdown │  │   HTML   │  │   PDF    │  │JSON/YAML │
│  Parser  │  │  Parser  │  │  Parser  │  │  Parser  │
└────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘
     │             │             │             │
     └─────────────┴──────┬──────┴─────────────┘
                          ▼
          ┌───────────────────────────────┐
          │     Unified Document          │
          │   - Headings (h1-h6 levels)   │
          │   - Sections (hierarchical)   │
          │   - CodeBlocks (with lang)    │
          │   - Links, Images, Tables     │
          │   - ReadableText (for LLM)    │
          └───────────────┬───────────────┘
                          ▼
          ┌───────────────────────────────┐
          │       MQL Query Engine        │
          │  .headings | .section("API")  │
          └───────────────────────────────┘
```

### Core Components

- **`lib/`** - Core document engine and unified types
- **`mql/`** - Query language (lexer, parser, executor)
- **`html/`** - HTML parser with Readability extraction
- **`pdf/`** - PDF parser using PyMuPDF for structure
- **`data/`** - JSON, JSONL, YAML parsers

### Format-Agnostic Types

| Type | Markdown | HTML | PDF | JSON/YAML |
|------|----------|------|-----|-----------|
| Heading | `# Title` | `<h1>` | Large/bold text | Top-level keys |
| Section | Under heading | `<section>` | Chapter/page | Nested objects |
| CodeBlock | Triple backticks | `<pre><code>` | Monospace | N/A |
| Table | Pipe syntax | `<table>` | Aligned grid | Uniform arrays |
| ReadableText | Full content | Main content | All text | Pretty-printed |

## Library Usage

```go
import mq "github.com/muqsitnawaz/mq/lib"

engine := mq.New()
doc, _ := engine.LoadDocument("README.md")

// Direct API
headings := doc.GetHeadings(1, 2)       // H1 and H2 only
section, _ := doc.GetSection("Install") // Get specific section
code := doc.GetCodeBlocks("go")         // Go code blocks
```

For MQL string queries, use the `mql` package:

```go
import "github.com/muqsitnawaz/mq/mql"

engine := mql.New()
doc, _ := engine.LoadDocument("README.md")
result, _ := engine.Query(doc, `.section("API") | .code("go")`)
```

See [docs/library.md](docs/library.md) for the full API reference.

### Direct Document API

```go
// Load and parse document
engine := mql.New()
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
```

## Performance

Benchmarked on Apple M3 Max.

### Parsing Speed by Format

| Format | 100KB | 1MB | Throughput |
|--------|-------|-----|------------|
| Markdown | 2.4ms | 22ms | 45 MB/s |
| HTML | 57ms | ~500ms | 2.5 MB/s |
| JSON | 12us | 81us | 12 GB/s |
| JSONL | 27us | 187us | 5.6 GB/s |
| PDF | - | 1.9s | ~1 MB/s |

### Context Window Budget (200k tokens = 800KB)

**Structure-first approach** - load structure, not full text:

| Format | Traditional | mq Structure-First | Improvement |
|--------|-------------|-------------------|-------------|
| PDF | 16 papers | **800 PDFs** | 50x |
| Markdown | 16 docs | 80 docs | 5x |
| HTML | 8 pages | 40 pages | 5x |
| JSON/JSONL | - | 800KB / 8000 lines | - |

The agent loads ~1KB structure per PDF (vs ~50KB full text), reasons over 800 structures, then extracts only the sections it needs.

### Query Performance (after parsing)

| Query | Time | Notes |
|-------|------|-------|
| GetSection | 10ns | O(1) - pre-indexed |
| ReadableText | 0.3ns | O(1) - cached |
| GetHeadings | 6us | O(n) on heading count |
| GetCodeBlocks | 1.6us | O(n) on block count |

See [`bench/results.md`](bench/results.md) for full benchmarks.

## Dependencies

- **Markdown**: [goldmark](https://github.com/yuin/goldmark) - extensible markdown parser
- **HTML**: [x/net/html](https://golang.org/x/net/html) + custom Readability
- **PDF**: [PyMuPDF](https://pymupdf.readthedocs.io/) - structure extraction via Python
- **JSON/YAML**: Go standard library + [yaml.v3](https://gopkg.in/yaml.v3)

## Development

```bash
# Run tests
go test ./...

# Build CLI
go build -o mq .

# Install locally
go install .
```

## License

MIT
