package compiler

import (
	"fmt"
	"strings"
)

// SemanticAnalyzer performs semantic analysis on the AST
type SemanticAnalyzer struct {
	errors  []string
	scopes  *ScopeStack
	contract *ContractDecl
	currentFunction *Function
	inLoop  bool
}

// NewSemanticAnalyzer creates a new semantic analyzer
func NewSemanticAnalyzer() *SemanticAnalyzer {
	return &SemanticAnalyzer{
		errors: []string{},
		scopes: NewScopeStack(),
	}
}

// Errors returns the list of semantic errors
func (sa *SemanticAnalyzer) Errors() []string {
	return sa.errors
}

// addError adds a semantic error
func (sa *SemanticAnalyzer) addError(msg string) {
	sa.errors = append(sa.errors, msg)
}

// ===========================
// SCOPE MANAGEMENT
// ===========================

// Scope represents a lexical scope
type Scope struct {
	parent  *Scope
	symbols map[string]*Symbol
}

// Symbol represents a symbol in the scope
type Symbol struct {
	Name      string
	Type      TypeNode
	Kind      SymbolKind
	IsMutable bool
}

// SymbolKind represents the kind of symbol
type SymbolKind int

const (
	SymbolVar SymbolKind = iota
	SymbolParam
	SymbolStateVar
	SymbolFunction
	SymbolModifier
	SymbolEvent
	SymbolStruct
)

// ScopeStack manages nested scopes
type ScopeStack struct {
	current *Scope
}

// NewScopeStack creates a new scope stack
func NewScopeStack() *ScopeStack {
	return &ScopeStack{
		current: &Scope{
			symbols: make(map[string]*Symbol),
		},
	}
}

// PushScope creates a new scope
func (ss *ScopeStack) PushScope() {
	ss.current = &Scope{
		parent:  ss.current,
		symbols: make(map[string]*Symbol),
	}
}

// PopScope removes the current scope
func (ss *ScopeStack) PopScope() {
	if ss.current.parent != nil {
		ss.current = ss.current.parent
	}
}

// Define adds a symbol to the current scope
func (ss *ScopeStack) Define(name string, symbol *Symbol) bool {
	if _, exists := ss.current.symbols[name]; exists {
		return false
	}
	ss.current.symbols[name] = symbol
	return true
}

// Lookup finds a symbol in the scope chain
func (ss *ScopeStack) Lookup(name string) *Symbol {
	scope := ss.current
	for scope != nil {
		if symbol, exists := scope.symbols[name]; exists {
			return symbol
		}
		scope = scope.parent
	}
	return nil
}

// ===========================
// TYPE CHECKING
// ===========================

// isPrimitiveType checks if a type is primitive
func isPrimitiveType(typeName string) bool {
	primitives := []string{
		"bool", "u8", "u16", "u32", "u64", "u128", "u256",
		"i8", "i16", "i32", "i64", "i128", "i256",
		"address", "hash", "bytes", "string",
	}
	for _, p := range primitives {
		if typeName == p {
			return true
		}
	}
	return false
}

// isNumericType checks if a type is numeric
func isNumericType(typeName string) bool {
	numerics := []string{
		"u8", "u16", "u32", "u64", "u128", "u256",
		"i8", "i16", "i32", "i64", "i128", "i256",
	}
	for _, n := range numerics {
		if typeName == n {
			return true
		}
	}
	return false
}

// isIntegerType checks if a type is integer
func isIntegerType(typeName string) bool {
	return isNumericType(typeName)
}

// typesMatch checks if two types are compatible
func (sa *SemanticAnalyzer) typesMatch(t1, t2 TypeNode) bool {
	if t1 == nil || t2 == nil {
		return false
	}
	n1, n2 := t1.TypeName(), t2.TypeName()
	if n1 == n2 {
		return true
	}
	// KSP-F2: familia numérica compatible — la seguridad la garantiza el
	// RANGE-CHECK en runtime (emitRangeCheck), no el checker estático.
	if isNumericType(n1) && isNumericType(n2) {
		return true
	}
	// KSP-1: identidad address = u256 en runtime (hash QSH completo), por lo
	// que address ↔ numérico es comparable/asignable estáticamente.
	if (n1 == "address" && isNumericType(n2)) || (n2 == "address" && isNumericType(n1)) {
		return true
	}
	return false
}

