# Zylisp Forms & Expansion Pipeline - Design Document

**Status:** Planning Stage
**Date:** October 2025

## Overview

This document establishes the terminology, conventions, and architecture for Zylisp's form system, covering the entire pipeline from user-written macros through to IR generation.

## Compilation Pipeline

The Zylisp compiler follows a four-stage pipeline:

1. **Parse** - Text → S-expressions
   - Reader macros are applied at this stage
   - Produces raw S-expression AST

2. **Expand** - S-expressions with macros/sugar → Expanded S-expressions
   - All macros are expanded
   - All syntactic sugar is desugared
   - Results in canonical form S-expressions

3. **Lower** - Expanded S-expressions → ZAST (IR)
   - Canonical forms are translated to the intermediate representation
   - ZAST uses Go AST node names for its representation

4. **Codegen** - ZAST → Go code
   - Final translation to executable Go

### Alternative Terminology Considered

- "Compile" or "translate to IR" instead of "lower"
- "Desugaring" when focusing specifically on syntactic sugar (subset of expansion)
- "Frontend" for the combined Parse→Expand process

**Decision:** Use "expansion" for macro/sugar processing and "lowering" for IR transformation.

## Form Categories

Zylisp has three distinct categories of forms, each serving a different purpose in the language design:

### Category 1: Canonical Sugar-Free Forms

These forms are identical before and after expansion. They represent the core primitives of the language.

**Characteristics:**

- Always hyphenated (e.g., `define-func`, `if-expr`, `let-expr`)
- No consistent verb-noun pattern required
- These are the target forms that macros expand into
- Direct mapping to ZAST nodes

**Examples:**

- `define-func` → `FuncDecl`
- `if-expr` → `IfStmt`
- `let-expr` → `BlockStmt` + `AssignStmt`/`DeclStmt`

### Category 2: Blessed Macros

These are the "one true way" to express certain constructs in Zylisp. They define the language's character and voice.

**Characteristics:**

- Short, easy to type
- Idiomatic and memorable
- No alternatives or aliases
- Always expand to canonical forms
- Part of Zylisp's core identity

**Examples:**

- `deffunc` → expands to `define-func`
- `defvar` → expands to `define-var`
- `defmacro` → expands to `define-macro`

**Design Principle:** These forms are intentionally limited and carefully chosen. They represent how we *want* people to write Zylisp code.

### Category 3: Convenience Macros

Multiple sugar forms that can expand to the same canonical form. These provide flexibility and expressiveness without bloating the core language.

**Characteristics:**

- Multiple options for the same semantic operation
- Named "whatever makes sense for that construct"
- All expand to the same canonical form
- Enable different programming styles

**Examples:**

```lisp
;; All expand to if-expr:
(when test body...)      ; sugar: implicit nil else
(unless test body...)    ; sugar: inverted test
(cond ...)              ; sugar: chained conditionals

;; All expand to loop (ForStmt):
(while test body...)
(until test body...)
(dotimes (i n) body...)
(dolist (x list) body...)
```

## Naming Conventions

### Three-Level Naming System

Every construct in Zylisp has up to three representations:

1. **Sugar/Macro Form** (what users type)
   - Short, memorable, easy to type
   - Examples: `deffunc`, `when`, `fn`

2. **Expanded Form** (canonical representation)
   - Always hyphenated
   - More explicit than sugar forms
   - Examples: `define-func`, `if-expr`, `lambda-expr`

3. **ZAST Node** (IR representation)
   - Uses Go AST node names
   - Examples: `FuncDecl`, `IfStmt`, `FuncLit`

### Notation for Missing Forms

In documentation and tables, a dash (`-`) indicates:

- **Column 1 (Sugar):** No sugar exists; the expanded form is the primary/only way to write it
- **Column 2 (Expanded):** The form doesn't expand; sugar form = expanded form
- **Column 3 (ZAST):** No direct Go AST equivalent; requires special handling in ZAST or runtime

