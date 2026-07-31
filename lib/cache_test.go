package mq_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCache(t *testing.T) *mq.Cache {
	t.Helper()
	dir := t.TempDir()
	c, err := mq.OpenCache(filepath.Join(dir, "test-cache.db"))
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })
	return c
}

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

// storeTestFile is a test helper that reads a file and stores it in the cache.
func storeTestFile(t *testing.T, c *mq.Cache, path string, doc *mq.Document) {
	t.Helper()
	content, info, err := mq.ReadFileWithStat(path)
	require.NoError(t, err)
	require.NoError(t, c.StoreFile(path, content, info, doc))
}

func makeTestDocument(path string, content []byte) *mq.Document {
	headings := []*mq.Heading{
		{Level: 1, Text: "Title", Line: 1},
		{Level: 2, Text: "Section A", Line: 3},
		{Level: 2, Text: "Section B", Line: 6},
	}
	textContent := "# Title\n\n## Section A\nSome content here.\n\n## Section B\nMore content."
	textBytes := []byte(textContent)

	sections := []*mq.Section{
		mq.NewSectionWithSource(headings[0], 1, 7, textBytes),
		mq.NewSectionWithSource(headings[1], 3, 5, textBytes),
		mq.NewSectionWithSource(headings[2], 6, 7, textBytes),
	}
	sections[1].Parent = sections[0]
	sections[2].Parent = sections[0]
	sections[0].Children = []*mq.Section{sections[1], sections[2]}

	codeBlocks := []*mq.CodeBlock{
		{Language: "go", Content: "func main() {}", Lines: 1},
	}
	links := []*mq.Link{
		{Text: "Example", URL: "https://example.com"},
	}

	return mq.NewDocument(
		content, path, mq.FormatMarkdown, "Title",
		headings, sections, codeBlocks, links, nil, nil, nil,
		textContent,
	)
}

// -------------------------------------------------------------------
// Basic cache operations
// -------------------------------------------------------------------

func TestCacheOpenClose(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cache.db")

	c, err := mq.OpenCache(dbPath)
	require.NoError(t, err)
	assert.Equal(t, dbPath, c.Path())

	files, dirs, docs := c.Stats()
	assert.Equal(t, 0, files)
	assert.Equal(t, 0, dirs)
	assert.Equal(t, 0, docs)

	require.NoError(t, c.Close())

	// Re-open should work
	c2, err := mq.OpenCache(dbPath)
	require.NoError(t, err)
	require.NoError(t, c2.Close())
}

func TestCacheSchemaVersionMismatch(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cache.db")

	// Create cache and store something
	c, err := mq.OpenCache(dbPath)
	require.NoError(t, err)

	testDir := t.TempDir()
	path := writeTestFile(t, testDir, "test.md", "# Hello\nWorld")
	content, _ := os.ReadFile(path)
	doc := makeTestDocument(path, content)

	storeTestFile(t, c, path, doc)
	_, _, docs := c.Stats()
	assert.Equal(t, 1, docs)
	require.NoError(t, c.Close())

	// Re-opening with same version should preserve data
	c2, err := mq.OpenCache(dbPath)
	require.NoError(t, err)
	_, _, docs = c2.Stats()
	assert.Equal(t, 1, docs)
	require.NoError(t, c2.Close())
}

// -------------------------------------------------------------------
// File-level caching
// -------------------------------------------------------------------

