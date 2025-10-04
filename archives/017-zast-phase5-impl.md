# Phase 5 Implementation Specification - Advanced Features

**Project**: zast  
**Phase**: 5 of 6  
**Goal**: Implement advanced features and edge cases  
**Estimated Effort**: 2-3 days  
**Prerequisites**: Phases 1-4 complete (basic nodes, easy wins, control flow, complex types)

---

## Overview

Phase 5 adds support for advanced Go features and completes the remaining expression and statement types. These nodes handle generics (Go 1.18+), comments/documentation, scopes/objects, and various edge cases. After Phase 5, you'll have near-complete Go AST coverage.

**What you'll be able to handle after Phase 5**:
- Generic type instantiation (Go 1.18+)
- Full comment and documentation preservation
- Symbol tables and scope information
- All remaining expression types
- Error nodes for malformed code
- Complete position and metadata tracking

---

## Implementation Checklist

### Generics (Go 1.18+) (1 node)
- [ ] `IndexListExpr` - Generic type instantiation

### Comments & Documentation (2 nodes)
- [ ] `Comment` - Single comment (full implementation)
- [ ] `CommentGroup` - Group of comments (full implementation)

### Scope & Objects (2 nodes)
- [ ] `Scope` - Symbol table (full implementation)
- [ ] `Object` - Name declaration info (full implementation)

### Remaining Expressions (3 nodes)
- [ ] `BadExpr` - Placeholder for syntax errors
- [ ] `File` (Package field) - Complete File implementation
- [ ] `Package` - Collection of files

### Updates to Existing Nodes
- [ ] Update all nodes to properly write/read comments (not just nil)
- [ ] Update File to properly handle Scope, Imports, Unresolved, Comments

---

## Part 1: Generics Support

### IndexListExpr (Go 1.18+)

**Go AST Structure**:
```go
type IndexListExpr struct {
    X       Expr      // expression
    Lbrack  token.Pos // position of "["
    Indices []Expr    // index expressions
    Rbrack  token.Pos // position of "]"
}
```

**Canonical S-Expression Format**:
```lisp
(IndexListExpr
  :x <expr>
  :lbrack <pos>
  :indices (<expr> ...)
  :rbrack <pos>)
```

**Examples**:

```go
// Generic type instantiation
List[int]

// (IndexListExpr
//   :x (Ident :namepos 10 :name "List" :obj nil)
//   :lbrack 14
//   :indices ((Ident :namepos 15 :name "int" :obj nil))
//   :rbrack 18)

// Multiple type parameters
Map[string, int]

// (IndexListExpr
//   :x (Ident :namepos 10 :name "Map" :obj nil)
//   :lbrack 13
//   :indices (
//     (Ident :namepos 14 :name "string" :obj nil)
//     (Ident :namepos 22 :name "int" :obj nil))
//   :rbrack 25)

// Nested generics
List[Map[string, int]]

// (IndexListExpr
//   :x (Ident :namepos 10 :name "List" :obj nil)
//   :lbrack 14
//   :indices (
//     (IndexListExpr
//       :x (Ident :namepos 15 :name "Map" :obj nil)
//       :lbrack 18
//       :indices (
//         (Ident :namepos 19 :name "string" :obj nil)
//         (Ident :namepos 27 :name "int" :obj nil))
//       :rbrack 30))
//   :rbrack 31)
```

**Note**: This node was introduced in Go 1.18 for generics. If you're targeting earlier Go versions, you can skip this node or implement it for forward compatibility.

**Implementation**:

