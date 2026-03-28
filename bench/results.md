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
| Code parse (Go, 50 lines) | cold: 1.26ms, cache: 9.7ms |
| Code parse (Go, 1200 lines) | cold: 377ms, cache: 9.9ms |
| Code parse (Python, 29 lines) | 2.1ms |
| Code parse (TypeScript, 48 lines) | 3.6ms |
| Code parse (Rust, 48 lines) | 2.2ms |
| CSV parse (8 rows x 5 cols) | 4.7us |
| DOCX parse (10 paragraphs) | 33us |
| XLSX parse (100 rows x 2 cols) | 859us |
| PPTX parse (5 slides) | 90us |

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

## Code Parsing (tree-sitter, pure Go)

`go test ./code -bench 'Benchmark(ParseGo|ParsePython|ParseTypeScript|ParseRust|ParseLargeGo)$' -run '^$' -benchmem -count=3`

| Language | File size | Cold parse | Allocs | Throughput |
|----------|-----------|------------|--------|------------|
| Go | 50 lines | 1.26ms | 6K | ~40 KB/s |
| Python | 29 lines | 2.1ms | 11K | ~14 KB/s |
| TypeScript | 48 lines | 3.6ms | 19K | ~13 KB/s |
| Rust | 48 lines | 2.2ms | 12K | ~22 KB/s |
| Go (large) | 1200 lines (36KB) | 377ms | 12K | ~95 KB/s |

Tree-sitter parsing is CPU-bound. The large file (lib/tree.go, 36KB) takes 377ms cold. TypeScript is the slowest per-line due to grammar complexity (JSX, generics, optional chaining).

### Code: Cold Parse vs Cache Hit

Real-world single-file measurements (CLI invocation, not synthetic loop):

| File | Cold parse | Warm cache hit | Speedup |
|------|------------|----------------|---------|
| Go, 745 lines | 360ms | 33ms | **11x** |
| Go, 2808 lines | 13.9s | 33ms | **420x** |

Cache hit latency is ~33ms per file (bbolt stat check + msgpack deserialize). For files over ~200 lines, cache wins. For small files, cold parse is comparable or faster.

## Office Document Parsing

`go test ./office -bench . -run '^$' -benchmem -count=3`

| Format | Input size | Parse time | Memory | Allocs |
|--------|-----------|------------|--------|--------|
| CSV | 8 rows x 5 cols | 4.7us | 10KB | 74 |
| DOCX | 10 paragraphs | 33us | 25KB | 424 |
| PPTX | 5 slides | 90us | 53KB | 978 |
| XLSX | 100 rows x 2 cols | 859us | 651KB | 8.5K |

CSV and DOCX are stdlib-only parsers (encoding/csv, archive/zip + encoding/xml). XLSX uses excelize which is heavier. All office formats parse under 1ms except XLSX at larger sizes.

## Real-World PDF Directory Benchmark

Tested on a corpus of 123 PDFs, 365MB total, 317K lines.

| Query | Files | Cold | Warm (cached) | Speedup |
|-------|-------|------|---------------|---------|
| `.tree` | 9 | 24.5s | 0.25s | **98x** |
| `.tree` | 29 | 2:24 | 0.62s | **233x** |
| `.tree` | 123 | 5:02 | 2.97s | **101x** |

## Real-World Code Directory Benchmarks

Tested on a mixed codebase with 560 source files and 50 markdown files, 111K total lines.

### Single File Cold vs Warm

| File | Lines | Cold | Warm | Speedup |
|------|-------|------|------|---------|
| 745-line source file | 745 | 360ms | 33ms | **11x** |
| 2808-line source file | 2808 | 13.9s | 33ms | **420x** |

### Directory Tree

| Query | Scope | Cold | Warm | Speedup |
|-------|-------|------|------|---------|
| `.tree \| depth(1)` | root (12 files) | 1.1s | 1.0s | ~1x (only root files parsed) |
| `.tree \| depth(1)` | subdirectory (60 files, 25K lines) | 94s | **0.54s** | **174x** |

### Key Findings

- Cache speedup is dramatic: **174x-420x** for warm lookups
- `depth(1)` is always fast (~1s) since it only parses root-level files
- The first `mq` invocation on a codebase pays the cold parse cost; every subsequent run is sub-second

## Notes

- The biggest user-visible win right now is PDF cache latency: repeated loads drop from roughly 11-13 seconds to roughly 11-17 milliseconds.
- On very large directories, warm exact-repeat search is much faster than cold search, but it still pays the directory-hash check first.
- Markdown is still the highest-throughput general parser path in the repo.
- HTML remains slower because readability extraction does more work than plain structural parsing.
- Code parsing via tree-sitter benefits enormously from cache: 174x-420x speedup on real directories.
- Office formats are all sub-millisecond except XLSX which carries the excelize dependency weight.
- CSV is by far the fastest parser at 4.7us -- stdlib encoding/csv is very efficient.
- Binary size increased from 8MB to 34MB due to gotreesitter embedding 206 grammars. Can be reduced with `GOTREESITTER_GRAMMAR_SET` env var at build time.

## Raw Output

See `results.txt` for historical raw benchmark output. The tables above are the current measured release numbers.
