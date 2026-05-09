---
sidebar_position: 1
---

# CLI Reference

## Global flags

These flags are available on all commands:

| Flag | Description |
|---|---|
| `--version` | Print sadr version |
| `--help` | Show help |

---

## `sadr init`

Initialize a sadr vault.

```bash
sadr init [flags]
```

| Flag | Description |
|---|---|
| `--global` | Initialize personal vault at `~/.sadr/` |
| `--preset <string>` | Skip interactive preset selection: `minimal` or `extended` |

**Examples:**

```bash
sadr init                    # initialize project vault in current directory
sadr init --global           # initialize personal vault
sadr init --preset minimal   # skip preset prompt, use minimal schema
```

---

## `sadr new`

Capture a new record.

```bash
sadr new [flags]
sadr new adr [flags]
sadr new snippet [flags]
```

| Subcommand | Captures |
|---|---|
| `sadr new` | Full record: snippet + all ADR fields |
| `sadr new adr` | ADR only — all fields, no snippet step |
| `sadr new snippet` | Snippet only — no ADR fields |

### Flags

| Flag | Short | Description |
|---|---|---|
| `--title <string>` | | Record title — skips interactive wizard |
| `--global` | `-g` | Save to personal vault (`~/.sadr/`) |
| `--clipboard` | `-c` | Read snippet from clipboard |
| `--file <path>` | `-f` | Read snippet from file (max 10 MB) |
| `--diff` | `-d` | Read snippet from `git diff HEAD` |
| `--smart` | `-s` | AI pre-fills fields from snippet |
| `--model <string>` | | Override AI model (implies `--smart`) |
| `--fine-tuning` | | Prompt for a custom AI instruction (requires `--smart`) |
| `--config <name>` | | Use a specific config file (e.g. `db`, `api`) |

`--clipboard`, `--file`, and `--diff` are mutually exclusive.

**Examples:**

```bash
sadr new --clipboard --smart
sadr new --diff --smart --fine-tuning
sadr new snippet --file src/handler.go
sadr new adr --title "Use PostgreSQL for primary storage"
sadr new --clipboard --smart --config db --model gemini-2.5-pro
```

---

## `sadr search`

Search records by title, tags, or content.

```bash
sadr search [query] [flags]
```

With no arguments, opens the interactive TUI hub.

| Flag | Short | Description |
|---|---|---|
| `--id <string>` | | Show a specific record by ID. Supports `user/id` format |
| `--deep` | | Also search inside snippet body and field values |
| `--global` | `-g` | Search personal vault (`~/.sadr/`) |
| `--user <string>` | | Filter results by author username |

`--id` and `--deep` are mutually exclusive.

**Examples:**

```bash
sadr search                      # open interactive hub
sadr search "retry"              # fuzzy search by title + tag match
sadr search "retry" --deep       # include snippet and field content
sadr search --id 5               # show record #5
sadr search --id alice/3         # show alice's record #3
sadr search "auth" --user alice  # filter by author
sadr search "config" --global    # search personal vault
```

---

## `sadr edit`

Open a record in `$EDITOR`.

```bash
sadr edit [flags]
```

| Flag | Short | Description |
|---|---|---|
| `--id <string>` | | Record ID to edit |
| `--global` | `-g` | Edit from personal vault |

```bash
sadr edit --id 5
sadr edit --id alice/2
sadr edit --id 3 --global
```

---

## `sadr delete`

Delete a record.

```bash
sadr delete [flags]
```

| Flag | Short | Description |
|---|---|---|
| `--id <string>` | | Record ID to delete |
| `--force` | | Skip confirmation prompt |
| `--global` | `-g` | Delete from personal vault |

```bash
sadr delete --id 5
sadr delete --id 5 --force
sadr delete --id alice/2
```

---

## `sadr export`

Export records to a self-contained HTML file.

```bash
sadr export [flags]
```

With no flags, opens the interactive hub to select what to export.

| Flag | Short | Description |
|---|---|---|
| `--id <string>` | | Export a specific record |
| `--all` | | Export all records |
| `--tags <string>` | | Export records matching tags (comma-separated) |
| `--mode <string>` | | Export mode: `full` (default), `adr`, `snippet` |
| `--global` | `-g` | Export from personal vault |

`--id`, `--all`, and `--tags` are mutually exclusive.

**Export modes:**

| Mode | Includes |
|---|---|
| `full` | Snippet + all fields |
| `adr` | All fields only (no snippet) |
| `snippet` | Snippet only (no fields) |

```bash
sadr export --all
sadr export --id 5
sadr export --tags "security,auth"
sadr export --all --mode adr
sadr export --id 5 --mode snippet --global
```

---

## `sadr ask`

Ask a persona about your architecture decisions.

```bash
sadr ask [flags]
```

| Flag | Description |
|---|---|
| `--role <string>` | Persona name — skips the selector (e.g. `"dba"`, `"tech lead"`) |
| `--question <string>` | Question to ask — skips the text input |
| `--tags <string>` | Filter records by tags (comma-separated) |
| `--field <string>` | Filter records by field value (`key=value`) |
| `--complete` | Include full snippet content in the prompt |
| `--dry-run` | Show token estimate without calling the AI |
| `--global` / `-g` | Use personal vault records |

**Examples:**

```bash
sadr ask
sadr ask --role "dba" --question "how is indexing handled?"
sadr ask --tags "security" --dry-run
sadr ask --field "status=active" --complete
sadr ask --global
```

---

## `sadr config`

Manage configuration files.

```bash
sadr config [flags]
```

| Flag | Description |
|---|---|
| `--global` | Open global config (`~/.sadr/global-config.yaml`) in `$EDITOR` |
| `--set-api-key <string>` | Write the Gemini API key directly to global config |
| `--check` | Validate all config files and report errors |
| `--setup-jira` | Interactive Jira credentials setup |
| `--disable-jira-warning` | Suppress the "Jira credentials set but no project URL" warning |

```bash
sadr config                       # open project config
sadr config --global              # open global config
sadr config --set-api-key "AIza..." 
sadr config --check
sadr config --setup-jira
```
