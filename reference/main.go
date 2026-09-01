package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: tinc <file.tin>")
		os.Exit(2)
	}
	src, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "tinc: %v\n", err)
		os.Exit(1)
	}
	toks := lex(string(src))
	prog := parse(toks)
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
