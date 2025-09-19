package mql_test

import (
	"testing"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/muqsitnawaz/mq/mql"
)

const testDoc = `---
owner: alice
tags: [golang, testing]
priority: medium
---

# Test Document

## Section One

This is the first section.

` + "```go" + `
func Hello() {
    fmt.Println("Hello, World!")
}
` + "```" + `

## Section Two

This has multiple code examples.

` + "```python" + `
def greet(name):
    print(f"Hello, {name}!")
` + "```" + `

` + "```javascript" + `
function greet(name) {
    console.log(` + "`Hello, ${name}!`" + `);
}
` + "```" + `

### Subsection

A nested section with content.
`

func TestLexer(t *testing.T) {
	tests := []struct {
		input    string
		expected []mql.TokenType
	}{
		{
			input: ".headings",
			expected: []mql.TokenType{
				mql.TokenDot,
				mql.TokenIdentifier,
				mql.TokenEOF,
			},
		},
		{
			input: ".section('Auth')",
			expected: []mql.TokenType{
				mql.TokenDot,
				mql.TokenIdentifier,
				mql.TokenLParen,
				mql.TokenString,
				mql.TokenRParen,
				mql.TokenEOF,
			},
		},
		{
			input: ".headings | .filter(.level == 2)",
			expected: []mql.TokenType{
				mql.TokenDot,
				mql.TokenIdentifier,
				mql.TokenPipe,
				mql.TokenDot,
				mql.TokenIdentifier,
				mql.TokenLParen,
				mql.TokenDot,
				mql.TokenIdentifier,
				mql.TokenEquals,
				mql.TokenNumber,
				mql.TokenRParen,
				mql.TokenEOF,
			},
		},
	}

	for _, test := range tests {
		tokens, err := mql.Lex(test.input)
		if err != nil {
			t.Errorf("Lexer error for '%s': %v", test.input, err)
			continue
		}

		if len(tokens) != len(test.expected) {
			t.Errorf("Token count mismatch for '%s': expected %d, got %d",
				test.input, len(test.expected), len(tokens))
			continue
		}

		for i, token := range tokens {
			if token.Type != test.expected[i] {
				t.Errorf("Token type mismatch at position %d for '%s': expected %v, got %v",
					i, test.input, test.expected[i], token.Type)
			}
		}
	}
}

func TestParser(t *testing.T) {
	tests := []struct {
		input       string
		shouldError bool
	}{
		{".headings", false},
		{".section('Test')", false},
		{".headings | .code", false},
		{".code('python', 'go')", false},
		{".select(.level == 2)", false},
		{".headings | select(.level <= 2)", false},
		{"", true},
		{"|", true},
		{".", true},
	}

	for _, test := range tests {
		_, err := mql.ParseString(test.input)
		if test.shouldError && err == nil {
			t.Errorf("Expected error for '%s', but got none", test.input)
		}
		if !test.shouldError && err != nil {
			t.Errorf("Unexpected error for '%s': %v", test.input, err)
		}
	}
}

func TestCompiler(t *testing.T) {
	// Parse test document
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(testDoc), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Create compiler
	compiler := mql.NewCompiler()

	tests := []struct {
		query    string
		validate func(interface{}) bool
		desc     string
	}{
		{
			query: ".headings",
			validate: func(result interface{}) bool {
				headings, ok := result.([]*mq.Heading)
				return ok && len(headings) > 0
			},
			desc: "get all headings",
		},
		{
			query: ".code",
			validate: func(result interface{}) bool {
				blocks, ok := result.([]*mq.CodeBlock)
				return ok && len(blocks) == 3
			},
			desc: "get all code blocks",
		},
		{
			query: ".code('python')",
			validate: func(result interface{}) bool {
				blocks, ok := result.([]*mq.CodeBlock)
				return ok && len(blocks) == 1
			},
			desc: "get Python code blocks",
		},
		{
			query: ".section('Section One')",
			validate: func(result interface{}) bool {
				section, ok := result.(*mq.Section)
				return ok && section.Heading.Text == "Section One"
			},
			desc: "get specific section",
		},
		{
			query: ".owner",
			validate: func(result interface{}) bool {
				owner, ok := result.(string)
				return ok && owner == "alice"
			},
			desc: "get document owner",
		},
		{
			query: ".tags",
			validate: func(result interface{}) bool {
				tags, ok := result.([]string)
				return ok && len(tags) == 2
			},
			desc: "get document tags",
		},
	}

	for _, test := range tests {
		plan, err := compiler.CompileString(test.query)
		if err != nil {
			t.Errorf("Failed to compile '%s': %v", test.query, err)
			continue
		}

		ctx := mql.NewEvalContext(doc)
		result, err := plan(ctx)
		if err != nil {
			t.Errorf("Failed to execute '%s': %v", test.query, err)
			continue
		}

		if !test.validate(result) {
			t.Errorf("Validation failed for '%s' (%s): got %T %v",
				test.query, test.desc, result, result)
		}
	}
}

