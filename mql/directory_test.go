package mql_test

import (
	"fmt"
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

// --- Exhaustive multi-format search tests ---

// searchFiles extracts unique base filenames from search results.
func searchFiles(results *mq.SearchResults) map[string]struct{} {
	files := make(map[string]struct{})
	for _, m := range results.Matches {
		files[filepath.Base(m.File)] = struct{}{}
	}
	return files
}

func TestSearchMarkdownSections(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "guide.md"), []byte(`# User Guide

## Installation

Install the authentication module with pip:

`+"`"+`bash
pip install auth-module
`+"`"+`

## Configuration

Set the database connection string in your config file.

## Troubleshooting

### Error: authentication failed

Check that your credentials are correct and the token has not expired.

### Error: connection timeout

Increase the timeout value in your configuration.
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "api.md"), []byte(`# API Reference

## Endpoints

### POST /login

Authenticates a user and returns a session token.

### GET /users

Returns a list of users. Requires admin authentication.

## Rate Limiting

Each API key is limited to 100 requests per minute.
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "unrelated.md"), []byte(`# Performance Tuning

## Caching Strategy

Use Redis for caching hot paths. Invalidate on write.

## Database Indexing

Add composite indexes for frequently joined columns.
`), 0o644))

	t.Run("finds matches in correct sections", func(t *testing.T) {
		results, err := mql.SearchDir(dir, "authentication")
		require.NoError(t, err)

		files := searchFiles(results)
		assert.Contains(t, files, "guide.md")
		assert.Contains(t, files, "api.md")
		assert.NotContains(t, files, "unrelated.md")

		// Verify section-level context
		for _, m := range results.Matches {
			assert.NotEmpty(t, m.Section, "match should have section context")
			assert.NotEmpty(t, m.Match, "match should have snippet")
			assert.NotEmpty(t, m.Lines, "match should have line range")
		}
	})

	t.Run("case insensitive across sections", func(t *testing.T) {
		upper, err := mql.SearchDir(dir, "AUTHENTICATION")
		require.NoError(t, err)
		lower, err := mql.SearchDir(dir, "authentication")
		require.NoError(t, err)
		assert.Equal(t, len(upper.Matches), len(lower.Matches))
	})

	t.Run("partial word match", func(t *testing.T) {
		results, err := mql.SearchDir(dir, "timeout")
		require.NoError(t, err)
		files := searchFiles(results)
		assert.Contains(t, files, "guide.md")
		assert.NotContains(t, files, "api.md")
	})
}

func TestSearchHTML(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "docs.html"), []byte(`<!DOCTYPE html>
<html>
<head><title>Product Documentation</title></head>
<body>
<nav><ul><li>Home</li><li>Docs</li><li>Support</li></ul></nav>
<main>
<h1>Product Documentation</h1>
<p>Welcome to the deployment guide for our platform.</p>
<h2>Quick Start</h2>
<p>Run the installer to begin the deployment process.</p>
<h2>Advanced Configuration</h2>
<p>For production environments, configure the load balancer and SSL certificates.</p>
</main>
<footer><p>Copyright 2025</p></footer>
</body>
</html>`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "blog.html"), []byte(`<!DOCTYPE html>
<html>
<head><title>Blog Post</title></head>
<body>
<main>
<article>
<h1>How We Scaled Our Deployment Pipeline</h1>
<p>In this post we describe how our deployment automation reduced release times by 80%.</p>
<p>The key insight was parallelizing the build and test stages.</p>
</article>
</main>
</body>
</html>`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "pricing.html"), []byte(`<!DOCTYPE html>
<html>
<head><title>Pricing</title></head>
<body>
<main>
<h1>Pricing Plans</h1>
<p>Choose the plan that fits your team. All plans include unlimited users.</p>
</main>
</body>
</html>`), 0o644))

	results, err := mql.SearchDir(dir, "deployment")
	require.NoError(t, err)

	files := searchFiles(results)
	assert.Contains(t, files, "docs.html")
	assert.Contains(t, files, "blog.html")
	assert.NotContains(t, files, "pricing.html")

	// Verify we get match context
	for _, m := range results.Matches {
		assert.NotEmpty(t, m.Match, "HTML match should have snippet")
	}
}

