---
number: 0035
title: "Zylisp Language Bootstrap Implementation Plan"
author: Unknown
created: 2025-10-08
updated: 2025-10-08
state: Draft
supersedes: None
superseded-by: None
---

# Zylisp Language Bootstrap Implementation Plan

**Version**: 1.0.0
**Date**: October 2025
**Status**: Implementation Guide

---

## Table of Contents

### Getting Started

- [Summary and Intent](#summary-and-intent)
- [Overview](#overview)
  - [The Bootstrap Strategy](#the-bootstrap-strategy)
  - [The Complete Pipeline](#the-complete-pipeline)
  - [Testing Strategy](#testing-strategy)
  - [Repository Structure for Bootstrap](#repository-structure-for-bootstrap)
  - [Phase Timeline](#phase-timeline)

### Implementation Phases

#### [Phase 1: Infrastructure + Literals (Week 1)](#phase-1-infrastructure--literals-week-1)

- [Context for Claude Code](#context-for-claude-code)
- [Task 1: S-Expression Types](#task-1-s-expression-types)
- [Task 2: Lexer](#task-2-lexer)
- [Task 3: Reader (S-Expression Parser)](#task-3-reader-s-expression-parser)
- [Task 4: Compiler Foundation](#task-4-compiler-foundation)
- [Task 5: Integration Test Harness](#task-5-integration-test-harness)
- [Task 6: Setup Instructions](#task-6-setup-instructions)
- [Phase 1 Summary](#summary-of-phase-1-deliverables)

#### [Phase 2: Core Forms (Weeks 2-3)](#phase-2-core-forms-weeks-2-3)

- [High-Level Implementation Guide](#phase-2-high-level-implementation-guide)
- Task 1: Variable Bindings (let-expr)
- Task 2: Function Definitions (define-func)
- Task 3: Function Calls
- Task 4: Conditionals (if-expr)
- Task 5: Integration Tests
- [Phase 2 Summary](#phase-2-summary)

#### [Phase 3: Sugar Layer (Week 4)](#phase-3-sugar-layer-week-4)

- [Implementation Guide](#phase-3-implementation-guide)
- Task 1: Macro Expander Foundation
- Task 2: Macro Implementations
- Task 3: Integration with Parser
- Task 4: Tests
- Task 5: Update Integration Test
- [Phase 3 Summary](#phase-3-summary)

#### [Phase 4: REPL Integration (Week 5)](#phase-4-repl-integration-week-5)

- [Implementation Guide](#phase-4-implementation-guide)
- Task 1: Simple Interpreter
- Task 2: REPL Server
- Task 3: REPL Client
- Task 4: REPL Main Program
- Task 5: Build and Test
- Task 6: README
- [Phase 4 Summary](#phase-4-summary)

### For Humans: Testing and Iteration

#### [Testing and Iteration Guide](#testing-and-iteration-guide-for-humans)

- [How to Test Your Implementation](#how-to-test-your-implementation)
  - Unit Testing
  - Integration Testing
  - REPL Testing
- [How to Iterate and Make Changes](#how-to-iterate-and-make-changes)
  - Adding a New Operator
  - Adding a New Core Form
  - Adding a New Macro
- [Debugging Tips](#debugging-tips)
- [Common Pitfalls](#common-pitfalls)
- [Getting Help](#getting-help)
- [Development Workflow](#development-workflow)

---

## Summary and Intent

This document provides a concrete, step-by-step implementation plan for bootstrapping the Zylisp language from zero to a working compiler. The approach is **bottom-up and testable at every step**: we start with the absolute minimum (integer literals), prove the entire compilation pipeline works end-to-end, then incrementally add forms one at a time.

**Key Principles**:

1. **Test First**: Every form has test cases before implementation
2. **Complete Pipeline**: From Zylisp source → Go AST → Compiled binary
3. **Incremental**: Add one form at a time, always maintaining a working system
4. **Validate Early**: Use `zast` and `go-ast-coverage` from day one
5. **No Premature Abstraction**: Build macro system only after core forms are proven

**Why This Approach?**

- **Immediate Feedback**: Compile and run code from day 1
- **De-risks zast**: Validates Go AST s-expression format early
- **Builds Intuition**: Each form teaches you about the compilation pipeline
- **Prevents Scope Creep**: Forces focus on minimal working system
- **Enables Iteration**: Easy to add forms once pipeline works

**End Goal of Bootstrap**: A minimal but complete Zylisp compiler that can compile simple programs with literals, arithmetic, variables, functions, and conditionals. This proves the architecture and provides a foundation for all future features.

## Overview

### The Bootstrap Strategy

We will implement Zylisp in **four phases**, each building on the previous:

- **Phase 1** (Week 1): Infrastructure + Literals - proves the pipeline
- **Phase 2** (Weeks 2-3): Core forms - builds complete language
- **Phase 3** (Week 4): Sugar layer - proves macros work
- **Phase 4** (Week 5): REPL integration - adds interactive development

### The Complete Pipeline

Every Zylisp expression flows through this pipeline:

    Zylisp Source (.zl file)
        ↓ Lexer
    Tokens
        ↓ Reader
    Surface Forms (S-expressions)
        ↓ Expander
    Core Forms (Canonical IR)
        ↓ Compiler
    Go AST S-expr (via zast)
        ↓ zast.Parse
    Go AST (ast.Node)
        ↓ go/printer
    Go Source (.go file)
        ↓ go build
    Binary
        ↓ execute
    Output

### Testing Strategy

For every form, we write three types of tests:

1. **Parser Tests**: Zylisp source to Core forms
2. **Compiler Tests**: Core forms to Go AST s-expressions
3. **Integration Tests**: End-to-end compilation and execution

### Repository Structure for Bootstrap

    zylisp/
    ├── lang/
    │   ├── parser/
    │   │   ├── lexer.go
    │   │   ├── lexer_test.go
    │   │   ├── reader.go
    │   │   ├── reader_test.go
    │   │   ├── expander.go
    │   │   └── expander_test.go
    │   ├── sexpr/
    │   │   ├── types.go
    │   │   ├── print.go
    │   │   └── equals.go
    │   ├── compiler/
    │   │   ├── compiler.go
    │   │   ├── compiler_test.go
    │   │   ├── literals.go
    │   │   ├── operators.go
    │   │   ├── functions.go
    │   │   └── control.go
    │   └── testdata/
    │       ├── phase1/
    │       ├── phase2/
    │       └── integration_test.go
    ├── zast/
    └── go.mod

### Phase Timeline

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| Phase 1 | 1 week | Compile 42 and (+ 1 2) |
| Phase 2 | 2 weeks | Compile factorial function |
| Phase 3 | 1 week | Macros: deffunc, let, when |
| Phase 4 | 1 week | Basic REPL with tiered execution |
| **Total** | **5 weeks** | **Working Zylisp compiler + REPL** |

---

## Implementation Phases

### Phase 1: Infrastructure + Literals (Week 1)

**Goal**: Prove the entire compilation pipeline works with the simplest possible forms.

**What We'll Implement**:

We'll implement support for exactly two Zylisp forms:

1. Integer literals: `42`
2. Binary addition: `(+ 1 2)`

This minimal set proves every stage of the pipeline:

- Lexer can tokenize numbers and parentheses
- Reader can build s-expressions
- Compiler can generate Go AST s-expressions
- zast can convert to Go AST
- Generated code compiles and runs correctly

**Deliverables**:

- Working lexer for integers, `(`, `)`, `+`
- Working reader for s-expressions
- Compiler that generates `BasicLit` and `BinaryExpr` nodes
- Integration with `zast`
- Test harness that compiles and executes Zylisp code
- Two passing end-to-end tests

---

#### Phase 1: Detailed Implementation Instructions for Claude Code

##### Context for Claude Code

You are implementing the bootstrap phase of the Zylisp programming language. Zylisp is a Lisp dialect that compiles to Go. This is Phase 1: we're implementing the absolute minimum to prove the compilation pipeline works.

**Architecture**:

- Zylisp source → Core forms (IR) → Go AST s-expressions → Go AST → Go code → Binary
- We use the existing `zylisp/zast` library for Go AST ↔ s-expression conversion
- All positions (token.Pos) should be 0 or token.NoPos for now

**Files to Create**:

    zylisp/lang/
    ├── sexpr/
    │   └── types.go
    ├── parser/
    │   ├── lexer.go
    │   ├── lexer_test.go
    │   ├── reader.go
    │   └── reader_test.go
    ├── compiler/
    │   ├── compiler.go
    │   ├── compiler_test.go
    │   └── literals.go
    ├── testdata/
    │   └── phase1/
    │       ├── int_literal.zl
    │       ├── add.zl
    │       └── expected/
    │           ├── int_literal.txt
    │           └── add.txt
    └── integration_test.go

---

##### Task 1: S-Expression Types

**File**: `zylisp/lang/sexpr/types.go`

**Requirement**: Define the core s-expression types that represent parsed Zylisp code.

**Implementation**:

```go
package sexpr

import (
    "fmt"
    "strings"
)

// SExpr is the interface for all s-expression values
type SExpr interface {
    fmt.Stringer
    sexpr() // private method to seal the interface
}

// Int represents an integer literal
type Int struct {
    Value int64
}

func (i Int) String() string { return fmt.Sprintf("%d", i.Value) }
func (i Int) sexpr()          {}

// Symbol represents an identifier or operator
type Symbol struct {
    Name string
}

func (s Symbol) String() string { return s.Name }
func (s Symbol) sexpr()          {}

// List represents a list (function call, special form, etc.)
type List struct {
    Elements []SExpr
}

func (l List) String() string {
    if len(l.Elements) == 0 {
        return "()"
    }

    parts := make([]string, len(l.Elements))
    for i, elem := range l.Elements {
        parts[i] = elem.String()
    }
    return "(" + strings.Join(parts, " ") + ")"
}

func (l List) sexpr() {}

// Helper constructors
func NewInt(value int64) SExpr {
    return Int{Value: value}
}

func NewSymbol(name string) SExpr {
    return Symbol{Name: name}
}

func NewList(elements ...SExpr) SExpr {
    return List{Elements: elements}
}
```

**Test Requirements**:

- Create `types_test.go` with tests for `String()` methods
- Test: `Int{42}.String()` returns `"42"`
- Test: `Symbol{"+"}String()` returns `"+"`
- Test: `List{Symbol{"+"}, Int{1}, Int{2}}.String()` returns `"(+ 1 2)"`

func (l List) sexpr() {}

// Helper constructors
func NewInt(value int64) SExpr {
    return Int{Value: value}
}

func NewSymbol(name string) SExpr {
    return Symbol{Name: name}
}

func NewList(elements ...SExpr) SExpr {
    return List{Elements: elements}
}

```

**Test Requirements**:
- Create `types_test.go` with tests for `String()` methods
- Test: `Int{42}.String()` returns `"42"`
- Test: `Symbol{"+"}String()` returns `"+"`
- Test: `List{Symbol{"+"}, Int{1}, Int{2}}.String()` returns `"(+ 1 2)"`

---

##### Task 2: Lexer

**File**: `zylisp/lang/parser/lexer.go`

**Requirement**: Tokenize Zylisp source into tokens. For Phase 1, only support:
- Integers (e.g., `42`, `123`)
- Left paren `(`
- Right paren `)`
- Symbols (operators like `+`, later identifiers)

**Implementation**:
```go
package parser

import (
    "fmt"
    "strings"
    "unicode"
)

// TokenType represents the type of token
type TokenType int

const (
    TokenEOF TokenType = iota
    TokenInt           // 42
    TokenSymbol        // + or identifier
    TokenLParen        // (
    TokenRParen        // )
)

func (t TokenType) String() string {
    switch t {
    case TokenEOF:
        return "EOF"
    case TokenInt:
        return "INT"
    case TokenSymbol:
        return "SYMBOL"
    case TokenLParen:
        return "LPAREN"
    case TokenRParen:
        return "RPAREN"
    default:
        return "UNKNOWN"
    }
}

// Token represents a lexical token
type Token struct {
    Type   TokenType
    Value  string
    Offset int // Position in source (for error reporting)
}

func (t Token) String() string {
    if t.Value != "" {
        return fmt.Sprintf("%s(%s)", t.Type, t.Value)
    }
    return t.Type.String()
}

// Lexer tokenizes Zylisp source code
type Lexer struct {
    input  string
    pos    int  // current position
    offset int  // current offset for token
}

// NewLexer creates a new lexer
func NewLexer(input string) *Lexer {
    return &Lexer{
        input: input,
        pos:   0,
    }
}

// NextToken returns the next token
func (l *Lexer) NextToken() Token {
    l.skipWhitespace()

    if l.pos >= len(l.input) {
        return Token{Type: TokenEOF, Offset: l.pos}
    }

    l.offset = l.pos
    ch := l.input[l.pos]

    switch ch {
    case '(':
        l.pos++
        return Token{Type: TokenLParen, Value: "(", Offset: l.offset}
    case ')':
        l.pos++
        return Token{Type: TokenRParen, Value: ")", Offset: l.offset}
    default:
        if isDigit(ch) {
            return l.readInt()
        }
        if isSymbolStart(ch) {
            return l.readSymbol()
        }
        panic(fmt.Sprintf("unexpected character: %c at position %d", ch, l.pos))
    }
}

func (l *Lexer) skipWhitespace() {
    for l.pos < len(l.input) && isWhitespace(l.input[l.pos]) {
        l.pos++
    }
}

func (l *Lexer) readInt() Token {
    start := l.pos
    for l.pos < len(l.input) && isDigit(l.input[l.pos]) {
        l.pos++
    }
    value := l.input[start:l.pos]
    return Token{Type: TokenInt, Value: value, Offset: l.offset}
}

func (l *Lexer) readSymbol() Token {
    start := l.pos
    for l.pos < len(l.input) && isSymbolChar(l.input[l.pos]) {
        l.pos++
    }
    value := l.input[start:l.pos]
    return Token{Type: TokenSymbol, Value: value, Offset: l.offset}
}

func isWhitespace(ch byte) bool {
    return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isDigit(ch byte) bool {
    return ch >= '0' && ch <= '9'
}

func isSymbolStart(ch byte) bool {
    // For Phase 1, just operators
    return ch == '+' || ch == '-' || ch == '*' || ch == '/' ||
           unicode.IsLetter(rune(ch))
}

func isSymbolChar(ch byte) bool {
    // Symbols can contain letters, digits, and some special chars
    return unicode.IsLetter(rune(ch)) ||
           unicode.IsDigit(rune(ch)) ||
           strings.ContainsRune("+-*/<>=!?", rune(ch))
}

// AllTokens returns all tokens for testing
func (l *Lexer) AllTokens() []Token {
    var tokens []Token
    for {
        tok := l.NextToken()
        tokens = append(tokens, tok)
        if tok.Type == TokenEOF {
            break
        }
    }
    return tokens
}
```

**File**: `zylisp/lang/parser/lexer_test.go`

```go
package parser

import (
    "testing"
)

func TestLexer_IntLiteral(t *testing.T) {
    lexer := NewLexer("42")
    tokens := lexer.AllTokens()

    expected := []TokenType{TokenInt, TokenEOF}
    if len(tokens) != len(expected) {
        t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
    }

    if tokens[0].Type != TokenInt {
        t.Errorf("expected TokenInt, got %s", tokens[0].Type)
    }
    if tokens[0].Value != "42" {
        t.Errorf("expected value '42', got '%s'", tokens[0].Value)
    }
}

func TestLexer_SimpleAddition(t *testing.T) {
    lexer := NewLexer("(+ 1 2)")
    tokens := lexer.AllTokens()

    expectedTypes := []TokenType{
        TokenLParen,
        TokenSymbol,
        TokenInt,
        TokenInt,
        TokenRParen,
        TokenEOF,
    }

    if len(tokens) != len(expectedTypes) {
        t.Fatalf("expected %d tokens, got %d", len(expectedTypes), len(tokens))
    }

    for i, expected := range expectedTypes {
        if tokens[i].Type != expected {
            t.Errorf("token %d: expected %s, got %s", i, expected, tokens[i].Type)
        }
    }

    if tokens[1].Value != "+" {
        t.Errorf("expected operator '+', got '%s'", tokens[1].Value)
    }
    if tokens[2].Value != "1" {
        t.Errorf("expected value '1', got '%s'", tokens[2].Value)
    }
    if tokens[3].Value != "2" {
        t.Errorf("expected value '2', got '%s'", tokens[3].Value)
    }
}

func TestLexer_WhitespaceHandling(t *testing.T) {
    tests := []struct {
        name  string
        input string
    }{
        {"spaces", "(+ 1 2)"},
        {"tabs", "(+\t1\t2)"},
        {"newlines", "(+\n1\n2)"},
        {"mixed", "( +  1\n\t2 )"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            lexer := NewLexer(tt.input)
            tokens := lexer.AllTokens()

            // Should have same tokens regardless of whitespace
            if len(tokens) != 6 {
                t.Errorf("expected 6 tokens, got %d", len(tokens))
            }
        })
    }
}
```

**Test Requirements**:

- Run `go test ./parser` and ensure all tests pass
- Test cases cover: single int, addition expression, whitespace variations

---

##### Task 3: Reader (S-Expression Parser)

**File**: `zylisp/lang/parser/reader.go`

**Requirement**: Parse tokens into s-expressions. Convert token stream into structured `SExpr` values.

**Implementation**:

```go
package parser

import (
    "fmt"
    "strconv"

    "zylisp/lang/sexpr"
)

// Reader parses tokens into s-expressions
type Reader struct {
    tokens []Token
    pos    int
}

// NewReader creates a new reader from tokens
func NewReader(tokens []Token) *Reader {
    return &Reader{
        tokens: tokens,
        pos:    0,
    }
}

// Read parses one s-expression
func (r *Reader) Read() (sexpr.SExpr, error) {
    if r.pos >= len(r.tokens) {
        return nil, fmt.Errorf("unexpected EOF")
    }

    tok := r.tokens[r.pos]

    switch tok.Type {
    case TokenInt:
        r.pos++
        value, err := strconv.ParseInt(tok.Value, 10, 64)
        if err != nil {
            return nil, fmt.Errorf("invalid integer: %s", tok.Value)
        }
        return sexpr.NewInt(value), nil

    case TokenSymbol:
        r.pos++
        return sexpr.NewSymbol(tok.Value), nil

    case TokenLParen:
        return r.readList()

    case TokenRParen:
        return nil, fmt.Errorf("unexpected ')'")

    case TokenEOF:
        return nil, fmt.Errorf("unexpected EOF")

    default:
        return nil, fmt.Errorf("unexpected token: %s", tok)
    }
}

func (r *Reader) readList() (sexpr.SExpr, error) {
    // Consume '('
    r.pos++

    var elements []sexpr.SExpr

    for {
        if r.pos >= len(r.tokens) {
            return nil, fmt.Errorf("unclosed list")
        }

        tok := r.tokens[r.pos]

        if tok.Type == TokenRParen {
            r.pos++
            return sexpr.NewList(elements...), nil
        }

        elem, err := r.Read()
        if err != nil {
            return nil, err
        }

        elements = append(elements, elem)
    }
}

// Parse is a convenience function that lexes and reads
func Parse(input string) (sexpr.SExpr, error) {
    lexer := NewLexer(input)
    tokens := lexer.AllTokens()

    // Remove EOF token for reader
    if len(tokens) > 0 && tokens[len(tokens)-1].Type == TokenEOF {
        tokens = tokens[:len(tokens)-1]
    }

    if len(tokens) == 0 {
        return nil, fmt.Errorf("empty input")
    }

    reader := NewReader(tokens)
    return reader.Read()
}
```

**File**: `zylisp/lang/parser/reader_test.go`

```go
package parser

import (
    "testing"

    "zylisp/lang/sexpr"
)

func TestReader_IntLiteral(t *testing.T) {
    expr, err := Parse("42")
    if err != nil {
        t.Fatalf("parse error: %v", err)
    }

    intExpr, ok := expr.(sexpr.Int)
    if !ok {
        t.Fatalf("expected Int, got %T", expr)
    }

    if intExpr.Value != 42 {
        t.Errorf("expected value 42, got %d", intExpr.Value)
    }
}

func TestReader_SimpleAddition(t *testing.T) {
    expr, err := Parse("(+ 1 2)")
    if err != nil {
        t.Fatalf("parse error: %v", err)
    }

    list, ok := expr.(sexpr.List)
    if !ok {
        t.Fatalf("expected List, got %T", expr)
    }

    if len(list.Elements) != 3 {
        t.Fatalf("expected 3 elements, got %d", len(list.Elements))
    }

    // Check operator
    op, ok := list.Elements[0].(sexpr.Symbol)
    if !ok || op.Name != "+" {
        t.Errorf("expected operator '+', got %v", list.Elements[0])
    }

    // Check first operand
    arg1, ok := list.Elements[1].(sexpr.Int)
    if !ok || arg1.Value != 1 {
        t.Errorf("expected 1, got %v", list.Elements[1])
    }

    // Check second operand
    arg2, ok := list.Elements[2].(sexpr.Int)
    if !ok || arg2.Value != 2 {
        t.Errorf("expected 2, got %v", list.Elements[2])
    }
}

func TestReader_NestedLists(t *testing.T) {
    expr, err := Parse("(+ 1 (+ 2 3))")
    if err != nil {
        t.Fatalf("parse error: %v", err)
    }

    list, ok := expr.(sexpr.List)
    if !ok {
        t.Fatalf("expected List, got %T", expr)
    }

    if len(list.Elements) != 3 {
        t.Fatalf("expected 3 elements, got %d", len(list.Elements))
    }

    // Third element should be a list
    inner, ok := list.Elements[2].(sexpr.List)
    if !ok {
        t.Fatalf("expected nested List, got %T", list.Elements[2])
    }

    if len(inner.Elements) != 3 {
        t.Errorf("expected 3 elements in nested list, got %d", len(inner.Elements))
    }
}

func TestReader_Errors(t *testing.T) {
    tests := []struct {
        name  string
        input string
    }{
        {"unclosed list", "(+ 1 2"},
        {"unexpected rparen", ")"},
        {"empty", ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := Parse(tt.input)
            if err == nil {
                t.Errorf("expected error for input: %s", tt.input)
            }
        })
    }
}
```

**Test Requirements**:

- Run `go test ./parser` and ensure all reader tests pass
- Test cases cover: literals, simple lists, nested lists, error cases

---

##### Task 4: Compiler Foundation

**File**: `zylisp/lang/compiler/compiler.go`

**Requirement**: Main compiler that dispatches to specific handlers. Converts core forms (s-expressions) to Go AST s-expressions that `zast` can parse.

**Implementation**:

```go
package compiler

import (
    "fmt"

    "zylisp/lang/sexpr"
)

// Compiler converts Zylisp core forms to Go AST s-expressions
type Compiler struct {
    // Future: symbol table, environment, etc.
}

// NewCompiler creates a new compiler
func NewCompiler() *Compiler {
    return &Compiler{}
}

// Compile compiles a core form to a Go AST s-expression string
// The output is an s-expression that zast.Parse() can convert to ast.Node
func (c *Compiler) Compile(expr sexpr.SExpr) (string, error) {
    return c.compileExpr(expr)
}

func (c *Compiler) compileExpr(expr sexpr.SExpr) (string, error) {
    switch e := expr.(type) {
    case sexpr.Int:
        return c.compileInt(e)

    case sexpr.List:
        if len(e.Elements) == 0 {
            return "", fmt.Errorf("cannot compile empty list")
        }

        // Check if it's an operator
        if op, ok := e.Elements[0].(sexpr.Symbol); ok {
            switch op.Name {
            case "+", "-", "*", "/":
                return c.compileBinaryOp(e)
            default:
                return "", fmt.Errorf("unknown operator: %s", op.Name)
            }
        }

        return "", fmt.Errorf("list does not start with operator")

    default:
        return "", fmt.Errorf("cannot compile %T", expr)
    }
}

// CompileProgram wraps an expression in a main function
func (c *Compiler) CompileProgram(expr sexpr.SExpr) (string, error) {
    exprAST, err := c.compileExpr(expr)
    if err != nil {
        return "", err
    }

    // Wrap in a main function that prints the result
    program := fmt.Sprintf(`(File
  :package 1
  :name (Ident :namepos 9 :name "main")
  :decls (
    (FuncDecl
      :name (Ident :namepos 20 :name "main")
      :type (FuncType
              :func 15
              :params (FieldList :opening 24 :closing 25))
      :body (BlockStmt
              :lbrace 27
              :list (
                (ExprStmt
                  :x (CallExpr
                       :fun (Ident :namepos 33 :name "println")
                       :lparen 40
                       :args (%s)
                       :rparen 50)))
              :rbrace 52))))`, exprAST)

    return program, nil
}
```

**File**: `zylisp/lang/compiler/literals.go`

**Requirement**: Compile integer literals to Go `BasicLit` nodes.

**Implementation**:

```go
package compiler

import (
    "fmt"

    "zylisp/lang/sexpr"
)

// compileInt compiles an integer literal
func (c *Compiler) compileInt(i sexpr.Int) (string, error) {
    // Generate: (BasicLit :valuepos 0 :kind INT :value "42")
    return fmt.Sprintf(`(BasicLit :valuepos 0 :kind INT :value "%d")`, i.Value), nil
}

// compileBinaryOp compiles binary arithmetic operators
func (c *Compiler) compileBinaryOp(list sexpr.List) (string, error) {
    if len(list.Elements) != 3 {
        return "", fmt.Errorf("binary operator requires exactly 2 arguments")
    }

    op := list.Elements[0].(sexpr.Symbol)

    // Compile left operand
    left, err := c.compileExpr(list.Elements[1])
    if err != nil {
        return "", fmt.Errorf("compiling left operand: %w", err)
    }

    // Compile right operand
    right, err := c.compileExpr(list.Elements[2])
    if err != nil {
        return "", fmt.Errorf("compiling right operand: %w", err)
    }

    // Map Zylisp operator to Go token
    var goOp string
    switch op.Name {
    case "+":
        goOp = "ADD"
    case "-":
        goOp = "SUB"
    case "*":
        goOp = "MUL"
    case "/":
        goOp = "QUO"
    default:
        return "", fmt.Errorf("unknown operator: %s", op.Name)
    }

    // Generate: (BinaryExpr :x <left> :op ADD :y <right>)
    return fmt.Sprintf(`(BinaryExpr :x %s :oppos 0 :op %s :y %s)`,
        left, goOp, right), nil
}
```

**File**: `zylisp/lang/compiler/compiler_test.go`

```go
package compiler

import (
    "strings"
    "testing"

    "zylisp/lang/sexpr"
)

func TestCompiler_IntLiteral(t *testing.T) {
    c := NewCompiler()

    expr := sexpr.NewInt(42)
    result, err := c.Compile(expr)

    if err != nil {
        t.Fatalf("compile error: %v", err)
    }

    expected := `(BasicLit :valuepos 0 :kind INT :value "42")`
    if result != expected {
        t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
    }
}

func TestCompiler_BinaryAdd(t *testing.T) {
    c := NewCompiler()

    // (+ 1 2)
    expr := sexpr.NewList(
        sexpr.NewSymbol("+"),
        sexpr.NewInt(1),
        sexpr.NewInt(2),
    )

    result, err := c.Compile(expr)
    if err != nil {
        t.Fatalf("compile error: %v", err)
    }

    // Should contain BinaryExpr with ADD
    if !strings.Contains(result, "BinaryExpr") {
        t.Errorf("expected BinaryExpr in output")
    }
    if !strings.Contains(result, "ADD") {
        t.Errorf("expected ADD operator in output")
    }
    if !strings.Contains(result, `"1"`) {
        t.Errorf("expected literal 1 in output")
    }
    if !strings.Contains(result, `"2"`) {
        t.Errorf("expected literal 2 in output")
    }
}

func TestCompiler_NestedExpression(t *testing.T) {
    c := NewCompiler()

    // (+ 1 (+ 2 3))
    inner := sexpr.NewList(
        sexpr.NewSymbol("+"),
        sexpr.NewInt(2),
        sexpr.NewInt(3),
    )

    expr := sexpr.NewList(
        sexpr.NewSymbol("+"),
        sexpr.NewInt(1),
        inner,
    )

    result, err := c.Compile(expr)
    if err != nil {
        t.Fatalf("compile error: %v", err)
    }

    // Should have nested BinaryExpr
    count := strings.Count(result, "BinaryExpr")
    if count != 2 {
        t.Errorf("expected 2 BinaryExpr nodes, found %d", count)
    }
}
```

**Test Requirements**:

- Run `go test ./compiler` and ensure all tests pass
- Verify s-expression output format matches what zast expects

---

##### Task 5: Integration Test Harness

**File**: `zylisp/lang/testdata/phase1/int_literal.zl`

```
42
```

**File**: `zylisp/lang/testdata/phase1/expected/int_literal.txt`

```
42
```

**File**: `zylisp/lang/testdata/phase1/add.zl`

```
(+ 1 2)
```

**File**: `zylisp/lang/testdata/phase1/expected/add.txt`

```
3
```

**File**: `zylisp/lang/integration_test.go`

**Requirement**: End-to-end test that parses Zylisp, compiles to Go, executes, and verifies output.

**Implementation**:

```go
package lang_test

import (
    "bytes"
    "fmt"
    "go/format"
    "go/printer"
    "go/token"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"

    "zylisp/lang/compiler"
    zparser "zylisp/lang/parser"
    "zylisp/zast"
)

func TestPhase1Integration(t *testing.T) {
    tests := []struct {
        name string
        file string
    }{
        {"int_literal", "int_literal.zl"},
        {"add", "add.zl"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Read Zylisp source
            zlPath := filepath.Join("testdata", "phase1", tt.file)
            zlSource, err := os.ReadFile(zlPath)
            if err != nil {
                t.Fatalf("failed to read %s: %v", zlPath, err)
            }

            // Read expected output
            expectedPath := filepath.Join("testdata", "phase1", "expected", tt.name+".txt")
            expectedBytes, err := os.ReadFile(expectedPath)
            if err != nil {
                t.Fatalf("failed to read %s: %v", expectedPath, err)
            }
            expected := strings.TrimSpace(string(expectedBytes))

            // Parse Zylisp to core forms
            coreForm, err := zparser.Parse(string(zlSource))
            if err != nil {
                t.Fatalf("parse error: %v", err)
            }

            // Compile to Go AST s-expression
            c := compiler.NewCompiler()
            goASTSexpr, err := c.CompileProgram(coreForm)
            if err != nil {
                t.Fatalf("compile error: %v", err)
            }

            // Convert s-expression to Go AST using zast
            fset := token.NewFileSet()
            file, err := zast.ParseFile(fset, goASTSexpr)
            if err != nil {
                t.Fatalf("zast parse error: %v", err)
            }

            // Generate Go source code
            var buf bytes.Buffer
            if err := printer.Fprint(&buf, fset, file); err != nil {
                t.Fatalf("failed to generate Go code: %v", err)
            }

            goSource := buf.String()

            // Format the Go code
            formattedGo, err := format.Source(buf.Bytes())
            if err != nil {
                t.Logf("Generated Go code:\n%s", goSource)
                t.Fatalf("failed to format Go code: %v", err)
            }

            // Write Go source to temporary file
            tmpDir := t.TempDir()
            goFile := filepath.Join(tmpDir, "main.go")
            if err := os.WriteFile(goFile, formattedGo, 0644); err != nil {
                t.Fatalf("failed to write Go file: %v", err)
            }

            t.Logf("Generated Go code:\n%s", string(formattedGo))

            // Compile and run
            output, err := compileAndRun(tmpDir, goFile)
            if err != nil {
                t.Fatalf("failed to compile/run: %v", err)
            }

            // Verify output
            actual := strings.TrimSpace(output)
            if actual != expected {
                t.Errorf("output mismatch:\nexpected: %q\ngot:      %q", expected, actual)
            }
        })
    }
}

func compileAndRun(dir, goFile string) (string, error) {
    // Compile
    binary := filepath.Join(dir, "program")
    cmd := exec.Command("go", "build", "-o", binary, goFile)
    cmd.Dir = dir

    var stderr bytes.Buffer
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("compile failed: %v\nstderr: %s", err, stderr.String())
    }

    // Run
    cmd = exec.Command(binary)
    var stdout bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("execution failed: %v\nstderr: %s", err, stderr.String())
    }

    return stdout.String(), nil
}
```

**Test Requirements**:

- Run `go test ./...` from `zylisp/lang` directory
- Both integration tests should pass
- Verify generated Go code is correct
- Verify compiled binary produces expected output

---

##### Task 6: Setup Instructions

**Create**: `zylisp/lang/README.md`

```markdown
# Zylisp Language Implementation - Phase 1

## Setup

1. Ensure Go 1.21+ is installed
2. Clone the repository
3. Install dependencies:
```bash
   go mod download
```

## Running Tests

### All tests

```bash
cd zylisp/lang
go test ./...
```

### Individual packages

```bash
go test ./parser      # Lexer and reader tests
go test ./compiler    # Compiler tests
go test .             # Integration tests
```

### Verbose output

```bash
go test -v ./...
```

## Project Structure

```
lang/
├── sexpr/           # S-expression types
├── parser/          # Lexer and reader
├── compiler/        # Zylisp → Go AST compiler
├── testdata/        # Test cases
│   └── phase1/
│       ├── *.zl           # Zylisp source
│       └── expected/      # Expected outputs
└── integration_test.go    # End-to-end tests
```

## What Phase 1 Implements

- Integer literals: `42`
- Binary addition: `(+ 1 2)`
- Full compilation pipeline: Zylisp → Go AST → Binary

## Next Steps

See `PHASE2.md` for the next implementation phase.

```

**Create**: `zylisp/lang/go.mod`
```

module zylisp/lang

go 1.21

require zylisp/zast v0.1.0

// If zast is in the same repo:
replace zylisp/zast => ../zast

```

---

##### Summary of Phase 1 Deliverables

When Phase 1 is complete, you should have:

- ✅ Working lexer that tokenizes integers and operators
- ✅ Working reader that builds s-expressions
- ✅ Compiler that generates Go AST s-expressions
- ✅ Integration with zast for Go AST conversion
- ✅ End-to-end tests that compile and run Zylisp code
- ✅ Two working examples: `42` and `(+ 1 2)`

**Validation Checklist**:
```bash
cd zylisp/lang
go test ./parser           # Should pass 10+ tests
go test ./compiler         # Should pass 5+ tests
go test .                  # Should pass 2 integration tests
```

---

### Phase 2: Core Forms (Weeks 2-3)

**Goal**: Build a minimal but complete language with variables, functions, and conditionals.

**What We'll Implement**:

We'll add the following core forms (no sugar, just the canonical representations):

1. **Variables**: `let-expr` for local bindings
2. **Functions**: `define-func` for function definitions
3. **Function Calls**: Calling user-defined functions
4. **Conditionals**: `if-expr` for branching

This gives us a complete (if verbose) programming language. Users can write:

- Factorial function
- Fibonacci function
- Simple recursive programs

**Deliverables**:

- `let-expr` compilation
- `define-func` compilation
- Function call compilation
- `if-expr` compilation
- Multiple function definitions in one program
- Test suite including factorial and fibonacci

---

#### Phase 2: High-Level Implementation Guide

**Note**: Phase 2 is more complex than Phase 1. The detailed Claude Code instructions would be very long. Here's the high-level approach:

##### Task 1: Variable Bindings (let-expr)

**Core Form**:

```scheme
(let-expr
  ((x 10)
   (y 20))
  (+ x y))
```

**Implementation Steps**:

1. Add `Keyword` type to sexpr (for `:args`, `:return`, etc.)
2. Update lexer to recognize `:` prefix
3. Add `compileLetExpr` that generates `BlockStmt` with `DeclStmt` nodes
4. Add test case: `testdata/phase2/let.zl`

**Generated Go AST Pattern**:

```
(BlockStmt
  :list (
    (DeclStmt :decl (GenDecl :tok VAR :specs ((ValueSpec :names ((Ident :name "x")) :values (...)))))
    (DeclStmt :decl (GenDecl :tok VAR :specs ((ValueSpec :names ((Ident :name "y")) :values (...)))))
    (ExprStmt :x <body>)))
```

##### Task 2: Function Definitions (define-func)

**Core Form**:

```scheme
(define-func add (a b)
  (:args int int)
  (:return int)
  (+ a b))
```

**Implementation Steps**:

1. Add `compileFuncDef` that parses params, types, and body
2. Generate `FuncDecl` with `FuncType` and parameter list
3. Wrap body in `ReturnStmt`
4. Update `CompileProgram` to handle multiple top-level forms
5. Add test case: `testdata/phase2/function.zl`

**Generated Go AST Pattern**:

```
(FuncDecl
  :name (Ident :name "add")
  :type (FuncType
          :params (FieldList :list ((Field :names ((Ident :name "a")) :type (Ident :name "int"))
                                    (Field :names ((Ident :name "b")) :type (Ident :name "int"))))
          :results (FieldList :list ((Field :type (Ident :name "int")))))
  :body (BlockStmt :list ((ReturnStmt :results (...)))))
```

##### Task 3: Function Calls

**Implementation Steps**:

1. Add `compileFuncCall` that generates `CallExpr`
2. Update `compileExpr` to distinguish operators from function calls
3. Handle variable references (just `Ident` nodes)
4. Test with factorial function

**Generated Go AST Pattern**:

```
(CallExpr
  :fun (Ident :name "add")
  :args ((BasicLit ...) (BasicLit ...)))
```

##### Task 4: Conditionals (if-expr)

**Core Form**:

```scheme
(if-expr (< x 0)
  (- 0 x)
  x)
```

**Implementation Steps**:

1. Add comparison operators: `<`, `<=`, `>`, `>=`, `=`
2. Add `compileIfExpr` that generates `IfStmt`
3. Generate separate blocks for then/else branches
4. Test with absolute value function

**Generated Go AST Pattern**:

```
(IfStmt
  :cond (BinaryExpr ...)
  :body (BlockStmt :list ((ExprStmt ...)))
  :else (BlockStmt :list ((ExprStmt ...))))
```

##### Task 5: Integration Tests

**Test Cases to Create**:

1. `testdata/phase2/let.zl` - Variable bindings
2. `testdata/phase2/function.zl` - Simple function
3. `testdata/phase2/factorial.zl` - Recursive factorial
4. `testdata/phase2/fibonacci.zl` - Recursive fibonacci

**Example factorial.zl**:

```scheme
(define-func factorial (n)
  (:args int)
  (:return int)
  (if-expr (<= n 1)
    1
    (* n (factorial (- n 1)))))

(define-func main ()
  (:args)
  (:return)
  (let-expr
    ((result (factorial 5)))
    result))
```

Expected output: `120`

---

##### Phase 2 Summary

When Phase 2 is complete, you should have:

- ✅ Local variable bindings with `let-expr`
- ✅ Function definitions with `define-func`
- ✅ Function calls
- ✅ Conditionals with `if-expr`
- ✅ Comparison operators (`<`, `<=`, `>`, `>=`, `=`)
- ✅ Working recursive functions (factorial, fibonacci)

**Validation**:

```bash
cd zylisp/lang
go test ./...
# Should now pass 20+ tests including factorial and fibonacci
```

---

### Phase 3: Sugar Layer (Week 4)

**Goal**: Add macros that expand to core forms, proving the macro system works.

**What We'll Implement**:

1. **Macro Expander**: `parser/expander.go`
2. **First Macros**: `deffunc`, `let`, `when`
3. **Macro Tests**: Verify expansion to core forms

This proves that surface syntax can differ from core forms, and that the macro system works correctly.

**Deliverables**:

- Working macro expansion
- Three blessed macros
- Tests showing surface → core form transformation
- Updated integration tests using sugar syntax

---

#### Phase 3: Implementation Guide

##### Task 1: Macro Expander Foundation

**File**: `zylisp/lang/parser/expander.go`

**High-Level Structure**:

```go
package parser

type Expander struct {
    // Future: macro definitions table
}

func NewExpander() *Expander

func (e *Expander) Expand(expr sexpr.SExpr) (sexpr.SExpr, error)

func (e *Expander) expandList(list sexpr.List) (sexpr.SExpr, error)

func (e *Expander) expandDeffunc(list sexpr.List) (sexpr.SExpr, error)

func (e *Expander) expandLet(list sexpr.List) (sexpr.SExpr, error)

func (e *Expander) expandWhen(list sexpr.List) (sexpr.SExpr, error)
```

**Key Pattern**:

- Check first element of list
- If it matches a macro name, apply transformation
- Otherwise, recursively expand all elements

##### Task 2: Macro Implementations

**deffunc → define-func**:

```scheme
; Surface
(deffunc add (a b)
  (:args int int)
  (:return int)
  (+ a b))

; Expands to
(define-func add (a b)
  (:args int int)
  (:return int)
  (+ a b))
```

Simple replacement of the operator symbol.

**let → let-expr**:

```scheme
; Surface
(let ((x 10) (y 20))
  (+ x y))

; Expands to
(let-expr ((x 10) (y 20))
  (+ x y))
```

Simple replacement of the operator symbol.

**when → if-expr**:

```scheme
; Surface
(when (> x 0)
  (print x))

; Expands to
(if-expr (> x 0)
  (print x)
  0)  ; implicit else (using 0 as nil for now)
```

Adds implicit else branch.

##### Task 3: Integration with Parser

**Update**: `zylisp/lang/parser/reader.go`

Add new function:

```go
// ParseAndExpand parses Zylisp source and expands macros to core forms
func ParseAndExpand(input string) (sexpr.SExpr, error) {
    // Parse to surface forms
    surface, err := Parse(input)
    if err != nil {
        return nil, err
    }

    // Expand macros to core forms
    expander := NewExpander()
    core, err := expander.Expand(surface)
    if err != nil {
        return nil, fmt.Errorf("macro expansion: %w", err)
    }

    return core, nil
}
```

##### Task 4: Tests

**Expander Unit Tests** (`parser/expander_test.go`):

- Test each macro expansion individually
- Verify surface form → core form transformation
- Test nested expansions

**Integration Tests**:

**File**: `testdata/phase3/sugar_function.zl`

```scheme
(deffunc square (x)
  (:args int)
  (:return int)
  (* x x))

(deffunc main ()
  (:args)
  (:return)
  (square 5))
```

Expected output: `25`

**File**: `testdata/phase3/sugar_let.zl`

```scheme
(deffunc compute ()
  (:args)
  (:return int)
  (let ((x 10)
        (y 20)
        (z 30))
    (+ (+ x y) z)))

(deffunc main ()
  (:args)
  (:return)
  (compute))
```

Expected output: `60`

##### Task 5: Update Integration Test

**Update**: `zylisp/lang/integration_test.go`

Change all integration tests to use `ParseAndExpand` instead of `Parse`:

```go
// Parse and expand macros to core forms
coreForm, err := zparser.ParseAndExpand(string(zlSource))
if err != nil {
    t.Fatalf("parse/expand error: %v", err)
}
```

Add new Phase 3 test cases to test suite.

---

##### Phase 3 Summary

When Phase 3 is complete, you should have:

- ✅ Working macro expander
- ✅ Three blessed macros: `deffunc`, `let`, `when`
- ✅ Tests showing surface → core transformation
- ✅ Integration tests using sugar syntax
- ✅ Foundation for adding more macros

**Validation**:

```bash
cd zylisp/lang
go test ./parser -v       # Should pass expander tests
go test . -v              # Should pass Phase 3 integration tests
```

**Key Achievement**: You've now proven that Zylisp can have a friendly surface syntax that expands to a simple core language. This separation is crucial for language evolution.

---

### Phase 4: REPL Integration (Week 5)

**Goal**: Add basic REPL with tiered execution, proving the three-tier strategy works.

**What We'll Implement**:

1. **Simple Interpreter**: Direct evaluation for literals and arithmetic
2. **REPL Server**: Basic evaluation loop
3. **REPL Client**: Terminal interface
4. **Integration**: Wire up interpreter + compiler

This validates the REPL architecture and tiered execution strategy before adding complexity like worker supervision.

**Deliverables**:

- Working interpreter for Tier 1
- Basic REPL server
- Terminal client
- Demonstration of fast evaluation for simple expressions

---

#### Phase 4: Implementation Guide

##### Task 1: Simple Interpreter

**File**: `zylisp/lang/interpreter/eval.go`

**Key Components**:

1. **Environment** - stores variable bindings

```go
type Env struct {
    bindings map[string]sexpr.SExpr
}
```

2. **Eval Function** - evaluates simple expressions

```go
func Eval(expr sexpr.SExpr, env *Env) (sexpr.SExpr, error)
```

3. **CanInterpret** - determines if expression can be interpreted

```go
func CanInterpret(expr sexpr.SExpr) bool
```

**What Can Be Interpreted (Tier 1)**:

- Integer literals: `42`
- Simple arithmetic on literals: `(+ 1 2)`, `(* 3 4)`
- Nested arithmetic on literals: `(+ 1 (* 2 3))`
- Variable lookups (for REPL state)

**What Cannot Be Interpreted** (needs compilation):

- Function definitions
- Function calls
- Let bindings
- Conditionals

**Implementation Pattern**:

```go
func Eval(expr sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    switch e := expr.(type) {
    case sexpr.Int:
        return e, nil

    case sexpr.Symbol:
        // Variable lookup
        val, ok := env.Get(e.Name)
        if !ok {
            return nil, fmt.Errorf("undefined: %s", e.Name)
        }
        return val, nil

    case sexpr.List:
        // Check for operators
        if op, ok := e.Elements[0].(sexpr.Symbol); ok {
            switch op.Name {
            case "+":
                return evalAdd(e.Elements[1:], env)
            case "-":
                return evalSub(e.Elements[1:], env)
            // ... etc
            }
        }
    }
    return nil, fmt.Errorf("cannot interpret")
}
```

##### Task 2: REPL Server

**File**: `zylisp/repl/server/server.go`

**Structure**:

```go
type Server struct {
    env      *interpreter.Env
    compiler *compiler.Compiler
}

func NewServer() *Server

func (s *Server) Eval(source string) (string, error)
```

**Evaluation Strategy**:

```go
func (s *Server) Eval(source string) (string, error) {
    // 1. Parse and expand
    coreForm, err := parser.ParseAndExpand(source)
    if err != nil {
        return "", err
    }

    // 2. Try Tier 1: Direct interpretation
    if interpreter.CanInterpret(coreForm) {
        result, err := interpreter.Eval(coreForm, s.env)
        if err != nil {
            return "", err
        }
        return result.String(), nil
    }

    // 3. Tier 2/3: Compilation (not implemented in Phase 4)
    return "", fmt.Errorf("compilation not yet implemented in REPL")
}
```

**Note**: Phase 4 only implements Tier 1. Tiers 2 and 3 will be added in future phases.

##### Task 3: REPL Client

**File**: `zylisp/repl/client/client.go`

**Simple Terminal Interface**:

```go
type Client struct {
    server *server.Server
    reader *bufio.Reader
    writer io.Writer
}

func (c *Client) Run() error {
    fmt.Fprintln(c.writer, "Zylisp REPL v0.1.0 - Phase 4 Bootstrap")
    fmt.Fprintln(c.writer, "Type expressions to evaluate, or :quit to exit")

    for {
        fmt.Fprint(c.writer, "zylisp> ")

        line, err := c.reader.ReadString('\n')
        if err != nil {
            return err
        }

        line = strings.TrimSpace(line)

        if line == ":quit" || line == ":q" {
            return nil
        }

        result, err := c.server.Eval(line)
        if err != nil {
            fmt.Fprintf(c.writer, "Error: %v\n", err)
            continue
        }

        fmt.Fprintf(c.writer, "=> %s\n", result)
    }
}
```

##### Task 4: REPL Main Program

**File**: `zylisp/cmd/zylisp-repl/main.go`

```go
package main

import (
    "fmt"
    "os"

    "zylisp/repl/client"
)

func main() {
    c := client.NewClient(os.Stdin, os.Stdout)

    if err := c.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "REPL error: %v\n", err)
        os.Exit(1)
    }
}
```

**Module Setup**:

```
zylisp/
├── cmd/
│   └── zylisp-repl/
│       ├── main.go
│       └── go.mod
├── repl/
│   ├── server/
│   │   ├── server.go
│   │   └── server_test.go
│   ├── client/
│   │   └── client.go
│   └── go.mod
└── lang/
    └── ...
```

##### Task 5: Build and Test

**Build Instructions**:

```bash
cd zylisp/cmd/zylisp-repl
go mod init zylisp/cmd/zylisp-repl
go mod edit -replace zylisp/lang=../../lang
go mod edit -replace zylisp/repl=../../repl
go build -o zylisp-repl
```

**Test Session**:

```
$ ./zylisp-repl
Zylisp REPL v0.1.0 - Phase 4 Bootstrap
Type expressions to evaluate, or :quit to exit

zylisp> 42
=> 42

zylisp> (+ 1 2)
=> 3

zylisp> (+ 10 (* 5 6))
=> 40

zylisp> (* (+ 2 3) (- 10 4))
=> 30

zylisp> (deffunc square (x) (:args int) (:return int) (* x x))
Error: compilation not yet implemented in REPL

zylisp> :quit
Goodbye!
```

##### Task 6: README

**File**: `zylisp/repl/README.md`

Document:

- What works (Tier 1: literals and arithmetic)
- What doesn't work yet (functions, let, if)
- Build instructions
- Example session
- Next steps (Tier 2/3 will be added in future phases)

---

##### Phase 4 Summary

When Phase 4 is complete, you should have:

- ✅ Working interpreter for simple expressions
- ✅ Basic REPL server
- ✅ Terminal client interface
- ✅ Sub-millisecond evaluation for arithmetic
- ✅ Foundation for adding Tier 2/3 execution

**Validation**:

```bash
# Build REPL
cd zylisp/cmd/zylisp-repl
go build

# Run REPL
./zylisp-repl

# Try expressions:
# 42
# (+ 1 2)
# (* (+ 2 3) (- 10 4))
```

**Key Achievement**: You now have an interactive REPL that demonstrates the tiered execution strategy. Simple expressions are evaluated instantly without compilation overhead.

---

## Testing and Iteration Guide for Humans

### How to Test Your Implementation

#### 1. Unit Testing

After implementing each component, run its tests:

```bash
# Test lexer
cd zylisp/lang/parser
go test -v -run TestLexer

# Test reader
go test -v -run TestReader

# Test compiler
cd ../compiler
go test -v

# Test interpreter
cd ../interpreter
go test -v

# Test all
cd ..
go test ./...
```

#### 2. Integration Testing

Test end-to-end compilation:

```bash
cd zylisp/lang
go test -v .
```

Look for output showing:

- Zylisp source
- Generated Go code
- Execution output

#### 3. REPL Testing

Build and run the REPL:

```bash
cd zylisp/cmd/zylisp-repl
go build
./zylisp-repl
```

Try these test cases:

```
42                           # Should return: 42
(+ 1 2)                      # Should return: 3
(* 5 6)                      # Should return: 30
(+ 10 (* 5 6))              # Should return: 40
(/ (- 100 20) 4)            # Should return: 20
```

### How to Iterate and Make Changes

#### Adding a New Operator

1. **Update Lexer** (if new token needed)
   - Add token recognition in `lexer.go`
   - Add test case in `lexer_test.go`

2. **Update Compiler**
   - Add case in `compileBinaryOp` (for operators)
   - Map to appropriate Go token
   - Add test in `compiler_test.go`

3. **Update Interpreter** (for REPL support)
   - Add evaluation function
   - Add test in `eval_test.go`

4. **Test End-to-End**
   - Create `.zl` test file
   - Add expected output
   - Run integration test

**Example: Adding Modulo (`%`)**

```go
// In compiler/literals.go
case "%":
    goOp = "REM"

// In interpreter/eval.go
case "%":
    return evalMod(e.Elements[1:], env)

func evalMod(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    // Similar to evalDiv but with % operator
    if len(args) != 2 {
        return nil, fmt.Errorf("%% requires exactly 2 arguments")
    }

    left, err := Eval(args[0], env)
    if err != nil {
        return nil, err
    }

    right, err := Eval(args[1], env)
    if err != nil {
        return nil, err
    }

    leftInt, ok := left.(sexpr.Int)
    if !ok {
        return nil, fmt.Errorf("%% requires integer arguments")
    }

    rightInt, ok := right.(sexpr.Int)
    if !ok {
        return nil, fmt.Errorf("%% requires integer arguments")
    }

    if rightInt.Value == 0 {
        return nil, fmt.Errorf("modulo by zero")
    }

    return sexpr.NewInt(leftInt.Value % rightInt.Value), nil
}
```

#### Adding a New Core Form

1. **Define the Form**
   - Document syntax in comments
   - Add example usage

2. **Update Compiler**
   - Add compilation function
   - Handle in `compileExpr` switch
   - Add tests

3. **Add Test Cases**
   - Create `.zl` file in `testdata/`
   - Add expected output
   - Run integration test

4. **Update Documentation**
   - Add to forms reference
   - Include examples

**Example: Adding `begin` for Multiple Expressions**

```go
// In compiler/control.go
func (c *Compiler) compileBegin(list sexpr.List) (string, error) {
    // (begin expr1 expr2 expr3)
    // Compile all expressions as statements
    var stmts []string
    for _, expr := range list.Elements[1:] {
        compiled, err := c.compileExpr(expr)
        if err != nil {
            return "", err
        }
        stmt := fmt.Sprintf("(ExprStmt :x %s)", compiled)
        stmts = append(stmts, stmt)
    }

    return fmt.Sprintf("(BlockStmt :lbrace 0 :list (%s) :rbrace 0)",
        strings.Join(stmts, " ")), nil
}

// In compiler/compiler.go compileExpr switch:
case "begin":
    return c.compileBegin(e)
```

#### Adding a New Macro

1. **Define Expansion Rule**
   - Document what sugar expands to
   - Show examples

2. **Add to Expander**
   - Add case in `expandList`
   - Implement expansion function
   - Add test in `expander_test.go`

3. **Test Expansion**
   - Verify surface → core transformation
   - Test via integration tests

**Example: Adding `unless` Macro**

```go
// In parser/expander.go
case "unless":
    return e.expandUnless(list)

func (e *Expander) expandUnless(list sexpr.List) (sexpr.SExpr, error) {
    // (unless condition body)
    // -> (if-expr (not condition) body 0)

    if len(list.Elements) < 3 {
        return nil, fmt.Errorf("unless requires condition and body")
    }

    cond, err := e.Expand(list.Elements[1])
    if err != nil {
        return nil, err
    }

    // Negate condition - need to add 'not' operator first
    negatedCond := sexpr.NewList(sexpr.NewSymbol("not"), cond)

    body, err := e.Expand(list.Elements[2])
    if err != nil {
        return nil, err
    }

    return sexpr.NewList(
        sexpr.NewSymbol("if-expr"),
        negatedCond,
        body,
        sexpr.NewInt(0), // nil equivalent
    ), nil
}
```

### Debugging Tips

#### 1. Trace Compilation Pipeline

Add logging at each stage:

```go
// In integration test
coreForm, _ := parser.ParseAndExpand(source)
fmt.Printf("Core form: %s\n", coreForm)

goASTSexpr, _ := compiler.Compile(coreForm)
fmt.Printf("Go AST s-expr: %s\n", goASTSexpr)

goAST, _ := zast.Parse(goASTSexpr)
fmt.Printf("Go AST: %#v\n", goAST)
```

#### 2. Inspect Generated Go Code

Save to file for inspection:

```go
os.WriteFile("/tmp/generated.go", formattedGo, 0644)
```

Then examine:

```bash
cat /tmp/generated.go
go run /tmp/generated.go
```

#### 3. Compare with go-ast-coverage

When stuck, look at similar code in `go-ast-coverage`:

```bash
cd zylisp/go-ast-coverage
grep -r "BinaryExpr" .
cat source/basics/arithmetic.go
```

#### 4. Use zast Directly

Test zast parsing in isolation:

```go
sexprStr := `(BinaryExpr :x (BasicLit :kind INT :value "1") :op ADD :y (BasicLit :kind INT :value "2"))`

fset := token.NewFileSet()
node, err := zast.ParseExpr(fset, sexprStr)
if err != nil {
    t.Fatal(err)
}

// Print to see generated Go code
printer.Fprint(os.Stdout, fset, node)
```

#### 5. Incremental Testing

When something breaks:

1. Identify the smallest test case that fails
2. Test each stage individually (lexer → reader → compiler → zast)
3. Fix the earliest stage that's broken
4. Repeat

### Common Pitfalls

1. **Forgetting to Expand Macros**
   - Use `ParseAndExpand` not just `Parse`
   - Test core forms, not surface syntax

2. **Position Values**
   - Use `0` or `token.NoPos` for now
   - Will add proper positions in later phases

3. **Type Mismatches**
   - Check s-expression types with type assertions
   - Add helpful error messages

4. **Missing Test Cases**
   - Always add both unit and integration tests
   - Test error cases, not just happy path

5. **Nested Code Blocks in Strings**
   - When generating s-expressions with nested parens
   - Make sure parentheses are balanced
   - Use `fmt.Sprintf` carefully

6. **Import Paths**
   - Make sure `go.mod` files have correct `replace` directives
   - Run `go mod tidy` after adding dependencies

### Getting Help

If something isn't working:

1. **Check the Tests**
   - Run with `-v` flag to see details
   - Look at what's actually being compared

2. **Simplify**
   - Remove complexity until it works
   - Add back incrementally

3. **Compare with Examples**
   - Look at Phase 1 tests
   - Follow the same pattern

4. **Verify zast**
   - Make sure zast itself works
   - Test with known-good s-expressions from go-ast-coverage

5. **Read Error Messages Carefully**
   - Go's error messages are usually very specific
   - Look at line numbers and stack traces

### Development Workflow

**Recommended cycle for each new feature**:

1. **Write test case first** (TDD style)
   - Create `.zl` file in testdata
   - Add expected output
   - Test will fail initially

2. **Implement minimal code**
   - Add lexer support if needed
   - Add compiler case
   - Generate correct Go AST s-expression

3. **Run tests**

```bash
   go test -v ./...
```

4. **Debug failures**
   - Add logging
   - Inspect generated code
   - Compare with similar working examples

5. **Iterate until green**
   - Fix one issue at a time
   - Re-run tests frequently

6. **Refactor if needed**
   - Clean up code
   - Add comments
   - Ensure tests still pass

7. **Commit**
   - Commit working code
   - Write descriptive commit message

**Example workflow for adding subtraction operator**:

```bash
# 1. Create test
echo "(- 5 3)" > testdata/phase1/subtract.zl
echo "2" > testdata/phase1/expected/subtract.txt

# 2. Run test (will fail)
go test -v . -run TestPhase1Integration/subtract

# 3. Add compiler support
# Edit compiler/literals.go to add "-" case

# 4. Test again
go test -v . -run TestPhase1Integration/subtract

# 5. Add interpreter support for REPL
# Edit interpreter/eval.go to add evalSub

# 6. Test interpreter
go test -v ./interpreter -run TestEval

# 7. All tests passing? Commit!
git add .
git commit -m "Add subtraction operator"
```

---

## Conclusion

This bootstrap plan provides a concrete, testable path to a working Zylisp compiler:

- **Phase 1** (1 week): Proves the pipeline with minimal forms
- **Phase 2** (2 weeks): Builds a complete (if verbose) language
- **Phase 3** (1 week): Adds sugar and proves macros work
- **Phase 4** (1 week): Adds REPL with fast interpretation

Each phase delivers working, testable code. By the end, you'll have:

- ✅ Working compiler
- ✅ Macro system
- ✅ Basic REPL
- ✅ Foundation for all future features

The key is **incremental progress** with **continuous validation**. Never move to the next form until the current one compiles and runs correctly.

---

## Quick Reference

### File Structure Overview

```
zylisp/
├── lang/                           # Language implementation
│   ├── sexpr/                      # S-expression types
│   │   └── types.go
│   ├── parser/                     # Lexer, reader, expander
│   │   ├── lexer.go
│   │   ├── reader.go
│   │   └── expander.go            # Phase 3
│   ├── compiler/                   # Zylisp → Go AST
│   │   ├── compiler.go
│   │   ├── literals.go
│   │   ├── operators.go           # Phase 2
│   │   ├── functions.go           # Phase 2
│   │   └── control.go             # Phase 2
│   ├── interpreter/                # Phase 4
│   │   └── eval.go
│   └── testdata/
│       ├── phase1/
│       ├── phase2/
│       ├── phase3/
│       └── integration_test.go
│
├── repl/                           # Phase 4
│   ├── server/
│   │   └── server.go
│   └── client/
│       └── client.go
│
├── cmd/
│   └── zylisp-repl/               # Phase 4
│       └── main.go
│
└── zast/                           # Already exists
    └── ...
```

### Common Commands

```bash
# Run all tests
cd zylisp/lang
go test ./...

# Run specific package tests
go test ./parser -v
go test ./compiler -v
go test . -v

# Run single test
go test -v -run TestLexer_IntLiteral

# Build REPL
cd zylisp/cmd/zylisp-repl
go build

# Run REPL
./zylisp-repl

# Format code
go fmt ./...

# Clean test cache
go clean -testcache
```

### Test Case Template

**For each new feature**:

1. **Test file**: `testdata/phaseN/feature_name.zl`

```scheme
   ; Your Zylisp code
   (+ 1 2)
```

2. **Expected output**: `testdata/phaseN/expected/feature_name.txt`

```
   3
```

3. **Add to integration test**: In `integration_test.go`

```go
   {"feature_name", "feature_name.zl"},
```

### S-Expression to Go AST Quick Reference

**Common patterns you'll generate**:

```
Integer literal:
(BasicLit :valuepos 0 :kind INT :value "42")

Binary operation:
(BinaryExpr :x <left> :oppos 0 :op ADD :y <right>)

Variable declaration:
(DeclStmt :decl (GenDecl :tok VAR :specs ((ValueSpec :names ((Ident :name "x")) :values (<expr>)))))

Function declaration:
(FuncDecl
  :name (Ident :name "funcName")
  :type (FuncType :params (FieldList :list (...)) :results (FieldList :list (...)))
  :body (BlockStmt :list (...)))

Function call:
(CallExpr :fun (Ident :name "funcName") :args (...))

If statement:
(IfStmt :cond <expr> :body (BlockStmt ...) :else (BlockStmt ...))

Return statement:
(ReturnStmt :results (<expr>))
```

### Go Token Names

**Operators**:

- `+` → `ADD`
- `-` → `SUB`
- `*` → `MUL`
- `/` → `QUO`
- `%` → `REM`
- `<` → `LSS`
- `<=` → `LEQ`
- `>` → `GTR`
- `>=` → `GEQ`
- `==` → `EQL`
- `!=` → `NEQ`

**Keywords**:

- `VAR` for variable declarations
- `CONST` for constant declarations
- `TYPE` for type declarations
- `FUNC` for functions
- `IMPORT` for imports
- `PACKAGE` for package declarations

### Troubleshooting Checklist

**If tests fail**:

- [ ] Check parentheses are balanced in s-expressions
- [ ] Verify all field names match Go AST spec
- [ ] Ensure `:namepos`, `:valuepos` etc. are included
- [ ] Check that token names are uppercase (ADD not add)
- [ ] Verify `go.mod` has correct replace directives
- [ ] Run `go mod tidy`
- [ ] Clear test cache: `go clean -testcache`

**If compilation fails**:

- [ ] Check generated Go code: save to file and inspect
- [ ] Verify Go code is syntactically valid: `go fmt`
- [ ] Look at similar examples in go-ast-coverage
- [ ] Test zast directly with your s-expression

**If REPL fails**:

- [ ] Check that simple expressions work: `42`, `(+ 1 2)`
- [ ] Verify error messages are helpful
- [ ] Ensure `CanInterpret` correctly identifies simple expressions
- [ ] Check that complex forms report "not yet implemented"

---

## Next Steps Beyond Bootstrap

After completing these 4 phases, you'll have a solid foundation. Here are logical next steps:

### Phase 5: REPL Compilation (Tier 2/3)

- Implement compilation in REPL
- Add function definition caching (Tier 2)
- Implement JIT compilation (Tier 3)
- Test that compiled functions are cached and reused

### Phase 6: Worker Supervision

- Implement worker process isolation
- Add memory monitoring with gopsutil
- Implement worker restart logic
- Test plugin memory leak solution

### Phase 7: More Data Types

- Strings
- Booleans
- Lists/vectors
- Maps
- Floats

### Phase 8: More Core Forms

- Multiple expressions in function bodies (`begin`/`do`)
- More operators (comparison, logical)
- Loop constructs
- Pattern matching basics

### Phase 9: Standard Library

- I/O functions
- String manipulation
- List operations
- Math functions

### Phase 10: Advanced Features

- Modules and imports
- Error handling (Result types)
- Goroutines and channels
- Interface definitions

---

## Final Thoughts

**Remember**:

1. **Start small** - Get `42` working before attempting factorial
2. **Test continuously** - Every change should be tested immediately
3. **One thing at a time** - Don't add multiple features simultaneously
4. **Follow the plan** - Each phase builds on the previous
5. **Don't skip phases** - The order matters for learning and stability

**The bootstrap is designed to**:

- Prove the architecture works
- Build your understanding incrementally
- Provide working examples for reference
- Create a solid foundation for growth

**Success looks like**:

- Being able to compile and run simple Zylisp programs
- Understanding how each compilation stage works
- Having confidence to add new features
- A test suite that catches regressions

Good luck! You're building a real compiler, and this plan will get you there step by step.

**Happy Hacking!** 🚀

## Appendix: Quick Navigation by Role

**For Claude Code (AI Assistant)**:

- Start at [Phase 1: Context for Claude Code](#context-for-claude-code)
- Follow tasks sequentially: Task 1 → Task 2 → Task 3 → etc.
- Each task has complete implementation code
- Phase 2-4 have high-level guides (detailed specs on request)

**For Human Developers**:

- Start at [Testing and Iteration Guide](#testing-and-iteration-guide-for-humans)
- Reference [Quick Reference](#quick-reference) for common patterns
- Use [Troubleshooting Checklist](#troubleshooting-checklist) when stuck
- Follow [Development Workflow](#development-workflow) for new features

**For Project Planning**:

- Read [Summary and Intent](#summary-and-intent)
- Review [Phase Timeline](#phase-timeline)
- Check [Next Steps Beyond Bootstrap](#next-steps-beyond-bootstrap)
