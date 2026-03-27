package pdf_test

import (
	"os"
	"path/filepath"
	"testing"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/muqsitnawaz/mq/pdf"
)

// pdfTestFiles returns paths to test PDFs of varying sizes.
// Each entry has a short name and a path relative to testdata/.
var pdfTestFiles = []struct {
	name string
	path string
}{
	{"bert-760K", "testdata/papers/ml/bert.pdf"},
	{"attention-2M", "testdata/papers/ml/attention.pdf"},
	{"raft-6M", "testdata/papers/systems/raft.pdf"},
}

// BenchmarkPDFParseCold measures end-to-end cold parse (pdftotext + PyMuPDF + section building).
func BenchmarkPDFParseCold(b *testing.B) {
	parser := pdf.NewParser()

	for _, tf := range pdfTestFiles {
		absPath, err := filepath.Abs(tf.path)
		if err != nil {
			b.Skip("cannot resolve path:", tf.path)
		}
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			b.Skip("test PDF not found:", absPath)
		}

		b.Run(tf.name, func(b *testing.B) {
			info, _ := os.Stat(absPath)
			b.SetBytes(info.Size())
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				doc, err := parser.ParseFile(absPath)
				if err != nil {
					b.Fatal(err)
				}
				_ = doc
			}
		})
	}
}

// BenchmarkPDFCacheWarm measures cache-hit load time (no parsing, just deserialization).
func BenchmarkPDFCacheWarm(b *testing.B) {
	parser := pdf.NewParser()

	for _, tf := range pdfTestFiles {
		absPath, err := filepath.Abs(tf.path)
		if err != nil {
			b.Skip("cannot resolve path:", tf.path)
		}
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			b.Skip("test PDF not found:", absPath)
		}

		// Parse once, store in cache
		doc, err := parser.ParseFile(absPath)
		if err != nil {
			b.Fatal(err)
		}

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
			b.SetBytes(info.Size())
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
		absPath, err := filepath.Abs(tf.path)
		if err != nil {
			b.Skip("cannot resolve path:", tf.path)
		}
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			b.Skip("test PDF not found:", absPath)
		}

		doc, err := parser.ParseFile(absPath)
		if err != nil {
			b.Fatal(err)
		}

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
		absPath, err := filepath.Abs(tf.path)
		if err != nil {
			b.Skip("cannot resolve path:", tf.path)
		}
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			b.Skip("test PDF not found:", absPath)
		}

		doc, err := parser.ParseFile(absPath)
		if err != nil {
			b.Fatal(err)
		}

		b.Run(tf.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				results := doc.Search("attention")
				_ = results
			}
		})
	}
}
