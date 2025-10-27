package mql_test

import (
	"strings"
	"testing"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/muqsitnawaz/mq/mql"
)

const docWithFrontmatter = `---
title: API Documentation
author: Alice
version: 2.0.0
status: draft
tags: [api, documentation, v2]
config:
  theme: dark
  sidebar:
    enabled: true
    position: left
  settings:
    font_size: 14
nested:
  deep:
    value: 123
---

# API Reference v2.0

## Authentication

Use OAuth2 for authentication.

` + "```python" + `
def authenticate(token):
    return validate(token)
` + "```" + `

## Endpoints

### GET /users

Returns all users.
`

const docWithoutFrontmatter = `# Simple Document

## Section One

No frontmatter here.
`

const docFrontmatterOnly = `---
title: Metadata Only
author: Bob
version: 1.0
---`

func TestDirectFieldAccess(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(docWithFrontmatter), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	mqlEngine := mql.New()

	tests := []struct {
		query    string
		expected interface{}
		desc     string
	}{
		{
			query:    ".title",
			expected: "API Documentation",
			desc:     "direct field access",
		},
		{
			query:    ".author",
			expected: "Alice",
			desc:     "direct author field",
		},
		{
			query:    ".version",
			expected: "2.0.0",
			desc:     "direct version field",
		},
		{
			query:    ".status",
			expected: "draft",
			desc:     "direct status field",
		},
	}

	for _, test := range tests {
		result, err := mqlEngine.Query(doc, test.query)
		if err != nil {
			t.Errorf("Query '%s' (%s) failed: %v", test.query, test.desc, err)
			continue
		}

		if result != test.expected {
			t.Errorf("Query '%s' (%s): expected %v, got %v",
				test.query, test.desc, test.expected, result)
		}
	}
}

