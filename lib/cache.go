package mq

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	bolt "go.etcd.io/bbolt"
)

// Cache bucket names.
var (
	bucketMeta       = []byte("meta")
	bucketFiles      = []byte("files")
	bucketDirs       = []byte("dirs")
	bucketDocuments  = []byte("documents")
	bucketDirSearch  = []byte("dir_search")
	bucketFileSearch = []byte("file_search")
)

const (
	cacheSchemaVersion = "1"
	trimMaxAge         = 5 * 24 * time.Hour // Evict entries unused for 5 days
)

// Cache provides content-addressed caching for parsed documents.
// It uses bbolt for storage and a Merkle tree over directories
// so that unchanged subtrees can be skipped entirely.
type Cache struct {
	db   *bolt.DB
	path string
}

// fileMeta is stored per file path in the "files" bucket.
type fileMeta struct {
	Mtime       int64  `msgpack:"m"`
	Size        int64  `msgpack:"s"`
	ContentHash string `msgpack:"h"`
	LastUsed    int64  `msgpack:"u"` // Unix timestamp for eviction
}

// dirMeta is stored per directory path in the "dirs" bucket.
type dirMeta struct {
	Mtime      int64  `msgpack:"m"`
	MerkleHash string `msgpack:"h"` // Hash of sorted children hashes
	LastUsed   int64  `msgpack:"u"`
}

// cachedDocument is the serializable form of a Document.
type cachedDocument struct {
	Format       Format                 `msgpack:"f"`
	Title        string                 `msgpack:"t"`
	ReadableText string                 `msgpack:"r"`
	PageCount    int                    `msgpack:"pc"`
	Headings     []cachedHeading        `msgpack:"h"`
	Sections     []cachedSection        `msgpack:"s"`
	CodeBlocks   []cachedCode           `msgpack:"c"`
	Links        []cachedLink           `msgpack:"l"`
	Images       []cachedImage          `msgpack:"i"`
	Tables       []cachedTable          `msgpack:"tb"`
	Lists        []cachedList           `msgpack:"ls"`
	Metadata     map[string]interface{} `msgpack:"md"`
}

type cachedSearchResult struct {
	File    string   `msgpack:"f"`
	Section string   `msgpack:"s"`
	Lines   string   `msgpack:"l"`
	Match   string   `msgpack:"m"`
	Fields  []string `msgpack:"fs"`
	Text    string   `msgpack:"t"`
}

type cachedDirSearch struct {
	DirHash  string               `msgpack:"h"`
	Results  []cachedSearchResult `msgpack:"r"`
	LastUsed int64                `msgpack:"u"`
}

type cachedFileSearch struct {
	Mtime    int64                `msgpack:"m"`
	Size     int64                `msgpack:"s"`
	Results  []cachedSearchResult `msgpack:"r"`
	LastUsed int64                `msgpack:"u"`
}

type cachedHeading struct {
	Level int    `msgpack:"l"`
	Text  string `msgpack:"t"`
	ID    string `msgpack:"i"`
	Line  int    `msgpack:"n"`
	Page  int    `msgpack:"pg"`
}

type cachedSection struct {
	HeadingIdx    int   `msgpack:"h"` // Index into Headings
	Start         int   `msgpack:"s"`
	End           int   `msgpack:"e"`
	ParentIdx     int   `msgpack:"p"` // -1 if no parent
	ChildIdxs     []int `msgpack:"c"`
	CodeBlockIdxs []int `msgpack:"cb"` // Indices into document-level CodeBlocks
}

type cachedCode struct {
	Language string `msgpack:"l"`
	Content  string `msgpack:"c"`
	Lines    int    `msgpack:"n"`
}

type cachedLink struct {
	Text string `msgpack:"t"`
	URL  string `msgpack:"u"`
}

type cachedImage struct {
	AltText string `msgpack:"a"`
	URL     string `msgpack:"u"`
	Title   string `msgpack:"t"`
}

type cachedTable struct {
	Headers []string   `msgpack:"h"`
	Rows    [][]string `msgpack:"r"`
}

type cachedList struct {
	Ordered bool             `msgpack:"o"`
	Items   []cachedListItem `msgpack:"i"`
}

type cachedListItem struct {
	Text     string           `msgpack:"t"`
	Checked  *bool            `msgpack:"c"`
	Children []cachedListItem `msgpack:"ch"`
}

