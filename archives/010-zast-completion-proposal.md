# Building Out Complete zast Support - Implementation Plan

**Project**: zast (formerly go-sexp-ast)  
**Goal**: Support 100% of Go's AST nodes for complete language coverage  
**Status**: Phase 1 Complete (~15% coverage)  
**Date**: October 2025

---

## Executive Summary

This document outlines the work required to extend zast from its current Phase 1 implementation (covering basic "Hello, world" programs) to **complete Go AST coverage**. The project involves implementing support for ~60 additional AST node types across expressions, statements, declarations, and types.

**Total Estimated Effort**: 14-22 days of development work  
**Total Lines of Code**: ~10,000-15,500 additional lines  
**Recommended Approach**: Five-wave implementation strategy

---

## Current State (Phase 1)

### What's Implemented ✅

**Expressions (5/48 total)**
- `Ident` - Identifiers
- `BasicLit` - Basic literals (strings, numbers, etc.)
- `CallExpr` - Function calls
- `SelectorExpr` - Field/method selection (e.g., `fmt.Println`)

**Statements (2/15 total)**
- `ExprStmt` - Expression statements
- `BlockStmt` - Block statements

**Declarations (2/4 total)**
- `GenDecl` - General declarations (currently IMPORT only)
- `FuncDecl` - Function declarations

**Specs (1/3 total)**
- `ImportSpec` - Import specifications

**Types (2/12 total)**
- `FuncType` - Function types
- `FieldList` / `Field` - Parameter/result lists

**Infrastructure**
- Lexer with position tracking
- Parser for generic S-expressions
- Builder (S-expr → Go AST)
- Writer (Go AST → S-expr)
- Pretty printer
- Complete test suite
- Working demo with full round-trip

**Capabilities**: Can handle simple programs with imports and function calls.

---

## What's Missing

### Expressions (43 additional nodes needed)

#### Literals & Identifiers
- `CompositeLit` - Composite literals: `Point{X: 1, Y: 2}`, `[]int{1, 2, 3}`
- `FuncLit` - Function literals (closures): `func(x int) int { return x * 2 }`
- `Ellipsis` - Variadic parameter indicator: `...int`

#### Operators
- `UnaryExpr` - Unary operations: `!x`, `-y`, `*ptr`, `&addr`, `<-ch`
- `BinaryExpr` - Binary operations: `x + y`, `a && b`, `i < len(arr)`

#### Indexing & Slicing
- `IndexExpr` - Array/slice/map indexing: `arr[i]`, `m[key]`
- `IndexListExpr` - Generic type instantiation: `List[int, string]` (Go 1.18+)
- `SliceExpr` - Slicing: `arr[1:5]`, `arr[1:5:10]`

#### Type Operations
- `StarExpr` - Pointer types: `*int`, `*MyStruct`
- `ParenExpr` - Parenthesized expressions: `(x + y) * z`
- `TypeAssertExpr` - Type assertions: `x.(string)`, `x.(type)`

#### Type Definitions
- `ArrayType` - Array types: `[10]int`, `[N]string`
- `MapType` - Map types: `map[string]int`
- `ChanType` - Channel types: `chan int`, `<-chan string`, `chan<- bool`
- `StructType` - Struct type definitions
- `InterfaceType` - Interface type definitions

#### Special
- `KeyValueExpr` - Key-value pairs in composite literals: `{key: value}`
- `BadExpr` - Placeholder for expressions with syntax errors

### Statements (13 additional nodes needed)

#### Assignment & Declaration
- `AssignStmt` - Assignments: `x = 5`, `x, y := 1, 2`, `x += 10`
- `DeclStmt` - Declaration statements (var/const/type inside functions)
- `IncDecStmt` - Increment/decrement: `x++`, `y--`

#### Control Flow
- `IfStmt` - If statements with optional init
- `ForStmt` - For loops (all variants)
- `RangeStmt` - For-range loops: `for k, v := range m { ... }`
- `SwitchStmt` - Switch statements
- `TypeSwitchStmt` - Type switch statements
- `SelectStmt` - Select statements (channel operations)
- `CaseClause` - Cases in switch statements
- `CommClause` - Cases in select statements

#### Jumps & Control
- `BranchStmt` - `break`, `continue`, `goto`, `fallthrough`
- `ReturnStmt` - Return statements
- `LabeledStmt` - Labeled statements: `Label: stmt`
- `GoStmt` - Goroutine launch: `go func() { ... }()`
- `DeferStmt` - Defer statements: `defer cleanup()`
- `SendStmt` - Channel send: `ch <- value`

