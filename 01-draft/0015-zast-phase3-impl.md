---
number: 0015
title: Phase 3 Implementation Specification - Control Flow
author: Duncan McGreggor
created: 2025-10-04
updated: 2025-10-04
state: Draft
supersedes: None
superseded-by: None
---

# Phase 3 Implementation Specification - Control Flow

**Project**: zast  
**Phase**: 3 of 6  
**Goal**: Implement control flow constructs (9 nodes)  
**Estimated Effort**: 3-4 days  
**Prerequisites**: Phase 1 (basic nodes) and Phase 2 (Wave 1 - easy wins) complete

---

## Overview

Phase 3 adds support for all control flow constructs in Go. These nodes are moderately complex due to their multiple optional components and nested structures. After Phase 3, you'll be able to handle complete programs with conditionals, loops, switches, and channel operations.

**What you'll be able to handle after Phase 3**:
- If/else statements with optional init
- All forms of for loops (traditional, while-style, infinite)
- For-range loops over arrays, slices, maps, channels
- Switch statements with multiple cases
- Type switch statements
- Select statements for channel operations
- Declaration statements inside functions

---

## Implementation Checklist

### Statements (9 nodes)
- [ ] `IfStmt` - If/else statements
- [ ] `ForStmt` - For loops (all variants)
- [ ] `RangeStmt` - For-range loops
- [ ] `SwitchStmt` - Switch statements
- [ ] `TypeSwitchStmt` - Type switch statements
- [ ] `SelectStmt` - Select statements (channels)
- [ ] `CaseClause` - Cases in switch statements
- [ ] `CommClause` - Cases in select statements
- [ ] `DeclStmt` - Declaration statements in functions

---

## Part 1: Conditional Statements

### IfStmt

**Go AST Structure**:
```go
type IfStmt struct {
    If   token.Pos // position of "if" keyword
    Init Stmt      // initialization statement; or nil
    Cond Expr      // condition
    Body *BlockStmt
    Else Stmt      // else branch; or nil
}
```

**Canonical S-Expression Format**:
```lisp
(IfStmt
  :if <pos>
  :init <stmt-or-nil>
  :cond <expr>
  :body <BlockStmt>
  :else <stmt-or-nil>)
```

**Examples**:
```go
// Simple if
if x > 10 {
    return true
}

// (IfStmt
//   :if 10
//   :init nil
//   :cond (BinaryExpr :x (Ident...) :op GTR :y (BasicLit...))
//   :body (BlockStmt...)
//   :else nil)

// If with init
if x := getValue(); x > 0 {
    process(x)
}

// (IfStmt
//   :if 15
//   :init (AssignStmt :lhs (...) :tok DEFINE :rhs (...))
//   :cond (BinaryExpr...)
//   :body (BlockStmt...)
//   :else nil)

// If-else
if x > 10 {
    return true
} else {
    return false
}

// (IfStmt
//   :if 20
//   :init nil
//   :cond (BinaryExpr...)
//   :body (BlockStmt...)
//   :else (BlockStmt...))

// If-else-if chain
if x > 10 {
    return 1
} else if x > 5 {
    return 2
} else {
    return 3
}

// (IfStmt
//   :if 25
//   :init nil
//   :cond (BinaryExpr...)
//   :body (BlockStmt...)
//   :else (IfStmt...))  // Else is another IfStmt
```

**Builder Implementation** (`builder.go`):
```go
func (b *Builder) buildIfStmt(s sexp.SExp) (*ast.IfStmt, error) {
    list, ok := b.expectList(s, "IfStmt")
    if !ok {
        return nil, fmt.Errorf("not a list")
    }

    if !b.expectSymbol(list.Elements[0], "IfStmt") {
        return nil, fmt.Errorf("not an IfStmt node")
    }

    args := b.parseKeywordArgs(list.Elements)

    ifVal, ok := b.requireKeyword(args, "if", "IfStmt")
    if !ok {
        return nil, fmt.Errorf("missing if")
    }

    condVal, ok := b.requireKeyword(args, "cond", "IfStmt")
    if !ok {
        return nil, fmt.Errorf("missing cond")
    }

    bodyVal, ok := b.requireKeyword(args, "body", "IfStmt")
    if !ok {
        return nil, fmt.Errorf("missing body")
    }

    cond, err := b.buildExpr(condVal)
    if err != nil {
        return nil, fmt.Errorf("invalid cond: %v", err)
    }

    body, err := b.buildBlockStmt(bodyVal)
    if err != nil {
        return nil, fmt.Errorf("invalid body: %v", err)
    }

    // Optional init
    var init ast.Stmt
    if initVal, ok := args["init"]; ok && !b.parseNil(initVal) {
        init, err = b.buildStmt(initVal)
        if err != nil {
            return nil, fmt.Errorf("invalid init: %v", err)
        }
    }

    // Optional else
    var els ast.Stmt
    if elseVal, ok := args["else"]; ok && !b.parseNil(elseVal) {
        els, err = b.buildStmt(elseVal)
        if err != nil {
            return nil, fmt.Errorf("invalid else: %v", err)
        }
    }

    return &ast.IfStmt{
        If:   b.parsePos(ifVal),
        Init: init,
        Cond: cond,
        Body: body,
        Else: els,
    }, nil
}
```

