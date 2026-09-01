# Tin Compiler — Advanced Topics

A roadmap, not a tutorial. Each section names a topic the main
[`guide.md`](guide.md) deliberately skipped, sketches what it involves, and
points at where to start. None of these are fleshed out yet — they're the
next guides to write.

---

## 1. Error recovery and better messages

**What the core does:** stops at the first error, reports a line number, exits.

**What to add:**
- Track **column** as well as line in the lexer, so errors can point at an
  exact position.
- Print a **caret** under the offending token (the `^` pointing at the code).
- **Recover** instead of dying: at a syntax error, skip tokens until a known
  synchronization point (usually the next `;` or `}`), then keep parsing. This
  lets one compile report many errors — the single biggest usability win.

**Where to start:** the parser's `expect`/`fail` helpers. Give the parser a
"panic mode" that records an error and calls a `synchronize()` routine instead
of exiting. *Crafting Interpreters*, ch. 6 ("Parsing Expressions"), has the
canonical treatment.

---

## 2. A type checker

**What the core does:** nothing — every value is an `i64`, so `1 + (2 < 3)`
compiles and runs.

**What to add:**
- A distinct `bool` type produced by comparisons and consumed by
  `if`/`while` conditions.
- A **checking pass** between parser and codegen that walks the AST, infers a
  type for every expression, and reports mismatches (`can't add int and bool`).
- A symbol table carrying types, not just stack offsets.

This is the natural first "middle pass" and the gateway to everything richer:
more types, function signatures, later even generics.

**Where to start:** a new `check.go` pass that runs on the `Program` before
`codegen`. Model it as `func check(p *Program) []error`.

---

## 3. Arrays and strings

**What the core does:** only handles values that fit in one 64-bit register.

**What to add:** the first types that *don't* fit in a register, which forces
real decisions:
- **Memory layout** — where array elements live and how they're addressed
  (`base + index*8`).
- **Pointers** — variables that hold addresses; load/store through them.
- **Allocation** — stack arrays are easy; heap arrays mean calling `malloc`.
- Strings are just byte arrays plus a `.string` in `.rodata` and a way to print
  them (`printf("%s")`).

**Where to start:** add an index expression `a[i]` to the grammar and an array
`let`. Codegen computes the element address, then loads/stores through it.

---

## 4. Register allocation

**What the core does:** the naive stack machine — every intermediate value is
pushed to and popped from the stack.

**What to add:** keep values in registers instead, spilling to the stack only
when you run out. This is the single biggest jump in emitted-code quality.
- Start with a **local** scheme: track which registers are free within one
  expression and use them directly instead of push/pop.
- The full version builds a **live-range interval** or **interference graph**
  and colors it (linear-scan allocation is the practical, teachable algorithm).

**Where to start:** rewrite `gen.expr` to return *which register* holds the
result rather than always using `%rax`, with a small free-register pool and a
spill path. Then read up on linear-scan allocation.

---

## 5. An optimizer and an IR

**What the core does:** emits assembly directly from the AST, unoptimized.

**What to add:**
- An **intermediate representation** (IR) — a simpler, flatter form than the
  AST that's easier to transform. Three-address code or SSA
  (static single assignment) are the usual choices.
- Optimization passes over the IR: **constant folding** (`2 * 3` → `6` at
  compile time), **dead-code elimination**, **common-subexpression
  elimination**.

Constant folding on the AST is a good warm-up you can do without a full IR —
it's a single recursive pass. The IR pays off once you want passes that reason
about control flow.

**Where to start:** write a constant-folding pass over the AST first
(`fold(Expr) Expr`). When that feels limiting, that's your signal to design an
IR.

---

## 6. Other directions

Noted for completeness; unscoped for now.

- **More control flow** — `for`, `break`/`continue`, `switch`.
- **Closures and first-class functions** — captured variables, which force a
  heap-allocated environment.
- **Multiple compilation units** — separate files, an `import`/module system,
  and linking your own objects together.
- **A different backend** — emit LLVM IR (and get its optimizer and every
  target for free) or WebAssembly instead of x86-64.
- **Self-hosting** — rewrite the Tin compiler *in Tin*. The traditional rite of
  passage, and it forces the language to grow until it's actually usable.
