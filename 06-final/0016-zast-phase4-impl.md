---
number: 0016
title: Phase 4 Implementation Specification - Complex Types
author: Duncan McGreggor
created: 2025-10-04
updated: 2025-10-04
state: Final
supersedes: None
superseded-by: None
---

# Phase 4 Implementation Specification - Complex Types

**Project**: zast  
**Phase**: 4 of 6  
**Goal**: Implement complex type nodes (5 nodes)  
**Estimated Effort**: 2-3 days  
**Prerequisites**: Phase 1 (basic nodes), Phase 2 (easy wins), Phase 3 (control flow) complete

---

## Overview

Phase 4 adds support for Go's complex composite types and type operations. These nodes are the most intricate because they involve nested structures, recursive definitions, and sophisticated type semantics. After Phase 4, you'll be able to handle full OOP patterns, closures, and complex data structures.

**What you'll be able to handle after Phase 4**:
- Struct definitions with tags and embedded fields
- Interface definitions with methods and embedded interfaces
- Composite literals (struct, array, slice, map initialization)
- Type assertions and type testing
- Function literals (closures and anonymous functions)
- Ellipsis for variadic parameters

---

## Implementation Checklist

### Type Nodes (2 nodes)
- [ ] `StructType` (full implementation) - Struct definitions
- [ ] `InterfaceType` (full implementation) - Interface definitions

### Expression Nodes (3 nodes)
- [ ] `CompositeLit` - Composite literals
- [ ] `TypeAssertExpr` - Type assertions
- [ ] `FuncLit` - Function literals (closures)

### Supporting Node (1 node)
- [ ] `Ellipsis` - Variadic parameters (...)

---

## Part 1: Struct Types

### StructType

**Go AST Structure**:
```go
type StructType struct {
    Struct     token.Pos  // position of "struct" keyword
    Fields     *FieldList // list of field declarations
    Incomplete bool       // true if (source) fields are missing in the Fields list
}
```

**Canonical S-Expression Format**:
```lisp
(StructType
  :struct <pos>
  :fields <FieldList>
  :incomplete <bool>)
```

**Examples**:

```go
// Simple struct
type Point struct {
    X int
    Y int
}

// (StructType
//   :struct 10
//   :fields (FieldList
//             :opening 17
//             :list (
//               (Field
//                 :doc nil
//                 :names ((Ident :namepos 23 :name "X" :obj nil))
//                 :type (Ident :namepos 25 :name "int" :obj nil)
//                 :tag nil
//                 :comment nil)
//               (Field
//                 :doc nil
//                 :names ((Ident :namepos 33 :name "Y" :obj nil))
//                 :type (Ident :namepos 35 :name "int" :obj nil)
//                 :tag nil
//                 :comment nil))
//             :closing 40)
//   :incomplete false)

// Struct with tags
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

// (StructType
//   :struct 10
//   :fields (FieldList
//             :opening 17
//             :list (
//               (Field
//                 :doc nil
//                 :names ((Ident :namepos 23 :name "ID" :obj nil))
//                 :type (Ident :namepos 28 :name "int" :obj nil)
//                 :tag (BasicLit :valuepos 32 :kind STRING :value "`json:\"id\"`")
//                 :comment nil)
//               (Field
//                 :doc nil
//                 :names ((Ident :namepos 50 :name "Name" :obj nil))
//                 :type (Ident :namepos 55 :name "string" :obj nil)
//                 :tag (BasicLit :valuepos 62 :kind STRING :value "`json:\"name\"`")
//                 :comment nil))
//             :closing 80)
//   :incomplete false)

// Embedded struct
type Employee struct {
    Person  // embedded field (no field name)
    ID int
}

// (StructType
//   :struct 10
//   :fields (FieldList
//             :opening 20
//             :list (
//               (Field
//                 :doc nil
//                 :names ()  // Empty names = embedded field
//                 :type (Ident :namepos 26 :name "Person" :obj nil)
//                 :tag nil
//                 :comment nil)
//               (Field
//                 :doc nil
//                 :names ((Ident :namepos 37 :name "ID" :obj nil))
//                 :type (Ident :namepos 40 :name "int" :obj nil)
//                 :tag nil
//                 :comment nil))
//             :closing 44)
//   :incomplete false)

// Anonymous struct (empty)
var config struct{}