**Writer Implementation** (`writer.go`):
```go
func (w *Writer) writeIfStmt(stmt *ast.IfStmt) error {
    w.openList()
    w.writeSymbol("IfStmt")
    w.writeSpace()
    w.writeKeyword("if")
    w.writeSpace()
    w.writePos(stmt.If)
    w.writeSpace()
    w.writeKeyword("init")
    w.writeSpace()
    if err := w.writeStmt(stmt.Init); err != nil {
        return err
    }
    w.writeSpace()
    w.writeKeyword("cond")
    w.writeSpace()
    if err := w.writeExpr(stmt.Cond); err != nil {
        return err
    }
    w.writeSpace()
    w.writeKeyword("body")
    w.writeSpace()
    if err := w.writeBlockStmt(stmt.Body); err != nil {
        return err
    }
    w.writeSpace()
    w.writeKeyword("else")
    w.writeSpace()
    if err := w.writeStmt(stmt.Else); err != nil {
        return err
    }
    w.closeList()
    return nil
}
```

**Tests**:
```go
func TestBuildIfStmt(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        hasInit  bool
        hasElse  bool
    }{
        {
            name: "simple if",
            input: `(IfStmt
                :if 10
                :init nil
                :cond (BinaryExpr :x (Ident :namepos 13 :name "x" :obj nil) :oppos 15 :op GTR :y (BasicLit :valuepos 17 :kind INT :value "10"))
                :body (BlockStmt :lbrace 20 :list () :rbrace 22)
                :else nil)`,
            hasInit: false,
            hasElse: false,
        },
        {
            name: "if with init",
            input: `(IfStmt
                :if 10
                :init (AssignStmt :lhs ((Ident :namepos 13 :name "x" :obj nil)) :tokpos 15 :tok DEFINE :rhs ((BasicLit :valuepos 18 :kind INT :value "5")))
                :cond (BinaryExpr :x (Ident :namepos 21 :name "x" :obj nil) :oppos 23 :op GTR :y (BasicLit :valuepos 25 :kind INT :value "0"))
                :body (BlockStmt :lbrace 28 :list () :rbrace 30)
                :else nil)`,
            hasInit: true,
            hasElse: false,
        },
        {
            name: "if-else",
            input: `(IfStmt
                :if 10
                :init nil
                :cond (Ident :namepos 13 :name "ready" :obj nil)
                :body (BlockStmt :lbrace 19 :list () :rbrace 21)
                :else (BlockStmt :lbrace 27 :list () :rbrace 29))`,
            hasInit: false,
            hasElse: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parser := sexp.NewParser(tt.input)
            sexpNode, err := parser.Parse()
            require.NoError(t, err)

            builder := NewBuilder()
            stmt, err := builder.buildIfStmt(sexpNode)
            require.NoError(t, err)

            if tt.hasInit {
                assert.NotNil(t, stmt.Init)
            } else {
                assert.Nil(t, stmt.Init)
            }

            if tt.hasElse {
                assert.NotNil(t, stmt.Else)
            } else {
                assert.Nil(t, stmt.Else)
            }

            assert.NotNil(t, stmt.Cond)
            assert.NotNil(t, stmt.Body)
        })
    }
}
```

---

## Part 2: Loop Statements

### ForStmt

**Go AST Structure**:
```go
type ForStmt struct {
    For  token.Pos // position of "for" keyword
    Init Stmt      // initialization statement; or nil
    Cond Expr      // condition; or nil
    Post Stmt      // post iteration statement; or nil
    Body *BlockStmt
}
```