#### Special
- `BadStmt` - Placeholder for statements with syntax errors
- `EmptyStmt` - Empty statement (just semicolon)

### Declaration Specs (2 additional nodes needed)

Currently have `ImportSpec`, need:
- `ValueSpec` - Variable and constant declarations
- `TypeSpec` - Type alias and type definitions

**Note**: `GenDecl` already handles IMPORT, CONST, TYPE, and VAR tokens. Just need to implement the additional spec types.

### Types (10 additional nodes needed)

Already implemented: `FuncType`, `FieldList`, `Field`

Need:
- Full `StructType` implementation with tags and embedded fields
- Full `InterfaceType` implementation with methods and embedded interfaces
- `ArrayType` - Array type specifications
- `MapType` - Map type specifications  
- `ChanType` - Channel type specifications with direction

### Supporting Nodes

#### Comments (currently written as nil)
- `Comment` - Single line or block comment
- `CommentGroup` - Group of adjacent comments

#### Scope & Objects (currently written as nil)
- `Scope` - Symbol table for a scope
- `Object` - Information about a declared name

---

## Complexity Analysis

### Level 1: Straightforward (~40 nodes)

These follow the same pattern as existing Phase 1 nodes:
- Parse keyword arguments
- Recursively build child nodes
- Construct Go AST node with proper fields

**Examples**: `UnaryExpr`, `BinaryExpr`, `ReturnStmt`, `ArrayType`, `MapType`

**Effort per node**: 20-30 lines each (writer + builder)
- Writer: ~800-1,200 lines total
- Builder: ~1,200-1,600 lines total
- Tests: ~2,000-3,000 lines total

### Level 2: Moderate Complexity (~15 nodes)

These require special handling for complex structures:

**Examples**:
- `IfStmt` - Init statement, condition, then/else branches
- `ForStmt` - Init, condition, post, body
- `SwitchStmt` - Multiple cases with fallthrough
- `CompositeLit` - Type information plus nested values
- `RangeStmt` - Key/value assignments plus range expression

**Effort per node**: 40-60 lines each
- Writer: ~600-900 lines total
- Builder: ~800-1,200 lines total
- Tests: ~1,500-2,500 lines total

### Level 3: Complex (~5 nodes)

These are genuinely complex with intricate semantics:

**Examples**:
- `InterfaceType` - Methods, embedded interfaces, type parameters (Go 1.18+)
- `StructType` - Tags, embedded fields, anonymous fields
- `Scope` / `Object` - Symbol tables with cross-references
- `IndexListExpr` - Generic type instantiation with type arguments
- `TypeSwitchStmt` - Type assertions with assignments

**Effort per node**: 80-150 lines each
- Writer: ~400-750 lines total
- Builder: ~600-1,000 lines total
- Tests: ~1,000-2,000 lines total

---

## Total Effort Estimate

| Component | Lines of Code | Time Estimate |
|-----------|---------------|---------------|
| Writer additions | 2,000-3,000 | 3-5 days |
| Builder additions | 3,000-4,500 | 4-6 days |
| Test coverage | 5,000-8,000 | 5-8 days |
| Debugging & polish | - | 2-3 days |
| **Total** | **10,000-15,500** | **14-22 days** |

**With AI assistance (Claude Code)**: Potentially 7-12 days with AI doing implementation and human doing review/testing.

---

## Implementation Strategy

### Recommended Approach: Wave-Based Implementation

Implement in five waves, ordered by complexity and dependencies:

## Wave 1: Easy Wins (3-4 days)

**Goal**: Add all straightforward nodes that follow existing patterns.

### Expressions
- `UnaryExpr` - Unary operations
- `BinaryExpr` - Binary operations
- `ParenExpr` - Parenthesized expressions
- `StarExpr` - Pointer dereference/address-of
- `IndexExpr` - Array/map indexing
- `SliceExpr` - Slice operations
- `KeyValueExpr` - Key-value pairs

### Statements
- `ReturnStmt` - Return statements
- `AssignStmt` - Assignment statements
- `IncDecStmt` - Increment/decrement
- `BranchStmt` - Break, continue, goto, fallthrough
- `DeferStmt` - Defer statements
- `GoStmt` - Goroutine launch
- `SendStmt` - Channel send
- `EmptyStmt` - Empty statements
- `LabeledStmt` - Labeled statements

