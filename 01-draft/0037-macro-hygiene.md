---
number: 0037
title: "Macro Hygiene and Gensym in Zylisp"
author: Unknown
created: 2025-10-10
updated: 2025-10-10
state: Draft
supersedes: None
superseded-by: None
---

# Macro Hygiene and Gensym in Zylisp

**Project**: zylisp/lang  
**Purpose**: Design and implement hygienic macro expansion with source tracking  
**Status**: Design specification

---

## Overview

This document specifies how Zylisp achieves hygienic macros through `gensym` and proper symbol scoping, integrated with the source map architecture for accurate error reporting.

### Why Zylisp Can Have Hygienic Macros

Unlike LFE (which runs on the Erlang BEAM), Zylisp is implemented in Go and compiles to native code or bytecode. This gives us significant advantages:

#### LFE's Constraints

- **Global atom table**: Erlang has a fixed-size global atom table (~1M atoms by default)
- **No garbage collection**: Atoms are never GC'd and live forever
- **Distributed coordination**: In distributed Erlang, `gensym` would need cross-node coordination
- **Runtime cost**: Every generated symbol exists at runtime permanently
- **VM crash risk**: Exhausting the atom table crashes the entire VM

#### Zylisp's Advantages

- **Garbage collected strings**: Go's strings are heap-allocated and GC'd automatically
- **No fixed limits**: Symbol table is bounded only by available heap memory
- **Single process**: No distributed node concerns (at least initially)
- **Compile-time only**: Generated symbols only exist during compilation
- **Safe to generate**: Can create millions of temporary symbols without risk

#### The Key Difference

**LFE**: Generated symbols must exist at runtime in the atom table forever.

**Zylisp**: Generated symbols only exist during compilation. After lowering to bytecode/native:
- Local variables become stack slots/registers (no names needed)
- Global references are resolved to addresses/indices
- Only user-visible symbols need preservation for debugging

### Comparison Table

| Aspect | LFE (Erlang) | Zylisp (Go) |
|--------|--------------|-------------|
| Symbol storage | Global atom table (fixed size) | Heap (GC'd) |
| Runtime cost | Atoms live forever | Symbols only during compilation |
| Memory limit | ~1M atoms globally | Available heap memory |
| Garbage collection | Never | Automatic |
| `gensym` risk | VM crash | None |
| Distributed concerns | Complex coordination needed | N/A (single process) |

---

## Macro Hygiene Fundamentals

### What is Macro Hygiene?

A macro is **hygienic** if it prevents unintended variable capture between:
1. Variables introduced by the macro implementation
2. Variables in the macro call site
3. Variables in macro arguments

### The Variable Capture Problem

```lisp
;; Non-hygienic macro (bad!)
(defmacro swap! (a b)
  `(let ((tmp ,a))
     (set! ,a ,b)
     (set! ,b tmp)))

;; Problem case:
(let ((tmp 5))
  (swap! x y)
  tmp)  ; => WRONG! Returns y instead of 5
  
;; The macro expansion captures the user's `tmp`:
(let ((tmp 5))
  (let ((tmp x))      ; macro's tmp shadows user's tmp
    (set! x y)
    (set! y tmp))
  tmp)  ; => returns y, not the original 5