// (StructType
//   :struct 15
//   :fields (FieldList :opening 22 :list () :closing 23)
//   :incomplete false)
```

**Implementation** (`builder.go`):

```go
func (b *Builder) buildStructType(s sexp.SExp) (*ast.StructType, error) {
    list, ok := b.expectList(s, "StructType")
    if !ok {
        return nil, fmt.Errorf("not a list")
    }

    if !b.expectSymbol(list.Elements[0], "StructType") {
        return nil, fmt.Errorf("not a StructType node")
    }

    args := b.parseKeywordArgs(list.Elements)

    structVal, ok := b.requireKeyword(args, "struct", "StructType")
    if !ok {
        return nil, fmt.Errorf("missing struct")
    }

    fieldsVal, ok := b.requireKeyword(args, "fields", "StructType")
    if !ok {
        return nil, fmt.Errorf("missing fields")
    }

    incompleteVal, ok := b.requireKeyword(args, "incomplete", "StructType")
    if !ok {
        return nil, fmt.Errorf("missing incomplete")
    }

    fields, err := b.buildFieldList(fieldsVal)
    if err != nil {
        return nil, fmt.Errorf("invalid fields: %v", err)
    }

    incomplete, err := b.parseBool(incompleteVal)
    if err != nil {
        return nil, fmt.Errorf("invalid incomplete: %v", err)
    }

    return &ast.StructType{
        Struct:     b.parsePos(structVal),
        Fields:     fields,
        Incomplete: incomplete,
    }, nil
}
```

**Implementation** (`writer.go`):

```go
func (w *Writer) writeStructType(typ *ast.StructType) error {
    w.openList()
    w.writeSymbol("StructType")
    w.writeSpace()
    w.writeKeyword("struct")
    w.writeSpace()
    w.writePos(typ.Struct)
    w.writeSpace()
    w.writeKeyword("fields")
    w.writeSpace()
    if err := w.writeFieldList(typ.Fields); err != nil {
        return err
    }
    w.writeSpace()
    w.writeKeyword("incomplete")
    w.writeSpace()
    w.writeBool(typ.Incomplete)
    w.closeList()
    return nil
}
```

**Note on Field**: The Field node already exists from Phase 1, but ensure it properly handles:
- Empty `Names` list for embedded fields
- Non-nil `Tag` for struct tags
- The Tag field is a `*BasicLit` with STRING kind

**Tests**:

```go
func TestBuildStructType(t *testing.T) {
    tests := []struct {
        name       string
        input      string
        numFields  int
        hasTag     bool
        isEmbedded bool
    }{
        {
            name: "simple struct",
            input: `(StructType
                :struct 10
                :fields (FieldList
                    :opening 17
                    :list (
                        (Field :doc nil :names ((Ident :namepos 23 :name "X" :obj nil)) :type (Ident :namepos 25 :name "int" :obj nil) :tag nil :comment nil)
                        (Field :doc nil :names ((Ident :namepos 33 :name "Y" :obj nil)) :type (Ident :namepos 35 :name "int" :obj nil) :tag nil :comment nil))
                    :closing 40)
                :incomplete false)`,
            numFields:  2,
            hasTag:     false,
            isEmbedded: false,
        },
        {
            name: "struct with tag",
            input: `(StructType
                :struct 10
                :fields (FieldList
                    :opening 17
                    :list (
                        (Field :doc nil :names ((Ident :namepos 23 :name "ID" :obj nil)) :type (Ident :namepos 26 :name "int" :obj nil) :tag (BasicLit :valuepos 30 :kind STRING :value "\`json:\"id\"\`") :comment nil))
                    :closing 50)
                :incomplete false)`,
            numFields:  1,
            hasTag:     true,
            isEmbedded: false,
        },
        {
            name: "embedded field",
            input: `(StructType
                :struct 10
                :fields (FieldList
                    :opening 20
                    :list (
                        (Field :doc nil :names () :type (Ident :namepos 26 :name "Person" :obj nil) :tag nil :comment nil))
                    :closing 35)
                :incomplete false)`,
            numFields:  1,
            hasTag:     false,
            isEmbedded: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parser := sexp.NewParser(tt.input)
            sexpNode, err := parser.Parse()
            require.NoError(t, err)

            builder := NewBuilder()
            structType, err := builder.buildStructType(sexpNode)
            require.NoError(t, err)

            assert.Equal(t, tt.numFields, len(structType.Fields.List))
            
            if tt.hasTag {
                assert.NotNil(t, structType.Fields.List[0].Tag)
            }
            
            if tt.isEmbedded {
                assert.Equal(t, 0, len(structType.Fields.List[0].Names))
            }
        })
    }
}
```

---

## Part 2: Interface Types

### InterfaceType

**Go AST Structure**:
```go
type InterfaceType struct {
    Interface  token.Pos  // position of "interface" keyword
    Methods    *FieldList // list of methods
    Incomplete bool       // true if (source) methods are missing in the Methods list
}
```

**Canonical S-Expression Format**:
```lisp
(InterfaceType
  :interface <pos>
  :methods <FieldList>
  :incomplete <bool>)
```

**Examples**:

```go
// Simple interface
type Reader interface {
    Read(p []byte) (n int, err error)
}

