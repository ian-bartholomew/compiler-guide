package main

import (
	"fmt"
	"strconv"
)

// ----- AST: expressions -----

type Expr interface{ isExpr() }

type IntLit struct{ Value int64 }
type Var struct{ Name string }
type Unary struct {
	Op string
	X  Expr
}
type Binary struct {
	Op   string
	L, R Expr
}
type Call struct {
	Name string
	Args []Expr
}

func (*IntLit) isExpr() {}
func (*Var) isExpr()    {}
func (*Unary) isExpr()  {}
func (*Binary) isExpr() {}
func (*Call) isExpr()   {}

// ----- AST: statements -----

type Stmt interface{ isStmt() }

type LetStmt struct {
	Name  string
	Value Expr
}
type AssignStmt struct {
	Name  string
	Value Expr
}
type IfStmt struct {
	Cond       Expr
	Then, Else []Stmt // Else may be nil
}
type WhileStmt struct {
	Cond Expr
	Body []Stmt
}
type ReturnStmt struct{ Value Expr }
type PrintStmt struct{ Value Expr }
type ExprStmt struct{ X Expr }

func (*LetStmt) isStmt()    {}
func (*AssignStmt) isStmt() {}
func (*IfStmt) isStmt()     {}
func (*WhileStmt) isStmt()  {}
func (*ReturnStmt) isStmt() {}
func (*PrintStmt) isStmt()  {}
func (*ExprStmt) isStmt()   {}

// ----- AST: top level -----

type FuncDecl struct {
	Name   string
	Params []string
	Body   []Stmt
}

type Program struct{ Funcs []FuncDecl }

// ----- parser -----

type parser struct {
	toks []Token
	pos  int
}

func parse(toks []Token) *Program {
	p := &parser{toks: toks}
	prog := &Program{}
	for p.peek().Kind != TEOF {
		prog.Funcs = append(prog.Funcs, p.funcDecl())
	}
	return prog
}

func (p *parser) peek() Token    { return p.toks[p.pos] }
func (p *parser) advance() Token { t := p.toks[p.pos]; p.pos++; return t }

func (p *parser) match(k TokenKind) bool {
	if p.peek().Kind == k {
		p.pos++
		return true
	}
	return false
}

func (p *parser) expect(k TokenKind) Token {
	if p.peek().Kind != k {
		p.fail("expected " + tokenName(k))
	}
	return p.advance()
}

func (p *parser) fail(msg string) {
	fatalf(p.peek().Line, "%s, got %q", msg, p.peek().Text)
}

func (p *parser) funcDecl() FuncDecl {
	p.expect(TFunc)
	name := p.expect(TIdent).Text
	p.expect(TLParen)
	var params []string
	if p.peek().Kind != TRParen {
		params = append(params, p.expect(TIdent).Text)
		for p.match(TComma) {
			params = append(params, p.expect(TIdent).Text)
		}
	}
	p.expect(TRParen)
	return FuncDecl{Name: name, Params: params, Body: p.block()}
}

func (p *parser) block() []Stmt {
	p.expect(TLBrace)
	var stmts []Stmt
	for p.peek().Kind != TRBrace && p.peek().Kind != TEOF {
		stmts = append(stmts, p.statement())
	}
	p.expect(TRBrace)
	return stmts
}

func (p *parser) statement() Stmt {
	switch p.peek().Kind {
	case TLet:
		p.advance()
		name := p.expect(TIdent).Text
		p.expect(TAssign)
		val := p.expr()
		p.expect(TSemicolon)
		return &LetStmt{Name: name, Value: val}
	case TIf:
		p.advance()
		cond := p.expr()
		then := p.block()
		var els []Stmt
		if p.match(TElse) {
			els = p.block()
		}
		return &IfStmt{Cond: cond, Then: then, Else: els}
	case TWhile:
		p.advance()
		cond := p.expr()
		return &WhileStmt{Cond: cond, Body: p.block()}
	case TReturn:
		p.advance()
		val := p.expr()
		p.expect(TSemicolon)
		return &ReturnStmt{Value: val}
	case TPrint:
		p.advance()
		val := p.expr()
		p.expect(TSemicolon)
		return &PrintStmt{Value: val}
	default:
		// assignment `name = expr;` or a bare expression statement
		if p.peek().Kind == TIdent && p.toks[p.pos+1].Kind == TAssign {
			name := p.advance().Text
			p.advance() // consume '='
			val := p.expr()
			p.expect(TSemicolon)
			return &AssignStmt{Name: name, Value: val}
		}
		e := p.expr()
		p.expect(TSemicolon)
		return &ExprStmt{X: e}
	}
}

// Expression grammar, one method per precedence level.

func (p *parser) expr() Expr { return p.equality() }

func (p *parser) equality() Expr {
	left := p.comparison()
	for p.peek().Kind == TEq || p.peek().Kind == TNe {
		op := p.advance().Text
		left = &Binary{Op: op, L: left, R: p.comparison()}
	}
	return left
}

func (p *parser) comparison() Expr {
	left := p.term()
	for {
		switch p.peek().Kind {
		case TLt, TGt, TLe, TGe:
			op := p.advance().Text
			left = &Binary{Op: op, L: left, R: p.term()}
		default:
			return left
		}
	}
}

func (p *parser) term() Expr {
	left := p.factor()
	for p.peek().Kind == TPlus || p.peek().Kind == TMinus {
		op := p.advance().Text
		left = &Binary{Op: op, L: left, R: p.factor()}
	}
	return left
}

func (p *parser) factor() Expr {
	left := p.unary()
	for p.peek().Kind == TStar || p.peek().Kind == TSlash {
		op := p.advance().Text
		left = &Binary{Op: op, L: left, R: p.unary()}
	}
	return left
}

func (p *parser) unary() Expr {
	if p.peek().Kind == TMinus {
		p.advance()
		return &Unary{Op: "-", X: p.unary()}
	}
	return p.primary()
}

func (p *parser) primary() Expr {
	tok := p.peek()
	switch tok.Kind {
	case TInt:
		p.advance()
		v, err := strconv.ParseInt(tok.Text, 10, 64)
		if err != nil {
			fatalf(tok.Line, "invalid integer %q", tok.Text)
		}
		return &IntLit{Value: v}
	case TIdent:
		p.advance()
		if p.match(TLParen) { // a call: name(args)
			var args []Expr
			if p.peek().Kind != TRParen {
				args = append(args, p.expr())
				for p.match(TComma) {
					args = append(args, p.expr())
				}
			}
			p.expect(TRParen)
			return &Call{Name: tok.Text, Args: args}
		}
		return &Var{Name: tok.Text}
	case TLParen:
		p.advance()
		e := p.expr()
		p.expect(TRParen)
		return e
	}
	p.fail("expected an expression")
	return nil
}

func tokenName(k TokenKind) string {
	switch k {
	case TLParen:
		return "'('"
	case TRParen:
		return "')'"
	case TLBrace:
		return "'{'"
	case TRBrace:
		return "'}'"
	case TSemicolon:
		return "';'"
	case TAssign:
		return "'='"
	case TComma:
		return "','"
	case TIdent:
		return "identifier"
	default:
		return fmt.Sprintf("token %d", k)
	}
}
