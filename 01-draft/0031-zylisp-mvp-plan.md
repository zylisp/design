---
number: 0031
title: "Zylisp MVP Development Plan"
author: Unknown
created: 2025-10-05
updated: 2025-10-05
state: Draft
supersedes: None
superseded-by: None
---

# Zylisp MVP Development Plan

**Version**: 1.0.0  
**Date**: October 5, 2025  
**Target**: Working REPL for language experimentation

---

## Table of Contents

1. [Overview](#1-overview)
2. [Deep Dive: What We're Building](#2-deep-dive-what-were-building)
3. [Phase-by-Phase Implementation](#3-phase-by-phase-implementation)
4. [Testing Strategy](#4-testing-strategy)
5. [Success Criteria](#5-success-criteria)

---

## 1. Overview

### 1.1 Near-Term Goal

**Get to a working REPL as fast as possible** so we can start experimenting with Zylisp language features.

We want to be able to:

```zylisp
> (+ 1 2)
3

> (define square (lambda (n) (* n n)))
<function>

> (square 5)
25

> (define x 10)
10

> (+ x (* 2 3))
16
```

### 1.2 What We're Building (In Scope)

**Three repositories with minimal implementations:**

1. **zylisp/lang** - Basic parsing and interpretation
   - Lexer: Tokenize Zylisp source
   - Reader: Parse tokens → S-expressions
   - Interpreter: Evaluate S-expressions directly (Tier 1 only)
   - Basic types: Numbers, Symbols, Lists, Strings, Booleans

2. **zylisp/repl** - Simple in-process REPL
   - Server: In-process evaluation (no networking yet)
   - Client: Direct function calls (no sockets)
   - Basic environment management

3. **zylisp/cli** - Minimal command-line wrapper
   - Simple REPL loop
   - Read input → Eval → Print → Loop

**Core Language Features (MVP):**
- Integer arithmetic: `+`, `-`, `*`, `/`
- Variable definition: `define`
- Lambda functions: `lambda`
- Function application
- Basic control flow: `if`
- Lists and list operations

### 1.3 What We're Saving for Later (Out of Scope)

**Not in this document:**

- ❌ Macro expansion (Stage 1 compiler Phase 2)
- ❌ Stage 2 compiler (Core forms → Go code)
- ❌ Compilation and caching (Tiers 2 & 3)
- ❌ Type system and annotations
- ❌ zast implementation
- ❌ Network-based REPL (nREPL protocol)
- ❌ Worker process supervision
- ❌ Memory management
- ❌ rely library integration
- ❌ Multiple clients
- ❌ Session management
- ❌ Standard library beyond primitives

**Why save these for later?**
- We need a working REPL to experiment with language design
- These features require the basic interpreter to work first
- Each can be added incrementally once we have the foundation

### 1.4 Timeline Estimate

- **Phase 1**: 4-6 hours (zylisp/lang foundation)
- **Phase 2**: 2-3 hours (zylisp/repl in-process server)
- **Phase 3**: 1-2 hours (zylisp/cli wrapper)
- **Phase 4**: 2-3 hours (Testing and polish)

**Total**: 1-2 days of focused development

---

## 2. Deep Dive: What We're Building

### 2.1 zylisp/lang Package Structure

```
zylisp/lang/
├── go.mod
├── README.md
├── sexpr/
│   ├── types.go          # Core S-expression types
│   ├── types_test.go     # Type tests
│   ├── print.go          # String representation
│   └── print_test.go     # Print tests
├── parser/
│   ├── lexer.go          # Tokenization
│   ├── lexer_test.go     # Lexer tests
│   ├── reader.go         # Parse tokens → S-expressions
│   └── reader_test.go    # Reader tests
└── interpreter/
    ├── env.go            # Environment management
    ├── env_test.go       # Environment tests
    ├── eval.go           # Evaluation logic
    ├── eval_test.go      # Eval tests
    ├── primitives.go     # Built-in functions
    └── primitives_test.go # Primitive tests
```

### 2.2 Core Data Types

**S-Expression Types (sexpr/types.go):**

```go
// Base interface
type SExpr interface {
    String() string
}

// Atomic types
type Number struct {
    Value int64
}

type Symbol struct {
    Name string
}

type String struct {
    Value string
}

type Bool struct {
    Value bool
}

type Nil struct {}

// Composite types
type List struct {
    Elements []SExpr
}

// Functions
type Func struct {
    Params []Symbol
    Body   SExpr
    Env    *Env
}

type Primitive struct {
    Name string
    Fn   func([]SExpr, *Env) (SExpr, error)
}
```

### 2.3 Lexer Design

**Token Types:**

```go
type TokenType int

const (
    LPAREN TokenType = iota  // (
    RPAREN                    // )
    NUMBER                    // 123, -456
    SYMBOL                    // +, define, lambda
    STRING                    // "hello"
    BOOL                      // true, false
    EOF
)

type Token struct {
    Type  TokenType
    Value string
    Line  int
    Col   int
}
```

**What it handles:**

- Whitespace: space, tab, newline (skip)
- Comments: `;` to end of line (skip)
- Numbers: integers only for MVP (can add floats later)
- Symbols: letters, digits, and special chars (`+`, `-`, `*`, etc.)
- Strings: `"double quoted"`
- Booleans: `true`, `false`
- Parens: `(` and `)`

### 2.4 Reader Design

**Parses tokens into S-expressions:**

```
Tokens: [LPAREN, SYMBOL(+), NUMBER(1), NUMBER(2), RPAREN]
   ↓
SExpr:  List([Symbol("+"), Number(1), Number(2)])
```

**Handles:**
- Lists: `(+ 1 2)`
- Nested lists: `(+ (* 2 3) 4)`
- Atoms: `42`, `hello`, `"string"`
- Empty lists: `()`

### 2.5 Environment Design

**Simple map-based environment with parent chain:**

```go
type Env struct {
    bindings map[string]SExpr
    parent   *Env
}

func (e *Env) Define(name string, value SExpr)
func (e *Env) Lookup(name string) (SExpr, error)
func (e *Env) Extend() *Env  // Create child environment
```

**Used for:**
- Global definitions: `(define x 42)`
- Function parameters: `(lambda (a b) (+ a b))`
- Lexical scoping

### 2.6 Evaluator Design

**Core evaluation logic:**

```go
func Eval(expr SExpr, env *Env) (SExpr, error) {
    switch e := expr.(type) {
    
    case Number, String, Bool, Nil:
        return e, nil  // Self-evaluating
    
    case Symbol:
        return env.Lookup(e.Name)
    
    case List:
        if len(e.Elements) == 0 {
            return Nil{}, nil
        }
        
        // Special forms
        first := e.Elements[0]
        if sym, ok := first.(Symbol); ok {
            switch sym.Name {
            case "define":
                return evalDefine(e, env)
            case "lambda":
                return evalLambda(e, env)
            case "if":
                return evalIf(e, env)
            case "quote":
                return e.Elements[1], nil
            }
        }
        
        // Function application
        return evalApply(e, env)
    }
}
```

**Special forms to implement:**

- `define`: `(define name value)` - Bind value to name
- `lambda`: `(lambda (params...) body)` - Create function
- `if`: `(if test then else)` - Conditional
- `quote`: `(quote expr)` - Return expr unevaluated

**Primitives to implement:**

- Arithmetic: `+`, `-`, `*`, `/`
- Comparison: `=`, `<`, `>`, `<=`, `>=`
- List operations: `list`, `car`, `cdr`, `cons`
- Type predicates: `number?`, `symbol?`, `list?`, `null?`

### 2.7 REPL Server Design

**In-process server (no networking yet):**

```go
type Server struct {
    env *interpreter.Env
}

func NewServer() *Server {
    env := interpreter.NewEnv(nil)
    interpreter.LoadPrimitives(env)  // Add +, -, *, etc.
    return &Server{env: env}
}

func (s *Server) Eval(source string) (string, error) {
    // Tokenize
    tokens, err := lexer.Tokenize(source)
    if err != nil {
        return "", err
    }
    
    // Parse
    expr, err := reader.Read(tokens)
    if err != nil {
        return "", err
    }
    
    // Evaluate
    result, err := interpreter.Eval(expr, s.env)
    if err != nil {
        return "", err
    }
    
    return result.String(), nil
}
```

### 2.8 CLI Design

**Simple REPL loop:**

```go
func main() {
    server := server.NewServer()
    scanner := bufio.NewScanner(os.Stdin)
    
    fmt.Println("Zylisp REPL v0.0.1")
    fmt.Println("Type expressions and press Enter")
    fmt.Println()
    
    for {
        fmt.Print("> ")
        
        if !scanner.Scan() {
            break
        }
        
        line := scanner.Text()
        if line == "" {
            continue
        }
        
        if line == "exit" || line == "quit" {
            break
        }
        
        result, err := server.Eval(line)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
        } else {
            fmt.Println(result)
        }
    }
    
    fmt.Println("\nGoodbye!")
}
```

---

## 3. Phase-by-Phase Implementation

### Phase 1: zylisp/lang Foundation

#### Phase 1.1: Repository Setup and Core Types

**Duration**: 30 minutes

**Create repository structure:**

```bash
mkdir -p zylisp/lang
cd zylisp/lang
go mod init github.com/yourusername/zylisp/lang
```

**Create directory structure:**

```bash
mkdir -p sexpr parser interpreter
```

**File: `README.md`**

```markdown
# zylisp/lang

Core language implementation for Zylisp.

## Packages

- `sexpr`: S-expression types and utilities
- `parser`: Lexer and reader for parsing Zylisp source
- `interpreter`: Direct evaluation of S-expressions

## Status

MVP implementation - supports basic arithmetic, variables, and lambda functions.
```

**File: `sexpr/types.go`**

```go
package sexpr

import "fmt"

// SExpr is the base interface for all S-expression types
type SExpr interface {
    String() string
}

// Number represents an integer
type Number struct {
    Value int64
}

func (n Number) String() string {
    return fmt.Sprintf("%d", n.Value)
}

// Symbol represents a name/identifier
type Symbol struct {
    Name string
}

func (s Symbol) String() string {
    return s.Name
}

// String represents a string literal
type String struct {
    Value string
}

func (s String) String() string {
    return fmt.Sprintf("%q", s.Value)
}

// Bool represents a boolean value
type Bool struct {
    Value bool
}

func (b Bool) String() string {
    if b.Value {
        return "true"
    }
    return "false"
}

// Nil represents the empty value
type Nil struct{}

func (n Nil) String() string {
    return "nil"
}

// List represents a sequence of expressions
type List struct {
    Elements []SExpr
}

func (l List) String() string {
    if len(l.Elements) == 0 {
        return "()"
    }
    
    result := "("
    for i, elem := range l.Elements {
        if i > 0 {
            result += " "
        }
        result += elem.String()
    }
    result += ")"
    return result
}

// Func represents a user-defined function
type Func struct {
    Params []Symbol
    Body   SExpr
    Env    *Env  // Will define in interpreter package
}

func (f Func) String() string {
    return "<function>"
}

// Primitive represents a built-in function
type Primitive struct {
    Name string
    Fn   func([]SExpr, *Env) (SExpr, error)
}

func (p Primitive) String() string {
    return fmt.Sprintf("<primitive:%s>", p.Name)
}

// Env is forward-declared here but implemented in interpreter
type Env interface {
    Define(name string, value SExpr)
    Lookup(name string) (SExpr, error)
}
```

**File: `sexpr/types_test.go`**

```go
package sexpr

import "testing"

func TestNumberString(t *testing.T) {
    tests := []struct {
        value    int64
        expected string
    }{
        {42, "42"},
        {-17, "-17"},
        {0, "0"},
    }
    
    for _, tt := range tests {
        n := Number{Value: tt.value}
        if got := n.String(); got != tt.expected {
            t.Errorf("Number(%d).String() = %q, want %q", 
                tt.value, got, tt.expected)
        }
    }
}

func TestSymbolString(t *testing.T) {
    tests := []struct {
        name     string
        expected string
    }{
        {"x", "x"},
        {"+", "+"},
        {"lambda", "lambda"},
    }
    
    for _, tt := range tests {
        s := Symbol{Name: tt.name}
        if got := s.String(); got != tt.expected {
            t.Errorf("Symbol(%q).String() = %q, want %q",
                tt.name, got, tt.expected)
        }
    }
}

func TestBoolString(t *testing.T) {
    tests := []struct {
        value    bool
        expected string
    }{
        {true, "true"},
        {false, "false"},
    }
    
    for _, tt := range tests {
        b := Bool{Value: tt.value}
        if got := b.String(); got != tt.expected {
            t.Errorf("Bool(%v).String() = %q, want %q",
                tt.value, got, tt.expected)
        }
    }
}

func TestListString(t *testing.T) {
    tests := []struct {
        name     string
        list     List
        expected string
    }{
        {
            "empty list",
            List{Elements: []SExpr{}},
            "()",
        },
        {
            "single element",
            List{Elements: []SExpr{Number{Value: 42}}},
            "(42)",
        },
        {
            "multiple elements",
            List{Elements: []SExpr{
                Symbol{Name: "+"},
                Number{Value: 1},
                Number{Value: 2},
            }},
            "(+ 1 2)",
        },
        {
            "nested list",
            List{Elements: []SExpr{
                Symbol{Name: "+"},
                List{Elements: []SExpr{
                    Symbol{Name: "*"},
                    Number{Value: 2},
                    Number{Value: 3},
                }},
                Number{Value: 4},
            }},
            "(+ (* 2 3) 4)",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := tt.list.String(); got != tt.expected {
                t.Errorf("List.String() = %q, want %q", got, tt.expected)
            }
        })
    }
}
```

**Run tests:**

```bash
cd sexpr
go test -v
cd ..
```

#### Phase 1.2: Lexer Implementation

**Duration**: 45 minutes

**File: `parser/lexer.go`**

```go
package parser

import (
    "fmt"
    "strings"
    "unicode"
)

// TokenType represents the type of a token
type TokenType int

const (
    LPAREN TokenType = iota
    RPAREN
    NUMBER
    SYMBOL
    STRING
    BOOL
    EOF
    ILLEGAL
)

func (tt TokenType) String() string {
    switch tt {
    case LPAREN:
        return "LPAREN"
    case RPAREN:
        return "RPAREN"
    case NUMBER:
        return "NUMBER"
    case SYMBOL:
        return "SYMBOL"
    case STRING:
        return "STRING"
    case BOOL:
        return "BOOL"
    case EOF:
        return "EOF"
    case ILLEGAL:
        return "ILLEGAL"
    default:
        return "UNKNOWN"
    }
}

// Token represents a lexical token
type Token struct {
    Type  TokenType
    Value string
    Line  int
    Col   int
}

func (t Token) String() string {
    return fmt.Sprintf("%s(%q)", t.Type, t.Value)
}

// Lexer tokenizes Zylisp source code
type Lexer struct {
    input   string
    pos     int  // current position
    line    int  // current line
    col     int  // current column
    tokens  []Token
}

// NewLexer creates a new lexer for the given input
func NewLexer(input string) *Lexer {
    return &Lexer{
        input: input,
        pos:   0,
        line:  1,
        col:   1,
    }
}

// Tokenize returns all tokens from the input
func Tokenize(input string) ([]Token, error) {
    lexer := NewLexer(input)
    return lexer.Tokenize()
}

// Tokenize produces all tokens
func (l *Lexer) Tokenize() ([]Token, error) {
    for {
        tok := l.nextToken()
        l.tokens = append(l.tokens, tok)
        
        if tok.Type == EOF {
            break
        }
        
        if tok.Type == ILLEGAL {
            return nil, fmt.Errorf("illegal token at line %d, col %d: %q",
                tok.Line, tok.Col, tok.Value)
        }
    }
    
    return l.tokens, nil
}

// nextToken returns the next token
func (l *Lexer) nextToken() Token {
    l.skipWhitespaceAndComments()
    
    if l.isAtEnd() {
        return l.makeToken(EOF, "")
    }
    
    ch := l.peek()
    
    switch ch {
    case '(':
        return l.makeSingleCharToken(LPAREN)
    case ')':
        return l.makeSingleCharToken(RPAREN)
    case '"':
        return l.scanString()
    }
    
    if isDigit(ch) || (ch == '-' && l.peekNext() != 0 && isDigit(l.peekNext())) {
        return l.scanNumber()
    }
    
    if isSymbolStart(ch) {
        return l.scanSymbol()
    }
    
    return l.makeToken(ILLEGAL, string(ch))
}

// skipWhitespaceAndComments skips whitespace and comments
func (l *Lexer) skipWhitespaceAndComments() {
    for !l.isAtEnd() {
        ch := l.peek()
        
        if ch == ';' {
            // Skip comment to end of line
            for !l.isAtEnd() && l.peek() != '\n' {
                l.advance()
            }
            continue
        }
        
        if isWhitespace(ch) {
            l.advance()
            continue
        }
        
        break
    }
}

// scanNumber scans a number token
func (l *Lexer) scanNumber() Token {
    start := l.pos
    startCol := l.col
    
    if l.peek() == '-' {
        l.advance()
    }
    
    for !l.isAtEnd() && isDigit(l.peek()) {
        l.advance()
    }
    
    value := l.input[start:l.pos]
    return Token{Type: NUMBER, Value: value, Line: l.line, Col: startCol}
}

// scanSymbol scans a symbol token
func (l *Lexer) scanSymbol() Token {
    start := l.pos
    startCol := l.col
    
    for !l.isAtEnd() && isSymbolChar(l.peek()) {
        l.advance()
    }
    
    value := l.input[start:l.pos]
    
    // Check for boolean literals
    if value == "true" || value == "false" {
        return Token{Type: BOOL, Value: value, Line: l.line, Col: startCol}
    }
    
    return Token{Type: SYMBOL, Value: value, Line: l.line, Col: startCol}
}

// scanString scans a string token
func (l *Lexer) scanString() Token {
    startCol := l.col
    l.advance() // consume opening quote
    
    var value strings.Builder
    
    for !l.isAtEnd() && l.peek() != '"' {
        ch := l.peek()
        
        if ch == '\\' {
            l.advance()
            if l.isAtEnd() {
                return l.makeToken(ILLEGAL, "unterminated string")
            }
            
            // Handle escape sequences
            escaped := l.peek()
            switch escaped {
            case 'n':
                value.WriteByte('\n')
            case 't':
                value.WriteByte('\t')
            case 'r':
                value.WriteByte('\r')
            case '"':
                value.WriteByte('"')
            case '\\':
                value.WriteByte('\\')
            default:
                value.WriteByte(escaped)
            }
            l.advance()
        } else {
            value.WriteByte(ch)
            l.advance()
        }
    }
    
    if l.isAtEnd() {
        return l.makeToken(ILLEGAL, "unterminated string")
    }
    
    l.advance() // consume closing quote
    
    return Token{Type: STRING, Value: value.String(), Line: l.line, Col: startCol}
}

// Helper functions

func (l *Lexer) peek() byte {
    if l.isAtEnd() {
        return 0
    }
    return l.input[l.pos]
}

func (l *Lexer) peekNext() byte {
    if l.pos+1 >= len(l.input) {
        return 0
    }
    return l.input[l.pos+1]
}

func (l *Lexer) advance() byte {
    if l.isAtEnd() {
        return 0
    }
    
    ch := l.input[l.pos]
    l.pos++
    
    if ch == '\n' {
        l.line++
        l.col = 1
    } else {
        l.col++
    }
    
    return ch
}

func (l *Lexer) isAtEnd() bool {
    return l.pos >= len(l.input)
}

func (l *Lexer) makeToken(typ TokenType, value string) Token {
    return Token{Type: typ, Value: value, Line: l.line, Col: l.col}
}

func (l *Lexer) makeSingleCharToken(typ TokenType) Token {
    ch := l.peek()
    l.advance()
    return Token{Type: typ, Value: string(ch), Line: l.line, Col: l.col - 1}
}

// Character classification

func isWhitespace(ch byte) bool {
    return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isDigit(ch byte) bool {
    return ch >= '0' && ch <= '9'
}

func isSymbolStart(ch byte) bool {
    return unicode.IsLetter(rune(ch)) || isSymbolSpecial(ch)
}

func isSymbolChar(ch byte) bool {
    return unicode.IsLetter(rune(ch)) || isDigit(ch) || isSymbolSpecial(ch)
}

func isSymbolSpecial(ch byte) bool {
    return strings.ContainsRune("+-*/<>=!?&|%$_", rune(ch))
}
```

**File: `parser/lexer_test.go`**

```go
package parser

import (
    "reflect"
    "testing"
)

func TestLexerSimple(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected []TokenType
    }{
        {
            "empty",
            "",
            []TokenType{EOF},
        },
        {
            "single number",
            "42",
            []TokenType{NUMBER, EOF},
        },
        {
            "single symbol",
            "hello",
            []TokenType{SYMBOL, EOF},
        },
        {
            "empty list",
            "()",
            []TokenType{LPAREN, RPAREN, EOF},
        },
        {
            "simple list",
            "(+ 1 2)",
            []TokenType{LPAREN, SYMBOL, NUMBER, NUMBER, RPAREN, EOF},
        },
        {
            "nested list",
            "(+ (* 2 3) 4)",
            []TokenType{LPAREN, SYMBOL, LPAREN, SYMBOL, NUMBER, NUMBER, 
                       RPAREN, NUMBER, RPAREN, EOF},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tokens, err := Tokenize(tt.input)
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            
            if len(tokens) != len(tt.expected) {
                t.Fatalf("got %d tokens, want %d", len(tokens), len(tt.expected))
            }
            
            for i, tok := range tokens {
                if tok.Type != tt.expected[i] {
                    t.Errorf("token %d: got %v, want %v", 
                        i, tok.Type, tt.expected[i])
                }
            }
        })
    }
}

func TestLexerTokenValues(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected []Token
    }{
        {
            "numbers",
            "42 -17 0",
            []Token{
                {Type: NUMBER, Value: "42"},
                {Type: NUMBER, Value: "-17"},
                {Type: NUMBER, Value: "0"},
                {Type: EOF, Value: ""},
            },
        },
        {
            "symbols",
            "+ hello-world foo?",
            []Token{
                {Type: SYMBOL, Value: "+"},
                {Type: SYMBOL, Value: "hello-world"},
                {Type: SYMBOL, Value: "foo?"},
                {Type: EOF, Value: ""},
            },
        },
        {
            "strings",
            `"hello" "world"`,
            []Token{
                {Type: STRING, Value: "hello"},
                {Type: STRING, Value: "world"},
                {Type: EOF, Value: ""},
            },
        },
        {
            "booleans",
            "true false",
            []Token{
                {Type: BOOL, Value: "true"},
                {Type: BOOL, Value: "false"},
                {Type: EOF, Value: ""},
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tokens, err := Tokenize(tt.input)
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            
            if len(tokens) != len(tt.expected) {
                t.Fatalf("got %d tokens, want %d", len(tokens), len(tt.expected))
            }
            
            for i, tok := range tokens {
                if tok.Type != tt.expected[i].Type {
                    t.Errorf("token %d type: got %v, want %v",
                        i, tok.Type, tt.expected[i].Type)
                }
                if tok.Value != tt.expected[i].Value {
                    t.Errorf("token %d value: got %q, want %q",
                        i, tok.Value, tt.expected[i].Value)
                }
            }
        })
    }
}

func TestLexerComments(t *testing.T) {
    input := `
; This is a comment
(+ 1 2) ; inline comment
; another comment
42
`
    expected := []TokenType{LPAREN, SYMBOL, NUMBER, NUMBER, RPAREN, NUMBER, EOF}
    
    tokens, err := Tokenize(input)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    var types []TokenType
    for _, tok := range tokens {
        types = append(types, tok.Type)
    }
    
    if !reflect.DeepEqual(types, expected) {
        t.Errorf("got %v, want %v", types, expected)
    }
}

func TestLexerStringEscapes(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {`"hello\nworld"`, "hello\nworld"},
        {`"tab\there"`, "tab\there"},
        {`"quote\"here"`, `quote"here`},
        {`"backslash\\here"`, `backslash\here`},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            tokens, err := Tokenize(tt.input)
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            
            if len(tokens) != 2 { // STRING + EOF
                t.Fatalf("got %d tokens, want 2", len(tokens))
            }
            
            if tokens[0].Value != tt.expected {
                t.Errorf("got %q, want %q", tokens[0].Value, tt.expected)
            }
        })
    }
}
```

**Run tests:**

```bash
cd parser
go test -v
cd ..
```

#### Phase 1.3: Reader Implementation

**Duration**: 45 minutes

**File: `parser/reader.go`**

```go
package parser

import (
    "fmt"
    "strconv"
    
    "github.com/yourusername/zylisp/lang/sexpr"
)

// Reader parses tokens into S-expressions
type Reader struct {
    tokens []Token
    pos    int
}

// NewReader creates a new reader for the given tokens
func NewReader(tokens []Token) *Reader {
    return &Reader{tokens: tokens, pos: 0}
}

// Read parses tokens into an S-expression
func Read(tokens []Token) (sexpr.SExpr, error) {
    reader := NewReader(tokens)
    return reader.readExpr()
}

// readExpr reads a single expression
func (r *Reader) readExpr() (sexpr.SExpr, error) {
    if r.isAtEnd() {
        return nil, fmt.Errorf("unexpected end of input")
    }
    
    tok := r.peek()
    
    switch tok.Type {
    case LPAREN:
        return r.readList()
    case NUMBER:
        return r.readNumber()
    case SYMBOL:
        return r.readSymbol()
    case STRING:
        return r.readString()
    case BOOL:
        return r.readBool()
    case RPAREN:
        return nil, fmt.Errorf("unexpected closing paren at line %d, col %d",
            tok.Line, tok.Col)
    case EOF:
        return nil, fmt.Errorf("unexpected end of file")
    default:
        return nil, fmt.Errorf("unexpected token %v at line %d, col %d",
            tok.Type, tok.Line, tok.Col)
    }
}

// readList reads a list expression
func (r *Reader) readList() (sexpr.SExpr, error) {
    r.advance() // consume LPAREN
    
    var elements []sexpr.SExpr
    
    for !r.isAtEnd() && r.peek().Type != RPAREN {
        expr, err := r.readExpr()
        if err != nil {
            return nil, err
        }
        elements = append(elements, expr)
    }
    
    if r.isAtEnd() {
        return nil, fmt.Errorf("unclosed list")
    }
    
    r.advance() // consume RPAREN
    
    return sexpr.List{Elements: elements}, nil
}

// readNumber reads a number expression
func (r *Reader) readNumber() (sexpr.SExpr, error) {
    tok := r.advance()
    
    value, err := strconv.ParseInt(tok.Value, 10, 64)
    if err != nil {
        return nil, fmt.Errorf("invalid number %q at line %d, col %d: %v",
            tok.Value, tok.Line, tok.Col, err)
    }
    
    return sexpr.Number{Value: value}, nil
}

// readSymbol reads a symbol expression
func (r *Reader) readSymbol() (sexpr.SExpr, error) {
    tok := r.advance()
    return sexpr.Symbol{Name: tok.Value}, nil
}

// readString reads a string expression
func (r *Reader) readString() (sexpr.SExpr, error) {
    tok := r.advance()
    return sexpr.String{Value: tok.Value}, nil
}

// readBool reads a boolean expression
func (r *Reader) readBool() (sexpr.SExpr, error) {
    tok := r.advance()
    value := tok.Value == "true"
    return sexpr.Bool{Value: value}, nil
}

// Helper functions

func (r *Reader) peek() Token {
    if r.isAtEnd() {
        return Token{Type: EOF}
    }
    return r.tokens[r.pos]
}

func (r *Reader) advance() Token {
    if r.isAtEnd() {
        return Token{Type: EOF}
    }
    tok := r.tokens[r.pos]
    r.pos++
    return tok
}

func (r *Reader) isAtEnd() bool {
    return r.pos >= len(r.tokens) || r.tokens[r.pos].Type == EOF
}
```

**File: `parser/reader_test.go`**

```go
package parser

import (
    "reflect"
    "testing"
    
    "github.com/yourusername/zylisp/lang/sexpr"
)

func TestReaderNumbers(t *testing.T) {
    tests := []struct {
        input    string
        expected sexpr.SExpr
    }{
        {"42", sexpr.Number{Value: 42}},
        {"-17", sexpr.Number{Value: -17}},
        {"0", sexpr.Number{Value: 0}},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            tokens, err := Tokenize(tt.input)
            if err != nil {
                t.Fatalf("tokenize error: %v", err)
            }
            
            result, err := Read(tokens)
            if err != nil {
                t.Fatalf("read error: %v", err)
            }
            
            if !reflect.DeepEqual(result, tt.expected) {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}

func TestReaderSymbols(t *testing.T) {
    tests := []struct {
        input    string
        expected sexpr.SExpr
    }{
        {"x", sexpr.Symbol{Name: "x"}},
        {"+", sexpr.Symbol{Name: "+"}},
        {"hello-world", sexpr.Symbol{Name: "hello-world"}},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            tokens, err := Tokenize(tt.input)
            if err != nil {
                t.Fatalf("tokenize error: %v", err)
            }
            
            result, err := Read(tokens)
            if err != nil {
                t.Fatalf("read error: %v", err)
            }
            
            if !reflect.DeepEqual(result, tt.expected) {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}

func TestReaderLists(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected sexpr.SExpr
    }{
        {
            "empty list",
            "()",
            sexpr.List{Elements: []sexpr.SExpr{}},
        },
        {
            "single element",
            "(42)",
            sexpr.List{Elements: []sexpr.SExpr{
                sexpr.Number{Value: 42},
            }},
        },
        {
            "simple list",
            "(+ 1 2)",
            sexpr.List{Elements: []sexpr.SExpr{
                sexpr.Symbol{Name: "+"},
                sexpr.Number{Value: 1},
                sexpr.Number{Value: 2},
            }},
        },
        {
            "nested list",
            "(+ (* 2 3) 4)",
            sexpr.List{Elements: []sexpr.SExpr{
                sexpr.Symbol{Name: "+"},
                sexpr.List{Elements: []sexpr.SExpr{
                    sexpr.Symbol{Name: "*"},
                    sexpr.Number{Value: 2},
                    sexpr.Number{Value: 3},
                }},
                sexpr.Number{Value: 4},
            }},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tokens, err := Tokenize(tt.input)
            if err != nil {
                t.Fatalf("tokenize error: %v", err)
            }
            
            result, err := Read(tokens)
            if err != nil {
                t.Fatalf("read error: %v", err)
            }
            
            if !reflect.DeepEqual(result, tt.expected) {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}

func TestReaderBooleans(t *testing.T) {
    tests := []struct {
        input    string
        expected sexpr.SExpr
    }{
        {"true", sexpr.Bool{Value: true}},
        {"false", sexpr.Bool{Value: false}},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            tokens, err := Tokenize(tt.input)
            if err != nil {
                t.Fatalf("tokenize error: %v", err)
            }
            
            result, err := Read(tokens)
            if err != nil {
                t.Fatalf("read error: %v", err)
            }
            
            if !reflect.DeepEqual(result, tt.expected) {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}

func TestReaderStrings(t *testing.T) {
    tests := []struct {
        input    string
        expected sexpr.SExpr
    }{
        {`"hello"`, sexpr.String{Value: "hello"}},
        {`"hello world"`, sexpr.String{Value: "hello world"}},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            tokens, err := Tokenize(tt.input)
            if err != nil {
                t.Fatalf("tokenize error: %v", err)
            }
            
            result, err := Read(tokens)
            if err != nil {
                t.Fatalf("read error: %v", err)
            }
            
            if !reflect.DeepEqual(result, tt.expected) {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}

func TestReaderErrors(t *testing.T) {
    tests := []struct {
        name  string
        input string
    }{
        {"unclosed list", "(+ 1 2"},
        {"extra closing paren", "(+ 1 2))"},
        {"just closing paren", ")"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tokens, err := Tokenize(tt.input)
            if err != nil {
                // Lexer error is fine for some test cases
                return
            }
            
            _, err = Read(tokens)
            if err == nil {
                t.Errorf("expected error, got nil")
            }
        })
    }
}
```

**Run tests:**

```bash
cd parser
go test -v
cd ..
```

#### Phase 1.4: Environment Implementation

**Duration**: 30 minutes

**File: `interpreter/env.go`**

```go
package interpreter

import (
    "fmt"
    
    "github.com/yourusername/zylisp/lang/sexpr"
)

// Env represents a lexical environment for variable bindings
type Env struct {
    bindings map[string]sexpr.SExpr
    parent   *Env
}

// NewEnv creates a new environment with an optional parent
func NewEnv(parent *Env) *Env {
    return &Env{
        bindings: make(map[string]sexpr.SExpr),
        parent:   parent,
    }
}

// Define binds a value to a name in this environment
func (e *Env) Define(name string, value sexpr.SExpr) {
    e.bindings[name] = value
}

// Set updates an existing binding, searching parent environments
func (e *Env) Set(name string, value sexpr.SExpr) error {
    if _, ok := e.bindings[name]; ok {
        e.bindings[name] = value
        return nil
    }
    
    if e.parent != nil {
        return e.parent.Set(name, value)
    }
    
    return fmt.Errorf("undefined variable: %s", name)
}

// Lookup finds a value by name, searching parent environments
func (e *Env) Lookup(name string) (sexpr.SExpr, error) {
    if value, ok := e.bindings[name]; ok {
        return value, nil
    }
    
    if e.parent != nil {
        return e.parent.Lookup(name)
    }
    
    return nil, fmt.Errorf("undefined variable: %s", name)
}

// Extend creates a child environment
func (e *Env) Extend() *Env {
    return NewEnv(e)
}
```

**File: `interpreter/env_test.go`**

```go
package interpreter

import (
    "testing"
    
    "github.com/yourusername/zylisp/lang/sexpr"
)

func TestEnvDefineAndLookup(t *testing.T) {
    env := NewEnv(nil)
    
    // Define a variable
    env.Define("x", sexpr.Number{Value: 42})
    
    // Look it up
    value, err := env.Lookup("x")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    num, ok := value.(sexpr.Number)
    if !ok {
        t.Fatalf("expected Number, got %T", value)
    }
    
    if num.Value != 42 {
        t.Errorf("got %d, want 42", num.Value)
    }
}

func TestEnvUndefinedVariable(t *testing.T) {
    env := NewEnv(nil)
    
    _, err := env.Lookup("undefined")
    if err == nil {
        t.Error("expected error for undefined variable")
    }
}

func TestEnvParentLookup(t *testing.T) {
    parent := NewEnv(nil)
    parent.Define("x", sexpr.Number{Value: 42})
    
    child := NewEnv(parent)
    child.Define("y", sexpr.Number{Value: 17})
    
    // Child can see its own bindings
    value, err := child.Lookup("y")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if value.(sexpr.Number).Value != 17 {
        t.Errorf("got %v, want 17", value)
    }
    
    // Child can see parent bindings
    value, err = child.Lookup("x")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if value.(sexpr.Number).Value != 42 {
        t.Errorf("got %v, want 42", value)
    }
    
    // Parent cannot see child bindings
    _, err = parent.Lookup("y")
    if err == nil {
        t.Error("parent should not see child bindings")
    }
}

func TestEnvShadowing(t *testing.T) {
    parent := NewEnv(nil)
    parent.Define("x", sexpr.Number{Value: 42})
    
    child := NewEnv(parent)
    child.Define("x", sexpr.Number{Value: 17})
    
    // Child sees its own binding
    value, err := child.Lookup("x")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if value.(sexpr.Number).Value != 17 {
        t.Errorf("got %v, want 17", value)
    }
    
    // Parent still has original binding
    value, err = parent.Lookup("x")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if value.(sexpr.Number).Value != 42 {
        t.Errorf("got %v, want 42", value)
    }
}

func TestEnvExtend(t *testing.T) {
    parent := NewEnv(nil)
    parent.Define("x", sexpr.Number{Value: 42})
    
    child := parent.Extend()
    
    // Child can see parent binding
    value, err := child.Lookup("x")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if value.(sexpr.Number).Value != 42 {
        t.Errorf("got %v, want 42", value)
    }
}
```

**Run tests:**

```bash
cd interpreter
go test -v env_test.go env.go
cd ..
```

#### Phase 1.5: Core Evaluator

**Duration**: 1 hour

**First, update `sexpr/types.go` to fix circular dependency:**

```go
// Remove the Env interface and update Func and Primitive

// Func represents a user-defined function
type Func struct {
    Params []Symbol
    Body   SExpr
    Env    interface{} // Use interface{} to avoid circular import
}

// Primitive represents a built-in function
type Primitive struct {
    Name string
    Fn   func([]SExpr, interface{}) (SExpr, error)
}
```

**File: `interpreter/eval.go`**

```go
package interpreter

import (
    "fmt"
    
    "github.com/yourusername/zylisp/lang/sexpr"
)

// Eval evaluates an S-expression in an environment
func Eval(expr sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    switch e := expr.(type) {
    
    // Self-evaluating types
    case sexpr.Number:
        return e, nil
    case sexpr.String:
        return e, nil
    case sexpr.Bool:
        return e, nil
    case sexpr.Nil:
        return e, nil
    
    // Symbol lookup
    case sexpr.Symbol:
        return env.Lookup(e.Name)
    
    // List evaluation
    case sexpr.List:
        return evalList(e, env)
    
    default:
        return nil, fmt.Errorf("cannot evaluate: %v", expr)
    }
}

// evalList evaluates a list expression
func evalList(list sexpr.List, env *Env) (sexpr.SExpr, error) {
    if len(list.Elements) == 0 {
        return sexpr.Nil{}, nil
    }
    
    first := list.Elements[0]
    
    // Check for special forms
    if sym, ok := first.(sexpr.Symbol); ok {
        switch sym.Name {
        case "define":
            return evalDefine(list, env)
        case "lambda":
            return evalLambda(list, env)
        case "if":
            return evalIf(list, env)
        case "quote":
            return evalQuote(list, env)
        }
    }
    
    // Function application
    return evalApply(list, env)
}

// evalDefine handles (define name value)
func evalDefine(list sexpr.List, env *Env) (sexpr.SExpr, error) {
    if len(list.Elements) != 3 {
        return nil, fmt.Errorf("define requires 2 arguments, got %d",
            len(list.Elements)-1)
    }
    
    name, ok := list.Elements[1].(sexpr.Symbol)
    if !ok {
        return nil, fmt.Errorf("define: first argument must be a symbol")
    }
    
    value, err := Eval(list.Elements[2], env)
    if err != nil {
        return nil, err
    }
    
    env.Define(name.Name, value)
    return value, nil
}

// evalLambda handles (lambda (params...) body)
func evalLambda(list sexpr.List, env *Env) (sexpr.SExpr, error) {
    if len(list.Elements) != 3 {
        return nil, fmt.Errorf("lambda requires 2 arguments, got %d",
            len(list.Elements)-1)
    }
    
    paramsList, ok := list.Elements[1].(sexpr.List)
    if !ok {
        return nil, fmt.Errorf("lambda: parameters must be a list")
    }
    
    var params []sexpr.Symbol
    for _, p := range paramsList.Elements {
        sym, ok := p.(sexpr.Symbol)
        if !ok {
            return nil, fmt.Errorf("lambda: parameter must be a symbol, got %v", p)
        }
        params = append(params, sym)
    }
    
    body := list.Elements[2]
    
    return sexpr.Func{
        Params: params,
        Body:   body,
        Env:    env,
    }, nil
}

// evalIf handles (if test then else)
func evalIf(list sexpr.List, env *Env) (sexpr.SExpr, error) {
    if len(list.Elements) != 4 {
        return nil, fmt.Errorf("if requires 3 arguments, got %d",
            len(list.Elements)-1)
    }
    
    test, err := Eval(list.Elements[1], env)
    if err != nil {
        return nil, err
    }
    
    if isTruthy(test) {
        return Eval(list.Elements[2], env)
    }
    return Eval(list.Elements[3], env)
}

// evalQuote handles (quote expr)
func evalQuote(list sexpr.List, env *Env) (sexpr.SExpr, error) {
    if len(list.Elements) != 2 {
        return nil, fmt.Errorf("quote requires 1 argument, got %d",
            len(list.Elements)-1)
    }
    
    return list.Elements[1], nil
}

// evalApply handles function application
func evalApply(list sexpr.List, env *Env) (sexpr.SExpr, error) {
    // Evaluate the function
    fn, err := Eval(list.Elements[0], env)
    if err != nil {
        return nil, err
    }
    
    // Evaluate arguments
    var args []sexpr.SExpr
    for _, arg := range list.Elements[1:] {
        value, err := Eval(arg, env)
        if err != nil {
            return nil, err
        }
        args = append(args, value)
    }
    
    // Apply function
    switch f := fn.(type) {
    case sexpr.Primitive:
        return f.Fn(args, env)
    
    case sexpr.Func:
        return applyFunc(f, args)
    
    default:
        return nil, fmt.Errorf("not a function: %v", fn)
    }
}

// applyFunc applies a user-defined function
func applyFunc(fn sexpr.Func, args []sexpr.SExpr) (sexpr.SExpr, error) {
    if len(args) != len(fn.Params) {
        return nil, fmt.Errorf("function expects %d arguments, got %d",
            len(fn.Params), len(args))
    }
    
    // Create new environment extending the function's closure
    funcEnv := fn.Env.(*Env).Extend()
    
    // Bind parameters to arguments
    for i, param := range fn.Params {
        funcEnv.Define(param.Name, args[i])
    }
    
    // Evaluate body in new environment
    return Eval(fn.Body, funcEnv)
}

// isTruthy determines if a value is truthy
func isTruthy(value sexpr.SExpr) bool {
    switch v := value.(type) {
    case sexpr.Bool:
        return v.Value
    case sexpr.Nil:
        return false
    default:
        return true
    }
}
```

**File: `interpreter/eval_test.go`**

```go
package interpreter

import (
    "testing"
    
    "github.com/yourusername/zylisp/lang/parser"
    "github.com/yourusername/zylisp/lang/sexpr"
)

func testEval(t *testing.T, input string, expected sexpr.SExpr) {
    t.Helper()
    
    tokens, err := parser.Tokenize(input)
    if err != nil {
        t.Fatalf("tokenize error: %v", err)
    }
    
    expr, err := parser.Read(tokens)
    if err != nil {
        t.Fatalf("read error: %v", err)
    }
    
    env := NewEnv(nil)
    result, err := Eval(expr, env)
    if err != nil {
        t.Fatalf("eval error: %v", err)
    }
    
    if result.String() != expected.String() {
        t.Errorf("got %v, want %v", result, expected)
    }
}

func TestEvalSelfEvaluating(t *testing.T) {
    tests := []struct {
        input    string
        expected sexpr.SExpr
    }{
        {"42", sexpr.Number{Value: 42}},
        {`"hello"`, sexpr.String{Value: "hello"}},
        {"true", sexpr.Bool{Value: true}},
        {"false", sexpr.Bool{Value: false}},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            testEval(t, tt.input, tt.expected)
        })
    }
}

func TestEvalDefine(t *testing.T) {
    tokens, _ := parser.Tokenize("(define x 42)")
    expr, _ := parser.Read(tokens)
    
    env := NewEnv(nil)
    result, err := Eval(expr, env)
    if err != nil {
        t.Fatalf("eval error: %v", err)
    }
    
    // Should return the value
    if result.(sexpr.Number).Value != 42 {
        t.Errorf("got %v, want 42", result)
    }
    
    // Should be defined in environment
    value, err := env.Lookup("x")
    if err != nil {
        t.Fatalf("lookup error: %v", err)
    }
    if value.(sexpr.Number).Value != 42 {
        t.Errorf("got %v, want 42", value)
    }
}

func TestEvalSymbolLookup(t *testing.T) {
    env := NewEnv(nil)
    env.Define("x", sexpr.Number{Value: 42})
    
    tokens, _ := parser.Tokenize("x")
    expr, _ := parser.Read(tokens)
    
    result, err := Eval(expr, env)
    if err != nil {
        t.Fatalf("eval error: %v", err)
    }
    
    if result.(sexpr.Number).Value != 42 {
        t.Errorf("got %v, want 42", result)
    }
}

func TestEvalLambda(t *testing.T) {
    tokens, _ := parser.Tokenize("(lambda (x) x)")
    expr, _ := parser.Read(tokens)
    
    env := NewEnv(nil)
    result, err := Eval(expr, env)
    if err != nil {
        t.Fatalf("eval error: %v", err)
    }
    
    _, ok := result.(sexpr.Func)
    if !ok {
        t.Errorf("expected Func, got %T", result)
    }
}

func TestEvalIf(t *testing.T) {
    tests := []struct {
        input    string
        expected int64
    }{
        {"(if true 1 2)", 1},
        {"(if false 1 2)", 2},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            tokens, _ := parser.Tokenize(tt.input)
            expr, _ := parser.Read(tokens)
            
            env := NewEnv(nil)
            result, err := Eval(expr, env)
            if err != nil {
                t.Fatalf("eval error: %v", err)
            }
            
            if result.(sexpr.Number).Value != tt.expected {
                t.Errorf("got %v, want %d", result, tt.expected)
            }
        })
    }
}

func TestEvalQuote(t *testing.T) {
    tokens, _ := parser.Tokenize("(quote (+ 1 2))")
    expr, _ := parser.Read(tokens)
    
    env := NewEnv(nil)
    result, err := Eval(expr, env)
    if err != nil {
        t.Fatalf("eval error: %v", err)
    }
    
    list, ok := result.(sexpr.List)
    if !ok {
        t.Fatalf("expected List, got %T", result)
    }
    
    if len(list.Elements) != 3 {
        t.Errorf("got %d elements, want 3", len(list.Elements))
    }
}
```

**Run tests:**

```bash
cd interpreter
go test -v eval_test.go eval.go env.go
cd ..
```

#### Phase 1.6: Primitive Functions

**Duration**: 45 minutes

**File: `interpreter/primitives.go`**

```go
package interpreter

import (
    "fmt"
    
    "github.com/yourusername/zylisp/lang/sexpr"
)

// LoadPrimitives adds all primitive functions to an environment
func LoadPrimitives(env *Env) {
    // Arithmetic
    env.Define("+", makePrimitive("+", primAdd))
    env.Define("-", makePrimitive("-", primSub))
    env.Define("*", makePrimitive("*", primMul))
    env.Define("/", makePrimitive("/", primDiv))
    
    // Comparison
    env.Define("=", makePrimitive("=", primEq))
    env.Define("<", makePrimitive("<", primLt))
    env.Define(">", makePrimitive(">", primGt))
    env.Define("<=", makePrimitive("<=", primLte))
    env.Define(">=", makePrimitive(">=", primGte))
    
    // List operations
    env.Define("list", makePrimitive("list", primList))
    env.Define("car", makePrimitive("car", primCar))
    env.Define("cdr", makePrimitive("cdr", primCdr))
    env.Define("cons", makePrimitive("cons", primCons))
    
    // Type predicates
    env.Define("number?", makePrimitive("number?", primIsNumber))
    env.Define("symbol?", makePrimitive("symbol?", primIsSymbol))
    env.Define("list?", makePrimitive("list?", primIsList))
    env.Define("null?", makePrimitive("null?", primIsNull))
}

func makePrimitive(name string, fn func([]sexpr.SExpr, *Env) (sexpr.SExpr, error)) sexpr.Primitive {
    return sexpr.Primitive{
        Name: name,
        Fn: func(args []sexpr.SExpr, envInterface interface{}) (sexpr.SExpr, error) {
            env := envInterface.(*Env)
            return fn(args, env)
        },
    }
}

// Arithmetic primitives

func primAdd(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    if len(args) == 0 {
        return sexpr.Number{Value: 0}, nil
    }
    
    var sum int64
    for _, arg := range args {
        num, ok := arg.(sexpr.Number)
        if !ok {
            return nil, fmt.Errorf("+: expected number, got %v", arg)
        }
        sum += num.Value
    }
    
    return sexpr.Number{Value: sum}, nil
}

func primSub(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    if len(args) == 0 {
        return nil, fmt.Errorf("-: requires at least 1 argument")
    }
    
    first, ok := args[0].(sexpr.Number)
    if !ok {
        return nil, fmt.Errorf("-: expected number, got %v", args[0])
    }
    
    if len(args) == 1 {
        return sexpr.Number{Value: -first.Value}, nil
    }
    
    result := first.Value
    for _, arg := range args[1:] {
        num, ok := arg.(sexpr.Number)
        if !ok {
            return nil, fmt.Errorf("-: expected number, got %v", arg)
        }
        result -= num.Value
    }
    
    return sexpr.Number{Value: result}, nil
}

func primMul(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    if len(args) == 0 {
        return sexpr.Number{Value: 1}, nil
    }
    
    product := int64(1)
    for _, arg := range args {
        num, ok := arg.(sexpr.Number)
        if !ok {
            return nil, fmt.Errorf("*: expected number, got %v", arg)
        }
        product *= num.Value
    }
    
    return sexpr.Number{Value: product}, nil
}

func primDiv(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    if len(args) == 0 {
        return nil, fmt.Errorf("/: requires at least 1 argument")
    }
    
    first, ok := args[0].(sexpr.Number)
    if !ok {
        return nil, fmt.Errorf("/: expected number, got %v", args[0])
    }
    
    if len(args) == 1 {
        if first.Value == 0 {
            return nil, fmt.Errorf("/: division by zero")
        }
        return sexpr.Number{Value: 1 / first.Value}, nil
    }
    
    result := first.Value
    for _, arg := range args[1:] {
        num, ok := arg.(sexpr.Number)
        if !ok {
            return nil, fmt.Errorf("/: expected number, got %v", arg)
        }
        if num.Value == 0 {
            return nil, fmt.Errorf("/: division by zero")
        }
        result /= num.Value
    }
    
    return sexpr.Number{Value: result}, nil
}

// Comparison primitives

func primEq(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    if len(args) != 2 {
        return nil, fmt.Errorf("=: requires 2 arguments, got %d", len(args))
    }
    
    a, ok1 := args[0].(sexpr.Number)
    b, ok2 := args[1].(sexpr.Number)
    
    if !ok1 || !ok2 {
        return nil, fmt.Errorf("=: expected numbers")
    }
    
    return sexpr.Bool{Value: a.Value == b.Value}, nil
}

func primLt(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    if len(args) != 2 {
        return nil, fmt.Errorf("<: requires 2 arguments, got %d", len(args))
    }
    
    a, ok1 := args[0].(sexpr.Number)
    b, ok2 := args[1].(sexpr.Number)
    
    if !ok1 || !ok2 {
        return nil, fmt.Errorf("<: expected numbers")
    }
    
    return sexpr.Bool{Value: a.Value < b.Value}, nil
}

func primGt(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    if len(args) != 2 {
        return nil, fmt.Errorf(">: requires 2 arguments, got %d", len(args))
    }
    
    a, ok1 := args[0].(sexpr.Number)
    b, ok2 := args[1].(sexpr.Number)
    
    if !ok1 || !ok2 {
        return nil, fmt.Errorf(">: expected numbers")
    }
    
    return sexpr.Bool{Value: a.Value > b.Value}, nil
}

func primLte(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    if len(args) != 2 {
        return nil, fmt.Errorf("<=: requires 2 arguments, got %d", len(args))
    }
    
    a, ok1 := args[0].(sexpr.Number)
    b, ok2 := args[1].(sexpr.Number)
    
    if !ok1 || !ok2 {
        return nil, fmt.Errorf("<=: expected numbers")
    }
    
    return sexpr.Bool{Value: a.Value <= b.Value}, nil
}

func primGte(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    if len(args) != 2 {
        return nil, fmt.Errorf(">=: requires 2 arguments, got %d", len(args))
    }
    
    a, ok1 := args[0].(sexpr.Number)
    b, ok2 := args[1].(sexpr.Number)
    
    if !ok1 || !ok2 {
        return nil, fmt.Errorf(">=: expected numbers")
    }
    
    return sexpr.Bool{Value: a.Value >= b.Value}, nil
}

// List primitives

func primList(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    return sexpr.List{Elements: args}, nil
}

func primCar(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    if len(args) != 1 {
        return nil, fmt.Errorf("car: requires 1 argument, got %d", len(args))
    }
    
    list, ok := args[0].(sexpr.List)
    if !ok {
        return nil, fmt.Errorf("car: expected list, got %v", args[0])
    }
    
    if len(list.Elements) == 0 {
        return nil, fmt.Errorf("car: cannot take car of empty list")
    }
    
    return list.Elements[0], nil
}

func primCdr(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    if len(args) != 1 {
        return nil, fmt.Errorf("cdr: requires 1 argument, got %d", len(args))
    }
    
    list, ok := args[0].(sexpr.List)
    if !ok {
        return nil, fmt.Errorf("cdr: expected list, got %v", args[0])
    }
    
    if len(list.Elements) == 0 {
        return nil, fmt.Errorf("cdr: cannot take cdr of empty list")
    }
    
    return sexpr.List{Elements: list.Elements[1:]}, nil
}

func primCons(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    if len(args) != 2 {
        return nil, fmt.Errorf("cons: requires 2 arguments, got %d", len(args))
    }
    
    list, ok := args[1].(sexpr.List)
    if !ok {
        return nil, fmt.Errorf("cons: second argument must be a list, got %v", args[1])
    }
    
    elements := make([]sexpr.SExpr, 0, len(list.Elements)+1)
    elements = append(elements, args[0])
    elements = append(elements, list.Elements...)
    
    return sexpr.List{Elements: elements}, nil
}

// Type predicates

func primIsNumber(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    if len(args) != 1 {
        return nil, fmt.Errorf("number?: requires 1 argument, got %d", len(args))
    }
    
    _, ok := args[0].(sexpr.Number)
    return sexpr.Bool{Value: ok}, nil
}

func primIsSymbol(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    if len(args) != 1 {
        return nil, fmt.Errorf("symbol?: requires 1 argument, got %d", len(args))
    }
    
    _, ok := args[0].(sexpr.Symbol)
    return sexpr.Bool{Value: ok}, nil
}

func primIsList(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    if len(args) != 1 {
        return nil, fmt.Errorf("list?: requires 1 argument, got %d", len(args))
    }
    
    _, ok := args[0].(sexpr.List)
    return sexpr.Bool{Value: ok}, nil
}

func primIsNull(args []sexpr.SExpr, env *Env) (sexpr.SExpr, error) {
    if len(args) != 1 {
        return nil, fmt.Errorf("null?: requires 1 argument, got %d", len(args))
    }
    
    list, ok := args[0].(sexpr.List)
    if !ok {
        return sexpr.Bool{Value: false}, nil
    }
    
    return sexpr.Bool{Value: len(list.Elements) == 0}, nil
}
```

**File: `interpreter/primitives_test.go`**

```go
package interpreter

import (
    "testing"
    
    "github.com/yourusername/zylisp/lang/parser"
    "github.com/yourusername/zylisp/lang/sexpr"
)

func testEvalWithPrimitives(t *testing.T, input string, expected sexpr.SExpr) {
    t.Helper()
    
    tokens, err := parser.Tokenize(input)
    if err != nil {
        t.Fatalf("tokenize error: %v", err)
    }
    
    expr, err := parser.Read(tokens)
    if err != nil {
        t.Fatalf("read error: %v", err)
    }
    
    env := NewEnv(nil)
    LoadPrimitives(env)
    
    result, err := Eval(expr, env)
    if err != nil {
        t.Fatalf("eval error: %v", err)
    }
    
    if result.String() != expected.String() {
        t.Errorf("got %v, want %v", result, expected)
    }
}

func TestPrimAdd(t *testing.T) {
    tests := []struct {
        input    string
        expected int64
    }{
        {"(+ 1 2)", 3},
        {"(+ 1 2 3 4)", 10},
        {"(+)", 0},
        {"(+ 42)", 42},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            testEvalWithPrimitives(t, tt.input, sexpr.Number{Value: tt.expected})
        })
    }
}

func TestPrimSub(t *testing.T) {
    tests := []struct {
        input    string
        expected int64
    }{
        {"(- 5 3)", 2},
        {"(- 10 3 2)", 5},
        {"(- 42)", -42},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            testEvalWithPrimitives(t, tt.input, sexpr.Number{Value: tt.expected})
        })
    }
}

func TestPrimMul(t *testing.T) {
    tests := []struct {
        input    string
        expected int64
    }{
        {"(* 2 3)", 6},
        {"(* 2 3 4)", 24},
        {"(*)", 1},
        {"(* 42)", 42},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            testEvalWithPrimitives(t, tt.input, sexpr.Number{Value: tt.expected})
        })
    }
}

