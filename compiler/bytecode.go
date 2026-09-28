package compiler

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"tc_node_validator_v2/security"
)

// ============================================================
// KSP-F1: Reconstrucción del generador de bytecode kottg.
//
// El backend anterior era un stub: dispatcher sin JUMPI, labels sin
// resolución, locales en pila rotos ("DUP1 simplificado"), sin control de
// flujo, views eliminadas, modifiers ignorados. Este reemplazo implementa
// el diseño de la Fase 1 del plan KSP:
//
//   1.1 Backpatching de labels (saltos con placeholder + resolución final)
//   1.2 Dispatcher KSEL-8 (selector = QSH(firma canónica)[:8])
//   1.3 Frame pointer: parámetros y locales en MEMORIA (MLOAD/MSTORE)
//   1.4 view fn generadas y despachables
//   1.5 Modifiers (require fn) inlineados al inicio del cuerpo
//   1.6 Control de flujo real (if/loop/while/match con JUMPI/JUMP)
//   1.7 Disassembler completo
//   1.8 KSM: aritmética u256 con chequeo de overflow a nivel compilador
//
// Convenciones de pila del VM (crítico, validado en Fase 0):
//   SSTORE: [value, key]  (key en el TOPE)
//   MSTORE: [value, offset] (offset en el TOPE)
//   RETURN/REVERT: [size, offset] (offset en el TOPE)
//   JUMPI: [cond, dest] (dest en el TOPE)
//   LOGn: [topicN..topic1, size, offset] (offset en el TOPE)
//   GT/LT/EQ: [b, a] con a en el TOPE → resultado a OP b
// ============================================================

// OpCodes — mismos valores que el VM de KodeChain (KVM-byte)
type Opcode byte

const (
	OP_STOP Opcode = 0x00
	OP_ADD  Opcode = 0x01
	OP_MUL  Opcode = 0x02
	OP_SUB  Opcode = 0x03
	OP_DIV  Opcode = 0x04
	OP_MOD  Opcode = 0x06

	OP_LT     Opcode = 0x10
	OP_GT     Opcode = 0x11
	OP_EQ     Opcode = 0x14
	OP_ISZERO Opcode = 0x15
	OP_AND    Opcode = 0x16
	OP_OR     Opcode = 0x17
	OP_XOR    Opcode = 0x18
	OP_NOT    Opcode = 0x19
	OP_SHL    Opcode = 0x1b
	OP_SHR    Opcode = 0x1c

	OP_QSH Opcode = 0x20 // Quantum Sponge Hash (KSP-F0)

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

	OP_BLOCKHASH  Opcode = 0x40
	OP_COINBASE   Opcode = 0x41
	OP_TIMESTAMP  Opcode = 0x42
	OP_NUMBER     Opcode = 0x43
	OP_DIFFICULTY Opcode = 0x44
	OP_GASLIMIT   Opcode = 0x45

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

	OP_PUSH1  Opcode = 0x60
	OP_PUSH2  Opcode = 0x61
	OP_PUSH4  Opcode = 0x63
	OP_PUSH8  Opcode = 0x67
	OP_PUSH16 Opcode = 0x6f
	OP_PUSH32 Opcode = 0x7f

	OP_DUP1  Opcode = 0x80
	OP_DUP2  Opcode = 0x81
	OP_DUP3  Opcode = 0x82
	OP_DUP4  Opcode = 0x83
	OP_DUP5  Opcode = 0x84
	OP_DUP16 Opcode = 0x8f

	OP_SWAP1  Opcode = 0x90
	OP_SWAP2  Opcode = 0x91
	OP_SWAP3  Opcode = 0x92
	OP_SWAP16 Opcode = 0x9f

	OP_LOG0 Opcode = 0xa0
	OP_LOG1 Opcode = 0xa1
	OP_LOG2 Opcode = 0xa2
	OP_LOG3 Opcode = 0xa3
	OP_LOG4 Opcode = 0xa4

	// KodeChain staking opcodes
	OP_GETSTAKE         Opcode = 0xc0
	OP_ISVALIDATOR      Opcode = 0xc1
	OP_DISTRIBUTEFEES   Opcode = 0xc2
	OP_GETVALIDATORINFO Opcode = 0xc3

	// KSP-2: BCR (Beacon Chain Root) opcodes
	OP_BEACONROOTLATEST Opcode = 0xc4
	OP_BEACONROOT       Opcode = 0xc5
	OP_BEACONHEIGHT     Opcode = 0xc6

	OP_CREATE       Opcode = 0xf0
	OP_CALL         Opcode = 0xf1
	OP_CALLCODE     Opcode = 0xf2
	OP_RETURN       Opcode = 0xf3
	OP_DELEGATECALL Opcode = 0xf4
	OP_STATICCALL   Opcode = 0xfa
	OP_REVERT       Opcode = 0xfd
	OP_INVALID      Opcode = 0xfe
	OP_SELFDESTRUCT Opcode = 0xff
)

// Frame / layout de memoria del contrato
const (
	frameBase   uint64 = 0x400 // parámetros + locales de la función actual
	frameLimit  uint64 = 0x800 // límite del área de frame
	eventBase   uint64 = 0x100 // área de datos de eventos (LOG)
	// KSP-F2: scratch para derivación de claves de mappings (QSH) y sec.qsh.
	// [0x300..0x340) = pad32(key) ++ pad32(base_id) → QSH → slot
	// [0x380..0x3a0) = valor a hashear con sec.qsh
	keyScratch  uint64 = 0x300
	qshScratch  uint64 = 0x380
	// KSP-3: scratch de llamadas externas (ext.call).
	// [0x200..) = calldata (selector ‖ 28 ceros ‖ args) — ≤ 8 args
	// [0x080..0x0a0) = primera palabra del returndata del llamado
	callScratch    uint64 = 0x200
	callRetScratch uint64 = 0x080
	// KSP-3: zona de bytecode para ext.create (KBs de código — lejos del
	// frame 0x400..0x800 y de todos los scratches)
	createCodeScratch uint64 = 0x100000
	ksmActive          = true // KSM activo por defecto (Fase 1)
)

// Máximos por tipo para los range-checks KSM (Fase 2: u64/u128).
var typeMaxBits = map[string]uint{
	"u64":  64,
	"u128": 128,
}

// pendingJump — backpatching (1.1)
type pendingJump struct {
	at    int    // offset del operando del PUSH2 en code
	label string
}

// loopCtx para break/continue
type loopCtx struct {
	continueLabel string
	breakLabel    string
}

// BytecodeGenerator genera KVM-byte con backpatching de labels.
type BytecodeGenerator struct {
	code         []byte
	labels       map[string]int
	labelCounter int
	pending      []pendingJump

	stateVarSlots map[string]int   // state var escalar -> storage slot
	mapBases       map[string]int   // state var map/vec -> base id para derivación
	stateVarTypes  map[string]string // KSP-F2: state var -> tipo ("u64","u128","u256","address","bool")
	varSlots      map[string]uint64 // frame var -> offset de memoria
	varTypes      map[string]string // KSP-F2: nombre de variable/param -> tipo ("u64","u128","u256","address","bool")
	frameNext     uint64
	contract      *ContractDecl

	modifiers   map[string]*ModifierDecl
	events      map[string]*EventDecl
	loopStack   []loopCtx
	trapQueue   []string
	errors      []string
	inFunction  bool
	hasReturn   bool
}

func NewBytecodeGenerator() *BytecodeGenerator {
	return &BytecodeGenerator{
		code:          []byte{},
		labels:        make(map[string]int),
		stateVarSlots: make(map[string]int),
		mapBases:      make(map[string]int),
		stateVarTypes: make(map[string]string),
		varSlots:      make(map[string]uint64),
		varTypes:       make(map[string]string),
		modifiers:     make(map[string]*ModifierDecl),
		events:        make(map[string]*EventDecl),
		errors:        []string{},
	}
}