func TestCacheStoreAndLookup(t *testing.T) {
	c := newTestCache(t)
	testDir := t.TempDir()

	path := writeTestFile(t, testDir, "doc.md", "# Hello\n\n## World\nContent here.")
	content, _ := os.ReadFile(path)
	doc := makeTestDocument(path, content)

	// Not cached yet
	assert.Nil(t, c.LookupFile(path))

	// Store it
	storeTestFile(t, c, path, doc)

	// Should be cached now
	cached := c.LookupFile(path)
	require.NotNil(t, cached)

	// Verify structure is preserved
	assert.Equal(t, doc.Title(), cached.Title())
	assert.Equal(t, doc.Format(), cached.Format())
	assert.Equal(t, len(doc.GetHeadings()), len(cached.GetHeadings()))
	assert.Equal(t, len(doc.GetSections()), len(cached.GetSections()))
	assert.Equal(t, len(doc.GetCodeBlocks()), len(cached.GetCodeBlocks()))
	assert.Equal(t, len(doc.GetLinks()), len(cached.GetLinks()))

	// Verify heading content
	origHeadings := doc.GetHeadings()
	cachedHeadings := cached.GetHeadings()
	for i := range origHeadings {
		assert.Equal(t, origHeadings[i].Text, cachedHeadings[i].Text)
		assert.Equal(t, origHeadings[i].Level, cachedHeadings[i].Level)
		assert.Equal(t, origHeadings[i].Line, cachedHeadings[i].Line)
	}

	// Verify section hierarchy
	origSections := doc.GetSections()
	cachedSections := cached.GetSections()
	for i := range origSections {
		assert.Equal(t, origSections[i].Heading.Text, cachedSections[i].Heading.Text)
		assert.Equal(t, origSections[i].Start, cachedSections[i].Start)
		assert.Equal(t, origSections[i].End, cachedSections[i].End)
		assert.Equal(t, len(origSections[i].Children), len(cachedSections[i].Children))
	}

	// Verify parent/child relationships
	assert.Nil(t, cachedSections[0].Parent)
	assert.NotNil(t, cachedSections[1].Parent)
	assert.Equal(t, cachedSections[0].Heading.Text, cachedSections[1].Parent.Heading.Text)
}

func TestCacheSectionTextPreserved(t *testing.T) {
	c := newTestCache(t)
	testDir := t.TempDir()

	mdContent := "# Title\n\n## Section A\nSome content here.\n\n## Section B\nMore content."
	path := writeTestFile(t, testDir, "doc.md", mdContent)
	content, _ := os.ReadFile(path)
	doc := makeTestDocument(path, content)

	storeTestFile(t, c, path, doc)
	cached := c.LookupFile(path)
	require.NotNil(t, cached)

	// Section text should be extractable
	for _, section := range cached.GetSections() {
		text := section.GetText()
		if section.Start > 0 {
			assert.NotEmpty(t, text, "section %q should have text", section.Heading.Text)
		}
	}
}

func TestCacheInvalidatedOnFileChange(t *testing.T) {
	c := newTestCache(t)
	testDir := t.TempDir()

	path := writeTestFile(t, testDir, "doc.md", "# Original")
	content, _ := os.ReadFile(path)
	doc := makeTestDocument(path, content)

	storeTestFile(t, c, path, doc)
	assert.NotNil(t, c.LookupFile(path))

	// Modify the file (need to ensure mtime changes)
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(path, []byte("# Modified content"), 0644)

	// Cache should miss — mtime changed
	assert.Nil(t, c.LookupFile(path))
}

func TestCacheMissingFile(t *testing.T) {
	c := newTestCache(t)
	assert.Nil(t, c.LookupFile("/nonexistent/path.md"))
}

// -------------------------------------------------------------------
// Merkle directory tree
// -------------------------------------------------------------------

func TestDirChangedEmptyDir(t *testing.T) {
	c := newTestCache(t)
	dir := t.TempDir()

	// First check — no cached hash, so DirChanged returns true
	assert.True(t, c.DirChanged(dir))

	// Store the hash
	c.UpdateDirHash(dir)

	// Same dir, nothing changed
	assert.False(t, c.DirChanged(dir))
}

func TestDirChangedDetectsNewFile(t *testing.T) {
	c := newTestCache(t)
	dir := t.TempDir()

	c.UpdateDirHash(dir)
	assert.False(t, c.DirChanged(dir))

	// Add a file
	writeTestFile(t, dir, "new.md", "# New")

	// Should detect the change
	assert.True(t, c.DirChanged(dir))
}

func TestDirChangedDetectsModifiedFile(t *testing.T) {
	c := newTestCache(t)
	dir := t.TempDir()

	path := writeTestFile(t, dir, "doc.md", "# Original")
	c.UpdateDirHash(dir)
	assert.False(t, c.DirChanged(dir))

	// Modify the file
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(path, []byte("# Modified"), 0644)

	assert.True(t, c.DirChanged(dir))
}

