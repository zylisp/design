# Bootstrap Demo Implementation Specification

## Overview

Implement a complete end-to-end demonstration that proves the round-trip conversion works. This bootstrap demo will take Go source code, convert it to S-expressions, write it to disk, read it back, convert to Go AST, generate Go source, compile it, and run it.

## File Location

Create: `go-sexp-ast/cmd/demo/main.go`

## Purpose

This demo serves multiple purposes:

1. **Proof of Concept**: Demonstrates that the entire pipeline works
2. **Integration Test**: Tests all components working together
3. **Documentation**: Shows how to use the library
4. **Debugging Tool**: Helps identify issues in the round-trip process

## Complete Workflow

```
┌─────────────────────────────────────────────────────────────────┐
│                    Bootstrap Demo Workflow                       │
└─────────────────────────────────────────────────────────────────┘

1. Define Hello World Go source (in-memory string)
   ↓
2. Parse Go source → Go AST (using go/parser)
   ↓
3. Convert Go AST → S-expression (using Writer)
   ↓
4. Write S-expression to file (hello.sexp)
   ↓
5. Read S-expression from file
   ↓
6. Parse S-expression → generic tree (using Parser)
   ↓
7. Convert generic tree → Go AST (using Builder)
   ↓
8. Generate Go source from AST (using go/printer)
   ↓
9. Write Go source to file (hello_generated.go)
   ↓
10. Compile Go source → binary (using go build)
   ↓
11. Execute binary and capture output
   ↓
12. Verify output matches expected
```

## Implementation

### Constants

```go
const (
    helloWorldSource = `package main

import "fmt"

func main() {
    fmt.Println("Hello, world!")
}
`

    expectedOutput = "Hello, world!\n"
)
```

### Main Function Structure

```go
func main() {
    fmt.Println("=== Go-Lisp Bootstrap Demo ===\n")
    
    // Create temporary directory for our work
    tmpDir, err := os.MkdirTemp("", "go-sexp-demo-*")
    if err != nil {
        log.Fatal(err)
    }
    defer os.RemoveAll(tmpDir)
    
    fmt.Printf("Working directory: %s\n\n", tmpDir)
    
    // Step 1: Parse Go source to AST
    fmt.Println("Step 1: Parsing Go source to AST...")
    fset, astFile := parseGoSource(helloWorldSource)
    fmt.Println("✓ Parsed successfully\n")
    
    // Step 2: Convert AST to S-expression
    fmt.Println("Step 2: Converting AST to S-expression...")
    sexpText := astToSexp(fset, astFile)
    fmt.Println("✓ Converted successfully\n")
    
    // Step 3: Write S-expression to file
    fmt.Println("Step 3: Writing S-expression to file...")
    sexpPath := writeSexpToFile(tmpDir, sexpText)
    fmt.Printf("✓ Written to %s\n\n", sexpPath)
    
    // Step 4: Read S-expression from file
    fmt.Println("Step 4: Reading S-expression from file...")
    sexpTextRead := readSexpFromFile(sexpPath)
    fmt.Println("✓ Read successfully\n")
    
    // Step 5: Parse S-expression to generic tree
    fmt.Println("Step 5: Parsing S-expression to generic tree...")
    sexpTree := parseSexp(sexpTextRead)
    fmt.Println("✓ Parsed successfully\n")
    
    // Step 6: Convert S-expression to AST
    fmt.Println("Step 6: Converting S-expression to AST...")
    fset2, astFile2 := sexpToAST(sexpTree)
    fmt.Println("✓ Converted successfully\n")
    
    // Step 7: Generate Go source from AST
    fmt.Println("Step 7: Generating Go source from AST...")
    goSource := astToGoSource(fset2, astFile2)
    fmt.Println("✓ Generated successfully\n")
    
    // Step 8: Write Go source to file
    fmt.Println("Step 8: Writing Go source to file...")
    goPath := writeGoToFile(tmpDir, goSource)
    fmt.Printf("✓ Written to %s\n\n", goPath)
    
    // Step 9: Compile Go source to binary
    fmt.Println("Step 9: Compiling Go source to binary...")
    binaryPath := compileGo(tmpDir, goPath)
    fmt.Printf("✓ Compiled to %s\n\n", binaryPath)
    
    // Step 10: Execute binary
    fmt.Println("Step 10: Executing binary...")
    output := executeBinary(binaryPath)
    fmt.Printf("✓ Output: %q\n\n", output)
    
    // Step 11: Verify output
    fmt.Println("Step 11: Verifying output...")
    verifyOutput(output, expectedOutput)
    fmt.Println("✓ Output verified!\n")
    
    // Success!
    fmt.Println("=== SUCCESS! ===")
    fmt.Println("Complete round-trip successful:")
    fmt.Println("  Go → AST → S-expr → file → S-expr → AST → Go → binary → execution")
    fmt.Println("\nAll components working correctly! 🎉")
    
    // Optionally show the S-expression
    if len(os.Args) > 1 && os.Args[1] == "--show-sexp" {
        fmt.Println("\n=== S-Expression Output ===")
        fmt.Println(sexpText)
    }
}
```

