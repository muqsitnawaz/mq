# mq Benchmark Results

Benchmarked on Apple M3 Max, Go 1.24, on 2026-03-27.

## Headline Summary

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
| Directory search (`private-manuscript`) | cold: 2.21s, warm exact-repeat: 11.98ms, partial invalidation: 1.62s |
| Directory search (`rush-sessions`) | cold: 51.86s, warm exact-repeat: 440.34ms |
| MQL `.section("X") \| .text` | 9.58us after parse |

## Markdown Parsing

`go test ./lib -bench 'BenchmarkMarkdownParsing$' -run '^$' -count=1`

| Size | Time | Throughput |
|------|------|------------|
| 1KB | 38.30us | 37.99 MB/s |
| 10KB | 244.58us | 42.69 MB/s |
| 100KB | 2.70ms | 37.92 MB/s |
| 1MB | 23.48ms | 44.66 MB/s |
| 10MB | 224.74ms | 46.66 MB/s |

## HTML Parsing

`go test ./html -bench 'BenchmarkHTMLParsing$' -run '^$' -count=1`

| Size | Time | Throughput |
|------|------|------------|
| 1KB | 0.98ms | 1.09 MB/s |
| 10KB | 10.63ms | 1.01 MB/s |
| 100KB | 157.77ms | 0.65 MB/s |

HTML includes readability-style main-content extraction, so these numbers cover DOM parse plus content selection.

## YAML Parsing

`go test ./data -bench 'BenchmarkYAMLParsing$' -run '^$' -count=1`

| Size | Time | Throughput |
|------|------|------------|
| 1KB | 0.12ms | 10.63 MB/s |
| 10KB | 0.88ms | 11.65 MB/s |
| 100KB | 12.39ms | 8.28 MB/s |

## PDF Parse / Cache / Query

`go test ./pdf -bench . -run '^$' -count=1`

| File | Size | Cold parse | Warm cache hit | BuildTree | Search |
|------|------|------------|----------------|-----------|--------|
| `bert.pdf` | 757KB | 13.25s | 16.68ms | 0.377ms | 0.973ms |
| `attention.pdf` | 2.1MB | 10.86s | 11.16ms | 0.567ms | 0.845ms |
| `raft.pdf` | 6.6MB | 13.42s | 12.00ms | 0.216ms | 0.754ms |

Cold parse covers the full PDF pipeline: text extraction, structure extraction, normalization, and section building. Warm cache hit measures `Cache.LookupFile`, which validates the file, then deserializes the cached `Document`.

## Core Query Performance

`go test ./lib -bench 'Benchmark(HeadingsQuery|CodeBlockQuery|SectionQuery|ReadableText)$' -run '^$' -count=1`

| Query | Current result | Notes |
|-------|----------------|-------|
| `GetSection` | 9.2ns | Exact title lookup |
| `GetSectionFuzzy` | 10.5ns | Fuzzy title lookup |
| `ReadableText` | 0.28ns | Cached string access |
| `GetHeadings` | 0.14us (1KB) to 8.34us (1MB) | Scales with heading count |
| `GetCodeBlocks` | 28ns (1KB) to 1.86us (1MB) | Scales with code block count |

## MQL Query Pipeline

`go test ./mql -bench 'BenchmarkMQLQuery$' -run '^$' -count=1`

| Query | Time |
|-------|------|
| `.headings` | 0.55us |
| `.sections` | 0.27us |
| `.code("go")` | 0.56us |
| `.section("Section 1") \| .text` | 9.58us |
| `.headings \| filter(.level == 2)` | 3.18us |

## Directory Search Cache

`go test ./mql -bench 'BenchmarkDirectorySearch$' -run '^$' -benchtime=1x -count=1`

| Corpus | Cold | Warm exact repeat | Partial invalidation |
|--------|------|-------------------|----------------------|
| `private-manuscript` (185 files, 178 Markdown docs, 65,175 Markdown lines) | 2.21s | 11.98ms | 1.62s |
| `~/.rush/sessions` | 51.86s | 440.34ms | - |

These benchmarks hit the real cache-aware directory search path through `mql.Engine.SearchDir`, not the older markdown-only `doc.Search(...)` helper benchmark.

## Multi-Document Scale

`go test ./lib -bench 'BenchmarkMultipleDocuments$' -run '^$' -count=1`

| Documents | Doc Size | Total Corpus | Time |
|-----------|----------|--------------|------|
| 10 | 10KB | 100KB | 2.93ms |
| 100 | 10KB | 1MB | 24.17ms |
| 1000 | 10KB | 10MB | 222.97ms |

This benchmark parses a batch of same-sized markdown documents, then queries all of them for headings. It is a real repo benchmark, but it does not measure directory cache behavior.

## Notes

- The biggest user-visible win right now is PDF cache latency: repeated loads drop from roughly 11-13 seconds to roughly 11-17 milliseconds.
- On very large directories, warm exact-repeat search is much faster than cold search, but it still pays the directory-hash check first.
- Markdown is still the highest-throughput general parser path in the repo.
- HTML remains slower because readability extraction does more work than plain structural parsing.

## Raw Output

See `results.txt` for historical raw benchmark output. The tables above are the current measured release numbers.
