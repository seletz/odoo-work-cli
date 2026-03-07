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

func TestLoadFromTOML_ExtraFields(t *testing.T) {
	content := `
url = "https://odoo.example.com"
database = "odoo.170"

[models.project]
extra_fields = [
  { name = "product_owner", field = "x_studio_productowner", type = "many2one" },
  { name = "status", field = "x_studio_status", type = "char" },
]

[models.task]
extra_fields = []
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromTOML(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.URL != "https://odoo.example.com" {
		t.Errorf("URL = %q, want %q", cfg.URL, "https://odoo.example.com")
	}

	// Check models map exists.
	if cfg.Models == nil {
		t.Fatal("Models is nil")
	}
	if len(cfg.Models) != 2 {
		t.Fatalf("len(Models) = %d, want 2", len(cfg.Models))
	}

	// Check project model extra fields.
	proj, ok := cfg.Models["project"]
	if !ok {
		t.Fatal("Models[\"project\"] not found")
	}
	if len(proj.ExtraFields) != 2 {
		t.Fatalf("len(project.ExtraFields) = %d, want 2", len(proj.ExtraFields))
	}
	ef := proj.ExtraFields[0]
	if ef.Name != "product_owner" {
		t.Errorf("ExtraFields[0].Name = %q, want %q", ef.Name, "product_owner")
	}
	if ef.Field != "x_studio_productowner" {
		t.Errorf("ExtraFields[0].Field = %q, want %q", ef.Field, "x_studio_productowner")
	}
	if ef.Type != "many2one" {
		t.Errorf("ExtraFields[0].Type = %q, want %q", ef.Type, "many2one")
	}

	// Check task model has empty extra fields.
	task, ok := cfg.Models["task"]
	if !ok {
		t.Fatal("Models[\"task\"] not found")
	}
	if len(task.ExtraFields) != 0 {
		t.Errorf("len(task.ExtraFields) = %d, want 0", len(task.ExtraFields))
	}
}

func TestMerge_Models(t *testing.T) {
	base := &Config{
		URL: "https://base.example.com",
		Models: map[string]ModelConfig{
			"project": {
				ExtraFields: []ExtraField{
					{Name: "owner", Field: "x_owner", Type: "many2one"},
				},
			},
		},
	}
	overlay := &Config{
		Models: map[string]ModelConfig{
			"task": {
				ExtraFields: []ExtraField{
					{Name: "priority", Field: "x_priority", Type: "char"},
				},
			},
		},
	}

	base.Merge(overlay)

	if len(base.Models) != 2 {
		t.Fatalf("len(Models) = %d, want 2", len(base.Models))
	}
	if _, ok := base.Models["project"]; !ok {
		t.Error("Models[\"project\"] missing after merge")
	}
	if _, ok := base.Models["task"]; !ok {
		t.Error("Models[\"task\"] missing after merge")
	}
}

func TestMerge_ModelsOverlayOverrides(t *testing.T) {
	base := &Config{
		Models: map[string]ModelConfig{
			"project": {
				ExtraFields: []ExtraField{
					{Name: "owner", Field: "x_owner", Type: "many2one"},
				},
			},
		},
	}
	overlay := &Config{
		Models: map[string]ModelConfig{
			"project": {
				ExtraFields: []ExtraField{
					{Name: "new_field", Field: "x_new", Type: "char"},
				},
			},
		},
	}

	base.Merge(overlay)

	proj := base.Models["project"]
	if len(proj.ExtraFields) != 1 {
		t.Fatalf("len(ExtraFields) = %d, want 1", len(proj.ExtraFields))
	}
	if proj.ExtraFields[0].Name != "new_field" {
		t.Errorf("ExtraFields[0].Name = %q, want %q", proj.ExtraFields[0].Name, "new_field")
	}
}
