---
name: mq
description: Query markdown, HTML, and PDF files with mq CLI. Triggers on: exploring doc structure, extracting sections from large .md/.html/.pdf files, 'use mq', or when reading full documents wastes tokens.
---

# mq Skill: Efficient Document Querying

`mq` doesn't compute answers - it externalizes document structure into your context so you can reason to answers yourself.

```
Documents → mq query → Structure enters your context → You reason → Results
```

## The Pattern

```
1. See structure    →  mq <path> .tree            → Map enters your context
2. Find relevant    →  mq <path> ".search('x')"   → Locations enter your context
3. Extract content  →  mq <path> ".section('Y') | .text"  → Content enters your context
                       mq <path> ".search('x') | .text" → Flatten matched structured results
                       mq <path> ".search('x') | .nth(0)" → Show one raw matched result
                       mq <path> ".search('x') | .nth(0) | .raw" → Explicit raw record
4. Reason           →  You compute the answer from what's now in your context
```

Your context accumulates structure. You do the final reasoning.

## Quick Reference

```bash
# Structure (your working index)
mq file.md .tree                    # Document structure (headings, sections, previews)
mq dir/ .tree                       # Directory overview (all files with sections + previews)

# Search
mq file.md ".search('term')"        # Find sections containing term
mq dir/ ".search('term')"           # Search across all files
mq log.jsonl ".search('error')"     # JSONL: line-level search with record context

# Extract
mq file.md ".section('Name') | .text"   # Get section content
mq file.md ".code('python')"            # Get code blocks by language
mq file.md .links                       # Get all links
mq file.md .metadata                    # Get YAML frontmatter
mq log.jsonl ".search('error') | .text"            # Flatten matching records
mq log.jsonl ".search('error') | .nth(0)"          # Narrow to one raw matching record
mq log.jsonl ".search('error') | .nth(0) | .raw"   # Explicit raw record
```

## Efficient Workflow

### Starting: Get the Map

```bash
# For a single file
mq README.md .tree

# For a directory (start here for multi-file exploration)
mq docs/ .tree
```

Output shows you the territory:
```
docs/ (7 files, 42 sections)
├── API.md (234 lines, 12 sections)
│   ├── # API Reference
│   │        "Complete reference for all REST endpoints..."
│   ├── ## Authentication
│   │        "All requests require Bearer token..."
```

Now you know: API.md has auth info, 234 lines, section called "Authentication".

### Finding: Narrow Down

If you need something specific but don't know where:

```bash
mq docs/ ".search('OAuth')"
```

Output points you to exact locations:
```
Found 3 matches for "OAuth":

docs/auth.md:
  ## Authentication (lines 34-89)
     "...OAuth 2.0 authentication flow..."
  ## OAuth Flow (lines 45-67)
```

Now you know: auth.md, section "OAuth Flow", lines 45-67.

### Extracting: Get Only What You Need

Don't read the whole file. Extract the section:

```bash
mq docs/auth.md ".section('OAuth Flow') | .text"
```

This returns just that section's content.

## Anti-Patterns

**Bad**: Reading entire files
```bash
cat docs/auth.md  # Wastes tokens on irrelevant content
```

**Good**: Query then extract
```bash
mq docs/auth.md .tree                           # See structure
mq docs/auth.md ".section('OAuth Flow') | .text"  # Get only what's needed
```

**Bad**: Re-querying structure you already have
```bash
mq docs/ .tree    # First time - good
mq docs/ .tree    # Again - wasteful, you already have this in context
```

**Good**: Use what's in your context
```bash
mq docs/ .tree    # Once - now you know the structure
# Use the structure you learned to make targeted queries
mq docs/auth.md ".section('OAuth') | .text"
```

## Context as Working Memory

Every mq output enters your context. Your context becomes a working index that grows as you explore:

```
Query 1: mq docs/ .tree
→ You now see: file list, line counts, section counts
→ You can reason: "auth.md looks relevant to my question"

Query 2: mq docs/auth.md .tree
→ You now see: auth.md's full section hierarchy
→ You can reason: "OAuth Flow section has what I need"

Query 3: mq docs/auth.md ".section('OAuth Flow') | .text"
→ You now have: the actual content
→ You can reason: compute the final answer
```

mq externalizes structure. You do the thinking. Don't re-query what you already see.

## Format Casts

Cast operators reinterpret a string value as a different document format mid-pipeline.
Use when structured content (markdown, HTML, JSON, YAML) is embedded inside another format.

```bash
# Cast operators
.text | .md | .headings          # parse string as markdown
.text | .html | .links           # parse string as HTML
.raw | .json | .section("key")   # parse raw JSON, navigate keys
.text | .yaml | .tree            # parse string as YAML
```

Cast reference:

| Operator | Parses as | Use when field contains |
|----------|-----------|------------------------|
| `.md` | Markdown | `# Headings`, `- lists`, `` `code` `` |
| `.html` | HTML | `<h1>`, `<a href>`, `<table>` |
| `.json` | JSON | `{"key": "value"}` |
| `.yaml` | YAML | `key: value` |

Casts work on `string`, `[]string` (joined with newlines), and nested structures
(recursively extracts `text`/`content` fields from arrays of objects).

## Examples by Task

### "Find something in a JSONL session file"
```bash
mq session.jsonl ".search('deploy')"  # Line-level matches with record type
# → [line 5] user/user
#     content: Can you deploy the new version?
#     ts: 2026-02-01T20:25:29Z
# → [line 8] assistant/tool_use: Bash
#     ts: 2026-02-01T20:25:34Z

mq session.jsonl ".search('deploy') | .text"           # Flatten matching records
mq session.jsonl ".search('deploy') | .nth(1)"         # Narrow to one raw matching record
mq session.jsonl ".search('deploy') | .nth(1) | .raw"  # Explicit raw record
```

### "Query Claude session files"

Claude stores conversations as JSONL at `~/.claude/projects/{project-id}/{session-id}.jsonl`.
Each line is a JSON record with `type`, `message.role`, `message.content`, `timestamp`.

```bash
# Search all sessions in a project
mq ~/.claude/projects/-Users-you-project/ '.search("auth")'

# Drill into markdown content inside a JSONL record
mq session.jsonl '.search("REPORT") | .nth(0) | .raw | .json | .section("content") | .text | .md | .headings'

# Extract a specific markdown section from a conversation
mq session.jsonl '.search("REPORT") | .nth(0) | .raw | .json | .section("content") | .text | .md | .section("Recommendations") | .text'

# Get code blocks from markdown inside a JSONL record
mq session.jsonl '.search("impl") | .nth(0) | .raw | .json | .section("content") | .text | .md | .code("go")'
```

The pipeline pattern for nested content:
```
.search("term")     → find JSONL record
  | .nth(0)          → pick one result
  | .raw             → get raw JSON line
  | .json            → parse as JSON document
  | .section("key")  → navigate to a field
  | .text            → extract string value
  | .md              → cast to markdown
  | .headings        → structural query on inner content
```

### "Find how authentication works"
```bash
mq docs/ ".search('auth')"           # Find relevant files/sections
mq docs/auth.md ".section('Overview') | .text"  # Read the overview
```

### "Get all Python examples"
```bash
mq docs/ .tree                       # Find files with examples
mq docs/examples.md ".code('python')"  # Extract all Python code
```

### "Understand the API structure"
```bash
mq docs/api.md .tree                 # See all endpoints/sections
mq docs/api.md ".section('Endpoints') | .tree"  # Drill into endpoints
mq docs/api.md ".section('POST /users') | .text"  # Get specific endpoint
```

### "Find configuration options"
```bash
mq . ".search('config')"             # Search entire project
mq config.md ".section('Options') | .text"  # Extract options
```
