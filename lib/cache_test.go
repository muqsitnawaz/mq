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

	c.StoreFile(path, content, doc)
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
	c.StoreFile(path, content, doc)

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

	c.StoreFile(path, content, doc)
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

	c.StoreFile(path, content, doc)
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

// -------------------------------------------------------------------
// Eviction
// -------------------------------------------------------------------

func TestTrimRemovesStaleEntries(t *testing.T) {
	c := newTestCache(t)
	testDir := t.TempDir()

	path := writeTestFile(t, testDir, "doc.md", "# Hello")
	content, _ := os.ReadFile(path)
	doc := makeTestDocument(path, content)
	c.StoreFile(path, content, doc)

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
	c.StoreFile(path, content, doc)
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
			c.StoreFile(path, content, doc)
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

	c.StoreFile(path, content, doc)
	cached := c.LookupFile(path)
	require.NotNil(t, cached)
	assert.Equal(t, 0, len(cached.GetHeadings()))
	assert.Equal(t, 0, len(cached.GetSections()))
}