### Helper Functions

Implement each step as a separate function for clarity:

#### Step 1: Parse Go Source

```go
func parseGoSource(source string) (*token.FileSet, *ast.File) {
    fset := token.NewFileSet()
    file, err := parser.ParseFile(fset, "hello.go", source, parser.ParseComments)
    if err != nil {
        log.Fatalf("Failed to parse Go source: %v", err)
    }
    return fset, file
}
```

#### Step 2: AST to S-expression

```go
func astToSexp(fset *token.FileSet, file *ast.File) string {
    writer := NewWriter(fset)
    sexpText, err := writer.WriteProgram([]*ast.File{file})
    if err != nil {
        log.Fatalf("Failed to convert AST to S-expression: %v", err)
    }
    return sexpText
}
```

#### Step 3: Write S-expression to File

```go
func writeSexpToFile(dir string, sexpText string) string {
    path := filepath.Join(dir, "hello.sexp")
    err := os.WriteFile(path, []byte(sexpText), 0644)
    if err != nil {
        log.Fatalf("Failed to write S-expression to file: %v", err)
    }
    return path
}
```

#### Step 4: Read S-expression from File

```go
func readSexpFromFile(path string) string {
    data, err := os.ReadFile(path)
    if err != nil {
        log.Fatalf("Failed to read S-expression from file: %v", err)
    }
    return string(data)
}
```

#### Step 5: Parse S-expression

```go
func parseSexp(sexpText string) sexp.SExp {
    parser := sexp.NewParser(sexpText)
    tree, err := parser.Parse()
    if err != nil {
        log.Fatalf("Failed to parse S-expression: %v", err)
    }
    return tree
}
```

#### Step 6: S-expression to AST

```go
func sexpToAST(tree sexp.SExp) (*token.FileSet, *ast.File) {
    builder := NewBuilder()
    fset, files, err := builder.BuildProgram(tree)
    if err != nil {
        log.Fatalf("Failed to convert S-expression to AST: %v", err)
    }
    if len(files) != 1 {
        log.Fatalf("Expected 1 file, got %d", len(files))
    }
    return fset, files[0]
}
```

#### Step 7: AST to Go Source

```go
func astToGoSource(fset *token.FileSet, file *ast.File) string {
    var buf bytes.Buffer
    err := printer.Fprint(&buf, fset, file)
    if err != nil {
        log.Fatalf("Failed to generate Go source: %v", err)
    }
    return buf.String()
}
```

#### Step 8: Write Go Source to File

```go
func writeGoToFile(dir string, source string) string {
    path := filepath.Join(dir, "hello_generated.go")
    err := os.WriteFile(path, []byte(source), 0644)
    if err != nil {
        log.Fatalf("Failed to write Go source to file: %v", err)
    }
    return path
}
```

#### Step 9: Compile Go Source

```go
func compileGo(dir string, goPath string) string {
    binaryPath := filepath.Join(dir, "hello")
    
    cmd := exec.Command("go", "build", "-o", binaryPath, goPath)
    cmd.Dir = dir
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        log.Fatalf("Failed to compile Go source: %v\nOutput: %s", err, output)
    }
    
    return binaryPath
}
```

#### Step 10: Execute Binary

```go
func executeBinary(binaryPath string) string {
    cmd := exec.Command(binaryPath)
    output, err := cmd.Output()
    if err != nil {
        log.Fatalf("Failed to execute binary: %v", err)
    }
    return string(output)
}
```

#### Step 11: Verify Output

```go
func verifyOutput(actual, expected string) {
    if actual != expected {
        log.Fatalf("Output mismatch!\nExpected: %q\nActual: %q", expected, actual)
    }
}
```

## Usage

### Basic Run

```bash
cd go-sexp-ast
go run cmd/demo/main.go
```

