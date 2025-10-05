---
number: 0005
title: S-Expression Parser Implementation Specification
author: Duncan McGreggor
created: 2025-10-01
updated: 2025-10-04
state: Final
supersedes: None
superseded-by: None
---

# S-Expression Parser Implementation Specification

## Overview

Implement a parser that converts a stream of tokens from the lexer into a generic S-expression tree structure. This parser operates on the token stream and builds an Abstract Syntax Tree (AST) representation of S-expressions.

## File Location

Create: `go-sexp-ast/sexp/parser.go`

## S-Expression Types

Define a type hierarchy for representing S-expressions:

```go
// SExp is the interface all S-expression nodes implement
type SExp interface {
    sexp()
    Pos() Position  // Returns position in source
}

// Position tracks source location
type Position struct {
    Offset int  // byte offset
    Line   int  // line number (1-based)
    Column int  // column number (1-based)
}

// Symbol represents an identifier/symbol
type Symbol struct {
    Position Position
    Value    string
}

// Keyword represents a keyword argument (starts with :)
type Keyword struct {
    Position Position
    Name     string  // without the : prefix
}

// String represents a string literal
type String struct {
    Position Position
    Value    string  // processed value (without quotes, escapes resolved)
}

// Number represents a numeric literal
type Number struct {
    Position Position
    Value    string  // string representation
}

// Nil represents the nil value
type Nil struct {
    Position Position
}

// List represents a list of S-expressions
type List struct {
    Position Position  // position of opening paren
    Elements []SExp
}
```

## Parser Structure

```go
type Parser struct {
    lexer   *Lexer
    current Token
    peek    Token
    errors  []string
}
```

## Public API

```go
// NewParser creates a new parser for the given input
func NewParser(input string) *Parser

// Parse parses the input and returns the S-expression tree
func (p *Parser) Parse() (SExp, error)

// ParseList parses input expecting a list and returns it
func (p *Parser) ParseList() (*List, error)

// Errors returns all parsing errors encountered
func (p *Parser) Errors() []string
```

## Implementation Requirements

### 1. Initialization

`NewParser` should:
- Create a new Lexer from the input
- Initialize current and peek tokens by calling `nextToken()` twice
- Initialize empty errors slice

### 2. Token Management

Implement `nextToken()`:
- Advances current to peek
- Reads next token from lexer into peek
- This maintains a one-token lookahead

Implement convenience methods:
```go
func (p *Parser) currentTokenIs(t TokenType) bool
func (p *Parser) peekTokenIs(t TokenType) bool
func (p *Parser) expectPeek(t TokenType) bool  // advances if match, error if not
```

### 3. Error Handling

Implement `addError(msg string)`:
- Appends formatted error message to errors slice
- Include current position information
- Format: `"line X, column Y: error message"`

Implement `peekError(t TokenType)`:
- Adds error for unexpected token type
- Format: `"expected next token to be X, got Y instead"`

### 4. Core Parsing Logic

#### Main Parse Method

`Parse()` should:
- Call `parseSExp()` to parse a single S-expression
- Return the result and any errors
- If errors slice is not empty, return `nil, error` with combined error message

#### Parse S-Expression

`parseSExp()` should dispatch based on current token type:

```go
func (p *Parser) parseSExp() SExp {
    switch p.current.Type {
    case LPAREN:
        return p.parseList()
    case SYMBOL:
        return p.parseSymbol()
    case KEYWORD:
        return p.parseKeyword()
    case STRING:
        return p.parseString()
    case NUMBER:
        return p.parseNumber()
    case NIL:
        return p.parseNil()
    default:
        p.addError(fmt.Sprintf("unexpected token: %v", p.current.Type))
        return nil
    }
}
```

#### Parse List

`parseList()` should:
- Verify current token is LPAREN
- Store position of LPAREN
- Create empty Elements slice
- Loop:
  - Advance to next token
  - If RPAREN, break
  - If EOF, error "unterminated list"
  - Otherwise, parse S-expression and append to Elements
- Verify we end on RPAREN
- Return List with position and elements

#### Parse Atomic Values

Each parse method for atomic types (Symbol, Keyword, String, Number, Nil):
- Extract position from current token
- Extract and process value
- Create appropriate node type
- Return the node

**String processing**:
- Remove surrounding quotes
- Process escape sequences (already done by lexer, but validate)

**Keyword processing**:
- Remove leading ':' from the name
- Store just the keyword name

