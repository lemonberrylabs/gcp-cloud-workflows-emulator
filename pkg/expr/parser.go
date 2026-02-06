package expr

import (
	"fmt"
	"strings"
)

// MaxExpressionLength is the maximum allowed length for a single expression.
const MaxExpressionLength = 400

// Parser is a recursive descent parser for GCW expressions.
type Parser struct {
	tokens []Token
	pos    int
}

// ParseExpression parses a complete expression string (without ${} wrapper).
func ParseExpression(input string) (Node, error) {
	if len(input) > MaxExpressionLength {
		return nil, fmt.Errorf("expression exceeds maximum length of %d characters", MaxExpressionLength)
	}

	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, fmt.Errorf("lexer error: %w", err)
	}

	p := &Parser{tokens: tokens}
	node, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.current().Type != TokenEOF {
		return nil, fmt.Errorf("unexpected token %s at position %d", p.current().Type, p.current().Pos)
	}

	return node, nil
}

// ParseValue parses a YAML value that may contain ${} expressions.
// Returns an expression Node. If the value is a plain string without ${},
// returns a LiteralNode. If it contains ${}, returns the appropriate node.
func ParseValue(v interface{}) (Node, error) {
	switch val := v.(type) {
	case nil:
		return &LiteralNode{TokenType: TokenNull}, nil
	case bool:
		if val {
			return &LiteralNode{TokenType: TokenTrue}, nil
		}
		return &LiteralNode{TokenType: TokenFalse}, nil
	case int:
		return &LiteralNode{TokenType: TokenInt, IntVal: int64(val)}, nil
	case int64:
		return &LiteralNode{TokenType: TokenInt, IntVal: val}, nil
	case float64:
		return &LiteralNode{TokenType: TokenFloat, FloatVal: val}, nil
	case string:
		return parseStringValue(val)
	case []interface{}:
		elements := make([]Node, len(val))
		for i, item := range val {
			node, err := ParseValue(item)
			if err != nil {
				return nil, err
			}
			elements[i] = node
		}
		return &ListNode{Elements: elements}, nil
	case map[string]interface{}:
		keys := make([]Node, 0, len(val))
		values := make([]Node, 0, len(val))
		for k, v := range val {
			keyNode, err := ParseValue(k)
			if err != nil {
				return nil, err
			}
			valNode, err := ParseValue(v)
			if err != nil {
				return nil, err
			}
			keys = append(keys, keyNode)
			values = append(values, valNode)
		}
		return &MapNode{Keys: keys, Values: values}, nil
	default:
		return nil, fmt.Errorf("unsupported value type %T", v)
	}
}

// parseStringValue handles a string that may be a ${} expression or contain interpolation.
func parseStringValue(s string) (Node, error) {
	// Check if entire string is a single ${...} expression
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "${") && strings.HasSuffix(trimmed, "}") {
		// Verify it's a single expression by counting balanced braces
		inner := trimmed[2 : len(trimmed)-1]
		if isBalanced(inner) {
			return ParseExpression(inner)
		}
	}

	// Check for embedded ${} expressions (string interpolation)
	if strings.Contains(s, "${") {
		return parseInterpolation(s)
	}

	// Plain string literal
	return &LiteralNode{TokenType: TokenString, StrVal: s}, nil
}

