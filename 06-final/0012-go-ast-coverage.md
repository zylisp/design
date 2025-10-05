---
number: 0012
title: Claude Code Prompt: Comprehensive Go AST Coverage Test Suite
author: Duncan McGreggor
created: 2025-10-02
updated: 2025-10-02
state: Draft
supersedes: None
superseded-by: None
---

# Claude Code Prompt: Comprehensive Go AST Coverage Test Suite

## Objective
Create a complete test suite of Go files that collectively demonstrate and exercise every AST node type defined in Go's `go/ast` package. This suite will serve as both a reference implementation and a testing framework for Go AST parsing tools.

## Project Structure Requirements

Create the following directory structure:
```
go-ast-coverage/
├── README.md
├── go.mod
├── main.go (orchestrator that runs all tests)
├── ast-nodes/
│   ├── basic_literals.go
│   ├── identifiers.go
│   ├── expressions.go
│   ├── statements.go
│   ├── declarations.go
│   ├── types.go
│   ├── function_types.go
│   ├── interface_types.go
│   ├── struct_types.go
│   ├── array_slice_types.go
│   ├── map_channel_types.go
│   ├── control_flow.go
│   ├── comments.go
│   ├── imports.go
│   ├── generics.go
│   └── edge_cases.go
├── ast-analyzer/
│   └── analyzer.go (AST inspection utility)
└── coverage-report/
    └── report.go (generates coverage report)
```

## Core Requirements

### 1. Compilation and Execution Requirements
- **Every `.go` file MUST compile successfully with `go build`**
- **Every `.go` file MUST be executable and produce meaningful stdout output**
- **Each file's output MUST clearly indicate which AST nodes it exercises**
- **The main.go orchestrator MUST run all test files and aggregate results**

### 2. AST Node Coverage Requirements
Based on the `go/ast` package, ensure coverage of ALL node types including:

#### Expression Nodes (`ast.Expr`)
- `*ast.BadExpr`
- `*ast.Ident`
- `*ast.Ellipsis`
- `*ast.BasicLit`
- `*ast.FuncLit`
- `*ast.CompositeLit`
- `*ast.ParenExpr`
- `*ast.SelectorExpr`
- `*ast.IndexExpr`
- `*ast.IndexListExpr` (Go 1.18+ generics)
- `*ast.SliceExpr`
- `*ast.TypeAssertExpr`
- `*ast.CallExpr`
- `*ast.StarExpr`
- `*ast.UnaryExpr`
- `*ast.BinaryExpr`
- `*ast.KeyValueExpr`

#### Statement Nodes (`ast.Stmt`)
- `*ast.BadStmt`
- `*ast.DeclStmt`
- `*ast.EmptyStmt`
- `*ast.LabeledStmt`
- `*ast.ExprStmt`
- `*ast.SendStmt`
- `*ast.IncDecStmt`
- `*ast.AssignStmt`
- `*ast.GoStmt`
- `*ast.DeferStmt`
- `*ast.ReturnStmt`
- `*ast.BranchStmt`
- `*ast.BlockStmt`
- `*ast.IfStmt`
- `*ast.CaseClause`
- `*ast.SwitchStmt`
- `*ast.TypeSwitchStmt`
- `*ast.CommClause`
- `*ast.SelectStmt`
- `*ast.ForStmt`
- `*ast.RangeStmt`

#### Declaration Nodes (`ast.Decl`)
- `*ast.BadDecl`
- `*ast.GenDecl`
- `*ast.FuncDecl`

#### Spec Nodes (`ast.Spec`)
- `*ast.ImportSpec`
- `*ast.ValueSpec`
- `*ast.TypeSpec`

#### Other Important Nodes
- `*ast.File`
- `*ast.Package`
- `*ast.Comment`
- `*ast.CommentGroup`
- `*ast.Field`
- `*ast.FieldList`
- `*ast.Scope`
- `*ast.Object`

### 3. Go Language Feature Coverage
Ensure 100% coverage of Go language features:

#### Basic Features
- All primitive types (bool, int variants, uint variants, float32/64, complex64/128, string, byte, rune)
- Variables, constants, type definitions
- Functions (regular, variadic, methods, anonymous)
- Arrays, slices, maps, channels
- Structs, interfaces, embedding
- Pointers and pointer operations

#### Advanced Features
- Generics (type parameters, type constraints, type inference)
- Goroutines and channels
- Defer and panic/recover
- Reflection usage
- Unsafe operations
- Build tags and conditional compilation
- CGO integration examples

#### Control Flow
- All loop types (for, range)
- All conditional types (if/else, switch, type switch, select)
- Labels and goto
- Break and continue with labels

