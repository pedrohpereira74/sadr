package templates

import (
	"bytes"
	"html/template"
)

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

type ExportField struct {
	Key   string
	Value string
}

type ExportData struct {
	Title      string
	Type       string
	FileRef    string
	HasFileRef bool
	Snippet    string
	Tags       string
	Fields     []ExportField
}

const recordHTML = `<!DOCTYPE html>
<html><head>
<meta charset="utf-8">
<title>{{.Title}}</title>
<meta name="sadr-type" content="{{.Type}}">
{{- if .HasFileRef}}
<meta name="sadr-file-ref" content="{{.FileRef}}">
{{- end}}
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github.min.css">
<script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/highlight.min.js"></script>
<style>
body { font-family: system-ui, sans-serif; max-width: 800px; margin: 40px auto; padding: 0 20px; line-height: 1.6; }
p { white-space: pre-wrap; }
code { font-family: monospace; background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
pre { background: #f6f8fa; padding: 16px; border-radius: 6px; overflow-x: auto; tab-size: 4; }
pre code {
    display: block;
    white-space: pre-wrap;
    word-break: normal;
    overflow-wrap: anywhere;
    background: transparent;
    padding: 0; }
@media print {
	@page { margin: 0; }
	body { margin: 1.5cm; max-width: 100%; }
	p { page-break-inside: avoid; }
	h1, h2 { page-break-after: avoid; }
}
</style>
</head><body>
<h1>{{.Title}}</h1>
{{- if .Tags}}
<p><strong>Tags:</strong> {{.Tags}}</p>
{{- end}}
{{- if .Snippet}}
<h2>Snippet</h2>
<pre><code>{{.Snippet}}</code></pre>
{{- end}}
{{- range .Fields}}
<h2>{{.Key}}</h2>
<p>{{.Value}}</p>
{{- end}}
<script>hljs.highlightAll();</script>
</body></html>`

var recordTmpl = template.Must(template.New("record").Parse(recordHTML))

func RenderRecord(data ExportData) (string, error) {
	var buf bytes.Buffer
	if err := recordTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
