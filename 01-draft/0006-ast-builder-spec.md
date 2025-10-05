---
number: 0006
title: AST Builder Implementation Specification
author: Duncan McGreggor
created: 2025-10-01
updated: 2025-10-01
state: Draft
supersedes: None
superseded-by: None
---

# AST Builder Implementation Specification

## Overview

Implement the AST Builder that converts generic S-expression trees into typed Go AST nodes. This is the critical component that bridges our canonical S-expression format and Go's AST representation.

## File Location

Create: `go-sexp-ast/builder.go`

## Dependencies

```go
import (
    "fmt"
    "go/ast"
    "go/token"
    "strconv"
    
    "go-sexp-ast/sexp"
)
```

## Core Structure

```go
type Builder struct {
    fset   *token.FileSet
    errors []string
}

// FileSetInfo stores the parsed FileSet information
type FileSetInfo struct {
    Base  int
    Files []FileInfo
}

type FileInfo struct {
    Name  string
    Base  int
    Size  int
    Lines []int  // byte offsets of line starts
}
```

## Public API

```go
// NewBuilder creates a new AST builder
func NewBuilder() *Builder

// BuildProgram parses a Program s-expression and returns FileSet and Files
func (b *Builder) BuildProgram(sexp sexp.SExp) (*token.FileSet, []*ast.File, error)

// BuildFile converts a File s-expression to *ast.File
func (b *Builder) BuildFile(sexp sexp.SExp) (*ast.File, error)

// Errors returns accumulated errors
func (b *Builder) Errors() []string
```

## Implementation Strategy

### Phase 1: Structure Validation

For each `Build*` method, follow this pattern:

1. **Type assertion**: Verify the SExp is a List
2. **Head validation**: Verify first element is the expected Symbol
3. **Keyword extraction**: Parse keyword arguments into a map
4. **Field validation**: Verify required fields are present
5. **Recursive building**: Build child nodes
6. **Node construction**: Create the Go AST node

### Phase 2: Keyword Argument Parsing

Implement a helper to extract keyword arguments from a list:

```go
// parseKeywordArgs converts a list of alternating keywords and values into a map
// Input: [:name "foo" :type "int"]
// Output: map[string]sexp.SExp{"name": String{"foo"}, "type": String{"int"}}
func (b *Builder) parseKeywordArgs(elements []sexp.SExp) map[string]sexp.SExp
```

**Algorithm**:
- Start at index 1 (skip the node type symbol)
- Iterate in pairs
- Expect Keyword, then value
- Store in map with keyword name as key

### Phase 3: Helper Methods

Implement these essential helpers:

```go
// expectList verifies sexp is a List and returns it
func (b *Builder) expectList(sexp sexp.SExp, context string) (*sexp.List, bool)

// expectSymbol verifies sexp is a Symbol with expected value
func (b *Builder) expectSymbol(sexp sexp.SExp, expected string) bool

// getKeyword retrieves a keyword value from the args map
func (b *Builder) getKeyword(args map[string]sexp.SExp, name string) (sexp.SExp, bool)

// requireKeyword gets a keyword value or adds an error if missing
func (b *Builder) requireKeyword(args map[string]sexp.SExp, name string, context string) (sexp.SExp, bool)

// parseInt converts a Number or Symbol to int
func (b *Builder) parseInt(sexp sexp.SExp) (int, error)

// parsePos converts a Number to token.Pos
func (b *Builder) parsePos(sexp sexp.SExp) token.Pos

// parseString extracts string value from String node
func (b *Builder) parseString(sexp sexp.SExp) (string, error)

// parseNil checks if value is nil
func (b *Builder) parseNil(sexp sexp.SExp) bool

// addError records an error
func (b *Builder) addError(format string, args ...interface{})
```

## Node Builders - Phase 1 Scope

Implement builders for these nodes (Phase 1 - Hello World support):

### Program and FileSet

```go
// BuildProgram parses:
// (Program
//   :fileset (FileSet ...)
//   :files ((File ...) ...))
func (b *Builder) BuildProgram(sexp sexp.SExp) (*token.FileSet, []*ast.File, error)
```

**Steps**:
1. Verify it's a List starting with Symbol "Program"
2. Parse keyword args
3. Extract `:fileset` and build FileSetInfo
4. Extract `:files` and build each File
5. Create token.FileSet from FileSetInfo
6. Return FileSet and File list

```go
// buildFileSet parses:
// (FileSet :base 1 :files (...))
func (b *Builder) buildFileSet(sexp sexp.SExp) (*FileSetInfo, error)
```

```go
// buildFileInfo parses:
// (FileInfo :name "main.go" :base 1 :size 100 :lines (1 15 30 ...))
func (b *Builder) buildFileInfo(sexp sexp.SExp) (*FileInfo, error)
```

