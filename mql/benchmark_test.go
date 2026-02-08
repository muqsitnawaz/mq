package mql

import (
	"bytes"
	"fmt"
	"testing"

	mq "github.com/muqsitnawaz/mq/lib"
)

// generateMarkdownForMQL creates a markdown document for MQL benchmarks.
func generateMarkdownForMQL() []byte {
	var buf bytes.Buffer

	buf.WriteString("# Main Document\n\n")
	buf.WriteString("This is the introduction.\n\n")

	for i := 1; i <= 20; i++ {
		buf.WriteString(fmt.Sprintf("## Section %d\n\n", i))
		buf.WriteString(fmt.Sprintf("Content for section %d with some details.\n\n", i))
		buf.WriteString(fmt.Sprintf("### Subsection %d.1\n\n", i))
		buf.WriteString("More detailed content here.\n\n")

		buf.WriteString(fmt.Sprintf("```go\nfunc example%d() string {\n    return \"hello\"\n}\n```\n\n", i))

		buf.WriteString(fmt.Sprintf("```python\ndef example_%d():\n    return \"hello\"\n```\n\n", i))

		buf.WriteString(fmt.Sprintf("### Subsection %d.2\n\n", i))
		buf.WriteString("Additional content with [links](https://example.com).\n\n")
	}

	return buf.Bytes()
}

func BenchmarkMQLQuery(b *testing.B) {
	content := generateMarkdownForMQL()

	engine := mq.New()
	doc, err := engine.ParseDocument(content, "test.md")
	if err != nil {
		b.Fatal(err)
	}

	queries := []struct {
		name  string
		query string
	}{
		{"headings", ".headings"},
		{"sections", ".sections"},
		{"code_go", `.code("go")`},
		{"section_pipe_text", `.section("Section 1") | .text`},
		{"headings_filter", `.headings | filter(.level == 2)`},
	}

	for _, q := range queries {
		b.Run(q.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result, err := ExecuteQuery(doc, q.query)
				if err != nil {
					b.Fatal(err)
				}
				_ = result
			}
		})
	}
}
