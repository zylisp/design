---
number: 0020
title: "Immutable Data in Go: Challenges and Research Areas"
author: Duncan McGreggor
created: 2025-10-04
updated: 2025-10-04
state: Draft
supersedes: None
superseded-by: None
---

# Immutable Data in Go: Challenges and Research Areas

## Core Challenges

### Language-Level Issues

**No Immutability Primitives**
- Go lacks `const`, `readonly`, or `immutable` keywords for data structures
- No compiler enforcement of immutability contracts
- Must rely entirely on discipline and conventions

**Reference Semantics for Collections**
- Maps are always references - assignment doesn't copy
- Slices share underlying arrays - copying the slice header doesn't copy data
- Interfaces can hide mutable concrete types
- Pointers in structs break value semantics

**Value vs Reference Confusion**
- Structs copy by value, but their pointer/slice/map fields maintain references
- Method receivers can be value or pointer, changing mutation semantics
- Interface values obscure whether underlying type is mutable

### Practical Runtime Issues

**Garbage Collection Pressure**
- True immutability requires constant allocation of new objects
- Copying large data structures on every "mutation" is expensive
- Go's GC is generational but can struggle with high allocation rates

**Memory Overhead**
- Naive copying wastes memory
- Without structural sharing, similar data structures duplicate storage
- Cache efficiency suffers from scattered allocations

**Closure Capture Semantics**
- Closures capture references to slices/maps, not copies
- Easy to accidentally share mutable state through closures
- No way to force capture-by-value for reference types

## Potential Solutions

### Structural Sharing with Persistent Data Structures

**Technique**: Use tree-based data structures that share unchanged portions between versions

**Implementations**:
- Persistent lists via cons cells (simple, efficient)
- Persistent vectors via tree structures with wide branching factor (32-way tries)
- Persistent hash maps via Hash Array Mapped Tries (HAMT)

**Trade-offs**:
- Better memory efficiency than naive copying
- O(log₃₂ N) access instead of O(1) for native slices
- More complex implementation
- Additional pointer chasing affects cache locality

### Opaque Type Wrappers

**Technique**: Wrap mutable Go types in structs with unexported fields, export only immutable operations

```go
type ImmutableList struct {
    items []Value // unexported
}

func (l ImmutableList) Append(v Value) ImmutableList {
    // return new list, don't modify receiver
}
```

**Benefits**:
- Clear API boundaries
- Compile-time prevention of direct mutation
- Can optimize internally while maintaining external immutability

**Limitations**:
- Runtime overhead of copying
- Doesn't prevent type assertions or reflection-based access
- Verbose API compared to native collections

### Copy-on-Write Strategies

**Technique**: Share data until mutation is required, then copy

**Approaches**:
- Reference counting to track sharing
- Lazy copying with dirty flags
- Generation counters to invalidate shared references

**Challenges**:
- Requires runtime overhead to track sharing
- Thread safety becomes complex
- Go's lack of move semantics makes this harder

### Code Generation and Tooling

**Technique**: Generate immutable wrappers or verify immutability statically

**Options**:
- Custom linters to enforce immutability patterns
- Code generators for boilerplate immutable types
- Struct tags or comments to mark immutable contracts

**Limitations**:
- Doesn't change language semantics
- Can be worked around by determined programmers
- Adds build complexity

## Areas for Additional Research

### Performance Benchmarking

**Questions**:
- What is the allocation overhead of persistent data structures vs naive copying?
- How do persistent structures perform with Go's GC compared to mutable versions?
- What are the cache effects of pointer-heavy tree structures?
- Can object pooling mitigate allocation costs for common operations?

**Next Steps**:
- Implement microbenchmarks for list, vector, and map operations
- Profile realistic Lisp workloads
- Test with varying GC settings (GOGC)

### Persistent Data Structure Implementations

**Research Needed**:
- Survey existing Go libraries (immu, immutable, peds)
- Compare HAMT implementations for hash maps
- Evaluate RRB-trees vs simple 32-way tries for vectors
- Investigate bit-partitioned tries for better cache locality

**Questions**:
- Can we optimize for Go's specific memory model?
- Are there Go-specific tricks (unsafe pointer arithmetic) worth using?
- How do these structures interact with Go's escape analysis?

### Thread Safety and Concurrency

**Questions**:
- Do immutable structures need locks at all?
- How do concurrent readers affect GC with structural sharing?
- Can we leverage Go's channel patterns for safe sharing?
- What happens with high-frequency updates from multiple goroutines?

**Research**:
- Test concurrent access patterns
- Explore lock-free persistent structures
- Investigate generation-based validation schemes

### Interop with Go Ecosystem

**Challenges**:
- How to accept/return Go's native collections safely?
- Freezing/thawing between mutable and immutable views
- Integration with encoding/json, database/sql, etc.

**Questions**:
- Should we provide conversion functions?
- Can we build adapters that make our types satisfy Go interfaces?
- What's the performance cost of boundary crossings?

### Memory Management Strategies

**Areas to Explore**:
- Arena allocation for transient operations
- Object pooling for common node types
- Compact memory layouts for cache efficiency
- Custom allocators for persistent structures

**Questions**:
- Can we reduce pointer chasing with better layouts?
- Would a slab allocator help with fragmentation?
- Are there opportunities for inline storage of small collections?

### Formal Verification

**Possibilities**:
- Prove immutability properties for core data structures
- Use property-based testing to verify no observable mutation
- Static analysis tools to detect mutation violations

**Tools to Evaluate**:
- Go's race detector (for concurrent access)
- Property-based testing libraries (gopter, rapid)
- Custom static analysis with go/analysis

### Language Feature Investigation

**Monitor**:
- Go 2 proposals for immutability features
- Generic type constraints that might help
- Any future const or readonly proposals

**Current Limitations**:
- Generics don't support type constraints for immutability
- No way to express "this method doesn't mutate" in type system
- Method receivers can't be marked as requiring value semantics

## Recommended Priorities

1. **Immediate**: Implement basic persistent list and benchmark vs naive copying
2. **Short-term**: Survey existing persistent structure libraries, choose or fork one
3. **Medium-term**: Build comprehensive benchmarks with realistic Lisp workloads
4. **Ongoing**: Document discipline patterns and gotchas for team
5. **Future**: Investigate custom allocators if GC pressure becomes problematic

## Key Decision Points

- **Native persistent structures vs wrappers**: Impacts performance and complexity
- **Pure immutability vs pragmatic mutation**: Where to draw the line for practicality
- **Structural sharing granularity**: How much complexity for how much memory savings
- **Thread safety guarantees**: Immutable-by-default vs explicitly synchronized

## References to Investigate

- Clojure's persistent data structures implementation
- OCaml's approach to immutability on imperative runtime
- Haskell's lazy immutable structures
- Papers on HAMT and RRB-tree implementations
- Go's internal slice and map implementation details