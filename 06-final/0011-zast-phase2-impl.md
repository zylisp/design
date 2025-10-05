---
number: 0011
title: Phase 2 Implementation Specification - Easy Wins
author: Duncan McGreggor
created: 2025-10-03
updated: 2025-10-04
state: Final
supersedes: None
superseded-by: None
---

# Phase 2 Implementation Specification - Easy Wins

**Project**: zast
**Wave**: 1 of 5
**Goal**: Implement 25 straightforward AST nodes
**Estimated Effort**: 3-4 days
**Prerequisites**: Phase 1 complete (lexer, parser, builder, writer, tests)

---

## Overview

Phase 2 adds support for basic expressions, statements, types, and declaration specs that follow the same patterns established in Phase 1. These nodes enable handling of variables, assignments, basic operations, and type definitions.

**What you'll be able to handle after Phase 2**:

- Variable declarations and assignments
- Arithmetic and logical operations
- Basic type definitions
- Simple goroutines and channels
- Return statements and control flow jumps

---

## Coding Style and Testing

Note that the following rules override any example code show below that may contradict these rules.

- Follow the example of existing code
- When practical, avoid use of raw strings; instead, define a const and use the const
- Keep errors consolidated in errors.go and <package>/errors.go
- Define methods where appropriate and use those (improves legibility)
- When testing the binary, don't compile it -- just run `go run ./cmd/demo`
- When running the tests, use `go test -v ./...`

---

## Implementation Checklist

### Expressions (7 nodes)

- [ ] `UnaryExpr` - Unary operations: `!x`, `-y`, `*ptr`, `&addr`
- [ ] `BinaryExpr` - Binary operations: `x + y`, `a && b`
- [ ] `ParenExpr` - Parenthesized expressions: `(x + y)`
- [ ] `StarExpr` - Pointer type/dereference: `*int`, `*ptr`
- [ ] `IndexExpr` - Array/map indexing: `arr[i]`
- [ ] `SliceExpr` - Slice operations: `arr[1:5]`
- [ ] `KeyValueExpr` - Key-value pairs: `{key: value}`

### Statements (9 nodes)

- [ ] `ReturnStmt` - Return statements
- [ ] `AssignStmt` - Assignments: `x = 5`, `x, y := 1, 2`
- [ ] `IncDecStmt` - Increment/decrement: `x++`, `y--`
- [ ] `BranchStmt` - `break`, `continue`, `goto`, `fallthrough`
- [ ] `DeferStmt` - Defer statements
- [ ] `GoStmt` - Goroutine launch
- [ ] `SendStmt` - Channel send: `ch <- value`
- [ ] `EmptyStmt` - Empty statement
- [ ] `LabeledStmt` - Labeled statements

### Types (3 nodes)

- [ ] `ArrayType` - Array types: `[10]int`
- [ ] `MapType` - Map types: `map[string]int`
- [ ] `ChanType` - Channel types: `chan int`

### Specs (2 nodes)

- [ ] `ValueSpec` - Variable/constant declarations
- [ ] `TypeSpec` - Type declarations

### Updates to Existing

- [ ] Update `GenDecl` to handle `ValueSpec` and `TypeSpec`

---

## Part 1: Expression Nodes

### UnaryExpr

**Go AST Structure**:

```go
type UnaryExpr struct {
    OpPos token.Pos   // position of Op
    Op    token.Token // operator (!, -, +, *, &, <-, etc.)
    X     Expr        // operand
}
```

**Canonical S-Expression Format**:

```lisp
(UnaryExpr
  :oppos <pos>
  :op <token>
  :x <expr>)
```

**Example**:

```go
!ready        // (UnaryExpr :oppos 10 :op NOT :x (Ident ...))
-value        // (UnaryExpr :oppos 15 :op SUB :x (Ident ...))
*ptr          // (UnaryExpr :oppos 20 :op MUL :x (Ident ...))  // dereference
&addr         // (UnaryExpr :oppos 25 :op AND :x (Ident ...))  // address-of
<-ch          // (UnaryExpr :oppos 30 :op ARROW :x (Ident ...)) // channel receive
```

**Token Mapping** (add to `parseToken` and `writeToken`):

- `NOT` → `token.NOT` (!)
- `SUB` → `token.SUB` (-)
- `ADD` → `token.ADD` (+)
- `MUL` → `token.MUL` (*) - used for both multiplication and pointer dereference
- `AND` → `token.AND` (&) - used for both bitwise-and and address-of
- `XOR` → `token.XOR` (^)
- `ARROW` → `token.ARROW` (<-)

**Builder Implementation** (`builder.go`):

```go
func (b *Builder) buildUnaryExpr(s sexp.SExp) (*ast.UnaryExpr, error) {
    list, ok := b.expectList(s, "UnaryExpr")
    if !ok {
        return nil, fmt.Errorf("not a list")
    }

    if !b.expectSymbol(list.Elements[0], "UnaryExpr") {
        return nil, fmt.Errorf("not a UnaryExpr node")
    }

    args := b.parseKeywordArgs(list.Elements)

    opposVal, ok := b.requireKeyword(args, "oppos", "UnaryExpr")
    if !ok {
        return nil, fmt.Errorf("missing oppos")
    }

    opVal, ok := b.requireKeyword(args, "op", "UnaryExpr")
    if !ok {
        return nil, fmt.Errorf("missing op")
    }

    xVal, ok := b.requireKeyword(args, "x", "UnaryExpr")
    if !ok {
        return nil, fmt.Errorf("missing x")
    }

    op, err := b.parseToken(opVal)
    if err != nil {
        return nil, fmt.Errorf("invalid op: %v", err)
    }

    x, err := b.buildExpr(xVal)
    if err != nil {
        return nil, fmt.Errorf("invalid x: %v", err)
    }

    return &ast.UnaryExpr{
        OpPos: b.parsePos(opposVal),
        Op:    op,
        X:     x,
    }, nil
}
```