#### Modern Go Features
- Modules and go.mod
- Generics and type parameters (Go 1.18+)
- Type sets in interfaces
- Type inference
- Any, comparable built-in types

## Implementation Guidelines

### 4. File-Specific Requirements

#### Each `.go` file should:
1. **Have a clear, focused purpose** - target 5-15 specific AST node types
2. **Include comprehensive documentation** explaining what AST nodes it covers
3. **Produce structured output** showing exactly which nodes were exercised
4. **Include edge cases** and boundary conditions for its targeted nodes
5. **Be independently runnable** with `go run filename.go`

#### Output Format Requirements
Each file should output in this format:
```
=== [FILENAME] AST Node Coverage ===
Exercising AST Nodes:
  ✓ ast.BasicLit (INT): 42
  ✓ ast.BasicLit (STRING): "hello"
  ✓ ast.BinaryExpr (ADD): 5 + 3 = 8
  ✓ ast.CallExpr: fmt.Println called
  ✓ ast.Ident: variable names [x, y, result]
Summary: 5 unique AST node types exercised
========================================
```

### 5. Special Implementation Requirements

#### AST Analyzer Component
Create `ast-analyzer/analyzer.go` with functions to:
- Parse any Go file and enumerate all AST nodes found
- Generate a detailed report of node types and frequencies
- Validate that a file exercises specific expected nodes
- Compare actual vs expected AST node coverage

#### Coverage Report Generator
Create `coverage-report/report.go` that:
- Analyzes all files in the test suite
- Generates a comprehensive coverage matrix
- Identifies any missing AST node types
- Produces both human-readable and machine-readable reports

#### Main Orchestrator
Create `main.go` that:
- Runs each test file in sequence
- Collects and aggregates all output
- Runs the AST analyzer on each file
- Generates final coverage statistics
- Reports any gaps in coverage

### 6. Code Quality Requirements

#### Testing and Validation
- Include unit tests for the analyzer and coverage components
- Add integration tests that verify complete coverage
- Include benchmarks for AST parsing performance
- Add linting and formatting checks

#### Documentation
- Comprehensive README.md with usage instructions
- Inline documentation for every function and complex logic
- Examples of how to extend the suite with new test cases
- Architecture documentation explaining the design decisions

#### Error Handling
- Robust error handling for file operations and AST parsing
- Graceful degradation when individual test files fail
- Clear error messages indicating exactly what failed and why

### 7. Advanced Requirements

#### Modern Go Practices
- Use Go modules with appropriate versioning
- Follow Go's official style guidelines
- Use context.Context for cancellable operations where appropriate
- Implement proper logging with structured output

#### Extensibility
- Design the system to easily add new test files
- Support for custom AST node validators
- Plugin architecture for different output formats
- Configuration system for customizing behavior

#### Performance Considerations
- Efficient AST parsing and traversal
- Concurrent execution of independent test files where safe
- Memory-efficient handling of large AST trees
- Benchmarking and performance regression detection

## Deliverables

### Phase 1: Core Implementation
1. All AST node test files with comprehensive coverage
2. AST analyzer utility
3. Main orchestrator
4. Basic coverage reporting

### Phase 2: Advanced Features
1. Coverage report generator with detailed analytics
2. Integration tests and validation suite
3. Performance benchmarks
4. Comprehensive documentation

### Phase 3: Polish and Extension
1. CI/CD integration examples
2. Plugin system for extensibility
3. Advanced reporting formats (JSON, HTML, etc.)
4. Integration with existing Go tooling

## Success Criteria

### Functional Requirements
- [ ] 100% coverage of all AST node types defined in go/ast
- [ ] 100% coverage of all Go language features
- [ ] All files compile and run successfully
- [ ] Comprehensive, accurate reporting of coverage
- [ ] Easy to understand and extend

### Quality Requirements
- [ ] Well-documented, maintainable code
- [ ] Comprehensive test suite with >95% code coverage
- [ ] Performance benchmarks showing acceptable performance
- [ ] Integration with standard Go tooling (go test, go build, etc.)

### Usability Requirements
- [ ] Clear, actionable output from all components
- [ ] Simple command-line interface for running tests
- [ ] Detailed documentation with examples
- [ ] Easy setup and configuration

## Notes and Considerations

- **Pay special attention to Go 1.18+ generics features** - these introduce new AST nodes
- **Include both positive and negative test cases** - exercise error conditions where appropriate
- **Consider different build environments** - ensure compatibility across Go versions
- **Think about maintainability** - the suite should be easy to update as Go evolves
- **Focus on practical usage** - the test cases should represent realistic Go code patterns

This comprehensive test suite will serve as the definitive reference for Go AST coverage and provide invaluable tooling for anyone working with Go's abstract syntax trees.