```

### Solution: Gensym

`gensym` generates **guaranteed unique** symbols that cannot conflict with user code:

```lisp
;; Hygienic macro (good!)
(defmacro swap! (a b)
  (let ((tmp (gensym "tmp")))
    `(let ((,tmp ,a))
       (set! ,a ,b)
       (set! ,b ,tmp))))

;; Safe usage:
(let ((tmp 5))
  (swap! x y)
  tmp)  ; => 5 (correct!)
  
;; Expands to:
(let ((tmp 5))
  (let ((tmp#1234 x))   ; unique symbol, no capture
    (set! x y)
    (set! y tmp#1234))
  tmp)  ; => 5
```

---

## Gensym Implementation

### Basic Design

Location: `zylisp/lang/macro/gensym.go`

```go
package macro

import (
    "fmt"
    "sync/atomic"
)

// GensymCounter generates unique symbol IDs
type GensymCounter struct {
    counter uint64
}

// NewGensymCounter creates a new counter
func NewGensymCounter() *GensymCounter {
    return &GensymCounter{counter: 0}
}

// Next returns the next unique ID
func (g *GensymCounter) Next() uint64 {
    return atomic.AddUint64(&g.counter, 1)
}

// Gensym generates a unique symbol name
// The "#" character makes it impossible to create via the reader
func Gensym(prefix string, counter *GensymCounter) string {
    id := counter.Next()
    return fmt.Sprintf("%s#%d", prefix, id)
}
```

### Why Use "#" in Generated Names?

The `#` character is chosen because:
1. It cannot appear in normal Zylisp identifiers (reserved by reader)
2. Makes generated symbols visually distinct
3. Prevents accidental name collisions with user code
4. Easy to filter in debugging/error messages

### Symbol Representation

Location: `zylisp/lang/ast/symbol.go`

```go
package ast

import "zylisp/core/sourcemap"

type Symbol struct {
    Name      string              // The symbol name (e.g., "tmp#1234")
    ID        sourcemap.NodeID    // Unique AST node ID
    
    // Hygiene tracking
    Generated bool                // Created by gensym?
    ScopeID   uint32              // Which lexical scope introduced this
    
    // Source tracking
    SourceLoc *sourcemap.SourceLocation  // Original location (if any)
    MacroInfo *MacroExpansionInfo        // Macro that generated it (if any)
}

type MacroExpansionInfo struct {
    MacroName string                     // Name of the macro
    CallSite  *sourcemap.SourceLocation  // Where the macro was called
}

// IsGenerated returns true if this symbol was created by gensym
func (s *Symbol) IsGenerated() bool {
    return s.Generated
}

// DisplayName returns the name to show in error messages
func (s *Symbol) DisplayName() string {
    if s.Generated && s.MacroInfo != nil {
        return fmt.Sprintf("%s (generated by %s macro at %s:%d)",
            s.Name,
            s.MacroInfo.MacroName,
            s.MacroInfo.CallSite.File,
            s.MacroInfo.CallSite.Line,
        )
    }
    return s.Name
}
```

---

## Macro Expansion with Source Tracking

### Expander Structure

Location: `zylisp/lang/macro/expander.go`

```go
package macro

import (
    "zylisp/core/sourcemap"
    "zylisp/lang/ast"
)

type Expander struct {
    // Source mapping
    idGen     *sourcemap.IDGenerator
    sourceMap *sourcemap.SourceMap
    
    // Hygiene
    gensymCounter *GensymCounter
    scopeDepth    uint32
    
    // Macro definitions
    macros    map[string]*MacroDefinition
}

type MacroDefinition struct {
    Name       string
    Parameters []string
    Body       ast.Expr
    SourceLoc  *sourcemap.SourceLocation
}

func NewExpander(previousMap *sourcemap.SourceMap) *Expander {
    return &Expander{
        idGen:         sourcemap.NewIDGenerator(),
        sourceMap:     sourcemap.NewSourceMap("macro-expand", previousMap),
        gensymCounter: NewGensymCounter(),
        scopeDepth:    0,
        macros:        make(map[string]*MacroDefinition),
    }
}

func (e *Expander) GetSourceMap() *sourcemap.SourceMap {
    return e.sourceMap
}
```

### Macro Expansion Process

```go
// Expand expands a macro call expression
func (e *Expander) Expand(expr *ast.Expr) (*ast.Expr, error) {
    // Check if this is a macro call
    if !e.isMacroCall(expr) {
        return expr, nil
    }
    
    callExpr := expr.(*ast.CallExpr)
    macroName := callExpr.Func.(*ast.Symbol).Name
    macro := e.macros[macroName]
    
    // Enter new scope for this expansion
    e.scopeDepth++
    defer func() { e.scopeDepth-- }()
    
    // Expand the macro body with arguments substituted
    expanded, err := e.expandMacroBody(macro, callExpr.Args, expr.ID)
    if err != nil {
        return nil, err
    }
    
    // Assign new IDs to all generated nodes
    e.assignNewIDs(expanded, expr.ID)
    
    // Recursively expand any nested macros
    return e.expandRecursive(expanded)
}

// expandMacroBody substitutes arguments and evaluates the macro template
func (e *Expander) expandMacroBody(
    macro *MacroDefinition, 
    args []*ast.Expr,
    callSiteID sourcemap.NodeID,
) (*ast.Expr, error) {
    // Create environment binding parameters to arguments
    env := make(map[string]*ast.Expr)
    for i, param := range macro.Parameters {
        if i < len(args) {
            env[param] = args[i]
        }
    }
    
    // Evaluate the macro body (which is a template)
    result, err := e.evalTemplate(macro.Body, env, callSiteID)
    if err != nil {
        return nil, err
    }
    
    return result, nil
}

// assignNewIDs walks the expanded tree and assigns new node IDs
// All new nodes are linked back to the macro call site
func (e *Expander) assignNewIDs(expr *ast.Expr, parentID sourcemap.NodeID) {
    newID := e.idGen.Next()
    
    // Link this generated node back to the macro call site
    e.sourceMap.RecordTransform(newID, parentID)
    
    // Store macro expansion info for debugging
    if sym, ok := (*expr).(*ast.Symbol); ok && sym.Generated {
        // Get the call site location from the parent
        if callLoc := e.sourceMap.OriginalLocation(parentID); callLoc != nil {
            sym.MacroInfo = &MacroExpansionInfo{
                MacroName: e.currentMacroName(),
                CallSite:  callLoc,
            }
        }
    }
    
    // Update the expr's ID
    e.setExprID(expr, newID)
    
    // Recursively process children
    for _, child := range e.getChildren(expr) {
        e.assignNewIDs(child, newID)
    }
}
```

### Template Evaluation with Gensym

```go
// evalTemplate evaluates a quasiquote template
func (e *Expander) evalTemplate(
    template ast.Expr,
    env map[string]*ast.Expr,
    callSiteID sourcemap.NodeID,
) (*ast.Expr, error) {
    switch t := template.(type) {
    case *ast.UnquoteExpr:
        // Evaluate the unquoted expression
        return e.evalInEnv(t.Expr, env, callSiteID)
        
    case *ast.Symbol:
        // Check if this is a template variable
        if val, ok := env[t.Name]; ok {
            return val, nil
        }
        return &template, nil
        
    case *ast.CallExpr:
        // Check for special form: (gensym "prefix")
        if sym, ok := t.Func.(*ast.Symbol); ok && sym.Name == "gensym" {
            if len(t.Args) != 1 {
                return nil, fmt.Errorf("gensym requires exactly 1 argument")
            }
            
            // Get the prefix
            prefixExpr, err := e.evalInEnv(t.Args[0], env, callSiteID)
            if err != nil {
                return nil, err
            }
            
            prefix, ok := (*prefixExpr).(*ast.StringLit)
            if !ok {
                return nil, fmt.Errorf("gensym prefix must be a string")
            }
            
            // Generate unique symbol
            genName := Gensym(prefix.Value, e.gensymCounter)
            genSym := &ast.Symbol{
                Name:      genName,
                Generated: true,
                ScopeID:   e.scopeDepth,
            }
            
            result := ast.Expr(genSym)
            return &result, nil
        }
        
        // Regular call - evaluate function and arguments
        fn, err := e.evalTemplate(t.Func, env, callSiteID)
        if err != nil {
            return nil, err
        }
        
        args := make([]*ast.Expr, len(t.Args))
        for i, arg := range t.Args {
            args[i], err = e.evalTemplate(arg, env, callSiteID)
            if err != nil {
                return nil, err
            }
        }
        
        result := ast.Expr(&ast.CallExpr{Func: fn, Args: args})
        return &result, nil
        
    case *ast.ListExpr:
        // Recursively process list elements
        elements := make([]*ast.Expr, len(t.Elements))
        for i, elem := range t.Elements {
            var err error
            elements[i], err = e.evalTemplate(elem, env, callSiteID)
            if err != nil {
                return nil, err
            }
        }
        result := ast.Expr(&ast.ListExpr{Elements: elements})
        return &result, nil
        
    default:
        // Literals and other forms pass through unchanged
        return &template, nil
    }
}
```

---

## Integration with Source Mapping

### Tracking Macro Expansions

When a macro expands, we need to track:
1. **Where the macro was called** (the call site)
2. **What code the macro generated** (the expansion)
3. **The chain from generated code back to call site**

```go
// Example: Expanding (swap! x y)
func (e *Expander) trackExpansion(
    callExpr *ast.CallExpr,
    expanded *ast.Expr,
) {
    // The call expression has an ID from parsing
    callID := callExpr.ID
    
    // The expanded code gets new IDs
    expandedID := e.idGen.Next()
    (*expanded).SetID(expandedID)
    
    // Link expansion back to call site
    e.sourceMap.RecordTransform(expandedID, callID)
    
    // For every node in the expansion, link to the expansion root
    e.linkGeneratedNodes(expanded, expandedID)
}

// linkGeneratedNodes ensures all generated nodes trace to expansion root
func (e *Expander) linkGeneratedNodes(expr *ast.Expr, rootID sourcemap.NodeID) {
    for _, child := range e.getChildren(expr) {
        childID := e.idGen.Next()
        (*child).SetID(childID)
        
        // Link child to expansion root
        e.sourceMap.RecordTransform(childID, rootID)
        
        // Recursively process grandchildren
        e.linkGeneratedNodes(child, childID)
    }
}
```

### Error Reporting from Macro-Generated Code

When an error occurs in macro-generated code, we want to show:
1. The error location in the macro's expansion
2. The macro call site that caused the expansion
3. Context from the original source

```go
// In zylisp/core/errors/compiler_error.go

func (e *CompilerError) MacroExpansionContext() string {
    loc := e.Location()
    if loc == nil {
        return e.Error()
    }
    
    // Check if this error originated from macro-generated code
    // by walking the source map chain looking for macro expansion layers
    trace := e.SourceMap.DebugTrace(e.NodeID)
    
    var b strings.Builder
    
    // Primary error location
    fmt.Fprintf(&b, "%s:%d:%d: %s\n", 
        loc.File, loc.Line, loc.Column, e.Message)
    
    // Show macro expansion chain
    inMacroExpansion := false
    for _, step := range trace {
        if strings.Contains(step, "macro-expand") {
            inMacroExpansion = true
            b.WriteString("  in expansion of macro at:\n")
        }
        if inMacroExpansion {
            b.WriteString("    ")
            b.WriteString(step)
            b.WriteByte('\n')
        }
    }
    
    return b.String()
}
```

### Example Error Output

```lisp
;; Original code in user.zl:
(defmacro my-let (bindings & body)
  (let ((tmp (gensym "temp")))
    `(let ((,tmp 42))
       (let ,bindings
         ,@body))))