// DefaultCachePath returns the default cache database path.
func DefaultCachePath() string {
	if dir, err := os.UserCacheDir(); err == nil {
		return filepath.Join(dir, "mq", "cache.db")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".cache", "mq", "cache.db")
	}
	return ""
}

// OpenCache opens or creates the cache database.
func OpenCache(path string) (*Cache, error) {
	if path == "" {
		path = DefaultCachePath()
	}
	if path == "" {
		return nil, fmt.Errorf("cannot determine cache path")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open cache db: %w", err)
	}

	c := &Cache{db: db, path: path}

	// Initialize buckets and check schema version
	if err := c.init(); err != nil {
		db.Close()
		return nil, err
	}

	return c, nil
}

func (c *Cache) init() error {
	return c.db.Update(func(tx *bolt.Tx) error {
		for _, name := range [][]byte{bucketMeta, bucketFiles, bucketDirs, bucketDocuments, bucketDirSearch, bucketFileSearch} {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return err
			}
		}

		// Check schema version — wipe cache if mismatched
		meta := tx.Bucket(bucketMeta)
		v := meta.Get([]byte("version"))
		if v != nil && string(v) != cacheSchemaVersion {
			// Schema changed — clear all data buckets
			for _, name := range [][]byte{bucketFiles, bucketDirs, bucketDocuments, bucketDirSearch, bucketFileSearch} {
				b := tx.Bucket(name)
				c := b.Cursor()
				for k, _ := c.First(); k != nil; k, _ = c.Next() {
					b.Delete(k)
				}
			}
		}
		return meta.Put([]byte("version"), []byte(cacheSchemaVersion))
	})
}

// Close closes the cache database.
func (c *Cache) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// Path returns the cache database file path.
func (c *Cache) Path() string {
	return c.path
}

// -------------------------------------------------------------------
// File-level caching
// -------------------------------------------------------------------

// LookupFile checks if a file's parsed Document is cached.
// It uses a two-level check: stat (mtime+size) then content hash.
// Returns the cached Document or nil if not cached / stale.
func (c *Cache) LookupFile(path string) *Document {
	// Read file + stat from the same fd to avoid TOCTOU races.
	source, info, err := ReadFileWithStat(path)
	if err != nil {
		return nil
	}

	return c.LookupFileWithContent(path, source, info)
}

// LookupFileWithContent checks if a file's parsed Document is cached using bytes
// already read by the caller.
func (c *Cache) LookupFileWithContent(path string, content []byte, info os.FileInfo) *Document {
	if info == nil {
		return nil
	}

	pathKey := pathHash(path)

	var contentHash string
	statMatched := false
	c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketFiles)
		data := b.Get(pathKey)
		if data == nil {
			return nil
		}
		var fm fileMeta
		if err := msgpack.Unmarshal(data, &fm); err != nil {
			return nil
		}
		// Fast path: if mtime and size match, trust the cached content hash
		if fm.Mtime == info.ModTime().UnixNano() && fm.Size == info.Size() {
			contentHash = fm.ContentHash
			statMatched = true
		}
		return nil
	})

	if contentHash == "" {
		contentHash = ContentHash(content)
	}

	doc := c.lookupDocument(contentHash, content, path)
	if doc != nil && !statMatched {
		_ = c.storeFileMeta(path, contentHash, info)
	}
	return doc
}

// StoreFile caches a parsed Document for a file.
// info must be from the same read as content to avoid TOCTOU races.
func (c *Cache) StoreFile(path string, content []byte, info os.FileInfo, doc *Document) error {
	contentHash := ContentHash(content)

	// Store document
	cd := documentToCache(doc)
	cdData, err := msgpack.Marshal(&cd)
	if err != nil {
		return fmt.Errorf("marshal document: %w", err)
	}

	return c.db.Update(func(tx *bolt.Tx) error {
		if err := c.putFileMeta(tx, path, contentHash, info, time.Now().Unix()); err != nil {
			return err
		}
		return tx.Bucket(bucketDocuments).Put([]byte(contentHash), cdData)
	})
}

