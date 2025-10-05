# Instructions for Building zdp.go (Zylisp Design Proposal Tool)

## Overview

Create a Go program called `zdp.go` that manages state transitions for Zylisp design documents. The tool should handle moving documents between state directories and updating their metadata headers.

## Program Requirements

### Command-Line Interface

The program supports three modes of operation:

#### Mode 1: Transition to New State
```bash
go run zdp.go <relative-path/doc.md> <new-state>
```

- Takes a document path and a new state name
- Updates the document's `state:` field in the YAML frontmatter
- Moves the document to the appropriate state directory
- Validates that the new state differs from the current state
- Validates that the new state is one of the supported states

#### Mode 2: Move to Directory Matching Header State
```bash
go run zdp.go <relative-path/doc.md>
```

- Takes only a document path
- Reads the `state:` field from the document's YAML frontmatter
- Moves the document to the directory matching that state
- Validates that the document is not already in the correct directory
- Validates that the state in the header is a supported state

#### Mode 3: List All Documents by State
```bash
go run zdp.go
```

- Lists all design documents organized by state
- Shows state names in title case (no number prefixes)
- Lists documents under each state with bullet points

Output format:
```
Draft
 - 0001-go-lisp-intent.md
 - 0015-zast-phase3-impl.md

Under Review
 - 0023-zast-position-removal.md

Final
 - 0007-writer-spec.md
```

#### Mode 4: List Supported States
```bash
go run zdp.go states
```

- Lists all supported state names (title case)
- One state per line

Output format:
```
Draft
Under Review
Revised
Accepted
Active
Final
Deferred
Rejected
Withdrawn
Superseded
```

#### Mode 5: Add Document to Index
```bash
go run zdp.go index <relative-path/doc.md>
```

- Reads the document's metadata (number, title, state)
- Adds the document to `00-index.md` if not already present
- Updates both the "All Documents by Number" table and the "Documents by State" section
- Ensures the document appears in the correct state section based on its header
- Does nothing if the document is already correctly indexed

## State Mapping

The program must support these states and their corresponding directories:

| State Name (title case) | Directory Name | Header Value |
|-------------------------|----------------|--------------|
| Draft | 01-draft | Draft |
| Under Review | 02-under-review | Under Review |
| Revised | 03-revised | Revised |
| Accepted | 04-accepted | Accepted |
| Active | 05-active | Active |
| Final | 06-final | Final |
| Deferred | 07-deferred | Deferred |
| Rejected | 08-rejected | Rejected |
| Withdrawn | 09-withdrawn | Withdrawn |
| Superseded | 10-superseded | Superseded |

**Important Notes:**
- State names in headers use title case (e.g., "Under Review")
- Directory names use lowercase with hyphens (e.g., "02-under-review")
- When users specify states on command line, accept both formats case-insensitively

## Implementation Details

### YAML Frontmatter Parsing

Documents have YAML frontmatter like this:

```yaml
---
number: 0001
title: Go Lisp Intent
author: John Doe
created: 2024-01-15
updated: 2024-03-20
state: Draft
supersedes: None
superseded-by: None
---
```

The program must:
1. Read the entire file
2. Parse the YAML frontmatter (between `---` markers)
3. Extract the `state:` field value
4. Preserve all other metadata exactly as-is
5. When updating state, only modify the `state:` field and `updated:` field (set to current date)

### Error Handling and Validation

The program must panic with informative errors in these cases:

1. **Same state error**: When new state equals current state
   ```
   Error: Document is already in state "Draft"
   ```

2. **Unsupported state error**: When state is not in the supported list
   ```
   Error: Unsupported state "InProgress". Supported states are:
   Draft, Under Review, Revised, Accepted, Active, Final, Deferred, Rejected, Withdrawn, Superseded
   ```

3. **Already in correct directory error**: When using Mode 2 and document is already in the directory matching its header state
   ```
   Error: Document is already in the correct directory for state "Draft"
   ```