func TestPrimDiv(t *testing.T) {
    tests := []struct {
        input    string
        expected int64
    }{
        {"(/ 6 2)", 3},
        {"(/ 24 3 2)", 4},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            testEvalWithPrimitives(t, tt.input, sexpr.Number{Value: tt.expected})
        })
    }
}

func TestPrimComparisons(t *testing.T) {
    tests := []struct {
        input    string
        expected bool
    }{
        {"(= 1 1)", true},
        {"(= 1 2)", false},
        {"(< 1 2)", true},
        {"(< 2 1)", false},
        {"(> 2 1)", true},
        {"(> 1 2)", false},
        {"(<= 1 1)", true},
        {"(<= 1 2)", true},
        {"(<= 2 1)", false},
        {"(>= 1 1)", true},
        {"(>= 2 1)", true},
        {"(>= 1 2)", false},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            testEvalWithPrimitives(t, tt.input, sexpr.Bool{Value: tt.expected})
        })
    }
}

func TestPrimList(t *testing.T) {
    input := "(list 1 2 3)"
    expected := sexpr.List{
        Elements: []sexpr.SExpr{
            sexpr.Number{Value: 1},
            sexpr.Number{Value: 2},
            sexpr.Number{Value: 3},
        },
    }
    
    testEvalWithPrimitives(t, input, expected)
}

