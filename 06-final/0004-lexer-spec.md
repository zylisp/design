---
number: 0004
title: "S-Expression Lexer Implementation Specification"
author: Duncan McGreggor
created: 2025-10-01
updated: 2025-10-04
state: Final
supersedes: None
superseded-by: None
---

# S-Expression Lexer Implementation Specification

## Overview

Implement a lexer that tokenizes S-expression input into a stream of tokens. This lexer is specifically designed for parsing the canonical S-expression format used to represent Go AST nodes.

## File Location

Create: `go-sexp-ast/sexp/lexer.go`

## Token Types

Define the following token types:

```go
type TokenType int

const (
    // Special tokens
    EOF TokenType = iota
    ILLEGAL
    
    // Delimiters
    LPAREN  // (
    RPAREN  // )
    
    // Literals
    SYMBOL   // foo, bar, GenDecl, etc.
    STRING   // "hello"
    NUMBER   // 42, 3.14, -10
    KEYWORD  // :name, :package, :tok
    NIL      // nil
)
```

## Token Structure

```go
type Token struct {
    Type    TokenType
    Literal string
    Pos     int  // byte offset in input
    Line    int  // line number (1-based)
    Column  int  // column number (1-based)
}
```

## Lexer Structure

```go
type Lexer struct {
    input        string
    position     int  // current position in input (points to current char)
    readPosition int  // current reading position (after current char)
    ch           byte // current char under examination
    line         int  // current line number
    column       int  // current column number
}
```

## Public API

```go
// NewLexer creates a new lexer for the given input
func NewLexer(input string) *Lexer

// NextToken returns the next token from the input
func (l *Lexer) NextToken() Token

// Peek returns the next token without consuming it
func (l *Lexer) Peek() Token
```

## Implementation Requirements

### 1. Initialization

`NewLexer` should:
- Initialize the lexer with the input string
- Set position, readPosition to 0
- Set line to 1, column to 1
- Call `readChar()` to load the first character

### 2. Character Reading

Implement `readChar()`:
- Advances position to readPosition
- Sets ch to the character at readPosition
- Increments readPosition
- Updates line and column tracking:
  - When encountering '\n', increment line and reset column to 1
  - Otherwise, increment column
- Sets ch to 0 (null byte) when reaching end of input

Implement `peekChar()`:
- Returns the character at readPosition without advancing
- Returns 0 if at end of input

### 3. Whitespace Handling

Implement `skipWhitespace()`:
- Skip spaces, tabs, newlines, carriage returns
- Continue reading until a non-whitespace character is found

### 4. Token Recognition

#### Parentheses
- '(' → LPAREN
- ')' → RPAREN

#### Keywords
- Start with ':' followed by identifier characters
- Example: `:name`, `:package`, `:tok`
- Literal includes the ':' prefix

#### Symbols
- Start with letter or allowed symbol characters
- Continue with letters, digits, or allowed symbol characters
- Allowed characters: `a-zA-Z0-9_-+*/<>=!?`
- Examples: `File`, `GenDecl`, `IMPORT`, `main`

#### Special Symbol: nil
- The symbol "nil" should be recognized as the NIL token type
- This is the only symbol that gets special token type treatment

#### Strings
- Enclosed in double quotes
- Support escape sequences:
  - `\"` → `"`
  - `\\` → `\`
  - `\n` → newline
  - `\t` → tab
  - `\r` → carriage return
- Literal includes the surrounding quotes
- Error on unterminated string

#### Numbers
- Integer: optional '-' or '+', followed by digits
- Float: integer part, '.', fractional part
- Examples: `42`, `-10`, `3.14`, `0`, `-0.5`
- No scientific notation needed for Phase 1

#### Comments
- Semicolon `;` starts a comment
- Comment extends to end of line
- Comments are skipped (not returned as tokens)
- Whitespace handling continues after comment

### 5. Error Handling

For illegal characters:
- Return ILLEGAL token
- Set Literal to string representation of the character
- Position should point to the illegal character

### 6. Position Tracking

Every token must have accurate:
- `Pos`: byte offset in input where token starts
- `Line`: 1-based line number
- `Column`: 1-based column number

### 7. Helper Methods

Implement these private helper methods:

```go
func (l *Lexer) readChar()
func (l *Lexer) peekChar() byte
func (l *Lexer) skipWhitespace()
func (l *Lexer) skipComment()
func (l *Lexer) readSymbol() string
func (l *Lexer) readKeyword() string
func (l *Lexer) readString() string
func (l *Lexer) readNumber() string
func (l *Lexer) isLetter(ch byte) bool
func (l *Lexer) isDigit(ch byte) bool
func (l *Lexer) isSymbolChar(ch byte) bool
```

## Edge Cases to Handle

1. **Empty input** → return EOF token
2. **Unterminated string** → return ILLEGAL token with error message
3. **Multiple consecutive whitespace** → skip all, return next real token
4. **Comment at end of file** → skip comment, return EOF
5. **Empty list `()`** → LPAREN, RPAREN tokens
6. **Nested lists** → lexer doesn't need to track nesting, just tokenize

## Testing Requirements

The implementation should include tests for:

1. **Basic tokens**:
   - `()` → LPAREN, RPAREN
   - `nil` → NIL
   
2. **Symbols**:
   - `File` → SYMBOL
   - `GenDecl` → SYMBOL
   - `IMPORT` → SYMBOL
   
3. **Keywords**:
   - `:name` → KEYWORD
   - `:package` → KEYWORD
   - `:tok` → KEYWORD
   
4. **Strings**:
   - `"hello"` → STRING with literal `"hello"`
   - `"hello\nworld"` → STRING with escape processed
   - `"unterminated` → ILLEGAL
   
5. **Numbers**:
   - `42` → NUMBER
   - `-10` → NUMBER
   - `3.14` → NUMBER
   - `0` → NUMBER
   
6. **Comments**:
   - `; comment\n(` → LPAREN (comment skipped)
   
7. **Whitespace**:
   - `(  foo  )` → LPAREN, SYMBOL, RPAREN
   
8. **Position tracking**:
   - Verify line and column numbers are correct
   
9. **Complete S-expression**:
```lisp
(File :package 1 :name (Ident :namepos 9 :name "main"))
```

Should tokenize correctly with accurate positions.

## Example Usage

```go
input := `(File :package 1 :name (Ident :namepos 9 :name "main"))`
lexer := NewLexer(input)

for {
    tok := lexer.NextToken()
    fmt.Printf("%v\n", tok)
    if tok.Type == EOF {
        break
    }
}
```

## Success Criteria

- All token types correctly recognized
- Position information accurate for every token
- Comments properly skipped
- Whitespace properly handled
- Escape sequences in strings processed
- Clean, readable code with good error messages
- Comprehensive test coverage

## Notes

- Keep it simple - this is a straightforward lexer
- Focus on correctness over performance
- Good error messages are important
- Position tracking is critical for later error reporting