func (c *Cache) lookupDocument(contentHash string, source []byte, filePath string) *Document {
	var doc *Document
	c.db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket(bucketDocuments).Get([]byte(contentHash))
		if data == nil {
			return nil
		}
		var cd cachedDocument
		if err := msgpack.Unmarshal(data, &cd); err != nil {
			return nil
		}
		doc = cacheToDocument(&cd, source, filePath)
		return nil
	})

	// Touch the file entry's LastUsed
	if doc != nil {
		pathKey := pathHash(filePath)
		c.db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket(bucketFiles)
			data := b.Get(pathKey)
			if data == nil {
				return nil
			}
			var fm fileMeta
			if err := msgpack.Unmarshal(data, &fm); err != nil {
				return nil
			}
			fm.LastUsed = time.Now().Unix()
			updated, _ := msgpack.Marshal(&fm)
			return b.Put(pathKey, updated)
		})
	}

	return doc
}

func (c *Cache) storeFileMeta(path string, contentHash string, info os.FileInfo) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		return c.putFileMeta(tx, path, contentHash, info, time.Now().Unix())
	})
}

func (c *Cache) putFileMeta(tx *bolt.Tx, path string, contentHash string, info os.FileInfo, lastUsed int64) error {
	fm := fileMeta{
		Mtime:       info.ModTime().UnixNano(),
		Size:        info.Size(),
		ContentHash: contentHash,
		LastUsed:    lastUsed,
	}
	fmData, err := msgpack.Marshal(&fm)
	if err != nil {
		return fmt.Errorf("marshal file meta: %w", err)
	}
	return tx.Bucket(bucketFiles).Put(pathHash(path), fmData)
}

// LookupDirSearch returns cached directory search results if the directory hash
// still matches the current on-disk tree.
func (c *Cache) LookupDirSearch(dirPath string, query string) (*SearchResults, string, bool) {
	currentHash := c.computeDirHash(dirPath)
	if currentHash == "" {
		return nil, "", false
	}

	key := searchKey(dirPath, query)
	var results *SearchResults
	c.db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket(bucketDirSearch).Get(key)
		if data == nil {
			return nil
		}
		var cached cachedDirSearch
		if err := msgpack.Unmarshal(data, &cached); err != nil {
			return nil
		}
		if cached.DirHash != currentHash {
			return nil
		}
		results = cacheToSearchResults(query, cached.Results)
		return nil
	})

	if results != nil {
		c.touchDirSearch(key)
		return results, currentHash, true
	}
	return nil, currentHash, false
}

// StoreDirSearch stores search results for a directory hash and query.
func (c *Cache) StoreDirSearch(dirPath string, query string, dirHash string, results *SearchResults) error {
	if dirHash == "" {
		dirHash = c.computeDirHash(dirPath)
	}
	if dirHash == "" {
		return nil
	}

	entry := cachedDirSearch{
		DirHash:  dirHash,
		Results:  searchResultsToCache(results),
		LastUsed: time.Now().Unix(),
	}
	data, err := msgpack.Marshal(&entry)
	if err != nil {
		return fmt.Errorf("marshal dir search: %w", err)
	}

	if err := c.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketDirSearch).Put(searchKey(dirPath, query), data)
	}); err != nil {
		return err
	}

	return c.storeDirHash(dirPath, dirHash)
}

// LookupFileSearch returns cached per-file search results if the file stat still matches.
func (c *Cache) LookupFileSearch(path string, query string, info os.FileInfo) *SearchResults {
	if info == nil {
		return nil
	}

	key := searchKey(path, query)
	var results *SearchResults
	c.db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket(bucketFileSearch).Get(key)
		if data == nil {
			return nil
		}
		var cached cachedFileSearch
		if err := msgpack.Unmarshal(data, &cached); err != nil {
			return nil
		}
		if cached.Mtime != info.ModTime().UnixNano() || cached.Size != info.Size() {
			return nil
		}
		results = cacheToSearchResults(query, cached.Results)
		return nil
	})

	if results != nil {
		c.touchFileSearch(key)
	}
	return results
}

// StoreFileSearch stores per-file search results keyed by path and query.
func (c *Cache) StoreFileSearch(path string, query string, info os.FileInfo, results *SearchResults) error {
	if info == nil {
		return nil
	}

	entry := cachedFileSearch{
		Mtime:    info.ModTime().UnixNano(),
		Size:     info.Size(),
		Results:  searchResultsToCache(results),
		LastUsed: time.Now().Unix(),
	}
	data, err := msgpack.Marshal(&entry)
	if err != nil {
		return fmt.Errorf("marshal file search: %w", err)
	}

	return c.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketFileSearch).Put(searchKey(path, query), data)
	})
}

