---
number: 0003
title: Canonical S-Expression Format for Go AST
author: Duncan McGreggor
created: 2025-10-01
updated: 2025-10-01
state: Draft
supersedes: None
superseded-by: None
---

# Canonical S-Expression Format for Go AST

**Version**: 0.1.0 (Phase 1)  
**Date**: October 2025  
**Status**: Draft

---

## Overview

This document defines the canonical S-expression format for representing Go's Abstract Syntax Tree (AST). This format provides a 1:1 bidirectional mapping between Go AST nodes and S-expressions.

### Design Principles

1. **Faithful Representation**: Every field in Go's AST is represented
2. **Position Preservation**: All `token.Pos` information is maintained
3. **Self-Contained**: FileSet information is embedded in the output
4. **Keyword Arguments**: Explicit field names for clarity and extensibility
5. **Go Semantics**: Use `nil` for null values, matching Go conventions

---

## File Structure

Every S-expression file contains a root `Program` node:

```lisp
(Program
  :fileset (FileSet ...)
  :files (
    (File ...)
    ...))
```

### FileSet Representation

The FileSet tracks source file information for position mapping:

```lisp
(FileSet
  :base 1
  :files (
    (FileInfo
      :name "main.go"
      :base 1
      :size 150
      :lines (1 15 30 45 60 75 90 105 120 135 150))))
```

**Fields**:
- `:base` - Base offset for the FileSet
- `:files` - List of FileInfo nodes
- `:name` - Source filename
- `:size` - File size in bytes
- `:lines` - List of byte offsets for each line start (for line/column calculation)

---

## Core Syntax Rules

### Node Format

```lisp
(NodeType :field1 value1 :field2 value2 ...)
```

### Position Fields

All `token.Pos` fields are represented as integers:
- `0` means "no position" (Go convention)
- Positive integers are byte offsets into the file

### Nil Values

Use `nil` for null/missing values:

```lisp
(FuncDecl
  :doc nil
  :recv nil
  :name (Ident :namepos 10 :name "main" :obj nil))
```

### Lists

Lists of nodes use nested S-expressions:

```lisp
:decls (
  (GenDecl ...)
  (FuncDecl ...))
```

Empty lists:

```lisp
:list ()
```

---

## Phase 1 Node Types

These nodes support a minimal "Hello, world" program.

### File (Root Node)

```lisp
(File
  :package <pos>
  :name <Ident>
  :decls (<Decl> ...)
  :scope <Scope>
  :imports (<ImportSpec> ...)
  :unresolved (<Ident> ...)
  :comments (<CommentGroup> ...))
```

**Fields**:
- `:package` - Position of "package" keyword
- `:name` - Package name identifier
- `:decls` - Top-level declarations
- `:scope` - Package scope (may be nil)
- `:imports` - List of imports (after resolution)
- `:unresolved` - Unresolved identifiers (may be empty)
- `:comments` - All comments in source (may be empty)

### Ident (Identifier)

```lisp
(Ident
  :namepos <pos>
  :name <string>
  :obj <Object>)
```

**Fields**:
- `:namepos` - Position of identifier
- `:name` - Identifier string
- `:obj` - Denoted object (nil if unresolved)

### BasicLit (Basic Literal)

```lisp
(BasicLit
  :valuepos <pos>
  :kind <token>
  :value <string>)
```

**Fields**:
- `:valuepos` - Position of literal
- `:kind` - Token kind (INT, FLOAT, IMAG, CHAR, STRING)
- `:value` - Literal value as string (includes quotes for strings)

**Token Kinds**:
- `INT` - Integer literal
- `FLOAT` - Float literal
- `IMAG` - Imaginary literal
- `CHAR` - Character literal
- `STRING` - String literal

### GenDecl (General Declaration)

Used for imports, constants, types, and variables.

```lisp
(GenDecl
  :doc <CommentGroup>
  :tok <token>
  :tokpos <pos>
  :lparen <pos>
  :specs (<Spec> ...)
  :rparen <pos>)
```

**Fields**:
- `:doc` - Associated documentation (may be nil)
- `:tok` - Token (IMPORT, CONST, TYPE, VAR)
- `:tokpos` - Position of token
- `:lparen` - Position of '(' if present, else 0
- `:specs` - List of specifications
- `:rparen` - Position of ')' if present, else 0

### ImportSpec

```lisp
(ImportSpec
  :doc <CommentGroup>
  :name <Ident>
  :path <BasicLit>
  :comment <CommentGroup>
  :endpos <pos>)
```

**Fields**:
- `:doc` - Associated documentation (may be nil)
- `:name` - Local package name (may be nil)
- `:path` - Import path (BasicLit with STRING kind)
- `:comment` - Line comment (may be nil)
- `:endpos` - End of spec (position of newline or semicolon)

### FuncDecl (Function Declaration)

```lisp
(FuncDecl
  :doc <CommentGroup>
  :recv <FieldList>
  :name <Ident>
  :type <FuncType>
  :body <BlockStmt>)
```

**Fields**:
- `:doc` - Associated documentation (may be nil)
- `:recv` - Receiver (for methods; nil for functions)
- `:name` - Function name
- `:type` - Function type (signature)
- `:body` - Function body (may be nil for declarations)

### FuncType (Function Type)

```lisp
(FuncType
  :func <pos>
  :params <FieldList>
  :results <FieldList>)
```

**Fields**:
- `:func` - Position of "func" keyword (may be 0)
- `:params` - Parameter list
- `:results` - Result list (may be nil)

### FieldList

