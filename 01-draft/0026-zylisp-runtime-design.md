---
number: 0026
title: Zylisp Runtime Library Design
author: Duncan McGreggor
created: 2025-10-04
updated: 2025-10-04
state: Draft
supersedes: None
superseded-by: None
---

# Zylisp Runtime Library Design

## Overview

The Zylisp runtime library (`zylisp/runtime`) is a **Go package that provides runtime support for compiled Zylisp programs**. It is NOT used by the compiler itself - rather, it is imported by the Go code that the compiler generates.

This document explains why we need a runtime library, what it contains, and how it fits into the Zylisp ecosystem.

---

## Why We Need a Runtime Library

### The Core Problem

Zylisp provides high-level features that don't map directly to Go primitives:

1. **Pattern matching** - Requires runtime checks and field extraction
2. **Dynamic operations** - Map access with error handling
3. **Lisp-style string operations** - String concatenation that handles any type
4. **List operations** - Generic operations on sequences
5. **Error handling** - Pattern match failures, arity errors

We have two options for implementing these features:

**Option A: Inline everything**
```go
// Generated code without runtime library
func greet__1(arg$1 map[string]interface{}) string {
    var name interface{}
    if v, ok := arg$1["name"]; ok {
        name = v
    } else {
        panic(fmt.Sprintf("key %q not found in map", "name"))
    }
    
    var age interface{}
    if v, ok := arg$1["age"]; ok {
        age = v
    } else {
        panic(fmt.Sprintf("key %q not found in map", "age"))
    }
    
    // ... repeat for email ...
    
    var parts []string
    parts = append(parts, "Hello ")
    parts = append(parts, fmt.Sprint(name))
    return strings.Join(parts, "")
}
```

**Option B: Use runtime library**
```go
import zrt "github.com/zylisp/runtime"

func greet__1(arg$1 map[string]interface{}) string {
    name := zrt.MapGetOrError(arg$1, "name")
    age := zrt.MapGetOrError(arg$1, "age")
    email := zrt.MapGetOrError(arg$1, "email")
    return zrt.Str("Hello ", name)
}
```

### Benefits of Option B (Runtime Library)

1. **Concise generated code** - Much easier to read and debug
2. **Consistent behavior** - All programs use the same runtime functions
3. **Easy to update** - Fix bugs or improve performance in one place
4. **Better error messages** - Runtime can provide rich, helpful errors
5. **Optimization potential** - Runtime functions can be optimized once
6. **Code reuse** - No duplication across generated programs

### Tradeoffs

**Pros**:
- Generated code is much cleaner
- Single source of truth for runtime behavior
- Easier to maintain and evolve

**Cons**:
- External dependency for compiled programs
- Need to version and maintain the runtime
- Slightly harder to understand what generated code does (need to look at runtime source)

**Decision**: The benefits far outweigh the costs. A runtime library is the right choice.

---

## What Goes in the Runtime

### Pattern Matching Support

The core functionality needed for Zylisp's pattern matching feature:

```go
package runtime

import "fmt"

// MapGetOrError extracts a value from a map, panicking if the key is missing
func MapGetOrError(m map[string]interface{}, key string) interface{} {
    if v, ok := m[key]; ok {
        return v
    }
    panic(fmt.Sprintf("pattern match failed: key %q not found in map", key))
}

// MapGetOrDefault extracts a value from a map with a default value
func MapGetOrDefault(m map[string]interface{}, key string, def interface{}) interface{} {
    if v, ok := m[key]; ok {
        return v
    }
    return def
}

// MapHasKey checks if a map contains a key
func MapHasKey(m map[string]interface{}, key string) bool {
    _, ok := m[key]
    return ok
}

// MapKeys returns all keys from a map
func MapKeys(m map[string]interface{}) []string {
    keys := make([]string, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    return keys
}
```

### Type Checking Predicates

Runtime type checks needed for pattern matching and guards:

```go
// IsMap checks if a value is a map
func IsMap(v interface{}) bool {
    _, ok := v.(map[string]interface{})
    return ok
}

// IsList checks if a value is a list/slice
func IsList(v interface{}) bool {
    switch v.(type) {
    case []interface{}:
        return true
    default:
        return false
    }
}

// IsString checks if a value is a string
func IsString(v interface{}) bool {
    _, ok := v.(string)
    return ok
}

// IsNumber checks if a value is numeric
func IsNumber(v interface{}) bool {
    switch v.(type) {
    case int, int8, int16, int32, int64:
        return true
    case uint, uint8, uint16, uint32, uint64:
        return true
    case float32, float64:
        return true
    default:
        return false
    }
}

// IsBool checks if a value is a boolean
func IsBool(v interface{}) bool {
    _, ok := v.(bool)
    return ok
}

// IsNil checks if a value is nil
func IsNil(v interface{}) bool {
    return v == nil
}
```

### Error Handling

Helper functions for generating helpful error messages:

```go
// MatchError creates a detailed pattern match error
type MatchError struct {
    Function string
    Expected string
    Got      interface{}
    Reason   string
}

func (e *MatchError) Error() string {
    return fmt.Sprintf(
        "pattern match failed in %s: expected %s, got %T: %s",
        e.Function, e.Expected, e.Got, e.Reason,
    )
}

// ThrowMatchError panics with a match error
func ThrowMatchError(funcName string, value interface{}) {
    panic(&MatchError{
        Function: funcName,
        Expected: "matching pattern",
        Got:      value,
        Reason:   fmt.Sprintf("value %v did not match any pattern", value),
    })
}

// ArityError represents a function arity mismatch
type ArityError struct {
    Function       string
    Called         int
    ValidArities   []int
}

func (e *ArityError) Error() string {
    return fmt.Sprintf(
        "arity error in %s: called with %d arguments, valid arities: %v",
        e.Function, e.Called, e.ValidArities,
    )
}

// ThrowArityError panics with an arity error
func ThrowArityError(funcName string, called int, valid ...int) {
    panic(&ArityError{
        Function:     funcName,
        Called:       called,
        ValidArities: valid,
    })
}
```

### String Operations

Zylisp's `str` function concatenates values of any type:

```go
import "strings"

// Str concatenates any number of values into a string
func Str(parts ...interface{}) string {
    if len(parts) == 0 {
        return ""
    }
    
    var sb strings.Builder
    for _, p := range parts {
        sb.WriteString(ToString(p))
    }
    return sb.String()
}

// ToString converts any value to its string representation
func ToString(v interface{}) string {
    if v == nil {
        return ""
    }
    
    switch val := v.(type) {
    case string:
        return val
    case fmt.Stringer:
        return val.String()
    default:
        return fmt.Sprint(v)
    }
}
```

### List/Sequence Operations

Generic operations on lists and sequences:

```go
// Length returns the length of a list/slice
func Length(lst interface{}) int {
    switch v := lst.(type) {
    case []interface{}:
        return len(v)
    case string:
        return len(v)
    default:
        panic(fmt.Sprintf("length: expected list or string, got %T", lst))
    }
}

// Nth returns the nth element of a list (0-indexed)
func Nth(lst interface{}, n int) interface{} {
    switch v := lst.(type) {
    case []interface{}:
        if n < 0 || n >= len(v) {
            panic(fmt.Sprintf("nth: index %d out of bounds for list of length %d", n, len(v)))
        }
        return v[n]
    default:
        panic(fmt.Sprintf("nth: expected list, got %T", lst))
    }
}

// First returns the first element of a list
func First(lst interface{}) interface{} {
    switch v := lst.(type) {
    case []interface{}:
        if len(v) == 0 {
            return nil
        }
        return v[0]
    default:
        panic(fmt.Sprintf("first: expected list, got %T", lst))
    }
}

// Rest returns all but the first element of a list
func Rest(lst interface{}) []interface{} {
    switch v := lst.(type) {
    case []interface{}:
        if len(v) == 0 {
            return []interface{}{}
        }
        return v[1:]
    default:
        panic(fmt.Sprintf("rest: expected list, got %T", lst))
    }
}

// Cons prepends an element to a list
func Cons(elem interface{}, lst interface{}) []interface{} {
    switch v := lst.(type) {
    case []interface{}:
        result := make([]interface{}, len(v)+1)
        result[0] = elem
        copy(result[1:], v)
        return result
    default:
        panic(fmt.Sprintf("cons: expected list, got %T", lst))
    }
}

// Append appends elements to a list (returns new list)
func Append(lst interface{}, elems ...interface{}) []interface{} {
    switch v := lst.(type) {
    case []interface{}:
        result := make([]interface{}, len(v)+len(elems))
        copy(result, v)
        copy(result[len(v):], elems)
        return result
    default:
        panic(fmt.Sprintf("append: expected list, got %T", lst))
    }
}
```

