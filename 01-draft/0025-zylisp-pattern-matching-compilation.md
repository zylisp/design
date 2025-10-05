---
number: 0025
title: "Zylisp Pattern Matching: Compilation to Go AST"
author: Duncan McGreggor
created: 2025-10-04
updated: 2025-10-04
state: Draft
supersedes: None
superseded-by: None
---

# Zylisp Pattern Matching: Compilation to Go AST

## Overview

This document describes how Zylisp's pattern matching features compile down through the compilation pipeline:

```
Zylisp Syntax → Desugared Zylisp → Zylisp S-expr IR (Go AST) → Go Code
```

Pattern matching is a **Zylisp-level feature** that desugars to simpler Zylisp forms before being compiled to the canonical S-expression IR format used by `zast`.

---

## Compilation Pipeline

### Stage 1: User-Written Zylisp

The syntax users write, featuring rich pattern matching:

```lisp
(deffunc greet [{:keys [name age email]}]
  (:args UserMap)
  (:return string)
  (str "Hello " name))
```

### Stage 2: Desugared Zylisp

After macro expansion and pattern desugaring, but still in Zylisp syntax:

```lisp
(deffunc greet [arg$1]
  (:args UserMap)
  (:return string)
  (let [name  (map-get-or-error arg$1 :name)
        age   (map-get-or-error arg$1 :age)
        email (map-get-or-error arg$1 :email)]
    (str "Hello " name)))
```

Key transformations:
- Pattern `{:keys [name age email]}` → simple parameter `arg$1`
- Destructuring → explicit `let` bindings with runtime checks
- Field extraction calls added

### Stage 3: Zylisp S-expression IR (Go AST)

The canonical S-expression format that maps 1:1 to Go's AST nodes:

```lisp
(FuncDecl
  :name (Ident :name "greet")
  :type (FuncType
    :params (FieldList
      :list [(Field
               :names [(Ident :name "arg$1")]
               :type (Ident :name "UserMap"))])
    :results (FieldList
      :list [(Field :type (Ident :name "string"))]))
  :body (BlockStmt
    :list [
      ; Variable declaration for name
      (DeclStmt
        :decl (GenDecl
          :tok VAR
          :specs [(ValueSpec
                    :names [(Ident :name "name")]
                    :values [(CallExpr
                               :fun (SelectorExpr
                                      :x (Ident :name "zrt")
                                      :sel (Ident :name "MapGetOrError"))
                               :args [(Ident :name "arg$1")
                                      (BasicLit :kind STRING :value "\"name\"")])])]))
      
      ; Variable declaration for age
      (DeclStmt
        :decl (GenDecl
          :tok VAR
          :specs [(ValueSpec
                    :names [(Ident :name "age")]
                    :values [(CallExpr
                               :fun (SelectorExpr
                                      :x (Ident :name "zrt")
                                      :sel (Ident :name "MapGetOrError"))
                               :args [(Ident :name "arg$1")
                                      (BasicLit :kind STRING :value "\"age\"")])])]))
      
      ; Variable declaration for email
      (DeclStmt
        :decl (GenDecl
          :tok VAR
          :specs [(ValueSpec
                    :names [(Ident :name "email")]
                    :values [(CallExpr
                               :fun (SelectorExpr
                                      :x (Ident :name "zrt")
                                      :sel (Ident :name "MapGetOrError"))
                               :args [(Ident :name "arg$1")
                                      (BasicLit :kind STRING :value "\"email\"")])])]))
      
      ; Return statement
      (ReturnStmt
        :results [(CallExpr
                    :fun (SelectorExpr
                           :x (Ident :name "zrt")
                           :sel (Ident :name "Str"))
                    :args [(BasicLit :kind STRING :value "\"Hello \"")
                           (Ident :name "name")])])]))
```

### Stage 4: Go Code

The final generated Go code:

```go
import zrt "github.com/zylisp/runtime"

func greet(arg$1 UserMap) string {
    name := zrt.MapGetOrError(arg$1, "name")
    age := zrt.MapGetOrError(arg$1, "age")
    email := zrt.MapGetOrError(arg$1, "email")
    return zrt.Str("Hello ", name)
}
```

---

## Key Desugaring Patterns

### 1. Let Bindings → Sequential Variable Declarations

Zylisp's `let` form creates a new lexical scope with bindings. In Go, this becomes sequential variable declarations:

**Zylisp:**
```lisp
(let [x 10
      y 20
      z (+ x y)]
  (* z 2))
```

