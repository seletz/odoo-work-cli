package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed default_config.toml
var defaultConfigTemplate []byte

// DefaultConfigTemplate returns the embedded default configuration template.
func DefaultConfigTemplate() []byte {
	return defaultConfigTemplate
}

// InstallConfig writes the default configuration template to the given path.
// Returns an error if the file already exists. Creates parent directories
// as needed.
func InstallConfig(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists: %s", path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	if err := os.WriteFile(path, defaultConfigTemplate, 0o644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
