# Source Map Architecture for Zylisp

**Project**: zylisp/core  
**Purpose**: Track source code positions through all compilation phases  
**Status**: Design specification

---

## Overview

This document specifies a **source map** system that tracks the provenance of code through Zylisp's multi-stage compilation pipeline. This enables reporting errors at any compilation stage with accurate references back to the original source code location.

## Problem Statement

Zylisp's compilation pipeline has multiple transformation stages:

```
Original .zl source
    ↓
Desugared syntax
    ↓
Macro-expanded Zylisp
    ↓
Zylisp IR (S-expression AST)
    ↓
Go AST
    ↓
Generated Go code
```

When an error occurs at any stage, we need to report it with the location in the **original .zl source**, not the intermediate representation where the error was detected.

Traditional approaches (like Go's `token.Pos`) break down because:
- Positions are absolute byte offsets tied to a specific file
- Transformations invalidate these positions
- Round-tripping through serialization (S-expressions) loses position context

## Solution: Provenance Chain via Node IDs

Instead of trying to preserve positions through transformations, we:

1. **Assign unique IDs** to every AST node at parse time
2. **Track parent relationships** as each transformation creates new nodes
3. **Store original positions** only at the input layer
4. **Walk the chain backward** when reporting errors

This is how TypeScript, ClojureScript, Elm, and other compile-to-X languages work.

---

## Architecture

### Core Types

Location in `zylisp/core/sourcemap/types.go`:

```go
package sourcemap

// SourceLocation represents a position in original source code
type SourceLocation struct {
    File   string // File path
    Line   int    // 1-based line number
    Column int    // 1-based column number
    Length int    // Length of the source span in characters
}

// NodeID is a unique identifier for an AST node
// IDs are globally unique within a compilation session
type NodeID uint64

// SourceMap tracks the provenance of nodes through transformations
type SourceMap struct {
    // Name of this transformation layer (for debugging)
    layer string
    
    // Maps nodes in this layer to nodes in the previous layer
    parentNode map[NodeID]NodeID
    
    // Original source positions (only populated at input layer)
    originalPos map[NodeID]SourceLocation
    
    // Link to previous transformation's source map
    previous *SourceMap
}

// NewSourceMap creates a new source map for a transformation layer
func NewSourceMap(layer string, previous *SourceMap) *SourceMap {
    return &SourceMap{
        layer:       layer,
        parentNode:  make(map[NodeID]NodeID),
        originalPos: make(map[NodeID]SourceLocation),
        previous:    previous,
    }
}

// NewInputSourceMap creates a source map for the input layer (parser)
func NewInputSourceMap(layer string) *SourceMap {
    return &SourceMap{
        layer:       layer,
        parentNode:  make(map[NodeID]NodeID),
        originalPos: make(map[NodeID]SourceLocation),
        previous:    nil,
    }
}

// RecordOriginal records the original source location for a node (input layer only)
func (sm *SourceMap) RecordOriginal(nodeID NodeID, loc SourceLocation) {
    sm.originalPos[nodeID] = loc
}

// RecordTransform records that newNodeID was created from oldNodeID
func (sm *SourceMap) RecordTransform(newNodeID, oldNodeID NodeID) {
    sm.parentNode[newNodeID] = oldNodeID
}

// OriginalLocation walks the chain to find the original source location
func (sm *SourceMap) OriginalLocation(nodeID NodeID) *SourceLocation {
    current := sm
    currentID := nodeID
    
    // Walk backward through the transformation chain
    for current != nil {
        // Check if this layer has the original position
        if loc, ok := current.originalPos[currentID]; ok {
            return &loc
        }
        
        // Otherwise, follow the parent link
        if parentID, ok := current.parentNode[currentID]; ok {
            currentID = parentID
            current = current.previous
        } else {
            // Dead end - no parent and no original position
            break
        }
    }
    
    return nil
}

// GetLayer returns the layer name (for debugging)
func (sm *SourceMap) GetLayer() string {
    return sm.layer
}

// DebugTrace returns the full provenance chain for debugging
func (sm *SourceMap) DebugTrace(nodeID NodeID) []string {
    var trace []string
    current := sm
    currentID := nodeID
    
    for current != nil {
        trace = append(trace, fmt.Sprintf("%s: node %d", current.layer, currentID))
        
        if loc, ok := current.originalPos[currentID]; ok {
            trace = append(trace, fmt.Sprintf("  → %s:%d:%d", loc.File, loc.Line, loc.Column))
            break
        }
        
        if parentID, ok := current.parentNode[currentID]; ok {
            currentID = parentID
            current = current.previous
        } else {
            trace = append(trace, "  → (no parent)")
            break
        }
    }
    
    return trace
}
```

### Node ID Generator

Location in `zylisp/core/sourcemap/idgen.go`:

```go
package sourcemap

import "sync/atomic"

// IDGenerator generates unique node IDs
type IDGenerator struct {
    counter uint64
}

// NewIDGenerator creates a new ID generator
func NewIDGenerator() *IDGenerator {
    return &IDGenerator{counter: 0}
}

// Next returns the next unique ID
func (g *IDGenerator) Next() NodeID {
    return NodeID(atomic.AddUint64(&g.counter, 1))
}

// GlobalIDGen is a global ID generator for convenience
// Each compilation session should create its own, but this is available for simple cases
var GlobalIDGen = NewIDGenerator()
```

---

## Integration at Each Layer

### Layer 1: Parser (zylisp/lang)

The parser is the **input layer** that records original source positions.

Location in `zylisp/lang/parser/parser.go`:

```go
package parser

import "zylisp/core/sourcemap"

type Parser struct {
    source    string
    file      string
    pos       int
    line      int
    column    int
    
    idGen     *sourcemap.IDGenerator
    sourceMap *sourcemap.SourceMap
}

func NewParser(source, filename string) *Parser {
    return &Parser{
        source:    source,
        file:      filename,
        pos:       0,
        line:      1,
        column:    1,
        idGen:     sourcemap.NewIDGenerator(),
        sourceMap: sourcemap.NewInputSourceMap("parser"),
    }
}

func (p *Parser) ParseExpr() (*Expr, error) {
    startLine := p.line
    startCol := p.column
    startPos := p.pos
    
    // Generate unique ID
    id := p.idGen.Next()
    
    // Parse the expression
    expr := p.parseExprImpl()
    expr.ID = id
    
    // Record original position
    length := p.pos - startPos
    p.sourceMap.RecordOriginal(id, sourcemap.SourceLocation{
        File:   p.file,
        Line:   startLine,
        Column: startCol,
        Length: length,
    })
    
    return expr, nil
}

// GetSourceMap returns the parser's source map
func (p *Parser) GetSourceMap() *sourcemap.SourceMap {
    return p.sourceMap
}
```

Every AST node needs an ID field:

```go
// In zylisp/lang/ast/expr.go
type Expr struct {
    ID   sourcemap.NodeID  // Unique identifier
    Type ExprType
    // ... other fields
}
```

### Layer 2: Desugarer (zylisp/lang)

Location in `zylisp/lang/desugar/desugar.go`:

```go
package desugar

import "zylisp/core/sourcemap"

type Desugarer struct {
    idGen     *sourcemap.IDGenerator
    sourceMap *sourcemap.SourceMap
}

func NewDesugarer(previousMap *sourcemap.SourceMap) *Desugarer {
    return &Desugarer{
        idGen:     sourcemap.NewIDGenerator(),
        sourceMap: sourcemap.NewSourceMap("desugar", previousMap),
    }
}

func (d *Desugarer) Desugar(expr *ast.Expr) (*ast.Expr, error) {
    newID := d.idGen.Next()
    
    // Desugar the expression
    result := d.desugarImpl(expr)
    result.ID = newID
    
    // Record that new node came from original
    d.sourceMap.RecordTransform(newID, expr.ID)
    
    return result, nil
}

func (d *Desugarer) GetSourceMap() *sourcemap.SourceMap {
    return d.sourceMap
}
```

### Layer 3: Macro Expander (zylisp/lang)

Location in `zylisp/lang/macro/expander.go`:

```go
package macro

import "zylisp/core/sourcemap"

type Expander struct {
    idGen     *sourcemap.IDGenerator
    sourceMap *sourcemap.SourceMap
}

func NewExpander(previousMap *sourcemap.SourceMap) *Expander {
    return &Expander{
        idGen:     sourcemap.NewIDGenerator(),
        sourceMap: sourcemap.NewSourceMap("macro-expand", previousMap),
    }
}

func (e *Expander) Expand(expr *ast.Expr) ([]*ast.Expr, error) {
    // Macro might expand to multiple expressions
    var results []*ast.Expr
    
    for _, expanded := range e.expandImpl(expr) {
        newID := e.idGen.Next()
        expanded.ID = newID
        
        // All generated nodes trace back to the macro call site
        e.sourceMap.RecordTransform(newID, expr.ID)
        
        results = append(results, expanded)
    }
    
    return results, nil
}

func (e *Expander) GetSourceMap() *sourcemap.SourceMap {
    return e.sourceMap
}
```

### Layer 4: IR Generator (zylisp/lang)

Location in `zylisp/lang/ir/generator.go`:

```go
package ir

import "zylisp/core/sourcemap"

type Generator struct {
    idGen     *sourcemap.IDGenerator
    sourceMap *sourcemap.SourceMap
}

func NewGenerator(previousMap *sourcemap.SourceMap) *Generator {
    return &Generator{
        idGen:     sourcemap.NewIDGenerator(),
        sourceMap: sourcemap.NewSourceMap("ir-gen", previousMap),
    }
}

func (g *Generator) Generate(expr *ast.Expr) (*IR, error) {
    newID := g.idGen.Next()
    
    ir := g.generateImpl(expr)
    ir.ID = newID
    
    g.sourceMap.RecordTransform(newID, expr.ID)
    
    return ir, nil
}

func (g *Generator) GetSourceMap() *sourcemap.SourceMap {
    return g.sourceMap
}
```

### Layer 5: Go AST Generator (zylisp/zast)

This is where we **stop caring about Go positions** and just track provenance.

Location in `zylisp/zast/codegen/generator.go`:

```go
package codegen

import (
    "go/ast"
    "go/token"
    "zylisp/core/sourcemap"
)

type Generator struct {
    idGen     *sourcemap.IDGenerator
    sourceMap *sourcemap.SourceMap
    
    // Map Go AST nodes to their IDs (since ast.Node doesn't have ID field)
    goNodeIDs map[ast.Node]sourcemap.NodeID
}

func NewGenerator(previousMap *sourcemap.SourceMap) *Generator {
    return &Generator{
        idGen:     sourcemap.NewIDGenerator(),
        sourceMap: sourcemap.NewSourceMap("go-ast", previousMap),
        goNodeIDs: make(map[ast.Node]sourcemap.NodeID),
    }
}

func (g *Generator) Generate(ir *IR) (*ast.File, error) {
    // Generate Go AST with DUMMY positions
    // We don't care about token.Pos - we track provenance separately
    file := &ast.File{
        Package: token.NoPos,  // Use NoPos or 0
        Name:    &ast.Ident{
            NamePos: token.NoPos,
            Name:    "main",
        },
        // ...
    }
    
    // Track provenance for the file node
    fileID := g.idGen.Next()
    g.goNodeIDs[file] = fileID
    g.sourceMap.RecordTransform(fileID, ir.ID)
    
    return file, nil
}

func (g *Generator) generateExpr(irExpr *IR) ast.Expr {
    // Create Go expression with dummy positions
    goExpr := &ast.BinaryExpr{
        X:     /* ... */,
        OpPos: token.NoPos,
        Op:    token.ADD,
        Y:     /* ... */,
    }
    
    // Track provenance
    exprID := g.idGen.Next()
    g.goNodeIDs[goExpr] = exprID
    g.sourceMap.RecordTransform(exprID, irExpr.ID)
    
    return goExpr
}

func (g *Generator) GetSourceMap() *sourcemap.SourceMap {
    return g.sourceMap
}

// GetNodeID returns the ID for a Go AST node
func (g *Generator) GetNodeID(node ast.Node) (sourcemap.NodeID, bool) {
    id, ok := g.goNodeIDs[node]
    return id, ok
}
```

---

## Error Reporting

Location in `zylisp/core/errors/compiler_error.go`:

```go
package errors

import (
    "fmt"
    "zylisp/core/sourcemap"
)

type CompilerError struct {
    Message      string
    NodeID       sourcemap.NodeID
    SourceMap    *sourcemap.SourceMap
    
    // Cached location (computed lazily)
    location     *sourcemap.SourceLocation
}

func NewCompilerError(msg string, nodeID sourcemap.NodeID, sm *sourcemap.SourceMap) *CompilerError {
    return &CompilerError{
        Message:   msg,
        NodeID:    nodeID,
        SourceMap: sm,
    }
}

func (e *CompilerError) Location() *sourcemap.SourceLocation {
    if e.location == nil {
        e.location = e.SourceMap.OriginalLocation(e.NodeID)
    }
    return e.location
}

func (e *CompilerError) Error() string {
    loc := e.Location()
    if loc != nil {
        return fmt.Sprintf("%s:%d:%d: %s", 
            loc.File, loc.Line, loc.Column, e.Message)
    }
    return fmt.Sprintf("(unknown location): %s", e.Message)
}

// WithContext adds source code context to the error message
func (e *CompilerError) WithContext(source string) string {
    loc := e.Location()
    if loc == nil {
        return e.Error()
    }
    
    // Extract the relevant line from source
    lines := strings.Split(source, "\n")
    if loc.Line < 1 || loc.Line > len(lines) {
        return e.Error()
    }
    
    line := lines[loc.Line-1]
    
    // Build error with context
    var b strings.Builder
    fmt.Fprintf(&b, "%s:%d:%d: %s\n", loc.File, loc.Line, loc.Column, e.Message)
    fmt.Fprintf(&b, "%s\n", line)
    
    // Add caret pointing to error location
    for i := 1; i < loc.Column; i++ {
        b.WriteByte(' ')
    }
    b.WriteByte('^')
    if loc.Length > 1 {
        for i := 1; i < loc.Length; i++ {
            b.WriteByte('~')
        }
    }
    
    return b.String()
}

// DebugTrace returns the full provenance chain
func (e *CompilerError) DebugTrace() string {
    trace := e.SourceMap.DebugTrace(e.NodeID)
    var b strings.Builder
    b.WriteString("Error provenance:\n")
    for _, step := range trace {
        b.WriteString("  ")
        b.WriteString(step)
        b.WriteByte('\n')
    }
    return b.String()
}
```

### Translating Go Compiler Errors

When the Go compiler reports an error in generated code:

Location in `zylisp/cli/compile/go_errors.go`:

```go
package compile

import (
    "go/ast"
    "zylisp/core/errors"
    "zylisp/core/sourcemap"
)

type GoErrorTranslator struct {
    generator *codegen.Generator  // Has the Go AST and source map
}

func (t *GoErrorTranslator) TranslateError(goErr error, node ast.Node) *errors.CompilerError {
    // Find the node ID for the Go AST node that caused the error
    nodeID, ok := t.generator.GetNodeID(node)
    if !ok {
        // Fallback: error with no source tracking
        return errors.NewCompilerError(goErr.Error(), 0, nil)
    }
    
    // Create error with full source map chain
    return errors.NewCompilerError(
        goErr.Error(),
        nodeID,
        t.generator.GetSourceMap(),
    )
}
```

---

## Example Usage Flow

### Compilation Pipeline

Location in `zylisp/cli/compile/pipeline.go`:

```go
package compile

import "zylisp/core/sourcemap"

type CompilationPipeline struct {
    source    string
    filename  string
    
    parser    *parser.Parser
    desugarer *desugar.Desugarer
    expander  *macro.Expander
    irGen     *ir.Generator
    codeGen   *codegen.Generator
}

func NewPipeline(source, filename string) *CompilationPipeline {
    return &CompilationPipeline{
        source:   source,
        filename: filename,
    }
}

func (p *CompilationPipeline) Compile() (*ast.File, error) {
    // Layer 1: Parse
    p.parser = parser.NewParser(p.source, p.filename)
    expr, err := p.parser.ParseExpr()
    if err != nil {
        return nil, err
    }
    
    // Layer 2: Desugar
    p.desugarer = desugar.NewDesugarer(p.parser.GetSourceMap())
    desugared, err := p.desugarer.Desugar(expr)
    if err != nil {
        return nil, err
    }
    
    // Layer 3: Expand macros
    p.expander = macro.NewExpander(p.desugarer.GetSourceMap())
    expanded, err := p.expander.Expand(desugared)
    if err != nil {
        return nil, err
    }
    
    // Layer 4: Generate IR
    p.irGen = ir.NewGenerator(p.expander.GetSourceMap())
    irCode, err := p.irGen.Generate(expanded[0])
    if err != nil {
        return nil, err
    }
    
    // Layer 5: Generate Go AST
    p.codeGen = codegen.NewGenerator(p.irGen.GetSourceMap())
    goFile, err := p.codeGen.Generate(irCode)
    if err != nil {
        return nil, err
    }
    
    return goFile, nil
}

// GetFinalSourceMap returns the complete source map chain
func (p *CompilationPipeline) GetFinalSourceMap() *sourcemap.SourceMap {
    return p.codeGen.GetSourceMap()
}
```

### Error Reporting in REPL

Location in `zylisp/repl/repl.go`:

```go
package repl

func (r *REPL) Eval(input string) {
    pipeline := compile.NewPipeline(input, "<repl>")
    
    goFile, err := pipeline.Compile()
    if err != nil {
        // Error during compilation
        if compErr, ok := err.(*errors.CompilerError); ok {
            fmt.Println(compErr.WithContext(input))
            
            if r.debug {
                fmt.Println(compErr.DebugTrace())
            }
        } else {
            fmt.Println("Error:", err)
        }
        return
    }
    
    // Try to run the generated code
    err = r.runGoCode(goFile)
    if err != nil {
        // Error from Go compiler on generated code
        translator := &compile.GoErrorTranslator{/* ... */}
        compErr := translator.TranslateError(err, /* node */)
        fmt.Println(compErr.WithContext(input))
    }
}
```

---

## Repository Structure

```
zylisp/
├── core/                      # NEW: Core shared infrastructure
│   ├── sourcemap/
│   │   ├── types.go          # SourceMap, SourceLocation, NodeID
│   │   ├── idgen.go          # ID generator
│   │   └── sourcemap_test.go
│   ├── errors/
│   │   ├── compiler_error.go # CompilerError with source tracking
│   │   └── errors_test.go
│   └── go.mod
│
├── lang/                      # Language implementation
│   ├── ast/
│   │   └── expr.go           # Add NodeID field to all AST nodes
│   ├── parser/
│   │   └── parser.go         # Input layer: records original positions
│   ├── desugar/
│   │   └── desugar.go        # Tracks transformations
│   ├── macro/
│   │   └── expander.go       # Tracks macro expansions
│   ├── ir/
│   │   └── generator.go      # Tracks IR generation
│   └── go.mod                # Depends on zylisp/core
│
├── zast/                      # Go AST <-> S-expr
│   ├── codegen/
│   │   └── generator.go      # NEW: Generate Go AST with provenance
│   └── go.mod                # Depends on zylisp/core
│
├── repl/
│   ├── repl.go               # Uses CompilerError for display
│   └── go.mod                # Depends on zylisp/core, zylisp/lang
│
└── cli/
    ├── compile/
    │   ├── pipeline.go       # Orchestrates compilation with source maps
    │   └── go_errors.go      # Translates Go compiler errors
    └── go.mod                # Depends on zylisp/core, zylisp/lang, zylisp/zast
```

---

## Testing Strategy

### Unit Tests

Location in `zylisp/core/sourcemap/sourcemap_test.go`:

```go
func TestSourceMapChain(t *testing.T) {
    // Input layer
    inputMap := NewInputSourceMap("parser")
    inputMap.RecordOriginal(1, SourceLocation{
        File: "test.zl",
        Line: 5,
        Column: 10,
        Length: 8,
    })
    
    // Transform layer
    transformMap := NewSourceMap("desugar", inputMap)
    transformMap.RecordTransform(100, 1)
    
    // Should find original location
    loc := transformMap.OriginalLocation(100)
    assert.NotNil(t, loc)
    assert.Equal(t, "test.zl", loc.File)
    assert.Equal(t, 5, loc.Line)
    assert.Equal(t, 10, loc.Column)
}

func TestMultiLayerChain(t *testing.T) {
    // Simulate full pipeline
    inputMap := NewInputSourceMap("parser")
    inputMap.RecordOriginal(1, SourceLocation{File: "test.zl", Line: 1, Column: 1})
    
    desugarMap := NewSourceMap("desugar", inputMap)
    desugarMap.RecordTransform(10, 1)
    
    macroMap := NewSourceMap("macro", desugarMap)
    macroMap.RecordTransform(100, 10)
    
    irMap := NewSourceMap("ir", macroMap)
    irMap.RecordTransform(1000, 100)
    
    goMap := NewSourceMap("go-ast", irMap)
    goMap.RecordTransform(10000, 1000)
    
    // Should trace all the way back
    loc := goMap.OriginalLocation(10000)
    assert.NotNil(t, loc)
    assert.Equal(t, "test.zl", loc.File)
    assert.Equal(t, 1, loc.Line)
}
```

### Integration Tests

Location in `zylisp/cli/compile/pipeline_test.go`:

```go
func TestErrorReporting(t *testing.T) {
    source := `(defn add [x y] (+ x "hello"))`  // Type error
    
    pipeline := NewPipeline(source, "test.zl")
    _, err := pipeline.Compile()
    
    require.Error(t, err)
    
    compErr, ok := err.(*errors.CompilerError)
    require.True(t, ok)
    
    loc := compErr.Location()
    require.NotNil(t, loc)
    
    assert.Equal(t, "test.zl", loc.File)
    assert.Equal(t, 1, loc.Line)
    // Column should point to the problematic `+` expression
    
    // Error message should reference original source
    errMsg := compErr.WithContext(source)
    assert.Contains(t, errMsg, "test.zl:1:")
    assert.Contains(t, errMsg, source)
}
```

---

## Migration Path

### Phase 1: Create zylisp/core
1. Create new repository/module
2. Implement sourcemap package
3. Implement errors package
4. Write comprehensive tests

### Phase 2: Update zylisp/lang
1. Add NodeID field to all AST node types
2. Update parser to use sourcemap
3. Update desugarer to track transformations
4. Update macro expander to track transformations
5. Update IR generator to track transformations

### Phase 3: Update zylisp/zast
1. Create codegen package
2. Generate Go AST with dummy positions
3. Track provenance via source maps
4. Implement GoErrorTranslator

### Phase 4: Update zylisp/repl and zylisp/cli
1. Use CompilerError for all error reporting
2. Display errors with source context
3. Add debug mode for provenance traces

### Phase 5: Testing & Refinement
1. End-to-end integration tests
2. Performance profiling
3. Error message quality review
4. Documentation

---

## Critical Design Decisions

### Why Not Use token.Pos?

`token.Pos` is designed for a single-stage compiler working with one FileSet. It breaks down when:
- Serializing/deserializing ASTs
- Multiple transformation stages
- Code generation from non-Go source

Our approach separates concerns:
- **Provenance tracking**: Via SourceMap (our responsibility)
- **Position info**: Via token.Pos (Go's printer's responsibility)

### Why Global Node IDs?

Node IDs must be globally unique across all transformation stages so we can trace backward through the chain. Using per-layer IDs would require complex mapping tables.

### Why Store Original Positions Only at Input?

Only the parser knows the true source positions. All other layers just track transformations. This keeps the system simple and correct.

### Why Dummy Go AST Positions?

The Go AST positions are only used by Go's printer for formatting. We don't need them for error tracking. Using dummy positions (0 or token.NoPos) makes it clear we're not relying on them.

---

## Performance Considerations

### Memory Usage

Each transformation layer creates:
- A SourceMap struct (~48 bytes + maps)
- Map entries for each transformed node (~16 bytes per entry)

For a 1000-node program through 5 layers:
- ~5000 map entries × 16 bytes = ~80 KB
- 5 SourceMap structs = ~240 bytes
- **Total: ~80 KB** (negligible)

### Time Complexity

Looking up original location:
- O(L) where L = number of transformation layers
- For 5 layers: 5 map lookups = ~500 ns
- Only done when reporting errors (rare)

**Conclusion**: Performance impact is negligible.

---

## Future Enhancements

### Optimization: Position Ranges

Instead of single positions, track ranges:

```go
type SourceRange struct {
    Start SourceLocation
    End   SourceLocation
}
```

This enables better error highlighting and IDE integration.

### Enhancement: Multiple Parents

Some transformations create nodes from multiple sources (e.g., inlining):

```go
type SourceMap struct {
    // ...
    parentNodes map[NodeID][]NodeID  // Multiple parents
}
```

Report all source locations that contributed to the error.

### Tool: Source Map Visualizer

Build a visualization tool that shows the transformation chain:

```
test.zl:5:10 (defn add [x y])
    ↓ desugar
  node 10
    ↓ macro-expand  
  node 100
    ↓ ir-gen
  node 1000
    ↓ go-ast
  return x + y
```

---

## Conclusion

This source map architecture provides:
- ✅ Accurate error reporting at every compilation stage
- ✅ References to original source code
- ✅ Minimal performance overhead
- ✅ Clean separation of concerns
- ✅ Works with S-expression serialization
- ✅ Scales to complex transformation pipelines

The key insight: **Stop fighting token.Pos. Track provenance separately.**
