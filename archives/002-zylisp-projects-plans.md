# Zylisp Architecture & Project Structure

## Overview

Zylisp uses a two-stage compiler architecture with a clean separation between the user-facing language and the Go interop layer.

## Architecture Principles

### Two-Stage Compilation

```
Zylisp Syntax → Canonical S-Expressions → Go AST → Go Code → Binary
  (Stage 1)              (IR)              (Stage 2)
```

**Stage 1**: `zylisp/lang`
- Compiles Zylisp syntax to canonical s-expressions
- Handles macros, type checking, optimization
- Can evolve rapidly without affecting Go interop

**Stage 2**: `zylisp/go-sexp-ast`
- Bidirectional conversion between canonical s-expressions and Go AST
- 1:1 mapping - explicit and unambiguous
- Stable foundation that rarely changes

### Benefits

- **Syntax Flexibility**: Experiment with Zylisp syntax without touching Go interop
- **Stable Interface**: Canonical s-expressions act as intermediate representation (IR)
- **Debugging**: Inspect/validate IR between compilation stages
- **Reusability**: Other tools can target the canonical s-expression format
- **Testability**: Each layer can be tested in isolation

## Project Structure

### zylisp/go-sexp-ast

**Purpose**: Bidirectional s-expression ↔ Go AST conversion

**Responsibilities**:
- Define canonical s-expression format specification
- Parse canonical s-expressions to Go AST
- Generate canonical s-expressions from Go AST
- Provide comprehensive validation

**Characteristics**:
- Stable and boring (in a good way)
- Well-tested and reliable
- Minimal dependencies
- Acts as "assembly for Go"

**Priority**: Build this first and get it solid

### zylisp/lang

**Purpose**: The Zylisp language compiler and CLI tool

**Responsibilities**:
- Parse Zylisp syntax
- Expand macros
- Type checking and validation
- Compile to canonical s-expressions
- CLI tool (`zyc`) for orchestrating compilation
- File watching, build caching, error formatting

**Characteristics**:
- Experimental and volatile
- Where language design happens
- Depends on `go-sexp-ast` for output format

**Note**: Initially includes the `zyc` CLI. May be split into a separate project if CLI grows complex.

### zylisp/core (Future)

**Purpose**: Shared utilities and standard library

**Creation Criteria**:
- Only create when there's clear shared code between projects
- Don't create prematurely

**Potential Contents**:
- Standard library functions
- Runtime support
- Common utilities used by multiple projects

## Development Order

1. **Define canonical s-expression format**
   - Write specification document
   - Define syntax for all Go AST node types
   - Establish conventions and patterns

2. **Build `go-sexp-ast`**
   - Implement s-expression parser
   - Implement Go AST → s-expression generator
   - Implement s-expression → Go AST converter
   - Write comprehensive test suite
   - Validate round-trip conversion

3. **Build Zylisp compiler in `lang`**
   - Design initial Zylisp syntax
   - Implement parser
   - Implement macro system
   - Compile to canonical s-expressions

4. **Iterate on Zylisp syntax**
   - `go-sexp-ast` remains stable
   - Language features evolve freely

## Key Design Questions

### Canonical S-Expression Minimalism
- Should be explicit and unambiguous
- Minimal syntactic sugar
- 1:1 mapping to Go AST nodes
- Think "assembly for Go"

### Error Handling Strategy
- Preserve source locations through both compilation stages
- Map errors back to original Zylisp source
- Provide helpful error messages at each stage

### Macro Expansion Timing
- Macros expand within Stage 1 (Zylisp → canonical s-expressions)
- Output is already-expanded canonical s-expressions
- No macro processing in `go-sexp-ast`

## Repository Organization

```
github.com/zylisp/
├── go-sexp-ast/     # Stable foundation
├── lang/            # Zylisp compiler + zyc CLI
└── core/            # (Created only when needed)
```

## Future Considerations

- Split `zyc` CLI into separate project if it grows complex
- Consider plugins or extensions to `go-sexp-ast` for advanced features
- Potential for other tools to use canonical s-expression format
- May want to version the canonical s-expression spec

## Success Criteria

- Can compile simple Zylisp programs to working Go code
- Round-trip Go code → s-expressions → Go code produces equivalent output
- Clear error messages that reference original Zylisp source
- Fast compilation times
- Easy to experiment with new Zylisp language features