// (InterfaceType
//   :interface 10
//   :methods (FieldList
//             :opening 20
//             :list (
//               (Field
//                 :doc nil
//                 :names ((Ident :namepos 26 :name "Read" :obj nil))
//                 :type (FuncType
//                         :func 0
//                         :params (FieldList
//                                   :opening 30
//                                   :list ((Field :doc nil :names ((Ident :namepos 31 :name "p" :obj nil)) :type (ArrayType :lbrack 33 :len nil :elt (Ident :namepos 35 :name "byte" :obj nil)) :tag nil :comment nil))
//                                   :closing 39)
//                         :results (FieldList
//                                    :opening 41
//                                    :list (
//                                      (Field :doc nil :names ((Ident :namepos 42 :name "n" :obj nil)) :type (Ident :namepos 44 :name "int" :obj nil) :tag nil :comment nil)
//                                      (Field :doc nil :names ((Ident :namepos 49 :name "err" :obj nil)) :type (Ident :namepos 53 :name "error" :obj nil) :tag nil :comment nil))
//                                    :closing 58))
//                 :tag nil
//                 :comment nil))
//             :closing 60)
//   :incomplete false)

// Embedded interface
type ReadWriter interface {
    Reader  // embedded interface
    Writer
}

// (InterfaceType
//   :interface 10
//   :methods (FieldList
//             :opening 21
//             :list (
//               (Field :doc nil :names () :type (Ident :namepos 27 :name "Reader" :obj nil) :tag nil :comment nil)
//               (Field :doc nil :names () :type (Ident :namepos 38 :name "Writer" :obj nil) :tag nil :comment nil))
//             :closing 45)
//   :incomplete false)

// Empty interface
interface{}

// (InterfaceType
//   :interface 10
//   :methods (FieldList :opening 20 :list () :closing 21)
//   :incomplete false)
```

**Note**: 
- Method fields have Names (the method name) and Type (FuncType)
- Embedded interfaces have empty Names and Type is an Ident (interface name)
- For Go 1.18+ generics, interfaces can also have type constraints, but those are advanced

**Implementation**: Very similar to StructType, just different field names.

```go
func (b *Builder) buildInterfaceType(s sexp.SExp) (*ast.InterfaceType, error) {
    list, ok := b.expectList(s, "InterfaceType")
    if !ok {
        return nil, fmt.Errorf("not a list")
    }

    if !b.expectSymbol(list.Elements[0], "InterfaceType") {
        return nil, fmt.Errorf("not an InterfaceType node")
    }

    args := b.parseKeywordArgs(list.Elements)

    interfaceVal, ok := b.requireKeyword(args, "interface", "InterfaceType")
    if !ok {
        return nil, fmt.Errorf("missing interface")
    }

    methodsVal, ok := b.requireKeyword(args, "methods", "InterfaceType")
    if !ok {
        return nil, fmt.Errorf("missing methods")
    }

    incompleteVal, ok := b.requireKeyword(args, "incomplete", "InterfaceType")
    if !ok {
        return nil, fmt.Errorf("missing incomplete")
    }

    methods, err := b.buildFieldList(methodsVal)
    if err != nil {
        return nil, fmt.Errorf("invalid methods: %v", err)
    }

    incomplete, err := b.parseBool(incompleteVal)
    if err != nil {
        return nil, fmt.Errorf("invalid incomplete: %v", err)
    }

    return &ast.InterfaceType{
        Interface:  b.parsePos(interfaceVal),
        Methods:    methods,
        Incomplete: incomplete,
    }, nil
}

func (w *Writer) writeInterfaceType(typ *ast.InterfaceType) error {
    w.openList()
    w.writeSymbol("InterfaceType")
    w.writeSpace()
    w.writeKeyword("interface")
    w.writeSpace()
    w.writePos(typ.Interface)
    w.writeSpace()
    w.writeKeyword("methods")
    w.writeSpace()
    if err := w.writeFieldList(typ.Methods); err != nil {
        return err
    }
    w.writeSpace()
    w.writeKeyword("incomplete")
    w.writeSpace()
    w.writeBool(typ.Incomplete)
    w.closeList()
    return nil
}
```

---

## Part 3: Composite Literals

### CompositeLit

**Go AST Structure**:
```go
type CompositeLit struct {
    Type       Expr      // literal type; or nil
    Lbrace     token.Pos // position of "{"
    Elts       []Expr    // list of composite elements; or nil
    Rbrace     token.Pos // position of "}"
    Incomplete bool      // true if (source) expressions are missing in the Elts list
}
```

**Canonical S-Expression Format**:
```lisp
(CompositeLit
  :type <expr-or-nil>
  :lbrace <pos>
  :elts (<expr> ...)
  :rbrace <pos>
  :incomplete <bool>)
```

**Examples**:

```go
// Struct literal
Point{X: 1, Y: 2}

// (CompositeLit
//   :type (Ident :namepos 10 :name "Point" :obj nil)
//   :lbrace 15
//   :elts (
//     (KeyValueExpr
//       :key (Ident :namepos 16 :name "X" :obj nil)
//       :colon 17
//       :value (BasicLit :valuepos 19 :kind INT :value "1"))
//     (KeyValueExpr
//       :key (Ident :namepos 22 :name "Y" :obj nil)
//       :colon 23
//       :value (BasicLit :valuepos 25 :kind INT :value "2")))
//   :rbrace 26
//   :incomplete false)