func TestPrimCar(t *testing.T) {
    input := "(car (list 1 2 3))"
    expected := sexpr.Number{Value: 1}
    
    testEvalWithPrimitives(t, input, expected)
}

func TestPrimCdr(t *testing.T) {
    input := "(cdr (list 1 2 3))"
    expected := sexpr.List{
        Elements: []sexpr.SExpr{
            sexpr.Number{Value: 2},
            sexpr.Number{Value: 3},
        },
    }
    
    testEvalWithPrimitives(t, input, expected)
}

func TestPrimCons(t *testing.T) {
    input := "(cons 0 (list 1 2 3))"
    expected := sexpr.List{
        Elements: []sexpr.SExpr{
            sexpr.Number{Value: 0},
            sexpr.Number{Value: 1},
            sexpr.Number{Value: 2},
            sexpr.Number{Value: 3},
        },
    }
    
    testEvalWithPrimitives(t, input, expected)
}

func TestPrimTypePredicates(t *testing.T) {
    tests := []struct {
        input    string
        expected bool
    }{
        {"(number? 42)", true},
        {"(number? (quote x))", false},
        {"(symbol? (quote x))", true},
        {"(symbol? 42)", false},
        {"(list? (list 1 2))", true},
        {"(list? 42)", false},
        {"(null? (list))", true},
        {"(null? (list 1))", false},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            testEvalWithPrimitives(t, tt.input, sexpr.Bool{Value: tt.expected})
        })
    }
}

