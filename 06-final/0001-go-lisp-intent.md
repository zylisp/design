---
number: 0001
title: Go-Lisp: A Letter of Intent
author: Duncan McGreggor
created: 2025-10-01
updated: 2025-10-04
state: Final
supersedes: None
superseded-by: None
---

# Go-Lisp: A Letter of Intent

**Status**: Pre-design exploration  
**Date**: October 2025  
**Mission**: To create a Zetalisp-inspired S-expression representation of Go's AST

---

## The Vision

We're embarking on a project to create a bidirectional transformation between Go source code and a Lisp-based AST representation. This isn't just another code transformation tool—it's about bringing the power of Lisp's homoiconicity and metaprogramming to the Go ecosystem while respecting Go's semantics and philosophy.

## Core Decisions

### The Foundation: Go's AST

Go provides excellent built-in tooling for AST manipulation:

- **`go/parser`** - Parse Go source into AST
- **`go/ast`** - Complete AST node specifications
- **`go/printer`** - Convert AST back to formatted Go source
- **`go/token`** - Token and position tracking

The Go AST spec is comprehensive and well-defined, with every node type explicitly documented at https://pkg.go.dev/go/ast.

### The Representation: S-Expressions

We'll represent Go's AST as S-expressions that can be:
- Read and written by humans
- Programmatically manipulated
- Converted back to valid Go code

Example transformation:
```go
func add(a, b int) int {
    return a + b
}
```

Becomes:
```lisp
(defun add ((a int) (b int)) int
  (return (+ a b)))
```

### Lisp-1 Semantics

**Decision: We're using Lisp-1 (single namespace) semantics.**

**Why?** Go itself is Lisp-1:
- Functions and variables share the same namespace per scope
- Functions are first-class values
- Local declarations can shadow anything
- `var add = func() {}` and `func add() {}` occupy the same namespace

This semantic alignment is more important than the syntactic convenience of Lisp-2's separate namespaces.

### Zetalisp Aesthetic

**Decision: We're basing our syntax on Zetalisp (Lisp Machine Lisp), not Common Lisp.**

**Why Zetalisp?**
- Cleaner, more orthogonal design than CL's committee compromises
- Beautiful keyword syntax (`:type`, `:return-type`)
- Elegant Flavors system (simpler than CLOS)
- Better historical alignment with Lisp's pure vision
- More fun!

## Design Inspirations

### From Typed Lisps

We drew inspiration from several typed Lisp dialects:

- **Typed Racket**: Mature gradual typing, excellent tooling
- **Coalton**: Hindley-Milner inference for Common Lisp
- **Shen**: Sophisticated sequent calculus type system
- **Carp**: Static typing for real-time applications

### From LFE and Erlang

Robert Virding's work on LFE (Lisp Flavored Erlang) showed us why matching the host language's namespace semantics matters. While LFE chose Lisp-2 to match Erlang's separate function/variable namespaces, we're choosing Lisp-1 to match Go's unified namespace.

## Example Mappings

### Variable Declaration
```go
var x int = 42
```
```lisp
(defvar x int 42)
```

### Function Call
```go
fmt.Println("hello")
```
```lisp
(send fmt :println "hello")
```

### Binary Expression
```go
a + b * 2
```
```lisp
(+ a (* b 2))
```

### If Statement
```go
if x > 10 {
    return true
}
```
```lisp
(if (> x 10)
    (return true))
```

### Struct Definition
```go
type Point struct {
    X int
    Y int
}
```
```lisp
(defstruct point
  (x :type int)
  (y :type int))
```

## The Path Forward

### Phase 1: Design
- Formalize the S-expression syntax
- Map all Go AST node types to S-expression forms
- Define transformation rules
- Create comprehensive examples

### Phase 2: Parser
- Build S-expression → Go AST transformer
- Handle all Go language constructs
- Preserve type information and semantics

### Phase 3: Printer
- Build Go AST → S-expression transformer
- Ensure round-trip fidelity
- Pretty-printing and formatting

### Phase 4: Tooling
- REPL for interactive exploration
- Macro system for code generation
- Integration with Go toolchain
- Editor support

## Why This Matters

**For Go Developers:**
- Powerful metaprogramming capabilities
- Code generation and analysis tools
- Alternative way to think about Go code structure

**For Lisp Enthusiasts:**
- Modern, practical language with Lisp manipulation
- Type safety with Lisp expressiveness
- Bridge between two powerful paradigms

**For Everyone:**
- Exploring language design boundaries
- Learning through alternative representations
- Having fun with code as data

---

## Closing Thoughts

This project sits at the intersection of Lisp's timeless elegance and Go's modern pragmatism. We're not trying to make Go into Lisp or Lisp into Go—we're creating a faithful representation that respects both languages' philosophies.

The goal isn't to replace Go source code, but to complement it. To provide a different lens through which to view, analyze, transform, and generate Go programs. To bring decades of Lisp wisdom to bear on contemporary software engineering challenges.

**Engage!** 🚀

---

*"In Lisp, code is data. In Go, simplicity is elegance. In Go-Lisp, we get both."*