package compiler

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

// Opcode represents a KodeChain VM opcode
type Opcode byte

// KodeChain VM Opcodes (based on EVM with extensions)
const (
	// Arithmetic opcodes (0x00-0x0f)
	OP_STOP     Opcode = 0x00
	OP_ADD      Opcode = 0x01
	OP_MUL      Opcode = 0x02
	OP_SUB      Opcode = 0x03
	OP_DIV      Opcode = 0x04
	OP_SDIV     Opcode = 0x05
	OP_MOD      Opcode = 0x06
	OP_SMOD     Opcode = 0x07
	OP_ADDMOD   Opcode = 0x08
	OP_MULMOD   Opcode = 0x09
	OP_EXP      Opcode = 0x0a

	// Comparison & bitwise opcodes (0x10-0x1f)
	OP_LT     Opcode = 0x10
	OP_GT     Opcode = 0x11
	OP_SLT    Opcode = 0x12
	OP_SGT    Opcode = 0x13
	OP_EQ     Opcode = 0x14
	OP_ISZERO Opcode = 0x15
	OP_AND    Opcode = 0x16
	OP_OR     Opcode = 0x17
	OP_XOR    Opcode = 0x18
	OP_NOT    Opcode = 0x19

	// Stack, memory, storage opcodes (0x50-0x5f)
	OP_POP      Opcode = 0x50
	OP_MLOAD    Opcode = 0x51
	OP_MSTORE   Opcode = 0x52
	OP_MSTORE8  Opcode = 0x53
	OP_SLOAD    Opcode = 0x54
	OP_SSTORE   Opcode = 0x55
	OP_JUMP     Opcode = 0x56
	OP_JUMPI    Opcode = 0x57
	OP_PC       Opcode = 0x58
	OP_MSIZE    Opcode = 0x59
	OP_GAS      Opcode = 0x5a
	OP_JUMPDEST Opcode = 0x5b

	// Push opcodes (0x60-0x7f)
	OP_PUSH1  Opcode = 0x60
	OP_PUSH2  Opcode = 0x61
	OP_PUSH4  Opcode = 0x63
	OP_PUSH8  Opcode = 0x67
	OP_PUSH32 Opcode = 0x7f

	// Dup opcodes (0x80-0x8f)
	OP_DUP1  Opcode = 0x80
	OP_DUP2  Opcode = 0x81
	OP_DUP3  Opcode = 0x82
	OP_DUP4  Opcode = 0x83
	OP_DUP16 Opcode = 0x8f

	// Swap opcodes (0x90-0x9f)
	OP_SWAP1  Opcode = 0x90
	OP_SWAP2  Opcode = 0x91
	OP_SWAP3  Opcode = 0x92
	OP_SWAP16 Opcode = 0x9f

	// Log opcodes (0xa0-0xa4)
	OP_LOG0 Opcode = 0xa0
	OP_LOG1 Opcode = 0xa1
	OP_LOG2 Opcode = 0xa2
	OP_LOG3 Opcode = 0xa3
	OP_LOG4 Opcode = 0xa4

	// System opcodes (0xf0-0xff)
	OP_CREATE       Opcode = 0xf0
	OP_CALL         Opcode = 0xf1
	OP_CALLCODE     Opcode = 0xf2
	OP_RETURN       Opcode = 0xf3
	OP_DELEGATECALL Opcode = 0xf4
	OP_STATICCALL   Opcode = 0xfa
	OP_REVERT       Opcode = 0xfd
	OP_INVALID      Opcode = 0xfe
	OP_SELFDESTRUCT Opcode = 0xff

	// KodeChain custom opcodes for staking (0xc0-0xc3)
	OP_GETSTAKE        Opcode = 0xc0 // Get validator stake
	OP_ISVALIDATOR     Opcode = 0xc1 // Check if address is validator
	OP_DISTRIBUTEFEES  Opcode = 0xc2 // Distribute fees to validator
	OP_GETVALIDATORINFO Opcode = 0xc3 // Get validator info

	// Context opcodes (0x30-0x3f)
	OP_ADDRESS      Opcode = 0x30
	OP_BALANCE      Opcode = 0x31
	OP_ORIGIN       Opcode = 0x32
	OP_CALLER       Opcode = 0x33
	OP_CALLVALUE    Opcode = 0x34
	OP_CALLDATALOAD Opcode = 0x35
	OP_CALLDATASIZE Opcode = 0x36
	OP_CALLDATACOPY Opcode = 0x37
	OP_CODESIZE     Opcode = 0x38
	OP_CODECOPY     Opcode = 0x39
	OP_GASPRICE     Opcode = 0x3a
	OP_COINBASE     Opcode = 0x41 // Block validator
	OP_TIMESTAMP    Opcode = 0x42
	OP_NUMBER       Opcode = 0x43
)

