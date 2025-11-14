package compiler

import (
	"fmt"
)

// TokenType represents the type of token
type TokenType int

const (
	// Special tokens
	TOKEN_EOF TokenType = iota
	TOKEN_ILLEGAL

	// Literals
	TOKEN_IDENT   // identifiers: myVar, x, y
	TOKEN_INT     // integer literals: 123, 0x1a
	TOKEN_STRING  // string literals: "hello"
	TOKEN_BOOL    // true, false

	// Keywords
	TOKEN_CONTRACT
	TOKEN_STATE
	TOKEN_INIT
	TOKEN_FN
	TOKEN_PUB
	TOKEN_PRIV
	TOKEN_VIEW
	TOKEN_REQUIRE
	TOKEN_ASSERT
	TOKEN_REVERT
	TOKEN_EMIT
	TOKEN_EVENT
	TOKEN_IF
	TOKEN_ELSE
	TOKEN_MATCH
	TOKEN_LOOP
	TOKEN_WHILE
	TOKEN_IN
	TOKEN_RETURN
	TOKEN_LET
	TOKEN_MUT
	TOKEN_BREAK
	TOKEN_CONTINUE
	TOKEN_SELF
	TOKEN_AS
	TOKEN_MSG
	TOKEN_BLOCK
	TOKEN_TX
	TOKEN_PRECOMPILED
	TOKEN_TRUE
	TOKEN_FALSE
	TOKEN_STRUCT
	TOKEN_ENUM
	TOKEN_INDEXED
	TOKEN_VEC
	TOKEN_MAP

	// Types
	TOKEN_BOOL_TYPE
	TOKEN_U8
	TOKEN_U16
	TOKEN_U32
	TOKEN_U64
	TOKEN_U128
	TOKEN_U256
	TOKEN_I8
	TOKEN_I16
	TOKEN_I32
	TOKEN_I64
	TOKEN_I128
	TOKEN_I256
	TOKEN_ADDRESS
	TOKEN_HASH
	TOKEN_BYTES
	TOKEN_STRING_TYPE

	// Operators
	TOKEN_ASSIGN       // =
	TOKEN_PLUS         // +
	TOKEN_MINUS        // -
	TOKEN_ASTERISK     // *
	TOKEN_SLASH        // /
	TOKEN_PERCENT      // %
	TOKEN_EQ           // ==
	TOKEN_NOT_EQ       // !=
	TOKEN_LT           // <
	TOKEN_GT           // >
	TOKEN_LT_EQ        // <=
	TOKEN_GT_EQ        // >=
	TOKEN_AND          // &&
	TOKEN_OR           // ||
	TOKEN_NOT          // !
	TOKEN_BANG         // ! (same as NOT, alias for consistency)
	TOKEN_PLUS_ASSIGN  // +=
	TOKEN_MINUS_ASSIGN // -=
	TOKEN_ASTERISK_EQ  // *=
	TOKEN_SLASH_EQ     // /=
	TOKEN_PERCENT_EQ   // %=
	TOKEN_PLUS_EQ      // += (alias)
	TOKEN_MINUS_EQ     // -= (alias)
	TOKEN_ARROW        // ->
	TOKEN_RANGE        // ..

	// Delimiters
	TOKEN_COMMA     // ,
	TOKEN_COLON     // :
	TOKEN_SEMICOLON // ;
	TOKEN_LPAREN    // (
	TOKEN_RPAREN    // )
	TOKEN_LBRACE    // {
	TOKEN_RBRACE    // }
	TOKEN_LBRACKET  // [
	TOKEN_RBRACKET  // ]
	TOKEN_DOT       // .
)

// Token represents a lexical token
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

// Lexer performs lexical analysis
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int
	column       int
}

