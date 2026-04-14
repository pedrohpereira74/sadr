package admin

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const configFile = "admin.yaml"

type Config struct {
	TokenHash string `yaml:"token_hash"`
}

func RequireAdmin(sadrRoot string) error {
	cfg, err := loadConfig(sadrRoot)
	if err != nil {
		return fmt.Errorf("failed to read admin config: %w", err)
	}
	if cfg.TokenHash == "" {
		return fmt.Errorf("admin not configured for this project — run: sadr config --setup-admin")
	}

	token := os.Getenv("SADR_ADMIN_TOKEN")
	if token == "" {
		return fmt.Errorf("this command requires admin authorization — set SADR_ADMIN_TOKEN")
	}

	if subtle.ConstantTimeCompare([]byte(hashToken(token)), []byte(cfg.TokenHash)) != 1 {
		return fmt.Errorf("invalid SADR_ADMIN_TOKEN")
	}
	return nil
}

func Setup(sadrRoot string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := hex.EncodeToString(raw)

	if err := saveConfig(sadrRoot, Config{TokenHash: hashToken(token)}); err != nil {
		return "", fmt.Errorf("failed to save admin config: %w", err)
	}
	return token, nil
}

func IsConfigured(sadrRoot string) bool {
	cfg, err := loadConfig(sadrRoot)
	return err == nil && cfg.TokenHash != ""
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func loadConfig(sadrRoot string) (Config, error) {
	data, err := os.ReadFile(filepath.Join(sadrRoot, configFile))
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, err
	}
	var cfg Config
	return cfg, yaml.Unmarshal(data, &cfg)
}

func saveConfig(sadrRoot string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(sadrRoot, configFile), data, 0644)
}
