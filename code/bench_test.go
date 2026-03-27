package code

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "testdata", name)
}

func BenchmarkParseGo(b *testing.B) {
	content, err := os.ReadFile(testdataPath("sample.go"))
	if err != nil {
		b.Fatal(err)
	}
	p := NewParser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(content, "sample.go")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParsePython(b *testing.B) {
	content, err := os.ReadFile(testdataPath("sample.py"))
	if err != nil {
		b.Fatal(err)
	}
	p := NewParser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(content, "sample.py")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseTypeScript(b *testing.B) {
	content, err := os.ReadFile(testdataPath("sample.ts"))
	if err != nil {
		b.Fatal(err)
	}
	p := NewParser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(content, "sample.ts")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseRust(b *testing.B) {
	content, err := os.ReadFile(testdataPath("sample.rs"))
	if err != nil {
		b.Fatal(err)
	}
	p := NewParser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(content, "sample.rs")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseLargeGo tests parsing a real, larger Go file.
func BenchmarkParseLargeGo(b *testing.B) {
	// Use mq's own tree.go as a realistic large file
	content, err := os.ReadFile(testdataPath("../lib/tree.go"))
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("file size: %d bytes", len(content))
	p := NewParser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(content, "tree.go")
		if err != nil {
			b.Fatal(err)
		}
	}
}
