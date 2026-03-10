package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestDefaultConfigTemplate_NonEmpty(t *testing.T) {
	data := DefaultConfigTemplate()
	if len(data) == 0 {
		t.Fatal("DefaultConfigTemplate() returned empty data")
	}
}

func TestDefaultConfigTemplate_ValidTOML(t *testing.T) {
	data := DefaultConfigTemplate()
	var raw map[string]interface{}
	if err := toml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("DefaultConfigTemplate() is not valid TOML: %v", err)
	}
}

func TestDefaultConfigTemplate_NoPasswordField(t *testing.T) {
	data := DefaultConfigTemplate()
	var raw map[string]interface{}
	if err := toml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := raw["password"]; ok {
		t.Fatal("DefaultConfigTemplate() must not contain a password field")
	}
}

func TestInstallConfig_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath: %v", err)
	}

	if err := InstallConfig(path); err != nil {
		t.Fatalf("InstallConfig: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading installed config: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("installed config file is empty")
	}

	// Verify it is valid TOML and loads without error.
	if _, err := LoadFromTOML(path); err != nil {
		t.Fatalf("LoadFromTOML on installed config: %v", err)
	}
}

func TestInstallConfig_RefusesOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath: %v", err)
	}

	// First install succeeds.
	if err := InstallConfig(path); err != nil {
		t.Fatalf("first InstallConfig: %v", err)
	}

	// Second install must fail.
	err = InstallConfig(path)
	if err == nil {
		t.Fatal("expected error on second install, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected 'already exists' error, got: %v", err)
	}
}

func TestInstallConfig_CreatesParentDirs(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "deep", "nested", "config.toml")

	if err := InstallConfig(path); err != nil {
		t.Fatalf("InstallConfig with nested dirs: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created at nested path: %v", err)
	}
}