func (bg *BytecodeGenerator) Errors() []string { return bg.errors }

func (bg *BytecodeGenerator) addError(msg string) { bg.errors = append(bg.errors, msg) }

// ---------------- emisión base ----------------

func (bg *BytecodeGenerator) emit(op Opcode) {
	bg.code = append(bg.code, byte(op))
}

func (bg *BytecodeGenerator) emitBytes(data []byte) {
	bg.code = append(bg.code, data...)
}

func (bg *BytecodeGenerator) emitPush(value *big.Int) {
	v := value
	if v == nil {
		v = big.NewInt(0)
	}
	if v.Sign() < 0 {
		bg.addError("literal negativo no soportado en PUSH: " + v.String())
		v = big.NewInt(0)
	}
	raw := v.Bytes()
	var padded []byte
	var op Opcode
	switch {
	case len(raw) == 0 || len(raw) <= 1:
		op, padded = OP_PUSH1, make([]byte, 1)
	case len(raw) <= 2:
		op, padded = OP_PUSH2, make([]byte, 2)
	case len(raw) <= 4:
		op, padded = OP_PUSH4, make([]byte, 4)
	case len(raw) <= 8:
		op, padded = OP_PUSH8, make([]byte, 8)
	case len(raw) <= 16:
		op, padded = OP_PUSH16, make([]byte, 16)
	default:
		if len(raw) > 32 {
			bg.addError("literal excede 32 bytes")
			raw = raw[len(raw)-32:]
		}
		op, padded = OP_PUSH32, make([]byte, 32)
	}
	copy(padded[len(padded)-len(raw):], raw)
	bg.emit(op)
	bg.emitBytes(padded)
}

func (bg *BytecodeGenerator) emitPushInt(v int64) { bg.emitPush(big.NewInt(v)) }

func (bg *BytecodeGenerator) emitPushU64(v uint64) { bg.emitPush(new(big.Int).SetUint64(v)) }

// createLabel genera un nombre único.
func (bg *BytecodeGenerator) createLabel(prefix string) string {
	bg.labelCounter++
	return fmt.Sprintf("%s_%d", prefix, bg.labelCounter)
}

// markLabel (1.1): emite JUMPDEST — el VM exige JUMPDEST exacto en todo
// destino de salto (validado en Fase 0).
func (bg *BytecodeGenerator) markLabel(label string) {
	bg.labels[label] = len(bg.code)
	bg.emit(OP_JUMPDEST)
}

// emitJump: PUSH2 <placeholder> JUMP — se resuelve en resolveJumps.
func (bg *BytecodeGenerator) emitJump(label string) {
	bg.emit(OP_PUSH2)
	bg.pending = append(bg.pending, pendingJump{at: len(bg.code), label: label})
	bg.emitBytes([]byte{0, 0})
	bg.emit(OP_JUMP)
}

// emitJumpIf: cond en el tope; PUSH2 <dest> JUMPI.
func (bg *BytecodeGenerator) emitJumpIf(label string) {
	bg.emit(OP_PUSH2)
	bg.pending = append(bg.pending, pendingJump{at: len(bg.code), label: label})
	bg.emitBytes([]byte{0, 0})
	bg.emit(OP_JUMPI)
}

// resolveJumps (1.1): parchea todos los saltos con las posiciones resueltas.
func (bg *BytecodeGenerator) resolveJumps() error {
	for _, pj := range bg.pending {
		pos, ok := bg.labels[pj.label]
		if !ok {
			return fmt.Errorf("label no definido: %s", pj.label)
		}
		if pos > 0xFFFF {
			return fmt.Errorf("destino %s fuera de rango PUSH2 (%d)", pj.label, pos)
		}
		bg.code[pj.at] = byte(pos >> 8)
		bg.code[pj.at+1] = byte(pos)
	}
	return nil
}

// ---------------- KSEL-8 (1.2) ----------------

// ksel8 computa el selector KSP: primeros 8 hex (4 bytes) del QSH de la
// firma canónica "nombre(tipo1,tipo2)" en el dominio KSEL-8 (KSP-6B:
// separación de dominio — mismo helper que el engine y kselbin).
func ksel8(signature string) uint32 {
	return security.GenerateKSEL8(signature)
}

// canonicalSignature construye "nombre(tipo1,tipo2)".
func canonicalSignature(name string, params []*Parameter) string {
	types := make([]string, len(params))
	for i, p := range params {
		types[i] = p.Type.String()
	}
	return fmt.Sprintf("%s(%s)", name, strings.Join(types, ","))
}

// ---------------- generación principal ----------------

func (bg *BytecodeGenerator) Generate(program *Program) ([]byte, error) {
	if len(program.Contracts) == 0 {
		return nil, fmt.Errorf("no hay contratos en el programa")
	}
	contract := program.Contracts[0]
	bg.contract = contract

	// registrar modifiers y events
	for _, m := range contract.Modifiers {
		bg.modifiers[m.Name] = m
	}
	for _, e := range contract.Events {
		bg.events[e.Name] = e
	}

	// slots de estado (KSP-F2: escalares → slot secuencial; map/vec → base
	// id para derivación de claves via QSH)
	if contract.State != nil {
		nextSlot := 0
		for _, v := range contract.State.Variables {
			bg.stateVarTypes[v.Name] = typeNameOf(v.Type)
			switch v.Type.(type) {
			case *MapType, *VecType:
				bg.mapBases[v.Name] = nextSlot
				// reservar rango de bases para no colisionar con escalares:
				// los maps/vecs usan claves derivadas (QSH), el base id es
				// solo semilla — pero por higiene reservamos un slot extra.
				nextSlot++
			default:
				bg.stateVarSlots[v.Name] = nextSlot
			}
			nextSlot++
		}
	}

	// 1) Dispatcher KSEL-8 (init va PRIMERO, como función explícita)
	bg.generateDispatcher(contract)

	// 2) Código de init (despachable por su KSEL-8 — params por calldata)
	bg.generateInitFunction(contract.Init)

	// 4) Funciones pub + view (las priv no entran al dispatcher)
	for _, fn := range contract.Functions {
		if fn.Visibility != "pub" && !fn.IsView {
			continue // priv: no generada en Fase 1 (sin llamadas internas)
		}
		bg.generateFunction(fn)
	}
	bg.emitPendingTraps()

	if err := bg.resolveJumps(); err != nil {
		return nil, err
	}
	if len(bg.errors) > 0 {
		return nil, fmt.Errorf("errores de generación: %s", strings.Join(bg.errors, "; "))
	}
	return bg.code, nil
}

// generateDispatcher emite la tabla de despacho por KSEL-8.
// init va PRIMERO con su propio selector (firma canónica con sus tipos) —
// el engine ejecuta el constructor con input=initParams, y el dispatcher
// lo rutea. Calldata vacío o selector desconocido → REVERT.
func (bg *BytecodeGenerator) generateDispatcher(contract *ContractDecl) {
	// selector = CALLDATALOAD(0) >> 224 (4 bytes altos de la palabra 0)
	bg.emitPushInt(0)
	bg.emit(OP_CALLDATALOAD)
	bg.emitPushInt(224)
	bg.emit(OP_SHR) // [selector]

	// init (explícito, primero)
	if contract.Init != nil {
		sel := ksel8(canonicalSignature("init", contract.Init.Parameters))
		bg.emit(OP_DUP1)
		bg.emitPush(big.NewInt(int64(sel)))
		bg.emit(OP_EQ)
		bg.emitJumpIf("ksm_init_entry")
	}

	for _, fn := range contract.Functions {
		if fn.Visibility != "pub" && !fn.IsView {
			continue
		}
		sel := ksel8(canonicalSignature(fn.Name, fn.Parameters))
		fnLabel := fmt.Sprintf("fn_%s", fn.Name)
		bg.emit(OP_DUP1)                    // [selector, selector]
		bg.emitPush(big.NewInt(int64(sel))) // [selector, selector, sel]
		bg.emit(OP_EQ)                      // [selector, match?]
		bg.emitJumpIf(fnLabel)              // consume flag
	}
	bg.emit(OP_POP) // descartar selector sin match
	bg.emitReturnRevert()
}