```go
func (b *Builder) buildIndexListExpr(s sexp.SExp) (*ast.IndexListExpr, error) {
    list, ok := b.expectList(s, "IndexListExpr")
    if !ok {
        return nil, fmt.Errorf("not a list")
    }

    if !b.expectSymbol(list.Elements[0], "IndexListExpr") {
        return nil, fmt.Errorf("not an IndexListExpr node")
    }

    args := b.parseKeywordArgs(list.Elements)

    xVal, ok := b.requireKeyword(args, "x", "IndexListExpr")
    if !ok {
        return nil, fmt.Errorf("missing x")
    }

    lbrackVal, ok := b.requireKeyword(args, "lbrack", "IndexListExpr")
    if !ok {
        return nil, fmt.Errorf("missing lbrack")
    }

    indicesVal, ok := b.requireKeyword(args, "indices", "IndexListExpr")
    if !ok {
        return nil, fmt.Errorf("missing indices")
    }

    rbrackVal, ok := b.requireKeyword(args, "rbrack", "IndexListExpr")
    if !ok {
        return nil, fmt.Errorf("missing rbrack")
    }

    x, err := b.buildExpr(xVal)
    if err != nil {
        return nil, fmt.Errorf("invalid x: %v", err)
    }

    // Build indices list
    var indices []ast.Expr
    indicesList, ok := b.expectList(indicesVal, "IndexListExpr indices")
    if ok {
        for _, indexSexp := range indicesList.Elements {
            index, err := b.buildExpr(indexSexp)
            if err != nil {
                return nil, fmt.Errorf("invalid index: %v", err)
            }
            indices = append(indices, index)
        }
    }

    return &ast.IndexListExpr{
        X:       x,
        Lbrack:  b.parsePos(lbrackVal),
        Indices: indices,
        Rbrack:  b.parsePos(rbrackVal),
    }, nil
}

func (w *Writer) writeIndexListExpr(expr *ast.IndexListExpr) error {
    w.openList()
    w.writeSymbol("IndexListExpr")
    w.writeSpace()
    w.writeKeyword("x")
    w.writeSpace()
    if err := w.writeExpr(expr.X); err != nil {
        return err
    }
    w.writeSpace()
    w.writeKeyword("lbrack")
    w.writeSpace()
    w.writePos(expr.Lbrack)
    w.writeSpace()
    w.writeKeyword("indices")
    w.writeSpace()
    if err := w.writeExprList(expr.Indices); err != nil {
        return err
    }
    w.writeSpace()
    w.writeKeyword("rbrack")
    w.writeSpace()
    w.writePos(expr.Rbrack)
    w.closeList()
    return nil
}
```

---

## Part 2: Comments and Documentation

### Comment

**Go AST Structure**:
```go
type Comment struct {
    Slash token.Pos // position of "/" starting the comment
    Text  string    // comment text (including "//" or "/*" and "*/")
}
```

**Canonical S-Expression Format**:
```lisp
(Comment
  :slash <pos>
  :text <string>)
```

**Examples**:

```go
// Single line comment
// This is a comment

// (Comment
//   :slash 10
//   :text "// This is a comment")

/* Block comment */
/* This is
   a multi-line
   comment */

// (Comment
//   :slash 15
//   :text "/* This is\n   a multi-line\n   comment */")
```

**Implementation**:

```go
func (b *Builder) buildComment(s sexp.SExp) (*ast.Comment, error) {
    list, ok := b.expectList(s, "Comment")
    if !ok {
        return nil, fmt.Errorf("not a list")
    }

    if !b.expectSymbol(list.Elements[0], "Comment") {
        return nil, fmt.Errorf("not a Comment node")
    }

    args := b.parseKeywordArgs(list.Elements)

    slashVal, ok := b.requireKeyword(args, "slash", "Comment")
    if !ok {
        return nil, fmt.Errorf("missing slash")
    }

    textVal, ok := b.requireKeyword(args, "text", "Comment")
    if !ok {
        return nil, fmt.Errorf("missing text")
    }

    text, err := b.parseString(textVal)
    if err != nil {
        return nil, fmt.Errorf("invalid text: %v", err)
    }

    return &ast.Comment{
        Slash: b.parsePos(slashVal),
        Text:  text,
    }, nil
}

func (w *Writer) writeComment(c *ast.Comment) error {
    if c == nil {
        w.writeSymbol("nil")
        return nil
    }

    w.openList()
    w.writeSymbol("Comment")
    w.writeSpace()
    w.writeKeyword("slash")
    w.writeSpace()
    w.writePos(c.Slash)
    w.writeSpace()
    w.writeKeyword("text")
    w.writeSpace()
    w.writeString(c.Text)
    w.closeList()
    return nil
}
```

---

### CommentGroup

**Go AST Structure**:
```go
type CommentGroup struct {
    List []*Comment // len(List) > 0
}
```

