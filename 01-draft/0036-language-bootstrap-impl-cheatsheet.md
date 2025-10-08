---
number: 0036
title: "Zylisp Language Bootstrap Implementation Plan - Cheat Sheet"
author: Unknown
created: 2025-10-08
updated: 2025-10-08
state: Draft
supersedes: None
superseded-by: None
---

# Zylisp Language Bootstrap Implementation Plan - Cheat Sheet

**Quick reference for implementing the Zylisp compiler**

---

## The Big Picture

```
Zylisp Source → Lexer → Reader → Expander → Compiler → zast → Go AST → Go Code → Binary
     .zl         Tokens   S-expr   Core Forms  S-expr    ast.Node   .go      executable
```

**4 Phases**: Literals (1w) → Core Forms (2w) → Macros (1w) → REPL (1w)

---

## Phase 1: Get `42` and `(+ 1 2)` Working

### Files to Create

```
lang/
├── sexpr/types.go          # Int, Symbol, List types
├── parser/
│   ├── lexer.go            # Tokenize: 42, (, ), +
│   ├── reader.go           # Build s-expressions
├── compiler/
│   ├── compiler.go         # Main compiler
│   └── literals.go         # Int → BasicLit, + → BinaryExpr
└── testdata/phase1/
    ├── int_literal.zl      # Contains: 42
    ├── add.zl              # Contains: (+ 1 2)
    └── expected/
        ├── int_literal.txt # Contains: 42
        └── add.txt         # Contains: 3
```

### Key Code Snippets

**S-Expression Types** (`sexpr/types.go`):

```go
type SExpr interface { String() string; sexpr() }
type Int struct { Value int64 }
type Symbol struct { Name string }
type List struct { Elements []SExpr }
```

**Compile Integer** (`compiler/literals.go`):

```go
func (c *Compiler) compileInt(i Int) (string, error) {
    return fmt.Sprintf(`(BasicLit :valuepos 0 :kind INT :value "%d")`, i.Value), nil
}
```

**Compile Addition** (`compiler/literals.go`):

```go
func (c *Compiler) compileBinaryOp(list List) (string, error) {
    left, _ := c.compileExpr(list.Elements[1])
    right, _ := c.compileExpr(list.Elements[2])
    return fmt.Sprintf(`(BinaryExpr :x %s :op ADD :y %s)`, left, right), nil
}
```

### Test & Validate

```bash
cd zylisp/lang
go test ./parser    # Lexer and reader
go test ./compiler  # Compiler
go test .           # Integration (compiles & runs)
```

**Success = Both integration tests pass!**

---

## Phase 2: Add Variables, Functions, Conditionals

### New Core Forms

**let-expr** (variables):

```scheme
(let-expr ((x 10) (y 20)) (+ x y))
```

→ Generates `DeclStmt` with `VAR` tokens

**define-func** (functions):

```scheme
(define-func add (a b)
  (:args int int)
  (:return int)
  (+ a b))
```

→ Generates `FuncDecl` with `FuncType`

**if-expr** (conditionals):

```scheme
(if-expr (< x 0) (- 0 x) x)
```

→ Generates `IfStmt` with then/else blocks

### Files to Add

```
compiler/
├── functions.go    # compileFuncDef, compileFuncCall
└── control.go      # compileLetExpr, compileIfExpr

testdata/phase2/
├── let.zl
├── function.zl
├── factorial.zl    # Recursive test!
└── expected/
```

### Key Pattern: Function Declaration

```go
(FuncDecl
  :name (Ident :name "add")
  :type (FuncType
          :params (FieldList :list (
            (Field :names ((Ident :name "a")) :type (Ident :name "int"))
            (Field :names ((Ident :name "b")) :type (Ident :name "int"))))
          :results (FieldList :list ((Field :type (Ident :name "int")))))
  :body (BlockStmt :list ((ReturnStmt :results (...)))))
```

---

## Phase 3: Add Macros (Sugar Syntax)

### Three Blessed Macros

**deffunc → define-func**:

```scheme
(deffunc add (a b) ...) → (define-func add (a b) ...)
```

**let → let-expr**:

```scheme
(let ((x 10)) ...) → (let-expr ((x 10)) ...)
```

**when → if-expr**:

```scheme
(when test body) → (if-expr test body 0)
```

### File to Create

```
parser/expander.go
```

### Key Function

```go
func (e *Expander) Expand(expr SExpr) (SExpr, error) {
    // Check first element, apply transformation, recurse
}
```

### Update Integration

```go
// Change all tests to use:
coreForm, err := parser.ParseAndExpand(source)
```

---

## Phase 4: Add Basic REPL

### Tier 1: Direct Interpretation