;; User code at line 15:
(my-let ((x "hello"))
  (+ x tmp))  ; Error: tmp is not defined
```

Error output:

```
user.zl:15:3: undefined variable: temp#1234
  (+ x tmp)
     ^
  in expansion of macro at:
    user.zl:15:1: (my-let ((x "hello")) ...)
    
Note: temp#1234 was generated by my-let macro
```

---

## Macro Definition

### Defining Macros

```go
// DefMacro registers a new macro definition
func (e *Expander) DefMacro(
    name string,
    params []string,
    body ast.Expr,
    sourceLoc *sourcemap.SourceLocation,
) error {
    if e.macros[name] != nil {
        return fmt.Errorf("macro %s already defined", name)
    }
    
    e.macros[name] = &MacroDefinition{
        Name:       name,
        Parameters: params,
        Body:       body,
        SourceLoc:  sourceLoc,
    }
    
    return nil
}
```

### Macro Definition Syntax

```lisp
;; Simple macro
(defmacro when (condition & body)
  `(if ,condition
       (do ,@body)
       nil))

;; Macro with gensym
(defmacro swap! (a b)
  (let ((tmp (gensym "tmp")))
    `(let ((,tmp ,a))
       (set! ,a ,b)
       (set! ,b ,tmp))))

;; Macro generating macros
(defmacro defn-pair (name)
  `(do
     (defn ,(symbol-append 'get- name) () 
       (get-field *state* ',name))
     (defn ,(symbol-append 'set- name) (x)
       (set-field! *state* ',name x))))
```

---

## Deliberate Symbol Capture

Sometimes we **want** to capture variables. Zylisp provides explicit operations:

### Symbol Construction vs Gensym

```go
// In the macro evaluator:

case "gensym":
    // ALWAYS generates unique symbols (hygiene)
    return &ast.Symbol{
        Name:      Gensym(prefix, e.gensymCounter),
        Generated: true,
    }

case "intern":
    // Creates a specific symbol (deliberate capture)
    return &ast.Symbol{
        Name:      symbolName,
        Generated: false,  // NOT generated, intentional
    }

case "symbol-append":
    // Constructs symbol names (deliberate)
    name := concatenateSymbols(args)
    return &ast.Symbol{
        Name:      name,
        Generated: false,
    }
```

### Example: Anaphoric Macros

```lisp
;; Anaphoric if - deliberately captures 'it'
(defmacro aif (condition then else)
  `(let ((it ,condition))
     (if it
         ,then
         ,else)))

