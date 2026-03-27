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
| PDF | `.pdf` | Headings (font-size inference), page numbers, tables, text |
| JSON | `.json` | Top-level keys as headings, nested structure |
| JSONL | `.jsonl`, `.ndjson` | Line-level search, per-record drill-in |
| YAML | `.yaml`, `.yml` | Keys as headings, nested structure |

### Directory Tree Labels

When browsing directories, mq uses format-aware labels:

```bash
$ mq project/ .tree
project/ (6 files)
├── config.json (12 lines, 3 keys)
├── config.yaml (15 lines, 4 keys)
├── README.md (80 lines, 5 sections)
├── report.pdf (24 pages, 8 sections)
├── events.jsonl (100 lines, 98 records)
└── index.html (45 lines, 3 sections)
```

Trees show format-specific heading labels and sizing:

```bash
$ mq project/ .tree
project/ (6 files)
├── config.json (12 lines, 3 keys)
│   ├── key name
│   └── key database
├── README.md (80 lines, 5 sections)
│   ├── # Overview
│   │        "Complete reference for..."
│   └── ## Install
│            "Run the install script..."
├── report.pdf (24 pages, 8 sections)
│   ├── H1 Introduction (p. 1)
│   │        "This report covers Q4 results..."
│   └── H2 Methodology (p. 5)
│            "We used a mixed-methods approach..."
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
mq docs/ .tree
mq docs/auth.md ".section('OAuth Flow') | .text"

# HTML - readable content from web pages
mq page.html '.headings'
mq page.html '.text'

# PDF - extract structure from papers
mq paper.pdf '.headings'
mq paper.pdf '.tables'

# JSON/YAML - query data files
mq config.json '.headings'      # Top-level keys
mq data.yaml '.text'            # Readable representation

# JSONL - search logs and session files
mq session.jsonl '.search("auth")'  # Line-level search with record context
mq session.jsonl '.search("auth") | .text'  # Expand all matched records
mq session.jsonl '.search("auth") | .nth(0) | .text'  # Narrow to one matched record
mq sessions/ '.search("requires OAuth") | .tree'  # Search whole session directories with structured record output
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

*When agent narrows down to specific file before running `.tree`

### The Scoping Insight

Running `.tree` on an entire repo is expensive. For 50 files, the tree output alone is ~22,000 characters before extracting any content.

```
Naive:   .tree on /repo           → 22K chars just for tree
Scoped:  .tree on /repo/docs/auth.md → 500 chars, then extract
```

**The fix**: Agents should explore directory structure first, identify the likely subdirectory, then run `.tree` only on that target.

### Scaling to Large Corpora

For repositories with thousands of files, use `depth()` and `limit()` to bound traversal:

```bash
# Level 0: See top-level structure (max 50 entries per directory)
mq corpus/ ".tree | depth(2) | limit(50)"

# Output shows what's truncated:
# corpus/ (10247 files, 500000 lines total)
# ├── auth/ (234 files, depth limit)
# ├── api/
# │   ├── v1/ (45 files, depth limit)
# │   ├── v2/ (38 files, depth limit)
# │   └── ... (12 more)
# └── ... (103 more)

# Level 1: Narrow to likely area
mq corpus/auth/ ".tree | limit(20)"

# Level 2: Extract what you need
mq corpus/auth/oauth.md ".section('Token Refresh') | .text"
```

The agent reasons at each level. No 10k-file index needed - this mirrors how humans explore large codebases.

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

### Why Markdown Is Still Easier

Markdown structure is explicit. Headings, code blocks, links, tables, and lists can be parsed directly from the AST with stable line ranges.

PDFs are supported too, but their structure is inferred from layout cues like font size, boldness, and page position. That makes PDF parsing slower and more heuristic than markdown, even though the query interface stays the same once the `Document` is built.

This is the tradeoff mq makes: keep one query language, but let each parser extract the strongest deterministic structure it can for that format.

## Roadmap: Vision Support

Text PDFs already go through the built-in PDF parser. The remaining frontier is image-heavy inputs: scanned PDFs, screenshots, diagrams, and pages where layout matters more than extracted text.

For those cases, we're exploring a sub-agent architecture:

```
Main Agent (Opus/Sonnet)
    └── spawns Explorer Sub-Agent (Haiku with vision)
            └── examines scanned page / image
            └── returns structured summary to main context