func TestDirChangedDetectsDeletedFile(t *testing.T) {
	c := newTestCache(t)
	dir := t.TempDir()

	path := writeTestFile(t, dir, "doc.md", "# Hello")
	c.UpdateDirHash(dir)
	assert.False(t, c.DirChanged(dir))

	// Delete the file
	os.Remove(path)
	assert.True(t, c.DirChanged(dir))
}

func TestDirChangedNestedSubdirs(t *testing.T) {
	c := newTestCache(t)
	dir := t.TempDir()

	// Create nested structure
	writeTestFile(t, dir, "top.md", "# Top")
	writeTestFile(t, dir, "sub1/a.md", "# A")
	writeTestFile(t, dir, "sub1/b.md", "# B")
	writeTestFile(t, dir, "sub2/deep/nested/c.md", "# C")

	c.UpdateDirHash(dir)
	assert.False(t, c.DirChanged(dir))

	// Modify deeply nested file
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(filepath.Join(dir, "sub2/deep/nested/c.md"), []byte("# Modified C"), 0644)

	// Root should detect the change (Merkle propagation)
	assert.True(t, c.DirChanged(dir))

	// sub1 should NOT have changed
	assert.False(t, c.DirChanged(filepath.Join(dir, "sub1")))

	// sub2 should have changed
	assert.True(t, c.DirChanged(filepath.Join(dir, "sub2")))
}

func TestDirChangedIgnoresHiddenFiles(t *testing.T) {
	c := newTestCache(t)
	dir := t.TempDir()

	writeTestFile(t, dir, "visible.md", "# Visible")
	c.UpdateDirHash(dir)
	assert.False(t, c.DirChanged(dir))

	// Add a hidden file — should NOT trigger change
	writeTestFile(t, dir, ".hidden", "secret")
	assert.False(t, c.DirChanged(dir))
}

func TestCacheStoreAndLookupDirSearch(t *testing.T) {
	c := newTestCache(t)
	dir := t.TempDir()
	writeTestFile(t, dir, "guide.md", "# Guide\n\nNeedle in docs.\n")

	results := &mq.SearchResults{
		Query: "needle",
		Matches: []*mq.SearchResult{
			{
				File:    filepath.Join(dir, "guide.md"),
				Section: "Guide",
				Lines:   "1-3",
				Match:   "Needle in docs.",
				Text:    "# Guide\n\nNeedle in docs.\n",
				Raw:     "# Guide\n\nNeedle in docs.\n",
			},
		},
	}

	_, dirHash, ok := c.LookupDirSearch(dir, "needle")
	assert.False(t, ok)
	require.NotEmpty(t, dirHash)
	require.NoError(t, c.StoreDirSearch(dir, "needle", dirHash, results))

	cached, _, ok := c.LookupDirSearch(dir, "NEEDLE")
	require.True(t, ok)
	require.NotNil(t, cached)
	require.Len(t, cached.Matches, 1)
	assert.Equal(t, "NEEDLE", cached.Query)
	assert.Equal(t, results.Matches[0].File, cached.Matches[0].File)
	assert.Equal(t, results.Matches[0].Match, cached.Matches[0].Match)
	assert.Equal(t, results.Matches[0].Text, cached.Matches[0].Text)
	assert.Equal(t, results.Matches[0].Raw, cached.Matches[0].Raw)
}

func TestCacheDirSearchInvalidatedOnDirChange(t *testing.T) {
	c := newTestCache(t)
	dir := t.TempDir()
	writeTestFile(t, dir, "guide.md", "# Guide\n\nNeedle in docs.\n")

	results := &mq.SearchResults{
		Query: "needle",
		Matches: []*mq.SearchResult{
			{
				File:    filepath.Join(dir, "guide.md"),
				Section: "Guide",
				Lines:   "1-3",
				Match:   "Needle in docs.",
			},
		},
	}

	_, dirHash, ok := c.LookupDirSearch(dir, "needle")
	assert.False(t, ok)
	require.NoError(t, c.StoreDirSearch(dir, "needle", dirHash, results))

	time.Sleep(10 * time.Millisecond)
	writeTestFile(t, dir, "added.md", "# Added\n\nFresh content.\n")

	cached, _, ok := c.LookupDirSearch(dir, "needle")
	assert.False(t, ok)
	assert.Nil(t, cached)
}