4. **File not found error**:
   ```
   Error: File not found: <path>
   ```

5. **Invalid YAML frontmatter error**:
   ```
   Error: Could not parse YAML frontmatter in <path>
   ```

6. **Missing state field error**:
   ```
   Error: No 'state' field found in document metadata
   ```

### File Operations

When transitioning a document:

1. **Read** the source file
2. **Parse** YAML frontmatter
3. **Update** the `state:` field to the new state value
4. **Update** the `updated:` field to today's date (YYYY-MM-DD format)
5. **Write** to the destination directory with the same filename
6. **Delete** the source file only after successful write
7. **Preserve** exact formatting and content of the document body
8. **Update** `00-index.md` to reflect the document's new location and state

### Index File Management

The `00-index.md` file has two main sections that must be kept in sync:

#### Section 1: All Documents by Number

This is a markdown table with columns:
- Number (4 digits)
- Title
- State
- Updated (date)

Example:
```markdown
## All Documents by Number

| Number | Title | State | Updated |
|--------|-------|-------|---------|
| 0001 | Go Lisp Intent | Draft | 2024-01-15 |
| 0015 | Zast Phase3 Impl | Under Review | 2024-03-20 |
| 0023 | Zast Position Removal | Final | 2024-02-10 |
```

**Important**: Rows must be sorted by Number in ascending order.

#### Section 2: Documents by State

This section has subsections for each state, with markdown links to documents.

Example:
```markdown
## Documents by State

### Draft (01-draft/)
- [0001 - Go Lisp Intent](01-draft/0001-go-lisp-intent.md)

### Under Review (02-under-review/)
- [0015 - Zast Phase3 Impl](02-under-review/0015-zast-phase3-impl.md)

### Final (06-final/)
- [0023 - Zast Position Removal](06-final/0023-zast-position-removal.md)
```

**Important**: 
- Only include state subsections that have documents
- Within each subsection, list documents sorted by number
- Use the exact format: `- [NUMBER - TITLE](path/to/file.md)`

### Index Update Operations

When transitioning a document (Modes 1 and 2):

1. **Update the table row** in "All Documents by Number":
   - Find the row by document number
   - Update the State column to the new state
   - Update the Updated column to today's date
   - Keep the row in its position (sorted by number)

2. **Update the state sections** in "Documents by State":
   - Remove the document from its old state subsection
   - Add the document to its new state subsection (maintaining number sort order)
   - If the old state subsection becomes empty, remove the entire subsection
   - If the new state subsection doesn't exist, create it

When adding to index (Mode 5):

1. **Check "All Documents by Number"**:
   - Parse the table to find if document number exists
   - If missing, insert a new row in the correct sorted position
   - Extract number, title, state, and updated date from document metadata

2. **Check "Documents by State"**:
   - Find the subsection for the document's current state
   - Check if the document link is present
   - If missing, insert in the correct sorted position within that subsection
   - Create the subsection if it doesn't exist

### Directory Scanning

For listing documents (Mode 3):

1. Scan all state directories (01-draft through 10-superseded)
2. Read each `.md` file's YAML frontmatter
3. Group files by their `state:` field value
4. Sort directories in numerical order (01, 02, 03, etc.)
5. Sort filenames within each state alphabetically
6. Display in the specified format

### Code Structure Recommendations

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "time"
)

// State mapping between names, directories, and header values
var states = map[string]string{
    "draft":        "01-draft",
    "under review": "02-under-review",
    "revised":      "03-revised",
    "accepted":     "04-accepted",
    "active":       "05-active",
    "final":        "06-final",
    "deferred":     "07-deferred",
    "rejected":     "08-rejected",
    "withdrawn":    "09-withdrawn",
    "superseded":   "10-superseded",
}