func (c *Cache) touchDirSearch(key []byte) {
	c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketDirSearch)
		data := b.Get(key)
		if data == nil {
			return nil
		}
		var cached cachedDirSearch
		if err := msgpack.Unmarshal(data, &cached); err != nil {
			return nil
		}
		cached.LastUsed = time.Now().Unix()
		updated, _ := msgpack.Marshal(&cached)
		return b.Put(key, updated)
	})
}

func (c *Cache) touchFileSearch(key []byte) {
	c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketFileSearch)
		data := b.Get(key)
		if data == nil {
			return nil
		}
		var cached cachedFileSearch
		if err := msgpack.Unmarshal(data, &cached); err != nil {
			return nil
		}
		cached.LastUsed = time.Now().Unix()
		updated, _ := msgpack.Marshal(&cached)
		return b.Put(key, updated)
	})
}

func (c *Cache) storeDirHash(dirPath string, hash string) error {
	if hash == "" {
		return nil
	}
	dm := dirMeta{
		Mtime:      time.Now().UnixNano(),
		MerkleHash: hash,
		LastUsed:   time.Now().Unix(),
	}
	data, err := msgpack.Marshal(&dm)
	if err != nil {
		return fmt.Errorf("marshal dir meta: %w", err)
	}
	return c.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketDirs).Put(pathHash(dirPath), data)
	})
}

// -------------------------------------------------------------------
// Merkle directory tree
// -------------------------------------------------------------------

// DirChanged checks if any file in a directory subtree has changed.
// Returns true if anything changed (and the subtree should be reparsed).
// Uses a Merkle hash: each directory's hash = SHA256(sorted children hashes).
func (c *Cache) DirChanged(dirPath string) bool {
	currentHash := c.computeDirHash(dirPath)
	if currentHash == "" {
		return true // Can't compute hash, assume changed
	}

	pathKey := pathHash(dirPath)
	var changed bool = true

	c.db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket(bucketDirs).Get(pathKey)
		if data == nil {
			return nil
		}
		var dm dirMeta
		if err := msgpack.Unmarshal(data, &dm); err != nil {
			return nil
		}
		if dm.MerkleHash == currentHash {
			changed = false
		}
		return nil
	})

	return changed
}

// UpdateDirHash stores the current Merkle hash for a directory and all subdirectories.
func (c *Cache) UpdateDirHash(dirPath string) {
	c.computeAndStoreDirHash(dirPath)
}

// computeAndStoreDirHash computes the Merkle hash of a directory, stores it (and all
// subdirectory hashes) in the DB, and returns the hash. Each directory is visited
// exactly once — no double recursion.
func (c *Cache) computeAndStoreDirHash(dirPath string) string {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return ""
	}

	var parts []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue // Skip hidden files
		}

		childPath := filepath.Join(dirPath, name)
		if entry.IsDir() {
			childHash := c.computeAndStoreDirHash(childPath)
			if childHash != "" {
				parts = append(parts, name+":d:"+childHash)
			}
		} else {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			statHash := fmt.Sprintf("%s:f:%d:%d", name, info.ModTime().UnixNano(), info.Size())
			parts = append(parts, statHash)
		}
	}

	sort.Strings(parts)
	h := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	hash := hex.EncodeToString(h[:])

	dm := dirMeta{
		Mtime:      time.Now().UnixNano(),
		MerkleHash: hash,
		LastUsed:   time.Now().Unix(),
	}
	data, _ := msgpack.Marshal(&dm)

	c.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketDirs).Put(pathHash(dirPath), data)
	})

	return hash
}

// computeDirHash computes the Merkle hash of a directory (read-only, no DB writes).
// Used by DirChanged to check current state against stored hash.
func (c *Cache) computeDirHash(dirPath string) string {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return ""
	}

	var parts []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue // Skip hidden files
		}

		childPath := filepath.Join(dirPath, name)
		if entry.IsDir() {
			childHash := c.computeDirHash(childPath)
			if childHash != "" {
				parts = append(parts, name+":d:"+childHash)
			}
		} else {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			statHash := fmt.Sprintf("%s:f:%d:%d", name, info.ModTime().UnixNano(), info.Size())
			parts = append(parts, statHash)
		}
	}

	sort.Strings(parts)
	h := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(h[:])
}