;; Usage:
(aif (find-user "bob")
     (print it)        ; 'it' bound to result of find-user
     (print "not found"))
```

This uses `intern` to create the symbol `it`, which is intentional capture.

---

## Scope Tracking

### Scope Management

```go
type Expander struct {
    // ... other fields ...
    
    scopeDepth uint32
    scopes     []Scope
}

type Scope struct {
    ID       uint32
    Bindings map[string]*ast.Symbol
    Parent   *Scope
}

// enterScope creates a new lexical scope
func (e *Expander) enterScope() {
    e.scopeDepth++
    scope := Scope{
        ID:       e.scopeDepth,
        Bindings: make(map[string]*ast.Symbol),
    }
    if len(e.scopes) > 0 {
        scope.Parent = &e.scopes[len(e.scopes)-1]
    }
    e.scopes = append(e.scopes, scope)
}

// exitScope removes the current scope
func (e *Expander) exitScope() {
    if len(e.scopes) > 0 {
        e.scopes = e.scopes[:len(e.scopes)-1]
        e.scopeDepth--
    }
}

// bindSymbol adds a symbol to the current scope
func (e *Expander) bindSymbol(sym *ast.Symbol) {
    if len(e.scopes) > 0 {
        currentScope := &e.scopes[len(e.scopes)-1]
        currentScope.Bindings[sym.Name] = sym
        sym.ScopeID = currentScope.ID
    }
}