// getExpressionType returns the type of an expression
func (sa *SemanticAnalyzer) getExpressionType(expr Expression) TypeNode {
	switch e := expr.(type) {
	case *Identifier:
		return sa.getIdentifierType(e)
	case *IntegerLiteral:
		return &PrimitiveType{Name: "u256"} // default integer type
	case *BoolLiteral:
		return &PrimitiveType{Name: "bool"}
	case *StringLiteral:
		return &PrimitiveType{Name: "string"}
	case *AddressLiteral:
		return &PrimitiveType{Name: "address"}
	case *BinaryExpr:
		return sa.getBinaryExprType(e)
	case *UnaryExpr:
		return sa.getUnaryExprType(e)
	case *CallExpr:
		return sa.getCallExprType(e)
	case *MemberExpr:
		return sa.getMemberExprType(e)
	case *IndexExpr:
		return sa.getIndexExprType(e)
	case *CastExpr:
		return e.TargetType
	case *VecLiteral:
		if len(e.Elements) > 0 {
			elemType := sa.getExpressionType(e.Elements[0])
			return &VecType{ElementType: elemType}
		}
		return nil
	default:
		return nil
	}
}

// getIdentifierType returns the type of an identifier
func (sa *SemanticAnalyzer) getIdentifierType(ident *Identifier) TypeNode {
	// Check for built-in globals
	switch ident.Name {
	case "msg":
		return &PrimitiveType{Name: "msg_context"}
	case "block":
		return &PrimitiveType{Name: "block_context"}
	case "tx":
		return &PrimitiveType{Name: "tx_context"}
	case "self":
		return &PrimitiveType{Name: "contract"}
	case "precompiled":
		return &PrimitiveType{Name: "precompiled"}
	case "sec":
		// KSP-F2: builtin cripto post-cuántico (sec.qsh)
		return &PrimitiveType{Name: "sec"}
	case "ext":
		// KSP-3: builtin de llamadas externas (ext.call / ext.create)
		return &PrimitiveType{Name: "ext"}
	}

	// Lookup in scope
	symbol := sa.scopes.Lookup(ident.Name)
	if symbol != nil {
		return symbol.Type
	}

	sa.addError(fmt.Sprintf("undefined identifier: %s", ident.Name))
	return nil
}

// getBinaryExprType returns the type of a binary expression
func (sa *SemanticAnalyzer) getBinaryExprType(expr *BinaryExpr) TypeNode {
	leftType := sa.getExpressionType(expr.Left)
	rightType := sa.getExpressionType(expr.Right)

	if leftType == nil || rightType == nil {
		return nil
	}

	// Check operator compatibility
	switch expr.Operator {
	case "+", "-", "*", "/", "%":
		// Arithmetic operators require numeric types
		if !isNumericType(leftType.TypeName()) || !isNumericType(rightType.TypeName()) {
			sa.addError(fmt.Sprintf("operator %s requires numeric operands, got %s and %s",
				expr.Operator, leftType.TypeName(), rightType.TypeName()))
			return nil
		}
		if !sa.typesMatch(leftType, rightType) {
			sa.addError(fmt.Sprintf("type mismatch: %s %s %s",
				leftType.TypeName(), expr.Operator, rightType.TypeName()))
			return nil
		}
		return leftType

	case "==", "!=":
		// Equality operators require matching types
		if !sa.typesMatch(leftType, rightType) {
			sa.addError(fmt.Sprintf("cannot compare %s with %s",
				leftType.TypeName(), rightType.TypeName()))
			return nil
		}
		return &PrimitiveType{Name: "bool"}

	case "<", ">", "<=", ">=":
		// Comparison operators require numeric types
		if !isNumericType(leftType.TypeName()) || !isNumericType(rightType.TypeName()) {
			sa.addError(fmt.Sprintf("operator %s requires numeric operands",
				expr.Operator))
			return nil
		}
		return &PrimitiveType{Name: "bool"}

	case "&&", "||":
		// Logical operators require boolean types
		if leftType.TypeName() != "bool" || rightType.TypeName() != "bool" {
			sa.addError(fmt.Sprintf("operator %s requires boolean operands",
				expr.Operator))
			return nil
		}
		return &PrimitiveType{Name: "bool"}
	}

	return nil
}

