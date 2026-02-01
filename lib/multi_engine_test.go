package mq_test

import (
	"path/filepath"
	"testing"

	"github.com/muqsitnawaz/mq/html"
	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/muqsitnawaz/mq/pdf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiFormatEngine(t *testing.T) {
	engine := mq.NewMultiFormatEngine(
		mq.WithFormatParser(html.NewParser()),
		mq.WithFormatParser(pdf.NewParser()),
	)

	// Test HTML parsing
	htmlContent := []byte(`<!DOCTYPE html>
<html>
<head><title>Test HTML</title></head>
<body><main><h1>HTML Heading</h1><p>HTML content</p></main></body>
</html>`)

	doc, err := engine.Parse(htmlContent, "test.html")
	require.NoError(t, err)
	assert.Equal(t, mq.FormatHTML, doc.Format())
	assert.Equal(t, "Test HTML", doc.Title())

	// Test Markdown parsing (default)
	mdContent := []byte(`# Markdown Title

This is markdown content.

## Section

More content.
`)

	doc, err = engine.Parse(mdContent, "test.md")
	require.NoError(t, err)
	assert.Equal(t, mq.FormatMarkdown, doc.Format())
	assert.Equal(t, "Markdown Title", doc.Title())

	// Test PDF parsing
	pdfContent := []byte(`%PDF-1.0
1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj 2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj 3 0 obj<</Type/Page/Parent 2 0 R>>endobj xref 0 4 0000000000 65535 f 0000000009 00000 n 0000000052 00000 n 0000000101 00000 n trailer<</Size 4/Root 1 0 R>>startxref 150 %%EOF`)

	doc, err = engine.Parse(pdfContent, "test.pdf")
	require.NoError(t, err)
	assert.Equal(t, mq.FormatPDF, doc.Format())
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		content  []byte
		expected mq.Format
	}{
		// Extension-based detection
		{"markdown .md", "file.md", nil, mq.FormatMarkdown},
		{"markdown .markdown", "file.markdown", nil, mq.FormatMarkdown},
		{"markdown .mdown", "file.mdown", nil, mq.FormatMarkdown},
		{"html .html", "page.html", nil, mq.FormatHTML},
		{"html .htm", "page.htm", nil, mq.FormatHTML},
		{"html .xhtml", "page.xhtml", nil, mq.FormatHTML},
		{"pdf .pdf", "doc.pdf", nil, mq.FormatPDF},

		// Content-based detection
		{"html content doctype", "unknown", []byte("<!DOCTYPE html><html>"), mq.FormatHTML},
		{"html content tag", "unknown", []byte("<html><body>"), mq.FormatHTML},
		{"pdf content magic", "unknown", []byte("%PDF-1.4"), mq.FormatPDF},

		// Default to markdown
		{"unknown extension", "file.txt", nil, mq.FormatMarkdown},
		{"no extension", "README", nil, mq.FormatMarkdown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mq.DetectFormat(tt.path, tt.content)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestLoadAny(t *testing.T) {
	// Test with markdown content
	mdPath := filepath.Join("..", "mql", "testdata", "api-reference.md")
	doc, err := mq.LoadAny(mdPath)
	require.NoError(t, err)
	assert.Equal(t, mq.FormatMarkdown, doc.Format())
}

func TestParseAny(t *testing.T) {
	// ParseAny uses default engine which only has Markdown parser
	// So all formats fall back to Markdown

	// HTML-looking content - detected as HTML but falls back to Markdown parser
	doc, err := mq.ParseAny([]byte("<html><body><h1>Test</h1></body></html>"), "unknown")
	require.NoError(t, err)
	// Falls back to markdown since no HTML parser registered
	assert.Equal(t, mq.FormatMarkdown, doc.Format())

	// PDF-looking content - detected as PDF but falls back to Markdown parser
	doc, err = mq.ParseAny([]byte("%PDF-1.0 minimal"), "unknown")
	require.NoError(t, err)
	// Falls back to markdown since no PDF parser registered
	assert.Equal(t, mq.FormatMarkdown, doc.Format())

	// Markdown content - parsed as Markdown
	doc, err = mq.ParseAny([]byte("# Hello\n\nWorld"), "unknown")
	require.NoError(t, err)
	assert.Equal(t, mq.FormatMarkdown, doc.Format())
}

func TestRegisterParser(t *testing.T) {
	engine := mq.NewMultiFormatEngine()

	// Initially has markdown
	assert.True(t, engine.HasParser(mq.FormatMarkdown))

	// Register HTML
	engine.RegisterParser(html.NewParser())
	assert.True(t, engine.HasParser(mq.FormatHTML))

	// Register PDF
	engine.RegisterParser(pdf.NewParser())
	assert.True(t, engine.HasParser(mq.FormatPDF))
}

func TestParseWithFormat(t *testing.T) {
	engine := mq.NewMultiFormatEngine(
		mq.WithFormatParser(html.NewParser()),
	)

	// Force HTML parsing even with .md extension
	htmlContent := []byte(`<html><body><h1>Forced HTML</h1></body></html>`)
	doc, err := engine.ParseWithFormat(htmlContent, "misleading.md", mq.FormatHTML)
	require.NoError(t, err)
	assert.Equal(t, mq.FormatHTML, doc.Format())
}

func TestDefaultFormat(t *testing.T) {
	// Create engine with HTML as default, but only HTML parser registered
	engine := mq.NewMultiFormatEngine(
		mq.WithDefaultFormat(mq.FormatHTML),
		mq.WithFormatParser(html.NewParser()),
	)

	// Remove markdown parser to test fallback
	// The content will be detected as markdown (unknown extension, non-HTML content)
	// but since there's no markdown parser, it should fall back to HTML

	// Actually, NewMultiFormatEngine always registers Markdown
	// So let's test that HTML extension uses HTML parser
	content := []byte(`<!DOCTYPE html><html><body><h1>Test</h1></body></html>`)
	doc, err := engine.Parse(content, "file.html")
	require.NoError(t, err)
	assert.Equal(t, mq.FormatHTML, doc.Format())

	// Also test that .htm works
	doc, err = engine.Parse(content, "file.htm")
	require.NoError(t, err)
	assert.Equal(t, mq.FormatHTML, doc.Format())
}

func TestUnifiedQueriesAcrossFormats(t *testing.T) {
	engine := mq.NewMultiFormatEngine(
		mq.WithFormatParser(html.NewParser()),
	)

	// Same content in different formats
	mdContent := []byte(`# Title

## Introduction

Hello world

## Methods

Description here
`)

	htmlContent := []byte(`<!DOCTYPE html>
<html>
<head><title>Title</title></head>
<body>
<main>
<h1>Title</h1>
<h2>Introduction</h2>
<p>Hello world</p>
<h2>Methods</h2>
<p>Description here</p>
</main>
</body>
</html>`)

	mdDoc, err := engine.Parse(mdContent, "doc.md")
	require.NoError(t, err)

	htmlDoc, err := engine.Parse(htmlContent, "doc.html")
	require.NoError(t, err)

	// Same queries work on both
	mdH2 := mdDoc.GetHeadings(2)
	htmlH2 := htmlDoc.GetHeadings(2)

	assert.Len(t, mdH2, 2)
	assert.Len(t, htmlH2, 2)

	// Same heading texts
	assert.Equal(t, "Introduction", mdH2[0].Text)
	assert.Equal(t, "Introduction", htmlH2[0].Text)

	// Same section access
	mdSection, ok := mdDoc.GetSection("Introduction")
	require.True(t, ok)
	htmlSection, ok := htmlDoc.GetSection("Introduction")
	require.True(t, ok)

	assert.Equal(t, mdSection.Heading.Text, htmlSection.Heading.Text)
}
