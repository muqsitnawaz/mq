package mq_test

import (
	"os"
	"path/filepath"
	"testing"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createSearchCorpus writes a realistic set of markdown files to dir.
// Returns the directory path. Files are designed so that only a subset
// match any given query, exercising the filter-then-parse optimization.
func createSearchCorpus(t *testing.T, dir string) {
	t.Helper()

	files := map[string]string{
		"api-auth.md": `# Authentication

All API requests require a bearer token.

## OAuth2 Flow

The OAuth2 authorization code flow works as follows:

1. Redirect the user to the authorization endpoint
2. User grants permission
3. Exchange the authorization code for an access token

### Token Refresh

Access tokens expire after 3600 seconds. Use the refresh token
to obtain a new access token without user interaction.

` + "```bash\ncurl -X POST https://api.example.com/oauth/token \\\n  -d grant_type=refresh_token \\\n  -d refresh_token=$REFRESH_TOKEN\n```\n",

		"api-endpoints.md": `# API Endpoints

## Users

### GET /users

Returns a paginated list of users.

### POST /users

Creates a new user account.

## Projects

### GET /projects

Returns all projects for the authenticated user.

### DELETE /projects/:id

Permanently deletes a project and all associated data.
`,

		"installation.md": `# Installation Guide

## Prerequisites

- Go 1.21 or later
- A working internet connection

## Quick Start

Download the latest release and run the installer:

` + "```bash\ncurl -sSL https://example.com/install.sh | bash\n```\n" + `
## Configuration

Create a config file at ~/.config/myapp/config.yaml:

` + "```yaml\nserver:\n  port: 8080\n  host: 0.0.0.0\ndatabase:\n  url: postgres://localhost/mydb\n```\n",

		"changelog.md": `# Changelog

## v2.1.0 (2025-03-15)

- Added bearer token rotation support
- Fixed memory leak in connection pool
- Improved error messages for authentication failures

## v2.0.0 (2025-01-01)

- Breaking: removed deprecated /v1 endpoints
- New OAuth2 authorization code flow
- Performance improvements across the board

## v1.5.0 (2024-09-01)

- Added rate limiting
- Bug fixes
`,

		"contributing.md": `# Contributing

## Code Style

We use gofmt for all Go code. Run it before submitting a PR.

## Testing

All changes must include tests. Run the test suite with:

` + "```bash\ngo test ./...\n```\n" + `
## Pull Requests

- Keep PRs focused on a single change
- Include a description of what changed and why
- Link to any relevant issues
`,

		"architecture.md": `# Architecture

## Overview

The system uses a layered architecture with clear separation of concerns.

## Components

### Parser Layer

The parser converts raw input into a structured AST. Each format
(Markdown, HTML, PDF) has its own parser that produces a unified
Document type.

### Query Engine

The query engine operates on the unified Document type. It supports
chaining operations via pipes, similar to jq.

### Cache Layer

Documents are cached after parsing. The cache uses content-addressed
storage with SHA-256 hashes for invalidation.
`,

		"troubleshooting.md": `# Troubleshooting

## Common Errors

### Error: invalid bearer token

This error occurs when your authentication token has expired.
Generate a new token using the OAuth2 flow described in the
Authentication guide.

### Error: rate limit exceeded

You have exceeded the API rate limit. Wait 60 seconds before
retrying. Consider implementing exponential backoff.

### Error: connection refused

The server may be down or your firewall may be blocking the
connection. Check your network settings.
`,
	}

	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}
}