// getUnaryExprType returns the type of a unary expression
func (sa *SemanticAnalyzer) getUnaryExprType(expr *UnaryExpr) TypeNode {
	operandType := sa.getExpressionType(expr.Operand)
	if operandType == nil {
		return nil
	}

	switch expr.Operator {
	case "!":
		if operandType.TypeName() != "bool" {
			sa.addError(fmt.Sprintf("operator ! requires boolean operand, got %s",
				operandType.TypeName()))
			return nil
		}
		return &PrimitiveType{Name: "bool"}

	case "-":
		if !isNumericType(operandType.TypeName()) {
			sa.addError(fmt.Sprintf("operator - requires numeric operand, got %s",
				operandType.TypeName()))
			return nil
		}
		return operandType
	}

	return nil
}

// getCallExprType returns the type of a call expression
func (sa *SemanticAnalyzer) getCallExprType(expr *CallExpr) TypeNode {
	// Handle member calls (e.g., precompiled.getStake)
	if memberExpr, ok := expr.Function.(*MemberExpr); ok {
		return sa.getMemberCallType(memberExpr, expr.Arguments)
	}

	// Handle regular function calls
	if ident, ok := expr.Function.(*Identifier); ok {
		symbol := sa.scopes.Lookup(ident.Name)
		if symbol == nil {
			sa.addError(fmt.Sprintf("undefined function: %s", ident.Name))
			return nil
		}

		if symbol.Kind != SymbolFunction {
			sa.addError(fmt.Sprintf("%s is not a function", ident.Name))
			return nil
		}

		// Return function's return type
		// For now, we store function info in the symbol
		return symbol.Type
	}

	return nil
}

// getMemberCallType returns the type of a member call
func (sa *SemanticAnalyzer) getMemberCallType(memberExpr *MemberExpr, args []Expression) TypeNode {
	objectType := sa.getExpressionType(memberExpr.Object)
	if objectType == nil {
		return nil
	}

	// Handle precompiled calls
	if objectType.TypeName() == "precompiled" {
		return sa.getPrecompiledCallType(memberExpr.Member, args)
	}

	// KSP-F2: sec.qsh(value) → u256 (hash post-cuántico runtime)
	if objectType.TypeName() == "sec" {
		if memberExpr.Member == "qsh" {
			if len(args) != 1 {
				sa.addError("sec.qsh requiere 1 argumento")
				return nil
			}
			return &PrimitiveType{Name: "u256"}
		}
		sa.addError("sec." + memberExpr.Member + " no existe (solo qsh)")
		return nil
	}

	// KSP-3: ext.call(to, "sig", args...) → u256; ext.create(code) → address
	if objectType.TypeName() == "ext" {
		switch memberExpr.Member {
		case "call":
			if len(args) < 2 {
				sa.addError("ext.call requiere: dirección, firma literal, args...")
				return nil
			}
			if lit, ok := args[1].(*StringLiteral); !ok || !strings.Contains(lit.Value, "(") {
				sa.addError("ext.call: la firma debe ser un literal \"nombre(tipos)\"")
				return nil
			}
			return &PrimitiveType{Name: "u256"}
		case "create":
			if len(args) != 1 {
				sa.addError("ext.create requiere 1 literal string con bytecode hex")
				return nil
			}
			if _, ok := args[0].(*StringLiteral); !ok {
				sa.addError("ext.create: el bytecode debe ser un literal string")
				return nil
			}
			return &PrimitiveType{Name: "address"}
		}
		sa.addError("ext." + memberExpr.Member + " no existe (solo call/create)")
		return nil
	}

	// Handle msg, block, tx context members
	if objectType.TypeName() == "msg_context" {
		return sa.getMsgMemberType(memberExpr.Member)
	}
	if objectType.TypeName() == "block_context" {
		return sa.getBlockMemberType(memberExpr.Member)
	}
	if objectType.TypeName() == "tx_context" {
		return sa.getTxMemberType(memberExpr.Member)
	}

	// Handle contract self members
	if objectType.TypeName() == "contract" {
		return sa.getContractMemberType(memberExpr.Member)
	}

	return nil
}