// Array literal
[3]int{1, 2, 3}

// (CompositeLit
//   :type (ArrayType :lbrack 10 :len (BasicLit :valuepos 11 :kind INT :value "3") :elt (Ident :namepos 13 :name "int" :obj nil))
//   :lbrace 16
//   :elts (
//     (BasicLit :valuepos 17 :kind INT :value "1")
//     (BasicLit :valuepos 20 :kind INT :value "2")
//     (BasicLit :valuepos 23 :kind INT :value "3"))
//   :rbrace 24
//   :incomplete false)

// Slice literal
[]string{"a", "b"}

// (CompositeLit
//   :type (ArrayType :lbrack 10 :len nil :elt (Ident :namepos 12 :name "string" :obj nil))
//   :lbrace 18
//   :elts (
//     (BasicLit :valuepos 19 :kind STRING :value "\"a\"")
//     (BasicLit :valuepos 24 :kind STRING :value "\"b\""))
//   :rbrace 27
//   :incomplete false)

// Map literal
map[string]int{"x": 1, "y": 2}

// (CompositeLit
//   :type (MapType :map 10 :key (Ident :namepos 14 :name "string" :obj nil) :value (Ident :namepos 21 :name "int" :obj nil))
//   :lbrace 24
//   :elts (
//     (KeyValueExpr
//       :key (BasicLit :valuepos 25 :kind STRING :value "\"x\"")
//       :colon 28
//       :value (BasicLit :valuepos 30 :kind INT :value "1"))
//     (KeyValueExpr
//       :key (BasicLit :valuepos 33 :kind STRING :value "\"y\"")
//       :colon 36
//       :value (BasicLit :valuepos 38 :kind INT :value "2")))
//   :rbrace 39
//   :incomplete false)

// Type-inferred literal (Type is nil)
x := Point{1, 2}  // when Point type is inferred

// (CompositeLit
//   :type nil
//   :lbrace 15
//   :elts (
//     (BasicLit :valuepos 16 :kind INT :value "1")
//     (BasicLit :valuepos 19 :kind INT :value "2"))
//   :rbrace 20
//   :incomplete false)

// Nested composite literal
[]Point{{X: 1}, {X: 2}}

// (CompositeLit
//   :type (ArrayType :lbrack 10 :len nil :elt (Ident :namepos 12 :name "Point" :obj nil))
//   :lbrace 17
//   :elts (
//     (CompositeLit
//       :type nil
//       :lbrace 18
//       :elts ((KeyValueExpr :key (Ident :namepos 19 :name "X" :obj nil) :colon 20 :value (BasicLit :valuepos 22 :kind INT :value "1")))
//       :rbrace 23
//       :incomplete false)
//     (CompositeLit
//       :type nil
//       :lbrace 26
//       :elts ((KeyValueExpr :key (Ident :namepos 27 :name "X" :obj nil) :colon 28 :value (BasicLit :valuepos 30 :kind INT :value "2")))
//       :rbrace 31
//       :incomplete false))
//   :rbrace 32
//   :incomplete false)
```

**Implementation**:

```go
func (b *Builder) buildCompositeLit(s sexp.SExp) (*ast.CompositeLit, error) {
    list, ok := b.expectList(s, "CompositeLit")
    if !ok {
        return nil, fmt.Errorf("not a list")
    }

    if !b.expectSymbol(list.Elements[0], "CompositeLit") {
        return nil, fmt.Errorf("not a CompositeLit node")
    }

    args := b.parseKeywordArgs(list.Elements)

    lbraceVal, ok := b.requireKeyword(args, "lbrace", "CompositeLit")
    if !ok {
        return nil, fmt.Errorf("missing lbrace")
    }

    eltsVal, ok := b.requireKeyword(args, "elts", "CompositeLit")
    if !ok {
        return nil, fmt.Errorf("missing elts")
    }

    rbraceVal, ok := b.requireKeyword(args, "rbrace", "CompositeLit")
    if !ok {
        return nil, fmt.Errorf("missing rbrace")
    }

    incompleteVal, ok := b.requireKeyword(args, "incomplete", "CompositeLit")
    if !ok {
        return nil, fmt.Errorf("missing incomplete")
    }

    // Optional type
    var typ ast.Expr
    var err error
    if typeVal, ok := args["type"]; ok && !b.parseNil(typeVal) {
        typ, err = b.buildExpr(typeVal)
        if err != nil {
            return nil, fmt.Errorf("invalid type: %v", err)
        }
    }

    // Build elements list
    var elts []ast.Expr
    eltsList, ok := b.expectList(eltsVal, "CompositeLit elts")
    if ok {
        for _, eltSexp := range eltsList.Elements {
            elt, err := b.buildExpr(eltSexp)
            if err != nil {
                return nil, fmt.Errorf("invalid element: %v", err)
            }
            elts = append(elts, elt)
        }
    }

    incomplete, err := b.parseBool(incompleteVal)
    if err != nil {
        return nil, fmt.Errorf("invalid incomplete: %v", err)
    }

    return &ast.CompositeLit{
        Type:       typ,
        Lbrace:     b.parsePos(lbraceVal),
        Elts:       elts,
        Rbrace:     b.parsePos(rbraceVal),
        Incomplete: incomplete,
    }, nil
}

