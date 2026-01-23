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

Traditional retrieval computes results for you. mq externalizes structure so the agent computes results itself:

```
Traditional: Documents → Query → Results (system computes the answer)
mq:          Documents → Query → Structure → Agent reasons → Results
```

mq is an **interface**, not an answer engine. It extracts structure into the agent's context, where the agent can reason over it directly.

**The insight**: LLMs already have semantic understanding and reasoning. They don't need another system to compute relevance - they need to **see** the structure so they can reason to answers themselves. mq makes documents legible to agents.

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

**Core insight**: qmd and PageIndex compute results for you. mq doesn't - it exposes structure so the agent reasons to results itself:

- **qmd**: System computes similarity scores → returns ranked files
- **PageIndex**: System's LLM reasons over tree → returns relevant nodes
- **mq**: Exposes structure → agent reasons → agent finds what it needs

When the consumer is an LLM, it already has reasoning capability. mq leverages that instead of adding redundant computation layers.

## Installation

```bash
go install github.com/muqsitnawaz/mq@latest
```

### Agent Integration

Add this to your `CLAUDE.md` (or equivalent config for other agents):

```markdown
Use `mq` to query markdown efficiently: `.tree("full")` shows structure with previews, `.section("X") | .text` extracts content. Scope queries to specific files or subdirs. Prefer to narrow down search scope when you can.
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
