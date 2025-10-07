---
number: 0032
title: "Zylisp Remote REPL Protocol - Design Document"
author: Unknown
created: 2025-10-06
updated: 2025-10-06
state: Final
supersedes: None
superseded-by: None
---

# Zylisp Remote REPL Protocol - Design Document

**Document Version:** 1.0
**Last Updated:** 2025-10-06
**Status:** Ready for Implementation

## Overview

This document specifies the design and implementation of a remote REPL protocol for Zylisp. The protocol enables interactive development by allowing clients to connect to a Zylisp REPL server over multiple transport mechanisms (in-process, Unix domain sockets, and TCP).

The design is inspired by Clojure's nREPL and Janet's spork/netrepl, but is tailored specifically for Zylisp and Go idioms, with no requirement for Clojure compatibility.

## Repository Context

**Repository:** `github.com/zylisp/repl`

**Existing Structure:**

```
.
├── client/
│   ├── client.go
│   └── client_test.go
├── server/
│   ├── server.go
│   └── server_test.go
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

**Implementation Strategy:** Extend and refactor existing code while maintaining backwards compatibility where possible. New protocol components should be added alongside existing implementations.

## Core Design Principles

1. **Interface-First Design:** Define interfaces for all major components to enable future extensibility
2. **Transport Agnostic:** Support in-process, Unix domain socket, and TCP with a unified API
3. **Errors as Data:** Follow Zylisp's philosophy - evaluation errors are returned as data, not exceptions
4. **Simple Session Model:** Connection = session (implicit sessions, no explicit session protocol)
5. **Synchronous Operations:** Start with synchronous eval; async can be added later via streaming
6. **Minimal Initial Feature Set:** Implement core operations first, add advanced features incrementally
7. **Leverage Standard Library:** Use Go's `net`, `encoding/json`, etc. - no custom implementations

## Architecture

### Layer 1: Protocol Message Format

**Location:** `protocol/message.go`

```go
package protocol

type Message struct {
    Op      string                 `json:"op"`               // Operation name
    ID      string                 `json:"id"`               // Unique message identifier
    Session string                 `json:"session,omitempty"` // Session ID (future use)
    Code    string                 `json:"code,omitempty"`    // Code to evaluate
    Status  []string               `json:"status,omitempty"`  // Status flags: "done", "error", etc.
    Value   interface{}            `json:"value,omitempty"`   // Evaluation result (including errors-as-data)
    Output  string                 `json:"output,omitempty"`  // Captured stdout/stderr
    ProtocolError string           `json:"protocol_error,omitempty"` // Protocol-level errors only
    Data    map[string]interface{} `json:"data,omitempty"`    // Additional operation-specific data
}
```

**Key Points:**

- `Value` contains Zylisp evaluation results, including error results (errors are data in Zylisp)
- `ProtocolError` is only for transport/protocol failures (malformed messages, connection issues)
- `Status` array indicates message state: `["done"]`, `["error"]`, `["interrupted"]`, etc.
- `Session` field reserved for future explicit session support

### Layer 2: Codec (Message Encoding)

**Location:** `protocol/codec.go`

```go
package protocol

import "io"

// Codec defines the interface for encoding/decoding messages
type Codec interface {
    Encode(msg *Message) error
    Decode(msg *Message) error
    Close() error
}

// NewCodec creates a codec based on format
func NewCodec(format string, rw io.ReadWriteCloser) (Codec, error)
```

**Implementations:**

1. **JSON Codec** (`protocol/json_codec.go`) - **IMPLEMENT FULLY**
   - Newline-delimited JSON using `encoding/json`
   - Use `json.NewEncoder()` and `json.NewDecoder()` - they handle framing automatically
   - Human-readable, easy to debug with telnet/netcat
   - This is the primary implementation for initial release

2. **MessagePack Codec** (`protocol/msgpack_codec.go`) - **PLACEHOLDER ONLY**
   - Add struct with panic("not implemented") methods
   - Future optimization for binary efficiency
   - Will use `github.com/vmihailenco/msgpack/v5` when implemented

### Layer 3: Server and Client Interfaces

**Location:** `repl.go` (root package)

```go
package repl

