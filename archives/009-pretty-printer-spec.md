# S-Expression Pretty Printer Implementation Specification

## Overview

Implement a sophisticated pretty printer for S-expressions that produces beautiful, readable output. This printer understands the structure of our canonical Go AST format and formats it with intelligent indentation, alignment, and line breaking.

## File Location

Create: `go-sexp-ast/sexp/pretty.go`

## Design Philosophy

The pretty printer should produce output that is:
- **Readable**: Easy to understand the structure at a glance
- **Consistent**: Same input always produces same output
- **Scannable**: Important information stands out
- **Idiomatic**: Follows Lisp formatting conventions
- **Aligned**: Related elements line up visually

## Core Structures

```go
type PrettyPrinter struct {
    buf           strings.Builder
    indentWidth   int
    maxLineWidth  int
    currentColumn int
    config        *Config
}

type Config struct {
    IndentWidth   int  // Spaces per indent level (default: 2)
    MaxLineWidth  int  // Target maximum line width (default: 80)
    AlignKeywords bool // Align keyword-value pairs (default: true)
    CompactSmall  bool // Keep small lists on one line (default: true)
    CompactLimit  int  // Max chars for compact lists (default: 60)
}

type FormStyle int

const (
    StyleDefault FormStyle = iota
    StyleKeywordPairs  // Format as :key value pairs with alignment
    StyleCompact       // Try to keep on one line
    StyleBody          // Special body formatting (BlockStmt, etc.)
    StyleList          // Simple list formatting
)
```

## Public API

```go
// NewPrettyPrinter creates a new pretty printer with default config
func NewPrettyPrinter() *PrettyPrinter

// NewPrettyPrinterWithConfig creates a printer with custom config
func NewPrettyPrinterWithConfig(config *Config) *PrettyPrinter

// Format formats an S-expression and returns the result
func (p *PrettyPrinter) Format(sexp SExp) string

// FormatToWriter formats an S-expression to an io.Writer
func (p *PrettyPrinter) FormatToWriter(sexp SExp, w io.Writer) error

// DefaultConfig returns the default configuration
func DefaultConfig() *Config
```

## Special Form Recognition

Define how different node types should be formatted:

```go
var formStyles = map[string]FormStyle{
    // Top-level structures
    "Program":   StyleKeywordPairs,
    "File":      StyleKeywordPairs,
    "FileSet":   StyleKeywordPairs,
    "FileInfo":  StyleKeywordPairs,
    
    // Declarations
    "FuncDecl":  StyleKeywordPairs,
    "GenDecl":   StyleKeywordPairs,
    
    // Types
    "FuncType":  StyleKeywordPairs,
    "FieldList": StyleKeywordPairs,
    "Field":     StyleKeywordPairs,
    
    // Statements
    "BlockStmt": StyleBody,
    "ExprStmt":  StyleKeywordPairs,
    
    // Expressions
    "CallExpr":     StyleKeywordPairs,
    "SelectorExpr": StyleCompact,
    "Ident":        StyleCompact,
    "BasicLit":     StyleCompact,
    
    // Specs
    "ImportSpec": StyleKeywordPairs,
    
    // Metadata
    "CommentGroup": StyleList,
    "Comment":      StyleCompact,
    "Scope":        StyleKeywordPairs,
    "Object":       StyleKeywordPairs,
}

func getFormStyle(nodeName string) FormStyle {
    if style, ok := formStyles[nodeName]; ok {
        return style
    }
    return StyleDefault
}
```

## Core Formatting Logic

### Main Format Method

```go
func (p *PrettyPrinter) Format(sexp SExp) string {
    p.buf.Reset()
    p.currentColumn = 0
    p.format(sexp, 0)
    return p.buf.String()
}

func (p *PrettyPrinter) format(sexp SExp, depth int) {
    switch s := sexp.(type) {
    case *Symbol:
        p.formatSymbol(s)
    case *Keyword:
        p.formatKeyword(s)
    case *String:
        p.formatString(s)
    case *Number:
        p.formatNumber(s)
    case *Nil:
        p.formatNil(s)
    case *List:
        p.formatList(s, depth)
    }
}
```