**Canonical S-Expression Format**:
```lisp
(CommentGroup
  :list (<Comment> ...))
```

**Examples**:

```go
// Multiple consecutive comments
// Line 1
// Line 2
// Line 3

// (CommentGroup
//   :list (
//     (Comment :slash 10 :text "// Line 1")
//     (Comment :slash 22 :text "// Line 2")
//     (Comment :slash 34 :text "// Line 3")))

// Doc comment
// Package main provides...
//
// This is the main package.
package main

// (CommentGroup
//   :list (
//     (Comment :slash 10 :text "// Package main provides...")
//     (Comment :slash 40 :text "//")
//     (Comment :slash 43 :text "// This is the main package.")))
```

**Implementation**:

```go
func (b *Builder) buildCommentGroup(s sexp.SExp) (*ast.CommentGroup, error) {
    if b.parseNil(s) {
        return nil, nil
    }

    list, ok := b.expectList(s, "CommentGroup")
    if !ok {
        return nil, fmt.Errorf("not a list")
    }

    if !b.expectSymbol(list.Elements[0], "CommentGroup") {
        return nil, fmt.Errorf("not a CommentGroup node")
    }

    args := b.parseKeywordArgs(list.Elements)

    listVal, ok := b.requireKeyword(args, "list", "CommentGroup")
    if !ok {
        return nil, fmt.Errorf("missing list")
    }

    // Build comments list
    var comments []*ast.Comment
    commentsList, ok := b.expectList(listVal, "CommentGroup list")
    if ok {
        for _, commentSexp := range commentsList.Elements {
            comment, err := b.buildComment(commentSexp)
            if err != nil {
                return nil, fmt.Errorf("invalid comment: %v", err)
            }
            comments = append(comments, comment)
        }
    }

    return &ast.CommentGroup{
        List: comments,
    }, nil
}

func (w *Writer) writeCommentGroup(cg *ast.CommentGroup) error {
    if cg == nil {
        w.writeSymbol("nil")
        return nil
    }

    w.openList()
    w.writeSymbol("CommentGroup")
    w.writeSpace()
    w.writeKeyword("list")
    w.writeSpace()
    w.openList()
    for i, c := range cg.List {
        if i > 0 {
            w.writeSpace()
        }
        if err := w.writeComment(c); err != nil {
            return err
        }
    }
    w.closeList()
    w.closeList()
    return nil
}
```

**Update all nodes**: Now that CommentGroup is implemented, update all nodes that currently write `:doc nil` and `:comment nil` to properly handle CommentGroup fields:
- `Field`
- `FuncDecl`
- `GenDecl`
- `ImportSpec`
- `ValueSpec`
- `TypeSpec`

---

## Part 3: Scope and Objects

### Object

**Go AST Structure**:
```go
type Object struct {
    Kind ObjKind
    Name string    // declared name
    Decl interface{} // corresponding Field, XxxSpec, FuncDecl, LabeledStmt, AssignStmt, Scope; or nil
    Data interface{} // object-specific data; or nil
    Type interface{} // placeholder for type information; may be nil
}

type ObjKind int
const (
    Bad ObjKind = iota // for error handling
    Pkg                // package
    Con                // constant
    Typ                // type
    Var                // variable
    Fun                // function or method
    Lbl                // label
)
```

**Canonical S-Expression Format**:
```lisp
(Object
  :kind <objkind>
  :name <string>
  :decl <node-or-nil>
  :data <any-or-nil>
  :type <any-or-nil>)
```

**Examples**:

```go
// Variable object
var x int

// (Object
//   :kind Var
//   :name "x"
//   :decl <reference-to-ValueSpec>
//   :data nil
//   :type nil)

// Function object
func foo() {}

// (Object
//   :kind Fun
//   :name "foo"
//   :decl <reference-to-FuncDecl>
//   :data nil
//   :type nil)
```

**Implementation Note**: Object's Decl, Data, and Type fields use `interface{}`, making them challenging to serialize. For Phase 5, we'll implement a simplified version:

**Simplified Approach**:
- For `Decl`: Store nil (full cross-reference tracking is complex)
- For `Data`: Store nil
- For `Type`: Store nil

**Alternative Full Approach** (if needed):
- Create a union type for Decl that can reference any declaration node
- Implement reference tracking to handle circular dependencies

