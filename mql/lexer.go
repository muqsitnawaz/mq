package mql

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType represents the type of a lexical token.
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenDot
	TokenPipe
	TokenLParen
	TokenRParen
	TokenLBracket
	TokenRBracket
	TokenLBrace
	TokenRBrace
	TokenComma
	TokenColon
	TokenString
	TokenNumber
	TokenIdentifier
	TokenEquals
	TokenNotEquals
	TokenLessThan
	TokenLessEqual
	TokenGreaterThan
	TokenGreaterEqual
	TokenAnd
	TokenOr
)

// Token represents a lexical token.
type Token struct {
	Type  TokenType
	Value string
	Pos   int
	Line  int
	Col   int
}

// String returns a string representation of the token.
func (t Token) String() string {
	switch t.Type {
	case TokenEOF:
		return "EOF"
	case TokenDot:
		return "."
	case TokenPipe:
		return "|"
	case TokenString:
		return fmt.Sprintf("STRING(%s)", t.Value)
	case TokenNumber:
		return fmt.Sprintf("NUMBER(%s)", t.Value)
	case TokenIdentifier:
		return fmt.Sprintf("ID(%s)", t.Value)
	default:
		return fmt.Sprintf("TOKEN(%d, %s)", t.Type, t.Value)
	}
}

// Lexer tokenizes MQL query strings.
type Lexer struct {
	input  string
	pos    int    // current position in input
	start  int    // start of current token
	line   int    // current line number
	col    int    // current column number
	tokens []Token // accumulated tokens
}

// NewLexer creates a new lexer for the given input.
func NewLexer(input string) *Lexer {
	return &Lexer{
		input:  input,
		line:   1,
		col:    1,
		tokens: []Token{},
	}
}

// Lex tokenizes the entire input and returns all tokens.
func Lex(input string) ([]Token, error) {
	l := NewLexer(input)
	return l.Tokenize()
}

// Tokenize tokenizes the entire input.
func (l *Lexer) Tokenize() ([]Token, error) {
	for {
		token, err := l.NextToken()
		if err != nil {
			return nil, err
		}

		l.tokens = append(l.tokens, token)

		if token.Type == TokenEOF {
			break
		}
	}

	return l.tokens, nil
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() (Token, error) {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return l.makeToken(TokenEOF, ""), nil
	}

	l.start = l.pos
	ch := l.peek()

	switch ch {
	case '.':
		l.advance()
		return l.makeToken(TokenDot, "."), nil

	case '|':
		l.advance()
		return l.makeToken(TokenPipe, "|"), nil

	case '(':
		l.advance()
		return l.makeToken(TokenLParen, "("), nil

	case ')':
		l.advance()
		return l.makeToken(TokenRParen, ")"), nil

	case '[':
		l.advance()
		return l.makeToken(TokenLBracket, "["), nil

	case ']':
		l.advance()
		return l.makeToken(TokenRBracket, "]"), nil

	case '{':
		l.advance()
		return l.makeToken(TokenLBrace, "{"), nil

	case '}':
		l.advance()
		return l.makeToken(TokenRBrace, "}"), nil

	case ',':
		l.advance()
		return l.makeToken(TokenComma, ","), nil

	case ':':
		l.advance()
		return l.makeToken(TokenColon, ":"), nil

	case '=':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(TokenEquals, "=="), nil
		}
		return Token{}, l.error("unexpected character '=', did you mean '=='?")

	case '!':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(TokenNotEquals, "!="), nil
		}
		return Token{}, l.error("unexpected character '!'")

	case '<':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(TokenLessEqual, "<="), nil
		}
		return l.makeToken(TokenLessThan, "<"), nil

	case '>':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(TokenGreaterEqual, ">="), nil
		}
		return l.makeToken(TokenGreaterThan, ">"), nil

	case '"', '\'':
		return l.scanString(ch)

	case '/':
		// Check for regex pattern
		if l.isRegexContext() {
			return l.scanRegex()
		}
		return Token{}, l.error("unexpected character '/'")

	default:
		if unicode.IsLetter(rune(ch)) || ch == '_' {
			return l.scanIdentifier()
		}
		if unicode.IsDigit(rune(ch)) {
			return l.scanNumber()
		}
		if ch == '-' && l.pos+1 < len(l.input) && unicode.IsDigit(rune(l.input[l.pos+1])) {
			return l.scanNumber()
		}
	}

	return Token{}, l.error(fmt.Sprintf("unexpected character '%c'", ch))
}

// skipWhitespace skips whitespace and updates line/col positions.
func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == ' ' || ch == '\t' || ch == '\r' {
			l.pos++
			l.col++
		} else if ch == '\n' {
			l.pos++
			l.line++
			l.col = 1
		} else {
			break
		}
	}
}

