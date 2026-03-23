# Tool Comparison: mq vs qmd vs PageIndex

Benchmark date: 2025-01-23

**Corpus**: LangChain monorepo
- 36 markdown files
- 1,804 total lines
- Path: `/tmp/langchain-bench`

## Summary

| Metric | mq | qmd | PageIndex |
|--------|-----|-----|-----------|
| **Setup time** | 0 | 29s | 6s/file |
| **Dependencies** | Single binary | 3.1GB models | Python + OpenAI API |
| **Query latency** | 3-22ms | 154ms-76s | 6.3s |
| **Cost per query** | $0 | $0 (local) | ~$0.01-0.10 |

## Setup Cost

### mq
```bash
$ go build .
# Time: ~1s
# Dependencies: None
# Models: None
```

### qmd
```bash
# Step 1: Install (804ms)
$ bun install
259 packages installed [804.00ms]

# Step 2: Create collection (467ms)
$ qmd collection add /tmp/langchain-bench --name langchain
Indexed: 32 new, 0 updated, 0 unchanged, 0 removed
real    0m0.467s

# Step 3: Generate embeddings (28.8s, downloads 329MB model)
$ qmd embed
Downloading hf_ggml-org_embeddinggemma-300M-Q8_0.gguf (328.58MB)...
Embedding 31 documents (37 chunks)...
real    0m28.852s

# Additional models downloaded on first semantic query:
# - Qwen3-1.7B-Q8_0.gguf: 2.17 GB (query expansion)
# - qwen3-reranker-0.6b-q8_0.gguf: 639 MB (reranking)
# Total model size: ~3.14 GB
```

### PageIndex
```bash
$ pip install -r requirements.txt
# Time: ~10s

$ python run_pageindex.py --md_path AGENTS.md
# Time: 6.3s per file (OpenAI API calls for summaries)
# Cost: ~$0.01-0.05 per file
```

## Query Latency Benchmarks

### mq

```bash
# Directory tree
$ time ./mq /tmp/langchain-bench .tree
# Latency: 13ms

# Directory tree with previews
$ time ./mq /tmp/langchain-bench '.tree("full")'
# Latency: 15ms

# Search
$ time ./mq /tmp/langchain-bench '.search("commit standards")'
# Latency: 22ms

# Section extraction
$ time ./mq /tmp/langchain-bench/AGENTS.md '.section("Commit standards") | .text'
# Latency: 3ms
```

### qmd

```bash
# BM25 search (keyword-based)
$ time qmd search "commit standards" -n 3
# Latency: 154ms

# Vector search (semantic) - requires CPU LLM inference
$ time qmd vsearch "authentication" -n 3
# Latency: 74.29s (on CPU, no GPU detected)

# Hybrid query with reranking
$ time qmd query "documentation standards" -n 3
# Latency: 76.46s (on CPU, no GPU detected)
```

**Note**: qmd's semantic search runs LLM inference locally. Without GPU, latency is 60-80s. With GPU, expected ~1-5s.

### PageIndex

```bash
$ time python run_pageindex.py --md_path /tmp/langchain-bench/AGENTS.md
Processing markdown file...
Extracting nodes from markdown...
Generating summaries for each node...  # <-- OpenAI API call
Tree structure saved to: ./results/AGENTS_structure.json
# Latency: 6.317s
```

## Output Comparison

Query: Find information about "commit standards"

### mq output
```bash
$ ./mq /tmp/langchain-bench '.search("commit standards")'
```
```
Found 3 matches for "commit standards":

AGENTS.md:
  #### Commit standards (lines 73-84)
     "...Suggest PR titles that follow Conventional Commits format...."
CLAUDE.md:
  #### Commit standards (lines 73-84)
     "...Suggest PR titles that follow Conventional Commits format...."
```

### qmd output
```bash
$ qmd search "commit standards" -n 3
```
```
qmd://langchain/claude.md:73 #7b7a4b
Title: Global development guidelines for the LangChain monorepo
Score: 100%

@@ -72,4 @@ (71 before, 116 after)

#### Commit standards

Suggest PR titles that follow Conventional Commits format. Refer to
.github/workflows/pr_lint for allowed types and scopes...
```

### PageIndex output
```bash
$ python run_pageindex.py --md_path AGENTS.md
```
```json
{
  "title": "Commit standards",
  "node_id": "0005",
  "summary": "#### Commit standards\n\nSuggest PR titles that follow Conventional Commits format. Refer to .github/workflows/pr_lint for allowed types and scopes...",
  "line_num": 73
}
```

## Key Differences

| Aspect | mq | qmd | PageIndex |
|--------|-----|-----|-----------|
| **What it returns** | Structure + content | Ranked paths + scores | Tree + LLM summaries |
| **Who computes relevance** | Agent | System (embeddings) | System (LLM) |
| **Requires pre-indexing** | No | Yes | Yes |
| **Works offline** | Yes | Yes (after model download) | No (needs API) |
| **Deterministic** | Yes | No (embeddings) | No (LLM) |

## Conclusion

**mq** is designed for agentic exploration - it exposes document structure so LLM agents can reason over it directly. The agent already has semantic understanding; it doesn't need another embedding layer.

**qmd** is designed for semantic search - it computes relevance using embeddings and returns ranked results. Best for "find documents about topic X" queries.

**PageIndex** is designed for document understanding - it uses LLMs to generate summaries and structure. Best for complex PDFs where structure isn't explicit.

For AI agent workflows with markdown documents (which have explicit `#` structure), mq provides the fastest, cheapest, and most transparent interface.

---

Full logs available in:
- `benchmark/logs/mq_queries.log`
- `benchmark/logs/qmd_setup.log`
- `benchmark/logs/qmd_queries.log`
- `benchmark/logs/pageindex_queries.log`