### Data Structure Constructors

Helpers for creating Zylisp data structures:

```go
// MakeList creates a list from variadic arguments
func MakeList(items ...interface{}) []interface{} {
    return items
}

// MakeMap creates a map from key-value pairs
// Usage: MakeMap(":name", "Alice", ":age", 30)
func MakeMap(pairs ...interface{}) map[string]interface{} {
    if len(pairs)%2 != 0 {
        panic("MakeMap: odd number of arguments")
    }
    
    m := make(map[string]interface{}, len(pairs)/2)
    for i := 0; i < len(pairs); i += 2 {
        key, ok := pairs[i].(string)
        if !ok {
            panic(fmt.Sprintf("MakeMap: key must be string, got %T", pairs[i]))
        }
        m[key] = pairs[i+1]
    }
    return m
}
```

### Numeric Operations

If Zylisp needs generic numeric operations:

```go
// Add performs addition on any numeric types
func Add(a, b interface{}) interface{} {
    // Try int first (most common)
    if ai, ok := a.(int); ok {
        if bi, ok := b.(int); ok {
            return ai + bi
        }
    }
    
    // Fall back to float64 for mixed types
    af := ToFloat64(a)
    bf := ToFloat64(b)
    return af + bf
}

// ToFloat64 converts any numeric type to float64
func ToFloat64(v interface{}) float64 {
    switch val := v.(type) {
    case int:
        return float64(val)
    case int64:
        return float64(val)
    case float64:
        return val
    case float32:
        return float64(val)
    default:
        panic(fmt.Sprintf("ToFloat64: expected number, got %T", v))
    }
}

// Similar for Sub, Mul, Div, etc.
```

---

## What Does NOT Go in the Runtime

It's important to be clear about what the runtime is NOT:

### Not a Virtual Machine

The runtime is NOT a bytecode interpreter or VM. Zylisp compiles to native Go code, which compiles to native machine code. The runtime just provides helper functions.

### Not the Compiler

The runtime is NOT used during compilation. It's used by the compiled programs:

```
zylisp/lang (compiler) does NOT import zylisp/runtime
Generated Go code DOES import zylisp/runtime
```

### Not a Standard Library

Core Zylisp functions (like `reduce`, `filter`, `map`) should probably be implemented as **regular Zylisp functions** that compile to Go, not as runtime primitives.

The runtime is only for **low-level operations** that can't be easily expressed in Zylisp itself or that benefit from being shared across all programs.

### Not Type-Specific Business Logic

User-defined types and their methods belong in user code, not the runtime:

```go
// ❌ Don't put this in runtime
func ProcessUser(u User) string { ... }

// ✅ Runtime provides generic primitives
func StructGetField(s interface{}, fieldName string) interface{} { ... }
```

---

## Repository Structure

### Recommended Organization

```
github.com/zylisp/runtime/
├── README.md
├── go.mod
├── LICENSE
├── runtime.go           # Main runtime functions
├── patterns.go          # Pattern matching support
├── types.go             # Type checking predicates
├── errors.go            # Error types and helpers
├── strings.go           # String operations
├── lists.go             # List/sequence operations
├── maps.go              # Map operations
├── numeric.go           # Numeric operations (if needed)
└── runtime_test.go      # Comprehensive tests
```

### Versioning Strategy

The runtime library should use **semantic versioning** strictly:

- **Major version** changes: Breaking API changes
- **Minor version** changes: New functions added (backwards compatible)
- **Patch version** changes: Bug fixes, performance improvements

