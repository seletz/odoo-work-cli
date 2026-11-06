package config

import (
	"errors"
	"fmt"
	"github.com/seletz/odoo-work-cli/internal/apperr"
	"os"
	"strings"
	"testing"
	"time"
)

// fakeOPInjectRunner returns a runner that resolves op:// references in templates.
func fakeOPInjectRunner(secrets map[string]string) opInjectRunner {
	return func(template string) (string, error) {
		result := template
		for ref, val := range secrets {
			result = strings.ReplaceAll(result, "{{ "+ref+" }}", val)
		}
		// Check for unresolved references.
		if strings.Contains(result, "{{ op://") {
			return "", fmt.Errorf("op inject: unresolved references in template")
		}
		return result, nil
	}
}

func TestResolveOPSecrets_AllFields(t *testing.T) {
	runner := fakeOPInjectRunner(map[string]string{
		"op://Work/odoo/url":      "https://odoo.example.com",
		"op://Work/odoo/database": "mydb",
		"op://Work/odoo/username": "admin@example.com",
		"op://Work/odoo/api-key":  "secret-api-key",
		"op://Work/odoo/password": "secret-password",
	})

	cfg := &Config{
		OPSecrets: &OPSecrets{
			URL:      "op://Work/odoo/url",
			Database: "op://Work/odoo/database",
			Username: "op://Work/odoo/username",
			APIKey:   "op://Work/odoo/api-key",
			Password: "op://Work/odoo/password",
		},
	}

	err := resolveOPSecrets(cfg, runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.URL != "https://odoo.example.com" {
		t.Errorf("URL = %q, want %q", cfg.URL, "https://odoo.example.com")
	}
	if cfg.Database != "mydb" {
		t.Errorf("Database = %q, want %q", cfg.Database, "mydb")
	}
	if cfg.Username != "admin@example.com" {
		t.Errorf("Username = %q, want %q", cfg.Username, "admin@example.com")
	}
	if cfg.Password != "secret-api-key" {
		t.Errorf("Password (API key) = %q, want %q", cfg.Password, "secret-api-key")
	}
	if cfg.WebPassword != "secret-password" {
		t.Errorf("WebPassword = %q, want %q", cfg.WebPassword, "secret-password")
	}
}

func TestResolveOPSecrets_APIKeyOnly(t *testing.T) {
	runner := fakeOPInjectRunner(map[string]string{
		"op://Work/odoo/api-key": "secret-api-key",
	})

	cfg := &Config{
		URL:      "https://already-set.example.com",
		Database: "existing-db",
		Username: "existing-user",
		OPSecrets: &OPSecrets{
			APIKey: "op://Work/odoo/api-key",
		},
	}

	err := resolveOPSecrets(cfg, runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.URL != "https://already-set.example.com" {
		t.Errorf("URL = %q, want existing value", cfg.URL)
	}
	if cfg.Password != "secret-api-key" {
		t.Errorf("Password (API key) = %q, want %q", cfg.Password, "secret-api-key")
	}
	if cfg.WebPassword != "" {
		t.Errorf("WebPassword = %q, want empty", cfg.WebPassword)
	}
}

func TestResolveOPSecrets_NilOPSecrets(t *testing.T) {
	cfg := &Config{
		URL: "https://odoo.example.com",
	}

	runner := fakeOPInjectRunner(nil)
	err := resolveOPSecrets(cfg, runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.URL != "https://odoo.example.com" {
		t.Errorf("URL = %q, want original value", cfg.URL)
	}
}

func TestResolveOPSecrets_RunnerError(t *testing.T) {
	runner := func(template string) (string, error) {
		return "", fmt.Errorf("op inject: not signed in")
	}

	cfg := &Config{
		OPSecrets: &OPSecrets{
			APIKey: "op://Work/odoo/api-key",
		},
	}

	err := resolveOPSecrets(cfg, runner)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolveOPSecrets_OverwritesExistingFields(t *testing.T) {
	runner := fakeOPInjectRunner(map[string]string{
		"op://Work/odoo/url": "https://new-from-op.example.com",
	})

	cfg := &Config{
		URL: "https://old-from-toml.example.com",
		OPSecrets: &OPSecrets{
			URL: "op://Work/odoo/url",
		},
	}

	err := resolveOPSecrets(cfg, runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.URL != "https://new-from-op.example.com" {
		t.Errorf("URL = %q, want op-resolved value", cfg.URL)
	}
}

func TestResolveOPSecrets_PlainValues(t *testing.T) {
	runner := fakeOPInjectRunner(map[string]string{
		"op://Work/odoo/api-key": "secret-api-key",
	})

	cfg := &Config{
		OPSecrets: &OPSecrets{
			Database: "odoo.170",
			APIKey:   "op://Work/odoo/api-key",
		},
	}

	err := resolveOPSecrets(cfg, runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Database != "odoo.170" {
		t.Errorf("Database = %q, want %q", cfg.Database, "odoo.170")
	}
	if cfg.Password != "secret-api-key" {
		t.Errorf("Password (API key) = %q, want %q", cfg.Password, "secret-api-key")
	}
}

func TestResolveOPSecrets_AllPlainValues(t *testing.T) {
	// Runner should never be called when all values are plain.
	runner := func(template string) (string, error) {
		t.Fatal("op inject should not be called for plain values only")
		return "", nil
	}

	cfg := &Config{
		OPSecrets: &OPSecrets{
			URL:      "https://odoo.example.com",
			Database: "mydb",
		},
	}

	err := resolveOPSecrets(cfg, runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.URL != "https://odoo.example.com" {
		t.Errorf("URL = %q, want %q", cfg.URL, "https://odoo.example.com")
	}
	if cfg.Database != "mydb" {
		t.Errorf("Database = %q, want %q", cfg.Database, "mydb")
	}
}

func TestResolveOPSecrets_PasswordMapsToWebPassword(t *testing.T) {
	runner := fakeOPInjectRunner(map[string]string{
		"op://Work/odoo/password": "web-secret",
	})

	cfg := &Config{
		OPSecrets: &OPSecrets{
			Password: "op://Work/odoo/password",
		},
	}

	err := resolveOPSecrets(cfg, runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.WebPassword != "web-secret" {
		t.Errorf("WebPassword = %q, want %q", cfg.WebPassword, "web-secret")
	}
	if cfg.Password != "" {
		t.Errorf("Password (API key) = %q, want empty", cfg.Password)
	}
}

func TestLoadFromTOML_OPSecrets(t *testing.T) {
	content := `
url = "https://odoo.example.com"
database = "mydb"
username = "admin"

[op_secrets]
api-key  = "op://Work/odoo/api-key"
password = "op://Work/odoo/password"
`
	dir := t.TempDir()
	path := dir + "/config.toml"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromTOML(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.OPSecrets == nil {
		t.Fatal("OPSecrets is nil")
	}
	if cfg.OPSecrets.APIKey != "op://Work/odoo/api-key" {
		t.Errorf("OPSecrets.APIKey = %q, want %q", cfg.OPSecrets.APIKey, "op://Work/odoo/api-key")
	}
	if cfg.OPSecrets.Password != "op://Work/odoo/password" {
		t.Errorf("OPSecrets.Password = %q, want %q", cfg.OPSecrets.Password, "op://Work/odoo/password")
	}
}

func TestLoadFromTOML_NoOPSecrets(t *testing.T) {
	content := `url = "https://odoo.example.com"`
	dir := t.TempDir()
	path := dir + "/config.toml"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromTOML(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.OPSecrets != nil {
		t.Errorf("OPSecrets = %v, want nil when section absent", cfg.OPSecrets)
	}
}

func TestMerge_OPSecrets(t *testing.T) {
	base := &Config{}
	overlay := &Config{
		OPSecrets: &OPSecrets{
			APIKey:   "op://Work/odoo/api-key",
			Password: "op://Work/odoo/password",
		},
	}

	base.Merge(overlay)

	if base.OPSecrets == nil {
		t.Fatal("OPSecrets is nil after merge")
	}
	if base.OPSecrets.APIKey != "op://Work/odoo/api-key" {
		t.Errorf("OPSecrets.APIKey = %q, want %q", base.OPSecrets.APIKey, "op://Work/odoo/api-key")
	}
	if base.OPSecrets.Password != "op://Work/odoo/password" {
		t.Errorf("OPSecrets.Password = %q, want %q", base.OPSecrets.Password, "op://Work/odoo/password")
	}
}

func TestMerge_OPSecrets_OverlayReplacesBase(t *testing.T) {
	base := &Config{
		OPSecrets: &OPSecrets{
			APIKey: "op://Old/odoo/key",
			URL:    "op://Old/odoo/url",
		},
	}
	overlay := &Config{
		OPSecrets: &OPSecrets{
			APIKey: "op://New/odoo/key",
		},
	}

	base.Merge(overlay)

	if base.OPSecrets.APIKey != "op://New/odoo/key" {
		t.Errorf("APIKey = %q, want overlay value", base.OPSecrets.APIKey)
	}
	if base.OPSecrets.URL != "op://Old/odoo/url" {
		t.Errorf("URL = %q, want base value preserved", base.OPSecrets.URL)
	}
}

func TestMerge_OPSecrets_NilOverlay(t *testing.T) {
	base := &Config{
		OPSecrets: &OPSecrets{
			APIKey: "op://Work/odoo/key",
		},
	}
	overlay := &Config{}

	base.Merge(overlay)

	if base.OPSecrets == nil || base.OPSecrets.APIKey != "op://Work/odoo/key" {
		t.Error("base OPSecrets should be preserved when overlay has none")
	}
}

func TestParseKeyValues(t *testing.T) {
	input := "url=https://odoo.example.com\ndatabase=mydb\napi-key=secret\n"
	got := parseKeyValues(input)

	if got["url"] != "https://odoo.example.com" {
		t.Errorf("url = %q", got["url"])
	}
	if got["database"] != "mydb" {
		t.Errorf("database = %q", got["database"])
	}
	if got["api-key"] != "secret" {
		t.Errorf("api-key = %q", got["api-key"])
	}
}

func TestParseKeyValues_EmptyInput(t *testing.T) {
	got := parseKeyValues("")
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestResolveOPSecrets_ErrorKinds(t *testing.T) {
	tests := []struct {
		name     string
		runErr   error
		wantKind apperr.Kind
		wantAny  bool // whether a kind is expected from apperr at all
	}{
		{"locked", fmt.Errorf("op inject: authorization timeout: %w", errOPLocked), apperr.KindSecretsLocked, true},
		{"other", fmt.Errorf("op inject: invalid reference"), "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{OPSecrets: &OPSecrets{URL: "op://Work/odoo/url"}}
			err := resolveOPSecrets(cfg, func(string) (string, error) { return "", tt.runErr })
			if err == nil {
				t.Fatal("expected error")
			}
			var ae *apperr.Error
			got := errors.As(err, &ae)
			if got != tt.wantAny {
				t.Fatalf("typed apperr = %v, want %v (err: %v)", got, tt.wantAny, err)
			}
			if tt.wantAny {
				if ae.Kind != tt.wantKind {
					t.Errorf("kind = %q, want %q", ae.Kind, tt.wantKind)
				}
				if !strings.Contains(err.Error(), "1Password is locked or not reachable: unlock it and retry") {
					t.Errorf("message = %q", err.Error())
				}
				if !errors.Is(err, errOPLocked) {
					t.Error("original error not wrapped")
				}
			}
		})
	}
}

func TestRunOPCommand(t *testing.T) {
	tests := []struct {
		name       string
		bin        string
		args       []string
		timeout    time.Duration
		wantLocked bool
		wantErr    bool
		wantOut    string
	}{
		{"success", "cat", nil, 5 * time.Second, false, false, "in"},
		{"timeout", "sleep", []string{"5"}, 100 * time.Millisecond, true, true, ""},
		{"locked stderr", "sh", []string{"-c", "echo 'authorization timeout' >&2; exit 1"}, 5 * time.Second, true, true, ""},
		{"not signed in", "sh", []string{"-c", "echo '[ERROR] You are not signed in' >&2; exit 1"}, 5 * time.Second, true, true, ""},
		{"other failure", "sh", []string{"-c", "echo 'invalid secret reference' >&2; exit 1"}, 5 * time.Second, false, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			out, err := runOPCommand(tt.timeout, tt.bin, tt.args, "in")
			if time.Since(start) > 3*time.Second {
				t.Fatal("did not fail fast")
			}
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if errors.Is(err, errOPLocked) != tt.wantLocked {
				t.Errorf("locked = %v, want %v (err: %v)", errors.Is(err, errOPLocked), tt.wantLocked, err)
			}
			if out != tt.wantOut {
				t.Errorf("out = %q, want %q", out, tt.wantOut)
			}
		})
	}
}

func TestOPTimeout(t *testing.T) {
	tests := []struct {
		env  string
		want time.Duration
	}{
		{"", defaultOPTimeout},
		{"2s", 2 * time.Second},
		{"garbage", defaultOPTimeout},
		{"-1s", defaultOPTimeout},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv("ODOO_OP_TIMEOUT", tt.env)
			if got := opTimeout(); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