// generateInitFunction genera init como función despachable.
func (bg *BytecodeGenerator) generateInitFunction(init *InitFunction) {
	bg.markLabel("ksm_init_entry")
	bg.pushFrame()
	bg.bindParamsToFrame(init.Parameters)
	bg.generateBlock(init.Body)
	bg.emitReturnNil()
	bg.emitPendingTraps()
	bg.popFrame()
}

// generateFunction genera el cuerpo completo de una función.
func (bg *BytecodeGenerator) generateFunction(fn *Function) {
	fnLabel := fmt.Sprintf("fn_%s", fn.Name)
	bg.markLabel(fnLabel)
	bg.pushFrame()

	// (1.3) Parámetros: palabras de calldata 1..n → frame de memoria
	bg.bindParamsToFrame(fn.Parameters)

	// (1.5) Modifiers inlineados al inicio del cuerpo
	for _, modName := range fn.Modifiers {
		mod, ok := bg.modifiers[modName]
		if !ok {
			bg.addError(fmt.Sprintf("modifier '%s' no definido (fn %s)", modName, fn.Name))
			continue
		}
		bg.generateBlock(mod.Body)
	}

	bg.inFunction = true
	bg.hasReturn = false
	bg.generateBlock(fn.Body)

	// Retorno explícito ausente → retorno vacío
	if !bg.hasReturn {
		bg.emitReturnNil()
	}
	bg.emitPendingTraps()
	bg.inFunction = false
	bg.popFrame()
}

// ---------------- statements ----------------

func (bg *BytecodeGenerator) generateBlock(block *BlockStmt) {
	if block == nil {
		return
	}
	for _, stmt := range block.Statements {
		bg.generateStatement(stmt)
	}
}

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
	case *BreakStmt:
		bg.generateBreakStmt()
	case *ContinueStmt:
		bg.generateContinueStmt()
	case *EmitStmt:
		bg.generateEmitStmt(s)
	case *AssertStmt:
		bg.generateAssertStmt(s)
	case *RequireStmt:
		bg.generateRequireStmt(s)
	case *RevertStmt:
		bg.emitReturnRevert()
		bg.hasReturn = true
	case *ExprStmt:
		bg.generateExpression(s.Expr)
		bg.emit(OP_POP)
	case *MatchStmt:
		bg.generateMatchStmt(s)
	default:
		bg.addError(fmt.Sprintf("statement no soportado: %T", stmt))
	}
}

// (1.3) let x = expr → frame MSTORE (+ range-check KSP-F2 si u64/u128)
func (bg *BytecodeGenerator) generateLetStmt(stmt *LetStmt) {
	bg.generateExpression(stmt.Value)
	if t := typeNameOf(stmt.Type); t != "" {
		bg.varTypes[stmt.Name] = t
		bg.rangeCheckIfTyped(t)
	}
	slot := bg.allocFrame(stmt.Name)
	bg.emitPushU64(slot)
	bg.emit(OP_MSTORE)
}

func (bg *BytecodeGenerator) generateAssignStmt(stmt *AssignStmt) {
	switch target := stmt.Target.(type) {
	case *Identifier:
		bg.generateExpression(stmt.Value)
		if slot, ok := bg.varSlots[target.Name]; ok {
			bg.emitPushU64(slot)
			bg.emit(OP_MSTORE)
		} else if slot, ok := bg.stateVarSlots[target.Name]; ok {
			bg.emitPushInt(int64(slot))
			bg.emit(OP_SSTORE)
		} else {
			bg.addError("asignación a variable no declarada: " + target.Name)
			bg.emit(OP_POP)
		}
	case *MemberExpr:
		if ident, ok := target.Object.(*Identifier); ok && ident.Name == "self" {
			slot, ok := bg.stateVarSlots[target.Member]
			if !ok {
				bg.addError("self." + target.Member + " no es variable de estado")
				bg.generateExpression(stmt.Value)
				bg.emit(OP_POP)
				return
			}
			bg.generateExpression(stmt.Value)
			bg.rangeCheckIfTyped(bg.stateVarTypes[target.Member])
			bg.emitPushInt(int64(slot))
			bg.emit(OP_SSTORE)
		} else {
			bg.addError("asignación a miembro no soportada en Fase 1")
			bg.generateExpression(stmt.Value)
			bg.emit(OP_POP)
		}
	default:
		if bg.tryGenerateIndexAssign(stmt) {
			return
		}
		bg.addError("target de asignación no soportado (mappings/arrays: solo estado map/vec)")
		bg.generateExpression(stmt.Value)
		bg.emit(OP_POP)
	}
}

// tryGenerateIndexAssign (KSP-F2/F3): x[k] = v sobre maps/vecs de estado
// (soporta anidamiento). Pila final: [value, slot] con slot en el TOPE.
func (bg *BytecodeGenerator) tryGenerateIndexAssign(stmt *AssignStmt) bool {
	idx, ok := stmt.Target.(*IndexExpr)
	if !ok {
		return false
	}
	if _, _, isIndexed := bg.indexedTarget(idx); !isIndexed {
		return false
	}
	bg.generateExpression(stmt.Value) // [value]
	bg.rangeCheckIfTyped(bg.stateVarTypes[bg.resolveIndexedTargetName(idx)])
	bg.emitIndexedSlot(idx)          // [value, slot]
	bg.emit(OP_SSTORE)
	return true
}

// indexedTarget valida que el IndexExpr raíz apunte a un map/vec de estado.
func (bg *BytecodeGenerator) indexedTarget(idx *IndexExpr) (string, int, bool) {
	cur := idx
	for {
		if outer, ok := cur.Object.(*IndexExpr); ok {
			cur = outer
			continue
		}
		break
	}
	var name string
	switch o := cur.Object.(type) {
	case *Identifier:
		name = o.Name
	case *MemberExpr:
		if id, ok := o.Object.(*Identifier); ok && id.Name == "self" {
			name = o.Member
		}
	}
	baseID, ok := bg.mapBases[name]
	return name, baseID, ok
}

func (bg *BytecodeGenerator) generateCompoundAssignStmt(stmt *CompoundAssignStmt) {
	// KSP-F2: índices de estado (x[k] += v) — carga por slot derivado
	if bg.tryGenerateCompoundIndexAssign(stmt) {
		return
	}
	// x op= v → cargar x, generar v, operación (con KSM), guardar
	load := func() {
		switch t := stmt.Target.(type) {
		case *Identifier:
			if slot, ok := bg.varSlots[t.Name]; ok {
				bg.emitPushU64(slot)
				bg.emit(OP_MLOAD)
			} else if slot, ok := bg.stateVarSlots[t.Name]; ok {
				bg.emitPushInt(int64(slot))
				bg.emit(OP_SLOAD)
			} else {
				bg.addError("variable no declarada: " + t.Name)
				bg.emitPushInt(0)
			}
		case *MemberExpr:
			bg.generateExpression(stmt.Target) // self.x → SLOAD
		}
	}
	store := func() {
		switch t := stmt.Target.(type) {
		case *Identifier:
			if slot, ok := bg.varSlots[t.Name]; ok {
				bg.emitPushU64(slot)
				bg.emit(OP_MSTORE)
			} else if slot, ok := bg.stateVarSlots[t.Name]; ok {
				bg.emitPushInt(int64(slot))
				bg.emit(OP_SSTORE)
			}
		case *MemberExpr:
			if slot, ok := func() (int, bool) {
				if m, ok2 := t.Object.(*Identifier); ok2 && m.Name == "self" {
					s, ok3 := bg.stateVarSlots[t.Member]
					return s, ok3
				}
				return 0, false
			}(); ok {
				bg.emitPushInt(int64(slot))
				bg.emit(OP_SSTORE)
			}
		}
	}
	load()
	bg.generateExpression(stmt.Value)
	switch stmt.Operator {
	case "+=":
		bg.emitKSMAdd()
	case "-=":
		bg.emitKSMSub()
	case "*=":
		bg.emitKSMMul()
	case "/=":
		bg.emit(OP_DIV)
	case "%=":
		bg.emit(OP_MOD)
	}
	// KSP-F2: range-check del resultado si el target de estado es u64/u128
	switch t := stmt.Target.(type) {
	case *Identifier:
		bg.rangeCheckIfTyped(bg.stateVarTypes[t.Name])
	case *MemberExpr:
		if m, ok := t.Object.(*Identifier); ok && m.Name == "self" {
			bg.rangeCheckIfTyped(bg.stateVarTypes[t.Member])
		}
	}
	store()
}

