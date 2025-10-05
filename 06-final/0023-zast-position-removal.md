---
number: 0023
title: Removing Position Tracking from zast
author: Duncan McGreggor
created: 2025-10-04
updated: 2025-10-04
state: Final
supersedes: None
superseded-by: None
---

# Removing Position Tracking from zast

**Project**: zast  
**Goal**: Accept that position information cannot be preserved through S-expression round-trips  
**Estimated Effort**: 2-3 hours  
**Context**: Interim solution until zylisp/core source map architecture is implemented

---

## Problem Statement

Currently, zast attempts to preserve `token.Pos` values through S-expression serialization. This fundamentally cannot work because:

1. `token.Pos` values are absolute byte offsets into a specific `token.FileSet`
2. When deserializing, we create a **new** FileSet
3. Go's `FileSet.AddFile()` doesn't let us add files at arbitrary positions
4. The preserved positions reference the wrong FileSet and become meaningless
5. Comments end up at wrong locations or at the end of files

**The Truth**: S-expression representation is for **code transformation**, not source archival. We need to accept position loss and document it clearly.

---

## Solution Overview

1. **Stop trying to preserve positions** - Set all positions to `token.NoPos` or 0
2. **Simplify FileSet handling** - Create minimal FileSet for validity
3. **Keep comment associations** - Comments stay attached to correct nodes via Doc/Comment fields
4. **Document the limitation** - Make it clear this is by design
5. **Update tests** - Compare AST structure, not positions

---

## Implementation Tasks

### Task 1: Update Documentation

**File**: `README.md`

Add a new section after the "Overview" section:

```markdown
## Important Limitations

### Position Information

**zast does not preserve exact source positions through round-trip conversion.**

When converting Go AST → S-expr → Go AST:
- Position information (`token.Pos`) is lost
- The reconstructed AST has all positions set to `token.NoPos` (0)
- Comments are preserved and attached to correct AST nodes, but their positions are reset
- Use `go/printer` to format the output - it will generate clean, valid formatting

This is **by design**: 
- S-expression representation is for code transformation and analysis
- Not for preserving exact source formatting or file positions
- If you need to preserve original positions, keep the original AST and FileSet

### What is Preserved

✅ Complete AST structure  
✅ All comments (attached to correct nodes)  
✅ All code semantics  
✅ Ability to generate valid Go code  

### What is Lost

❌ Exact original formatting (spacing, line breaks)  
❌ Source file positions  
❌ Ability to point to original source locations  

### Future Work

Full position tracking through compilation stages will be handled by the 
`zylisp/core` source map architecture (see `docs/source-map-architecture.md`).
```

### Task 2: Simplify Builder FileSet Handling

**File**: `builder/file.go`

Replace the `buildFileSet` function with a simplified version:

```go
// buildFileSet parses a FileSet node (simplified - positions are not preserved)
func (b *Builder) buildFileSet(s sexp.SExp) (*FileSetInfo, error) {
    list, ok := b.expectList(s, "FileSet")
    if !ok {
        return nil, errors.ErrNotList
    }

    if !b.expectSymbol(list.Elements[0], "FileSet") {
        return nil, errors.ErrExpectedNodeType("FileSet", "unknown")
    }

    // We parse the FileSet structure but don't use it for position reconstruction
    // Positions cannot be meaningfully preserved through S-expression serialization
    return &FileSetInfo{
        Base:  1,
        Files: nil,
    }, nil
}
```

Update `BuildProgram` to create a simple FileSet:

```go
func (b *Builder) BuildProgram(s sexp.SExp) (*token.FileSet, []*ast.File, error) {
    list, ok := b.expectList(s, "Program")
    if !ok {
        return nil, nil, errors.ErrNotList
    }

    if !b.expectSymbol(list.Elements[0], "Program") {
        return nil, nil, errors.ErrExpectedNodeType("Program", "unknown")
    }

    args := b.parseKeywordArgs(list.Elements)

    filesetVal, ok := b.requireKeyword(args, "fileset", "Program")
    if !ok {
        return nil, nil, errors.ErrMissingField("fileset")
    }

    filesVal, ok := b.requireKeyword(args, "files", "Program")
    if !ok {
        return nil, nil, errors.ErrMissingField("files")
    }

    // Parse FileSet (but don't use it - positions can't be preserved)
    _, err := b.buildFileSet(filesetVal)
    if err != nil {
        return nil, nil, errors.ErrInvalidField("fileset", err)
    }

    // Create a simple FileSet for the reconstructed AST
    // Positions will all be token.NoPos (0) - this is intentional
    fset := token.NewFileSet()
    b.fset = fset

    // Build files list
    var files []*ast.File
    filesList, ok := b.expectList(filesVal, "Program files")
    if ok {
        for _, fileSexp := range filesList.Elements {
            file, err := b.BuildFile(fileSexp)
            if err != nil {
                return nil, nil, errors.ErrInvalidField("file", err)
            }
            files = append(files, file)
            
            // Add each file to the FileSet with a generous size estimate
            // Actual positions don't matter since they're all NoPos anyway
            fset.AddFile(file.Name.Name, fset.Base(), 1000000)
        }
    }

    if len(b.errors) > 0 {
        return nil, nil, fmt.Errorf("build errors: %s", strings.Join(b.errors, "; "))
    }

    return fset, files, nil
}
```

### Task 3: Fix Comment Position Handling

**File**: `builder/comments.go`

Update `buildComment` to use `token.NoPos`:

```go
func (b *Builder) buildComment(s sexp.SExp) (*ast.Comment, error) {
    list, ok := b.expectList(s, "Comment")
    if !ok {
        return nil, errors.ErrNotList
    }

    if !b.expectSymbol(list.Elements[0], "Comment") {
        return nil, errors.ErrExpectedNodeType("Comment", "unknown")
    }

    args := b.parseKeywordArgs(list.Elements)

    // We parse slash position but don't use it - positions can't be preserved
    _, ok = b.requireKeyword(args, "slash", "Comment")
    if !ok {
        return nil, errors.ErrMissingField("slash")
    }

    textVal, ok := b.requireKeyword(args, "text", "Comment")
    if !ok {
        return nil, errors.ErrMissingField("text")
    }

    text, err := b.parseString(textVal)
    if err != nil {
        return nil, errors.ErrInvalidField("text", err)
    }

    return &ast.Comment{
        Slash: token.NoPos, // Positions are not preserved
        Text:  text,
    }, nil
}
```

### Task 4: Simplify Writer FileSet Output

**File**: `writer/file.go`

Simplify `writeFileSet` since the positions are meaningless:

```go
func (w *Writer) writeFileSet() error {
    w.openList()
    w.writeSymbol("FileSet")
    w.writeSpace()
    w.writeKeyword("base")
    w.writeSpace()
    w.writePos(token.Pos(1)) // Dummy base - positions are not preserved through round-trip
    w.writeSpace()
    w.writeKeyword("files")
    w.writeSpace()
    w.openList()
    // Empty files list - file metadata is not needed for round-trip
    w.closeList()
    w.closeList()
    return nil
}
```

Remove or simplify `writeFileInfo` and `collectFiles` since they're no longer used:

```go
// These functions can be removed entirely, or kept but not called
// func (w *Writer) writeFileInfo(file *token.File) error { ... }
// func (w *Writer) collectFiles() []*token.File { ... }
```

### Task 5: Update Tests

**File**: `integration_test.go` (or wherever round-trip tests are)

Update the round-trip test to compare AST structure, not positions:

```go
func testRoundTrip(t *testing.T, source string) {
    // Parse original Go source
    fset1 := token.NewFileSet()
    file1, err := parser.ParseFile(fset1, "test.go", source, parser.ParseComments)
    require.NoError(t, err)

    // Write to S-expression
    writer := NewWriter(fset1)
    sexp, err := writer.WriteProgram([]*ast.File{file1})
    require.NoError(t, err)

    // Parse S-expression
    sexpParser := sexp.NewParser(sexp)
    sexpNode, err := sexpParser.Parse()
    require.NoError(t, err)

    // Build back to AST
    builder := NewBuilder()
    fset2, files2, err := builder.BuildProgram(sexpNode)
    require.NoError(t, err)
    require.Len(t, files2, 1)

    // Compare by printing both to Go source
    // This tests that AST structure is preserved, ignoring position differences
    var buf1, buf2 bytes.Buffer
    err = printer.Fprint(&buf1, fset1, file1)
    require.NoError(t, err)
    err = printer.Fprint(&buf2, fset2, files2[0])
    require.NoError(t, err)

    // The printed Go source should be equivalent
    // (formatting may differ slightly, but structure should be identical)
    assert.Equal(t, buf1.String(), buf2.String())
}
```

