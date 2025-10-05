---
number: 0021
title: Tail Call Optimization Approaches for Zylisp
author: Duncan McGreggor
created: 2025-10-04
updated: 2025-10-04
state: Draft
supersedes: None
superseded-by: None
---

# Tail Call Optimization Approaches for Zylisp

## Background

Tail call optimization (TCO) is a crucial feature in functional programming languages, allowing recursive functions to execute in constant stack space. However, Zylisp faces a fundamental challenge: it compiles to Go AST, and **Go does not support tail call optimization natively**.

This document explores the options available for handling tail recursion in Zylisp.

## The Challenge

When a Lisp compiles to a host language without native TCO support, tail-recursive functions will consume stack space with each call, potentially leading to stack overflow errors. This is a well-known limitation:

- **Clojure on the JVM**: The JVM doesn't support tail calls, so Clojure provides the `recur` special form for self-recursion optimization
- **Scheme implementations on JavaScript**: Often use trampolining or CPS transformation
- **Other hosted Lisps**: Generally must work around the host platform's limitations

## Approach 1: No General TCO

**Description**: Accept Go's limitations and provide no automatic tail call optimization.

**Pros**:
- Simplest to implement
- Generated Go code is straightforward and readable
- No runtime overhead
- Predictable behavior matching Go's semantics

**Cons**:
- Deeply recursive code will overflow the stack
- Not idiomatic for Lisp programmers
- Limits functional programming patterns

**Verdict**: Not recommended as the sole approach, but acceptable as a baseline.

## Approach 2: Self-Recursion Optimization via `recur`

**Description**: Implement a `recur` special form (like Clojure) that optimizes tail-recursive calls to the same function by converting them to loops in the generated Go code.

**Example Zylisp**:
```lisp
(defn factorial [n acc]
  (if (<= n 1)
      acc
      (recur (- n 1) (* n acc))))
```

**Generated Go** (conceptual):
```go
func factorial(n int, acc int) int {
    for {
        if n <= 1 {
            return acc
        }
        // Update parameters for next iteration
        n_next := n - 1
        acc_next := n * acc
        n = n_next
        acc = acc_next
    }
}
```

**Pros**:
- Solves the most common use case (tail-recursive loops)
- Proven approach (Clojure has used this successfully for years)
- Generated Go code remains readable and debuggable
- No runtime overhead
- Can be implemented cleanly in AST generation
- Compile-time verification that `recur` is in tail position

**Cons**:
- Only works for self-recursion, not mutual recursion
- Requires programmer discipline to use `recur` instead of direct function calls
- Not transparent - programmers must know when to use `recur`

**Implementation Notes**:
- Detect when a function call in tail position is to the same function
- Transform to a labeled loop with parameter rebinding
- Could use Go's `goto` for jumping back to the loop start if preferred
- Need to validate that `recur` only appears in tail position (compile-time error otherwise)

**Verdict**: **Recommended as the primary approach**. This covers 90%+ of practical tail recursion needs and aligns with "Go Flavoured Lisp" philosophy.

## Approach 3: Trampolining

**Description**: Functions return thunks (zero-argument functions) representing the next computation, which are executed in a loop by a trampoline function.

**Example**:
```lisp
(defn factorial-impl [n acc]
  (if (<= n 1)
      acc
      (fn [] (factorial-impl (- n 1) (* n acc)))))

(defn factorial [n]
  (trampoline (factorial-impl n 1)))
```

**Pros**:
- Supports mutual recursion
- Can work across function boundaries
- Well-understood pattern

**Cons**:
- Significant runtime overhead (function allocations, indirect calls)
- Requires explicit programmer opt-in
- Generated Go code is more complex
- Not transparent - changes the programming model
- Less "Go flavoured" - allocates closures heavily

**Verdict**: Could be provided as a library function for advanced cases, but not as the primary TCO mechanism.

## Approach 4: Continuation-Passing Style (CPS) Transformation

**Description**: Automatically transform all functions to CPS, where every function takes an additional continuation parameter representing "what to do next".

**Pros**:
- Enables general tail call optimization
- Theoretically elegant
- Supports mutual recursion

**Cons**:
- Extremely complex to implement correctly
- Generated Go code becomes nearly unreadable
- Significant performance overhead
- Makes debugging very difficult
- Complicates interop with Go code
- Not "Go Flavoured" at all

**Verdict**: Not recommended for Zylisp. The complexity and performance costs outweigh the benefits.

## Approach 5: Hybrid Approach with `goto`

**Description**: Use Go's `goto` statement for tail position jumps within the same function, combined with loop transformation.

**Example Generated Go**:
```go
func factorial(n int, acc int) int {
start:
    if n <= 1 {
        return acc
    }
    n_temp := n - 1
    acc_temp := n * acc
    n = n_temp
    acc = acc_temp
    goto start
}
```

**Pros**:
- Very efficient (just a jump)
- Clear control flow in generated code
- Works well for self-recursion

**Cons**:
- `goto` can be controversial (though justified here)
- Same limitations as Approach 2 (self-recursion only)

**Verdict**: A valid implementation detail for Approach 2. Whether to use `goto` or `for` loop is mostly aesthetic.

## Recommendation

**Implement Approach 2: `recur` for Self-Recursion**

This provides the best balance of:
- **Practicality**: Covers the vast majority of tail recursion use cases
- **Performance**: Zero runtime overhead
- **Readability**: Generated Go code remains clear
- **Go Flavour**: Stays close to Go's execution model
- **Clojure Compatibility**: Familiar to Clojure developers

### Implementation Phases

**Phase 1** (MVP):
- Implement `recur` special form
- Compile-time validation that `recur` is in tail position
- Generate loop-based Go code

**Phase 2** (Optional):
- Provide trampoline library functions for advanced cases
- Document when to use each approach
- Possibly add compiler warnings for non-tail recursive calls

**Phase 3** (Future):
- Consider mutual recursion detection and transformation
- Explore optimization of common recursive patterns

## Comparison with Other Lisps

| Language | Platform | TCO Approach |
|----------|----------|--------------|
| Clojure | JVM | `recur` for self-recursion |
| ClojureScript | JavaScript | `recur` for self-recursion |
| Scheme (some impls) | JavaScript | Trampolining or CPS |
| Common Lisp | Native | Often full TCO (platform dependent) |
| Racket | Native | Full TCO |
| **Zylisp** | **Go** | **`recur` (recommended)** |

## Conclusion

While full tail call optimization isn't feasible when compiling to Go, the `recur` approach provides a pragmatic, performant solution that aligns with Zylisp's "Go Flavoured Lisp" philosophy. This approach has proven successful in Clojure and will feel natural to Lisp programmers while generating clean, debuggable Go code.