### File

```go
// BuildFile parses:
// (File
//   :package <pos>
//   :name <Ident>
//   :decls (...)
//   :scope <Scope>
//   :imports (...)
//   :unresolved (...)
//   :comments (...))
func (b *Builder) BuildFile(sexp sexp.SExp) (*ast.File, error)
```

**Steps**:
1. Parse keyword args
2. Build each field:
   - `:package` → token.Pos
   - `:name` → *ast.Ident
   - `:decls` → []ast.Decl (iterate list, build each)
   - `:scope` → *ast.Scope (may be nil)
   - `:imports` → []*ast.ImportSpec
   - `:unresolved` → []*ast.Ident
   - `:comments` → []*ast.CommentGroup
3. Construct *ast.File

### Ident

```go
// buildIdent parses:
// (Ident :namepos <pos> :name <string> :obj <Object>)
func (b *Builder) buildIdent(sexp sexp.SExp) (*ast.Ident, error)
```

### BasicLit

```go
// buildBasicLit parses:
// (BasicLit :valuepos <pos> :kind <token> :value <string>)
func (b *Builder) buildBasicLit(sexp sexp.SExp) (*ast.BasicLit, error)
```

**Token kind mapping**:
- Symbol "INT" → token.INT
- Symbol "FLOAT" → token.FLOAT
- Symbol "IMAG" → token.IMAG
- Symbol "CHAR" → token.CHAR
- Symbol "STRING" → token.STRING

### GenDecl

```go
// buildGenDecl parses:
// (GenDecl
//   :doc <CommentGroup>
//   :tok <token>
//   :tokpos <pos>
//   :lparen <pos>
//   :specs (...)
//   :rparen <pos>)
func (b *Builder) buildGenDecl(sexp sexp.SExp) (*ast.GenDecl, error)
```

**Token mapping**:
- "IMPORT" → token.IMPORT
- "CONST" → token.CONST
- "TYPE" → token.TYPE
- "VAR" → token.VAR

### ImportSpec

```go
// buildImportSpec parses:
// (ImportSpec
//   :doc <CommentGroup>
//   :name <Ident>
//   :path <BasicLit>
//   :comment <CommentGroup>
//   :endpos <pos>)
func (b *Builder) buildImportSpec(sexp sexp.SExp) (*ast.ImportSpec, error)
```

### FuncDecl

```go
// buildFuncDecl parses:
// (FuncDecl
//   :doc <CommentGroup>
//   :recv <FieldList>
//   :name <Ident>
//   :type <FuncType>
//   :body <BlockStmt>)
func (b *Builder) buildFuncDecl(sexp sexp.SExp) (*ast.FuncDecl, error)
```

### FuncType

```go
// buildFuncType parses:
// (FuncType
//   :func <pos>
//   :params <FieldList>
//   :results <FieldList>)
func (b *Builder) buildFuncType(sexp sexp.SExp) (*ast.FuncType, error)
```

### FieldList

```go
// buildFieldList parses:
// (FieldList
//   :opening <pos>
//   :list (...)
//   :closing <pos>)
func (b *Builder) buildFieldList(sexp sexp.SExp) (*ast.FieldList, error)
```

### Field

```go
// buildField parses:
// (Field
//   :doc <CommentGroup>
//   :names (...)
//   :type <Expr>
//   :tag <BasicLit>
//   :comment <CommentGroup>)
func (b *Builder) buildField(sexp sexp.SExp) (*ast.Field, error)
```

### BlockStmt

```go
// buildBlockStmt parses:
// (BlockStmt
//   :lbrace <pos>
//   :list (...)
//   :rbrace <pos>)
func (b *Builder) buildBlockStmt(sexp sexp.SExp) (*ast.BlockStmt, error)
```

### ExprStmt

```go
// buildExprStmt parses:
// (ExprStmt :x <Expr>)
func (b *Builder) buildExprStmt(sexp sexp.SExp) (*ast.ExprStmt, error)
```

### CallExpr

```go
// buildCallExpr parses:
// (CallExpr
//   :fun <Expr>
//   :lparen <pos>
//   :args (...)
//   :ellipsis <pos>
//   :rparen <pos>)
func (b *Builder) buildCallExpr(sexp sexp.SExp) (*ast.CallExpr, error)
```

### SelectorExpr

```go
// buildSelectorExpr parses:
// (SelectorExpr
//   :x <Expr>
//   :sel <Ident>)
func (b *Builder) buildSelectorExpr(sexp sexp.SExp) (*ast.SelectorExpr, error)
```

## Polymorphic Builders

Some fields can be multiple types. Implement dispatcher methods:

```go
// buildExpr dispatches to appropriate expression builder
func (b *Builder) buildExpr(sexp sexp.SExp) (ast.Expr, error) {
    list, ok := b.expectList(sexp, "expression")
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
    case "Ident":
        return b.buildIdent(sexp)
    case "BasicLit":
        return b.buildBasicLit(sexp)
    case "CallExpr":
        return b.buildCallExpr(sexp)
    case "SelectorExpr":
        return b.buildSelectorExpr(sexp)
    // Add more as needed
    default:
        return nil, fmt.Errorf("unknown expression type: %s", sym.Value)
    }
}

// buildStmt dispatches to appropriate statement builder
func (b *Builder) buildStmt(sexp sexp.SExp) (ast.Stmt, error)

// buildDecl dispatches to appropriate declaration builder
func (b *Builder) buildDecl(sexp sexp.SExp) (ast.Decl, error)

// buildSpec dispatches to appropriate spec builder
func (b *Builder) buildSpec(sexp sexp.SExp) (ast.Spec, error)
```

## Optional/Nil Handling

Many fields can be nil. Implement consistent nil checking:

```go
// buildOptionalIdent builds Ident or returns nil
func (b *Builder) buildOptionalIdent(sexp sexp.SExp) (*ast.Ident, error) {
    if b.parseNil(sexp) {
        return nil, nil
    }
    return b.buildIdent(sexp)
}

// Similar for other optional types:
// buildOptionalCommentGroup
// buildOptionalFieldList
// buildOptionalBlockStmt
// etc.
```

## Error Handling Strategy

1. **Accumulate errors**: Don't stop at first error, collect all
2. **Provide context**: Include node type and field name in errors
3. **Include positions**: Use the SExp position information
4. **Return early on critical errors**: Some errors prevent further processing

Example error format:
```
"File node: missing required field :name at line 5, column 10"
"Ident node: expected string for :name field, got Number at line 7, column 15"
```

## Testing Requirements

Create `builder_test.go` with tests for:

### 1. Individual Node Building

Test each builder method with valid input:

```go
func TestBuildIdent(t *testing.T) {
    input := `(Ident :namepos 10 :name "main" :obj nil)`
    parser := sexp.NewParser(input)
    sexpNode, _ := parser.Parse()
    
    builder := NewBuilder()
    ident, err := builder.buildIdent(sexpNode)
    
    assert.NoError(t, err)
    assert.Equal(t, "main", ident.Name)
    assert.Equal(t, token.Pos(10), ident.NamePos)
    assert.Nil(t, ident.Obj)
}
```

### 2. Complete File Building

Test building a complete File from hello world S-expression:

```go
func TestBuildCompleteFile(t *testing.T) {
    input := `(Program
      :fileset (FileSet :base 1 :files (
        (FileInfo :name "main.go" :base 1 :size 78 :lines (1 14 27))))
      :files (
        (File :package 1 :name (Ident :namepos 9 :name "main" :obj nil)
          :decls (
            (GenDecl :doc nil :tok IMPORT :tokpos 15 :lparen 0
              :specs ((ImportSpec :doc nil :name nil 
                       :path (BasicLit :valuepos 22 :kind STRING :value "\"fmt\"")
                       :comment nil :endpos 27))
              :rparen 0))
          :scope nil :imports () :unresolved () :comments ())))`
    
    parser := sexp.NewParser(input)
    sexpNode, _ := parser.Parse()
    
    builder := NewBuilder()
    fset, files, err := builder.BuildProgram(sexpNode)
    
    assert.NoError(t, err)
    assert.NotNil(t, fset)
    assert.Len(t, files, 1)
    assert.Equal(t, "main", files[0].Name.Name)
}
```

### 3. Error Cases

Test error handling:

```go
func TestBuildIdentMissingName(t *testing.T) {
    input := `(Ident :namepos 10 :obj nil)`  // missing :name
    parser := sexp.NewParser(input)
    sexpNode, _ := parser.Parse()
    
    builder := NewBuilder()
    _, err := builder.buildIdent(sexpNode)
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "required field :name")
}
```

### 4. Nil Handling

Test optional fields:

```go
func TestBuildFuncDeclWithNilDoc(t *testing.T) {
    // Test that :doc nil is properly handled
}
```

## Success Criteria

- All Phase 1 node types can be built
- Proper error reporting with context
- Nil values handled correctly
- Position information preserved
- Clean, readable code
- Comprehensive test coverage
- No panics on malformed input

## Notes

- Use consistent patterns across all builders
- Keep error messages informative
- Consider adding debug logging
- Position information is critical - preserve it accurately
- This is the most complex part of the system - take time to get it right
- Start with the simplest nodes (Ident, BasicLit) and work up to complex ones (File, FuncDecl)
