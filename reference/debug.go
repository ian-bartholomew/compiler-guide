package main

import (
	"fmt"
	"strings"
)

// kindName gives each token kind a readable label for `tinc -tokens`.
var kindName = map[TokenKind]string{
	TEOF: "EOF", TInt: "INT", TIdent: "IDENT",
	TFunc: "func", TLet: "let", TIf: "if", TElse: "else",
	TWhile: "while", TReturn: "return", TPrint: "print",
	TLParen: "(", TRParen: ")", TLBrace: "{", TRBrace: "}",
	TComma: ",", TSemicolon: ";", TAssign: "=",
	TPlus: "+", TMinus: "-", TStar: "*", TSlash: "/",
	TEq: "==", TNe: "!=", TLt: "<", TGt: ">", TLe: "<=", TGe: ">=",
}

// dumpTokens prints the token stream, one per line (used by `tinc -tokens`).
func dumpTokens(toks []Token) {
	for _, t := range toks {
		fmt.Printf("%3d  %-8s %q\n", t.Line, kindName[t.Kind], t.Text)
	}
}

// dumpAST prints the parse tree (used by `tinc -ast`). Expressions render as
// S-expressions, so precedence and associativity are visible at a glance.
func dumpAST(p *Program) {
	for i := range p.Funcs {
		f := &p.Funcs[i]
		fmt.Printf("func %s(%s)\n", f.Name, strings.Join(f.Params, ", "))
		dumpStmts(f.Body, 1)
	}
}

func dumpStmts(ss []Stmt, d int) {
	for _, s := range ss {
		dumpStmt(s, d)
	}
}

func dumpStmt(s Stmt, d int) {
	pad := strings.Repeat("  ", d)
	switch s := s.(type) {
	case *LetStmt:
		fmt.Printf("%slet %s = %s\n", pad, s.Name, exprStr(s.Value))
	case *AssignStmt:
		fmt.Printf("%sassign %s = %s\n", pad, s.Name, exprStr(s.Value))
	case *ReturnStmt:
		fmt.Printf("%sreturn %s\n", pad, exprStr(s.Value))
	case *PrintStmt:
		fmt.Printf("%sprint %s\n", pad, exprStr(s.Value))
	case *ExprStmt:
		fmt.Printf("%sexpr %s\n", pad, exprStr(s.X))
	case *IfStmt:
		fmt.Printf("%sif %s\n", pad, exprStr(s.Cond))
		dumpStmts(s.Then, d+1)
		if s.Else != nil {
			fmt.Printf("%selse\n", pad)
			dumpStmts(s.Else, d+1)
		}
	case *WhileStmt:
		fmt.Printf("%swhile %s\n", pad, exprStr(s.Cond))
		dumpStmts(s.Body, d+1)
	}
}

func exprStr(e Expr) string {
	switch e := e.(type) {
	case *IntLit:
		return fmt.Sprintf("%d", e.Value)
	case *Var:
		return e.Name
	case *Unary:
		return fmt.Sprintf("(%s %s)", e.Op, exprStr(e.X))
	case *Binary:
		return fmt.Sprintf("(%s %s %s)", e.Op, exprStr(e.L), exprStr(e.R))
	case *Call:
		parts := make([]string, len(e.Args))
		for i, a := range e.Args {
			parts[i] = exprStr(a)
		}
		return fmt.Sprintf("%s(%s)", e.Name, strings.Join(parts, ", "))
	}
	return "?"
}