// getPrecompiledCallType returns the type of a precompiled call
func (sa *SemanticAnalyzer) getPrecompiledCallType(method string, args []Expression) TypeNode {
	switch method {
	case "getStake":
		if len(args) != 1 {
			sa.addError(fmt.Sprintf("precompiled.getStake expects 1 argument, got %d", len(args)))
			return nil
		}
		return &PrimitiveType{Name: "u256"}

	case "beaconRoot":
		// KSP-2: precompiled.beaconRoot() → último; beaconRoot(i) → raíz en i
		if len(args) > 1 {
			sa.addError(fmt.Sprintf("precompiled.beaconRoot expects 0 or 1 arguments, got %d", len(args)))
			return nil
		}
		return &PrimitiveType{Name: "u256"}

	case "beaconHeight":
		// KSP-2: precompiled.beaconHeight() → índice del último bloque beacon
		if len(args) != 0 {
			sa.addError(fmt.Sprintf("precompiled.beaconHeight expects 0 arguments, got %d", len(args)))
			return nil
		}
		return &PrimitiveType{Name: "u256"}

	case "isValidator":
		if len(args) != 1 {
			sa.addError(fmt.Sprintf("precompiled.isValidator expects 1 argument, got %d", len(args)))
			return nil
		}
		return &PrimitiveType{Name: "bool"}

	case "distributeFees":
		if len(args) != 2 {
			sa.addError(fmt.Sprintf("precompiled.distributeFees expects 2 arguments, got %d", len(args)))
			return nil
		}
		return &PrimitiveType{Name: "bool"}

	case "getValidatorInfo":
		if len(args) != 1 {
			sa.addError(fmt.Sprintf("precompiled.getValidatorInfo expects 1 argument, got %d", len(args)))
			return nil
		}
		// Return ValidatorInfo struct type
		return &StructType{Name: "ValidatorInfo"}

	default:
		sa.addError(fmt.Sprintf("unknown precompiled method: %s", method))
		return nil
	}
}

// getMsgMemberType returns the type of a msg context member
func (sa *SemanticAnalyzer) getMsgMemberType(member string) TypeNode {
	msgMembers := map[string]string{
		"sender":  "address",
		"value":   "u256",
		"data":    "bytes",
		"gas":     "u64",
	}
	if typeName, ok := msgMembers[member]; ok {
		return &PrimitiveType{Name: typeName}
	}
	sa.addError(fmt.Sprintf("unknown msg member: %s", member))
	return nil
}

// getBlockMemberType returns the type of a block context member
func (sa *SemanticAnalyzer) getBlockMemberType(member string) TypeNode {
	blockMembers := map[string]string{
		"number":    "u64",
		"timestamp": "u64",
		"validator": "address",
	}
	if typeName, ok := blockMembers[member]; ok {
		return &PrimitiveType{Name: typeName}
	}
	sa.addError(fmt.Sprintf("unknown block member: %s", member))
	return nil
}

// getTxMemberType returns the type of a tx context member
func (sa *SemanticAnalyzer) getTxMemberType(member string) TypeNode {
	txMembers := map[string]string{
		"origin":   "address",
		"gasprice": "u256",
	}
	if typeName, ok := txMembers[member]; ok {
		return &PrimitiveType{Name: typeName}
	}
	sa.addError(fmt.Sprintf("unknown tx member: %s", member))
	return nil
}

// getContractMemberType returns the type of a contract self member
func (sa *SemanticAnalyzer) getContractMemberType(member string) TypeNode {
	// Built-in self members
	selfMembers := map[string]string{
		"address": "address",
		"balance": "u256",
	}
	if typeName, ok := selfMembers[member]; ok {
		return &PrimitiveType{Name: typeName}
	}

	// Check state variables
	if sa.contract != nil && sa.contract.State != nil {
		for _, stateVar := range sa.contract.State.Variables {
			if stateVar.Name == member {
				return stateVar.Type
			}
		}
	}

	sa.addError(fmt.Sprintf("unknown self member: %s", member))
	return nil
}

// getMemberExprType returns the type of a member expression
func (sa *SemanticAnalyzer) getMemberExprType(expr *MemberExpr) TypeNode {
	objectType := sa.getExpressionType(expr.Object)
	if objectType == nil {
		return nil
	}

	// Handle different object types
	switch objectType.TypeName() {
	case "msg_context":
		return sa.getMsgMemberType(expr.Member)
	case "block_context":
		return sa.getBlockMemberType(expr.Member)
	case "tx_context":
		return sa.getTxMemberType(expr.Member)
	case "contract":
		return sa.getContractMemberType(expr.Member)
	default:
		// Handle struct field access
		// TODO: Implement struct field type lookup
		return nil
	}
}