// BytecodeGenerator generates bytecode from AST
type BytecodeGenerator struct {
	program       *Program
	code          []byte
	data          []byte
	labels        map[string]int
	labelCounter  int
	functions     map[string]int // function name -> code offset
	stateVarSlots map[string]int // state variable -> storage slot
	localVars     map[string]int // local variable -> stack offset
	errors        []string
}

// NewBytecodeGenerator creates a new bytecode generator
func NewBytecodeGenerator() *BytecodeGenerator {
	return &BytecodeGenerator{
		code:          []byte{},
		data:          []byte{},
		labels:        make(map[string]int),
		functions:     make(map[string]int),
		stateVarSlots: make(map[string]int),
		localVars:     make(map[string]int),
		errors:        []string{},
	}
}

// Errors returns the list of codegen errors
func (bg *BytecodeGenerator) Errors() []string {
	return bg.errors
}

// addError adds a codegen error
func (bg *BytecodeGenerator) addError(msg string) {
	bg.errors = append(bg.errors, msg)
}

// emit emits a single opcode
func (bg *BytecodeGenerator) emit(op Opcode) {
	bg.code = append(bg.code, byte(op))
}

// emitBytes emits raw bytes
func (bg *BytecodeGenerator) emitBytes(data []byte) {
	bg.code = append(bg.code, data...)
}

// emitPush emits a PUSH instruction with data
func (bg *BytecodeGenerator) emitPush(value *big.Int) {
	bytes := value.Bytes()
	size := len(bytes)

	if size == 0 {
		bg.emit(OP_PUSH1)
		bg.emitBytes([]byte{0})
		return
	}

	// Pad to appropriate size
	if size <= 1 {
		bg.emit(OP_PUSH1)
		if len(bytes) == 0 {
			bg.emitBytes([]byte{0})
		} else {
			bg.emitBytes(bytes)
		}
	} else if size <= 2 {
		bg.emit(OP_PUSH2)
		padded := make([]byte, 2)
		copy(padded[2-len(bytes):], bytes)
		bg.emitBytes(padded)
	} else if size <= 4 {
		bg.emit(OP_PUSH4)
		padded := make([]byte, 4)
		copy(padded[4-len(bytes):], bytes)
		bg.emitBytes(padded)
	} else if size <= 8 {
		bg.emit(OP_PUSH8)
		padded := make([]byte, 8)
		copy(padded[8-len(bytes):], bytes)
		bg.emitBytes(padded)
	} else {
		bg.emit(OP_PUSH32)
		padded := make([]byte, 32)
		copy(padded[32-len(bytes):], bytes)
		bg.emitBytes(padded)
	}
}

// emitPushInt emits a PUSH instruction with an integer
func (bg *BytecodeGenerator) emitPushInt(value int64) {
	bg.emitPush(big.NewInt(value))
}

// createLabel creates a new unique label
func (bg *BytecodeGenerator) createLabel(prefix string) string {
	label := fmt.Sprintf("%s_%d", prefix, bg.labelCounter)
	bg.labelCounter++
	return label
}

// markLabel marks the current position with a label
func (bg *BytecodeGenerator) markLabel(label string) {
	bg.labels[label] = len(bg.code)
}

// ===========================
// BYTECODE GENERATION
// ===========================