**Canonical S-Expression Format**:
```lisp
(ForStmt
  :for <pos>
  :init <stmt-or-nil>
  :cond <expr-or-nil>
  :post <stmt-or-nil>
  :body <BlockStmt>)
```

**Examples**:
```go
// Traditional for loop
for i := 0; i < 10; i++ {
    process(i)
}

// (ForStmt
//   :for 10
//   :init (AssignStmt...)
//   :cond (BinaryExpr...)
//   :post (IncDecStmt...)
//   :body (BlockStmt...))

// While-style loop (condition only)
for x < 100 {
    x *= 2
}

// (ForStmt
//   :for 15
//   :init nil
//   :cond (BinaryExpr...)
//   :post nil
//   :body (BlockStmt...))

// Infinite loop (all nil)
for {
    doWork()
}

// (ForStmt
//   :for 20
//   :init nil
//   :cond nil
//   :post nil
//   :body (BlockStmt...))
```

**Implementation**: Similar to IfStmt with three optional components (init, cond, post) and one required (body).

---

### RangeStmt

**Go AST Structure**:
```go
type RangeStmt struct {
    For        token.Pos   // position of "for" keyword
    Key, Value Expr        // Key, Value may be nil
    TokPos     token.Pos   // position of Tok; invalid if Key == nil
    Tok        token.Token // ILLEGAL if Key == nil, ASSIGN, DEFINE
    X          Expr        // value to range over
    Body       *BlockStmt
}
```

**Canonical S-Expression Format**:
```lisp
(RangeStmt
  :for <pos>
  :key <expr-or-nil>
  :value <expr-or-nil>
  :tokpos <pos>
  :tok <token>
  :x <expr>
  :body <BlockStmt>)
```

**Examples**:
```go
// Range over slice with index and value
for i, v := range slice {
    process(i, v)
}

// (RangeStmt
//   :for 10
//   :key (Ident :namepos 14 :name "i" :obj nil)
//   :value (Ident :namepos 17 :name "v" :obj nil)
//   :tokpos 19
//   :tok DEFINE
//   :x (Ident :namepos 22 :name "slice" :obj nil)
//   :body (BlockStmt...))

// Range over map with key only
for k := range m {
    delete(m, k)
}

// (RangeStmt
//   :for 15
//   :key (Ident :namepos 19 :name "k" :obj nil)
//   :value nil
//   :tokpos 21
//   :tok DEFINE
//   :x (Ident :namepos 24 :name "m" :obj nil)
//   :body (BlockStmt...))

// Range without variables (just iteration)
for range slice {
    count++
}

// (RangeStmt
//   :for 20
//   :key nil
//   :value nil
//   :tokpos 0
//   :tok ILLEGAL
//   :x (Ident :namepos 26 :name "slice" :obj nil)
//   :body (BlockStmt...))
```

**Special Handling**: When Key is nil, Tok is ILLEGAL and TokPos is 0.

**Token Mapping** (add if not present):
- `ILLEGAL` → `token.ILLEGAL`

---

## Part 3: Switch Statements

### SwitchStmt

**Go AST Structure**:
```go
type SwitchStmt struct {
    Switch token.Pos  // position of "switch" keyword
    Init   Stmt       // initialization statement; or nil
    Tag    Expr       // tag expression; or nil
    Body   *BlockStmt // CaseClauses only
}
```

**Canonical S-Expression Format**:
```lisp
(SwitchStmt
  :switch <pos>
  :init <stmt-or-nil>
  :tag <expr-or-nil>
  :body <BlockStmt>)
```

**Examples**:
```go
// Switch with tag
switch x {
case 1:
    doOne()
case 2:
    doTwo()
default:
    doDefault()
}

// (SwitchStmt
//   :switch 10
//   :init nil
//   :tag (Ident :namepos 17 :name "x" :obj nil)
//   :body (BlockStmt
//           :lbrace 19
//           :list (
//             (CaseClause...)
//             (CaseClause...)
//             (CaseClause...))
//           :rbrace 50))

// Switch with init
switch x := getValue(); x {
case 0:
    return
}

// (SwitchStmt
//   :switch 15
//   :init (AssignStmt...)
//   :tag (Ident...)
//   :body (BlockStmt...))

// Type switch (tagless, true/false conditions)
switch {
case x > 10:
    big()
case x > 0:
    small()
}

// (SwitchStmt
//   :switch 20
//   :init nil
//   :tag nil
//   :body (BlockStmt...))
```

