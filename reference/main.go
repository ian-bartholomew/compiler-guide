package main

import (
	"fmt"
	"os"
)

func main() {
	// Optional leading flag: -tokens or -ast stops the pipeline early and
	// dumps the intermediate form, which is handy for following along and
	// for debugging. With no flag, tinc runs the whole pipeline.
	args := os.Args[1:]
	mode := "compile"
	if len(args) > 0 && (args[0] == "-tokens" || args[0] == "-ast") {
		mode = args[0][1:]
		args = args[1:]
	}
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: tinc [-tokens|-ast] <file.tin>")
		os.Exit(2)
	}
	src, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "tinc: %v\n", err)
		os.Exit(1)
	}

	toks := lex(string(src))
	if mode == "tokens" {
		dumpTokens(toks)
		return
	}
	prog := parse(toks)
	if mode == "ast" {
		dumpAST(prog)
		return
	}
	fmt.Print(codegen(prog))
}

// fatalf reports a compile error and exits. Line 0 means "no line info".
func fatalf(line int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if line > 0 {
		fmt.Fprintf(os.Stderr, "tinc: line %d: %s\n", line, msg)
	} else {
		fmt.Fprintf(os.Stderr, "tinc: %s\n", msg)
	}
	os.Exit(1)
}

// align16 rounds n up to the next multiple of 16 (stack alignment).
func align16(n int) int {
	if n%16 == 0 {
		return n
	}
	return n + (16 - n%16)
}