**Number processing**:
- Store the string representation as-is
- No need to parse to int/float at this stage

### 5. Position Tracking

Every node's Position should be set from the token's position:

```go
Position{
    Offset: token.Pos,
    Line:   token.Line,
    Column: token.Column,
}
```

### 6. Interface Implementation

Each S-expression type must implement:

```go
func (s *Symbol) sexp() {}
func (k *Keyword) sexp() {}
func (s *String) sexp() {}
func (n *Number) sexp() {}
func (n *Nil) sexp() {}
func (l *List) sexp() {}

func (s *Symbol) Pos() Position { return s.Position }
func (k *Keyword) Pos() Position { return k.Position }
func (s *String) Pos() Position { return s.Position }
func (n *Number) Pos() Position { return n.Position }
func (n *Nil) Pos() Position { return n.Position }
func (l *List) Pos() Position { return l.Position }
```

## Edge Cases to Handle

1. **Empty input** → error "no S-expression found"
2. **Empty list `()`** → List with empty Elements slice
3. **Nested lists `((a) (b))`** → List containing Lists
4. **Unterminated list `(foo bar`** → error with position
5. **Extra closing paren `foo)`** → error at `)`
6. **Multiple root expressions** → Parse only parses one; additional content is not an error for Parse(), but ParseList() should handle this appropriately
7. **Keywords at root level** → valid, return Keyword node

## Testing Requirements

Create comprehensive tests in `parser_test.go`:

### 1. Basic Parsing

Test parsing simple S-expressions:

```go
// Symbols
"foo" → Symbol{Value: "foo"}
"GenDecl" → Symbol{Value: "GenDecl"}

// Keywords
":name" → Keyword{Name: "name"}
":package" → Keyword{Name: "package"}

// Strings
`"hello"` → String{Value: "hello"}
`"hello\nworld"` → String{Value: "hello\nworld"}

// Numbers
"42" → Number{Value: "42"}
"-10" → Number{Value: "-10"}
"3.14" → Number{Value: "3.14"}

// Nil
"nil" → Nil{}
```

### 2. List Parsing

```go
// Empty list
"()" → List{Elements: []}

// Simple list
"(foo bar)" → List{Elements: [Symbol{"foo"}, Symbol{"bar"}]}

// Nested list
"(foo (bar baz))" → List with nested List

// Keywords in list
"(:name foo :type bar)" → List with Keywords and Symbols
```

### 3. Complex Structures

Test parsing the canonical format structures:

```go
input := `(File :package 1 :name (Ident :namepos 9 :name "main"))`

Expected structure:
List{
  Elements: [
    Symbol{"File"},
    Keyword{"package"},
    Number{"1"},
    Keyword{"name"},
    List{
      Elements: [
        Symbol{"Ident"},
        Keyword{"namepos"},
        Number{"9"},
        Keyword{"name"},
        String{"main"},
      ]
    }
  ]
}
```

### 4. Error Cases

Test that errors are properly reported:

```go
// Unterminated list
"(foo bar" → error with position

// Unexpected token
")" → error "unexpected token"

// Multiple expressions (Parse should only parse first)
"foo bar" → parses Symbol{"foo"}, leaves "bar"
```

### 5. Position Tracking

Verify that Position information is correct:

```go
input := "(foo\n  bar)"

// Symbol "bar" should have:
// Line: 2
// Column: 3
```

## Helper Methods for Testing

Consider adding these to make testing easier:

```go
// String returns a string representation for debugging
func (s *Symbol) String() string
func (k *Keyword) String() string
func (l *List) String() string
// ... etc
```

## Example Usage

```go
input := `(File :package 1 :name (Ident :namepos 9 :name "main"))`
parser := NewParser(input)

sexp, err := parser.Parse()
if err != nil {
    log.Fatal(err)
}

// sexp is now a *List containing the parsed structure
list := sexp.(*List)
fmt.Printf("Parsed %d elements\n", len(list.Elements))
```

## Success Criteria

- Correctly parse all S-expression types
- Accurate position tracking for all nodes
- Proper error messages with positions
- Handle nested structures correctly
- Clean, readable code
- Comprehensive test coverage
- No panics on malformed input

## Notes

- This is a simple recursive descent parser
- Keep it straightforward - no need for complex backtracking
- Focus on clear error messages
- Position information is critical for later stages
- The parser should be forgiving where possible but report errors clearly