// Helper functions needed:
// - parseYAML(content string) (map[string]string, error)
// - updateYAML(content string, newState string) (string, error)
// - getStateDir(stateName string) string
// - normalizeState(input string) string
// - getCurrentState(filePath string) (string, error)
// - listAllDocuments() map[string][]string
// - moveDocument(srcPath, dstPath string) error
// - updateIndex(docPath, oldState, newState string) error
// - addToIndex(docPath string) error
// - parseIndexTable(content string) []IndexRow
// - updateIndexTable(content string, docNumber, newState, newUpdated string) string
// - removeFromStateSection(content, docPath, state string) string
// - addToStateSection(content, docPath, state, title string) string
// - extractDocMetadata(docPath string) (number, title, state, updated string, err error)

func main() {
    // Parse arguments and route to appropriate function
}
```

## Testing Checklist

Create test scenarios for:

1. ✅ Transitioning a document from Draft to Under Review
2. ✅ Attempting to transition to the same state (should error)
3. ✅ Attempting to transition to invalid state (should error)
4. ✅ Moving a document when header state doesn't match directory
5. ✅ Attempting to move when already in correct directory (should error)
6. ✅ Listing all documents with no arguments
7. ✅ Listing supported states with `states` argument
8. ✅ Handling missing files gracefully
9. ✅ Handling malformed YAML gracefully
10. ✅ Preserving document content exactly (only updating metadata)
11. ✅ Index is updated when document transitions between states
12. ✅ Document is removed from old state section in index
13. ✅ Document is added to new state section in index
14. ✅ Table in "All Documents by Number" is updated with new state
15. ✅ Adding a missing document to index with `index` command
16. ✅ Running `index` command on already-indexed document (should be no-op)
17. ✅ State sections maintain alphabetical order by number
18. ✅ Empty state sections are removed from index
19. ✅ New state sections are created as needed

## README.md Updates

Add the following section to the README.md file after the "Document Metadata" section:

```markdown
## Managing Document States with zdp

The `zdp` tool (Zylisp Design Proposal) helps manage document state transitions and organization.

### Installation

No installation needed. Run directly with Go:

```bash
go run zdp.go [arguments]
```

### Usage

#### Transition a document to a new state

```bash
go run zdp.go <path-to-doc.md> <new-state>
```

Example:
```bash
go run zdp.go 01-draft/0015-zast-phase3-impl.md "Under Review"
```

This will:
- Update the document's `state:` field to "Under Review"
- Update the `updated:` field to today's date
- Move the document to `02-under-review/`

#### Move a document to match its header state

If you've manually updated a document's `state:` field but haven't moved it yet:

```bash
go run zdp.go <path-to-doc.md>
```

Example:
```bash
go run zdp.go 01-draft/0015-zast-phase3-impl.md
```

The tool will read the document's `state:` field and move it to the appropriate directory.

#### Add a document to the index

If you've created a new document or need to ensure a document is properly indexed:

```bash
go run zdp.go index <path-to-doc.md>
```

Example:
```bash
go run zdp.go index 01-draft/0028-new-feature.md
```

This will:
- Add the document to the "All Documents by Number" table if missing
- Add the document to the appropriate state section if missing
- Do nothing if the document is already properly indexed

**Note**: When transitioning documents with the other commands, the index is automatically updated. This command is only needed for manually created documents or to fix index inconsistencies.

#### List all documents by state

```bash
go run zdp.go
```

This displays all documents organized by their current state.

#### List supported states

```bash
go run zdp.go states
```

This shows all valid state names that can be used.

### Supported States

- Draft
- Under Review
- Revised
- Accepted
- Active
- Final
- Deferred
- Rejected
- Withdrawn
- Superseded