### Atomic Value Formatting

```go
func (p *PrettyPrinter) formatSymbol(s *Symbol) {
    p.write(s.Value)
}

func (p *PrettyPrinter) formatKeyword(k *Keyword) {
    p.write(":")
    p.write(k.Name)
}

func (p *PrettyPrinter) formatString(s *String) {
    p.write(`"`)
    p.write(s.Value)  // Already escaped
    p.write(`"`)
}

func (p *PrettyPrinter) formatNumber(n *Number) {
    p.write(n.Value)
}

func (p *PrettyPrinter) formatNil(n *Nil) {
    p.write("nil")
}
```

### List Formatting Strategy

```go
func (p *PrettyPrinter) formatList(list *List, depth int) {
    if len(list.Elements) == 0 {
        p.write("()")
        return
    }
    
    // Get the node type (first element)
    var nodeName string
    if sym, ok := list.Elements[0].(*Symbol); ok {
        nodeName = sym.Value
    }
    
    // Determine formatting style
    style := getFormStyle(nodeName)
    
    // Try compact formatting first if enabled
    if p.config.CompactSmall && p.shouldBeCompact(list, style) {
        if p.tryCompactFormat(list, depth) {
            return
        }
    }
    
    // Otherwise use style-specific formatting
    switch style {
    case StyleKeywordPairs:
        p.formatKeywordPairs(list, depth)
    case StyleCompact:
        p.formatCompact(list, depth)
    case StyleBody:
        p.formatBody(list, depth)
    case StyleList:
        p.formatSimpleList(list, depth)
    default:
        p.formatDefault(list, depth)
    }
}
```

### Compact Formatting

Try to fit the entire list on one line:

```go
func (p *PrettyPrinter) shouldBeCompact(list *List, style FormStyle) bool {
    // Only certain styles can be compact
    if style != StyleCompact && len(list.Elements) > 4 {
        return false
    }
    
    // Check estimated length
    est := p.estimateLength(list)
    return est < p.config.CompactLimit
}

func (p *PrettyPrinter) tryCompactFormat(list *List, depth int) bool {
    // Save current state
    savedBuf := p.buf.String()
    savedCol := p.currentColumn
    
    // Try compact format
    p.write("(")
    for i, elem := range list.Elements {
        if i > 0 {
            p.write(" ")
        }
        p.format(elem, depth)
        
        // If we exceeded line width, abort
        if p.currentColumn > p.config.MaxLineWidth {
            // Restore state
            p.buf.Reset()
            p.buf.WriteString(savedBuf)
            p.currentColumn = savedCol
            return false
        }
    }
    p.write(")")
    return true
}

func (p *PrettyPrinter) estimateLength(sexp SExp) int {
    switch s := sexp.(type) {
    case *Symbol:
        return len(s.Value)
    case *Keyword:
        return len(s.Name) + 1
    case *String:
        return len(s.Value) + 2
    case *Number:
        return len(s.Value)
    case *Nil:
        return 3
    case *List:
        total := 2 // for parens
        for i, elem := range s.Elements {
            if i > 0 {
                total += 1 // space
            }
            total += p.estimateLength(elem)
        }
        return total
    }
    return 0
}
```

### Keyword-Pair Formatting with Alignment

The most sophisticated formatting - aligns keywords and values:

```go
func (p *PrettyPrinter) formatKeywordPairs(list *List, depth int) {
    if len(list.Elements) == 0 {
        p.write("()")
        return
    }
    
    p.write("(")
    
    // First element (node type) on same line
    p.format(list.Elements[0], depth)
    
    // Collect keyword-value pairs and find max keyword width
    pairs := p.extractKeywordPairs(list.Elements[1:])
    maxKeywordWidth := p.findMaxKeywordWidth(pairs)
    
    // Format each pair with alignment
    for _, pair := range pairs {
        p.newline()
        p.indent(depth + 1)
        
        // Write keyword
        p.format(pair.keyword, depth + 1)
        
        // Pad to alignment column
        if p.config.AlignKeywords && maxKeywordWidth > 0 {
            keywordWidth := p.estimateLength(pair.keyword)
            padding := maxKeywordWidth - keywordWidth
            p.writeSpaces(padding + 1)
        } else {
            p.write(" ")
        }
        
        // Write value
        p.format(pair.value, depth + 1)
    }
    
    p.write(")")
}

