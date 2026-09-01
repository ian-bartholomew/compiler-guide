package main

import (
	"fmt"
	"strings"
)

// System V AMD64: the first six integer arguments go in these registers.
var argRegs = []string{"%rdi", "%rsi", "%rdx", "%rcx", "%r8", "%r9"}

// Comparison operator -> the setcc instruction that reads the flags.
var setInstr = map[string]string{
	"==": "sete",
	"!=": "setne",
	"<":  "setl",
	">":  "setg",
	"<=": "setle",
	">=": "setge",
}

type gen struct {
	buf     strings.Builder
	offsets map[string]int // variable name -> frame offset, e.g. -8
	depth   int            // values currently pushed on the value stack
	labelID int
	curFunc string
}

func codegen(prog *Program) string {
	if !hasMain(prog) {
		fatalf(0, "no 'main' function to run")
	}
	g := &gen{}
	g.line("    .section .rodata")
	g.line(".LC0:")
	g.line("    .string \"%ld\\n\"") // printf format: 64-bit int + newline
	g.line("    .text")
	g.line("    .globl main")
	for i := range prog.Funcs {
		g.function(&prog.Funcs[i])
	}
	return g.buf.String()
}

func hasMain(p *Program) bool {
	for i := range p.Funcs {
		if p.Funcs[i].Name == "main" {
			return true
		}
	}
	return false
}

func (g *gen) line(s string) { g.buf.WriteString(s); g.buf.WriteByte('\n') }
func (g *gen) label(s string) { g.line(s + ":") }
func (g *gen) emit(format string, args ...interface{}) {
	g.buf.WriteString("    ")
	fmt.Fprintf(&g.buf, format, args...)
	g.buf.WriteByte('\n')
}

func (g *gen) function(f *FuncDecl) {
	g.curFunc = f.Name
	g.offsets = map[string]int{}
	g.depth = 0

	// Assign a stack slot to every parameter, then every local.
	slot := 0
	for _, name := range f.Params {
		slot++
		g.offsets[name] = -8 * slot
	}
	assignSlots(f.Body, &slot, g.offsets)
	frame := align16(slot * 8)

	g.label(f.Name)
	g.emit("pushq %%rbp")
	g.emit("movq %%rsp, %%rbp")
	if frame > 0 {
		g.emit("subq $%d, %%rsp", frame)
	}
	// Spill incoming parameters from their argument registers into slots.
	for i, name := range f.Params {
		if i >= len(argRegs) {
			fatalf(0, "function %s has more than %d parameters", f.Name, len(argRegs))
		}
		g.emit("movq %s, %d(%%rbp)", argRegs[i], g.offsets[name])
	}

	g.stmts(f.Body)

	// Implicit "return 0" if control falls off the end.
	g.emit("movq $0, %%rax")
	g.emit("leave")
	g.emit("ret")
}

// assignSlots gives a frame slot to each variable the first time it is
// declared. Variables are function-scoped: a repeated name reuses its slot.
func assignSlots(stmts []Stmt, slot *int, offsets map[string]int) {
	declare := func(name string) {
		if _, ok := offsets[name]; !ok {
			*slot++
			offsets[name] = -8 * (*slot)
		}
	}
	for _, s := range stmts {
		switch s := s.(type) {
		case *LetStmt:
			declare(s.Name)
		case *IfStmt:
			assignSlots(s.Then, slot, offsets)
			assignSlots(s.Else, slot, offsets)
		case *WhileStmt:
			assignSlots(s.Body, slot, offsets)
		}
	}
}

func (g *gen) offset(name string) int {
	off, ok := g.offsets[name]
	if !ok {
		fatalf(0, "undefined variable %q in function %s", name, g.curFunc)
	}
	return off
}

func (g *gen) stmts(ss []Stmt) {
	for _, s := range ss {
		g.stmt(s)
	}
}

