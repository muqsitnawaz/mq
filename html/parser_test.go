package html_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/muqsitnawaz/mq/html"
	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserFormat(t *testing.T) {
	p := html.NewParser()
	assert.Equal(t, mq.FormatHTML, p.Format())
}

func TestParseSimpleHTML(t *testing.T) {
	parser := html.NewParser()
	doc, err := parser.ParseFile(filepath.Join("testdata", "simple.html"))
	require.NoError(t, err)

	// Check format
	assert.Equal(t, mq.FormatHTML, doc.Format())

	// Check title
	assert.Equal(t, "Test Page - Simple Document", doc.Title())

	// Check headings
	h1s := doc.GetHeadings(1)
	require.Len(t, h1s, 1)
	assert.Equal(t, "Main Article Title", h1s[0].Text)

	h2s := doc.GetHeadings(2)
	require.Len(t, h2s, 3)
	assert.Equal(t, "First Section", h2s[0].Text)
	assert.Equal(t, "Second Section", h2s[1].Text)
	assert.Equal(t, "Third Section", h2s[2].Text)

	h3s := doc.GetHeadings(3)
	require.Len(t, h3s, 1)
	assert.Equal(t, "Subsection", h3s[0].Text)

	// Check code blocks
	codeBlocks := doc.GetCodeBlocks()
	require.Len(t, codeBlocks, 2)

	pythonCode := doc.GetCodeBlocks("python")
	require.Len(t, pythonCode, 1)
	assert.Contains(t, pythonCode[0].Content, "def hello()")

	goCode := doc.GetCodeBlocks("go")
	require.Len(t, goCode, 1)
	assert.Contains(t, goCode[0].Content, "func main()")

	// Check links (should be only content link, not nav)
	links := doc.GetLinks()
	require.Len(t, links, 1)
	assert.Equal(t, "link", links[0].Text)
	assert.Equal(t, "https://example.com", links[0].URL)

	// Check tables
	tables := doc.GetTables()
	require.Len(t, tables, 1)
	assert.Equal(t, []string{"Name", "Value", "Description"}, tables[0].Headers)
	require.Len(t, tables[0].Rows, 2)

	// Check lists
	lists := doc.GetLists(nil)
	require.Len(t, lists, 1)
	assert.False(t, lists[0].Ordered)
	require.Len(t, lists[0].Items, 3)

	// Check images
	images := doc.GetImages()
	require.Len(t, images, 1)
	assert.Equal(t, "Test image", images[0].AltText)

	// Check readable text exists and doesn't include nav/footer
	readable := doc.ReadableText()
	assert.NotEmpty(t, readable)
	assert.Contains(t, readable, "Main Article Title")
	assert.Contains(t, readable, "important content")
	// Nav links should be stripped
	assert.NotContains(t, readable, "About")
	// Footer should be stripped
	assert.NotContains(t, readable, "Copyright 2024")
}

func TestParseLargeHTML(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("skipping large file test in short mode")
	}

	testFile := filepath.Join("testdata", "w3c-html53.html")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("large test file not found")
	}

	parser := html.NewParser()
	doc, err := parser.ParseFile(testFile)
	require.NoError(t, err)

	// Should have extracted many headings from W3C spec
	allHeadings := doc.GetHeadings()
	t.Logf("W3C HTML5 spec: %d headings extracted", len(allHeadings))
	assert.Greater(t, len(allHeadings), 50, "expected many headings in W3C spec")

	// Should have readable text
	readable := doc.ReadableText()
	t.Logf("W3C HTML5 spec: %d chars readable text", len(readable))
	assert.NotEmpty(t, readable)

	// Check size reduction
	source := doc.Source()
	t.Logf("Size reduction: %d bytes -> %d chars (%.1f%%)",
		len(source), len(readable),
		100.0*float64(len(readable))/float64(len(source)))
}

func TestParseEffectiveGo(t *testing.T) {
	testFile := filepath.Join("testdata", "effective-go.html")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("effective-go test file not found")
	}

	parser := html.NewParser()
	doc, err := parser.ParseFile(testFile)
	require.NoError(t, err)

	// Effective Go should have Go code examples
	goCode := doc.GetCodeBlocks("go")
	t.Logf("Effective Go: %d Go code blocks", len(goCode))

	// Should have meaningful content
	readable := doc.ReadableText()
	t.Logf("Effective Go: %d chars readable", len(readable))
	assert.NotEmpty(t, readable)

	// Get all headings for structure analysis
	headings := doc.GetHeadings()
	t.Logf("Effective Go: %d headings", len(headings))
}

