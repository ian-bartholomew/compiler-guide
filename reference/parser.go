package main

import (
	"fmt"
	"strconv"
)

// ----- AST: expressions (nodes that produce a value) -----

type Expr interface{ isExpr() }

type IntLit struct{ Value int64 } // a literal integer, e.g. 42
type Var struct{ Name string }    // a variable reference, e.g. x
type Unary struct {               // a prefix operator on one operand
	Op string // only "-" in Tin
	X  Expr   // the operand
}
type Binary struct { // two operands joined by an operator
	Op   string // "+" "-" "*" "/" "<" ">" "<=" ">=" "==" "!="
	L, R Expr   // left and right operands
}
type Call struct { // a function call, e.g. fib(n - 1)
	Name string // the function being called
	Args []Expr // the argument expressions
}

func (*IntLit) isExpr() {}
func (*Var) isExpr()    {}
func (*Unary) isExpr()  {}
func (*Binary) isExpr() {}
func (*Call) isExpr()   {}

// ----- AST: statements (nodes that do something) -----

type Stmt interface{ isStmt() }

type LetStmt struct { // declare a new local: let x = expr;
	Name  string
	Value Expr
}
type AssignStmt struct { // reassign an existing local: x = expr;
	Name  string
	Value Expr
}
type IfStmt struct {
	Cond       Expr
	Then, Else []Stmt // Else is nil when there is no else block
}
type WhileStmt struct {
	Cond Expr
	Body []Stmt
}
type ReturnStmt struct{ Value Expr } // return expr;
type PrintStmt struct{ Value Expr }  // print expr;
type ExprStmt struct{ X Expr }       // a bare expression, e.g. a call: f(x);

func (*LetStmt) isStmt()    {}
func (*AssignStmt) isStmt() {}
func (*IfStmt) isStmt()     {}
func (*WhileStmt) isStmt()  {}
func (*ReturnStmt) isStmt() {}
func (*PrintStmt) isStmt()  {}
func (*ExprStmt) isStmt()   {}

// ----- AST: top level -----

type FuncDecl struct {
	Name   string   // the function's name
	Params []string // parameter names, in declaration order
	Body   []Stmt   // the statements in its block
}

type Program struct{ Funcs []FuncDecl } // a program is just a list of functions

// ----- parser -----

type parser struct {
	toks []Token // the full token stream, ending in TEOF
	pos  int     // index of the next token to read
}

// parse is the top grammar rule: a program is zero or more function decls.
func parse(toks []Token) *Program {
	p := &parser{toks: toks}
	prog := &Program{}
	for p.peek().Kind != TEOF {
		prog.Funcs = append(prog.Funcs, p.funcDecl())
	}
	return prog
}

func (p *parser) peek() Token    { return p.toks[p.pos] }                  // look, don't consume
func (p *parser) advance() Token { t := p.toks[p.pos]; p.pos++; return t } // consume one token

// match consumes the next token if it is kind k, and reports whether it did.
// This is what drives the "{ ... }" (zero-or-more) parts of the grammar.
func (p *parser) match(k TokenKind) bool {
	if p.peek().Kind == k {
		p.pos++
		return true
	}
	return false
}

// expect consumes a token of kind k, or dies with a syntax error. Safe to call
// at any position because the TEOF sentinel is always there to peek at.
func (p *parser) expect(k TokenKind) Token {
	if p.peek().Kind != k {
		p.fail("expected " + tokenName(k))
	}
	return p.advance()
}

func (p *parser) fail(msg string) {
	fatalf(p.peek().Line, "%s, got %q", msg, p.peek().Text)
}

// funcDecl = "func" IDENT "(" [ params ] ")" block.
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

// block = "{" { statement } "}". The TEOF guard stops a runaway loop when the
// closing brace is missing; expect(TRBrace) then reports the error.
func (p *parser) block() []Stmt {
	p.expect(TLBrace)
	var stmts []Stmt
	for p.peek().Kind != TRBrace && p.peek().Kind != TEOF {
		stmts = append(stmts, p.statement())
	}
	p.expect(TRBrace)
	return stmts
}

// statement dispatches on the first token to the matching grammar rule.
func (p *parser) statement() Stmt {
	switch p.peek().Kind {
	case TLet: // let IDENT "=" expr ";"
		p.advance()
		name := p.expect(TIdent).Text
		p.expect(TAssign)
		val := p.expr()
		p.expect(TSemicolon)
		return &LetStmt{Name: name, Value: val}
	case TIf: // if expr block [ "else" block ]
		p.advance()
		cond := p.expr()
		then := p.block()
		var els []Stmt
		if p.match(TElse) { // the else block is optional
			els = p.block()
		}
		return &IfStmt{Cond: cond, Then: then, Else: els}
	case TWhile: // while expr block
		p.advance()
		cond := p.expr()
		return &WhileStmt{Cond: cond, Body: p.block()}
	case TReturn: // return expr ";"
		p.advance()
		val := p.expr()
		p.expect(TSemicolon)
		return &ReturnStmt{Value: val}
	case TPrint: // print expr ";"
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

// Expression grammar, one method per precedence level. Each level parses the
// level below it, then loops while it sees one of its own operators. Because the
// looser operators sit higher in the call chain, tighter ones end up deeper in
// the tree — that is how precedence falls out of the structure.

func (p *parser) expr() Expr { return p.equality() } // the whole expression rule

// equality = comparison { ("==" | "!=") comparison } — loosest binding.
func (p *parser) equality() Expr {
	left := p.comparison()
	for p.peek().Kind == TEq || p.peek().Kind == TNe {
		op := p.advance().Text // fold each operator into a left-leaning tree
		left = &Binary{Op: op, L: left, R: p.comparison()}
	}
	return left
}

// comparison = term { ("<" | ">" | "<=" | ">=") term }.
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

// term = factor { ("+" | "-") factor }.
func (p *parser) term() Expr {
	left := p.factor()
	for p.peek().Kind == TPlus || p.peek().Kind == TMinus {
		op := p.advance().Text
		left = &Binary{Op: op, L: left, R: p.factor()}
	}
	return left
}

// factor = unary { ("*" | "/") unary } — tightest of the binary operators.
func (p *parser) factor() Expr {
	left := p.unary()
	for p.peek().Kind == TStar || p.peek().Kind == TSlash {
		op := p.advance().Text
		left = &Binary{Op: op, L: left, R: p.unary()}
	}
	return left
}

// unary = "-" unary | primary. Recurses into itself so "- - x" parses.
func (p *parser) unary() Expr {
	if p.peek().Kind == TMinus {
		p.advance()
		return &Unary{Op: "-", X: p.unary()}
	}
	return p.primary()
}

// primary = INT | IDENT | IDENT "(" args ")" | "(" expr ")" — the leaves and the
// parenthesis rule, the bottom of the precedence chain.
func (p *parser) primary() Expr {
	tok := p.peek()
	switch tok.Kind {
	case TInt: // an integer literal
		p.advance()
		v, err := strconv.ParseInt(tok.Text, 10, 64)
		if err != nil {
			fatalf(tok.Line, "invalid integer %q", tok.Text)
		}
		return &IntLit{Value: v}
	case TIdent: // a variable, or a call if followed by "("
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
	case TLParen: // "(" expr ")" — recurse to expr, then require the closing ")"
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