```go
func (b *Builder) buildObject(s sexp.SExp) (*ast.Object, error) {
    if b.parseNil(s) {
        return nil, nil
    }

    list, ok := b.expectList(s, "Object")
    if !ok {
        return nil, fmt.Errorf("not a list")
    }

    if !b.expectSymbol(list.Elements[0], "Object") {
        return nil, fmt.Errorf("not an Object node")
    }

    args := b.parseKeywordArgs(list.Elements)

    kindVal, ok := b.requireKeyword(args, "kind", "Object")
    if !ok {
        return nil, fmt.Errorf("missing kind")
    }

    nameVal, ok := b.requireKeyword(args, "name", "Object")
    if !ok {
        return nil, fmt.Errorf("missing name")
    }

    kind, err := b.parseObjKind(kindVal)
    if err != nil {
        return nil, fmt.Errorf("invalid kind: %v", err)
    }

    name, err := b.parseString(nameVal)
    if err != nil {
        return nil, fmt.Errorf("invalid name: %v", err)
    }

    return &ast.Object{
        Kind: kind,
        Name: name,
        Decl: nil, // Simplified: not tracking cross-references
        Data: nil,
        Type: nil,
    }, nil
}

func (b *Builder) parseObjKind(s sexp.SExp) (ast.ObjKind, error) {
    sym, ok := s.(*sexp.Symbol)
    if !ok {
        return ast.Bad, fmt.Errorf("expected symbol for ObjKind, got %T", s)
    }

    switch sym.Value {
    case "Bad":
        return ast.Bad, nil
    case "Pkg":
        return ast.Pkg, nil
    case "Con":
        return ast.Con, nil
    case "Typ":
        return ast.Typ, nil
    case "Var":
        return ast.Var, nil
    case "Fun":
        return ast.Fun, nil
    case "Lbl":
        return ast.Lbl, nil
    default:
        return ast.Bad, fmt.Errorf("unknown ObjKind: %s", sym.Value)
    }
}

func (w *Writer) writeObject(obj *ast.Object) error {
    if obj == nil {
        w.writeSymbol("nil")
        return nil
    }

    w.openList()
    w.writeSymbol("Object")
    w.writeSpace()
    w.writeKeyword("kind")
    w.writeSpace()
    w.writeObjKind(obj.Kind)
    w.writeSpace()
    w.writeKeyword("name")
    w.writeSpace()
    w.writeString(obj.Name)
    w.writeSpace()
    w.writeKeyword("decl")
    w.writeSpace()
    w.writeSymbol("nil") // Simplified
    w.writeSpace()
    w.writeKeyword("data")
    w.writeSpace()
    w.writeSymbol("nil")
    w.writeSpace()
    w.writeKeyword("type")
    w.writeSpace()
    w.writeSymbol("nil")
    w.closeList()
    return nil
}

func (w *Writer) writeObjKind(kind ast.ObjKind) {
    switch kind {
    case ast.Bad:
        w.writeSymbol("Bad")
    case ast.Pkg:
        w.writeSymbol("Pkg")
    case ast.Con:
        w.writeSymbol("Con")
    case ast.Typ:
        w.writeSymbol("Typ")
    case ast.Var:
        w.writeSymbol("Var")
    case ast.Fun:
        w.writeSymbol("Fun")
    case ast.Lbl:
        w.writeSymbol("Lbl")
    default:
        w.writeSymbol("Bad")
    }
}
```

**Update Ident**: Now that Object is implemented, update `buildIdent` and `writeIdent` to handle the Obj field properly instead of always writing nil.

---

### Scope

**Go AST Structure**:
```go
type Scope struct {
    Outer   *Scope
    Objects map[string]*Object
}
```

**Canonical S-Expression Format**:
```lisp
(Scope
  :outer <Scope-or-nil>
  :objects ((<name> <Object>) ...))
```

**Examples**:

```go
// Package scope
package main
var x int

// (Scope
//   :outer nil
//   :objects (
//     ("x" (Object :kind Var :name "x" :decl nil :data nil :type nil))))

// Nested scope
func foo() {
    var x int  // local x
}

// (Scope
//   :outer <reference-to-package-scope>
//   :objects (
//     ("x" (Object :kind Var :name "x" :decl nil :data nil :type nil))))
```

