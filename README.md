# mq - Agentic Querying for Structured Documents

[![CI](https://github.com/muqsitnawaz/mq/actions/workflows/ci.yml/badge.svg)](https://github.com/muqsitnawaz/mq/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/muqsitnawaz/mq)](https://github.com/muqsitnawaz/mq/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/muqsitnawaz/mq)](https://goreportcard.com/report/github.com/muqsitnawaz/mq)

AI agents waste tokens reading entire files. mq lets them query structure first, then extract only what they need. The agent's context window becomes the working index.

**Result: Up to 83% fewer tokens when scoped correctly.**

[Install](#installation) | [Usage](#usage) | [Query Language](#query-language)

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
