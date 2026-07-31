# Changelog

## [0.4.0] - 2026-07-31

### Added

- **`--trim <N><unit>`**: control the snippet size under each heading in the file
  tree. Unit suffix — `L` lines, `P` paragraphs, `C` chars, `S` sentences, `W`
  words (e.g. `--trim 4L`, `--trim 3P`, `--trim 200C`). Default is an adaptive
  `2L`. A negative `N` keeps the tail (`--trim -3L`, like `tail`); `--trim 0`
  shows headings only.
- **Remainder marker**: truncated snippets now show a `…+N` marker of how much
  more there is, in the same unit as `--trim` (e.g. `…+3P`). `--more off` hides it.
- **`--full`** (whole section bodies) and **`--bare`** (headings only, no snippets
  or asset index).
- **`--depth N`**, **`--only <list>`**, **`--drop <list>`**: filter the heading
  spine by level and/or element kind. Hidden headings reparent their descendants
  so the tree stays connected. Tokens: `h1`..`h6` and
  `images, figures, links, tables, lists, code, svg, media`.
- **Asset index footer** for HTML/PDF (on by default, `--no-assets` to hide):
  a compact summary of images, figures, tables, lists, code (by language), links
  (external vs relative), inline SVG (counted with approximate size, never
  dumped), and skipped media (`video/audio/canvas/iframe/embed/object`).
- **HTML `<figure>`, inline `<svg>`, and media elements** are now captured or
  counted instead of silently dropped.

## [0.3.9] - 2026-03-29

### Documentation

- **PDF directory benchmark**: Tested on 123 real PDFs (365MB) — arXiv papers, NIST reports, OpenStax textbooks. Warm `.tree` on 58 PDFs in 0.96s, 123 PDFs in 2.97s. Search across 123 PDFs in ~4s.
- Updated `--help` SCALE section with real PDF corpus numbers instead of generic "sub-second" claims.
- Added PDF directory triage example to README showing real `.tree` output across a collection of papers and government reports.
- Updated SKILL.md with PDF-at-scale use case, level-specific heading queries, and fuzzy matching examples.
- Updated `bench/results.md` with full search benchmarks (algorithm, security, neural network) and per-file warm cost.

## [0.3.8] - 2026-03-27

### Added

- **Level-specific selectors**: `.h1` through `.h6` for precise heading-level queries.
  - `.h2` — list all H2 headings (triage mode)
  - `.h2("Auth") | .text` — full section under the matching H2 (read mode)
- **4-tier matching ladder** for `.section()` and `.hN()`: exact -> case-insensitive exact -> case-insensitive prefix -> case-insensitive contains. First match in document order wins.
  - `.section("Auth")` matches `## Authentication`
  - `.h3("Chapter 1")` matches `### Chapter 1: What Is an Agent?`

## [0.3.7] - 2026-03-27

### Improved

- Rewrote `--help` output to lead with **when to use mq** (triage directories, extract from large files, search across documents) instead of listing syntax. Agents now see decision criteria first.
- Added WORKFLOW section (Map / Narrow / Extract) showing the incremental exploration pattern.
- Added SCALE section noting sub-second performance on hundreds of files.
- Called out supported formats (Markdown, HTML, PDF) on the second line.
- Added short-circuit guidance: files under 100 lines, just read them directly.

## [0.3.6] - 2026-03-27

### Documentation

- Anonymized the private manuscript directory benchmark in `README.md` and `bench/results.md` while preserving safe scale context with file and line counts.
- Renamed the benchmark label in `mql/benchmark_test.go` to `private-manuscript` so benchmark output matches the published docs.

## [0.3.5] - 2026-03-27

### Benchmarks

- Added real cache-aware directory search benchmarks in `mql/benchmark_test.go` for cold search, warm exact-repeat cache hits, and partial invalidation.
- Benchmarks now cover two real local corpora when available: an anonymized private manuscript corpus and `~/.rush/sessions`.

### Documentation

- Replaced the “benchmark gap” note with measured directory-search cache numbers in `README.md` and `bench/results.md`.
- Restored multi-document scale data in `bench/results.md` alongside the new real-corpus directory benchmark section.

## [0.3.4] - 2026-03-27

### Documentation

- Refreshed `README.md` headline results to emphasize warm-cache PDF latency and cross-format query support.
- Replaced stale benchmark tables in `README.md` with current Apple M3 Max measurements for Markdown, HTML, YAML, PDF, core query primitives, and MQL execution.
- Rewrote `bench/results.md` to match the benchmark coverage that exists in the repo today, and removed misleading JSON/JSONL parse figures from the published report until a real structured-data benchmark lands.