func TestNestedFieldAccess(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(docWithFrontmatter), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	mqlEngine := mql.New()

	tests := []struct {
		query    string
		validate func(interface{}) bool
		desc     string
	}{
		{
			query: ".config",
			validate: func(result interface{}) bool {
				// YAML may return map[string]interface{} or map[interface{}]interface{}
				if m, ok := result.(map[string]interface{}); ok {
					return m != nil
				}
				if m, ok := result.(map[interface{}]interface{}); ok {
					return m != nil
				}
				return false
			},
			desc: "access nested config object",
		},
		{
			query: ".config | .theme",
			validate: func(result interface{}) bool {
				theme, ok := result.(string)
				return ok && theme == "dark"
			},
			desc: "drill into config.theme",
		},
		{
			query: ".config | .sidebar",
			validate: func(result interface{}) bool {
				// YAML may return map[string]interface{} or map[interface{}]interface{}
				if m, ok := result.(map[string]interface{}); ok {
					return m != nil
				}
				if m, ok := result.(map[interface{}]interface{}); ok {
					return m != nil
				}
				return false
			},
			desc: "access nested sidebar object",
		},
		{
			query: ".config | .sidebar | .enabled",
			validate: func(result interface{}) bool {
				enabled, ok := result.(bool)
				return ok && enabled == true
			},
			desc: "drill into config.sidebar.enabled",
		},
		{
			query: ".config | .sidebar | .position",
			validate: func(result interface{}) bool {
				pos, ok := result.(string)
				return ok && pos == "left"
			},
			desc: "drill into config.sidebar.position",
		},
		{
			query: ".nested | .deep | .value",
			validate: func(result interface{}) bool {
				// YAML may parse as int or int64
				switch v := result.(type) {
				case int:
					return v == 123
				case int64:
					return v == 123
				case float64:
					return v == 123.0
				default:
					return false
				}
			},
			desc: "deeply nested value access",
		},
	}

	for _, test := range tests {
		result, err := mqlEngine.Query(doc, test.query)
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

func TestContentSelectors(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(docWithFrontmatter), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	mqlEngine := mql.New()

	tests := []struct {
		query    string
		validate func(interface{}) bool
		desc     string
	}{
		{
			query: ".source",
			validate: func(result interface{}) bool {
				source, ok := result.(string)
				return ok && source == docWithFrontmatter
			},
			desc: "get source with frontmatter",
		},
		{
			query: ".frontmatter",
			validate: func(result interface{}) bool {
				fm, ok := result.(string)
				if !ok {
					return false
				}
				// Should contain the YAML frontmatter
				return strings.HasPrefix(fm, "---\n") &&
					strings.Contains(fm, "title: API Documentation") &&
					strings.Contains(fm, "author: Alice")
			},
			desc: "get frontmatter only",
		},
		{
			query: ".body",
			validate: func(result interface{}) bool {
				body, ok := result.(string)
				if !ok {
					return false
				}
				// Should start with the heading, not frontmatter
				return strings.HasPrefix(body, "\n# API Reference") &&
					!strings.Contains(body, "---\ntitle:")
			},
			desc: "get body without frontmatter",
		},
		{
			query: ".text",
			validate: func(result interface{}) bool {
				text, ok := result.(string)
				if !ok {
					return false
				}
				// Should contain plain text content
				return strings.Contains(text, "API Reference") &&
					strings.Contains(text, "Authentication") &&
					!strings.Contains(text, "---") &&
					!strings.Contains(text, "```")
			},
			desc: "get plain text content",
		},
	}

	for _, test := range tests {
		result, err := mqlEngine.Query(doc, test.query)
		if err != nil {
			t.Errorf("Query '%s' (%s) failed: %v", test.query, test.desc, err)
			continue
		}

		if !test.validate(result) {
			t.Errorf("Validation failed for '%s' (%s): got %T",
				test.query, test.desc, result)
		}
	}
}

func TestDocumentWithoutFrontmatter(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(docWithoutFrontmatter), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	mqlEngine := mql.New()

	tests := []struct {
		query    string
		validate func(interface{}) bool
		desc     string
	}{
		{
			query: ".title",
			validate: func(result interface{}) bool {
				// Should return nil or empty for non-existent field
				return result == nil || result == ""
			},
			desc: "non-existent field returns nil",
		},
		{
			query: ".frontmatter",
			validate: func(result interface{}) bool {
				fm, ok := result.(string)
				return ok && fm == ""
			},
			desc: "no frontmatter returns empty string",
		},
		{
			query: ".body",
			validate: func(result interface{}) bool {
				body, ok := result.(string)
				// When no frontmatter, body should be entire document
				return ok && strings.HasPrefix(body, "# Simple Document")
			},
			desc: "body is entire document when no frontmatter",
		},
		{
			query: ".source",
			validate: func(result interface{}) bool {
				source, ok := result.(string)
				return ok && source == docWithoutFrontmatter
			},
			desc: "source returns full document",
		},
	}

	for _, test := range tests {
		result, err := mqlEngine.Query(doc, test.query)
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

func TestFrontmatterOnlyDocument(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(docFrontmatterOnly), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	mqlEngine := mql.New()

	tests := []struct {
		query    string
		validate func(interface{}) bool
		desc     string
	}{
		{
			query: ".title",
			validate: func(result interface{}) bool {
				title, ok := result.(string)
				return ok && title == "Metadata Only"
			},
			desc: "can access frontmatter fields",
		},
		{
			query: ".body",
			validate: func(result interface{}) bool {
				body, ok := result.(string)
				// Body should be empty or just whitespace
				return ok && strings.TrimSpace(body) == ""
			},
			desc: "body is empty when only frontmatter",
		},
		{
			query: ".frontmatter",
			validate: func(result interface{}) bool {
				fm, ok := result.(string)
				return ok && strings.Contains(fm, "title: Metadata Only")
			},
			desc: "frontmatter is accessible",
		},
	}

	for _, test := range tests {
		result, err := mqlEngine.Query(doc, test.query)
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

func TestReservedKeywordConflicts(t *testing.T) {
	docWithConflict := `---
sections: 5
headings: important
code: custom_value
custom: no_conflict
---

# Test Document

## Section One
`

	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(docWithConflict), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	mqlEngine := mql.New()

	tests := []struct {
		query    string
		validate func(interface{}) bool
		desc     string
	}{
		{
			query: ".sections",
			validate: func(result interface{}) bool {
				// Should return structural sections, not frontmatter value
				sections, ok := result.([]*mq.Section)
				return ok && len(sections) > 0
			},
			desc: "structural selector takes precedence",
		},
		{
			query: ".headings",
			validate: func(result interface{}) bool {
				// Should return structural headings
				headings, ok := result.([]*mq.Heading)
				return ok && len(headings) > 0
			},
			desc: "headings selector returns structural data",
		},
		{
			query: ".code",
			validate: func(result interface{}) bool {
				// Should return structural code blocks (empty array in this case)
				_, ok := result.([]*mq.CodeBlock)
				return ok // OK if empty array
			},
			desc: "code selector returns structural data",
		},
		{
			query: ".metadata | .sections",
			validate: func(result interface{}) bool {
				// Should access frontmatter value via .metadata escape hatch
				switch v := result.(type) {
				case int:
					return v == 5
				case int64:
					return v == 5
				case float64:
					return v == 5.0
				default:
					return false
				}
			},
			desc: "frontmatter value accessible via .metadata",
		},
		{
			query: ".custom",
			validate: func(result interface{}) bool {
				// Non-conflicting field should work directly
				custom, ok := result.(string)
				return ok && custom == "no_conflict"
			},
			desc: "non-conflicting field works directly",
		},
	}

	for _, test := range tests {
		result, err := mqlEngine.Query(doc, test.query)
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

func TestComplexNestedQueries(t *testing.T) {
	engine := mq.New()
	doc, err := engine.ParseDocument([]byte(docWithFrontmatter), "test.md")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	mqlEngine := mql.New()

	tests := []struct {
		query    string
		validate func(interface{}) bool
		desc     string
	}{
		{
			query: ".tags",
			validate: func(result interface{}) bool {
				// tags should be accessible (already existed)
				tags, ok := result.([]string)
				return ok && len(tags) == 3
			},
			desc: "tags field works as before",
		},
		{
			query: ".config | .settings",
			validate: func(result interface{}) bool {
				// YAML may return map[string]interface{} or map[interface{}]interface{}
				if settings, ok := result.(map[string]interface{}); ok {
					fontSize, exists := settings["font_size"]
					return exists && fontSize != nil
				}
				if settings, ok := result.(map[interface{}]interface{}); ok {
					// Check with string key
					fontSize, exists := settings["font_size"]
					return exists && fontSize != nil
				}
				return false
			},
			desc: "deeply nested settings object",
		},
	}

	for _, test := range tests {
		result, err := mqlEngine.Query(doc, test.query)
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
