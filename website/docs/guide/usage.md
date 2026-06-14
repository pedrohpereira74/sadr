---
sidebar_position: 2
---

# Usage Guide

## `sadr new` — Capture a record

### Subcommands

| Command | What it captures |
|---|---|
| `sadr new` | Full record: snippet + all ADR fields |
| `sadr new adr` | ADR only — no snippet, all fields |
| `sadr new snippet` | Snippet only — no ADR fields |

### Snippet sources

Exactly one source flag at a time:

| Flag | Source |
|---|---|
| `--clipboard` / `-c` | Reads from system clipboard |
| `--file <path>` / `-f` | Reads content of a file (max 10 MB) |
| `--diff` / `-d` | Runs `git diff HEAD` and uses the output |
| _(none)_ | Skips snippet, or opens editor if `--smart` is used |

### AI assistance

```bash
# AI pre-fills all fields from the snippet
sadr new --diff --smart

# Use a specific model
sadr new --clipboard --smart --model gemini-2.5-pro

# Add a custom instruction to guide the AI
sadr new --diff --smart --fine-tuning
# → prompts: "add a custom instruction to guide the AI (optional):"
# → e.g. "focus on security implications"
```

### Saving to personal vault

```bash
sadr new --clipboard --smart --global
# → saves to ~/.sadr/records/ instead of the project vault
```

### Title shortcut

Skip the interactive wizard entirely with `--title`:

```bash
sadr new snippet --file src/handler.go --title "HTTP handler pattern"
```

### Using a specific config

```bash
sadr new --clipboard --smart --config db
```

---

## `sadr search` — Find records

### Interactive hub (no arguments)

```bash
sadr search
```

Opens a fuzzy-search TUI. Features:
- Type to filter by title in real time
- `↑` / `↓` to navigate results
- `Tab` to toggle deep search (also matches snippet and field content)
- `Enter` to open the selected record
- `Ctrl+D` to delete a record (with confirmation)
- `Ctrl+E` to export a record to HTML
- `Ctrl+C` to quit

### Text search

```bash
sadr search "retry logic"
```

Matches by:
1. Fuzzy match on titles
2. Exact tag match (any tag containing the query string)

```bash
# Also searches inside snippet body and all field values
sadr search "exponential" --deep
```

### View a specific record

```bash
sadr search --id 3
```

Prints the full record to stdout: title, type, snippet, all fields.

### Search personal vault

```bash
sadr search "auth" --global
```

### Filter by author (shared projects)

```bash
sadr search "retry" --user alice
```

---

## `sadr edit` — Edit a record

Opens a record in `$EDITOR` (or the editor set in global config):

```bash
sadr edit --id 5
sadr edit --id 5 --global    # edit from personal vault
```

---

## `sadr delete` — Delete a record

```bash
sadr delete --id 5
sadr delete --id 5 --force   # skip confirmation prompt
sadr delete --id 5 --global  # delete from personal vault
```

---

## `sadr export` — Export to HTML

Exports records to a single, self-contained HTML file in `.sadr/<user>/exports/`.

```bash
# Interactive hub to pick what to export
sadr export

# Export a specific record
sadr export --id 5

# Export all records
sadr export --all

# Export only records with specific tags
sadr export --tags "security,architecture"

# Export mode: full (default), adr (no snippet), snippet (no fields)
sadr export --all --mode adr
sadr export --id 5 --mode snippet

# Export from personal vault
sadr export --all --global
```

The exported HTML is completely standalone — no external dependencies, no server needed. Share it as a file or attach it to a ticket.

---

## `sadr ask` — Query a persona

Ask a specialized AI persona to analyze your architecture decisions. The persona answers based on the content of your records.

```bash
# Interactive: choose persona and type question
sadr ask

# Skip the selection prompts
sadr ask --role "dba" --question "how is the database schema evolving?"

# Filter which records are included
sadr ask --tags "security,auth"
sadr ask --field "team=platform"

# Include each record's snippet (compressed) in the prompt — more tokens, richer answer
sadr ask --complete

# Preview token count without calling the AI
sadr ask --dry-run

# Use personal vault records
sadr ask --global
```

Available built-in personas:

| Persona | Focus |
|---|---|
| Tech Lead | Architecture, technical debt, coupling, maintainability |
| DBA | Normalization, indexing, data integrity, query patterns |
| QA Engineer | Testability, edge cases, regression risks |
| Security Analyst | Vulnerabilities, auth gaps, data exposure |
| DevOps Engineer | Deployment risks, CI/CD, operational reliability |
| UX Designer | User flows, accessibility, UX impact of technical decisions |
| custom... | You describe the persona inline |

Answers are saved to `.sadr/<user>/answers/sadr-answer-XXXX-<persona>.md`.

---

## `sadr config` — Manage config

```bash
# Open project config in $EDITOR
sadr config

# Open global config
sadr config --global

# Set Gemini API key directly (no editor needed)
sadr config --set-api-key "AIza..."

# Validate all config files
sadr config --check

# Set up Jira interactively
sadr config --setup-jira

# Suppress the Jira-not-configured warning
sadr config --disable-jira-warning
```

---

## `sadr init` — Initialize

```bash
# Initialize project vault in current directory
sadr init

# Initialize personal global vault
sadr init --global

# Skip interactive preset selection
sadr init --preset minimal
sadr init --preset extended
```
