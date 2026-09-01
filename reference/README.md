# Tin reference compiler

The finished compiler from [`../guide.md`](../guide.md). Compiles the Tin
language to x86-64 assembly (AT&T syntax), which `gcc` assembles and links.

## Layout

| File | Stage |
|------|-------|
| `lexer.go`   | source text → tokens |
| `parser.go`  | tokens → AST (AST types live here too) |
| `codegen.go` | AST → x86-64 assembly |
| `main.go`    | CLI glue, shared helpers |

## Use

```sh
go build -o tinc .
./tinc examples/fib.tin > fib.s   # Tin → assembly
gcc fib.s -o fib                  # assemble + link
./fib                             # -> 55
```

## Test

```sh
./test.sh
```

Compiles, links, runs, and checks every `examples/*.tin` against its
`.out` file. Requires `go` and `gcc` on a Linux x86-64 host.
