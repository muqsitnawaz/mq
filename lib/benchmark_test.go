package mq

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// generateMarkdown creates a markdown document of approximately the given size
func generateMarkdown(sizeBytes int) []byte {
	var buf bytes.Buffer

	// Each section is roughly 1KB
	sectionTemplate := `
## Section %d

This is the content for section %d. It contains some text that is meant to
simulate real documentation content. Here's some more text to make it realistic.

### Subsection %d.1

More detailed content here with examples and explanations that would be
typical in a technical document.

` + "```python\n" + `def example_%d():
    """Example function for section %d."""
    return "Hello from section %d"
` + "```\n" + `

### Subsection %d.2

Additional content with links like [example](https://example.com) and
some **bold** and *italic* text for good measure.

| Column A | Column B | Column C |
|----------|----------|----------|
| Value 1  | Value 2  | Value 3  |
| Value 4  | Value 5  | Value 6  |

`

	buf.WriteString("# Benchmark Document\n\n")
	buf.WriteString("This is a generated document for benchmarking purposes.\n\n")

	sectionNum := 1
	for buf.Len() < sizeBytes {
		section := fmt.Sprintf(sectionTemplate,
			sectionNum, sectionNum, sectionNum, sectionNum, sectionNum, sectionNum, sectionNum)
		buf.WriteString(section)
		sectionNum++
	}

	return buf.Bytes()
}

// generateJSON creates a JSON document of approximately the given size
func generateJSON(sizeBytes int) []byte {
	var buf bytes.Buffer
	buf.WriteString("{\n")

	keyNum := 0
	for buf.Len() < sizeBytes-100 { // Leave room for closing
		if keyNum > 0 {
			buf.WriteString(",\n")
		}
		buf.WriteString(fmt.Sprintf(`  "key_%d": {
    "name": "Item %d",
    "description": "This is a description for item %d with some content",
    "values": [1, 2, 3, 4, 5],
    "nested": {
      "field1": "value1",
      "field2": "value2"
    }
  }`, keyNum, keyNum, keyNum))
		keyNum++
	}

	buf.WriteString("\n}")
	return buf.Bytes()
}

// generateJSONL creates a JSONL document of approximately the given size
func generateJSONL(sizeBytes int) []byte {
	var buf bytes.Buffer

	lineNum := 0
	for buf.Len() < sizeBytes {
		line := fmt.Sprintf(`{"id": %d, "name": "User %d", "email": "user%d@example.com", "role": "user", "data": {"field1": "value", "field2": 123}}`,
			lineNum, lineNum, lineNum)
		buf.WriteString(line)
		buf.WriteString("\n")
		lineNum++
	}

	return buf.Bytes()
}

