package mq_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mq "github.com/muqsitnawaz/mq/lib"
)

// Sample markdown content with frontmatter
const testMarkdown = `---
owner: alice
tags: [api, documentation, guide]
priority: high
---

# API Documentation

This is the main API documentation for our service.

## Introduction

Welcome to our API. This guide will help you get started.

## Authentication

All API requests require authentication using OAuth2.

### Getting Started

First, you need to register your application:

` + "```python" + `
import oauth2
client = oauth2.Client(client_id, client_secret)
client.authenticate()
` + "```" + `

### Token Management

Tokens expire after 1 hour and need to be refreshed.

` + "```python" + `
def refresh_token(client):
    return client.refresh()
` + "```" + `

## Endpoints

### User Management

The user management endpoints allow CRUD operations.

` + "```go" + `
func GetUser(id string) (*User, error) {
    // Implementation
    return nil, nil
}
` + "```" + `

## Rate Limiting

API requests are limited to 1000 per hour.

| Endpoint | Limit | Window |
|----------|-------|--------|
| /api/users | 100 | 1 hour |
| /api/posts | 500 | 1 hour |
| /api/search | 50 | 1 hour |

## Error Handling

All errors follow the standard format.

- [API Reference](https://api.example.com/reference)
- [Status Page](https://status.example.com)
`

func TestParseDocument(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(testMarkdown), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Test metadata extraction
	owner, ok := doc.GetOwner()
	if !ok || owner != "alice" {
		t.Errorf("Expected owner 'alice', got %s", owner)
	}

	tags := doc.GetTags()
	if len(tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(tags))
	}

	priority, ok := doc.GetPriority()
	if !ok || priority != "high" {
		t.Errorf("Expected priority 'high', got %s", priority)
	}
}

func TestGetHeadings(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(testMarkdown), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Test getting all headings
	allHeadings := doc.GetHeadings()
	if len(allHeadings) == 0 {
		t.Error("Expected headings, got none")
	}

	// Test getting specific levels
	h1s := doc.GetHeadings(1)
	if len(h1s) != 1 {
		t.Errorf("Expected 1 H1 heading, got %d", len(h1s))
	}
	if h1s[0].Text != "API Documentation" {
		t.Errorf("Expected 'API Documentation', got %s", h1s[0].Text)
	}

	h2s := doc.GetHeadings(2)
	if len(h2s) < 4 {
		t.Errorf("Expected at least 4 H2 headings, got %d", len(h2s))
	}
}

func TestGetSections(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(testMarkdown), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Test getting a specific section
	section, ok := doc.GetSection("Authentication")
	if !ok {
		t.Fatal("Expected to find Authentication section")
	}

	if section.Heading.Text != "Authentication" {
		t.Errorf("Expected section heading 'Authentication', got %s", section.Heading.Text)
	}

	// Test getting code blocks from section
	codeBlocks := section.GetCodeBlocks()
	if len(codeBlocks) == 0 {
		t.Error("Expected code blocks in Authentication section")
	}
}

func TestGetSectionMatchingLadder(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(testMarkdown), "test.md")
	require.NoError(t, err)

	// Tier 1: exact match
	section, ok := doc.GetSection("Authentication")
	assert.True(t, ok)
	assert.Equal(t, "Authentication", section.Heading.Text)

	// Tier 2: case-insensitive exact
	section, ok = doc.GetSection("authentication")
	assert.True(t, ok)
	assert.Equal(t, "Authentication", section.Heading.Text)

	// Tier 3: case-insensitive prefix ("Rate" prefixes "Rate Limiting")
	section, ok = doc.GetSection("Rate")
	assert.True(t, ok)
	assert.Equal(t, "Rate Limiting", section.Heading.Text)

	// Tier 4: case-insensitive contains ("Management" is inside "Token Management")
	section, ok = doc.GetSection("Management")
	assert.True(t, ok)
	// First match in doc order: Token Management (H3) comes before User Management (H3)
	assert.Equal(t, "Token Management", section.Heading.Text)

	// No match
	_, ok = doc.GetSection("Nonexistent")
	assert.False(t, ok)
}

