---
number: 0007
title: S-Expression Writer Implementation Specification
author: Duncan McGreggor
created: 2025-10-01
updated: 2025-10-04
state: Final
supersedes: None
superseded-by: None
---

# S-Expression Writer Implementation Specification

## Overview

Implement the Writer that converts Go AST nodes into canonical S-expression format. This is the reverse direction from the Builder - it takes typed Go AST nodes and produces S-expression text that conforms to our canonical format specification.

## File Location

Create: `go-sexp-ast/writer.go`

## Dependencies

```go
import (
    "fmt"
    "go/ast"
    "go/token"
    "io"
    "strings"
)
```

## Core Structure

```go
type Writer struct {
    fset   *token.FileSet
    indent int
    buf    strings.Builder
}

// WriterOptions configures the writer behavior
type WriterOptions struct {
    Indent      bool  // Whether to indent output (default: false for Phase 1)
    IndentWidth int   // Spaces per indent level (default: 2)
}
```

## Public API

```go
// NewWriter creates a new S-expression writer
func NewWriter(fset *token.FileSet) *Writer

// WriteProgram writes a complete Program with FileSet and Files
func (w *Writer) WriteProgram(files []*ast.File) (string, error)

// WriteFile writes a single File node
func (w *Writer) WriteFile(file *ast.File) (string, error)

// WriteTo writes the result to an io.Writer
func (w *Writer) WriteTo(wr io.Writer) (int64, error)
```

## Implementation Strategy

### Core Writing Pattern

Every `write*` method follows this pattern:

1. **Open list**: Write `(`
2. **Write node type**: Write the AST node type as a symbol (e.g., `File`, `Ident`)
3. **Write fields**: For each field, write `:fieldname value`
4. **Close list**: Write `)`

### Field Writing Rules

**Positions (token.Pos)**:
- Write as integers
- Use the actual Pos value directly
- Example: `:package 1`

**Strings**:
- Escape and quote
- Example: `:name "main"`

**Symbols/Identifiers**:
- Write as-is (no quotes)
- Example: `:tok IMPORT`

**Numbers**:
- Write as-is
- Example: `:base 1`

**Nil values**:
- Write the symbol `nil`
- Example: `:doc nil`

**Lists of nodes**:
- Open with `(`
- Write each element
- Close with `)`
- Example: `:decls ((GenDecl ...) (FuncDecl ...))`

**Empty lists**:
- Write `()`
- Example: `:list ()`

## Helper Methods

Implement these essential helpers:

```go
// writeString writes a properly escaped and quoted string
func (w *Writer) writeString(s string) {
    w.buf.WriteString(`"`)
    for _, ch := range s {
        switch ch {
        case '"':
            w.buf.WriteString(`\"`)
        case '\\':
            w.buf.WriteString(`\\`)
        case '\n':
            w.buf.WriteString(`\n`)
        case '\t':
            w.buf.WriteString(`\t`)
        case '\r':
            w.buf.WriteString(`\r`)
        default:
            w.buf.WriteRune(ch)
        }
    }
    w.buf.WriteString(`"`)
}

// writeSymbol writes a symbol (unquoted identifier)
func (w *Writer) writeSymbol(s string)

// writeKeyword writes a keyword (:name)
func (w *Writer) writeKeyword(name string)

// writePos writes a token.Pos as an integer
func (w *Writer) writePos(pos token.Pos)

// writeToken writes a token type as a symbol
func (w *Writer) writeToken(tok token.Token)

// writeSpace writes a space separator
func (w *Writer) writeSpace()

// openList writes opening parenthesis
func (w *Writer) openList()

// closeList writes closing parenthesis
func (w *Writer) closeList()
```

## Node Writers - Phase 1 Scope

Implement writers for all Phase 1 nodes:

### Program and FileSet

```go
// WriteProgram writes:
// (Program
//   :fileset (FileSet ...)
//   :files ((File ...) ...))
func (w *Writer) WriteProgram(files []*ast.File) (string, error)
```

**Steps**:
1. Write `(Program`
2. Write `:fileset` followed by the FileSet structure
3. Write `:files (` then each file, then `)`
4. Write `)`
5. Return the buffer contents

```go
// writeFileSet writes:
// (FileSet :base 1 :files (...))
func (w *Writer) writeFileSet() error
```

**Note**: Extract FileSet information from `w.fset`:
- Base: `w.fset.Base()`
- For each file in FileSet, write FileInfo

```go
// writeFileInfo writes:
// (FileInfo :name "main.go" :base 1 :size 100 :lines (1 15 30 ...))
func (w *Writer) writeFileInfo(file *token.File) error
```

**FileInfo extraction**:
- Name: `file.Name()`
- Base: `file.Base()`
- Size: `file.Size()`
- Lines: Iterate through file positions to get line starts

### File

```go
// WriteFile writes:
// (File
//   :package <pos>
//   :name <Ident>
//   :decls (...)
//   :scope <Scope>
//   :imports (...)
//   :unresolved (...)
//   :comments (...))
func (w *Writer) WriteFile(file *ast.File) (string, error)
```

**Implementation**:
```go
w.openList()
w.writeSymbol("File")