func TestQueryExecution(t *testing.T) {
	// Parse test document
	mqEngine := mq.New()
	doc, err := mqEngine.ParseDocument([]byte(testDoc), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Create MQL engine
	engine := mql.New()

	// Test MQL query execution through engine
	tests := []struct {
		query    string
		validate func(interface{}) bool
		desc     string
	}{
		{
			query: ".headings",
			validate: func(result interface{}) bool {
				headings, ok := result.([]*mq.Heading)
				return ok && len(headings) > 0
			},
			desc: "simple selector",
		},
		{
			query: ".headings | .text",
			validate: func(result interface{}) bool {
				// After .text on a collection, we should get array of strings
				texts, ok := result.([]string)
				return ok && len(texts) == 4 // Test Document, Section One, Section Two, Subsection
			},
			desc: "pipe to text extraction",
		},
		{
			query: ".code('go', 'python')",
			validate: func(result interface{}) bool {
				blocks, ok := result.([]*mq.CodeBlock)
				return ok && len(blocks) == 2
			},
			desc: "multiple language filter",
		},
		{
			query: ".metadata",
			validate: func(result interface{}) bool {
				meta, ok := result.(mq.Metadata)
				return ok && meta != nil
			},
			desc: "get metadata",
		},
	}

	for _, test := range tests {
		result, err := engine.Query(doc, test.query)
		if err != nil {
			t.Errorf("Query '%s' (%s) failed: %v", test.query, test.desc, err)
			continue
		}

		if !test.validate(result) {
			t.Errorf("Validation failed for '%s' (%s): got %T %v",
				test.query, test.desc, result, result)
		}
	}
}

func TestComplexQueries(t *testing.T) {
	// Create a more complex document
	complexDoc := `---
owner: bob
tags: [api, reference, v2]
priority: critical
---

# API Reference v2

## Overview

General API information.

## Authentication

### OAuth2 Flow

OAuth2 implementation details.

` + "```python" + `
# Python OAuth example
client = OAuth2Client(
    client_id="xyz",
    client_secret="secret"
)
token = client.get_token()
# Total lines: 6
` + "```" + `

### API Keys

Simple API key authentication.

` + "```bash" + `
curl -H "X-API-Key: your-key" https://api.example.com
` + "```" + `

## Endpoints

### GET /users

Retrieve user list.

` + "```javascript" + `
fetch('/api/users')
  .then(r => r.json())
  .then(console.log);
` + "```" + `

### POST /users

Create a new user.

` + "```python" + `
# Create user
response = requests.post(
    '/api/users',
    json={'name': 'Alice'}
)
` + "```" + `
`

	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(complexDoc), "api.md")
	if err != nil {
		t.Fatalf("Failed to parse complex document: %v", err)
	}

	// Test nested sections
	authSection, ok := doc.GetSection("Authentication")
	if !ok {
		t.Fatal("Failed to get Authentication section")
	}

	if len(authSection.Children) != 2 {
		t.Errorf("Expected 2 child sections, got %d", len(authSection.Children))
	}

	// Test filtering code blocks by language
	pythonBlocks := doc.GetCodeBlocks("python")
	if len(pythonBlocks) != 2 {
		t.Errorf("Expected 2 Python blocks, got %d", len(pythonBlocks))
	}

	// Test section with code blocks
	endpointsSection, ok := doc.GetSection("Endpoints")
	if !ok {
		t.Fatal("Failed to get Endpoints section")
	}

	sectionCode := endpointsSection.GetCodeBlocks()
	if len(sectionCode) != 2 {
		t.Errorf("Expected 2 code blocks in Endpoints section, got %d", len(sectionCode))
	}
}

