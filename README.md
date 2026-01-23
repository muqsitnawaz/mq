# mq - Agentic Querying for Structured Documents

[![CI](https://github.com/muqsitnawaz/mq/actions/workflows/ci.yml/badge.svg)](https://github.com/muqsitnawaz/mq/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/muqsitnawaz/mq)](https://github.com/muqsitnawaz/mq/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/muqsitnawaz/mq)](https://goreportcard.com/report/github.com/muqsitnawaz/mq)

AI agents waste tokens reading entire files. mq lets them query structure first, then extract only what they need. The agent's context window becomes the working index.

**Result: Up to 83% fewer tokens when scoped correctly.**

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

```bash
# Agent sees the structure (this IS the index)
mq docs/ '.tree("full")'
# → docs/ (12 files, 2847 lines)
# → ├── auth.md (234 lines, 8 sections)
# → │   ├── ## Authentication
# → │   │        "OAuth 2.0 and API key authentication..."
# → │   └── ## OAuth Flow
# → │            "Step-by-step OAuth implementation..."

# Agent extracts only what it needs
mq docs/auth.md '.section("OAuth Flow") | .text'
```

## Why This Works

Traditional retrieval computes results for you. mq externalizes structure so the agent computes results itself:

```
┌─────────────────────────────────────────────────────────────────────────┐
│  Traditional RAG                                                        │
│  Documents → Embeddings → Vector DB → Query → System computes → Results │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│  mq                                                                     │
│  Documents → Agent queries structure → Agent reasons → Agent extracts   │
│                    ↑                                                    │
│              (zero LLM cost)                                            │
└─────────────────────────────────────────────────────────────────────────┘
```

mq is an **interface**, not an answer engine. It extracts structure into the agent's context, where the agent can reason over it directly.

**The insight**: LLMs already have semantic understanding and reasoning. They don't need another system to compute relevance - they need to **see** the structure so they can reason to answers themselves. mq makes documents legible to agents.

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

Three approaches to document retrieval for AI agents:

| | **mq** | **[qmd](https://github.com/tobi/qmd)** | **[PageIndex](https://github.com/VectifyAI/PageIndex)** |
|--|--------|---------|---------------|
| **Target** | Markdown | Markdown | PDFs |
| **Technique** | AST parsing + query language | Vector embeddings + BM25 | LLM-generated tree structure |
| **Index** | On-demand (agent context) | Pre-built (SQLite + vectors) | Pre-built (JSON tree) |
| **Retrieval** | Deterministic query | Similarity search | LLM traverses tree |
| **Dependencies** | Single Go binary | 3GB models, Bun, SQLite | Python, OpenAI API |
| **Cost per query** | Zero | Local compute | LLM API calls |
| **Output** | Exact content | Paths + scores | Node references |

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

Or with Go:

```bash
go install github.com/muqsitnawaz/mq@latest
```

### Agent Integration

Add to your `CLAUDE.md` or system prompt:

```markdown
# Markdown Queries (mq)
- mq <path> '.tree("full")' shows structure with content previews
- mq <file> '.section("Name") | .text' extracts specific content
- Scope matters: tree on a single file is cheap, tree on 1000 files is expensive
- Use judgment: small repo? tree directly. Large repo? explore briefly to find the right subdir first.
```

The insight: agents that scope their queries save 80%+ tokens. But don't over-prescribe - let the agent judge based on repo size.

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

## Library Usage

```go
import "github.com/muqsitnawaz/mq/mql"

engine := mql.New()
doc, _ := engine.LoadDocument("README.md")
result, _ := engine.Query(doc, `.section("API") | .code("go")`)
```

## License

MIT