**Note**: Body contains only CaseClause statements. The BlockStmt wrapper is required by Go's AST.

---

### CaseClause

**Go AST Structure**:
```go
type CaseClause struct {
    Case  token.Pos // position of "case" or "default" keyword
    List  []Expr    // list of expressions or types; nil means default case
    Colon token.Pos // position of ":"
    Body  []Stmt    // statement list
}
```

**Canonical S-Expression Format**:
```lisp
(CaseClause
  :case <pos>
  :list (<expr> ...)
  :colon <pos>
  :body (<stmt> ...))
```

**Examples**:
```go
case 1, 2, 3:
    multi()

// (CaseClause
//   :case 10
//   :list (
//     (BasicLit :valuepos 15 :kind INT :value "1")
//     (BasicLit :valuepos 18 :kind INT :value "2")
//     (BasicLit :valuepos 21 :kind INT :value "3"))
//   :colon 22
//   :body ((ExprStmt...)))

default:
    doDefault()

// (CaseClause
//   :case 25
//   :list ()
//   :colon 32
//   :body ((ExprStmt...)))
```

**Note**: Empty list means default case.

---

### TypeSwitchStmt

**Go AST Structure**:
```go
type TypeSwitchStmt struct {
    Switch token.Pos  // position of "switch" keyword
    Init   Stmt       // initialization statement; or nil
    Assign Stmt       // x := y.(type) or y.(type)
    Body   *BlockStmt // CaseClauses only
}
```

**Canonical S-Expression Format**:
```lisp
(TypeSwitchStmt
  :switch <pos>
  :init <stmt-or-nil>
  :assign <stmt>
  :body <BlockStmt>)
```

**Example**:
```go
switch v := x.(type) {
case int:
    processInt(v)
case string:
    processString(v)
default:
    processOther()
}

// (TypeSwitchStmt
//   :switch 10
//   :init nil
//   :assign (AssignStmt
//             :lhs ((Ident :namepos 17 :name "v" :obj nil))
//             :tokpos 19
//             :tok DEFINE
//             :rhs ((TypeAssertExpr :x (Ident...) :lparen 24 :type nil :rparen 29)))
//   :body (BlockStmt
//           :list (
//             (CaseClause
//               :list ((Ident :namepos 40 :name "int" :obj nil))
//               ...)
//             ...)))
```

**Note**: Assign statement is typically an AssignStmt with a TypeAssertExpr on the right-hand side. The TypeAssertExpr has nil Type for `x.(type)`.

---

## Part 4: Select Statements

### SelectStmt

**Go AST Structure**:
```go
type SelectStmt struct {
    Select token.Pos  // position of "select" keyword
    Body   *BlockStmt // CommClauses only
}
```

**Canonical S-Expression Format**:
```lisp
(SelectStmt
  :select <pos>
  :body <BlockStmt>)
```

**Example**:
```go
select {
case msg := <-ch1:
    process(msg)
case ch2 <- value:
    sent()
default:
    timeout()
}

// (SelectStmt
//   :select 10
//   :body (BlockStmt
//           :lbrace 17
//           :list (
//             (CommClause...)
//             (CommClause...)
//             (CommClause...))
//           :rbrace 80))
```

---

### CommClause

**Go AST Structure**:
```go
type CommClause struct {
    Case  token.Pos // position of "case" or "default" keyword
    Comm  Stmt      // send or receive statement; nil means default case
    Colon token.Pos // position of ":"
    Body  []Stmt    // statement list
}
```

**Canonical S-Expression Format**:
```lisp
(CommClause
  :case <pos>
  :comm <stmt-or-nil>
  :colon <pos>
  :body (<stmt> ...))
```

**Examples**:
```go
case msg := <-ch:
    process(msg)

// (CommClause
//   :case 10
//   :comm (AssignStmt
//           :lhs ((Ident :namepos 15 :name "msg" :obj nil))
//           :tokpos 19
//           :tok DEFINE
//           :rhs ((UnaryExpr :oppos 22 :op ARROW :x (Ident :namepos 24 :name "ch" :obj nil))))
//   :colon 26
//   :body ((ExprStmt...)))

case ch <- value:
    sent()

// (CommClause
//   :case 30
//   :comm (SendStmt
//           :chan (Ident :namepos 35 :name "ch" :obj nil)
//           :arrow 38
//           :value (Ident :namepos 41 :name "value" :obj nil))
//   :colon 46
//   :body ((ExprStmt...)))

default:
    timeout()

// (CommClause
//   :case 50
//   :comm nil
//   :colon 57
//   :body ((ExprStmt...)))
```

