---
number: 0024
title: Zylisp Error Handling Design Document
author: Duncan McGreggor
created: 2025-10-04
updated: 2025-10-04
state: Draft
supersedes: None
superseded-by: None
---

# Zylisp Error Handling Design Document

**Version:** 0.1  
**Status:** Draft  
**Last Updated:** 2025-10-04

## Executive Summary

Zylisp will adopt an error-as-values approach inspired by Rust's `Result<T, E>` type and Haskell's Either monad, implemented atop Go's structural type system. This design eliminates exceptions in favor of explicit, composable error handling that maintains functional programming principles while leveraging Go's interface-based polymorphism.

## Design Principles

1. **No Exceptions** - All errors are values returned explicitly
2. **Composability** - Rich combinator library for error handling
3. **Type Safety** - Go's type system ensures compile-time guarantees
4. **Explicitness** - Errors cannot be ignored; must be handled or propagated
5. **Extensibility** - Users define their own error types freely
6. **Practicality** - Balance purity with real-world usability

## Core Types

### Result Type

The fundamental type for computations that may fail:

```scheme
;; Generic Result type over success type T and error type E
(data (Result T E)
  (Ok T)      ;; Success case containing value of type T
  (Err E))    ;; Error case containing error of type E
```

**Semantics:**
- `(Ok value)` - computation succeeded with `value`
- `(Err error)` - computation failed with `error`
- Every function that can fail returns a `Result`
- Pattern matching forces handling of both cases

### Maybe/Option Type

For optional values distinct from errors:

```scheme
;; Generic Option type over value type T
(data (Option T)
  (Some T)    ;; Present value
  (None))     ;; Absent value
```

**Use Cases:**
- `None` means "not applicable" or "not found" (not an error)
- `Err` means "something went wrong"
- Nullable references, optional configuration, partial functions

**Distinction:**
- Use `Option` when absence is expected/valid
- Use `Result` when failure needs explanation

## Error Interface

Following Go's structural typing, all errors implement a common interface:

```scheme
;; Base error interface - any type with Error method
(interface error
  (Error) string)  ;; Returns human-readable description
```

**Extended Error Protocol:**

```scheme
;; Optional enriched error interface
(interface DetailedError
  (Error) string           ;; Human description
  (Code) string            ;; Machine-readable code
  (Data) any)              ;; Structured error data

;; Optional error wrapping support
(interface WrappedError
  (Unwrap) error)          ;; Returns underlying cause

;; Optional error categorization
(interface TemporaryError
  (IsTemporary) bool)

(interface TimeoutError
  (IsTimeout) bool)
```

## User-Defined Error Types

Users define domain-specific errors as structured data:

```scheme
;; Parse errors
(type ParseError struct
  (line int)
  (column int)
  (message string)
  (context string))

(method (Error (self ParseError)) string
  (format "Parse error at ~d:~d: ~a" 
          (.line self) 
          (.column self) 
          (.message self)))

(method (Code (self ParseError)) string
  "PARSE_ERROR")

;; IO errors
(type IOError struct
  (operation string)
  (path string)
  (cause string))

(method (Error (self IOError)) string
  (format "IO error during ~a on ~a: ~a"
          (.operation self)
          (.path self)
          (.cause self)))

;; Validation errors
(type ValidationError struct
  (field string)
  (constraint string)
  (actual any))

(method (Error (self ValidationError)) string
  (format "Validation failed for ~a: ~a (got ~a)"
          (.field self)
          (.constraint self)
          (.actual self)))
```

## Error Composition

### Sum Types for Multiple Error Kinds

When a function can fail with different error types:

```scheme
;; Application-level error sum type
(data AppError
  (Parse ParseError)
  (IO IOError)
  (Validation ValidationError)
  (Database DatabaseError)
  (Network NetworkError))

(method (Error (self AppError)) string
  (match self
    [(Parse e) (Error e)]
    [(IO e) (Error e)]
    [(Validation e) (Error e)]
    [(Database e) (Error e)]
    [(Network e) (Error e)]))

;; Function with multiple failure modes
(define (load-config path) (Result Config AppError)
  (result-do
    [text <- (result-map-err IO (read-file path))]
    [parsed <- (result-map-err Parse (parse-config text))]
    [validated <- (result-map-err Validation (validate-config parsed))]
    (Ok validated)))
```

