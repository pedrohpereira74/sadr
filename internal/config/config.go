package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/pedrohpereira74/sadr/internal/model"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Fields []Field           `yaml:"fields"`
	Ask    AskConfig         `yaml:"ask"`
	Jira   ProjectJiraConfig `yaml:"jira"`
}

type AskConfig struct {
	Limit int    `yaml:"limit"`
	Range string `yaml:"range"`
}

type GlobalConfig struct {
	Username string     `yaml:"username"`
	Editor   string     `yaml:"editor"`
	Language string     `yaml:"language"`
	AI       AIConfig   `yaml:"ai"`
	Jira     JiraConfig `yaml:"jira"`
}

type JiraConfig struct {
	Username               string `yaml:"username"`
	Password               string `yaml:"password"`
	PasswordEnv            string `yaml:"password_env"`
	Token                  string `yaml:"token"`
	TokenEnv               string `yaml:"token_env"`
	ConsumerKey            string `yaml:"consumer_key"`
	PrivateKeyPath         string `yaml:"private_key_path"`
	AccessToken            string `yaml:"access_token"`
	AccessTokenSecret      string `yaml:"access_token_secret"`
	DisableProjectWarning  bool   `yaml:"disable_project_warning"`
}

type ProjectJiraConfig struct {
	URL string `yaml:"url"`
}
type AIConfig struct {
	Provider  string `yaml:"provider"`
	APIKey    string `yaml:"api_key"`
	APIKeyEnv string `yaml:"api_key_env"`
	Model     string `yaml:"model"`
	AIDepth   bool   `yaml:"ai_depth"`
}

type Field struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Required *bool    `yaml:"required"`
	Default  string   `yaml:"default"`
	Options  []string `yaml:"options"`
}

func LoadFromString(data string) (Config, error) {
	cfg := Config{}
	err := yaml.Unmarshal([]byte(data), &cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, validate(cfg)
}

func LoadFromFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	return LoadFromString(string(data))
}

func LoadGlobalFromString(data string) (GlobalConfig, error) {
	cfg := GlobalConfig{}
	err := yaml.Unmarshal([]byte(data), &cfg)

	return cfg, err
}

func LoadGlobalFromFile(path string) (GlobalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return GlobalConfig{}, err
	}
	return LoadGlobalFromString(string(data))
}

func SaveGlobalConfig(path string, cfg GlobalConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func validate(cfg Config) error {
	validTypes := map[string]bool{
		"text":        true,
		"select":      true,
		"multiselect": true,
		"list":        true,
		"jira":        true,
	}
	reserved := map[string]bool{
		model.FieldSnippet: true,
		model.FieldFileRef: true,
		model.FieldStatus:  true,
	}
	seen := make(map[string]bool, len(cfg.Fields))
	for _, field := range cfg.Fields {
		if field.Name == "" {
			return errors.New("field name must not be empty")
		}
		if reserved[field.Name] {
			return fmt.Errorf("field name %q is reserved", field.Name)
		}
		if seen[field.Name] {
			return fmt.Errorf("duplicate field name: %q", field.Name)
		}
		seen[field.Name] = true
		if !validTypes[field.Type] {
			return fmt.Errorf("invalid type '%s' for field '%s'", field.Type, field.Name)
		}
		if field.Required == nil {
			return fmt.Errorf("required field '%s' not set", field.Name)
		}
		if (field.Type == "select" || field.Type == "multiselect") && len(field.Options) == 0 {
			return fmt.Errorf("field '%s' is type '%s', but has no options defined", field.Name, field.Type)
		}
	}
	return nil
}
