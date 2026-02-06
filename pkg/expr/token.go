// Package expr implements the GCW expression language parser and evaluator.
// It handles ${...} expressions with arithmetic, comparison, logical, and
// property access operators.
package expr

// TokenType represents the type of a lexical token.
type TokenType int

const (
	// Literals
	TokenInt    TokenType = iota // integer literal
	TokenFloat                   // float literal
	TokenString                  // string literal
	TokenTrue                    // true
	TokenFalse                   // false
	TokenNull                    // null

	// Identifiers and operators
	TokenIdent // identifier (variable name)
	TokenDot   // .
	TokenComma // ,

	// Brackets
	TokenLParen   // (
	TokenRParen   // )
	TokenLBracket // [
	TokenRBracket // ]
	TokenLBrace   // {
	TokenRBrace   // }
	TokenColon    // :

	// Arithmetic
	TokenPlus       // +
	TokenMinus      // -
	TokenStar       // *
	TokenSlash      // /
	TokenPercent    // %
	TokenIntDiv     // //

	// Comparison
	TokenEq  // ==
	TokenNeq // !=
	TokenLt  // <
	TokenGt  // >
	TokenLte // <=
	TokenGte // >=

	// Logical
	TokenAnd // and
	TokenOr  // or
	TokenNot // not

	// Membership
	TokenIn // in

	// Special
	TokenEOF // end of expression
)

// Token represents a single lexical token.
type Token struct {
	Type    TokenType
	Value   string   // raw string value
	IntVal  int64    // parsed int (for TokenInt)
	FloatVal float64 // parsed float (for TokenFloat)
	StrVal  string   // parsed string (for TokenString, with escapes resolved)
	Pos     int      // position in source
}

// String returns a debug-friendly representation of the token type.
func (t TokenType) String() string {
	switch t {
	case TokenInt:
		return "INT"
	case TokenFloat:
		return "FLOAT"
	case TokenString:
		return "STRING"
	case TokenTrue:
		return "TRUE"
	case TokenFalse:
		return "FALSE"
	case TokenNull:
		return "NULL"
	case TokenIdent:
		return "IDENT"
	case TokenDot:
		return "DOT"
	case TokenComma:
		return "COMMA"
	case TokenLParen:
		return "LPAREN"
	case TokenRParen:
		return "RPAREN"
	case TokenLBracket:
		return "LBRACKET"
	case TokenRBracket:
		return "RBRACKET"
	case TokenLBrace:
		return "LBRACE"
	case TokenRBrace:
		return "RBRACE"
	case TokenColon:
		return "COLON"
	case TokenPlus:
		return "PLUS"
	case TokenMinus:
		return "MINUS"
	case TokenStar:
		return "STAR"
	case TokenSlash:
		return "SLASH"
	case TokenPercent:
		return "PERCENT"
	case TokenIntDiv:
		return "INTDIV"
	case TokenEq:
		return "EQ"
	case TokenNeq:
		return "NEQ"
	case TokenLt:
		return "LT"
	case TokenGt:
		return "GT"
	case TokenLte:
		return "LTE"
	case TokenGte:
		return "GTE"
	case TokenAnd:
		return "AND"
	case TokenOr:
		return "OR"
	case TokenNot:
		return "NOT"
	case TokenIn:
		return "IN"
	case TokenEOF:
		return "EOF"
	default:
		return "UNKNOWN"
	}
}
