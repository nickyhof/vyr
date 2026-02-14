package parser

// Node is the interface all AST nodes implement.
type Node interface {
	nodeType() string
}

// Param represents a function parameter with optional type annotation.
type Param struct {
	Name string
	Type string // type annotation, empty if untyped
}

// Program is the root AST node, containing top-level declarations.
type Program struct {
	Decls []Node
}

func (p *Program) nodeType() string { return "Program" }

// ImportDecl represents an import declaration: import "path"
type ImportDecl struct {
	Path string
}

func (i *ImportDecl) nodeType() string { return "ImportDecl" }

// FnDecl represents a function declaration: fn name(params): returnType { body }
type FnDecl struct {
	Name       string
	Params     []Param
	ReturnType string // empty if untyped
	Body       []Node
}

func (f *FnDecl) nodeType() string { return "FnDecl" }

// StringLit represents a string literal: "hello"
type StringLit struct {
	Value string
}

func (s *StringLit) nodeType() string { return "StringLit" }

// IntLit represents an integer literal: 42
type IntLit struct {
	Value int64
}

func (i *IntLit) nodeType() string { return "IntLit" }

// BoolLit represents a boolean literal: true or false
type BoolLit struct {
	Value bool
}

func (b *BoolLit) nodeType() string { return "BoolLit" }

// Ident represents an identifier (variable or function name).
type Ident struct {
	Name string
}

func (id *Ident) nodeType() string { return "Ident" }

// CallExpr represents a function call: fn(args...)
type CallExpr struct {
	Fn   Node
	Args []Node
}

func (c *CallExpr) nodeType() string { return "CallExpr" }

// LetStmt represents a variable binding: let name = expr (or let mut name = expr)
type LetStmt struct {
	Name    string
	Value   Node
	Mutable bool
}

func (l *LetStmt) nodeType() string { return "LetStmt" }

// BinaryExpr represents a binary operation: left op right
// Op can be arithmetic (+, -, *, /) or comparison (==, !=, <, >, <=, >=)
type BinaryExpr struct {
	Op    string
	Left  Node
	Right Node
}

func (b *BinaryExpr) nodeType() string { return "BinaryExpr" }

// ArrayLit represents an array literal: [a, b, c]
type ArrayLit struct {
	Elements []Node
}

func (a *ArrayLit) nodeType() string { return "ArrayLit" }

// FnLit represents an anonymous function (lambda): fn(params) { body }
type FnLit struct {
	Params []Param
	Body   []Node
}

func (f *FnLit) nodeType() string { return "FnLit" }

// FanOutExpr represents a fan-out pipe: value |> [f, g, h]
type FanOutExpr struct {
	Value Node
	Fns   []Node
}

func (f *FanOutExpr) nodeType() string { return "FanOutExpr" }

// MatchExpr represents pattern matching: match subject { pattern => body, ... }
type MatchExpr struct {
	Subject Node
	Arms    []MatchArm
}

func (m *MatchExpr) nodeType() string { return "MatchExpr" }

// MatchArm is a single arm of a match expression.
type MatchArm struct {
	Pattern Node
	Body    Node
}

// IfExpr represents an if/else expression: if cond { then } else { else }
type IfExpr struct {
	Condition Node
	Then      []Node
	Else      []Node // empty if no else clause
}

func (i *IfExpr) nodeType() string { return "IfExpr" }

// HashPair is a key-value pair in a hashmap literal.
type HashPair struct {
	Key   string
	Value Node
}

// HashLit represents a hashmap literal: #{key: val, key2: val2}
type HashLit struct {
	Pairs []HashPair
}

func (h *HashLit) nodeType() string { return "HashLit" }

// DotExpr represents dot access: expr.key
type DotExpr struct {
	Object Node
	Key    string
}

func (d *DotExpr) nodeType() string { return "DotExpr" }

// InterpLit represents an interpolated string: "hello ${name}, age ${age}"
// Parts alternates between StringLit (text fragments) and expression nodes.
type InterpLit struct {
	Parts []Node
}

func (il *InterpLit) nodeType() string { return "InterpLit" }

// WhileExpr represents a while loop: while cond { body }
type WhileExpr struct {
	Condition Node
	Body      []Node
}

func (w *WhileExpr) nodeType() string { return "WhileExpr" }

// AssignStmt represents variable reassignment: name = expr
type AssignStmt struct {
	Name  string
	Value Node
}

func (a *AssignStmt) nodeType() string { return "AssignStmt" }

// StructDecl declares a struct type: struct Name { field1, field2 }
type StructDecl struct {
	Name   string
	Fields []string
}

func (s *StructDecl) nodeType() string { return "StructDecl" }

// TypePattern is a pattern that matches on struct type: TypeName(binding)
type TypePattern struct {
	TypeName string
	Binding  string
}

func (tp *TypePattern) nodeType() string { return "TypePattern" }

// ReturnStmt represents an early return: return expr
type ReturnStmt struct {
	Value Node // nil means return nil
}

func (r *ReturnStmt) nodeType() string { return "ReturnStmt" }