### Error Wrapping and Context

Building context chains for debugging:

```scheme
;; Wrapped error with context
(type ErrorContext struct
  (message string)
  (source error))

(method (Error (self ErrorContext)) string
  (format "~a: ~a" (.message self) (Error (.source self))))

(method (Unwrap (self ErrorContext)) error
  (.source self))

;; Helper function
(define (with-context msg result)
  (result-map-err 
    (lambda (e) (ErrorContext msg e))
    result))

;; Usage
(define (process-user-file user-id)
  (-> (find-user user-id)
      (with-context (format "Finding user ~a" user-id))
      (result-bind get-user-file-path)
      (with-context "Getting user file path")
      (result-bind read-file)
      (with-context "Reading user file")))
```

### Multiple Error Accumulation

For validation scenarios needing all errors:

```scheme
;; Accumulating multiple errors
(type ErrorList struct
  (errors (list error)))

(method (Error (self ErrorList)) string
  (string-join (map Error (.errors self)) "\n"))

(define (validate-user user) (Result User ErrorList)
  (let ([errors '()])
    
    (when (string-empty? (.name user))
      (set! errors (cons (ValidationError "name" "required" "") errors)))
    
    (when (< (.age user) 0)
      (set! errors (cons (ValidationError "age" "non-negative" (.age user)) errors)))
    
    (when (not (valid-email? (.email user)))
      (set! errors (cons (ValidationError "email" "valid format" (.email user)) errors)))
    
    (if (null? errors)
        (Ok user)
        (Err (ErrorList errors)))))
```

## Functional Combinators

The core library of Result operations:

### Functor Operations

```scheme
;; Map over success value
(define (result-map f result) (Result B E)
  "Apply f to value if Ok, otherwise propagate error"
  (match result
    [(Ok val) (Ok (f val))]
    [(Err e) (Err e)]))

;; Map over error value
(define (result-map-err f result) (Result T F)
  "Apply f to error if Err, otherwise propagate success"
  (match result
    [(Ok val) (Ok val)]
    [(Err e) (Err (f e))]))

;; Bi-functor map
(define (result-bimap f-ok f-err result)
  "Apply f-ok to Ok value or f-err to Err value"
  (match result
    [(Ok val) (Ok (f-ok val))]
    [(Err e) (Err (f-err e))]))
```

### Monad Operations

```scheme
;; Bind (>>=) - chain operations returning Results
(define (result-bind result f) (Result B E)
  "Chain f if result is Ok, otherwise propagate error"
  (match result
    [(Ok val) (f val)]
    [(Err e) (Err e)]))

;; Return/Pure
(define (result-ok val) (Result T E)
  "Lift value into Result context"
  (Ok val))

(define (result-err e) (Result T E)
  "Lift error into Result context"
  (Err e))

;; Join - flatten nested Results
(define (result-join nested-result) (Result T E)
  "Flatten (Result (Result T E) E) to (Result T E)"
  (match nested-result
    [(Ok (Ok val)) (Ok val)]
    [(Ok (Err e)) (Err e)]
    [(Err e) (Err e)]))
```

### Applicative Operations

```scheme
;; Apply - for multi-argument functions
(define (result-ap result-fn result-val)
  (result-bind result-fn
    (lambda (f)
      (result-map f result-val))))

;; Lift binary function
(define (result-lift2 f result-a result-b)
  (result-do
    [a <- result-a]
    [b <- result-b]
    (Ok (f a b))))
```

### Utility Operations

```scheme
;; Unwrap with default
(define (result-unwrap-or result default)
  "Extract value or return default if error"
  (match result
    [(Ok val) val]
    [(Err _) default]))

;; Unwrap with lazy default
(define (result-unwrap-or-else result f)
  "Extract value or compute default from error"
  (match result
    [(Ok val) val]
    [(Err e) (f e)]))

;; Unwrap or panic (use sparingly!)
(define (result-unwrap result)
  "Extract value or panic - only use when error is impossible"
  (match result
    [(Ok val) val]
    [(Err e) (panic (format "Called unwrap on Err: ~a" (Error e)))]))

;; Expect with custom panic message
(define (result-expect result msg)
  "Extract value or panic with custom message"
  (match result
    [(Ok val) val]
    [(Err e) (panic (format "~a: ~a" msg (Error e)))]))

;; Predicates
(define (result-ok? result)
  (match result
    [(Ok _) #t]
    [(Err _) #f]))

(define (result-err? result)
  (not (result-ok? result)))

;; And/Or combinators
(define (result-and result-a result-b)
  "Return result-b if result-a is Ok, otherwise return result-a's error"
  (match result-a
    [(Ok _) result-b]
    [(Err e) (Err e)]))

(define (result-or result-a result-b)
  "Return result-a if Ok, otherwise return result-b"
  (match result-a
    [(Ok val) (Ok val)]
    [(Err _) result-b]))
```