**Desugared to Go AST:**
```lisp
(BlockStmt
  :list [
    (DeclStmt
      :decl (GenDecl
        :tok VAR
        :specs [(ValueSpec
                  :names [(Ident :name "x")]
                  :values [(BasicLit :kind INT :value "10")])]))
    (DeclStmt
      :decl (GenDecl
        :tok VAR
        :specs [(ValueSpec
                  :names [(Ident :name "y")]
                  :values [(BasicLit :kind INT :value "20")])]))
    (DeclStmt
      :decl (GenDecl
        :tok VAR
        :specs [(ValueSpec
                  :names [(Ident :name "z")]
                  :values [(BinaryExpr
                             :x (Ident :name "x")
                             :op ADD
                             :y (Ident :name "y"))])]))
    (ExprStmt
      :x (BinaryExpr
           :x (Ident :name "z")
           :op MUL
           :y (BasicLit :kind INT :value "2")))])
```

**Go code:**
```go
{
    x := 10
    y := 20
    z := x + y
    z * 2
}
```

### 2. Implicit Returns → Explicit ReturnStmt

Zylisp treats the last expression in a function body as the return value. Go requires explicit `return` statements:

**Zylisp:**
```lisp
(deffunc add [a b]
  (:args int int)
  (:return int)
  (+ a b))
```

**Go AST:**
```lisp
(FuncDecl
  :body (BlockStmt
    :list [(ReturnStmt
             :results [(BinaryExpr
                         :x (Ident :name "a")
                         :op ADD
                         :y (Ident :name "b"))])]))
```

### 3. Pattern Matching → Conditional Dispatch

Multi-clause pattern matching becomes conditional logic:

**Zylisp:**
```lisp
(deffunc handle
  ([{:status 200 :keys [body]}]
   (:args Response)
   (:return Result)
   (success body))
  
  ([{:status 404}]
   (:args Response)
   (:return Result)
   (not-found))
  
  ([{:status status :keys [error]}]
   (:args Response)
   (:return Result)
   (handle-error status error)))
```

**Desugared Zylisp:**
```lisp
(deffunc handle [arg$1]
  (:args Response)
  (:return Result)
  (let [status$ (. arg$1 status)]
    (cond
      (and (= status$ 200)
           (has-field? arg$1 :body))
      (let [body (. arg$1 body)]
        (success body))
      
      (= status$ 404)
      (not-found)
      
      (and (has-field? arg$1 :status)
           (has-field? arg$1 :error))
      (let [status (. arg$1 status)
            error (. arg$1 error)]
        (handle-error status error))
      
      :else
      (throw-match-error "handle" arg$1))))
```

**Go AST (simplified):**
```lisp
(FuncDecl
  :name (Ident :name "handle")
  :type (FuncType
    :params (FieldList :list [(Field :names [(Ident :name "arg$1")] :type (Ident :name "Response"))])
    :results (FieldList :list [(Field :type (Ident :name "Result"))]))
  :body (BlockStmt
    :list [
      ; Extract status once
      (DeclStmt ...)
      
      ; Switch or if-else chain
      (IfStmt
        :cond (BinaryExpr
                :x (BinaryExpr
                     :x (Ident :name "status$")
                     :op EQL
                     :y (BasicLit :kind INT :value "200"))
                :op LAND
                :y (CallExpr :fun (Ident :name "hasField") ...))
        :body (BlockStmt
                :list [(DeclStmt ...) ; extract body
                       (ReturnStmt ...)])
        :else (IfStmt
                :cond (BinaryExpr ...)
                :body (BlockStmt ...)
                :else ...))]))
```

---

## Runtime Support

Pattern matching relies on runtime support functions from the `zylisp/runtime` package (imported as `zrt`):

### Map Operations
```go
// Extract value from map, panic if key missing
func MapGetOrError(m map[string]interface{}, key string) interface{}

// Extract value from map with default
func MapGetOrDefault(m map[string]interface{}, key string, def interface{}) interface{}

// Check if map has key
func MapHasKey(m map[string]interface{}, key string) bool
```

### Type Checking
```go
func IsMap(v interface{}) bool
func IsList(v interface{}) bool
func IsStruct(v interface{}) bool
```

### Error Handling
```go
func ThrowMatchError(funcName string, value interface{})
func ThrowArityError(funcName string, arity int)
```

### Utility Functions
```go
// String concatenation
func Str(parts ...interface{}) string

// List operations
func Length(lst interface{}) int
func Nth(lst interface{}, n int) interface{}
func First(lst interface{}) interface{}
func Rest(lst interface{}) interface{}
```

---

## Type-Aware Desugaring

Pattern matching desugaring is **type-aware**. The `:args` type annotation determines how patterns are desugared.

### Struct Types → Direct Field Access

**Zylisp:**
```lisp
(deftype User
  {:name string
   :age int
   :email string})

(deffunc greet [{:keys [name age email]}]
  (:args User)
  (:return string)
  (str "Hello " name))
```

**Desugared:**
```lisp
(deffunc greet [arg$1]
  (:args User)
  (:return string)
  (let [name  (. arg$1 name)    ; Direct field access
        age   (. arg$1 age)
        email (. arg$1 email)]
    (str "Hello " name)))
```