import "context"

// Result represents the outcome of a REPL operation
type Result struct {
    ID     string
    Value  interface{} // Zylisp evaluation result (success or error-as-data)
    Output string      // Captured stdout/stderr
    Status []string    // Operation status flags
}

// Server defines the REPL server interface
type Server interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Addr() string // Returns listening address
}

// Client defines the REPL client interface
type Client interface {
    Connect(ctx context.Context, addr string) error
    Eval(ctx context.Context, code string) (*Result, error)
    Close() error
}

// ServerConfig provides server configuration
type ServerConfig struct {
    Transport string // "in-process", "unix", "tcp"
    Addr      string // Address to bind (format depends on transport)
    Codec     string // "json" or "msgpack" (only for unix/tcp)
    Evaluator func(code string) (interface{}, string, error) // Zylisp evaluator
}

func NewServer(config ServerConfig) (Server, error)
func NewClient() Client
```

**Key Design Decisions:**

- **Connection = Session:** Each connection maintains its own evaluation context
- **Context-aware:** All operations accept `context.Context` for cancellation/timeout
- **Auto-detection:** Client detects transport from address format
- **Unified API:** Same interface regardless of transport

### Layer 4: Transport Implementations

#### 4.1 In-Process Transport

**Location:** `transport/inprocess/`

**Purpose:** Zero-overhead communication for testing and embedded use cases

**Implementation Strategy:**

```go
type Server struct {
    evaluator func(string) (interface{}, string, error)
    requests  chan *protocol.Message
    clients   map[string]chan *protocol.Message // client ID -> response channel
    mu        sync.RWMutex
    ctx       context.Context
    cancel    context.CancelFunc
}
```

- Use Go channels for message passing (no network stack)
- Each client gets a unique response channel
- Server goroutine processes requests serially per connection
- No codec needed (direct Message struct passing)

**Client Implementation:**

```go
type Client struct {
    server    *Server // Direct reference
    responses chan *protocol.Message
    clientID  string
}
```

#### 4.2 Unix Domain Socket Transport

**Location:** `transport/unix/`

**Purpose:** High-performance local IPC for development tools

**Implementation Strategy:**

```go
type Server struct {
    config   ServerConfig
    listener net.Listener
    conns    map[net.Conn]bool
    mu       sync.RWMutex
    ctx      context.Context
    cancel   context.CancelFunc
}
```

- Use `net.Listen("unix", path)` for listener
- Use `protocol.Codec` for message framing
- One goroutine per connection handles requests
- Graceful shutdown closes all connections

**Client Implementation:**

```go
type Client struct {
    conn  net.Conn
    codec protocol.Codec
    mu    sync.Mutex
}
```

- Use `net.Dial("unix", path)` for connection
- Synchronous request/response (mutex for thread safety)

#### 4.3 TCP Transport

**Location:** `transport/tcp/`

**Purpose:** Remote REPL access across network

**Implementation:** Nearly identical to Unix socket, but:

- Use `net.Listen("tcp", addr)`
- Use `net.Dial("tcp", addr)`
- Everything else is the same (thanks to `net.Conn` interface)

### Layer 5: Operations

**Location:** `operations/operations.go`

Operations define the actions clients can request. We borrow from nREPL's well-designed operation set, omitting only those that conflict with our design decisions (e.g., explicit session management).

#### Core Operations (MUST IMPLEMENT)

**1. `eval`** - Evaluate code

```json
Request:  {"op": "eval", "id": "1", "code": "(+ 1 2)"}
Response: {"id": "1", "value": 3, "status": ["done"]}
```

**2. `load-file`** - Load and evaluate a file

```json
Request:  {"op": "load-file", "id": "2", "file": "/path/to/file.zylisp", "file-path": "/path/to/file.zylisp", "file-name": "file.zylisp"}
Response: {"id": "2", "value": "...", "status": ["done"]}
```

**3. `describe`** - Server capabilities and info

```json
Request:  {"op": "describe", "id": "3"}
Response: {
  "id": "3",
  "status": ["done"],
  "data": {
    "versions": {"zylisp": "0.1.0", "protocol": "0.1.0"},
    "ops": ["eval", "load-file", "describe", "interrupt"],
    "transports": ["in-process", "unix", "tcp"]
  }
}
```

**4. `interrupt`** - Interrupt running evaluation

```json
Request:  {"op": "interrupt", "id": "4", "interrupt-id": "1"}
Response: {"id": "4", "status": ["done"]}
```

Note: Initial implementation can return "not-implemented" status

#### Future Operations (PLACEHOLDER ONLY)

Add operation stubs that return `{"status": ["error"], "protocol_error": "operation not implemented"}`:

- `complete` - Code completion
- `info` - Symbol documentation
- `eldoc` - Function signature hints
- `lookup` - Symbol definition location
- `stdin` - Send input to running process
- `ls-sessions` - List active sessions (for future explicit session support)
- `clone` - Clone a session (for future explicit session support)
- `close` - Close a session (for future explicit session support)

### Layer 6: Universal Client

**Location:** `client/universal.go`

Implement a client that auto-detects transport and delegates to the appropriate implementation:

```go
type UniversalClient struct {
    impl Client // Actual transport-specific client
}

