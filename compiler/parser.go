package compiler

import (
	"fmt"
	"strconv"
)

// Parser converts tokens into an Abstract Syntax Tree
type Parser struct {
	lexer      *Lexer
	curToken   Token
	peekToken  Token
	errors     []string
}

// NewParser creates a new parser
func NewParser(lexer *Lexer) *Parser {
	p := &Parser{
		lexer:  lexer,
		errors: []string{},
	}
	// Read two tokens to initialize curToken and peekToken
	p.nextToken()
	p.nextToken()
	return p
}

// Errors returns the list of parsing errors
func (p *Parser) Errors() []string {
	return p.errors
}

// nextToken advances to the next token
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

// curTokenIs checks if current token matches expected type
func (p *Parser) curTokenIs(t TokenType) bool {
	return p.curToken.Type == t
}

// peekTokenIs checks if peek token matches expected type
func (p *Parser) peekTokenIs(t TokenType) bool {
	return p.peekToken.Type == t
}

// expectPeek advances if peek token matches, otherwise adds error
func (p *Parser) expectPeek(t TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

// peekError adds error for unexpected peek token
func (p *Parser) peekError(t TokenType) {
	msg := fmt.Sprintf("Line %d:%d - expected next token to be %s, got %s instead",
		p.peekToken.Line, p.peekToken.Column, t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

// addError adds a generic error message
func (p *Parser) addError(msg string) {
	fullMsg := fmt.Sprintf("Line %d:%d - %s", p.curToken.Line, p.curToken.Column, msg)
	p.errors = append(p.errors, fullMsg)
}

// ===========================
// PARSING ENTRY POINT
// ===========================

// ParseProgram parses the entire program
func (p *Parser) ParseProgram() *Program {
	program := &Program{
		Contracts: []*ContractDecl{},
		Structs:   []*StructDecl{},
	}

	for !p.curTokenIs(TOKEN_EOF) {
		switch p.curToken.Type {
		case TOKEN_CONTRACT:
			contract := p.parseContract()
			if contract != nil {
				program.Contracts = append(program.Contracts, contract)
			}
		case TOKEN_STRUCT:
			structDecl := p.parseStruct()
			if structDecl != nil {
				program.Structs = append(program.Structs, structDecl)
			}
		default:
			p.addError(fmt.Sprintf("unexpected token at top level: %s", p.curToken.Type))
		}
		p.nextToken()
	}

	return program
}

// ===========================
// CONTRACT PARSING
// ===========================

// parseContract parses a contract declaration
func (p *Parser) parseContract() *ContractDecl {
	if !p.curTokenIs(TOKEN_CONTRACT) {
		p.addError(fmt.Sprintf("expected 'contract', got %s", p.curToken.Type))
		return nil
	}

	contract := &ContractDecl{
		Functions: []*Function{},
		Modifiers: []*ModifierDecl{},
		Events:    []*EventDecl{},
		Structs:   []*StructDecl{},
	}

	// Contract name
	if !p.expectPeek(TOKEN_IDENT) {
		return nil
	}
	contract.Name = p.curToken.Literal

	// Opening brace
	if !p.expectPeek(TOKEN_LBRACE) {
		return nil
	}

	// Parse contract body
	p.nextToken()
	for !p.curTokenIs(TOKEN_RBRACE) && !p.curTokenIs(TOKEN_EOF) {
		switch p.curToken.Type {
		case TOKEN_STATE:
			contract.State = p.parseStateBlock()
			// parseStateBlock leaves curToken at the next section keyword,
			// so don't call p.nextToken() - continue loop with current token
			continue
		case TOKEN_INIT:
			contract.Init = p.parseInitFunction()
			p.nextToken()
		case TOKEN_PUB, TOKEN_PRIV, TOKEN_VIEW, TOKEN_FN:
			fn := p.parseFunction()
			if fn != nil {
				contract.Functions = append(contract.Functions, fn)
			}
			p.nextToken()
		case TOKEN_REQUIRE:
			if p.peekTokenIs(TOKEN_FN) {
				modifier := p.parseModifier()
				if modifier != nil {
					contract.Modifiers = append(contract.Modifiers, modifier)
				}
			} else {
				p.addError("unexpected 'require' keyword outside modifier declaration")
			}
			p.nextToken()
		case TOKEN_EVENT:
			event := p.parseEvent()
			if event != nil {
				contract.Events = append(contract.Events, event)
			}
			p.nextToken()
		case TOKEN_STRUCT:
			structDecl := p.parseStruct()
			if structDecl != nil {
				contract.Structs = append(contract.Structs, structDecl)
			}
			p.nextToken()
		default:
			p.addError(fmt.Sprintf("unexpected token in contract body: %s", p.curToken.Type))
			p.nextToken()
		}
	}

	return contract
}

// parseStateBlock parses the state block
func (p *Parser) parseStateBlock() *StateBlock {
	state := &StateBlock{
		Variables: []*StateVariable{},
	}

	// Expect colon
	if !p.expectPeek(TOKEN_COLON) {
		return nil
	}

	// Parse state variables (indented, no braces)
	p.nextToken()
	for !p.curTokenIs(TOKEN_RBRACE) && !p.curTokenIs(TOKEN_EOF) {
		// Stop if we encounter a keyword that starts a new section
		if p.curTokenIs(TOKEN_EVENT) || p.curTokenIs(TOKEN_STRUCT) ||
		   p.curTokenIs(TOKEN_INIT) || p.curTokenIs(TOKEN_FN) ||
		   p.curTokenIs(TOKEN_PUB) || p.curTokenIs(TOKEN_PRIV) ||
		   p.curTokenIs(TOKEN_VIEW) || p.curTokenIs(TOKEN_REQUIRE) {
			// Stop parsing state variables, don't advance
			return state
		}

		// Check if this looks like a state variable (identifier followed by colon)
		if p.curTokenIs(TOKEN_IDENT) && p.peekTokenIs(TOKEN_COLON) {
			varName := p.curToken.Literal
			p.nextToken() // move to colon
			p.nextToken() // move to type

			varType := p.parseType()
			if varType != nil {
				state.Variables = append(state.Variables, &StateVariable{
					Name: varName,
					Type: varType,
				})
			}
			p.nextToken()
		} else {
			// Not a state variable, stop
			return state
		}
	}

	return state
}

// parseInitFunction parses the init (constructor) function
func (p *Parser) parseInitFunction() *InitFunction {
	init := &InitFunction{}

	// Expect opening paren
	if !p.expectPeek(TOKEN_LPAREN) {
		return nil
	}

	// Parse parameters
	init.Parameters = p.parseParameterList()

	// Expect opening brace
	if !p.expectPeek(TOKEN_LBRACE) {
		return nil
	}

	// Parse body
	init.Body = p.parseBlockStatement()

	return init
}

// parseFunction parses a function declaration
func (p *Parser) parseFunction() *Function {
	fn := &Function{
		Parameters: []*Parameter{},
		Modifiers:  []string{},
	}

	// Parse visibility and view modifiers
	for p.curTokenIs(TOKEN_PUB) || p.curTokenIs(TOKEN_PRIV) || p.curTokenIs(TOKEN_VIEW) {
		if p.curTokenIs(TOKEN_PUB) {
			fn.Visibility = "pub"
		} else if p.curTokenIs(TOKEN_PRIV) {
			fn.Visibility = "priv"
		} else if p.curTokenIs(TOKEN_VIEW) {
			fn.IsView = true
		}
		p.nextToken()
	}

	// Expect 'fn' keyword
	if !p.curTokenIs(TOKEN_FN) {
		p.addError("expected 'fn' keyword")
		return nil
	}

	// Function name
	if !p.expectPeek(TOKEN_IDENT) {
		return nil
	}
	fn.Name = p.curToken.Literal

	// Parameters
	if !p.expectPeek(TOKEN_LPAREN) {
		return nil
	}
	fn.Parameters = p.parseParameterList()

	// Return type (optional)
	if p.peekTokenIs(TOKEN_ARROW) {
		p.nextToken() // move to arrow
		p.nextToken() // move to type
		fn.ReturnType = p.parseType()
	}

	// Modifiers (optional)
	for p.peekTokenIs(TOKEN_IDENT) {
		p.nextToken()
		fn.Modifiers = append(fn.Modifiers, p.curToken.Literal)
	}

	// Body
	if !p.expectPeek(TOKEN_LBRACE) {
		return nil
	}
	fn.Body = p.parseBlockStatement()

	return fn
}

// parseModifier parses a require modifier
func (p *Parser) parseModifier() *ModifierDecl {
	modifier := &ModifierDecl{}

	// Skip 'require'
	if !p.expectPeek(TOKEN_FN) {
		return nil
	}

	// Modifier name
	if !p.expectPeek(TOKEN_IDENT) {
		return nil
	}
	modifier.Name = p.curToken.Literal

	// Expect empty parameter list
	if !p.expectPeek(TOKEN_LPAREN) {
		return nil
	}
	if !p.expectPeek(TOKEN_RPAREN) {
		return nil
	}

	// Body
	if !p.expectPeek(TOKEN_LBRACE) {
		return nil
	}
	modifier.Body = p.parseBlockStatement()

	return modifier
}

// parseEvent parses an event declaration
func (p *Parser) parseEvent() *EventDecl {
	event := &EventDecl{
		Fields: []*EventField{},
	}

	// Event name
	if !p.expectPeek(TOKEN_IDENT) {
		return nil
	}
	event.Name = p.curToken.Literal

	// Opening brace
	if !p.expectPeek(TOKEN_LBRACE) {
		return nil
	}

	// Parse fields
	p.nextToken()
	for !p.curTokenIs(TOKEN_RBRACE) && !p.curTokenIs(TOKEN_EOF) {
		if p.curTokenIs(TOKEN_IDENT) {
			field := &EventField{}
			field.Name = p.curToken.Literal

			// Expect colon
			if !p.expectPeek(TOKEN_COLON) {
				return nil
			}

			// Parse type
			p.nextToken()
			field.Type = p.parseType()

			// Check for 'indexed' keyword
			if p.peekTokenIs(TOKEN_INDEXED) {
				p.nextToken()
				field.Indexed = true
			}

			event.Fields = append(event.Fields, field)
		}
		p.nextToken()
	}

	return event
}

// parseStruct parses a struct declaration
func (p *Parser) parseStruct() *StructDecl {
	structDecl := &StructDecl{
		Fields: []*StructField{},
	}

	// Struct name
	if !p.expectPeek(TOKEN_IDENT) {
		return nil
	}
	structDecl.Name = p.curToken.Literal

	// Opening brace
	if !p.expectPeek(TOKEN_LBRACE) {
		return nil
	}

	// Parse fields
	p.nextToken()
	for !p.curTokenIs(TOKEN_RBRACE) && !p.curTokenIs(TOKEN_EOF) {
		if p.curTokenIs(TOKEN_IDENT) {
			field := &StructField{}
			field.Name = p.curToken.Literal

			// Expect colon
			if !p.expectPeek(TOKEN_COLON) {
				return nil
			}

			// Parse type
			p.nextToken()
			field.Type = p.parseType()

			structDecl.Fields = append(structDecl.Fields, field)
		}
		p.nextToken()
	}

	return structDecl
}

// parseParameterList parses function parameters
func (p *Parser) parseParameterList() []*Parameter {
	params := []*Parameter{}

	p.nextToken() // move past opening paren

	if p.curTokenIs(TOKEN_RPAREN) {
		return params
	}

	// First parameter
	if p.curTokenIs(TOKEN_IDENT) {
		param := &Parameter{Name: p.curToken.Literal}
		if !p.expectPeek(TOKEN_COLON) {
			return nil
		}
		p.nextToken()
		param.Type = p.parseType()
		params = append(params, param)
	}

	// Additional parameters
	for p.peekTokenIs(TOKEN_COMMA) {
		p.nextToken() // move to comma
		p.nextToken() // move to param name

		if p.curTokenIs(TOKEN_IDENT) {
			param := &Parameter{Name: p.curToken.Literal}
			if !p.expectPeek(TOKEN_COLON) {
				return nil
			}
			p.nextToken()
			param.Type = p.parseType()
			params = append(params, param)
		}
	}

	if !p.expectPeek(TOKEN_RPAREN) {
		return nil
	}

	return params
}

// ===========================
// TYPE PARSING
// ===========================

// parseType parses a type annotation
func (p *Parser) parseType() TypeNode {
	switch p.curToken.Type {
	case TOKEN_VEC:
		return p.parseVecType()
	case TOKEN_MAP:
		return p.parseMapType()
	case TOKEN_IDENT:
		// Could be primitive type or struct type
		return &PrimitiveType{Name: p.curToken.Literal}
	default:
		// Primitive types
		return &PrimitiveType{Name: p.curToken.Literal}
	}
}

// parseVecType parses vec[T] type
func (p *Parser) parseVecType() TypeNode {
	if !p.expectPeek(TOKEN_LBRACKET) {
		return nil
	}
	p.nextToken()
	elementType := p.parseType()
	if !p.expectPeek(TOKEN_RBRACKET) {
		return nil
	}
	return &VecType{ElementType: elementType}
}

// parseMapType parses map[K,V] type
func (p *Parser) parseMapType() TypeNode {
	if !p.expectPeek(TOKEN_LBRACKET) {
		return nil
	}
	p.nextToken()
	keyType := p.parseType()
	if !p.expectPeek(TOKEN_COMMA) {
		return nil
	}
	p.nextToken()
	valueType := p.parseType()
	if !p.expectPeek(TOKEN_RBRACKET) {
		return nil
	}
	return &MapType{KeyType: keyType, ValueType: valueType}
}

// ===========================
// STATEMENT PARSING
// ===========================

// parseBlockStatement parses a block of statements
func (p *Parser) parseBlockStatement() *BlockStmt {
	block := &BlockStmt{
		Statements: []Statement{},
	}

	p.nextToken() // move past opening brace

	for !p.curTokenIs(TOKEN_RBRACE) && !p.curTokenIs(TOKEN_EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

// parseStatement parses a single statement
func (p *Parser) parseStatement() Statement {
	switch p.curToken.Type {
	case TOKEN_LET:
		return p.parseLetStatement()
	case TOKEN_RETURN:
		return p.parseReturnStatement()
	case TOKEN_IF:
		return p.parseIfStatement()
	case TOKEN_LOOP:
		return p.parseLoopStatement()
	case TOKEN_WHILE:
		return p.parseWhileStatement()
	case TOKEN_BREAK:
		return &BreakStmt{}
	case TOKEN_CONTINUE:
		return &ContinueStmt{}
	case TOKEN_EMIT:
		return p.parseEmitStatement()
	case TOKEN_ASSERT:
		return p.parseAssertStatement()
	case TOKEN_REQUIRE:
		return p.parseRequireStatement()
	case TOKEN_REVERT:
		return p.parseRevertStatement()
	case TOKEN_MATCH:
		return p.parseMatchStatement()
	default:
		return p.parseExpressionStatement()
	}
}

// parseLetStatement parses variable declaration
func (p *Parser) parseLetStatement() Statement {
	stmt := &LetStmt{}

	// Check for 'mut' keyword
	if p.peekTokenIs(TOKEN_MUT) {
		p.nextToken()
		stmt.IsMut = true
	}

	// Variable name
	if !p.expectPeek(TOKEN_IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	// Type annotation (optional)
	if p.peekTokenIs(TOKEN_COLON) {
		p.nextToken() // move to colon
		p.nextToken() // move to type
		stmt.Type = p.parseType()
	}

	// Assignment
	if !p.expectPeek(TOKEN_ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Value = p.parseExpression(PRECEDENCE_LOWEST)

	return stmt
}

// parseReturnStatement parses return statement
func (p *Parser) parseReturnStatement() Statement {
	stmt := &ReturnStmt{}

	p.nextToken()

	// Check if there's a return value
	if !p.curTokenIs(TOKEN_RBRACE) && !p.curTokenIs(TOKEN_EOF) {
		stmt.Value = p.parseExpression(PRECEDENCE_LOWEST)
	}

	return stmt
}

// parseIfStatement parses if/else statement
func (p *Parser) parseIfStatement() Statement {
	stmt := &IfStmt{}

	p.nextToken()
	stmt.Condition = p.parseExpression(PRECEDENCE_LOWEST)

	if !p.expectPeek(TOKEN_LBRACE) {
		return nil
	}
	stmt.ThenBranch = p.parseBlockStatement()

	// Check for else
	if p.peekTokenIs(TOKEN_ELSE) {
		p.nextToken() // move to else

		if p.peekTokenIs(TOKEN_IF) {
			p.nextToken() // move to if
			stmt.ElseBranch = p.parseIfStatement()
		} else if p.peekTokenIs(TOKEN_LBRACE) {
			p.nextToken()
			stmt.ElseBranch = p.parseBlockStatement()
		}
	}

	return stmt
}

// parseLoopStatement parses loop statement
func (p *Parser) parseLoopStatement() Statement {
	stmt := &LoopStmt{}

	// Iterator variable
	if !p.expectPeek(TOKEN_IDENT) {
		return nil
	}
	stmt.Iterator = p.curToken.Literal

	// 'in' keyword
	if !p.expectPeek(TOKEN_IN) {
		return nil
	}

	// Range expression (start..end)
	p.nextToken()
	stmt.Start = p.parseExpression(PRECEDENCE_LOWEST)

	if !p.expectPeek(TOKEN_RANGE) {
		return nil
	}

	p.nextToken()
	stmt.End = p.parseExpression(PRECEDENCE_LOWEST)

	// Body
	if !p.expectPeek(TOKEN_LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()

	return stmt
}

// parseWhileStatement parses while loop
func (p *Parser) parseWhileStatement() Statement {
	stmt := &WhileStmt{}

	p.nextToken()
	stmt.Condition = p.parseExpression(PRECEDENCE_LOWEST)

	if !p.expectPeek(TOKEN_LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()

	return stmt
}

// parseEmitStatement parses emit event statement
func (p *Parser) parseEmitStatement() Statement {
	stmt := &EmitStmt{
		Fields: []*EmitField{},
	}

	// Event name
	if !p.expectPeek(TOKEN_IDENT) {
		return nil
	}
	stmt.EventName = p.curToken.Literal

	// Opening brace
	if !p.expectPeek(TOKEN_LBRACE) {
		return nil
	}

	// Parse fields
	p.nextToken()
	for !p.curTokenIs(TOKEN_RBRACE) && !p.curTokenIs(TOKEN_EOF) {
		if p.curTokenIs(TOKEN_IDENT) {
			field := &EmitField{}
			field.Name = p.curToken.Literal

			// Expect colon
			if !p.expectPeek(TOKEN_COLON) {
				return nil
			}

			// Parse value expression
			p.nextToken()
			field.Value = p.parseExpression(PRECEDENCE_LOWEST)

			stmt.Fields = append(stmt.Fields, field)

			// Check for comma
			if p.peekTokenIs(TOKEN_COMMA) {
				p.nextToken()
			}
		}
		p.nextToken()
	}

	return stmt
}

// parseAssertStatement parses assert statement
func (p *Parser) parseAssertStatement() Statement {
	stmt := &AssertStmt{}

	if !p.expectPeek(TOKEN_LPAREN) {
		return nil
	}

	p.nextToken()
	stmt.Condition = p.parseExpression(PRECEDENCE_LOWEST)

	if p.peekTokenIs(TOKEN_COMMA) {
		p.nextToken() // move to comma
		p.nextToken() // move to message
		stmt.Message = p.parseExpression(PRECEDENCE_LOWEST)
	}

	if !p.expectPeek(TOKEN_RPAREN) {
		return nil
	}

	return stmt
}

// parseRequireStatement parses require statement
func (p *Parser) parseRequireStatement() Statement {
	stmt := &RequireStmt{}

	if !p.expectPeek(TOKEN_LPAREN) {
		return nil
	}

	p.nextToken()
	stmt.Condition = p.parseExpression(PRECEDENCE_LOWEST)

	if p.peekTokenIs(TOKEN_COMMA) {
		p.nextToken() // move to comma
		p.nextToken() // move to message
		stmt.Message = p.parseExpression(PRECEDENCE_LOWEST)
	}

	if !p.expectPeek(TOKEN_RPAREN) {
		return nil
	}

	return stmt
}

// parseRevertStatement parses revert statement
func (p *Parser) parseRevertStatement() Statement {
	stmt := &RevertStmt{}

	if !p.expectPeek(TOKEN_LPAREN) {
		return nil
	}

	p.nextToken()
	stmt.Message = p.parseExpression(PRECEDENCE_LOWEST)

	if !p.expectPeek(TOKEN_RPAREN) {
		return nil
	}

	return stmt
}

// parseMatchStatement parses match statement
func (p *Parser) parseMatchStatement() Statement {
	stmt := &MatchStmt{
		Arms: []*MatchArm{},
	}

	p.nextToken()
	stmt.Value = p.parseExpression(PRECEDENCE_LOWEST)

	if !p.expectPeek(TOKEN_LBRACE) {
		return nil
	}

	// Parse match arms
	p.nextToken()
	for !p.curTokenIs(TOKEN_RBRACE) && !p.curTokenIs(TOKEN_EOF) {
		arm := &MatchArm{}
		arm.Pattern = p.parseExpression(PRECEDENCE_LOWEST)

		if !p.expectPeek(TOKEN_ARROW) {
			return nil
		}

		if !p.expectPeek(TOKEN_LBRACE) {
			return nil
		}

		arm.Body = p.parseBlockStatement()
		stmt.Arms = append(stmt.Arms, arm)

		p.nextToken()
	}

	return stmt
}

// parseExpressionStatement parses an expression used as a statement
func (p *Parser) parseExpressionStatement() Statement {
	expr := p.parseExpression(PRECEDENCE_LOWEST)

	// Check for assignment operators
	if p.peekTokenIs(TOKEN_ASSIGN) {
		p.nextToken() // move to =
		p.nextToken() // move to value
		return &AssignStmt{
			Target: expr,
			Value:  p.parseExpression(PRECEDENCE_LOWEST),
		}
	}

	// Check for compound assignment
	if p.peekToken.Type == TOKEN_PLUS_ASSIGN || p.peekToken.Type == TOKEN_MINUS_ASSIGN ||
		p.peekToken.Type == TOKEN_ASTERISK_EQ || p.peekToken.Type == TOKEN_SLASH_EQ ||
		p.peekToken.Type == TOKEN_PERCENT_EQ {
		op := p.peekToken.Literal
		p.nextToken() // move to operator
		p.nextToken() // move to value
		return &CompoundAssignStmt{
			Target:   expr,
			Operator: op,
			Value:    p.parseExpression(PRECEDENCE_LOWEST),
		}
	}

	return &ExprStmt{Expr: expr}
}

// ===========================
// EXPRESSION PARSING (Pratt Parser)
// ===========================

// Operator precedence levels
const (
	PRECEDENCE_LOWEST      = iota
	PRECEDENCE_LOGICAL_OR  // ||
	PRECEDENCE_LOGICAL_AND // &&
	PRECEDENCE_EQUALS      // == !=
	PRECEDENCE_LESSGREATER // > < >= <=
	PRECEDENCE_SUM         // + -
	PRECEDENCE_PRODUCT     // * / %
	PRECEDENCE_PREFIX      // -x !x
	PRECEDENCE_CALL        // func(x) arr[i] obj.field
	PRECEDENCE_AS          // as (type cast)
)

var precedences = map[TokenType]int{
	TOKEN_OR:       PRECEDENCE_LOGICAL_OR,
	TOKEN_AND:      PRECEDENCE_LOGICAL_AND,
	TOKEN_EQ:       PRECEDENCE_EQUALS,
	TOKEN_NOT_EQ:   PRECEDENCE_EQUALS,
	TOKEN_LT:       PRECEDENCE_LESSGREATER,
	TOKEN_GT:       PRECEDENCE_LESSGREATER,
	TOKEN_LT_EQ:    PRECEDENCE_LESSGREATER,
	TOKEN_GT_EQ:    PRECEDENCE_LESSGREATER,
	TOKEN_PLUS:     PRECEDENCE_SUM,
	TOKEN_MINUS:    PRECEDENCE_SUM,
	TOKEN_ASTERISK: PRECEDENCE_PRODUCT,
	TOKEN_SLASH:    PRECEDENCE_PRODUCT,
	TOKEN_PERCENT:  PRECEDENCE_PRODUCT,
	TOKEN_LPAREN:   PRECEDENCE_CALL,
	TOKEN_LBRACKET: PRECEDENCE_CALL,
	TOKEN_DOT:      PRECEDENCE_CALL,
	TOKEN_RANGE:    PRECEDENCE_LOWEST,
	TOKEN_AS:       PRECEDENCE_AS,
}

func (p *Parser) peekPrecedence() int {
	if prec, ok := precedences[p.peekToken.Type]; ok {
		return prec
	}
	return PRECEDENCE_LOWEST
}

func (p *Parser) curPrecedence() int {
	if prec, ok := precedences[p.curToken.Type]; ok {
		return prec
	}
	return PRECEDENCE_LOWEST
}

// parseExpression parses expressions using Pratt parsing
func (p *Parser) parseExpression(precedence int) Expression {
	// Parse prefix expression
	var leftExp Expression

	switch p.curToken.Type {
	case TOKEN_IDENT:
		leftExp = &Identifier{Name: p.curToken.Literal}
	case TOKEN_SELF, TOKEN_MSG, TOKEN_BLOCK, TOKEN_TX, TOKEN_PRECOMPILED:
		// Built-in global identifiers
		leftExp = &Identifier{Name: p.curToken.Literal}
	case TOKEN_INT:
		leftExp = &IntegerLiteral{Value: p.curToken.Literal}
	case TOKEN_TRUE:
		leftExp = &BoolLiteral{Value: true}
	case TOKEN_FALSE:
		leftExp = &BoolLiteral{Value: false}
	case TOKEN_STRING:
		leftExp = &StringLiteral{Value: p.curToken.Literal}
	case TOKEN_NOT, TOKEN_MINUS:
		leftExp = p.parsePrefixExpression()
	case TOKEN_LPAREN:
		leftExp = p.parseGroupedExpression()
	case TOKEN_VEC:
		leftExp = p.parseVecLiteral()
	default:
		p.addError(fmt.Sprintf("no prefix parse function for %s", p.curToken.Type))
		return nil
	}

	// Parse infix expressions
	for !p.peekTokenIs(TOKEN_EOF) && precedence < p.peekPrecedence() {
		switch p.peekToken.Type {
		case TOKEN_PLUS, TOKEN_MINUS, TOKEN_ASTERISK, TOKEN_SLASH, TOKEN_PERCENT,
			TOKEN_EQ, TOKEN_NOT_EQ, TOKEN_LT, TOKEN_GT, TOKEN_LT_EQ, TOKEN_GT_EQ,
			TOKEN_AND, TOKEN_OR:
			p.nextToken()
			leftExp = p.parseInfixExpression(leftExp)
		case TOKEN_LPAREN:
			p.nextToken()
			leftExp = p.parseCallExpression(leftExp)
		case TOKEN_LBRACKET:
			p.nextToken()
			leftExp = p.parseIndexExpression(leftExp)
		case TOKEN_DOT:
			p.nextToken()
			leftExp = p.parseMemberExpression(leftExp)
		case TOKEN_RANGE:
			p.nextToken()
			leftExp = p.parseRangeExpression(leftExp)
		case TOKEN_AS:
			p.nextToken()
			leftExp = p.parseCastExpression(leftExp)
		default:
			return leftExp
		}
	}

	return leftExp
}

// parsePrefixExpression parses prefix expressions like -x or !x
func (p *Parser) parsePrefixExpression() Expression {
	expr := &UnaryExpr{
		Operator: p.curToken.Literal,
	}

	p.nextToken()
	expr.Operand = p.parseExpression(PRECEDENCE_PREFIX)

	return expr
}

// parseGroupedExpression parses (expression)
func (p *Parser) parseGroupedExpression() Expression {
	p.nextToken()
	exp := p.parseExpression(PRECEDENCE_LOWEST)
	if !p.expectPeek(TOKEN_RPAREN) {
		return nil
	}
	return exp
}

// parseVecLiteral parses vec[1, 2, 3] or vec[]
func (p *Parser) parseVecLiteral() Expression {
	vec := &VecLiteral{
		Elements: []Expression{},
	}

	if !p.expectPeek(TOKEN_LBRACKET) {
		return nil
	}

	p.nextToken()

	if p.curTokenIs(TOKEN_RBRACKET) {
		return vec
	}

	vec.Elements = append(vec.Elements, p.parseExpression(PRECEDENCE_LOWEST))

	for p.peekTokenIs(TOKEN_COMMA) {
		p.nextToken() // move to comma
		p.nextToken() // move to element
		vec.Elements = append(vec.Elements, p.parseExpression(PRECEDENCE_LOWEST))
	}

	if !p.expectPeek(TOKEN_RBRACKET) {
		return nil
	}

	return vec
}

// parseInfixExpression parses binary expressions
func (p *Parser) parseInfixExpression(left Expression) Expression {
	expr := &BinaryExpr{
		Left:     left,
		Operator: p.curToken.Literal,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(precedence)

	return expr
}

// parseCallExpression parses function calls
func (p *Parser) parseCallExpression(function Expression) Expression {
	expr := &CallExpr{
		Function:  function,
		Arguments: []Expression{},
	}

	p.nextToken() // move past (

	if p.curTokenIs(TOKEN_RPAREN) {
		return expr
	}

	expr.Arguments = append(expr.Arguments, p.parseExpression(PRECEDENCE_LOWEST))

	for p.peekTokenIs(TOKEN_COMMA) {
		p.nextToken() // move to comma
		p.nextToken() // move to argument
		expr.Arguments = append(expr.Arguments, p.parseExpression(PRECEDENCE_LOWEST))
	}

	if !p.expectPeek(TOKEN_RPAREN) {
		return nil
	}

	return expr
}

// parseIndexExpression parses array/map indexing
func (p *Parser) parseIndexExpression(left Expression) Expression {
	expr := &IndexExpr{
		Object: left,
	}

	p.nextToken()
	expr.Index = p.parseExpression(PRECEDENCE_LOWEST)

	if !p.expectPeek(TOKEN_RBRACKET) {
		return nil
	}

	return expr
}

// parseMemberExpression parses member access
func (p *Parser) parseMemberExpression(object Expression) Expression {
	// KSP-3: `address` es keyword (TOKEN_ADDRESS) — `self.address` debe
	// parsear como miembro. Aceptamos IDENT o la keyword address.
	if p.peekTokenIs(TOKEN_IDENT) || p.peekTokenIs(TOKEN_ADDRESS) {
		p.nextToken()
	} else {
		p.addError(fmt.Sprintf("expected member name after '.', got %s", p.peekToken.Type))
		return nil
	}

	return &MemberExpr{
		Object: object,
		Member: p.curToken.Literal,
	}
}

// parseRangeExpression parses range expressions
func (p *Parser) parseRangeExpression(start Expression) Expression {
	p.nextToken()
	return &RangeExpr{
		Start: start,
		End:   p.parseExpression(PRECEDENCE_LOWEST),
	}
}

// parseCastExpression parses type casting
func (p *Parser) parseCastExpression(value Expression) Expression {
	p.nextToken()
	return &CastExpr{
		Value:      value,
		TargetType: p.parseType(),
	}
}

// ===========================
// UTILITY FUNCTIONS
// ===========================

// ParseInt safely converts string to int64
func ParseInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
