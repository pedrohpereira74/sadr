---
sidebar_position: 1
---

# Configuration

sadr uses three configuration files with distinct scopes:

| File | Scope | Git-tracked |
|---|---|---|
| `.sadr/configs/default-config.yaml` | Project fields, Jira URL, `ask` limits | Yes |
| `~/.sadr/configs/default-config.yaml` | Personal fields for your private vault | No |
| `~/.sadr/global-config.yaml` | Username, editor, language, AI credentials, Jira credentials | No |

---

## Global config (`~/.sadr/global-config.yaml`)

Created by `sadr init --global`. Contains your personal preferences and credentials.

```yaml
username: alice
editor: vim
language: English

ai:
  provider: gemini
  api_key: ""            # set via: sadr config --set-api-key "key"
  api_key_env: ""        # alternatively, read from this env var
  model: gemini-3-flash-preview
  ai_depth: false        # if true, uses a more thorough Staff-level AI persona

jira:
  username: alice@company.com
  token: ""              # Jira API token
  token_env: JIRA_TOKEN  # alternatively, read from this env var
  # OAuth1 (server Jira):
  consumer_key: ""
  private_key_path: ""
  access_token: ""
  access_token_secret: ""
  disable_project_warning: false
```

Open it in your editor:

```bash
sadr config --global
```

---

## Project schema config (`.sadr/configs/default-config.yaml`)

Defines the fields captured when running `sadr new` in this project. Committed to git so the whole team shares the same schema.

The two examples below are exactly what `sadr init --preset minimal` and
`sadr init --preset extended` generate.

### Minimal preset

```yaml
fields:
  - name: title
    type: text
    required: true

  - name: tags
    type: multiselect
    required: true
    options: [architecture, api, database, security, performance, tooling, infrastructure, bugfix]

ask:
  limit: 50
  range: 6m
```

### Extended preset

```yaml
fields:
  - name: title
    type: text
    required: true

  - name: tags
    type: multiselect
    required: true
    options: [architecture, api, database, security, performance, tooling, infrastructure, bugfix]

  - name: context
    type: text
    required: false

  - name: decision
    type: text
    required: false

  - name: alternatives
    type: list
    required: false

  - name: consequences
    type: text
    required: false

ask:
  limit: 50
  range: 6m
```

Add your own fields (`select`, `multiselect`, `jira`, …) and a `jira:` block on
top of either preset — see [Field types](#field-types) below.

---

## Field types

### `text`

A freeform text field. Displayed as a textarea in the wizard.

```yaml
- name: context
  type: text
  required: true
```

### `list`

A freeform field where each item becomes a bullet point. The AI formats multi-item answers as comma-separated, which sadr converts to `- item` bullets.

```yaml
- name: alternatives
  type: list
  required: false
```

### `select`

A single-choice field. Rendered as a dropdown in the wizard.

```yaml
- name: risk
  type: select
  required: true
  options: [low, medium, high]
  default: low
```

### `multiselect`

A multi-choice field. The user picks one or more options.

```yaml
- name: category
  type: multiselect
  required: false
  options: [security, performance, architecture, data, devops]
```

### `jira`

A Jira card key field. When Jira is configured, sadr fetches the card's summary and description and passes them to the AI as extra context.

```yaml
- name: jira_card
  type: jira
  required: false
```

> See [Jira Integration](./ai-features#jira-integration) for setup instructions.

---

## Multiple configs per project

You can have multiple schemas for different capture contexts:

```
.sadr/configs/
├── default-config.yaml   ← used when no --config flag is passed
├── db.yaml               ← database-specific fields
└── api.yaml              ← API-specific fields
```

Use `--config` to select one:

```bash
sadr new --diff --smart --config db
sadr new --clipboard --config api
```

If more than one config exists and `--config` is not passed, sadr prompts you to choose.

---

## Checking for errors

Validate all config files in the project:

```bash
sadr config --check
```

This reports any YAML errors, missing required fields, or invalid field types without opening an editor.
