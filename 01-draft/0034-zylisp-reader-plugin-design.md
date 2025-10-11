---
number: 0034
title: "Zylisp Reader Macros and Plugin System Design"
author: Unknown
created: 2025-10-08
updated: 2025-10-08
state: Draft
supersedes: None
superseded-by: None
---

# Zylisp Reader Macros and Plugin System Design

**Status:** Exploration / Pre-Design
**Date:** October 2025
**Purpose:** Guide future development of Zylisp's extensibility mechanisms

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Background: Reader Macros in Lisp History](#background-reader-macros-in-lisp-history)
3. [Reader Macro Approaches for Zylisp](#reader-macro-approaches-for-zylisp)
4. [Tagged Literals: Recommended Approach](#tagged-literals-recommended-approach)
5. [Go Plugin System Architecture](#go-plugin-system-architecture)
6. [Plugin Isolation and Safety](#plugin-isolation-and-safety)
7. [Implementation Roadmap](#implementation-roadmap)
8. [Open Questions](#open-questions)

---

## Executive Summary

This document explores two complementary extensibility mechanisms for Zylisp:

1. **Tagged Literals** - A reader-level syntax extension system that balances power with implementation simplicity
2. **Go Plugin System** - A comprehensive plugin architecture enabling users to extend Zylisp at multiple compilation pipeline stages

### Key Decisions

- **Reader Macros**: Implement tagged literals (`#tag<content>`) rather than full Common Lisp-style reader macros
- **Plugin Strategy**: Support Go plugins at four levels: runtime primitives, macro expanders, reader extensions, and IR transformations
- **Safety Model**: Use tiered isolation (panic recovery → WASM → process isolation) based on trust level and performance requirements
- **Implementation Phases**: Start with runtime plugins and panic recovery, add WASM support for untrusted code, defer process isolation

### Why This Matters

These features would give Zylisp:

- **Reader extensibility** without architectural complexity
- **Native-speed user extensions** via Go plugins
- **True reader macros** (via plugins) - rare in modern Lisps
- **Safety guarantees** through isolation strategies
- **Rich ecosystem potential** through plugin registry

---

## Background: Reader Macros in Lisp History

### What Are Reader Macros?

Reader macros operate during the **read phase** of compilation, transforming text into S-expressions before any macro expansion or evaluation occurs. They allow extending the language's syntax at the most fundamental level.

### ZetaLisp and Lisp Machine Lisp

ZetaLisp (the dialect that ran on Lisp Machines) used the hash/sharpsign character (`#`) to introduce reader macros:

- `#'function` - equivalent to `(function ...)`
- `#\char` - character literals
- `#(...)` - vector/array literals
- `#"string"` - special string types
- `#o`, `#x`, `#b` - octal, hex, binary numbers

However, like LFE, **ZetaLisp did not support user-defined reader macros**. All reader macros were predefined by the system.

### Common Lisp's Evolution

Common Lisp (which evolved from ZetaLisp and other Lisp Machine dialects) added full support for user-defined reader macros through:

- **Readtables**: Objects that control how the reader parses text
- `set-macro-character`: Associate characters with reader functions
- `set-dispatch-macro-character`: Define `#X` dispatch macros

This enabled users to define arbitrary syntax extensions.

### Why ZetaLisp and LFE Couldn't Support User Reader Macros

Robert Virding's explanation for LFE applies equally to ZetaLisp:

> "The current tokeniser/parser split doesn't really support them in a general way to allow user defined ones."

**The fundamental problem**: Modern language implementations use a two-phase architecture:

1. **Scanner/Lexer**: Characters → Tokens
2. **Parser**: Tokens → AST/S-expressions

Reader macros need to operate at the **character level** while understanding **S-expression structure**. This creates an architectural mismatch when using a scanner/parser split.

Traditional Lisp readers avoid this by using a single-pass character stream → S-expressions architecture, with reader macros as dispatch characters that can recursively invoke the reader.

### Implications for Zylisp

Following the scanner/parser architecture (which Zylisp uses for clean compilation pipeline design), we have three options:

1. **Predefined reader macros only** (ZetaLisp/LFE approach) - Simple but not extensible
2. **Tagged literals with user-defined expanders** (hybrid approach) - Extensible without architectural complexity
3. **Full reader macros via plugins** (Go plugin approach) - Maximum power, more complexity

---

## Reader Macro Approaches for Zylisp

### Approach 1: Predefined Reader Macros Only

**What it means**: Hard-code specific reader macro behaviors into the lexer/parser.

#### Examples

```zylisp
#[1 2 3]        ; vector literal (already planned)
#{:a 1 :b 2}    ; set literal
#"raw string"   ; raw string (no escaping)
#b1010          ; binary number literal
#x1A2F          ; hex literal
#r"regex"       ; regex literal
```

#### Implementation

Add cases to the lexer for `#` followed by specific characters:

```go
func (l *Lexer) scanSharpDispatch() Token {
    ch := l.peek()

    switch ch {
    case '[':
        return l.scanVectorLiteral()
    case '{':
        return l.scanSetLiteral()
    case '"':
        return l.scanRawString()
    case 'b':
        return l.scanBinaryNumber()
    case 'x':
        return l.scanHexNumber()
    case 'r':
        return l.scanRegexLiteral()
    default:
        l.error("Unknown # dispatch character")
    }
}
```

#### Evaluation

**Advantages:**

- ✅ Clean implementation
- ✅ No security concerns
- ✅ Predictable behavior
- ✅ Easy to optimize

**Disadvantages:**

- ❌ Not extensible by users
- ❌ Language designers must anticipate all needs
- ❌ Can't experiment with syntax in user code

**Verdict**: Good baseline, but insufficient alone.

---

### Approach 2: Tagged Literals (Recommended)

**What it means**: The lexer recognizes a general pattern `#identifier<content>`, but user code provides the interpretation during macro expansion.

#### Syntax

```zylisp
#timestamp<2025-10-08T15:30:00Z>
#uuid<550e8400-e29b-41d4-a716-446655440000>
#duration<2h30m>
#color<#FF5733>
#json<{"key": "value"}>
#sql<SELECT * FROM users WHERE id = ?>
#regex</\d{3}-\d{3}-\d{4}$>
```

Multiple delimiter options:

```zylisp
#tag<content>   ; Angle brackets for simple content
#tag{content}   ; Braces to avoid escaping
#tag[content]   ; Brackets for patterns
#tag(content)   ; Parens for expressions
```

#### Architecture

**Phase 1: Lexer** recognizes the pattern and produces tokens

```go
type Token struct {
    Type     TokenType
    Tag      string    // The identifier after #
    Content  string    // Everything between delimiters
    Delim    rune      // Which delimiter: '<', '{', '[', '('
}

func (l *Lexer) scanTaggedLiteral() Token {
    tag := l.scanIdentifier()
    delim := l.peek()

    if !l.isDelimiter(delim) {
        l.error("Tagged literal must be followed by <, {, [, or (")
    }

    closingDelim := l.matchingDelimiter(delim)
    l.advance()

    content := l.scanUntilBalanced(closingDelim)

    return Token{
        Type:    TOKEN_TAGGED_LITERAL,
        Tag:     tag,
        Content: content,
        Delim:   delim,
    }
}
```

**Phase 2: Parser** converts to canonical S-expression

```go
func (p *Parser) parseTaggedLiteral(tok Token) sexpr.SExpr {
    // #timestamp<2025-10-08> becomes
    // (tagged-literal "timestamp" "2025-10-08")

    return sexpr.List{
        sexpr.Symbol("tagged-literal"),
        sexpr.String(tok.Tag),
        sexpr.String(tok.Content),
    }
}
```

**Phase 3: Expander** looks up and calls user-defined expanders

```go
type TagExpander func(content string) (sexpr.SExpr, error)

type Expander struct {
    tagExpanders map[string]TagExpander
}

func (e *Expander) expandTaggedLiteral(list sexpr.List) (sexpr.SExpr, error) {
    tag := string(list[1].(sexpr.String))
    content := string(list[2].(sexpr.String))

    if expander, exists := e.tagExpanders[tag]; exists {
        return expander(content)
    }

    // No expander registered - return as-is for later handling
    return list, nil
}
```

#### User Experience

**Level 1: Simple runtime function**

```zylisp
;; Define a function that parses the content
(deffunc parse-timestamp [s]
  (time:parse-iso8601 s))

;; Register it as a tagged literal expander
(register-tag-expander 'timestamp 'parse-timestamp)

;; Use it
(let ((t #timestamp<2025-10-08T15:30:00Z>))
  (time:format t "%Y-%m-%d"))
```

Expands to:

```zylisp
(let ((t (parse-timestamp "2025-10-08T15:30:00Z")))
  (time:format t "%Y-%m-%d"))
```

**Level 2: Custom expansion with validation**

```zylisp
;; Define a custom expander that validates at compile time
(deffunc expand-timestamp [content]
  ;; Parse at macro expansion time
  (let ((parsed (time:parse-iso8601 content)))
    ;; Return code that recreates this timestamp
    `(time:make-timestamp
       ,(time:year parsed)
       ,(time:month parsed)
       ,(time:day parsed)
       ,(time:hour parsed)
       ,(time:minute parsed)
       ,(time:second parsed))))

(register-tag-expander 'timestamp expand-timestamp)
```

Now `#timestamp<2025-10-08T15:30:00Z>` expands to:

```zylisp
(time:make-timestamp 2025 10 8 15 30 0)
```

**Level 3: Convenient macro for definition**

```zylisp
;; Macro for defining tag expanders
(defmacro define-tag-expander [tag [content-var] & body]
  `(register-tag-expander
     ',tag
     (fn [,content-var] ,@body)))

;; Usage
(define-tag-expander uuid [s]
  (uuid:parse s))

(define-tag-expander sql [query-string]
  ;; Parse SQL at expansion time, generate prepared statement
  (let ((parsed (sql:parse-query query-string)))
    `(sql:make-prepared-statement
       ,(sql:query-template parsed)
       ',(sql:query-params parsed))))
```

#### Advanced Features

**Nested parsing**: Tags can parse their content as Zylisp code

```zylisp
(define-tag-expander code [s]
  ;; Parse the content as Zylisp code
  (read-from-string s))

;; Usage:
#code<(+ 1 2 3)>  ; Expands to (+ 1 2 3)
```

**Compile-time validation**:

```zylisp
(define-tag-expander regex [pattern]
  ;; Validate regex at compile time
  (if (not (regex:valid? pattern))
      (error (string-append "Invalid regex: " pattern)))
  ;; Generate code to compile it at runtime
  `(regex:compile ,pattern))
```

If someone writes `#regex<[invalid(>`, they get a compile error, not a runtime error.

**Package-local tags**:

```zylisp
(package my-app)

;; Define a tag that's only available in this package
(define-tag-expander user-id [id-string]
  (parse-integer id-string))

;; In another package, #user-id<123> won't work
```

#### Complete Example: Database Connection Strings

```zylisp
(package my-db)

;; Define a tag for database connection strings
(define-tag-expander dbconn [spec]
  ;; Parse connection string at compile time
  (let ((parts (string:split spec ":")))
    (if (not (= (length parts) 4))
        (error "Invalid connection string format"))
    ;; Generate code to create connection at runtime
    `(db:make-connection
       :host ,(nth 0 parts)
       :port ,(parse-integer (nth 1 parts))
       :database ,(nth 2 parts)
       :username ,(nth 3 parts))))

;; Usage
(defvar *db* #dbconn<localhost:5432:mydb:admin>)

;; Expands to:
;; (defvar *db*
;;   (db:make-connection
;;     :host "localhost"
;;     :port 5432
;;     :database "mydb"
;;     :username "admin"))
```

#### Evaluation

**Advantages:**

- ✅ Extensible by users
- ✅ Clean scanner/parser architecture maintained
- ✅ Expansion happens during macro phase (not read phase)
- ✅ No security concerns (expansion is normal code)
- ✅ Easy to implement
- ✅ Package-scoped extensions possible

**Disadvantages:**

- ❌ Can't change tokenization behavior
- ❌ Limited to predefined delimiters
- ❌ Not as powerful as true reader macros

**Verdict**: Excellent balance of power and simplicity. Recommended as the primary approach.

---

### Approach 3: Full Reader Macros (Via Plugins)

Covered in detail in the [Go Plugin System](#go-plugin-system-architecture) section below.

**Summary**: Use Go plugins to allow users to extend the lexer itself, providing true reader macros like Common Lisp while maintaining Zylisp's architecture.

---

## Tagged Literals: Recommended Approach

### Implementation Requirements

#### What Zylisp Must Implement

**Core (Required):**

1. **Lexer modifications**
   - Recognize `#identifier<content>` pattern
   - Handle balanced delimiters in content
   - Support multiple delimiter types (`<>`, `{}`, `[]`, `()`)
   - Track delimiter type in token

2. **Parser modifications**
   - Convert tagged literals to `(tagged-literal "tag" "content")` forms

3. **Expander modifications**
   - Maintain registry of tag expanders
   - Look up and call registered expanders
   - Handle missing expanders gracefully

4. **Primitive functions**
   - `register-tag-expander` - add new tags
   - `unregister-tag-expander` - remove tags (for REPL)

5. **Built-in tags**
   - `#r<regex>` - regex literals
   - `#path<...>` - filesystem paths
   - Others as needed

**Nice to Have:**

- `define-tag-expander` macro for convenient definition
- `read-from-string` to allow tags to parse Zylisp code
- Package-local tag registration
- Better error messages (line numbers for invalid tag content)

**Future Enhancements:**

- Tag expanders that return multiple forms
- Context-aware expanders (see surrounding code)
- Reader-macro-like behavior (expander sees next form)

#### Implementation Checklist

```markdown
- [ ] Lexer: Recognize #identifier<content> pattern
- [ ] Lexer: Handle balanced delimiters
- [ ] Lexer: Support multiple delimiter types
- [ ] Parser: Convert to (tagged-literal ...) forms
- [ ] Expander: Tag registry implementation
- [ ] Expander: Lookup and call mechanism
- [ ] Primitive: register-tag-expander
- [ ] Primitive: unregister-tag-expander
- [ ] Built-in: #r<regex>
- [ ] Built-in: #path<...>
- [ ] Macro: define-tag-expander
- [ ] Function: read-from-string
- [ ] Feature: Package-local tags
- [ ] Tests: Basic tagged literal parsing
- [ ] Tests: Expander registration
- [ ] Tests: Error handling
- [ ] Docs: User guide for tag expanders
- [ ] Docs: API reference
```

### Integration with Compilation Pipeline

Tagged literals fit cleanly into Zylisp's planned pipeline:

```
Zylisp Source
    ↓
Parse (Lexer recognizes #tag<content>)
    ↓
Parse (Parser converts to (tagged-literal "tag" "content"))
    ↓
Expand (Expander looks up tag, calls user function)
    ↓
Expand (Result replaces tagged-literal form)
    ↓
Lower (Expanded form treated like any other code)
    ↓
Codegen
    ↓
Go Source
```

**No changes to Lower or Codegen phases required** - tagged literals are completely resolved during expansion.

### Performance Characteristics

**Lexer overhead**: Minimal - just pattern recognition and string scanning

**Parser overhead**: None - produces standard S-expressions

**Expansion overhead**: One map lookup + function call per tagged literal

**Runtime overhead**: Zero - tagged literals are expanded away

**Compilation caching**: Tagged literal expansion results can be cached like macro expansions

---

## Go Plugin System Architecture

### Overview

Allow users to write **custom language extensions in Go** that plug into various stages of the Zylisp compilation pipeline.

### Plugin Levels

#### Level 1: Runtime Primitives (Most Common)

Users write Go functions that become callable from Zylisp.

**User writes:**

```go
// userlib/strings.go
package userlib

import "strings"

func LevenshteinDistance(s1, s2 string) int {
    // Implementation...
    return distance
}

type Export struct {
    Name string
    Func interface{}
    Doc  string
}

var ZylispExports = []Export{
    {
        Name: "levenshtein-distance",
        Func: LevenshteinDistance,
        Doc:  "Calculate Levenshtein distance between two strings",
    },
}
```

**Zylisp implements:**

```go
type RuntimePlugin interface {
    Name() string
    Version() string
    Functions() []FunctionExport
}

type FunctionExport struct {
    ZylispName string
    GoFunc     interface{}
    Doc        string
}

func (pm *PluginManager) LoadRuntimePlugin(path string) error {
    plug, err := plugin.Open(path)
    if err != nil {
        return err
    }

    sym, err := plug.Lookup("ZylispPlugin")
    if err != nil {
        return err
    }

    runtimePlugin := sym.(RuntimePlugin)

    for _, export := range runtimePlugin.Functions() {
        pm.registerFunction(export)
    }

    return nil
}
```

**Usage:**

```zylisp
(use-plugin "userlib/strings.so")

(levenshtein-distance "kitten" "sitting")  ; => 3
```

**Benefits:**

- ✅ No change to compilation pipeline
- ✅ Native Go performance
- ✅ Easy to write and test
- ✅ Type-safe at Go level

**Tradeoffs:**

- ❌ Requires compiling Go code
- ❌ Platform-specific binaries

---

#### Level 2: Macro Expanders

Users write macro expanders in Go that run during the expansion phase.

**User writes:**

```go
package userlib

import "github.com/zylisp/zylisp-lang/sexpr"

func AwaitExpander(form sexpr.List, env *Env) (sexpr.SExpr, error) {
    // (await expr) expands to (channel:receive (go-async expr))

    if len(form) != 2 {
        return nil, fmt.Errorf("await requires exactly one argument")
    }

    return sexpr.List{
        sexpr.Symbol("channel:receive"),
        sexpr.List{
            sexpr.Symbol("go-async"),
            form[1],
        },
    }, nil
}

var ZylispMacros = []MacroExport{
    {
        Name:     "await",
        Expander: AwaitExpander,
        Doc:      "Await an asynchronous expression",
    },
}
```

**Usage:**

```zylisp
(use-plugin "userlib/macros.so")

(deffunc fetch-data []
  (let ((result (await (http:get "https://api.example.com"))))
    (process-result result)))
```

**Benefits:**

- ✅ Full control over macro expansion
- ✅ Native Go performance for complex expansion logic
- ✅ Can implement sophisticated DSLs

**Tradeoffs:**

- ❌ Must understand sexpr types
- ❌ More complex than Zylisp macros

---

#### Level 3: Reader Extensions (True Reader Macros!)

Users can extend the **lexer itself**.

**User writes:**

```go
package userlib

import (
    "github.com/zylisp/zylisp-lang/parser"
    "encoding/json"
)

type JSONLiteralReader struct{}

func (j *JSONLiteralReader) CanHandle(lexer *parser.Lexer) bool {
    // Check if we're at #json{
    return lexer.Peek() == '#' &&
           lexer.PeekN(1) == 'j' &&
           lexer.PeekN(5) == '{'
}

func (j *JSONLiteralReader) Read(lexer *parser.Lexer) (parser.Token, error) {
    lexer.ConsumeN(6)  // Consume #json{

    jsonContent := lexer.ScanBalanced('{', '}')

    // Parse JSON to Go structure
    var parsed interface{}
    err := json.Unmarshal([]byte(jsonContent), &parsed)
    if err != nil {
        return parser.Token{}, fmt.Errorf("invalid JSON: %w", err)
    }

    // Convert to Zylisp data structure at read time
    zylispData := convertJSONToSExpr(parsed)

    return parser.Token{
        Type:  parser.TOKEN_LITERAL,
        Value: zylispData,
    }, nil
}

var ZylispReaders = []ReaderExport{
    {
        Name:   "json",
        Reader: &JSONLiteralReader{},
        Doc:    "Embedded JSON literals: #json{...}",
    },
}
```

**Zylisp implements:**

```go
type ReaderMacro interface {
    CanHandle(lexer *Lexer) bool
    Read(lexer *Lexer) (Token, error)
}

type Lexer struct {
    input        []rune
    position     int
    readerMacros []ReaderMacro
}

func (l *Lexer) scanToken() Token {
    // Check if any plugin can handle this position
    for _, macro := range l.readerMacros {
        if macro.CanHandle(l) {
            return macro.Read(l)
        }
    }

    // Fall back to built-in scanning
    return l.scanBuiltinToken()
}
```

**Usage:**

```zylisp
(use-plugin "userlib/reader.so")

;; JSON is parsed at read-time!
(defvar *config* #json{
  "server": {
    "host": "localhost",
    "port": 8080
  }
})

(get-in *config* [:server :port])  ; => 8080
```

**Benefits:**

- ✅ TRUE reader macros like Common Lisp
- ✅ Can implement any syntax
- ✅ Validation at read time
- ✅ Native performance

**Tradeoffs:**

- ❌ Most complex to write
- ❌ Can break the reader if buggy
- ⚠️ Plugin load order matters

---

#### Level 4: IR Transformations

Users can add custom optimizations and analyses at the IR level.

**User writes:**

```go
package userlib

import "github.com/zylisp/zylisp-lang/ir"

type TailCallOptimizer struct{}

func (t *TailCallOptimizer) Name() string {
    return "tail-call-elimination"
}

func (t *TailCallOptimizer) Transform(node ir.Node) (ir.Node, error) {
    switch n := node.(type) {
    case *ir.FuncDecl:
        return t.optimizeFunction(n)
    default:
        return node, nil
    }
}

func (t *TailCallOptimizer) optimizeFunction(fn *ir.FuncDecl) (*ir.FuncDecl, error) {
    // Analyze function body for tail calls
    // Transform tail-recursive calls into loops
    return optimizedFn, nil
}

var ZylispTransforms = []TransformExport{
    {
        Name:      "tail-call-elimination",
        Transform: &TailCallOptimizer{},
        Phase:     "optimize",
    },
}
```

**Benefits:**

- ✅ Custom optimizations
- ✅ Static analysis passes
- ✅ Don't need to modify Zylisp core

**Tradeoffs:**

- ❌ Most complex plugin type
- ❌ Need deep understanding of IR

---

### Plugin Loading and Management

#### Project-Level Configuration

```zylisp
;; project.zl
(project
  :name "my-app"
  :version "0.1.0"

  :plugins [
    ;; Runtime plugins
    {:type :runtime
     :name "strings"
     :path "plugins/strings.so"}

    ;; Macro plugins
    {:type :macro
     :name "async"
     :path "plugins/async.so"}

    ;; Reader plugins
    {:type :reader
     :name "json-reader"
     :path "plugins/json.so"
     :priority 10}

    ;; Transform plugins
    {:type :transform
     :name "tail-call-opt"
     :path "plugins/optimizer.so"
     :phase :optimize
     :enabled-by-default false}
  ])
```

#### Dynamic Loading in REPL

```zylisp
;; Load plugins interactively
zylisp> (use-plugin :runtime "plugins/mylib.so")
Loaded runtime plugin: mylib (5 functions exported)

zylisp> (use-plugin :macro "plugins/async.so")
Loaded macro plugin: async (await, async, go-async)

;; List loaded plugins
zylisp> (list-plugins)
Runtime plugins:
  - mylib (v1.0.0) - Custom string functions
Macro plugins:
  - async (v1.0.0) - Async/await syntax
```

#### Plugin Discovery and Registry

```bash
# Search for plugins
$ zylisp plugin search json
Found 3 plugins:
  - json-reader (v1.2.0) - JSON literal syntax
  - json-schema (v0.5.0) - JSON schema validation

# Install plugin
$ zylisp plugin install json-reader

# Update plugins
$ zylisp plugin update
```

### Plugin Interface Stability

#### Versioned API

```go
const PluginAPIVersion = "1.0"

type Plugin interface {
    APIVersion() string
    Name() string
    Version() string
}

type RuntimePlugin interface {
    Plugin
    Functions() []FunctionExport
}

type MacroPlugin interface {
    Plugin
    Macros() []MacroExport
}
```

#### Compatibility Checking

```go
func (pm *PluginManager) LoadPlugin(path string) error {
    plug, err := plugin.Open(path)
    if err != nil {
        return err
    }

    sym, err := plug.Lookup("Plugin")
    if err != nil {
        return fmt.Errorf("not a valid Zylisp plugin")
    }

    basePlugin := sym.(Plugin)

    if !pm.isCompatible(basePlugin.APIVersion()) {
        return fmt.Errorf("plugin API version %s incompatible with %s",
            basePlugin.APIVersion(), PluginAPIVersion)
    }

    // Proceed with loading...
}
```

### Build Tooling

#### Plugin Development Kit

```bash
# Create new plugin project
$ zylisp plugin new my-plugin --type runtime

# Build plugin
$ zylisp plugin build

# Test plugin
$ zylisp plugin test

# Publish to plugin registry
$ zylisp plugin publish
```

#### Plugin Template

```go
// Generated by: zylisp plugin new my-plugin --type runtime
package main

import "github.com/zylisp/zylisp-lang/plugin"

const (
    PluginName    = "my-plugin"
    PluginVersion = "0.1.0"
)

type MyPlugin struct{}

func (p *MyPlugin) APIVersion() string { return plugin.APIVersion }
func (p *MyPlugin) Name() string       { return PluginName }
func (p *MyPlugin) Version() string    { return PluginVersion }

func (p *MyPlugin) Functions() []plugin.FunctionExport {
    return []plugin.FunctionExport{
        {
            ZylispName: "my-function",
            GoFunc:     MyFunction,
            Doc:        "My awesome function",
        },
    }
}

func MyFunction(arg string) string {
    return "Hello, " + arg
}

var Plugin MyPlugin
```

### Integration with Compilation Pipeline

Plugins integrate at phase boundaries without breaking the pipeline:

```
User code
    ↓
Reader plugins extend lexer
    ↓
Parse (with plugin-extended reader)
    ↓
Macro plugins extend expander
    ↓
Expand (with plugin macros)
    ↓
Transform plugins optimize IR
    ↓
Lower (with plugin transforms)
    ↓
Codegen
    ↓
Go source
    ↓
Compile with runtime plugins linked
    ↓
Binary
```

**Key points:**

- ✅ Parse → Expand → Lower → Codegen pipeline intact
- ✅ Each phase still does one job
- ✅ Can still compile incrementally
- ✅ No circular dependencies
- ⊕ Extension points at phase boundaries
- ⊕ Projects without plugins have zero overhead

---

## Plugin Isolation and Safety

### The Problem Space

#### What We're Protecting Against

**Category 1: Bugs (Recoverable)**

- Panics from nil pointer dereferences
- Index out of bounds errors
- Type assertions that fail
- Stack overflows

**Category 2: Resource Abuse (Somewhat Containable)**

- Infinite loops
- Memory leaks
- Goroutine leaks
- File descriptor exhaustion
- CPU hogging

**Category 3: Malice (Uncontainable in-process)**

- Deliberate memory corruption
- Accessing private data structures
- Calling unsafe code
- Breaking memory safety

#### Go's Limitations

Go plugins run in the same address space with no memory isolation:

- Can access any exported symbol
- Can corrupt shared memory
- Can crash the entire process
- Cannot be unloaded

Go does **not** provide:

- Memory isolation (like Erlang processes)
- Capability-based security
- True sandboxing
- Process-level fault isolation

---

### Strategy 1: In-Process Supervision (Limited Protection)

Protects against **Category 1 (bugs)** but not Categories 2 or 3.

#### Panic Recovery

```go
type PluginSupervisor struct {
    plugin      *Plugin
    restarts    int
    maxRestarts int
    window      time.Duration
    startTimes  []time.Time
}

func (ps *PluginSupervisor) CallWithRecovery(fn func() error) (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("plugin panicked: %v\n%s", r, debug.Stack())
            ps.logPanic(r)

            if ps.shouldRestart() {
                ps.restart()
            }
        }
    }()

    return fn()
}
```

#### Supervised Plugin Calls

```go
type SupervisedPlugin struct {
    supervisor *PluginSupervisor
    plugin     RuntimePlugin
}

func (sp *SupervisedPlugin) CallFunction(name string, args []interface{}) (interface{}, error) {
    var result interface{}
    var callErr error

    err := sp.supervisor.CallWithRecovery(func() error {
        fn := sp.plugin.GetFunction(name)
        result, callErr = fn(args...)
        return callErr
    })

    if err != nil {
        return nil, err  // Plugin panicked
    }

    return result, callErr
}
```

#### Resource Monitoring

```go
type ResourceMonitor struct {
    initialGoroutines int
    initialMemory     uint64
}

func (rm *ResourceMonitor) Check() error {
    current := runtime.NumGoroutine()

    if current > rm.initialGoroutines * 2 {
        return fmt.Errorf("goroutine leak detected")
    }

    var m runtime.MemStats
    runtime.ReadMemStats(&m)

    if m.Alloc > rm.initialMemory * 2 {
        return fmt.Errorf("memory leak detected")
    }

    return nil
}
```

**Benefits:**

- ✅ Recovers from panics
- ✅ Can restart plugins after crashes
- ✅ Limits restart frequency
- ✅ Logs failures for debugging

**Limitations:**

- ❌ Can't prevent memory corruption
- ❌ Can't stop infinite loops
- ❌ Can't contain malicious code
- ❌ Shared state may be corrupted after panic
- ❌ Can detect but not fix resource leaks

---

### Strategy 2: Out-of-Process Plugins (Real Isolation)

Run plugins in separate processes with IPC communication.

#### Architecture

```
Zylisp Compiler
    ↓ IPC
Plugin Process 1 (monitored by supervisor)
    ↓ IPC
Plugin Process 2 (monitored by supervisor)
```

#### Implementation

```go
type ProcessPlugin struct {
    path    string
    cmd     *exec.Cmd
    stdin   io.WriteCloser
    stdout  io.ReadCloser
    encoder *json.Encoder
    decoder *json.Decoder
}

func (pp *ProcessPlugin) CallFunction(name string, args []interface{}) (interface{}, error) {
    // Send request
    req := PluginRequest{
        Type:     "call",
        Function: name,
        Args:     args,
    }

    if err := pp.encoder.Encode(req); err != nil {
        return nil, err
    }

    // Receive response
    var resp PluginResponse
    if err := pp.decoder.Decode(&resp); err != nil {
        return nil, err
    }

    if resp.Error != "" {
        return nil, errors.New(resp.Error)
    }

    return resp.Result, nil
}
```

#### Plugin Process (User Code)

```go
// User's plugin as a standalone process
package main

import (
    "github.com/zylisp/plugin-sdk/process"
)

func LevenshteinDistance(s1, s2 string) int {
    // Implementation...
    return distance
}

func main() {
    plugin := process.NewPlugin()
    plugin.Register("levenshtein-distance", LevenshteinDistance)
    plugin.Serve(os.Stdin, os.Stdout)
}
```

#### Process Supervision

```go
type ProcessSupervisor struct {
    plugin      *ProcessPlugin
    restartSpec RestartSpec
}

func (ps *ProcessSupervisor) Monitor() {
    for {
        err := ps.plugin.cmd.Wait()

        if err != nil {
            log.Printf("Plugin process crashed: %v", err)

            if ps.shouldRestart() {
                ps.restart()
            } else {
                return
            }
        }
    }
}

func (ps *ProcessSupervisor) SetResourceLimits() error {
    // On Unix systems, use setrlimit
    return syscall.Setrlimit(syscall.RLIMIT_AS, &syscall.Rlimit{
        Cur: 512 * 1024 * 1024, // 512MB
        Max: 512 * 1024 * 1024,
    })
}
```

**Benefits:**

- ✅ True memory isolation
- ✅ Can kill runaway processes
- ✅ Can set resource limits (CPU, memory)
- ✅ Crash doesn't affect compiler
- ✅ Can enforce timeouts
- ✅ Can restart cleanly

**Limitations:**

- ❌ Significant IPC overhead (~500x slower)
- ❌ Can't share memory (must serialize)
- ❌ More complex to implement
- ❌ Process creation overhead

**Performance:**

- In-process call: ~100ns
- Out-of-process call: ~50µs (500x slower)

---

### Strategy 3: WASM Plugins (Best of Both Worlds)

WebAssembly provides **sandboxing** with near-native performance.

#### Implementation

```go
import (
    "github.com/tetratelabs/wazero"
    "github.com/tetratelabs/wazero/api"
)

type WASMPlugin struct {
    runtime wazero.Runtime
    module  api.Module
}

func (wp *WASMPlugin) Load(wasmBytes []byte) error {
    ctx := context.Background()
    wp.runtime = wazero.NewRuntime(ctx)

    var err error
    wp.module, err = wp.runtime.InstantiateWithConfig(
        ctx,
        wasmBytes,
        wazero.NewModuleConfig(),
    )

    return err
}

func (wp *WASMPlugin) CallFunction(name string, args ...uint64) ([]uint64, error) {
    fn := wp.module.ExportedFunction(name)
    if fn == nil {
        return nil, fmt.Errorf("function %s not exported", name)
    }

    ctx := context.Background()
    return fn.Call(ctx, args...)
}
```

#### Memory Limits

```go
func (wp *WASMPlugin) LoadWithLimits(wasmBytes []byte, maxMemory uint32) error {
    ctx := context.Background()

    wp.runtime = wazero.NewRuntimeWithConfig(
        ctx,
        wazero.NewRuntimeConfig().
            WithMemoryLimitPages(maxMemory), // Pages = 64KB each
    )

    return wp.Load(wasmBytes)
}
```

#### User Plugin (TinyGo)

```go
// User writes plugin in Go, compiles to WASM with TinyGo
package main

//export levenshtein_distance
func levenshtein_distance(s1ptr, s1len, s2ptr, s2len uint32) uint32 {
    s1 := readString(s1ptr, s1len)
    s2 := readString(s2ptr, s2len)

    // Implementation...
    return uint32(distance)
}

func main() {}
```

Build:

```bash
tinygo build -o plugin.wasm -target=wasi plugin.go
```

#### WASM with Timeout

```go
func (wp *WASMPlugin) CallWithTimeout(
    name string,
    timeout time.Duration,
    args ...uint64,
) ([]uint64, error) {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    fn := wp.module.ExportedFunction(name)
    results, err := fn.Call(ctx, args...)

    if ctx.Err() == context.DeadlineExceeded {
        // WASM execution stops immediately!
        return nil, fmt.Errorf("plugin timed out")
    }

    return results, err
}
```

**Benefits:**

- ✅ True sandboxing (memory isolation)
- ✅ Can enforce resource limits
- ✅ Fast (~3-5x slower than native, not 500x)
- ✅ Can kill cleanly
- ✅ Portable (not platform-specific)
- ✅ Can't corrupt host memory
- ✅ Execution can be truly interrupted

**Limitations:**

- ❌ More complex FFI (must marshal data)
- ❌ TinyGo has limitations (no full reflect)
- ❌ Can't use arbitrary Go libraries
- ❌ Still slower than native Go

---

### Strategy 4: Hybrid Approach (Recommended)

Use different isolation strategies based on plugin type and trust level.

#### Isolation Levels

```go
type IsolationLevel int

const (
    IsolationNone     IsolationLevel = iota  // In-process, no protection
    IsolationRecover                          // In-process, panic recovery
    IsolationProcess                          // Out-of-process
    IsolationWASM                             // WASM sandbox
)
```

#### Decision Matrix

| Plugin Type | Trust | Default Isolation | Rationale |
|-------------|-------|-------------------|-----------|
| Runtime | Trusted | Recover | Fast, panic recovery sufficient |
| Runtime | Untrusted | WASM | Safety without huge perf hit |
| Macro | Trusted | Recover | Compile-time, panics acceptable |
| Macro | Untrusted | Process | Can afford IPC overhead at compile time |
| Reader | Trusted | Recover | Critical path, need speed |
| Reader | Untrusted | WASM | Safety with acceptable perf |
| Transform | Any | Recover | Compile-time, can use IPC if needed |

#### Configuration

```zylisp
(project
  :plugins [
    ;; Trusted first-party - fast path
    {:name "mylib"
     :path "plugins/mylib.so"
     :type :runtime
     :trusted true
     :isolation :recover}

    ;; Third-party - safer
    {:name "community-lib"
     :path "plugins/community.wasm"
     :type :runtime
     :trusted false
     :isolation :wasm}

    ;; Experimental - maximum safety
    {:name "experimental-macro"
     :path "plugins/macro"
     :type :macro
     :trusted false
     :isolation :process}
  ])
```

### Security Considerations

#### Plugin Signing

```go
type SignedPlugin struct {
    Path      string
    Signature []byte
    PublicKey []byte
}

func (pm *PluginManager) LoadSignedPlugin(sp SignedPlugin) error {
    if !crypto.Verify(sp.Path, sp.Signature, sp.PublicKey) {
        return fmt.Errorf("invalid plugin signature")
    }

    return pm.LoadPlugin(sp.Path)
}
```

#### Permission Model

```zylisp
(use-plugin :runtime "plugins/networking.so"
  :permissions [:network :filesystem])
```

### Practical Recommendations

#### For Runtime Plugins

**Trusted, high-performance**:

```zylisp
(use-plugin :runtime "internal/strings.so"
  :isolation :recover)
```

**Third-party, need safety**:

```zylisp
(use-plugin :runtime "community/imageproc.wasm"
  :isolation :wasm
  :max-memory 100M
  :timeout 5s)
```

#### For Compile-Time Plugins

**Default**:

```zylisp
(use-plugin :macro "internal/async.so"
  :isolation :recover)
```

**Untrusted**:

```zylisp
(use-plugin :macro "experimental/dsl.exe"
  :isolation :process
  :timeout 30s)
```

---

## Implementation Roadmap

### Phase 1: Tagged Literals (Foundation)

**Goal**: Implement basic tagged literal support

**Tasks**:

1. Lexer modifications
   - Recognize `#identifier<content>` pattern
   - Handle balanced delimiters
   - Support multiple delimiter types
2. Parser modifications
   - Convert to `(tagged-literal ...)` forms
3. Expander modifications
   - Tag registry implementation
   - Lookup and call mechanism
4. Primitive functions
   - `register-tag-expander`
   - `unregister-tag-expander`
5. Built-in tags
   - `#r<regex>`
   - `#path<...>`
6. Documentation
   - User guide
   - API reference
7. Tests
   - Basic parsing
   - Expander registration
   - Error handling

**Duration**: 2-3 weeks

**Success Criteria**:

- Users can define and use custom tagged literals
- Built-in tags work correctly
- Good error messages
- Comprehensive test coverage

---

### Phase 2: Runtime Plugins (MVP)

**Goal**: Enable users to write Go functions callable from Zylisp

**Tasks**:

1. Plugin interface definition
   - `RuntimePlugin` interface
   - `FunctionExport` structure
   - API version mechanism
2. Plugin manager
   - Load plugins
   - Register functions
   - Compatibility checking
3. Function calling
   - Type conversion (Go ↔ Zylisp)
   - Error handling
4. REPL integration
   - `use-plugin` command
   - `list-plugins` command
5. Panic recovery (supervision)
   - `PluginSupervisor` implementation
   - Restart logic
   - Resource monitoring
6. Plugin SDK
   - Template generation
   - Helper functions
   - Documentation
7. Build tooling
   - `zylisp plugin new`
   - `zylisp plugin build`
   - `zylisp plugin test`

**Duration**: 3-4 weeks

**Success Criteria**:

- Users can create and load runtime plugins
- Plugin crashes don't crash compiler
- Good developer experience (SDK, templates)
- Performance acceptable (near-native)

---

### Phase 3: Macro Plugins

**Goal**: Enable Go-based macro expanders

**Tasks**:

1. Macro plugin interface
   - `MacroPlugin` interface
   - `MacroExpander` function type
2. Expander integration
   - Register macro plugins
   - Call plugin expanders
3. S-expression API
   - Expose sexpr types to plugins
   - Construction/pattern matching helpers
4. Examples
   - Async/await macro
   - DSL macro
5. Documentation
   - Macro writing guide
   - API reference

**Duration**: 2-3 weeks

**Success Criteria**:

- Users can write Go-based macros
- Performance acceptable (compile-time only)
- Easy to use API

---

### Phase 4: WASM Plugin Support

**Goal**: Safe execution of untrusted plugins

**Tasks**:

1. WASM runtime integration
   - Integrate wazero
   - Load WASM modules
2. FFI layer
   - Data marshaling
   - String handling
   - Error propagation
3. Resource limits
   - Memory limits
   - CPU time limits
   - Timeout handling
4. TinyGo support
   - Templates
   - Build scripts
   - Examples
5. Plugin registry changes
   - Support WASM plugins
   - Auto-detect format
6. Documentation
   - TinyGo plugin guide
   - Limitations and workarounds

**Duration**: 3-4 weeks

**Success Criteria**:

- WASM plugins work correctly
- Sandboxing effective
- Performance acceptable (3-5x slower than native)
- Good TinyGo experience

---

### Phase 5: Reader Plugins (Optional)

**Goal**: True reader macros via plugins

**Tasks**:

1. Reader plugin interface
   - `ReaderMacro` interface
   - Lexer access API
2. Lexer modifications
   - Plugin hook points
   - Safe state management
3. Priority/ordering
   - Plugin priority system
   - Conflict detection
4. Examples
   - JSON reader
   - Custom syntax reader
5. Documentation
   - Reader plugin guide
   - Safety considerations

**Duration**: 3-4 weeks

**Success Criteria**:

- Users can extend the reader
- No performance degradation when not used
- Safe (plugins can't break lexer)

**Decision Point**: Only proceed if evidence shows users need this. Tagged literals may be sufficient.

---

### Phase 6: Transform Plugins (Advanced)

**Goal**: Custom IR transformations

**Tasks**:

1. Transform plugin interface
   - `Transform` interface
   - IR access API
2. Pipeline integration
   - Register transforms
   - Phase ordering
3. Examples
   - Tail call optimization
   - Dead code elimination
4. Documentation
   - Transform writing guide
   - IR reference

**Duration**: 4-5 weeks

**Success Criteria**:

- Users can add optimizations
- No performance regression
- Safe (transforms can't break IR)

**Decision Point**: Only proceed if language matures and users have specific needs.

---

### Phase 7: Plugin Ecosystem

**Goal**: Complete plugin system

**Tasks**:

1. Plugin registry
   - Web service
   - Search API
   - Publishing
2. Package manager integration
   - `zylisp plugin install`
   - Dependency resolution
   - Version management
3. Plugin signing
   - Key generation
   - Signature verification
   - Trust model
4. Discovery and search
   - CLI commands
   - Web interface
5. Documentation site
   - Plugin catalog
   - Tutorials
   - Best practices

**Duration**: 6-8 weeks

**Success Criteria**:

- Easy to find plugins
- Safe to install plugins
- Active ecosystem

---

## Open Questions

### Technical Questions

1. **Go Plugin Limitations**
   - Q: Can we work around Go's inability to unload plugins?
   - Options: Process isolation, WASM, accept limitation
   - Decision: Start with accepting limitation, add isolation later if needed

2. **WASM Performance**
   - Q: Is 3-5x slowdown acceptable for untrusted plugins?
   - Research: Benchmark realistic workloads
   - Decision: TBD based on benchmarks

3. **Tagged Literal Delimiter Semantics**
   - Q: Should different delimiters have different semantics?
   - Options: All equivalent vs. type hints (e.g., `<>` for strings, `[]` for lists)
   - Decision: Start with all equivalent, add semantics later if useful

4. **Reader Plugin Ordering**
   - Q: How to resolve conflicts when multiple plugins want to handle the same syntax?
   - Options: Priority numbers, first-registered-wins, error on conflict
   - Decision: Priority numbers (like CSS z-index)

5. **Macro Plugin Caching**
   - Q: Can we cache macro expansion results to improve compilation speed?
   - Consideration: Cache invalidation when plugin updates
   - Decision: TBD after measuring compilation times

### Design Questions

6. **API Stability**
   - Q: How to evolve plugin APIs without breaking existing plugins?
   - Options: Semantic versioning, deprecation cycles, multiple versions
   - Decision: Semantic versioning with 1-year deprecation cycle

7. **Trust Model**
   - Q: How to indicate trusted vs. untrusted plugins?
   - Options: Explicit in config, signature-based, sandbox-all-by-default
   - Decision: Explicit in config, with warnings for unsigned plugins

8. **Package Management**
   - Q: Should plugins use Go modules or separate package manager?
   - Options: Go modules, custom registry, both
   - Decision: Go modules for development, registry for discovery

9. **Error Handling**
   - Q: How to handle plugin errors gracefully?
   - Considerations: Compilation errors vs. runtime errors
   - Decision: Compilation errors are fatal, runtime errors are recoverable

10. **Documentation**
    - Q: Where should plugin documentation live?
    - Options: In plugin package, central registry, both
    - Decision: Both - README in package, searchable in registry

### Ecosystem Questions

11. **Plugin Licensing**
    - Q: How to handle plugins with incompatible licenses?
    - Consideration: LGPL, GPL, proprietary plugins
    - Decision: Allow any license, display clearly to user

12. **Plugin Discovery**
    - Q: How do users find plugins?
    - Options: Central registry, GitHub topics, search engine
    - Decision: Central registry with GitHub integration

13. **Plugin Quality**
    - Q: How to ensure plugin quality?
    - Options: Curation, user ratings, automated testing
    - Decision: User ratings + security scanning, no curation

14. **Breaking Changes**
    - Q: What to do when Zylisp API changes break plugins?
    - Options: Version pinning, compatibility layers, deprecation
    - Decision: Deprecation with migration guide and compatibility shims

15. **Community Plugins**
    - Q: How to encourage plugin development?
    - Ideas: Showcase, tutorials, bounties, featured plugins
    - Decision: All of the above - start with showcase and tutorials

---

## Conclusion

This document outlines two powerful extensibility mechanisms for Zylisp:

1. **Tagged Literals**: A pragmatic approach to reader-level syntax extension that maintains Zylisp's clean architecture while providing significant user power. This should be implemented in Phase 1.

2. **Go Plugin System**: A comprehensive plugin architecture enabling native-speed extensions at multiple compilation stages. This should be implemented incrementally, starting with runtime plugins in Phase 2.

Together, these features would give Zylisp:

- **Unprecedented extensibility** in a modern Lisp
- **Native Go performance** for extensions
- **True reader macros** (via plugins)
- **Safety guarantees** through tiered isolation
- **Rich ecosystem potential** through plugin registry

The key to success is:

- **Start simple**: Tagged literals first, then runtime plugins
- **Add safety incrementally**: Panic recovery → WASM → process isolation
- **Let users drive features**: Only add advanced plugin types if evidence shows need
- **Maintain architecture**: Plugins extend at phase boundaries, don't break pipeline
- **Build ecosystem**: Good tools, documentation, and discovery are essential

This approach balances power with pragmatism, giving users extraordinary extensibility while keeping the implementation tractable and the architecture clean.

---

## Next Steps

1. **Review this document** with the Zylisp team
2. **Create formal design documents** for:
   - Tagged literals implementation (0028)
   - Plugin system architecture (0029)
   - Plugin isolation strategies (0030)
3. **Prototype tagged literals** to validate approach
4. **Benchmark Go plugin overhead** to inform isolation strategy
5. **Create plugin SDK design** for user experience
6. **Plan Phase 1 implementation** in detail

---

*This document synthesizes exploration of reader macros and plugin systems for Zylisp. It is not a formal design document but rather a foundation for future detailed designs and implementation plans.*