// getIndexExprType returns the type of an index expression
func (sa *SemanticAnalyzer) getIndexExprType(expr *IndexExpr) TypeNode {
	objectType := sa.getExpressionType(expr.Object)
	if objectType == nil {
		return nil
	}

	// Handle vec indexing
	if vecType, ok := objectType.(*VecType); ok {
		// Check index type is numeric
		indexType := sa.getExpressionType(expr.Index)
		if indexType != nil && !isNumericType(indexType.TypeName()) {
			sa.addError("vec index must be numeric")
		}
		return vecType.ElementType
	}

	// Handle map indexing
	if mapType, ok := objectType.(*MapType); ok {
		// Check index type matches key type
		indexType := sa.getExpressionType(expr.Index)
		if indexType != nil && !sa.typesMatch(indexType, mapType.KeyType) {
			sa.addError(fmt.Sprintf("map key type mismatch: expected %s, got %s",
				mapType.KeyType.TypeName(), indexType.TypeName()))
		}
		return mapType.ValueType
	}

	sa.addError(fmt.Sprintf("cannot index type %s", objectType.TypeName()))
	return nil
}

// ===========================
// ANALYSIS ENTRY POINT
// ===========================

// Analyze performs semantic analysis on the program
func (sa *SemanticAnalyzer) Analyze(program *Program) bool {
	// Analyze top-level structs
	for _, structDecl := range program.Structs {
		sa.analyzeStructDecl(structDecl, true)
	}

	// Analyze contracts
	for _, contract := range program.Contracts {
		sa.analyzeContract(contract)
	}

	return len(sa.errors) == 0
}

// analyzeStructDecl analyzes a struct declaration
func (sa *SemanticAnalyzer) analyzeStructDecl(structDecl *StructDecl, isTopLevel bool) {
	// Register struct in scope
	sa.scopes.Define(structDecl.Name, &Symbol{
		Name: structDecl.Name,
		Type: &StructType{Name: structDecl.Name},
		Kind: SymbolStruct,
	})

	// Check field types
	for _, field := range structDecl.Fields {
		if !sa.isValidType(field.Type) {
			sa.addError(fmt.Sprintf("struct %s: invalid field type %s",
				structDecl.Name, field.Type.TypeName()))
		}
	}
}

// isValidType checks if a type is valid
func (sa *SemanticAnalyzer) isValidType(typeNode TypeNode) bool {
	switch t := typeNode.(type) {
	case *PrimitiveType:
		return isPrimitiveType(t.Name) || sa.scopes.Lookup(t.Name) != nil
	case *VecType:
		return sa.isValidType(t.ElementType)
	case *MapType:
		return sa.isValidType(t.KeyType) && sa.isValidType(t.ValueType)
	case *StructType:
		return sa.scopes.Lookup(t.Name) != nil
	default:
		return false
	}
}

// analyzeContract analyzes a contract
func (sa *SemanticAnalyzer) analyzeContract(contract *ContractDecl) {
	sa.contract = contract

	// Create contract scope
	sa.scopes.PushScope()
	defer sa.scopes.PopScope()

	// Register contract structs
	for _, structDecl := range contract.Structs {
		sa.analyzeStructDecl(structDecl, false)
	}

	// Register state variables
	if contract.State != nil {
		for _, stateVar := range contract.State.Variables {
			if !sa.scopes.Define(stateVar.Name, &Symbol{
				Name:      stateVar.Name,
				Type:      stateVar.Type,
				Kind:      SymbolStateVar,
				IsMutable: true, // state variables are always mutable
			}) {
				sa.addError(fmt.Sprintf("duplicate state variable: %s", stateVar.Name))
			}

			// Check type validity
			if !sa.isValidType(stateVar.Type) {
				sa.addError(fmt.Sprintf("invalid type for state variable %s: %s",
					stateVar.Name, stateVar.Type.TypeName()))
			}
		}
	}

	// Register events
	for _, event := range contract.Events {
		sa.scopes.Define(event.Name, &Symbol{
			Name: event.Name,
			Type: nil, // events don't have a type
			Kind: SymbolEvent,
		})
	}

	// Register modifiers
	for _, modifier := range contract.Modifiers {
		sa.scopes.Define(modifier.Name, &Symbol{
			Name: modifier.Name,
			Type: nil,
			Kind: SymbolModifier,
		})
	}

	// Register functions
	for _, fn := range contract.Functions {
		sa.scopes.Define(fn.Name, &Symbol{
			Name: fn.Name,
			Type: fn.ReturnType,
			Kind: SymbolFunction,
		})
	}

	// Analyze init function
	if contract.Init != nil {
		sa.analyzeInitFunction(contract.Init)
	}

	// Analyze modifiers
	for _, modifier := range contract.Modifiers {
		sa.analyzeModifier(modifier)
	}

	// Analyze functions
	for _, fn := range contract.Functions {
		sa.analyzeFunction(fn)
	}
}

