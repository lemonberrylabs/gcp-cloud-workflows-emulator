package expr

// Node is the interface for all expression AST nodes.
type Node interface {
	nodeType() string
}

// LiteralNode represents a literal value (int, float, string, bool, null).
type LiteralNode struct {
	TokenType TokenType
	IntVal    int64
	FloatVal  float64
	StrVal    string
	BoolVal   bool
}

func (n *LiteralNode) nodeType() string { return "Literal" }

// IdentNode represents a variable reference.
type IdentNode struct {
	Name string
}

func (n *IdentNode) nodeType() string { return "Ident" }

// BinaryNode represents a binary operation (e.g., a + b, x == y, a and b).
type BinaryNode struct {
	Op    TokenType
	Left  Node
	Right Node
}

func (n *BinaryNode) nodeType() string { return "Binary" }

// UnaryNode represents a unary operation (e.g., -x, not x).
type UnaryNode struct {
	Op      TokenType
	Operand Node
}

func (n *UnaryNode) nodeType() string { return "Unary" }

// PropertyNode represents property access (e.g., obj.field).
type PropertyNode struct {
	Object   Node
	Property string
}

func (n *PropertyNode) nodeType() string { return "Property" }

// IndexNode represents index access (e.g., list[0], map["key"]).
type IndexNode struct {
	Object Node
	Index  Node
}

func (n *IndexNode) nodeType() string { return "Index" }

// CallNode represents a function call (e.g., len(x), keys(m)).
type CallNode struct {
	Function Node   // can be IdentNode or PropertyNode (for dotted names like http.get)
	Args     []Node
}

func (n *CallNode) nodeType() string { return "Call" }

// ListNode represents a list literal (e.g., [1, 2, 3]).
type ListNode struct {
	Elements []Node
}

func (n *ListNode) nodeType() string { return "List" }

// MapNode represents a map literal (e.g., {"key": "value"}).
type MapNode struct {
	Keys   []Node // must be string literals
	Values []Node
}

func (n *MapNode) nodeType() string { return "Map" }

// InNode represents a membership test (e.g., x in list, "key" in map).
type InNode struct {
	Value     Node
	Container Node
	Negated   bool // true for "not in"
}

func (n *InNode) nodeType() string { return "In" }

// StringInterpolation represents a string with embedded ${} expressions.
// Only used at the top level when the entire YAML value is a string containing ${}.
type StringInterpolation struct {
	Parts []Node // alternating string literals and expression nodes
}

func (n *StringInterpolation) nodeType() string { return "StringInterpolation" }