w.writeSpace()
w.writeKeyword("package")
w.writeSpace()
w.writePos(file.Package)

w.writeSpace()
w.writeKeyword("name")
w.writeSpace()
w.writeIdent(file.Name)

w.writeSpace()
w.writeKeyword("decls")
w.writeSpace()
w.writeDeclList(file.Decls)

// ... continue for all fields

w.closeList()
```

### Ident

```go
// writeIdent writes:
// (Ident :namepos <pos> :name <string> :obj <Object>)
func (w *Writer) writeIdent(ident *ast.Ident) error
```

**Handle nil**:
```go
if ident == nil {
    w.writeSymbol("nil")
    return nil
}
```

### BasicLit

```go
// writeBasicLit writes:
// (BasicLit :valuepos <pos> :kind <token> :value <string>)
func (w *Writer) writeBasicLit(lit *ast.BasicLit) error
```

**Token kind mapping** (reverse of builder):
- token.INT → "INT"
- token.FLOAT → "FLOAT"
- token.IMAG → "IMAG"
- token.CHAR → "CHAR"
- token.STRING → "STRING"

**Important**: The `:value` field should be written as-is (it already includes quotes for strings).

### GenDecl

```go
// writeGenDecl writes:
// (GenDecl
//   :doc <CommentGroup>
//   :tok <token>
//   :tokpos <pos>
//   :lparen <pos>
//   :specs (...)
//   :rparen <pos>)
func (w *Writer) writeGenDecl(decl *ast.GenDecl) error
```

**Token mapping**:
- token.IMPORT → "IMPORT"
- token.CONST → "CONST"
- token.TYPE → "TYPE"
- token.VAR → "VAR"

### ImportSpec

```go
// writeImportSpec writes:
// (ImportSpec
//   :doc <CommentGroup>
//   :name <Ident>
//   :path <BasicLit>
//   :comment <CommentGroup>
//   :endpos <pos>)
func (w *Writer) writeImportSpec(spec *ast.ImportSpec) error
```

### FuncDecl

```go
// writeFuncDecl writes:
// (FuncDecl
//   :doc <CommentGroup>
//   :recv <FieldList>
//   :name <Ident>
//   :type <FuncType>
//   :body <BlockStmt>)
func (w *Writer) writeFuncDecl(decl *ast.FuncDecl) error
```

### FuncType

```go
// writeFuncType writes:
// (FuncType
//   :func <pos>
//   :params <FieldList>
//   :results <FieldList>)
func (w *Writer) writeFuncType(typ *ast.FuncType) error
```

### FieldList

```go
// writeFieldList writes:
// (FieldList
//   :opening <pos>
//   :list (...)
//   :closing <pos>)
func (w *Writer) writeFieldList(fields *ast.FieldList) error
```

**Handle nil**:
```go
if fields == nil {
    w.writeSymbol("nil")
    return nil
}
```

### Field

```go
// writeField writes:
// (Field
//   :doc <CommentGroup>
//   :names (...)
//   :type <Expr>
//   :tag <BasicLit>
//   :comment <CommentGroup>)
func (w *Writer) writeField(field *ast.Field) error
```

### BlockStmt

```go
// writeBlockStmt writes:
// (BlockStmt
//   :lbrace <pos>
//   :list (...)
//   :rbrace <pos>)
func (w *Writer) writeBlockStmt(stmt *ast.BlockStmt) error
```

### ExprStmt

```go
// writeExprStmt writes:
// (ExprStmt :x <Expr>)
func (w *Writer) writeExprStmt(stmt *ast.ExprStmt) error
```

### CallExpr

```go
// writeCallExpr writes:
// (CallExpr
//   :fun <Expr>
//   :lparen <pos>
//   :args (...)
//   :ellipsis <pos>
//   :rparen <pos>)
func (w *Writer) writeCallExpr(expr *ast.CallExpr) error
```

### SelectorExpr

```go
// writeSelectorExpr writes:
// (SelectorExpr
//   :x <Expr>
//   :sel <Ident>)
func (w *Writer) writeSelectorExpr(expr *ast.SelectorExpr) error
```

### CommentGroup

```go
// writeCommentGroup writes:
// (CommentGroup :list (<Comment> ...))
func (w *Writer) writeCommentGroup(cg *ast.CommentGroup) error
```

**Handle nil**:
```go
if cg == nil {
    w.writeSymbol("nil")
    return nil
}
```

### Comment

```go
// writeComment writes:
// (Comment :slash <pos> :text <string>)
func (w *Writer) writeComment(c *ast.Comment) error
```

### Scope

```go
// writeScope writes:
// (Scope
//   :outer <Scope>
//   :objects ((<name> <Object>) ...))
func (w *Writer) writeScope(scope *ast.Scope) error
```

**Handle nil**:
```go
if scope == nil {
    w.writeSymbol("nil")
    return nil
}
```

**Note**: Scope serialization is complex. For Phase 1, we can write `nil` for scopes or a simplified version.

### Object

```go
// writeObject writes:
// (Object
//   :kind <ObjKind>
//   :name <string>
//   :decl <Node>
//   :data <any>
//   :type <any>)
func (w *Writer) writeObject(obj *ast.Object) error
```

**Object kind mapping**:
- ast.Bad → "Bad"
- ast.Pkg → "Pkg"
- ast.Con → "Con"
- ast.Typ → "Typ"
- ast.Var → "Var"
- ast.Fun → "Fun"
- ast.Lbl → "Lbl"

**Note**: For Phase 1, we can write simplified Objects or `nil`.

## Polymorphic Writers

Implement dispatcher methods for polymorphic fields:

```go
// writeExpr dispatches to appropriate expression writer
func (w *Writer) writeExpr(expr ast.Expr) error {
    if expr == nil {
        w.writeSymbol("nil")
        return nil
    }
    
    switch e := expr.(type) {
    case *ast.Ident:
        return w.writeIdent(e)
    case *ast.BasicLit:
        return w.writeBasicLit(e)
    case *ast.CallExpr:
        return w.writeCallExpr(e)
    case *ast.SelectorExpr:
        return w.writeSelectorExpr(e)
    // Add more as needed
    default:
        return fmt.Errorf("unknown expression type: %T", expr)
    }
}

