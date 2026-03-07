package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// ConfigReader provides access to configuration values.
type ConfigReader interface {
	OdooURL() string
	OdooDatabase() string
	OdooUsername() string
	OdooPassword() string
}

// ExtraField describes a custom Odoo field to fetch for a model.
type ExtraField struct {
	Name  string `toml:"name"`  // display name (e.g. "product_owner")
	Field string `toml:"field"` // Odoo field name (e.g. "x_studio_productowner")
	Type  string `toml:"type"`  // Odoo type: many2one, char, boolean, integer, float
}

// ModelConfig holds per-model configuration.
type ModelConfig struct {
	ExtraFields []ExtraField `toml:"extra_fields"`
}

// Config holds the application configuration.
type Config struct {
	URL      string                 `toml:"url"`
	Database string                 `toml:"database"`
	Username string                 `toml:"username"`
	Password string                 `toml:"-"`
	Models   map[string]ModelConfig `toml:"models"`
}

func (c *Config) OdooURL() string      { return c.URL }
func (c *Config) OdooDatabase() string  { return c.Database }
func (c *Config) OdooUsername() string   { return c.Username }
func (c *Config) OdooPassword() string   { return c.Password }

// LoadFromEnv reads configuration from environment variables.
// It reads whatever env vars are set without requiring any.
// Use Validate to check that all required fields are present.
func LoadFromEnv() *Config {
	return &Config{
		URL:      os.Getenv("ODOO_URL"),
		Database: os.Getenv("ODOO_DATABASE"),
		Username: os.Getenv("ODOO_USERNAME"),
		Password: os.Getenv("ODOO_PASSWORD"),
	}
}

// Validate checks that all required configuration fields are set.
// Returns an error listing missing fields.
func (c *Config) Validate() error {
	var missing []string
	if c.URL == "" {
		missing = append(missing, "URL")
	}
	if c.Database == "" {
		missing = append(missing, "Database")
	}
	if c.Username == "" {
		missing = append(missing, "Username")
	}
	if c.Password == "" {
		missing = append(missing, "Password")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required config fields: %v", missing)
	}
	return nil
}

// LoadFromTOML reads configuration from a TOML file.
// Returns an error if the file contains a password key — passwords
// must come from environment variables only, never from config files.
func LoadFromTOML(path string) (*Config, error) {
	cfg := &Config{}
	meta, err := toml.DecodeFile(path, cfg)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}
	if meta.IsDefined("password") {
		return nil, fmt.Errorf("config file %s contains a password field; passwords must be set via ODOO_PASSWORD env var, not in config files", path)
	}
	return cfg, nil
}

// Merge overlays values from other onto c. Non-empty fields in other
// take precedence. Model configs from other are merged key-by-key,
// with overlay models replacing base models of the same name.
func (c *Config) Merge(other *Config) {
	if other.URL != "" {
		c.URL = other.URL
	}
	if other.Database != "" {
		c.Database = other.Database
	}
	if other.Username != "" {
		c.Username = other.Username
	}
	if other.Password != "" {
		c.Password = other.Password
	}
	if other.Models != nil {
		if c.Models == nil {
			c.Models = make(map[string]ModelConfig)
		}
		for k, v := range other.Models {
			c.Models[k] = v
		}
	}
}