// Generate generates bytecode from a program
func (bg *BytecodeGenerator) Generate(program *Program) ([]byte, error) {
	bg.program = program

	// For now, we'll generate bytecode for the first contract
	if len(program.Contracts) == 0 {
		return nil, fmt.Errorf("no contracts to compile")
	}

	contract := program.Contracts[0]
	bg.generateContract(contract)

	if len(bg.errors) > 0 {
		return nil, fmt.Errorf("bytecode generation failed with %d errors", len(bg.errors))
	}

	return bg.code, nil
}

// generateContract generates bytecode for a contract
func (bg *BytecodeGenerator) generateContract(contract *ContractDecl) {
	// Assign storage slots to state variables
	if contract.State != nil {
		for i, stateVar := range contract.State.Variables {
			bg.stateVarSlots[stateVar.Name] = i
		}
	}

	// Generate constructor code (init function)
	if contract.Init != nil {
		bg.generateInitFunction(contract.Init)
	}

	// Generate function dispatch table
	bg.generateFunctionDispatcher(contract)

	// Generate each function
	for _, fn := range contract.Functions {
		if fn.Visibility == "pub" {
			bg.generateFunction(fn)
		}
	}
}

// generateInitFunction generates bytecode for init (constructor)
func (bg *BytecodeGenerator) generateInitFunction(init *InitFunction) {
	// Constructor runs once during deployment
	// For now, just generate the body
	bg.generateBlock(init.Body)
}

// generateFunctionDispatcher generates the function selector dispatch table
func (bg *BytecodeGenerator) generateFunctionDispatcher(contract *ContractDecl) {
	// Get function selector from calldata[0:4]
	bg.emitPushInt(0)
	bg.emit(OP_CALLDATALOAD)
	bg.emitPushInt(224) // Shift right 224 bits (28 bytes) to get first 4 bytes
	// For simplicity, we'll use a simple jump table

	// For each public function, check selector and jump to function
	for _, fn := range contract.Functions {
		if fn.Visibility != "pub" {
			continue
		}

		// Calculate function selector (first 4 bytes of keccak256(signature))
		// For now, use a simple hash based on function name
		selector := bg.functionSelector(fn.Name)

		// DUP the selector on stack
		bg.emit(OP_DUP1)
		bg.emitPushInt(int64(selector))
		bg.emit(OP_EQ)

		// If equal, jump to function
		funcLabel := fmt.Sprintf("func_%s", fn.Name)
		bg.markLabel(funcLabel)
		// JUMPI would go here, but we'll simplify for now
	}

	// If no function matches, revert
	bg.emit(OP_REVERT)
}

// functionSelector generates a simple function selector
func (bg *BytecodeGenerator) functionSelector(name string) uint32 {
	// Simple hash function for demo purposes
	// In production, this would be keccak256(signature)[:4]
	hash := uint32(0)
	for _, c := range name {
		hash = hash*31 + uint32(c)
	}
	return hash
}

// generateFunction generates bytecode for a function
func (bg *BytecodeGenerator) generateFunction(fn *Function) {
	funcLabel := fmt.Sprintf("func_%s", fn.Name)
	bg.markLabel(funcLabel)

	// Reset local variable mapping
	bg.localVars = make(map[string]int)

	// Map parameters to stack positions
	for i, param := range fn.Parameters {
		bg.localVars[param.Name] = i
	}

	// Generate function body
	bg.generateBlock(fn.Body)

	// If function doesn't have explicit return, add return
	if fn.ReturnType == nil {
		bg.emitPushInt(0)
		bg.emitPushInt(0)
		bg.emit(OP_RETURN)
	}
}

// generateBlock generates bytecode for a block
func (bg *BytecodeGenerator) generateBlock(block *BlockStmt) {
	for _, stmt := range block.Statements {
		bg.generateStatement(stmt)
	}
}

