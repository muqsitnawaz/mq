# Contributing to mq

We welcome contributions! Here's how you can help.

## Roadmap: First-Class Support for Other Formats

We're looking for contributors to add agentic querying support for:

- **PDFs** - Extract structure (TOC, sections, tables) for agent consumption
- **Excel/CSV** - Query spreadsheet structure and content
- **Images** - Extract text, diagrams, or structured data
- **Other structured formats** - YAML, TOML, XML with semantic querying

The goal: let agents query any document format the same way they query markdown - see structure first, extract what they need.

## How to Contribute

1. **Open an issue first** - Discuss your approach before writing code
2. **Keep it simple** - mq is a single binary with zero dependencies. Let's keep it that way.
3. **Match the philosophy** - mq exposes structure for agents to reason over. It doesn't compute answers.

## Code Style

- Run `gofmt` before committing
- Run `go vet ./...` to catch issues
- Add tests for new functionality

## Pull Requests

1. Fork the repo
2. Create a branch (`git checkout -b feature/pdf-support`)
3. Make your changes
4. Run tests (`go test ./...`)
5. Submit a PR

## Questions?

Open an issue or start a discussion. We're happy to help.