// writeStmt dispatches to appropriate statement writer
func (w *Writer) writeStmt(stmt ast.Stmt) error

// writeDecl dispatches to appropriate declaration writer
func (w *Writer) writeDecl(decl ast.Decl) error

// writeSpec dispatches to appropriate spec writer
func (w *Writer) writeSpec(spec ast.Spec) error
```

## List Writers

Implement helpers for writing lists:

```go
// writeExprList writes a list of expressions
func (w *Writer) writeExprList(exprs []ast.Expr) error {
    w.openList()
    for i, expr := range exprs {
        if i > 0 {
            w.writeSpace()
        }
        if err := w.writeExpr(expr); err != nil {
            return err
        }
    }
    w.closeList()
    return nil
}

// writeStmtList writes a list of statements
func (w *Writer) writeStmtList(stmts []ast.Stmt) error

// writeDeclList writes a list of declarations
func (w *Writer) writeDeclList(decls []ast.Decl) error

// writeSpecList writes a list of specs
func (w *Writer) writeSpecList(specs []ast.Spec) error

// writeIdentList writes a list of identifiers
func (w *Writer) writeIdentList(idents []*ast.Ident) error

// writeFieldList writes a list of fields
func (w *Writer) writeFieldList(fields []*ast.Field) error
```

## FileSet Iteration

To extract FileSet information, you'll need to iterate through files:

```go
// collectFiles extracts all files from the FileSet
func (w *Writer) collectFiles() []*token.File {
    var files []*token.File
    
    // Iterate through all positions to find files
    // This is a bit tricky - the FileSet doesn't expose files directly
    // One approach: use the files we're writing and ask for their token.File
    
    // For Phase 1, if we're only writing one file, we can:
    // file := w.fset.File(somePos)
    
    return files
}
```

**Note**: For Phase 1, since we're likely working with a single file, we can simplify this. The complete implementation may need to track which files are actually used.

## Testing Requirements

Create `writer_test.go` with tests for:

### 1. Individual Node Writing

Test each writer method:

```go
func TestWriteIdent(t *testing.T) {
    fset := token.NewFileSet()
    file := fset.AddFile("test.go", -1, 100)
    
    ident := &ast.Ident{
        NamePos: file.Pos(10),
        Name:    "main",
        Obj:     nil,
    }
    
    writer := NewWriter(fset)
    err := writer.writeIdent(ident)
    assert.NoError(t, err)
    
    output := writer.buf.String()
    expected := `(Ident :namepos 10 :name "main" :obj nil)`
    assert.Equal(t, expected, output)
}
```

### 2. Complete File Writing

Test writing a complete hello world file:

```go
func TestWriteCompleteFile(t *testing.T) {
    // Build a complete *ast.File for hello world
    fset := token.NewFileSet()
    file := buildHelloWorldAST(fset)  // helper function
    
    writer := NewWriter(fset)
    output, err := writer.WriteProgram([]*ast.File{file})
    
    assert.NoError(t, err)
    
    // Parse the output back
    parser := sexp.NewParser(output)
    sexpNode, err := parser.Parse()
    assert.NoError(t, err)
    
    // Verify structure
    list := sexpNode.(*sexp.List)
    assert.Equal(t, "Program", list.Elements[0].(*sexp.Symbol).Value)
}
```

### 3. Round-Trip Test

Most important test - verify we can round-trip:

```go
func TestRoundTrip(t *testing.T) {
    // 1. Parse Go source to AST
    fset := token.NewFileSet()
    file, err := parser.ParseFile(fset, "test.go", helloWorldSource, 0)
    assert.NoError(t, err)
    
    // 2. Write AST to S-expression
    writer := NewWriter(fset)
    sexpText, err := writer.WriteProgram([]*ast.File{file})
    assert.NoError(t, err)
    
    // 3. Parse S-expression
    parser := sexp.NewParser(sexpText)
    sexpNode, err := parser.Parse()
    assert.NoError(t, err)
    
    // 4. Build back to AST
    builder := NewBuilder()
    fset2, files2, err := builder.BuildProgram(sexpNode)
    assert.NoError(t, err)
    
    // 5. Compare ASTs (or write to Go and compare source)
    // This is the critical test!
}
```

### 4. Nil Handling

Test that nil values are properly written:

```go
func TestWriteNilValues(t *testing.T) {
    // Test that nil *ast.Ident writes as "nil"
    // Test that nil *ast.CommentGroup writes as "nil"
    // etc.
}
```

### 5. Empty Lists

Test that empty lists are written correctly:

```go
func TestWriteEmptyLists(t *testing.T) {
    // Empty declarations list
    // Empty parameter list
    // etc.
}
```

## String Escaping Tests

Important to test string escaping thoroughly:

```go
func TestStringEscaping(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {`hello`, `"hello"`},
        {`hello "world"`, `"hello \"world\""`},
        {"hello\nworld", `"hello\nworld"`},
        {`hello\world`, `"hello\\world"`},
        {"tab\there", `"tab\there"`},
    }
    
    for _, tt := range tests {
        writer := NewWriter(nil)
        writer.writeString(tt.input)
        assert.Equal(t, tt.expected, writer.buf.String())
    }
}
```

## Token Mapping Tests

Verify token mappings are correct:

```go
func TestTokenMapping(t *testing.T) {
    tests := []struct {
        token    token.Token
        expected string
    }{
        {token.IMPORT, "IMPORT"},
        {token.CONST, "CONST"},
        {token.TYPE, "TYPE"},
        {token.VAR, "VAR"},
        {token.INT, "INT"},
        {token.STRING, "STRING"},
    }
    
    for _, tt := range tests {
        writer := NewWriter(nil)
        writer.writeToken(tt.token)
        assert.Equal(t, tt.expected, writer.buf.String())
    }
}
```

## Success Criteria

- All Phase 1 node types can be written
- Output conforms to canonical S-expression format specification
- Nil values handled correctly
- Empty lists written as `()`
- String escaping correct
- Position information preserved
- Round-trip test passes (Go → S-expr → Go)
- Clean, readable code
- Comprehensive test coverage

## Notes

- Focus on correctness over performance
- The output doesn't need to be pretty-printed for Phase 1
- Every field must be written in the exact order specified in the canonical format
- Position tracking is critical - use the FileSet correctly
- String escaping must be perfect or round-trips will fail
- Test thoroughly with the complete hello world example