func TestQueryWithOwnership(t *testing.T) {
	doc1 := `---
owner: alice
---
# Alice's Document
`

	doc2 := `---
owner: bob
---
# Bob's Document
`

	engine := mq.New()

	aliceDoc, _ := engine.ParseDocument([]byte(doc1), "alice.md")
	bobDoc, _ := engine.ParseDocument([]byte(doc2), "bob.md")

	// Test ownership checks
	if !aliceDoc.CheckOwnership("alice") {
		t.Error("Alice should own alice.md")
	}

	if aliceDoc.CheckOwnership("bob") {
		t.Error("Bob should not own alice.md")
	}

	if !bobDoc.CheckOwnership("bob") {
		t.Error("Bob should own bob.md")
	}

	// Test query builder with ownership
	_, err := engine.From(aliceDoc).
		WhereOwner("alice").
		Headings().
		Execute()
	if err != nil {
		t.Error("Should allow Alice to query her document")
	}

	_, err = engine.From(aliceDoc).
		WhereOwner("charlie").
		Headings().
		Execute()
	if err == nil {
		t.Error("Should not allow Charlie to query Alice's document")
	}
}

func TestAdditionalSelectors(t *testing.T) {
	// Parse test document
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(testDoc), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Create MQL engine and compiler
	mqlEngine := mql.New()
	compiler := mql.NewCompiler()

	tests := []struct {
		query    string
		validate func(interface{}) bool
		desc     string
	}{
		{
			query: ".sections",
			validate: func(result interface{}) bool {
				sections, ok := result.([]*mq.Section)
				return ok && len(sections) > 0
			},
			desc: "get all sections",
		},
		{
			query: ".links",
			validate: func(result interface{}) bool {
				_, ok := result.([]*mq.Link)
				return ok // testDoc doesn't have links, so just check type
			},
			desc: "get all links",
		},
		{
			query: ".images",
			validate: func(result interface{}) bool {
				_, ok := result.([]*mq.Image)
				return ok // testDoc doesn't have images, so just check type
			},
			desc: "get all images",
		},
		{
			query: ".tables",
			validate: func(result interface{}) bool {
				_, ok := result.([]*mq.Table)
				return ok // testDoc doesn't have tables, so just check type
			},
			desc: "get all tables",
		},
		{
			query: ".lists",
			validate: func(result interface{}) bool {
				_, ok := result.([]*mq.List)
				return ok // testDoc doesn't have lists, so just check type
			},
			desc: "get all lists",
		},
		{
			query: ".priority",
			validate: func(result interface{}) bool {
				priority, ok := result.(string)
				return ok && priority == "medium"
			},
			desc: "get document priority",
		},
	}

	for _, test := range tests {
		// Test through compiler
		plan, err := compiler.CompileString(test.query)
		if err != nil {
			t.Errorf("Failed to compile '%s' (%s): %v", test.query, test.desc, err)
			continue
		}

		ctx := mql.NewEvalContext(doc)
		result, err := plan(ctx)
		if err != nil {
			t.Errorf("Failed to execute '%s' (%s): %v", test.query, test.desc, err)
			continue
		}

		if !test.validate(result) {
			t.Errorf("Validation failed for '%s' (%s): got %T %v",
				test.query, test.desc, result, result)
		}

		// Also test through engine
		engineResult, err := mqlEngine.Query(doc, test.query)
		if err != nil {
			t.Errorf("Engine query '%s' (%s) failed: %v", test.query, test.desc, err)
			continue
		}

		if !test.validate(engineResult) {
			t.Errorf("Engine validation failed for '%s' (%s): got %T %v",
				test.query, test.desc, engineResult, engineResult)
		}
	}
}

