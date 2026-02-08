# mq Benchmark Results

Benchmarked on Apple M4, Go 1.24

## Context Window Analysis (200k tokens = 800KB)

For a single AI agent with 200k token context:

### Traditional Approach (load full text)

| Format | Docs per Context | Total Parse Time | Memory | Bottleneck |
|--------|------------------|------------------|--------|------------|
| PDF | 16 papers | 30s | ~800MB | Full text in context |
| Markdown | 16 docs (50KB) | 350ms | 432MB | Full text in context |

### mq Structure-First Approach

| Format | Docs per Context | Total Parse Time | Memory | Bottleneck |
|--------|------------------|------------------|--------|------------|
| PDF | **800 PDFs** | ~25min* | ~50MB | Python subprocess |
| Markdown | 80 docs (10KB) | 16ms | 22MB | None |
| HTML | 40 pages | 2.3s | ~200MB | Readability extraction |
| JSON | 800KB total | <1ms | 1MB | None |
| JSONL | 8000 lines | <1ms | 1.2MB | None |
| YAML | 800KB total | <1ms | 1MB | None |

*PDF parsing is 1.9s each, but structure output is only ~1KB per PDF.

**Key insight**: Structure-first approach enables **50x more PDFs** in context (800 vs 16) because you load ~1KB structure instead of ~50KB full text. The agent reasons over structure, then extracts only the sections it needs.

### PDF Structure-First vs PageIndex

| Approach | PDFs Searchable | Index Size | Cost | Build Time |
|----------|-----------------|------------|------|------------|
| Traditional (full text) | 16 | 50KB/PDF | $0 | - |
| **mq** (structure-first) | **800** | 1KB/PDF | $0 | 1.9s/PDF |
| PageIndex (LLM tree) | 266 | 3KB/PDF | $0.01-0.10/PDF | 6s/PDF |

mq wins on density (800 vs 266) because local extraction produces more compact structure than LLM-generated trees.

## Markdown Parsing

| Size | Time | Throughput | Memory | Allocs |
|------|------|------------|--------|--------|
| 1KB | 20µs | 73 MB/s | 40KB | 414 |
| 10KB | 141µs | 74 MB/s | 284KB | 2,685 |
| 100KB | 1.7ms | 59 MB/s | 2.8MB | 25,316 |
| 1MB | 17ms | 61 MB/s | 29MB | 255,109 |
| 10MB | 161ms | 65 MB/s | 288MB | 2,533,525 |

Memory overhead: ~27x document size

## HTML Parsing

| Size | Time | Throughput | Memory | Allocs |
|------|------|------------|--------|--------|
| 1KB | 511µs | 2.1 MB/s | 1.2MB | 8,098 |
| 10KB | 5.5ms | 1.9 MB/s | 12.9MB | 84,269 |
| 100KB | 67ms | 1.5 MB/s | 125MB | 810,830 |

HTML parsing includes Readability extraction (main content identification, DOM scoring, cleanup). Readability-focused benchmark (13KB article with nav/sidebar/footer noise): 1.2ms at 10.8 MB/s.

## YAML Parsing

| Size | Time | Throughput | Memory | Allocs |
|------|------|------------|--------|--------|
| 1KB | 64µs | 20 MB/s | 75KB | 1,218 |
| 10KB | 496µs | 21 MB/s | 537KB | 9,166 |
| 100KB | 5.7ms | 18 MB/s | 5.6MB | 89,706 |

YAML parsing uses gopkg.in/yaml.v3 then converts to mq document structure.

## JSON/JSONL Parsing

| Format | Size | Time | Throughput | Memory |
|--------|------|------|------------|--------|
| JSON | 1KB | 153ns | 7.1 GB/s | 1.4KB |
| JSON | 10KB | 882ns | 11.7 GB/s | 11KB |
| JSON | 100KB | 7.3µs | 14 GB/s | 107KB |
| JSON | 1MB | 52µs | 20 GB/s | 1MB |
| JSONL | 1KB | 278ns | 3.8 GB/s | 1.6KB |
| JSONL | 10KB | 1.8µs | 5.7 GB/s | 13KB |
| JSONL | 100KB | 17µs | 5.9 GB/s | 120KB |
| JSONL | 1MB | 133µs | 8 GB/s | 1.2MB |
| JSONL | 10MB | 1.1ms | 9.3 GB/s | 12MB |

JSON/JSONL parsing is significantly faster than Markdown due to simpler structure.

## Query Performance (after parsing)

| Query | 1KB | 10KB | 100KB | 1MB |
|-------|-----|------|-------|-----|
| GetHeadings | 69ns | 109ns | 1µs | 6µs |
| GetCodeBlocks | 21ns | 32ns | 277ns | 1.6µs |
| GetSection | 7ns | 7ns | 7ns | 7ns |
| ReadableText | 0.2ns | 0.2ns | 0.2ns | 0.2ns |

- GetSection and ReadableText are O(1) - pre-indexed/cached
- GetHeadings/GetCodeBlocks scale with count, not document size

## Tree Rendering

| Size | Compact | Preview | Full |
|------|---------|---------|------|
| 1KB | 457ns | 8.3µs | 8.3µs |
| 10KB | 2.8µs | 253µs | 253µs |
| 100KB | 31µs | 23ms | 23ms |

- Compact mode (headings only) is 50-700x faster than preview/full
- Preview and full modes have identical cost for single documents

## Search Performance

| Size | Time | Memory |
|------|------|--------|
| 1KB | 19µs | 36KB |
| 10KB | 323µs | 946KB |
| 100KB | 24ms | 81MB |

Search scans all sections for matching content. Scales linearly with document size.

## MQL Query Pipeline

End-to-end time for lex -> parse -> compile -> execute:

| Query | Time | Allocs |
|-------|------|--------|
| `.headings` | 327ns | 12 |
| `.sections` | 552ns | 10 |
| `.code("go")` | 354ns | 15 |
| `.section("X") \| .text` | 5.6µs | 21 |
| `.headings \| filter(.level == 2)` | 1.5µs | 27 |

MQL overhead is minimal. Simple selectors add ~300ns to the raw query time. Piped queries with text extraction are dominated by the text operation itself.

## Multi-Document Scale

| Documents | Doc Size | Total | Time | Memory |
|-----------|----------|-------|------|--------|
| 10 | 10KB | 100KB | 1.5ms | 2.8MB |
| 100 | 10KB | 1MB | 17ms | 28MB |
| 1000 | 10KB | 10MB | 167ms | 284MB |

Linear scaling. For 1GB RAM, can hold ~3500 10KB documents.

## Comparison Notes

- **vs jq**: JSON parsing is ~10x faster than jq for equivalent operations
- **vs grep**: Structure-aware queries impossible with grep; mq provides semantic access
- **Memory trade-off**: Higher memory usage enables O(1) queries after initial parse

## Raw Output

See `results.txt` for full `go test -bench` output.