**Writer Implementation** (`writer.go`):

```go
func (w *Writer) writeUnaryExpr(expr *ast.UnaryExpr) error {
    w.openList()
    w.writeSymbol("UnaryExpr")
    w.writeSpace()
    w.writeKeyword("oppos")
    w.writeSpace()
    w.writePos(expr.OpPos)
    w.writeSpace()
    w.writeKeyword("op")
    w.writeSpace()
    w.writeToken(expr.Op)
    w.writeSpace()
    w.writeKeyword("x")
    w.writeSpace()
    if err := w.writeExpr(expr.X); err != nil {
        return err
    }
    w.closeList()
    return nil
}
```

**Add to Expression Dispatchers**:

In `buildExpr`, add:

```go
case "UnaryExpr":
    return b.buildUnaryExpr(s)
```

In `writeExpr`, add:

```go
case *ast.UnaryExpr:
    return w.writeUnaryExpr(e)
```

**Tests** (`builder_test.go` and `writer_test.go`):

```go
func TestBuildUnaryExpr(t *testing.T) {
    tests := []struct {
        input    string
        op       token.Token
        operand  string
    }{
        {`(UnaryExpr :oppos 10 :op NOT :x (Ident :namepos 11 :name "ready" :obj nil))`,
         token.NOT, "ready"},
        {`(UnaryExpr :oppos 15 :op SUB :x (Ident :namepos 16 :name "value" :obj nil))`,
         token.SUB, "value"},
        {`(UnaryExpr :oppos 20 :op MUL :x (Ident :namepos 21 :name "ptr" :obj nil))`,
         token.MUL, "ptr"},
    }

    for _, tt := range tests {
        parser := sexp.NewParser(tt.input)
        sexpNode, err := parser.Parse()
        if err != nil {
            t.Fatalf("parse error: %v", err)
        }

        builder := NewBuilder()
        expr, err := builder.buildUnaryExpr(sexpNode)
        if err != nil {
            t.Fatalf("build error: %v", err)
        }

        if expr.Op != tt.op {
            t.Fatalf("expected op %v, got %v", tt.op, expr.Op)
        }

        ident, ok := expr.X.(*ast.Ident)
        if !ok || ident.Name != tt.operand {
            t.Fatalf("expected operand %q, got %v", tt.operand, expr.X)
        }
    }
}
```

---

### BinaryExpr

**Go AST Structure**:

```go
type BinaryExpr struct {
    X     Expr        // left operand
    OpPos token.Pos   // position of Op
    Op    token.Token // operator
    Y     Expr        // right operand
}
```

**Canonical S-Expression Format**:

```lisp
(BinaryExpr
  :x <expr>
  :oppos <pos>
  :op <token>
  :y <expr>)
```

**Example**:

```go
x + y         // (BinaryExpr :x (Ident...) :oppos 15 :op ADD :y (Ident...))
a && b        // (BinaryExpr :x (Ident...) :oppos 20 :op LAND :y (Ident...))
i < len(arr)  // (BinaryExpr :x (Ident...) :oppos 25 :op LSS :y (CallExpr...))
```

**Additional Token Mapping**:

- `ADD` → `token.ADD` (+)
- `SUB` → `token.SUB` (-)
- `MUL` → `token.MUL` (*)
- `QUO` → `token.QUO` (/)
- `REM` → `token.REM` (%)
- `AND` → `token.AND` (&)
- `OR` → `token.OR` (|)
- `XOR` → `token.XOR` (^)
- `SHL` → `token.SHL` (<<)
- `SHR` → `token.SHR` (>>)
- `AND_NOT` → `token.AND_NOT` (&^)
- `LAND` → `token.LAND` (&&)
- `LOR` → `token.LOR` (||)
- `EQL` → `token.EQL` (==)
- `NEQ` → `token.NEQ` (!=)
- `LSS` → `token.LSS` (<)
- `LEQ` → `token.LEQ` (<=)
- `GTR` → `token.GTR` (>)
- `GEQ` → `token.GEQ` (>=)

**Implementation Pattern**: Follow UnaryExpr pattern with four fields instead of three.

---

### ParenExpr

**Go AST Structure**:

```go
type ParenExpr struct {
    Lparen token.Pos // position of "("
    X      Expr      // parenthesized expression
    Rparen token.Pos // position of ")"
}
```

**Canonical S-Expression Format**:

```lisp
(ParenExpr
  :lparen <pos>
  :x <expr>
  :rparen <pos>)
```

**Example**:

```go
(x + y) * z   // (BinaryExpr :x (ParenExpr :lparen 10 :x (BinaryExpr...) :rparen 15) ...)
```

**Implementation**: Straightforward three-field node.

