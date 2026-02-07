package mql

import (
	"fmt"
	"reflect"
	"strings"

	mq "github.com/muqsitnawaz/mq/lib"
)

// ExecutionPlan is a compiled query ready for execution.
type ExecutionPlan func(*EvalContext) (interface{}, error)

// EvalContext maintains state during query execution.
type EvalContext struct {
	Document  *mq.Document
	Current   interface{}
	Variables map[string]interface{}
}

// NewEvalContext creates a new evaluation context.
func NewEvalContext(doc *mq.Document) *EvalContext {
	return &EvalContext{
		Document:  doc,
		Current:   doc,
		Variables: make(map[string]interface{}),
	}
}

// Compiler compiles query AST to executable plans.
type Compiler struct {
	// Options
	strict bool // Strict type checking
}

// CompilerOption configures the compiler.
type CompilerOption func(*Compiler)

// NewCompiler creates a new compiler.
func NewCompiler(opts ...CompilerOption) *Compiler {
	c := &Compiler{
		strict: false,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithStrictMode enables strict type checking.
func WithStrictMode() CompilerOption {
	return func(c *Compiler) {
		c.strict = true
	}
}

// Compile compiles an AST node to an execution plan.
func (c *Compiler) Compile(node QueryNode) ExecutionPlan {
	return func(ctx *EvalContext) (interface{}, error) {
		visitor := &compilerVisitor{
			compiler: c,
			context:  ctx,
		}
		return node.Accept(visitor)
	}
}

// CompileString compiles a query string directly.
func (c *Compiler) CompileString(query string) (ExecutionPlan, error) {
	ast, err := ParseString(query)
	if err != nil {
		return nil, fmt.Errorf("parsing query: %w", err)
	}

	return c.Compile(ast), nil
}

// compilerVisitor implements the Visitor pattern for compilation.
type compilerVisitor struct {
	compiler *Compiler
	context  *EvalContext
}

// SetContext sets the evaluation context.
func (v *compilerVisitor) SetContext(ctx *EvalContext) {
	v.context = ctx
}

// VisitPipe compiles a pipe operation.
func (v *compilerVisitor) VisitPipe(node *PipeNode) (interface{}, error) {
	// Execute left side
	leftResult, err := node.Left.Accept(v)
	if err != nil {
		return nil, err
	}

	// Update context with left result
	oldCurrent := v.context.Current
	v.context.Current = leftResult

	// Execute right side with updated context
	rightResult, err := node.Right.Accept(v)
	if err != nil {
		return nil, err
	}

	// Restore context
	v.context.Current = oldCurrent

	return rightResult, nil
}

// VisitSelector compiles a selector operation.
func (v *compilerVisitor) VisitSelector(node *SelectorNode) (interface{}, error) {
	// Check if selector is a property accessor on current item
	if v.context.Current != nil && v.context.Current != v.context.Document {
		// Try to handle as property access
		result, handled := v.handlePropertyAccess(node.Name)
		if handled {
			return result, nil
		}

		// Check if it's a property extraction on a collection
		result, handled = v.handleCollectionPropertyAccess(node.Name)
		if handled {
			return result, nil
		}

		// Special handling for .code selector on sections
		if node.Name == "code" {
			if section, ok := v.context.Current.(*mq.Section); ok {
				// Evaluate arguments if any
				args := make([]interface{}, len(node.Args))
				for i, arg := range node.Args {
					val, err := arg.Accept(v)
					if err != nil {
						return nil, err
					}
					args[i] = val
				}
				langs := extractStringArgs(args)
				return section.GetCodeBlocks(langs...), nil
			}
		}
	}

	// Get the document from context
	doc := v.context.Document
	if doc == nil {
		return nil, fmt.Errorf("no document in context")
	}

	// Evaluate arguments
	args := make([]interface{}, len(node.Args))
	for i, arg := range node.Args {
		val, err := arg.Accept(v)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}

	// Execute selector based on name
	switch node.Name {
	case "headings":
		levels := extractIntArgs(args)
		return doc.GetHeadings(levels...), nil

	case "section":
		if len(args) == 0 {
			return nil, fmt.Errorf("Error: .section requires a title argument\nUsage: .section(\"Section Title\")\nHint: Use .sections to get all sections")
		}
		title, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("Error: .section requires a string title, got %T\nUsage: .section(\"Section Title\")", args[0])
		}
		section, found := doc.GetSection(title)
		if !found {
			return nil, fmt.Errorf("section not found: %s", title)
		}
		return section, nil

	case "sections":
		return doc.GetSections(), nil

	case "code":
		langs := extractStringArgs(args)
		return doc.GetCodeBlocks(langs...), nil

	case "links":
		return doc.GetLinks(), nil

	case "images":
		return doc.GetImages(), nil

	case "tables":
		return doc.GetTables(), nil

	case "lists":
		if len(args) > 0 {
			if ordered, ok := args[0].(bool); ok {
				return doc.GetLists(&ordered), nil
			}
		}
		return doc.GetLists(nil), nil

	case "metadata":
		return doc.Metadata(), nil

	case "owner":
		owner, ok := doc.GetOwner()
		if !ok {
			return "", nil
		}
		return owner, nil

	case "tags":
		return doc.GetTags(), nil

	case "priority":
		priority, _ := doc.GetPriority()
		return priority, nil

	case "text":
		// Extract text from current context
		return extractTextFromAny(v.context.Current), nil

	case "length":
		return getLength(v.context.Current), nil

	case "select", "filter":
		// These are treated as filters with predicates
		if len(node.Args) == 0 {
			return nil, fmt.Errorf("Error: .%s requires a predicate\nUsage: .%s(.property == \"value\")\nExample: .headings | .filter(.level == 2)", node.Name, node.Name)
		}
		// Create a filter node and visit it
		filterNode := &FilterNode{
			Predicate: node.Args[0],
		}
		return v.VisitFilter(filterNode)

	case "tree":
		// Check if we're operating on a section or the whole document
		mode := mq.TreeModeDefault
		if len(args) > 0 {
			if s, ok := args[0].(string); ok {
				switch s {
				case "compact":
					mode = mq.TreeModeCompact
				case "preview":
					mode = mq.TreeModePreview
				case "full":
					mode = mq.TreeModeFull
				}
			}
		}

		// If current context is a section, build tree for that section
		if section, ok := v.context.Current.(*mq.Section); ok {
			return buildSectionTree(section, mode), nil
		}

		// Otherwise, build tree for the whole document
		return doc.BuildTree(mode), nil

	case "search":
		if len(args) == 0 {
			return nil, fmt.Errorf("Error: .search requires a query string\nUsage: .search(\"query\")")
		}
		query, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("Error: .search requires a string query, got %T\nUsage: .search(\"query\")", args[0])
		}
		return doc.Search(query), nil

	default:
		return nil, formatUnknownSelectorError(node.Name)
	}
}