func (c *UniversalClient) Connect(ctx context.Context, addr string) error {
    transport := detectTransport(addr)
    switch transport {
    case "in-process":
        c.impl = inprocess.NewClient()
    case "unix":
        c.impl = unix.NewClient()
    case "tcp":
        c.impl = tcp.NewClient()
    }
    return c.impl.Connect(ctx, addr)
}

func detectTransport(addr string) string {
    // "in-process" or "" -> in-process
    // Starts with "/" or "." -> unix
    // "unix://path" -> unix
    // "tcp://host:port" -> tcp
    // "host:port" -> tcp (default)
}
```

## Implementation Plan

### Phase 1: Core Protocol (Weeks 1-2)

1. **Protocol Package:**
   - [ ] Implement `protocol/message.go` with Message struct
   - [ ] Implement `protocol/codec.go` with Codec interface
   - [ ] Implement `protocol/json_codec.go` fully (using `encoding/json`)
   - [ ] Add `protocol/msgpack_codec.go` as placeholder (panic stubs)
   - [ ] Write unit tests for JSON codec

2. **Root Package Interfaces:**
   - [ ] Define `Server`, `Client`, `Result` interfaces in `repl.go`
   - [ ] Define `ServerConfig` struct
   - [ ] Implement `NewServer()` factory
   - [ ] Implement `NewClient()` factory

### Phase 2: In-Process Transport (Week 2)

3. **In-Process Implementation:**
   - [ ] Implement `transport/inprocess/server.go`
   - [ ] Implement `transport/inprocess/client.go`
   - [ ] Write comprehensive tests (easiest to test, no networking)
   - [ ] Verify error-as-data handling works correctly

### Phase 3: Socket Transports (Weeks 3-4)

4. **Unix Domain Socket:**
   - [ ] Implement `transport/unix/server.go`
   - [ ] Implement `transport/unix/client.go`
   - [ ] Integration tests with real Unix sockets
   - [ ] Test graceful shutdown and cleanup

5. **TCP Transport:**
   - [ ] Implement `transport/tcp/server.go`
   - [ ] Implement `transport/tcp/client.go`
   - [ ] Integration tests with TCP connections
   - [ ] Test concurrent clients

### Phase 4: Operations and Dispatch (Week 4)

6. **Operations Layer:**
   - [ ] Implement `operations/operations.go` with operation handlers
   - [ ] Implement `eval` operation
   - [ ] Implement `load-file` operation
   - [ ] Implement `describe` operation
   - [ ] Add `interrupt` operation stub
   - [ ] Add future operation stubs
   - [ ] Wire operations into server implementations

### Phase 5: Universal Client (Week 5)

7. **Client Integration:**
   - [ ] Implement `client/universal.go` with transport detection
   - [ ] Update existing `client/client.go` to use new system
   - [ ] Maintain backwards compatibility if existing tests depend on it
   - [ ] End-to-end tests across all transports

### Phase 6: Documentation and Polish (Week 6)

8. **Documentation:**
   - [ ] Update `README.md` with protocol specification
   - [ ] Add usage examples for each transport
   - [ ] Document message format and operations
   - [ ] Add architecture diagrams
   - [ ] Document error handling conventions

9. **Testing and Cleanup:**
   - [ ] Comprehensive integration tests
   - [ ] Benchmark tests (compare transports)
   - [ ] Error path testing
   - [ ] Code review and refactoring

## Protocol Specification

### Message Flow

**Successful Evaluation:**

```
Client -> Server: {"op":"eval", "id":"1", "code":"(+ 1 2)"}
Server -> Client: {"id":"1", "value":3, "output":"", "status":["done"]}
```

**Zylisp Error (Error as Data):**

```
Client -> Server: {"op":"eval", "id":"2", "code":"(/ 1 0)"}
Server -> Client: {
  "id":"2",
  "value": {"error": "division by zero", "type": "arithmetic-error"},
  "status":["done"]
}
```

Note: This is still a successful protocol exchange - the error is in the value

**Protocol Error:**

```
Client -> Server: {"op":"invalid", "id":"3"}
Server -> Client: {
  "id":"3",
  "protocol_error": "unknown operation: invalid",
  "status":["error"]
}
```

**With Output:**

```
Client -> Server: {"op":"eval", "id":"4", "code":"(println \"hello\")"}
Server -> Client: {
  "id":"4",
  "value": null,
  "output": "hello\n",
  "status":["done"]
}
```

### Status Values

- `["done"]` - Operation completed successfully
- `["error"]` - Protocol-level error (see `protocol_error` field)
- `["interrupted"]` - Evaluation was interrupted
- `["need-input"]` - Requires stdin (future)
- `["session-idle"]` - Session has no running evaluation (future)

### Address Formats

| Format | Transport | Example |
|--------|-----------|---------|
| `""` or `"in-process"` | In-process | `"in-process"` |
| Path starting with `/` or `.` | Unix | `"/tmp/zylisp.sock"` |
| `unix://path` | Unix | `"unix:///tmp/zylisp.sock"` |
| `tcp://host:port` | TCP | `"tcp://localhost:5555"` |
| `host:port` | TCP (default) | `"localhost:5555"` |

