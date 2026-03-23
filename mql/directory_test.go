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

func TestBuildDirTreeWithLimit(t *testing.T) {
	// Use mql/testdata which has 8 markdown files
	testdataDir := "testdata"

	opts := mq.TreeOptions{Mode: mq.TreeModeDefault, Limit: 3}
	tree, err := mql.BuildDirTreeWithOptions(testdataDir, opts)
	require.NoError(t, err)

	// Should only show 3 files
	assert.Equal(t, 3, len(tree.Root))

	// Should track truncation
	assert.Equal(t, 5, tree.RootTruncated) // 8 total - 3 shown = 5 truncated

	// Rendered output should show truncation hint
	rendered := tree.String()
	assert.Contains(t, rendered, "... (5 more)")
}

func TestBuildDirTreeWithDepth(t *testing.T) {
	dir := t.TempDir()

	// Create nested structure: dir/sub1/sub2/file.md
	sub1 := filepath.Join(dir, "sub1")
	sub2 := filepath.Join(sub1, "sub2")
	require.NoError(t, os.MkdirAll(sub2, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "root.md"), []byte("# Root"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sub1, "level1.md"), []byte("# Level 1"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sub2, "level2.md"), []byte("# Level 2"), 0o644))

	// Depth 1: only root files, subdirs shown as truncated
	opts := mq.TreeOptions{Mode: mq.TreeModeDefault, Depth: 1}
	tree, err := mql.BuildDirTreeWithOptions(dir, opts)
	require.NoError(t, err)

	// Should have root.md and sub1/ (truncated)
	assert.Equal(t, 2, len(tree.Root))

	// Find sub1 node
	var sub1Node *mq.DirFileNode
	for _, node := range tree.Root {
		if node.Name == "sub1" {
			sub1Node = node
			break
		}
	}
	require.NotNil(t, sub1Node)
	assert.True(t, sub1Node.IsDir)
	assert.Equal(t, -1, sub1Node.Truncated) // -1 indicates depth limit
	assert.Equal(t, 2, sub1Node.TotalFiles) // Contains level1.md and sub2/level2.md

	// Rendered output should show depth limit hint
	rendered := tree.String()
	assert.Contains(t, rendered, "depth limit")
}

func TestBuildDirTreeWithDepthAndLimit(t *testing.T) {
	dir := t.TempDir()

	// Create structure with multiple subdirs and files
	for i := 1; i <= 5; i++ {
		subdir := filepath.Join(dir, filepath.Base(dir)+"-sub"+string(rune('0'+i)))
		require.NoError(t, os.MkdirAll(subdir, 0o755))
		for j := 1; j <= 3; j++ {
			require.NoError(t, os.WriteFile(
				filepath.Join(subdir, "file"+string(rune('0'+j))+".md"),
				[]byte("# File"), 0o644))
		}
	}

	// Depth 2, Limit 2: show 2 subdirs, each with up to 2 files
	opts := mq.TreeOptions{Mode: mq.TreeModeDefault, Depth: 2, Limit: 2}
	tree, err := mql.BuildDirTreeWithOptions(dir, opts)
	require.NoError(t, err)

	// Should show 2 subdirs at root
	assert.Equal(t, 2, len(tree.Root))
	assert.Equal(t, 3, tree.RootTruncated) // 5 - 2 = 3 truncated

	// Each subdir should have 2 files with 1 truncated
	for _, node := range tree.Root {
		if node.IsDir {
			assert.Equal(t, 2, len(node.Children))
			assert.Equal(t, 1, node.Truncated) // 3 - 2 = 1 truncated
		}
	}

	rendered := tree.String()
	assert.Contains(t, rendered, "... (3 more)") // root level
	assert.Contains(t, rendered, "... (1 more)") // subdir level
}