func TestNestedExpressions(t *testing.T) {
    tests := []struct {
        input    string
        expected int64
    }{
        {"(+ (* 2 3) 4)", 10},
        {"(* (+ 1 2) (- 5 2))", 9},
        {"(/ (+ 10 6) (- 6 2))", 4},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            testEvalWithPrimitives(t, tt.input, sexpr.Number{Value: tt.expected})
        })
    }
}

func TestUserDefinedFunctions(t *testing.T) {
    input := `
        (define square (lambda (x) (* x x)))
        (square 5)
    `
    
    tokens, _ := parser.Tokenize(input)
    
    env := NewEnv(nil)
    LoadPrimitives(env)
    
    // Read and eval first expression (define)
    expr1, _ := parser.Read(tokens)
    _, err := Eval(expr1, env)
    if err != nil {
        t.Fatalf("eval define error: %v", err)
    }
    
    // Read and eval second expression (square 5)
    reader := parser.NewReader(tokens)
    reader.Tokenize() // Skip already-read tokens
    expr2, _ := reader.readExpr()
    result, err := Eval(expr2, env)
    if err != nil {
        t.Fatalf("eval call error: %v", err)
    }
    
    expected := sexpr.Number{Value: 25}
    if result.String() != expected.String() {
        t.Errorf("got %v, want %v", result, expected)
    }
}
```

**Run tests:**

```bash
cd interpreter
go test -v
cd ..
```

---

### Phase 2: zylisp/repl In-Process Server

#### Phase 2.1: Repository Setup

**Duration**: 15 minutes

**Create repository structure:**

```bash
cd ..
mkdir -p zylisp/repl
cd zylisp/repl
go mod init github.com/yourusername/zylisp/repl
go get github.com/yourusername/zylisp/lang
```

**Create directory structure:**

```bash
mkdir -p server client
```

**File: `README.md`**

```markdown
# zylisp/repl