// lookupSymbol finds a symbol in the scope chain
func (e *Expander) lookupSymbol(name string) (*ast.Symbol, bool) {
    for i := len(e.scopes) - 1; i >= 0; i-- {
        if sym, ok := e.scopes[i].Bindings[name]; ok {
            return sym, true
        }
    }
    return nil, false
}
```

### Detecting Accidental Capture

```go
// checkHygiene verifies that macro expansion doesn't accidentally capture
func (e *Expander) checkHygiene(expanded *ast.Expr, callSiteScope uint32) error {
    return e.walkExpr(expanded, func(expr ast.Expr) error {
        if sym, ok := expr.(*ast.Symbol); ok {
            // Check if this is a generated symbol
            if sym.Generated {
                // Generated symbols should never match user symbols
                if existing, ok := e.lookupSymbol(sym.Name); ok {
                    if !existing.Generated {
                        return fmt.Errorf(
                            "hygiene violation: generated symbol %s conflicts with user symbol",
                            sym.DisplayName(),
                        )
                    }
                }
            }
        }
        return nil
    })
}
```

---

## Memory Management

### Compilation-Time Symbol Table

```go
type SymbolTable struct {
    symbols map[string]*ast.Symbol
    mu      sync.RWMutex
}

func (st *SymbolTable) Intern(name string) *ast.Symbol {
    st.mu.RLock()
    if sym, exists := st.symbols[name]; exists {
        st.mu.RUnlock()
        return sym
    }
    st.mu.RUnlock()
    
    st.mu.Lock()
    defer st.mu.Unlock()
    
    // Double-check after acquiring write lock
    if sym, exists := st.symbols[name]; exists {
        return sym
    }
    
    sym := &ast.Symbol{Name: name}
    st.symbols[name] = sym
    return sym
}