### Types
- `ArrayType` - Array types
- `MapType` - Map types
- `ChanType` - Channel types

### Specs
- `ValueSpec` - Variable/constant declarations
- `TypeSpec` - Type declarations

**Deliverables**:
- ~25 new nodes implemented
- ~3,000-4,000 lines of code
- Test coverage for all new nodes
- Can now handle: variables, basic control flow, type definitions

**Test Programs After Wave 1**:
- Variable declarations and assignments
- Basic arithmetic and logic
- Simple type definitions
- Goroutines and channels (send only)

## Wave 2: Control Flow (3-4 days)

**Goal**: Add all control flow constructs.

### Statements
- `IfStmt` - If/else statements
- `ForStmt` - For loops (all variants)
- `RangeStmt` - For-range loops
- `SwitchStmt` - Switch statements
- `TypeSwitchStmt` - Type switch statements
- `SelectStmt` - Select statements
- `CaseClause` - Case clauses for switch
- `CommClause` - Communication clauses for select
- `DeclStmt` - Declaration statements in blocks

**Deliverables**:
- ~9 new nodes implemented
- ~2,500-3,500 lines of code
- Complex control flow test cases
- Can now handle: loops, conditionals, switches, channel selection

**Test Programs After Wave 2**:
- Fibonacci (recursive and iterative)
- FizzBuzz
- Simple HTTP server with routing
- Concurrent worker pools

## Wave 3: Complex Types (2-3 days)

**Goal**: Full support for composite types.

### Types
- `StructType` (full implementation)
  - Embedded fields
  - Field tags
  - Anonymous fields
- `InterfaceType` (full implementation)
  - Method sets
  - Embedded interfaces
  - Type constraints (Go 1.18+)

### Expressions
- `CompositeLit` - Composite literals
  - Struct literals
  - Array/slice literals
  - Map literals
- `TypeAssertExpr` - Type assertions
- `FuncLit` - Function literals (closures)

**Deliverables**:
- ~5 complex nodes implemented
- ~2,000-3,000 lines of code
- Complex type test cases
- Can now handle: full OOP patterns, closures, complex data structures

**Test Programs After Wave 3**:
- Interface-based polymorphism
- Struct embedding and composition
- Closure-based iterators
- Builder patterns

## Wave 4: Advanced Features (2-3 days)

**Goal**: Handle modern Go features and edge cases.

### Generics (Go 1.18+)
- `IndexListExpr` - Generic type instantiation
- Type parameters in `InterfaceType`
- Type constraints

### Comments & Documentation
- `Comment` - Single comments
- `CommentGroup` - Comment groups
- Proper doc comment handling

### Scope & Objects
- `Scope` - Symbol tables
- `Object` - Name declarations
- Cross-reference tracking

### Error Handling
- `BadExpr` - Malformed expressions
- `BadStmt` - Malformed statements
- Improved error messages

**Deliverables**:
- Generics support (if targeting Go 1.18+)
- Full comment preservation
- Symbol table support
- Better error handling
- ~1,500-2,500 lines of code

**Test Programs After Wave 4**:
- Generic data structures (List[T], Map[K,V])
- Fully documented packages
- Complex symbol resolution

## Wave 5: Polish & Integration (2-3 days)

**Goal**: Complete the implementation with testing and documentation.

### Testing
- 100% code coverage
- Edge case testing
- Malformed input handling
- Large real-world programs

### Documentation
- Complete API documentation
- Usage examples
- Migration guide from Phase 1

### Performance
- Benchmarking
- Optimization of hot paths
- Memory usage profiling

### Integration
- Test with real Go packages
- Standard library examples
- Third-party package support

**Deliverables**:
- Comprehensive test suite
- Complete documentation
- Performance benchmarks
- Real-world validation

**Test Programs After Wave 5**:
- Parse and round-trip entire Go stdlib packages
- Complex third-party libraries
- Large production codebases

---

## Alternative Strategy: By Feature Category

An alternative approach is to group by what features they enable:

### Category 1: Variables & Constants (2 days)
- `AssignStmt`, `ValueSpec`, `DeclStmt`
- `UnaryExpr`, `BinaryExpr`
- Basic arithmetic and logic

### Category 2: Control Flow (3 days)
- `IfStmt`, `ForStmt`, `RangeStmt`, `SwitchStmt`
- `BranchStmt`, `ReturnStmt`
- All loop and conditional constructs

