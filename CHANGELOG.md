# Changelog

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
