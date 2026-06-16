---
sidebar_position: 3
---

# Record Format

Each sadr record is a Markdown file stored in `.sadr/<username>/records/`. The format is plain Markdown with a YAML frontmatter block — readable by humans and tools without sadr installed.

## File naming

```
sadr-record-<id>-<slug>.md
```

| Part | Description |
|---|---|
| `<id>` | 4-digit zero-padded sequential integer (per user) |
| `<slug>` | URL-safe slug derived from the title (max 80 chars) |

Examples:
```
sadr-record-0001-retry-logic.md
sadr-record-0042-switch-to-postgresql-for-primary-storage.md
```

---

## File structure

```markdown
---
schema_version: 1
type: full
title: Switch to PostgreSQL for primary storage
file_ref: src/db/connection.go
created: 2024-03-15T14:32:00Z
author: alice
tags:
  - architecture
  - data
status: active
fine_tuning_hint: "focus on migration risk and rollback strategy"
---

# Switch to PostgreSQL for primary storage

**Tags:** `#architecture` `#data`

**Status:** `#active`

**Question:**
> focus on migration risk and rollback strategy

## Context

The previous SQLite setup was a bottleneck under concurrent load...

## Decision

Migrate to PostgreSQL 15 for the primary data store...

## Alternatives

- Continue with SQLite and add connection pooling
- Switch to MySQL for better tooling support

## Consequences

The migration requires a one-time data export and reimport...


## Snippet

```go
func NewDB(cfg Config) (*sql.DB, error) {
    return sql.Open("postgres", cfg.DSN)
}
```
```

The body is ordered decision-first: the title, tags and status come first, then
the ADR fields in schema order, and the code snippet (with its optional
`**Question:**` capture hint) is rendered last — so the reasoning is what you
read first and the code is reference at the bottom.

---

## Frontmatter fields

| Field | Type | Description |
|---|---|---|
| `schema_version` | int | Always `1`. Used for forward-compatibility checks. |
| `type` | string | Record type: `full`, `adr`, or `snippet` |
| `title` | string | Record title |
| `file_ref` | string | Associated file path(s), comma-separated, or `N/A` |
| `created` | RFC3339 | Creation timestamp |
| `author` | string | Username of the creator |
| `tags` | string[] | List of tags |
| `status` | string | Lifecycle status (see below) |
| `related` | string[] | Record IDs (`author/id`) this record intentionally coexists with — see [CI Gatekeeper](../guide/ci-gatekeeper) |
| `fine_tuning_hint` | string | Custom AI instruction used at capture time |

---

## Record types

| Type | Snippet step | ADR fields |
|---|---|---|
| `full` | Yes | Yes |
| `adr` | No | Yes |
| `snippet` | Yes | No |

---

## Status values

| Status | Meaning |
|---|---|
| `active` | Current, valid decision |
| `proposed` | Under discussion, not yet approved |
| `draft` | Work in progress, incomplete |
| `deprecated` | No longer recommended but not replaced |
| `superseded` | Replaced by a newer decision |

`sadr ask` only considers records with `status: active` by default. Records with `proposed`, `deprecated`, or `superseded` status are excluded from AI queries.

---

## Body sections

The record body uses `##` headings to define sections. Section names map directly to field names defined in the config schema (lowercased, spaces replaced by `_`).

The `## Snippet` section is always parsed as a code block — sadr strips the surrounding backticks when reading back the file.

Any `##` section not defined in the schema is still stored and displayed — sadr does not enforce strict field matching on read.

---

## Reading records without sadr

Since records are plain Markdown with YAML frontmatter, any Markdown renderer (GitHub, GitLab, VS Code, Obsidian) displays them correctly. No sadr installation is required to read them.