REPL server and client for Zylisp.

## Status

MVP implementation - in-process server only (no networking yet).

## Packages

- `server`: In-process REPL server
- `client`: Direct client library

## Future

- Network-based nREPL protocol
- Multiple client support
- Session management
```

#### Phase 2.2: REPL Server Implementation

**Duration**: 30 minutes

**File: `server/server.go`**

```go
package server

import (
    "fmt"
    
    "github.com/yourusername/zylisp/lang/interpreter"
    "github.com/yourusername/zylisp/lang/parser"
)

// Server represents a REPL server
type Server struct {
    env *interpreter.Env
}

// NewServer creates a new REPL server
func NewServer() *Server {
    env := interpreter.NewEnv(nil)
    interpreter.LoadPrimitives(env)
    
    return &Server{env: env}
}

// Eval evaluates a Zylisp expression and returns the result as a string
func (s *Server) Eval(source string) (string, error) {
    // Tokenize
    tokens, err := parser.Tokenize(source)
    if err != nil {
        return "", fmt.Errorf("tokenize error: %w", err)
    }
    
    // Parse
    expr, err := parser.Read(tokens)
    if err != nil {
        return "", fmt.Errorf("parse error: %w", err)
    }
    
    // Evaluate
    result, err := interpreter.Eval(expr, s.env)
    if err != nil {
        return "", fmt.Errorf("eval error: %w", err)
    }
    
    return result.String(), nil
}

