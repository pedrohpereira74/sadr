---
sidebar_position: 3
---

# AI Features

sadr uses the Gemini API for two distinct AI-powered features: **assisted capture** (`--smart`) and **architectural queries** (`sadr ask`).

## Setup

Set your API key once:

```bash
sadr config --set-api-key "AIza..."
```

Or use an environment variable:

```yaml
# ~/.sadr/global-config.yaml
ai:
  api_key_env: GEMINI_API_KEY
```

---

## `--smart`: AI-assisted capture

When you pass `--smart` to `sadr new`, the AI reads your snippet and pre-fills all fields defined in your schema.

```bash
sadr new --clipboard --smart
sadr new --diff --smart
sadr new --file src/handler.go --smart
```

### What the AI fills in

The AI respects your schema. For a schema with fields `context`, `decision`, `alternatives`, `consequences`:

- `context` → explains what problem the code solves and why it exists
- `decision` → describes what was chosen and the rationale
- `alternatives` → lists what else was considered
- `consequences` → describes trade-offs and downstream implications
- `title` → always generated as a short phrase (≤ 10 words)
- `tags` → inferred from the snippet content

`select` and `multiselect` fields are **not filled by the AI** — they require a human choice. `jira` fields are also excluded from AI filling.

### Fine-tuning the AI

Guide the AI with a custom instruction for this specific capture:

```bash
sadr new --diff --smart --fine-tuning
```

You will be prompted:

```
add a custom instruction to guide the AI (optional):
e.g. focus on security implications, this is a React component...
```

Examples of useful fine-tuning hints:
- `"This is a migration script — focus on data safety and rollback risks"`
- `"We're replacing legacy code — explain what the old approach was"`
- `"This is a temporary workaround — note that in the decision field"`

The hint is stored in the record's frontmatter (`fine_tuning_hint`) and reused if you re-run AI on the record.

### Depth mode

For more thorough, senior-level analysis, enable `ai_depth` in your global config:

```yaml
# ~/.sadr/global-config.yaml
ai:
  ai_depth: true
```

With depth mode on, the AI switches to a Staff Engineer persona and writes longer, more opinionated `text` and `list` fields — analyzing trade-offs, hidden coupling, and architectural debt.

### Override the model

```bash
sadr new --clipboard --smart --model gemini-2.5-pro
```

Any Gemini model name is accepted. If omitted, uses the model in your global config (default: `gemini-2.0-flash`).

### Fallback behavior

If the AI call fails (network error, invalid key, rate limit), sadr falls back to the interactive wizard automatically after a 3-second warning. No record is lost.

---

## `sadr ask`: Architectural queries

Query a senior persona about your saved decisions. The persona reads your records and answers based solely on what's documented.

```bash
sadr ask
```

### How it works

1. sadr loads all `active` records from your vault (filtered by `ask.limit` and `ask.range` in your config)
2. You choose a persona (Tech Lead, DBA, Security Analyst, etc.)
3. You type a question
4. sadr builds a prompt with the persona's instruction, your records, and optionally the source code context
5. The AI responds; the answer is saved to `.sadr/<user>/answers/`

### Filtering records for the query

```bash
# Only pass records tagged "security"
sadr ask --tags "security"

# Only pass records where field "team" equals "platform"
sadr ask --field "team=platform"
```

### Including snippet content

By default, snippets are compressed before being sent (to reduce tokens). Pass `--complete` to include the full snippet content:

```bash
sadr ask --complete
```

### Dry run: estimate cost before calling

```bash
sadr ask --dry-run
```

Output:

```
18 records | ~45,200 chars | ~11,300 tokens estimated
proceed?
```

### `ask` config options

Control how many records are included:

```yaml
# .sadr/configs/default-config.yaml
ask:
  limit: 50     # max records to include (most recent)
  range: 6m     # only records from the last 6 months (d/w/m/y units)
```

### Answer files

Each `sadr ask` call saves the response to:

```
.sadr/<user>/answers/sadr-answer-0001-tech-lead.md
```

Format:

```markdown
**Persona:** Tech Lead
**Question:** How is error handling evolving in this codebase?

---

Based on the documented decisions...
```

---

## Jira integration

When a field of type `jira` is defined in your schema and Jira is configured, sadr fetches the card's summary and description and passes them to the AI alongside the snippet. This gives the AI the business context for the change.

### Setup

Configure Jira credentials in your global config:

```bash
sadr config --setup-jira
```

This prompts you for your Jira URL, username, and API token (or OAuth1 credentials for on-premise Jira Server).

Add the project URL to your project config:

```yaml
# .sadr/configs/default-config.yaml
jira:
  url: https://your-company.atlassian.net
```

Add a `jira` type field to your schema:

```yaml
fields:
  - name: jira_card
    type: jira
    required: false
```

### Usage

When you run `sadr new --smart` and a `jira` field is present, sadr prompts:

```
jira card key for 'jira_card':
e.g. PROJ-123
```

Enter the card key (e.g. `PROJ-456`). sadr fetches the card, includes its content in the AI prompt, and stores the key in the record field.

If Jira is not reachable or the card is not found, sadr continues without it — no error, just a warning.