// Clear removes all symbols (called between compilation units)
func (st *SymbolTable) Clear() {
    st.mu.Lock()
    defer st.mu.Unlock()
    st.symbols = make(map[string]*ast.Symbol)
}
```

### Per-Compilation-Unit Lifecycle

```go
func (c *Compiler) CompileFile(filename string) error {
    // Create fresh symbol table for this file
    symbolTable := NewSymbolTable()
    
    // ... parse, expand macros, etc ...
    
    // After compilation, Go's GC will clean up:
    // - All temporary symbols (including gensym'd ones)
    // - AST nodes
    // - Intermediate forms
    
    // Only compiled output remains
    return nil
}
```

**Key insight**: Unlike Erlang, generated symbols don't persist. They exist only during the compilation of a single file, then get GC'd.

---

## Testing Strategy

### Unit Tests for Gensym

Location: `zylisp/lang/macro/gensym_test.go`

```go
func TestGensymUniqueness(t *testing.T) {
    counter := NewGensymCounter()
    
    sym1 := Gensym("tmp", counter)
    sym2 := Gensym("tmp", counter)
    sym3 := Gensym("tmp", counter)
    
    assert.NotEqual(t, sym1, sym2)
    assert.NotEqual(t, sym2, sym3)
    assert.NotEqual(t, sym1, sym3)
    
    assert.Contains(t, sym1, "#")
    assert.Contains(t, sym2, "#")
}

func TestGensymPrefix(t *testing.T) {
    counter := NewGensymCounter()
    
    sym := Gensym("myvar", counter)
    assert.True(t, strings.HasPrefix(sym, "myvar#"))
}

func TestConcurrentGensym(t *testing.T) {
    counter := NewGensymCounter()
    
    seen := sync.Map{}
    var wg sync.WaitGroup
    
    // Generate 10000 symbols concurrently
    for i := 0; i < 10000; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            sym := Gensym("tmp", counter)
            
            // Should never see duplicates
            if _, exists := seen.LoadOrStore(sym, true); exists {
                t.Errorf("duplicate symbol: %s", sym)
            }
        }()
    }
    
    wg.Wait()
}
```

### Integration Tests for Hygiene

Location: `zylisp/lang/macro/hygiene_test.go`

```go
func TestSwapMacroHygiene(t *testing.T) {
    source := `
        (defmacro swap! (a b)
          (let ((tmp (gensym "tmp")))
            \`(let ((,tmp ,a))
               (set! ,a ,b)
               (set! ,b ,tmp))))
        
        (let ((tmp 5)
              (x 1)
              (y 2))
          (swap! x y)
          (list x y tmp))  ; Should be (2 1 5)
    `
    
    result, err := testEval(source)
    require.NoError(t, err)
    
    list := result.(*ast.ListExpr)
    assert.Equal(t, 2, list.Elements[0])
    assert.Equal(t, 1, list.Elements[1])
    assert.Equal(t, 5, list.Elements[2]) // tmp unchanged
}