**Implementation**:

```go
func (b *Builder) buildScope(s sexp.SExp) (*ast.Scope, error) {
    if b.parseNil(s) {
        return nil, nil
    }

    list, ok := b.expectList(s, "Scope")
    if !ok {
        return nil, fmt.Errorf("not a list")
    }

    if !b.expectSymbol(list.Elements[0], "Scope") {
        return nil, fmt.Errorf("not a Scope node")
    }

    args := b.parseKeywordArgs(list.Elements)

    objectsVal, ok := b.requireKeyword(args, "objects", "Scope")
    if !ok {
        return nil, fmt.Errorf("missing objects")
    }

    // Optional outer
    var outer *ast.Scope
    var err error
    if outerVal, ok := args["outer"]; ok && !b.parseNil(outerVal) {
        outer, err = b.buildScope(outerVal)
        if err != nil {
            return nil, fmt.Errorf("invalid outer: %v", err)
        }
    }

    // Build objects map
    objects := make(map[string]*ast.Object)
    objectsList, ok := b.expectList(objectsVal, "Scope objects")
    if ok {
        for _, objEntry := range objectsList.Elements {
            entryList, ok := b.expectList(objEntry, "Scope object entry")
            if !ok || len(entryList.Elements) != 2 {
                return nil, fmt.Errorf("invalid object entry")
            }

            name, err := b.parseString(entryList.Elements[0])
            if err != nil {
                return nil, fmt.Errorf("invalid object name: %v", err)
            }

            obj, err := b.buildObject(entryList.Elements[1])
            if err != nil {
                return nil, fmt.Errorf("invalid object: %v", err)
            }

            objects[name] = obj
        }
    }

    return &ast.Scope{
        Outer:   outer,
        Objects: objects,
    }, nil
}

func (w *Writer) writeScope(scope *ast.Scope) error {
    if scope == nil {
        w.writeSymbol("nil")
        return nil
    }

    w.openList()
    w.writeSymbol("Scope")
    w.writeSpace()
    w.writeKeyword("outer")
    w.writeSpace()
    if err := w.writeScope(scope.Outer); err != nil {
        return err
    }
    w.writeSpace()
    w.writeKeyword("objects")
    w.writeSpace()
    w.openList()
    
    // Sort keys for deterministic output
    names := make([]string, 0, len(scope.Objects))
    for name := range scope.Objects {
        names = append(names, name)
    }
    sort.Strings(names)
    
    for i, name := range names {
        if i > 0 {
            w.writeSpace()
        }
        w.openList()
        w.writeString(name)
        w.writeSpace()
        if err := w.writeObject(scope.Objects[name]); err != nil {
            return err
        }
        w.closeList()
    }
    
    w.closeList()
    w.closeList()
    return nil
}
```

**Update File**: Now that Scope is implemented, update File to properly handle the Scope field.

---

## Part 4: Error Handling Nodes

### BadExpr

**Go AST Structure**:
```go
type BadExpr struct {
    From, To token.Pos // position range of bad expression
}
```

**Canonical S-Expression Format**:
```lisp
(BadExpr
  :from <pos>
  :to <pos>)
```

**Example**:
```go
// Malformed expression
x + + y

// The second + creates a BadExpr
// (BadExpr :from 15 :to 16)
```

**Implementation**:

```go
func (b *Builder) buildBadExpr(s sexp.SExp) (*ast.BadExpr, error) {
    list, ok := b.expectList(s, "BadExpr")
    if !ok {
        return nil, fmt.Errorf("not a list")
    }

    if !b.expectSymbol(list.Elements[0], "BadExpr") {
        return nil, fmt.Errorf("not a BadExpr node")
    }

    args := b.parseKeywordArgs(list.Elements)

    fromVal, ok := b.requireKeyword(args, "from", "BadExpr")
    if !ok {
        return nil, fmt.Errorf("missing from")
    }

    toVal, ok := b.requireKeyword(args, "to", "BadExpr")
    if !ok {
        return nil, fmt.Errorf("missing to")
    }

    return &ast.BadExpr{
        From: b.parsePos(fromVal),
        To:   b.parsePos(toVal),
    }, nil
}

func (w *Writer) writeBadExpr(expr *ast.BadExpr) error {
    w.openList()
    w.writeSymbol("BadExpr")
    w.writeSpace()
    w.writeKeyword("from")
    w.writeSpace()
    w.writePos(expr.From)
    w.writeSpace()
    w.writeKeyword("to")
    w.writeSpace()
    w.writePos(expr.To)
    w.closeList()
    return nil
}
```