func (w *Writer) writeCompositeLit(expr *ast.CompositeLit) error {
    w.openList()
    w.writeSymbol("CompositeLit")
    w.writeSpace()
    w.writeKeyword("type")
    w.writeSpace()
    if err := w.writeExpr(expr.Type); err != nil {
        return err
    }
    w.writeSpace()
    w.writeKeyword("lbrace")
    w.writeSpace()
    w.writePos(expr.Lbrace)
    w.writeSpace()
    w.writeKeyword("elts")
    w.writeSpace()
    if err := w.writeExprList(expr.Elts); err != nil {
        return err
    }
    w.writeSpace()
    w.writeKeyword("rbrace")
    w.writeSpace()
    w.writePos(expr.Rbrace)
    w.writeSpace()
    w.writeKeyword("incomplete")
    w.writeSpace()
    w.writeBool(expr.Incomplete)
    w.closeList()
    return nil
}
```

---

## Part 4: Type Assertions

### TypeAssertExpr

**Go AST Structure**:
```go
type TypeAssertExpr struct {
    X      Expr      // expression
    Lparen token.Pos // position of "("
    Type   Expr      // asserted type; nil means type switch x.(type)
    Rparen token.Pos // position of ")"
}
```

**Canonical S-Expression Format**:
```lisp
(TypeAssertExpr
  :x <expr>
  :lparen <pos>
  :type <expr-or-nil>
  :rparen <pos>)
```

**Examples**:

```go
// Type assertion
x.(string)

// (TypeAssertExpr
//   :x (Ident :namepos 10 :name "x" :obj nil)
//   :lparen 11
//   :type (Ident :namepos 12 :name "string" :obj nil)
//   :rparen 18)

// Type assertion with complex type
value.([]int)

// (TypeAssertExpr
//   :x (Ident :namepos 10 :name "value" :obj nil)
//   :lparen 15
//   :type (ArrayType :lbrack 16 :len nil :elt (Ident :namepos 18 :name "int" :obj nil))
//   :rparen 21)

// Type switch (Type is nil)
switch x.(type) { ... }

// (TypeAssertExpr
//   :x (Ident :namepos 17 :name "x" :obj nil)
//   :lparen 18
//   :type nil
//   :rparen 23)
```

**Implementation**:

```go
func (b *Builder) buildTypeAssertExpr(s sexp.SExp) (*ast.TypeAssertExpr, error) {
    list, ok := b.expectList(s, "TypeAssertExpr")
    if !ok {
        return nil, fmt.Errorf("not a list")
    }

    if !b.expectSymbol(list.Elements[0], "TypeAssertExpr") {
        return nil, fmt.Errorf("not a TypeAssertExpr node")
    }

    args := b.parseKeywordArgs(list.Elements)

    xVal, ok := b.requireKeyword(args, "x", "TypeAssertExpr")
    if !ok {
        return nil, fmt.Errorf("missing x")
    }

    lparenVal, ok := b.requireKeyword(args, "lparen", "TypeAssertExpr")
    if !ok {
        return nil, fmt.Errorf("missing lparen")
    }

    rparenVal, ok := b.requireKeyword(args, "rparen", "TypeAssertExpr")
    if !ok {
        return nil, fmt.Errorf("missing rparen")
    }

    x, err := b.buildExpr(xVal)
    if err != nil {
        return nil, fmt.Errorf("invalid x: %v", err)
    }

    // Optional type (nil for type switch)
    var typ ast.Expr
    if typeVal, ok := args["type"]; ok && !b.parseNil(typeVal) {
        typ, err = b.buildExpr(typeVal)
        if err != nil {
            return nil, fmt.Errorf("invalid type: %v", err)
        }
    }

    return &ast.TypeAssertExpr{
        X:      x,
        Lparen: b.parsePos(lparenVal),
        Type:   typ,
        Rparen: b.parsePos(rparenVal),
    }, nil
}

