// Package mq provides a markdown processing and query evaluation engine.
// It enables efficient extraction and manipulation of markdown content
// with support for YAML frontmatter metadata and pre-computed indexes
// for fast lookups.
//
// The library provides multiple ways to interact with markdown documents:
//   - Direct API calls for type-safe access
//   - Fluent builder pattern for chainable operations
//   - Support for external query languages via compilation
//
// Example usage:
//
//	engine := mq.New()
//	doc, err := engine.LoadDocument("README.md")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	Direct API access
//	headings := doc.GetHeadings(1, 2)  // Get H1 and H2 headings
//	section, _ := doc.GetSection("Introduction")
//	codeBlocks := doc.GetCodeBlocks("go", "python")
package mq
