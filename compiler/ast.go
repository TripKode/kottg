package compiler

import (
	"fmt"
	"strings"
)

// Node is the base interface for all AST nodes
type Node interface {
	String() string
}

// ===========================
// TOP-LEVEL NODES
// ===========================

// Program represents the entire kottg source file
type Program struct {
	Contracts []*ContractDecl
	Structs   []*StructDecl // Top-level structs
}

func (p *Program) String() string {
	var out strings.Builder
	for _, s := range p.Structs {
		out.WriteString(s.String())
		out.WriteString("\n")
	}
	for _, contract := range p.Contracts {
		out.WriteString(contract.String())
		out.WriteString("\n")
	}
	return out.String()
}

// ContractDecl represents a contract declaration
type ContractDecl struct {
	Name       string
	State      *StateBlock
	Init       *InitFunction
	Functions  []*Function
	Modifiers  []*ModifierDecl
	Events     []*EventDecl
	Structs    []*StructDecl
}

func (c *ContractDecl) String() string {
	return fmt.Sprintf("contract %s { ... }", c.Name)
}

// StateBlock represents the state variable declarations
type StateBlock struct {
	Variables []*StateVariable
}

func (s *StateBlock) String() string {
	return "state { ... }"
}

// StateVariable represents a state variable
type StateVariable struct {
	Name string
	Type TypeNode
}

func (s *StateVariable) String() string {
	return fmt.Sprintf("%s: %s", s.Name, s.Type.String())
}

// ===========================
// TYPE NODES
// ===========================

// TypeNode represents a type
type TypeNode interface {
	Node
	TypeName() string
}

// PrimitiveType represents primitive types (bool, u256, address, etc.)
type PrimitiveType struct {
	Name string
}

func (p *PrimitiveType) String() string {
	return p.Name
}

func (p *PrimitiveType) TypeName() string {
	return p.Name
}

// VecType represents vec[T] type
type VecType struct {
	ElementType TypeNode
}

func (v *VecType) String() string {
	return fmt.Sprintf("vec[%s]", v.ElementType.String())
}

func (v *VecType) TypeName() string {
	return fmt.Sprintf("vec[%s]", v.ElementType.TypeName())
}

// MapType represents map[K,V] type
type MapType struct {
	KeyType   TypeNode
	ValueType TypeNode
}

func (m *MapType) String() string {
	return fmt.Sprintf("map[%s,%s]", m.KeyType.String(), m.ValueType.String())
}

func (m *MapType) TypeName() string {
	return fmt.Sprintf("map[%s,%s]", m.KeyType.TypeName(), m.ValueType.TypeName())
}

// StructType represents a struct type reference
type StructType struct {
	Name string
}

func (s *StructType) String() string {
	return s.Name
}

func (s *StructType) TypeName() string {
	return s.Name
}

// ===========================
// FUNCTION NODES
// ===========================

// Function represents a function declaration
type Function struct {
	Visibility   string         // "pub", "priv", "" (default priv)
	IsView       bool           // true for view functions
	Name         string
	Parameters   []*Parameter
	ReturnType   TypeNode       // nil if void
	Modifiers    []string       // modifier names
	Body         *BlockStmt
}

func (f *Function) String() string {
	vis := ""
	if f.Visibility != "" {
		vis = f.Visibility + " "
	}
	viewStr := ""
	if f.IsView {
		viewStr = "view "
	}
	return fmt.Sprintf("%s%sfn %s(...)", vis, viewStr, f.Name)
}

// InitFunction represents the init (constructor) function
type InitFunction struct {
	Parameters []*Parameter
	Body       *BlockStmt
}

func (i *InitFunction) String() string {
	return "init(...)"
}

// Parameter represents a function parameter
type Parameter struct {
	Name string
	Type TypeNode
}

func (p *Parameter) String() string {
	return fmt.Sprintf("%s: %s", p.Name, p.Type.String())
}

// ModifierDecl represents a require modifier declaration
type ModifierDecl struct {
	Name string
	Body *BlockStmt
}

func (m *ModifierDecl) String() string {
	return fmt.Sprintf("require fn %s()", m.Name)
}

// ===========================
// STRUCT AND EVENT NODES
// ===========================

// StructDecl represents a struct declaration
type StructDecl struct {
	Name   string
	Fields []*StructField
}

func (s *StructDecl) String() string {
	return fmt.Sprintf("struct %s { ... }", s.Name)
}

// StructField represents a field in a struct
type StructField struct {
	Name string
	Type TypeNode
}

func (s *StructField) String() string {
	return fmt.Sprintf("%s: %s", s.Name, s.Type.String())
}

// EventDecl represents an event declaration
type EventDecl struct {
	Name   string
	Fields []*EventField
}