func (g *gen) stmt(s Stmt) {
	switch s := s.(type) {
	case *LetStmt:
		g.expr(s.Value)
		g.emit("movq %%rax, %d(%%rbp)", g.offset(s.Name))
	case *AssignStmt:
		g.expr(s.Value)
		g.emit("movq %%rax, %d(%%rbp)", g.offset(s.Name))
	case *ReturnStmt:
		g.expr(s.Value)
		g.emit("leave")
		g.emit("ret")
	case *PrintStmt:
		g.expr(s.Value)
		g.emit("movq %%rax, %%rsi")
		g.emit("leaq .LC0(%%rip), %%rdi")
		g.emit("movq $0, %%rax")
		g.callAligned("printf@PLT")
	case *ExprStmt:
		g.expr(s.X)
	case *IfStmt:
		g.ifStmt(s)
	case *WhileStmt:
		g.whileStmt(s)
	}
}

func (g *gen) ifStmt(s *IfStmt) {
	els := g.newLabel()
	end := g.newLabel()
	g.expr(s.Cond)
	g.emit("cmpq $0, %%rax")
	g.emit("je %s", els)
	g.stmts(s.Then)
	g.emit("jmp %s", end)
	g.label(els)
	g.stmts(s.Else) // nil is fine — emits nothing
	g.label(end)
}

func (g *gen) whileStmt(s *WhileStmt) {
	start := g.newLabel()
	end := g.newLabel()
	g.label(start)
	g.expr(s.Cond)
	g.emit("cmpq $0, %%rax")
	g.emit("je %s", end)
	g.stmts(s.Body)
	g.emit("jmp %s", start)
	g.label(end)
}

func (g *gen) newLabel() string {
	l := fmt.Sprintf(".L%d", g.labelID)
	g.labelID++
	return l
}

// expr generates code that leaves its result in %rax.
func (g *gen) expr(e Expr) {
	switch e := e.(type) {
	case *IntLit:
		g.emit("movq $%d, %%rax", e.Value)
	case *Var:
		g.emit("movq %d(%%rbp), %%rax", g.offset(e.Name))
	case *Unary: // only "-"
		g.expr(e.X)
		g.emit("negq %%rax")
	case *Binary:
		g.expr(e.L) // left -> rax
		g.push()    // save it
		g.expr(e.R) // right -> rax
		g.emit("movq %%rax, %%rcx")
		g.pop("%rax") // left back into rax
		g.binop(e.Op) // rax = rax OP rcx
	case *Call:
		g.call(e)
	}
}

func (g *gen) binop(op string) {
	switch op {
	case "+":
		g.emit("addq %%rcx, %%rax")
	case "-":
		g.emit("subq %%rcx, %%rax")
	case "*":
		g.emit("imulq %%rcx, %%rax")
	case "/":
		g.emit("cqto")
		g.emit("idivq %%rcx")
	default: // comparisons
		g.emit("cmpq %%rcx, %%rax")
		g.emit("%s %%al", setInstr[op])
		g.emit("movzbq %%al, %%rax")
	}
}

func (g *gen) call(c *Call) {
	if len(c.Args) > len(argRegs) {
		fatalf(0, "call to %s has more than %d arguments", c.Name, len(argRegs))
	}
	// Evaluate every argument onto the value stack first, so a nested call
	// can't clobber an argument register we've already loaded.
	for _, a := range c.Args {
		g.expr(a)
		g.push()
	}
	for i := len(c.Args) - 1; i >= 0; i-- {
		g.pop(argRegs[i])
	}
	g.callAligned(c.Name)
}

// callAligned emits a call, fixing 16-byte stack alignment if the value
// stack currently holds an odd number of temporaries.
func (g *gen) callAligned(target string) {
	pad := g.depth%2 == 1
	if pad {
		g.emit("subq $8, %%rsp")
	}
	g.emit("call %s", target)
	if pad {
		g.emit("addq $8, %%rsp")
	}
}

func (g *gen) push() {
	g.emit("pushq %%rax")
	g.depth++
}

func (g *gen) pop(reg string) {
	g.emit("popq %s", reg)
	g.depth--
}
