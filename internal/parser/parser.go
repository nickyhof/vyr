package parser

import (
	"fmt"
	"strconv"

	"github.com/nickyhof/vyr/internal/lexer"
)

// Parser turns a token stream into an AST with pipe desugaring.
type Parser struct {
	tokens []lexer.Token
	pos    int
}

// New creates a Parser for the given tokens.
func New(tokens []lexer.Token) *Parser {
	return &Parser{tokens: tokens, pos: 0}
}

// Parse parses the full program and returns the root AST node.
func (p *Parser) Parse() (*Program, error) {
	prog := &Program{}
	for !p.atEnd() {
		decl, err := p.parseDecl()
		if err != nil {
			return nil, err
		}
		if decl != nil {
			prog.Decls = append(prog.Decls, decl)
		}
	}
	return prog, nil
}

func (p *Parser) parseDecl() (Node, error) {
	if p.check(lexer.IMPORT) {
		return p.parseImport()
	}
	if p.check(lexer.FN) {
		return p.parseFnDecl()
	}
	if p.check(lexer.STRUCT) {
		return p.parseStructDecl()
	}
	return nil, p.errorf("expected declaration, got %s", p.current())
}

func (p *Parser) parseImport() (*ImportDecl, error) {
	p.advance() // consume 'import'
	if !p.check(lexer.STRING) {
		return nil, p.errorf("expected string path after 'import', got %s", p.current())
	}
	path := p.current().Literal
	p.advance()
	return &ImportDecl{Path: path}, nil
}

func (p *Parser) parseFnDecl() (*FnDecl, error) {
	p.advance() // consume 'fn'
	if !p.check(lexer.IDENT) {
		return nil, p.errorf("expected function name, got %s", p.current())
	}
	name := p.current().Literal
	p.advance()

	params, err := p.parseParamList()
	if err != nil {
		return nil, err
	}

	// Optional return type: ): type {
	var returnType string
	if p.check(lexer.COLON) {
		p.advance()
		if !p.check(lexer.IDENT) {
			return nil, p.errorf("expected return type after ':', got %s", p.current())
		}
		returnType = p.current().Literal
		p.advance()
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &FnDecl{Name: name, Params: params, ReturnType: returnType, Body: body}, nil
}

// parseParamList parses (name: type, name: type, ...)
func (p *Parser) parseParamList() ([]Param, error) {
	if err := p.expect(lexer.LPAREN); err != nil {
		return nil, err
	}
	var params []Param
	for !p.check(lexer.RPAREN) && !p.atEnd() {
		if !p.check(lexer.IDENT) {
			return nil, p.errorf("expected parameter name, got %s", p.current())
		}
		param := Param{Name: p.current().Literal}
		p.advance()
		// Optional type annotation
		if p.check(lexer.COLON) {
			p.advance()
			if !p.check(lexer.IDENT) {
				return nil, p.errorf("expected type name after ':', got %s", p.current())
			}
			param.Type = p.current().Literal
			p.advance()
		}
		params = append(params, param)
		if p.check(lexer.COMMA) {
			p.advance()
		}
	}
	if err := p.expect(lexer.RPAREN); err != nil {
		return nil, err
	}
	return params, nil
}

// parseBlock parses { stmt... }
func (p *Parser) parseBlock() ([]Node, error) {
	if err := p.expect(lexer.LBRACE); err != nil {
		return nil, err
	}
	var body []Node
	for !p.check(lexer.RBRACE) && !p.atEnd() {
		stmt, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		body = append(body, stmt)
	}
	if err := p.expect(lexer.RBRACE); err != nil {
		return nil, err
	}
	return body, nil
}

func (p *Parser) parseStmt() (Node, error) {
	if p.check(lexer.LET) {
		return p.parseLet()
	}
	if p.check(lexer.WHILE) {
		return p.parseWhile()
	}
	// Check for assignment: ident = expr
	if p.check(lexer.IDENT) && p.peekType() == lexer.EQUAL {
		name := p.current().Literal
		p.advance() // consume ident
		p.advance() // consume =
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		return &AssignStmt{Name: name, Value: value}, nil
	}
	return p.parseExpr()
}

func (p *Parser) parseLet() (*LetStmt, error) {
	p.advance() // consume 'let'
	// Check for 'mut' keyword
	mutable := false
	if p.check(lexer.MUT) {
		mutable = true
		p.advance()
	}
	if !p.check(lexer.IDENT) {
		return nil, p.errorf("expected variable name after 'let', got %s", p.current())
	}
	name := p.current().Literal
	p.advance()
	if err := p.expect(lexer.EQUAL); err != nil {
		return nil, err
	}
	value, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &LetStmt{Name: name, Value: value, Mutable: mutable}, nil
}

// Expression precedence (lowest → highest):
// pipe |> → comparison ==,!=,<,>,<=,>= → add/sub +,- → mul/div *,/ → primary

func (p *Parser) parseExpr() (Node, error) {
	return p.parsePipe()
}

func (p *Parser) parsePipe() (Node, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	for p.check(lexer.PIPE) {
		p.advance()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		switch r := right.(type) {
		case *ArrayLit:
			left = &FanOutExpr{Value: left, Fns: r.Elements}
		case *MatchExpr:
			if r.Subject == nil {
				r.Subject = left
			}
			left = r
		case *CallExpr:
			r.Args = append([]Node{left}, r.Args...)
			left = r
		default:
			left = &CallExpr{Fn: right, Args: []Node{left}}
		}
	}

	return left, nil
}

func (p *Parser) parseComparison() (Node, error) {
	left, err := p.parseAddSub()
	if err != nil {
		return nil, err
	}

	for p.isComparison() {
		op := p.current().Literal
		p.advance()
		right, err := p.parseAddSub()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op, Left: left, Right: right}
	}

	return left, nil
}