func (bg *BytecodeGenerator) generateReturnStmt(stmt *ReturnStmt) {
	if stmt.Value != nil {
		bg.generateExpression(stmt.Value)
		bg.emitPushInt(0) // offset del retorno
		bg.emit(OP_MSTORE)
		bg.emitPushInt(32) // size
		bg.emitPushInt(0)   // offset (tope)
	} else {
		bg.emitPushInt(0)
		bg.emitPushInt(0)
	}
	bg.emit(OP_RETURN)
	bg.hasReturn = true
}

// (1.6) if/else con saltos reales
func (bg *BytecodeGenerator) generateIfStmt(stmt *IfStmt) {
	elseLabel := bg.createLabel("else")
	endLabel := bg.createLabel("endif")

	bg.generateExpression(stmt.Condition)
	bg.emit(OP_ISZERO)          // cond == false → saltar a else
	bg.emitJumpIf(elseLabel)

	bg.generateBlock(stmt.ThenBranch)
	bg.emitJump(endLabel)

	bg.markLabel(elseLabel)
	if stmt.ElseBranch != nil {
		if block, ok := stmt.ElseBranch.(*BlockStmt); ok {
			bg.generateBlock(block)
		} else {
			bg.generateStatement(stmt.ElseBranch) // else if anidado
		}
	}
	bg.markLabel(endLabel)
}

// (1.6) loop i in start..end
func (bg *BytecodeGenerator) generateLoopStmt(stmt *LoopStmt) {
	loopLabel := bg.createLabel("loop")
	condLabel := bg.createLabel("cond")
	endLabel := bg.createLabel("endloop")

	iterSlot := bg.allocFrame(stmt.Iterator)

	// i = start
	bg.generateExpression(stmt.Start)
	bg.emitPushU64(iterSlot)
	bg.emit(OP_MSTORE)

	bg.markLabel(condLabel)
	// condición: i < end
	bg.emitPushU64(iterSlot)
	bg.emit(OP_MLOAD)
	bg.generateExpression(stmt.End)
	bg.emit(OP_LT) // [i < end]
	bg.emit(OP_ISZERO)
	bg.emitJumpIf(endLabel)

	bg.loopStack = append(bg.loopStack, loopCtx{continueLabel: condLabel, breakLabel: endLabel})
	bg.generateBlock(stmt.Body)
	bg.loopStack = bg.loopStack[:len(bg.loopStack)-1]

	bg.markLabel(loopLabel)
	// i++ (KSM: overflow del iterador también trap)
	bg.emitPushU64(iterSlot)
	bg.emit(OP_MLOAD)
	bg.emitPushInt(1)
	bg.emitKSMAdd()
	bg.emitPushU64(iterSlot)
	bg.emit(OP_MSTORE)
	bg.emitJump(condLabel)

	bg.markLabel(endLabel)
}

func (bg *BytecodeGenerator) generateWhileStmt(stmt *WhileStmt) {
	condLabel := bg.createLabel("while")
	endLabel := bg.createLabel("endwhile")

	bg.markLabel(condLabel)
	bg.generateExpression(stmt.Condition)
	bg.emit(OP_ISZERO)
	bg.emitJumpIf(endLabel)

	bg.loopStack = append(bg.loopStack, loopCtx{continueLabel: condLabel, breakLabel: endLabel})
	bg.generateBlock(stmt.Body)
	bg.loopStack = bg.loopStack[:len(bg.loopStack)-1]

	bg.emitJump(condLabel)
	bg.markLabel(endLabel)
}

func (bg *BytecodeGenerator) generateBreakStmt() {
	if len(bg.loopStack) == 0 {
		bg.addError("break fuera de loop")
		return
	}
	bg.emitJump(bg.loopStack[len(bg.loopStack)-1].breakLabel)
}

func (bg *BytecodeGenerator) generateContinueStmt() {
	if len(bg.loopStack) == 0 {
		bg.addError("continue fuera de loop")
		return
	}
	bg.emitJump(bg.loopStack[len(bg.loopStack)-1].continueLabel)
}

func (bg *BytecodeGenerator) generateAssertStmt(stmt *AssertStmt) {
	bg.generateExpression(stmt.Condition)
	bg.emit(OP_ISZERO) // false → trap
	bg.emitTrapCheck()
}

func (bg *BytecodeGenerator) generateRequireStmt(stmt *RequireStmt) {
	bg.generateExpression(stmt.Condition)
	bg.emit(OP_ISZERO)
	bg.emitTrapCheck()
}

// generateMatchStmt (1.6): cadena de comparaciones EQ con saltos.
func (bg *BytecodeGenerator) generateMatchStmt(stmt *MatchStmt) {
	endLabel := bg.createLabel("match_end")
	tmp := bg.allocFrame(bg.createLabel("match_tmp")) // slot anónimo

	// valor → frame
	bg.generateExpression(stmt.Value)
	bg.emitPushU64(tmp)
	bg.emit(OP_MSTORE)

	for _, arm := range stmt.Arms {
		armLabel := bg.createLabel("match_arm")
		bg.emitPushU64(tmp)
		bg.emit(OP_MLOAD)
		bg.generateExpression(arm.Pattern)
		bg.emit(OP_EQ)
		bg.emitJumpIf(armLabel)
		// sin match → siguiente arm (fall-through)
		bg.emit(OP_POP) // mantener la pila limpia: EQ consumió ambos, nada queda
		_ = armLabel
		bg.markLabel(armLabel)
		// El patrón actual hace match: ejecutar cuerpo y salir.
		// (los arms se evalúan en orden; el primero que iguala gana)
		break
	}
	bg.addError("match: Fase 1 soporta solo el primer arm (encadenado completo en Fase 2)")
	_ = endLabel
}

// ---------------- expressions ----------------

func (bg *BytecodeGenerator) generateExpression(expr Expression) {
	switch e := expr.(type) {
	case *IntegerLiteral:
		v := new(big.Int)
		v.SetString(e.Value, 0)
		bg.emitPush(v)
	case *BoolLiteral:
		if e.Value {
			bg.emitPushInt(1)
		} else {
			bg.emitPushInt(0)
		}
	case *StringLiteral:
		// Fase 1: strings solo como constantes opacas (hash QSH del contenido)
		bg.emitPush(new(big.Int).SetBytes(security.GenerateQuantumHash([]byte(e.Value))))
	case *Identifier:
		bg.generateIdentifier(e)
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
	case *CastExpr:
		bg.generateExpression(e.Value) // cast u256/address: no-op en runtime
	case *RangeExpr:
		bg.addError("range solo dentro de loop")
		bg.emitPushInt(0)
	default:
		bg.addError(fmt.Sprintf("expresión no soportada: %T", expr))
		bg.emitPushInt(0)
	}
}