// formatUnknownSelectorError generates helpful error message for unknown selectors.
func formatUnknownSelectorError(name string) error {
	// Known selectors for suggestions
	knownSelectors := []string{
		"headings", "section", "sections", "code", "links", "images",
		"tables", "lists", "metadata", "owner", "tags", "priority",
		"text", "length", "tree", "search",
	}

	// Find closest match using simple string distance
	suggestion := findClosestMatch(name, knownSelectors)

	if suggestion != "" {
		return fmt.Errorf("Error: unknown selector: .%s\nDid you mean: .%s?", name, suggestion)
	}

	return fmt.Errorf("Error: unknown selector: .%s\nAvailable selectors: .headings, .section(title), .sections, .code, .links, .images, .tables, .lists, .metadata, .owner, .tags, .priority, .text, .length, .tree, .search(query)", name)
}

// findClosestMatch finds the closest matching string using simple heuristics.
func findClosestMatch(input string, candidates []string) string {
	input = strings.ToLower(input)

	// First check for exact prefix or suffix matches
	for _, candidate := range candidates {
		lower := strings.ToLower(candidate)
		if strings.HasPrefix(lower, input) || strings.HasPrefix(input, lower) {
			return candidate
		}
		if strings.HasSuffix(lower, input) || strings.HasSuffix(input, lower) {
			return candidate
		}
	}

	// Check for singular vs plural
	if !strings.HasSuffix(input, "s") {
		plural := input + "s"
		for _, candidate := range candidates {
			if strings.ToLower(candidate) == plural {
				return candidate
			}
		}
	} else {
		singular := strings.TrimSuffix(input, "s")
		for _, candidate := range candidates {
			if strings.ToLower(candidate) == singular {
				return candidate
			}
		}
	}

	// Check if any candidate contains the input or vice versa
	for _, candidate := range candidates {
		lower := strings.ToLower(candidate)
		if strings.Contains(lower, input) || strings.Contains(input, lower) {
			return candidate
		}
	}

	return ""
}