Example:
```
v1.0.0 - Initial release
v1.1.0 - Added MakeMap helper
v1.1.1 - Fixed bug in MapGetOrError error message
v2.0.0 - Changed MapGetOrError to return (value, error) instead of panicking
```

### Stability Guarantee

Once `v1.0.0` is released, the runtime API must remain **stable**. Any breaking change requires a major version bump.

This is critical because compiled Zylisp programs will depend on specific runtime versions.

---

## Usage in Generated Code

### Import Convention

Generated code imports the runtime with a short alias:

```go
package main

import (
    "fmt"
    zrt "github.com/zylisp/runtime"  // Standard alias: zrt
)

func greet__1(arg$1 map[string]interface{}) string {
    name := zrt.MapGetOrError(arg$1, "name")
    return zrt.Str("Hello ", name)
}
```

### Example Generated Code

From this Zylisp:

```lisp
(deffunc process-user [{:keys [name age email]}]
  (:args (map string interface{}))
  (:return string)
  (if (> age 18)
    (str name " is an adult")
    (str name " is a minor")))
```

Generated Go code:

```go
package main

import zrt "github.com/zylisp/runtime"

func process_user__1(arg$1 map[string]interface{}) string {
    name := zrt.MapGetOrError(arg$1, "name")
    age := zrt.MapGetOrError(arg$1, "age")
    email := zrt.MapGetOrError(arg$1, "email")
    
    // Convert age to int for comparison
    ageInt, ok := age.(int)
    if !ok {
        panic("age must be int")
    }
    
    if ageInt > 18 {
        return zrt.Str(name, " is an adult")
    } else {
        return zrt.Str(name, " is a minor")
    }
}
```

---

## Dependency Graph

The runtime sits **outside** the compilation dependency chain:

```
Compilation Dependencies:
┌──────────────┐
│ zylisp/core  │  ← Source maps, positions, errors
└──────┬───────┘
       │
       ├─────────────────┬─────────────────┐
       ↓                 ↓                 ↓
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ zylisp/zast  │  │ zylisp/lang  │  │ zylisp/cli   │
└──────────────┘  └──────┬───────┘  └──────────────┘
                         │
                         ↓
                  ┌──────────────┐
                  │ Generated Go │
                  │     Code     │
                  └──────┬───────┘
                         │
                         ↓
Runtime Dependency:      │
                  ┌──────────────┐
                  │ zylisp/      │ ← Used ONLY by generated code
                  │  runtime     │
                  └──────────────┘
```

Key points:
- The **compiler** (`zylisp/lang`) does NOT depend on `zylisp/runtime`
- The **generated Go code** depends on `zylisp/runtime`
- No circular dependencies

---

## Performance Considerations

### When to Use Runtime vs. Inline

For very hot paths or simple operations, the compiler could inline instead of calling runtime:

**Simple case - inline it:**
```go
// Instead of: x := zrt.First(lst)
// Generate:
x := lst[0]  // Direct array access
```

**Complex case - use runtime:**
```go
// Pattern matching with multiple checks
name := zrt.MapGetOrError(arg$1, "name")  // Better as runtime function
```

### Optimization Opportunities

The runtime can be optimized independently:

1. **Benchmarking** - Profile runtime functions to find bottlenecks
2. **Type-specific fast paths** - Check for common types first
3. **Memory pooling** - Reuse allocations where appropriate
4. **Compiler hints** - Use `//go:inline` directives strategically

### Example Optimization

```go
// Optimized version with fast path
func MapGetOrError(m map[string]interface{}, key string) interface{} {
    // Fast path: direct lookup (no error formatting unless needed)
    if v, ok := m[key]; ok {
        return v
    }
    
    // Slow path: build detailed error only when necessary
    availableKeys := make([]string, 0, len(m))
    for k := range m {
        availableKeys = append(availableKeys, k)
    }
    
    panic(fmt.Sprintf(
        "pattern match failed: key %q not found in map\nAvailable keys: %v",
        key, availableKeys,
    ))
}
```

---

## Testing Strategy

The runtime must be **extensively tested** since all generated code depends on it.

### Test Categories