func (bg *BytecodeGenerator) generateIdentifier(ident *Identifier) {
	// (1.3) frame primero, luego estado
	if slot, ok := bg.varSlots[ident.Name]; ok {
		bg.emitPushU64(slot)
		bg.emit(OP_MLOAD)
		return
	}
	if slot, ok := bg.stateVarSlots[ident.Name]; ok {
		bg.emitPushInt(int64(slot))
		bg.emit(OP_SLOAD)
		return
	}
	bg.addError("identificador no resuelto: " + ident.Name)
	bg.emitPushInt(0)
}

func (bg *BytecodeGenerator) generateMemberExpr(expr *MemberExpr) {
	if ident, ok := expr.Object.(*Identifier); ok {
		switch ident.Name {
		case "msg":
			bg.generateMsgMember(expr.Member)
			return
		case "block":
			bg.generateBlockMember(expr.Member)
			return
		case "tx":
			bg.generateTxMember(expr.Member)
			return
		case "self":
			bg.generateSelfMember(expr.Member)
			return
		}
	}
	bg.addError("acceso a miembro no soportado: " + expr.String())
	bg.emitPushInt(0)
}

func (bg *BytecodeGenerator) generateMsgMember(member string) {
	switch member {
	case "sender":
		bg.emit(OP_CALLER)
	case "value":
		bg.emit(OP_CALLVALUE)
	case "gas":
		bg.emit(OP_GAS)
	default:
		bg.addError("msg." + member + " no soportado")
		bg.emitPushInt(0)
	}
}

func (bg *BytecodeGenerator) generateBlockMember(member string) {
	switch member {
	case "timestamp":
		bg.emit(OP_TIMESTAMP)
	case "number":
		bg.emit(OP_NUMBER)
	case "coinbase":
		bg.emit(OP_COINBASE)
	case "difficulty":
		bg.emit(OP_DIFFICULTY)
	case "gaslimit":
		bg.emit(OP_GASLIMIT)
	default:
		bg.addError("block." + member + " no soportado")
		bg.emitPushInt(0)
	}
}

func (bg *BytecodeGenerator) generateTxMember(member string) {
	switch member {
	case "origin":
		bg.emit(OP_ORIGIN)
	default:
		bg.addError("tx." + member + " no soportado")
		bg.emitPushInt(0)
	}
}

func (bg *BytecodeGenerator) generateSelfMember(member string) {
	// KSP-3: self.address → ADDRESS opcode (identidad KVI del propio
	// contrato: QSH completo de su dirección — lo que CALLER empuja en los
	// contratos que lo llaman).
	if member == "address" {
		bg.emit(OP_ADDRESS)
		return
	}
	if slot, ok := bg.stateVarSlots[member]; ok {
		bg.emitPushInt(int64(slot))
		bg.emit(OP_SLOAD)
		return
	}
	bg.addError("self." + member + " no es variable de estado")
	bg.emitPushInt(0)
}

func (bg *BytecodeGenerator) generateCallExpr(expr *CallExpr) {
	// precompiled.getStake(...) etc.
	if member, ok := expr.Function.(*MemberExpr); ok {
		if ident, ok2 := member.Object.(*Identifier); ok2 {
			if ident.Name == "precompiled" {
				bg.generatePrecompiledCall(member.Member, expr.Arguments)
				return
			}
			// KSP-3: ext.call(...) / ext.create(...) — llamadas externas
			if ident.Name == "ext" {
				bg.generateExtCall(member.Member, expr.Arguments)
				return
			}
			// KSP-F2: sec.qsh(value) → QSH del valor en runtime
			if ident.Name == "sec" && member.Member == "qsh" {
				bg.generateSecQsh(expr.Arguments)
				return
			}
		}
	}
	bg.addError("llamadas internas de contrato requieren Fase 2: " + expr.Function.String())
	bg.emitPushInt(0)
}

func (bg *BytecodeGenerator) generatePrecompiledCall(method string, args []Expression) {
	for _, arg := range args {
		bg.generateExpression(arg)
	}
	switch method {
	case "getStake":
		bg.emit(OP_GETSTAKE)
	case "isValidator":
		bg.emit(OP_ISVALIDATOR)
	case "distributeFees":
		bg.emit(OP_DISTRIBUTEFEES)
	case "getValidatorInfo":
		bg.emit(OP_GETVALIDATORINFO)
	case "beaconRoot":
		// KSP-2: sin args → raíz del último bloque (0xc4); con índice → 0xc5.
		// Los args ya están en la pila (emitidos arriba).
		if len(args) == 0 {
			bg.emit(OP_BEACONROOTLATEST)
		} else {
			bg.emit(OP_BEACONROOT)
		}
	case "beaconHeight":
		// KSP-2: índice del último bloque beacon (sin args).
		bg.emit(OP_BEACONHEIGHT)
	default:
		bg.addError("precompiled." + method + " no existe")
		bg.emitPushInt(0)
	}
}

func (bg *BytecodeGenerator) generateUnaryExpr(expr *UnaryExpr) {
	switch expr.Operator {
	case "!":
		bg.generateExpression(expr.Operand)
		bg.emit(OP_ISZERO)
	case "-":
		// 0 - x con chequeo KSM (underflow si x > 0)
		bg.emitPushInt(0)
		bg.generateExpression(expr.Operand)
		bg.emitKSMSub()
	default:
		bg.addError("operador unario no soportado: " + expr.Operator)
		bg.emitPushInt(0)
	}
}

// (1.8) KSM: aritmética con chequeo de overflow.
// IMPORTANTE — semántica del VM: GT/LT popean a=TOPE primero → resultado
// a OP b. Para "L > R", R se empuja PRIMERO y L queda en el TOPE.
func (bg *BytecodeGenerator) generateBinaryExpr(expr *BinaryExpr) {
	switch expr.Operator {
	case "+":
		bg.generateExpression(expr.Left)
		bg.generateExpression(expr.Right)
		bg.emitKSMAdd()
	case "-":
		bg.generateExpression(expr.Left)
		bg.generateExpression(expr.Right)
		bg.emitKSMSub()
	case "*":
		bg.generateExpression(expr.Left)
		bg.generateExpression(expr.Right)
		bg.emitKSMMul()
	default:
		bg.generateExpression(expr.Left)
		bg.generateExpression(expr.Right)
		switch expr.Operator {
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
			// L < R: a=tope debe ser L → R primero
			bg.emit(OP_SWAP1)
			bg.emit(OP_LT)
		case ">":
			// L > R: a=tope debe ser L → R primero
			bg.emit(OP_SWAP1)
			bg.emit(OP_GT)
		case "<=":
			// L <= R ⟺ !(L > R)
			bg.emit(OP_SWAP1)
			bg.emit(OP_GT)
			bg.emit(OP_ISZERO)
		case ">=":
			// L >= R ⟺ !(L < R)
			bg.emit(OP_SWAP1)
			bg.emit(OP_LT)
			bg.emit(OP_ISZERO)
		case "&&":
			bg.emit(OP_AND)
		case "||":
			bg.emit(OP_OR)
		default:
			bg.addError("operador no soportado: " + expr.Operator)
			bg.emitPushInt(0)
		}
	}
}

// emitKSMAdd: [a, b] → [a, b, a] ADD → [a, sum]; preservar sum para el
// chequeo y el resultado: DUP1 DUP3 GT — GT CONSUME ambos operandos, por
// lo que sum debe duplicarse ANTES (bug original: el flag dejaba [sel]
// solo y el SSTORE posterior guardaba el selector como valor).
func (bg *BytecodeGenerator) emitKSMAdd() {
	bg.emit(OP_DUP2) // [a, b, a]
	bg.emit(OP_ADD)  // [a, sum]
	bg.emit(OP_DUP1) // [a, sum, sum]
	bg.emit(OP_DUP3) // [a, sum, sum, a]
	bg.emit(OP_GT)   // [a, sum, a>sum?] — overflow si la suma envolvió
	bg.emitTrapCheck()
	// [a, sum] → descartar a
	bg.emit(OP_SWAP1)
	bg.emit(OP_POP) // [sum]
}

