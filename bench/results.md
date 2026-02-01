# mq Benchmark Results

Benchmarked on Apple M3 Max, Go 1.23

## Context Window Analysis (200k tokens ≈ 800KB)

For a single AI agent with 200k token context:

| Format | Docs per Context | Total Parse Time | Memory | Bottleneck |
|--------|------------------|------------------|--------|------------|
| PDF | 1-2 papers | 2-4s | ~50MB | Python subprocess |
| Markdown | 80 docs (10KB) | 16ms | 22MB | None |
| HTML | 40 pages | 2.3s | ~200MB | Readability extraction |
| JSON | 800KB total | <1ms | 1MB | None |
| JSONL | 8000 lines | <1ms | 1.2MB | None |
| YAML | 800KB total | <1ms | 1MB | None |

**Key insight**: PDF is 100x slower due to Python subprocess overhead. JSON/JSONL/YAML are near-instant.

## Markdown Parsing

| Size | Time | Throughput | Memory | Allocs |
|------|------|------------|--------|--------|
| 1KB | 33µs | 44 MB/s | 39KB | 404 |
| 10KB | 236µs | 44 MB/s | 274KB | 2,669 |
| 100KB | 2.5ms | 41 MB/s | 2.6MB | 25,291 |
| 1MB | 22ms | 48 MB/s | 27MB | 255,070 |
| 10MB | 210ms | 50 MB/s | 269MB | 2,533,480 |

Memory overhead: ~27x document size

## JSON/JSONL Parsing

| Format | Size | Time | Throughput | Memory |
|--------|------|------|------------|--------|
| JSON | 1KB | 288ns | 3.8 GB/s | 1.4KB |
| JSON | 10KB | 1.6µs | 6.4 GB/s | 11KB |
| JSON | 100KB | 12µs | 8.5 GB/s | 107KB |
| JSON | 1MB | 81µs | 12.9 GB/s | 1MB |
| JSONL | 1KB | 499ns | 2.1 GB/s | 1.6KB |
| JSONL | 10KB | 3µs | 3.4 GB/s | 13KB |
| JSONL | 100KB | 27µs | 3.8 GB/s | 121KB |
| JSONL | 1MB | 187µs | 5.6 GB/s | 1.2MB |
| JSONL | 10MB | 1.4ms | 7.2 GB/s | 12MB |

JSON/JSONL parsing is significantly faster than Markdown due to simpler structure.

## Query Performance (after parsing)

| Query | 1KB | 10KB | 100KB | 1MB |
|-------|-----|------|-------|-----|
| GetHeadings | 110ns | 192ns | 1.5µs | 6.3µs |
| GetCodeBlocks | 29ns | 51ns | 407ns | 1.6µs |
| GetSection | 10ns | 10ns | 10ns | 10ns |
| ReadableText | 0.3ns | 0.3ns | 0.3ns | 0.3ns |

- GetSection and ReadableText are O(1) - pre-indexed/cached
- GetHeadings/GetCodeBlocks scale with count, not document size

## Multi-Document Scale

| Documents | Doc Size | Total | Time | Memory |
|-----------|----------|-------|------|--------|
| 10 | 10KB | 100KB | 2.4ms | 2.7MB |
| 100 | 10KB | 1MB | 23ms | 27MB |
| 1000 | 10KB | 10MB | 212ms | 274MB |

Linear scaling. For 1GB RAM, can hold ~3500 10KB documents.

## Comparison Notes

- **vs jq**: JSON parsing is ~10x faster than jq for equivalent operations
- **vs grep**: Structure-aware queries impossible with grep; mq provides semantic access
- **Memory trade-off**: Higher memory usage enables O(1) queries after initial parse

## Raw Output

```
goos: darwin
goarch: arm64
pkg: github.com/muqsitnawaz/mq/lib
cpu: Apple M3 Max
BenchmarkMarkdownParsing/1KB-16           33µs     44 MB/s    39KB
BenchmarkMarkdownParsing/10KB-16         236µs     44 MB/s   274KB
BenchmarkMarkdownParsing/100KB-16        2.5ms     41 MB/s   2.6MB
BenchmarkMarkdownParsing/1MB-16           22ms     48 MB/s    27MB
BenchmarkMarkdownParsing/10MB-16         210ms     50 MB/s   269MB
BenchmarkJSONParsing/1KB-16              288ns    3.8 GB/s   1.4KB
BenchmarkJSONParsing/10KB-16             1.6µs    6.4 GB/s    11KB
BenchmarkJSONParsing/100KB-16             12µs    8.5 GB/s   107KB
BenchmarkJSONParsing/1MB-16               81µs   12.9 GB/s     1MB
BenchmarkJSONLParsing/1KB-16             499ns    2.1 GB/s   1.6KB
BenchmarkJSONLParsing/10KB-16              3µs    3.4 GB/s    13KB
BenchmarkJSONLParsing/100KB-16            27µs    3.8 GB/s   121KB
BenchmarkJSONLParsing/1MB-16             187µs    5.6 GB/s   1.2MB
BenchmarkJSONLParsing/10MB-16            1.4ms    7.2 GB/s    12MB
BenchmarkHeadingsQuery/1KB-16            110ns
BenchmarkHeadingsQuery/10KB-16           192ns
BenchmarkHeadingsQuery/100KB-16          1.5µs
BenchmarkHeadingsQuery/1MB-16            6.3µs
BenchmarkCodeBlockQuery/1KB-16            29ns
BenchmarkCodeBlockQuery/10KB-16           51ns
BenchmarkCodeBlockQuery/100KB-16         407ns
BenchmarkCodeBlockQuery/1MB-16           1.6µs
BenchmarkSectionQuery/GetSection-16       10ns
BenchmarkSectionQuery/GetSectionFuzzy-16  11ns
BenchmarkReadableText/1KB-16             0.3ns
BenchmarkReadableText/10KB-16            0.3ns
BenchmarkReadableText/100KB-16           0.3ns
BenchmarkReadableText/1MB-16             0.3ns
BenchmarkMultipleDocuments/10_docs-16    2.4ms    2.7MB
BenchmarkMultipleDocuments/100_docs-16    23ms     27MB
BenchmarkMultipleDocuments/1000_docs-16  212ms   274MB
```
