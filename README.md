# Writing Your Own Compiler

A build-along guide to writing a small compiler from scratch. You build
**Tin** — a tiny language with functions, recursion, loops, and integer
arithmetic — in Go, emitting real **x86-64 assembly** that `gcc` turns into a
native Linux executable.

No assembly knowledge assumed; the guide opens with a primer covering just
enough to read what the compiler generates.

## Start here

- **[guide.md](guide.md)** — the guide. Primer → lexer → parser → codegen →
  assembling and running. Diagrams render inline on GitHub.
- **[reference/](reference/)** — the finished, runnable compiler to check your
  work against.
- **[advanced.md](advanced.md)** — where to go next: error recovery, a type
  checker, arrays, register allocation, an optimizer.

## Quick run

```sh
cd reference
go build -o tinc .
./tinc examples/fib.tin > fib.s   # Tin → assembly
gcc fib.s -o fib                  # assemble + link
./fib                             # -> 55
```

`./reference/test.sh` compiles, links, runs, and checks every example.
Requires `go` and `gcc` on a Linux x86-64 host.