func TestSearchDirFilterThenParse(t *testing.T) {
	dir := t.TempDir()
	createSearchCorpus(t, dir)

	t.Run("finds matches across multiple files", func(t *testing.T) {
		results, err := mq.SearchDir(dir, "bearer token")
		require.NoError(t, err)

		// "bearer token" appears in api-auth.md, changelog.md, and troubleshooting.md
		files := matchedFiles(results)
		assert.Contains(t, files, "api-auth.md")
		assert.Contains(t, files, "troubleshooting.md")
		assert.Contains(t, files, "changelog.md")

		// Should NOT match files that don't contain "bearer token"
		assert.NotContains(t, files, "installation.md")
		assert.NotContains(t, files, "contributing.md")
		assert.NotContains(t, files, "api-endpoints.md")
		assert.NotContains(t, files, "architecture.md")
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		results, err := mq.SearchDir(dir, "OAuth2")
		require.NoError(t, err)

		files := matchedFiles(results)
		assert.Contains(t, files, "api-auth.md")
		assert.Contains(t, files, "changelog.md")

		// Also works lowercase
		results2, err := mq.SearchDir(dir, "oauth2")
		require.NoError(t, err)
		assert.Equal(t, len(results.Matches), len(results2.Matches))
	})

	t.Run("returns section context for matches", func(t *testing.T) {
		results, err := mq.SearchDir(dir, "refresh token")
		require.NoError(t, err)

		// Should get section-level context, not just file-level
		found := false
		for _, m := range results.Matches {
			if filepath.Base(m.File) == "api-auth.md" {
				found = true
				assert.NotEmpty(t, m.Section, "should have section context")
				assert.NotEmpty(t, m.Match, "should have match snippet")
			}
		}
		assert.True(t, found, "should find match in api-auth.md")
	})

	t.Run("no results for absent query", func(t *testing.T) {
		results, err := mq.SearchDir(dir, "xyzzy_nonexistent_term_42")
		require.NoError(t, err)
		assert.Empty(t, results.Matches)
	})

	t.Run("query matching single file", func(t *testing.T) {
		results, err := mq.SearchDir(dir, "gofmt")
		require.NoError(t, err)

		files := matchedFiles(results)
		assert.Equal(t, 1, len(files), "gofmt only appears in contributing.md")
		assert.Contains(t, files, "contributing.md")
	})

	t.Run("skips non-traversal files", func(t *testing.T) {
		// Write a .txt file with the query — should be ignored
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "notes.txt"),
			[]byte("bearer token is here too"),
			0o644,
		))

		results, err := mq.SearchDir(dir, "bearer token")
		require.NoError(t, err)

		files := matchedFiles(results)
		assert.NotContains(t, files, "notes.txt")
	})

	t.Run("skips hidden files", func(t *testing.T) {
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, ".hidden.md"),
			[]byte("# Hidden\nbearer token secret"),
			0o644,
		))

		results, err := mq.SearchDir(dir, "bearer token")
		require.NoError(t, err)

		files := matchedFiles(results)
		assert.NotContains(t, files, ".hidden.md")
	})
}