Expected output:
```
=== Go-Lisp Bootstrap Demo ===

Working directory: /tmp/go-sexp-demo-1234567

Step 1: Parsing Go source to AST...
✓ Parsed successfully

Step 2: Converting AST to S-expression...
✓ Converted successfully

Step 3: Writing S-expression to file...
✓ Written to /tmp/go-sexp-demo-1234567/hello.sexp

Step 4: Reading S-expression from file...
✓ Read successfully

Step 5: Parsing S-expression to generic tree...
✓ Parsed successfully

Step 6: Converting S-expression to AST...
✓ Converted successfully

Step 7: Generating Go source from AST...
✓ Generated successfully

Step 8: Writing Go source to file...
✓ Written to /tmp/go-sexp-demo-1234567/hello_generated.go

Step 9: Compiling Go source to binary...
✓ Compiled to /tmp/go-sexp-demo-1234567/hello

Step 10: Executing binary...
✓ Output: "Hello, world!\n"

Step 11: Verifying output...
✓ Output verified!

=== SUCCESS! ===
Complete round-trip successful:
  Go → AST → S-expr → file → S-expr → AST → Go → binary → execution

All components working correctly! 🎉
```

### Show S-expression Output

```bash
go run cmd/demo/main.go --show-sexp
```

This will include the S-expression output at the end.

## Advanced Features (Optional)

### Preserve Intermediate Files

Add a flag to keep the temporary directory:

```go
var keepFiles = flag.Bool("keep", false, "Keep intermediate files")

func main() {
    flag.Parse()
    
    tmpDir, err := os.MkdirTemp("", "go-sexp-demo-*")
    if err != nil {
        log.Fatal(err)
    }
    
    if !*keepFiles {
        defer os.RemoveAll(tmpDir)
    } else {
        fmt.Printf("Files preserved in: %s\n", tmpDir)
    }
    
    // ... rest of main
}
```

### Verbose Mode

Add detailed logging:

```go
var verbose = flag.Bool("verbose", false, "Verbose output")

func log(format string, args ...interface{}) {
    if *verbose {
        fmt.Printf("  → "+format+"\n", args...)
    }
}
```

### Diff Original vs Generated

Compare the original Go source with the generated Go source:

```go
func compareSource(original, generated string) {
    if original == generated {
        fmt.Println("✓ Generated source is identical to original!")
        return
    }
    
    fmt.Println("⚠ Generated source differs from original (this is OK - formatting may differ)")
    
    // Show diff if verbose
    if *verbose {
        fmt.Println("\n=== Original ===")
        fmt.Println(original)
        fmt.Println("\n=== Generated ===")
        fmt.Println(generated)
    }
}
```

## Testing the Demo

Create a test that runs the demo:

```go
// cmd/demo/main_test.go

func TestDemo(t *testing.T) {
    // Capture stdout
    old := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w
    
    // Run main
    main()
    
    // Restore stdout
    w.Close()
    os.Stdout = old
    
    // Read output
    var buf bytes.Buffer
    io.Copy(&buf, r)
    output := buf.String()
    
    // Verify success message
    assert.Contains(t, output, "SUCCESS")
    assert.Contains(t, output, "All components working correctly")
}
```

## Error Handling Examples

Add examples of what happens when things go wrong:

```go
func demonstrateErrorHandling() {
    fmt.Println("\n=== Error Handling Demo ===\n")
    
    // Example 1: Invalid S-expression
    fmt.Println("Example 1: Invalid S-expression")
    invalidSexp := "(File :package"  // unclosed
    parser := sexp.NewParser(invalidSexp)
    _, err := parser.Parse()
    fmt.Printf("  Error (expected): %v\n\n", err)
    
    // Example 2: Missing required field
    fmt.Println("Example 2: Missing required field in AST")
    missingSexp := "(File :package 1)"  // missing :name
    // ... demonstrate error
}
```

## Success Criteria

- Demo runs without errors on a fresh system
- All intermediate files are created correctly
- Binary executes and produces correct output
- Clear, informative output at each step
- Helpful error messages if something fails
- Optional verbose mode for debugging
- Optional preservation of intermediate files
- Clean code with good comments

## Dependencies

Add to `go.mod`:

```
module go-sexp-ast

go 1.21

require (
    // No external dependencies - using only stdlib!
)
```

## Documentation

Add a README section explaining how to run the demo:

```markdown
## Quick Start

Run the bootstrap demo to see the complete round-trip in action:

```bash
go run cmd/demo/main.go
```

This will:
1. Parse a "Hello, world!" Go program
2. Convert it to S-expressions
3. Write it to disk
4. Read it back
5. Convert back to Go AST
6. Generate Go source
7. Compile and run it

The demo proves that the entire pipeline works correctly.
```

## Notes

- Keep the demo simple and focused
- Prioritize clarity over cleverness
- Include plenty of status messages
- Make errors obvious and actionable
- The demo should inspire confidence in the system
- It should be easy to modify for experimentation
- Consider it living documentation

## Future Enhancements

Ideas for extending the demo:

1. **Multiple examples**: Hello world, fibonacci, http server
2. **Comparison mode**: Compare original vs generated source
3. **Benchmark mode**: Measure performance of each step
4. **Interactive mode**: REPL for experimenting with conversions
5. **Visualization**: Generate diagrams showing the AST structure
