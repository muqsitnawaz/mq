#!/bin/bash
# mq Benchmark: Compare agent performance with and without mq
#
# Usage: ./scripts/bench.sh [question_id]
#   question_id: optional, run only specific question (q1, q2, etc.)
#
# Requirements:
# - claude CLI installed
# - mq binary built (go build .)
# - jq for JSON parsing

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
REPO_PATH="/tmp/langchain-bench"
RESULTS_DIR="$SCRIPT_DIR/bench_results"
QUESTIONS_FILE="$SCRIPT_DIR/bench_questions.json"
MQ_BIN="$PROJECT_DIR/mq"
CLAUDE_SESSIONS_DIR="$HOME/.claude/projects"
# File to track our session ID mappings
SESSION_MAP_FILE="$RESULTS_DIR/session_map.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[bench]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[warn]${NC} $1"
}

error() {
    echo -e "${RED}[error]${NC} $1"
    exit 1
}

# Check dependencies
check_deps() {
    command -v claude >/dev/null 2>&1 || error "claude CLI not found. Install with: npm install -g @anthropic-ai/claude-code"
    command -v jq >/dev/null 2>&1 || error "jq not found. Install with: apt install jq"
    [ -f "$MQ_BIN" ] || error "mq binary not found. Build with: go build ."
    [ -f "$QUESTIONS_FILE" ] || error "Questions file not found: $QUESTIONS_FILE"
}

# Clone test repo if needed
setup_repo() {
    if [ ! -d "$REPO_PATH" ]; then
        log "Cloning langchain repo..."
        git clone --depth 1 https://github.com/langchain-ai/langchain.git "$REPO_PATH"
    else
        log "Using existing repo at $REPO_PATH"
    fi
}

# Temporarily hide CLAUDE.md to force agents to search
hide_claude_md() {
    if [ -f "$REPO_PATH/CLAUDE.md" ]; then
        mv "$REPO_PATH/CLAUDE.md" "$REPO_PATH/.CLAUDE.md.bak"
        log "Temporarily hidden CLAUDE.md"
    fi
}

# Restore CLAUDE.md after benchmark
restore_claude_md() {
    if [ -f "$REPO_PATH/.CLAUDE.md.bak" ]; then
        mv "$REPO_PATH/.CLAUDE.md.bak" "$REPO_PATH/CLAUDE.md"
        log "Restored CLAUDE.md"
    fi
}

# Find session file by session ID
find_session_file() {
    local session_id="$1"
    # Session files are stored as .jsonl (JSON Lines) in project directories
    local session_file=""

    # Look in projects directory for .jsonl files
    session_file=$(find "$CLAUDE_SESSIONS_DIR" -name "${session_id}.jsonl" -type f 2>/dev/null | head -1)

    if [ -z "$session_file" ]; then
        # Also check .claude directly
        session_file=$(find "$HOME/.claude" -name "${session_id}.jsonl" -type f 2>/dev/null | head -1)
    fi

    echo "$session_file"
}

# Extract metrics from session file (JSONL format)
extract_session_metrics() {
    local session_file="$1"
    local output_file="$2"

    if [ ! -f "$session_file" ]; then
        warn "Session file not found"
        echo "0" > "${output_file%.txt}.input_tokens"
        echo "0" > "${output_file%.txt}.output_tokens"
        echo "0" > "${output_file%.txt}.cache_read_tokens"
        return
    fi

    # Session files are JSONL (one JSON object per line)
    # Extract from assistant messages: input_tokens, output_tokens, cache_read_input_tokens
    local input_tokens=$(cat "$session_file" | jq -s '[.[] | select(.type == "assistant") | .message.usage.input_tokens // 0] | add // 0' 2>/dev/null || echo "0")
    local output_tokens=$(cat "$session_file" | jq -s '[.[] | select(.type == "assistant") | .message.usage.output_tokens // 0] | add // 0' 2>/dev/null || echo "0")
    local cache_read_tokens=$(cat "$session_file" | jq -s '[.[] | select(.type == "assistant") | .message.usage.cache_read_input_tokens // 0] | add // 0' 2>/dev/null || echo "0")
    local cache_creation_tokens=$(cat "$session_file" | jq -s '[.[] | select(.type == "assistant") | .message.usage.cache_creation_input_tokens // 0] | add // 0' 2>/dev/null || echo "0")

    # Total tokens consumed = input + cache_read + cache_creation
    local total_input=$((input_tokens + cache_read_tokens + cache_creation_tokens))

    echo "$total_input" > "${output_file%.txt}.input_tokens"
    echo "$output_tokens" > "${output_file%.txt}.output_tokens"
    echo "$cache_read_tokens" > "${output_file%.txt}.cache_read_tokens"

    log "  Metrics: input=${total_input} tokens (cache_read=${cache_read_tokens}), output=${output_tokens} tokens"
}