type keywordPair struct {
    keyword SExp
    value   SExp
}

func (p *PrettyPrinter) extractKeywordPairs(elements []SExp) []keywordPair {
    var pairs []keywordPair
    
    for i := 0; i < len(elements); i += 2 {
        if i+1 >= len(elements) {
            break
        }
        
        pairs = append(pairs, keywordPair{
            keyword: elements[i],
            value:   elements[i+1],
        })
    }
    
    return pairs
}

func (p *PrettyPrinter) findMaxKeywordWidth(pairs []keywordPair) int {
    max := 0
    for _, pair := range pairs {
        width := p.estimateLength(pair.keyword)
        if width > max {
            max = width
        }
    }
    return max
}
```

### Body Formatting

Special formatting for bodies (like BlockStmt):

```go
func (p *PrettyPrinter) formatBody(list *List, depth int) {
    if len(list.Elements) == 0 {
        p.write("()")
        return
    }
    
    p.write("(")
    
    // First element (node type)
    p.format(list.Elements[0], depth)
    
    // Process keyword-value pairs, but :list gets special treatment
    i := 1
    for i < len(list.Elements) {
        if i+1 >= len(list.Elements) {
            break
        }
        
        keyword := list.Elements[i]
        value := list.Elements[i+1]
        
        // Check if this is the :list field
        if kw, ok := keyword.(*Keyword); ok && kw.Name == "list" {
            // Format list body with extra indentation
            p.newline()
            p.indent(depth + 1)
            p.format(keyword, depth + 1)
            p.write(" ")
            
            if valueList, ok := value.(*List); ok {
                p.formatListBody(valueList, depth + 2)
            } else {
                p.format(value, depth + 1)
            }
        } else {
            // Regular keyword-value pair
            p.newline()
            p.indent(depth + 1)
            p.format(keyword, depth + 1)
            p.write(" ")
            p.format(value, depth + 1)
        }
        
        i += 2
    }
    
    p.write(")")
}

func (p *PrettyPrinter) formatListBody(list *List, depth int) {
    if len(list.Elements) == 0 {
        p.write("()")
        return
    }
    
    p.write("(")
    for i, elem := range list.Elements {
        if i > 0 {
            p.newline()
            p.indent(depth)
        } else {
            p.newline()
            p.indent(depth)
        }
        p.format(elem, depth)
    }
    p.write(")")
}
```

### Simple List Formatting

For lists that aren't keyword-pairs:

```go
func (p *PrettyPrinter) formatSimpleList(list *List, depth int) {
    p.write("(")
    
    for i, elem := range list.Elements {
        if i > 0 {
            p.newline()
            p.indent(depth + 1)
        }
        p.format(elem, depth + 1)
    }
    
    p.write(")")
}
```

### Default Formatting

Fallback for unknown structures:

```go
func (p *PrettyPrinter) formatDefault(list *List, depth int) {
    p.write("(")
    
    for i, elem := range list.Elements {
        if i > 0 {
            if p.shouldBreakBefore(elem) {
                p.newline()
                p.indent(depth + 1)
            } else {
                p.write(" ")
            }
        }
        p.format(elem, depth + 1)
    }
    
    p.write(")")
}

func (p *PrettyPrinter) shouldBreakBefore(sexp SExp) bool {
    // Break before lists
    _, isList := sexp.(*List)
    return isList
}
```

## Helper Methods

```go
func (p *PrettyPrinter) write(s string) {
    p.buf.WriteString(s)
    p.currentColumn += len(s)
}