Add a test that explicitly verifies positions are not preserved:

```go
func TestPositionsNotPreserved(t *testing.T) {
    source := `package main

// This is a comment
func main() {
    x := 42
}
`
    
    fset1 := token.NewFileSet()
    file1, err := parser.ParseFile(fset1, "test.go", source, parser.ParseComments)
    require.NoError(t, err)

    // Original file has real positions
    assert.NotEqual(t, token.NoPos, file1.Package)
    assert.NotEqual(t, token.NoPos, file1.Name.NamePos)

    // Write and rebuild
    writer := NewWriter(fset1)
    sexp, err := writer.WriteProgram([]*ast.File{file1})
    require.NoError(t, err)

    sexpParser := sexp.NewParser(sexp)
    sexpNode, err := sexpParser.Parse()
    require.NoError(t, err)

    builder := NewBuilder()
    fset2, files2, err := builder.BuildProgram(sexpNode)
    require.NoError(t, err)

    file2 := files2[0]

    // Rebuilt file has NoPos positions - this is expected and correct
    assert.Equal(t, token.NoPos, file2.Package)
    assert.Equal(t, token.NoPos, file2.Name.NamePos)
    
    // But structure is preserved
    assert.Equal(t, file1.Name.Name, file2.Name.Name)
    assert.Equal(t, len(file1.Decls), len(file2.Decls))
    assert.Equal(t, len(file1.Comments), len(file2.Comments))
}
```

### Task 6: Add Warning Comments

**File**: `builder/builder.go`

Add a package-level comment:

```go
// Package builder converts S-expressions back to Go AST.
//
// IMPORTANT: Position information (token.Pos) is NOT preserved through
// S-expression round-trips. All positions in the rebuilt AST will be
// token.NoPos (0). This is by design - S-expressions are for code
// transformation, not source archival.
//
// Comments are preserved and attached to the correct AST nodes, but
// their positions are reset. Use go/printer to format the output.
package builder
```

**File**: `writer/writer.go`

Add a similar comment:

```go
// Package writer converts Go AST to S-expressions.
//
// IMPORTANT: While the writer preserves position information in the
// S-expression output, this information CANNOT be meaningfully restored
// during round-trip. The builder will create ASTs with all positions
// set to token.NoPos (0).
//
// This is by design - use zast for code transformation and analysis,
// not for preserving exact source formatting.
package writer
```

### Task 7: Update Examples

**File**: `examples/roundtrip/main.go` (if it exists)

Update any examples to reflect the limitation:

```go
package main

import (
    "bytes"
    "fmt"
    "go/parser"
    "go/printer"
    "go/token"
    "log"
    
    "zylisp/zast/builder"
    "zylisp/zast/sexp"
    "zylisp/zast/writer"
)

func main() {
    source := `package main

import "fmt"

func main() {
    fmt.Println("Hello, world!")
}
`

    // Parse Go source
    fset := token.NewFileSet()
    file, err := parser.ParseFile(fset, "example.go", source, 0)
    if err != nil {
        log.Fatal(err)
    }

    // Convert to S-expression
    w := writer.NewWriter(fset)
    sexp, err := w.WriteProgram([]*ast.File{file})
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("S-expression:")
    fmt.Println(sexp)
    fmt.Println()

    // Parse S-expression
    p := sexp.NewParser(sexp)
    sexpNode, err := p.Parse()
    if err != nil {
        log.Fatal(err)
    }

    // Build back to AST
    b := builder.NewBuilder()
    fset2, files, err := b.BuildProgram(sexpNode)
    if err != nil {
        log.Fatal(err)
    }

    // Print the reconstructed Go code
    // Note: Positions are lost, but go/printer generates clean output
    fmt.Println("Reconstructed Go code:")
    var buf bytes.Buffer
    printer.Fprint(&buf, fset2, files[0])
    fmt.Println(buf.String())
}
```

### Task 8: Create Migration Guide

**File**: `docs/MIGRATION.md` (new file)

