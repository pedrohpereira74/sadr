---
sidebar_position: 2
---

# Config Schema Reference

## Project config (`.sadr/configs/*.yaml`)

Full schema with all supported fields:

```yaml
# Field definitions — defines what the wizard captures
fields:
  - name: <string>          # required. field identifier (used as section heading)
    type: <field-type>      # required. see field types below
    required: <bool>        # required. whether the wizard enforces a non-empty answer
    default: <string>       # optional. pre-filled default value
    options:                # required for select/multiselect types
      - option1
      - option2

# AI query settings
ask:
  limit: <int>              # max number of records passed to sadr ask (default: 50)
  range: <duration>         # only include records from the last N units (default: 6m)
                            # units: d (days), w (weeks), m (months), y (years)

# Jira project settings
jira:
  url: <string>             # your Jira base URL, e.g. https://company.atlassian.net
```

### Field types

| Type | Description | `options` required |
|---|---|---|
| `text` | Freeform textarea | No |
| `multitext` | Multiple freeform entries | No |
| `list` | Freeform, rendered as bullet list | No |
| `select` | Single-choice dropdown | Yes |
| `multiselect` | Multi-choice selector | Yes |
| `jira` | Jira card key — fetches card context for AI | No |

### Reserved field names

These names cannot be used in your schema — they are managed internally by sadr:

| Name | Description |
|---|---|
| `snippet` | The code snippet content |
| `file_ref` | The file path associated with the record |
| `status` | The record lifecycle status |

`title` and `tags` are special: they are always captured and don't need to be declared in the schema.

### Full example

```yaml
fields:
  - name: context
    type: text
    required: true

  - name: decision
    type: text
    required: true

  - name: alternatives
    type: list
    required: false

  - name: consequences
    type: text
    required: false

  - name: status
    type: select
    required: true
    default: proposed
    options:
      - proposed
      - active
      - deprecated
      - superseded

  - name: category
    type: multiselect
    required: false
    options:
      - security
      - performance
      - architecture
      - data
      - devops
      - frontend

  - name: team
    type: select
    required: false
    options:
      - platform
      - product
      - data
      - mobile

  - name: jira_card
    type: jira
    required: false

ask:
  limit: 40
  range: 3m

jira:
  url: https://your-company.atlassian.net
```

---

## Global config (`~/.sadr/global-config.yaml`)

```yaml
username: <string>       # your identifier in shared projects
editor: <string>         # editor command, e.g. vim, nano, "code --wait"
language: <string>       # language for AI output, e.g. English, Portuguese

ai:
  provider: gemini       # currently only gemini is supported
  api_key: <string>      # Gemini API key (stored plaintext — use api_key_env in shared envs)
  api_key_env: <string>  # env var name to read the key from (takes precedence over api_key)
  model: <string>        # Gemini model name (default: gemini-2.0-flash)
  ai_depth: <bool>       # if true, uses a Staff-level persona with more thorough output

jira:
  # Jira Cloud (API token):
  username: <string>         # your Jira account email
  token: <string>            # Jira API token
  token_env: <string>        # env var name to read the token from

  # Jira Server (OAuth1):
  consumer_key: <string>
  private_key_path: <string> # path to RSA private key file
  access_token: <string>
  access_token_secret: <string>

  # Jira Server (basic auth):
  password: <string>
  password_env: <string>

  disable_project_warning: <bool>  # suppress "Jira configured but no project URL" warning
```

### Precedence for AI key

1. `ai.api_key_env` (env var) — checked first
2. `ai.api_key` (plaintext in file)
3. `AI_API_KEY` environment variable
4. `GEMINI_API_KEY` environment variable