func (p *Parser) isComparison() bool {
	return p.check(lexer.EQ) || p.check(lexer.NEQ) ||
		p.check(lexer.LT) || p.check(lexer.GT) ||
		p.check(lexer.LTE) || p.check(lexer.GTE)
}

func (p *Parser) parseAddSub() (Node, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return nil, err
	}
	for p.check(lexer.PLUS) || p.check(lexer.MINUS) {
		op := p.current().Literal
		p.advance()
		right, err := p.parseMulDiv()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseMulDiv() (Node, error) {
	left, err := p.parsePostfix()
	if err != nil {
		return nil, err
	}
	for p.check(lexer.STAR) || p.check(lexer.SLASH) {
		op := p.current().Literal
		p.advance()
		right, err := p.parsePostfix()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op, Left: left, Right: right}
	}
	return left, nil
}

// parsePostfix handles dot access and call chains after a primary expression.
func (p *Parser) parsePostfix() (Node, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		if p.check(lexer.DOT) {
			p.advance()
			if !p.check(lexer.IDENT) {
				return nil, p.errorf("expected identifier after '.', got %s", p.current())
			}
			key := p.current().Literal
			p.advance()
			expr = &DotExpr{Object: expr, Key: key}
		} else {
			break
		}
	}
	return expr, nil
}

