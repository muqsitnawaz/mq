# Changelog

## [0.3.2] - 2026-03-27

### Features

- **Fallback PDF structure inference**: when font-based heading detection finds nothing, `mq` now derives structure from label-like lines, section markers, and bullet lists so `.tree` can still show useful hierarchy on scanned forms and administrative PDFs.
- **Structured non-markdown text output**: document-level `.text` now returns readable whole-document content for PDFs and markdown, matching the section-level extraction behavior.
- **Structured search rendering**: search results and non-markdown data output are formatted more cleanly for downstream terminal use.

### Fixes

- Removed whole-document `.text` struct dumps for PDFs with OCR text by routing document extraction through `ReadableText()`.
- Preserved page-aware text normalization for inferred PDF sections so fallback tree nodes retain correct page numbers in `.tree`.

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
