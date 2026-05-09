---
sidebar_position: 1
---

# Architecture Overview

## Directory layout

```
sadr/
├── cmd/                   Cobra CLI commands (one file per command)
│   ├── root.go            Root command, help template
│   ├── new.go             sadr new (+ adr, snippet subcommands)
│   ├── search.go          sadr search
│   ├── edit.go            sadr edit
│   ├── delete.go          sadr delete
│   ├── export.go          sadr export
│   ├── ask.go             sadr ask
│   ├── config.go          sadr config
│   ├── init.go            sadr init
│   ├── helpers.go         Shared utilities (path resolution, TUI widgets, editor)
│   ├── inject.go          Dependency injection hooks (for testing)
│   ├── setupjira.go       sadr config --setup-jira flow
│   └── setupadmin.go      Admin setup helpers
│
├── internal/
│   ├── ai/                Gemini API client, prompt builder, response parser
│   ├── ask/               Persona definitions and ask prompt builder
│   ├── compress/          Snippet compression (reduces token count for AI)
│   ├── config/            YAML config loading and validation
│   ├── discover/          Finds .sadr/ directory by walking up the tree
│   ├── dryrun/            Token estimation for sadr ask --dry-run
│   ├── enricher/          Context enrichers for sadr ask (source code, Jira)
│   ├── filepicker/        TUI file picker (used in new --diff to select changed files)
│   ├── hub/               Interactive TUI hub (Bubble Tea)
│   ├── jira/              Jira API client (Cloud token + Server OAuth1)
│   ├── model/             Record struct, field constants, tag parsing
│   ├── search/            Fuzzy match, tag filter, deep content search
│   ├── storage/           Read/write records to disk (YAML frontmatter + Markdown)
│   ├── templates/         HTML export templates
│   ├── ui/                Terminal output helpers (Error, Info, Success, Warning)
│   └── wizard/            Interactive field wizard (Bubble Tea)
│
└── main.go
```

---

## Data flow: `sadr new --diff --smart`

```
sadr new --diff --smart
  │
  ├─ cmd/new.go: readSnippetFromSource()
  │   └─ exec git diff HEAD → snippetContent
  │
  ├─ cmd/new.go: maybeCaptureSmartSnippet()
  │   └─ opens editor if snippet is empty + smart (not needed here)
  │
  ├─ cmd/new.go: collectJiraContext()     ← if jira field in schema
  │   └─ internal/jira: fetch card summary + description
  │
  ├─ cmd/new.go: collectFineTuningHint() ← if --fine-tuning
  │   └─ TUI textarea prompt
  │
  ├─ cmd/new.go: loadAISuggestions()
  │   ├─ internal/compress: ZipSnippet()      reduce token count
  │   ├─ internal/ai: BuildPrompt()           assemble Gemini prompt
  │   └─ internal/ai: GenerateText()          HTTP call to Gemini API
  │       └─ internal/ai: ParseResponse()     JSON → map[string]string
  │
  ├─ cmd/new.go: wizardRunner()
  │   └─ internal/wizard: Run()
  │       └─ Bubble Tea TUI — pre-filled with AI suggestions
  │           user reviews each field, accepts or overrides
  │
  ├─ cmd/new.go: buildRecordFromWizard()
  │   └─ internal/model: NewRecordWithOptions()
  │       populate fields, tags, snippet, file_ref, status
  │
  └─ internal/storage: SaveRecord()
      ├─ build YAML frontmatter
      ├─ format Markdown body
      └─ write sadr-record-<id>-<slug>.md
```

---

## Storage layer

Records are stored as plain Markdown files. `storage.Storage` handles:

- **Write** (`SaveRecord`): builds frontmatter + body, acquires a mutex, finds the next available ID, writes atomically via `O_CREATE|O_EXCL`
- **Read** (`LoadRecord`): splits on `---`, parses YAML frontmatter, parses `##` sections from body
- **List** (`ListRecordEntries`): reads all `.md` files in the directory, sorted by filename (which sorts by ID)
- **ID allocation**: scans existing filenames to find the current max ID, increments by 1

The mutex in `Storage` ensures concurrent `sadr new` calls in the same process don't race. Concurrent processes in the same directory rely on `O_EXCL` (atomic file creation) to avoid duplicates.

---

## Path resolution

`discover.FindSadrDir()` walks up the directory tree from the current working directory until it finds a `.sadr/` directory, similar to how git finds `.git/`. If none is found, it falls back to `~/.sadr/` with a user prompt.

Within `.sadr/`, paths are organized per username:

```
.sadr/
└── <username>/
    ├── records/    ← sadr-record-*.md
    ├── exports/    ← sadr-export-*.html
    └── answers/    ← sadr-answer-*.md
```

---

## AI layer

`internal/ai` is a thin HTTP client for the Gemini API. It does not use the Gemini SDK — it calls the REST endpoint directly to avoid a heavy dependency.

The prompt template enforces:
- JSON-only output (no markdown in the response)
- Exact field keys matching the schema
- Language from the user's global config
- Optional depth mode (switches persona to Staff Engineer)
- Optional Jira context and fine-tuning hint sections

`ParseResponse` is lenient: it handles both string and array values in the JSON, strips accidental markdown code fences, and normalizes keys to lowercase.

---

## TUI components

sadr uses [Bubble Tea](https://github.com/charmbracelet/bubbletea) for all interactive terminal UI:

| Component | Package | Used in |
|---|---|---|
| Search hub | `internal/hub` | `sadr search`, `sadr export` (no args) |
| Field wizard | `internal/wizard` | `sadr new` (field-by-field capture) |
| File picker | `internal/filepicker` | `sadr new --diff` (select changed files) |
| Select widget | `cmd/helpers.go` | persona selection, config selection, confirmations |
| Textarea widget | `cmd/helpers.go` | question input, fine-tuning hint |

---

## Testing strategy

Tests live alongside the code they test (`*_test.go`). The `cmd/inject.go` file exposes hooks that replace real implementations (clipboard reader, editor opener, wizard runner, AI calls) with test doubles during unit tests. This allows testing the full command flow without external dependencies.

Integration-style tests in `cmd/setup_test.go` create temporary `.sadr/` directories on disk and exercise the full storage layer.