### Collection Operations

```scheme
;; Convert list of Results to Result of list
(define (result-collect results) (Result (list T) E)
  "Fail fast: return first error or list of all successes"
  (fold-right
    (lambda (result acc)
      (result-lift2 cons result acc))
    (Ok '())
    results))

;; Partition Results into successes and failures
(define (result-partition results)
  "Returns (values (list T) (list E))"
  (fold-right
    (lambda (result acc)
      (match-let ([(oks errs) acc])
        (match result
          [(Ok val) (values (cons val oks) errs)]
          [(Err e) (values oks (cons e errs))])))
    (values '() '())
    results))

;; Traverse with short-circuiting
(define (result-traverse f list) (Result (list B) E)
  "Apply f to each element, collecting results or first error"
  (result-collect (map f list)))

;; Fold that can fail
(define (result-fold-left f init list) (Result A E)
  "Fold that short-circuits on first error"
  (if (null? list)
      (Ok init)
      (result-bind (f init (car list))
        (lambda (acc)
          (result-fold-left f acc (cdr list))))))
```

## Syntactic Sugar

### Do-Notation (Monadic Sequencing)

```scheme
;; result-do macro for imperative-style sequencing
(define-syntax result-do
  (syntax-rules (<-)
    ;; Base case: final expression
    [(result-do expr)
     expr]
    
    ;; Bind case: pattern <- computation
    [(result-do [pattern <- computation] rest ...)
     (result-bind computation
       (lambda (pattern)
         (result-do rest ...)))]
    
    ;; Let binding
    [(result-do [pattern = expr] rest ...)
     (let ([pattern expr])
       (result-do rest ...))]
    
    ;; Guard/when
    [(result-do (when condition error) rest ...)
     (if condition
         (result-do rest ...)
         (Err error))]))

;; Usage example
(define (complex-operation x y z)
  (result-do
    [a <- (divide x y)]
    [b <- (square-root a)]
    (when (< b 0) (ValidationError "result" "non-negative" b))
    [c <- (divide b z)]
    (Ok (* c 2))))
```

### Try Operator (Rust's ? equivalent)

```scheme
;; try! macro for early returns
(define-syntax try!
  (syntax-rules ()
    [(try! expr)
     (match expr
       [(Ok val) val]
       [(Err e) (return (Err e))])]))

;; Usage (requires function to support early return)
(define (process-request request)
  (let* ([user (try! (authenticate request))]
         [data (try! (parse-body request))]
         [validated (try! (validate data))]
         [result (try! (save-to-db validated))])
    (Ok result)))
```

### Pipeline-Friendly Operators

```scheme
;; Threading macro with Result awareness
(define-syntax ->?
  (syntax-rules ()
    [(->? expr) expr]
    [(->? expr (fn args ...) rest ...)
     (->? (result-bind expr (lambda (x) (fn x args ...))) rest ...)]
    [(->? expr fn rest ...)
     (->? (result-bind expr fn) rest ...)]))

;; Usage
(define result
  (->? (parse-int user-input)
       (divide 100)
       validate-positive
       format-output))
```

## Standard Library Organization

