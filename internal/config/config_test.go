package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromEnv(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want map[string]string
	}{
		{
			name: "all set",
			env: map[string]string{
				"ODOO_URL":      "https://odoo.example.com",
				"ODOO_DATABASE": "mydb",
				"ODOO_USERNAME": "admin",
				"ODOO_PASSWORD": "secret",
			},
			want: map[string]string{
				"url": "https://odoo.example.com", "database": "mydb",
				"username": "admin", "password": "secret",
			},
		},
		{
			name: "none set",
			env:  map[string]string{},
			want: map[string]string{
				"url": "", "database": "", "username": "", "password": "",
			},
		},
		{
			name: "partial",
			env: map[string]string{
				"ODOO_URL":      "https://odoo.example.com",
				"ODOO_USERNAME": "admin",
			},
			want: map[string]string{
				"url": "https://odoo.example.com", "database": "",
				"username": "admin", "password": "",
			},
		},
	}

	envKeys := []string{"ODOO_URL", "ODOO_DATABASE", "ODOO_USERNAME", "ODOO_PASSWORD"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, k := range envKeys {
				t.Setenv(k, "")
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			cfg := LoadFromEnv()

			if cfg.OdooURL() != tt.want["url"] {
				t.Errorf("URL = %q, want %q", cfg.OdooURL(), tt.want["url"])
			}
			if cfg.OdooDatabase() != tt.want["database"] {
				t.Errorf("Database = %q, want %q", cfg.OdooDatabase(), tt.want["database"])
			}
			if cfg.OdooUsername() != tt.want["username"] {
				t.Errorf("Username = %q, want %q", cfg.OdooUsername(), tt.want["username"])
			}
			if cfg.OdooPassword() != tt.want["password"] {
				t.Errorf("Password = %q, want %q", cfg.OdooPassword(), tt.want["password"])
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "all set",
			cfg: &Config{
				URL: "https://odoo.example.com", Database: "mydb",
				Username: "admin", Password: "secret",
			},
			wantErr: false,
		},
		{
			name:    "all missing",
			cfg:     &Config{},
			wantErr: true,
			errMsg:  "URL",
		},
		{
			name: "missing password",
			cfg: &Config{
				URL: "https://odoo.example.com", Database: "mydb",
				Username: "admin",
			},
			wantErr: true,
			errMsg:  "Password",
		},
		{
			name: "missing url and database",
			cfg: &Config{
				Username: "admin", Password: "secret",
			},
			wantErr: true,
			errMsg:  "URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
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

func TestLoadFromTOML_RejectsPassword(t *testing.T) {
	content := "url = \"https://odoo.example.com\"\npassword = \"should-be-rejected\"\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFromTOML(path)
	if err == nil {
		t.Fatal("expected error when config file contains password, got nil")
	}
	if !strings.Contains(err.Error(), "password") {
		t.Errorf("error %q should mention password", err.Error())
	}
}

func TestLoadFromTOML_NoPasswordIsOK(t *testing.T) {
	content := "url = \"https://odoo.example.com\"\ndatabase = \"mydb\"\n"
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

func TestLoadFromTOML_Filters(t *testing.T) {
	content := `
url = "https://odoo.example.com"

[models.task]
filters = [
  { field = "company_id.name", op = "=", value = "Company A" },
  { field = "stage_id.name", op = "!=", value = "Cancelled" },
]

[models.project]
extra_fields = [
  { name = "owner", field = "x_owner", type = "many2one" },
]
filters = [
  { field = "active", op = "=", value = "true" },
]
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

	// Check task filters.
	task, ok := cfg.Models["task"]
	if !ok {
		t.Fatal("Models[\"task\"] not found")
	}
	if len(task.Filters) != 2 {
		t.Fatalf("len(task.Filters) = %d, want 2", len(task.Filters))
	}
	if task.Filters[0].Field != "company_id.name" {
		t.Errorf("Filters[0].Field = %q, want %q", task.Filters[0].Field, "company_id.name")
	}
	if task.Filters[0].Op != "=" {
		t.Errorf("Filters[0].Op = %q, want %q", task.Filters[0].Op, "=")
	}
	if task.Filters[0].Value != "Company A" {
		t.Errorf("Filters[0].Value = %q, want %q", task.Filters[0].Value, "Company A")
	}

	// Check project has both extra_fields and filters.
	proj, ok := cfg.Models["project"]
	if !ok {
		t.Fatal("Models[\"project\"] not found")
	}
	if len(proj.ExtraFields) != 1 {
		t.Fatalf("len(project.ExtraFields) = %d, want 1", len(proj.ExtraFields))
	}
	if len(proj.Filters) != 1 {
		t.Fatalf("len(project.Filters) = %d, want 1", len(proj.Filters))
	}
	if proj.Filters[0].Field != "active" {
		t.Errorf("Filters[0].Field = %q, want %q", proj.Filters[0].Field, "active")
	}
}

func TestMerge_ModelsOverlayOverrides(t *testing.T) {
	base := &Config{
		Models: map[string]ModelConfig{
			"project": {
				ExtraFields: []ExtraField{
					{Name: "owner", Field: "x_owner", Type: "many2one"},
				},
				Filters: []Filter{
					{Field: "active", Op: "=", Value: "true"},
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
	// Overlay ExtraFields replaces base.
	if len(proj.ExtraFields) != 1 {
		t.Fatalf("len(ExtraFields) = %d, want 1", len(proj.ExtraFields))
	}
	if proj.ExtraFields[0].Name != "new_field" {
		t.Errorf("ExtraFields[0].Name = %q, want %q", proj.ExtraFields[0].Name, "new_field")
	}
	// Base Filters survive when overlay has none.
	if len(proj.Filters) != 1 {
		t.Fatalf("len(Filters) = %d, want 1 (base filters should survive)", len(proj.Filters))
	}
	if proj.Filters[0].Field != "active" {
		t.Errorf("Filters[0].Field = %q, want %q", proj.Filters[0].Field, "active")
	}
}

func TestMerge_FiltersAccumulate(t *testing.T) {
	base := &Config{
		Models: map[string]ModelConfig{
			"task": {
				Filters: []Filter{
					{Field: "company_id.name", Op: "=", Value: "Company A"},
				},
			},
		},
	}
	overlay := &Config{
		Models: map[string]ModelConfig{
			"task": {
				Filters: []Filter{
					{Field: "project_id.name", Op: "=", Value: "Project X"},
				},
			},
		},
	}

	base.Merge(overlay)

	task := base.Models["task"]
	if len(task.Filters) != 2 {
		t.Fatalf("len(Filters) = %d, want 2 (filters should accumulate)", len(task.Filters))
	}
}

func TestLoadFromTOML_HoursLimits(t *testing.T) {
	content := `
url = "https://odoo.example.com"

[hours]
daily_low = 7.0
daily_high = 10.0
weekly_low = 30.0
weekly_high = 45.0
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
	if cfg.Hours.DailyLow != 7.0 {
		t.Errorf("DailyLow = %f, want 7.0", cfg.Hours.DailyLow)
	}
	if cfg.Hours.DailyHigh != 10.0 {
		t.Errorf("DailyHigh = %f, want 10.0", cfg.Hours.DailyHigh)
	}
	if cfg.Hours.WeeklyLow != 30.0 {
		t.Errorf("WeeklyLow = %f, want 30.0", cfg.Hours.WeeklyLow)
	}
	if cfg.Hours.WeeklyHigh != 45.0 {
		t.Errorf("WeeklyHigh = %f, want 45.0", cfg.Hours.WeeklyHigh)
	}
}

func TestLoadFromTOML_HoursLimitsDefaults(t *testing.T) {
	content := `url = "https://odoo.example.com"`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromTOML(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d := DefaultHoursLimits()
	if cfg.Hours.DailyLow != d.DailyLow {
		t.Errorf("DailyLow = %f, want default %f", cfg.Hours.DailyLow, d.DailyLow)
	}
	if cfg.Hours.WeeklyHigh != d.WeeklyHigh {
		t.Errorf("WeeklyHigh = %f, want default %f", cfg.Hours.WeeklyHigh, d.WeeklyHigh)
	}
}

func TestMerge_HoursLimits(t *testing.T) {
	base := &Config{Hours: DefaultHoursLimits()}
	overlay := &Config{Hours: HoursLimits{DailyHigh: 10.0, WeeklyLow: 30.0}}

	base.Merge(overlay)

	if base.Hours.DailyHigh != 10.0 {
		t.Errorf("DailyHigh = %f, want 10.0", base.Hours.DailyHigh)
	}
	if base.Hours.WeeklyLow != 30.0 {
		t.Errorf("WeeklyLow = %f, want 30.0", base.Hours.WeeklyLow)
	}
	// Unset overlay fields keep base defaults.
	if base.Hours.DailyLow != 6.0 {
		t.Errorf("DailyLow = %f, want 6.0 (base default)", base.Hours.DailyLow)
	}
	if base.Hours.WeeklyHigh != 40.0 {
		t.Errorf("WeeklyHigh = %f, want 40.0 (base default)", base.Hours.WeeklyHigh)
	}
}

func TestMerge_FiltersSameFieldOverride(t *testing.T) {
	base := &Config{
		Models: map[string]ModelConfig{
			"task": {
				Filters: []Filter{
					{Field: "company_id.name", Op: "=", Value: "Company A"},
					{Field: "active", Op: "=", Value: "true"},
				},
			},
		},
	}
	overlay := &Config{
		Models: map[string]ModelConfig{
			"task": {
				Filters: []Filter{
					{Field: "company_id.name", Op: "=", Value: "Company B"},
				},
			},
		},
	}

	base.Merge(overlay)

	task := base.Models["task"]
	if len(task.Filters) != 2 {
		t.Fatalf("len(Filters) = %d, want 2", len(task.Filters))
	}
	// Find the company filter — should be overridden.
	var companyFilter *Filter
	for i := range task.Filters {
		if task.Filters[i].Field == "company_id.name" {
			companyFilter = &task.Filters[i]
		}
	}
	if companyFilter == nil {
		t.Fatal("company_id.name filter not found")
	}
	if companyFilter.Value != "Company B" {
		t.Errorf("company filter Value = %q, want %q", companyFilter.Value, "Company B")
	}
}