func TestSearchJSON(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{
  "database": {
    "host": "db.production.internal",
    "port": 5432,
    "name": "appdb",
    "pool_size": 20
  },
  "cache": {
    "provider": "redis",
    "ttl": 3600
  }
}`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "users.json"), []byte(`{
  "admin": {
    "name": "Alice",
    "role": "administrator",
    "database_access": true
  },
  "readonly": {
    "name": "Bob",
    "role": "viewer"
  }
}`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "empty.json"), []byte(`{
  "version": "1.0",
  "features": ["logging", "metrics"]
}`), 0o644))

	results, err := mql.SearchDir(dir, "database")
	require.NoError(t, err)

	files := searchFiles(results)
	assert.Contains(t, files, "config.json")
	assert.Contains(t, files, "users.json")
	assert.NotContains(t, files, "empty.json")
}

func TestSearchYAML(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "deploy.yaml"), []byte(`
service:
  name: web-frontend
  replicas: 3
  deployment_strategy: rolling
  health_check:
    path: /healthz
    interval: 30s
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "monitoring.yml"), []byte(`
alerts:
  - name: high_latency
    threshold: 500ms
    channel: pagerduty
  - name: error_rate
    threshold: 5%
    channel: slack
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "ci.yaml"), []byte(`
pipeline:
  stages:
    - lint
    - test
    - build
  timeout: 30m
`), 0o644))

	results, err := mql.SearchDir(dir, "deployment")
	require.NoError(t, err)

	files := searchFiles(results)
	assert.Contains(t, files, "deploy.yaml")
	assert.NotContains(t, files, "monitoring.yml")
	assert.NotContains(t, files, "ci.yaml")

	// Also test .yml extension
	results2, err := mql.SearchDir(dir, "pagerduty")
	require.NoError(t, err)
	files2 := searchFiles(results2)
	assert.Contains(t, files2, "monitoring.yml")
}

func TestSearchJSONL(t *testing.T) {
	dir := t.TempDir()

	// Multi-record JSONL — only some records contain the query
	require.NoError(t, os.WriteFile(filepath.Join(dir, "events.jsonl"), []byte(
		`{"event":"user_signup","user":"alice","timestamp":"2025-01-01"}
{"event":"deployment_started","service":"api","timestamp":"2025-01-02"}
{"event":"user_login","user":"bob","timestamp":"2025-01-03"}
{"event":"deployment_completed","service":"api","status":"success","timestamp":"2025-01-04"}
{"event":"error","message":"connection refused","timestamp":"2025-01-05"}
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "access.ndjson"), []byte(
		`{"method":"GET","path":"/api/users","status":200}
{"method":"POST","path":"/api/deploy","status":201}
{"method":"GET","path":"/api/health","status":200}
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "metrics.jsonl"), []byte(
		`{"metric":"cpu_usage","value":45.2}
{"metric":"memory_usage","value":72.1}
{"metric":"disk_io","value":15.3}
`), 0o644))

	t.Run("matches records across files", func(t *testing.T) {
		results, err := mql.SearchDir(dir, "deployment")
		require.NoError(t, err)

		files := searchFiles(results)
		assert.Contains(t, files, "events.jsonl")
		assert.NotContains(t, files, "metrics.jsonl")
	})

	t.Run("ndjson extension works", func(t *testing.T) {
		results, err := mql.SearchDir(dir, "deploy")
		require.NoError(t, err)

		files := searchFiles(results)
		assert.Contains(t, files, "access.ndjson")
	})

	t.Run("per-record granularity", func(t *testing.T) {
		results, err := mql.SearchDir(dir, "deployment")
		require.NoError(t, err)

		// Should match exactly 2 records in events.jsonl (deployment_started, deployment_completed)
		eventMatches := 0
		for _, m := range results.Matches {
			if filepath.Base(m.File) == "events.jsonl" {
				eventMatches++
			}
		}
		assert.Equal(t, 2, eventMatches, "should match 2 deployment records")
	})
}

func TestSearchPDF(t *testing.T) {
	// Use a real PDF from testdata
	testPDF := filepath.Join("..", "pdf", "testdata", "papers", "ml", "attention.pdf")
	if _, err := os.Stat(testPDF); os.IsNotExist(err) {
		t.Skip("attention.pdf not found in testdata")
	}

	dir := t.TempDir()

	// Copy the real PDF into our test directory
	pdfBytes, err := os.ReadFile(testPDF)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "attention.pdf"), pdfBytes, 0o644))

	// Also add a markdown file to verify mixed-format search
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.md"), []byte(`# Paper Notes

## Attention Is All You Need

The transformer architecture uses self-attention mechanisms.
`), 0o644))

	// Add a file that should NOT match
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unrelated.md"), []byte(`# Grocery List

- Milk
- Eggs
- Bread
`), 0o644))

	t.Run("finds text in PDF", func(t *testing.T) {
		results, err := mql.SearchDir(dir, "attention")
		require.NoError(t, err)

		files := searchFiles(results)
		assert.Contains(t, files, "attention.pdf")
		assert.Contains(t, files, "notes.md")
		assert.NotContains(t, files, "unrelated.md")
	})

	t.Run("does not match absent term", func(t *testing.T) {
		results, err := mql.SearchDir(dir, "xyzzy_nonexistent_42")
		require.NoError(t, err)
		assert.Empty(t, results.Matches)
	})
}

func TestSearchMixedFormatDirectory(t *testing.T) {
	dir := t.TempDir()

	// Create a realistic project directory with mixed formats
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte(`# MyProject

A tool for managing deployments across multiple environments.

## Features

- Blue-green deployments
- Canary releases
- Rollback support
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "api.html"), []byte(`<!DOCTYPE html>
<html><head><title>API Docs</title></head>
<body><main>
<h1>API Documentation</h1>
<h2>Deployment Endpoints</h2>
<p>Use POST /api/deployments to trigger a new deployment.</p>
<h2>Monitoring</h2>
<p>Check GET /api/health for service status.</p>
</main></body></html>`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "schema.json"), []byte(`{
  "deployment": {
    "type": "object",
    "properties": {
      "service": {"type": "string"},
      "environment": {"type": "string"},
      "version": {"type": "string"}
    }
  }
}`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "config.yaml"), []byte(`
environments:
  staging:
    deployment_timeout: 300
    replicas: 2
  production:
    deployment_timeout: 600
    replicas: 5
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "audit.jsonl"), []byte(
		`{"action":"deployment","user":"alice","env":"staging","ts":"2025-03-01"}
{"action":"rollback","user":"bob","env":"production","ts":"2025-03-02"}
{"action":"deployment","user":"alice","env":"production","ts":"2025-03-03"}
`), 0o644))

	// Unrelated file
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "style.md"), []byte(`# Code Style Guide

Use 4-space indentation. Prefer explicit over implicit.
`), 0o644))

	results, err := mql.SearchDir(dir, "deployment")
	require.NoError(t, err)

	files := searchFiles(results)

	// Every format with "deployment" should match
	assert.Contains(t, files, "README.md", "markdown should match")
	assert.Contains(t, files, "api.html", "HTML should match")
	assert.Contains(t, files, "schema.json", "JSON should match")
	assert.Contains(t, files, "config.yaml", "YAML should match")
	assert.Contains(t, files, "audit.jsonl", "JSONL should match")

	// Unrelated file should not match
	assert.NotContains(t, files, "style.md", "unrelated file should not match")

	// Total matches should be reasonable (more than 5 since some formats have multiple sections/records)
	assert.GreaterOrEqual(t, len(results.Matches), 5, "should have at least one match per file")
}

func TestSearchFilterThenParseSkipsNonMatching(t *testing.T) {
	// Create a large directory where only a few files match.
	// This verifies the filter-then-parse optimization doesn't
	// change correctness at scale across all formats.
	dir := t.TempDir()

	// 20 markdown files with no match
	for i := 0; i < 20; i++ {
		content := fmt.Sprintf("# Document %d\n\nGeneric content about software patterns.\n", i)
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, fmt.Sprintf("doc-%02d.md", i)),
			[]byte(content), 0o644))
	}

	// 10 JSON files with no match
	for i := 0; i < 10; i++ {
		content := fmt.Sprintf(`{"id":%d,"name":"item-%d","status":"active"}`, i, i)
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, fmt.Sprintf("data-%02d.json", i)),
			[]byte(content), 0o644))
	}

	// 5 HTML files with no match
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf(`<!DOCTYPE html><html><head><title>Page %d</title></head><body><main><p>Static content here.</p></main></body></html>`, i)
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, fmt.Sprintf("page-%02d.html", i)),
			[]byte(content), 0o644))
	}

	// Plant exactly 4 needles across different formats
	require.NoError(t, os.WriteFile(filepath.Join(dir, "doc-07.md"),
		[]byte("# Special\n\nContains the quantum_entanglement_key phrase.\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data-03.json"),
		[]byte(`{"id":3,"name":"quantum_entanglement_key","status":"found"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "page-02.html"),
		[]byte(`<!DOCTYPE html><html><head><title>Found</title></head><body><main><p>The quantum_entanglement_key is here.</p></main></body></html>`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "events.jsonl"),
		[]byte("{\"event\":\"quantum_entanglement_key\",\"ts\":1}\n{\"event\":\"normal\",\"ts\":2}\n"), 0o644))

	results, err := mql.SearchDir(dir, "quantum_entanglement_key")
	require.NoError(t, err)

	files := searchFiles(results)
	assert.Equal(t, 4, len(files), "should match exactly 4 files")
	assert.Contains(t, files, "doc-07.md")
	assert.Contains(t, files, "data-03.json")
	assert.Contains(t, files, "page-02.html")
	assert.Contains(t, files, "events.jsonl")
}