func TestSearchDirMultipleMarkdownFiles(t *testing.T) {
	dir := t.TempDir()

	// Multiple markdown files — only some match
	require.NoError(t, os.WriteFile(filepath.Join(dir, "guide.md"),
		[]byte("# Guide\n\nThe deployment process is automated.\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "faq.md"),
		[]byte("# FAQ\n\n## How do I deploy?\n\nThe deployment is triggered by CI.\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unrelated.md"),
		[]byte("# Unrelated\n\nThis file has nothing relevant.\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "setup.md"),
		[]byte("# Setup\n\n## Local Development\n\nRun `make dev` to start the server.\n"), 0o644))

	results, err := mq.SearchDir(dir, "deployment")
	require.NoError(t, err)

	files := matchedFiles(results)
	assert.Contains(t, files, "guide.md")
	assert.Contains(t, files, "faq.md")
	assert.NotContains(t, files, "unrelated.md")
	assert.NotContains(t, files, "setup.md")
}

func TestSearchDirSubdirectories(t *testing.T) {
	dir := t.TempDir()

	// Create nested structure
	sub := filepath.Join(dir, "docs", "api")
	require.NoError(t, os.MkdirAll(sub, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "root.md"),
		[]byte("# Root\n\nAuthentication is required.\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docs", "overview.md"),
		[]byte("# Overview\n\nAuthentication uses OAuth2.\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "auth.md"),
		[]byte("# Auth API\n\nAuthentication endpoint details.\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "users.md"),
		[]byte("# Users API\n\nUser management endpoints.\n"), 0o644))

	results, err := mq.SearchDir(dir, "authentication")
	require.NoError(t, err)

	files := matchedFiles(results)
	assert.Contains(t, files, "root.md")
	assert.Contains(t, files, "overview.md")
	assert.Contains(t, files, "auth.md")
	assert.NotContains(t, files, "users.md")
	assert.Equal(t, 3, len(files))
}

func TestSearchDirEmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	results, err := mq.SearchDir(dir, "anything")
	require.NoError(t, err)
	assert.Empty(t, results.Matches)
}

func TestSearchDirLargeCorpus(t *testing.T) {
	// Create 50 markdown files, only 3 contain the target query.
	// This exercises the filter-then-parse optimization at scale —
	// without the optimization, all 50 files would be fully parsed.
	dir := t.TempDir()

	for i := 0; i < 50; i++ {
		content := "# Document " + string(rune('A'+i%26)) + "\n\nGeneric content about software engineering.\n"
		name := filepath.Join(dir, "doc-"+padInt(i)+".md")
		require.NoError(t, os.WriteFile(name, []byte(content), 0o644))
	}

	// Plant 3 files with the needle
	for _, idx := range []int{7, 23, 41} {
		name := filepath.Join(dir, "doc-"+padInt(idx)+".md")
		content := "# Special Document\n\nThis contains the cryptographic_nonce_value we are looking for.\n"
		require.NoError(t, os.WriteFile(name, []byte(content), 0o644))
	}

	results, err := mq.SearchDir(dir, "cryptographic_nonce_value")
	require.NoError(t, err)

	assert.Equal(t, 3, len(results.Matches))
	for _, m := range results.Matches {
		assert.Contains(t, m.Match, "cryptographic_nonce_value")
	}
}

// matchedFiles extracts unique base filenames from search results.
func matchedFiles(results *mq.SearchResults) map[string]struct{} {
	files := make(map[string]struct{})
	for _, m := range results.Matches {
		files[filepath.Base(m.File)] = struct{}{}
	}
	return files
}

// TestBuildTreeLineCountNonMarkdown verifies that BuildTree uses readableText
// line count for non-markdown formats instead of counting newlines in raw binary source.
func TestBuildTreeLineCountNonMarkdown(t *testing.T) {
	readableText := "# Chapter 1\nIntro paragraph.\n\n## Section A\nContent A.\n\n## Section B\nContent B."
	textBytes := []byte(readableText)
	expectedLines := 8 // 8 lines in readableText (7 newlines + 1)

	// Simulate binary PDF source with many more newlines than actual content
	binarySource := make([]byte, 5000)
	for i := range binarySource {
		if i%10 == 0 {
			binarySource[i] = '\n'
		} else {
			binarySource[i] = 0xAB
		}
	}

	headings := []*mq.Heading{
		{Level: 1, Text: "Chapter 1", Line: 1},
		{Level: 2, Text: "Section A", Line: 4},
		{Level: 2, Text: "Section B", Line: 7},
	}
	sections := []*mq.Section{
		mq.NewSectionWithSource(headings[0], 1, 8, textBytes),
		mq.NewSectionWithSource(headings[1], 4, 6, textBytes),
		mq.NewSectionWithSource(headings[2], 7, 8, textBytes),
	}

	// FormatPDF document with binary source but readable text
	doc := mq.NewDocument(
		binarySource, "test.pdf", mq.FormatPDF, "Chapter 1",
		headings, sections, nil, nil, nil, nil, nil,
		readableText,
	)

	tree := doc.BuildTree()
	assert.Equal(t, expectedLines, tree.Lines,
		"PDF tree should count lines in readableText, not raw binary source")

	// Verify markdown still counts from source
	mdSections := []*mq.Section{
		mq.NewSectionWithSource(headings[0], 1, 8, textBytes),
		mq.NewSectionWithSource(headings[1], 4, 6, textBytes),
		mq.NewSectionWithSource(headings[2], 7, 8, textBytes),
	}
	mdDoc := mq.NewDocument(
		[]byte(readableText), "test.md", mq.FormatMarkdown, "Chapter 1",
		headings, mdSections, nil, nil, nil, nil, nil,
		readableText,
	)
	mdTree := mdDoc.BuildTree()
	assert.Equal(t, expectedLines, mdTree.Lines,
		"Markdown tree should count lines in source")
}

// TestBuildTreePreviewNonMarkdown verifies that section previews show readable
// text, not binary content, for non-markdown documents.
func TestBuildTreePreviewNonMarkdown(t *testing.T) {
	readableText := "# Overview\nThis document explains the authentication flow.\n\n## Details\nMore info here."
	textBytes := []byte(readableText)

	// Binary junk that would produce garbage previews if used as section source
	binarySource := []byte("%PDF-1.4\x00\x01\x02\x03\n\xAB\xCD\xEF\n" + readableText)

	headings := []*mq.Heading{
		{Level: 1, Text: "Overview", Line: 1, Page: 1},
		{Level: 2, Text: "Details", Line: 4, Page: 2},
	}
	sections := []*mq.Section{
		mq.NewSectionWithSource(headings[0], 1, 5, textBytes),
		mq.NewSectionWithSource(headings[1], 4, 5, textBytes),
	}

	doc := mq.NewDocument(
		binarySource, "test.pdf", mq.FormatPDF, "Overview",
		headings, sections, nil, nil, nil, nil, nil,
		readableText,
	)

	tree := doc.BuildTree()
	require.NotEmpty(t, tree.Root)

	// The preview should contain readable content from the section
	preview := tree.Root[0].Preview
	assert.Contains(t, preview, "authentication flow",
		"PDF tree preview should show readable text, not binary garbage")

	// Verify page numbers are rendered instead of line numbers
	output := tree.String()
	assert.Contains(t, output, "(p. 1)", "PDF tree should show page numbers")
	assert.Contains(t, output, "(p. 2)", "PDF tree should show page numbers for child sections")
	assert.Contains(t, output, "(2 pages)", "PDF tree header should show page count")
	assert.NotContains(t, output, "lines", "PDF tree should not mention lines")
}

// padInt zero-pads an int to 2 digits.
func padInt(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