func (p *Parser) parsePrimary() (Node, error) {
	tok := p.current()

	switch tok.Type {
	case lexer.STRING:
		p.advance()
		// Check if this is an interpolated string
		if p.check(lexer.INTERP_START) {
			return p.parseInterpLit(tok.Literal)
		}
		return &StringLit{Value: tok.Literal}, nil

	case lexer.INT:
		p.advance()
		val, err := strconv.ParseInt(tok.Literal, 10, 64)
		if err != nil {
			return nil, p.errorf("invalid integer: %s", tok.Literal)
		}
		return &IntLit{Value: val}, nil

	case lexer.TRUE:
		p.advance()
		return &BoolLit{Value: true}, nil

	case lexer.FALSE:
		p.advance()
		return &BoolLit{Value: false}, nil

	case lexer.IDENT:
		p.advance()
		ident := &Ident{Name: tok.Literal}
		if p.check(lexer.LPAREN) {
			return p.parseCall(ident)
		}
		return ident, nil

	case lexer.FN:
		return p.parseFnLit()

	case lexer.LBRACKET:
		return p.parseArrayLit()

	case lexer.MATCH:
		return p.parseMatch()

	case lexer.IF:
		return p.parseIfExpr()

	case lexer.HASH_LBRACE:
		return p.parseHashLit()

	case lexer.LPAREN:
		p.advance()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.expect(lexer.RPAREN); err != nil {
			return nil, err
		}
		return expr, nil

	default:
		return nil, p.errorf("unexpected token: %s", tok)
	}
}

func (p *Parser) parseFnLit() (*FnLit, error) {
	p.advance() // consume 'fn'
	params, err := p.parseParamList()
	if err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &FnLit{Params: params, Body: body}, nil
}

func (p *Parser) parseArrayLit() (*ArrayLit, error) {
	p.advance() // consume [
	var elements []Node
	for !p.check(lexer.RBRACKET) && !p.atEnd() {
		elem, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)
		if p.check(lexer.COMMA) {
			p.advance()
		}
	}
	if err := p.expect(lexer.RBRACKET); err != nil {
		return nil, err
	}
	return &ArrayLit{Elements: elements}, nil
}

func (p *Parser) parseHashLit() (*HashLit, error) {
	p.advance() // consume #{
	var pairs []HashPair
	for !p.check(lexer.RBRACE) && !p.atEnd() {
		// Key: identifier or string
		var key string
		switch {
		case p.check(lexer.IDENT):
			key = p.current().Literal
			p.advance()
		case p.check(lexer.STRING):
			key = p.current().Literal
			p.advance()
		default:
			return nil, p.errorf("expected key (identifier or string), got %s", p.current())
		}
		if err := p.expect(lexer.COLON); err != nil {
			return nil, err
		}
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, HashPair{Key: key, Value: value})
		if p.check(lexer.COMMA) {
			p.advance()
		}
	}
	if err := p.expect(lexer.RBRACE); err != nil {
		return nil, err
	}
	return &HashLit{Pairs: pairs}, nil
}

func (p *Parser) parseMatch() (*MatchExpr, error) {
	p.advance() // consume 'match'
	var subject Node
	if !p.check(lexer.LBRACE) {
		var err error
		subject, err = p.parseAddSub()
		if err != nil {
			return nil, err
		}
	}
	if err := p.expect(lexer.LBRACE); err != nil {
		return nil, err
	}
	var arms []MatchArm
	for !p.check(lexer.RBRACE) && !p.atEnd() {
		pattern, err := p.parseMatchPattern()
		if err != nil {
			return nil, err
		}
		if err := p.expect(lexer.ARROW); err != nil {
			return nil, err
		}
		body, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		arms = append(arms, MatchArm{Pattern: pattern, Body: body})
	}
	if err := p.expect(lexer.RBRACE); err != nil {
		return nil, err
	}
	return &MatchExpr{Subject: subject, Arms: arms}, nil
}

func (p *Parser) parseIfExpr() (*IfExpr, error) {
	p.advance() // consume 'if'
	condition, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	thenBody, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	var elseBody []Node
	if p.check(lexer.ELSE) {
		p.advance()
		elseBody, err = p.parseBlock()
		if err != nil {
			return nil, err
		}
	}
	return &IfExpr{Condition: condition, Then: thenBody, Else: elseBody}, nil
}

func (p *Parser) parseCall(fn Node) (*CallExpr, error) {
	p.advance() // consume (
	var args []Node
	for !p.check(lexer.RPAREN) && !p.atEnd() {
		arg, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if p.check(lexer.COMMA) {
			p.advance()
		}
	}
	if err := p.expect(lexer.RPAREN); err != nil {
		return nil, err
	}
	return &CallExpr{Fn: fn, Args: args}, nil
}

