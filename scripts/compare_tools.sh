#!/bin/bash
# Compare mq vs qmd vs PageIndex on the same markdown corpus
#
# Measures:
# - Setup time and cost
# - Query latency
# - Output characteristics
#
# Requirements:
# - mq binary built
# - qmd installed (bun install -g https://github.com/tobi/qmd)
# - PageIndex cloned at .refs/PageIndex with deps installed
# - OPENAI_API_KEY set (for PageIndex)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
REPO_PATH="/tmp/langchain-bench"
MQ_BIN="$PROJECT_DIR/mq"
PAGEINDEX_DIR="$PROJECT_DIR/.refs/PageIndex"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${GREEN}[compare]${NC} $1"; }
section() { echo -e "\n${BLUE}=== $1 ===${NC}"; }

# Clone test corpus if needed
setup_corpus() {
    if [ ! -d "$REPO_PATH" ]; then
        log "Cloning LangChain repo for benchmark..."
        git clone --depth 1 https://github.com/langchain-ai/langchain "$REPO_PATH" 2>/dev/null
    fi

    # Count markdown files
    MD_COUNT=$(find "$REPO_PATH" -name "*.md" | wc -l)
    MD_LINES=$(find "$REPO_PATH" -name "*.md" -exec cat {} \; | wc -l)
    log "Corpus: $MD_COUNT markdown files, $MD_LINES total lines"
}

# Benchmark mq
bench_mq() {
    section "mq"

    echo "Setup: None required (single binary)"
    echo "Dependencies: None"
    echo ""

    # Query latency
    echo "Query latency:"

    echo -n "  .tree on directory:     "
    time_ms=$(( $(date +%s%N) ))
    "$MQ_BIN" "$REPO_PATH" .tree > /dev/null 2>&1
    time_ms=$(( ($(date +%s%N) - time_ms) / 1000000 ))
    echo "${time_ms}ms"

    echo -n "  .tree(\"full\") on dir:   "
    time_ms=$(( $(date +%s%N) ))
    "$MQ_BIN" "$REPO_PATH" '.tree("full")' > /dev/null 2>&1
    time_ms=$(( ($(date +%s%N) - time_ms) / 1000000 ))
    echo "${time_ms}ms"

    echo -n "  .search(\"OAuth\"):       "
    time_ms=$(( $(date +%s%N) ))
    "$MQ_BIN" "$REPO_PATH" '.search("OAuth")' > /dev/null 2>&1
    time_ms=$(( ($(date +%s%N) - time_ms) / 1000000 ))
    echo "${time_ms}ms"

    echo -n "  .section() | .text:     "
    time_ms=$(( $(date +%s%N) ))
    "$MQ_BIN" "$REPO_PATH/CONTRIBUTING.md" '.section("Getting Started") | .text' > /dev/null 2>&1
    time_ms=$(( ($(date +%s%N) - time_ms) / 1000000 ))
    echo "${time_ms}ms"

    echo ""
    echo "Cost per query: \$0"
}

# Benchmark qmd
bench_qmd() {
    section "qmd"

    if ! command -v qmd &> /dev/null; then
        echo "qmd not installed. Install with: bun install -g https://github.com/tobi/qmd"
        echo "Skipping qmd benchmark."
        return
    fi

    QMD_CACHE="$HOME/.cache/qmd"

    echo "Setup required:"
    echo "  1. qmd collection add <path> --name bench"
    echo "  2. qmd embed (downloads ~3GB models, generates embeddings)"
    echo ""
    echo "Dependencies: Bun, SQLite with extensions, node-llama-cpp"
    echo "Model sizes: embeddinggemma (~300MB) + qwen3-reranker (~640MB) + Qwen3-1.7B (~2.2GB)"
    echo ""

    # Check if already indexed
    if qmd collection list 2>/dev/null | grep -q "langchain"; then
        echo "Existing collection found. Measuring query latency..."

        echo ""
        echo "Query latency:"

        echo -n "  search (BM25):          "
        time_ms=$(( $(date +%s%N) ))
        qmd search "OAuth" -n 5 > /dev/null 2>&1
        time_ms=$(( ($(date +%s%N) - time_ms) / 1000000 ))
        echo "${time_ms}ms"

        echo -n "  vsearch (vector):       "
        time_ms=$(( $(date +%s%N) ))
        qmd vsearch "authentication flow" -n 5 > /dev/null 2>&1
        time_ms=$(( ($(date +%s%N) - time_ms) / 1000000 ))
        echo "${time_ms}ms"

        echo -n "  query (hybrid+rerank):  "
        time_ms=$(( $(date +%s%N) ))
        qmd query "how to authenticate" -n 5 > /dev/null 2>&1
        time_ms=$(( ($(date +%s%N) - time_ms) / 1000000 ))
        echo "${time_ms}ms"
    else
        echo "No collection indexed. To benchmark qmd:"
        echo "  qmd collection add $REPO_PATH --name langchain"
        echo "  qmd embed"
        echo "  # Then re-run this script"
    fi

    echo ""
    echo "Cost per query: \$0 (local compute)"
}

# Benchmark PageIndex
bench_pageindex() {
    section "PageIndex"

    if [ ! -d "$PAGEINDEX_DIR" ]; then
        echo "PageIndex not found at $PAGEINDEX_DIR"
        echo "Skipping PageIndex benchmark."
        return
    fi

    echo "Setup required:"
    echo "  1. pip install -r requirements.txt"
    echo "  2. Set OPENAI_API_KEY"
    echo "  3. python run_pageindex.py --md_path <file> (for each file)"
    echo ""
    echo "Dependencies: Python, OpenAI API"
    echo ""

    if [ -z "$OPENAI_API_KEY" ] && [ -z "$CHATGPT_API_KEY" ]; then
        echo "OPENAI_API_KEY not set. Cannot benchmark PageIndex."
        echo ""
        echo "To benchmark PageIndex:"
        echo "  export OPENAI_API_KEY=your_key"
        echo "  # Then re-run this script"
        return
    fi

    echo "Tree generation cost: ~\$0.01-0.05 per document (GPT-4o API)"
    echo "Query cost: ~\$0.01-0.10 per query (LLM reasoning over tree)"
    echo ""
    echo "Note: PageIndex is designed for PDFs. For markdown, it parses # headers"
    echo "similar to mq, but adds LLM-generated summaries."
}

# Summary comparison
summary() {
    section "Summary Comparison"

    cat << 'EOF'
| Metric              | mq           | qmd                  | PageIndex           |
|---------------------|--------------|----------------------|---------------------|
| Setup time          | 0            | ~5-30 min (embed)    | ~1-5 min/doc (API)  |
| Setup cost          | $0           | $0 (local)           | ~$0.01-0.05/doc     |
| Dependencies        | None         | 3GB models, Bun, SQL | Python, OpenAI API  |
| Query latency       | <100ms       | 100-2000ms           | 1-5s (API call)     |
| Query cost          | $0           | $0 (local)           | ~$0.01-0.10/query   |
| Output              | Structure    | Ranked files+scores  | Node references     |
| Agent computes      | Yes          | No (system ranks)    | No (LLM reasons)    |
EOF
}

# Main
main() {
    log "Setting up test corpus..."
    setup_corpus

    bench_mq
    bench_qmd
    bench_pageindex
    summary
}

main "$@"