## Mapping to Go AST

### Direct Mappings

Many Zylisp forms map cleanly to Go AST nodes:

- `deffunc` → `define-func` → `FuncDecl`
- `if` → `if-expr` → `IfStmt`
- `fn` → `lambda-expr` → `FuncLit`
- `.` → `method-call` → `SelectorExpr`

### Composite Mappings

Some Zylisp forms require multiple Go AST nodes:

- `let-expr` → `BlockStmt` + `AssignStmt`/`DeclStmt`
- `defmethod` → `FuncDecl` (with receiver)
- Threading macros (`->`, `->>`) → nested `CallExpr`

### No Direct Mapping

Some forms exist only in the Zylisp layer and don't map to Go AST:

- `quote`, `quasiquote`, `unquote`, `unquote-splicing` - compile-time only
- `defmacro`, `macro-expand` - processed during expansion phase
- `list-lit` - no Go equivalent (runtime data structure)
- `keyword` - Zylisp-specific data type
- `try` - must expand to defer/recover pattern

## Design Principles

### User Experience Priorities

1. **Macros and sugar should be short and easy to type**
   - Users write these frequently
   - Optimize for ergonomics

2. **Expanded forms can be longer and more explicit**
   - Users rarely see these
   - Optimize for clarity and unambiguity

3. **Blessed macros define the language's character**
   - Carefully chosen
   - No alternatives or aliases
   - Examples: `deffunc` is THE way to define functions

### Multiple Sugars to One Canonical Form

The language explicitly supports multiple convenience macros expanding to the same canonical form. This enables:

- Different programming styles (imperative vs functional)
- Domain-specific expressiveness
- Gradual learning (simple sugar → advanced features)

**Examples of this pattern:**

- Control flow: `when`, `unless`, `cond` → `if-expr`
- Iteration: `while`, `until`, `dotimes`, `dolist` → `loop`
- Lambdas: `fn`, `λ` → `lambda-expr`

## Implementation Strategy

### Phase 1: Core Forms

Implement the minimal set of canonical forms that map directly to Go AST:

- Function definitions
- Variable/constant definitions
- Basic control flow (if, loop)
- Function calls and operators

### Phase 2: Blessed Macros

Implement the core macros that define Zylisp's voice:

- `deffunc`, `defvar`, `defconst`
- `deftype`, `defstruct`, `definterface`
- Other category 2 forms

### Phase 3: Convenience Macros

Add convenience forms based on usage patterns:

- `when`, `unless`, `cond`
- `while`, `until`, `dotimes`
- Threading macros if desired

### Phase 4: Advanced Features

- Macro system (`defmacro`, quote/unquote)
- Pattern matching
- Advanced error handling patterns

## Open Questions

1. **Should `let` and `let*` have different expanded forms?**
   - Current assumption: both → `let-expr`
   - May need separate forms if sequential vs parallel binding matters

2. **How granular should loop constructs be?**
   - All → `loop` (simple)
   - Separate `for-stmt` vs `range-stmt` (matches Go more closely)

3. **Threading macros (`->`, `->>`) - include in core?**
   - Very useful in functional style
   - Common in Clojure
   - Decision pending based on usage patterns

4. **Error handling - how to handle `try` expansion?**
   - No Go equivalent
   - Must expand to defer/recover pattern
   - Needs careful design

## Zylisp Forms Reference

Three-column mapping of sugar forms → expanded forms → Go AST nodes

### Functions & Definitions

| Sugar/Macro Form | Expanded Form | Go AST Node |
|-----------------|---------------|-------------|
| `deffunc` | `define-func` | `FuncDecl` |
| `defvar` | `define-var` | `ValueSpec` (in `GenDecl`) |
| `defconst` | `define-const` | `ValueSpec` (in `GenDecl`) |
| `deftype` | `define-type` | `TypeSpec` (in `GenDecl`) |
| `defstruct` | `define-struct` | `TypeSpec` (in `GenDecl`) |
| `definterface` | `define-interface` | `TypeSpec` (in `GenDecl`) |
| `defmethod` | `define-method` | `FuncDecl` (with receiver) |