func TestReadabilityExtraction(t *testing.T) {
	// HTML with lots of boilerplate
	htmlContent := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<nav>
    <a href="/home">Home</a>
    <a href="/products">Products</a>
    <a href="/contact">Contact</a>
</nav>
<div class="sidebar">
    <h3>Popular Posts</h3>
    <ul>
        <li><a href="/post1">Post 1</a></li>
        <li><a href="/post2">Post 2</a></li>
    </ul>
</div>
<main>
    <article>
        <h1>Important Article</h1>
        <p>This is the main content that matters.</p>
        <p>More important information here.</p>
    </article>
</main>
<footer>
    <p>Copyright 2024</p>
    <div class="social-share">Share this!</div>
</footer>
<script>alert('should be removed')</script>
</body>
</html>`

	parser := html.NewParser(html.WithReadability(true))
	doc, err := parser.Parse([]byte(htmlContent), "test.html")
	require.NoError(t, err)

	readable := doc.ReadableText()

	// Main content should be present
	assert.Contains(t, readable, "Important Article")
	assert.Contains(t, readable, "main content that matters")

	// Nav/sidebar/footer should be stripped
	assert.NotContains(t, readable, "Popular Posts")
	assert.NotContains(t, readable, "Copyright 2024")
	assert.NotContains(t, readable, "social-share")
}

func TestWithoutReadability(t *testing.T) {
	htmlContent := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<nav><a href="/">Nav Link</a></nav>
<main><p>Content</p></main>
<footer><p>Footer</p></footer>
</body>
</html>`

	// With readability disabled, nav/footer should be included
	parser := html.NewParser(html.WithReadability(false))
	doc, err := parser.Parse([]byte(htmlContent), "test.html")
	require.NoError(t, err)

	readable := doc.ReadableText()
	assert.Contains(t, readable, "Content")
}

func TestSectionHierarchy(t *testing.T) {
	htmlContent := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<main>
<h1>Title</h1>
<h2>Chapter 1</h2>
<h3>Section 1.1</h3>
<h3>Section 1.2</h3>
<h2>Chapter 2</h2>
<h3>Section 2.1</h3>
</main>
</body>
</html>`

	parser := html.NewParser()
	doc, err := parser.Parse([]byte(htmlContent), "test.html")
	require.NoError(t, err)

	// Check sections exist
	chapter1, ok := doc.GetSection("Chapter 1")
	require.True(t, ok)
	assert.NotNil(t, chapter1)

	chapter2, ok := doc.GetSection("Chapter 2")
	require.True(t, ok)
	assert.NotNil(t, chapter2)

	// Check TOC
	toc := doc.GetTableOfContents()
	t.Logf("TOC has %d top-level sections", len(toc))
}

func TestParseHTMLConvenienceFuncs(t *testing.T) {
	htmlContent := []byte(`<!DOCTYPE html>
<html>
<head><title>Quick Test</title></head>
<body><h1>Hello</h1></body>
</html>`)

	// Test ParseHTML
	doc, err := html.ParseHTML(htmlContent, "test.html")
	require.NoError(t, err)
	assert.Equal(t, "Quick Test", doc.Title())

	// Test ParseHTMLWithOptions
	doc, err = html.ParseHTMLWithOptions(htmlContent, "test.html", html.WithReadability(false))
	require.NoError(t, err)
	assert.Equal(t, "Quick Test", doc.Title())
}

func TestCodeLanguageDetection(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "language-prefix",
			html:     `<pre><code class="language-python">print("hello")</code></pre>`,
			expected: "python",
		},
		{
			name:     "lang-prefix",
			html:     `<pre><code class="lang-javascript">console.log('hi')</code></pre>`,
			expected: "javascript",
		},
		{
			name:     "highlight-prefix",
			html:     `<pre><code class="highlight-rust">fn main() {}</code></pre>`,
			expected: "rust",
		},
		{
			name:     "standalone-class",
			html:     `<pre><code class="go">func main() {}</code></pre>`,
			expected: "go",
		},
		{
			name:     "pre-class",
			html:     `<pre class="language-ruby">puts "hello"</pre>`,
			expected: "ruby",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullHTML := `<!DOCTYPE html><html><body><main>` + tt.html + `</main></body></html>`
			doc, err := html.ParseHTML([]byte(fullHTML), "test.html")
			require.NoError(t, err)

			blocks := doc.GetCodeBlocks()
			require.Len(t, blocks, 1)
			assert.Equal(t, tt.expected, blocks[0].Language)
		})
	}
}

func TestSkipElements(t *testing.T) {
	htmlContent := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<main>
<h1>Main</h1>
<p>Content</p>
<div class="ad-banner">Advertisement</div>
<div class="sidebar-widget">Widget</div>
<div id="comments">Comments section</div>
<div aria-hidden="true">Hidden content</div>
<script>malicious()</script>
<style>.hidden{display:none}</style>
</main>
</body>
</html>`

	parser := html.NewParser()
	doc, err := parser.Parse([]byte(htmlContent), "test.html")
	require.NoError(t, err)

	readable := doc.ReadableText()

	// Main content present
	assert.Contains(t, readable, "Main")
	assert.Contains(t, readable, "Content")

	// Skip patterns should remove these
	assert.NotContains(t, readable, "Advertisement")
	assert.NotContains(t, readable, "Widget")
	assert.NotContains(t, readable, "Comments section")
	assert.NotContains(t, readable, "Hidden content")
}
