package mql

import (
	"fmt"
	"strings"
)

// QueryNode represents a node in the query AST.
type QueryNode interface {
	String() string
	Accept(visitor Visitor) (interface{}, error)
}

// Visitor is the visitor pattern interface for AST traversal.
type Visitor interface {
	VisitPipe(*PipeNode) (interface{}, error)
	VisitSelector(*SelectorNode) (interface{}, error)
	VisitFilter(*FilterNode) (interface{}, error)
	VisitFunction(*FunctionNode) (interface{}, error)
	VisitBinary(*BinaryNode) (interface{}, error)
	VisitUnary(*UnaryNode) (interface{}, error)
	VisitLiteral(*LiteralNode) (interface{}, error)
	VisitIdentifier(*IdentifierNode) (interface{}, error)
	VisitIndex(*IndexNode) (interface{}, error)
	VisitSlice(*SliceNode) (interface{}, error)
}

// PipeNode represents a pipe operation (|).
type PipeNode struct {
	Left  QueryNode
	Right QueryNode
}

func (n *PipeNode) String() string {
	return fmt.Sprintf("%s | %s", n.Left, n.Right)
}

func (n *PipeNode) Accept(v Visitor) (interface{}, error) {
	return v.VisitPipe(n)
}

// SelectorNode represents a selector operation (.headings, .code, etc).
type SelectorNode struct {
	Name string
	Args []QueryNode
}

func (n *SelectorNode) String() string {
	if len(n.Args) == 0 {
		return fmt.Sprintf(".%s", n.Name)
	}

	args := make([]string, len(n.Args))
	for i, arg := range n.Args {
		args[i] = arg.String()
	}
	return fmt.Sprintf(".%s(%s)", n.Name, strings.Join(args, ", "))
}

func (n *SelectorNode) Accept(v Visitor) (interface{}, error) {
	return v.VisitSelector(n)
}

// FilterNode represents a filter operation with a predicate.
type FilterNode struct {
	Predicate QueryNode
}

func (n *FilterNode) String() string {
	return fmt.Sprintf("select(%s)", n.Predicate)
}

func (n *FilterNode) Accept(v Visitor) (interface{}, error) {
	return v.VisitFilter(n)
}

// FunctionNode represents a function call.
type FunctionNode struct {
	Name string
	Args []QueryNode
}

func (n *FunctionNode) String() string {
	args := make([]string, len(n.Args))
	for i, arg := range n.Args {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", n.Name, strings.Join(args, ", "))
}

func (n *FunctionNode) Accept(v Visitor) (interface{}, error) {
	return v.VisitFunction(n)
}

// BinaryNode represents a binary operation.
type BinaryNode struct {
	Left     QueryNode
	Operator string
	Right    QueryNode
}

func (n *BinaryNode) String() string {
	return fmt.Sprintf("(%s %s %s)", n.Left, n.Operator, n.Right)
}

func (n *BinaryNode) Accept(v Visitor) (interface{}, error) {
	return v.VisitBinary(n)
}

// UnaryNode represents a unary operation.
type UnaryNode struct {
	Operator string
	Operand  QueryNode
}

func (n *UnaryNode) String() string {
	return fmt.Sprintf("%s%s", n.Operator, n.Operand)
}

func (n *UnaryNode) Accept(v Visitor) (interface{}, error) {
	return v.VisitUnary(n)
}

// LiteralNode represents a literal value (string, number, boolean).
type LiteralNode struct {
	Value interface{}
	Type  LiteralType
}

// LiteralType represents the type of a literal.
type LiteralType int

const (
	LiteralString LiteralType = iota
	LiteralNumber
	LiteralBoolean
	LiteralNull
)

func (n *LiteralNode) String() string {
	switch n.Type {
	case LiteralString:
		return fmt.Sprintf("%q", n.Value)
	case LiteralNull:
		return "null"
	default:
		return fmt.Sprintf("%v", n.Value)
	}
}

func (n *LiteralNode) Accept(v Visitor) (interface{}, error) {
	return v.VisitLiteral(n)
}

// IdentifierNode represents an identifier.
type IdentifierNode struct {
	Name string
}

func (n *IdentifierNode) String() string {
	return n.Name
}

func (n *IdentifierNode) Accept(v Visitor) (interface{}, error) {
	return v.VisitIdentifier(n)
}

// IndexNode represents array/object indexing (e.g., [0] or ["key"]).
type IndexNode struct {
	Object QueryNode
	Index  QueryNode
}

func (n *IndexNode) String() string {
	return fmt.Sprintf("%s[%s]", n.Object, n.Index)
}

func (n *IndexNode) Accept(v Visitor) (interface{}, error) {
	return v.VisitIndex(n)
}

// SliceNode represents array slicing (e.g., [1:3]).
type SliceNode struct {
	Object QueryNode
	Start  QueryNode // can be nil
	End    QueryNode // can be nil
}

func (n *SliceNode) String() string {
	start := ""
	end := ""

	if n.Start != nil {
		start = n.Start.String()
	}
	if n.End != nil {
		end = n.End.String()
	}

	return fmt.Sprintf("%s[%s:%s]", n.Object, start, end)
}

func (n *SliceNode) Accept(v Visitor) (interface{}, error) {
	return v.VisitSlice(n)
}

// Helper functions for creating AST nodes

// NewPipe creates a new pipe node.
func NewPipe(left, right QueryNode) *PipeNode {
	return &PipeNode{Left: left, Right: right}
}

// NewSelector creates a new selector node.
func NewSelector(name string, args ...QueryNode) *SelectorNode {
	return &SelectorNode{Name: name, Args: args}
}

// NewFilter creates a new filter node.
func NewFilter(predicate QueryNode) *FilterNode {
	return &FilterNode{Predicate: predicate}
}

// NewFunction creates a new function node.
func NewFunction(name string, args ...QueryNode) *FunctionNode {
	return &FunctionNode{Name: name, Args: args}
}

// NewBinary creates a new binary operation node.
func NewBinary(left QueryNode, op string, right QueryNode) *BinaryNode {
	return &BinaryNode{Left: left, Operator: op, Right: right}
}

// NewUnary creates a new unary operation node.
func NewUnary(op string, operand QueryNode) *UnaryNode {
	return &UnaryNode{Operator: op, Operand: operand}
}

// NewLiteral creates a new literal node.
func NewLiteral(value interface{}, typ LiteralType) *LiteralNode {
	return &LiteralNode{Value: value, Type: typ}
}

// NewIdentifier creates a new identifier node.
func NewIdentifier(name string) *IdentifierNode {
	return &IdentifierNode{Name: name}
}

// NewIndex creates a new index node.
func NewIndex(object, index QueryNode) *IndexNode {
	return &IndexNode{Object: object, Index: index}
}

// NewSlice creates a new slice node.
func NewSlice(object, start, end QueryNode) *SliceNode {
	return &SliceNode{Object: object, Start: start, End: end}
}