func TestGetSectionByLevel(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(testMarkdown), "test.md")
	require.NoError(t, err)

	// .h2("Auth") - prefix match at level 2
	section, ok := doc.GetSectionByLevel(2, "Auth")
	assert.True(t, ok)
	assert.Equal(t, "Authentication", section.Heading.Text)
	assert.Equal(t, 2, section.Heading.Level)

	// .h3("Getting") - prefix match at level 3
	section, ok = doc.GetSectionByLevel(3, "Getting")
	assert.True(t, ok)
	assert.Equal(t, "Getting Started", section.Heading.Text)

	// .h3("Management") - contains match, first H3 in doc order
	section, ok = doc.GetSectionByLevel(3, "Management")
	assert.True(t, ok)
	assert.Equal(t, "Token Management", section.Heading.Text)

	// .h1("API") - level 1 prefix match
	section, ok = doc.GetSectionByLevel(1, "API")
	assert.True(t, ok)
	assert.Equal(t, "API Documentation", section.Heading.Text)

	// Wrong level: "Authentication" is H2, not H3
	_, ok = doc.GetSectionByLevel(3, "Authentication")
	assert.False(t, ok, "should not find H2 heading at level 3")
}

func TestGetSectionsByLevel(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(testMarkdown), "test.md")
	require.NoError(t, err)

	h2s := doc.GetSectionsByLevel(2)
	assert.Len(t, h2s, 5, "should have 5 H2 sections") // Introduction, Authentication, Endpoints, Rate Limiting, Error Handling

	h3s := doc.GetSectionsByLevel(3)
	assert.Len(t, h3s, 3, "should have 3 H3 sections") // Getting Started, Token Management, User Management

	h4s := doc.GetSectionsByLevel(4)
	assert.Len(t, h4s, 0, "should have no H4 sections")
}

func TestGetCodeBlocks(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(testMarkdown), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Test getting all code blocks
	allCode := doc.GetCodeBlocks()
	if len(allCode) != 3 {
		t.Errorf("Expected 3 code blocks, got %d", len(allCode))
	}

	// Test filtering by language
	pythonCode := doc.GetCodeBlocks("python")
	if len(pythonCode) != 2 {
		t.Errorf("Expected 2 Python code blocks, got %d", len(pythonCode))
	}

	goCode := doc.GetCodeBlocks("go")
	if len(goCode) != 1 {
		t.Errorf("Expected 1 Go code block, got %d", len(goCode))
	}
}

func TestGetTables(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(testMarkdown), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	tables := doc.GetTables()
	if len(tables) != 1 {
		t.Errorf("Expected 1 table, got %d", len(tables))
	}

	if len(tables) > 0 {
		table := tables[0]
		if len(table.Headers) != 3 {
			t.Errorf("Expected 3 headers, got %d", len(table.Headers))
		}
		if len(table.Rows) != 3 {
			t.Errorf("Expected 3 rows, got %d", len(table.Rows))
		}
	}
}

func TestGetLinks(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(testMarkdown), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	links := doc.GetLinks()
	if len(links) != 2 {
		t.Errorf("Expected 2 links, got %d", len(links))
	}
}

func TestFluentQueryBuilder(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(testMarkdown), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Test chaining operations
	result, err := engine.From(doc).
		Section("Authentication").
		Execute()
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}

	section, ok := result.(*mq.Section)
	if !ok {
		t.Error("Expected Section result")
	}
	if section.Heading.Text != "Authentication" {
		t.Errorf("Expected 'Authentication', got %s", section.Heading.Text)
	}

	// Test code filtering
	codeResult, err := engine.From(doc).
		Code("python").
		Execute()
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}

	codeBlocks, ok := codeResult.([]*mq.CodeBlock)
	if !ok {
		t.Error("Expected CodeBlock array result")
	}
	if len(codeBlocks) != 2 {
		t.Errorf("Expected 2 Python code blocks, got %d", len(codeBlocks))
	}

	// Test ownership check
	_, err = engine.From(doc).
		WhereOwner("alice").
		Section("Authentication").
		Execute()
	if err != nil {
		t.Errorf("Owner check failed: %v", err)
	}

	_, err = engine.From(doc).
		WhereOwner("bob").
		Section("Authentication").
		Execute()
	if err == nil {
		t.Error("Expected ownership check to fail for 'bob'")
	}
}