// generateStatement generates bytecode for a statement
func (bg *BytecodeGenerator) generateStatement(stmt Statement) {
	switch s := stmt.(type) {
	case *LetStmt:
		bg.generateLetStmt(s)
	case *AssignStmt:
		bg.generateAssignStmt(s)
	case *CompoundAssignStmt:
		bg.generateCompoundAssignStmt(s)
	case *ReturnStmt:
		bg.generateReturnStmt(s)
	case *IfStmt:
		bg.generateIfStmt(s)
	case *LoopStmt:
		bg.generateLoopStmt(s)
	case *WhileStmt:
		bg.generateWhileStmt(s)
	case *EmitStmt:
		bg.generateEmitStmt(s)
	case *AssertStmt:
		bg.generateAssertStmt(s)
	case *RequireStmt:
		bg.generateRequireStmt(s)
	case *RevertStmt:
		bg.generateRevertStmt(s)
	case *ExprStmt:
		bg.generateExpression(s.Expr)
		bg.emit(OP_POP) // Discard result
	case *BlockStmt:
		bg.generateBlock(s)
	}
}

// generateLetStmt generates bytecode for let statement
func (bg *BytecodeGenerator) generateLetStmt(stmt *LetStmt) {
	// Generate value expression
	bg.generateExpression(stmt.Value)

	// Store in local variable (stack position)
	varIndex := len(bg.localVars)
	bg.localVars[stmt.Name] = varIndex
	// Value is now on top of stack
}

// generateAssignStmt generates bytecode for assignment
func (bg *BytecodeGenerator) generateAssignStmt(stmt *AssignStmt) {
	// Generate value
	bg.generateExpression(stmt.Value)

	// Store to target
	switch target := stmt.Target.(type) {
	case *Identifier:
		// Check if it's a local variable or state variable
		if _, isLocal := bg.localVars[target.Name]; isLocal {
			// Local variable - would need proper stack management
			// For now, just pop the value
			bg.emit(OP_POP)
		} else if slot, isState := bg.stateVarSlots[target.Name]; isState {
			// State variable - store in storage
			bg.emitPushInt(int64(slot))
			bg.emit(OP_SSTORE)
		}

	case *MemberExpr:
		// Handle self.field assignment
		if ident, ok := target.Object.(*Identifier); ok && ident.Name == "self" {
			if slot, exists := bg.stateVarSlots[target.Member]; exists {
				bg.emitPushInt(int64(slot))
				bg.emit(OP_SSTORE)
			}
		}

	case *IndexExpr:
		// Handle map/array assignment
		bg.generateIndexAssignment(target)
	}
}

// generateIndexAssignment generates bytecode for index assignment
func (bg *BytecodeGenerator) generateIndexAssignment(indexExpr *IndexExpr) {
	// Generate index
	bg.generateExpression(indexExpr.Index)

	// For map storage, we need to compute storage slot: keccak256(key . slot)
	// This is simplified for demonstration
	bg.emit(OP_SSTORE)
}

// generateCompoundAssignStmt generates bytecode for compound assignment
func (bg *BytecodeGenerator) generateCompoundAssignStmt(stmt *CompoundAssignStmt) {
	// Load current value
	bg.generateExpression(stmt.Target)

	// Generate value to add/subtract/etc
	bg.generateExpression(stmt.Value)

	// Perform operation
	switch stmt.Operator {
	case "+=":
		bg.emit(OP_ADD)
	case "-=":
		bg.emit(OP_SUB)
	case "*=":
		bg.emit(OP_MUL)
	case "/=":
		bg.emit(OP_DIV)
	case "%=":
		bg.emit(OP_MOD)
	}

	// Store result back (simplified)
	bg.emit(OP_POP)
}

// generateReturnStmt generates bytecode for return
func (bg *BytecodeGenerator) generateReturnStmt(stmt *ReturnStmt) {
	if stmt.Value != nil {
		bg.generateExpression(stmt.Value)
		// Store return value in memory and return
		bg.emitPushInt(0)
		bg.emit(OP_MSTORE)
		bg.emitPushInt(32)
		bg.emitPushInt(0)
		bg.emit(OP_RETURN)
	} else {
		bg.emitPushInt(0)
		bg.emitPushInt(0)
		bg.emit(OP_RETURN)
	}
}

