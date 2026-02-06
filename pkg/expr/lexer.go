package expr

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Lexer tokenizes a GCW expression string.
type Lexer struct {
	input  string
	pos    int
	tokens []Token
}

// NewLexer creates a new lexer for the given input.
func NewLexer(input string) *Lexer {
	return &Lexer{input: input}
}

// Tokenize scans the entire input and returns all tokens.
func (l *Lexer) Tokenize() ([]Token, error) {
	for {
		tok, err := l.next()
		if err != nil {
			return nil, err
		}
		l.tokens = append(l.tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return l.tokens, nil
}

// next returns the next token from the input.
func (l *Lexer) next() (Token, error) {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, Pos: l.pos}, nil
	}

	ch := l.input[l.pos]

	// String literals
	if ch == '"' || ch == '\'' {
		return l.readString(ch)
	}

	// Number literals
	if ch >= '0' && ch <= '9' {
		return l.readNumber()
	}

	// Negative numbers (unary minus handled in parser, but we recognize negative literals)
	// This is handled by the parser as unary minus

	// Two-character operators
	if l.pos+1 < len(l.input) {
		two := l.input[l.pos : l.pos+2]
		switch two {
		case "//":
			l.pos += 2
			return Token{Type: TokenIntDiv, Value: "//", Pos: l.pos - 2}, nil
		case "==":
			l.pos += 2
			return Token{Type: TokenEq, Value: "==", Pos: l.pos - 2}, nil
		case "!=":
			l.pos += 2
			return Token{Type: TokenNeq, Value: "!=", Pos: l.pos - 2}, nil
		case "<=":
			l.pos += 2
			return Token{Type: TokenLte, Value: "<=", Pos: l.pos - 2}, nil
		case ">=":
			l.pos += 2
			return Token{Type: TokenGte, Value: ">=", Pos: l.pos - 2}, nil
		}
	}

	// Single-character operators
	switch ch {
	case '+':
		l.pos++
		return Token{Type: TokenPlus, Value: "+", Pos: l.pos - 1}, nil
	case '-':
		l.pos++
		return Token{Type: TokenMinus, Value: "-", Pos: l.pos - 1}, nil
	case '*':
		l.pos++
		return Token{Type: TokenStar, Value: "*", Pos: l.pos - 1}, nil
	case '/':
		l.pos++
		return Token{Type: TokenSlash, Value: "/", Pos: l.pos - 1}, nil
	case '%':
		l.pos++
		return Token{Type: TokenPercent, Value: "%", Pos: l.pos - 1}, nil
	case '<':
		l.pos++
		return Token{Type: TokenLt, Value: "<", Pos: l.pos - 1}, nil
	case '>':
		l.pos++
		return Token{Type: TokenGt, Value: ">", Pos: l.pos - 1}, nil
	case '(':
		l.pos++
		return Token{Type: TokenLParen, Value: "(", Pos: l.pos - 1}, nil
	case ')':
		l.pos++
		return Token{Type: TokenRParen, Value: ")", Pos: l.pos - 1}, nil
	case '[':
		l.pos++
		return Token{Type: TokenLBracket, Value: "[", Pos: l.pos - 1}, nil
	case ']':
		l.pos++
		return Token{Type: TokenRBracket, Value: "]", Pos: l.pos - 1}, nil
	case '.':
		l.pos++
		return Token{Type: TokenDot, Value: ".", Pos: l.pos - 1}, nil
	case ',':
		l.pos++
		return Token{Type: TokenComma, Value: ",", Pos: l.pos - 1}, nil
	case '{':
		l.pos++
		return Token{Type: TokenLBrace, Value: "{", Pos: l.pos - 1}, nil
	case '}':
		l.pos++
		return Token{Type: TokenRBrace, Value: "}", Pos: l.pos - 1}, nil
	case ':':
		l.pos++
		return Token{Type: TokenColon, Value: ":", Pos: l.pos - 1}, nil
	}

	// Identifiers and keywords
	if isIdentStart(ch) {
		return l.readIdentifier()
	}

	return Token{}, fmt.Errorf("unexpected character %q at position %d", string(ch), l.pos)
}

// readString reads a quoted string literal.
func (l *Lexer) readString(quote byte) (Token, error) {
	start := l.pos
	l.pos++ // skip opening quote

	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == '\\' && l.pos+1 < len(l.input) {
			l.pos++
			escaped := l.input[l.pos]
			switch escaped {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			case '\'':
				sb.WriteByte('\'')
			default:
				sb.WriteByte('\\')
				sb.WriteByte(escaped)
			}
			l.pos++
			continue
		}
		if ch == quote {
			l.pos++ // skip closing quote
			return Token{
				Type:   TokenString,
				Value:  l.input[start:l.pos],
				StrVal: sb.String(),
				Pos:    start,
			}, nil
		}
		sb.WriteByte(ch)
		l.pos++
	}

	return Token{}, fmt.Errorf("unterminated string starting at position %d", start)
}

// readNumber reads an integer or float literal.
func (l *Lexer) readNumber() (Token, error) {
	start := l.pos
	isFloat := false

	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch >= '0' && ch <= '9' {
			l.pos++
		} else if ch == '.' && !isFloat {
			// Check it's not a method call like 123.toString
			if l.pos+1 < len(l.input) && l.input[l.pos+1] >= '0' && l.input[l.pos+1] <= '9' {
				isFloat = true
				l.pos++
			} else {
				break
			}
		} else if ch == 'e' || ch == 'E' {
			isFloat = true
			l.pos++
			if l.pos < len(l.input) && (l.input[l.pos] == '+' || l.input[l.pos] == '-') {
				l.pos++
			}
		} else {
			break
		}
	}

	raw := l.input[start:l.pos]
	if isFloat {
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return Token{}, fmt.Errorf("invalid float %q at position %d", raw, start)
		}
		return Token{Type: TokenFloat, Value: raw, FloatVal: f, Pos: start}, nil
	}

	i, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return Token{}, fmt.Errorf("invalid integer %q at position %d", raw, start)
	}
	return Token{Type: TokenInt, Value: raw, IntVal: i, Pos: start}, nil
}

// readIdentifier reads an identifier or keyword.
func (l *Lexer) readIdentifier() (Token, error) {
	start := l.pos
	for l.pos < len(l.input) && isIdentPart(l.input[l.pos]) {
		l.pos++
	}

	word := l.input[start:l.pos]
	switch word {
	case "true", "True", "TRUE":
		return Token{Type: TokenTrue, Value: word, Pos: start}, nil
	case "false", "False", "FALSE":
		return Token{Type: TokenFalse, Value: word, Pos: start}, nil
	case "null", "None":
		return Token{Type: TokenNull, Value: word, Pos: start}, nil
	case "and":
		return Token{Type: TokenAnd, Value: word, Pos: start}, nil
	case "or":
		return Token{Type: TokenOr, Value: word, Pos: start}, nil
	case "not":
		return Token{Type: TokenNot, Value: word, Pos: start}, nil
	case "in":
		return Token{Type: TokenIn, Value: word, Pos: start}, nil
	default:
		return Token{Type: TokenIdent, Value: word, Pos: start}, nil
	}
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
		l.pos++
	}
}

func isIdentStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9')
}
