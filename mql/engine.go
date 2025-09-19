package mql

import (
	mq "github.com/muqsitnawaz/mq/lib"
)

// Engine provides the MQL query language on top of mq.Engine.
type Engine struct {
	mqEngine *mq.Engine
	executor *QueryExecutor
}

// New creates a new MQL engine.
func New() *Engine {
	return &Engine{
		mqEngine: mq.New(),
		executor: NewQueryExecutor(),
	}
}

// NewWithOptions creates a new MQL engine with options.
func NewWithOptions(mqEngine *mq.Engine, opts ...QueryOption) *Engine {
	return &Engine{
		mqEngine: mqEngine,
		executor: NewQueryExecutor(opts...),
	}
}

// LoadDocument loads and parses a markdown file.
func (e *Engine) LoadDocument(path string) (*mq.Document, error) {
	return e.mqEngine.LoadDocument(path)
}

// ParseDocument parses markdown content.
func (e *Engine) ParseDocument(content []byte, path string) (*mq.Document, error) {
	return e.mqEngine.ParseDocument(content, path)
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
	return e.mqEngine.From(doc)
}

// GetMQEngine returns the underlying mq.Engine for direct API access.
func (e *Engine) GetMQEngine() *mq.Engine {
	return e.mqEngine
}
