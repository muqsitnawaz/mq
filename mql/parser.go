package mql

import (
	"fmt"
	"strconv"
)

// Parser parses MQL query strings into AST.
type Parser struct {
	tokens []Token
	pos    int
}

// NewParser creates a new parser from tokens.
func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens: tokens,
		pos:    0,
	}
}

// Parse parses tokens into a query AST.
func Parse(tokens []Token) (QueryNode, error) {
	p := NewParser(tokens)
	return p.Parse()
}

// ParseString parses a query string directly.
func ParseString(query string) (QueryNode, error) {
	tokens, err := Lex(query)
	if err != nil {
		return nil, fmt.Errorf("lexing failed: %w", err)
	}

	return Parse(tokens)
}

// Parse parses the tokens into an AST.
func (p *Parser) Parse() (QueryNode, error) {
	ast, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Ensure we've consumed all tokens except EOF
	if p.current().Type != TokenEOF {
		return nil, p.error("unexpected token: %s", p.current())
	}

	return ast, nil
}

// parseExpression parses a full expression (handles pipes).
func (p *Parser) parseExpression() (QueryNode, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	// Handle pipe operations
	for p.current().Type == TokenPipe {
		p.advance() // consume pipe

		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}

		left = NewPipe(left, right)
	}

	return left, nil
}

// parsePrimary parses a primary expression.
func (p *Parser) parsePrimary() (QueryNode, error) {
	token := p.current()

	switch token.Type {
	case TokenDot:
		return p.parseSelector()

	case TokenIdentifier:
		// Check if it's a function call or selector
		if p.peek().Type == TokenLParen {
			return p.parseFunction()
		}
		// Standalone identifier (for use in predicates)
		p.advance()
		return NewIdentifier(token.Value), nil

	case TokenLParen:
		// Grouped expression
		p.advance() // consume (
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		return expr, nil

	case TokenString:
		p.advance()
		return NewLiteral(token.Value, LiteralString), nil

	case TokenNumber:
		p.advance()
		num, err := p.parseNumber(token.Value)
		if err != nil {
			return nil, err
		}
		return NewLiteral(num, LiteralNumber), nil

	default:
		return nil, p.error("unexpected token in primary expression: %s", token)
	}
}

// parseSelector parses a selector expression (.headings, .code, etc).
func (p *Parser) parseSelector() (QueryNode, error) {
	if err := p.expect(TokenDot); err != nil {
		return nil, err
	}

	if p.current().Type != TokenIdentifier {
		return nil, p.error("expected identifier after '.', got %s", p.current())
	}

	name := p.current().Value
	p.advance()

	// Check for arguments
	var args []QueryNode
	if p.current().Type == TokenLParen {
		var err error
		args, err = p.parseArguments()
		if err != nil {
			return nil, err
		}
	}

	// Check for special selectors that need special handling
	switch name {
	case "select", "filter":
		// These require a predicate
		if len(args) == 0 {
			return nil, p.error("%s requires a predicate argument", name)
		}
		return NewFilter(args[0]), nil

	case "map":
		if len(args) == 0 {
			return nil, p.error("map requires a transformation argument")
		}
		return NewFunction("map", args...), nil

	default:
		// Regular selector
		return NewSelector(name, args...), nil
	}
}

// parseFunction parses a function call.
func (p *Parser) parseFunction() (QueryNode, error) {
	if p.current().Type != TokenIdentifier {
		return nil, p.error("expected function name, got %s", p.current())
	}

	name := p.current().Value
	p.advance()

	args, err := p.parseArguments()
	if err != nil {
		return nil, err
	}

	// Special handling for certain functions
	switch name {
	case "select", "filter":
		if len(args) == 0 {
			return nil, p.error("%s requires a predicate argument", name)
		}
		return NewFilter(args[0]), nil

	default:
		return NewFunction(name, args...), nil
	}
}

// parseArguments parses function arguments.
func (p *Parser) parseArguments() ([]QueryNode, error) {
	if err := p.expect(TokenLParen); err != nil {
		return nil, err
	}

	var args []QueryNode

	// Handle empty argument list
	if p.current().Type == TokenRParen {
		p.advance()
		return args, nil
	}

	// Parse arguments
	for {
		arg, err := p.parseArgument()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)

		if p.current().Type == TokenComma {
			p.advance() // consume comma
			continue
		}

		if p.current().Type == TokenRParen {
			p.advance() // consume )
			break
		}

		return nil, p.error("expected ',' or ')' in argument list, got %s", p.current())
	}

	return args, nil
}

// parseArgument parses a single argument (could be expression or predicate).
func (p *Parser) parseArgument() (QueryNode, error) {
	// Try to parse as a comparison/predicate first
	return p.parseComparison()
}

