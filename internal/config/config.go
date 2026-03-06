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

// Config holds the application configuration.
type Config struct {
	URL      string `toml:"url"`
	Database string `toml:"database"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

func (c *Config) OdooURL() string      { return c.URL }
func (c *Config) OdooDatabase() string  { return c.Database }
func (c *Config) OdooUsername() string   { return c.Username }
func (c *Config) OdooPassword() string   { return c.Password }

// LoadFromEnv reads configuration from environment variables.
// Required: ODOO_URL, ODOO_DATABASE, ODOO_USERNAME, ODOO_PASSWORD.
func LoadFromEnv() (*Config, error) {
	cfg := &Config{
		URL:      os.Getenv("ODOO_URL"),
		Database: os.Getenv("ODOO_DATABASE"),
		Username: os.Getenv("ODOO_USERNAME"),
		Password: os.Getenv("ODOO_PASSWORD"),
	}

	var missing []string
	if cfg.URL == "" {
		missing = append(missing, "ODOO_URL")
	}
	if cfg.Database == "" {
		missing = append(missing, "ODOO_DATABASE")
	}
	if cfg.Username == "" {
		missing = append(missing, "ODOO_USERNAME")
	}
	if cfg.Password == "" {
		missing = append(missing, "ODOO_PASSWORD")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return cfg, nil
}

// LoadFromTOML reads configuration from a TOML file.
// Secret fields (username, password) should come from environment variables;
// this is intended for non-secret fields like URL and database.
func LoadFromTOML(path string) (*Config, error) {
	cfg := &Config{}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}
	return cfg, nil
}

// Merge overlays values from other onto c. Non-empty fields in other
// take precedence.
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
}