var keywords = map[string]TokenType{
	"contract":    TOKEN_CONTRACT,
	"state":       TOKEN_STATE,
	"init":        TOKEN_INIT,
	"fn":          TOKEN_FN,
	"pub":         TOKEN_PUB,
	"priv":        TOKEN_PRIV,
	"view":        TOKEN_VIEW,
	"require":     TOKEN_REQUIRE,
	"assert":      TOKEN_ASSERT,
	"revert":      TOKEN_REVERT,
	"emit":        TOKEN_EMIT,
	"event":       TOKEN_EVENT,
	"if":          TOKEN_IF,
	"else":        TOKEN_ELSE,
	"match":       TOKEN_MATCH,
	"loop":        TOKEN_LOOP,
	"while":       TOKEN_WHILE,
	"in":          TOKEN_IN,
	"return":      TOKEN_RETURN,
	"let":         TOKEN_LET,
	"mut":         TOKEN_MUT,
	"break":       TOKEN_BREAK,
	"continue":    TOKEN_CONTINUE,
	"as":          TOKEN_AS,
	"self":        TOKEN_SELF,
	"msg":         TOKEN_MSG,
	"block":       TOKEN_BLOCK,
	"tx":          TOKEN_TX,
	"precompiled": TOKEN_PRECOMPILED,
	"true":        TOKEN_TRUE,
	"false":       TOKEN_FALSE,
	"struct":      TOKEN_STRUCT,
	"enum":        TOKEN_ENUM,
	"indexed":     TOKEN_INDEXED,
	"vec":         TOKEN_VEC,
	"map":         TOKEN_MAP,
	"bool":        TOKEN_BOOL_TYPE,
	"u8":          TOKEN_U8,
	"u16":         TOKEN_U16,
	"u32":         TOKEN_U32,
	"u64":         TOKEN_U64,
	"u128":        TOKEN_U128,
	"u256":        TOKEN_U256,
	"i8":          TOKEN_I8,
	"i16":         TOKEN_I16,
	"i32":         TOKEN_I32,
	"i64":         TOKEN_I64,
	"i128":        TOKEN_I128,
	"i256":        TOKEN_I256,
	"address":     TOKEN_ADDRESS,
	"hash":        TOKEN_HASH,
	"bytes":       TOKEN_BYTES,
	"string":      TOKEN_STRING_TYPE,
}

// NewLexer creates a new Lexer
func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar()
	return l
}

// readChar reads the next character
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // ASCII code for NUL
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.column++
}