func TestFunctionsAndOperations(t *testing.T) {
	// Create a document with varied content for testing
	docContent := `---
owner: test-user
tags: [api, testing]
priority: high
---

# Main Title

## First Section

This section contains some text with keywords.

### Subsection

More content here.

## API Documentation

The API provides various endpoints.

## Testing Section

This is for testing purposes.
`

	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(docContent), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	compiler := mql.NewCompiler()

	tests := []struct {
		query    string
		validate func(interface{}) bool
		desc     string
	}{
		{
			query: ".headings | select(.level == 2)",
			validate: func(result interface{}) bool {
				headings, ok := result.([]*mq.Heading)
				if !ok {
					return false
				}
				// Check all headings are level 2
				for _, h := range headings {
					if h.Level != 2 {
						return false
					}
				}
				return len(headings) == 3 // First Section, API Documentation, Testing Section
			},
			desc: "filter headings by level",
		},
		{
			query: ".headings | filter(.level <= 2)",
			validate: func(result interface{}) bool {
				headings, ok := result.([]*mq.Heading)
				if !ok {
					return false
				}
				for _, h := range headings {
					if h.Level > 2 {
						return false
					}
				}
				return true
			},
			desc: "filter headings with comparison",
		},
		// Skipping nested property access for now - needs parser update
		// {
		// 	query: `.sections | select(.heading.text == "First Section")`,
		// 	validate: func(result interface{}) bool {
		// 		sections, ok := result.([]*mq.Section)
		// 		return ok && len(sections) == 1 && sections[0].Heading.Text == "First Section"
		// 	},
		// 	desc: "filter sections by heading text",
		// },
	}

	for _, test := range tests {
		plan, err := compiler.CompileString(test.query)
		if err != nil {
			t.Errorf("Failed to compile '%s' (%s): %v", test.query, test.desc, err)
			continue
		}

		ctx := mql.NewEvalContext(doc)
		result, err := plan(ctx)
		if err != nil {
			t.Errorf("Failed to execute '%s' (%s): %v", test.query, test.desc, err)
			continue
		}

		if !test.validate(result) {
			t.Errorf("Validation failed for '%s' (%s): got %T %v",
				test.query, test.desc, result, result)
		}
	}
}

func TestComplexPipelines(t *testing.T) {
	// Create a rich document for testing
	docContent := `---
owner: developer
tags: [golang, testing, documentation]
priority: critical
---

# Project Documentation

## Installation Guide

Install using the following command:

` + "```bash" + `
go get github.com/example/project
` + "```" + `

## API Reference

### Authentication

The API uses OAuth2 for authentication.

` + "```go" + `
func Authenticate(token string) error {
    // Implementation
    return nil
}
` + "```" + `

### Endpoints

Various endpoints are available:

` + "```python" + `
def get_users():
    return []

def create_user(name):
    pass
` + "```" + `

## Testing

Run tests with:

` + "```bash" + `
go test ./...
` + "```" + `
`

	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(docContent), "project.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	mqlEngine := mql.New()

	tests := []struct {
		query       string
		validate    func(interface{}) bool
		shouldError bool
		desc        string
	}{
		{
			query: `.section("API Reference") | .code`,
			validate: func(result interface{}) bool {
				blocks, ok := result.([]*mq.CodeBlock)
				return ok && len(blocks) == 2 // go and python blocks
			},
			desc: "get code blocks from specific section",
		},
		{
			query: `.section("API Reference") | .code("python")`,
			validate: func(result interface{}) bool {
				blocks, ok := result.([]*mq.CodeBlock)
				// The section should have only 1 Python block
				return ok && len(blocks) == 1 && blocks[0].Language == "python"
			},
			desc: "filter code blocks by language in section",
		},
		{
			query: `.code | .text`,
			validate: func(result interface{}) bool {
				texts, ok := result.([]string)
				return ok && len(texts) == 4 // All code block contents
			},
			desc: "extract text from code blocks",
		},
		{
			query: `.headings | filter(.level == 2) | .text`,
			validate: func(result interface{}) bool {
				texts, ok := result.([]string)
				if !ok {
					return false
				}
				expected := []string{"Installation Guide", "API Reference", "Testing"}
				if len(texts) != len(expected) {
					return false
				}
				for i, text := range texts {
					if text != expected[i] {
						return false
					}
				}
				return true
			},
			desc: "complex pipeline with filter and text extraction",
		},
		{
			query: `.sections | .heading | .text`,
			validate: func(result interface{}) bool {
				// This should extract heading text from all sections
				texts, ok := result.([]string)
				return ok && len(texts) > 0
			},
			desc: "chain property access through sections",
		},
	}

	for _, test := range tests {
		result, err := mqlEngine.Query(doc, test.query)
		if test.shouldError {
			if err == nil {
				t.Errorf("Expected error for '%s' (%s), but got none", test.query, test.desc)
			}
			continue
		}

		if err != nil {
			t.Errorf("Query '%s' (%s) failed: %v", test.query, test.desc, err)
			continue
		}

		if !test.validate(result) {
			t.Errorf("Validation failed for '%s' (%s): got %T %v",
				test.query, test.desc, result, result)
		}
	}
}
