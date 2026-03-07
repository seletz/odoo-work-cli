package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigPath(t *testing.T) {
	t.Run("uses XDG_CONFIG_HOME when set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/tmp/fakexdg")
		got, err := DefaultConfigPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "/tmp/fakexdg/odoo-work-cli/config.toml"
		if got != want {
			t.Errorf("DefaultConfigPath() = %q, want %q", got, want)
		}
	})

	t.Run("falls back to ~/.config", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		got, err := DefaultConfigPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasSuffix(got, ".config/odoo-work-cli/config.toml") {
			t.Errorf("DefaultConfigPath() = %q, want suffix .config/odoo-work-cli/config.toml", got)
		}
	})
}

func TestWalkConfigFiles(t *testing.T) {
	t.Run("no files found", func(t *testing.T) {
		dir := t.TempDir()
		files, err := WalkConfigFiles(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 0 {
			t.Errorf("expected no files, got %v", files)
		}
	})

	t.Run("single file in startDir", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, ".odoo-work-cli.toml")
		if err := os.WriteFile(cfgPath, []byte("url = \"test\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		files, err := WalkConfigFiles(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}
		if files[0] != cfgPath {
			t.Errorf("got %q, want %q", files[0], cfgPath)
		}
	})

	t.Run("multiple levels root-most first", func(t *testing.T) {
		root := t.TempDir()
		child := filepath.Join(root, "a", "b")
		if err := os.MkdirAll(child, 0o755); err != nil {
			t.Fatal(err)
		}
		rootCfg := filepath.Join(root, ".odoo-work-cli.toml")
		childCfg := filepath.Join(child, ".odoo-work-cli.toml")
		for _, p := range []string{rootCfg, childCfg} {
			if err := os.WriteFile(p, []byte(""), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		files, err := WalkConfigFiles(child)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 2 {
			t.Fatalf("expected 2 files, got %d: %v", len(files), files)
		}
		// Root-most first.
		if files[0] != rootCfg {
			t.Errorf("files[0] = %q, want root %q", files[0], rootCfg)
		}
		if files[1] != childCfg {
			t.Errorf("files[1] = %q, want child %q", files[1], childCfg)
		}
	})
}

func TestDiscover(t *testing.T) {
	envKeys := []string{"ODOO_URL", "ODOO_DATABASE", "ODOO_USERNAME", "ODOO_PASSWORD"}

	t.Run("configFlag loads only that file", func(t *testing.T) {
		for _, k := range envKeys {
			t.Setenv(k, "")
		}
		t.Setenv("ODOO_USERNAME", "admin")
		t.Setenv("ODOO_PASSWORD", "secret")

		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "my.toml")
		if err := os.WriteFile(cfgPath, []byte("url = \"https://flag.example.com\"\ndatabase = \"flagdb\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		result, err := Discover(cfgPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Files) != 1 {
			t.Fatalf("expected 1 file, got %d: %v", len(result.Files), result.Files)
		}
		if result.Files[0] != cfgPath {
			t.Errorf("Files[0] = %q, want %q", result.Files[0], cfgPath)
		}
		if result.Config.URL != "https://flag.example.com" {
			t.Errorf("URL = %q, want flag value", result.Config.URL)
		}
	})

	t.Run("configFlag nonexistent errors", func(t *testing.T) {
		for _, k := range envKeys {
			t.Setenv(k, "")
		}
		_, err := Discover("/nonexistent/path.toml")
		if err == nil {
			t.Fatal("expected error for nonexistent config file")
		}
	})

	t.Run("env vars overlay walk files", func(t *testing.T) {
		for _, k := range envKeys {
			t.Setenv(k, "")
		}
		t.Setenv("ODOO_URL", "https://env.example.com")
		t.Setenv("ODOO_DATABASE", "envdb")
		t.Setenv("ODOO_USERNAME", "envuser")
		t.Setenv("ODOO_PASSWORD", "envpass")
		// Use XDG to point to a nonexistent global so it doesn't interfere.
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		// Create a walk file in cwd.
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, ".odoo-work-cli.toml")
		if err := os.WriteFile(cfgPath, []byte("url = \"https://walk.example.com\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		origDir, _ := os.Getwd()
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(origDir) })

		result, err := Discover("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Env should override walk file URL.
		if result.Config.URL != "https://env.example.com" {
			t.Errorf("URL = %q, want env value", result.Config.URL)
		}
		if result.Config.Username != "envuser" {
			t.Errorf("Username = %q, want envuser", result.Config.Username)
		}
	})

	t.Run("walk files override global", func(t *testing.T) {
		for _, k := range envKeys {
			t.Setenv(k, "")
		}
		t.Setenv("ODOO_USERNAME", "user")
		t.Setenv("ODOO_PASSWORD", "pass")

		// Set up global config.
		globalDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", globalDir)
		globalCfgDir := filepath.Join(globalDir, "odoo-work-cli")
		if err := os.MkdirAll(globalCfgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(globalCfgDir, "config.toml"),
			[]byte("url = \"https://global.example.com\"\ndatabase = \"globaldb\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Set up walk file that overrides URL.
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ".odoo-work-cli.toml"),
			[]byte("url = \"https://local.example.com\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		origDir, _ := os.Getwd()
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(origDir) })

		result, err := Discover("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Walk file overrides global URL.
		if result.Config.URL != "https://local.example.com" {
			t.Errorf("URL = %q, want local value", result.Config.URL)
		}
		// Global database still present.
		if result.Config.Database != "globaldb" {
			t.Errorf("Database = %q, want globaldb", result.Config.Database)
		}
	})
}