func TestNestedMacroExpansion(t *testing.T) {
    source := `
        (defmacro when (condition & body)
          \`(if ,condition (do ,@body) nil))
        
        (defmacro unless (condition & body)
          \`(when (not ,condition) ,@body))
        
        (let ((x 5))
          (unless (> x 10)
            (set! x 42))
          x)  ; Should be 42
    `
    
    result, err := testEval(source)
    require.NoError(t, err)
    assert.Equal(t, 42, result)
}
```

### Tests for Source Tracking

```go
func TestMacroErrorReporting(t *testing.T) {
    source := `
        (defmacro broken (x)
          \`(+ ,x "hello"))  ; Type error
        
        (broken 42)
    `
    
    _, err := testCompile(source)
    require.Error(t, err)
    
    compErr, ok := err.(*errors.CompilerError)
    require.True(t, ok)
    
    // Should report error at the macro call site
    loc := compErr.Location()
    require.NotNil(t, loc)
    assert.Contains(t, loc.File, "test")
    
    // Error message should mention macro expansion
    msg := compErr.MacroExpansionContext()
    assert.Contains(t, msg, "broken")
    assert.Contains(t, msg, "macro")
}
```

---

## Advanced Features

### Syntax Objects (Future Enhancement)

For Racket-style macro power, we could implement full syntax objects:

```go
type SyntaxObject struct {
    Expr      ast.Expr
    Scope     ScopeSet
    SourceLoc *sourcemap.SourceLocation
}

type ScopeSet struct {
    scopes []uint32
}

// With syntax objects, we can implement syntax-case:
// (syntax-case stx ()
//   [(id arg ...) #'(lambda (arg ...) body)])
```

### Macro-Generating Macros

```lisp
(defmacro define-accessors (struct-name & fields)
  `(do
     ,@(map (lambda (field)
              `(defn ,(symbol-append 'get- field) (obj)
                 (get-field obj ',field)))
            fields)))

;; Generates multiple function definitions:
(define-accessors person name age email)
```

Implementation requires tracking multiple generated definitions back to single macro call.

### Compiler Macros (Future)

Macros that run during compilation to optimize specific patterns:

```lisp
(define-compiler-macro + (&rest args)
  (if (all-literals? args)
      (fold-constants args)
      `(runtime-plus ,@args)))
```

---

## Implementation Phases

### Phase 1: Basic Gensym (MVP)
- Implement `GensymCounter` and `Gensym` function
- Add `Generated` flag to `Symbol` type
- Use `#` in generated names
- Basic macro expansion with gensym support

### Phase 2: Source Map Integration
- Link macro expansions to call sites
- Track which macro generated which code
- Implement `MacroExpansionInfo`
- Enhanced error messages showing macro context

### Phase 3: Scope Tracking
- Implement `Scope` type
- Track lexical scoping during expansion
- Associate symbols with their introduction scope
- Detect hygiene violations

### Phase 4: Advanced Error Reporting
- Full provenance chains in error messages
- Context display with macro expansion traces
- Debug mode showing all transformation layers

### Phase 5: Enhanced Macro System
- Deliberate capture with `intern`
- Symbol construction with `symbol-append`
- Macro-generating macros
- Syntax-case (if needed)

---

## Conclusion

Zylisp can implement fully hygienic macros thanks to Go's memory model. The key advantages:

✅ **No atom table constraints** - Generate unlimited symbols during compilation  
✅ **Automatic garbage collection** - Temporary symbols cleaned up after compilation  
✅ **Simple implementation** - Gensym is just counter + string formatting  
✅ **Integrated source tracking** - Macros work seamlessly with source maps  
✅ **Safe by default** - Hygiene prevents accidental capture  
✅ **Escape hatches** - Deliberate capture when needed via `intern`

The implementation work is straightforward:
1. Counter-based gensym (5 lines)
2. Source map integration (already designed)
3. Scope tracking (optional enhancement)
4. Error reporting (builds on source maps)

No fundamental obstacles exist, unlike in LFE/Erlang.
