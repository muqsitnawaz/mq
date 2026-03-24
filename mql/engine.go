package mql

import (
	"fmt"

	"github.com/muqsitnawaz/mq/data"
	"github.com/muqsitnawaz/mq/html"
	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/muqsitnawaz/mq/pdf"
)

// Engine provides the MQL query language on top of mq.MultiFormatEngine.
type Engine struct {
	mqEngine    *mq.Engine            // For backwards compatibility
	multiEngine *mq.MultiFormatEngine // For multi-format support
	executor    *QueryExecutor
	cache       *mq.Cache // Document parse cache (nil if cache unavailable)
}

// New creates a new MQL engine with multi-format support.
func New() *Engine {
	e := &Engine{
		mqEngine: mq.New(),
		multiEngine: mq.NewMultiFormatEngine(
			mq.WithFormatParser(html.NewParser()),
			mq.WithFormatParser(pdf.NewParser()),
			mq.WithFormatParser(data.NewJSONParser()),
			mq.WithFormatParser(data.NewJSONLParser()),
			mq.WithFormatParser(data.NewYAMLParser()),
		),
		executor: NewQueryExecutor(),
	}

	// Open cache silently — cache failures should never block normal operation
	if c, err := mq.OpenCache(""); err == nil {
		e.cache = c
		// Non-fatal: evict stale entries to prevent unbounded growth
		c.Trim()
	}

	return e
}

// NewWithOptions creates a new MQL engine with options.
func NewWithOptions(mqEngine *mq.Engine, opts ...QueryOption) *Engine {
	return &Engine{
		mqEngine: mqEngine,
		executor: NewQueryExecutor(opts...),
	}
}

// LoadDocument loads and parses a file (auto-detects format).
// Uses the cache to avoid re-parsing unchanged files.
func (e *Engine) LoadDocument(path string) (*mq.Document, error) {
	// Try cache first
	if e.cache != nil {
		if doc := e.cache.LookupFile(path); doc != nil {
			return doc, nil
		}
	}

	// Cache miss — read once, parse from those bytes, cache with same bytes+stat.
	// This avoids TOCTOU races where the file changes between read and stat.
	content, info, err := mq.ReadFileWithStat(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	doc, err := e.multiEngine.Parse(content, path)
	if err != nil {
		return nil, err
	}

	// Store in cache for next time
	if e.cache != nil {
		e.cache.StoreFile(path, content, info, doc)
	}

	return doc, nil
}

// Close releases cache resources. Call when the engine is no longer needed.
func (e *Engine) Close() {
	if e.cache != nil {
		e.cache.Close()
	}
}

// ParseDocument parses content (auto-detects format).
func (e *Engine) ParseDocument(content []byte, path string) (*mq.Document, error) {
	return e.multiEngine.Parse(content, path)
}

// Query executes an MQL query string on a document.
func (e *Engine) Query(doc *mq.Document, queryStr string) (interface{}, error) {
	return ExecuteQuery(doc, queryStr)
}

// QueryWithExecutor uses the configured executor for caching support.
func (e *Engine) QueryWithExecutor(doc *mq.Document, queryStr string) (interface{}, error) {
	return e.executor.Execute(doc, queryStr)
}

// From creates a fluent query builder (direct API from mq).
func (e *Engine) From(doc *mq.Document) *mq.QueryBuilder {
	return e.multiEngine.From(doc)
}

// GetMQEngine returns the underlying mq.Engine for direct API access.
func (e *Engine) GetMQEngine() *mq.Engine {
	return e.mqEngine
}
