package mql

import (
	"fmt"

	mq "github.com/muqsitnawaz/mq/lib"
)

// ExecuteQuery executes an MQL query string on a document.
func ExecuteQuery(doc *mq.Document, queryStr string) (interface{}, error) {
	// Parse the query
	ast, err := ParseString(queryStr)
	if err != nil {
		return nil, fmt.Errorf("parsing query: %w", err)
	}

	// Compile the query
	compiler := NewCompiler()
	plan := compiler.Compile(ast)

	// Create evaluation context
	ctx := NewEvalContext(doc)

	// Execute the plan
	return plan(ctx)
}

// QueryOption configures query execution.
type QueryOption func(*queryOptions)

type queryOptions struct {
	strict bool
	cache  bool
}

// WithQueryCache enables query plan caching.
func WithQueryCache() QueryOption {
	return func(o *queryOptions) {
		o.cache = true
	}
}

// QueryExecutor provides advanced query execution with options.
type QueryExecutor struct {
	engine   *Engine
	compiler *Compiler
	cache    map[string]ExecutionPlan
}

// NewQueryExecutor creates a new query executor.
func NewQueryExecutor(opts ...QueryOption) *QueryExecutor {
	options := &queryOptions{}
	for _, opt := range opts {
		opt(options)
	}

	var compilerOpts []CompilerOption
	if options.strict {
		compilerOpts = append(compilerOpts, WithStrictMode())
	}

	qe := &QueryExecutor{
		compiler: NewCompiler(compilerOpts...),
	}

	if options.cache {
		qe.cache = make(map[string]ExecutionPlan)
	}

	return qe
}

// Execute executes a query on a document.
func (qe *QueryExecutor) Execute(doc *mq.Document, query string) (interface{}, error) {
	var plan ExecutionPlan
	var err error

	// Check cache if enabled
	if qe.cache != nil {
		if cached, ok := qe.cache[query]; ok {
			plan = cached
		} else {
			plan, err = qe.compiler.CompileString(query)
			if err != nil {
				return nil, err
			}
			qe.cache[query] = plan
		}
	} else {
		plan, err = qe.compiler.CompileString(query)
		if err != nil {
			return nil, err
		}
	}

	// Execute the plan
	ctx := NewEvalContext(doc)
	return plan(ctx)
}
