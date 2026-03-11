package config

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// OPSecrets holds 1Password vault references (op:// URIs) for config fields.
// When present in the config file, these are resolved at runtime via the op CLI.
type OPSecrets struct {
	URL      string `toml:"url"`
	Database string `toml:"database"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

// opInjectRunner abstracts the op inject call for testability.
// It takes a template string and returns the resolved output.
type opInjectRunner func(template string) (string, error)

// defaultOPInjectRunner calls the real op inject CLI.
func defaultOPInjectRunner(template string) (string, error) {
	cmd := exec.Command("op", "inject")
	cmd.Stdin = strings.NewReader(template)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("op inject: %s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return stdout.String(), nil
}

// opAvailable checks whether the op CLI is on the PATH.
func opAvailable() bool {
	_, err := exec.LookPath("op")
	return err == nil
}

// resolveOPSecrets resolves op:// references in OPSecrets and applies them
// to the config using a single `op inject` call. Plain values (without op://
// prefix) are applied directly. Returns an error if op inject fails.
func resolveOPSecrets(cfg *Config, runner opInjectRunner) error {
	if cfg.OPSecrets == nil {
		return nil
	}

	type field struct {
		key    string
		ref    string
		target *string
	}
	fields := []field{
		{"url", cfg.OPSecrets.URL, &cfg.URL},
		{"database", cfg.OPSecrets.Database, &cfg.Database},
		{"username", cfg.OPSecrets.Username, &cfg.Username},
		{"password", cfg.OPSecrets.Password, &cfg.Password},
	}

	// Apply plain values directly; collect op:// refs for batch resolve.
	var opFields []field
	for _, f := range fields {
		if f.ref == "" {
			continue
		}
		if !strings.HasPrefix(f.ref, "op://") {
			*f.target = f.ref
			continue
		}
		opFields = append(opFields, f)
	}

	if len(opFields) == 0 {
		return nil
	}

	// Build template: one KEY={{ op://... }} per line.
	var tmpl strings.Builder
	for _, f := range opFields {
		fmt.Fprintf(&tmpl, "%s={{ %s }}\n", f.key, f.ref)
	}

	out, err := runner(tmpl.String())
	if err != nil {
		return err
	}

	// Parse resolved output: KEY=value per line.
	resolved := parseKeyValues(out)
	for _, f := range opFields {
		val, ok := resolved[f.key]
		if !ok {
			return fmt.Errorf("op inject: missing resolved value for %s", f.key)
		}
		*f.target = val
	}
	return nil
}

// parseKeyValues parses KEY=value lines into a map.
func parseKeyValues(s string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		result[key] = val
	}
	return result
}