func (p *PrettyPrinter) writeSpaces(n int) {
    for i := 0; i < n; i++ {
        p.buf.WriteString(" ")
        p.currentColumn++
    }
}

func (p *PrettyPrinter) newline() {
    p.buf.WriteString("\n")
    p.currentColumn = 0
}

func (p *PrettyPrinter) indent(depth int) {
    spaces := depth * p.config.IndentWidth
    p.writeSpaces(spaces)
}
```

## Example Output

### Input S-Expression (compact from Writer)

```lisp
(File :package 1 :name (Ident :namepos 9 :name "main" :obj nil) :decls ((GenDecl :doc nil :tok IMPORT :tokpos 15 :lparen 0 :specs ((ImportSpec :doc nil :name nil :path (BasicLit :valuepos 22 :kind STRING :value "\"fmt\"") :comment nil :endpos 27)) :rparen 0) (FuncDecl :doc nil :recv nil :name (Ident :namepos 33 :name "main" :obj nil) :type (FuncType :func 28 :params (FieldList :opening 37 :list () :closing 38) :results nil) :body (BlockStmt :lbrace 40 :list ((ExprStmt :x (CallExpr :fun (SelectorExpr :x (Ident :namepos 46 :name "fmt" :obj nil) :sel (Ident :namepos 50 :name "Println" :obj nil)) :lparen 57 :args ((BasicLit :valuepos 58 :kind STRING :value "\"Hello, world!\"")) :ellipsis 0 :rparen 74))) :rbrace 76))) :scope nil :imports ((ImportSpec :doc nil :name nil :path (BasicLit :valuepos 22 :kind STRING :value "\"fmt\"") :comment nil :endpos 27)) :unresolved () :comments ())
```

### Pretty-Printed Output

```lisp
(File
  :package    1
  :name       (Ident :namepos 9 :name "main" :obj nil)
  :decls      (
                (GenDecl
                  :doc     nil
                  :tok     IMPORT
                  :tokpos  15
                  :lparen  0
                  :specs   (
                             (ImportSpec
                               :doc      nil
                               :name     nil
                               :path     (BasicLit :valuepos 22 :kind STRING :value "\"fmt\"")
                               :comment  nil
                               :endpos   27))
                  :rparen  0)
                (FuncDecl
                  :doc   nil
                  :recv  nil
                  :name  (Ident :namepos 33 :name "main" :obj nil)
                  :type  (FuncType
                           :func     28
                           :params   (FieldList :opening 37 :list () :closing 38)
                           :results  nil)
                  :body  (BlockStmt
                           :lbrace  40
                           :list    (
                                      (ExprStmt
                                        :x  (CallExpr
                                              :fun       (SelectorExpr
                                                           :x    (Ident :namepos 46 :name "fmt" :obj nil)
                                                           :sel  (Ident :namepos 50 :name "Println" :obj nil))
                                              :lparen    57
                                              :args      ((BasicLit :valuepos 58 :kind STRING :value "\"Hello, world!\""))
                                              :ellipsis  0
                                              :rparen    74)))
                           :rbrace  76)))
  :scope      nil
  :imports    (
                (ImportSpec
                  :doc      nil
                  :name     nil
                  :path     (BasicLit :valuepos 22 :kind STRING :value "\"fmt\"")
                  :comment  nil
                  :endpos   27))
  :unresolved ()
  :comments   ())
