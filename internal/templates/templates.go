package templates

const MinimalConfig = `# .sadr/config.yaml
# Customize fields to match your workflow.

fields:
  - name: title
    type: text
    required: true

  - name: tags
    type: multiselect
    required: true
    options: [architecture, api, database, security, performance, tooling, infrastructure, bugfix]

quick_fields: [title, tags]
`

const ExtendedConfig = `# .sadr/config.yaml
# Customize fields to match your workflow.

fields:
  - name: title
    type: text
    required: true

  - name: tags
    type: multiselect
    required: true
    options: [architecture, api, database, security, performance, tooling, infrastructure, bugfix]

  - name: status
    type: select
    required: true
    options: [proposed, accepted, deprecated, superseded]
    default: proposed

  - name: context
    type: text
    required: true

  - name: decision
    type: text
    required: true

  - name: alternatives
    type: text
    required: false

  - name: consequences
    type: text
    required: false

quick_fields: [title, tags]
`

const GlobalConfig = `# Global sadr config — personal preferences
# This file is never versioned.

editor: 

ai:
  provider: gemini
  api_key_env: GEMINI_API_KEY
`