// parseComparison parses comparison expressions.
func (p *Parser) parseComparison() (QueryNode, error) {
	left, err := p.parseLogical()
	if err != nil {
		return nil, err
	}

	// Check for comparison operators
	token := p.current()
	switch token.Type {
	case TokenEquals, TokenNotEquals, TokenLessThan, TokenLessEqual, TokenGreaterThan, TokenGreaterEqual:
		p.advance()
		right, err := p.parseLogical()
		if err != nil {
			return nil, err
		}
		return NewBinary(left, token.Value, right), nil
	}

	return left, nil
}

// parseLogical parses logical operations (and/or).
func (p *Parser) parseLogical() (QueryNode, error) {
	left, err := p.parseProperty()
	if err != nil {
		return nil, err
	}

	for {
		token := p.current()
		if token.Type == TokenAnd || token.Type == TokenOr {
			p.advance()
			right, err := p.parseProperty()
			if err != nil {
				return nil, err
			}
			left = NewBinary(left, token.Value, right)
		} else {
			break
		}
	}

	return left, nil
}

// parseProperty parses property access and literals.
func (p *Parser) parseProperty() (QueryNode, error) {
	token := p.current()

	switch token.Type {
	case TokenDot:
		// Property access starting with dot
		p.advance()
		if p.current().Type != TokenIdentifier {
			return nil, p.error("expected property name after '.', got %s", p.current())
		}
		name := p.current().Value
		p.advance()

		// Check for further property access or function call
		node := QueryNode(NewIdentifier(name))

		// Handle array/object indexing
		for p.current().Type == TokenLBracket {
			node, _ = p.parseIndex(node)
		}

		// Handle function calls on properties
		if p.current().Type == TokenLParen {
			args, err := p.parseArguments()
			if err != nil {
				return nil, err
			}
			return NewFunction(name, args...), nil
		}

		return node, nil

	case TokenIdentifier:
		// Simple identifier
		p.advance()
		node := QueryNode(NewIdentifier(token.Value))

		// Handle array/object indexing
		for p.current().Type == TokenLBracket {
			node, _ = p.parseIndex(node)
		}

		// Handle function call
		if p.current().Type == TokenLParen {
			args, err := p.parseArguments()
			if err != nil {
				return nil, err
			}
			return NewFunction(token.Value, args...), nil
		}

		return node, nil

	case TokenString:
		p.advance()
		return NewLiteral(token.Value, LiteralString), nil

	case TokenNumber:
		p.advance()
		num, err := p.parseNumber(token.Value)
		if err != nil {
			return nil, err
		}
		return NewLiteral(num, LiteralNumber), nil

	case TokenLParen:
		// Grouped expression
		p.advance()
		expr, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		if err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		return expr, nil

	default:
		return nil, p.error("unexpected token in property: %s", token)
	}
}

// parseIndex parses array/object indexing.
func (p *Parser) parseIndex(object QueryNode) (QueryNode, error) {
	if err := p.expect(TokenLBracket); err != nil {
		return nil, err
	}

	// Check for slice notation
	if p.current().Type == TokenColon {
		// [:end]
		p.advance() // consume :
		end, err := p.parseProperty()
		if err != nil {
			return nil, err
		}
		if err := p.expect(TokenRBracket); err != nil {
			return nil, err
		}
		return NewSlice(object, nil, end), nil
	}

	// Parse start index/key
	start, err := p.parseProperty()
	if err != nil {
		return nil, err
	}

	// Check for slice or simple index
	if p.current().Type == TokenColon {
		// [start:] or [start:end]
		p.advance() // consume :

		if p.current().Type == TokenRBracket {
			// [start:]
			p.advance()
			return NewSlice(object, start, nil), nil
		}

		// [start:end]
		end, err := p.parseProperty()
		if err != nil {
			return nil, err
		}
		if err := p.expect(TokenRBracket); err != nil {
			return nil, err
		}
		return NewSlice(object, start, end), nil
	}

	// Simple index
	if err := p.expect(TokenRBracket); err != nil {
		return nil, err
	}
	return NewIndex(object, start), nil
}

// parseNumber parses a number from string.
func (p *Parser) parseNumber(s string) (interface{}, error) {
	// Try integer first
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i, nil
	}

	// Try float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}

	return nil, fmt.Errorf("invalid number: %s", s)
}

// Helper methods

// current returns the current token.
func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

// peek returns the next token without advancing.
func (p *Parser) peek() Token {
	if p.pos+1 >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos+1]
}

// advance moves to the next token.
func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

// expect consumes and validates a specific token type.
func (p *Parser) expect(typ TokenType) error {
	if p.current().Type != typ {
		return p.error("expected %v, got %s", typ, p.current())
	}
	p.advance()
	return nil
}

// error creates a parser error with context.
func (p *Parser) error(format string, args ...interface{}) error {
	token := p.current()
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("parse error at line %d, column %d: %s", token.Line, token.Col, msg)
}