## Testing Strategy

### Unit Tests

- Codec encode/decode round-trips
- Message validation
- Transport detection logic
- Operation handlers in isolation

### Integration Tests

- Full client-server workflows for each transport
- Concurrent client connections
- Error propagation (both protocol and Zylisp errors)
- Graceful shutdown
- Connection cleanup

### Benchmark Tests

- Compare in-process vs Unix vs TCP latency
- Message throughput
- Codec performance (JSON baseline for future MessagePack comparison)

## Future Enhancements

These should NOT be implemented initially, but the architecture should support them:

1. **Explicit Session Management:**
   - Add `clone`, `close`, `ls-sessions` operations
   - Multiple sessions per connection
   - Session persistence across reconnections

2. **Streaming Responses:**
   - Multiple response messages per request
   - Separate stdout/stderr streaming
   - Progress updates for long operations

3. **MessagePack Codec:**
   - Implement `protocol/msgpack_codec.go`
   - Performance benchmarks vs JSON
   - Binary efficiency for large data

4. **Advanced Operations:**
   - Code completion
   - Symbol documentation lookup
   - Definition location (jump-to-def)
   - Namespace inspection

5. **Security:**
   - TLS support for TCP transport
   - Authentication/authorization
   - Permission-based operation restrictions

6. **Middleware Architecture:**
   - Pluggable middleware for cross-cutting concerns
   - Logging, metrics, rate limiting
   - Custom operation handlers