### Control Flow

| Sugar/Macro Form | Expanded Form | Go AST Node |
|-----------------|---------------|-------------|
| `if` | `if-expr` | `IfStmt` |
| `when` | `if-expr` | `IfStmt` |
| `unless` | `if-expr` | `IfStmt` |
| `cond` | `if-expr` | `IfStmt` (nested) |
| - | `switch-expr` | `SwitchStmt` |
| `match` | `switch-expr` | `SwitchStmt` |
| - | `type-switch` | `TypeSwitchStmt` |

### Loops

| Sugar/Macro Form | Expanded Form | Go AST Node |
|-----------------|---------------|-------------|
| - | `loop` | `ForStmt` |
| `while` | `loop` | `ForStmt` |
| `until` | `loop` | `ForStmt` |
| `dotimes` | `loop` | `ForStmt` |
| `dolist` | `loop` | `RangeStmt` |
| `for` | `loop` | `ForStmt` |
| `for-range` | `loop` | `RangeStmt` |
| `break` | `break-stmt` | `BranchStmt` (Break) |
| `continue` | `continue-stmt` | `BranchStmt` (Continue) |
| - | `goto-stmt` | `BranchStmt` (Goto) |
| `return` | `return-stmt` | `ReturnStmt` |

### Bindings & Scope

| Sugar/Macro Form | Expanded Form | Go AST Node |
|-----------------|---------------|-------------|
| `let` | `let-expr` | `BlockStmt` + `AssignStmt`/`DeclStmt` |
| `let*` | `let-expr` | `BlockStmt` + `AssignStmt`/`DeclStmt` |
| - | `lambda-expr` | `FuncLit` |
| `fn` | `lambda-expr` | `FuncLit` |
| `λ` | `lambda-expr` | `FuncLit` |
| `set!` | `set-expr` | `AssignStmt` |

### Data Structures

| Sugar/Macro Form | Expanded Form | Go AST Node |
|-----------------|---------------|-------------|
| - | `list-lit` | - |
| - | `vector-lit` | `CompositeLit` (slice) |
| - | `map-lit` | `CompositeLit` (map) |
| `#[...]` | `vector-lit` | `CompositeLit` (slice) |
| `{...}` | `map-lit` | `CompositeLit` (map) |
| - | `array-lit` | `CompositeLit` (array) |
| - | `struct-lit` | `CompositeLit` (struct) |
| - | `slice-expr` | `SliceExpr` |
| - | `index-expr` | `IndexExpr` |

### Function Application & Operators

| Sugar/Macro Form | Expanded Form | Go AST Node |
|-----------------|---------------|-------------|
| - | `call-expr` | `CallExpr` |
| - | `method-call` | `SelectorExpr` + `CallExpr` |
| `.` | `method-call` | `SelectorExpr` |
| `->` | `call-expr` | `CallExpr` (nested) |
| `->>` | `call-expr` | `CallExpr` (nested) |
| - | `binary-op` | `BinaryExpr` |
| - | `unary-op` | `UnaryExpr` |
| - | `field-access` | `SelectorExpr` |

### Error Handling

| Sugar/Macro Form | Expanded Form | Go AST Node |
|-----------------|---------------|-------------|
| `try` | `try-expr` | - |
| - | `defer-stmt` | `DeferStmt` |
| - | `panic-expr` | `CallExpr` (to panic) |
| - | `recover-expr` | `CallExpr` (to recover) |

### Concurrency