func (w *Writer) writeTypeAssertExpr(expr *ast.TypeAssertExpr) error {
    w.openList()
    w.writeSymbol("TypeAssertExpr")
    w.writeSpace()
    w.writeKeyword("x")
    w.writeSpace()
    if err := w.writeExpr(expr.X); err != nil {
        return err
    }
    w.writeSpace()
    w.writeKeyword("lparen")
    w.writeSpace()
    w.writePos(expr.Lparen)
    w.writeSpace()
    w.writeKeyword("type")
    w.writeSpace()
    if err := w.writeExpr(expr.Type); err != nil {
        return err
    }
    w.writeSpace()
    w.writeKeyword("rparen")
    w.writeSpace()
    w.writePos(expr.Rparen)
    w.closeList()
    return nil
}
```

---

## Part 5: Function Literals

### FuncLit

**Go AST Structure**:
```go
type FuncLit struct {
    Type *FuncType
    Body *BlockStmt
}
```

**Canonical S-Expression Format**:
```lisp
(FuncLit
  :type <FuncType>
  :body <BlockStmt>)
```

**Examples**:

```go
// Simple closure
func(x int) int {
    return x * 2
}

// (FuncLit
//   :type (FuncType
//           :func 10
//           :params (FieldList
//                     :opening 14
//                     :list ((Field :doc nil :names ((Ident :namepos 15 :name "x" :obj nil)) :type (Ident :namepos 17 :name "int" :obj nil) :tag nil :comment nil))
//                     :closing 20)
//           :results (FieldList
//                      :opening 0
//                      :list ((Field :doc nil :names () :type (Ident :namepos 22 :name "int" :obj nil) :tag nil :comment nil))
//                      :closing 0))
//   :body (BlockStmt
//           :lbrace 26
//           :list ((ReturnStmt :return 32 :results ((BinaryExpr :x (Ident :namepos 39 :name "x" :obj nil) :oppos 41 :op MUL :y (BasicLit :valuepos 43 :kind INT :value "2")))))
//           :rbrace 45))

// Closure with no parameters
func() {
    doSomething()
}

// (FuncLit
//   :type (FuncType
//           :func 10
//           :params (FieldList :opening 14 :list () :closing 15)
//           :results nil)
//   :body (BlockStmt
//           :lbrace 17
//           :list ((ExprStmt :x (CallExpr :fun (Ident :namepos 23 :name "doSomething" :obj nil) :lparen 34 :args () :ellipsis 0 :rparen 35)))
//           :rbrace 37))

// Closure as argument
sort.Slice(items, func(i, j int) bool {
    return items[i] < items[j]
})

// The FuncLit would be:
// (FuncLit
//   :type (FuncType ...)
//   :body (BlockStmt ...))
```

**Implementation**:

```go
func (b *Builder) buildFuncLit(s sexp.SExp) (*ast.FuncLit, error) {
    list, ok := b.expectList(s, "FuncLit")
    if !ok {
        return nil, fmt.Errorf("not a list")
    }

    if !b.expectSymbol(list.Elements[0], "FuncLit") {
        return nil, fmt.Errorf("not a FuncLit node")
    }

    args := b.parseKeywordArgs(list.Elements)

    typeVal, ok := b.requireKeyword(args, "type", "FuncLit")
    if !ok {
        return nil, fmt.Errorf("missing type")
    }

    bodyVal, ok := b.requireKeyword(args, "body", "FuncLit")
    if !ok {
        return nil, fmt.Errorf("missing body")
    }

    funcType, err := b.buildFuncType(typeVal)
    if err != nil {
        return nil, fmt.Errorf("invalid type: %v", err)
    }

    body, err := b.buildBlockStmt(bodyVal)
    if err != nil {
        return nil, fmt.Errorf("invalid body: %v", err)
    }

    return &ast.FuncLit{
        Type: funcType,
        Body: body,
    }, nil
}

func (w *Writer) writeFuncLit(expr *ast.FuncLit) error {
    w.openList()
    w.writeSymbol("FuncLit")
    w.writeSpace()
    w.writeKeyword("type")
    w.writeSpace()
    if err := w.writeFuncType(expr.Type); err != nil {
        return err
    }
    w.writeSpace()
    w.writeKeyword("body")
    w.writeSpace()
    if err := w.writeBlockStmt(expr.Body); err != nil {
        return err
    }
    w.closeList()
    return nil
}
```

---

## Part 6: Ellipsis

### Ellipsis

**Go AST Structure**:
```go
type Ellipsis struct {
    Ellipsis token.Pos // position of "..."
    Elt      Expr      // ellipsis element type (parameter lists only); or nil
}
```

**Canonical S-Expression Format**:
```lisp
(Ellipsis
  :ellipsis <pos>
  :elt <expr-or-nil>)
```

**Examples**:

```go
// Variadic parameter
func printf(format string, args ...interface{}) { }

// The ...interface{} parameter has type:
// (Ellipsis
//   :ellipsis 35
//   :elt (InterfaceType :interface 38 :methods (FieldList :opening 47 :list () :closing 48) :incomplete false))

// Array literal with ellipsis
[...]int{1, 2, 3}

// The [...] has Len:
// (Ellipsis
//   :ellipsis 10
//   :elt nil)