// parseInterpLit parses an interpolated string like "hello ${name}, age ${age}"
// The initial text fragment has already been consumed; firstText is that fragment.
func (p *Parser) parseInterpLit(firstText string) (Node, error) {
	parts := []Node{&StringLit{Value: firstText}}

	for p.check(lexer.INTERP_START) {
		p.advance() // consume INTERP_START

		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		parts = append(parts, expr)

		if err := p.expect(lexer.INTERP_END); err != nil {
			return nil, err
		}

		// After INTERP_END there should be a STRING token (the next text fragment)
		if p.check(lexer.STRING) {
			parts = append(parts, &StringLit{Value: p.current().Literal})
			p.advance()
		}
	}

	return &InterpLit{Parts: parts}, nil
}

// --- helpers ---

func (p *Parser) current() lexer.Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return lexer.Token{Type: lexer.EOF}
}

func (p *Parser) check(typ lexer.TokenType) bool { return p.current().Type == typ }

func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

func (p *Parser) atEnd() bool { return p.current().Type == lexer.EOF }

func (p *Parser) expect(typ lexer.TokenType) error {
	if !p.check(typ) {
		return p.errorf("expected %s, got %s", typ, p.current())
	}
	p.advance()
	return nil
}

func (p *Parser) errorf(format string, args ...interface{}) error {
	tok := p.current()
	prefix := fmt.Sprintf("parse error at line %d, col %d: ", tok.Line, tok.Col)
	return fmt.Errorf(prefix+format, args...)
}

// peekType returns the token type of the next token without advancing.
func (p *Parser) peekType() lexer.TokenType {
	if p.pos+1 < len(p.tokens) {
		return p.tokens[p.pos+1].Type
	}
	return lexer.EOF
}

// parseWhile parses: while condition { body }
func (p *Parser) parseWhile() (*WhileExpr, error) {
	p.advance() // consume 'while'
	condition, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &WhileExpr{Condition: condition, Body: body}, nil
}

// parseStructDecl parses: struct Name { field1, field2, ... }
func (p *Parser) parseStructDecl() (*StructDecl, error) {
	p.advance() // consume 'struct'
	if !p.check(lexer.IDENT) {
		return nil, p.errorf("expected struct name, got %s", p.current())
	}
	name := p.current().Literal
	p.advance()
	if err := p.expect(lexer.LBRACE); err != nil {
		return nil, err
	}
	var fields []string
	for !p.check(lexer.RBRACE) && !p.atEnd() {
		if !p.check(lexer.IDENT) {
			return nil, p.errorf("expected field name, got %s", p.current())
		}
		fields = append(fields, p.current().Literal)
		p.advance()
		// Optional type annotation (skip for now)
		if p.check(lexer.COLON) {
			p.advance()
			if !p.check(lexer.IDENT) {
				return nil, p.errorf("expected type after ':', got %s", p.current())
			}
			p.advance() // skip type name
		}
		if p.check(lexer.COMMA) {
			p.advance()
		}
	}
	if err := p.expect(lexer.RBRACE); err != nil {
		return nil, err
	}
	return &StructDecl{Name: name, Fields: fields}, nil
}

// parseMatchPattern parses a match arm pattern, including type patterns: TypeName(binding)
func (p *Parser) parseMatchPattern() (Node, error) {
	if p.check(lexer.IDENT) && p.peekType() == lexer.LPAREN {
		typeName := p.current().Literal
		p.advance() // consume type name
		p.advance() // consume (
		if !p.check(lexer.IDENT) {
			return nil, p.errorf("expected binding name in type pattern, got %s", p.current())
		}
		binding := p.current().Literal
		p.advance()
		if err := p.expect(lexer.RPAREN); err != nil {
			return nil, err
		}
		return &TypePattern{TypeName: typeName, Binding: binding}, nil
	}
	return p.parsePrimary()
}
