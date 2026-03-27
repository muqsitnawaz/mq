package mql

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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

type benchmarkDirCorpus struct {
	name  string
	path  string
	query string
}

func BenchmarkDirectorySearch(b *testing.B) {
	corpora := resolveBenchmarkDirCorpora(b)

	for _, corpus := range corpora {
		corpus := corpus
		b.Run(corpus.name, func(b *testing.B) {
			b.Run("cold", func(b *testing.B) {
				benchmarkDirectorySearchCold(b, corpus)
			})
			b.Run("warm_exact_repeat", func(b *testing.B) {
				benchmarkDirectorySearchWarm(b, corpus)
			})
			if corpus.name == "growth-book" {
				b.Run("partial_invalidation", func(b *testing.B) {
					benchmarkDirectorySearchPartialInvalidation(b, corpus)
				})
			}
		})
	}
}

func resolveBenchmarkDirCorpora(b testing.TB) []benchmarkDirCorpus {
	b.Helper()

	var corpora []benchmarkDirCorpus

	if path := growthBookBenchmarkDir(); path != "" {
		corpora = append(corpora, benchmarkDirCorpus{
			name:  "growth-book",
			path:  path,
			query: "calibration",
		})
	}

	if home, err := os.UserHomeDir(); err == nil {
		if path := existingBenchmarkDir(filepath.Join(home, ".rush", "sessions")); path != "" {
			corpora = append(corpora, benchmarkDirCorpus{
				name:  "rush-sessions",
				path:  path,
				query: "requires OAuth",
			})
		}
	}

	if len(corpora) == 0 {
		b.Skip("no local directory benchmark corpus found")
	}

	return corpora
}

func growthBookBenchmarkDir() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), ".."))
	return existingBenchmarkDir(filepath.Join(repoRoot, "..", "agents", "growth", "book"))
}

func existingBenchmarkDir(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		return ""
	}
	return absPath
}

func benchmarkDirectorySearchCold(b *testing.B, corpus benchmarkDirCorpus) {
	engine := newBenchmarkDirectoryEngine(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if err := engine.cache.Clear(); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		results, err := engine.SearchDir(corpus.path, corpus.query)
		if err != nil {
			b.Fatal(err)
		}
		if len(results.Matches) == 0 {
			b.Fatalf("expected matches for %q in %s", corpus.query, corpus.path)
		}
	}
}

func benchmarkDirectorySearchWarm(b *testing.B, corpus benchmarkDirCorpus) {
	engine := newBenchmarkDirectoryEngine(b)

	primed, err := engine.SearchDir(corpus.path, corpus.query)
	if err != nil {
		b.Fatal(err)
	}
	expectedMatches := len(primed.Matches)
	if expectedMatches == 0 {
		b.Fatalf("expected matches for %q in %s", corpus.query, corpus.path)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := engine.SearchDir(corpus.path, corpus.query)
		if err != nil {
			b.Fatal(err)
		}
		if len(results.Matches) != expectedMatches {
			b.Fatalf("warm cache changed match count: got %d want %d", len(results.Matches), expectedMatches)
		}
	}
}

func benchmarkDirectorySearchPartialInvalidation(b *testing.B, corpus benchmarkDirCorpus) {
	copiedDir := filepath.Join(b.TempDir(), corpus.name)
	if err := copyBenchmarkDir(corpus.path, copiedDir); err != nil {
		b.Fatal(err)
	}

	mutableFile, err := pickMutableSupportedFile(copiedDir)
	if err != nil {
		b.Fatal(err)
	}

	engine := newBenchmarkDirectoryEngine(b)
	primed, err := engine.SearchDir(copiedDir, corpus.query)
	if err != nil {
		b.Fatal(err)
	}
	expectedMatches := len(primed.Matches)
	if expectedMatches == 0 {
		b.Fatalf("expected matches for %q in %s", corpus.query, copiedDir)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if err := bumpBenchmarkFileMtime(mutableFile, i); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		results, err := engine.SearchDir(copiedDir, corpus.query)
		if err != nil {
			b.Fatal(err)
		}
		if len(results.Matches) != expectedMatches {
			b.Fatalf("partial invalidation changed match count: got %d want %d", len(results.Matches), expectedMatches)
		}
	}
}

func newBenchmarkDirectoryEngine(b testing.TB) *Engine {
	b.Helper()

	engine := New()
	if engine.cache != nil {
		_ = engine.cache.Close()
	}

	cache, err := mq.OpenCache(filepath.Join(b.TempDir(), "benchmark-cache.db"))
	if err != nil {
		b.Fatal(err)
	}
	engine.cache = cache
	b.Cleanup(func() {
		engine.Close()
	})

	return engine
}

func copyBenchmarkDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		in, err := os.Open(path)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			in.Close()
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			in.Close()
			out.Close()
			return err
		}
		if err := in.Close(); err != nil {
			out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}

		return os.Chtimes(target, info.ModTime(), info.ModTime())
	})
}

func pickMutableSupportedFile(root string) (string, error) {
	var picked string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !isBenchmarkTraversalFile(path) {
			return nil
		}
		picked = path
		return io.EOF
	})
	if err != nil && err != io.EOF {
		return "", err
	}
	if picked == "" {
		return "", fmt.Errorf("no supported file found under %s", root)
	}
	return picked, nil
}

func isBenchmarkTraversalFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".mdown", ".mkd", ".html", ".htm", ".xhtml", ".pdf", ".json", ".jsonl", ".ndjson", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func bumpBenchmarkFileMtime(path string, iter int) error {
	now := time.Unix(0, time.Now().UnixNano()+int64(iter+1))
	return os.Chtimes(path, now, now)
}