// VisitFilter compiles a filter operation.
func (v *compilerVisitor) VisitFilter(node *FilterNode) (interface{}, error) {
	current := v.context.Current
	if current == nil {
		return nil, fmt.Errorf("Error: no data to filter\nHint: Use a selector before filter, e.g., .headings | .filter(.level == 2)")
	}

	// Handle different collection types
	switch data := current.(type) {
	case []*mq.Heading:
		return v.filterHeadings(data, node.Predicate, v)

	case []*mq.Section:
		return v.filterSections(data, node.Predicate, v)

	case []*mq.CodeBlock:
		return v.filterCodeBlocks(data, node.Predicate, v)

	case []*mq.Link:
		return v.filterLinks(data, node.Predicate, v)

	default:
		return nil, fmt.Errorf("Error: cannot filter type: %T\nHint: filter works on collections like headings, sections, code blocks, and links", current)
	}
}

// filterHeadings filters headings based on predicate.
func (c *compilerVisitor) filterHeadings(headings []*mq.Heading, predicate QueryNode, v *compilerVisitor) ([]*mq.Heading, error) {
	var result []*mq.Heading

	for _, heading := range headings {
		// Set current item for predicate evaluation
		oldCurrent := v.context.Current
		v.context.Current = heading

		// Evaluate predicate
		match, err := predicate.Accept(v)
		if err != nil {
			return nil, err
		}

		// Restore context
		v.context.Current = oldCurrent

		// Check if predicate matched
		if toBool(match) {
			result = append(result, heading)
		}
	}

	return result, nil
}

// filterSections filters sections based on predicate.
func (c *compilerVisitor) filterSections(sections []*mq.Section, predicate QueryNode, v *compilerVisitor) ([]*mq.Section, error) {
	var result []*mq.Section

	for _, section := range sections {
		oldCurrent := v.context.Current
		v.context.Current = section

		match, err := predicate.Accept(v)
		if err != nil {
			return nil, err
		}

		v.context.Current = oldCurrent

		if toBool(match) {
			result = append(result, section)
		}
	}

	return result, nil
}

// filterCodeBlocks filters code blocks based on predicate.
func (c *compilerVisitor) filterCodeBlocks(blocks []*mq.CodeBlock, predicate QueryNode, v *compilerVisitor) ([]*mq.CodeBlock, error) {
	var result []*mq.CodeBlock

	for _, block := range blocks {
		oldCurrent := v.context.Current
		v.context.Current = block

		match, err := predicate.Accept(v)
		if err != nil {
			return nil, err
		}

		v.context.Current = oldCurrent

		if toBool(match) {
			result = append(result, block)
		}
	}

	return result, nil
}

// filterLinks filters links based on predicate.
func (c *compilerVisitor) filterLinks(links []*mq.Link, predicate QueryNode, v *compilerVisitor) ([]*mq.Link, error) {
	var result []*mq.Link

	for _, link := range links {
		oldCurrent := v.context.Current
		v.context.Current = link

		match, err := predicate.Accept(v)
		if err != nil {
			return nil, err
		}

		v.context.Current = oldCurrent

		if toBool(match) {
			result = append(result, link)
		}
	}

	return result, nil
}