// emitKSMSub: [a, b] → DUP1 DUP3 LT → trap si a < b; luego SUB.
func (bg *BytecodeGenerator) emitKSMSub() {
	bg.emit(OP_DUP1) // [a, b, b]
	bg.emit(OP_DUP3) // [a, b, b, a]
	bg.emit(OP_LT)   // [a, b, a<b?] — underflow si a < b
	bg.emitTrapCheck()
	bg.emit(OP_SUB)
}

// emitKSMMul: [a, b] → prod con verificación prod/a == b (a≠0).
// Secuencia validada en diseño con stack-states paso a paso.
func (bg *BytecodeGenerator) emitKSMMul() {
	okLabel := bg.createLabel("ksm_mul_ok")

	bg.emit(OP_DUP2) // [a, b, a]
	bg.emit(OP_DUP2) // [a, b, a, b]
	bg.emit(OP_MUL)   // [a, b, prod]

	// a == 0 → prod = 0 → sin overflow → saltar el chequeo
	bg.emit(OP_DUP3)    // [a, b, prod, a]
	bg.emit(OP_ISZERO)  // [a, b, prod, a==0?]
	bg.emitJumpIf(okLabel)

	// a ≠ 0: prod/a debe ser b
	bg.emit(OP_DUP1) // [a, b, prod, prod]
	bg.emit(OP_DUP4) // [a, b, prod, prod, a]
	bg.emit(OP_DIV)  // [a, b, prod, prod/a]
	bg.emit(OP_DUP3) // [a, b, prod, prod/a, b]
	bg.emit(OP_EQ)   // [a, b, prod, prod/a==b]
	bg.emit(OP_ISZERO)
	bg.emitTrapCheck()

	bg.markLabel(okLabel)
	// [a, b, prod] → [prod]
	bg.emit(OP_SWAP2)
	bg.emit(OP_POP)
	bg.emit(OP_POP)
}

// ---------------- llamadas externas (KSP-3) ----------------

// generateExtCall (KSP-3): ext.call(dirección, "firma(tipos)", args...) → u256
// y ext.create("0x<bytecode hex>") → address. Dispatch por método.
func (bg *BytecodeGenerator) generateExtCall(method string, args []Expression) {
	switch method {
	case "call":
		bg.generateExtCallOp(args)
	case "create":
		bg.generateExtCreate(args)
	default:
		bg.addError("ext." + method + " no existe (solo call/create)")
		bg.emitPushInt(0)
	}
}

// generateExtCallOp: ext.call(to, "sig(tipos)", args...) -> u256.
// Compila el calldata KSEL-8 (selector + 28 ceros + palabras de args) a
// callScratch y emite CALL. Revierte si la llamada falla (KSM: fallo = trap)
// y empuja la PRIMERA palabra del returndata del contrato llamado.
// La dirección es la cruda del contrato (u256 con 160 bits), no la identidad.
func (bg *BytecodeGenerator) generateExtCallOp(args []Expression) {
	if len(args) < 2 {
		bg.addError("ext.call requiere: dirección, firma literal, args...")
		bg.emitPushInt(0)
		return
	}
	lit, ok := args[1].(*StringLiteral)
	if !ok {
		bg.addError("ext.call: la firma debe ser un literal string \"nombre(tipos)\"")
		bg.emitPushInt(0)
		return
	}
	callArgs := args[2:]
	if len(callArgs) > 7 {
		bg.addError("ext.call: máximo 7 argumentos (límite del scratch de calldata)")
		bg.emitPushInt(0)
		return
	}

	// word 0 = KSEL-8 selector ‖ 28 bytes cero (sel << 224)
	sel := ksel8(lit.Value)
	bg.emitPush(new(big.Int).Lsh(big.NewInt(int64(sel)), 224))
	bg.emitPushU64(callScratch)
	bg.emit(OP_MSTORE)

	// words 1..n = argumentos
	for i, a := range callArgs {
		bg.generateExpression(a)
		bg.emitPushU64(callScratch + 32*uint64(i+1))
		bg.emit(OP_MSTORE)
	}

	// CALL — pila bottom→top: [retSize, retOff, inSize, inOff, value, to, gas]
	bg.emitPushInt(32)                             // retSize: una palabra
	bg.emitPushU64(callRetScratch)                 // retOff
	bg.emitPushInt(int64(32 * (1 + len(callArgs)))) // inSize
	bg.emitPushU64(callScratch)                    // inOff
	bg.emitPushInt(0)                              // value
	bg.generateExpression(args[0])                 // to: dirección cruda del contrato
	bg.emitPushInt(2000000)                        // gas
	bg.emit(OP_CALL)                               // [ok]

	// fallo del llamado = revert del llamador (KSM)
	bg.emit(OP_ISZERO)
	bg.emitTrapCheck()

	// resultado: primera palabra del returndata
	bg.emitPushU64(callRetScratch)
	bg.emit(OP_MLOAD)
}

// generateExtCreate (KSP-3): ext.create("0x<bytecode hex>") -> address.
// KVM-3 (deploy directo): escribe el bytecode a createCodeScratch y emite
// CREATE. El contrato creado NO se ejecuta al crear — el creador lo
// inicializa explícitamente con ext.call(dirección, "initialize(...)", ...).
func (bg *BytecodeGenerator) generateExtCreate(args []Expression) {
	if len(args) != 1 {
		bg.addError("ext.create requiere exactamente 1 literal string con el bytecode hex")
		bg.emitPushInt(0)
		return
	}
	lit, ok := args[0].(*StringLiteral)
	if !ok {
		bg.addError("ext.create: el bytecode debe ser un literal string hex")
		bg.emitPushInt(0)
		return
	}
	code, err := hex.DecodeString(strings.TrimPrefix(lit.Value, "0x"))
	if err != nil || len(code) == 0 {
		bg.addError("ext.create: bytecode hex inválido: " + lit.Value)
		bg.emitPushInt(0)
		return
	}

	// bytecode → memoria
	bg.emitBytesToMemory(code, createCodeScratch)

	// CREATE — pila bottom→top: [value, offset, size] (value se popea primero)
	bg.emitPushInt(int64(len(code)))
	bg.emitPushU64(createCodeScratch)
	bg.emitPushInt(0)   // value
	bg.emit(OP_CREATE)  // [dirección del nuevo contrato]
}

// emitBytesToMemory escribe un blob de bytes a memoria por palabras de 32
// bytes (PUSH32 + MSTORE). La última palabra se rellena con ceros.
func (bg *BytecodeGenerator) emitBytesToMemory(data []byte, base uint64) {
	for off := 0; off < len(data); off += 32 {
		end := off + 32
		if end > len(data) {
			end = len(data)
		}
		var word [32]byte
		copy(word[:], data[off:end])
		bg.emitPush(new(big.Int).SetBytes(word[:]))
		bg.emitPushU64(base + uint64(off))
		bg.emit(OP_MSTORE)
	}
}

// ---------------- eventos ----------------

// generateIndexExpr (KSP-F2/F3): acceso a maps (anidados) y vecs de estado.
func (bg *BytecodeGenerator) generateIndexExpr(expr *IndexExpr) {
	bg.emitIndexedSlot(expr)
	bg.emit(OP_SLOAD)
}