// analyzeInitFunction analyzes the init function
func (sa *SemanticAnalyzer) analyzeInitFunction(init *InitFunction) {
	sa.scopes.PushScope()
	defer sa.scopes.PopScope()

	// Register parameters
	for _, param := range init.Parameters {
		if !sa.scopes.Define(param.Name, &Symbol{
			Name:      param.Name,
			Type:      param.Type,
			Kind:      SymbolParam,
			IsMutable: false,
		}) {
			sa.addError(fmt.Sprintf("duplicate parameter: %s", param.Name))
		}
	}

	// Analyze body
	sa.analyzeBlock(init.Body)
}

// analyzeModifier analyzes a modifier
func (sa *SemanticAnalyzer) analyzeModifier(modifier *ModifierDecl) {
	sa.scopes.PushScope()
	defer sa.scopes.PopScope()

	sa.analyzeBlock(modifier.Body)
}

// analyzeFunction analyzes a function
func (sa *SemanticAnalyzer) analyzeFunction(fn *Function) {
	sa.currentFunction = fn
	defer func() { sa.currentFunction = nil }()

	// Check view function constraints
	if fn.IsView {
		// View functions cannot modify state
		// This will be checked during statement analysis
	}

	sa.scopes.PushScope()
	defer sa.scopes.PopScope()

	// Register parameters
	for _, param := range fn.Parameters {
		if !sa.scopes.Define(param.Name, &Symbol{
			Name:      param.Name,
			Type:      param.Type,
			Kind:      SymbolParam,
			IsMutable: false,
		}) {
			sa.addError(fmt.Sprintf("function %s: duplicate parameter: %s", fn.Name, param.Name))
		}
	}

	// Verify modifiers exist
	for _, modifierName := range fn.Modifiers {
		if sa.scopes.Lookup(modifierName) == nil {
			sa.addError(fmt.Sprintf("function %s: undefined modifier: %s", fn.Name, modifierName))
		}
	}

	// Analyze body
	sa.analyzeBlock(fn.Body)

	// Check return type
	if fn.ReturnType != nil {
		// TODO: Check that all code paths return a value
	}
}

// analyzeBlock analyzes a block of statements
func (sa *SemanticAnalyzer) analyzeBlock(block *BlockStmt) {
	for _, stmt := range block.Statements {
		sa.analyzeStatement(stmt)
	}
}

// analyzeStatement analyzes a statement
func (sa *SemanticAnalyzer) analyzeStatement(stmt Statement) {
	switch s := stmt.(type) {
	case *LetStmt:
		sa.analyzeLetStmt(s)
	case *AssignStmt:
		sa.analyzeAssignStmt(s)
	case *CompoundAssignStmt:
		sa.analyzeCompoundAssignStmt(s)
	case *ReturnStmt:
		sa.analyzeReturnStmt(s)
	case *IfStmt:
		sa.analyzeIfStmt(s)
	case *LoopStmt:
		sa.analyzeLoopStmt(s)
	case *WhileStmt:
		sa.analyzeWhileStmt(s)
	case *BreakStmt:
		if !sa.inLoop {
			sa.addError("break statement outside of loop")
		}
	case *ContinueStmt:
		if !sa.inLoop {
			sa.addError("continue statement outside of loop")
		}
	case *EmitStmt:
		sa.analyzeEmitStmt(s)
	case *AssertStmt:
		sa.analyzeAssertStmt(s)
	case *RequireStmt:
		sa.analyzeRequireStmt(s)
	case *RevertStmt:
		sa.analyzeRevertStmt(s)
	case *ExprStmt:
		sa.getExpressionType(s.Expr)
	case *BlockStmt:
		sa.scopes.PushScope()
		sa.analyzeBlock(s)
		sa.scopes.PopScope()
	}
}

