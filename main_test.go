package main

import (
	"os"
	"path/filepath"
	"testing"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMethodCall(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantMethod string
		wantArg    string
		wantOk     bool
	}{
		// Basic cases
		{"bare method", ".tree", "tree", "", true},
		{"method no arg", ".search", "search", "", true},

		// Double quotes
		{"double quotes", `.tree("full")`, "tree", "full", true},
		{"double quotes preview", `.tree("preview")`, "tree", "preview", true},
		{"double quotes search", `.search("term")`, "search", "term", true},

		// Single quotes (Windows-friendly)
		{"single quotes", `.tree('full')`, "tree", "full", true},
		{"single quotes search", `.search('test query')`, "search", "test query", true},

		// No quotes (Windows CMD may strip them)
		{"no quotes", ".tree(full)", "tree", "full", true},
		{"no quotes search", ".search(term)", "search", "term", true},

		// Edge cases
		{"empty arg with quotes", `.tree("")`, "tree", "", true},
		{"empty arg no quotes", ".tree()", "tree", "", true},
		{"arg with spaces", `.search("hello world")`, "search", "hello world", true},

		// Invalid cases
		{"no dot prefix", "tree", "", "", false},
		{"missing close paren", ".tree(full", "", "", false},
		{"random text", "invalid", "", "", false},
		{"empty string", "", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMethod, gotArg, gotOk := parseMethodCall(tt.input)
			if gotMethod != tt.wantMethod {
				t.Errorf("parseMethodCall(%q) method = %q, want %q", tt.input, gotMethod, tt.wantMethod)
			}
			if gotArg != tt.wantArg {
				t.Errorf("parseMethodCall(%q) arg = %q, want %q", tt.input, gotArg, tt.wantArg)
			}
			if gotOk != tt.wantOk {
				t.Errorf("parseMethodCall(%q) ok = %v, want %v", tt.input, gotOk, tt.wantOk)
			}
		})
	}
}

func TestRunDirectoryQuerySearchPipeline(t *testing.T) {
	dir := t.TempDir()
	content := `{"level":"error","server_name":"notion","message":"MCP server notion requires OAuth: "}` + "\n" +
		`{"level":"info","message":"all good"}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "logs.jsonl"), []byte(content), 0o644))

	t.Run("text", func(t *testing.T) {
		result, err := runDirectoryQuery(dir, `.search("requires OAuth") | .text`)
		require.NoError(t, err)

		texts, ok := result.([]string)
		require.True(t, ok, "expected []string, got %T", result)
		require.Len(t, texts, 1)
		assert.Contains(t, texts[0], `"server_name": "notion"`)
		assert.Contains(t, texts[0], `"level": "error"`)
	})

	t.Run("tree", func(t *testing.T) {
		result, err := runDirectoryQuery(dir, `.search("requires OAuth") | .tree`)
		require.NoError(t, err)

		tree, ok := result.(*mq.SearchTreeResult)
		require.True(t, ok, "expected *mq.SearchTreeResult, got %T", result)
		rendered := tree.String()
		assert.Contains(t, rendered, "[line 1] level: error")
		assert.Contains(t, rendered, "server_name: notion")
		assert.Contains(t, rendered, "message: MCP server notion requires OAuth:")
	})

	t.Run("length", func(t *testing.T) {
		result, err := runDirectoryQuery(dir, `.search("requires OAuth") | .length`)
		require.NoError(t, err)
		assert.Equal(t, 1, result)
	})

	t.Run("nth", func(t *testing.T) {
		result, err := runDirectoryQuery(dir, `.search("requires OAuth") | .nth(0) | .text`)
		require.NoError(t, err)

		text, ok := result.(string)
		require.True(t, ok, "expected string, got %T", result)
		assert.Contains(t, text, `"server_name": "notion"`)
	})
}