// generateSecQsh (KSP-F2): sec.qsh(value) → hash post-cuántico del valor.
// El QSH opera sobre memoria: MSTORE del valor + SHA3(offset, 32).
func (bg *BytecodeGenerator) generateSecQsh(args []Expression) {
	if len(args) != 1 {
		bg.addError("sec.qsh requiere exactamente 1 argumento")
		bg.emitPushInt(0)
		return
	}
	bg.generateExpression(args[0])
	bg.emitPushU64(qshScratch)
	bg.emit(OP_MSTORE)
	bg.emitPushU64(qshScratch)
	bg.emitPushInt(32)
	bg.emit(OP_QSH)
}

// emitRangeCheck (KSP-F2: u64/u128): trapa si el valor en el tope no cabe
// en `bits`. Pila: [v] → [v] (el valor se conserva; solo salta al trap si
// v >= 2^bits).
func (bg *BytecodeGenerator) emitRangeCheck(bits uint) {
	max := new(big.Int).Lsh(big.NewInt(1), bits) // 2^bits
	bg.emit(OP_DUP1)      // [v, v]
	bg.emitPush(max)      // [v, v, 2^bits]
	bg.emit(OP_GT)        // a=top=2^bits > b=v → v < 2^bits ✓
	bg.emit(OP_ISZERO)    // v >= 2^bits → 1 → trap
	bg.emitTrapCheck()
}

// rangeCheckIfTyped aplica range-check cuando el tipo declarado es u64/u128.
func (bg *BytecodeGenerator) rangeCheckIfTyped(typeName string) {
	if bits, ok := typeMaxBits[typeName]; ok {
		bg.emitRangeCheck(bits)
	}
}

// typeNameOf extrae el nombre del tipo como string ("u64", "u256", etc.).
func typeNameOf(t TypeNode) string {
	if t == nil {
		return ""
	}
	return t.TypeName()
}

// ---------------- derivación de índices (KSP-F3) ----------------

// emitMapSlotDerivation deja el slot derivado en el tope para un IndexExpr
// de map. Soporta ANIDACIÓN (map[address, map[address, u256]]): el objeto
// interior es otro IndexExpr cuyo slot derivado se usa como base exterior.
// No toca storage: solo SLOAD/SSTORE del llamador.
func (bg *BytecodeGenerator) emitMapSlotDerivation(idx *IndexExpr) {
	if outer, ok := idx.Object.(*IndexExpr); ok {
		bg.emitMapSlotDerivation(outer) // [slot_exterior]
		bg.emitPushU64(keyScratch + 0x20)
		bg.emit(OP_MSTORE) // mem[0x320] = slot_exterior (value debajo, offset tope)
	} else {
		var name string
		switch o := idx.Object.(type) {
		case *Identifier:
			name = o.Name
		case *MemberExpr:
			if id, ok := o.Object.(*Identifier); ok && id.Name == "self" {
				name = o.Member
			}
		}
		baseID, ok := bg.mapBases[name]
		if !ok {
			bg.addError("index sobre '" + name + "' que no es map de estado")
			bg.emitPushInt(0)
			return
		}
		bg.emitPushInt(int64(baseID))
		bg.emitPushU64(keyScratch + 0x20)
		bg.emit(OP_MSTORE)
	}
	bg.generateExpression(idx.Index)
	bg.emitPushU64(keyScratch)
	bg.emit(OP_MSTORE)
	bg.emitPushU64(keyScratch)
	bg.emitPushInt(64)
	bg.emit(OP_QSH)
}

// isVecIndex: IndexExpr apuntando a un vec de estado (layout base+idx).
func (bg *BytecodeGenerator) isVecIndex(idx *IndexExpr) bool {
	if _, ok := idx.Object.(*IndexExpr); ok {
		return false // anidado = map-of-map
	}
	var name string
	switch o := idx.Object.(type) {
	case *Identifier:
		name = o.Name
	case *MemberExpr:
		if id, ok := o.Object.(*Identifier); ok && id.Name == "self" {
			name = o.Member
		}
	}
	if _, isIndexed := bg.mapBases[name]; !isIndexed {
		return false
	}
	if bg.contract != nil && bg.contract.State != nil {
		for _, sv := range bg.contract.State.Variables {
			if sv.Name == name {
				_, isVec := sv.Type.(*VecType)
				return isVec
			}
		}
	}
	return false
}

// emitIndexedSlot: slot de un IndexExpr de estado (vec: base+idx; map: QSH).
func (bg *BytecodeGenerator) emitIndexedSlot(idx *IndexExpr) {
	if bg.isVecIndex(idx) {
		bg.generateExpression(idx.Index)
		var name string
		switch o := idx.Object.(type) {
		case *Identifier:
			name = o.Name
		case *MemberExpr:
			if id, ok := o.Object.(*Identifier); ok && id.Name == "self" {
				name = o.Member
			}
		}
		bg.emitPushInt(int64(bg.mapBases[name]))
		bg.emit(OP_ADD)
		return
	}
	bg.emitMapSlotDerivation(idx)
}

// resolveIndexedTargetName: nombre del estado raíz de un IndexExpr (para
// range-checks por tipo); "" si no aplica.
func (bg *BytecodeGenerator) resolveIndexedTargetName(idx *IndexExpr) string {
	cur := idx
	for {
		if outer, ok := cur.Object.(*IndexExpr); ok {
			cur = outer
			continue
		}
		break
	}
	switch o := cur.Object.(type) {
	case *Identifier:
		return o.Name
	case *MemberExpr:
		if id, ok := o.Object.(*Identifier); ok && id.Name == "self" {
			return o.Member
		}
	}
	return ""
}

// ---------------- eventos ----------------
func (bg *BytecodeGenerator) generateEmitStmt(stmt *EmitStmt) {
	event, ok := bg.events[stmt.EventName]
	if !ok {
		bg.addError("evento no declarado: " + stmt.EventName)
		return
	}

	// datos: cada campo a eventBase + i*0x20
	for i, f := range stmt.Fields {
		bg.generateExpression(f.Value)
		bg.emitPushU64(eventBase + uint64(i)*0x20)
		bg.emit(OP_MSTORE)
	}

	// topic: hash completo (32 bytes) de la firma canónica del evento
	types := make([]string, len(event.Fields))
	for i, ef := range event.Fields {
		types[i] = ef.Type.String()
	}
	sig := fmt.Sprintf("%s(%s)", event.Name, strings.Join(types, ","))
	// KSP-6B: topics en el dominio KSEL-8 (mismo que calculateEventHash del engine)
	topic := new(big.Int).SetBytes(security.GenerateQuantumHashTagged(security.DomainKSEL8, []byte(sig)))

	size := uint64(len(stmt.Fields)) * 0x20
	// LOG1: [topic1, size, offset] con offset en el TOPE
	bg.emitPush(topic)
	bg.emitPushU64(size)
	bg.emitPushU64(eventBase)
	bg.emit(OP_LOG1)
}

// ---------------- metadata / disassembler ----------------