// -------------------------------------------------------------------
// Eviction
// -------------------------------------------------------------------

// Trim removes cache entries unused for more than trimMaxAge.
func (c *Cache) Trim() error {
	cutoff := time.Now().Add(-trimMaxAge).Unix()

	return c.db.Update(func(tx *bolt.Tx) error {
		// Collect stale content hashes to potentially remove from documents bucket
		staleContentHashes := make(map[string]bool)

		// Trim files bucket
		fb := tx.Bucket(bucketFiles)
		fc := fb.Cursor()
		for k, v := fc.First(); k != nil; k, v = fc.Next() {
			var fm fileMeta
			if err := msgpack.Unmarshal(v, &fm); err != nil {
				fc.Delete()
				continue
			}
			if fm.LastUsed < cutoff {
				staleContentHashes[fm.ContentHash] = true
				fc.Delete()
			}
		}

		// Trim dirs bucket
		db := tx.Bucket(bucketDirs)
		dc := db.Cursor()
		for k, v := dc.First(); k != nil; k, v = dc.Next() {
			var dm dirMeta
			if err := msgpack.Unmarshal(v, &dm); err != nil {
				dc.Delete()
				continue
			}
			if dm.LastUsed < cutoff {
				dc.Delete()
			}
		}

		// Trim directory search results
		dirSearch := tx.Bucket(bucketDirSearch)
		dirCursor := dirSearch.Cursor()
		for k, v := dirCursor.First(); k != nil; k, v = dirCursor.Next() {
			var cached cachedDirSearch
			if err := msgpack.Unmarshal(v, &cached); err != nil {
				dirCursor.Delete()
				continue
			}
			if cached.LastUsed < cutoff {
				dirCursor.Delete()
			}
		}

		// Trim file search results
		fileSearch := tx.Bucket(bucketFileSearch)
		fileCursor := fileSearch.Cursor()
		for k, v := fileCursor.First(); k != nil; k, v = fileCursor.Next() {
			var cached cachedFileSearch
			if err := msgpack.Unmarshal(v, &cached); err != nil {
				fileCursor.Delete()
				continue
			}
			if cached.LastUsed < cutoff {
				fileCursor.Delete()
			}
		}

		// Remove documents whose content hash is no longer referenced
		// by any file entry
		if len(staleContentHashes) > 0 {
			// Check if any remaining file still references these hashes
			fc2 := fb.Cursor()
			for k, v := fc2.First(); k != nil; k, v = fc2.Next() {
				var fm fileMeta
				if err := msgpack.Unmarshal(v, &fm); err == nil {
					delete(staleContentHashes, fm.ContentHash)
				}
			}
			// Delete truly orphaned documents
			docBucket := tx.Bucket(bucketDocuments)
			for hash := range staleContentHashes {
				docBucket.Delete([]byte(hash))
			}
		}

		return nil
	})
}

// Stats returns cache statistics.
func (c *Cache) Stats() (files, dirs, docs int) {
	c.db.View(func(tx *bolt.Tx) error {
		files = tx.Bucket(bucketFiles).Stats().KeyN
		dirs = tx.Bucket(bucketDirs).Stats().KeyN
		docs = tx.Bucket(bucketDocuments).Stats().KeyN
		return nil
	})
	return
}

// Clear removes all cached data.
func (c *Cache) Clear() error {
	return c.db.Update(func(tx *bolt.Tx) error {
		for _, name := range [][]byte{bucketFiles, bucketDirs, bucketDocuments, bucketDirSearch, bucketFileSearch} {
			b := tx.Bucket(name)
			c := b.Cursor()
			for k, _ := c.First(); k != nil; k, _ = c.Next() {
				b.Delete(k)
			}
		}
		return nil
	})
}

// -------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------

// ReadFileWithStat opens a file, stats from the same fd, and reads content.
// This avoids TOCTOU races where the file changes between a stat and a read.
func ReadFileWithStat(path string) ([]byte, os.FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}

	content, err := io.ReadAll(f)
	if err != nil {
		return nil, nil, err
	}

	return content, info, nil
}

// ContentHash returns the SHA256 hex digest of content.
func ContentHash(content []byte) string {
	h := sha256.Sum256(content)
	return hex.EncodeToString(h[:])
}