```scheme
;; result.zy - Result type and core operations
(module zylisp/result
  (export
    ;; Types
    Result Ok Err
    
    ;; Constructors
    ok err
    
    ;; Query
    ok? err?
    
    ;; Functor
    map map-err bimap
    
    ;; Monad
    bind return join
    
    ;; Applicative
    ap lift2 lift3
    
    ;; Utilities
    unwrap unwrap-or unwrap-or-else expect
    and or and-then or-else
    
    ;; Collections
    collect partition traverse
    fold-left fold-right
    
    ;; Context
    with-context wrap-error))

;; option.zy - Option type
(module zylisp/option
  (export
    Option Some None
    some none
    some? none?
    map bind
    unwrap-or unwrap-or-else
    and or
    to-result from-result))

;; error.zy - Error interface and utilities
(module zylisp/error
  (export
    error DetailedError WrappedError
    TemporaryError TimeoutError
    
    ;; Standard error types
    ErrorContext ErrorList
    
    ;; Helpers
    new-error wrap-error
    is-temporary? is-timeout?))
```

## Integration with Go

### FFI Interop

Converting between Go's `(value, error)` tuples and Zylisp Results:

```scheme
;; Wrapper for Go functions returning (T, error)
(define (go-call->result go-fn)
  (lambda args
    (let-values ([(val err) (apply go-fn args)])
      (if (nil? err)
          (Ok val)
          (Err err)))))

;; Usage
(define read-file
  (go-call->result go:os/ReadFile))

;; Result to Go tuple
(define (result->go-values result)
  (match result
    [(Ok val) (values val nil)]
    [(Err e) (values nil e)]))
```

### Go Interface Implementation

Zylisp error types automatically implement Go's `error` interface:

```go
// Go side sees Zylisp errors as standard Go errors
type ZylispError interface {
    error
}

// All Zylisp error types implement Error() string
```

## Best Practices

### When to Use Result vs Option

- **Use Result when:**
  - Operation can fail and you need to explain why
  - Caller needs to handle different error cases
  - Error contains useful context for recovery

- **Use Option when:**
  - Absence is a valid, expected state
  - No explanation needed for absence (e.g., lookup in map)
  - Semantic meaning is "not applicable" rather than "failed"

### Error Granularity

```scheme
;; Too coarse - hard to handle specifically
(define GenericError (error-new "something went wrong"))

;; Too fine - explosion of error types
(type UserNameTooShortError ...)
(type UserNameTooLongError ...)
(type UserNameInvalidCharError ...)

;; Just right - structured with data
(type ValidationError struct
  (field string)
  (reason string)
  (value any))
```

### When to Unwrap

```scheme
;; ❌ BAD: Unwrap without checking (can panic)
(define user (result-unwrap (find-user id)))

;; ✅ GOOD: Handle the error
(define user-result (find-user id))
(match user-result
  [(Ok user) (process user)]
  [(Err e) (handle-error e)])

;; ✅ ACCEPTABLE: Unwrap with proof of safety
(define user
  ;; We just created this user, it must exist
  (result-expect (find-user newly-created-id)
                 "BUG: Newly created user not found"))

;; ✅ GOOD: Provide default
(define user (result-unwrap-or (find-user id) guest-user))
```

### Error Context

```scheme
;; Add context at boundaries
(define (handle-request request)
  (-> (authenticate request)
      (with-context "Authentication failed")
      (result-bind authorize)
      (with-context "Authorization failed")
      (result-bind process)
      (with-context "Processing failed")))

;; Now errors have full context chain:
;; "Processing failed: Authorization failed: insufficient permissions"
```

## Future Considerations

### Optional Enhancements

1. **Static Type Checking**
   - Exhaustiveness checking for pattern matches
   - Compile-time detection of unhandled Results
   - Generic type inference

2. **Effect System Integration**
   - Track error types in function signatures
   - Automatic error propagation at type level
   - Effect handlers for flexible control flow

3. **Async/Concurrent Results**
   - Result integration with promises/futures
   - Parallel error collection
   - Timeout-aware Results

4. **Serialization**
   - Standard JSON representation for errors
   - Wire format for Result types
   - Interop with other languages

5. **Tooling**
   - REPL helpers for Result inspection
   - Debugger integration for error traces
   - Linter rules for Result best practices

## Conclusion

This design provides Zylisp with a principled, composable approach to error handling that:

- Eliminates exceptions while remaining practical
- Leverages Go's type system for safety and performance
- Provides rich functional abstractions for error handling
- Allows users to define domain-specific error types
- Scales from simple scripts to large applications

The Result type, backed by Go's structural interfaces and enhanced with functional combinators, gives Zylisp developers the best of both worlds: the rigor of Rust/Haskell error handling with the simplicity and performance of Go.
