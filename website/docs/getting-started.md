---
sidebar_position: 2
---

# Getting Started

## Installation

### macOS / Linux — native script

```bash
curl -sSfL https://raw.githubusercontent.com/pedrohpereira74/sadr/main/install.sh | sh
```

### macOS / Linux — Homebrew

```bash
brew install pedrohpereira74/homebrew-tap/sadr
```

### Windows — Scoop

```bash
scoop bucket add pedrohpereira74 https://github.com/pedrohpereira74/scoop-bucket
scoop install sadr
```

:::tip
If you run into permission errors on Windows, run the commands in an elevated terminal (Run as Administrator).
:::

### Debian / Ubuntu

```bash
wget https://github.com/pedrohpereira74/sadr/releases/latest/download/sadr_linux_amd64.deb
sudo apt install ./sadr_linux_amd64.deb
```

### Fedora / Red Hat

```bash
sudo rpm -i https://github.com/pedrohpereira74/sadr/releases/latest/download/sadr_linux_amd64.rpm
```

### Go (from source)

Requires Go 1.26+:

```bash
go install github.com/pedrohpereira74/sadr@latest
```

---

## Step 1: Global setup (one-time)

Create your personal vault and set your preferences:

```bash
sadr init --global
```

This creates `~/.sadr/` with a default config and prompts you for:
- **Username** — used to identify your records in shared projects
- **Editor** — fallback when `$EDITOR` / `$VISUAL` are not set
- **Language** — language for AI-generated content (e.g. `English`, `Portuguese`)

Then add your Gemini API key to enable AI-powered features:

```bash
sadr config --set-api-key "your-gemini-api-key"
```

:::info
Get a Gemini API key at [ai.google.dev](https://ai.google.dev) (a free tier is available). The default model is `gemini-3-flash-preview`.
:::

---

## Step 2: Initialize a project vault

Inside a git repository, run:

```bash
sadr init
```

This creates a `.sadr/` directory with:

```
.sadr/
└── configs/
    └── default-config.yaml   ← your team's field schema
```

Commit it so the whole team shares the same schema:

```bash
git add .sadr/
git commit -m "chore: initialize sadr"
```

---

## Step 3: Capture your first record

**From clipboard, AI-assisted (recommended):**

```bash
# 1. Copy some code to your clipboard
# 2. Run:
sadr new --clipboard --smart
```

sadr will analyze the snippet and pre-fill fields like title, context, and decision. Review each suggestion and press Enter to accept or type to override.

**From the current git diff:**

```bash
sadr new --diff --smart
```

**From a file:**

```bash
sadr new snippet --file src/api/client.go
```

**ADR only (no snippet):**

```bash
sadr new adr
```

The record is saved to `.sadr/<your-username>/records/` as a Markdown file:

```
.sadr/
└── alice/
    └── records/
        └── sadr-record-0001-retry-logic.md
```

---

## Step 4: Browse and search

Open the **interactive hub** (no query needed):

```bash
sadr search
```

Use `↑↓` to navigate, type to fuzzy-filter, `Tab` to toggle deep search, `Enter` to open the selected record, `Ctrl+D` to delete, and `Ctrl+E` to export.

Or search from the command line:

```bash
sadr search "retry logic"
sadr search "auth" --deep       # also searches inside snippet body and fields
sadr search --id 3              # view record #3 in full
```

---

## What's next?

- [Configuration](./guide/configuration) — define custom fields for your team
- [Usage Guide](./guide/usage) — all commands and flags in depth
- [AI Features](./guide/ai-features) — get the most out of `--smart` and `sadr ask`
