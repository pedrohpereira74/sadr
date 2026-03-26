package config

import (
	"os"
	"testing"
)

func TestLoadFromStringParseFields(t *testing.T) {
	yamlData := `
fields:
  - name: title
    type: text
    required: true
  - name: status
    type: select
    required: false
    options: [proposed, accepted]
`
	cfg, err := LoadFromString(yamlData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(cfg.Fields))
	}
	if cfg.Fields[0].Name != "title" {
		t.Errorf("expected first field to be 'title', got '%s'", cfg.Fields[0].Name)
	}
	if cfg.Fields[1].Type != "select" {
		t.Errorf("expected second field to be 'select', got '%s'", cfg.Fields[1].Type)
	}
}


func TestLoadFromStringParseFieldDefault(t *testing.T) {
	yamlData := `
fields:
  - name: status
    type: select
    required: false
    options: [proposed, accepted]
    default: proposed
`
	cfg, err := LoadFromString(yamlData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(cfg.Fields))
	}
	if cfg.Fields[0].Default != "proposed" {
		t.Errorf("expected default to be 'proposed', got '%s'", cfg.Fields[0].Default)
	}
}

func TestLoadFromStringParseRequiredAsBoolPointer(t *testing.T) {
	yamlData := `
fields:
  - name: title
    type: text
    required: true
  - name: context
    type: text
    required: false
`
	cfg, err := LoadFromString(yamlData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Fields[0].Required == nil {
		t.Fatal("expected Required to be set, got nil")
	}
	if *cfg.Fields[0].Required != true {
		t.Errorf("expected Required true, got false")
	}
	if *cfg.Fields[1].Required != false {
		t.Errorf("expected Required false, got true")
	}
}

func TestLoadGlobalFromStringParseAIConfig(t *testing.T) {
	yamlData := `
editor: "vim"
ai:
  provider: "gemini"
  api_key_env: "GEMINI_API_KEY"
`
	cfg, err := LoadGlobalFromString(yamlData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AI.Provider != "gemini" {
		t.Errorf("expected provider 'gemini', got '%s'", cfg.AI.Provider)
	}
	if cfg.AI.APIKey != "" {
		t.Errorf("expected empty APIKey, got %s", cfg.AI.APIKey)
	}
}

func TestLoadFromStringRejectsInvalidFieldType(t *testing.T) {
	yamlData := `
fields:
  - name: title
    type: banana
    required: true
`
	_, err := LoadFromString(yamlData)
	if err == nil {
		t.Fatal("expected error for invalid type 'banana', got nil")
	}
}

func TestLoadFromStringRejectsMissingRequired(t *testing.T) {
	yamlData := `
fields:
  - name: title
    type: text
`
	_, err := LoadFromString(yamlData)
	if err == nil {
		t.Fatal("expected error for missing 'required', got nil")
	}
}

func TestLoadFromStringRejectsSelectWithoutOptions(t *testing.T) {
	yamlData := `
fields:
  - name: status
    type: select
    required: true
`
	_, err := LoadFromString(yamlData)
	if err == nil {
		t.Fatal("expected error for select without options, got nil")
	}
}

func TestLoadFromStringRejectsEmptyFieldName(t *testing.T) {
	yamlData := `
fields:
  - name: ""
    type: text
    required: true
`
	_, err := LoadFromString(yamlData)
	if err == nil {
		t.Fatal("expected error for empty field name, got nil")
	}
}

func TestLoadFromFileReadsYAML(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.yaml"
	content := `
fields:
  - name: title
    type: text
    required: true

`
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(cfg.Fields))
	}
}

func TestLoadGlobalFromFileReadsYAML(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.yaml"
	content := `
editor: "vim"
ai:
  provider: "gemini"
  api_key_env: "GEMINI_API_KEY"
`
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg, err := LoadGlobalFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Editor != "vim" {
		t.Errorf("expected editor 'vim', got '%s'", cfg.Editor)
	}
}
