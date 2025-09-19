---
title: "MQ Query Language Tutorial"
author: "Engram Team"
date: "2024-01-20"
version: "1.0.0"
tags:
  - tutorial
  - mq
  - markdown
  - query-language
difficulty: beginner
estimated_time: "45 minutes"
prerequisites:
  - basic_markdown: true
  - programming_experience: false
  - command_line: true
---

# MQ Query Language Tutorial

Welcome to the comprehensive tutorial for the **mq** (markdown query) language. This guide will teach you how to efficiently query and manipulate markdown documents.

## Chapter 1: Introduction

### What is MQ?

MQ is a powerful query language designed specifically for markdown documents. It follows a pipeline model similar to `jq`, making it intuitive for developers familiar with command-line tools.

![MQ Architecture](./images/mq-architecture.png)

### Why Use MQ?

1. **Efficient Content Extraction** - Query specific sections without parsing entire documents
2. **AI-Optimized** - Built for LLM context windows and semantic search
3. **Cached Processing** - Parse once, query many times
4. **Flexible Output** - JSON, YAML, plain text, or structured formats

## Chapter 2: Getting Started

### Your First Query

Let's start with a simple example. Suppose you have this markdown:

```markdown
# Welcome

This is a paragraph.

## Features

- Fast parsing
- Efficient caching
- Semantic search
```

To extract all headings:

```bash
mq '.headings' document.md
```

Output:
```json
[
  {"level": 1, "text": "Welcome"},
  {"level": 2, "text": "Features"}
]
```

### Understanding the Pipeline

MQ uses the pipe operator (`|`) to chain operations:

```bash
mq '.headings | .text' document.md
```

This:
1. Selects all headings
2. Extracts just the text from each heading

## Chapter 3: Basic Selectors

### Heading Selectors

Let's explore different ways to select headings:

```bash
# All level 1 headings
mq '.h1' tutorial.md

# Headings level 2 and below
mq '.headings(2:6)' tutorial.md

# Headings matching a pattern
mq '.heading(/Chapter.*/)' tutorial.md
```

### Exercise 1: Heading Navigation

Try these queries on this document:

1. Find all chapter headings
2. Get only the main title
3. Extract subheadings under "Chapter 3"

#### Solution

```bash
# 1. Chapter headings
mq '.heading(/^Chapter \d+/)' tutorial.md

# 2. Main title
mq '.h1 | first | .text' tutorial.md

# 3. Subheadings under Chapter 3
mq '.section("Chapter 3") | .h3' tutorial.md
```

### Content Type Selectors

#### Code Blocks

Extract code blocks by language:

```python
# Python code block
def hello_world():
    print("Hello, MQ!")
    return True
```

```javascript
// JavaScript code block
function helloWorld() {
    console.log("Hello, MQ!");
    return true;
}
```

Query examples:
```bash
# All code blocks
mq '.code' tutorial.md

# Only Python code
mq '.code("python")' tutorial.md

# Multiple languages
mq '.code("python", "javascript")' tutorial.md
```

#### Lists

Different list types in this document:

**Unordered List:**
- First item
- Second item
  - Nested item
  - Another nested
- Third item

**Ordered List:**
1. Step one
2. Step two
3. Step three
   a. Sub-step A
   b. Sub-step B

**Task List:**
- [x] Completed task
- [ ] Pending task
- [ ] Another todo

Query them:
```bash
# All lists
mq '.lists' tutorial.md

# Unordered lists only
mq '.unordered_lists' tutorial.md

# Task lists with status
mq '.task_lists | map({task: .text, done: .checked})' tutorial.md
```

## Chapter 4: Advanced Queries

### Section Navigation

Sections are content blocks under headings. Here's how to navigate them:

```bash
# Get the Introduction section
mq '.section("Introduction")' tutorial.md

# Nested section selection
mq '.section("Chapter 4").section("Section Navigation")' tutorial.md

# All sections with code examples
mq '.sections | select(has(.code))' tutorial.md
```

### Context Building

When working with AI agents, context is crucial:

```bash
# Find relevant content with context
mq '.context("code blocks") | .expand_context' tutorial.md

# Include surrounding paragraphs
mq '.code("python") | .around(2)' tutorial.md
```

### Exercise 2: Complex Queries

Given this markdown structure, write queries to:

1. Extract all Python code with their preceding explanations
2. Find all completed tasks
3. Get a table of contents with only chapters

#### Solution Walkthrough

```bash
# 1. Python code with explanations
mq '.code("python") | .with_context | .json' tutorial.md

# 2. Completed tasks
mq '.task_lists | .list_items | select(.checked == true)' tutorial.md

# 3. Chapter TOC
mq '.headings | select(.text | startswith("Chapter")) | .toc' tutorial.md
```

## Chapter 5: Semantic Features

### AI-Powered Operations

MQ includes AI-optimized features for intelligent content processing:

```bash
# Semantic search
mq '.search("how to extract code blocks")' tutorial.md

# Automatic summarization
mq '.section("Chapter 4") | .summary(100)' tutorial.md

# Extract key concepts
mq '.keywords | .top(10)' tutorial.md
```

### Memory Optimization

For AI agents with limited context windows:

```bash
# Chunk document semantically
mq '.chunk_semantic | .limit(4000)' tutorial.md

# Extract only essential information
mq '.summarize_for_memory' tutorial.md

# Get facts and instructions separately
mq '.extract_facts | .merge(.extract_instructions)' tutorial.md
```