// VisitFunction compiles a function call.
func (v *compilerVisitor) VisitFunction(node *FunctionNode) (interface{}, error) {
	// Evaluate arguments
	args := make([]interface{}, len(node.Args))
	for i, arg := range node.Args {
		val, err := arg.Accept(v)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}

	// Execute function
	switch node.Name {
	case "map":
		if len(node.Args) != 1 {
			return nil, fmt.Errorf("Error: map requires 1 argument\nUsage: .collection | map(.property)")
		}
		return v.mapOperation(node.Args[0])

	case "contains":
		if len(args) != 1 {
			return nil, fmt.Errorf("Error: contains requires 1 argument\nUsage: .property | contains(\"substring\")")
		}
		return contains(v.context.Current, args[0])

	case "startswith":
		if len(args) != 1 {
			return nil, fmt.Errorf("Error: startswith requires 1 argument\nUsage: .property | startswith(\"prefix\")")
		}
		return startsWith(v.context.Current, args[0])

	case "endswith":
		if len(args) != 1 {
			return nil, fmt.Errorf("Error: endswith requires 1 argument\nUsage: .property | endswith(\"suffix\")")
		}
		return endsWith(v.context.Current, args[0])

	case "length":
		return getLength(v.context.Current), nil

	default:
		return nil, formatUnknownFunctionError(node.Name)
	}
}

// formatUnknownFunctionError generates helpful error message for unknown functions.
func formatUnknownFunctionError(name string) error {
	knownFunctions := []string{"map", "contains", "startswith", "endswith", "length"}
	suggestion := findClosestMatch(name, knownFunctions)

	if suggestion != "" {
		return fmt.Errorf("Error: unknown function: %s()\nDid you mean: %s()?", name, suggestion)
	}

	return fmt.Errorf("Error: unknown function: %s()\nAvailable functions: map(), contains(), startswith(), endswith(), length()", name)
}

// VisitBinary compiles a binary operation.
func (v *compilerVisitor) VisitBinary(node *BinaryNode) (interface{}, error) {
	// Evaluate left operand
	left, err := node.Left.Accept(v)
	if err != nil {
		return nil, err
	}

	// Short-circuit evaluation for logical operators
	switch node.Operator {
	case "and":
		if !toBool(left) {
			return false, nil
		}
	case "or":
		if toBool(left) {
			return true, nil
		}
	}

	// Evaluate right operand
	right, err := node.Right.Accept(v)
	if err != nil {
		return nil, err
	}

	// Execute operation
	switch node.Operator {
	case "==":
		return equals(left, right), nil
	case "!=":
		return !equals(left, right), nil
	case "<":
		return lessThan(left, right)
	case "<=":
		return lessEqual(left, right)
	case ">":
		return greaterThan(left, right)
	case ">=":
		return greaterEqual(left, right)
	case "and":
		return toBool(left) && toBool(right), nil
	case "or":
		return toBool(left) || toBool(right), nil
	default:
		return nil, fmt.Errorf("Error: unknown operator: %s\nSupported operators: ==, !=, <, <=, >, >=, and, or", node.Operator)
	}
}

// VisitUnary compiles a unary operation.
func (v *compilerVisitor) VisitUnary(node *UnaryNode) (interface{}, error) {
	operand, err := node.Operand.Accept(v)
	if err != nil {
		return nil, err
	}

	switch node.Operator {
	case "!":
		return !toBool(operand), nil
	case "-":
		return negate(operand)
	default:
		return nil, fmt.Errorf("Error: unknown unary operator: %s\nSupported operators: !, -", node.Operator)
	}
}

// VisitLiteral compiles a literal value.
func (v *compilerVisitor) VisitLiteral(node *LiteralNode) (interface{}, error) {
	return node.Value, nil
}

// VisitIdentifier compiles an identifier (property access).
func (v *compilerVisitor) VisitIdentifier(node *IdentifierNode) (interface{}, error) {
	// Check variables first
	if val, ok := v.context.Variables[node.Name]; ok {
		return val, nil
	}

	// Access property on current object
	return getProperty(v.context.Current, node.Name)
}

