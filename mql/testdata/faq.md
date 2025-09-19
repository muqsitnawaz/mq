# Frequently Asked Questions

## General Questions

### What is Engram?

Engram is a hierarchical memory system designed specifically for AI agents. It provides natural language-based storage with structured formatting, enabling efficient context management and retrieval for Large Language Models (LLMs).

### How does Engram differ from traditional databases?

Unlike traditional databases that use SQL or NoSQL queries, Engram:
- Uses markdown as the native format
- Provides the mq query language optimized for AI
- Supports semantic search out of the box
- Caches parsed documents for performance
- Handles hierarchical data naturally

### Is Engram suitable for production use?

Yes, Engram is production-ready with:
- Thread-safe operations
- Horizontal scaling support
- Enterprise authentication
- Comprehensive monitoring
- 99.9% uptime SLA for cloud version

## Installation & Setup

### Q: What are the system requirements for running Engram?

**A:** Minimum requirements:
- 8GB RAM (16GB recommended)
- 10GB disk space
- Go 1.21+ or Docker
- Linux, macOS, or Windows 10+

### Q: Can I install Engram without root/administrator privileges?

**A:** Yes, you can install Engram in your user directory:
```bash
go install github.com/engram/mq@latest
# Binary will be in $(go env GOPATH)/bin
```

### Q: How do I upgrade to the latest version?

**A:** Depending on your installation method:
- Go: `go install github.com/engram/mq@latest`
- Homebrew: `brew upgrade mq`
- Docker: `docker pull engram:latest`

## Query Language (MQ)

### Q: How do I extract all code blocks from a markdown file?

**A:** Use the `.code` selector:
```bash
mq '.code' document.md           # All code blocks
mq '.code("python")' document.md  # Only Python code
```

### Q: Can I search across multiple files simultaneously?

**A:** Yes, use the glob pattern:
```bash
mq '.glob("docs/**/*.md") | .section("API") | .merge'
```

### Q: What's the difference between `.text` and `.text_content`?

**A:**
- `.text` preserves some formatting
- `.text_content` strips all markdown syntax

Example:
```bash
echo "# **Bold** Title" | mq '.text'          # "Bold Title"
echo "# **Bold** Title" | mq '.text_content'  # "Bold Title"
```

### Q: How do I filter results based on conditions?

**A:** Use the `select()` function with conditions:
```bash
mq '.code | select(.language == "python" and .lines > 10)'
mq '.headings | select(.level <= 2)'
mq '.sections | select(contains("API"))'
```

### Q: Can I output results in different formats?

**A:** Yes, MQ supports multiple output formats:
```bash
mq '.headings | .json'      # JSON (default)
mq '.headings | .yaml'      # YAML
mq '.headings | .text'      # Plain text
mq '.headings | .markdown'  # Preserve markdown
mq '.headings | .csv'       # CSV format
```

## Performance & Optimization

### Q: Why is my query slow on large documents?

**A:** Enable caching for better performance:
```bash
# First run parses and caches
mq --cache '.complex_query' large-file.md

# Subsequent runs use cache
mq --cache '.another_query' large-file.md
```

### Q: How much memory does the cache use?

**A:** Default cache size is 100MB, configurable in `~/.engram/config.yaml`:
```yaml
cache:
  size: 200MB  # Increase for large document sets
  ttl: 48h     # How long to keep cached documents
```

### Q: Can I pre-index documents for faster queries?

**A:** Yes, use the load command:
```bash
mq load docs/*.md --index
mq query '.headings' --cached  # Uses pre-built index
```

## AI & Semantic Features

### Q: How does semantic search work in MQ?

**A:** Semantic search uses embeddings to find similar content:
```bash
mq '.search("user authentication flow")'  # Finds related sections
mq '.similar_to(.section("Overview"))'    # Finds similar sections
```

### Q: What AI providers are supported for embeddings?

**A:** Currently supported:
- OpenAI (text-embedding-ada-002)
- Cohere
- HuggingFace models
- Local embeddings with Ollama

### Q: How do I optimize content for LLM context windows?

**A:** Use chunking features:
```bash
mq '.chunk(4000)'         # Token-based chunks
mq '.chunk_semantic'      # Semantic boundaries
mq '.chunk_by_section'    # Section-based chunks
```

### Q: Can MQ generate summaries of content?

**A:** Yes, with AI features enabled:
```bash
mq '.section("Documentation") | .summary(100)'  # 100-word summary
mq '.summarize_for_memory'                      # Concise version
```

## Common Issues & Troubleshooting

### Q: I get "command not found" after installation. What's wrong?

**A:** Your PATH might not include the Go bin directory:
```bash
# Add to ~/.bashrc or ~/.zshrc
export PATH=$PATH:$(go env GOPATH)/bin

# Reload shell configuration
source ~/.bashrc  # or ~/.zshrc
```

### Q: Why does my query return empty results when I know the content exists?

**A:** Check these common issues:
1. **Case sensitivity:** Use regex for case-insensitive matching
   ```bash
   mq '.heading(/api/i)' document.md
   ```