func TestCacheStoreAndLookupFileSearch(t *testing.T) {
	c := newTestCache(t)
	dir := t.TempDir()
	path := writeTestFile(t, dir, "events.jsonl", "{\"message\":\"needle found\"}\n")
	info, err := os.Stat(path)
	require.NoError(t, err)

	results := &mq.SearchResults{
		Query: "needle",
		Matches: []*mq.SearchResult{
			{
				File:    path,
				Section: "message",
				Lines:   "1",
				Match:   "needle found",
				Text:    "message: needle found",
				Raw:     "{\"message\":\"needle found\"}",
			},
		},
	}

	assert.Nil(t, c.LookupFileSearch(path, "needle", info))
	require.NoError(t, c.StoreFileSearch(path, "needle", info, results))

	cached := c.LookupFileSearch(path, "NEEDLE", info)
	require.NotNil(t, cached)
	require.Len(t, cached.Matches, 1)
	assert.Equal(t, "NEEDLE", cached.Query)
	assert.Equal(t, path, cached.Matches[0].File)
	assert.Equal(t, "needle found", cached.Matches[0].Match)
	assert.Equal(t, "message: needle found", cached.Matches[0].Text)
	assert.Equal(t, "{\"message\":\"needle found\"}", cached.Matches[0].Raw)
}

func TestCacheFileSearchInvalidatedOnFileChange(t *testing.T) {
	c := newTestCache(t)
	dir := t.TempDir()
	path := writeTestFile(t, dir, "events.jsonl", "{\"message\":\"needle found\"}\n")
	info, err := os.Stat(path)
	require.NoError(t, err)

	results := &mq.SearchResults{
		Query: "needle",
		Matches: []*mq.SearchResult{
			{
				File:    path,
				Section: "message",
				Lines:   "1",
				Match:   "needle found",
			},
		},
	}

	require.NoError(t, c.StoreFileSearch(path, "needle", info, results))
	require.NotNil(t, c.LookupFileSearch(path, "needle", info))

	time.Sleep(10 * time.Millisecond)
	require.NoError(t, os.WriteFile(path, []byte("{\"message\":\"updated\"}\n"), 0o644))
	updatedInfo, err := os.Stat(path)
	require.NoError(t, err)

	assert.Nil(t, c.LookupFileSearch(path, "needle", updatedInfo))
}

// -------------------------------------------------------------------
// Eviction
// -------------------------------------------------------------------

func TestTrimRemovesStaleEntries(t *testing.T) {
	c := newTestCache(t)
	testDir := t.TempDir()

	path := writeTestFile(t, testDir, "doc.md", "# Hello")
	content, _ := os.ReadFile(path)
	doc := makeTestDocument(path, content)
	storeTestFile(t, c, path, doc)

	files, _, docs := c.Stats()
	assert.Equal(t, 1, files)
	assert.Equal(t, 1, docs)

	// Trim with current time — should NOT remove (entries are fresh)
	require.NoError(t, c.Trim())
	files, _, docs = c.Stats()
	assert.Equal(t, 1, files)
	assert.Equal(t, 1, docs)
}

func TestClear(t *testing.T) {
	c := newTestCache(t)
	testDir := t.TempDir()

	path := writeTestFile(t, testDir, "doc.md", "# Hello")
	content, _ := os.ReadFile(path)
	doc := makeTestDocument(path, content)
	storeTestFile(t, c, path, doc)
	c.UpdateDirHash(testDir)

	files, dirs, docs := c.Stats()
	assert.Equal(t, 1, files)
	assert.Equal(t, 1, dirs)
	assert.Equal(t, 1, docs)

	require.NoError(t, c.Clear())

	files, dirs, docs = c.Stats()
	assert.Equal(t, 0, files)
	assert.Equal(t, 0, dirs)
	assert.Equal(t, 0, docs)
}

