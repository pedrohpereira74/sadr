package templates

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/yuin/goldmark"
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

ask:
  limit: 50
  range: 6m
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
`

const GlobalConfig = `# Global sadr config — personal preferences
# This file is never versioned.

username: ""

editor:

language: "english"

ai:
  provider: "gemini"
  api_key: ""
  model: "gemini-3-flash-preview"
  ai_depth: true

# jira:
#   url: "https://yourorg.atlassian.net"
#   email: "you@example.com"
#   api_token: ""
#   api_token_env: "JIRA_API_TOKEN"
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

const markdownHTML = `<!DOCTYPE html>
<html><head>
<meta charset="utf-8">
<title>{{.Title}}</title>
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github.min.css">
<script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/highlight.min.js"></script>
<style>
body { font-family: system-ui, sans-serif; max-width: 800px; margin: 40px auto; padding: 0 20px; line-height: 1.6; }
code { font-family: monospace; background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
pre { background: #f6f8fa; padding: 16px; border-radius: 6px; overflow-x: auto; tab-size: 4; }
pre code { display: block; background: transparent; padding: 0; }
h1 { border-bottom: 2px solid #eee; padding-bottom: 8px; }
h2 { margin-top: 2em; color: #333; }
@media print {
	@page { margin: 0; }
	body { margin: 1.5cm; max-width: 100%; }
	h1, h2 { page-break-after: avoid; }
}
</style>
</head><body>
<h1>{{.Title}}</h1>
{{.Body}}
<script>hljs.highlightAll();</script>
</body></html>`

var markdownTmpl = template.Must(template.New("markdown").Parse(markdownHTML))

func RenderMarkdownHTML(title, content string) (string, error) {
	var mdBuf bytes.Buffer
	if err := goldmark.Convert([]byte(content), &mdBuf); err != nil {
		return "", fmt.Errorf("failed to convert markdown: %v", err)
	}

	var buf bytes.Buffer
	data := struct {
		Title string
		Body  template.HTML
	}{
		Title: title,
		Body:  template.HTML(mdBuf.String()),
	}
	if err := markdownTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
