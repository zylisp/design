# Zylisp Design Documents

Design proposals and specifications for the Zylisp programming language.

This repository contains all design documents, proposals, and specifications for Zylisp. Each document follows a structured lifecycle from initial draft through review, acceptance, implementation, and finalization. This process ensures that language design decisions are well-documented, reviewed by the community, and tracked throughout their lifecycle.

## Design Document States

Design documents progress through a series of states that track their evolution from initial proposal to final implementation or closure. Understanding these states helps contributors know where each proposal stands and what actions are needed next.

### State Definitions

- **Draft** - Initial proposal being written and refined by the author
- **Under Review** - Submitted for community/team feedback and discussion
- **Revised** - Incorporating feedback from review; may cycle back to review
- **Accepted** - Approved for implementation
- **Active** - Currently being implemented
- **Final** - Implementation complete and deployed
- **Deferred** - Put on hold for later consideration
- **Rejected** - Decided not to proceed with the proposal
- **Withdrawn** - Author has withdrawn the proposal
- **Superseded** - Replaced by a newer proposal

### State Transitions

Documents typically flow through these states as follows:

- **Draft** → Under Review (when ready for feedback)
- **Under Review** → Revised (feedback received) | Accepted (approved) | Rejected | Deferred | Withdrawn
- **Revised** → Under Review (resubmitted after changes)
- **Accepted** → Active (implementation begins) | Deferred
- **Active** → Final (implementation complete) | Withdrawn
- **Deferred** → Under Review (reconsidered) | Rejected | Withdrawn
- **Final** → Superseded (replaced by newer proposal)

```mermaid
stateDiagram-v2
    [*] --> Draft: Create proposal

    Draft --> UnderReview: Submit for review
    Draft --> Withdrawn: Author withdraws

    UnderReview --> Revised: Feedback received
    UnderReview --> Accepted: Approved
    UnderReview --> Rejected: Not approved
    UnderReview --> Deferred: Postponed
    UnderReview --> Withdrawn: Author withdraws

    Revised --> UnderReview: Resubmit
    Revised --> Withdrawn: Author withdraws

    Accepted --> Active: Begin implementation
    Accepted --> Deferred: Delayed start

    Active --> Final: Implementation complete
    Active --> Withdrawn: Abandoned

    Deferred --> UnderReview: Reconsidered
    Deferred --> Rejected: No longer viable
    Deferred --> Withdrawn: Author withdraws

    Final --> Superseded: Newer proposal replaces

    Rejected --> [*]
    Withdrawn --> [*]
    Superseded --> [*]
    Final --> [*]
```

## Directory Structure

```
design-docs/
├── README.md                      # This file
├── 00-index.md                    # Master index/catalog of all design docs
├── 01-draft/                      # Proposals being written
├── 02-under-review/               # Submitted for feedback
├── 03-revised/                    # Being updated based on feedback
├── 04-accepted/                   # Approved, awaiting implementation
├── 05-active/                     # Currently being implemented
├── 06-final/                      # Implemented and complete
├── 07-deferred/                   # On hold for later
├── 08-rejected/                   # Not proceeding
├── 09-withdrawn/                  # Author withdrew
├── 10-superseded/                 # Replaced by newer proposals
└── templates/
    └── design-doc-template.md     # Template for new docs
```

## Document Naming Convention

Documents follow the pattern: `NNNN-short-title.md`

Examples:

- `0001-go-lisp-intent.md`
- `0015-zast-phase3-impl.md`
- `0023-zast-position-removal.md`

## Document Metadata

Each design document includes a YAML frontmatter header with the following fields:

```yaml
---
number: 0001
title: Full Title of the Proposal
author: Author Name(s)
created: YYYY-MM-DD
updated: YYYY-MM-DD
state: Draft
supersedes: None
superseded-by: None
---
```

### Metadata Field Descriptions

- **number**: Four-digit document number (padded with leading zeros)
- **title**: Full descriptive title of the proposal
- **author**: Name(s) of the document author(s)
- **created**: Date the document was first created
- **updated**: Date of the most recent modification
- **state**: Current state in the workflow (see States above)
- **supersedes**: Document number(s) this proposal replaces, or "None"
- **superseded-by**: Document number that replaces this one, or "None"

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
- Update `00-index.md` to reflect the new state and location

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
go run zdp.go index 01-draft/0029-new-feature.md
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

## Contributing

When creating a new design document:

1. Use the template in `templates/design-doc-template.md`
2. Assign the next available document number
3. Place the document in `01-draft/`
4. Add the document to the index: `go run zdp.go index 01-draft/NNNN-your-doc.md`
5. As the document progresses, use `zdp` to transition it: `go run zdp.go 01-draft/NNNN-your-doc.md "Under Review"`