func pathHash(path string) []byte {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	h := sha256.Sum256([]byte(abs))
	return h[:]
}

func searchKey(path string, query string) []byte {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	h := sha256.Sum256([]byte(abs + "\x00" + normalizeSearchQuery(query)))
	return h[:]
}

func normalizeSearchQuery(query string) string {
	return strings.ToLower(query)
}

func searchResultsToCache(results *SearchResults) []cachedSearchResult {
	if results == nil {
		return nil
	}

	cached := make([]cachedSearchResult, 0, len(results.Matches))
	for _, match := range results.Matches {
		if match == nil {
			continue
		}
		cached = append(cached, cachedSearchResult{
			File:    match.File,
			Section: match.Section,
			Lines:   match.Lines,
			Match:   match.Match,
			Fields:  append([]string(nil), match.Fields...),
			Text:    match.Text,
		})
	}
	return cached
}

func cacheToSearchResults(query string, cached []cachedSearchResult) *SearchResults {
	results := &SearchResults{Query: query}
	for _, match := range cached {
		results.Matches = append(results.Matches, &SearchResult{
			File:    match.File,
			Section: match.Section,
			Lines:   match.Lines,
			Match:   match.Match,
			Fields:  append([]string(nil), match.Fields...),
			Text:    match.Text,
		})
	}
	return results
}

// -------------------------------------------------------------------
// Document <-> Cache serialization
// -------------------------------------------------------------------

func documentToCache(doc *Document) cachedDocument {
	cd := cachedDocument{
		Format:       doc.format,
		Title:        doc.title,
		ReadableText: doc.readableText,
		PageCount:    doc.pageCount,
		Metadata:     doc.metadata,
	}

	// Headings
	allHeadings := doc.GetHeadings()
	for _, h := range allHeadings {
		cd.Headings = append(cd.Headings, cachedHeading{
			Level: h.Level,
			Text:  h.Text,
			ID:    h.ID,
			Line:  h.Line,
			Page:  h.Page,
		})
	}

	// Build heading index for section references
	headingIdx := make(map[*Heading]int)
	for i, h := range allHeadings {
		headingIdx[h] = i
	}

	// Code blocks — build pointer-to-index map for section cross-references
	codeBlockIdx := make(map[*CodeBlock]int)
	for i, cb := range doc.codeBlocks {
		codeBlockIdx[cb] = i
		cd.CodeBlocks = append(cd.CodeBlocks, cachedCode{
			Language: cb.Language,
			Content:  cb.Content,
			Lines:    cb.Lines,
		})
	}

	// Sections
	sections := doc.GetSections()
	sectionIdx := make(map[*Section]int)
	for i, s := range sections {
		sectionIdx[s] = i
	}

	for _, s := range sections {
		cs := cachedSection{
			Start:     s.Start,
			End:       s.End,
			ParentIdx: -1,
		}
		if idx, ok := headingIdx[s.Heading]; ok {
			cs.HeadingIdx = idx
		}
		if s.Parent != nil {
			if idx, ok := sectionIdx[s.Parent]; ok {
				cs.ParentIdx = idx
			}
		}
		for _, child := range s.Children {
			if idx, ok := sectionIdx[child]; ok {
				cs.ChildIdxs = append(cs.ChildIdxs, idx)
			}
		}
		// Store codeblock indices for this section
		for _, cb := range s.codeBlocks {
			if idx, ok := codeBlockIdx[cb]; ok {
				cs.CodeBlockIdxs = append(cs.CodeBlockIdxs, idx)
			}
		}
		cd.Sections = append(cd.Sections, cs)
	}

	// Links
	for _, l := range doc.links {
		cd.Links = append(cd.Links, cachedLink{Text: l.Text, URL: l.URL})
	}

	// Images
	for _, img := range doc.images {
		cd.Images = append(cd.Images, cachedImage{
			AltText: img.AltText, URL: img.URL, Title: img.Title,
		})
	}

	// Tables
	for _, t := range doc.tables {
		cd.Tables = append(cd.Tables, cachedTable{
			Headers: t.Headers, Rows: t.Rows,
		})
	}

	// Lists
	for _, l := range doc.lists {
		cd.Lists = append(cd.Lists, listToCache(l))
	}

	return cd
}

