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
| `--complete` | Include each record's snippet (compressed) in the prompt |
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

Manage configuration files. With no flags, opens a project config in `$EDITOR`;
pass a `[name]` to open a specific project config (e.g. `sadr config db`).

```bash
sadr config [name] [flags]
```

| Flag | Description |
|---|---|
| `--global` | Open the global config (`~/.sadr/global-config.yaml`); created on first use |
| `--set-api-key <string>` | Write the Gemini API key directly to the global config |
| `--check` | Validate all project configs and report errors (add `--global` to check personal configs) |
| `--setup-jira` | Interactive Jira credentials setup (basic auth, bearer token, or OAuth 1.0a) |
| `--setup-jira-admin` | Generate an RSA key pair for a Jira OAuth 1.0a application link (run once per organization) |
| `--setup-admin` | Generate an admin token for privileged commands |
| `--disable-jira-warning` | Suppress the warning shown when Jira credentials exist but the project has no Jira URL |

The action flags above are mutually exclusive.

```bash
sadr config                       # open the default project config
sadr config db                    # open the project's "db" config
sadr config --global              # open the global config
sadr config --set-api-key "AIza..."
sadr config --check
sadr config --setup-jira
```

---

## `sadr doctor`

Validate records and flag changed files documented by conflicting (overlapping)
records. Deterministic — no AI. Built to run in CI as a merge gatekeeper; resolve
a conflict by deprecating the stale record. See
[CI Gatekeeper](../guide/ci-gatekeeper).

```bash
sadr doctor [flags]
```

| Flag | Description |
|---|---|
| `--ci` | Non-interactive CI mode with structured output for ChatOps |
| `--base <string>` | Base branch (or ref) of the pull request (default `main`) |
| `--apply <string>` | Comma-separated record IDs to deprecate — runs the apply phase |

It exits non-zero when there are orphan records or unresolved conflicts, so a
required status check blocks the merge.

```bash
sadr doctor --ci --base origin/main                  # detect phase (CI)
sadr doctor --ci --base origin/main --apply alice/1  # deprecate a stale record
```