State names are case-insensitive when used on the command line.
```

## Implementation Notes for Claude Code

1. **YAML Parsing**: Use a simple regex-based parser or the `gopkg.in/yaml.v3` package
2. **File Operations**: Use `os.ReadFile`, `os.WriteFile`, and `os.Remove` for atomic operations
3. **Path Handling**: Use `filepath` package for cross-platform compatibility
4. **Date Formatting**: Use `time.Now().Format("2006-01-02")` for YYYY-MM-DD format
5. **Case Handling**: Normalize state names to lowercase with spaces for comparison
6. **Error Messages**: Use `panic()` with descriptive error messages as specified
7. **Directory Traversal**: Use `filepath.Walk` or `os.ReadDir` for scanning directories
8. **Index Parsing**: Use regex or string manipulation to parse markdown tables and lists
9. **Index Updates**: Read entire index file, modify in memory, write atomically
10. **Sorting**: Ensure document numbers are compared as integers (not strings) for proper ordering
11. **State Section Management**: Track which state sections have documents and only include non-empty sections

### Index File Structure Details

The index file has this general structure:

```markdown
# Zylisp Design Documents Index

## All Documents by Number

| Number | Title | State | Updated |
|--------|-------|-------|---------|
| 0001 | Go Lisp Intent | Draft | 2024-01-15 |
...

## Documents by State

### Draft (01-draft/)
- [0001 - Go Lisp Intent](01-draft/0001-go-lisp-intent.md)
...

### Under Review (02-under-review/)
- [0015 - Zast Phase3 Impl](02-under-review/0015-zast-phase3-impl.md)
...
```

**Parsing Strategy**:
1. Split file into sections using `## ` headers as delimiters
2. Parse table section: extract rows between `|---|` header separator and next `##`
3. Parse state sections: find `### State Name (NN-directory/)` headers and extract bullet lists until next `###` or `##`

**Update Strategy**:
1. Read entire file into memory
2. Parse into structured data (table rows + state section maps)
3. Make modifications to data structure
4. Rebuild markdown from modified structure
5. Write back to file atomically

## Deliverables

1. `zdp.go` - The complete Go program
2. Updated `README.md` with zdp usage instructions
3. Test the program with at least one document transition
4. Verify all error cases produce appropriate messages

## Example Session

```bash
# List current documents
$ go run zdp.go
Draft
 - 0001-go-lisp-intent.md
 - 0015-zast-phase3-impl.md

# Move a document to Under Review
$ go run zdp.go 01-draft/0015-zast-phase3-impl.md "Under Review"
Moved 0015-zast-phase3-impl.md from Draft to Under Review
Updated index

# Verify index was updated
$ cat 00-index.md
# Zylisp Design Documents Index

## All Documents by Number

| Number | Title | State | Updated |
|--------|-------|-------|---------|
| 0001 | Go Lisp Intent | Draft | 2024-01-15 |
| 0015 | Zast Phase3 Impl | Under Review | 2024-10-04 |

## Documents by State

### Draft (01-draft/)
- [0001 - Go Lisp Intent](01-draft/0001-go-lisp-intent.md)

### Under Review (02-under-review/)
- [0015 - Zast Phase3 Impl](02-under-review/0015-zast-phase3-impl.md)

# List again to verify
$ go run zdp.go
Draft
 - 0001-go-lisp-intent.md

Under Review
 - 0015-zast-phase3-impl.md

# Try to move to same state (error)
$ go run zdp.go 02-under-review/0015-zast-phase3-impl.md "Under Review"
Error: Document is already in state "Under Review"

# Try invalid state (error)
$ go run zdp.go 02-under-review/0015-zast-phase3-impl.md "InProgress"
Error: Unsupported state "InProgress". Supported states are:
Draft, Under Review, Revised, Accepted, Active, Final, Deferred, Rejected, Withdrawn, Superseded

# Create a new document and add to index
$ go run zdp.go index 01-draft/0028-new-feature.md
Added 0028-new-feature.md to index

# Try to add again (no-op)
$ go run zdp.go index 01-draft/0028-new-feature.md
Document already indexed correctly
```
