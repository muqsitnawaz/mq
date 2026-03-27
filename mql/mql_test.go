package mql_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/muqsitnawaz/mq/mql"
	"github.com/muqsitnawaz/mq/pdf"
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
		{
			query: ".text",
			validate: func(result interface{}) bool {
				text, ok := result.(string)
				return ok && text == testDoc
			},
			desc: "document-level .text returns markdown source via readable text",
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

func TestHeadingLevelSelectors(t *testing.T) {
	mqEngine := mq.New()
	doc, err := mqEngine.ParseDocument([]byte(testDoc), "test.md")
	require.NoError(t, err)

	engine := mql.New()

	// .h2 with no args returns headings at level 2
	result, err := engine.Query(doc, ".h2")
	require.NoError(t, err)
	headings, ok := result.([]*mq.Heading)
	require.True(t, ok)
	assert.Len(t, headings, 2) // Section One, Section Two

	// .h2("Section One") returns the full section
	result, err = engine.Query(doc, `.h2("Section One")`)
	require.NoError(t, err)
	section, ok := result.(*mq.Section)
	require.True(t, ok)
	assert.Equal(t, "Section One", section.Heading.Text)

	// .h2("section") - case-insensitive prefix matches "Section One" (first in order)
	result, err = engine.Query(doc, `.h2("section")`)
	require.NoError(t, err)
	section, ok = result.(*mq.Section)
	require.True(t, ok)
	assert.Equal(t, "Section One", section.Heading.Text)

	// .h3("Sub") - prefix match on H3
	result, err = engine.Query(doc, `.h3("Sub")`)
	require.NoError(t, err)
	section, ok = result.(*mq.Section)
	require.True(t, ok)
	assert.Equal(t, "Subsection", section.Heading.Text)

	// .h3("Section One") - wrong level, should fail
	_, err = engine.Query(doc, `.h3("Section One")`)
	assert.Error(t, err, "H2 heading should not be found at level 3")

	// .h2("Section One") | .text - pipe to text
	result, err = engine.Query(doc, `.h2("Section One") | .text`)
	require.NoError(t, err)
	text, ok := result.(string)
	require.True(t, ok)
	assert.Contains(t, text, "first section")
}

func TestQueryExecutionPDFDocumentText(t *testing.T) {
	minimalPDF := []byte(`%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>
endobj
4 0 obj
<< /Length 44 >>
stream
BT /F1 12 Tf 100 700 Td (Hello PDF) Tj ET
endstream
endobj
xref
0 5
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
0000000214 00000 n
trailer
<< /Size 5 /Root 1 0 R >>
startxref
312
%%EOF`)

	doc, err := pdf.NewParser().Parse(minimalPDF, "test.pdf")
	require.NoError(t, err)

	engine := mql.New()
	result, err := engine.Query(doc, ".text")
	require.NoError(t, err)

	text, ok := result.(string)
	require.True(t, ok, "expected string, got %T", result)
	assert.Equal(t, doc.ReadableText(), text)
	assert.False(t, strings.HasPrefix(text, "&{"), "expected readable text, got struct dump")
}

func TestRemovedRecordSelector(t *testing.T) {
	engine := mql.New()
	defer engine.Close()

	doc, err := engine.ParseDocument([]byte(`{"event":"x"}`), "events.jsonl")
	require.NoError(t, err)

	_, err = engine.Query(doc, `.record(1)`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown selector: .record")
}

func TestSearchPipelineOnJSONL(t *testing.T) {
	jsonl := strings.Join([]string{
		`{"event":"tool_use_start","message":"first tool_use event"}`,
		`{"event":"ignore","message":"other event"}`,
		`{"event":"tool_use_finish","message":"second tool_use event"}`,
	}, "\n")

	engine := mql.New()
	defer engine.Close()

	doc, err := engine.ParseDocument([]byte(jsonl), "events.jsonl")
	require.NoError(t, err)

	textResult, err := engine.Query(doc, `.search("tool_use") | .text`)
	require.NoError(t, err)

	texts, ok := textResult.([]string)
	require.True(t, ok, "expected []string, got %T", textResult)
	require.Len(t, texts, 2)
	assert.Contains(t, texts[0], "event: tool_use_start")
	assert.Contains(t, texts[0], "message: first tool_use event")
	assert.Contains(t, texts[1], "event: tool_use_finish")
	assert.NotContains(t, texts[0], "Found 2 matches")
	assert.NotContains(t, texts[1], "Found 2 matches")

	lengthResult, err := engine.Query(doc, `.search("tool_use") | .length`)
	require.NoError(t, err)
	assert.Equal(t, 2, lengthResult)

	treeResult, err := engine.Query(doc, `.search("tool_use") | .tree`)
	require.NoError(t, err)

	tree, ok := treeResult.(*mq.SearchTreeResult)
	require.True(t, ok, "expected *mq.SearchTreeResult, got %T", treeResult)
	rendered := tree.String()
	assert.Contains(t, rendered, "[line 1] event: tool_use_start")
	assert.Contains(t, rendered, "message: first tool_use event")
	assert.Contains(t, rendered, "[line 3] event: tool_use_finish")

	rawResult, err := engine.Query(doc, `.search("tool_use") | .raw`)
	require.NoError(t, err)

	raws, ok := rawResult.([]string)
	require.True(t, ok, "expected []string, got %T", rawResult)
	require.Len(t, raws, 2)
	assert.Equal(t, `{"event":"tool_use_start","message":"first tool_use event"}`, raws[0])
	assert.Equal(t, `{"event":"tool_use_finish","message":"second tool_use event"}`, raws[1])

	nthResult, err := engine.Query(doc, `.search("tool_use") | .nth(1) | .text`)
	require.NoError(t, err)
	text, ok := nthResult.(string)
	require.True(t, ok, "expected string, got %T", nthResult)
	assert.Contains(t, text, "event: tool_use_finish")

	nthRawResult, err := engine.Query(doc, `.search("tool_use") | .nth(1) | .raw`)
	require.NoError(t, err)
	raw, ok := nthRawResult.(string)
	require.True(t, ok, "expected string, got %T", nthRawResult)
	assert.Equal(t, `{"event":"tool_use_finish","message":"second tool_use event"}`, raw)
}

func TestStructuredDocumentTextAndRaw(t *testing.T) {
	engine := mql.New()
	defer engine.Close()

	t.Run("json", func(t *testing.T) {
		source := []byte(`{"event":"Needle in json payload","meta":{"user":"muqsit"},"ok":true}`)
		doc, err := engine.ParseDocument(source, "config.json")
		require.NoError(t, err)

		textResult, err := engine.Query(doc, `.text`)
		require.NoError(t, err)
		text, ok := textResult.(string)
		require.True(t, ok, "expected string, got %T", textResult)
		assert.Equal(t, "event: Needle in json payload\nmeta.user: muqsit\nok: true", text)

		rawResult, err := engine.Query(doc, `.raw`)
		require.NoError(t, err)
		raw, ok := rawResult.(string)
		require.True(t, ok, "expected string, got %T", rawResult)
		assert.Equal(t, string(source), raw)
	})

	t.Run("yaml", func(t *testing.T) {
		source := []byte("service:\n  name: web\n  enabled: true\n")
		doc, err := engine.ParseDocument(source, "deploy.yaml")
		require.NoError(t, err)

		textResult, err := engine.Query(doc, `.text`)
		require.NoError(t, err)
		text, ok := textResult.(string)
		require.True(t, ok, "expected string, got %T", textResult)
		assert.Equal(t, "service.enabled: true\nservice.name: web", text)

		rawResult, err := engine.Query(doc, `.raw`)
		require.NoError(t, err)
		raw, ok := rawResult.(string)
		require.True(t, ok, "expected string, got %T", rawResult)
		assert.Equal(t, string(source), raw)
	})

	t.Run("jsonl document", func(t *testing.T) {
		source := []byte("{\"event\":\"first\"}\n{\"event\":\"second\",\"ok\":true}\n")
		doc, err := engine.ParseDocument(source, "events.jsonl")
		require.NoError(t, err)

		textResult, err := engine.Query(doc, `.text`)
		require.NoError(t, err)
		text, ok := textResult.(string)
		require.True(t, ok, "expected string, got %T", textResult)
		assert.Equal(t, "[0].event: first\n[1].event: second\n[1].ok: true", text)

		rawResult, err := engine.Query(doc, `.raw`)
		require.NoError(t, err)
		raw, ok := rawResult.(string)
		require.True(t, ok, "expected string, got %T", rawResult)
		assert.Equal(t, string(source), raw)
	})
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

// -------------------------------------------------------------------
// Cache integration tests (TOCTOU regression)
// -------------------------------------------------------------------

func writeTestMD(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

const richMD = "---\ntitle: Cache Test\nowner: test-team\ntags: [cache, integration]\npriority: high\n---\n\n# Cache Test\n\nOverview of the caching system.\n\n## Architecture\n\nThe cache uses bbolt for storage.\n\n```go\nfunc main() {\n    fmt.Println(\"hello\")\n}\n```\n\n## Configuration\n\n- Set cache path via environment\n- Default TTL is 5 days\n\n| Setting | Default |\n|---------|---------|\n| path    | ~/.cache/mq |\n| ttl     | 5d      |\n"

func TestLoadDocumentCacheRoundtrip(t *testing.T) {
	engine := mql.New()
	defer engine.Close()

	dir := t.TempDir()
	path := writeTestMD(t, dir, "doc.md", richMD)

	// First load — cache miss, parses fresh
	doc1, err := engine.LoadDocument(path)
	require.NoError(t, err)

	// Second load — cache hit
	doc2, err := engine.LoadDocument(path)
	require.NoError(t, err)

	// Both must produce identical results
	assert.Equal(t, doc1.Title(), doc2.Title())
	assert.Equal(t, doc1.Format(), doc2.Format())
	assert.Equal(t, doc1.ReadableText(), doc2.ReadableText())

	h1, h2 := doc1.GetHeadings(), doc2.GetHeadings()
	require.Equal(t, len(h1), len(h2))
	for i := range h1 {
		assert.Equal(t, h1[i].Text, h2[i].Text)
		assert.Equal(t, h1[i].Level, h2[i].Level)
	}

	s1, s2 := doc1.GetSections(), doc2.GetSections()
	require.Equal(t, len(s1), len(s2))
	for i := range s1 {
		assert.Equal(t, s1[i].GetText(), s2[i].GetText(), "section %d text mismatch", i)
	}

	assert.Equal(t, len(doc1.GetCodeBlocks()), len(doc2.GetCodeBlocks()))
	assert.Equal(t, len(doc1.GetLinks()), len(doc2.GetLinks()))
	assert.Equal(t, len(doc1.GetTables()), len(doc2.GetTables()))
	assert.Equal(t, len(doc1.GetLists(nil)), len(doc2.GetLists(nil)))
}

func TestLoadDocumentCacheInvalidation(t *testing.T) {
	engine := mql.New()
	defer engine.Close()

	dir := t.TempDir()
	path := writeTestMD(t, dir, "doc.md", "# Version 1\n\n## Old Section\nOld content.\n")

	doc1, err := engine.LoadDocument(path)
	require.NoError(t, err)
	assert.Equal(t, "Version 1", doc1.Title())

	// Modify the file
	time.Sleep(15 * time.Millisecond)
	require.NoError(t, os.WriteFile(path, []byte("# Version 2\n\n## New Section\nNew content.\n"), 0644))

	doc2, err := engine.LoadDocument(path)
	require.NoError(t, err)
	assert.Equal(t, "Version 2", doc2.Title())

	_, ok := doc2.GetSection("New Section")
	assert.True(t, ok, "modified doc should have 'New Section'")
	_, ok = doc2.GetSection("Old Section")
	assert.False(t, ok, "modified doc should NOT have 'Old Section'")
}

func TestLoadDocumentSectionTextMatchesSource(t *testing.T) {
	engine := mql.New()
	defer engine.Close()

	md := "# Guide\n\n## Setup\nRun `go install` to get started.\n\n## Usage\nCall `mq file.md '.section(\"X\")'` to query.\n"
	dir := t.TempDir()
	path := writeTestMD(t, dir, "guide.md", md)

	// Fresh parse
	fresh, err := engine.LoadDocument(path)
	require.NoError(t, err)

	setupFresh, ok := fresh.GetSection("Setup")
	require.True(t, ok)
	freshText := setupFresh.GetText()
	assert.Contains(t, freshText, "go install")

	// Cached parse
	cached, err := engine.LoadDocument(path)
	require.NoError(t, err)

	setupCached, ok := cached.GetSection("Setup")
	require.True(t, ok)
	cachedText := setupCached.GetText()

	// Critical assertion: section text must be identical from cache vs fresh parse
	assert.Equal(t, freshText, cachedText, "section text must be identical from cache vs fresh parse")
}