// Reset clears the environment and reloads primitives
func (s *Server) Reset() {
    s.env = interpreter.NewEnv(nil)
    interpreter.LoadPrimitives(s.env)
}
```

**File: `server/server_test.go`**

```go
package server

import (
    "testing"
)

func TestServerBasicEval(t *testing.T) {
    server := NewServer()
    
    tests := []struct {
        input    string
        expected string
    }{
        {"42", "42"},
        {"(+ 1 2)", "3"},
        {"(* 2 3)", "6"},
        {`"hello"`, `"hello"`},
        {"true", "true"},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            result, err := server.Eval(tt.input)
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            
            if result != tt.expected {
                t.Errorf("got %q, want %q", result, tt.expected)
            }
        })
    }
}

func TestServerDefine(t *testing.T) {
    server := NewServer()
    
    // Define a variable
    _, err := server.Eval("(define x 42)")
    if err != nil {
        t.Fatalf("define error: %v", err)
    }
    
    // Use the variable
    result, err := server.Eval("x")
    if err != nil {
        t.Fatalf("lookup error: %v", err)
    }
    
    if result != "42" {
        t.Errorf("got %q, want \"42\"", result)
    }
}

func TestServerLambda(t *testing.T) {
    server := NewServer()
    
    // Define a function
    _, err := server.Eval("(define square (lambda (x) (* x x)))")
    if err != nil {
        t.Fatalf("define error: %v", err)
    }
    
    // Call the function
    result, err := server.Eval("(square 5)")
    if err != nil {
        t.Fatalf("call error: %v", err)
    }
    
    if result != "25" {
        t.Errorf("got %q, want \"25\"", result)
    }
}