// generateIfStmt generates bytecode for if statement
func (bg *BytecodeGenerator) generateIfStmt(stmt *IfStmt) {
	elseLabel := bg.createLabel("else")
	endLabel := bg.createLabel("endif")

	// Generate condition
	bg.generateExpression(stmt.Condition)

	// If false, jump to else
	bg.emit(OP_ISZERO)
	// JUMPI to elseLabel (simplified - actual implementation would need label resolution)

	// Generate then branch
	bg.generateBlock(stmt.ThenBranch)

	// Jump to end
	// JUMP to endLabel

	// Else branch
	bg.markLabel(elseLabel)
	if stmt.ElseBranch != nil {
		bg.generateStatement(stmt.ElseBranch)
	}

	bg.markLabel(endLabel)
}

// generateLoopStmt generates bytecode for loop
func (bg *BytecodeGenerator) generateLoopStmt(stmt *LoopStmt) {
	loopLabel := bg.createLabel("loop")
	endLabel := bg.createLabel("endloop")

	// Initialize iterator
	bg.generateExpression(stmt.Start)
	// Store iterator

	bg.markLabel(loopLabel)

	// Check condition: iterator < end
	bg.generateExpression(stmt.End)
	bg.emit(OP_LT)
	bg.emit(OP_ISZERO)
	// JUMPI to endLabel

	// Generate body
	bg.generateBlock(stmt.Body)

	// Increment iterator
	bg.emitPushInt(1)
	bg.emit(OP_ADD)

	// JUMP to loopLabel

	bg.markLabel(endLabel)
}

// generateWhileStmt generates bytecode for while loop
func (bg *BytecodeGenerator) generateWhileStmt(stmt *WhileStmt) {
	loopLabel := bg.createLabel("while")
	endLabel := bg.createLabel("endwhile")

	bg.markLabel(loopLabel)

	// Generate condition
	bg.generateExpression(stmt.Condition)
	bg.emit(OP_ISZERO)
	// JUMPI to endLabel

	// Generate body
	bg.generateBlock(stmt.Body)

	// JUMP to loopLabel

	bg.markLabel(endLabel)
}

// generateEmitStmt generates bytecode for emit event
func (bg *BytecodeGenerator) generateEmitStmt(stmt *EmitStmt) {
	// Calculate event signature hash
	// For now, use simple hash
	eventHash := bg.functionSelector(stmt.EventName)
	bg.emitPushInt(int64(eventHash))

	// Emit event fields
	for _, field := range stmt.Fields {
		bg.generateExpression(field.Value)
	}

	// Emit LOG opcode based on number of topics
	numTopics := 0
	for _, field := range stmt.Fields {
		_ = field // Count indexed fields
		numTopics++
	}

	switch numTopics {
	case 0:
		bg.emit(OP_LOG0)
	case 1:
		bg.emit(OP_LOG1)
	case 2:
		bg.emit(OP_LOG2)
	case 3:
		bg.emit(OP_LOG3)
	case 4:
		bg.emit(OP_LOG4)
	}
}

// generateAssertStmt generates bytecode for assert
func (bg *BytecodeGenerator) generateAssertStmt(stmt *AssertStmt) {
	// Generate condition
	bg.generateExpression(stmt.Condition)

	// If false, revert
	bg.emit(OP_ISZERO)
	// JUMPI to revert

	// Generate error message if provided
	if stmt.Message != nil {
		bg.generateExpression(stmt.Message)
	}

	bg.emit(OP_REVERT)
}

// generateRequireStmt generates bytecode for require
func (bg *BytecodeGenerator) generateRequireStmt(stmt *RequireStmt) {
	// Same as assert
	bg.generateAssertStmt(&AssertStmt{
		Condition: stmt.Condition,
		Message:   stmt.Message,
	})
}

// generateRevertStmt generates bytecode for revert
func (bg *BytecodeGenerator) generateRevertStmt(stmt *RevertStmt) {
	// Generate error message
	bg.generateExpression(stmt.Message)
	bg.emit(OP_REVERT)
}

// generateExpression generates bytecode for an expression
func (bg *BytecodeGenerator) generateExpression(expr Expression) {
	switch e := expr.(type) {
	case *Identifier:
		bg.generateIdentifier(e)
	case *IntegerLiteral:
		bg.generateIntegerLiteral(e)
	case *BoolLiteral:
		bg.generateBoolLiteral(e)
	case *StringLiteral:
		bg.generateStringLiteral(e)
	case *BinaryExpr:
		bg.generateBinaryExpr(e)
	case *UnaryExpr:
		bg.generateUnaryExpr(e)
	case *CallExpr:
		bg.generateCallExpr(e)
	case *MemberExpr:
		bg.generateMemberExpr(e)
	case *IndexExpr:
		bg.generateIndexExpr(e)
	}
}

