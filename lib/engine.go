package mq

import (
	"fmt"
)

// Engine is the main entry point for the MQ library.
type Engine struct {
	parser *Parser
}

// EngineOption configures the engine.
type EngineOption func(*Engine)

// New creates a new MQ engine.
func New(opts ...EngineOption) *Engine {
	e := &Engine{
		parser: NewParser(),
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// WithParser sets a custom parser.
func WithParser(parser *Parser) EngineOption {
	return func(e *Engine) {
		e.parser = parser
	}
}

// LoadDocument loads and parses a markdown file.
func (e *Engine) LoadDocument(path string) (*Document, error) {
	return e.parser.ParseFile(path)
}

// ParseDocument parses markdown content.
func (e *Engine) ParseDocument(content []byte, path string) (*Document, error) {
	return e.parser.Parse(content, path)
}

// From creates a fluent query builder for a document.
func (e *Engine) From(doc *Document) *QueryBuilder {
	return &QueryBuilder{
		engine: e,
		doc:    doc,
	}
}

// QueryBuilder provides a fluent API for building queries.
type QueryBuilder struct {
	engine  *Engine
	doc     *Document
	current interface{}
	err     error
}

// Headings selects headings, optionally filtered by level.
func (qb *QueryBuilder) Headings(levels ...int) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.current = qb.doc.GetHeadings(levels...)
	return qb
}

// Section selects a section by title.
func (qb *QueryBuilder) Section(title string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	section, ok := qb.doc.GetSection(title)
	if !ok {
		qb.err = fmt.Errorf("section not found: %s", title)
		return qb
	}
	qb.current = section
	return qb
}

// Sections selects all sections.
func (qb *QueryBuilder) Sections() *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.current = qb.doc.GetSections()
	return qb
}

// Code selects code blocks, optionally filtered by language.
func (qb *QueryBuilder) Code(languages ...string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.current = qb.doc.GetCodeBlocks(languages...)
	return qb
}

// Links selects all links.
func (qb *QueryBuilder) Links() *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.current = qb.doc.GetLinks()
	return qb
}

// Images selects all images.
func (qb *QueryBuilder) Images() *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.current = qb.doc.GetImages()
	return qb
}

// Tables selects all tables.
func (qb *QueryBuilder) Tables() *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.current = qb.doc.GetTables()
	return qb
}

// Lists selects lists, optionally filtered by type.
func (qb *QueryBuilder) Lists(ordered *bool) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.current = qb.doc.GetLists(ordered)
	return qb
}

// WhereOwner filters by document owner.
func (qb *QueryBuilder) WhereOwner(owner string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	if !qb.doc.CheckOwnership(owner) {
		qb.err = fmt.Errorf("ownership check failed for: %s", owner)
		qb.current = nil
	}
	return qb
}

// WhereTag filters by document tags.
func (qb *QueryBuilder) WhereTag(tag string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	tags := qb.doc.GetTags()
	found := false
	for _, t := range tags {
		if t == tag {
			found = true
			break
		}
	}
	if !found {
		qb.err = fmt.Errorf("tag not found: %s", tag)
		qb.current = nil
	}
	return qb
}

// WherePriority filters by document priority.
func (qb *QueryBuilder) WherePriority(priority string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	docPriority, ok := qb.doc.GetPriority()
	if !ok || docPriority != priority {
		qb.err = fmt.Errorf("priority mismatch: expected %s", priority)
		qb.current = nil
	}
	return qb
}

// Filter applies a filter to the current results.
func (qb *QueryBuilder) Filter(predicate func(interface{}) bool) *QueryBuilder {
	if qb.err != nil || qb.current == nil {
		return qb
	}

	// Type-specific filtering
	switch v := qb.current.(type) {
	case []*Heading:
		qb.current = Filter(v, func(h *Heading) bool {
			return predicate(h)
		})
	case []*Section:
		qb.current = Filter(v, func(s *Section) bool {
			return predicate(s)
		})
	case []*CodeBlock:
		qb.current = Filter(v, func(cb *CodeBlock) bool {
			return predicate(cb)
		})
	case []*Link:
		qb.current = Filter(v, func(l *Link) bool {
			return predicate(l)
		})
	default:
		qb.err = fmt.Errorf("filter not supported for type: %T", qb.current)
	}

	return qb
}

