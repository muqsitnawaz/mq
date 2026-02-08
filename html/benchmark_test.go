package html

import (
	"bytes"
	"fmt"
	"testing"
)

// generateHTML creates an HTML document of approximately the given size.
func generateHTML(sizeBytes int) []byte {
	var buf bytes.Buffer

	buf.WriteString(`<!DOCTYPE html>
<html>
<head><title>Benchmark Document</title></head>
<body>
<main>
<h1>Benchmark Document</h1>
<p>This is a generated document for benchmarking purposes.</p>
`)

	sectionTemplate := `
<h2>Section %d</h2>
<p>This is the content for section %d. It contains some text that is meant to
simulate real documentation content. Here is some more text to make it realistic.</p>

<h3>Subsection %d.1</h3>
<p>More detailed content here with examples and explanations that would be
typical in a technical document.</p>

<pre><code class="language-python">def example_%d():
    """Example function for section %d."""
    return "Hello from section %d"
</code></pre>

<h3>Subsection %d.2</h3>
<p>Additional content with links like <a href="https://example.com">example</a> and
some <strong>bold</strong> and <em>italic</em> text for good measure.</p>

<table>
<thead><tr><th>Column A</th><th>Column B</th><th>Column C</th></tr></thead>
<tbody>
<tr><td>Value 1</td><td>Value 2</td><td>Value 3</td></tr>
<tr><td>Value 4</td><td>Value 5</td><td>Value 6</td></tr>
</tbody>
</table>
`

	sectionNum := 1
	for buf.Len() < sizeBytes {
		section := fmt.Sprintf(sectionTemplate,
			sectionNum, sectionNum, sectionNum, sectionNum, sectionNum, sectionNum, sectionNum)
		buf.WriteString(section)
		sectionNum++
	}

	buf.WriteString("</main>\n</body>\n</html>")
	return buf.Bytes()
}

func BenchmarkHTMLParsing(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
	}

	parser := NewParser()

	for _, size := range sizes {
		content := generateHTML(size.size)

		b.Run(size.name, func(b *testing.B) {
			b.SetBytes(int64(len(content)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := parser.Parse(content, "test.html")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkHTMLReadability(b *testing.B) {
	// Generate a page with lots of noise (nav, sidebar, footer) around main content
	var buf bytes.Buffer
	buf.WriteString(`<!DOCTYPE html>
<html>
<head><title>Readability Test</title></head>
<body>
<nav><ul><li><a href="/">Home</a></li><li><a href="/about">About</a></li></ul></nav>
<aside class="sidebar"><p>Sidebar content that should be stripped.</p></aside>
<main>
<article>
<h1>Main Article Title</h1>
`)
	for i := 0; i < 50; i++ {
		buf.WriteString(fmt.Sprintf(`<p>This is paragraph %d of the main article content. It contains enough text
to simulate a real article that would benefit from readability extraction. The algorithm
should identify this as the primary content area and strip away navigation and sidebars.</p>
`, i))
	}
	buf.WriteString(`</article>
</main>
<footer><p>Footer content with links and copyright.</p></footer>
</body>
</html>`)

	content := buf.Bytes()
	parser := NewParser()

	b.SetBytes(int64(len(content)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(content, "test.html")
		if err != nil {
			b.Fatal(err)
		}
	}
}