// generateIdentifier generates bytecode for identifier
func (bg *BytecodeGenerator) generateIdentifier(ident *Identifier) {
	// Check if it's a local variable
	if _, isLocal := bg.localVars[ident.Name]; isLocal {
		// Load from stack (simplified)
		bg.emit(OP_DUP1)
		return
	}

	// Check if it's a state variable
	if slot, isState := bg.stateVarSlots[ident.Name]; isState {
		bg.emitPushInt(int64(slot))
		bg.emit(OP_SLOAD)
		return
	}
}

// generateIntegerLiteral generates bytecode for integer literal
func (bg *BytecodeGenerator) generateIntegerLiteral(lit *IntegerLiteral) {
	value := new(big.Int)
	value.SetString(lit.Value, 0) // Auto-detect base (hex/decimal)
	bg.emitPush(value)
}

// generateBoolLiteral generates bytecode for boolean literal
func (bg *BytecodeGenerator) generateBoolLiteral(lit *BoolLiteral) {
	if lit.Value {
		bg.emitPushInt(1)
	} else {
		bg.emitPushInt(0)
	}
}

// generateStringLiteral generates bytecode for string literal
func (bg *BytecodeGenerator) generateStringLiteral(lit *StringLiteral) {
	// Store string in data section and push offset
	offset := len(bg.data)
	bg.data = append(bg.data, []byte(lit.Value)...)
	bg.emitPushInt(int64(offset))
}

// generateBinaryExpr generates bytecode for binary expression
func (bg *BytecodeGenerator) generateBinaryExpr(expr *BinaryExpr) {
	// Generate left operand
	bg.generateExpression(expr.Left)

	// Generate right operand
	bg.generateExpression(expr.Right)

	// Generate operation
	switch expr.Operator {
	case "+":
		bg.emit(OP_ADD)
	case "-":
		bg.emit(OP_SUB)
	case "*":
		bg.emit(OP_MUL)
	case "/":
		bg.emit(OP_DIV)
	case "%":
		bg.emit(OP_MOD)
	case "==":
		bg.emit(OP_EQ)
	case "!=":
		bg.emit(OP_EQ)
		bg.emit(OP_ISZERO)
	case "<":
		bg.emit(OP_LT)
	case ">":
		bg.emit(OP_GT)
	case "<=":
		bg.emit(OP_GT)
		bg.emit(OP_ISZERO)
	case ">=":
		bg.emit(OP_LT)
		bg.emit(OP_ISZERO)
	case "&&":
		bg.emit(OP_AND)
	case "||":
		bg.emit(OP_OR)
	}
}

// generateUnaryExpr generates bytecode for unary expression
func (bg *BytecodeGenerator) generateUnaryExpr(expr *UnaryExpr) {
	bg.generateExpression(expr.Operand)

	switch expr.Operator {
	case "!":
		bg.emit(OP_ISZERO)
	case "-":
		// Negate: 0 - x
		bg.emitPushInt(0)
		bg.emit(OP_SWAP1)
		bg.emit(OP_SUB)
	}
}

// generateCallExpr generates bytecode for function call
func (bg *BytecodeGenerator) generateCallExpr(expr *CallExpr) {
	// Handle member calls (e.g., precompiled.getStake)
	if memberExpr, ok := expr.Function.(*MemberExpr); ok {
		bg.generateMemberCall(memberExpr, expr.Arguments)
		return
	}

	// Generate arguments
	for _, arg := range expr.Arguments {
		bg.generateExpression(arg)
	}

	// Call function (simplified - would need proper call handling)
	bg.emit(OP_CALL)
}

// generateMemberCall generates bytecode for member function call
func (bg *BytecodeGenerator) generateMemberCall(memberExpr *MemberExpr, args []Expression) {
	// Check if it's a precompiled call
	if ident, ok := memberExpr.Object.(*Identifier); ok && ident.Name == "precompiled" {
		bg.generatePrecompiledCall(memberExpr.Member, args)
		return
	}
}