```

**The insight**: vision-capable models can recover structure when text extraction and layout heuristics stop being enough. Instead of pre-processing everything with a separate service, reuse the agent infrastructure only for the hard cases:

- **No pre-processing step** - explore on demand
- **Cheaper models for exploration** - Haiku has vision but costs less
- **Disposable context** - sub-agent's work doesn't pollute main context
- **Unified interface** - same high-level workflow: structure, search, extract

This extends the mq philosophy: ordinary markdown, HTML, JSON, YAML, JSONL, and text PDFs stay on the fast local path; sub-agents are reserved for inputs that do not expose usable structure directly.

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
Use `mq` to query markdown files. Narrow down to a specific file/subdir first, then run `mq <path> .tree` to see structure before reading.
```

## Usage

> **Shell quoting:** Examples use double quotes for the outer string (`"..."`), which works on all platforms including Windows. On macOS and Linux, single quotes also work: `mq doc.md '.section("API")'`.

The CLI shape does not change by format: `mq <path> [query]`.

The same three-step pattern works on every format: **structure -> search -> extract**.

### See Structure

```bash
# Any single file
mq README.md .tree
mq paper.pdf .tree
mq page.html .tree

# Directory overview (all formats, with previews)
mq docs/ .tree
```

### Search

```bash
# Works the same across formats
mq README.md ".search('OAuth')"
mq paper.pdf ".search('methodology')"
mq docs/ ".search('authentication')"

# JSONL: line-level search with record type + structure
mq session.jsonl ".search('auth')"
# → [line 3] assistant/tool_use: Grep
#     ts: 2026-02-01T20:25:34Z
#     > ...searching for auth configuration...

# Expand matching records directly
mq session.jsonl ".search('auth') | .text"

# Tree view of matched records
mq sessions/ ".search('requires OAuth') | .tree"

# Expand all matched records across a directory
mq sessions/ ".search('requires OAuth') | .text"

# Pick one matched record only if you need to narrow (0-based), jq-style
mq session.jsonl ".search('auth') | .nth(0) | .text"
```

### Extract Content

```bash
# Same selectors, any format
mq doc.md ".section('API') | .text"
mq paper.pdf ".section('Results') | .text"
mq page.html ".section('Features') | .text"

# Format-specific content
mq doc.md ".code('python')"                    # Code blocks (Markdown, HTML)
mq doc.md ".section('Examples') | .code('go')" # Code within a section
mq doc.md .links                                # Links
mq doc.md .metadata                             # YAML frontmatter

# Data formats
mq config.json .tree                            # Keys as structure
mq data.yaml ".section('database') | .text"     # YAML sections
```

### PDF-Specific Output

PDFs show page numbers instead of line numbers:

```bash
$ mq paper.pdf .tree
paper.pdf (12 pages)
├── H1 Abstract (p. 1)
│        "We propose a new architecture for..."
├── H1 Introduction (p. 1)
│        "Recent advances in deep learning..."
├── H1 Methodology (p. 3)
│        "Our approach builds on transformer..."
│   ├── H2 Data Collection (p. 3)
│   └── H2 Model Architecture (p. 5)
└── H1 Results (p. 8)
         "Table 1 shows the comparison..."

$ mq paper.pdf ".section('Methodology') | .text"
# Returns the full text of that section
```

## Query Language

mq uses a jq-inspired query syntax with piping and selectors. If you're familiar with jq, see [docs/syntax.md](docs/syntax.md) for differences and design rationale.

The query language stays the same across formats. What changes is the structure that the parser can populate for a given document.

### Selectors

| Selector | Description |
|----------|-------------|
| `.tree` | Document structure (adapts to file vs directory) |
| `.search("term")` | Find sections containing term (JSONL: line-level) |
| `.nth(N)` | Pick the Nth item from current results (0-based) |
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
| `depth(N)` | Limit tree traversal to N levels |
| `limit(N)` | Show max N entries per directory |

### Examples