func TestSearchDirCacheRoundtrip(t *testing.T) {
	// Regression: directory search through cached markdown documents must
	// produce the same results as the first (cold cache) run.
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "guide.md"), []byte(
		"# Deployment Guide\n\n## Rolling Deployment\n\nA rolling deployment replaces pods.\n\n## Monitoring\n\nCheck metrics after deployment.\n",
	), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unrelated.md"), []byte(
		"# Unrelated\n\nNo match here.\n",
	), 0o644))

	// Run 1: cold cache
	results1, err := mql.SearchDir(dir, "deployment")
	require.NoError(t, err)
	files1 := searchFiles(results1)
	require.Contains(t, files1, "guide.md", "cold cache should find guide.md")
	require.NotContains(t, files1, "unrelated.md")
	count1 := len(results1.Matches)
	require.Greater(t, count1, 0)

	// Run 2: warm cache — must produce identical results
	results2, err := mql.SearchDir(dir, "deployment")
	require.NoError(t, err)
	files2 := searchFiles(results2)
	assert.Contains(t, files2, "guide.md", "warm cache should still find guide.md")
	assert.NotContains(t, files2, "unrelated.md")
	assert.Equal(t, count1, len(results2.Matches), "warm cache should return same match count")
}

func TestSearchExcludesUnsupportedFormats(t *testing.T) {
	dir := t.TempDir()

	// Supported formats
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.md"),
		[]byte("# Readme\n\nSearch target here.\n"), 0o644))

	// Unsupported formats that happen to contain the text
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"),
		[]byte("Search target here."), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "code.go"),
		[]byte("// Search target here.\npackage main\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.csv"),
		[]byte("col1,col2\nSearch target here.,value\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "script.py"),
		[]byte("# Search target here.\nprint('hello')\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.toml"),
		[]byte("[section]\nkey = \"Search target here.\"\n"), 0o644))

	results, err := mql.SearchDir(dir, "search target")
	require.NoError(t, err)

	files := searchFiles(results)
	assert.Equal(t, 1, len(files), "only .md should match")
	assert.Contains(t, files, "readme.md")
}
