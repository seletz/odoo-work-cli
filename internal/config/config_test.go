package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{
			name: "all set",
			env: map[string]string{
				"ODOO_URL":      "https://odoo.example.com",
				"ODOO_DATABASE": "mydb",
				"ODOO_USERNAME": "admin",
				"ODOO_PASSWORD": "secret",
			},
			wantErr: false,
		},
		{
			name:    "none set",
			env:     map[string]string{},
			wantErr: true,
		},
		{
			name: "missing password",
			env: map[string]string{
				"ODOO_URL":      "https://odoo.example.com",
				"ODOO_DATABASE": "mydb",
				"ODOO_USERNAME": "admin",
			},
			wantErr: true,
		},
		{
			name: "missing url and database",
			env: map[string]string{
				"ODOO_USERNAME": "admin",
				"ODOO_PASSWORD": "secret",
			},
			wantErr: true,
		},
	}

	envKeys := []string{"ODOO_URL", "ODOO_DATABASE", "ODOO_USERNAME", "ODOO_PASSWORD"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all relevant env vars.
			for _, k := range envKeys {
				t.Setenv(k, "")
			}
			// Set test-specific env vars.
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			cfg, err := LoadFromEnv()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.OdooURL() != tt.env["ODOO_URL"] {
				t.Errorf("URL = %q, want %q", cfg.OdooURL(), tt.env["ODOO_URL"])
			}
			if cfg.OdooDatabase() != tt.env["ODOO_DATABASE"] {
				t.Errorf("Database = %q, want %q", cfg.OdooDatabase(), tt.env["ODOO_DATABASE"])
			}
			if cfg.OdooUsername() != tt.env["ODOO_USERNAME"] {
				t.Errorf("Username = %q, want %q", cfg.OdooUsername(), tt.env["ODOO_USERNAME"])
			}
			if cfg.OdooPassword() != tt.env["ODOO_PASSWORD"] {
				t.Errorf("Password = %q, want %q", cfg.OdooPassword(), tt.env["ODOO_PASSWORD"])
			}
		})
	}
}

func TestLoadFromTOML(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantErr  bool
		wantURL  string
		wantDB   string
	}{
		{
			name:    "valid config",
			content: "url = \"https://odoo.example.com\"\ndatabase = \"mydb\"\n",
			wantURL: "https://odoo.example.com",
			wantDB:  "mydb",
		},
		{
			name:    "empty file",
			content: "",
			wantURL: "",
			wantDB:  "",
		},
		{
			name:    "invalid toml",
			content: "url = [broken",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.toml")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadFromTOML(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.URL != tt.wantURL {
				t.Errorf("URL = %q, want %q", cfg.URL, tt.wantURL)
			}
			if cfg.Database != tt.wantDB {
				t.Errorf("Database = %q, want %q", cfg.Database, tt.wantDB)
			}
		})
	}
}

func TestLoadFromTOML_FileNotFound(t *testing.T) {
	_, err := LoadFromTOML("/nonexistent/path/config.toml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestMerge(t *testing.T) {
	base := &Config{
		URL:      "https://base.example.com",
		Database: "basedb",
	}
	overlay := &Config{
		URL:      "https://overlay.example.com",
		Username: "admin",
		Password: "secret",
	}

	base.Merge(overlay)

	if base.URL != "https://overlay.example.com" {
		t.Errorf("URL = %q, want overlay value", base.URL)
	}
	if base.Database != "basedb" {
		t.Errorf("Database = %q, want base value", base.Database)
	}
	if base.Username != "admin" {
		t.Errorf("Username = %q, want overlay value", base.Username)
	}
}