// Variadic call
printf("hello %s", args...)

// The args... in call has:
// In CallExpr, Ellipsis field is non-zero position of "..."
```

**Note**: Ellipsis is used in two contexts:
1. **Variadic parameters**: `...T` in function parameter list (Elt is the element type)
2. **Array literals**: `[...]T` in array type (Elt is nil)

**Implementation**:

```go
func (b *Builder) buildEllipsis(s sexp.SExp) (*ast.Ellipsis, error) {
    list, ok := b.expectList(s, "Ellipsis")
    if !ok {
        return nil, fmt.Errorf("not a list")
    }

    if !b.expectSymbol(list.Elements[0], "Ellipsis") {
        return nil, fmt.Errorf("not an Ellipsis node")
    }

    args := b.parseKeywordArgs(list.Elements)

    ellipsisVal, ok := b.requireKeyword(args, "ellipsis", "Ellipsis")
    if !ok {
        return nil, fmt.Errorf("missing ellipsis")
    }

    // Optional elt
    var elt ast.Expr
    var err error
    if eltVal, ok := args["elt"]; ok && !b.parseNil(eltVal) {
        elt, err = b.buildExpr(eltVal)
        if err != nil {
            return nil, fmt.Errorf("invalid elt: %v", err)
        }
    }

    return &ast.Ellipsis{
        Ellipsis: b.parsePos(ellipsisVal),
        Elt:      elt,
    }, nil
}

func (w *Writer) writeEllipsis(expr *ast.Ellipsis) error {
    w.openList()
    w.writeSymbol("Ellipsis")
    w.writeSpace()
    w.writeKeyword("ellipsis")
    w.writeSpace()
    w.writePos(expr.Ellipsis)
    w.writeSpace()
    w.writeKeyword("elt")
    w.writeSpace()
    if err := w.writeExpr(expr.Elt); err != nil {
        return err
    }
    w.closeList()
    return nil
}
```

---

## Part 7: Dispatcher Updates

Add all Phase 4 nodes to expression dispatcher:

**In `buildExpr`**:
```go
case "StructType":
    return b.buildStructType(s)
case "InterfaceType":
    return b.buildInterfaceType(s)
case "CompositeLit":
    return b.buildCompositeLit(s)
case "TypeAssertExpr":
    return b.buildTypeAssertExpr(s)
case "FuncLit":
    return b.buildFuncLit(s)
case "Ellipsis":
    return b.buildEllipsis(s)
```

**In `writeExpr`**:
```go
case *ast.StructType:
    return w.writeStructType(e)
case *ast.InterfaceType:
    return w.writeInterfaceType(e)
case *ast.CompositeLit:
    return w.writeCompositeLit(e)
case *ast.TypeAssertExpr:
    return w.writeTypeAssertExpr(e)
case *ast.FuncLit:
    return w.writeFuncLit(e)
case *ast.Ellipsis:
    return w.writeEllipsis(e)