func (e *EventDecl) String() string {
	return fmt.Sprintf("event %s { ... }", e.Name)
}

// EventField represents a field in an event
type EventField struct {
	Name     string
	Type     TypeNode
	Indexed  bool
}

func (e *EventField) String() string {
	indexed := ""
	if e.Indexed {
		indexed = " indexed"
	}
	return fmt.Sprintf("%s: %s%s", e.Name, e.Type.String(), indexed)
}

// ===========================
// STATEMENT NODES
// ===========================

// Statement represents any statement
type Statement interface {
	Node
	statementNode()
}

// BlockStmt represents a block of statements { ... }
type BlockStmt struct {
	Statements []Statement
}

func (b *BlockStmt) String() string {
	return "{ ... }"
}

func (b *BlockStmt) statementNode() {}

// LetStmt represents variable declaration: let x: u256 = expr
type LetStmt struct {
	Name     string
	Type     TypeNode // can be nil for type inference
	Value    Expression
	IsMut    bool
}

func (l *LetStmt) String() string {
	mut := ""
	if l.IsMut {
		mut = "mut "
	}
	return fmt.Sprintf("let %s%s = ...", mut, l.Name)
}

func (l *LetStmt) statementNode() {}

// AssignStmt represents assignment: x = expr
type AssignStmt struct {
	Target Expression // Can be Identifier or MemberExpr
	Value  Expression
}

func (a *AssignStmt) String() string {
	return fmt.Sprintf("%s = ...", a.Target.String())
}

func (a *AssignStmt) statementNode() {}

// CompoundAssignStmt represents compound assignment: x += expr, x *= expr, etc.
type CompoundAssignStmt struct {
	Target   Expression
	Operator string // "+=", "-=", "*=", "/=", "%="
	Value    Expression
}

func (c *CompoundAssignStmt) String() string {
	return fmt.Sprintf("%s %s ...", c.Target.String(), c.Operator)
}

func (c *CompoundAssignStmt) statementNode() {}

// ReturnStmt represents return statement
type ReturnStmt struct {
	Value Expression // can be nil for void return
}

func (r *ReturnStmt) String() string {
	return "return ..."
}

func (r *ReturnStmt) statementNode() {}

// IfStmt represents if/else statement
type IfStmt struct {
	Condition  Expression
	ThenBranch *BlockStmt
	ElseBranch Statement // can be another IfStmt or BlockStmt or nil
}

func (i *IfStmt) String() string {
	return "if ... { ... }"
}

func (i *IfStmt) statementNode() {}

// LoopStmt represents loop statement
type LoopStmt struct {
	Iterator string     // variable name (e.g., "i")
	Start    Expression // start expression
	End      Expression // end expression
	Body     *BlockStmt
}

func (l *LoopStmt) String() string {
	return fmt.Sprintf("loop %s in ...", l.Iterator)
}

func (l *LoopStmt) statementNode() {}

// WhileStmt represents while loop
type WhileStmt struct {
	Condition Expression
	Body      *BlockStmt
}

func (w *WhileStmt) String() string {
	return "while ... { ... }"
}

func (w *WhileStmt) statementNode() {}

// BreakStmt represents break statement
type BreakStmt struct{}

func (b *BreakStmt) String() string {
	return "break"
}

func (b *BreakStmt) statementNode() {}

// ContinueStmt represents continue statement
type ContinueStmt struct{}

func (c *ContinueStmt) String() string {
	return "continue"
}

func (c *ContinueStmt) statementNode() {}

// EmitStmt represents emit event statement
type EmitStmt struct {
	EventName string
	Fields    []*EmitField
}

func (e *EmitStmt) String() string {
	return fmt.Sprintf("emit %s { ... }", e.EventName)
}

func (e *EmitStmt) statementNode() {}

// EmitField represents a field in emit statement
type EmitField struct {
	Name  string
	Value Expression
}

func (e *EmitField) String() string {
	return fmt.Sprintf("%s: ...", e.Name)
}

// AssertStmt represents assert statement
type AssertStmt struct {
	Condition Expression
	Message   Expression // should be string
}

func (a *AssertStmt) String() string {
	return "assert(...)"
}

func (a *AssertStmt) statementNode() {}

// RequireStmt represents require statement
type RequireStmt struct {
	Condition Expression
	Message   Expression // should be string
}

func (r *RequireStmt) String() string {
	return "require(...)"
}

func (r *RequireStmt) statementNode() {}

// RevertStmt represents revert statement
type RevertStmt struct {
	Message Expression // should be string
}

func (r *RevertStmt) String() string {
	return "revert(...)"
}

func (r *RevertStmt) statementNode() {}