// analyzeLetStmt analyzes a let statement
func (sa *SemanticAnalyzer) analyzeLetStmt(stmt *LetStmt) {
	// Check value type
	valueType := sa.getExpressionType(stmt.Value)
	if valueType == nil {
		return
	}

	// If type annotation exists, check compatibility
	if stmt.Type != nil {
		if !sa.typesMatch(stmt.Type, valueType) {
			sa.addError(fmt.Sprintf("type mismatch in variable %s: declared as %s, assigned %s",
				stmt.Name, stmt.Type.TypeName(), valueType.TypeName()))
		}
	} else {
		// Infer type from value
		stmt.Type = valueType
	}

	// Define variable in scope
	if !sa.scopes.Define(stmt.Name, &Symbol{
		Name:      stmt.Name,
		Type:      stmt.Type,
		Kind:      SymbolVar,
		IsMutable: stmt.IsMut,
	}) {
		sa.addError(fmt.Sprintf("variable %s already defined", stmt.Name))
	}
}

// analyzeAssignStmt analyzes an assignment statement
func (sa *SemanticAnalyzer) analyzeAssignStmt(stmt *AssignStmt) {
	targetType := sa.getExpressionType(stmt.Target)
	valueType := sa.getExpressionType(stmt.Value)

	if targetType == nil || valueType == nil {
		return
	}

	// Check type compatibility
	if !sa.typesMatch(targetType, valueType) {
		sa.addError(fmt.Sprintf("cannot assign %s to %s",
			valueType.TypeName(), targetType.TypeName()))
	}

	// Check mutability
	if ident, ok := stmt.Target.(*Identifier); ok {
		symbol := sa.scopes.Lookup(ident.Name)
		if symbol != nil && !symbol.IsMutable && symbol.Kind == SymbolVar {
			sa.addError(fmt.Sprintf("cannot assign to immutable variable %s", ident.Name))
		}
	}

	// Check view function constraint
	if sa.currentFunction != nil && sa.currentFunction.IsView {
		if memberExpr, ok := stmt.Target.(*MemberExpr); ok {
			if ident, ok := memberExpr.Object.(*Identifier); ok && ident.Name == "self" {
				sa.addError("view functions cannot modify state")
			}
		}
	}
}

// analyzeCompoundAssignStmt analyzes compound assignment
func (sa *SemanticAnalyzer) analyzeCompoundAssignStmt(stmt *CompoundAssignStmt) {
	targetType := sa.getExpressionType(stmt.Target)
	valueType := sa.getExpressionType(stmt.Value)

	if targetType == nil || valueType == nil {
		return
	}

	// Check type compatibility
	if !sa.typesMatch(targetType, valueType) {
		sa.addError(fmt.Sprintf("type mismatch in compound assignment: %s and %s",
			targetType.TypeName(), valueType.TypeName()))
	}

	// Compound assignment requires numeric types
	if !isNumericType(targetType.TypeName()) {
		sa.addError(fmt.Sprintf("compound assignment requires numeric type, got %s",
			targetType.TypeName()))
	}
}

// analyzeReturnStmt analyzes a return statement
func (sa *SemanticAnalyzer) analyzeReturnStmt(stmt *ReturnStmt) {
	if sa.currentFunction == nil {
		return
	}

	if stmt.Value == nil {
		if sa.currentFunction.ReturnType != nil {
			sa.addError(fmt.Sprintf("function %s expects return value of type %s",
				sa.currentFunction.Name, sa.currentFunction.ReturnType.TypeName()))
		}
		return
	}

	valueType := sa.getExpressionType(stmt.Value)
	if valueType == nil {
		return
	}

	if sa.currentFunction.ReturnType == nil {
		sa.addError(fmt.Sprintf("function %s does not have a return type",
			sa.currentFunction.Name))
		return
	}

	if !sa.typesMatch(sa.currentFunction.ReturnType, valueType) {
		sa.addError(fmt.Sprintf("return type mismatch: expected %s, got %s",
			sa.currentFunction.ReturnType.TypeName(), valueType.TypeName()))
	}
}

