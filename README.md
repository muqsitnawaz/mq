# mq - Agentic Querying for Structured Documents

AI agents waste tokens reading entire files. mq lets them query structure first, then extract only what they need. The agent's context window becomes the working index.

**Result: Up to 74% fewer tokens, 49% faster responses.**

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

Traditional retrieval separates indexing from querying. mq unifies them:

```
Traditional: Documents → Build Index → Store in DB → Query Index → Results
mq:          Documents → Query → Structure emerges in agent's context
```

Each query returns structure that accumulates in the agent's context window. The context **becomes** the working index. No vector database, no embeddings, no storage layer.

**The insight**: LLMs already have semantic understanding. They can try synonyms, rephrase queries, and decide where to explore. Adding another embedding layer is redundant when the agent itself is the reasoning engine.

## Benchmark: 74% Token Reduction

We benchmarked agents answering questions about the [LangChain](https://github.com/langchain-ai/langchain) monorepo (50+ markdown files):

| Metric | Without mq | With mq | Improvement |
|--------|------------|---------|-------------|
| Total input tokens | 1,353,699 | 1,097,370 | **19% fewer** |
| Excluding outliers | 945,926 | 551,662 | **42% fewer** |
| Best case | 412,668 | 108,225 | **74% fewer** |

The "with mq" agent uses `.tree("full")` to see structure, then `.section() | .text` to extract. The "without mq" agent reads entire files.

<details>
<summary>Full benchmark results</summary>

| Question | Mode | Input Tokens | Latency | Savings |
|----------|------|--------------|---------|---------|
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

Run it yourself: `./scripts/bench.sh`
</details>

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

**Core insight**: Markdown has explicit structure (`#` headings). All three tools parse the same headers - they differ in how they navigate:

- **qmd**: Converts text to vectors, searches by mathematical similarity
- **PageIndex**: Builds tree, uses LLM to reason which branch to follow
- **mq**: Exposes tree directly, lets the agent reason over it

When the consumer is an LLM agent, adding another embedding layer (qmd) or another LLM layer (PageIndex) is redundant. The agent navigates structure directly.

## Installation

```bash
go install github.com/muqsitnawaz/mq@latest
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
