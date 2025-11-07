package main

import (
	"fmt"
	"strings"
)

// TokenType represents the type of a token
type TokenType int

const (
	TOKEN_EOF TokenType = iota
	TOKEN_NEWLINE
	TOKEN_LABEL
	TOKEN_INSTRUCTION
	TOKEN_REGISTER
	TOKEN_NUMBER
	TOKEN_HEXNUM
	TOKEN_STRING
	TOKEN_COMMA
	TOKEN_EQUALS
	TOKEN_WHITESPACE
	TOKEN_COMMENT
)

// Token represents a lexical token
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

// Lexer tokenizes CASL2 source code
type Lexer struct {
	input   string
	pos     int
	line    int
	column  int
	lastCol int
}

// NewLexer creates a new lexer for the given input
func NewLexer(input string) *Lexer {
	return &Lexer{
		input:  input,
		pos:    0,
		line:   1,
		column: 1,
	}
}

// peek returns the current character without advancing
func (l *Lexer) peek() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

// peekN returns the character at offset n without advancing
func (l *Lexer) peekN(n int) byte {
	pos := l.pos + n
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
}

// advance moves to the next character
func (l *Lexer) advance() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	ch := l.input[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.lastCol = l.column
		l.column = 1
	} else {
		l.column++
	}
	return ch
}

// isLetter checks if a character is a valid label start character
func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		ch == '$' || ch == '%' || ch == '_' || ch == '.'
}

// isDigit checks if a character is a digit
func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// isHexDigit checks if a character is a hexadecimal digit
func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

// isLabelChar checks if a character can be part of a label
func isLabelChar(ch byte) bool {
	return isLetter(ch) || isDigit(ch)
}

// isWhitespace checks if a character is whitespace (not newline)
func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t'
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() Token {
	ch := l.peek()

	// Skip whitespace but track it
	if isWhitespace(ch) {
		return l.scanWhitespace()
	}

	// Handle newlines
	if ch == '\n' || ch == '\r' {
		return l.scanNewline()
	}

	// Handle EOF
	if ch == 0 {
		return Token{Type: TOKEN_EOF, Line: l.line, Column: l.column}
	}

	// Handle comments
	if ch == ';' {
		return l.scanComment()
	}

	// Handle comma
	if ch == ',' {
		line, col := l.line, l.column
		l.advance()
		return Token{Type: TOKEN_COMMA, Value: ",", Line: line, Column: col}
	}

	// Handle equals
	if ch == '=' {
		line, col := l.line, l.column
		l.advance()
		return Token{Type: TOKEN_EQUALS, Value: "=", Line: line, Column: col}
	}

	// Handle strings
	if ch == '\'' {
		return l.scanString()
	}

	// Handle hex numbers
	if ch == '#' {
		return l.scanHexNumber()
	}

	// Handle numbers (including signed)
	if isDigit(ch) || ((ch == '+' || ch == '-') && isDigit(l.peekN(1))) {
		return l.scanNumber()
	}

	// Handle labels, instructions, and registers
	if isLetter(ch) {
		return l.scanIdentifier()
	}

	// Unknown character - return as error
	line, col := l.line, l.column
	l.advance()
	return Token{
		Type:   TOKEN_EOF,
		Value:  fmt.Sprintf("unexpected character: %c", ch),
		Line:   line,
		Column: col,
	}
}

// scanWhitespace scans whitespace characters
func (l *Lexer) scanWhitespace() Token {
	line, col := l.line, l.column
	start := l.pos
	for isWhitespace(l.peek()) {
		l.advance()
	}
	return Token{
		Type:   TOKEN_WHITESPACE,
		Value:  l.input[start:l.pos],
		Line:   line,
		Column: col,
	}
}

// scanNewline scans newline characters
func (l *Lexer) scanNewline() Token {
	line, col := l.line, l.column
	ch := l.advance()
	// Handle \r\n
	if ch == '\r' && l.peek() == '\n' {
		l.advance()
	}
	return Token{Type: TOKEN_NEWLINE, Value: "\n", Line: line, Column: col}
}