// -------------------------------------------------------------------
// ContentHash
// -------------------------------------------------------------------

func TestContentHash(t *testing.T) {
	h1 := mq.ContentHash([]byte("hello"))
	h2 := mq.ContentHash([]byte("hello"))
	h3 := mq.ContentHash([]byte("world"))

	assert.Equal(t, h1, h2, "same content should produce same hash")
	assert.NotEqual(t, h1, h3, "different content should produce different hash")
	assert.Len(t, h1, 64, "SHA256 hex should be 64 chars")
}

// -------------------------------------------------------------------
// Concurrent access
// -------------------------------------------------------------------

func TestCacheConcurrentAccess(t *testing.T) {
	c := newTestCache(t)
	testDir := t.TempDir()

	// Create several files
	for i := 0; i < 10; i++ {
		name := filepath.Join(testDir, filepath.Base(t.TempDir())+".md")
		content := []byte("# File " + name)
		os.WriteFile(name, content, 0644)
	}

	// Store files concurrently
	entries, _ := os.ReadDir(testDir)
	done := make(chan bool, len(entries))
	for _, entry := range entries {
		go func(name string) {
			path := filepath.Join(testDir, name)
			content, _ := os.ReadFile(path)
			doc := makeTestDocument(path, content)
			storeTestFile(t, c, path, doc)
			done <- true
		}(entry.Name())
	}

	for range entries {
		<-done
	}

	files, _, _ := c.Stats()
	assert.Equal(t, len(entries), files)
}

// -------------------------------------------------------------------
// Edge cases
// -------------------------------------------------------------------

func TestCacheNonexistentDir(t *testing.T) {
	c := newTestCache(t)
	assert.True(t, c.DirChanged("/nonexistent/path"))
}

func TestCacheEmptyDocument(t *testing.T) {
	c := newTestCache(t)
	testDir := t.TempDir()

	path := writeTestFile(t, testDir, "empty.md", "")
	content, _ := os.ReadFile(path)

	// Minimal document with no headings/sections
	doc := mq.NewDocument(
		content, path, mq.FormatMarkdown, "",
		nil, nil, nil, nil, nil, nil, nil, "",
	)

	storeTestFile(t, c, path, doc)
	cached := c.LookupFile(path)
	require.NotNil(t, cached)
	assert.Equal(t, 0, len(cached.GetHeadings()))
	assert.Equal(t, 0, len(cached.GetSections()))
}

// -------------------------------------------------------------------
// Real-file roundtrip tests (parse actual testdata files)
// -------------------------------------------------------------------

func parseRealMarkdown(t *testing.T, path string) (*mq.Document, []byte) {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	p := mq.NewParser()
	doc, err := p.Parse(content, path)
	require.NoError(t, err)
	return doc, content
}

func TestCacheRoundtripSectionText(t *testing.T) {
	c := newTestCache(t)
	path := "../testdata/code-heavy.md"

	doc, _ := parseRealMarkdown(t, path)
	storeTestFile(t, c, path, doc)

	cached := c.LookupFile(path)
	require.NotNil(t, cached)

	origSections := doc.GetSections()
	cachedSections := cached.GetSections()
	require.Equal(t, len(origSections), len(cachedSections))

	for i, orig := range origSections {
		got := cachedSections[i].GetText()
		want := orig.GetText()
		assert.Equal(t, want, got, "section %q text mismatch after roundtrip", orig.Heading.Text)
	}
}

func TestCacheRoundtripLists(t *testing.T) {
	c := newTestCache(t)
	path := "../testdata/tables-lists.md"

	doc, _ := parseRealMarkdown(t, path)
	storeTestFile(t, c, path, doc)

	cached := c.LookupFile(path)
	require.NotNil(t, cached)

	origLists := doc.GetLists(nil)
	cachedLists := cached.GetLists(nil)
	require.Equal(t, len(origLists), len(cachedLists), "list count mismatch")

	for i, orig := range origLists {
		got := cachedLists[i]
		assert.Equal(t, orig.Ordered, got.Ordered, "list %d Ordered mismatch", i)
		assert.Equal(t, len(orig.Items), len(got.Items), "list %d item count mismatch", i)
		for j, origItem := range orig.Items {
			assert.Equal(t, origItem.Text, got.Items[j].Text, "list %d item %d text mismatch", i, j)
			// Verify checked state for task lists
			if origItem.Checked != nil {
				require.NotNil(t, got.Items[j].Checked, "list %d item %d checked should not be nil", i, j)
				assert.Equal(t, *origItem.Checked, *got.Items[j].Checked, "list %d item %d checked mismatch", i, j)
			}
		}
	}
}