// peekChar looks ahead at the next character without consuming it
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_EQ, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = Token{Type: TOKEN_ASSIGN, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
		}
	case '+':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_PLUS_ASSIGN, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = Token{Type: TOKEN_PLUS, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
		}
	case '-':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_MINUS_ASSIGN, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_ARROW, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = Token{Type: TOKEN_MINUS, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
		}
	case '*':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_ASTERISK_EQ, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = Token{Type: TOKEN_ASTERISK, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
		}
	case '/':
		// Check for comments
		if l.peekChar() == '/' {
			l.skipLineComment()
			return l.NextToken()
		} else if l.peekChar() == '*' {
			l.skipBlockComment()
			return l.NextToken()
		} else if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_SLASH_EQ, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = Token{Type: TOKEN_SLASH, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
		}
	case '%':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_PERCENT_EQ, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = Token{Type: TOKEN_PERCENT, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_NOT_EQ, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = Token{Type: TOKEN_NOT, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_LT_EQ, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = Token{Type: TOKEN_LT, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_GT_EQ, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = Token{Type: TOKEN_GT, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_AND, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = Token{Type: TOKEN_ILLEGAL, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_OR, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = Token{Type: TOKEN_ILLEGAL, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
		}
	case '.':
		if l.peekChar() == '.' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TOKEN_RANGE, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = Token{Type: TOKEN_DOT, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
		}
	case ',':
		tok = Token{Type: TOKEN_COMMA, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
	case ':':
		tok = Token{Type: TOKEN_COLON, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
	case ';':
		tok = Token{Type: TOKEN_SEMICOLON, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
	case '(':
		tok = Token{Type: TOKEN_LPAREN, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
	case ')':
		tok = Token{Type: TOKEN_RPAREN, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
	case '{':
		tok = Token{Type: TOKEN_LBRACE, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
	case '}':
		tok = Token{Type: TOKEN_RBRACE, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
	case '[':
		tok = Token{Type: TOKEN_LBRACKET, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
	case ']':
		tok = Token{Type: TOKEN_RBRACKET, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
	case '"':
		tok.Type = TOKEN_STRING
		tok.Literal = l.readString()
	case 0:
		tok.Literal = ""
		tok.Type = TOKEN_EOF
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = lookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Type = TOKEN_INT
			tok.Literal = l.readNumber()
			return tok
		} else {
			tok = Token{Type: TOKEN_ILLEGAL, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
		}
	}

	l.readChar()
	return tok
}

// skipWhitespace skips whitespace characters
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		l.readChar()
	}
}

// skipLineComment skips // comments
func (l *Lexer) skipLineComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

// skipBlockComment skips /* */ comments
func (l *Lexer) skipBlockComment() {
	l.readChar() // skip '/'
	l.readChar() // skip '*'

	for {
		if l.ch == 0 {
			break
		}
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar() // skip '*'
			l.readChar() // skip '/'
			break
		}
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		l.readChar()
	}
}

// readIdentifier reads an identifier or keyword
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber reads a number literal
func (l *Lexer) readNumber() string {
	position := l.position

	// Handle hex numbers (0x...)
	if l.ch == '0' && (l.peekChar() == 'x' || l.peekChar() == 'X') {
		l.readChar() // skip '0'
		l.readChar() // skip 'x'
		for isHexDigit(l.ch) {
			l.readChar()
		}
		return l.input[position:l.position]
	}

	// Handle decimal numbers
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readString reads a string literal
func (l *Lexer) readString() string {
	position := l.position + 1 // skip opening "
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
		if l.ch == '\\' && l.peekChar() == '"' {
			l.readChar() // skip backslash
		}
	}
	return l.input[position:l.position]
}

// isLetter checks if a character is a letter or underscore
func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

// isDigit checks if a character is a digit
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// isHexDigit checks if a character is a hexadecimal digit
func isHexDigit(ch byte) bool {
	return ('0' <= ch && ch <= '9') || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}

// lookupIdent checks if an identifier is a keyword
func lookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return TOKEN_IDENT
}

// TokenTypeString returns a string representation of a token type
func TokenTypeString(t TokenType) string {
	names := map[TokenType]string{
		TOKEN_EOF:          "EOF",
		TOKEN_ILLEGAL:      "ILLEGAL",
		TOKEN_IDENT:        "IDENT",
		TOKEN_INT:          "INT",
		TOKEN_STRING:       "STRING",
		TOKEN_BOOL:         "BOOL",
		TOKEN_CONTRACT:     "contract",
		TOKEN_STATE:        "state",
		TOKEN_INIT:         "init",
		TOKEN_FN:           "fn",
		TOKEN_PUB:          "pub",
		TOKEN_VIEW:         "view",
		TOKEN_REQUIRE:      "require",
		TOKEN_ASSERT:       "assert",
		TOKEN_REVERT:       "revert",
		TOKEN_EMIT:         "emit",
		TOKEN_EVENT:        "event",
		TOKEN_IF:           "if",
		TOKEN_ELSE:         "else",
		TOKEN_MATCH:        "match",
		TOKEN_LOOP:         "loop",
		TOKEN_WHILE:        "while",
		TOKEN_IN:           "in",
		TOKEN_RETURN:       "return",
		TOKEN_LET:          "let",
		TOKEN_MUT:          "mut",
		TOKEN_SELF:         "self",
		TOKEN_MSG:          "msg",
		TOKEN_BLOCK:        "block",
		TOKEN_TX:           "tx",
		TOKEN_PRECOMPILED:  "precompiled",
		TOKEN_TRUE:         "true",
		TOKEN_FALSE:        "false",
		TOKEN_ASSIGN:       "=",
		TOKEN_PLUS:         "+",
		TOKEN_MINUS:        "-",
		TOKEN_ASTERISK:     "*",
		TOKEN_SLASH:        "/",
		TOKEN_EQ:           "==",
		TOKEN_NOT_EQ:       "!=",
		TOKEN_LT:           "<",
		TOKEN_GT:           ">",
		TOKEN_AND:          "&&",
		TOKEN_OR:           "||",
		TOKEN_NOT:          "!",
		TOKEN_ARROW:        "->",
		TOKEN_COMMA:        ",",
		TOKEN_COLON:        ":",
		TOKEN_SEMICOLON:    ";",
		TOKEN_LPAREN:       "(",
		TOKEN_RPAREN:       ")",
		TOKEN_LBRACE:       "{",
		TOKEN_RBRACE:       "}",
		TOKEN_LBRACKET:     "[",
		TOKEN_RBRACKET:     "]",
		TOKEN_DOT:          ".",
	}
	if name, ok := names[t]; ok {
		return name
	}
	return fmt.Sprintf("TokenType(%d)", t)
}

// String implements fmt.Stringer for Token
func (t Token) String() string {
	return fmt.Sprintf("%s(%q) at %d:%d", TokenTypeString(t.Type), t.Literal, t.Line, t.Column)
}