// analyzeIfStmt analyzes an if statement
func (sa *SemanticAnalyzer) analyzeIfStmt(stmt *IfStmt) {
	// Check condition is boolean
	condType := sa.getExpressionType(stmt.Condition)
	if condType != nil && condType.TypeName() != "bool" {
		sa.addError(fmt.Sprintf("if condition must be boolean, got %s", condType.TypeName()))
	}

	// Analyze branches
	sa.scopes.PushScope()
	sa.analyzeBlock(stmt.ThenBranch)
	sa.scopes.PopScope()

	if stmt.ElseBranch != nil {
		sa.scopes.PushScope()
		sa.analyzeStatement(stmt.ElseBranch)
		sa.scopes.PopScope()
	}
}

// analyzeLoopStmt analyzes a loop statement
func (sa *SemanticAnalyzer) analyzeLoopStmt(stmt *LoopStmt) {
	// Check range bounds are numeric
	startType := sa.getExpressionType(stmt.Start)
	endType := sa.getExpressionType(stmt.End)

	if startType != nil && !isNumericType(startType.TypeName()) {
		sa.addError("loop start must be numeric")
	}
	if endType != nil && !isNumericType(endType.TypeName()) {
		sa.addError("loop end must be numeric")
	}

	// Analyze body with iterator in scope
	sa.scopes.PushScope()
	sa.scopes.Define(stmt.Iterator, &Symbol{
		Name:      stmt.Iterator,
		Type:      &PrimitiveType{Name: "u64"},
		Kind:      SymbolVar,
		IsMutable: false,
	})

	prevInLoop := sa.inLoop
	sa.inLoop = true
	sa.analyzeBlock(stmt.Body)
	sa.inLoop = prevInLoop

	sa.scopes.PopScope()
}

// analyzeWhileStmt analyzes a while statement
func (sa *SemanticAnalyzer) analyzeWhileStmt(stmt *WhileStmt) {
	// Check condition is boolean
	condType := sa.getExpressionType(stmt.Condition)
	if condType != nil && condType.TypeName() != "bool" {
		sa.addError(fmt.Sprintf("while condition must be boolean, got %s", condType.TypeName()))
	}

	// Analyze body
	sa.scopes.PushScope()
	prevInLoop := sa.inLoop
	sa.inLoop = true
	sa.analyzeBlock(stmt.Body)
	sa.inLoop = prevInLoop
	sa.scopes.PopScope()
}

// analyzeEmitStmt analyzes an emit statement
func (sa *SemanticAnalyzer) analyzeEmitStmt(stmt *EmitStmt) {
	// Check event exists
	symbol := sa.scopes.Lookup(stmt.EventName)
	if symbol == nil || symbol.Kind != SymbolEvent {
		sa.addError(fmt.Sprintf("undefined event: %s", stmt.EventName))
		return
	}

	// TODO: Validate event fields match declaration
}

// analyzeAssertStmt analyzes an assert statement
func (sa *SemanticAnalyzer) analyzeAssertStmt(stmt *AssertStmt) {
	// Check condition is boolean
	condType := sa.getExpressionType(stmt.Condition)
	if condType != nil && condType.TypeName() != "bool" {
		sa.addError("assert condition must be boolean")
	}

	// Check message is string
	if stmt.Message != nil {
		msgType := sa.getExpressionType(stmt.Message)
		if msgType != nil && msgType.TypeName() != "string" {
			sa.addError("assert message must be string")
		}
	}
}

// analyzeRequireStmt analyzes a require statement
func (sa *SemanticAnalyzer) analyzeRequireStmt(stmt *RequireStmt) {
	// Check condition is boolean
	condType := sa.getExpressionType(stmt.Condition)
	if condType != nil && condType.TypeName() != "bool" {
		sa.addError("require condition must be boolean")
	}

	// Check message is string
	if stmt.Message != nil {
		msgType := sa.getExpressionType(stmt.Message)
		if msgType != nil && msgType.TypeName() != "string" {
			sa.addError("require message must be string")
		}
	}
}

// analyzeRevertStmt analyzes a revert statement
func (sa *SemanticAnalyzer) analyzeRevertStmt(stmt *RevertStmt) {
	// Check message is string
	msgType := sa.getExpressionType(stmt.Message)
	if msgType != nil && msgType.TypeName() != "string" {
		sa.addError("revert message must be string")
	}
}

// PrettyPrint returns formatted semantic errors
func (sa *SemanticAnalyzer) PrettyPrint() string {
	if len(sa.errors) == 0 {
		return "✅ No semantic errors"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("❌ Found %d semantic error(s):\n\n", len(sa.errors)))
	for i, err := range sa.errors {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, err))
	}
	return sb.String()
}