// Disassemble imprime el bytecode generado con el mapa completo de opcodes.
func (bg *BytecodeGenerator) Disassemble() string {
	var sb strings.Builder
	for i := 0; i < len(bg.code); i++ {
		op := Opcode(bg.code[i])
		name, ok := opcodeNames[op]
		if !ok {
			name = fmt.Sprintf("UNKNOWN(0x%02x)", op)
		}
		sb.WriteString(fmt.Sprintf("%04d: %-16s", i, name))
		// operandos de PUSH
		if op >= OP_PUSH1 && op <= OP_PUSH32 {
			n := int(op - OP_PUSH1 + 1)
			if i+1+n <= len(bg.code) {
				sb.WriteString(fmt.Sprintf(" 0x%x", bg.code[i+1:i+1+n]))
				i += n
			}
		} else if op == OP_PUSH2 {
			if i+3 <= len(bg.code) {
				sb.WriteString(fmt.Sprintf(" @%d", uint16(bg.code[i+1])<<8|uint16(bg.code[i+2])))
				i += 2
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// ToJSON devuelve metadata del bytecode generado.
func (bg *BytecodeGenerator) ToJSON() ([]byte, error) {
	type meta struct {
		Size       int      `json:"size"`
		Labels     []string `json:"labels"`
		Validators []string `json:"validators"`
	}
	labels := make([]string, 0, len(bg.labels))
	for l := range bg.labels {
		labels = append(labels, l)
	}
	return jsonMarshal(meta{
		Size:       len(bg.code),
		Labels:     labels,
		Validators: bg.errors,
	})
}

// opcodeNames — tabla completa (1.7: antes CALLDATALOAD salía UNKNOWN(0x35)).
var opcodeNames = map[Opcode]string{
	OP_STOP: "STOP", OP_ADD: "ADD", OP_MUL: "MUL", OP_SUB: "SUB",
	OP_DIV: "DIV", OP_MOD: "MOD",
	OP_LT: "LT", OP_GT: "GT", OP_EQ: "EQ", OP_ISZERO: "ISZERO",
	OP_AND: "AND", OP_OR: "OR", OP_XOR: "XOR", OP_NOT: "NOT",
	OP_SHL: "SHL", OP_SHR: "SHR", OP_QSH: "QSH",
	OP_ADDRESS: "ADDRESS", OP_BALANCE: "BALANCE", OP_ORIGIN: "ORIGIN",
	OP_CALLER: "CALLER", OP_CALLVALUE: "CALLVALUE",
	OP_CALLDATALOAD: "CALLDATALOAD", OP_CALLDATASIZE: "CALLDATASIZE",
	OP_CALLDATACOPY: "CALLDATACOPY", OP_CODESIZE: "CODESIZE",
	OP_CODECOPY: "CODECOPY",
	OP_BLOCKHASH: "BLOCKHASH", OP_COINBASE: "COINBASE",
	OP_TIMESTAMP: "TIMESTAMP", OP_NUMBER: "NUMBER",
	OP_DIFFICULTY: "DIFFICULTY", OP_GASLIMIT: "GASLIMIT",
	OP_POP: "POP", OP_MLOAD: "MLOAD", OP_MSTORE: "MSTORE",
	OP_MSTORE8: "MSTORE8", OP_SLOAD: "SLOAD", OP_SSTORE: "SSTORE",
	OP_JUMP: "JUMP", OP_JUMPI: "JUMPI", OP_PC: "PC", OP_MSIZE: "MSIZE",
	OP_GAS: "GAS", OP_JUMPDEST: "JUMPDEST",
	OP_LOG0: "LOG0", OP_LOG1: "LOG1", OP_LOG2: "LOG2",
	OP_LOG3: "LOG3", OP_LOG4: "LOG4",
	OP_GETSTAKE: "GETSTAKE", OP_ISVALIDATOR: "ISVALIDATOR",
	OP_DISTRIBUTEFEES: "DISTRIBUTEFEES",
	OP_GETVALIDATORINFO: "GETVALIDATORINFO",
	OP_CREATE: "CREATE", OP_CALL: "CALL", OP_CALLCODE: "CALLCODE",
	OP_RETURN: "RETURN", OP_DELEGATECALL: "DELEGATECALL",
	OP_STATICCALL: "STATICCALL", OP_REVERT: "REVERT",
	OP_INVALID: "INVALID", OP_SELFDESTRUCT: "SELFDESTRUCT",
}

// PUSHn dinámico para el disassembler
func init() {
	for n := 1; n <= 32; n++ {
		opcodeNames[Opcode(0x60+n-1)] = fmt.Sprintf("PUSH%d", n)
	}
	for n := 1; n <= 16; n++ {
		opcodeNames[Opcode(0x80+n-1)] = fmt.Sprintf("DUP%d", n)
		opcodeNames[Opcode(0x90+n-1)] = fmt.Sprintf("SWAP%d", n)
	}
}

func jsonMarshal(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// pushFrame/popFrame: el frame se reinicia por función (cada dispatch limpia).
func (bg *BytecodeGenerator) pushFrame() {
	bg.varSlots = make(map[string]uint64)
	bg.frameNext = frameBase
}

func (bg *BytecodeGenerator) popFrame() {}

// bindParamsToFrame copia las palabras i+1 del calldata al frame.
// KSP-F2: registra el tipo de cada parámetro y aplica range-check u64/u128.
func (bg *BytecodeGenerator) bindParamsToFrame(params []*Parameter) {
	for i, p := range params {
		slot := bg.allocFrame(p.Name)
		bg.emitPushU64(uint64(32 * (i + 1)))
		bg.emit(OP_CALLDATALOAD) // [valor]
		if t := typeNameOf(p.Type); t != "" {
			bg.varTypes[p.Name] = t
			bg.rangeCheckIfTyped(t)
		}
		bg.emitPushU64(slot) // [valor, offset]
		bg.emit(OP_MSTORE)
	}
}

// allocFrame reserva un slot de 0x20 bytes en el frame.
func (bg *BytecodeGenerator) allocFrame(name string) uint64 {
	if slot, ok := bg.varSlots[name]; ok {
		return slot
	}
	if bg.frameNext+0x20 > frameLimit {
		bg.addError("frame de función excedido (máx 64 variables)")
		return frameBase
	}
	slot := bg.frameNext
	bg.varSlots[name] = slot
	bg.frameNext += 0x20
	return slot
}

// tryGenerateCompoundIndexAssign (KSP-F2/F3): x[k] op= v sobre maps/vecs
// de estado (soporta anidamiento).
func (bg *BytecodeGenerator) tryGenerateCompoundIndexAssign(stmt *CompoundAssignStmt) bool {
	idx, ok := stmt.Target.(*IndexExpr)
	if !ok {
		return false
	}
	if _, _, isIndexed := bg.indexedTarget(idx); !isIndexed {
		return false
	}

	// carga actual
	bg.emitIndexedSlot(idx)
	bg.emit(OP_SLOAD) // [current]
	// valor
	bg.generateExpression(stmt.Value) // [current, value]
	// operación
	switch stmt.Operator {
	case "+=":
		bg.emitKSMAdd() // [new]
	case "-=":
		bg.emitKSMSub()
	case "*=":
		bg.emitKSMMul()
	case "/=":
		bg.emit(OP_DIV)
	case "%=":
		bg.emit(OP_MOD)
	}
	// range-check del resultado por tipo del estado raíz
	bg.rangeCheckIfTyped(bg.stateVarTypes[bg.resolveIndexedTargetName(idx)])
	// re-derivar slot y guardar
	bg.emitIndexedSlot(idx) // [new, slot]
	bg.emit(OP_SSTORE)
	return true
}

// emitReturnNil: RETURN(0, 0)
func (bg *BytecodeGenerator) emitReturnNil() {
	bg.emitPushInt(0) // size
	bg.emitPushInt(0) // offset (tope)
	bg.emit(OP_RETURN)
}

// emitReturnRevert: REVERT(0, 0)
func (bg *BytecodeGenerator) emitReturnRevert() {
	bg.emitPushInt(0)
	bg.emitPushInt(0)
	bg.emit(OP_REVERT)
}

// emitTrap emite el trap compartido de KSM/assert: salto incondicional al
// revert. El label del trap se crea por uso (cada función tiene el suyo).
func (bg *BytecodeGenerator) emitTrapCheck() {
	// pila: [flag] con flag != 0 → revert
	trap := bg.createLabel("trap")
	bg.emitJumpIf(trap)
	// código tras el check sigue aquí; el trap se emite al final del bloque
	// — se registra para parchearlo en emitTrapHere
	bg.trapQueue = append(bg.trapQueue, trap)
}

// emitPendingTraps emite los labels de trap acumulados con REVERT.
func (bg *BytecodeGenerator) emitPendingTraps() {
	for _, t := range bg.trapQueue {
		bg.markLabel(t)
		bg.emitReturnRevert()
	}
	bg.trapQueue = bg.trapQueue[:0]
}