---

### StarExpr

**Go AST Structure**:

```go
type StarExpr struct {
    Star token.Pos // position of "*"
    X    Expr      // operand
}
```

**Canonical S-Expression Format**:

```lisp
(StarExpr
  :star <pos>
  :x <expr>)
```

**Example**:

```go
*MyType       // (StarExpr :star 10 :x (Ident :namepos 11 :name "MyType" :obj nil))
```

**Note**: StarExpr is used for pointer types in type contexts. In expression contexts, pointer dereference uses UnaryExpr with MUL operator.

---

### IndexExpr

**Go AST Structure**:

```go
type IndexExpr struct {
    X      Expr      // expression
    Lbrack token.Pos // position of "["
    Index  Expr      // index expression
    Rbrack token.Pos // position of "]"
}
```

**Canonical S-Expression Format**:

```lisp
(IndexExpr
  :x <expr>
  :lbrack <pos>
  :index <expr>
  :rbrack <pos>)
```

**Example**:

```go
arr[i]        // (IndexExpr :x (Ident...) :lbrack 15 :index (Ident...) :rbrack 17)
m[key]        // (IndexExpr :x (Ident...) :lbrack 20 :index (Ident...) :rbrack 24)
```

---

### SliceExpr

**Go AST Structure**:

```go
type SliceExpr struct {
    X      Expr      // expression
    Lbrack token.Pos // position of "["
    Low    Expr      // begin of slice range; or nil
    High   Expr      // end of slice range; or nil
    Max    Expr      // maximum capacity of slice; or nil
    Slice3 bool      // true if 3-index slice (2 colons present)
    Rbrack token.Pos // position of "]"
}
```

**Canonical S-Expression Format**:

```lisp
(SliceExpr
  :x <expr>
  :lbrack <pos>
  :low <expr-or-nil>
  :high <expr-or-nil>
  :max <expr-or-nil>
  :slice3 <bool>
  :rbrack <pos>)
```

**Examples**:

```go
arr[1:5]      // (SliceExpr :x ... :low (BasicLit...) :high (BasicLit...) :max nil :slice3 false ...)
arr[1:5:10]   // (SliceExpr :x ... :low ... :high ... :max (BasicLit...) :slice3 true ...)
arr[:]        // (SliceExpr :x ... :low nil :high nil :max nil :slice3 false ...)
arr[1:]       // (SliceExpr :x ... :low (BasicLit...) :high nil :max nil :slice3 false ...)
```

**Special Handling**: Need to represent boolean value for `slice3` field. Use Symbol "true" or "false".

**Builder additions**:

```go
func (b *Builder) parseBool(s sexp.SExp) (bool, error) {
    sym, ok := s.(*sexp.Symbol)
    if !ok {
        return false, fmt.Errorf("expected symbol for bool, got %T", s)
    }

    switch sym.Value {
    case "true":
        return true, nil
    case "false":
        return false, nil
    default:
        return false, fmt.Errorf("invalid bool value: %s", sym.Value)
    }
}
```

**Writer additions**:

```go
func (w *Writer) writeBool(b bool) {
    if b {
        w.writeSymbol("true")
    } else {
        w.writeSymbol("false")
    }
}
```

---

### KeyValueExpr

**Go AST Structure**:

```go
type KeyValueExpr struct {
    Key   Expr
    Colon token.Pos // position of ":"
    Value Expr
}
```

**Canonical S-Expression Format**:

```lisp
(KeyValueExpr
  :key <expr>
  :colon <pos>
  :value <expr>)
```

**Example**:

```go
{x: 10, y: 20}  // In CompositeLit (Phase 4), contains KeyValueExprs
```

---

## Part 2: Statement Nodes

### ReturnStmt

**Go AST Structure**:

```go
type ReturnStmt struct {
    Return  token.Pos // position of "return" keyword
    Results []Expr    // result expressions; or nil
}
```

**Canonical S-Expression Format**:

```lisp
(ReturnStmt
  :return <pos>
  :results (<expr> ...))
```

**Examples**:

```go
return              // (ReturnStmt :return 10 :results ())
return x            // (ReturnStmt :return 15 :results ((Ident...)))
return x, y         // (ReturnStmt :return 20 :results ((Ident...) (Ident...)))
return x + y        // (ReturnStmt :return 25 :results ((BinaryExpr...)))
```

**Implementation Pattern**: Similar to other nodes with expression list.

---

### AssignStmt

**Go AST Structure**:

```go
type AssignStmt struct {
    Lhs    []Expr
    TokPos token.Pos   // position of Tok
    Tok    token.Token // assignment token (ASSIGN, DEFINE, or operator)
    Rhs    []Expr
}
```

**Canonical S-Expression Format**:

```lisp
(AssignStmt
  :lhs (<expr> ...)
  :tokpos <pos>
  :tok <token>
  :rhs (<expr> ...))
```

**Examples**:

```go
x = 5               // (AssignStmt :lhs ((Ident...)) :tokpos 10 :tok ASSIGN :rhs ((BasicLit...)))
x, y := 1, 2        // (AssignStmt :lhs (...) :tokpos 15 :tok DEFINE :rhs (...))
x += 10             // (AssignStmt :lhs (...) :tokpos 20 :tok ADD_ASSIGN :rhs (...))
```