func TestCacheRoundtripMetadata(t *testing.T) {
	c := newTestCache(t)
	path := "../testdata/frontmatter.md"

	doc, _ := parseRealMarkdown(t, path)
	storeTestFile(t, c, path, doc)

	cached := c.LookupFile(path)
	require.NotNil(t, cached)

	origMeta := doc.Metadata()
	cachedMeta := cached.Metadata()
	require.NotNil(t, cachedMeta, "metadata should survive cache roundtrip")
	assert.Equal(t, origMeta["owner"], cachedMeta["owner"])
	assert.Equal(t, origMeta["priority"], cachedMeta["priority"])
	assert.Equal(t, origMeta["title"], cachedMeta["title"])

	// Verify via typed accessors
	owner, ok := cached.GetOwner()
	assert.True(t, ok)
	assert.Equal(t, "platform-team", owner)

	priority, ok := cached.GetPriority()
	assert.True(t, ok)
	assert.Equal(t, "high", priority)
}

func TestCacheRoundtripSectionCodeBlocks(t *testing.T) {
	c := newTestCache(t)
	path := "../testdata/code-heavy.md"

	doc, _ := parseRealMarkdown(t, path)
	storeTestFile(t, c, path, doc)

	cached := c.LookupFile(path)
	require.NotNil(t, cached)

	// Find "Python Examples" section — should have code blocks
	origSection, ok := doc.GetSection("Python Examples")
	require.True(t, ok, "original doc should have 'Python Examples' section")
	origCBs := origSection.GetCodeBlocks()
	require.NotEmpty(t, origCBs, "original section should have code blocks")

	cachedSection, ok := cached.GetSection("Python Examples")
	require.True(t, ok, "cached doc should have 'Python Examples' section")
	cachedCBs := cachedSection.GetCodeBlocks()
	require.Equal(t, len(origCBs), len(cachedCBs), "section code block count mismatch")

	for i, orig := range origCBs {
		assert.Equal(t, orig.Language, cachedCBs[i].Language, "codeblock %d language mismatch", i)
		assert.Equal(t, orig.Content, cachedCBs[i].Content, "codeblock %d content mismatch", i)
	}
}

func TestCacheRoundtripFullStructure(t *testing.T) {
	c := newTestCache(t)
	path := "../testdata/code-heavy.md"

	doc, _ := parseRealMarkdown(t, path)
	storeTestFile(t, c, path, doc)

	cached := c.LookupFile(path)
	require.NotNil(t, cached)

	// Verify all top-level counts match
	assert.Equal(t, doc.Title(), cached.Title())
	assert.Equal(t, doc.Format(), cached.Format())
	assert.Equal(t, doc.ReadableText(), cached.ReadableText())
	assert.Equal(t, len(doc.GetHeadings()), len(cached.GetHeadings()))
	assert.Equal(t, len(doc.GetSections()), len(cached.GetSections()))
	assert.Equal(t, len(doc.GetCodeBlocks()), len(cached.GetCodeBlocks()))
	assert.Equal(t, len(doc.GetLinks()), len(cached.GetLinks()))
	assert.Equal(t, len(doc.GetImages()), len(cached.GetImages()))
	assert.Equal(t, len(doc.GetTables()), len(cached.GetTables()))
	assert.Equal(t, len(doc.GetLists(nil)), len(cached.GetLists(nil)))

	// Verify code blocks content
	origCBs := doc.GetCodeBlocks()
	cachedCBs := cached.GetCodeBlocks()
	for i := range origCBs {
		assert.Equal(t, origCBs[i].Language, cachedCBs[i].Language)
		assert.Equal(t, origCBs[i].Content, cachedCBs[i].Content)
	}
}

