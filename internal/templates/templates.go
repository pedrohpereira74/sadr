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
    required: false
    options: [proposed, accepted, deprecated, superseded]
    default: proposed

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

`

const GlobalConfig = `# Global sadr config — personal preferences
# This file is never versioned.

editor: 

language: "english"

ai:
  provider: "gemini"
  api_key: ""
  model: "gemini-3-flash-preview"
  ai_depth: true
`