```

## Testing Requirements

Create `pretty_test.go` with comprehensive tests:

### 1. Basic Formatting Tests

```go
func TestFormatAtomicValues(t *testing.T) {
    tests := []struct {
        name     string
        input    SExp
        expected string
    }{
        {"symbol", &Symbol{Value: "foo"}, "foo"},
        {"keyword", &Keyword{Name: "name"}, ":name"},
        {"string", &String{Value: "hello"}, `"hello"`},
        {"number", &Number{Value: "42"}, "42"},
        {"nil", &Nil{}, "nil"},
    }
    
    pp := NewPrettyPrinter()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := pp.Format(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### 2. Compact Formatting Tests

```go
func TestCompactFormatting(t *testing.T) {
    // Small list should stay on one line
    input := &List{Elements: []SExp{
        &Symbol{Value: "Ident"},
        &Keyword{Name: "namepos"},
        &Number{Value: "9"},
        &Keyword{Name: "name"},
        &String{Value: "main"},
    }}
    
    pp := NewPrettyPrinter()
    result := pp.Format(input)
    
    // Should be on one line (no newlines)
    assert.NotContains(t, result, "\n")
    assert.Equal(t, `(Ident :namepos 9 :name "main")`, result)
}
```

### 3. Alignment Tests

```go
func TestKeywordAlignment(t *testing.T) {
    input := &List{Elements: []SExp{
        &Symbol{Value: "File"},
        &Keyword{Name: "package"},
        &Number{Value: "1"},
        &Keyword{Name: "name"},
        &Symbol{Value: "main"},
        &Keyword{Name: "decls"},
        &List{Elements: []SExp{}},
    }}
    
    pp := NewPrettyPrinter()
    result := pp.Format(input)
    
    // Check that keywords are aligned
    lines := strings.Split(result, "\n")
    assert.True(t, len(lines) > 3)
    
    // Extract keyword column positions
    var positions []int
    for _, line := range lines[1:] { // skip first line
        if idx := strings.Index(line, ":"); idx != -1 {
            positions = append(positions, idx)
        }
    }
    
    // All keywords should start at same column
    if len(positions) > 1 {
        first := positions[0]
        for _, pos := range positions[1:] {
            assert.Equal(t, first, pos, "Keywords should be aligned")
        }
    }
}
```

### 4. Body Formatting Tests

```go
func TestBodyFormatting(t *testing.T) {
    input := &List{Elements: []SExp{
        &Symbol{Value: "BlockStmt"},
        &Keyword{Name: "lbrace"},
        &Number{Value: "10"},
        &Keyword{Name: "list"},
        &List{Elements: []SExp{
            &List{Elements: []SExp{&Symbol{Value: "ExprStmt"}}},
            &List{Elements: []SExp{&Symbol{Value: "ReturnStmt"}}},
        }},
        &Keyword{Name: "rbrace"},
        &Number{Value: "20"},
    }}
    
    pp := NewPrettyPrinter()
    result := pp.Format(input)
    
    // Verify list elements are indented
    assert.Contains(t, result, "ExprStmt")
    assert.Contains(t, result, "ReturnStmt")
}
```

### 5. Configuration Tests

```go
func TestCustomConfiguration(t *testing.T) {
    config := &Config{
        IndentWidth:   4,
        MaxLineWidth:  40,
        AlignKeywords: false,
        CompactSmall:  false,
    }
    
    pp := NewPrettyPrinterWithConfig(config)
    
    input := &List{Elements: []SExp{
        &Symbol{Value: "Test"},
        &Keyword{Name: "a"},
        &Number{Value: "1"},
    }}
    
    result := pp.Format(input)
    
    // With 4-space indent
    lines := strings.Split(result, "\n")
    if len(lines) > 1 {
        assert.True(t, strings.HasPrefix(lines[1], "    "))
    }
}
```

### 6. Round-Trip Tests

```go
func TestPrettyPrintRoundTrip(t *testing.T) {
    // Parse compact S-expression
    input := `(File :package 1 :name (Ident :namepos 9 :name "main"))`
    parser := NewParser(input)
    sexp, err := parser.Parse()
    require.NoError(t, err)
    
    // Pretty print it
    pp := NewPrettyPrinter()
    pretty := pp.Format(sexp)
    
    // Parse the pretty-printed version
    parser2 := NewParser(pretty)
    sexp2, err := parser2.Parse()
    require.NoError(t, err)
    
    // Should produce same structure
    // (implement DeepEqual for SExp types)
    assert.True(t, sexpsEqual(sexp, sexp2))
}
```

### 7. Real-World Example Tests

```go
func TestCompleteFileFormatting(t *testing.T) {
    // Use actual hello world S-expression from writer
    input := loadHelloWorldSexp(t)
    
    pp := NewPrettyPrinter()
    result := pp.Format(input)
    
    // Should be readable
    assert.Contains(t, result, "File")
    assert.Contains(t, result, "FuncDecl")
    assert.Contains(t, result, "Hello, world!")
    
    // Should have consistent indentation
    // Should have aligned keywords
    // Should be parseable
    parser := NewParser(result)
    _, err := parser.Parse()
    assert.NoError(t, err)
}
```

## Usage Examples

### Basic Usage

```go
// Format an S-expression
pp := sexp.NewPrettyPrinter()
pretty := pp.Format(mySexp)
fmt.Println(pretty)
```

### Custom Configuration

```go
config := &sexp.Config{
    IndentWidth:   4,
    MaxLineWidth:  100,
    AlignKeywords: true,
    CompactSmall:  true,
    CompactLimit:  80,
}

pp := sexp.NewPrettyPrinterWithConfig(config)
pretty := pp.Format(mySexp)
```

### Format to File

```go
pp := sexp.NewPrettyPrinter()
file, err := os.Create("output.sexp")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

err = pp.FormatToWriter(mySexp, file)
```

### Integration with Writer

```go
// Write AST to S-expression
writer := NewWriter(fset)
sexpText, _ := writer.WriteProgram(files)

// Parse it
parser := sexp.NewParser(sexpText)
sexpTree, _ := parser.Parse()

// Pretty print it
pp := sexp.NewPrettyPrinter()
prettyText := pp.Format(sexpTree)

// Write pretty version to file
os.WriteFile("output.sexp", []byte(prettyText), 0644)
```

## CLI Tool (Optional)

Create `cmd/sexp-fmt/main.go` for formatting files:

```go
func main() {
    flag.Parse()
    
    if flag.NArg() == 0 {
        fmt.Fprintln(os.Stderr, "Usage: sexp-fmt <file.sexp>")
        os.Exit(1)
    }
    
    filename := flag.Arg(0)
    data, err := os.ReadFile(filename)
    if err != nil {
        log.Fatal(err)
    }
    
    parser := sexp.NewParser(string(data))
    tree, err := parser.Parse()
    if err != nil {
        log.Fatal(err)
    }
    
    pp := sexp.NewPrettyPrinter()
    pretty := pp.Format(tree)
    
    // Write back to file
    err = os.WriteFile(filename, []byte(pretty), 0644)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Formatted %s\n", filename)
}
```

## Success Criteria

- Produces readable, consistently formatted output
- Keywords align vertically within the same node
- Compact lists stay on one line when appropriate
- Long lists break intelligently
- Body structures (BlockStmt) format specially
- Configurable indentation and line width
- Round-trip preserves structure (parse → format → parse = same tree)
- Comprehensive test coverage
- Clean, maintainable code

## Performance Considerations

- Use `strings.Builder` for efficient string concatenation
- Estimate lengths before formatting to make smart decisions
- Cache form styles in a map for fast lookup
- Don't over-optimize - readability is more important than speed for pretty printing

## Future Enhancements

Ideas for later:

1. **Syntax highlighting**: Add ANSI color codes for terminal output
2. **Comment preservation**: Maintain and format comments
3. **Custom form styles**: Allow users to register custom formatting rules
4. **Diff-friendly output**: Stable ordering of keyword-value pairs
5. **Width-aware breaking**: More sophisticated line breaking based on actual character widths

## Notes

- The pretty printer should never change the meaning of the S-expression
- Alignment makes debugging much easier
- Consistent formatting aids in diffs and version control
- This is a quality-of-life tool, not performance-critical
- Focus on making output that humans love to read