### Category 3: Types & Composites (3 days)
- `StructType`, `InterfaceType`, `CompositeLit`
- `ArrayType`, `MapType`, `ChanType`
- Full type system support

### Category 4: Functions & Concurrency (2 days)
- `FuncLit`, `DeferStmt`, `GoStmt`
- `SelectStmt`, `SendStmt`
- Closures and goroutines

### Category 5: Advanced Features (2-3 days)
- Generics (Go 1.18+)
- Comments and documentation
- Scopes and objects

---

## Success Metrics

### After Each Wave

- [ ] All nodes in wave implemented in Writer
- [ ] All nodes in wave implemented in Builder
- [ ] Test coverage >90% for new nodes
- [ ] Round-trip tests pass for wave-specific programs
- [ ] Pretty printer handles new nodes correctly

### Final Success Criteria

- [ ] 100% Go AST node coverage
- [ ] Can parse and round-trip entire Go standard library
- [ ] Can handle real-world Go projects
- [ ] Test coverage >95%
- [ ] Complete documentation
- [ ] Performance acceptable for large files (>10k LOC)

---

## Risk Assessment

### Low Risk
- Wave 1 nodes (follow existing patterns exactly)
- Wave 2 control flow (well-understood semantics)
- Testing and documentation

### Medium Risk
- Wave 3 complex types (intricate nested structures)
- Wave 4 generics (newer feature, less documentation)
- Performance with very large files

### High Risk
- Scope/Object implementation (complex cross-references)
- Edge cases in type system
- Undocumented Go AST quirks

### Mitigation Strategies
- Start with low-risk waves to build momentum
- Extensive testing at each stage
- Reference existing Go AST tools for complex cases
- Incremental integration testing with real code

---

## Development Process

### For Each Wave

1. **Specification**: Create detailed `.md` specs for all nodes in wave
2. **Implementation**: Use Claude Code to implement from specs
3. **Review**: Human review of generated code
4. **Testing**: Write and run comprehensive tests
5. **Integration**: Test with real Go programs
6. **Documentation**: Update docs with new capabilities
7. **Commit**: Commit working wave before starting next

### Quality Gates

Before moving to next wave:
- [ ] All tests passing
- [ ] No compiler warnings
- [ ] Code reviewed
- [ ] Documentation updated
- [ ] Integration tests pass

---

## Next Steps

### Immediate Actions

1. **Review and approve this plan**
2. **Choose implementation strategy** (Wave-based recommended)
3. **Create Wave 1 specifications** (4-5 detailed `.md` files)
4. **Set up tracking** (GitHub issues, project board, etc.)
5. **Begin Wave 1 implementation**

### Wave 1 Specifications Needed

1. `wave1-expressions.md` - UnaryExpr, BinaryExpr, ParenExpr, StarExpr, IndexExpr, SliceExpr, KeyValueExpr
2. `wave1-statements.md` - ReturnStmt, AssignStmt, IncDecStmt, BranchStmt, DeferStmt, GoStmt, SendStmt, EmptyStmt, LabeledStmt
3. `wave1-types.md` - ArrayType, MapType, ChanType
4. `wave1-specs.md` - ValueSpec, TypeSpec
5. `wave1-integration-tests.md` - Test programs and validation criteria

---

## Timeline

### Optimistic (with Claude Code assistance): 7-9 days
- Wave 1: 2 days
- Wave 2: 2 days
- Wave 3: 1.5 days
- Wave 4: 1.5 days
- Wave 5: 1 day

### Realistic: 14-18 days
- Wave 1: 3 days
- Wave 2: 3 days
- Wave 3: 2.5 days
- Wave 4: 2.5 days
- Wave 5: 2 days

### Conservative: 20-25 days
- Wave 1: 4 days
- Wave 2: 4 days
- Wave 3: 3 days
- Wave 4: 3 days
- Wave 5: 3 days
- Buffer for unexpected issues: 3-5 days

---

## Conclusion

Extending zast to 100% Go AST coverage is a **well-scoped, achievable project**. The wave-based approach provides:

- **Clear milestones** at each wave completion
- **Incremental value** with each wave
- **Risk mitigation** by tackling simple nodes first
- **Testability** with real programs after each wave

The foundation from Phase 1 (lexer, parser, builder, writer, pretty printer, tests) provides a **proven pattern** to follow. The additional ~60 nodes are largely variations on what's already working.

**Recommended next step**: Create Wave 1 specifications and begin implementation.

---

*"The journey of a thousand nodes begins with a single AST." - Ancient Go Proverb*