**Note**: Comm is nil for default case. For receive, it's usually an AssignStmt with UnaryExpr (ARROW op). For send, it's a SendStmt.

---

## Part 5: Declaration Statements

### DeclStmt

**Go AST Structure**:
```go
type DeclStmt struct {
    Decl Decl // *GenDecl with CONST, TYPE, or VAR token
}
```

**Canonical S-Expression Format**:
```lisp
(DeclStmt
  :decl <Decl>)
```

**Examples**:
```go
func process() {
    var x int = 10
    const Pi = 3.14
    type MyInt int
}

// Inside the function body:
// (DeclStmt
//   :decl (GenDecl
//           :tok VAR
//           :specs ((ValueSpec...))))

// (DeclStmt
//   :decl (GenDecl
//           :tok CONST
//           :specs ((ValueSpec...))))

// (DeclStmt
//   :decl (GenDecl
//           :tok TYPE
//           :specs ((TypeSpec...))))
```

**Note**: DeclStmt is only used for declarations inside function bodies. Top-level declarations use GenDecl directly.

**Implementation**:
```go
func (b *Builder) buildDeclStmt(s sexp.SExp) (*ast.DeclStmt, error) {
    list, ok := b.expectList(s, "DeclStmt")
    if !ok {
        return nil, fmt.Errorf("not a list")
    }

    if !b.expectSymbol(list.Elements[0], "DeclStmt") {
        return nil, fmt.Errorf("not a DeclStmt node")
    }

    args := b.parseKeywordArgs(list.Elements)

    declVal, ok := b.requireKeyword(args, "decl", "DeclStmt")
    if !ok {
        return nil, fmt.Errorf("missing decl")
    }

    decl, err := b.buildDecl(declVal)
    if err != nil {
        return nil, fmt.Errorf("invalid decl: %v", err)
    }

    return &ast.DeclStmt{
        Decl: decl,
    }, nil
}

func (w *Writer) writeDeclStmt(stmt *ast.DeclStmt) error {
    w.openList()
    w.writeSymbol("DeclStmt")
    w.writeSpace()
    w.writeKeyword("decl")
    w.writeSpace()
    if err := w.writeDecl(stmt.Decl); err != nil {
        return err
    }
    w.closeList()
    return nil
}
```

---

## Part 6: Dispatcher Updates

### Update Statement Dispatcher

Add all Phase 3 nodes to `buildStmt` in `builder.go`:

```go
func (b *Builder) buildStmt(s sexp.SExp) (ast.Stmt, error) {
    // ... existing code ...

    switch sym.Value {
    // Phase 1 nodes
    case "ExprStmt":
        return b.buildExprStmt(s)
    case "BlockStmt":
        return b.buildBlockStmt(s)
    
    // Phase 2 nodes (Wave 1)
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
    
    // Phase 3 nodes
    case "IfStmt":
        return b.buildIfStmt(s)
    case "ForStmt":
        return b.buildForStmt(s)
    case "RangeStmt":
        return b.buildRangeStmt(s)
    case "SwitchStmt":
        return b.buildSwitchStmt(s)
    case "TypeSwitchStmt":
        return b.buildTypeSwitchStmt(s)
    case "SelectStmt":
        return b.buildSelectStmt(s)
    case "CaseClause":
        return b.buildCaseClause(s)
    case "CommClause":
        return b.buildCommClause(s)
    case "DeclStmt":
        return b.buildDeclStmt(s)
    
    default:
        return nil, fmt.Errorf("unknown statement type: %s", sym.Value)
    }
}
```

Add to `writeStmt` in `writer.go`:

```go
func (w *Writer) writeStmt(stmt ast.Stmt) error {
    if stmt == nil {
        w.writeSymbol("nil")
        return nil
    }

    switch s := stmt.(type) {
    // Phase 1 nodes
    case *ast.ExprStmt:
        return w.writeExprStmt(s)
    case *ast.BlockStmt:
        return w.writeBlockStmt(s)
    
    // Phase 2 nodes (Wave 1)
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
    
    // Phase 3 nodes
    case *ast.IfStmt:
        return w.writeIfStmt(s)
    case *ast.ForStmt:
        return w.writeForStmt(s)
    case *ast.RangeStmt:
        return w.writeRangeStmt(s)
    case *ast.SwitchStmt:
        return w.writeSwitchStmt(s)
    case *ast.TypeSwitchStmt:
        return w.writeTypeSwitchStmt(s)
    case *ast.SelectStmt:
        return w.writeSelectStmt(s)
    case *ast.CaseClause:
        return w.writeCaseClause(s)
    case *ast.CommClause:
        return w.writeCommClause(s)
    case *ast.DeclStmt:
        return w.writeDeclStmt(s)
    
    default:
        return fmt.Errorf("unknown statement type: %T", stmt)
    }
}
```

---

## Part 7: Integration Tests

### Test 1: If Statements

```go
func TestPhase3IfStatements(t *testing.T) {
    source := `package main

func check(x int) bool {
    if x > 10 {
        return true
    } else if x > 0 {
        return false
    } else {
        return false
    }
}

func conditional() {
    if x := getValue(); x != nil {
        process(x)
    }
}
`
    testRoundTrip(t, source)
}
```

### Test 2: For Loops

```go
func TestPhase3ForLoops(t *testing.T) {
    source := `package main

func loops() {
    // Traditional for
    for i := 0; i < 10; i++ {
        process(i)
    }
    
    // While-style
    x := 0
    for x < 100 {
        x *= 2
    }
    
    // Infinite loop
    for {
        if done() {
            break
        }
        work()
    }
}
`
    testRoundTrip(t, source)
}
```

### Test 3: Range Loops

```go
func TestPhase3RangeLoops(t *testing.T) {
    source := `package main

func rangeExamples() {
    slice := []int{1, 2, 3}
    m := map[string]int{"a": 1}
    
    // Index and value
    for i, v := range slice {
        process(i, v)
    }
    
    // Index only
    for i := range slice {
        update(i)
    }
    
    // Value only (ignore index with _)
    for _, v := range slice {
        use(v)
    }
    
    // Just iteration
    for range slice {
        count++
    }
    
    // Map iteration
    for k, v := range m {
        store(k, v)
    }
}
`
    testRoundTrip(t, source)
}
```

### Test 4: Switch Statements

```go
func TestPhase3SwitchStatements(t *testing.T) {
    source := `package main

func switches(x int) {
    // Simple switch
    switch x {
    case 1:
        one()
    case 2, 3:
        twoOrThree()
    default:
        other()
    }
    
    // Switch with init
    switch y := compute(); y {
    case 0:
        zero()
    }
    
    // Expression switch (tagless)
    switch {
    case x > 10:
        big()
    case x > 0:
        small()
    default:
        negative()
    }
}
`
    testRoundTrip(t, source)
}
```

### Test 5: Type Switch

```go
func TestPhase3TypeSwitch(t *testing.T) {
    source := `package main

func typeSwitch(x interface{}) {
    switch v := x.(type) {
    case int:
        processInt(v)
    case string:
        processString(v)
    case nil:
        handleNil()
    default:
        unknown(v)
    }
    
    // Without assignment
    switch x.(type) {
    case bool:
        handleBool()
    }
}
`
    testRoundTrip(t, source)
}
```

### Test 6: Select Statements

```go
func TestPhase3SelectStatements(t *testing.T) {
    source := `package main

func selectExample(ch1, ch2 chan int) {
    select {
    case msg := <-ch1:
        process(msg)
    case ch2 <- 42:
        sent()
    default:
        timeout()
    }
    
    // Blocking select
    select {
    case v := <-ch1:
        handle(v)
    case <-ch2:
        signal()
    }
}
`
    testRoundTrip(t, source)
}
```

### Test 7: Declaration Statements

```go
func TestPhase3DeclStmt(t *testing.T) {
    source := `package main

func declarations() {
    var x int
    var y, z = 1, 2
    const Pi = 3.14
    type MyInt int
    
    if true {
        var local string
        const LocalConst = 100
    }
}
`
    testRoundTrip(t, source)
}
```

### Test 8: Complex Control Flow

```go
func TestPhase3ComplexControlFlow(t *testing.T) {
    source := `package main

func fibonacci(n int) int {
    if n <= 1 {
        return n
    }
    
    a, b := 0, 1
    for i := 2; i <= n; i++ {
        a, b = b, a+b
    }
    return b
}