**Token Mapping** (add to parseToken/writeToken):

- `ASSIGN` → `token.ASSIGN` (=)
- `DEFINE` → `token.DEFINE` (:=)
- `ADD_ASSIGN` → `token.ADD_ASSIGN` (+=)
- `SUB_ASSIGN` → `token.SUB_ASSIGN` (-=)
- `MUL_ASSIGN` → `token.MUL_ASSIGN` (*=)
- `QUO_ASSIGN` → `token.QUO_ASSIGN` (/=)
- `REM_ASSIGN` → `token.REM_ASSIGN` (%=)
- `AND_ASSIGN` → `token.AND_ASSIGN` (&=)
- `OR_ASSIGN` → `token.OR_ASSIGN` (|=)
- `XOR_ASSIGN` → `token.XOR_ASSIGN` (^=)
- `SHL_ASSIGN` → `token.SHL_ASSIGN` (<<=)
- `SHR_ASSIGN` → `token.SHR_ASSIGN` (>>=)
- `AND_NOT_ASSIGN` → `token.AND_NOT_ASSIGN` (&^=)

---

### IncDecStmt

**Go AST Structure**:

```go
type IncDecStmt struct {
    X      Expr
    TokPos token.Pos   // position of Tok
    Tok    token.Token // INC or DEC
}
```

**Canonical S-Expression Format**:

```lisp
(IncDecStmt
  :x <expr>
  :tokpos <pos>
  :tok <token>)
```

**Examples**:

```go
x++                 // (IncDecStmt :x (Ident...) :tokpos 10 :tok INC)
y--                 // (IncDecStmt :x (Ident...) :tokpos 15 :tok DEC)
```

**Token Mapping**:

- `INC` → `token.INC` (++)
- `DEC` → `token.DEC` (--)

---

### BranchStmt

**Go AST Structure**:

```go
type BranchStmt struct {
    TokPos token.Pos   // position of Tok
    Tok    token.Token // keyword token (BREAK, CONTINUE, GOTO, FALLTHROUGH)
    Label  *Ident      // label name; or nil
}
```

**Canonical S-Expression Format**:

```lisp
(BranchStmt
  :tokpos <pos>
  :tok <token>
  :label <ident-or-nil>)
```

**Examples**:

```go
break               // (BranchStmt :tokpos 10 :tok BREAK :label nil)
continue            // (BranchStmt :tokpos 15 :tok CONTINUE :label nil)
goto Label          // (BranchStmt :tokpos 20 :tok GOTO :label (Ident...))
fallthrough         // (BranchStmt :tokpos 25 :tok FALLTHROUGH :label nil)
```

**Token Mapping**:

- `BREAK` → `token.BREAK`
- `CONTINUE` → `token.CONTINUE`
- `GOTO` → `token.GOTO`
- `FALLTHROUGH` → `token.FALLTHROUGH`

---

### DeferStmt

**Go AST Structure**:

```go
type DeferStmt struct {
    Defer token.Pos // position of "defer" keyword
    Call  *CallExpr
}
```

**Canonical S-Expression Format**:

```lisp
(DeferStmt
  :defer <pos>
  :call <CallExpr>)
```

**Example**:

```go
defer cleanup()     // (DeferStmt :defer 10 :call (CallExpr...))
```

**Note**: Call must be a CallExpr, not just any Expr.

---

### GoStmt

**Go AST Structure**:

```go
type GoStmt struct {
    Go   token.Pos // position of "go" keyword
    Call *CallExpr
}
```

**Canonical S-Expression Format**:

```lisp
(GoStmt
  :go <pos>
  :call <CallExpr>)
```

**Example**:

```go
go worker()         // (GoStmt :go 10 :call (CallExpr...))
```

---

### SendStmt

**Go AST Structure**:

```go
type SendStmt struct {
    Chan  Expr
    Arrow token.Pos // position of "<-"
    Value Expr
}
```

**Canonical S-Expression Format**:

```lisp
(SendStmt
  :chan <expr>
  :arrow <pos>
  :value <expr>)
```

**Example**:

```go
ch <- value         // (SendStmt :chan (Ident...) :arrow 15 :value (Ident...))
```

---

### EmptyStmt

**Go AST Structure**:

```go
type EmptyStmt struct {
    Semicolon token.Pos // position of following ";"
    Implicit  bool      // if set, ";" was omitted in the source
}
```

**Canonical S-Expression Format**:

```lisp
(EmptyStmt
  :semicolon <pos>
  :implicit <bool>)
```

**Example**:

```go
;                   // (EmptyStmt :semicolon 10 :implicit false)
```

---

### LabeledStmt

**Go AST Structure**:

```go
type LabeledStmt struct {
    Label *Ident
    Colon token.Pos // position of ":"
    Stmt  Stmt
}
```

**Canonical S-Expression Format**:

```lisp
(LabeledStmt
  :label <Ident>
  :colon <pos>
  :stmt <Stmt>)
```

**Example**:

```go
Loop:               // (LabeledStmt :label (Ident...) :colon 15
    for { ... }     //   :stmt (ForStmt...))
```

---

## Part 3: Type Nodes

### ArrayType

**Go AST Structure**:

```go
type ArrayType struct {
    Lbrack token.Pos // position of "["
    Len    Expr      // Ellipsis node for [...]T array types, nil for slice types
    Elt    Expr      // element type
}
```