// VisitIndex compiles an index operation.
func (v *compilerVisitor) VisitIndex(node *IndexNode) (interface{}, error) {
	// Evaluate object
	obj, err := node.Object.Accept(v)
	if err != nil {
		return nil, err
	}

	// Evaluate index
	index, err := node.Index.Accept(v)
	if err != nil {
		return nil, err
	}

	return getIndex(obj, index)
}

// VisitSlice compiles a slice operation.
func (v *compilerVisitor) VisitSlice(node *SliceNode) (interface{}, error) {
	// Evaluate object
	obj, err := node.Object.Accept(v)
	if err != nil {
		return nil, err
	}

	// Evaluate start index
	var start interface{}
	if node.Start != nil {
		start, err = node.Start.Accept(v)
		if err != nil {
			return nil, err
		}
	}

	// Evaluate end index
	var end interface{}
	if node.End != nil {
		end, err = node.End.Accept(v)
		if err != nil {
			return nil, err
		}
	}

	return getSlice(obj, start, end)
}

// Helper functions for property access

func getProperty(obj interface{}, name string) (interface{}, error) {
	switch v := obj.(type) {
	case *mq.Heading:
		switch name {
		case "level":
			return v.Level, nil
		case "text":
			return v.Text, nil
		case "id":
			return v.ID, nil
		default:
			available := []string{"level", "text", "id"}
			suggestion := findClosestMatch(name, available)
			if suggestion != "" {
				return nil, fmt.Errorf("Error: heading has no property: .%s\nDid you mean: .%s?\nAvailable: .level, .text, .id", name, suggestion)
			}
			return nil, fmt.Errorf("Error: heading has no property: .%s\nAvailable: .level, .text, .id", name)
		}

	case *mq.Section:
		switch name {
		case "heading":
			return v.Heading, nil
		case "text":
			return v.GetText(), nil
		case "start":
			return v.Start, nil
		case "end":
			return v.End, nil
		default:
			available := []string{"heading", "text", "start", "end"}
			suggestion := findClosestMatch(name, available)
			if suggestion != "" {
				return nil, fmt.Errorf("Error: section has no property: .%s\nDid you mean: .%s?\nAvailable: .heading, .text, .start, .end", name, suggestion)
			}
			return nil, fmt.Errorf("Error: section has no property: .%s\nAvailable: .heading, .text, .start, .end", name)
		}

	case *mq.CodeBlock:
		switch name {
		case "language":
			return v.Language, nil
		case "content":
			return v.Content, nil
		case "lines":
			return v.GetLines(), nil
		default:
			available := []string{"language", "content", "lines"}
			suggestion := findClosestMatch(name, available)
			if suggestion != "" {
				return nil, fmt.Errorf("Error: code block has no property: .%s\nDid you mean: .%s?\nAvailable: .language, .content, .lines", name, suggestion)
			}
			return nil, fmt.Errorf("Error: code block has no property: .%s\nAvailable: .language, .content, .lines", name)
		}

	case *mq.Link:
		switch name {
		case "text":
			return v.Text, nil
		case "url":
			return v.URL, nil
		default:
			available := []string{"text", "url"}
			suggestion := findClosestMatch(name, available)
			if suggestion != "" {
				return nil, fmt.Errorf("Error: link has no property: .%s\nDid you mean: .%s?\nAvailable: .text, .url", name, suggestion)
			}
			return nil, fmt.Errorf("Error: link has no property: .%s\nAvailable: .text, .url", name)
		}

	default:
		return nil, fmt.Errorf("Error: cannot access property .%s on type %T", name, obj)
	}
}

// Helper functions for type conversion and comparison

func extractIntArgs(args []interface{}) []int {
	var result []int
	for _, arg := range args {
		switch v := arg.(type) {
		case int:
			result = append(result, v)
		case int64:
			result = append(result, int(v))
		case float64:
			result = append(result, int(v))
		}
	}
	return result
}

