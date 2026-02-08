package data

import (
	"bytes"
	"fmt"
	"testing"
)

// generateYAML creates a YAML document of approximately the given size.
func generateYAML(sizeBytes int) []byte {
	var buf bytes.Buffer

	sectionTemplate := `section_%d:
  name: "Section %d"
  description: "This is a description for section %d with some content"
  enabled: true
  values:
    - 1
    - 2
    - 3
  nested:
    field1: "value1"
    field2: "value2"
    subsection:
      key1: "data"
      key2: 42
`

	sectionNum := 0
	for buf.Len() < sizeBytes {
		buf.WriteString(fmt.Sprintf(sectionTemplate, sectionNum, sectionNum, sectionNum))
		sectionNum++
	}

	return buf.Bytes()
}

func BenchmarkYAMLParsing(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
	}

	parser := NewYAMLParser()

	for _, size := range sizes {
		content := generateYAML(size.size)

		b.Run(size.name, func(b *testing.B) {
			b.SetBytes(int64(len(content)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := parser.Parse(content, "test.yaml")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