**Canonical S-Expression Format**:

```lisp
(ArrayType
  :lbrack <pos>
  :len <expr-or-nil>
  :elt <expr>)
```

**Examples**:

```go
[10]int             // (ArrayType :lbrack 10 :len (BasicLit :valuepos 11 :kind INT :value "10") :elt (Ident...))
[]string            // (ArrayType :lbrack 15 :len nil :elt (Ident...))  // slice type
[...]int            // (ArrayType :lbrack 20 :len (Ellipsis...) :elt (Ident...))
```

**Note**: Len is nil for slice types, an Ellipsis node for `[...]T`, or an expression for fixed-size arrays.

---

### MapType

**Go AST Structure**:

```go
type MapType struct {
    Map   token.Pos // position of "map" keyword
    Key   Expr
    Value Expr
}
```

**Canonical S-Expression Format**:

```lisp
(MapType
  :map <pos>
  :key <expr>
  :value <expr>)
```

**Example**:

```go
map[string]int      // (MapType :map 10 :key (Ident...) :value (Ident...))
```

---

### ChanType

**Go AST Structure**:

```go
type ChanType struct {
    Begin token.Pos  // position of "chan" keyword or "<-" (whichever comes first)
    Arrow token.Pos  // position of "<-" (SEND or RECV); or NoPos
    Dir   ChanDir    // channel direction
    Value Expr       // value type
}

type ChanDir int
const (
    SEND ChanDir = 1 << iota
    RECV
)
```

**Canonical S-Expression Format**:

```lisp
(ChanType
  :begin <pos>
  :arrow <pos>
  :dir <chandir>
  :value <expr>)
```

**Examples**:

```go
chan int            // (ChanType :begin 10 :arrow 0 :dir SEND_RECV :value (Ident...))
<-chan int          // (ChanType :begin 15 :arrow 15 :dir RECV :value (Ident...))
chan<- int          // (ChanType :begin 20 :arrow 24 :dir SEND :value (Ident...))
```

**ChanDir Mapping**:

- `SEND_RECV` → `ast.SEND | ast.RECV` (3) - bidirectional
- `SEND` → `ast.SEND` (1) - send-only
- `RECV` → `ast.RECV` (2) - receive-only

**Builder helper**:

```go
func (b *Builder) parseChanDir(s sexp.SExp) (ast.ChanDir, error) {
    sym, ok := s.(*sexp.Symbol)
    if !ok {
        return 0, fmt.Errorf("expected symbol for ChanDir, got %T", s)
    }

    switch sym.Value {
    case "SEND":
        return ast.SEND, nil
    case "RECV":
        return ast.RECV, nil
    case "SEND_RECV":
        return ast.SEND | ast.RECV, nil
    default:
        return 0, fmt.Errorf("unknown ChanDir: %s", sym.Value)
    }
}
```

**Writer helper**:

```go
func (w *Writer) writeChanDir(dir ast.ChanDir) {
    switch dir {
    case ast.SEND:
        w.writeSymbol("SEND")
    case ast.RECV:
        w.writeSymbol("RECV")
    case ast.SEND | ast.RECV:
        w.writeSymbol("SEND_RECV")
    default:
        w.writeSymbol("SEND_RECV")
    }
}
```

---

## Part 4: Spec Nodes

### ValueSpec

**Go AST Structure**:

```go
type ValueSpec struct {
    Doc     *CommentGroup // associated documentation; or nil
    Names   []*Ident      // value names (len(Names) > 0)
    Type    Expr          // value type; or nil
    Values  []Expr        // initial values; or nil
    Comment *CommentGroup // line comments; or nil
}
```

**Canonical S-Expression Format**:

```lisp
(ValueSpec
  :doc <CommentGroup-or-nil>
  :names (<Ident> ...)
  :type <expr-or-nil>
  :values (<expr> ...)
  :comment <CommentGroup-or-nil>)
```

**Examples**:

```go
var x int           // (ValueSpec :doc nil :names ((Ident...)) :type (Ident...) :values () :comment nil)
var x, y = 1, 2     // (ValueSpec :doc nil :names (...) :type nil :values (...) :comment nil)
const Pi = 3.14     // (ValueSpec :doc nil :names ((Ident...)) :type nil :values ((BasicLit...)) :comment nil)
```

**Note**: For Phase 1, write `:doc nil` and `:comment nil`. Full comment support comes in Phase 5.

---

### TypeSpec

**Go AST Structure**:

```go
type TypeSpec struct {
    Doc        *CommentGroup // associated documentation; or nil
    Name       *Ident        // type name
    TypeParams *FieldList    // type parameters; or nil (Go 1.18+)
    Assign     token.Pos     // position of '=', if any
    Type       Expr          // type definition
    Comment    *CommentGroup // line comments; or nil
}
```

**Canonical S-Expression Format**:

```lisp
(TypeSpec
  :doc <CommentGroup-or-nil>
  :name <Ident>
  :typeparams <FieldList-or-nil>
  :assign <pos>
  :type <expr>
  :comment <CommentGroup-or-nil>)
```

**Examples**:

```go
type MyInt int      // (TypeSpec :doc nil :name (Ident...) :typeparams nil :assign 0 :type (Ident...) :comment nil)
type Point struct{} // (TypeSpec :doc nil :name (Ident...) :typeparams nil :assign 0 :type (StructType...) :comment nil)
```