func extractStringArgs(args []interface{}) []string {
	var result []string
	for _, arg := range args {
		if s, ok := arg.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func toBool(v interface{}) bool {
	if v == nil {
		return false
	}

	switch val := v.(type) {
	case bool:
		return val
	case int, int64:
		return val != 0
	case float64:
		return val != 0.0
	case string:
		return val != ""
	default:
		// Collections are truthy if non-empty
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			return rv.Len() > 0
		default:
			return true // Non-nil objects are truthy
		}
	}
}

func equals(a, b interface{}) bool {
	// Try numeric comparison first
	na, aIsNum := toNumber(a)
	nb, bIsNum := toNumber(b)
	if aIsNum && bIsNum {
		return na == nb
	}

	// Fall back to DeepEqual for other types
	return reflect.DeepEqual(a, b)
}

func lessThan(a, b interface{}) (bool, error) {
	// Convert to comparable numeric types
	na, aIsNum := toNumber(a)
	nb, bIsNum := toNumber(b)

	if aIsNum && bIsNum {
		return na < nb, nil
	}

	// String comparison
	switch va := a.(type) {
	case string:
		if vb, ok := b.(string); ok {
			return va < vb, nil
		}
	}

	return false, fmt.Errorf("Error: cannot compare %T and %T\nHint: comparison operators work with numbers or strings", a, b)
}

// toNumber converts various numeric types to float64 for comparison
func toNumber(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case float64:
		return val, true
	case float32:
		return float64(val), true
	}
	return 0, false
}

func lessEqual(a, b interface{}) (bool, error) {
	lt, err := lessThan(a, b)
	if err != nil {
		return false, err
	}
	return lt || equals(a, b), nil
}

func greaterThan(a, b interface{}) (bool, error) {
	lt, err := lessEqual(a, b)
	if err != nil {
		return false, err
	}
	return !lt, nil
}

func greaterEqual(a, b interface{}) (bool, error) {
	lt, err := lessThan(a, b)
	if err != nil {
		return false, err
	}
	return !lt, nil
}

func negate(v interface{}) (interface{}, error) {
	switch val := v.(type) {
	case int:
		return -val, nil
	case int64:
		return -val, nil
	case float64:
		return -val, nil
	default:
		return nil, fmt.Errorf("Error: cannot negate %T\nHint: negation (-) only works with numbers", v)
	}
}

func contains(obj, search interface{}) (bool, error) {
	objStr := fmt.Sprintf("%v", obj)
	searchStr := fmt.Sprintf("%v", search)
	return strings.Contains(objStr, searchStr), nil
}

func startsWith(obj, prefix interface{}) (bool, error) {
	objStr := fmt.Sprintf("%v", obj)
	prefixStr := fmt.Sprintf("%v", prefix)
	return strings.HasPrefix(objStr, prefixStr), nil
}

func endsWith(obj, suffix interface{}) (bool, error) {
	objStr := fmt.Sprintf("%v", obj)
	suffixStr := fmt.Sprintf("%v", suffix)
	return strings.HasSuffix(objStr, suffixStr), nil
}

func getLength(obj interface{}) int {
	if obj == nil {
		return 0
	}

	rv := reflect.ValueOf(obj)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
		return rv.Len()
	default:
		return 0
	}
}

func extractText(obj interface{}) string {
	switch v := obj.(type) {
	case *mq.Heading:
		return v.Text
	case *mq.Section:
		return v.GetText()
	case *mq.CodeBlock:
		return v.Content
	case *mq.Link:
		return v.Text
	case string:
		return v
	default:
		return fmt.Sprintf("%v", obj)
	}
}

// handleCollectionPropertyAccess handles property access on collections
func (v *compilerVisitor) handleCollectionPropertyAccess(property string) (interface{}, bool) {
	current := v.context.Current

	switch items := current.(type) {
	case []*mq.Section:
		switch property {
		case "heading":
			results := make([]*mq.Heading, len(items))
			for i, section := range items {
				results[i] = section.Heading
			}
			return results, true
		case "text":
			results := make([]string, len(items))
			for i, section := range items {
				results[i] = section.GetText()
			}
			return results, true
		}
	case []*mq.Heading:
		// Already handled by extractTextFromAny for .text
		// Add other properties if needed
	case []*mq.CodeBlock:
		// Already handled by extractTextFromAny for .text
		// Add other properties if needed
	}

	return nil, false
}