// isBalanced checks if a string has balanced braces, brackets, and parens.
func isBalanced(s string) bool {
	depth := 0
	inStr := false
	strChar := byte(0)
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inStr {
			if ch == '\\' && i+1 < len(s) {
				i++ // skip escaped char
				continue
			}
			if ch == strChar {
				inStr = false
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			inStr = true
			strChar = ch
			continue
		}
		switch ch {
		case '{', '(', '[':
			depth++
		case '}', ')', ']':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}

// parseInterpolation parses a string containing ${} expressions.
func parseInterpolation(s string) (Node, error) {
	var parts []Node
	i := 0
	for i < len(s) {
		idx := strings.Index(s[i:], "${")
		if idx == -1 {
			// Remaining text
			if i < len(s) {
				parts = append(parts, &LiteralNode{TokenType: TokenString, StrVal: s[i:]})
			}
			break
		}

		// Text before ${}
		if idx > 0 {
			parts = append(parts, &LiteralNode{TokenType: TokenString, StrVal: s[i : i+idx]})
		}

		// Find matching }
		start := i + idx + 2
		depth := 1
		j := start
		inStr := false
		strChar := byte(0)
		for j < len(s) && depth > 0 {
			ch := s[j]
			if inStr {
				if ch == '\\' && j+1 < len(s) {
					j++
				} else if ch == strChar {
					inStr = false
				}
			} else {
				if ch == '"' || ch == '\'' {
					inStr = true
					strChar = ch
				} else if ch == '{' {
					depth++
				} else if ch == '}' {
					depth--
				}
			}
			j++
		}

		if depth != 0 {
			return nil, fmt.Errorf("unmatched ${ in string")
		}

		exprStr := s[start : j-1]
		node, err := ParseExpression(exprStr)
		if err != nil {
			return nil, fmt.Errorf("error in expression ${%s}: %w", exprStr, err)
		}
		parts = append(parts, node)
		i = j
	}

	if len(parts) == 1 {
		return parts[0], nil
	}

	return &StringInterpolation{Parts: parts}, nil
}

// current returns the current token.
func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

// peek returns the next token without consuming it.
func (p *Parser) peek() Token {
	if p.pos+1 >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos+1]
}

// advance consumes the current token and returns it.
func (p *Parser) advance() Token {
	tok := p.current()
	p.pos++
	return tok
}

// expect consumes a token of the expected type or returns an error.
func (p *Parser) expect(tt TokenType) (Token, error) {
	tok := p.current()
	if tok.Type != tt {
		return tok, fmt.Errorf("expected %s, got %s at position %d", tt, tok.Type, tok.Pos)
	}
	p.advance()
	return tok, nil
}

// parseExpression is the entry point: handles the lowest precedence operators.
// Precedence (low to high):
//   or
//   and
//   not
//   in, not in
//   ==, !=, <, >, <=, >=
//   +, -
//   *, /, %, //
//   unary -, unary not
//   property access, index, function call
func (p *Parser) parseExpression() (Node, error) {
	return p.parseOr()
}

func (p *Parser) parseOr() (Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenOr {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryNode{Op: TokenOr, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseAnd() (Node, error) {
	left, err := p.parseNotExpr()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenAnd {
		p.advance()
		right, err := p.parseNotExpr()
		if err != nil {
			return nil, err
		}
		left = &BinaryNode{Op: TokenAnd, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseNotExpr() (Node, error) {
	if p.current().Type == TokenNot {
		// Check if it's "not in"
		if p.peek().Type == TokenIn {
			// Let parseComparison handle "not in"
			return p.parseComparison()
		}
		p.advance()
		operand, err := p.parseNotExpr()
		if err != nil {
			return nil, err
		}
		return &UnaryNode{Op: TokenNot, Operand: operand}, nil
	}
	return p.parseComparison()
}

func (p *Parser) parseComparison() (Node, error) {
	left, err := p.parseAddition()
	if err != nil {
		return nil, err
	}

	switch p.current().Type {
	case TokenEq, TokenNeq, TokenLt, TokenGt, TokenLte, TokenGte:
		op := p.advance().Type
		right, err := p.parseAddition()
		if err != nil {
			return nil, err
		}
		return &BinaryNode{Op: op, Left: left, Right: right}, nil
	case TokenIn:
		p.advance()
		right, err := p.parseAddition()
		if err != nil {
			return nil, err
		}
		return &InNode{Value: left, Container: right}, nil
	case TokenNot:
		if p.peek().Type == TokenIn {
			p.advance() // consume 'not'
			p.advance() // consume 'in'
			right, err := p.parseAddition()
			if err != nil {
				return nil, err
			}
			return &InNode{Value: left, Container: right, Negated: true}, nil
		}
	}

	return left, nil
}

func (p *Parser) parseAddition() (Node, error) {
	left, err := p.parseMultiplication()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenPlus || p.current().Type == TokenMinus {
		op := p.advance().Type
		right, err := p.parseMultiplication()
		if err != nil {
			return nil, err
		}
		left = &BinaryNode{Op: op, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseMultiplication() (Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenStar || p.current().Type == TokenSlash ||
		p.current().Type == TokenPercent || p.current().Type == TokenIntDiv {
		op := p.advance().Type
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &BinaryNode{Op: op, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseUnary() (Node, error) {
	if p.current().Type == TokenMinus {
		p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryNode{Op: TokenMinus, Operand: operand}, nil
	}
	if p.current().Type == TokenNot {
		p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryNode{Op: TokenNot, Operand: operand}, nil
	}
	return p.parsePostfix()
}

func (p *Parser) parsePostfix() (Node, error) {
	node, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		switch p.current().Type {
		case TokenDot:
			p.advance()
			name, err := p.expect(TokenIdent)
			if err != nil {
				return nil, fmt.Errorf("expected property name after '.': %w", err)
			}
			// Check if this is a function call: obj.method(...)
			if p.current().Type == TokenLParen {
				node = &PropertyNode{Object: node, Property: name.Value}
				args, err := p.parseArgList()
				if err != nil {
					return nil, err
				}
				node = &CallNode{Function: node, Args: args}
			} else {
				node = &PropertyNode{Object: node, Property: name.Value}
			}
		case TokenLBracket:
			p.advance()
			index, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			_, err = p.expect(TokenRBracket)
			if err != nil {
				return nil, fmt.Errorf("expected ']': %w", err)
			}
			node = &IndexNode{Object: node, Index: index}
		case TokenLParen:
			// Function call on the node directly
			args, err := p.parseArgList()
			if err != nil {
				return nil, err
			}
			node = &CallNode{Function: node, Args: args}
		default:
			return node, nil
		}
	}
}

func (p *Parser) parsePrimary() (Node, error) {
	tok := p.current()

	switch tok.Type {
	case TokenInt:
		p.advance()
		return &LiteralNode{TokenType: TokenInt, IntVal: tok.IntVal}, nil
	case TokenFloat:
		p.advance()
		return &LiteralNode{TokenType: TokenFloat, FloatVal: tok.FloatVal}, nil
	case TokenString:
		p.advance()
		return &LiteralNode{TokenType: TokenString, StrVal: tok.StrVal}, nil
	case TokenTrue:
		p.advance()
		return &LiteralNode{TokenType: TokenTrue, BoolVal: true}, nil
	case TokenFalse:
		p.advance()
		return &LiteralNode{TokenType: TokenFalse, BoolVal: false}, nil
	case TokenNull:
		p.advance()
		return &LiteralNode{TokenType: TokenNull}, nil
	case TokenIdent:
		p.advance()
		return &IdentNode{Name: tok.Value}, nil
	case TokenLParen:
		p.advance()
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		_, err = p.expect(TokenRParen)
		if err != nil {
			return nil, fmt.Errorf("expected ')': %w", err)
		}
		return expr, nil
	case TokenLBracket:
		return p.parseListLiteral()
	case TokenLBrace:
		return p.parseMapLiteral()
	default:
		return nil, fmt.Errorf("unexpected token %s (%q) at position %d", tok.Type, tok.Value, tok.Pos)
	}
}

// parseListLiteral parses [expr, expr, ...].
func (p *Parser) parseListLiteral() (Node, error) {
	p.advance() // consume [

	var elements []Node
	for p.current().Type != TokenRBracket {
		if len(elements) > 0 {
			_, err := p.expect(TokenComma)
			if err != nil {
				return nil, fmt.Errorf("expected ',' in list: %w", err)
			}
		}
		elem, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)
	}

	_, err := p.expect(TokenRBracket)
	if err != nil {
		return nil, fmt.Errorf("expected ']': %w", err)
	}

	return &ListNode{Elements: elements}, nil
}

// parseMapLiteral parses { key: value, key: value, ... }.
func (p *Parser) parseMapLiteral() (Node, error) {
	p.advance() // consume {

	var keys []Node
	var values []Node
	for p.current().Type != TokenRBrace {
		if len(keys) > 0 {
			_, err := p.expect(TokenComma)
			if err != nil {
				return nil, fmt.Errorf("expected ',' in map literal: %w", err)
			}
		}
		key, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		_, err = p.expect(TokenColon)
		if err != nil {
			return nil, fmt.Errorf("expected ':' in map literal: %w", err)
		}
		value, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
		values = append(values, value)
	}

	_, err := p.expect(TokenRBrace)
	if err != nil {
		return nil, fmt.Errorf("expected '}': %w", err)
	}

	return &MapNode{Keys: keys, Values: values}, nil
}

// parseArgList parses (expr, expr, ...).
func (p *Parser) parseArgList() ([]Node, error) {
	_, err := p.expect(TokenLParen)
	if err != nil {
		return nil, fmt.Errorf("expected '(': %w", err)
	}

	var args []Node
	for p.current().Type != TokenRParen {
		if len(args) > 0 {
			_, err := p.expect(TokenComma)
			if err != nil {
				return nil, fmt.Errorf("expected ',' in arguments: %w", err)
			}
		}
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}

	_, err = p.expect(TokenRParen)
	if err != nil {
		return nil, fmt.Errorf("expected ')': %w", err)
	}

	return args, nil
}
