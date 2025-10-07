---
number: 0033
title: "Zylisp CLI Migration to New REPL Protocol"
author: Unknown
created: 2025-10-06
updated: 2025-10-06
state: Final
supersedes: None
superseded-by: None
---

# Zylisp CLI Migration to New REPL Protocol

**Document Version:** 1.0
**Date:** 2025-10-06
**Status:** Ready for Implementation

## Context

The Zylisp CLI tool (`github.com/zylisp/cli`) currently uses an MVP implementation with direct, in-memory server/client communication. A new, production-ready REPL protocol has been implemented in `github.com/zylisp/repl` that supports:

- Multiple transports (in-process, Unix domain socket, TCP)
- Structured message protocol with JSON encoding
- Proper error handling (errors-as-data vs protocol errors)
- Operation-based architecture (`eval`, `load-file`, `describe`, etc.)
- Context-aware operations with cancellation support

## Current CLI Architecture

**File Structure:**

```
github.com/zylisp/cli/
├── main.go              # CLI entry point and REPL loop
├── integration_test.go  # Integration tests
├── EXAMPLES.md          # Zylisp language examples
└── README.md            # Basic documentation
```

**Current Implementation Analysis:**

The existing `main.go`:

- ✅ Clean REPL interface with banner and prompt
- ✅ Good command handling (`:reset`, `:help`, `exit`, `quit`)
- ✅ Simple scanner-based input loop
- ❌ Uses old `server.NewServer()` and `client.NewClient(srv)` API
- ❌ No support for remote connections
- ❌ No transport selection
- ❌ Limited error handling
- ❌ No context/cancellation support

The existing integration tests:

- ✅ Good coverage of basic operations
- ✅ Stateful testing
- ✅ Complex examples (factorial, list processing)
- ❌ Use old server/client API
- ❌ Test only in-process mode
- ❌ No protocol-level testing

## Migration Strategy

**Recommended Approach: Evolutionary Refactoring**

Rather than a complete rewrite, we should:

1. **Preserve the Good Parts:**
   - Keep the existing REPL UX (banner, prompt, commands)
   - Maintain the command handling structure
   - Keep the clean separation of concerns

2. **Modernize the Foundation:**
   - Switch to new `github.com/zylisp/repl` protocol
   - Add transport configuration
   - Improve error handling
   - Add context support

3. **Enhance Capabilities:**
   - Support remote REPL connections
   - Add configuration options (flags)
   - Support both client and server modes
   - Improve error messages

## Implementation Plan

### Phase 1: Update Dependencies and Imports

**File:** `main.go`

Update imports to use the new protocol:

```go
import (
    "context"
    "bufio"
    "fmt"
    "os"
    "os/signal"
    "strings"
    "syscall"
    "time"

    "github.com/zylisp/repl"
)
```

Add flag parsing for configuration:

```go
import "flag"

var (
    mode      = flag.String("mode", "local", "Mode: 'local', 'server', or 'client'")
    transport = flag.String("transport", "in-process", "Transport: 'in-process', 'unix', or 'tcp'")
    addr      = flag.String("addr", "", "Server address (for server/client modes)")
    codec     = flag.String("codec", "json", "Codec: 'json' or 'msgpack'")
)
```

### Phase 2: Implement Server Mode

Add a new function to run as a REPL server:

```go
func runServer(ctx context.Context) error {
    config := repl.ServerConfig{
        Transport: *transport,
        Addr:      *addr,
        Codec:     *codec,
        Evaluator: evaluateZylisp, // Implement this function
    }

    server, err := repl.NewServer(config)
    if err != nil {
        return fmt.Errorf("failed to create server: %w", err)
    }

    fmt.Printf("Starting Zylisp REPL server on %s (%s)\n", server.Addr(), *transport)

    return server.Start(ctx)
}

// evaluateZylisp is the bridge to the actual Zylisp evaluator
// This function needs to be implemented to call into the Zylisp interpreter
func evaluateZylisp(code string) (interface{}, string, error) {
    // TODO: Call actual Zylisp evaluator
    // For now, return placeholder
    return nil, "", fmt.Errorf("evaluator not yet implemented")
}
```

**Key Design Points:**

- Server mode starts a REPL server and waits for connections
- Uses context for graceful shutdown
- Supports all three transport types
- The `evaluator` function bridges to the actual Zylisp interpreter

