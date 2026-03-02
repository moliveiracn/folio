package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port         int    `yaml:"port"`
	DataDir      string `yaml:"data_dir"`
	PasswordHash string `yaml:"password_hash"`
	LogLevel     string `yaml:"log_level"`
	MaxUploadMB  int    `yaml:"max_upload_mb"`
}

// reads config.yaml from path and returns a filled Config.
func Load(path string) (*Config, error) {
	cfg := Config{
		Port:        8080,
		LogLevel:    "info",
		MaxUploadMB: 200,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validate checks that required fields are present and values make sense.
func (c *Config) validate() error {
	if c.PasswordHash == "" {
		return fmt.Errorf("password_hash is missing — run: folio init")
	}
	if c.DataDir == "" {
		return fmt.Errorf("data_dir cannot be empty")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be 1–65535, got %d", c.Port)
	}
	return nil
}

// DBPath returns the full path to folio.db inside DataDir.
func (c *Config) DBPath() string {
	return filepath.Join(c.DataDir, "folio.db")
}

// BooksDir returns the full path to the books folder inside DataDir.
func (c *Config) BooksDir() string {
	return filepath.Join(c.DataDir, "books")
}

// PluginsDir returns the full path to the plugins folder inside DataDir.
func (c *Config) PluginsDir() string {
	return filepath.Join(c.DataDir, "plugins")
}