// ExprStmt represents an expression used as a statement
type ExprStmt struct {
	Expr Expression
}

func (e *ExprStmt) String() string {
	return e.Expr.String()
}

func (e *ExprStmt) statementNode() {}

// MatchStmt represents match statement
type MatchStmt struct {
	Value Expression
	Arms  []*MatchArm
}

func (m *MatchStmt) String() string {
	return "match ... { ... }"
}

func (m *MatchStmt) statementNode() {}

// MatchArm represents a match arm
type MatchArm struct {
	Pattern Expression // literal or identifier
	Body    *BlockStmt
}

func (m *MatchArm) String() string {
	return fmt.Sprintf("%s => { ... }", m.Pattern.String())
}

// ===========================
// EXPRESSION NODES
// ===========================

// Expression represents any expression
type Expression interface {
	Node
	expressionNode()
}

// Identifier represents a variable or function name
type Identifier struct {
	Name string
}

func (i *Identifier) String() string {
	return i.Name
}

func (i *Identifier) expressionNode() {}

// IntegerLiteral represents integer literal
type IntegerLiteral struct {
	Value string
}

func (i *IntegerLiteral) String() string {
	return i.Value
}

func (i *IntegerLiteral) expressionNode() {}

// BoolLiteral represents boolean literal
type BoolLiteral struct {
	Value bool
}

func (b *BoolLiteral) String() string {
	if b.Value {
		return "true"
	}
	return "false"
}

func (b *BoolLiteral) expressionNode() {}

// StringLiteral represents string literal
type StringLiteral struct {
	Value string
}

func (s *StringLiteral) String() string {
	return fmt.Sprintf("\"%s\"", s.Value)
}

func (s *StringLiteral) expressionNode() {}

// AddressLiteral represents address literal (0x...)
type AddressLiteral struct {
	Value string
}

func (a *AddressLiteral) String() string {
	return a.Value
}

func (a *AddressLiteral) expressionNode() {}

// VecLiteral represents vec literal: vec[1, 2, 3]
type VecLiteral struct {
	Elements []Expression
}

func (v *VecLiteral) String() string {
	return "vec[...]"
}

func (v *VecLiteral) expressionNode() {}

// BinaryExpr represents binary expression: a + b, a == b, etc.
type BinaryExpr struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (b *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left.String(), b.Operator, b.Right.String())
}

func (b *BinaryExpr) expressionNode() {}

// UnaryExpr represents unary expression: !a, -a
type UnaryExpr struct {
	Operator string
	Operand  Expression
}

func (u *UnaryExpr) String() string {
	return fmt.Sprintf("(%s%s)", u.Operator, u.Operand.String())
}

func (u *UnaryExpr) expressionNode() {}

// CallExpr represents function call: foo(a, b)
type CallExpr struct {
	Function  Expression // usually Identifier or MemberExpr
	Arguments []Expression
}

func (c *CallExpr) String() string {
	return fmt.Sprintf("%s(...)", c.Function.String())
}

func (c *CallExpr) expressionNode() {}

// MemberExpr represents member access: obj.field or obj.method()
type MemberExpr struct {
	Object Expression
	Member string
}

func (m *MemberExpr) String() string {
	return fmt.Sprintf("%s.%s", m.Object.String(), m.Member)
}

func (m *MemberExpr) expressionNode() {}

// IndexExpr represents index access: arr[i] or map[key]
type IndexExpr struct {
	Object Expression
	Index  Expression
}

func (i *IndexExpr) String() string {
	return fmt.Sprintf("%s[...]", i.Object.String())
}

func (i *IndexExpr) expressionNode() {}

// CastExpr represents type casting: value as u256
type CastExpr struct {
	Value      Expression
	TargetType TypeNode
}

func (c *CastExpr) String() string {
	return fmt.Sprintf("%s as %s", c.Value.String(), c.TargetType.String())
}

func (c *CastExpr) expressionNode() {}

// RangeExpr represents range: start..end
type RangeExpr struct {
	Start Expression
	End   Expression
}

func (r *RangeExpr) String() string {
	return fmt.Sprintf("%s..%s", r.Start.String(), r.End.String())
}

func (r *RangeExpr) expressionNode() {}

// StructLiteral represents struct instantiation
type StructLiteral struct {
	StructName string
	Fields     []*StructLiteralField
}

func (s *StructLiteral) String() string {
	return fmt.Sprintf("%s { ... }", s.StructName)
}

func (s *StructLiteral) expressionNode() {}

// StructLiteralField represents a field in struct literal
type StructLiteralField struct {
	Name  string
	Value Expression
}

func (s *StructLiteralField) String() string {
	return fmt.Sprintf("%s: ...", s.Name)
}