Similarly, implement `BadStmt` and `BadDecl` if they appear in parsed code.

---

## Part 5: Complete File Implementation

Update the `File` node to properly handle all fields now that we have CommentGroup, Scope, and other supporting nodes:

**Go AST Structure** (complete):
```go
type File struct {
    Doc        *CommentGroup   // associated documentation; or nil
    Package    token.Pos       // position of "package" keyword
    Name       *Ident          // package name
    Decls      []Decl          // top-level declarations; or nil
    Scope      *Scope          // package scope (this file only)
    Imports    []*ImportSpec   // imports in this file
    Unresolved []*Ident        // unresolved identifiers in this file
    Comments   []*CommentGroup // list of all comments in the source file
}
```

**Update `buildFile` and `writeFile`**:

```go
func (b *Builder) BuildFile(s sexp.SExp) (*ast.File, error) {
    list, ok := b.expectList(s, "File")
    if !ok {
        return nil, fmt.Errorf("not a list")
    }

    if !b.expectSymbol(list.Elements[0], "File") {
        return nil, fmt.Errorf("not a File node")
    }

    args := b.parseKeywordArgs(list.Elements)

    packageVal, ok := b.requireKeyword(args, "package", "File")
    if !ok {
        return nil, fmt.Errorf("missing package")
    }

    nameVal, ok := b.requireKeyword(args, "name", "File")
    if !ok {
        return nil, fmt.Errorf("missing name")
    }

    declsVal, ok := b.requireKeyword(args, "decls", "File")
    if !ok {
        return nil, fmt.Errorf("missing decls")
    }

    name, err := b.buildIdent(nameVal)
    if err != nil {
        return nil, fmt.Errorf("invalid name: %v", err)
    }

    // Build declarations list
    var decls []ast.Decl
    declsList, ok := b.expectList(declsVal, "File decls")
    if ok {
        for _, declSexp := range declsList.Elements {
            decl, err := b.buildDecl(declSexp)
            if err != nil {
                return nil, fmt.Errorf("invalid declaration: %v", err)
            }
            decls = append(decls, decl)
        }
    }

    // Optional doc
    var doc *ast.CommentGroup
    if docVal, ok := args["doc"]; ok {
        doc, err = b.buildCommentGroup(docVal)
        if err != nil {
            return nil, fmt.Errorf("invalid doc: %v", err)
        }
    }

    // Optional scope
    var scope *ast.Scope
    if scopeVal, ok := args["scope"]; ok {
        scope, err = b.buildScope(scopeVal)
        if err != nil {
            return nil, fmt.Errorf("invalid scope: %v", err)
        }
    }

    // Optional imports
    var imports []*ast.ImportSpec
    if importsVal, ok := args["imports"]; ok {
        importsList, ok := b.expectList(importsVal, "File imports")
        if ok {
            for _, importSexp := range importsList.Elements {
                importSpec, err := b.buildImportSpec(importSexp)
                if err != nil {
                    return nil, fmt.Errorf("invalid import: %v", err)
                }
                imports = append(imports, importSpec)
            }
        }
    }

    // Optional unresolved
    var unresolved []*ast.Ident
    if unresolvedVal, ok := args["unresolved"]; ok {
        unresolvedList, ok := b.expectList(unresolvedVal, "File unresolved")
        if ok {
            for _, identSexp := range unresolvedList.Elements {
                ident, err := b.buildIdent(identSexp)
                if err != nil {
                    return nil, fmt.Errorf("invalid unresolved ident: %v", err)
                }
                unresolved = append(unresolved, ident)
            }
        }
    }

    // Optional comments
    var comments []*ast.CommentGroup
    if commentsVal, ok := args["comments"]; ok {
        commentsList, ok := b.expectList(commentsVal, "File comments")
        if ok {
            for _, cgSexp := range commentsList.Elements {
                cg, err := b.buildCommentGroup(cgSexp)
                if err != nil {
                    return nil, fmt.Errorf("invalid comment group: %v", err)
                }
                comments = append(comments, cg)
            }
        }
    }

    file := &ast.File{
        Doc:        doc,
        Package:    b.parsePos(packageVal),
        Name:       name,
        Decls:      decls,
        Scope:      scope,
        Imports:    imports,
        Unresolved: unresolved,
        Comments:   comments,
    }

    return file, nil
}
```

