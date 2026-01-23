# Tool Comparison: mq vs qmd vs PageIndex

Benchmarks run on the LangChain monorepo (36 markdown files, 1,804 lines).

## Setup Cost

| Tool | Setup Time | Models/Dependencies | First-run Cost |
|------|------------|---------------------|----------------|
| **mq** | 0 | Single Go binary | $0 |
| **qmd** | ~29s | 3.1GB models (embedding 329MB + expansion 2.2GB + reranker 639MB) | $0 (local) |
| **PageIndex** | ~6s/file | Python + OpenAI API | ~$0.01-0.05/file |

## Query Latency

| Tool | Operation | Latency | Notes |
|------|-----------|---------|-------|
| **mq** | `.tree` (directory) | 13ms | Instant |
| **mq** | `.tree("full")` | 15ms | With previews |
| **mq** | `.search("...")` | 22ms | Substring match |
| **mq** | `.section() \| .text` | 3ms | Content extraction |
| **qmd** | `search` (BM25) | 154ms | Keyword search |
| **qmd** | `vsearch` (vector) | 74s | CPU LLM inference |
| **qmd** | `query` (hybrid) | 76s | CPU LLM inference |
| **PageIndex** | Tree generation | 6.3s | OpenAI API call |

**Note**: qmd's semantic search (vsearch, query) is slow because it runs LLM inference locally on CPU. With GPU, latency would be ~1-5s. However, this still requires downloading and loading 3.1GB of models.

## Output Comparison

Same query: "commit standards"

### mq
```bash
$ mq /tmp/langchain-bench '.search("commit standards")'
# Time: 22ms
```
Output:
```
Found 3 matches for "commit standards":

AGENTS.md:
  #### Commit standards (lines 73-84)
     "...Suggest PR titles that follow Conventional Commits format..."
```

### qmd
```bash
$ qmd search "commit standards" -n 3
# Time: 154ms
```
Output:
```
qmd://langchain/claude.md:73 #7b7a4b
Title: Global development guidelines for the LangChain monorepo
Score: 100%

@@ -72,4 @@ (71 before, 116 after)
#### Commit standards
Suggest PR titles that follow Conventional Commits format...
```

### PageIndex
```bash
$ python run_pageindex.py --md_path AGENTS.md
# Time: 6.3s (generates tree with LLM summaries)
```
Output (tree structure):
```json
{
  "title": "Commit standards",
  "node_id": "0005",
  "summary": "#### Commit standards\n\nSuggest PR titles that follow Conventional Commits format...",
  "line_num": 73
}
```

## Key Findings

### Speed
- **mq is 7-10x faster** than qmd's BM25 search
- **mq is 3000x faster** than qmd's semantic search (on CPU)
- **mq is 280x faster** than PageIndex tree generation

### Cost
- **mq**: $0 always
- **qmd**: $0 but requires 3.1GB disk space and significant CPU/GPU for semantic features
- **PageIndex**: ~$0.01-0.10 per document (OpenAI API)

### What Each Tool Returns
- **mq**: Structure + exact content (agent reasons over it)
- **qmd**: Ranked file paths + relevance scores (system computes relevance)
- **PageIndex**: Tree structure + LLM summaries (system pre-computes summaries)

## Conclusion

For AI agent workflows with markdown documents:

1. **mq** exposes structure for agents to reason over (fastest, free)
2. **qmd** computes relevance for you (slower, requires models)
3. **PageIndex** pre-computes summaries (slowest, costs money)

When the consumer is an LLM agent, it already has reasoning capability. mq leverages that instead of adding redundant computation layers.