**Go AST:**
```lisp
(DeclStmt
  :decl (GenDecl
    :tok VAR
    :specs [(ValueSpec
              :names [(Ident :name "name")]
              :values [(SelectorExpr
                         :x (Ident :name "arg$1")
                         :sel (Ident :name "name"))])]))
```

**Go code:**
```go
func greet(arg$1 User) string {
    name := arg$1.name
    age := arg$1.age
    email := arg$1.email
    return zrt.Str("Hello ", name)
}
```

### Map Types → Runtime Extraction

**Zylisp:**
```lisp
(deffunc greet [{:keys [name age email]}]
  (:args (map string interface{}))
  (:return string)
  (str "Hello " name))
```

**Desugared:**
```lisp
(deffunc greet [arg$1]
  (:args (map string interface{}))
  (:return string)
  (let [name  (map-get-or-error arg$1 "name")
        age   (map-get-or-error arg$1 "age")
        email (map-get-or-error arg$1 "email")]
    (str "Hello " name)))
```

**Go code:**
```go
func greet(arg$1 map[string]interface{}) string {
    name := zrt.MapGetOrError(arg$1, "name")
    age := zrt.MapGetOrError(arg$1, "age")
    email := zrt.MapGetOrError(arg$1, "email")
    return zrt.Str("Hello ", name)
}
```

---

## Arity Overloading via Name Mangling

Functions with different arities compile to separate Go functions with mangled names:

### Name Mangling Scheme

```
function-name/arity → function_name__arity
```

**Examples:**
```
greet/0  → greet__0
greet/1  → greet__1
greet/2  → greet__2

my-cool-func/3 → my_cool_func__3
```

### Multi-Arity Example

**Zylisp:**
```lisp
(deffunc greet
  ([]
   (:return string)
   (greet "World"))
  
  ([name]
   (:args string)
   (:return string)
   (str "Hello " name))
  
  ([title name]
   (:args string string)
   (:return string)
   (str "Hello " title " " name)))
```

**Compiles to three separate Go functions:**

```go
func greet__0() string {
    return greet__1("World")
}

func greet__1(name string) string {
    return zrt.Str("Hello ", name)
}

func greet__2(title string, name string) string {
    return zrt.Str("Hello ", title, " ", name)
}
```

### Optional Dynamic Dispatcher

For dynamic calls where arity isn't known at compile time, a dispatcher can be generated:

```go
func greet(args ...interface{}) interface{} {
    switch len(args) {
    case 0:
        return greet__0()
    case 1:
        return greet__1(args[0].(string))
    case 2:
        return greet__2(args[0].(string), args[1].(string))
    default:
        panic(zrt.ArityError("greet", len(args)))
    }
}
```

However, most Zylisp calls have known arity at compile time and call the specific mangled function directly.

---

## Symbol Name Translation

Zylisp allows hyphens and special characters in identifiers, which are translated for Go compatibility:

### Translation Rules

```
Hyphens     -  → _
Stars       *  → _STAR_
Questions   ?  → _QMARK_
Bangs       !  → _BANG_
Arity       /N → __N
```

### Examples

```lisp
my-cool-function     → my_cool_function
http-get-request     → http_get_request
*special-var*        → _STAR_special_var_STAR_
valid?               → valid_QMARK_
reset!               → reset_BANG_

my-function/2        → my_function__2
get-user-by-id/1     → get_user_by_id__1
```

---

## Type System Integration

### Zylisp Type Syntax

Zylisp uses a typed syntax with explicit Go types in function signatures:

```lisp
(deffunc add [a b]
  (:args int int)
  (:return int)
  (+ a b))
```

The `:args` and `:return` forms are **positional** and map directly to Go's function type signature.

### Basic Types

```lisp
; Simple types
(:args int string bool)
(:return float64)

; Slices
(:args []int []string)

; Arrays
(:args [10]int [5]string)

; Maps
(:args (map string int) (map int User))

; Pointers
(:args *User *int)

; Channels
(:args (chan int) (chan<- string) (<-chan bool))

; Functions
(:args (func int string) (func (int string) bool))
```

### Variadic Functions

```lisp
(deffunc sum [numbers...]
  (:args ...int)
  (:return int)
  (reduce + 0 numbers))
```

Maps to Go AST:
```lisp
(FuncType
  :params (FieldList
    :list [(Field 
             :names [(Ident :name "numbers")]
             :type (Ellipsis :elt (Ident :name "int")))])
  :results (FieldList
    :list [(Field :type (Ident :name "int"))]))
```

Go code:
```go
func sum(numbers ...int) int {
    // ...
}
```

### Multiple Return Values

