package mql_test

import (
	"os"
	"path/filepath"
	"testing"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/muqsitnawaz/mq/mql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchDirSupportsMultipleFormats(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "page.html"), []byte("<!DOCTYPE html><html><head><title>HTML Doc</title></head><body><main><h1>Heading</h1><p>Needle in html</p></main></body></html>"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.json"), []byte(`{"name":"json doc","content":"Needle in json"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.yaml"), []byte("name: yaml doc\ncontent: Needle in yaml\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "events.jsonl"), []byte("{\"event\":\"Needle in jsonl\"}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("Needle in text file"), 0o644))

	results, err := mql.SearchDir(dir, "needle")
	require.NoError(t, err)

	files := make(map[string]struct{})
	for _, match := range results.Matches {
		files[filepath.Base(match.File)] = struct{}{}
	}

	assert.Contains(t, files, "page.html")
	assert.Contains(t, files, "data.json")
	assert.Contains(t, files, "data.yaml")
	assert.Contains(t, files, "events.jsonl")
	assert.NotContains(t, files, "ignore.txt")
	assert.NotContains(t, files, "doc.md")
}

func TestBuildDirTreeSupportsNonMarkdownFormats(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "page.html"), []byte("<!DOCTYPE html><html><head><title>HTML Doc</title></head><body><main><h1>Heading</h1><p>content</p></main></body></html>"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.json"), []byte(`{"name":"json doc","content":"value"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.yaml"), []byte("name: yaml doc\ncontent: value\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "events.jsonl"), []byte("{\"event\":\"value\"}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("should be ignored"), 0o644))

	tree, err := mql.BuildDirTree(dir, mq.TreeModePreview)
	require.NoError(t, err)

	assert.Equal(t, 4, tree.TotalFiles)

	files := make(map[string]struct{})
	for _, node := range tree.Root {
		files[node.Name] = struct{}{}
	}

	assert.Contains(t, files, "page.html")
	assert.Contains(t, files, "data.json")
	assert.Contains(t, files, "data.yaml")
	assert.Contains(t, files, "events.jsonl")
	assert.NotContains(t, files, "ignore.txt")
	assert.NotContains(t, files, "doc.md")

	rendered := tree.String()
	assert.Contains(t, rendered, "data.json")
	assert.Contains(t, rendered, "2 keys")
	assert.Contains(t, rendered, "data.yaml")
	assert.Contains(t, rendered, "events.jsonl")
	assert.Contains(t, rendered, "1 record")
	assert.Contains(t, rendered, "page.html")
	assert.Contains(t, rendered, "1 section")
	assert.Contains(t, rendered, "key content")
	assert.Contains(t, rendered, "H1 Heading")
	assert.NotContains(t, rendered, "# content")
}