// Map transforms the current results.
func (qb *QueryBuilder) Map(transform func(interface{}) interface{}) *QueryBuilder {
	if qb.err != nil || qb.current == nil {
		return qb
	}

	// Type-specific mapping
	switch v := qb.current.(type) {
	case []*Heading:
		result := make([]interface{}, len(v))
		for i, h := range v {
			result[i] = transform(h)
		}
		qb.current = result
	case []*Section:
		result := make([]interface{}, len(v))
		for i, s := range v {
			result[i] = transform(s)
		}
		qb.current = result
	case []*CodeBlock:
		result := make([]interface{}, len(v))
		for i, cb := range v {
			result[i] = transform(cb)
		}
		qb.current = result
	default:
		qb.err = fmt.Errorf("map not supported for type: %T", qb.current)
	}

	return qb
}

// Take limits the results to n items.
func (qb *QueryBuilder) Take(n int) *QueryBuilder {
	if qb.err != nil || qb.current == nil {
		return qb
	}

	switch v := qb.current.(type) {
	case []*Heading:
		qb.current = Take(v, n)
	case []*Section:
		qb.current = Take(v, n)
	case []*CodeBlock:
		qb.current = Take(v, n)
	case []*Link:
		qb.current = Take(v, n)
	case []*Image:
		qb.current = Take(v, n)
	case []*Table:
		qb.current = Take(v, n)
	case []*List:
		qb.current = Take(v, n)
	default:
		qb.err = fmt.Errorf("take not supported for type: %T", qb.current)
	}

	return qb
}

// Skip skips the first n items.
func (qb *QueryBuilder) Skip(n int) *QueryBuilder {
	if qb.err != nil || qb.current == nil {
		return qb
	}

	switch v := qb.current.(type) {
	case []*Heading:
		qb.current = Skip(v, n)
	case []*Section:
		qb.current = Skip(v, n)
	case []*CodeBlock:
		qb.current = Skip(v, n)
	case []*Link:
		qb.current = Skip(v, n)
	default:
		qb.err = fmt.Errorf("skip not supported for type: %T", qb.current)
	}

	return qb
}

// Count returns the number of items in the current result.
func (qb *QueryBuilder) Count() (int, error) {
	if qb.err != nil {
		return 0, qb.err
	}

	if qb.current == nil {
		return 0, nil
	}

	switch v := qb.current.(type) {
	case []*Heading:
		return len(v), nil
	case []*Section:
		return len(v), nil
	case []*CodeBlock:
		return len(v), nil
	case []*Link:
		return len(v), nil
	case []*Image:
		return len(v), nil
	case []*Table:
		return len(v), nil
	case []*List:
		return len(v), nil
	case *Section:
		return 1, nil
	default:
		return 0, fmt.Errorf("count not supported for type: %T", qb.current)
	}
}

// Result returns the final query result.
func (qb *QueryBuilder) Result() (interface{}, error) {
	if qb.err != nil {
		return nil, qb.err
	}
	return qb.current, nil
}

// Execute is an alias for Result for consistency.
func (qb *QueryBuilder) Execute() (interface{}, error) {
	return qb.Result()
}

// AsHeadings casts the result to headings.
func (qb *QueryBuilder) AsHeadings() ([]*Heading, error) {
	if qb.err != nil {
		return nil, qb.err
	}
	if headings, ok := qb.current.([]*Heading); ok {
		return headings, nil
	}
	return nil, fmt.Errorf("result is not []*Heading, got %T", qb.current)
}

// AsSections casts the result to sections.
func (qb *QueryBuilder) AsSections() ([]*Section, error) {
	if qb.err != nil {
		return nil, qb.err
	}
	if sections, ok := qb.current.([]*Section); ok {
		return sections, nil
	}
	return nil, fmt.Errorf("result is not []*Section, got %T", qb.current)
}

// AsSection casts the result to a single section.
func (qb *QueryBuilder) AsSection() (*Section, error) {
	if qb.err != nil {
		return nil, qb.err
	}
	if section, ok := qb.current.(*Section); ok {
		return section, nil
	}
	return nil, fmt.Errorf("result is not *Section, got %T", qb.current)
}

// AsCodeBlocks casts the result to code blocks.
func (qb *QueryBuilder) AsCodeBlocks() ([]*CodeBlock, error) {
	if qb.err != nil {
		return nil, qb.err
	}
	if blocks, ok := qb.current.([]*CodeBlock); ok {
		return blocks, nil
	}
	return nil, fmt.Errorf("result is not []*CodeBlock, got %T", qb.current)
}

// AsLinks casts the result to links.
func (qb *QueryBuilder) AsLinks() ([]*Link, error) {
	if qb.err != nil {
		return nil, qb.err
	}
	if links, ok := qb.current.([]*Link); ok {
		return links, nil
	}
	return nil, fmt.Errorf("result is not []*Link, got %T", qb.current)
}