| Sugar/Macro Form | Expanded Form | Go AST Node |
|-----------------|---------------|-------------|
| `go` | `go-expr` | `GoStmt` |
| - | `chan-expr` | `ChanType` or `MakeExpr` |
| - | `send-expr` | `SendStmt` |
| - | `recv-expr` | `UnaryExpr` (with `<-`) |
| `select` | `select-expr` | `SelectStmt` |

### Packages & Imports

| Sugar/Macro Form | Expanded Form | Go AST Node |
|-----------------|---------------|-------------|
| `package` | `package-decl` | `File` (Package field) |
| `import` | `import-decl` | `ImportSpec` (in `GenDecl`) |
| `use` | `import-decl` | `ImportSpec` (in `GenDecl`) |

### Special Forms (Macro System)

| Sugar/Macro Form | Expanded Form | Go AST Node |
|-----------------|---------------|-------------|
| `quote` | `quote-expr` | - |
| `'` | `quote-expr` | - |
| `quasiquote` | `quasiquote-expr` | - |
| `` ` `` | `quasiquote-expr` | - |
| `unquote` | `unquote-expr` | - |
| `,` | `unquote-expr` | - |
| `unquote-splicing` | `unquote-splice-expr` | - |
| `,@` | `unquote-splice-expr` | - |
| `defmacro` | `define-macro` | - |
| `macro-expand` | `macro-expand-expr` | - |

### Type Operations

| Sugar/Macro Form | Expanded Form | Go AST Node |
|-----------------|---------------|-------------|
| - | `type-assert` | `TypeAssertExpr` |
| - | `type-cast` | `CallExpr` (type conversion) |
| `as` | `type-cast` | `CallExpr` (type conversion) |
| - | `typeof-expr` | - |

### Comments & Documentation

| Sugar/Macro Form | Expanded Form | Go AST Node |
|-----------------|---------------|-------------|
| `;` | - | `Comment` |
| `;;;` | `doc-comment` | `CommentGroup` |

### Literals & Primitives

| Sugar/Macro Form | Expanded Form | Go AST Node |
|-----------------|---------------|-------------|
| - | `int-lit` | `BasicLit` (INT) |
| - | `float-lit` | `BasicLit` (FLOAT) |
| - | `string-lit` | `BasicLit` (STRING) |
| - | `bool-lit` | `Ident` (true/false) |
| - | `nil-lit` | `Ident` (nil) |
| - | `symbol` | `Ident` |
| - | `keyword` | - |
| - | `char-lit` | `BasicLit` (CHAR) |
| - | `imaginary-lit` | `BasicLit` (IMAG) |

### Expression & Statement Wrappers

| Sugar/Macro Form | Expanded Form | Go AST Node |
|-----------------|---------------|-------------|
| - | `expr-stmt` | `ExprStmt` |
| - | `block-stmt` | `BlockStmt` |
| - | `empty-stmt` | `EmptyStmt` |
| - | `labeled-stmt` | `LabeledStmt` |
| - | `incr-stmt` | `IncDecStmt` |
| - | `decr-stmt` | `IncDecStmt` |

### Notes

- A dash (`-`) in column 1 means no sugar form (the expanded form is the primary way to write it)
- A dash (`-`) in column 3 means no direct Go AST equivalent (will need special handling in ZAST or runtime)
- Multiple rows with the same Expanded/Go AST values indicate multiple sugar forms that expand to the same thing
- Some Go AST nodes appear multiple times because they're used in different contexts (e.g., `CallExpr` for calls, type conversions, panic/recover)

## Reference

See accompanying **Zylisp Forms Reference** table for complete mapping of:

- Sugar/Macro forms → Expanded forms → Go AST nodes

This table serves as the working document for implementation decisions.

## Next Steps

1. Review and annotate the forms reference table
2. Finalize decisions on open questions
3. Implement parser for core canonical forms
4. Implement macro expander
5. Implement lowering to ZAST
6. Begin codegen implementation

---

*This document will evolve as implementation proceeds and design decisions are validated through practice.*
