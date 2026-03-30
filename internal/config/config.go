package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Fields []Field `yaml:"fields"`
}

type GlobalConfig struct {
	Editor   string   `yaml:"editor"`
	Language string   `yaml:"language"`
	AI       AIConfig `yaml:"ai"`
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

func validate(cfg Config) error {
	ValidTypes := []string{
		"text",
		"select",
		"multitext",
		"multiselect",
		"list",
	}
	for _, field := range cfg.Fields {
		if field.Name == "" {
			return errors.New("field name must not be empty")
		}
		found := false
		for _, validType := range ValidTypes {
			if field.Type == validType {
				found = true
			}
		}
		if !found {
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
