# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Experimental WebAssembly support for browser-based queries
- New `.parallel()` operator for concurrent query execution
- GraphQL endpoint for API queries

### Changed
- Improved cache performance by 40% using new compression algorithm

### Deprecated
- The `.legacy_search()` operator will be removed in v2.0.0

## [1.0.0] - 2024-01-20

### Added
* **Core Query Language** - Complete mq query language implementation
* **Semantic Search** - AI-powered content discovery with embeddings
* **Multi-format Support** - JSON, YAML, XML, CSV output formats
* **Caching System** - LRU cache with automatic invalidation
* **File Watching** - Real-time updates on file changes
* **Plugin System** - Extensible architecture for custom operators

### Changed
* **Breaking:** Changed from SQL-like syntax to pipeline-based syntax
* **Breaking:** Renamed `--output-format` to `--format`
* Improved query parser performance by 60%
* Reduced memory usage for large files by 45%

### Fixed
* Fixed memory leak in cache eviction (#234)
* Resolved race condition in concurrent file access (#256)
* Corrected Unicode handling in regex patterns (#189)

### Security
* Updated dependencies to patch CVE-2024-xxxxx
* Added input sanitization for query parameters
* Implemented rate limiting for API endpoints

## [0.9.0] - 2023-12-15

### Added
- Beta version of semantic search capabilities
- PostgreSQL and MongoDB support
- Docker container distribution
- Batch processing for multiple files

### Changed
- Migrated from JSON to YAML for default configuration
- Updated Go version requirement to 1.21
- Refactored internal AST representation

### Removed
- Dropped support for Go versions < 1.21
- Removed deprecated `--legacy` flag

### Fixed
- Issue with nested section extraction ([#187](https://github.com/engram/engram/issues/187))
- Windows path handling problems ([#143](https://github.com/engram/engram/issues/143))
- Memory leak in long-running processes ([#201](https://github.com/engram/engram/issues/201))

## [0.8.0] - 2023-11-01

### Added
1. **gRPC API** - Alternative to REST API
2. **Health Checks** - `/health` endpoint for monitoring
3. **Metrics Export** - Prometheus-compatible metrics
4. **Shell Completion** - Bash, Zsh, and Fish support

### Changed
- Redesigned CLI interface for better usability
- Optimized regex engine for 30% faster matching
- Updated documentation with video tutorials

### Deprecated
- REST API v0 endpoints (use v1 instead)

## [0.7.5] - 2023-10-10

### Fixed
#### Critical Fixes
- **CRITICAL:** Fixed data corruption bug in cache writes
- **HIGH:** Resolved security vulnerability in YAML parsing
- **MEDIUM:** Fixed incorrect handling of Windows line endings

#### Other Fixes
- Corrected timezone handling in frontmatter dates
- Fixed panic on malformed markdown input
- Resolved edge case in table parsing

## [0.7.0] - 2023-09-20

### Added
* Code syntax highlighting in terminal output
* Support for TOML frontmatter
* New operators:
  - `.group_by()` - Group results by field
  - `.unique()` - Remove duplicates
  - `.sort()` - Sort results
  - `.reverse()` - Reverse order

### Changed
* Performance improvements:
  * 50% faster startup time
  * 35% reduction in memory usage
  * Optimized regex compilation

### Fixed
* Fixed incorrect line numbers in error messages
* Resolved issues with non-ASCII characters
* Corrected behavior of `.around()` operator near document edges

## [0.6.0] - 2023-08-15

### Added
- **Goldmark Integration** - Replaced custom parser with Goldmark
- **AST Caching** - Cache parsed AST for repeated queries
- **Query Optimization** - Automatic query plan optimization
- **Streaming Mode** - Process large files without loading into memory

### Changed
- Improved error messages with suggestions
- Better handling of malformed markdown
- Enhanced regex performance using RE2

### Removed
- Custom markdown parser (replaced by Goldmark)
- Deprecated configuration options

## [0.5.0] - 2023-07-01

### Added
- Initial public release
- Basic query language implementation
- File system operations
- JSON output support
- Simple caching mechanism

### Known Issues
- Limited regex support
- No semantic search
- Single-file queries only
- Memory intensive for large files

## [0.4.0-beta] - 2023-06-01

### Added
- Beta testing phase begins
- Core selector implementation
- Pipeline operations
- Basic error handling

### Changed
- Rewrote parser from scratch
- Improved test coverage to 80%

## [0.3.0-alpha] - 2023-05-01

### Added
- Alpha release for early adopters
- Proof of concept implementation
- Basic documentation

### Notes
- Not recommended for production use
- API subject to change
- Limited functionality

## [0.2.0-dev] - 2023-04-01

### Added
- Development preview
- Initial prototype
- Concept validation

## [0.1.0-dev] - 2023-03-01

### Added
- Project inception
- Initial design document
- Repository creation

---

## Version History Summary

| Version | Release Date | Status | Go Version | Breaking Changes |
|---------|-------------|--------|------------|------------------|
| 1.0.0 | 2024-01-20 | **Stable** | 1.21+ | Yes |
| 0.9.0 | 2023-12-15 | Beta | 1.21+ | Yes |
| 0.8.0 | 2023-11-01 | Beta | 1.20+ | No |
| 0.7.5 | 2023-10-10 | Beta | 1.20+ | No |
| 0.7.0 | 2023-09-20 | Beta | 1.20+ | No |
| 0.6.0 | 2023-08-15 | Alpha | 1.19+ | Yes |
| 0.5.0 | 2023-07-01 | Alpha | 1.19+ | N/A |

## Upgrade Guide

### From 0.9.x to 1.0.0

**Breaking Changes:**
1. Query syntax changed from SQL-like to pipeline-based
2. Configuration file structure updated
3. API endpoints moved from v0 to v1

**Migration Steps:**
```bash
# 1. Backup your configuration
cp ~/.engram/config.yaml ~/.engram/config.yaml.backup

# 2. Run migration tool
mq migrate --from=0.9 --to=1.0

# 3. Update your scripts
# Old: mq "SELECT headings FROM document.md"
# New: mq '.headings' document.md

# 4. Clear cache (required)
mq cache clear

# 5. Rebuild indexes
mq index rebuild
```

### From 0.8.x to 0.9.x

**Changes:**
- Configuration format changed from JSON to YAML
- New Go version requirement

**Migration Steps:**
```bash
# Convert configuration
mq config convert --from=json --to=yaml

# Update Go version
go install github.com/engram/mq@v0.9.0
```

## Release Schedule

- **Major releases (x.0.0):** Annually (January)
- **Minor releases (1.x.0):** Quarterly
- **Patch releases (1.0.x):** As needed for critical fixes

## Support Policy

| Version | Support Level | End of Support |
|---------|--------------|----------------|
| 1.0.x | **Active** | January 2026 |
| 0.9.x | Security Only | July 2024 |
| 0.8.x | Security Only | April 2024 |
| < 0.8 | **Unsupported** | - |

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](https://github.com/engram/engram/blob/main/CONTRIBUTING.md) for details.

### Contributors

Special thanks to all our contributors:
- [@alice](https://github.com/alice) - Core query engine
- [@bob](https://github.com/bob) - Cache implementation
- [@carol](https://github.com/carol) - Documentation
- [@dave](https://github.com/dave) - Testing framework
- [Full list](https://github.com/engram/engram/graphs/contributors)

---

[Unreleased]: https://github.com/engram/engram/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/engram/engram/compare/v0.9.0...v1.0.0
[0.9.0]: https://github.com/engram/engram/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/engram/engram/compare/v0.7.5...v0.8.0
[0.7.5]: https://github.com/engram/engram/compare/v0.7.0...v0.7.5
[0.7.0]: https://github.com/engram/engram/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/engram/engram/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/engram/engram/releases/tag/v0.5.0