func TestReadFileWithStatError(t *testing.T) {
	// ReadFileWithStat should error for nonexistent paths
	_, _, err := mq.ReadFileWithStat("/nonexistent/file.md")
	assert.Error(t, err)
}

func TestCacheSearchRoundtrip(t *testing.T) {
	// Regression: search through a cached markdown document must produce
	// the same results as search through a freshly parsed document.
	c := newTestCache(t)
	testDir := t.TempDir()

	mdContent := "# Deployment Guide\n\n## Rolling Deployment\n\nA rolling deployment gradually replaces old pods.\n\n## Monitoring\n\nCheck metrics after each deployment completes.\n"
	path := writeTestFile(t, testDir, "guide.md", mdContent)
	content, err := os.ReadFile(path)
	require.NoError(t, err)

	// Parse fresh
	parser := mq.NewParser()
	freshDoc, err := parser.Parse(content, path)
	require.NoError(t, err)

	freshResults := freshDoc.Search("deployment")
	require.NotEmpty(t, freshResults.Matches, "fresh parse should find 'deployment'")

	// Store in cache, then look up
	storeTestFile(t, c, path, freshDoc)
	cachedDoc := c.LookupFile(path)
	require.NotNil(t, cachedDoc)

	cachedResults := cachedDoc.Search("deployment")
	require.Equal(t, len(freshResults.Matches), len(cachedResults.Matches),
		"cached doc search should return same number of matches as fresh parse")

	for i, fresh := range freshResults.Matches {
		cached := cachedResults.Matches[i]
		assert.Equal(t, fresh.Section, cached.Section, "match %d section mismatch", i)
		assert.Equal(t, fresh.Lines, cached.Lines, "match %d lines mismatch", i)
		assert.Equal(t, fresh.Match, cached.Match, "match %d snippet mismatch", i)
	}
}

// -------------------------------------------------------------------
// TOCTOU regression tests
// -------------------------------------------------------------------

func TestReadFileWithStat(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		dir := t.TempDir()
		content := "# Title\n\nBody with **markdown**.\n"
		path := writeTestFile(t, dir, "file.md", content)

		bytes, info, err := mq.ReadFileWithStat(path)
		require.NoError(t, err)
		assert.Equal(t, []byte(content), bytes)
		assert.Equal(t, int64(len(content)), info.Size())

		stat, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, stat.Size(), info.Size())
		assert.True(t, info.ModTime().Equal(stat.ModTime()), "modtime should match os.Stat")
	})

	t.Run("nonexistent", func(t *testing.T) {
		_, _, err := mq.ReadFileWithStat(filepath.Join(t.TempDir(), "missing.md"))
		assert.Error(t, err)
	})

	t.Run("directory", func(t *testing.T) {
		dir := t.TempDir()
		_, _, err := mq.ReadFileWithStat(dir)
		assert.Error(t, err)
	})
}

func TestReadFileWithStatConsistency(t *testing.T) {
	content := "consistency check\nsecond line\n"
	path := writeTestFile(t, t.TempDir(), "consistent.md", content)

	bytes, info, err := mq.ReadFileWithStat(path)
	require.NoError(t, err)

	assert.Equal(t, len(bytes), int(info.Size()))

	stat, err := os.Stat(path)
	require.NoError(t, err)
	assert.True(t, info.ModTime().Equal(stat.ModTime()), "stat and info modtimes should match")
}

func TestLookupFileUsesAtomicRead(t *testing.T) {
	c := newTestCache(t)
	testDir := t.TempDir()

	mdContent := "# Title\n\n## Atomic Section\nContent with **bold** emphasis.\n"
	path := writeTestFile(t, testDir, "atomic.md", mdContent)

	doc, _ := parseRealMarkdown(t, path)
	storeTestFile(t, c, path, doc)

	cached := c.LookupFile(path)
	require.NotNil(t, cached)

	section, ok := cached.GetSection("Atomic Section")
	require.True(t, ok, "cached document should contain section")
	text := section.GetText()
	assert.Contains(t, text, "**bold**", "section text should use raw file content with markdown markers")
	assert.Equal(t, "## Atomic Section\nContent with **bold** emphasis.\n", text)

	actual, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(actual), text, "cached section text must come from the file on disk")
}

