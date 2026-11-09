package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

const configFileName = ".odoo-work-cli.toml"

// DefaultConfigPath returns the path to the global config file.
// Uses $XDG_CONFIG_HOME/odoo-work-cli/config.toml if set,
// otherwise ~/.config/odoo-work-cli/config.toml.
func DefaultConfigPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determining home directory: %w", err)
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "odoo-work-cli", "config.toml"), nil
}

// WalkConfigFiles walks from startDir upward to the filesystem root,
// collecting .odoo-work-cli.toml files. Returns paths ordered root-most
// first (merge order: root-most has lowest priority).
func WalkConfigFiles(startDir string) ([]string, error) {
	startDir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, err
	}

	var files []string
	dir := startDir
	for {
		candidate := filepath.Join(dir, configFileName)
		if _, err := os.Stat(candidate); err == nil {
			files = append(files, candidate)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Reverse so root-most comes first.
	slices.Reverse(files)
	return files, nil
}

// DiscoverResult holds the result of configuration discovery.
type DiscoverResult struct {
	Files  []string // config files that were loaded, in merge order
	Config *Config
}

// Discover loads configuration using file discovery and environment variables.
//
// If configFlag is non-empty, only that file is loaded (skip discovery).
// Otherwise: load global config (if exists), walk files (root-most to cwd),
// then overlay env vars. Env vars always have highest priority.
func Discover(configFlag string) (*DiscoverResult, error) {
	result := &DiscoverResult{Config: &Config{}}

	if configFlag != "" {
		fileCfg, err := LoadFromTOML(configFlag)
		if err != nil {
			return nil, err
		}
		result.Config.Merge(fileCfg)
		result.Files = append(result.Files, configFlag)
	} else {
		// Global config.
		globalPath, err := DefaultConfigPath()
		if err != nil {
			return nil, err
		}
		if _, statErr := os.Stat(globalPath); statErr == nil {
			fileCfg, err := LoadFromTOML(globalPath)
			if err != nil {
				return nil, err
			}
			result.Config.Merge(fileCfg)
			result.Files = append(result.Files, globalPath)
		}

		// Walk config files.
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting working directory: %w", err)
		}
		walkFiles, err := WalkConfigFiles(cwd)
		if err != nil {
			return nil, err
		}
		for _, f := range walkFiles {
			fileCfg, err := LoadFromTOML(f)
			if err != nil {
				return nil, err
			}
			result.Config.Merge(fileCfg)
			result.Files = append(result.Files, f)
		}
	}

	envCfg := LoadFromEnv()

	if NoOPEnabled() {
		// ODOO_WORK_CLI_NO_OP: ignore [op_secrets] completely, plain
		// values included. Missing fields surface in Validate.
		result.Config.OPSecrets = nil
	}

	// Resolve op:// secrets if [op_secrets] is present and op CLI is
	// available. Fields already set by env vars are dropped first: env
	// wins anyway, so op is only called for what is really missing.
	if result.Config.OPSecrets != nil && opIsAvailable() {
		result.Config.OPSecrets = withoutEnvFields(result.Config.OPSecrets, envCfg)
		if err := resolveOPSecrets(result.Config, opInject); err != nil {
			return nil, fmt.Errorf("resolving 1Password secrets: %w", err)
		}
	}

	// Env vars always overlay last (highest priority).
	result.Config.Merge(envCfg)

	return result, nil
}