2. **Section boundaries:** Sections end at same or higher level headings
3. **Escaping:** Special characters need escaping in strings

### Q: How do I debug a complex query?

**A:** Use verbose mode and break down the pipeline:
```bash
# Enable debug output
mq --verbose '.complex | .query | .pipeline' file.md

# Test each stage separately
mq '.complex' file.md
mq '.complex | .query' file.md
mq '.complex | .query | .pipeline' file.md
```

### Q: Can I use MQ in my CI/CD pipeline?

**A:** Yes, MQ works well in automation:
```yaml
# GitHub Actions example
- name: Validate documentation
  run: |
    mq '.headings | .count' docs/*.md
    mq '.code | .validate' examples/*.md
```

## Advanced Usage

### Q: How do I create custom operators?

**A:** Extend MQ with custom Go functions:
```go
func init() {
    mq.RegisterOperator("custom", customOperator)
}

func customOperator(ctx *Context) (*Context, error) {
    // Your implementation
    return ctx, nil
}
```

### Q: Can I use MQ as a library in my Go application?

**A:** Yes, MQ is designed to be embedded:
```go
import "github.com/engram/mq"

engine := mq.New(mq.WithCache(true))
doc, _ := engine.Load("document.md")
results := engine.Query(doc).Section("API").Execute()
```

### Q: Is there a way to extend MQ with plugins?

**A:** Plugin support is planned for v2.0. Currently, you can:
- Fork and add custom operators
- Use MQ as a library with extensions
- Contribute operators to the main project

### Q: How do I handle non-English content?

**A:** MQ fully supports Unicode:
```bash
mq '.heading("第一章")' chinese.md
mq '.section("Введение")' russian.md
mq '.text' arabic.md  # RTL text supported
```

## Integration & Ecosystem

### Q: Can MQ integrate with my existing documentation tools?

**A:** Yes, MQ works with:
- **Static Site Generators:** Jekyll, Hugo, MkDocs
- **Documentation Systems:** Sphinx, Docusaurus
- **Knowledge Bases:** Obsidian, Notion (via export)
- **IDEs:** VS Code extension available

### Q: Is there a REST API for MQ?

**A:** Yes, run MQ in server mode:
```bash
mq server --port 8080

# Query via HTTP
curl -X POST http://localhost:8080/query \
  -d '{"query": ".headings", "file": "doc.md"}'
```

### Q: Can I use MQ with my RAG (Retrieval Augmented Generation) pipeline?

**A:** Absolutely! MQ is designed for RAG:
```python
import requests

# Query relevant sections
response = requests.post('http://mq-server/query', json={
    'query': '.context("user question") | .chunk(2000)',
    'files': ['knowledge/*.md']
})

context = response.json()
# Pass context to your LLM
```

## Security & Privacy

### Q: Is my data encrypted when using cloud features?

**A:** Yes:
- Data encrypted at rest (AES-256)
- TLS 1.3 for data in transit
- API keys use secure token generation
- Optional end-to-end encryption available

### Q: Can I self-host all components?

**A:** Yes, Engram is fully self-hostable:
- No phone-home or telemetry
- Works completely offline
- Private embedding models supported
- On-premise deployment guides available

### Q: How are API keys managed?

**A:** Best practices:
- Store in environment variables
- Use `.env` files (never commit)
- Rotate keys regularly
- Scope keys to specific operations

## Licensing & Support

### Q: What license is Engram released under?

**A:** Engram uses the MIT License for the core engine. Some enterprise features require a commercial license.

### Q: Is commercial support available?

**A:** Yes, we offer:
- **Community:** GitHub issues, Discord
- **Pro:** Email support, 48h response
- **Enterprise:** 24/7 support, SLA, training

### Q: How can I contribute to the project?

**A:** We welcome contributions:
1. Check [CONTRIBUTING.md](https://github.com/engram/engram/blob/main/CONTRIBUTING.md)
2. Look for "good first issue" labels
3. Join our Discord for guidance
4. Submit PRs with tests

### Q: Where can I report bugs or request features?

**A:**
- **Bugs:** [GitHub Issues](https://github.com/engram/engram/issues)
- **Features:** [Discussions](https://github.com/engram/engram/discussions)
- **Security:** security@engram.ai

## Migration & Compatibility

### Q: Can I migrate from other markdown tools?

**A:** Yes, we provide migration guides for:
- grep/ripgrep patterns → MQ queries
- SQL queries → MQ equivalents
- jq pipelines → MQ pipelines

### Q: Is MQ backward compatible?

**A:** We follow semantic versioning:
- Patch versions: Always compatible
- Minor versions: Additive changes only
- Major versions: May have breaking changes (rare)

### Q: What markdown flavors are supported?

**A:** MQ supports:
- CommonMark (default)
- GitHub Flavored Markdown
- Pandoc Markdown (partial)
- Custom extensions via configuration

---

*Can't find your answer? Ask in our [Discord](https://discord.gg/engram) or check the [documentation](https://docs.engram.ai).*