func TestStoreFileUsesProvidedInfo(t *testing.T) {
	c := newTestCache(t)
	path := writeTestFile(t, t.TempDir(), "store.md", "# Version 1\n")

	content, info, err := mq.ReadFileWithStat(path)
	require.NoError(t, err)

	parser := mq.NewParser()
	doc, err := parser.Parse(content, path)
	require.NoError(t, err)

	require.NoError(t, c.StoreFile(path, content, info, doc))
	require.NotNil(t, c.LookupFile(path), "cache should hit immediately after store")

	// Modify the file to change mtime and content
	time.Sleep(15 * time.Millisecond)
	require.NoError(t, os.WriteFile(path, []byte("# Version 2\nchanged"), 0644))

	assert.Nil(t, c.LookupFile(path), "cache should miss after file modification")
}

// TestCacheRoundtripNonMarkdownSectionText verifies that cached non-markdown
// documents (PDF, HTML) preserve readable section text after cache roundtrip,
// even though the on-disk file is binary.
func TestCacheRoundtripNonMarkdownSectionText(t *testing.T) {
	c := newTestCache(t)
	dir := t.TempDir()

	readableText := "# Introduction\nWelcome to the guide.\n\n## Setup\nInstall the dependencies.\n\n## Usage\nRun the command."
	textBytes := []byte(readableText)

	// Write a fake "PDF" file with binary content
	binaryContent := make([]byte, 2000)
	for i := range binaryContent {
		if i%8 == 0 {
			binaryContent[i] = '\n'
		} else {
			binaryContent[i] = byte(i % 256)
		}
	}
	path := filepath.Join(dir, "doc.pdf")
	require.NoError(t, os.WriteFile(path, binaryContent, 0644))

	headings := []*mq.Heading{
		{Level: 1, Text: "Introduction", Line: 1, Page: 1},
		{Level: 2, Text: "Setup", Line: 4, Page: 1},
		{Level: 2, Text: "Usage", Line: 7, Page: 2},
	}
	sections := []*mq.Section{
		mq.NewSectionWithSource(headings[0], 1, 8, textBytes),
		mq.NewSectionWithSource(headings[1], 4, 6, textBytes),
		mq.NewSectionWithSource(headings[2], 7, 8, textBytes),
	}
	sections[1].Parent = sections[0]
	sections[2].Parent = sections[0]
	sections[0].Children = []*mq.Section{sections[1], sections[2]}

	doc := mq.NewDocument(
		binaryContent, path, mq.FormatPDF, "Introduction",
		headings, sections, nil, nil, nil, nil, nil,
		readableText,
	)
	doc.SetPageCount(4)

	// Verify sections have readable text before caching
	setupSection, ok := doc.GetSection("Setup")
	require.True(t, ok)
	origText := setupSection.GetText()
	assert.Contains(t, origText, "Install the dependencies")

	// Store and retrieve from cache
	storeTestFile(t, c, path, doc)
	cached := c.LookupFile(path)
	require.NotNil(t, cached, "cache should return document")

	// Verify section text survives roundtrip with readable content
	cachedSetup, ok := cached.GetSection("Setup")
	require.True(t, ok)
	cachedText := cachedSetup.GetText()
	assert.Contains(t, cachedText, "Install the dependencies",
		"cached PDF section should have readable text, not binary garbage")
	assert.Equal(t, origText, cachedText,
		"cached section text should match original")

	// Verify tree uses page numbers and readable text line count
	tree := cached.BuildTree(mq.DefaultFileTreeOptions())
	assert.Equal(t, 4, tree.Pages,
		"cached PDF page count should survive roundtrip")
	output := tree.String()
	assert.Contains(t, output, "(4 pages)",
		"cached PDF tree header should show pages, not lines")
	assert.Contains(t, output, "(p. 1)",
		"cached PDF tree sections should show page numbers")
}