```bash
mq doc.md ".headings | filter(.level == 2) | .text"
mq doc.md ".section('Examples') | .code('python')"
mq doc.md ".section('API') | .tree"
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

General parser throughput numbers below were benchmarked on Apple M4.

### Parsing Speed by Format

| Format | 100KB | 1MB | Throughput |
|--------|-------|-----|------------|
| Markdown | 1.7ms | 17ms | 65 MB/s |
| HTML | 67ms | ~600ms | 1.7 MB/s |
| YAML | 5.7ms | ~50ms | 19 MB/s |
| JSON | 7.3us | 52us | 20 GB/s |
| JSONL | 17us | 133us | 8 GB/s |
| PDF | - | 1.9s | ~1 MB/s |

### PDF Benchmark Profile (real PDFs, Apple M3 Max)

Measured with:

```bash
go test ./pdf/... -bench=BenchmarkPDF -benchmem -count=1
```

| File | Size | Cold parse | Warm cache hit | BuildTree | Search |
|------|------|------------|----------------|-----------|--------|
| `bert.pdf` | 757KB | 10.84s | 10.88ms | 0.322ms | 0.685ms |
| `attention.pdf` | 2.1MB | 9.90s | 11.44ms | 0.508ms | 0.731ms |
| `raft.pdf` | 6.6MB | 11.08s | 12.33ms | 0.194ms | 0.686ms |

Cold parse covers the full PDF pipeline. Warm cache hit measures `Cache.LookupFile`, which skips parsing and deserializes the cached `Document`.

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
| GetSection | 7ns | O(1) - pre-indexed |
| ReadableText | 0.2ns | O(1) - cached |
| GetHeadings | 6us | O(n) on heading count |
| GetCodeBlocks | 1.6us | O(n) on block count |
| MQL `.headings` | 327ns | Full lex/parse/compile/exec |
| MQL `.section("X") \| .text` | 5.6us | Piped query with extraction |

### Parse + Search Cache (v0.3.3+)

Parsed documents and directory search results are cached in a content-addressed bbolt database (`~/Library/Caches/mq/cache.db` on macOS). Subsequent queries on the same file skip parsing, and repeated directory searches can skip the full scan when the tree hash is unchanged.

On the PDF corpus above, repeated loads drop from roughly 10-11 seconds to roughly 11-12 milliseconds once the cache is warm.

On a real session corpus (`~/.rush/sessions`), repeated directory search improved from roughly 5.1-5.6 seconds to roughly 1.38-1.40 seconds for:

```bash
mq ~/.rush/sessions '.search("requires OAuth") | .tree'
```

Corpus shape for that run:
- `8,495` supported files total
- `4,644` `.json` files
- `3,083` `.jsonl` / `.ndjson` files
- `587,122,579` bytes across all supported files
- `352,048,374` bytes across the JSON/JSONL subset
- `308,408` non-empty JSONL records
- Query result: `86` matched paths and `176` matched records for `"requires OAuth"`

That is roughly a 3.7x speedup, even when measured via `go run` with process startup noise included.

**How it works:**
1. **Parse cache**: SHA256 content hash keys the parsed `Document`, so repeated file queries skip reparsing and deserialize the cached structure instead.
2. **Directory search cache**: `(directory hash, query)` keys exact-repeat directory searches, so unchanged trees can return cached `SearchResults` immediately.
3. **Per-file search cache**: `(path, query, mtime, size)` caches file-level matches so partially changed trees only reread the files that actually changed.
4. **Byte reuse on matched files**: directory search reuses bytes already read during the scan instead of rereading matched files before parse.
5. **Merkle directory tree**: each directory stores a hash of its children's metadata, so repeated searches can detect unchanged trees without re-reading file contents first.
6. **Auto-eviction**: entries unused for 5+ days are trimmed on startup

Cache can be cleared by deleting the database file or running `rm ~/Library/Caches/mq/cache.db`.

See [`bench/results.md`](bench/results.md) for full benchmarks.

## Dependencies

- **Markdown**: [goldmark](https://github.com/yuin/goldmark) - extensible markdown parser
- **HTML**: [x/net/html](https://golang.org/x/net/html) + custom Readability
- **PDF**: [PyMuPDF](https://pymupdf.readthedocs.io/) - structure extraction via Python
- **JSON/YAML**: Go standard library + [yaml.v3](https://gopkg.in/yaml.v3)
- **Cache**: [bbolt](https://github.com/etcd-io/bbolt) - single-file embedded database
- **Serialization**: [msgpack](https://github.com/vmihailenco/msgpack) - fast binary encoding (5x faster than gob)

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