```

---

## Part 8: Integration Tests

### Test 1: Struct Definitions

```go
func TestPhase4Structs(t *testing.T) {
    source := `package main

type Point struct {
    X, Y int
}

type Person struct {
    Name string ` + "`json:\"name\"`" + `
    Age  int    ` + "`json:\"age\"`" + `
}

type Employee struct {
    Person  // embedded
    ID int
}
`
    testRoundTrip(t, source)
}
```

### Test 2: Interface Definitions

```go
func TestPhase4Interfaces(t *testing.T) {
    source := `package main

type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

type ReadWriter interface {
    Reader
    Writer
}

type Closer interface {
    Close() error
}
`
    testRoundTrip(t, source)
}
```

### Test 3: Composite Literals

```go
func TestPhase4CompositeLiterals(t *testing.T) {
    source := `package main

type Point struct {
    X, Y int
}

func literals() {
    // Struct literal
    p1 := Point{X: 1, Y: 2}
    p2 := Point{1, 2}
    
    // Array literal
    arr := [3]int{1, 2, 3}
    
    // Slice literal
    slice := []string{"a", "b", "c"}
    
    // Map literal
    m := map[string]int{
        "x": 1,
        "y": 2,
    }
    
    // Nested
    points := []Point{
        {X: 1, Y: 2},
        {X: 3, Y: 4},
    }
}
`
    testRoundTrip(t, source)
}
```

### Test 4: Type Assertions

```go
func TestPhase4TypeAssertions(t *testing.T) {
    source := `package main

func assertions(x interface{}) {
    // Type assertion
    s := x.(string)
    
    // Type assertion with check
    if str, ok := x.(string); ok {
        process(str)
    }
    
    // Type switch
    switch v := x.(type) {
    case int:
        processInt(v)
    case string:
        processString(v)
    default:
        processOther(v)
    }
}
`
    testRoundTrip(t, source)
}
```

### Test 5: Function Literals

```go
func TestPhase4FunctionLiterals(t *testing.T) {
    source := `package main

import "sort"

func closures() {
    // Simple closure
    add := func(x, y int) int {
        return x + y
    }
    result := add(1, 2)
    
    // Closure capturing variable
    count := 0
    increment := func() {
        count++
    }
    increment()
    
    // Closure as argument
    items := []int{3, 1, 4, 1, 5}
    sort.Slice(items, func(i, j int) bool {
        return items[i] < items[j]
    })
}
`
    testRoundTrip(t, source)
}
```

### Test 6: Variadic Functions

```go
func TestPhase4VariadicFunctions(t *testing.T) {
    source := `package main

func sum(nums ...int) int {
    total := 0
    for _, n := range nums {
        total += n
    }
    return total
}

func printf(format string, args ...interface{}) {
    // implementation
}

func main() {
    sum(1, 2, 3)
    
    nums := []int{1, 2, 3, 4}
    sum(nums...)
}
`
    testRoundTrip(t, source)
}
```

### Test 7: Complete OOP Example

```go
func TestPhase4OOPPattern(t *testing.T) {
    source := `package main

type Shape interface {
    Area() float64
}

type Circle struct {
    Radius float64
}

func (c Circle) Area() float64 {
    return 3.14 * c.Radius * c.Radius
}

type Rectangle struct {
    Width, Height float64
}

func (r Rectangle) Area() float64 {
    return r.Width * r.Height
}

func totalArea(shapes []Shape) float64 {
    total := 0.0
    for _, s := range shapes {
        total += s.Area()
    }
    return total
}

func main() {
    shapes := []Shape{
        Circle{Radius: 5},
        Rectangle{Width: 10, Height: 20},
    }
    total := totalArea(shapes)
}
`
    testRoundTrip(t, source)
}
```

---

## Part 9: Pretty Printer Updates

Update `formStyles` in `sexp/pretty.go`:

```go
var formStyles = map[string]FormStyle{
    // Existing...
    
    // Phase 4 Complex Types
    "StructType":      StyleKeywordPairs,
    "InterfaceType":   StyleKeywordPairs,
    "CompositeLit":    StyleKeywordPairs,
    "TypeAssertExpr":  StyleKeywordPairs,
    "FuncLit":         StyleKeywordPairs,
    "Ellipsis":        StyleCompact,
}
```

---

## Success Criteria

### Code Completeness
- [ ] All 6 Phase 4 nodes implemented in Builder
- [ ] All 6 Phase 4 nodes implemented in Writer
- [ ] Expression dispatcher updated
- [ ] Pretty printer updated
- [ ] Field node properly handles tags and embedded fields

### Testing
- [ ] Unit tests for each new node (builder_test.go)
- [ ] Unit tests for each new node (writer_test.go)
- [ ] 7 integration tests passing
- [ ] Test coverage >90% for new code

### Documentation
- [ ] Update README with Phase 4 capabilities
- [ ] Add OOP pattern examples
- [ ] Update canonical S-expression format spec

### Validation
- [ ] Can parse struct definitions with tags
- [ ] Can parse interface definitions
- [ ] Can parse all composite literal forms
- [ ] Can parse type assertions and type switches
- [ ] Can parse closures and function literals
- [ ] Can parse variadic functions
- [ ] Round-trip tests pass for all integration tests

---

## Implementation Tips

1. **Start with StructType and InterfaceType**: These are similar and build on existing FieldList
2. **Test Embedded Fields**: Make sure empty Names list works correctly
3. **Test Struct Tags**: Ensure BasicLit STRING tags are preserved
4. **CompositeLit Complexity**: Test all forms (struct, array, slice, map, nested)
5. **Type Inference**: CompositeLit Type can be nil when inferred
6. **TypeAssertExpr for Type Switch**: Type is nil for `x.(type)`
7. **Ellipsis Two Uses**: Parameter lists (Elt != nil) vs array literals (Elt == nil)

---

## Estimated Timeline

- **Part 1 (StructType)**: 0.5 days
- **Part 2 (InterfaceType)**: 0.5 days
- **Part 3 (CompositeLit)**: 1 day
- **Part 4 (TypeAssertExpr)**: 0.25 days
- **Part 5 (FuncLit)**: 0.5 days
- **Part 6 (Ellipsis)**: 0.25 days
- **Part 7-9 (Updates & Tests)**: 0.5 days

**Total**: 2-3 days

---

## Next Steps After Phase 4

Once Phase 4 is complete and all tests pass:

1. **Update documentation** with OOP examples
2. **Create Phase 5 specifications** for advanced features (generics, comments, scopes)
3. **Test with real codebases**: Try parsing actual Go libraries
4. **Celebrate** - You can now handle sophisticated Go programs with full OOP!

---

Phase 4 is the most intellectually demanding phase due to the recursive nature of composite types. Take your time, test thoroughly, and you'll have a complete type system implementation. Good luck!
