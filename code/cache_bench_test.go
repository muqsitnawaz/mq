package code

import (
	"os"
	"path/filepath"
	"testing"

	mq "github.com/muqsitnawaz/mq/lib"
)

// BenchmarkColdParse measures parsing without cache.
func BenchmarkColdParse(b *testing.B) {
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

// BenchmarkWarmCacheHit measures cache lookup after initial parse.
func BenchmarkWarmCacheHit(b *testing.B) {
	// Write test file to a real path so cache can stat it
	tmp := b.TempDir()
	src := testdataPath("sample.go")
	content, err := os.ReadFile(src)
	if err != nil {
		b.Fatal(err)
	}
	dest := filepath.Join(tmp, "sample.go")
	if err := os.WriteFile(dest, content, 0644); err != nil {
		b.Fatal(err)
	}

	// Open cache in temp dir
	cache, err := mq.OpenCache(filepath.Join(tmp, "cache.db"))
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()

	// Cold parse + store
	p := NewParser()
	doc, err := p.Parse(content, dest)
	if err != nil {
		b.Fatal(err)
	}
	info, err := os.Stat(dest)
	if err != nil {
		b.Fatal(err)
	}
	cache.StoreFile(dest, content, info, doc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cached := cache.LookupFile(dest)
		if cached == nil {
			b.Fatal("cache miss on warm lookup")
		}
	}
}

// BenchmarkColdParseLargeGo measures parsing a large file without cache.
func BenchmarkColdParseLargeGo(b *testing.B) {
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

// BenchmarkWarmCacheHitLargeGo measures cache lookup for a large file.
func BenchmarkWarmCacheHitLargeGo(b *testing.B) {
	tmp := b.TempDir()
	src := testdataPath("../lib/tree.go")
	content, err := os.ReadFile(src)
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("file size: %d bytes", len(content))
	dest := filepath.Join(tmp, "tree.go")
	if err := os.WriteFile(dest, content, 0644); err != nil {
		b.Fatal(err)
	}

	cache, err := mq.OpenCache(filepath.Join(tmp, "cache.db"))
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()

	p := NewParser()
	doc, err := p.Parse(content, dest)
	if err != nil {
		b.Fatal(err)
	}
	info, _ := os.Stat(dest)
	cache.StoreFile(dest, content, info, doc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cached := cache.LookupFile(dest)
		if cached == nil {
			b.Fatal("cache miss on warm lookup")
		}
	}
}