```markdown
# Migration Guide

## Position Handling Changes

As of version X.Y.Z, zast no longer attempts to preserve `token.Pos` values
through S-expression round-trips.

### What Changed

**Before**: 
- zast attempted to serialize FileSet information
- Positions were written to S-expressions
- Builder tried to reconstruct positions

**After**:
- FileSet serialization is simplified
- All positions in rebuilt ASTs are `token.NoPos` (0)
- This is documented as the intended behavior

### Why This Changed

Position preservation through S-expressions is fundamentally impossible because:
1. Positions are absolute offsets into a specific FileSet
2. Deserializing creates a new FileSet
3. Go's FileSet API doesn't support reconstructing positions
4. The old approach created incorrect positions that broke comment placement

### Impact on Your Code

If you were using zast to:

✅ **Transform code**: No impact - positions weren't reliable anyway  
✅ **Analyze AST structure**: No impact - structure is fully preserved  
✅ **Generate code**: No impact - use `go/printer` which works fine with NoPos  

❌ **Preserve exact formatting**: This never worked reliably - consider alternatives:
   - Keep original source text and AST together
   - Use `go/format` or `gofmt` for consistent formatting
   - Wait for `zylisp/core` source map architecture

### Migration Steps

No code changes needed. Just be aware that positions are now consistently `token.NoPos`
rather than inconsistently wrong values.

### Future: Source Maps

Full position tracking through compilation stages will be handled by the
`zylisp/core` source map architecture (see `source-map-architecture.md`).
This will provide proper source location tracking for error reporting in
the Zylisp compiler pipeline.
```

---

## Testing Checklist

After implementing all changes:

- [ ] All existing tests pass
- [ ] New test for position loss passes
- [ ] Round-trip tests compare structure, not positions
- [ ] No warnings or errors from `go vet`
- [ ] No warnings from `golangci-lint`
- [ ] Documentation updated
- [ ] Examples updated (if any exist)

---

## Expected Behavior After Changes

### What Works

```go
// Parse Go code
fset1 := token.NewFileSet()
file1, _ := parser.ParseFile(fset1, "test.go", source, parser.ParseComments)

// Round-trip through S-expressions
writer := writer.NewWriter(fset1)
sexp, _ := writer.WriteProgram([]*ast.File{file1})

builder := builder.NewBuilder()
fset2, files2, _ := builder.BuildProgram(sexpNode)

// Print reconstructed code - works perfectly
printer.Fprint(os.Stdout, fset2, files2[0])
// Output: valid, well-formatted Go code

// Comments are attached to correct nodes
assert.NotNil(t, files2[0].Decls[0].(*ast.FuncDecl).Doc)
```

### What Doesn't Work (and that's OK)

```go
// Trying to use original positions in rebuilt AST
file2 := files2[0]
fmt.Println(fset2.Position(file2.Package)) // Prints "0:0" - position is NoPos

// Trying to get exact original formatting
// The printer will use its own formatting rules, not original spacing
```

---

## Communication Plan

When releasing this change:

1. **Release notes** should clearly state:
   - Position preservation was broken and is now removed
   - This is a simplification that makes behavior consistent
   - AST structure and comments are fully preserved
   - Use go/printer for output formatting

2. **GitHub issue** (if there is one about comment positions) should be updated:
   - Explain why position preservation is impossible
   - Link to source map architecture for future solution
   - Mark as "working as intended"

3. **Users should understand**:
   - zast is for code transformation, not archival
   - Positions are a Go-specific implementation detail
   - The tool works perfectly for its intended purpose

---

## Time Estimate

- Task 1 (Documentation): 15 minutes
- Task 2 (Builder FileSet): 20 minutes  
- Task 3 (Comment positions): 10 minutes
- Task 4 (Writer FileSet): 15 minutes
- Task 5 (Update tests): 30 minutes
- Task 6 (Warning comments): 10 minutes
- Task 7 (Examples): 15 minutes
- Task 8 (Migration guide): 15 minutes
- Testing and verification: 30 minutes

**Total: 2.5 hours**

---

## Success Criteria

- [ ] Code is simpler and more maintainable
- [ ] Behavior is consistent and documented
- [ ] Tests verify AST structure preservation
- [ ] Tests verify positions are NoPos (not wrong values)
- [ ] Users understand the tool's purpose and limitations
- [ ] Foundation is laid for future source map integration