// generatePrecompiledCall generates bytecode for precompiled contract call
func (bg *BytecodeGenerator) generatePrecompiledCall(method string, args []Expression) {
	// Generate arguments
	for _, arg := range args {
		bg.generateExpression(arg)
	}

	// Emit custom KodeChain opcode
	switch method {
	case "getStake":
		bg.emit(OP_GETSTAKE)
	case "isValidator":
		bg.emit(OP_ISVALIDATOR)
	case "distributeFees":
		bg.emit(OP_DISTRIBUTEFEES)
	case "getValidatorInfo":
		bg.emit(OP_GETVALIDATORINFO)
	}
}

// generateMemberExpr generates bytecode for member access
func (bg *BytecodeGenerator) generateMemberExpr(expr *MemberExpr) {
	if ident, ok := expr.Object.(*Identifier); ok {
		switch ident.Name {
		case "msg":
			bg.generateMsgMember(expr.Member)
		case "block":
			bg.generateBlockMember(expr.Member)
		case "tx":
			bg.generateTxMember(expr.Member)
		case "self":
			bg.generateSelfMember(expr.Member)
		}
	}
}

// generateMsgMember generates bytecode for msg.* access
func (bg *BytecodeGenerator) generateMsgMember(member string) {
	switch member {
	case "sender":
		bg.emit(OP_CALLER)
	case "value":
		bg.emit(OP_CALLVALUE)
	case "gas":
		bg.emit(OP_GAS)
	}
}

// generateBlockMember generates bytecode for block.* access
func (bg *BytecodeGenerator) generateBlockMember(member string) {
	switch member {
	case "number":
		bg.emit(OP_NUMBER)
	case "timestamp":
		bg.emit(OP_TIMESTAMP)
	case "validator":
		bg.emit(OP_COINBASE)
	}
}

// generateTxMember generates bytecode for tx.* access
func (bg *BytecodeGenerator) generateTxMember(member string) {
	switch member {
	case "origin":
		bg.emit(OP_ORIGIN)
	case "gasprice":
		bg.emit(OP_GASPRICE)
	}
}

// generateSelfMember generates bytecode for self.* access
func (bg *BytecodeGenerator) generateSelfMember(member string) {
	switch member {
	case "address":
		bg.emit(OP_ADDRESS)
	case "balance":
		bg.emit(OP_BALANCE)
	default:
		// State variable access
		if slot, exists := bg.stateVarSlots[member]; exists {
			bg.emitPushInt(int64(slot))
			bg.emit(OP_SLOAD)
		}
	}
}

// generateIndexExpr generates bytecode for index access
func (bg *BytecodeGenerator) generateIndexExpr(expr *IndexExpr) {
	// Generate object
	bg.generateExpression(expr.Object)

	// Generate index
	bg.generateExpression(expr.Index)

	// Load from storage (simplified - would need proper map/array handling)
	bg.emit(OP_SLOAD)
}

// GetBytecode returns the generated bytecode
func (bg *BytecodeGenerator) GetBytecode() []byte {
	return bg.code
}

// DisassembleResult represents disassembled bytecode
type DisassembleResult struct {
	Offset int
	Opcode string
	Data   string
}

// Disassemble disassembles bytecode to human-readable format
func (bg *BytecodeGenerator) Disassemble() []DisassembleResult {
	result := []DisassembleResult{}
	i := 0

	for i < len(bg.code) {
		op := Opcode(bg.code[i])
		offset := i
		opName := opcodeToString(op)
		data := ""

		i++

		// Handle PUSH opcodes that have data
		if op >= OP_PUSH1 && op <= OP_PUSH32 {
			size := int(op - OP_PUSH1 + 1)
			if i+size <= len(bg.code) {
				dataBytes := bg.code[i : i+size]
				data = fmt.Sprintf("0x%x", dataBytes)
				i += size
			}
		}

		result = append(result, DisassembleResult{
			Offset: offset,
			Opcode: opName,
			Data:   data,
		})
	}

	return result
}