## Chapter 6: Practical Examples

### Example 1: Documentation Analysis

Analyze a project's documentation:

```bash
# Count code examples by language
mq '.code | group_by(.language) | map({lang: .key, count: .value | length})' docs/*.md

# Find all API endpoints
mq '.section("API") | .code("curl") | .extract_endpoint' api.md

# Generate a feature list
mq '.section("Features") | .lists | .list_items | .text' readme.md
```

### Example 2: Content Extraction for Training

Prepare markdown content for LLM training:

```bash
# Extract Q&A pairs
mq '.extract_qa | .format_jsonl' faq.md

# Get code examples with explanations
mq '.code | .with_context | map({
    explanation: .context.before,
    code: .content,
    language: .language,
    output: .context.after | .code
})' tutorial.md
```

### Example 3: Cross-Document Queries

Query across multiple files:

```bash
# Find all installation instructions
mq '.glob("docs/**/*.md") | .section(/Install/) | .merge_sections'

# Collect all error messages
mq '.glob("**/*.md") | .extract_errors | .unique | .sort'

# Build a unified API reference
mq '.glob("api/*.md") | .extract_api_specs | .combine'
```

## Chapter 7: Best Practices

### Performance Tips

1. **Use specific selectors**
   ```bash
   # Good: Specific section
   mq '.section("API").code("python")'

   # Bad: Search everything
   mq '.descendants | select(contains("API"))'
   ```

2. **Leverage caching**
   ```bash
   # Load once, query multiple times
   mq load docs/*.md
   mq query '.headings' --cached
   mq query '.code' --cached
   ```

3. **Filter early in pipeline**
   ```bash
   # Good: Filter first
   mq '.code("python") | .lines | select(. > 10)'

   # Bad: Filter last
   mq '.code | .lines | select(.language == "python" and . > 10)'
   ```

### Common Patterns

#### Pattern 1: Extract and Transform

```bash
# Extract all URLs from markdown links
mq '.links | map(.url) | .unique | .sort'

# Convert headings to navigation menu
mq '.headings | map({
    title: .text,
    level: .level,
    anchor: .text | .slugify
})'
```

#### Pattern 2: Conditional Processing

```bash
# Include examples only if they exist
mq '.if_exists(.section("Examples")) |
    .section("Examples") |
    .code |
    .else("No examples found")'
```

#### Pattern 3: Building Context

```bash
# Build context for code review
mq '.code | map({
    file: .file,
    function: .functions | first,
    complexity: .complexity_score,
    issues: .lint_issues
})'
```

## Chapter 8: Troubleshooting

### Common Issues

#### Issue: Query returns empty results

**Debugging steps:**
```bash
# Check document structure
mq '.outline' document.md

# Test with simpler query
mq '.headings' document.md

# Use verbose mode
mq --verbose '.section("Missing")' document.md
```

#### Issue: Performance degradation

**Solutions:**
```bash
# Enable caching
mq --cache '.complex_query' large-file.md

# Use pagination
mq '.results | .paginate(100)' document.md

# Optimize query
mq '.optimize(.slow_query)' document.md
```

### Error Messages

| Error | Meaning | Solution |
|-------|---------|----------|
| `Syntax error at position X` | Invalid query syntax | Check parentheses and quotes |
| `Selector not found` | Unknown selector | Verify selector name |
| `Type mismatch` | Invalid operation | Check data types in pipeline |

## Summary

Congratulations! You've learned:

✅ Basic MQ syntax and selectors
✅ Pipeline operations
✅ Content extraction techniques
✅ AI-optimized features
✅ Best practices and patterns

### Next Steps

1. **Practice with your own documents** - Try MQ on your project's markdown files
2. **Explore advanced features** - Check the [API Reference](./api-reference.md)
3. **Join the community** - Share queries and get help on Discord
4. **Contribute** - Submit your useful queries to our examples repository

### Quick Reference Card

```bash
# Essential queries
.headings           # All headings
.h1, .h2, .h3      # Specific levels
.section("Name")    # Section content
.code("lang")       # Code blocks
.lists             # All lists
.links             # All links
.tables            # All tables

# Transformations
| .text            # Extract text
| .markdown        # Keep formatting
| .json            # JSON output
| select(...)      # Filter
| map(...)         # Transform
| first, last      # Position

# AI Features
.context("query")   # Semantic search
.summary(100)       # Summarize
.chunk(4000)       # Split for LLM
.with_context      # Include context
```

## Appendix: Resources

### Official Documentation

- [MQ Language Specification](https://docs.engram.ai/mq/spec)
- [API Reference](./api-reference.md)
- [Configuration Guide](./configuration.md)

### Community Resources

- [MQ Cookbook](https://cookbook.engram.ai)
- [Video Tutorials](https://youtube.com/@engram)
- [Discord Server](https://discord.gg/engram)

### Related Tools

- [jq](https://jqlang.github.io/jq/) - JSON processor
- [pup](https://github.com/ericchiang/pup) - HTML processor
- [yq](https://mikefarah.gitbook.io/yq/) - YAML processor

---

*Tutorial Version: 1.0.0 | Last Updated: January 2024*
*Found an error? [Submit a correction](https://github.com/engram/docs/edit/main/tutorial.md)*