```lisp
(deffunc divide [a b]
  (:args float64 float64)
  (:return float64 error)
  (if (= b 0.0)
    (values 0.0 (error "division by zero"))
    (values (/ a b) nil)))
```

Maps to:
```lisp
(FuncType
  :params (FieldList
    :list [(Field :names [(Ident :name "a")] :type (Ident :name "float64"))
           (Field :names [(Ident :name "b")] :type (Ident :name "float64"))])
  :results (FieldList
    :list [(Field :type (Ident :name "float64"))
           (Field :type (Ident :name "error"))]))
```

### Generics (Go 1.18+)

**Note: This is preliminary and needs more design work.**

```lisp
(deffunc map [f xs]
  (:type-params (T any) (U any))
  (:args (func T U) []T)
  (:return []U)
  ...)
```

Would map to:
```lisp
(FuncType
  :type-params (FieldList
    :list [(Field :names [(Ident :name "T")] :type (Ident :name "any"))
           (Field :names [(Ident :name "U")] :type (Ident :name "any"))])
  :params (FieldList
    :list [(Field :names [(Ident :name "f")] :type (FuncType ...))
           (Field :names [(Ident :name "xs")] :type (ArrayType :elt (Ident :name "T")))])
  :results (FieldList
    :list [(Field :type (ArrayType :elt (Ident :name "U")))]))
```

### User-Defined Types

```lisp
; Type alias
(defalias MyInt int)

; New type definition
(deftype UserId int64)

; Struct type
(deftype Point
  {:x float64
   :y float64})

; Interface type (preliminary)
(definterface Reader
  (Read [[]byte] (:return int error)))
```

Type names are resolved through a symbol table:
1. **Built-in Go types** - Direct mapping (`int`, `string`, `bool`, etc.)
2. **Composite types** - Parsed from syntax (`[]int`, `map[string]int`, etc.)
3. **User-defined types** - Looked up in the type table

---

## Areas Requiring Further Design

The following aspects of the type system and pattern matching integration need additional research and discussion:

### 1. Type Inference

- How much type inference should Zylisp support?
- When are explicit type annotations required vs. optional?
- Type inference for `let` bindings and local variables
- Type inference across function boundaries

### 2. Pattern Matching Type Integration

- How to specify expected field types in patterns?
  ```lisp
  ; Option 1: Inline type hints
  {:keys [(name string) (age int) (email string)]}
  
  ; Option 2: Rely on :args type
  {:keys [name age email]}  ; Types inferred from :args declaration
  ```

- Handling `interface{}` values in patterns
- Type assertions in patterns
- Exhaustiveness checking with type information

### 3. Generic Types

- Complete syntax for type parameters
- Type constraints beyond `any` and `comparable`
- Generic type instantiation syntax
- How generics interact with pattern matching

### 4. Interface Types

- Syntax for interface definitions
- Interface embedding
- Type assertions and type switches
- How to pattern match on interface values

### 5. Pointer Types

- When to use pointers vs. values in patterns
- Dereferencing in destructuring
- Nil handling in patterns
- Mutation through pointer patterns

### 6. Error Handling Integration

- How pattern matching failures integrate with Go's error handling
- Panic vs. error return for match failures
- Pattern matching on `error` types
- Result types and error propagation

### 7. Type Validation

- Compile-time validation of patterns against types
- Runtime type checks and their performance implications
- Type safety guarantees
- When to allow dynamic typing vs. enforce static typing

### 8. Complex Type Expressions

How to represent these in Zylisp syntax:
- Nested generic types: `map[string][]User`
- Function types with multiple parameters: `func(int, string) (bool, error)`
- Channel directions: `<-chan int` vs. `chan<- int`
- Struct tags and metadata

---

## Summary

Zylisp's pattern matching compiles through a clean pipeline:

1. **User syntax** with rich patterns → 
2. **Desugared Zylisp** with simple parameters and explicit bindings →
3. **Canonical S-expressions** mapping 1:1 to Go AST →
4. **Go code**

Key principles:
- Pattern matching is a **Zylisp-level feature** that desugars before reaching the IR
- The **IR remains simple and Go-centric** (managed by `zast`)
- **Type information flows through all stages** via `:args` and `:return` annotations
- **Runtime support** is provided by the `zylisp/runtime` package
- **Name mangling** enables arity overloading and supports Lisp-style naming

The type system integration is **well-started but requires further design work** to handle the full complexity of Go's type system, generics, and the interaction with pattern matching.

---

## Next Steps

1. Complete the design of pattern matching type integration
2. Finalize generic type syntax and semantics
3. Design interface type support
4. Implement the desugaring passes in the Zylisp compiler
5. Build out the `zylisp/runtime` package with pattern matching support
6. Create comprehensive tests for all compilation stages
7. Document the complete type system specification

---

*This document will be updated as design decisions are finalized.*