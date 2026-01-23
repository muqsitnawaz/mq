# mq Skill: Efficient Document Querying

Use `mq` to query markdown documents. Your context window is the working index - each query builds your understanding of the document structure.

## The Pattern

```
1. See structure    →  mq <path> .tree
2. Find relevant    →  mq <path> '.search("term")'
3. Extract content  →  mq <path> '.section("Name") | .text'
```

Your context accumulates structure. Don't re-query what you already know.

## Quick Reference

```bash
# Structure (your working index)
mq file.md .tree                    # Document structure
mq file.md '.tree("preview")'       # Structure + content previews
mq dir/ .tree                       # Directory overview
mq dir/ '.tree("full")'             # All files with sections + previews

# Search
mq file.md '.search("term")'        # Find sections containing term
mq dir/ '.search("term")'           # Search across all files

# Extract
mq file.md '.section("Name") | .text'   # Get section content
mq file.md '.code("python")'            # Get code blocks by language
mq file.md .links                       # Get all links
mq file.md .metadata                    # Get YAML frontmatter
```

## Efficient Workflow

### Starting: Get the Map

```bash
# For a single file
mq README.md .tree

# For a directory (start here for multi-file exploration)
mq docs/ '.tree("full")'
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
mq docs/ '.search("OAuth")'
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
mq docs/auth.md '.section("OAuth Flow") | .text'
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
mq docs/auth.md '.section("OAuth Flow") | .text'  # Get only what's needed
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
mq docs/auth.md '.section("OAuth") | .text'
```

## Context as Index

Every `.tree` output you receive becomes part of your working memory. Think of your context window as an index that grows as you explore:

```
Query 1: mq docs/ .tree
→ Context now contains: file list, line counts, section counts

Query 2: mq docs/auth.md .tree
→ Context now contains: file list + auth.md's full section hierarchy

Query 3: mq docs/auth.md '.section("OAuth") | .text'
→ Context now contains: structure + actual OAuth content
```

You're building a mental map. Use it - don't rebuild it.

## Examples by Task

### "Find how authentication works"
```bash
mq docs/ '.search("auth")'           # Find relevant files/sections
mq docs/auth.md '.section("Overview") | .text'  # Read the overview
```

### "Get all Python examples"
```bash
mq docs/ '.tree("full")'             # Find files with examples
mq docs/examples.md '.code("python")'  # Extract all Python code
```

### "Understand the API structure"
```bash
mq docs/api.md .tree                 # See all endpoints/sections
mq docs/api.md '.section("Endpoints") | .tree'  # Drill into endpoints
mq docs/api.md '.section("POST /users") | .text'  # Get specific endpoint
```

### "Find configuration options"
```bash
mq . '.search("config")'             # Search entire project
mq config.md '.section("Options") | .text'  # Extract options
```
