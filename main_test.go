package main

import (
	"bytes"
	"io"
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
		assert.Contains(t, texts[0], "level: error")
		assert.Contains(t, texts[0], "server_name: notion")
	})

	t.Run("raw", func(t *testing.T) {
		result, err := runDirectoryQuery(dir, `.search("requires OAuth") | .raw`)
		require.NoError(t, err)

		raws, ok := result.([]string)
		require.True(t, ok, "expected []string, got %T", result)
		require.Len(t, raws, 1)
		assert.Equal(t, `{"level":"error","server_name":"notion","message":"MCP server notion requires OAuth: "}`, raws[0])
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
		result, err := runDirectoryQuery(dir, `.search("requires OAuth") | .nth(0)`)
		require.NoError(t, err)

		match, ok := result.(*mq.SearchResult)
		require.True(t, ok, "expected *mq.SearchResult, got %T", result)
		assert.Equal(t, `{"level":"error","server_name":"notion","message":"MCP server notion requires OAuth: "}`, match.RawContent())

		rendered := captureOutput(t, func() {
			displayResult(match)
		})
		assert.Equal(t, "{\"level\":\"error\",\"server_name\":\"notion\",\"message\":\"MCP server notion requires OAuth: \"}\n", rendered)
	})
}

func TestDisplayResultSearchMatchUsesRawContent(t *testing.T) {
	dir := t.TempDir()
	content := "# Guide\n\nNeedle in docs.\n\n## Details\n\nMore context.\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "guide.md"), []byte(content), 0o644))

	result, err := runDirectoryQuery(dir, `.search("Needle") | .nth(0)`)
	require.NoError(t, err)

	match, ok := result.(*mq.SearchResult)
	require.True(t, ok, "expected *mq.SearchResult, got %T", result)
	assert.Contains(t, match.RawContent(), "# Guide")
	assert.Contains(t, match.RawContent(), "Needle in docs.")

	rendered := captureOutput(t, func() {
		displayResult(match)
	})
	assert.Contains(t, rendered, "# Guide")
	assert.Contains(t, rendered, "Needle in docs.")
	assert.NotContains(t, rendered, "Result type: *mq.SearchResult")
}

func TestDisplayResultDirectoryTreeUsesTreeString(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "guide.md"), []byte("# Guide\n\n## API\n\nNeedle in docs.\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "page.html"), []byte("<!DOCTYPE html><html><body><main><h1>Overview</h1><p>Needle in html content.</p></main></body></html>"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.json"), []byte(`{"name":"json doc","content":"Needle in json payload"}`), 0o644))

	result, err := runDirectoryQuery(dir, ".tree")
	require.NoError(t, err)

	tree, ok := result.(*mq.DirTreeResult)
	require.True(t, ok, "expected *mq.DirTreeResult, got %T", result)
	require.Contains(t, tree.String(), "guide.md (6 lines, 2 sections)")
	require.Contains(t, tree.String(), "# Guide")
	require.Contains(t, tree.String(), "page.html")
	require.Contains(t, tree.String(), "H1 Overview")
	require.Contains(t, tree.String(), "data.json")
	require.Contains(t, tree.String(), "key content")

	rendered := captureOutput(t, func() {
		displayResult(tree)
	})
	assert.Equal(t, tree.String(), rendered)
	assert.NotContains(t, rendered, "Result type: *mq.DirTreeResult")
}

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	require.NoError(t, w.Close())

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return buf.String()
}

func TestParseTrim(t *testing.T) {
	cases := []struct {
		in       string
		wantN    int
		wantUnit string
		wantTail bool
		wantFull bool
		wantErr  bool
	}{
		{in: "4L", wantN: 4, wantUnit: "L"},
		{in: "3P", wantN: 3, wantUnit: "P"},
		{in: "200C", wantN: 200, wantUnit: "C"},
		{in: "2s", wantN: 2, wantUnit: "S"},
		{in: "5", wantN: 5, wantUnit: "L"}, // bare number -> default unit
		{in: "-3L", wantN: 3, wantUnit: "L", wantTail: true},
		{in: "0", wantN: 0, wantUnit: "L"},
		{in: "full", wantFull: true},
		{in: "all", wantFull: true},
		{in: "3x", wantErr: true},  // trailing garbage
		{in: "3xL", wantErr: true}, // bad unit letter
		{in: "abc", wantErr: true},
	}
	for _, c := range cases {
		var opts mq.TreeOptions
		err := parseTrim(c.in, &opts)
		if c.wantErr {
			assert.Error(t, err, "parseTrim(%q) should error", c.in)
			continue
		}
		require.NoError(t, err, "parseTrim(%q)", c.in)
		assert.Equal(t, c.wantFull, opts.TrimFull, "%q full", c.in)
		if !c.wantFull {
			assert.Equal(t, c.wantN, opts.TrimN, "%q n", c.in)
			assert.Equal(t, c.wantUnit, opts.TrimUnit, "%q unit", c.in)
			assert.Equal(t, c.wantTail, opts.TrimTail, "%q tail", c.in)
		}
	}
}

func TestParseFileTreeFlags(t *testing.T) {
	// Positional args survive; flags are consumed.
	pos, opts, assetsSet, err := parseFileTreeFlags([]string{"file.html", "--trim", "3P", "--depth", "2"})
	require.NoError(t, err)
	assert.Equal(t, []string{"file.html"}, pos)
	assert.Equal(t, 3, opts.TrimN)
	assert.Equal(t, "P", opts.TrimUnit)
	assert.Equal(t, 2, opts.MaxLevel)
	assert.False(t, assetsSet)

	// --bare zeroes trim and turns off assets explicitly.
	_, opts, assetsSet, err = parseFileTreeFlags([]string{"f", "--bare"})
	require.NoError(t, err)
	assert.Equal(t, 0, opts.TrimN)
	assert.False(t, opts.Assets)
	assert.True(t, assetsSet)

	// --only / --drop token sets.
	_, opts, _, err = parseFileTreeFlags([]string{"f", "--only", "h1,h2", "--drop", "svg,media"})
	require.NoError(t, err)
	assert.True(t, opts.Only["h1"] && opts.Only["h2"])
	assert.True(t, opts.Drop["svg"] && opts.Drop["media"])

	// --trim=4L inline form.
	_, opts, _, err = parseFileTreeFlags([]string{"f", "--trim=4L"})
	require.NoError(t, err)
	assert.Equal(t, 4, opts.TrimN)
	assert.Equal(t, "L", opts.TrimUnit)

	// Unknown flag errors instead of being swallowed as a positional.
	_, _, _, err = parseFileTreeFlags([]string{"f", "--trm", "4L"})
	assert.Error(t, err)

	// -h/--version pass through as positional (handled by the subcommand switch).
	pos, _, _, err = parseFileTreeFlags([]string{"--help"})
	require.NoError(t, err)
	assert.Equal(t, []string{"--help"}, pos)

	// Missing value errors.
	_, _, _, err = parseFileTreeFlags([]string{"f", "--trim"})
	assert.Error(t, err)
}
