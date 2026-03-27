package pdf_test

import (
	"os"
	"path/filepath"
	"testing"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/muqsitnawaz/mq/pdf"
)

// pdfTestFiles defines benchmark PDFs of varying sizes.
var pdfTestFiles = []struct {
	name string
	path string
}{
	{"bert-760K", "testdata/papers/ml/bert.pdf"},
	{"attention-2M", "testdata/papers/ml/attention.pdf"},
	{"raft-6M", "testdata/papers/systems/raft.pdf"},
}

func resolveBenchmarkPDF(b testing.TB, relPath string) (string, int64) {
	b.Helper()

	absPath, err := filepath.Abs(relPath)
	if err != nil {
		b.Skip("cannot resolve path:", relPath)
	}

	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		b.Skip("test PDF not found:", absPath)
	}
	if err != nil {
		b.Fatal(err)
	}

	return absPath, info.Size()
}

func mustParseBenchmarkPDF(b testing.TB, parser *pdf.Parser, path string) *mq.Document {
	b.Helper()

	doc, err := parser.ParseFile(path)
	if err != nil {
		b.Fatal(err)
	}
	return doc
}

// BenchmarkPDFParseCold measures end-to-end cold parse (pdftotext + PyMuPDF + section building).
func BenchmarkPDFParseCold(b *testing.B) {
	parser := pdf.NewParser()

	for _, tf := range pdfTestFiles {
		absPath, size := resolveBenchmarkPDF(b, tf.path)

		b.Run(tf.name, func(b *testing.B) {
			b.SetBytes(size)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = mustParseBenchmarkPDF(b, parser, absPath)
			}
		})
	}
}

// BenchmarkPDFCacheWarm measures cache-hit load time (no parsing, just deserialization).
func BenchmarkPDFCacheWarm(b *testing.B) {
	parser := pdf.NewParser()

	for _, tf := range pdfTestFiles {
		absPath, size := resolveBenchmarkPDF(b, tf.path)

		// Parse once, store in cache
		doc := mustParseBenchmarkPDF(b, parser, absPath)

		dbPath := filepath.Join(b.TempDir(), "bench-cache.db")
		cache, err := mq.OpenCache(dbPath)
		if err != nil {
			b.Fatal(err)
		}
		b.Cleanup(func() { cache.Close() })

		content, info, err := mq.ReadFileWithStat(absPath)
		if err != nil {
			b.Fatal(err)
		}
		if err := cache.StoreFile(absPath, content, info, doc); err != nil {
			b.Fatal(err)
		}

		b.Run(tf.name, func(b *testing.B) {
			b.SetBytes(size)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				cached := cache.LookupFile(absPath)
				if cached == nil {
					b.Fatal("cache miss on warm benchmark")
				}
			}
		})
	}
}

// BenchmarkPDFBuildTree measures tree building from an already-parsed PDF document.
func BenchmarkPDFBuildTree(b *testing.B) {
	parser := pdf.NewParser()

	for _, tf := range pdfTestFiles {
		absPath, _ := resolveBenchmarkPDF(b, tf.path)
		doc := mustParseBenchmarkPDF(b, parser, absPath)

		b.Run(tf.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				tree := doc.BuildTree()
				_ = tree.String()
			}
		})
	}
}

// BenchmarkPDFSearch measures section search on a parsed PDF document.
func BenchmarkPDFSearch(b *testing.B) {
	parser := pdf.NewParser()

	for _, tf := range pdfTestFiles {
		absPath, _ := resolveBenchmarkPDF(b, tf.path)
		doc := mustParseBenchmarkPDF(b, parser, absPath)

		b.Run(tf.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				results := doc.Search("attention")
				_ = results
			}
		})
	}
}