### Phase 3: Update Local Mode (In-Process REPL)

Refactor the existing REPL loop to use the new protocol:

```go
func runLocal(ctx context.Context) error {
    // Create in-process server
    config := repl.ServerConfig{
        Transport: "in-process",
        Evaluator: evaluateZylisp,
    }

    server, err := repl.NewServer(config)
    if err != nil {
        return fmt.Errorf("failed to create server: %w", err)
    }

    // Start server in background
    serverCtx, serverCancel := context.WithCancel(ctx)
    defer serverCancel()

    go func() {
        if err := server.Start(serverCtx); err != nil {
            fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
        }
    }()

    // Create client connected to in-process server
    client := repl.NewClient()
    if err := client.Connect(ctx, "in-process"); err != nil {
        return fmt.Errorf("failed to connect client: %w", err)
    }
    defer client.Close()

    // Run REPL loop
    return replLoop(ctx, client)
}
```

### Phase 4: Implement Client Mode

Add a function to connect to a remote REPL server:

```go
func runClient(ctx context.Context) error {
    if *addr == "" {
        return fmt.Errorf("--addr required in client mode")
    }

    client := repl.NewClient()

    fmt.Printf("Connecting to Zylisp REPL at %s...\n", *addr)

    if err := client.Connect(ctx, *addr); err != nil {
        return fmt.Errorf("failed to connect: %w", err)
    }
    defer client.Close()

    fmt.Println("Connected!")

    // Run REPL loop
    return replLoop(ctx, client)
}
```

### Phase 5: Refactor REPL Loop

Extract the REPL loop into a reusable function that works with any client:

```go
func replLoop(ctx context.Context, client repl.Client) error {
    fmt.Print(banner)

    scanner := bufio.NewScanner(os.Stdin)

    for {
        fmt.Print("> ")

        if !scanner.Scan() {
            break
        }

        line := strings.TrimSpace(scanner.Text())

        if line == "" {
            continue
        }

        // Handle special commands
        if shouldExit := handleCommand(line, client); shouldExit {
            return nil
        }

        // Create context with timeout for evaluation
        evalCtx, cancel := context.WithTimeout(ctx, 30*time.Second)

        // Evaluate expression
        result, err := client.Eval(evalCtx, line)
        cancel()

        if err != nil {
            // Protocol error
            fmt.Printf("Protocol Error: %v\n", err)
            continue
        }

        // Check for output
        if result.Output != "" {
            fmt.Print(result.Output)
        }

        // Display result value
        // Note: In Zylisp, errors are data, so result.Value might contain an error
        fmt.Println(formatValue(result.Value))
    }

    if err := scanner.Err(); err != nil {
        return fmt.Errorf("scanner error: %w", err)
    }

    return nil
}

// formatValue converts the result value to a display string
func formatValue(value interface{}) string {
    // TODO: Implement proper Zylisp value formatting
    // This should handle different Zylisp types appropriately
    // For now, use basic formatting
    return fmt.Sprintf("%v", value)
}
```

### Phase 6: Update Command Handling

Update the command handler to work with the new client interface:

```go
// handleCommand processes special REPL commands
// Returns true if the REPL should exit
func handleCommand(line string, client repl.Client) bool {
    switch line {
    case "exit", "quit":
        fmt.Println("\nGoodbye!")
        return true

    case ":reset":
        // TODO: Implement reset via protocol
        // For now, inform user that reset requires reconnection
        fmt.Println("Reset not yet implemented - please reconnect")
        return false

    case ":help":
        showHelp()
        return false

    case ":info":
        showServerInfo(client)
        return false

    default:
        return false
    }
}

// showServerInfo queries server capabilities
func showServerInfo(client repl.Client) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // TODO: Implement describe operation call
    // This requires the client to support the describe operation
    fmt.Println("Server info not yet implemented")
}
```

### Phase 7: Wire Everything Together

Update the `main()` function to route to the appropriate mode:

```go
func main() {
    flag.Parse()

    // Set up context with signal handling
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle interrupt signals gracefully
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

    go func() {
        <-sigCh
        fmt.Println("\nShutting down...")
        cancel()
    }()

    // Run appropriate mode
    var err error
    switch *mode {
    case "local":
        err = runLocal(ctx)
    case "server":
        err = runServer(ctx)
    case "client":
        err = runClient(ctx)
    default:
        fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", *mode)
        flag.Usage()
        os.Exit(1)
    }

    if err != nil && err != context.Canceled {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

### Phase 8: Update Integration Tests

Migrate integration tests to use the new protocol:

```go
package main

