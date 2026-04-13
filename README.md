# mq - Agentic Querying for Structured Documents

[![CI](https://github.com/muqsitnawaz/mq/actions/workflows/ci.yml/badge.svg)](https://github.com/muqsitnawaz/mq/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/muqsitnawaz/mq)](https://github.com/muqsitnawaz/mq/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/muqsitnawaz/mq)](https://goreportcard.com/report/github.com/muqsitnawaz/mq)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

AI agents waste tokens reading entire files. mq lets them query structure first, then extract only what they need. The agent's context window becomes the working index.

Embedding-based retrieval is [provably limited](https://arxiv.org/abs/2508.21038) by dimensionality — a fundamental ceiling, not a training problem. Stuffing full documents into context [degrades performance 14–85%](https://arxiv.org/abs/2510.05381) even with perfect retrieval. And [keyword search + agent reasoning matches 90%+ of RAG](https://arxiv.org/abs/2602.23368) without a vector database. Anthropic themselves [replaced RAG with agentic search](https://arxiv.org/abs/2501.09136) in Claude Code.

mq is built on this: expose structure, let the agent reason. No embeddings, no vector DB, no external APIs.

**Results:**
- **123 PDFs (365MB) triaged in 2.97s** with warm cache — full structural map of every document
- **83% fewer tokens** for markdown when scoped correctly
- **~20ms per PDF** on warm cache — sub-second `.tree` on 60+ PDFs, ~3s on 123
- **50x more PDFs** searchable (800 vs 16 in 200k context) via structure-first approach

One query model works across markdown, HTML, PDF, JSON, JSONL, and YAML.

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

When browsing directories, mq uses format-aware labels and expands per-file structure when available:

```bash
$ mq project/ .tree
project/ (6 files)
├── config.json (12 lines, 3 keys)
│   ├── key name
│   └── key database
├── config.yaml (15 lines, 4 keys)
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
             "Needle in html content."
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
mq data.yaml '.text'            # Flattened path:value text
mq data.yaml '.raw'             # Original source text

# JSONL - search logs and session files
mq session.jsonl '.search("auth")'  # Line-level search with record context
mq session.jsonl '.search("auth") | .text'  # Flatten all matched records
mq session.jsonl '.search("auth") | .nth(0)'  # Show one raw matched record
mq session.jsonl '.search("auth") | .nth(0) | .raw'  # Explicit raw record
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

### Test-Time Semantic Search

mq is grep for a caller that already understands meaning.

Traditional semantic search pre-computes embeddings and finds "nearness" — how close a query is to stored documents in vector space. Smart index, dumb query. mq inverts this: dumb index, smart caller.

An LLM already knows that "token refresh" is semantically near "OAuth," "session expiry," "credential rotation." It doesn't need a vector database to tell it that. So instead of pre-computing embeddings, let the model generate the right exact-match search terms itself:

1. **Read structure** (`.tree`) — see what each document contains and how it's organized
2. **Reason about nearness** — which terms would appear close to the target concept in these documents
3. **Search** (`.search("term")`) — fast, exact, deterministic
4. **Read matched sections, narrow further** — iterate until found

The semantic computation moves from a pre-built index to the model's inference pass. The LLM performs the "embedding" and "similarity search" implicitly when it decides what to search for. No pre-processing step, because the model that searches is the model that understands.

Structure is what makes this work. A flat text dump doesn't tell the model what's near what. Section headings, document hierarchy, and content previews give the model context to reason about better queries. mq exposes that structure; the model does the rest.

And unlike static embeddings, the model's sense of nearness is **contextual**. A vector embedding for "authentication" is the same vector regardless of what you're doing. A model searching for "authentication" while debugging logouts will look for different terms than one adding SSO. The search adapts to the task. Pre-computed embeddings can't.

mq is an **interface**, not an answer engine. It extracts structure into the agent's context, where the agent can reason over it directly. Agents like Claude Code and Codex are already LLMs with reasoning capability. Adding embedding APIs and rerankers just adds latency and cost. The agent can find what it needs — it just needs to **see** the structure.

## Research Background

Recent research validates the structure-first, agent-driven approach over traditional embedding pipelines.

### Embeddings Are Provably Limited

Weller et al. (2025) prove mathematically that the number of distinct top-k result sets an embedding model can return is bounded by its dimensionality — a fundamental limit of the single-vector paradigm, not a training problem. State-of-the-art models fail on straightforward retrieval tasks in their LIMIT benchmark, even when embeddings are optimized directly on test data.

> *"These theoretical limits manifest in realistic settings with simple queries... requiring entirely new approaches rather than incremental improvements."*
> — [On the Theoretical Limitations of Embedding-Based Retrieval](https://arxiv.org/abs/2508.21038)

Benescu & de Jong (2026) argue that "similarity is a short-sighted interpretation of relevance" and that LLM-based reasoning should theoretically outperform embedding retrieval — but current benchmarks can't measure the difference because human annotations contain the same short-sightedness.

> — [Why LLMs can Secretly Outperform Embedding Similarity in IR](https://arxiv.org/abs/2603.08077)

### Context Stuffing Hurts — Structure Helps

Longer context doesn't mean better results. Du et al. (EMNLP 2025) show that even when models can perfectly retrieve all relevant information, performance still degrades 13.9–85% as input length increases — sheer token volume hurts reasoning regardless of retrieval quality.

> *"Even when all relevant evidence is placed immediately before the question, performance degrades substantially."*
> — [Context Length Alone Hurts LLM Performance Despite Perfect Retrieval](https://arxiv.org/abs/2510.05381)

Chroma Research (2025) tested 18 models (Claude Opus 4, GPT-4.1, Gemini 2.5 Pro) and found performance declined with increasing context across all of them. A single distractor reduces accuracy. Models performed better on randomly shuffled haystacks than coherent ones — meaning how you organize context matters more than having it all.

> — [Context Rot](https://www.trychroma.com/research/context-rot)

This is why mq loads ~1KB of structure per document instead of ~50KB of full text. The agent sees more documents and reasons better over less noise.

### Keyword Search + Agent Reasoning Matches RAG

Subramanian et al. at Amazon (2025) show that tool-based keyword search within an agentic framework achieves over 90% of traditional RAG performance — without a vector database. Simpler to implement, cheaper to run, and no index to maintain.

> — [Keyword Search Is All You Need](https://arxiv.org/abs/2602.23368)

Wang et al. (2025) propose ELITE, an embedding-less retrieval system using iterative LLM reasoning. It outperforms embedding baselines on long-context QA with over an order of magnitude reduction in storage and runtime:

> *"Embedding-based retrieval can retrieve content that is semantically similar in form but misaligned with the question's true intent."*
> — [ELITE: Embedding-Less Retrieval with Iterative Text Exploration](https://arxiv.org/abs/2505.11908)

### Agentic Search Wins in Practice

Anthropic built a full RAG pipeline for Claude Code with embeddings and vector DB, then replaced it with agentic search (grep, glob, file reads). Boris Cherny, creator of Claude Code: *"We found pretty quickly that agentic search generally works better. It is also simpler and doesn't have the same issues around security, privacy, staleness, and reliability."*

Google DeepMind's LOFT benchmark (2024) found that long-context LLMs show *"surprising ability to rival state-of-the-art retrieval and RAG systems, despite never having been explicitly trained for these tasks"* on tasks requiring up to millions of tokens of context.

> — [Can Long-Context Language Models Subsume Retrieval, RAG, SQL, and More?](https://arxiv.org/abs/2406.13121)

Microsoft Research's Code Researcher (2025) validates the Map-Narrow-Extract pattern: agents that explore 10 unique files per trajectory achieve 58% crash resolution vs 37.5% for agents that explore 1.33 files. Depth of structural exploration directly correlates with success.

> — [Code Researcher: Deep Research Agent for Large Systems](https://arxiv.org/abs/2506.11060)

### Long Context Beats RAG — When Used Right

Li et al. at Google (EMNLP 2024) found that *"when resourced sufficiently, long-context consistently outperforms RAG in average performance."* Their Self-Route hybrid routes queries to RAG or long-context based on model self-reflection, using only 38–61% of tokens while matching full long-context performance.

> — [Retrieval Augmented Generation or Long-Context LLMs?](https://arxiv.org/abs/2407.16833)

The Agentic RAG survey (Singh et al., 2025) establishes the taxonomy: traditional RAG operates through *"static workflows and lacks adaptability for multi-step reasoning."* Agentic RAG uses *"reflection, planning, tool use, and multi-agent collaboration to dynamically manage retrieval strategies."*

> — [Agentic Retrieval-Augmented Generation: A Survey](https://arxiv.org/abs/2501.09136)

### The Direction

The research consensus is clear: naive single-shot embedding lookup is being superseded. The future is agents that reason over structure iteratively — which is exactly what mq enables. Expose structure, let the agent reason, extract only what's needed.

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
mq session.jsonl ".search('auth') | .nth(0)"
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

### PDF Directory Triage

Run `.tree` on a directory of PDFs to get a structural map of every document:

```bash
$ mq papers/ .tree
papers/ (9 files, 11143 lines total)
├── ai_2301.00001.pdf (11 pages, 20 sections)
│   └── H2 NFTrig: Using Blockchain Technologies for Math Education
│            "JORDAN THOMPSON, Augustana College, USA"
├── cl_2302.00001.pdf (20 pages, 27 sections)
│   ├── H2 Quantum Computing for Plasma Physics
│   │        "Oscar Amaro and Diogo Cruz"
│   ├── H2 Introduction
│   │        "Quantum Computing (QC) is a branch of computing..."
│   ├── H2 Conclusions
│   └── H2 References
├── govt_nist_ai_risk.pdf (48 pages, 131 sections)
│   ├── H1 Artificial Intelligence Risk Management
│   ├── H1 Framework (AI RMF 1.0)
│   │        "NIST AI 100-1"
│   ├── H2 Executive Summary
│   └── H2 How AI Risks Differ from Traditional Software Risks
├── govt_nist_cybersecurity.pdf (55 pages, 696 sections)
│   ├── H1 Critical Infrastructure Cybersecurity
│   ├── H2 Executive Summary
│   │        "The United States depends on the reliable..."
│   └── H2 Appendix A: Framework Core
└── govt_nist_zero_trust.pdf (59 pages, 100 sections)
    ├── H1 NIST Special Publication 800-207
    └── H1 Zero Trust Architecture
```

One call. Title, authors, page count, section count, and heading hierarchy for every PDF. With warm cache, this runs in **<1s for 60 PDFs** and **~3s for 123 PDFs**.

## Query Language

mq uses a jq-inspired query syntax with piping and selectors. If you're familiar with jq, see [docs/syntax.md](docs/syntax.md) for differences and design rationale.

The query language stays the same across formats. What changes is the structure that the parser can populate for a given document.

### Selectors

| Selector | Description |
|----------|-------------|
| `.tree` | Document structure (adapts to file vs directory) |
| `.search("term")` | Find sections containing term (JSONL: line-level) |
| `.nth(N)` | Pick the Nth item from current results (0-based) |
| `.text` | Extract text content / flattened structured text |
| `.raw` | Extract source text / raw matched record |
| `.section("name")` | Section by heading |
| `.sections` | All sections |
| `.headings` | All headings |
| `.headings(2)` | H2 headings only |
| `.code` / `.code("lang")` | Code blocks |
| `.links` / `.images` / `.tables` | Other elements |
| `.metadata` / `.owner` / `.tags` | Frontmatter |
| `.md` / `.html` / `.json` / `.yaml` | Format cast: reparse string as another format |

### Operations

| Operation | Description |
|-----------|-------------|
| `.text` | Extract raw content |
| `\| .tree` | Pipe to tree view |
| `filter(.level == 2)` | Filter results |
| `depth(N)` | Limit tree traversal to N levels |
| `limit(N)` | Show max N entries per directory |

### Format Casts

Cast operators reinterpret a string value as a different document format mid-pipeline.
Use when structured content is embedded inside another format (e.g. markdown inside JSONL).

| Cast | Parses as | Example |
|------|-----------|---------|
| `.md` | Markdown | `.text \| .md \| .headings` |
| `.html` | HTML | `.text \| .html \| .links` |
| `.json` | JSON | `.raw \| .json \| .section("key")` |
| `.yaml` | YAML | `.text \| .yaml \| .tree` |

```bash
# JSON field containing markdown -> extract headings
mq data.json '.section("readme") | .text | .md | .headings'

# JSONL record -> parse as JSON -> drill to a field -> cast to markdown
mq log.jsonl '.search("report") | .nth(0) | .raw | .json | .section("content") | .text | .md | .section("Summary") | .text'

# Claude session files: search conversations, extract structured content
mq ~/.claude/projects/-Users-you-project/ '.search("auth")'
mq session.jsonl '.search("AUDIT") | .nth(0) | .raw | .json | .section("content") | .text | .md | .headings'
```

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

Benchmarked on Apple M3 Max, Go 1.24. The tables below only include benchmark paths that currently hit the real parser/query implementations.

### Headline Numbers

| Path | Current benchmark result |
|------|--------------------------|
| Markdown parse | 100KB: 2.70ms, 1MB: 23.48ms, 10MB: 224.74ms |
| Markdown throughput | ~38-47 MB/s across 100KB-10MB |
| HTML parse | 1KB: 0.98ms, 10KB: 10.63ms, 100KB: 157.77ms |
| HTML throughput | ~0.65-1.09 MB/s |
| YAML parse | 1KB: 0.12ms, 10KB: 0.88ms, 100KB: 12.39ms |
| YAML throughput | ~8.28-11.65 MB/s |
| PDF cold parse | 10.86s-13.42s on 757KB-6.6MB real PDFs |
| PDF warm cache hit | 11.16ms-16.68ms |
| PDF BuildTree | 0.216ms-0.567ms |
| PDF Search | 0.754ms-0.973ms |
| MQL `.section("X") \| .text` | 9.58us after parse |

### PDF Benchmark Profile (real PDFs, Apple M3 Max)

Measured with:

```bash
go test ./pdf/... -bench=BenchmarkPDF -benchmem -count=1
```

| File | Size | Cold parse | Warm cache hit | BuildTree | Search |
|------|------|------------|----------------|-----------|--------|
| `bert.pdf` | 757KB | 13.25s | 16.68ms | 0.377ms | 0.973ms |
| `attention.pdf` | 2.1MB | 10.86s | 11.16ms | 0.567ms | 0.845ms |
| `raft.pdf` | 6.6MB | 13.42s | 12.00ms | 0.216ms | 0.754ms |

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
| GetSection | 9.2ns | O(1) exact title lookup |
| GetSectionFuzzy | 10.5ns | O(1) fuzzy title lookup |
| ReadableText | 0.28ns | O(1) cached string access |
| GetHeadings | 0.14us (1KB) to 8.34us (1MB) | Scales with heading count |
| GetCodeBlocks | 28ns (1KB) to 1.86us (1MB) | Scales with code block count |
| MQL `.headings` | 0.55us | Full lex/parse/compile/exec |
| MQL `.section("X") \| .text` | 9.58us | Piped query with extraction |

### PDF Directory Benchmark (real corpus, 123 arXiv/NIST/OpenStax PDFs)

Tested on Apple M3 Max. Corpus: 123 PDFs, 365MB, 317K lines across arXiv papers, NIST reports, and OpenStax textbooks.

| Query | Files | Cold | Warm (cached) | Speedup |
|-------|-------|------|---------------|---------|
| `.tree` | 9 | 24.5s | 0.25s | **98x** |
| `.tree` | 29 | 2:24 | 0.62s | **233x** |
| `.tree` | 58 | 1:40 | 0.96s | **104x** |
| `.tree` | 123 | 5:02 | **2.97s** | **101x** |
| `.search("algorithm")` | 123 | — | **4.0s** | — |
| `.search("security")` | 123 | — | **4.4s** | 3,311 match lines |
| `.section("risk") \| .text` | 1 (48pg) | — | **0.2s** | — |

Cold parse is the one-time cost (PDF text + structure extraction). The cache (56MB bbolt DB for 123 PDFs) persists across sessions. Per-file warm cost: ~20ms.

### Parse + Search Cache (v0.3.3+)

Parsed documents and directory search results are cached in a content-addressed bbolt database (`~/Library/Caches/mq/cache.db` on macOS). Subsequent queries on the same file skip parsing, and repeated directory searches can skip the full scan when the tree hash is unchanged.

On the PDF corpus above, repeated loads drop from roughly 10.9-13.4 seconds to roughly 11-17 milliseconds once the cache is warm.

Warm cache hits still validate the file and deserialize the cached `Document`, so the main user-visible win is latency, not just throughput.

### Directory Search Cache (real corpora, Apple M3 Max)

Measured with:

```bash
go test ./mql -bench 'BenchmarkDirectorySearch$' -run '^$' -benchtime=1x -count=1
```

| Corpus | Cold | Warm exact repeat | Partial invalidation |
|--------|------|-------------------|----------------------|
| `private-manuscript` (185 files, 178 Markdown docs, 65,175 Markdown lines) | 2.21s | 11.98ms | 1.62s |
| `~/.rush/sessions` (4.2GB) | 51.86s | 440.34ms | - |

Warm exact-repeat is still not free on very large trees because `LookupDirSearch` first recomputes the current directory hash before reusing cached results.

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
