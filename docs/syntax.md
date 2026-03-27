# mq Syntax Guide

## Relationship to jq

mq is inspired by jq's piping model and query syntax, but designed specifically for markdown's semantic structure. If you know jq, you'll recognize the patterns - but there are intentional differences.

## What's the Same as jq

**Piping** - Chain operations with `|`:
```bash
mq doc.md ".section('API') | .code('go')"
mq doc.md ".headings | filter(.level == 2)"
```

**Indexing** - Access elements by position:
```bash
mq doc.md '.headings[0]'          # First heading
mq doc.md '.code[1:3]'            # Slice: 2nd and 3rd code blocks
```

**Filtering** - Select elements with predicates:
```bash
mq doc.md '.headings | filter(.level > 1)'
mq doc.md '.code | filter(.lang == "python")'
```

**Chaining** - Compose multiple operations:
```bash
mq doc.md ".section('Examples') | .code('python') | .[0]"
```

## Key Differences (and Why)

### 1. Arguments Required for Semantic Lookups

**jq**: `.section` accesses a JSON field named "section"
**mq**: `.section("title")` requires the section heading text

```bash
# This is necessary because markdown sections are identified by heading text,
# not implicit keys like JSON fields
mq doc.md ".section('Installation')"
mq doc.md ".code('python')"
```

**Why**: In JSON, fields have explicit keys. In markdown, sections are identified by their heading text. We can't magically know which section you want without the title.

### 2. Selectors Use Dot Notation

**jq**: `. | tree` (piping to a function)
**mq**: `.tree` (selector with dot)

```bash
# mq syntax
mq doc.md .tree
mq doc.md ".section('API') | .tree"
mq doc.md ".section('API') | .text"

# We chose this because it's cleaner - not every operation needs to be piped
```

**Why**: We prioritized ergonomics. `.tree` is cleaner than `. | tree`. The dot indicates "apply this selector to the current document/section", which aligns with the jq mental model of "current value".

### 3. `.text` is a Selector, Not a Function

**jq**: `. | text` would be a function
**mq**: `| .text` is a selector

```bash
mq doc.md ".section('API') | .text"
```

**Why**: Same reason as above - cleaner syntax. Text extraction is so common in markdown querying that making it a selector reduces verbosity.

## Full Selector Reference

### Structure Selectors

| Selector | Description | Example |
|----------|-------------|---------|
| `.tree` | Document structure (adapts to file vs directory) | `mq doc.md .tree` |

### Content Selectors

| Selector | Description | Example |
|----------|-------------|---------|
| `.section("name")` | Section by heading | `mq doc.md ".section('API')"` |
| `.sections` | All sections | `mq doc.md .sections` |
| `.headings` | All headings | `mq doc.md .headings` |
| `.headings(N)` | Headings at level N | `mq doc.md '.headings(2)'` |
| `.code` | All code blocks | `mq doc.md .code` |
| `.code("lang")` | Code blocks by language | `mq doc.md ".code('python')"` |
| `.links` | All links | `mq doc.md .links` |
| `.images` | All images | `mq doc.md .images` |
| `.tables` | All tables | `mq doc.md .tables` |

### Search & Navigation

| Selector | Description | Example |
|----------|-------------|---------|
| `.search("term")` | Find sections with term | `mq doc.md ".search('auth')"` |
| `.nth(N)` | Pick the Nth item from current results | `mq log.jsonl ".search('auth') | .nth(0)"` |
| `.metadata` | YAML frontmatter | `mq doc.md .metadata` |
| `.owner` | Owner field from metadata | `mq doc.md .owner` |
| `.tags` | Tags from metadata | `mq doc.md .tags` |

### Extraction

| Selector | Description | Example |
|----------|-------------|---------|
| `.text` | Extract raw text | `mq doc.md ".section('API') \| .text"` |

## Operations

### Piping

Chain selectors with `|`:
```bash
mq doc.md ".section('Examples') | .code('go')"
mq doc.md ".headings | filter(.level == 2) | .text"
```

### Indexing

Access by position (0-based):
```bash
mq doc.md '.headings[0]'        # First heading
mq doc.md '.code[-1]'           # Last code block
mq doc.md '.sections[1:3]'      # Slice: 2nd and 3rd sections
```

### Filtering

Select elements matching predicate:
```bash
mq doc.md '.headings | filter(.level == 2)'
mq doc.md '.code | filter(.lang == "python")'
mq doc.md '.links | filter(.text contains "API")'
```

## Common Patterns

### Explore then Extract
```bash
# First, see structure
mq docs/api.md .tree

# Then extract specific section
mq docs/api.md ".section('Authentication') | .text"
```

### Search then Refine
```bash
# Search for sections about auth
mq docs/ ".search('authentication')"

# Refine to specific file and section
mq docs/auth.md ".section('OAuth Flow') | .code('javascript')"
```

### JSONL: Search then Expand
```bash
# Search returns line numbers with record type context
mq session.jsonl ".search('error')"
# → [line 14] assistant/tool_use: Bash
#     ts: 2026-02-01T20:25:38Z
#     > ...command failed with error...

# Expand all matching records
mq session.jsonl ".search('error') | .text"

# Narrow to one matching record only if needed
mq session.jsonl ".search('error') | .nth(0) | .text"
```

### Filter by Level
```bash
# Get all H2 headings
mq doc.md '.headings | filter(.level == 2)'

# Get top-level sections
mq doc.md '.sections | filter(.level == 1)'
```

### Directory Operations
```bash
# Overview of all files (with sections + previews)
mq docs/ .tree

# Search across directory
mq docs/ ".search('error handling')"
```

## For Agent Integration

When integrating mq into agent workflows, follow this pattern:

1. **Explore structure first** - Don't read entire files blindly
   ```bash
   mq docs/ .tree
   ```

2. **Narrow down** - Identify the specific file or subdirectory
   ```bash
   mq docs/api.md .tree
   ```

3. **Extract only what's needed** - Pull specific sections
   ```bash
   mq docs/api.md ".section('Error Codes') | .text"
   ```

This structure-first approach reduces token usage by 50-83% compared to reading full files.

## Why These Design Choices?

**jq-inspired, not jq-compatible**: We borrow jq's piping model because it's intuitive for chaining operations. But we don't force jq purity when it hurts ergonomics.

**Arguments for semantic lookups**: Markdown sections are identified by heading text, not implicit keys. `.section("API")` is explicit and clear.

**Dot notation for selectors**: `.tree` and `.text` are cleaner than piping everything. The dot indicates "apply to current value", which aligns with jq's mental model.

**Designed for agents**: The query patterns optimize for how LLMs reason - explore structure, then extract specifics. This is more important than syntax purity.