# Run a single benchmark
run_benchmark() {
    local question_id="$1"
    local mode="$2"  # "without_mq" or "with_mq"
    local question="$3"

    local prompt_file="$SCRIPT_DIR/bench_prompts/${mode}.txt"
    local output_dir="$RESULTS_DIR/$mode"
    local output_file="$output_dir/${question_id}.txt"
    # Generate a proper UUID for session ID
    local session_id=$(uuidgen 2>/dev/null || cat /proc/sys/kernel/random/uuid)

    mkdir -p "$output_dir"

    # Save session ID for later lookup
    echo "$session_id" > "$output_dir/${question_id}.session_id"

    # Substitute variables in prompt
    local prompt
    prompt=$(cat "$prompt_file")
    prompt="${prompt//\{\{REPO_PATH\}\}/$REPO_PATH}"
    prompt="${prompt//\{\{QUESTION\}\}/$question}"
    prompt="${prompt//\{\{MQ_BIN\}\}/$MQ_BIN}"

    log "Running $question_id ($mode) [session: $session_id]..."

    # Run claude with session ID and capture output
    local start_time=$(date +%s.%N)

    # Use claude CLI with --session-id to track metrics
    cd "$REPO_PATH"
    claude --session-id "$session_id" --print --allowedTools "Bash,Glob,Grep,Read" -p "$prompt" > "$output_file" 2>&1 || true
    cd "$PROJECT_DIR"

    local end_time=$(date +%s.%N)
    local wall_time=$(echo "$end_time - $start_time" | bc)

    echo "$wall_time" > "$output_dir/${question_id}.wall_time"

    # Wait a moment for session file to be written
    sleep 1

    # Find and parse session file for real metrics
    local session_file
    session_file=$(find_session_file "$session_id")

    if [ -n "$session_file" ]; then
        log "  Found session: $session_file"
        extract_session_metrics "$session_file" "$output_file"
    else
        warn "  Session file not found for $session_id"
        echo "0" > "${output_file%.txt}.input_tokens"
        echo "0" > "${output_file%.txt}.output_tokens"
        echo "0" > "${output_file%.txt}.duration_ms"
    fi

    log "  Wall time: ${wall_time}s"
}

# Run all benchmarks for a question
run_question() {
    local question_id="$1"
    local question
    question=$(jq -r ".[] | select(.id == \"$question_id\") | .question" "$QUESTIONS_FILE")

    if [ -z "$question" ] || [ "$question" == "null" ]; then
        error "Question not found: $question_id"
    fi

    log "Question: $question"
    echo ""

    run_benchmark "$question_id" "without_mq" "$question"
    run_benchmark "$question_id" "with_mq" "$question"
}

# Generate summary report
generate_summary() {
    local summary_file="$RESULTS_DIR/summary.md"

    log "Generating summary..."

    cat > "$summary_file" << 'EOF'
# mq Benchmark Results

Comparing agent performance with and without mq tool.

## Results

| Question | Mode | Input Tokens | Output Tokens | Wall Time (s) |
|----------|------|--------------|---------------|---------------|
EOF

    local total_without_input=0
    local total_with_input=0
    local total_without_output=0
    local total_with_output=0
    local count=0

    # Parse results for each question
    for qfile in "$RESULTS_DIR/without_mq"/*.txt; do
        [ -f "$qfile" ] || continue
        local qid=$(basename "$qfile" .txt)

        # Without mq metrics
        local without_input=$(cat "$RESULTS_DIR/without_mq/${qid}.input_tokens" 2>/dev/null || echo "0")
        local without_output=$(cat "$RESULTS_DIR/without_mq/${qid}.output_tokens" 2>/dev/null || echo "0")
        local without_wall=$(cat "$RESULTS_DIR/without_mq/${qid}.wall_time" 2>/dev/null | cut -d. -f1 || echo "0")

        # With mq metrics
        local with_input=$(cat "$RESULTS_DIR/with_mq/${qid}.input_tokens" 2>/dev/null || echo "0")
        local with_output=$(cat "$RESULTS_DIR/with_mq/${qid}.output_tokens" 2>/dev/null || echo "0")
        local with_wall=$(cat "$RESULTS_DIR/with_mq/${qid}.wall_time" 2>/dev/null | cut -d. -f1 || echo "0")

        echo "| $qid | without_mq | $without_input | $without_output | $without_wall |" >> "$summary_file"
        echo "| $qid | with_mq | $with_input | $with_output | $with_wall |" >> "$summary_file"

        # Accumulate totals for averages
        total_without_input=$((total_without_input + without_input))
        total_with_input=$((total_with_input + with_input))
        total_without_output=$((total_without_output + without_output))
        total_with_output=$((total_with_output + with_output))
        count=$((count + 1))
    done

    # Calculate savings
    if [ $count -gt 0 ] && [ $total_without_input -gt 0 ]; then
        local input_reduction=$(echo "scale=1; (1 - $total_with_input / $total_without_input) * 100" | bc 2>/dev/null || echo "N/A")
        local output_reduction=$(echo "scale=1; (1 - $total_with_output / $total_without_output) * 100" | bc 2>/dev/null || echo "N/A")

        cat >> "$summary_file" << EOF

## Summary

- **Questions tested**: $count
- **Total input tokens (without mq)**: $total_without_input
- **Total input tokens (with mq)**: $total_with_input
- **Input token reduction**: ${input_reduction}%
- **Total output tokens (without mq)**: $total_without_output
- **Total output tokens (with mq)**: $total_with_output
- **Output token reduction**: ${output_reduction}%

## Interpretation

Input tokens represent the context consumed by the agent (files read, tool outputs).
A reduction in input tokens means mq helped the agent be more efficient by reading
only the relevant sections instead of entire files.
EOF
    else
        cat >> "$summary_file" << 'EOF'

## Summary

No valid results to summarize. Run the benchmark first.
EOF
    fi

    log "Summary written to $summary_file"
}

# Main
main() {
    check_deps
    setup_repo

    # Hide CLAUDE.md to force agents to search (restore on exit)
    trap restore_claude_md EXIT
    hide_claude_md

    mkdir -p "$RESULTS_DIR"

    if [ -n "$1" ]; then
        # Run specific question
        run_question "$1"
    else
        # Run all questions
        local question_ids
        question_ids=$(jq -r '.[].id' "$QUESTIONS_FILE")

        for qid in $question_ids; do
            run_question "$qid"
            echo ""
        done
    fi

    generate_summary

    log "Benchmark complete!"
    echo ""
    cat "$RESULTS_DIR/summary.md"
}

main "$@"