```lisp
(FieldList
  :opening <pos>
  :list (<Field> ...)
  :closing <pos>)
```

**Fields**:
- `:opening` - Position of opening delimiter ('(' or '{')
- `:list` - List of fields
- `:closing` - Position of closing delimiter (')' or '}')

### Field

```lisp
(Field
  :doc <CommentGroup>
  :names (<Ident> ...)
  :type <Expr>
  :tag <BasicLit>
  :comment <CommentGroup>)
```

**Fields**:
- `:doc` - Associated documentation (may be nil)
- `:names` - Field/parameter names (may be empty)
- `:type` - Field type
- `:tag` - Struct tag (may be nil)
- `:comment` - Line comment (may be nil)

### BlockStmt (Block Statement)

```lisp
(BlockStmt
  :lbrace <pos>
  :list (<Stmt> ...)
  :rbrace <pos>)
```

**Fields**:
- `:lbrace` - Position of '{'
- `:list` - List of statements
- `:rbrace` - Position of '}'

### ExprStmt (Expression Statement)

```lisp
(ExprStmt
  :x <Expr>)
```

**Fields**:
- `:x` - Expression

### CallExpr (Call Expression)

```lisp
(CallExpr
  :fun <Expr>
  :lparen <pos>
  :args (<Expr> ...)
  :ellipsis <pos>
  :rparen <pos>)
```

**Fields**:
- `:fun` - Function expression
- `:lparen` - Position of '('
- `:args` - Argument expressions
- `:ellipsis` - Position of "..." (0 if not variadic)
- `:rparen` - Position of ')'

### SelectorExpr (Selector Expression)

```lisp
(SelectorExpr
  :x <Expr>
  :sel <Ident>)
```

**Fields**:
- `:x` - Expression (left side)
- `:sel` - Selector identifier (right side)

### CommentGroup (Comment Group)

```lisp
(CommentGroup
  :list (<Comment> ...))
```

**Fields**:
- `:list` - List of comments

### Comment

```lisp
(Comment
  :slash <pos>
  :text <string>)
```

**Fields**:
- `:slash` - Position of "//" or "/*"
- `:text` - Comment text (including delimiters)

### Object (Referenced Object)

```lisp
(Object
  :kind <ObjKind>
  :name <string>
  :decl <Node>
  :data <any>
  :type <any>)
```

**Fields**:
- `:kind` - Object kind (Bad, Pkg, Con, Typ, Var, Fun, Lbl)
- `:name` - Object name
- `:decl` - Declaration node (may be nil)
- `:data` - Object-specific data (may be nil)
- `:type` - Object type (may be nil)

**Object Kinds**:
- `Bad` - Error sentinel
- `Pkg` - Package
- `Con` - Constant
- `Typ` - Type
- `Var` - Variable
- `Fun` - Function
- `Lbl` - Label

### Scope

```lisp
(Scope
  :outer <Scope>
  :objects (
    (<name> <Object>)
    ...))
```

**Fields**:
- `:outer` - Outer (parent) scope (may be nil)
- `:objects` - Map of name to Object

---

## Complete Example: Hello World

### Go Source

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, world!")
}
```

### Canonical S-Expression

```lisp
(Program
  :fileset (FileSet
    :base 1
    :files (
      (FileInfo
        :name "main.go"
        :base 1
        :size 78
        :lines (1 14 27 42 78))))
  
  :files (
    (File
      :package 1
      :name (Ident :namepos 9 :name "main" :obj nil)
      :decls (
        (GenDecl
          :doc nil
          :tok IMPORT
          :tokpos 15
          :lparen 0
          :specs (
            (ImportSpec
              :doc nil
              :name nil
              :path (BasicLit :valuepos 22 :kind STRING :value "\"fmt\"")
              :comment nil
              :endpos 27))
          :rparen 0)
        
        (FuncDecl
          :doc nil
          :recv nil
          :name (Ident :namepos 33 :name "main" :obj nil)
          :type (FuncType
                  :func 28
                  :params (FieldList :opening 37 :list () :closing 38)
                  :results nil)
          :body (BlockStmt
                  :lbrace 40
                  :list (
                    (ExprStmt
                      :x (CallExpr
                           :fun (SelectorExpr
                                  :x (Ident :namepos 46 :name "fmt" :obj nil)
                                  :sel (Ident :namepos 50 :name "Println" :obj nil))
                           :lparen 57
                           :args (
                             (BasicLit :valuepos 58 :kind STRING :value "\"Hello, world!\""))
                           :ellipsis 0
                           :rparen 74)))
                  :rbrace 76)))
      
      :scope nil
      :imports (
        (ImportSpec
          :doc nil
          :name nil
          :path (BasicLit :valuepos 22 :kind STRING :value "\"fmt\"")
          :comment nil
          :endpos 27))
      :unresolved ()
      :comments ())))
```

---

## Implementation Notes

### Position Calculation

To convert `token.Pos` to line/column:
1. Find the FileInfo containing the position
2. Binary search the `:lines` array to find the line
3. Column = position - line_start_offset

### Round-Trip Guarantees

The format guarantees:
1. Go source → AST → S-expr → AST → Go source produces equivalent code
2. All position information is preserved
3. All semantic information is preserved

### Future Phases

Phase 2+ will add:
- Control flow (if, for, switch, select)
- More expressions (binary, unary, index, slice, type assertions)
- Type declarations (struct, interface, etc.)
- More statements (assign, return, defer, go, etc.)

---

## Version History

- **0.1.0** (October 2025) - Initial draft, Phase 1 nodes only