func listToCache(l *List) cachedList {
	cl := cachedList{Ordered: l.Ordered}
	for _, item := range l.Items {
		cl.Items = append(cl.Items, listItemToCache(item))
	}
	return cl
}

func listItemToCache(item ListItem) cachedListItem {
	ci := cachedListItem{
		Text:    item.Text,
		Checked: item.Checked,
	}
	for _, child := range item.Children {
		ci.Children = append(ci.Children, listItemToCache(child))
	}
	return ci
}

func cacheToDocument(cd *cachedDocument, source []byte, path string) *Document {
	// Rebuild headings
	headings := make([]*Heading, len(cd.Headings))
	for i, ch := range cd.Headings {
		headings[i] = &Heading{
			Level: ch.Level,
			Text:  ch.Text,
			ID:    ch.ID,
			Line:  ch.Line,
			Page:  ch.Page,
		}
	}

	// Rebuild code blocks (before sections, so we can cross-reference)
	codeBlocks := make([]*CodeBlock, len(cd.CodeBlocks))
	for i, cc := range cd.CodeBlocks {
		codeBlocks[i] = &CodeBlock{
			Language: cc.Language, Content: cc.Content, Lines: cc.Lines,
		}
	}

	// Rebuild sections — for non-markdown formats (HTML, PDF), section line numbers
	// refer to the extracted readable text, not the raw file bytes. Use readableText
	// as the source so GetText() returns human-readable content instead of binary data.
	sectionSource := source
	if cd.Format != FormatMarkdown && cd.ReadableText != "" {
		sectionSource = []byte(cd.ReadableText)
	}
	sections := make([]*Section, len(cd.Sections))
	for i, cs := range cd.Sections {
		var h *Heading
		if cs.HeadingIdx >= 0 && cs.HeadingIdx < len(headings) {
			h = headings[cs.HeadingIdx]
		}
		if cs.Start > 0 {
			sections[i] = NewSectionWithSource(h, cs.Start, cs.End, sectionSource)
		} else {
			sections[i] = &Section{Heading: h, Start: cs.Start, End: cs.End}
		}
	}

	// Wire parent/child and re-attach code blocks
	for i, cs := range cd.Sections {
		if cs.ParentIdx >= 0 && cs.ParentIdx < len(sections) {
			sections[i].Parent = sections[cs.ParentIdx]
		}
		for _, childIdx := range cs.ChildIdxs {
			if childIdx >= 0 && childIdx < len(sections) {
				sections[i].Children = append(sections[i].Children, sections[childIdx])
			}
		}
		for _, cbIdx := range cs.CodeBlockIdxs {
			if cbIdx >= 0 && cbIdx < len(codeBlocks) {
				sections[i].AddCodeBlock(codeBlocks[cbIdx])
			}
		}
	}

	// Rebuild links
	var links []*Link
	for _, cl := range cd.Links {
		links = append(links, &Link{Text: cl.Text, URL: cl.URL})
	}

	// Rebuild images
	var images []*Image
	for _, ci := range cd.Images {
		images = append(images, &Image{
			AltText: ci.AltText, URL: ci.URL, Title: ci.Title,
		})
	}

	// Rebuild tables
	var tables []*Table
	for _, ct := range cd.Tables {
		tables = append(tables, &Table{Headers: ct.Headers, Rows: ct.Rows})
	}

	// Rebuild lists
	var lists []*List
	for _, cl := range cd.Lists {
		lists = append(lists, cacheToList(cl))
	}

	doc := NewDocument(
		source, path, cd.Format, cd.Title,
		headings, sections, codeBlocks, links, images, tables, lists,
		cd.ReadableText,
	)
	doc.SetPageCount(cd.PageCount)

	// Restore metadata (unexported field, settable within lib package)
	if cd.Metadata != nil {
		doc.metadata = cd.Metadata
	}

	return doc
}

func cacheToList(cl cachedList) *List {
	l := &List{Ordered: cl.Ordered}
	for _, ci := range cl.Items {
		l.Items = append(l.Items, cacheToListItem(ci))
	}
	return l
}

func cacheToListItem(ci cachedListItem) ListItem {
	item := ListItem{
		Text:    ci.Text,
		Checked: ci.Checked,
	}
	for _, child := range ci.Children {
		item.Children = append(item.Children, cacheToListItem(child))
	}
	return item
}