Update `writeFileNode` similarly to write all fields properly.

---

## Part 6: Package Node

### Package

**Go AST Structure**:
```go
type Package struct {
    Name    string             // package name
    Scope   *Scope             // package scope across all files
    Imports map[string]*Object // map of package id -> package object
    Files   map[string]*File   // Go source files by filename
}
```

**Canonical S-Expression Format**:
```lisp
(Package
  :name <string>
  :scope <Scope>
  :imports ((<string> <Object>) ...)
  :files ((<string> <File>) ...))
```

**Note**: Package is rarely used in typical AST workflows (most tools work with individual Files). Implement if needed for completeness.

---

## Part 7: Dispatcher Updates

Add Phase 5 nodes to expression dispatcher:

**In `buildExpr`**:
```go
case "IndexListExpr":
    return b.buildIndexListExpr(s)
case "BadExpr":
    return b.buildBadExpr(s)
```

**In `writeExpr`**:
```go
case *ast.IndexListExpr:
    return w.writeIndexListExpr(e)
case *ast.BadExpr:
    return w.writeBadExpr(e)
```

---

## Part 8: Integration Tests

### Test 1: Generic Types (Go 1.18+)

```go
func TestPhase5Generics(t *testing.T) {
    // Only run if Go version supports generics
    if !supportsGenerics() {
        t.Skip("Generics require Go 1.18+")
    }

    source := `package main

type List[T any] struct {
    items []T
}

func (l *List[T]) Add(item T) {
    l.items = append(l.items, item)
}

func Map[T, U any](items []T, f func(T) U) []U {
    result := make([]U, len(items))
    for i, item := range items {
        result[i] = f(item)
    }
    return result
}

func main() {
    intList := List[int]{}
    intList.Add(42)
    
    strList := List[string]{}
    strList.Add("hello")
}
`
    testRoundTrip(t, source)
}
```

### Test 2: Comments and Documentation

```go
func TestPhase5Comments(t *testing.T) {
    source := `package main

// Package main provides the main entry point.
//
// This is a longer description that spans
// multiple lines.

// Add adds two numbers together.
// It returns the sum.
func Add(a, b int) int {
    // Return the sum
    return a + b
}

/* Block comment
   spanning multiple
   lines */
var x int

// Inline comment
var y int // end of line comment
`
    
    // Parse with comments
    fset := token.NewFileSet()
    file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
    require.NoError(t, err)
    
    // Verify comments are preserved
    assert.NotNil(t, file.Doc)
    assert.Greater(t, len(file.Comments), 0)
    
    testRoundTrip(t, source)
}
```

### Test 3: Scopes and Objects

```go
func TestPhase5Scopes(t *testing.T) {
    source := `package main

var globalX int

func outer() {
    var outerX int
    
    func inner() {
        var innerX int
        _ = outerX
        _ = globalX
    }
}
`
    
    fset := token.NewFileSet()
    file, err := parser.ParseFile(fset, "test.go", source, 0)
    require.NoError(t, err)
    
    // Verify scope exists
    assert.NotNil(t, file.Scope)
    
    // Verify objects in scope
    assert.NotNil(t, file.Scope.Objects["globalX"])
    assert.NotNil(t, file.Scope.Objects["outer"])
    
    testRoundTrip(t, source)
}
```

### Test 4: Complete File with All Features

```go
func TestPhase5CompleteFile(t *testing.T) {
    source := `// Package calculator provides basic arithmetic operations.
//
// This package demonstrates various Go features including
// documentation, types, functions, and methods.
package calculator

import (
    "fmt"
    "math"
)

// Operation represents an arithmetic operation.
type Operation int