// opcodeToString converts opcode to string name
func opcodeToString(op Opcode) string {
	names := map[Opcode]string{
		OP_STOP:             "STOP",
		OP_ADD:              "ADD",
		OP_MUL:              "MUL",
		OP_SUB:              "SUB",
		OP_DIV:              "DIV",
		OP_MOD:              "MOD",
		OP_LT:               "LT",
		OP_GT:               "GT",
		OP_EQ:               "EQ",
		OP_ISZERO:           "ISZERO",
		OP_AND:              "AND",
		OP_OR:               "OR",
		OP_NOT:              "NOT",
		OP_POP:              "POP",
		OP_SLOAD:            "SLOAD",
		OP_SSTORE:           "SSTORE",
		OP_JUMP:             "JUMP",
		OP_JUMPI:            "JUMPI",
		OP_JUMPDEST:         "JUMPDEST",
		OP_PUSH1:            "PUSH1",
		OP_PUSH2:            "PUSH2",
		OP_PUSH4:            "PUSH4",
		OP_PUSH8:            "PUSH8",
		OP_PUSH32:           "PUSH32",
		OP_DUP1:             "DUP1",
		OP_SWAP1:            "SWAP1",
		OP_LOG0:             "LOG0",
		OP_LOG1:             "LOG1",
		OP_LOG2:             "LOG2",
		OP_RETURN:           "RETURN",
		OP_REVERT:           "REVERT",
		OP_GETSTAKE:         "GETSTAKE",
		OP_ISVALIDATOR:      "ISVALIDATOR",
		OP_DISTRIBUTEFEES:   "DISTRIBUTEFEES",
		OP_GETVALIDATORINFO: "GETVALIDATORINFO",
		OP_CALLER:           "CALLER",
		OP_CALLVALUE:        "CALLVALUE",
		OP_GAS:              "GAS",
		OP_COINBASE:         "COINBASE",
		OP_TIMESTAMP:        "TIMESTAMP",
		OP_NUMBER:           "NUMBER",
		OP_ORIGIN:           "ORIGIN",
		OP_GASPRICE:         "GASPRICE",
		OP_ADDRESS:          "ADDRESS",
		OP_BALANCE:          "BALANCE",
	}

	if name, ok := names[op]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN(0x%02x)", byte(op))
}

// ToJSON exports bytecode metadata to JSON
func (bg *BytecodeGenerator) ToJSON() (string, error) {
	metadata := map[string]interface{}{
		"bytecode": fmt.Sprintf("0x%x", bg.code),
		"length":   len(bg.code),
		"functions": bg.functions,
		"stateVars": bg.stateVarSlots,
	}

	jsonBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// PrettyPrint returns formatted bytecode information
func (bg *BytecodeGenerator) PrettyPrint() string {
	var sb strings.Builder

	sb.WriteString("═══════════════════════════════════════════\n")
	sb.WriteString("BYTECODE GENERATION COMPLETE\n")
	sb.WriteString("═══════════════════════════════════════════\n\n")

	sb.WriteString(fmt.Sprintf("Bytecode Size: %d bytes\n", len(bg.code)))
	sb.WriteString(fmt.Sprintf("Data Size: %d bytes\n", len(bg.data)))
	sb.WriteString(fmt.Sprintf("Functions: %d\n", len(bg.functions)))
	sb.WriteString(fmt.Sprintf("State Variables: %d\n\n", len(bg.stateVarSlots)))

	sb.WriteString("Disassembly:\n")
	sb.WriteString("─────────────────────────────────────────\n")

	for _, instr := range bg.Disassemble() {
		if instr.Data != "" {
			sb.WriteString(fmt.Sprintf("%04d: %s %s\n", instr.Offset, instr.Opcode, instr.Data))
		} else {
			sb.WriteString(fmt.Sprintf("%04d: %s\n", instr.Offset, instr.Opcode))
		}
	}

	return sb.String()
}

// WriteBytecodeToFile writes bytecode to a binary file
func (bg *BytecodeGenerator) WriteBytecodeToFile(filename string) error {
	// This would write to file system in a real implementation
	return nil
}