// peek returns the current character without advancing.
func (l *Lexer) peek() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

// peekNext returns the next character without advancing.
func (l *Lexer) peekNext() byte {
	if l.pos+1 >= len(l.input) {
		return 0
	}
	return l.input[l.pos+1]
}

// advance moves to the next character.
func (l *Lexer) advance() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	ch := l.input[l.pos]
	l.pos++
	l.col++
	return ch
}

// scanString scans a string literal.
func (l *Lexer) scanString(quote byte) (Token, error) {
	l.advance() // skip opening quote
	start := l.pos
	_ = start

	var value strings.Builder
	escaped := false

	for l.pos < len(l.input) {
		ch := l.peek()

		if escaped {
			// Handle escape sequences
			switch ch {
			case 'n':
				value.WriteByte('\n')
			case 't':
				value.WriteByte('\t')
			case 'r':
				value.WriteByte('\r')
			case '\\':
				value.WriteByte('\\')
			case quote:
				value.WriteByte(quote)
			default:
				value.WriteByte(ch)
			}
			l.advance()
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			l.advance()
			continue
		}

		if ch == quote {
			l.advance() // skip closing quote
			return l.makeToken(TokenString, value.String()), nil
		}

		if ch == '\n' {
			return Token{}, l.error("unterminated string (newline in string)")
		}

		value.WriteByte(ch)
		l.advance()
	}

	return Token{}, l.error("unterminated string")
}

// scanIdentifier scans an identifier or keyword.
func (l *Lexer) scanIdentifier() (Token, error) {
	start := l.pos

	for l.pos < len(l.input) {
		ch := l.peek()
		if unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '_' {
			l.advance()
		} else {
			break
		}
	}

	value := l.input[start:l.pos]

	// Check for keywords
	switch value {
	case "and":
		return l.makeToken(TokenAnd, value), nil
	case "or":
		return l.makeToken(TokenOr, value), nil
	case "select", "map", "filter", "headings", "section", "code",
		"links", "images", "tables", "lists", "owner", "metadata",
		"text", "markdown", "html", "json", "yaml", "length",
		"contains", "startswith", "endswith", "level", "language":
		// These are all valid identifiers
		return l.makeToken(TokenIdentifier, value), nil
	default:
		return l.makeToken(TokenIdentifier, value), nil
	}
}

// scanNumber scans a number literal.
func (l *Lexer) scanNumber() (Token, error) {
	start := l.pos

	// Handle negative numbers
	if l.peek() == '-' {
		l.advance()
	}

	// Scan integer part
	for l.pos < len(l.input) && unicode.IsDigit(rune(l.peek())) {
		l.advance()
	}

	// Check for decimal point
	if l.pos+1 < len(l.input) && l.peek() == '.' && unicode.IsDigit(rune(l.input[l.pos+1])) {
		l.advance() // skip '.'
		for l.pos < len(l.input) && unicode.IsDigit(rune(l.peek())) {
			l.advance()
		}
	}

	value := l.input[start:l.pos]
	return l.makeToken(TokenNumber, value), nil
}

// scanRegex scans a regex pattern.
func (l *Lexer) scanRegex() (Token, error) {
	l.advance() // skip '/'
	_ = l.pos

	var value strings.Builder
	escaped := false

	for l.pos < len(l.input) {
		ch := l.peek()

		if escaped {
			value.WriteByte(ch)
			l.advance()
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			value.WriteByte(ch)
			l.advance()
			continue
		}

		if ch == '/' {
			l.advance() // skip closing '/'
			return l.makeToken(TokenString, value.String()), nil // Treat regex as string
		}

		if ch == '\n' {
			return Token{}, l.error("unterminated regex pattern")
		}

		value.WriteByte(ch)
		l.advance()
	}

	return Token{}, l.error("unterminated regex pattern")
}

// isRegexContext checks if we're in a context where a regex is expected.
func (l *Lexer) isRegexContext() bool {
	// Look back at previous tokens to determine context
	// For now, assume regex after certain identifiers
	for i := len(l.tokens) - 1; i >= 0; i-- {
		token := l.tokens[i]
		if token.Type == TokenIdentifier {
			switch token.Value {
			case "heading", "section", "contains", "match":
				return true
			}
		}
		if token.Type != TokenDot && token.Type != TokenPipe {
			break
		}
	}
	return false
}

// makeToken creates a token with current position info.
func (l *Lexer) makeToken(typ TokenType, value string) Token {
	return Token{
		Type:  typ,
		Value: value,
		Pos:   l.start,
		Line:  l.line,
		Col:   l.col - (l.pos - l.start),
	}
}

// error creates a lexer error with position information.
func (l *Lexer) error(msg string) error {
	return fmt.Errorf("lexer error at line %d, column %d: %s", l.line, l.col, msg)
}