---
number: 0038
title: "Zylisp: Project/Module Structure and Build System Design"
author: Duncan McGreggor
created: 2025-10-10
updated: 2025-10-10
state: Draft
supersedes: None
superseded-by: None
---

# Zylisp: Project/Module Structure and Build System Design

**Document Number**: 0036
**Title**: Project Structure, Module System, and Build Pipeline
**Author**: Duncan McGreggor
**Created**: 2025-10-10
**Updated**: 2025-10-10
**State**: Draft
**Supersedes**: None
**Superseded-by**: None

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [The Fundamental Question](#the-fundamental-question)
3. [Design Exploration](#design-exploration)
   - [Traditional Build Approaches](#traditional-build-approaches)
   - [The Key Insight](#the-key-insight)
4. [The Unified Model](#the-unified-model)
5. [Project Definition: `defproject`](#project-definition-defproject)
6. [Module System: `defmodule`](#module-system-defmodule)
7. [Command System: `defcmd`](#command-system-defcmd)
8. [File System Representation](#file-system-representation)
9. [Build System Integration](#build-system-integration)
10. [REPL Integration](#repl-integration)
11. [Implementation Plan](#implementation-plan)
12. [Open Questions](#open-questions)
13. [Conclusion](#conclusion)
14. [Appendix: Complete Examples](#appendix-complete-examples)

---

## Executive Summary

This document defines Zylisp's project structure, module system, and build pipeline. The design is based on a fundamental insight: **with the right syntax, we can represent an entire project as a single data structure that can be viewed and manipulated in multiple ways**.

**Key Design Decisions**:

1. **Single-file project definition** - `project.zl` as source of truth
2. **Explode/collapse transformations** - Convert between single-file and multi-file layouts
3. **Module system** - `defmodule` for library code
4. **Command system** - `defcmd` for executable binaries
5. **Unified syntax** - Same forms work in files and REPL
6. **Leverage Go tooling** - Generate Go code, use `go build` for linking

**Benefits**:

- ✅ Homoiconic project structure (code as data applies to projects)
- ✅ Flexible development workflow (single or multi-file)
- ✅ Natural REPL integration
- ✅ Gradual complexity scaling
- ✅ Standard Go interoperability

---

## The Fundamental Question

During design discussions, we condensed the project structure problem to one question with two sides:

> **How do I want to represent Zylisp projects?**
>
> - On the file system?
> - In the REPL?

The insight: These aren't separate problems. With Lisp's homoiconicity, we can define a project as **data** that has multiple views:

- **Spatial view**: Multiple files on disk
- **Temporal view**: Loaded into REPL session
- **Canonical view**: Single `project.zl` definition

All three are the same underlying data structure.

---

## Design Exploration

### Traditional Build Approaches

We initially explored four strategies for building large Zylisp projects:

#### Strategy 1: Single-Pass Whole-Program Compilation

Compile all `.zl` files into one Go package.

**Pros**:

- Simple mental model
- Easy cross-file optimization

**Cons**:

- Slow builds (recompile everything)
- No incremental compilation
- Large intermediate files

**Best for**: Small projects (<10k LOC)

#### Strategy 2: Package-Based Compilation

Map Zylisp packages to Go packages, compile independently.

```
myproject/
├── src/main.zl           → out/main/main.go
├── http/server.zl        → out/http/server.go
└── db/queries.zl         → out/db/queries.go
```

**Pros**:

- Incremental compilation
- Leverages Go's build system
- Scales to large projects

**Cons**:

- More complex orchestration
- Need package metadata system

**Best for**: Large projects with clear modules

#### Strategy 3: Hybrid with Caching

Intelligent caching with dependency tracking.

**Pros**:

- Fast incremental builds
- Good for development iteration

**Cons**:

- Cache invalidation complexity
- Cache management overhead

**Best for**: Active development

#### Strategy 4: Ahead-of-Time (AOT) Compilation

Pre-compile to Go, check Go code into repo.

**Pros**:

- No compiler in production
- Users build with `go build`
- Easy debugging

**Cons**:

- Repo bloat
- Manual regeneration
- Merge conflicts

**Best for**: Libraries for non-Zylisp users

### The Key Insight

While exploring these strategies, we realized: **The file system layout shouldn't dictate the project structure. The project structure is data, and the file system is just one view of it.**

This led to the unified model:

**With the right syntax, we could define a complete, very large project in a single file:**

```lisp
(defproject myproj
  (defmodule mymod1
    :export [...]
    (defconst ...)
    (defvar ...)
    ...)
  (defmodule mymod2
    ...)
  (defcmd server
    ...))
```

The Zylisp tooling can then:

- **Explode** this to multiple files for editing
- **Collapse** multiple files back to single definition
- **Build** from either representation
- **Load** into REPL as unified structure

---

## The Unified Model

### Core Concept

A Zylisp project is **a data structure** that can be represented in three equivalent ways:

#### 1. Canonical Form: Single File

```lisp
;; project.zl - The source of truth
(defproject myapp
  :version "0.1.0"
  :description "My awesome application"

  (defmodule http
    :export [new-server start]
    (deffunc new-server [port] ...))

  (defmodule db
    :export [connect query]
    (deffunc connect [dsn] ...))

  (defcmd server
    (import myapp.http)
    (http/start (http/new-server 8080))))
```

#### 2. Exploded Form: Multiple Files

```
myapp/
├── project.zl          # Metadata and structure
├── http.zl             # Generated from (defmodule http ...)
├── db.zl               # Generated from (defmodule db ...)
└── cmd/
    └── server/
        └── main.zl     # Generated from (defcmd server ...)
```

#### 3. REPL Form: Loaded Namespace

```lisp
zylisp> (load-project "project.zl")
Project: myapp v0.1.0
Modules: http, db
Commands: server

zylisp> (in-module 'http)
http> (new-server 8080)
=> #<Server :port 8080>
```

### Transformations

```
        explode
project.zl ←→ multiple .zl files
        collapse

        load
project.zl → REPL session

        build
project.zl → Go packages → binary
```

### Beautiful Symmetry

The **representation is the same**—it's just whether you're viewing it:

- **Spatially** (files on disk)
- **Temporally** (in REPL)
- **Canonically** (project definition)

---

## Project Definition: `defproject`

### Basic Syntax

```lisp
(defproject <name>
  ;; Metadata
  :version <string>
  :description <string>
  :go-version <string>
  :dependencies [<package> <version> ...]

  ;; Module definitions
  (defmodule <name> ...)
  (defmodule <name> ...)

  ;; Command definitions
  (defcmd <name> ...)
  (defcmd <name> ...))
```

### Complete Example

```lisp
(defproject myapp
  :version "0.1.0"
  :description "A complete web application"
  :go-version "1.21"
  :dependencies [
    "github.com/gorilla/mux" "v1.8.0"
    "github.com/lib/pq" "v1.10.0"]

  ;; Shared library modules
  (defmodule http
    :file "internal/http.zl"
    :export [new-server start])

  (defmodule db
    :file "internal/db.zl"
    :export [connect query])

  (defmodule config
    :file "internal/config.zl"
    :export [load-config])

  ;; Executable commands
  (defcmd server
    :description "HTTP API server"
    :output "bin/server"

    (import myapp.http)
    (import myapp.config)

    (let [cfg (config/load-config "config.toml")
          srv (http/new-server (. cfg port))]
      (http/start srv)))

  (defcmd worker
    :description "Background job processor"
    :output "bin/worker"

    (import myapp.db)
    (import myapp.config)

    (let [cfg (config/load-config "config.toml")
          conn (db/connect (. cfg database-url))]
      (process-jobs conn))))
```

### Metadata Fields

| Field | Type | Description |
|-------|------|-------------|
| `:version` | string | Semantic version |
| `:description` | string | Project description |
| `:go-version` | string | Required Go version |
| `:dependencies` | list | External Go packages |
| `:author` | string | Author name |
| `:license` | string | License identifier |
| `:repository` | string | Repository URL |

---

## Module System: `defmodule`

### Purpose

Modules are **library code** - reusable functionality that can be imported by other modules or commands.

### Syntax Options

#### Option 1: Inline Definition

```lisp
(defmodule http
  :export [new-server start stop]
  :doc "HTTP server implementation"

  (import "net/http")
  (import myapp.config)

  (deftype Server
    {:port int
     :router *http.ServeMux})

  (deffunc new-server [port]
    (:args int)
    (:return *Server)
    ...)

  (deffunc start [srv]
    (:args *Server)
    (:return error)
    ...))
```

#### Option 2: External File Reference

```lisp
(defmodule http
  :file "internal/http/server.zl"
  :export [new-server start stop]
  :doc "HTTP server implementation")
```

#### Option 3: Hybrid (Metadata + File)

```lisp
(defmodule http
  :file "internal/http.zl"
  :export [new-server start stop]
  :doc "HTTP server implementation"
  :tests "tests/http_test.zl"
  :author "Duncan McGreggor")
```

### Module File Format

When exploded to a file:

```lisp
;; http.zl
(module http
  :export [new-server start stop]
  :doc "HTTP server implementation")

;; Module body
(import "net/http")

(deftype Server ...)
(deffunc new-server [...] ...)
(deffunc start [...] ...)
```

### Import Resolution

```lisp
;; Internal module (same project)
(import myapp.http)

;; External Zylisp package
(import github.com/zylisp/stdlib.strings)

;; Go package
(import "net/http")

;; Aliased import
(import [myapp.http :as h])
(import ["database/sql" :as sql])
```

### Export Control

Only explicitly exported symbols are visible outside the module:

```lisp
(defmodule utils
  :export [public-func public-var]

  (deffunc public-func []
    (:return string)
    (private-helper "data"))

  (deffunc private-helper [data]    ; Not exported
    (:args string)
    (:return string)
    ...)

  (defvar public-var 42)
  (defvar private-var 13))          ; Not exported
```

---

## Command System: `defcmd`

### Purpose

Commands are **executable binaries** - programs with a `main()` entry point that can be compiled to standalone executables. They represent Go's `cmd/` convention.

### The Go `cmd/` Convention

Go projects typically structure executables as:

```
myproject/
├── cmd/
│   ├── server/
│   │   └── main.go
│   ├── worker/
│   │   └── main.go
│   └── cli/
│       └── main.go
└── internal/
    └── ...
```

Each `cmd/*/main.go` is a separate binary that can be built independently.

### Unified `defcmd` Syntax

**Key insight**: The syntax should **infer intent from structure**, supporting three patterns naturally:

1. **Script style** - Just expressions, wrapped in `main()`
2. **Explicit main** - Define helper functions + `main()`
3. **Full module** - Nested `(defmodule main ...)` for complete control

### Pattern 1: Script Style

**No explicit functions, just expressions:**

```lisp
(defcmd server
  :description "HTTP server"

  (import myapp.http)
  (import "fmt")

  ;; These become the body of main()
  (fmt/Println "Starting server...")
  (let [srv (http/new-server 8080)]
    (http/start srv)))
```

**Expands to:**

```lisp
(defmodule cmd.server.main
  :package "main"

  (import myapp.http)
  (import "fmt")

  (deffunc main []
    (:return)
    (fmt/Println "Starting server...")
    (let [srv (http/new-server 8080)]
      (http/start srv))))
```

### Pattern 2: Explicit Main Function

**Has `(deffunc main ...)` and helper functions:**

```lisp
(defcmd cli
  :description "CLI tool"

  (import "fmt" "os")

  ;; Helper functions
  (deffunc parse-args [args]
    (:args []string)
    (:return Command)
    ...)

  (deffunc show-help []
    (:return)
    (fmt/Println "Usage: cli [command]"))

  ;; Explicit main - use this as entry point
  (deffunc main []
    (:return)
    (when (< (length os.Args) 2)
      (show-help)
      (os/Exit 1))
    (let [cmd (parse-args os.Args)]
      (run-command cmd))))
```

**Expands to:**

```lisp
(defmodule cmd.cli.main
  :package "main"

  (import "fmt" "os")

  (deffunc parse-args [args]
    (:args []string)
    (:return Command)
    ...)

  (deffunc show-help []
    (:return)
    (fmt/Println "Usage: cli [command]"))

  (deffunc main []
    (:return)
    (when (< (length os.Args) 2)
      (show-help)
      (os/Exit 1))
    (let [cmd (parse-args os.Args)]
      (run-command cmd))))
```

### Pattern 3: Explicit Module

**Has nested `(defmodule main ...)` for full control:**

```lisp
(defcmd worker
  :description "Background worker"

  ;; Explicit module definition
  (defmodule main
    :package "main"
    :doc "Worker process entry point"

    (import myproject.jobs)
    (import myproject.db)
    (import "fmt")

    (deffunc initialize []
      (:return error)
      ...)

    (deffunc shutdown []
      (:return)
      ...)

    (deffunc main []
      (:return)
      (when-err (err (initialize))
        (fmt/Fprintf os.Stderr "Init failed: %v\n" err)
        (os/Exit 1))

      (defer (shutdown))

      (process-jobs))))
```

### Mixed Pattern: Script with Helpers

**Combines helper functions with implicit main:**

```lisp
(defcmd processor
  :description "Data processor"

  (import myproject.data)
  (import "fmt")

  ;; Helper function
  (deffunc process-file [path]
    (:args string)
    (:return error)
    (let [data (myproject.data/read path)]
      (myproject.data/transform data)))

  ;; No explicit main, so these expressions become main()
  (fmt/Println "Processing files...")
  (loop [files (get-files)]
    (when (first files)
      (process-file (first files))
      (recur (rest files)))))
```

**Expands to:**

```lisp
(defmodule cmd.processor.main
  :package "main"

  (import myproject.data)
  (import "fmt")

  (deffunc process-file [path]
    (:args string)
    (:return error)
    (let [data (myproject.data/read path)]
      (myproject.data/transform data)))

  ;; Generated main
  (deffunc main []
    (:return)
    (fmt/Println "Processing files...")
    (loop [files (get-files)]
      (when (first files)
        (process-file (first files))
        (recur (rest files))))))
```

### Expansion Algorithm

```
Input: (defcmd <name> <body>...)

1. Check for nested (defmodule main ...)
   → If found: Use it as-is, ensure package = "main"

2. Else, check for (deffunc main ...)
   → If found: Wrap all forms in module
   → main becomes entry point
   → Other deffunc become helpers

3. Else (only expressions and imports)
   → Collect (import ...) forms
   → Collect remaining expressions
   → Wrap expressions in (deffunc main [] ...)
   → Create module with imports + generated main
```

### Command Metadata

```lisp
(defcmd server
  ;; Basic metadata
  :description "HTTP server for the application"
  :version "0.1.0"
  :output "bin/server"                    ; Default: bin/<name>
  :aliases ["myserver" "serve"]           ; Shell command aliases

  ;; Build configuration
  :platforms ["linux/amd64" "darwin/arm64"]
  :build-tags ["production"]
  :ldflags ["-X main.version={{.Version}}"]

  ;; Documentation
  :doc "Starts the HTTP server on the configured port"
  :author "Duncan McGreggor"

  ...)
```

### Command Flags

```lisp
(defcmd server
  :description "HTTP server"
  :flags [
    (flag :port
      :type int
      :default 8080
      :description "Server port")

    (flag :config
      :type string
      :default "config.toml"
      :description "Configuration file")

    (flag :verbose
      :type bool
      :short "v"
      :description "Verbose logging")]

  (import myapp.http)
  (import "flag")

  ;; Flags are automatically parsed and available as *port*, *verbose*
  (let [srv (http/new-server *port*)]
    (when *verbose*
      (http/enable-debug-logging srv))
    (http/start srv)))
```

**Generates:**

```go
package main

import (
    "flag"
    http "myapp/http"
)

var (
    port    = flag.Int("port", 8080, "Server port")
    config  = flag.String("config", "config.toml", "Configuration file")
    verbose = flag.Bool("verbose", false, "Verbose logging")
)

func main() {
    flag.Parse()

    srv := http.New_server__1(*port)
    if *verbose {
        http.Enable_debug_logging__1(srv)
    }
    http.Start__1(srv)
}
```

### Subcommands

For complex CLI tools with multiple subcommands (like `git`, `docker`):

```lisp
(defcmd myproject
  :description "MyProject CLI tool"

  (import "os" "fmt")

  (defsubcommand server
    :description "Start the server"
    :flags [(flag :port :type int :default 8080)]

    (start-server *port*))

  (defsubcommand migrate
    :description "Run database migrations"
    :flags [(flag :direction :type string :default "up")]

    (run-migrations *direction*))

  (defsubcommand version
    :description "Show version"

    (fmt/Println "myproject version 0.1.0"))

  ;; Main dispatcher (can be explicit or implicit)
  (deffunc main []
    (:return)
    (if (< (length os.Args) 2)
      (do
        (fmt/Println "Usage: myproject [command]")
        (fmt/Println "Commands: server, migrate, version")
        (os/Exit 1))
      (dispatch-subcommand (nth os.Args 1)))))
```

**Usage:**

```bash
./myproject server --port 9000
./myproject migrate --direction down
./myproject version
```

### Examples Across All Patterns

#### Simplest: One-Liner

```lisp
(defcmd hello
  (import "fmt")
  (fmt/Println "Hello, world!"))
```

#### Simple Script

```lisp
(defcmd echo
  (import "fmt" "os")
  (when (> (length os.Args) 1)
    (fmt/Println (nth os.Args 1))))
```

#### Script with Helper

```lisp
(defcmd greet
  (import "fmt" "os")

  (deffunc get-name []
    (:return string)
    (if (> (length os.Args) 1)
      (nth os.Args 1)
      "World"))

  (fmt/Printf "Hello, %s!\n" (get-name)))
```

#### Explicit Main

```lisp
(defcmd greet
  (import "fmt" "os")

  (deffunc get-name []
    (:return string)
    (if (> (length os.Args) 1)
      (nth os.Args 1)
      "World"))

  (deffunc main []
    (:return)
    (fmt/Printf "Hello, %s!\n" (get-name))))
```

#### Full Module

```lisp
(defcmd greet
  (defmodule main
    :doc "A friendly greeting program"

    (import "fmt" "os" "strings")

    (deffunc get-name []
      (:return string)
      (if (> (length os.Args) 1)
        (strings/Title (nth os.Args 1))
        "World"))

    (deffunc format-greeting [name]
      (:args string)
      (:return string)
      (str "Hello, " name "!"))

    (deffunc main []
      (:return)
      (let [name (get-name)
            greeting (format-greeting name)]
        (fmt/Println greeting)))))
```

---

## File System Representation

### Hybrid Approach (Recommended)

Use `project.zl` as manifest with external files for implementation:

```
myapp/
├── project.zl          # Metadata + module/command declarations
├── go.mod              # Generated by build system
├── internal/
│   ├── http.zl        # Full module implementation
│   ├── db.zl
│   └── config.zl
└── cmd/
    ├── server/
    │   └── main.zl    # Generated from defcmd
    └── worker/
        └── main.zl    # Generated from defcmd
```

**project.zl** (manifest):

```lisp
(defproject myapp
  :version "0.1.0"
  :go-version "1.21"
  :dependencies [
    "github.com/gorilla/mux" "v1.8.0"]

  ;; Module declarations (implementations in files)
  (defmodule http
    :file "internal/http.zl"
    :export [new-server start])

  (defmodule db
    :file "internal/db.zl"
    :export [connect query])

  (defmodule config
    :file "internal/config.zl"
    :export [load-config])

  ;; Command definitions (inline or can reference files)
  (defcmd server
    :description "HTTP API server"
    :output "bin/server"

    (import myapp.http)
    (import myapp.config)

    (let [cfg (config/load-config "config.toml")
          srv (http/new-server (. cfg port))]
      (http/start srv)))

  (defcmd worker
    :description "Background worker"
    :output "bin/worker"
    :file "cmd/worker/main.zl"))  ; Or reference external file
```

**internal/http.zl** (implementation):

```lisp
(module http
  :export [new-server start]
  :doc "HTTP server implementation")

(import "net/http")
(import myapp.config)

(deftype Server
  {:port int
   :router *http.ServeMux})

(deffunc new-server [port]
  (:args int)
  (:return *Server)
  ...)

(deffunc start [srv]
  (:args *Server)
  (:return error)
  ...)
```

### Explode/Collapse Commands

#### Explode Command

```bash
zylisp explode project.zl
```

**What it does**:

1. Parse `project.zl`
2. Extract each `(defmodule ...)` form
3. Write to `<module-name>.zl` or specified `:file` location
4. Extract each `(defcmd ...)` form
5. Write to `cmd/<name>/main.zl`
6. Generate basic `go.mod`
7. Create `.zylisp/project-cache.edn` with metadata

**Generated go.mod**:

```go
module myapp

go 1.21

// Dependencies will be populated by 'go mod tidy'
```

#### Collapse Command

```bash
zylisp collapse internal/*.zl cmd/**/*.zl -o project.zl
```

**What it does**:

1. Read all specified `.zl` files
2. Parse each as a module or command
3. Combine into single `defproject` form
4. Write to `project.zl`

#### Sync Command

```bash
zylisp sync
```

**What it does**:

- If `project.zl` is newer than any file: explode
- If any `.zl` file is newer than `project.zl`: collapse
- Keep everything synchronized automatically

### Development Workflows

#### Workflow A: Single-File Development

Work entirely in `project.zl`, explode only for compilation:

```bash
# Edit project.zl
vim project.zl

# Build (auto-explodes if needed)
zylisp build

# REPL (loads unified structure)
zylisp repl project.zl
```

#### Workflow B: Multi-File Development

Work in separate files, collapse periodically:

```bash
# Initial setup
zylisp explode project.zl

# Edit individual files
vim internal/http.zl
vim internal/db.zl

# Build directly from files
zylisp build

# Sync back to canonical form
zylisp sync
```

#### Workflow C: Hybrid (Recommended)

Manifest in `project.zl`, implementations in files:

```bash
# Edit metadata and structure
vim project.zl

# Edit implementations
vim internal/http.zl

# Build (smart about what changed)
zylisp build

# REPL loads from combined structure
zylisp repl
```

---

## Build System Integration

### Build Process Overview

```
project.zl or *.zl files
    ↓ Parse
Module and Command definitions
    ↓ Build dependency graph
Topologically sorted modules
    ↓ For each module
Zylisp → Core Forms → Go AST → Go Source
    ↓ Generate
.zylisp/gen/ directory with Go packages
    ↓ go mod tidy
Resolved dependencies in go.mod
    ↓ go build
Binary executables in bin/
```

### Build Commands

```bash
# Build all commands
zylisp build

# Build specific command
zylisp build server

# Build multiple commands
zylisp build server worker

# Build with profile
zylisp build --profile production

# Build for specific platform
zylisp build --target linux/amd64

# Clean build artifacts
zylisp clean

# Generate Go code without building
zylisp generate
```

### Generated Directory Structure

From this project:

```lisp
(defproject myapp
  (defmodule http ...)
  (defmodule db ...)
  (defcmd server ...)
  (defcmd worker ...))
```

Generates:

```
.zylisp/
├── gen/
│   ├── go.mod              # module myapp
│   ├── http/
│   │   └── http.go
│   ├── db/
│   │   └── db.go
│   └── cmd/
│       ├── server/
│       │   └── main.go
│       └── worker/
│           └── main.go
└── cache/
    └── ...                 # Build cache for incremental compilation
```

### Build Algorithm

```go
func BuildProject(config ProjectConfig) error {
    // 1. Discover all modules and commands
    project := parseProject(config.ProjectFile)

    // 2. Build dependency graph
    depGraph := buildDependencyGraph(project)

    // 3. Topological sort for build order
    buildOrder := topologicalSort(depGraph)

    // 4. Compile each module in order
    for _, module := range buildOrder {
        if needsRebuild(module) {
            compileModule(module, config.GenDir)
        }
    }

    // 5. Compile each command
    for _, cmd := range project.Commands {
        compileCommand(cmd, config.GenDir)
    }

    // 6. Generate/update go.mod
    generateGoMod(project, config.GenDir)

    // 7. Run go mod tidy
    exec.Command("go", "mod", "tidy").Run()

    // 8. Build binaries
    for _, cmd := range project.Commands {
        goBuild(cmd, config.GenDir)
    }

    return nil
}

func needsRebuild(module Module) bool {
    cache := loadCache(module)

    // Check if source file changed
    if module.SourceFile.ModTime > cache.BuildTime {
        return true
    }

    // Check if dependencies changed
    for _, dep := range module.Dependencies {
        if depCache := loadCache(dep); depCache.BuildTime > cache.BuildTime {
            return true
        }
    }

    return false
}
```

### Incremental Compilation

The build system tracks:

- **File modification times**
- **Dependency relationships**
- **Cached compilation artifacts**

Only recompile when:

1. Source file has changed
2. A dependency has been rebuilt
3. Build flags have changed

### Cross-Compilation

```bash
# Build for multiple platforms
zylisp build --platforms linux/amd64,darwin/arm64,windows/amd64

# Generates:
# bin/server-linux-amd64
# bin/server-darwin-arm64
# bin/server-windows-amd64.exe
```

Implemented by setting `GOOS` and `GOARCH` environment variables when calling `go build`.

---

## REPL Integration

### Loading Projects

```lisp
;; Load from single file
zylisp> (load-project "project.zl")
Project: myapp v0.1.0
Modules: http, db, config
Commands: server, worker
Dependencies: 2 packages loaded

;; Load from directory (discovers files)
zylisp> (load-project ".")
Discovered 3 modules, 2 commands

;; Load specific module
zylisp> (load-module "internal/http.zl")
Module: http
Exports: new-server, start
```

### Navigating Modules

```lisp
zylisp> (list-modules)
=> (http db config)

zylisp> (show-module 'http)
Module: http
Exports: new-server, start
Functions: 2
Types: 1 (Server)

zylisp> (in-module 'http)
http> (exports)
=> (new-server start)

http> (new-server 8080)
=> #<Server :port 8080 :router ...>

http> (in-module 'main)
main>
```

### Testing Commands in REPL

```lisp
;; List commands
zylisp> (list-commands)
=> ((server "HTTP API server")
    (worker "Background job processor"))

;; Enter command context (test without running main)
zylisp> (in-command 'server)
server> (import myapp.http)
server> (myapp.http/new-server 8080)
=> #<Server :port 8080>

;; Actually run command main (for testing)
zylisp> (run-command 'server)
Starting server...
Server listening on :8080
^C

;; Or compile and run externally
zylisp> (compile-command 'server)
Built: bin/server
=> 0
```

### Hot Reloading

```lisp
;; Reload specific module
http> (reload-module 'http)
Module http reloaded
Updated 2 functions

;; Reload entire project
zylisp> (reload-project)
Reloaded: http, db, config
Recompiled: server, worker
```

### Interactive Development

```lisp
;; Load project
zylisp> (load-project "project.zl")

;; Experiment with module functions
zylisp> (in-module 'http)
http> (defvar test-server (new-server 8080))
http> (show test-server)
#<Server :port 8080 :router #<...>>

;; Modify and reload
http> (edit 'new-server)  ; Opens editor
;; ... make changes ...
http> (reload-module 'http)
http> (defvar test-server2 (new-server 9000))
http> (compare test-server test-server2)
=> {:port [8080 9000], :router [#<...> #<...>]}
```

---

## Implementation Plan

### Phase 1: Basic Module System (Week 1-2)

**Goal**: Support basic modules with imports/exports

**Deliverables**:

- [ ] `(defmodule ...)` form parser
- [ ] `(module ...)` form for files
- [ ] Module-to-file mapping
- [ ] Basic import resolution
- [ ] Export control
- [ ] Module compilation to Go packages

**Test Cases**:

- Two modules with mutual imports
- Private vs. public functions
- Import aliasing

### Phase 2: Project Definition (Week 3)

**Goal**: Support complete project definition

**Deliverables**:

- [ ] `(defproject ...)` form parser
- [ ] Metadata handling (version, dependencies, etc.)
- [ ] Explode command implementation
- [ ] Collapse command implementation
- [ ] Sync command
- [ ] `go.mod` generation
- [ ] Build integration

**Test Cases**:

- Single-file to multi-file conversion
- Multi-file to single-file conversion
- Sync with modifications on both sides

### Phase 3: Command System (Week 4)

**Goal**: Support executable commands with all three patterns

**Deliverables**:

- [ ] `(defcmd ...)` form parser
- [ ] Pattern detection (script vs. explicit main vs. full module)
- [ ] Expansion algorithm
- [ ] Flag system
- [ ] Subcommand support
- [ ] Command compilation to `cmd/*/main.go`
- [ ] Multiple binary building

**Test Cases**:

- Script-style command
- Command with helpers
- Command with explicit main
- Command with subcommands
- Command with flags

### Phase 4: Build System (Week 5-6)

**Goal**: Complete build pipeline with incremental compilation

**Deliverables**:

- [ ] Dependency graph construction
- [ ] Topological sort
- [ ] Build cache system
- [ ] Incremental compilation
- [ ] Cross-compilation support
- [ ] Build profiles (dev, production)
- [ ] `zylisp build` command suite

**Test Cases**:

- Full project build
- Incremental rebuild after module change
- Incremental rebuild after command change
- Cross-platform build

### Phase 5: REPL Integration (Week 7-8)

**Goal**: Full REPL support for projects and modules

**Deliverables**:

- [ ] Load project in REPL
- [ ] Module navigation
- [ ] Command testing
- [ ] Hot reload
- [ ] Module inspection
- [ ] Interactive documentation

**Test Cases**:

- Load and navigate multi-module project
- Test command in REPL without running main
- Hot reload after module edit
- Cross-module function calls in REPL

### Phase 6: Advanced Features (Week 9-10)

**Goal**: Polish and advanced capabilities

**Deliverables**:

- [ ] Build tags and conditional compilation
- [ ] Documentation generation
- [ ] Test integration
- [ ] Dependency version management
- [ ] Watch mode for development
- [ ] Workspace support (multiple projects)

**Test Cases**:

- Conditional compilation with build tags
- Generated documentation
- Test discovery and execution

---

## Open Questions

### 1. Dependency Management

**Question**: Should Zylisp have its own package manager, or rely entirely on Go modules?

**Options**:

**A. Pure Go Modules**

- Zylisp projects are Go modules
- Use `go.mod` directly
- Simple, leverages existing ecosystem

**B. Zylisp Package Manager**

- Separate Zylisp package registry
- Can manage Zylisp-specific metadata
- More control over versioning

**C. Hybrid**

- Use Go modules for Go dependencies
- Zylisp registry for Zylisp libraries
- Best of both worlds

**Recommendation**: Start with Option A (Pure Go Modules), add Zylisp registry later if needed.

### 2. Module Versioning

**Question**: How do we specify compatible Zylisp compiler versions?

**Options**:

```lisp
;; In project.zl
(defproject myapp
  :zylisp-version ">=0.5.0"
  :zylisp-version "^0.5.0"        ; Caret (SemVer)
  :zylisp-version "~0.5.0"        ; Tilde (patch updates)
  ...)
```

**Recommendation**: Follow Go's approach - modules declare minimum required version.

### 3. Test Organization

**Question**: Where do tests live?

**Options**:

**A. Alongside source**

```
internal/
├── http.zl
└── http_test.zl
```

**B. Separate test directory**

```
internal/http.zl
tests/http_test.zl
```

**C. Inline in defmodule**

```lisp
(defmodule http
  :tests [
    (deftest test-new-server ...)
    (deftest test-start ...)])
```

**Recommendation**: Option A (alongside source), matching Go convention.

### 4. Generated Code Formatting

**Question**: Should generated Go code be optimized for readability or size?

**Considerations**:

- Readable → easier debugging
- Compact → smaller repo, faster builds
- Formatted → consistent style

**Recommendation**: Format generated code with `go fmt`, but don't obsess over perfect style since it's machine-generated.

### 5. Workspace Mode

**Question**: How do we handle multiple related Zylisp projects in one workspace?

**Example**:

```
workspace/
├── app/
│   └── project.zl
├── lib/
│   └── project.zl
└── tools/
    └── project.zl
```

**Recommendation**: Defer to Phase 6. Use Go's workspace mode for now.

### 6. Build Plugins

**Question**: Support for custom build steps (e.g., protobuf generation)?

**Example**:

```lisp
(defproject myapp
  :build-hooks [
    (:pre-build generate-protos)
    (:post-build copy-assets)])
```

**Recommendation**: Add in Phase 6 if needed.

---

## Conclusion

### Summary of Decisions

1. **Single-file canonical form** (`project.zl`) as source of truth
2. **Explode/collapse transformations** for flexible development
3. **Module system** (`defmodule`) for library code
4. **Command system** (`defcmd`) for executables with unified syntax supporting three patterns
5. **Hybrid file layout** (manifest + implementation files)
6. **Leverage Go tooling** (generate Go, use `go build`)
7. **Full REPL integration** with module navigation and hot reload

### Key Benefits

**Homoiconicity**: Projects are data structures, can be manipulated programmatically

**Flexibility**: Work in single file or multiple files, switch freely

**Gradual Complexity**: Start simple (script-style commands), add structure as needed

**Natural REPL Integration**: Same structures work in files and REPL

**Go Interoperability**: Generated code is idiomatic Go, uses Go toolchain

**Scalability**: Package-based compilation scales to large projects

### Next Steps

1. **Review and approve** this design document
2. **Prioritize phases** for implementation
3. **Create detailed specifications** for Phase 1
4. **Begin implementation** of basic module system
5. **Iterate and refine** based on real-world usage

### Philosophy

This design embodies the Lisp philosophy:

> **"Code is data, data is code."**

In Zylisp, this extends to:

> **"Projects are data, data is projects."**

The file system is just one view. The REPL is another view. The canonical form is the data itself. All transformations are just manipulations of that data structure.

This is the power of homoiconicity applied at the project level.

---

## Appendix: Complete Examples

### Example 1: Simple Web Server

```lisp
(defproject simple-server
  :version "0.1.0"
  :go-version "1.21"
  :dependencies ["github.com/gorilla/mux" "v1.8.0"]

  (defmodule http
    :export [new-server]

    (import "net/http" "fmt")
    (import "github.com/gorilla/mux")

    (deffunc new-server [port]
      (:args int)
      (:return *http.Server)
      (let [router (mux/NewRouter)]
        (. router HandleFunc "/" handle-root)
        {:Addr (fmt/Sprintf ":%d" port)
         :Handler router})))

  (defcmd server
    :description "Simple HTTP server"
    :flags [(flag :port :type int :default 8080)]

    (import simple-server.http)
    (import "fmt")

    (let [srv (http/new-server *port*)]
      (fmt/Printf "Listening on :%d\n" *port*)
      (http/ListenAndServe srv))))
```

### Example 2: CLI Tool with Subcommands

```lisp
(defproject mytool
  :version "0.1.0"

  (defmodule config
    :export [load save]

    (deffunc load [path]
      (:args string)
      (:return Config error)
      ...)

    (deffunc save [cfg path]
      (:args Config string)
      (:return error)
      ...))

  (defcmd mytool
    :description "A versatile CLI tool"

    (import mytool.config)
    (import "fmt" "os")

    (defsubcommand init
      :description "Initialize configuration"
      :flags [(flag :path :type string :default ".config")]

      (let [cfg (config/new-config)]
        (config/save cfg *path*)
        (fmt/Println "Initialized")))

    (defsubcommand run
      :description "Run the tool"
      :flags [(flag :config :type string :default ".config")]

      (let [(cfg err) (config/load *config*)]
        (when err
          (fmt/Fprintf os.Stderr "Error: %v\n" err)
          (os/Exit 1))
        (run-tool cfg)))

    (deffunc main []
      (:return)
      (if (< (length os.Args) 2)
        (do
          (fmt/Println "Usage: mytool [command]")
          (fmt/Println "Commands: init, run")
          (os/Exit 1))
        (dispatch-subcommand (nth os.Args 1))))))
```

### Example 3: Multi-Service Application

```lisp
(defproject multi-service
  :version "0.1.0"
  :dependencies [
    "github.com/gorilla/mux" "v1.8.0"
    "github.com/lib/pq" "v1.10.0"
    "github.com/go-redis/redis/v8" "v8.11.0"]

  ;; Shared modules
  (defmodule config
    :file "internal/config.zl"
    :export [load-config])

  (defmodule db
    :file "internal/db.zl"
    :export [connect query])

  (defmodule cache
    :file "internal/cache.zl"
    :export [new-client get set])

  (defmodule http
    :file "internal/http.zl"
    :export [new-server register-routes])

  (defmodule jobs
    :file "internal/jobs.zl"
    :export [process-job])

  ;; API server
  (defcmd api
    :description "HTTP API server"
    :output "bin/api"
    :flags [
      (flag :port :type int :default 8080)
      (flag :config :type string :default "config.toml")]

    (import multi-service.http)
    (import multi-service.db)
    (import multi-service.config)
    (import "fmt")

    (let [cfg (config/load-config *config*)
          db-conn (db/connect (. cfg database-url))
          srv (http/new-server *port* db-conn)]
      (fmt/Printf "API server listening on :%d\n" *port*)
      (http/start srv)))

  ;; Background worker
  (defcmd worker
    :description "Background job processor"
    :output "bin/worker"
    :flags [(flag :config :type string :default "config.toml")]

    (import multi-service.jobs)
    (import multi-service.db)
    (import multi-service.config)
    (import "fmt" "time")

    (let [cfg (config/load-config *config*)
          db-conn (db/connect (. cfg database-url))]
      (fmt/Println "Worker started")
      (loop []
        (when-let [job (db/query db-conn "SELECT * FROM jobs WHERE status = 'pending' LIMIT 1")]
          (jobs/process-job job db-conn))
        (time/Sleep (* 5 time.Second))
        (recur))))

  ;; Admin CLI
  (defcmd admin
    :description "Administration tool"
    :output "bin/admin"

    (import multi-service.db)
    (import multi-service.config)
    (import "fmt" "os")

    (defsubcommand migrate
      (let [cfg (config/load-config "config.toml")
            db-conn (db/connect (. cfg database-url))]
        (db/migrate db-conn)
        (fmt/Println "Migrations complete")))

    (defsubcommand user-create
      :flags [
        (flag :email :type string :required true)
        (flag :role :type string :default "user")]

      (let [cfg (config/load-config "config.toml")
            db-conn (db/connect (. cfg database-url))]
        (db/execute db-conn
          "INSERT INTO users (email, role) VALUES ($1, $2)"
          *email* *role*)
        (fmt/Printf "Created user: %s (%s)\n" *email* *role*)))

    (deffunc main []
      (:return)
      (if (< (length os.Args) 2)
        (do
          (fmt/Println "Usage: admin [command]")
          (fmt/Println "Commands: migrate, user-create")
          (os/Exit 1))
        (dispatch-subcommand (nth os.Args 1))))))
```

---

**End of Document**