// handlePropertyAccess handles property access on current context item
func (v *compilerVisitor) handlePropertyAccess(property string) (interface{}, bool) {
	current := v.context.Current

	switch item := current.(type) {
	case *mq.Heading:
		switch property {
		case "text":
			return item.Text, true
		case "level":
			return item.Level, true
		case "id":
			return item.ID, true
		}

	case *mq.Section:
		switch property {
		case "text":
			return item.GetText(), true
		case "heading":
			return item.Heading, true
		case "children":
			return item.Children, true
		case "start":
			return item.Start, true
		case "end":
			return item.End, true
			// Note: "code" is handled specially in VisitSelector to support arguments
		}

	case *mq.CodeBlock:
		switch property {
		case "content", "text":
			return item.Content, true
		case "language":
			return item.Language, true
		case "lines":
			return item.GetLines(), true
		}

	case *mq.Link:
		switch property {
		case "text":
			return item.Text, true
		case "url":
			return item.URL, true
		}

	case *mq.Image:
		switch property {
		case "text", "alttext", "alt":
			return item.AltText, true
		case "url":
			return item.URL, true
		}

	case *mq.Table:
		switch property {
		case "headers":
			return item.Headers, true
		case "rows":
			return item.Rows, true
		}
	}

	// Property not handled
	return nil, false
}

// mapOperation applies a transformation to each element in a collection
func (v *compilerVisitor) mapOperation(transform QueryNode) (interface{}, error) {
	current := v.context.Current
	if current == nil {
		return nil, fmt.Errorf("Error: no data to map\nHint: Use a selector before map, e.g., .sections | map(.text)")
	}

	// Handle different collection types
	switch data := current.(type) {
	case []*mq.Heading:
		results := make([]interface{}, len(data))
		for i, item := range data {
			// Set current context to individual item
			oldCurrent := v.context.Current
			v.context.Current = item

			// Apply transformation
			result, err := transform.Accept(v)
			if err != nil {
				return nil, err
			}
			results[i] = result

			// Restore context
			v.context.Current = oldCurrent
		}
		return results, nil

	case []*mq.Section:
		results := make([]interface{}, len(data))
		for i, item := range data {
			oldCurrent := v.context.Current
			v.context.Current = item
			result, err := transform.Accept(v)
			if err != nil {
				return nil, err
			}
			results[i] = result
			v.context.Current = oldCurrent
		}
		return results, nil

	case []*mq.CodeBlock:
		results := make([]interface{}, len(data))
		for i, item := range data {
			oldCurrent := v.context.Current
			v.context.Current = item
			result, err := transform.Accept(v)
			if err != nil {
				return nil, err
			}
			results[i] = result
			v.context.Current = oldCurrent
		}
		return results, nil

	case []*mq.Link:
		results := make([]interface{}, len(data))
		for i, item := range data {
			oldCurrent := v.context.Current
			v.context.Current = item
			result, err := transform.Accept(v)
			if err != nil {
				return nil, err
			}
			results[i] = result
			v.context.Current = oldCurrent
		}
		return results, nil

	case []*mq.Image:
		results := make([]interface{}, len(data))
		for i, item := range data {
			oldCurrent := v.context.Current
			v.context.Current = item
			result, err := transform.Accept(v)
			if err != nil {
				return nil, err
			}
			results[i] = result
			v.context.Current = oldCurrent
		}
		return results, nil

	case []interface{}:
		results := make([]interface{}, len(data))
		for i, item := range data {
			oldCurrent := v.context.Current
			v.context.Current = item
			result, err := transform.Accept(v)
			if err != nil {
				return nil, err
			}
			results[i] = result
			v.context.Current = oldCurrent
		}
		return results, nil

	default:
		return nil, fmt.Errorf("Error: map can only be applied to collections, got %T\nHint: map works on arrays of items, e.g., .headings | map(.text)", current)
	}
}