import (
    "context"
    "fmt"
    "testing"
    "time"

    "github.com/zylisp/repl"
)

func setupTestREPL(t *testing.T) (repl.Client, func()) {
    t.Helper()

    // Create in-process server
    config := repl.ServerConfig{
        Transport: "in-process",
        Evaluator: evaluateZylisp,
    }

    server, err := repl.NewServer(config)
    if err != nil {
        t.Fatalf("failed to create server: %v", err)
    }

    // Start server
    ctx, cancel := context.WithCancel(context.Background())
    go func() {
        if err := server.Start(ctx); err != nil {
            t.Logf("server error: %v", err)
        }
    }()

    // Create and connect client
    client := repl.NewClient()
    if err := client.Connect(ctx, "in-process"); err != nil {
        cancel()
        t.Fatalf("failed to connect: %v", err)
    }

    cleanup := func() {
        client.Close()
        cancel()
    }

    return client, cleanup
}

func TestIntegrationBasic(t *testing.T) {
    client, cleanup := setupTestREPL(t)
    defer cleanup()

    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"number", "42", "42"},
        {"add", "(+ 1 2)", "3"},
        {"nested", "(+ (* 2 3) 4)", "10"},
    }

    ctx := context.Background()

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := client.Eval(ctx, tt.input)
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }

            resultStr := formatValue(result.Value)
            if resultStr != tt.expected {
                t.Errorf("got %q, want %q", resultStr, tt.expected)
            }
        })
    }
}

// Add similar updates for other test functions...
```

## New Features to Add

### 1. Configuration File Support

Add support for a `~/.zylisprc` configuration file:

```go
type Config struct {
    DefaultTransport string
    DefaultAddr      string
    DefaultCodec     string
    History          bool
    HistoryFile      string
}