func BenchmarkMarkdownParsing(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
	}

	engine := New()

	for _, size := range sizes {
		content := generateMarkdown(size.size)

		b.Run(size.name, func(b *testing.B) {
			b.SetBytes(int64(len(content)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := engine.ParseDocument(content, "test.md")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkJSONParsing(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
	}

	for _, size := range sizes {
		content := generateJSON(size.size)

		b.Run(size.name, func(b *testing.B) {
			b.SetBytes(int64(len(content)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := DetectAndParse(content, "test.json")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkJSONLParsing(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
	}

	for _, size := range sizes {
		content := generateJSONL(size.size)

		b.Run(size.name, func(b *testing.B) {
			b.SetBytes(int64(len(content)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := DetectAndParse(content, "test.jsonl")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkHeadingsQuery(b *testing.B) {
	sizes := []struct {
		name             string
		size             int
		expectedHeadings int
	}{
		{"1KB", 1024, 5},
		{"10KB", 10 * 1024, 30},
		{"100KB", 100 * 1024, 300},
		{"1MB", 1024 * 1024, 3000},
	}

	engine := New()

	for _, size := range sizes {
		content := generateMarkdown(size.size)
		doc, err := engine.ParseDocument(content, "test.md")
		if err != nil {
			b.Fatal(err)
		}

		b.Run(size.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				headings := doc.GetHeadings()
				if len(headings) < size.expectedHeadings/2 {
					b.Fatalf("expected at least %d headings, got %d", size.expectedHeadings/2, len(headings))
				}
			}
		})
	}
}

func BenchmarkCodeBlockQuery(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
	}

	engine := New()

	for _, size := range sizes {
		content := generateMarkdown(size.size)
		doc, err := engine.ParseDocument(content, "test.md")
		if err != nil {
			b.Fatal(err)
		}

		b.Run(size.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				blocks := doc.GetCodeBlocks("python")
				_ = blocks
			}
		})
	}
}

func BenchmarkSectionQuery(b *testing.B) {
	// Create document with known section names
	content := []byte(`# Main Document

## Introduction

This is the introduction section.

## API Reference

This is the API reference section with lots of content.

### Authentication

How to authenticate.

### Endpoints

List of endpoints.

## Examples

Code examples here.

## Conclusion

Final thoughts.
`)

	engine := New()
	doc, err := engine.ParseDocument(content, "test.md")
	if err != nil {
		b.Fatal(err)
	}

	b.Run("GetSection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			section, _ := doc.GetSection("API Reference")
			_ = section
		}
	})

	b.Run("GetSectionFuzzy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			section, _ := doc.GetSection("api")
			_ = section
		}
	})
}

func BenchmarkReadableText(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
	}

	engine := New()

	for _, size := range sizes {
		content := generateMarkdown(size.size)
		doc, err := engine.ParseDocument(content, "test.md")
		if err != nil {
			b.Fatal(err)
		}

		b.Run(size.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				text := doc.ReadableText()
				_ = text
			}
		})
	}
}

func BenchmarkMultipleDocuments(b *testing.B) {
	// Simulate loading multiple documents
	docCounts := []int{10, 100, 1000}

	engine := New()
	content := generateMarkdown(10 * 1024) // 10KB each

	for _, count := range docCounts {
		b.Run(fmt.Sprintf("%d_docs", count), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				docs := make([]*Document, count)
				for j := 0; j < count; j++ {
					doc, err := engine.ParseDocument(content, fmt.Sprintf("doc%d.md", j))
					if err != nil {
						b.Fatal(err)
					}
					docs[j] = doc
				}

				// Query all documents
				totalHeadings := 0
				for _, doc := range docs {
					totalHeadings += len(doc.GetHeadings())
				}
			}
		})
	}
}

func BenchmarkMemoryUsage(b *testing.B) {
	// This benchmark helps understand memory patterns
	sizes := []int{1024, 10 * 1024, 100 * 1024, 1024 * 1024}

	engine := New()

	for _, size := range sizes {
		content := generateMarkdown(size)

		b.Run(fmt.Sprintf("%dKB", size/1024), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				doc, _ := engine.ParseDocument(content, "test.md")
				_ = doc.GetHeadings()
				_ = doc.GetCodeBlocks()
				_ = doc.GetSections()
			}
		})
	}
}

func BenchmarkBuildTree(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
	}

	modes := []struct {
		name string
		mode TreeMode
	}{
		{"compact", TreeModeCompact},
		{"preview", TreeModePreview},
		{"full", TreeModeFull},
	}

	engine := New()

	for _, size := range sizes {
		content := generateMarkdown(size.size)
		doc, err := engine.ParseDocument(content, "test.md")
		if err != nil {
			b.Fatal(err)
		}

		for _, mode := range modes {
			b.Run(size.name+"/"+mode.name, func(b *testing.B) {
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					result := doc.BuildTree(mode.mode)
					_ = result
				}
			})
		}
	}
}

func BenchmarkSearch(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
	}

	engine := New()

	for _, size := range sizes {
		content := generateMarkdown(size.size)
		doc, err := engine.ParseDocument(content, "test.md")
		if err != nil {
			b.Fatal(err)
		}

		b.Run(size.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				results := doc.Search("example")
				_ = results
			}
		})
	}
}

// DetectAndParse is a helper for benchmarking - parses based on extension
func DetectAndParse(content []byte, path string) (*Document, error) {
	format := DetectFormat(path, content)

	// For now, only markdown is supported directly in lib
	// Other formats go through mql engine
	if format == FormatMarkdown {
		engine := New()
		return engine.ParseDocument(content, path)
	}

	// Stub for other formats - they need the mql engine
	doc := &Document{
		path:   path,
		format: format,
	}

	// Parse based on format
	switch format {
	case FormatJSON:
		// Simple key extraction
		text := string(content)
		if strings.Contains(text, "{") {
			doc.readableText = text
		}
	case FormatJSONL:
		lines := strings.Split(string(content), "\n")
		doc.readableText = fmt.Sprintf("JSONL with %d lines", len(lines))
	}

	return doc, nil
}