## [0.3.3] - 2026-03-27

### Features

- **Fallback PDF structure inference**: when font-based heading detection finds nothing, `mq` now derives structure from label-like lines, section markers, and bullet lists so `.tree` can still show useful hierarchy on scanned forms and administrative PDFs.
- **Structured non-markdown text output**: document-level `.text` now returns readable whole-document content for PDFs and markdown, matching the section-level extraction behavior.
- **Structured search rendering**: search results and non-markdown data output are formatted more cleanly for downstream terminal use.
- **Cached structured directory search for JSONL-heavy trees**: repeated `.search(...) | .tree` and `.search(...) | .text` queries on large directories now cache whole-query results by directory hash, cache per-file matches, reuse already-read bytes for matched files, and fast-path JSONL line search without reparsing unmatched records.

### Fixes

- Removed whole-document `.text` struct dumps for PDFs with OCR text by routing document extraction through `ReadableText()`.
- Preserved page-aware text normalization for inferred PDF sections so fallback tree nodes retain correct page numbers in `.tree`.
- Directory search now closes its cache DB handle per invocation, matching CLI usage and avoiding stale long-lived search state.

### Performance

| Scenario | Before | After |
|----------|--------|-------|
| Real directory search on `~/.rush/sessions` (`.search("requires OAuth") \| .tree`, corpus: 8,495 supported files / 4,644 `.json` / 3,083 `.jsonl` / 587MB total / 308,408 JSONL records / 86 matched paths / 176 matched records) | ~5.1-5.6s | ~1.38-1.40s |
| Improvement | baseline | ~3.7x faster |

## [0.2.1] - 2026-03-23

### Features

- **Content-addressed parse cache with Merkle directory tree**: Parsed documents are cached in bbolt (`~/Library/Caches/mq/cache.db`). Subsequent queries on the same file skip parsing entirely. **85x speedup on cached PDF hits** (1.69s -> 23ms). For directories, a Merkle hash tree tracks changes per-subtree so unchanged subdirectories are skipped without re-reading any files.
- **Stat-based short-circuit**: mtime+size check before content hashing — avoids reading file content when nothing changed.
- **Auto-eviction**: Cache entries unused for 5+ days are trimmed automatically.
- **HTML sections now have text content**: `.section("X") | .text` and `.search("term")` work on HTML files with line ranges and previews, matching markdown and PDF behavior.
- **Unified `.tree` command**: No more preview/full distinction. `.tree` always shows previews. Both files and directories default to `.tree`.

### Performance

| Scenario | Before | After |
|----------|--------|-------|
| PDF query (cold) | 1.69s | 1.69s (first parse) |
| PDF query (cached) | 1.69s | 0.02s (85x faster) |
| Directory re-scan (unchanged) | O(files) stat+parse | O(dirs) Merkle check |

### Dependencies

- Added `go.etcd.io/bbolt` for cache storage (pure Go, single-file DB)
- Added `github.com/vmihailenco/msgpack/v5` for fast serialization (5x faster than gob)

## [0.1.9] - 2026-03-23

### Features

- **PDF support works end-to-end**: `.tree("preview")`, `.section("X") | .text`, `.search("term")` all work on PDFs
- **Default output is now tree with previews**: running `mq file` shows `.tree("preview")`, `mq dir/` shows `.tree("full")`
- **Embedded PDF extraction script**: `extract_structure.py` is embedded in the binary via `go:embed`, no separate Python script install needed
- **Ordered sections**: sections now display in document order (fixed map iteration bug)
- **CHANGELOG shown on upgrade**: `mq upgrade` now displays what changed

### Fixes

- PDF sections now have text content (Start/End/source populated from pdftotext output)
- Heading-to-text matching uses score-based fuzzy matching for PDFs with section numbers
- `GetSections()` and `GetTableOfContents()` return sections in document order instead of random map order

## [0.1.0] - 2025-01-23

Initial release.

### Features

- Query markdown files with `.tree`, `.tree("full")`, `.tree("preview")`, `.tree("compact")`
- Search content with `.search("term")`
- Extract sections with `.section("Name") | .text`
- Extract code blocks with `.code("language")`
- Directory traversal with recursive markdown discovery
- Frontmatter parsing with `.metadata`, `.owner`, `.tags`
- Link and image extraction with `.links`, `.images`
- Pipeline operations for chaining queries