// scanComment scans a comment
func (l *Lexer) scanComment() Token {
	line, col := l.line, l.column
	start := l.pos
	l.advance() // skip ';'
	for l.peek() != '\n' && l.peek() != '\r' && l.peek() != 0 {
		l.advance()
	}
	return Token{
		Type:   TOKEN_COMMENT,
		Value:  l.input[start:l.pos],
		Line:   line,
		Column: col,
	}
}

// scanString scans a string literal
func (l *Lexer) scanString() Token {
	line, col := l.line, l.column
	start := l.pos
	l.advance() // skip opening '

	for {
		ch := l.peek()
		if ch == 0 {
			// Unterminated string
			return Token{
				Type:   TOKEN_STRING,
				Value:  l.input[start:l.pos],
				Line:   line,
				Column: col,
			}
		}
		if ch == '\'' {
			l.advance()
			// Check for escaped quote ''
			if l.peek() == '\'' {
				l.advance()
				continue
			}
			// End of string
			break
		}
		l.advance()
	}

	return Token{
		Type:   TOKEN_STRING,
		Value:  l.input[start:l.pos],
		Line:   line,
		Column: col,
	}
}

// scanHexNumber scans a hexadecimal number
func (l *Lexer) scanHexNumber() Token {
	line, col := l.line, l.column
	start := l.pos
	l.advance() // skip '#'

	for isHexDigit(l.peek()) {
		l.advance()
	}

	return Token{
		Type:   TOKEN_HEXNUM,
		Value:  l.input[start:l.pos],
		Line:   line,
		Column: col,
	}
}

// scanNumber scans a decimal number
func (l *Lexer) scanNumber() Token {
	line, col := l.line, l.column
	start := l.pos

	// Handle sign
	if l.peek() == '+' || l.peek() == '-' {
		l.advance()
	}

	for isDigit(l.peek()) {
		l.advance()
	}

	return Token{
		Type:   TOKEN_NUMBER,
		Value:  l.input[start:l.pos],
		Line:   line,
		Column: col,
	}
}

// scanIdentifier scans an identifier (label, instruction, or register)
func (l *Lexer) scanIdentifier() Token {
	line, col := l.line, l.column
	start := l.pos

	// Check for register (GR0-GR7)
	if l.peek() == 'G' && l.peekN(1) == 'R' {
		next := l.peekN(2)
		if next >= '0' && next <= '7' {
			// Check if followed by a label character
			if !isLabelChar(l.peekN(3)) && l.peekN(3) != 0 && !isWhitespace(l.peekN(3)) && l.peekN(3) != ',' && l.peekN(3) != '\n' && l.peekN(3) != '\r' && l.peekN(3) != ';' {
				// Not a register, continue as identifier
			} else {
				l.advance()
				l.advance()
				l.advance()
				return Token{
					Type:   TOKEN_REGISTER,
					Value:  l.input[start:l.pos],
					Line:   line,
					Column: col,
				}
			}
		}
	}

	// Scan the identifier
	for isLabelChar(l.peek()) {
		l.advance()
	}

	value := l.input[start:l.pos]

	// Always return as LABEL - the parser will determine context
	// This allows labels like "FLUSH" to be used as operands
	return Token{
		Type:   TOKEN_LABEL,
		Value:  value,
		Line:   line,
		Column: col,
	}
}

// ParsedLine represents a parsed line of CASL2 code
type ParsedLine struct {
	Label       string
	Instruction string
	Operands    []string
	Line        int
}