**Note**:

- For Phase 1, write `:doc nil` and `:comment nil`
- `:typeparams nil` unless targeting Go 1.18+ generics
- `:assign 0` for type definitions, non-zero for type aliases

---

## Part 5: Update GenDecl

**Current GenDecl** only handles `ImportSpec`. Update to handle all spec types:

**In `buildSpec` dispatcher**, add:

```go
case "ValueSpec":
    return b.buildValueSpec(s)
case "TypeSpec":
    return b.buildTypeSpec(s)
```

**In `writeSpec` dispatcher**, add:

```go
case *ast.ValueSpec:
    return w.writeValueSpec(s)
case *ast.TypeSpec:
    return w.writeTypeSpec(s)
```

**Test that GenDecl works with all spec types**:

```go
func TestGenDeclWithValueSpec(t *testing.T) {
    input := `(GenDecl
        :doc nil
        :tok VAR
        :tokpos 10
        :lparen 0
        :specs ((ValueSpec :doc nil :names ((Ident :namepos 14 :name "x" :obj nil)) :type (Ident :namepos 16 :name "int" :obj nil) :values () :comment nil))
        :rparen 0)`

    parser := sexp.NewParser(input)
    sexpNode, _ := parser.Parse()

    builder := NewBuilder()
    decl, err := builder.buildGenDecl(sexpNode)

    assert.NoError(t, err)
    assert.Equal(t, token.VAR, decl.Tok)
    assert.Len(t, decl.Specs, 1)

    valueSpec, ok := decl.Specs[0].(*ast.ValueSpec)
    assert.True(t, ok)
    assert.Equal(t, "x", valueSpec.Names[0].Name)
}
```

---

## Part 6: Statement Dispatcher Updates

Update `buildStmt` in `builder.go`:

```go
func (b *Builder) buildStmt(s sexp.SExp) (ast.Stmt, error) {
    list, ok := b.expectList(s, "statement")
    if !ok {
        return nil, fmt.Errorf("expected list")
    }

    if len(list.Elements) == 0 {
        return nil, fmt.Errorf("empty list")
    }

    sym, ok := list.Elements[0].(*sexp.Symbol)
    if !ok {
        return nil, fmt.Errorf("expected symbol as first element")
    }

    switch sym.Value {
    case "ExprStmt":
        return b.buildExprStmt(s)
    case "BlockStmt":
        return b.buildBlockStmt(s)
    case "ReturnStmt":
        return b.buildReturnStmt(s)
    case "AssignStmt":
        return b.buildAssignStmt(s)
    case "IncDecStmt":
        return b.buildIncDecStmt(s)
    case "BranchStmt":
        return b.buildBranchStmt(s)
    case "DeferStmt":
        return b.buildDeferStmt(s)
    case "GoStmt":
        return b.buildGoStmt(s)
    case "SendStmt":
        return b.buildSendStmt(s)
    case "EmptyStmt":
        return b.buildEmptyStmt(s)
    case "LabeledStmt":
        return b.buildLabeledStmt(s)
    default:
        return nil, fmt.Errorf("unknown statement type: %s", sym.Value)
    }
}
```

Update `writeStmt` in `writer.go`:

```go
func (w *Writer) writeStmt(stmt ast.Stmt) error {
    if stmt == nil {
        w.writeSymbol("nil")
        return nil
    }

    switch s := stmt.(type) {
    case *ast.ExprStmt:
        return w.writeExprStmt(s)
    case *ast.BlockStmt:
        return w.writeBlockStmt(s)
    case *ast.ReturnStmt:
        return w.writeReturnStmt(s)
    case *ast.AssignStmt:
        return w.writeAssignStmt(s)
    case *ast.IncDecStmt:
        return w.writeIncDecStmt(s)
    case *ast.BranchStmt:
        return w.writeBranchStmt(s)
    case *ast.DeferStmt:
        return w.writeDeferStmt(s)
    case *ast.GoStmt:
        return w.writeGoStmt(s)
    case *ast.SendStmt:
        return w.writeSendStmt(s)
    case *ast.EmptyStmt:
        return w.writeEmptyStmt(s)
    case *ast.LabeledStmt:
        return w.writeLabeledStmt(s)
    default:
        return fmt.Errorf("unknown statement type: %T", stmt)
    }
}
```

---

## Part 7: Expression Dispatcher Updates

Update `buildExpr` to include all new expression types:

```go
switch sym.Value {
case "Ident":
    return b.buildIdent(s)
case "BasicLit":
    return b.buildBasicLit(s)
case "CallExpr":
    return b.buildCallExpr(s)
case "SelectorExpr":
    return b.buildSelectorExpr(s)
case "UnaryExpr":
    return b.buildUnaryExpr(s)
case "BinaryExpr":
    return b.buildBinaryExpr(s)
case "ParenExpr":
    return b.buildParenExpr(s)
case "StarExpr":
    return b.buildStarExpr(s)
case "IndexExpr":
    return b.buildIndexExpr(s)
case "SliceExpr":
    return b.buildSliceExpr(s)
case "KeyValueExpr":
    return b.buildKeyValueExpr(s)
case "ArrayType":
    return b.buildArrayType(s)
case "MapType":
    return b.buildMapType(s)
case "ChanType":
    return b.buildChanType(s)
default:
    return nil, fmt.Errorf("unknown expression type: %s", sym.Value)
}
```