func loadConfig() (*Config, error) {
    // TODO: Load from ~/.zylisprc
    return &Config{
        DefaultTransport: "in-process",
        DefaultCodec:     "json",
        History:          true,
        HistoryFile:      "~/.zylisp_history",
    }, nil
}
```

### 2. Command History

Add readline support for command history and editing:

```go
// Consider using github.com/chzyer/readline
// This provides a much better REPL experience
```

### 3. Startup Script

Support loading a startup script on REPL launch:

```go
func loadStartupScript(client repl.Client) error {
    startupPath := os.ExpandEnv("$HOME/.zylisp_startup")

    if _, err := os.Stat(startupPath); os.IsNotExist(err) {
        return nil // No startup script
    }

    content, err := os.ReadFile(startupPath)
    if err != nil {
        return err
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    _, err = client.Eval(ctx, string(content))
    return err
}
```

### 4. Multi-line Input

Add support for multi-line expressions:

```go
func readExpression(scanner *bufio.Scanner) (string, error) {
    var expr strings.Builder
    var parenCount int

    for {
        if expr.Len() > 0 {
            fmt.Print("... ")
        } else {
            fmt.Print("> ")
        }

        if !scanner.Scan() {
            return "", scanner.Err()
        }

        line := scanner.Text()
        expr.WriteString(line)
        expr.WriteString("\n")

        // Count parentheses
        for _, ch := range line {
            if ch == '(' {
                parenCount++
            } else if ch == ')' {
                parenCount--
            }
        }

        // If balanced, we're done
        if parenCount == 0 {
            return expr.String(), nil
        }

        // If negative, syntax error
        if parenCount < 0 {
            return "", fmt.Errorf("unmatched closing parenthesis")
        }
    }
}
```

## Updated Help Text

Update the help command to reflect new capabilities:

```go
func showHelp() {
    fmt.Print(`
Zylisp REPL - Command Reference

REPL Commands:
  exit, quit    - Exit the REPL
  :reset        - Reset the environment (reconnect required)
  :help         - Show this help message
  :info         - Show server information

Language Features:
  Special Forms: define, lambda, if, quote
  Arithmetic:    +, -, *, /
  Comparison:    =, <, >, <=, >=
  Lists:         list, car, cdr, cons
  Predicates:    number?, symbol?, list?, null?

Connection Modes:
  Local mode:  Built-in REPL (default)
    $ zylisp

  Server mode: Start a REPL server
    $ zylisp --mode=server --transport=tcp --addr=:5555

  Client mode: Connect to a remote REPL
    $ zylisp --mode=client --addr=localhost:5555

Examples:
  > (+ 1 2)
  3

  > (define square (lambda (x) (* x x)))
  <function>

  > (square 5)
  25

For more examples, see EXAMPLES.md
`)
}
```

## Implementation Checklist

- [ ] Update imports and add flag parsing
- [ ] Implement `evaluateZylisp()` bridge function
- [ ] Implement `runServer()` for server mode
- [ ] Refactor existing code into `runLocal()` for local mode
- [ ] Implement `runClient()` for client mode
- [ ] Extract REPL loop into `replLoop()` function
- [ ] Update command handling to work with new client
- [ ] Update `main()` to route between modes
- [ ] Migrate all integration tests to new protocol
- [ ] Add context and signal handling
- [ ] Implement proper value formatting
- [ ] Update help text
- [ ] Update README.md with new usage instructions
- [ ] Add examples for remote REPL usage
- [ ] Test all three modes (local, server, client)
- [ ] Test all three transports (in-process, unix, tcp)

## Usage Examples After Migration

### Local REPL (Default)

```bash
$ zylisp
# Uses in-process transport, same as before
```

### Start a REPL Server

```bash
# TCP server
$ zylisp --mode=server --transport=tcp --addr=:5555

# Unix socket server
$ zylisp --mode=server --transport=unix --addr=/tmp/zylisp.sock
```

### Connect to Remote REPL

```bash
# Connect via TCP
$ zylisp --mode=client --addr=localhost:5555

# Connect via Unix socket
$ zylisp --mode=client --addr=/tmp/zylisp.sock
```

### Advanced Usage

```bash
# Server with MessagePack (when implemented)
$ zylisp --mode=server --transport=tcp --addr=:5555 --codec=msgpack

# Client connecting to MessagePack server
$ zylisp --mode=client --addr=localhost:5555 --codec=msgpack
```

## Testing Strategy

### Unit Tests

- Test command parsing
- Test value formatting
- Test configuration loading
- Test multi-line input handling

### Integration Tests

- Migrate all existing tests to new protocol
- Test each transport type
- Test server/client mode interaction
- Test error handling (both protocol and Zylisp errors)
- Test graceful shutdown

### Manual Testing

1. Start local REPL and verify basic operations
2. Start server in one terminal, connect client in another
3. Test Unix socket communication
4. Test TCP over network (different machines)
5. Test signal handling (Ctrl-C)
6. Test long-running evaluations with timeout

## Migration Notes

### Breaking Changes

- Old `server.NewServer()` API is replaced
- Old `client.NewClient(srv)` API is replaced
- Tests need to be updated to use context

### Backwards Compatibility

- REPL UX remains the same
- Default behavior (local mode) works as before
- All examples in EXAMPLES.md continue to work

### Performance Considerations

- In-process mode should have similar performance to before
- Remote modes have network overhead
- Consider timeout values for remote operations

## Future Enhancements

After basic migration:

1. Add readline/libedit support for better input editing
2. Implement command history persistence
3. Add tab completion (requires protocol support)
4. Add syntax highlighting
5. Support `:load` command to load files
6. Add `:doc` command for documentation lookup
7. Implement proper `:reset` via protocol
8. Add REPL transcript recording
9. Support configuration profiles

## Success Criteria

The migration is complete when:

- ✅ All three modes work correctly (local, server, client)
- ✅ All three transports are supported
- ✅ All existing integration tests pass
- ✅ Help text is updated
- ✅ README has clear usage instructions
- ✅ Error handling properly distinguishes protocol vs Zylisp errors
- ✅ Graceful shutdown works with Ctrl-C
- ✅ The user experience for local mode is unchanged
- ✅ Remote REPL capabilities are documented

## Questions to Resolve During Implementation

1. **Evaluator Bridge**: How exactly does the CLI call into the Zylisp interpreter? We need the actual function signature.

2. **Value Formatting**: What's the proper way to format Zylisp values for display? Do we have a `String()` method?

3. **Error Types**: How are Zylisp errors represented as data? Is there a specific struct type?

4. **Reset Operation**: Should `:reset` be a protocol operation, or should it just tell the user to reconnect?

5. **Session State**: Does each connection need to maintain isolated state, or is there shared global state?