## Error Handling Philosophy

**Two Error Categories:**

1. **Protocol Errors** (Go `error`, set `protocol_error` field):
   - Connection failures
   - Malformed messages
   - Unknown operations
   - Codec failures
   - These are returned via Go's `error` return value

2. **Zylisp Evaluation Results** (returned as data in `value` field):
   - Type errors
   - Runtime errors
   - Arithmetic errors
   - These are NOT Go errors - they're successful evaluations that produced error values

**Example:**

```go
// Protocol error - connection failed
result, err := client.Eval(ctx, "(+ 1 2)")
if err != nil {
    // This is a protocol/transport error
    return err
}

// Zylisp error - evaluation produced an error value
// err is nil here - the protocol worked fine
if result.Value is an error value {
    // Handle Zylisp error-as-data
}
```

## Integration with Existing Code

The existing `client/` and `server/` packages should be:

1. **Analyzed** for any APIs that external code depends on
2. **Refactored** to use the new transport system internally
3. **Maintained** for backwards compatibility if needed
4. **Deprecated** gracefully if new APIs are better

Suggested approach:

- Keep existing `client.go` and `server.go` as high-level wrappers
- Implement new system in parallel
- Migrate existing tests incrementally
- Add deprecation notices if breaking changes needed

## Success Criteria

The implementation is complete when:

1. ✅ All three transports (in-process, Unix, TCP) work correctly
2. ✅ JSON codec is fully implemented and tested
3. ✅ Core operations (`eval`, `load-file`, `describe`) work
4. ✅ Errors-as-data are correctly propagated
5. ✅ Protocol errors are clearly distinguished from Zylisp errors
6. ✅ Client can auto-detect and connect to any transport
7. ✅ Comprehensive tests cover all major paths
8. ✅ Documentation explains usage and architecture
9. ✅ Existing code is either integrated or deprecated cleanly
10. ✅ Code is ready for MessagePack and advanced features (interfaces in place)

## Example Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/zylisp/repl"
)

func main() {
    // Start a TCP server
    server, _ := repl.NewServer(repl.ServerConfig{
        Transport: "tcp",
        Addr:      ":5555",
        Codec:     "json",
        Evaluator: myZylispEval,
    })
    go server.Start(context.Background())

    // Connect a client
    client := repl.NewClient()
    client.Connect(context.Background(), "localhost:5555")

    // Evaluate code
    result, err := client.Eval(context.Background(), "(+ 1 2)")
    if err != nil {
        // Protocol error
        panic(err)
    }

    // Check if Zylisp evaluation succeeded or returned error-as-data
    fmt.Printf("Result: %v\n", result.Value)
}
```

## Notes for Implementation

- Use `context.Context` throughout for proper cancellation
- Leverage `net.Conn` interface - it works for both Unix and TCP
- Use `sync.Mutex` for thread-safe client operations
- Use `sync.RWMutex` for server connection tracking
- Implement graceful shutdown with deadline contexts
- Log errors appropriately but don't over-log
- Keep code idiomatic Go - follow standard patterns
- Write tests as you go - don't leave them for the end
- Use table-driven tests where appropriate
- Keep functions small and focused

## Questions for Future Discussion

These don't need answers now, but should be considered during implementation:

1. Should we support multiple concurrent eval requests per connection?
2. How should we handle extremely long-running evaluations?
3. Should there be a maximum message size?
4. Should we support server-initiated messages (push notifications)?
5. How should we handle client disconnection during evaluation?