func TestServerReset(t *testing.T) {
    server := NewServer()
    
    // Define a variable
    server.Eval("(define x 42)")
    
    // Reset the server
    server.Reset()
    
    // Variable should be undefined now
    _, err := server.Eval("x")
    if err == nil {
        t.Error("expected error after reset, got nil")
    }
}

func TestServerErrors(t *testing.T) {
    server := NewServer()
    
    tests := []string{
        "(+",           // unclosed paren
        "(+ 1 x)",      // undefined variable
        "(1 2 3)",      // not a function
        "(/ 1 0)",      // division by zero
    }
    
    for _, input := range tests {
        t.Run(input, func(t *testing.T) {
            _, err := server.Eval(input)
            if err == nil {
                t.Errorf("expected error for %q, got nil", input)
            }
        })
    }
}
```

**Run tests:**

```bash
cd server
go test -v
cd ..
```

#### Phase 2.3: REPL Client Implementation

**Duration**: 15 minutes

**File: `client/client.go`**

```go
package client

import (
    "github.com/yourusername/zylisp/repl/server"
)

// Client represents a REPL client
type Client struct {
    server *server.Server
}

// NewClient creates a new REPL client
func NewClient(srv *server.Server) *Client {
    return &Client{server: srv}
}

// Send sends an expression to the server and returns the result
func (c *Client) Send(expr string) (string, error) {
    return c.server.Eval(expr)
}