Update `writeExpr` similarly for the writer.

---

## Part 8: Token Mapping Expansion

Expand `parseToken` in `builder.go` to handle all new tokens:

```go
func (b *Builder) parseToken(s sexp.SExp) (token.Token, error) {
    sym, ok := s.(*sexp.Symbol)
    if !ok {
        return token.ILLEGAL, fmt.Errorf("expected symbol for token, got %T", s)
    }

    switch sym.Value {
    // Existing tokens
    case "IMPORT":
        return token.IMPORT, nil
    case "CONST":
        return token.CONST, nil
    case "TYPE":
        return token.TYPE, nil
    case "VAR":
        return token.VAR, nil
    case "INT":
        return token.INT, nil
    case "FLOAT":
        return token.FLOAT, nil
    case "IMAG":
        return token.IMAG, nil
    case "CHAR":
        return token.CHAR, nil
    case "STRING":
        return token.STRING, nil

    // Operators (Phase 2)
    case "ADD":
        return token.ADD, nil
    case "SUB":
        return token.SUB, nil
    case "MUL":
        return token.MUL, nil
    case "QUO":
        return token.QUO, nil
    case "REM":
        return token.REM, nil
    case "AND":
        return token.AND, nil
    case "OR":
        return token.OR, nil
    case "XOR":
        return token.XOR, nil
    case "SHL":
        return token.SHL, nil
    case "SHR":
        return token.SHR, nil
    case "AND_NOT":
        return token.AND_NOT, nil
    case "LAND":
        return token.LAND, nil
    case "LOR":
        return token.LOR, nil
    case "ARROW":
        return token.ARROW, nil
    case "INC":
        return token.INC, nil
    case "DEC":
        return token.DEC, nil

    // Comparison
    case "EQL":
        return token.EQL, nil
    case "LSS":
        return token.LSS, nil
    case "GTR":
        return token.GTR, nil
    case "ASSIGN":
        return token.ASSIGN, nil
    case "NOT":
        return token.NOT, nil
    case "NEQ":
        return token.NEQ, nil
    case "LEQ":
        return token.LEQ, nil
    case "GEQ":
        return token.GEQ, nil
    case "DEFINE":
        return token.DEFINE, nil

    // Assignment operators
    case "ADD_ASSIGN":
        return token.ADD_ASSIGN, nil
    case "SUB_ASSIGN":
        return token.SUB_ASSIGN, nil
    case "MUL_ASSIGN":
        return token.MUL_ASSIGN, nil
    case "QUO_ASSIGN":
        return token.QUO_ASSIGN, nil
    case "REM_ASSIGN":
        return token.REM_ASSIGN, nil
    case "AND_ASSIGN":
        return token.AND_ASSIGN, nil
    case "OR_ASSIGN":
        return token.OR_ASSIGN, nil
    case "XOR_ASSIGN":
        return token.XOR_ASSIGN, nil
    case "SHL_ASSIGN":
        return token.SHL_ASSIGN, nil
    case "SHR_ASSIGN":
        return token.SHR_ASSIGN, nil
    case "AND_NOT_ASSIGN":
        return token.AND_NOT_ASSIGN, nil

    // Keywords
    case "BREAK":
        return token.BREAK, nil
    case "CONTINUE":
        return token.CONTINUE, nil
    case "GOTO":
        return token.GOTO, nil
    case "FALLTHROUGH":
        return token.FALLTHROUGH, nil

    default:
        return token.ILLEGAL, fmt.Errorf("unknown token: %s", sym.Value)
    }
}
```

Mirror this expansion in `writeToken` in `writer.go`.

---

## Part 9: Integration Tests

After implementing all Phase 2 nodes, create comprehensive integration tests:

### Test 1: Variables and Arithmetic

```go
func TestWave1Variables(t *testing.T) {
    source := `package main

var x int = 42
var y, z = 10, 20

func calculate() int {
    result := x + y - z
    result *= 2
    return result
}
`
    // Parse -> Write -> Parse -> Build -> Compare
    testRoundTrip(t, source)
}
```

### Test 2: Pointer Operations

```go
func TestWave1Pointers(t *testing.T) {
    source := `package main

type Point struct {
    X int
    Y int
}

func modify(p *Point) {
    p.X = 10
    p.Y = 20
}

func main() {
    pt := &Point{X: 1, Y: 2}
    modify(pt)
}
`
    testRoundTrip(t, source)
}
```

### Test 3: Arrays and Slices

```go
func TestWave1ArraysSlices(t *testing.T) {
    source := `package main

var arr [10]int
var slice []string

func process() {
    x := arr[0]
    y := slice[1:5]
    arr[2] = x + 100
}
`
    testRoundTrip(t, source)
}
```

### Test 4: Maps

```go
func TestWave1Maps(t *testing.T) {
    source := `package main

var m map[string]int

func lookup(key string) int {
    return m[key]
}
`
    testRoundTrip(t, source)
}
```

### Test 5: Channels and Goroutines

```go
func TestWave1Channels(t *testing.T) {
    source := `package main

var ch chan int

func sender() {
    ch <- 42
}