// ParseLine parses a single line using the lexer
func ParseLine(line string, lineNum int) (*ParsedLine, error) {
	lexer := NewLexer(line)
	result := &ParsedLine{Line: lineNum}

	tokens := []Token{}
	hasLeadingWhitespace := false
	firstToken := true
	
	for {
		tok := lexer.NextToken()
		if tok.Type == TOKEN_EOF {
			break
		}
		if tok.Type == TOKEN_COMMENT {
			break
		}
		if tok.Type == TOKEN_NEWLINE {
			break
		}
		if tok.Type == TOKEN_WHITESPACE {
			if firstToken {
				hasLeadingWhitespace = true
			}
			continue
		}
		firstToken = false
		tokens = append(tokens, tok)
	}

	if len(tokens) == 0 {
		return result, nil
	}

	pos := 0

	// If line starts with whitespace, first token is instruction
	// Otherwise, first token could be label or instruction
	if !hasLeadingWhitespace && pos < len(tokens) && tokens[pos].Type == TOKEN_LABEL {
		// Check if this is an instruction by checking CASL2TBL
		if isInstruction(tokens[pos].Value) {
			// It's an instruction (no label)
			result.Instruction = tokens[pos].Value
			pos++
		} else {
			// It's a label
			result.Label = tokens[pos].Value
			pos++
			
			// Next token should be instruction if present
			if pos < len(tokens) && tokens[pos].Type == TOKEN_LABEL {
				if isInstruction(tokens[pos].Value) {
					result.Instruction = tokens[pos].Value
					pos++
				}
			}
		}
	} else if hasLeadingWhitespace && pos < len(tokens) && tokens[pos].Type == TOKEN_LABEL {
		// Leading whitespace means first token must be instruction
		if isInstruction(tokens[pos].Value) {
			result.Instruction = tokens[pos].Value
			pos++
		} else {
			return nil, fmt.Errorf("expected instruction after leading whitespace, got %s", tokens[pos].Value)
		}
	}

	// Parse operands
	for pos < len(tokens) {
		tok := tokens[pos]
		
		// Handle literals (=...)
		if tok.Type == TOKEN_EQUALS {
			if pos+1 >= len(tokens) {
				return nil, fmt.Errorf("expected value after =")
			}
			nextTok := tokens[pos+1]
			var literal string
			if nextTok.Type == TOKEN_NUMBER || nextTok.Type == TOKEN_HEXNUM || nextTok.Type == TOKEN_STRING {
				literal = "=" + nextTok.Value
				pos += 2
			} else if nextTok.Type == TOKEN_LABEL {
				literal = "=" + nextTok.Value
				pos += 2
			} else {
				return nil, fmt.Errorf("invalid literal value")
			}
			result.Operands = append(result.Operands, literal)
		} else if tok.Type == TOKEN_COMMA {
			pos++
		} else if tok.Type == TOKEN_REGISTER || tok.Type == TOKEN_LABEL || 
				  tok.Type == TOKEN_NUMBER || tok.Type == TOKEN_HEXNUM || 
				  tok.Type == TOKEN_STRING {
			result.Operands = append(result.Operands, tok.Value)
			pos++
		} else {
			return nil, fmt.Errorf("unexpected token: %s", tok.Value)
		}
	}

	return result, nil
}

// isInstruction checks if a string is a known CASL2 instruction
func isInstruction(s string) bool {
	_, exists := CASL2TBL[s]
	return exists
}

// Helper functions for checking token types without regex

// IsValidLabel checks if a string is a valid label using character-by-character analysis
func IsValidLabel(s string) bool {
	if len(s) == 0 {
		return false
	}
	if !isLetter(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isLabelChar(s[i]) {
			return false
		}
	}
	return true
}

// IsRegister checks if a string is a register (GR0-GR7)
func IsRegister(s string) bool {
	s = strings.ToUpper(s)
	if len(s) != 3 {
		return false
	}
	if s[0] != 'G' || s[1] != 'R' {
		return false
	}
	return s[2] >= '0' && s[2] <= '7'
}

// CheckRegister parses a register and returns its number (0-7)
func CheckRegister(register string) (int, error) {
	s := strings.ToUpper(register)
	if !IsRegister(s) {
		return 0, fmt.Errorf("Invalid register \"%s\"", register)
	}
	return int(s[2] - '0'), nil
}
