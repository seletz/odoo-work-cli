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

// Filter describes a default query filter for a model.
type Filter struct {
	Field string `toml:"field"` // Odoo field path (e.g. "company_id.name")
	Op    string `toml:"op"`    // comparison operator (e.g. "=", "!=", "ilike")
	Value string `toml:"value"` // filter value
}

// ModelConfig holds per-model configuration.
type ModelConfig struct {
	ExtraFields []ExtraField `toml:"extra_fields"`
	Filters     []Filter     `toml:"filters"`
}

// DefaultBundesland is the default German federal state for holidays.
const DefaultBundesland = "Baden-Württemberg"

// HoursLimits holds threshold values for hour coloring in the TUI.
type HoursLimits struct {
	DailyLow   float64 `toml:"daily_low"`   // below this: yellow (default 6)
	DailyHigh  float64 `toml:"daily_high"`  // above this: red (default 9)
	WeeklyLow  float64 `toml:"weekly_low"`  // below this: yellow (default 35)
	WeeklyHigh float64 `toml:"weekly_high"` // above this: red (default 40)
}

// DefaultHoursLimits returns the default work hour thresholds.
func DefaultHoursLimits() HoursLimits {
	return HoursLimits{
		DailyLow:   6,
		DailyHigh:  9,
		WeeklyLow:  35,
		WeeklyHigh: 40,
	}
}

// Config holds the application configuration.
type Config struct {
	URL        string                 `toml:"url"`
	Database   string                 `toml:"database"`
	Username   string                 `toml:"username"`
	Password   string                 `toml:"-"`
	Models     map[string]ModelConfig `toml:"models"`
	Hours      HoursLimits            `toml:"hours"`
	Bundesland string                 `toml:"bundesland"` // German federal state for holidays (e.g. "Bayern")
}

func (c *Config) OdooURL() string      { return c.URL }
func (c *Config) OdooDatabase() string { return c.Database }
func (c *Config) OdooUsername() string { return c.Username }
func (c *Config) OdooPassword() string { return c.Password }

// LoadFromEnv reads configuration from environment variables.
// It reads whatever env vars are set without requiring any.
// Use Validate to check that all required fields are present.
func LoadFromEnv() *Config {
	return &Config{
		URL:        os.Getenv("ODOO_URL"),
		Database:   os.Getenv("ODOO_DATABASE"),
		Username:   os.Getenv("ODOO_USERNAME"),
		Password:   os.Getenv("ODOO_PASSWORD"),
		Hours:      DefaultHoursLimits(),
		Bundesland: DefaultBundesland,
	}
}

// ApplyDefaults fills in zero-valued fields with defaults.
func (c *Config) ApplyDefaults() {
	d := DefaultHoursLimits()
	if c.Hours.DailyLow == 0 {
		c.Hours.DailyLow = d.DailyLow
	}
	if c.Hours.DailyHigh == 0 {
		c.Hours.DailyHigh = d.DailyHigh
	}
	if c.Hours.WeeklyLow == 0 {
		c.Hours.WeeklyLow = d.WeeklyLow
	}
	if c.Hours.WeeklyHigh == 0 {
		c.Hours.WeeklyHigh = d.WeeklyHigh
	}
	if c.Bundesland == "" {
		c.Bundesland = DefaultBundesland
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
	cfg.ApplyDefaults()
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
	if other.Bundesland != "" {
		c.Bundesland = other.Bundesland
	}
	if other.Hours.DailyLow != 0 {
		c.Hours.DailyLow = other.Hours.DailyLow
	}
	if other.Hours.DailyHigh != 0 {
		c.Hours.DailyHigh = other.Hours.DailyHigh
	}
	if other.Hours.WeeklyLow != 0 {
		c.Hours.WeeklyLow = other.Hours.WeeklyLow
	}
	if other.Hours.WeeklyHigh != 0 {
		c.Hours.WeeklyHigh = other.Hours.WeeklyHigh
	}
	if other.Models != nil {
		if c.Models == nil {
			c.Models = make(map[string]ModelConfig)
		}
		for k, overlay := range other.Models {
			base, exists := c.Models[k]
			if !exists {
				c.Models[k] = overlay
				continue
			}
			if len(overlay.ExtraFields) > 0 {
				base.ExtraFields = overlay.ExtraFields
			}
			base.Filters = mergeFilters(base.Filters, overlay.Filters)
			c.Models[k] = base
		}
	}
}

// mergeFilters accumulates filters from base and overlay.
// If overlay has a filter with the same Field as a base filter, the overlay
// entry replaces the base entry.
func mergeFilters(base, overlay []Filter) []Filter {
	if len(overlay) == 0 {
		return base
	}
	if len(base) == 0 {
		return overlay
	}
	// Build result starting from base, replacing same-field entries.
	overrideFields := make(map[string]Filter, len(overlay))
	for _, f := range overlay {
		overrideFields[f.Field] = f
	}
	var result []Filter
	seen := make(map[string]bool)
	for _, f := range base {
		if ov, ok := overrideFields[f.Field]; ok {
			result = append(result, ov)
			seen[f.Field] = true
		} else {
			result = append(result, f)
		}
	}
	for _, f := range overlay {
		if !seen[f.Field] {
			result = append(result, f)
		}
	}
	return result
}
