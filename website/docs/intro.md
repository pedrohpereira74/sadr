---
slug: /
sidebar_position: 1
---

# sadr

**Capture code with context. Because snippets without a "why" are sadr.**

sadr is a CLI tool that blends a snippet manager with Architectural Decision Records (ADR). It captures code from your clipboard, a file, or a git diff — and guides you to document the *why* behind it: the context, the decision, the trade-offs. Or any custom fields your team needs.

## Why sadr?

Every time you solve a non-trivial problem, you have context in your head that the code alone cannot express:

- Why did you choose this approach over the alternatives?
- What constraints forced this trade-off?
- What would you do differently next time?

That context evaporates fast. sadr captures it *at the moment you write the code*, stores it alongside the snippet as a Markdown file in your repo, and makes it searchable and queryable months later.

## Core concepts

| Concept | Description |
|---|---|
| **Record** | A single capture: an optional code snippet + structured metadata (title, tags, custom fields) |
| **ADR** | An Architecture Decision Record — a record focused on a decision, without a snippet |
| **Snippet** | A record focused on capturing a code snippet, without ADR fields |
| **Vault** | A `.sadr/` directory in your project that stores all records as Markdown files |
| **Schema** | A `config.yaml` that defines what fields to capture and how — fully customizable per project |

## Key features

- **AI-assisted capture** — `--smart` analyzes your snippet and pre-fills fields using Gemini
- **Multiple sources** — clipboard, file path, or `git diff HEAD`
- **Customizable schemas** — define your own `text`, `multitext`, `select`, `multiselect`, `list`, and `jira` fields
- **Interactive TUI hub** — `sadr search` opens a fuzzy-search interface for browsing, deleting, and exporting
- **AI persona queries** — `sadr ask` lets a Tech Lead, DBA, Security Analyst, or custom persona answer questions grounded in your records
- **Jira integration** — link records to Jira cards; the AI uses the card summary as extra context
- **HTML export** — self-contained, standalone file you can share without any tooling
- **Personal + team vaults** — project vault in `.sadr/` (git-tracked) + private vault in `~/.sadr/`
- **Multi-user** — each contributor's records are stored under their username within `.sadr/`

## How a typical workflow looks

```
You solve a problem
  → sadr new --diff --smart
  → AI suggests title, context, decision, trade-offs
  → You review, adjust, confirm
  → .sadr/alice/records/sadr-record-0042-retry-logic.md is saved

Later, a teammate:
  → sadr search "retry"               finds the record
  → sadr ask --role "dba"             asks about persistence decisions
  → sadr export --all                 exports everything to HTML for a report
```

## Next steps

- [Getting Started](./getting-started) — install and capture your first record in 5 minutes
- [Configuration](./guide/configuration) — customize fields and schemas for your team
- [CLI Reference](./reference/cli) — complete command and flag reference