// Reset resets the server environment
func (c *Client) Reset() {
    c.server.Reset()
}
```

**File: `client/client_test.go`**

```go
package client

import (
    "testing"
    
    "github.com/yourusername/zylisp/repl/server"
)

func TestClientSend(t *testing.T) {
    srv := server.NewServer()
    client := NewClient(srv)
    
    result, err := client.Send("(+ 1 2)")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    if result != "3" {
        t.Errorf("got %q, want \"3\"", result)
    }
}

func TestClientReset(t *testing.T) {
    srv := server.NewServer()
    client := NewClient(srv)
    
    // Define a variable
    client.Send("(define x 42)")
    
    // Reset
    client.Reset()
    
    // Variable should be undefined
    _, err := client.Send("x")
    if err == nil {
        t.Error("expected error after reset, got nil")
    }
}
```

**Run tests:**

```bash
cd client
go test -v
cd ..
```

---

### Phase 3: zylisp/cli Command-Line Tool

#### Phase 3.1: Repository Setup and CLI Implementation

**Duration**: 45 minutes

**Create repository structure:**

```bash
cd ..
mkdir -p zylisp/cli
cd zylisp/cli
go mod init github.com/yourusername/zylisp/cli
go get github.com/yourusername/zylisp/repl
```

**File: `README.md`**

```markdown
# zylisp/cli

Command-line tools for Zylisp.

## Installation

```bash
go install github.com/yourusername/zylisp/cli@latest
```

## Usage

Start the REPL:

```bash
zylisp
```

## Status

MVP implementation - basic REPL only.
```

**File: `main.go`**

```go
package main

import (
    "bufio"
    "fmt"
    "os"
    "strings"
    
    "github.com/yourusername/zylisp/repl/client"
    "github.com/yourusername/zylisp/repl/server"
)

const banner = `
╔═══════════════════════════════════════╗
║                                       ║
║           Zylisp REPL v0.0.1         ║
║                                       ║
║  A Lisp that compiles to Go          ║
║                                       ║
╚═══════════════════════════════════════╝

Type expressions and press Enter.
Type 'exit' or 'quit' to leave.
Type ':reset' to clear the environment.
Type ':help' for more commands.

`

func main() {
    fmt.Print(banner)
    
    // Create server and client
    srv := server.NewServer()
    cli := client.NewClient(srv)
    
    // Create scanner for input
    scanner := bufio.NewScanner(os.Stdin)
    
    // REPL loop
    for {
        fmt.Print("> ")
        
        // Read input
        if !scanner.Scan() {
            break
        }
        
        line := strings.TrimSpace(scanner.Text())
        
        // Skip empty lines
        if line == "" {
            continue
        }
        
        // Handle special commands
        if handleCommand(line, cli) {
            continue
        }
        
        // Evaluate expression
        result, err := cli.Send(line)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
        } else {
            fmt.Println(result)
        }
    }
    
    // Check for scanner errors
    if err := scanner.Err(); err != nil {
        fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Println("\nGoodbye!")
}

// handleCommand handles special REPL commands
// Returns true if the command was handled, false otherwise
func handleCommand(line string, cli *client.Client) bool {
    switch line {
    case "exit", "quit":
        fmt.Println("\nGoodbye!")
        os.Exit(0)
        return true
    
    case ":reset":
        cli.Reset()
        fmt.Println("Environment reset")
        return true
    
    case ":help":
        showHelp()
        return true
    
    default:
        return false
    }
}

func showHelp() {
    fmt.Println(`
Available Commands:
  exit, quit    - Exit the REPL
  :reset        - Reset the environment
  :help         - Show this help message

Special Forms:
  define        - Define a variable: (define x 42)
  lambda        - Create a function: (lambda (x) (* x x))
  if            - Conditional: (if test then else)
  quote         - Quote an expression: (quote (1 2 3))

Primitives:
  Arithmetic:   +, -, *, /
  Comparison:   =, <, >, <=, >=
  Lists:        list, car, cdr, cons
  Predicates:   number?, symbol?, list?, null?

Examples:
  > (+ 1 2)
  3
  
  > (define square (lambda (x) (* x x)))
  <function>
  
  > (square 5)
  25
  
  > (if (> 5 3) "yes" "no")
  "yes"
`)
}
```

**Build and test:**

```bash
go build -o zylisp
./zylisp
```

**Try it out:**

```zylisp
> (+ 1 2)
3

> (define x 10)
10

> (* x x)
100

> (define square (lambda (n) (* n n)))
<function>

> (square 7)
49

> (if (> 5 3) "yes" "no")
"yes"

> (list 1 2 3 4 5)
(1 2 3 4 5)

> (car (list 1 2 3))
1

> (cdr (list 1 2 3))
(2 3)

> exit
```

---

### Phase 4: Testing and Polish

#### Phase 4.1: Integration Tests

**Duration**: 1 hour

**File: `zylisp/cli/integration_test.go`**

```go
package main

import (
    "testing"
    
    "github.com/yourusername/zylisp/repl/client"
    "github.com/yourusername/zylisp/repl/server"
)

func TestIntegrationBasic(t *testing.T) {
    srv := server.NewServer()
    cli := client.NewClient(srv)
    
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"number", "42", "42"},
        {"add", "(+ 1 2)", "3"},
        {"nested", "(+ (* 2 3) 4)", "10"},
        {"string", `"hello"`, `"hello"`},
        {"boolean", "true", "true"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := cli.Send(tt.input)
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            
            if result != tt.expected {
                t.Errorf("got %q, want %q", result, tt.expected)
            }
        })
    }
}

func TestIntegrationStateful(t *testing.T) {
    srv := server.NewServer()
    cli := client.NewClient(srv)
    
    steps := []struct {
        input    string
        expected string
    }{
        {"(define x 10)", "10"},
        {"x", "10"},
        {"(define y 20)", "20"},
        {"(+ x y)", "30"},
        {"(define add (lambda (a b) (+ a b)))", "<function>"},
        {"(add x y)", "30"},
        {"(add 5 7)", "12"},
    }
    
    for i, step := range steps {
        result, err := cli.Send(step.input)
        if err != nil {
            t.Fatalf("step %d error: %v", i, err)
        }
        
        if result != step.expected {
            t.Errorf("step %d: got %q, want %q", i, result, step.expected)
        }
    }
}

func TestIntegrationFactorial(t *testing.T) {
    srv := server.NewServer()
    cli := client.NewClient(srv)
    
    // Define factorial function
    factorialDef := `
        (define factorial
          (lambda (n)
            (if (<= n 1)
                1
                (* n (factorial (- n 1))))))
    `
    
    _, err := cli.Send(factorialDef)
    if err != nil {
        t.Fatalf("define factorial error: %v", err)
    }
    
    tests := []struct {
        n        int
        expected string
    }{
        {0, "1"},
        {1, "1"},
        {5, "120"},
        {6, "720"},
    }
    
    for _, tt := range tests {
        input := fmt.Sprintf("(factorial %d)", tt.n)
        result, err := cli.Send(input)
        if err != nil {
            t.Fatalf("factorial(%d) error: %v", tt.n, err)
        }
        
        if result != tt.expected {
            t.Errorf("factorial(%d): got %q, want %q", tt.n, result, tt.expected)
        }
    }
}

func TestIntegrationListProcessing(t *testing.T) {
    srv := server.NewServer()
    cli := client.NewClient(srv)
    
    // Define sum function
    sumDef := `
        (define sum
          (lambda (lst)
            (if (null? lst)
                0
                (+ (car lst) (sum (cdr lst))))))
    `
    
    _, err := cli.Send(sumDef)
    if err != nil {
        t.Fatalf("define sum error: %v", err)
    }
    
    result, err := cli.Send("(sum (list 1 2 3 4 5))")
    if err != nil {
        t.Fatalf("sum error: %v", err)
    }
    
    if result != "15" {
        t.Errorf("sum: got %q, want \"15\"", result)
    }
}
```

**Run integration tests:**

```bash
go test -v
```

#### Phase 4.2: Documentation and Examples

**Duration**: 30 minutes

**File: `zylisp/cli/EXAMPLES.md`**

```markdown
# Zylisp REPL Examples

## Basic Arithmetic

```zylisp
> (+ 1 2 3)
6

> (- 10 3)