1. **Unit tests** - Test each function in isolation
2. **Property tests** - Use fuzzing to test edge cases
3. **Integration tests** - Test combinations of runtime functions
4. **Performance benchmarks** - Ensure performance is acceptable

### Example Tests

```go
func TestMapGetOrError(t *testing.T) {
    tests := []struct {
        name      string
        m         map[string]interface{}
        key       string
        want      interface{}
        wantPanic bool
    }{
        {
            name: "key exists",
            m:    map[string]interface{}{"name": "Alice"},
            key:  "name",
            want: "Alice",
        },
        {
            name:      "key missing",
            m:         map[string]interface{}{"name": "Alice"},
            key:       "age",
            wantPanic: true,
        },
        {
            name:      "empty map",
            m:         map[string]interface{}{},
            key:       "name",
            wantPanic: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.wantPanic {
                defer func() {
                    if r := recover(); r == nil {
                        t.Error("expected panic, got none")
                    }
                }()
            }
            
            got := MapGetOrError(tt.m, tt.key)
            if !tt.wantPanic && got != tt.want {
                t.Errorf("MapGetOrError() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

---

## Future Extensions

### Potential Additions

1. **Metadata support** - Attach metadata to values
2. **Protocols/Interfaces** - Runtime support for Zylisp protocols
3. **Lazy sequences** - Iterator/generator support
4. **Concurrency primitives** - Channels, goroutine helpers
5. **Reflection helpers** - Generic struct field access
6. **JSON/Serialization** - Convert between Zylisp values and JSON
7. **Debug support** - Pretty printing, value inspection

### Plugin System

The runtime could support plugins for extended functionality:

```go
// Allow users to register custom type handlers
func RegisterTypeHandler(typeName string, handler TypeHandler)

// Allow custom error formatters
func RegisterErrorFormatter(formatter ErrorFormatter)
```

---

## Documentation Requirements

### Public API Documentation

Every exported function must have:
- **Clear godoc** explaining what it does
- **Examples** showing usage
- **Panic conditions** documenting when it panics
- **Performance characteristics** (O(n), O(1), etc.)

Example:

```go
// MapGetOrError extracts a value from a map by key.
// If the key is not found, it panics with a detailed error message
// listing available keys.
//
// Example:
//   m := map[string]interface{}{"name": "Alice", "age": 30}
//   name := MapGetOrError(m, "name")  // Returns "Alice"
//   email := MapGetOrError(m, "email") // Panics: key not found
//
// Panics:
//   - If the key is not present in the map
//
// Performance: O(1) for lookup, O(n) for error message (only when panicking)
func MapGetOrError(m map[string]interface{}, key string) interface{} {
    // ...
}
```

### Migration Guides

When breaking changes are necessary (major version bump), provide:
- **Migration guide** from old version to new
- **Deprecation warnings** in advance
- **Compatibility shims** when possible

---

## Summary

The Zylisp runtime library is a **critical piece of infrastructure** that:

1. **Provides runtime support** for compiled Zylisp programs
2. **Lives in a separate repository** (`zylisp/runtime`)
3. **Has no dependencies** on other Zylisp components
4. **Is imported by generated Go code**, not by the compiler
5. **Must be stable and well-tested** (it's a dependency of all Zylisp programs)
6. **Uses semantic versioning** with strong backwards compatibility guarantees

Key functions:
- **Pattern matching support** - Map extraction, type checking
- **Error handling** - Match errors, arity errors
- **String operations** - Generic string concatenation
- **List operations** - Generic sequence manipulation
- **Data structure constructors** - Creating Zylisp values

The runtime enables the compiler to generate **clean, readable Go code** while providing **consistent, well-tested implementations** of common operations across all Zylisp programs.

---

## Next Steps

1. **Create `zylisp/runtime` repository**
2. **Implement core pattern matching functions** (MapGetOrError, etc.)
3. **Write comprehensive tests**
4. **Add documentation and examples**
5. **Publish v0.1.0** for early experimentation
6. **Iterate based on compiler needs**
7. **Release v1.0.0** when stable

---

*The runtime is the foundation that compiled Zylisp programs are built on. It must be rock-solid.*