func extractTextFromAny(obj interface{}) interface{} {
	// Handle collections
	switch v := obj.(type) {
	case []*mq.Heading:
		results := make([]string, len(v))
		for i, h := range v {
			results[i] = h.Text
		}
		return results
	case []*mq.Section:
		results := make([]string, len(v))
		for i, s := range v {
			results[i] = s.GetText()
		}
		return results
	case []*mq.CodeBlock:
		results := make([]string, len(v))
		for i, c := range v {
			results[i] = c.Content
		}
		return results
	case []*mq.Link:
		results := make([]string, len(v))
		for i, l := range v {
			results[i] = l.Text
		}
		return results
	case []*mq.Image:
		results := make([]string, len(v))
		for i, img := range v {
			results[i] = img.AltText
		}
		return results
	case []interface{}:
		results := make([]string, len(v))
		for i, item := range v {
			results[i] = extractText(item)
		}
		return results
	default:
		// Single item
		return extractText(obj)
	}
}

func getIndex(obj, index interface{}) (interface{}, error) {
	rv := reflect.ValueOf(obj)

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		idx, ok := index.(int)
		if !ok {
			if i64, ok := index.(int64); ok {
				idx = int(i64)
			} else {
				return nil, fmt.Errorf("array index must be integer")
			}
		}

		if idx < 0 || idx >= rv.Len() {
			return nil, fmt.Errorf("index out of range: %d", idx)
		}

		return rv.Index(idx).Interface(), nil

	case reflect.Map:
		key := reflect.ValueOf(index)
		value := rv.MapIndex(key)
		if !value.IsValid() {
			return nil, nil // Key not found
		}
		return value.Interface(), nil

	default:
		return nil, fmt.Errorf("cannot index type %T", obj)
	}
}

func getSlice(obj, start, end interface{}) (interface{}, error) {
	rv := reflect.ValueOf(obj)

	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, fmt.Errorf("cannot slice type %T", obj)
	}

	length := rv.Len()

	// Convert start index
	startIdx := 0
	if start != nil {
		if idx, ok := toInt(start); ok {
			startIdx = idx
			if startIdx < 0 {
				startIdx = 0
			}
		}
	}

	// Convert end index
	endIdx := length
	if end != nil {
		if idx, ok := toInt(end); ok {
			endIdx = idx
			if endIdx > length {
				endIdx = length
			}
		}
	}

	if startIdx > endIdx {
		startIdx = endIdx
	}

	return rv.Slice(startIdx, endIdx).Interface(), nil
}

func toInt(v interface{}) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	default:
		return 0, false
	}
}

// buildSectionTree builds a tree result for a single section.
func buildSectionTree(section *mq.Section, mode mq.TreeMode) *mq.TreeResult {
	result := &mq.TreeResult{
		Path:  section.Heading.Text,
		Lines: section.End - section.Start + 1,
		Mode:  mode,
	}

	node := buildSectionNode(section, mode)
	result.Root = []*mq.TreeNode{node}

	return result
}

// buildSectionNode recursively builds tree nodes from a section.
func buildSectionNode(section *mq.Section, mode mq.TreeMode) *mq.TreeNode {
	node := &mq.TreeNode{
		Type:  "section",
		Text:  section.Heading.Text,
		Start: section.Start,
		End:   section.End,
		Level: section.Heading.Level,
	}

	// Add preview text for preview/full modes
	if mode == mq.TreeModePreview || mode == mq.TreeModeFull {
		node.Preview = mq.ExtractPreview(section.GetText(), 50)
	}

	// Add child sections
	for _, child := range section.Children {
		childNode := buildSectionNode(child, mode)
		node.Children = append(node.Children, childNode)
	}

	// Add special elements (only in default mode)
	if mode == mq.TreeModeDefault {
		codeBlocks := section.GetCodeBlocks()
		if len(codeBlocks) > 0 {
			for lang, count := range mq.CountCodeByLanguage(codeBlocks) {
				meta := fmt.Sprintf("%d block", count)
				if count > 1 {
					meta = fmt.Sprintf("%d blocks", count)
				}
				node.Children = append(node.Children, &mq.TreeNode{
					Type: "code",
					Text: lang,
					Meta: meta,
				})
			}
		}
	}

	return node
}