**Can interpret**:

- `42` → instant
- `(+ 1 2)` → instant
- `(+ 1 (* 2 3))` → instant

**Cannot interpret** (needs compilation):

- Functions, let, if

### Files to Create

```
lang/interpreter/eval.go
repl/server/server.go
repl/client/client.go
cmd/zylisp-repl/main.go
```

### Key Pattern: Eval

```go
func Eval(expr SExpr, env *Env) (SExpr, error) {
    switch e := expr.(type) {
    case Int:
        return e, nil
    case List:
        // Dispatch on operator: +, -, *, /
    }
}
```

### Build & Run

```bash
cd cmd/zylisp-repl
go build
./zylisp-repl
```

---

## Common Patterns Reference

### Go AST S-Expressions

| Zylisp | Go AST S-expr |
|--------|---------------|
| `42` | `(BasicLit :kind INT :value "42")` |
| `(+ a b)` | `(BinaryExpr :x <a> :op ADD :y <b>)` |
| `x` | `(Ident :name "x")` |
| `(f a b)` | `(CallExpr :fun (Ident :name "f") :args (<a> <b>))` |

### Go Operators

| Zylisp | Go Token |
|--------|----------|
| `+` | `ADD` |
| `-` | `SUB` |
| `*` | `MUL` |
| `/` | `QUO` |
| `%` | `REM` |
| `<` | `LSS` |
| `<=` | `LEQ` |
| `>` | `GTR` |
| `>=` | `GEQ` |
| `=` | `EQL` |

---

## Essential Commands

```bash
# Test everything
go test ./...

# Test one package
go test ./parser -v

# Test one specific test
go test -v -run TestLexer_IntLiteral

# Clear test cache
go clean -testcache

# Format code
go fmt ./...

# Build REPL
cd cmd/zylisp-repl && go build

# Save generated Go for inspection
# (add to integration test)
os.WriteFile("/tmp/generated.go", formattedGo, 0644)
```

---

## Debugging Checklist

**Test fails?**

- [ ] Run with `-v` flag
- [ ] Check parentheses are balanced
- [ ] Verify token names are UPPERCASE
- [ ] Check all `:field` names are correct
- [ ] Clear test cache

**Generated Go won't compile?**

- [ ] Save to file and inspect
- [ ] Run `go fmt` on it
- [ ] Compare with go-ast-coverage examples
- [ ] Test zast directly with your s-expr

**REPL crashes?**

- [ ] Test simple expr: `42`
- [ ] Check `CanInterpret` logic
- [ ] Add logging to see what's happening
- [ ] Test each operator individually

---

## Adding Features (Quick Guide)

### New Operator

1. Add to lexer (if new token)
2. Add case in `compileBinaryOp`
3. Add eval function in interpreter
4. Add test case
5. Run `go test ./...`

### New Core Form

1. Design the syntax
2. Add `compileXXX` function
3. Add case in `compileExpr`
4. Create test file in testdata
5. Run integration test

### New Macro

1. Design expansion rule
2. Add case in `expandList`
3. Implement expansion function
4. Add test in `expander_test.go`
5. Test with integration test

---

## Success Criteria

**Phase 1**: ✅ `42` and `(+ 1 2)` compile and run
**Phase 2**: ✅ Factorial function works
**Phase 3**: ✅ `deffunc` expands to `define-func`
**Phase 4**: ✅ REPL evaluates arithmetic instantly

---

## Quick Reference: File Locations

```
zylisp/
├── lang/
│   ├── sexpr/types.go           ← S-expression types
│   ├── parser/
│   │   ├── lexer.go             ← Tokenization
│   │   ├── reader.go            ← Parse to s-expr
│   │   └── expander.go          ← Macro expansion
│   ├── compiler/
│   │   ├── compiler.go          ← Main compiler
│   │   ├── literals.go          ← Literals & operators
│   │   ├── functions.go         ← Functions
│   │   └── control.go           ← let, if
│   ├── interpreter/eval.go      ← Direct evaluation
│   └── testdata/
│       ├── phase1/
│       ├── phase2/
│       └── phase3/
├── repl/
│   ├── server/server.go         ← REPL evaluation
│   └── client/client.go         ← Terminal interface
└── cmd/zylisp-repl/main.go      ← REPL entry point
```

---

## Help

**Stuck?** Check these in order:

1. Read error message carefully
2. Run test with `-v` flag
3. Add `fmt.Printf` debugging
4. Simplify to minimal failing case
5. Compare with working Phase 1 code
6. Check go-ast-coverage for examples
7. Test zast directly

**Remember**: One feature at a time. Get it working, then move on!

---

**Version**: 1.0.0 | **Date**: October 2025