func receiver() {
    value := <-ch
    return value
}

func main() {
    go sender()
    defer cleanup()
}
`
    testRoundTrip(t, source)
}
```

### Test 6: Type Definitions

```go
func TestWave1TypeDefs(t *testing.T) {
    source := `package main

type MyInt int
type StringSlice []string
type IntMap map[string]int

const Pi = 3.14
const MaxSize int = 100
`
    testRoundTrip(t, source)
}
```

### Helper Function

```go
func testRoundTrip(t *testing.T, source string) {
    // 1. Parse Go source
    fset := token.NewFileSet()
    file, err := parser.ParseFile(fset, "test.go", source, 0)
    require.NoError(t, err)

    // 2. Write to S-expression
    writer := NewWriter(fset)
    sexpText, err := writer.WriteProgram([]*ast.File{file})
    require.NoError(t, err)

    // 3. Parse S-expression
    sexpr := sexp.NewParser(sexpText)
    sexpNode, err := sexpr.Parse()
    require.NoError(t, err)

    // 4. Build back to AST
    builder := NewBuilder()
    fset2, files2, err := builder.BuildProgram(sexpNode)
    require.NoError(t, err)
    require.Len(t, files2, 1)

    // 5. Write both ASTs to Go source and compare
    var buf1, buf2 bytes.Buffer
    err = printer.Fprint(&buf1, fset, file)
    require.NoError(t, err)
    err = printer.Fprint(&buf2, fset2, files2[0])
    require.NoError(t, err)

    // Sources should be equivalent (may differ in formatting)
    assert.Equal(t, buf1.String(), buf2.String())
}
```

---

## Part 10: Pretty Printer Updates

Update `formStyles` in `sexp/pretty.go` to include new node types:

```go
var formStyles = map[string]FormStyle{
    // Existing...

    // Phase 2 Expressions
    "UnaryExpr":     StyleCompact,
    "BinaryExpr":    StyleCompact,
    "ParenExpr":     StyleCompact,
    "StarExpr":      StyleCompact,
    "IndexExpr":     StyleCompact,
    "SliceExpr":     StyleKeywordPairs,
    "KeyValueExpr":  StyleCompact,

    // Phase 2 Statements
    "ReturnStmt":    StyleKeywordPairs,
    "AssignStmt":    StyleKeywordPairs,
    "IncDecStmt":    StyleCompact,
    "BranchStmt":    StyleCompact,
    "DeferStmt":     StyleKeywordPairs,
    "GoStmt":        StyleKeywordPairs,
    "SendStmt":      StyleKeywordPairs,
    "EmptyStmt":     StyleCompact,
    "LabeledStmt":   StyleKeywordPairs,

    // Phase 2 Types
    "ArrayType":     StyleKeywordPairs,
    "MapType":       StyleKeywordPairs,
    "ChanType":      StyleKeywordPairs,

    // Phase 2 Specs
    "ValueSpec":     StyleKeywordPairs,
    "TypeSpec":      StyleKeywordPairs,
}
```

---

## Success Criteria

### Code Completeness

- [ ] All 25 nodes implemented in Builder
- [ ] All 25 nodes implemented in Writer
- [ ] All token mappings added
- [ ] Statement dispatcher updated
- [ ] Expression dispatcher updated
- [ ] Spec dispatcher updated
- [ ] Pretty printer updated

### Testing

- [ ] Unit tests for each new node (builder_test.go)
- [ ] Unit tests for each new node (writer_test.go)
- [ ] 6 integration tests passing
- [ ] Test coverage >90% for new code

### Documentation

- [ ] Update README with Phase 2 capabilities
- [ ] Add examples to documentation
- [ ] Update canonical S-expression format spec

### Validation

- [ ] Can parse programs with variables
- [ ] Can parse programs with operators
- [ ] Can parse programs with type definitions
- [ ] Can parse programs with channels/goroutines
- [ ] Round-trip tests pass for all integration tests

---

## Tips for Implementation

1. **Work in Order**: Implement expressions first, then statements, then types, then specs
2. **Test as You Go**: Write and run tests for each node before moving to the next
3. **Copy Patterns**: Use existing nodes (like `Ident`, `CallExpr`) as templates
4. **Check Positions**: Verify all position fields are preserved
5. **Handle Nil**: Test nil cases for optional fields
6. **Token Mapping**: Test both directions (parse and write) for all tokens
7. **Integration Early**: Run integration tests as soon as you have enough nodes

---

## Estimated Timeline

- **Part 1 (Expressions)**: 1 day
- **Part 2 (Statements)**: 1 day
- **Part 3-4 (Types & Specs)**: 0.5 days
- **Part 5-8 (Updates & Dispatchers)**: 0.5 days
- **Part 9 (Integration Tests)**: 0.5 days
- **Part 10 (Pretty Printer)**: 0.25 days
- **Buffer/Polish**: 0.25 days

**Total**: 3-4 days

---

## Next Steps After Phase 2

Once Phase 2 is complete and all tests pass:

1. **Update documentation** with new capabilities
2. **Create Phase 3 specifications** for control flow
3. **Celebrate** - you've expanded coverage significantly!

---

Good luck! This is the largest wave, but once complete, you'll have covered a huge portion of Go's AST nodes. The patterns established here will make the remaining waves much faster.