func TestOperators(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(testMarkdown), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Test Filter
	headings := doc.GetHeadings()
	filtered := mq.Filter(headings, func(h *mq.Heading) bool {
		return h.Level == 2
	})
	if len(filtered) < 4 {
		t.Errorf("Expected at least 4 H2 headings, got %d", len(filtered))
	}

	// Test Map
	texts := mq.Map(filtered, func(h *mq.Heading) string {
		return h.Text
	})
	if len(texts) != len(filtered) {
		t.Error("Map operation failed")
	}

	// Test Chain
	chain := mq.NewChain(doc.GetHeadings()).
		Filter(func(h *mq.Heading) bool {
			return h.Level <= 2
		}).
		Take(3)

	result := chain.Result()
	if len(result) > 3 {
		t.Errorf("Expected at most 3 results, got %d", len(result))
	}
}

func TestSectionEndLineNumbers(t *testing.T) {
	const sectionEndTestMarkdown = `# First Section

Content in first section.

## Nested Section

Content in nested section.

# Second Section

Content in second section.

## Another Nested

More content.
`

	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(sectionEndTestMarkdown), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	sections := doc.GetSections()
	if len(sections) == 0 {
		t.Fatal("Expected sections, got none")
	}

	for _, section := range sections {
		if section.End < 0 {
			t.Errorf("Section %q has negative End: %d", section.Heading.Text, section.End)
		}
		if section.Start < 0 {
			t.Errorf("Section %q has negative Start: %d", section.Heading.Text, section.Start)
		}
		if section.End < section.Start {
			t.Errorf("Section %q has End (%d) < Start (%d)", section.Heading.Text, section.End, section.Start)
		}
	}

	firstSection, ok := doc.GetSection("First Section")
	if !ok {
		t.Fatal("Expected to find 'First Section'")
	}
	if firstSection.End <= 0 {
		t.Errorf("First Section should have positive End, got %d", firstSection.End)
	}

	secondSection, ok := doc.GetSection("Second Section")
	if !ok {
		t.Fatal("Expected to find 'Second Section'")
	}
	if secondSection.End <= 0 {
		t.Errorf("Second Section should have positive End, got %d", secondSection.End)
	}
}

func TestComplexQueries(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(testMarkdown), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Get all H2 headings and filter them
	h2Headings := mq.FilterHeadingsByLevel(doc.GetHeadings(), 2)
	if len(h2Headings) < 4 {
		t.Errorf("Expected at least 4 H2 headings, got %d", len(h2Headings))
	}

	// Filter headings by text pattern
	authHeadings := mq.FilterHeadingsByText(doc.GetHeadings(), "Auth")
	if len(authHeadings) != 1 {
		t.Errorf("Expected 1 heading containing 'Auth', got %d", len(authHeadings))
	}

	// Filter code blocks by language and line count
	codeBlocks := doc.GetCodeBlocks()
	largeBlocks := mq.FilterCodeBlocksByLines(codeBlocks, 3)
	if len(largeBlocks) == 0 {
		t.Error("Expected some code blocks with 3+ lines")
	}

	// Test unique operations
	langs := []string{"python", "go", "python", "javascript", "go"}
	uniqueLangs := mq.Unique(langs)
	if len(uniqueLangs) != 3 {
		t.Errorf("Expected 3 unique languages, got %d", len(uniqueLangs))
	}
}