const (
    // Add represents addition
    Add Operation = iota
    // Subtract represents subtraction
    Subtract
    // Multiply represents multiplication
    Multiply
    // Divide represents division
    Divide
)

// Calculator performs arithmetic operations.
type Calculator struct {
    // result stores the current result
    result float64
}

// NewCalculator creates a new calculator.
func NewCalculator() *Calculator {
    return &Calculator{result: 0}
}

// Compute performs the given operation.
func (c *Calculator) Compute(op Operation, value float64) float64 {
    switch op {
    case Add:
        c.result += value
    case Subtract:
        c.result -= value
    case Multiply:
        c.result *= value
    case Divide:
        if value != 0 {
            c.result /= value
        }
    }
    return c.result
}

// Result returns the current result.
func (c *Calculator) Result() float64 {
    return c.result
}
`
    
    fset := token.NewFileSet()
    file, err := parser.ParseFile(fset, "calculator.go", source, parser.ParseComments)
    require.NoError(t, err)
    
    // Verify all features
    assert.NotNil(t, file.Doc)
    assert.Greater(t, len(file.Comments), 0)
    assert.NotNil(t, file.Scope)
    assert.Equal(t, 2, len(file.Imports))
    
    testRoundTrip(t, source)
}
```

---

## Part 9: Pretty Printer Updates

Update `formStyles` in `sexp/pretty.go`:

```go
var formStyles = map[string]FormStyle{
    // Existing...
    
    // Phase 5 Advanced Features
    "IndexListExpr": StyleCompact,
    "Comment":       StyleCompact,
    "CommentGroup":  StyleList,
    "Object":        StyleKeywordPairs,
    "Scope":         StyleKeywordPairs,
    "BadExpr":       StyleCompact,
    "Package":       StyleKeywordPairs,
}
```

---

## Success Criteria

### Code Completeness
- [ ] IndexListExpr implemented (if targeting Go 1.18+)
- [ ] Comment and CommentGroup fully implemented
- [ ] Object and Scope fully implemented
- [ ] BadExpr implemented
- [ ] File updated to handle all fields properly
- [ ] All nodes updated to properly write/read comments
- [ ] Expression dispatcher updated
- [ ] Pretty printer updated

### Testing
- [ ] Unit tests for each new node
- [ ] Generics test (if applicable)
- [ ] Comments preservation test
- [ ] Scope/object tracking test
- [ ] Complete file with all features test
- [ ] Test coverage >90% for new code

### Documentation
- [ ] Update README with Phase 5 capabilities
- [ ] Document generics support (or lack thereof)
- [ ] Document comment preservation
- [ ] Update canonical S-expression format spec

### Validation
- [ ] Can parse files with full documentation
- [ ] Comments are preserved through round-trip
- [ ] Scopes and objects are tracked
- [ ] Generic types work (if supported)
- [ ] All integration tests pass

---

## Implementation Tips

1. **Generics Optional**: If not targeting Go 1.18+, skip IndexListExpr
2. **Comment Parsing**: Use `parser.ParseComments` flag when parsing
3. **Scope Building**: Go's parser automatically builds scopes
4. **Object References**: Simplified approach (nil) is acceptable for Phase 5
5. **Testing Comments**: Verify Text field includes comment delimiters
6. **Deterministic Output**: Sort scope objects by name for consistent output

---

## Estimated Timeline

- **Part 1 (IndexListExpr)**: 0.5 days (or 0 if skipped)
- **Part 2 (Comments)**: 0.75 days
- **Part 3 (Scope/Objects)**: 1 day
- **Part 4 (BadExpr)**: 0.25 days
- **Part 5-6 (File/Package)**: 0.5 days
- **Part 7-9 (Updates & Tests)**: 0.5 days

**Total**: 2-3 days

---

## Next Steps After Phase 5

Once Phase 5 is complete and all tests pass:

1. **Update documentation** with advanced features
2. **Create Phase 6 specifications** for final polish
3. **Test with stdlib packages**: Parse actual Go standard library code
4. **Celebrate** - You have near-complete Go AST support!

---

Phase 5 completes the sophisticated features that make zast production-ready. With comment preservation, scope tracking, and generics support, you can now handle real-world Go codebases with full fidelity. Good luck!