func processItems(items []string) {
    for i, item := range items {
        switch {
        case len(item) == 0:
            continue
        case len(item) > 100:
            if i == 0 {
                return
            }
            break
        default:
            process(item)
        }
    }
}
`
    testRoundTrip(t, source)
}
```

---

## Part 8: Pretty Printer Updates

Update `formStyles` in `sexp/pretty.go`:

```go
var formStyles = map[string]FormStyle{
    // Existing...
    
    // Phase 3 Control Flow
    "IfStmt":          StyleKeywordPairs,
    "ForStmt":         StyleKeywordPairs,
    "RangeStmt":       StyleKeywordPairs,
    "SwitchStmt":      StyleKeywordPairs,
    "TypeSwitchStmt":  StyleKeywordPairs,
    "SelectStmt":      StyleKeywordPairs,
    "CaseClause":      StyleKeywordPairs,
    "CommClause":      StyleKeywordPairs,
    "DeclStmt":        StyleKeywordPairs,
}
```

---

## Part 9: Special Considerations

### CaseClause vs CommClause

Both have similar structures but are used in different contexts:
- `CaseClause` used in `SwitchStmt` and `TypeSwitchStmt`
- `CommClause` used in `SelectStmt`

Make sure to handle them separately in statement dispatchers.

### Type Assertions in Type Switch

In `TypeSwitchStmt`, the Assign field contains a statement (usually `AssignStmt`) where the RHS contains a `TypeAssertExpr` with `Type` field set to `nil` (representing `x.(type)`).

Phase 3 doesn't implement `TypeAssertExpr` yet (that's in a later phase), so for now, type switches will have limited support. Document this limitation.

**Alternative**: Implement a minimal `TypeAssertExpr` for Phase 3:

```go
type TypeAssertExpr struct {
    X      Expr      // expression
    Lparen token.Pos // position of "("
    Type   Expr      // asserted type; nil means type switch x.(type)
    Rparen token.Pos // position of ")"
}
```

This allows full type switch support in Phase 3.

---

## Success Criteria

### Code Completeness
- [ ] All 9 Phase 3 nodes implemented in Builder
- [ ] All 9 Phase 3 nodes implemented in Writer
- [ ] Statement dispatchers updated
- [ ] Pretty printer updated
- [ ] Optional: TypeAssertExpr implemented for type switch support

### Testing
- [ ] Unit tests for each node (builder_test.go)
- [ ] Unit tests for each node (writer_test.go)
- [ ] 8 integration tests passing
- [ ] Test coverage >90% for new code

### Documentation
- [ ] Update README with Phase 3 capabilities
- [ ] Document any limitations (e.g., TypeAssertExpr if not implemented)
- [ ] Add examples to documentation

### Validation
- [ ] Can parse programs with if/else
- [ ] Can parse programs with all loop types
- [ ] Can parse programs with switch statements
- [ ] Can parse programs with select statements
- [ ] Round-trip tests pass for all integration tests
- [ ] Can compile and run fibonacci, FizzBuzz, etc.

---

## Implementation Tips

1. **Start with IfStmt and ForStmt**: These are the most common and well-understood
2. **Test Optional Fields**: Many fields are optional (init, else, cond, etc.) - test nil cases
3. **Handle Empty Lists**: CaseClause and CommClause can have empty lists (default cases)
4. **Test Nesting**: If-else chains, nested loops, switch inside loops
5. **Position Tracking**: Verify all position fields are preserved
6. **TypeAssertExpr Decision**: Decide early whether to implement it in Phase 3 or defer

---

## Estimated Timeline

- **Part 1 (IfStmt)**: 0.5 days
- **Part 2 (ForStmt, RangeStmt)**: 1 day
- **Part 3 (Switch statements)**: 1 day
- **Part 4 (Select statements)**: 0.5 days
- **Part 5 (DeclStmt)**: 0.25 days
- **Part 6-8 (Updates & Tests)**: 0.75 days

**Total**: 3-4 days

---

## Next Steps After Phase 3

Once Phase 3 is complete and all tests pass:

1. **Update documentation** with control flow examples
2. **Create Phase 4 specifications** for complex types (structs, interfaces, composites)
3. **Test with real programs**: Implement classic algorithms (sorting, searching, etc.)
4. **Celebrate** - You can now handle most practical Go programs!

---

Phase 3 is where zast becomes truly useful. With control flow complete, you can parse and transform substantial Go programs. Good luck!
