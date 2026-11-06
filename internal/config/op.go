package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/seletz/odoo-work-cli/internal/apperr"
)

// OPSecrets holds 1Password vault references (op:// URIs) for config fields.
// When present in the config file, these are resolved at runtime via the op CLI.
type OPSecrets struct {
	URL        string `toml:"url"`
	Database   string `toml:"database"`
	Username   string `toml:"username"`
	APIKey     string `toml:"api-key"`
	Password   string `toml:"password"`
	TOTPSecret string `toml:"totp_secret"`
}

// opInjectRunner abstracts the op inject call for testability.
// It takes a template string and returns the resolved output.
type opInjectRunner func(template string) (string, error)

// defaultOPTimeout bounds the op inject call. A locked 1Password otherwise
// blocks for about 60 s before op gives up with "authorization timeout".
const defaultOPTimeout = 5 * time.Second

// errOPLocked marks an op failure caused by a locked or unreachable
// 1Password (timeout or an explicit "not signed in"/"authorization" error).
var errOPLocked = errors.New("1password locked or unreachable")

// opTimeout returns the op inject timeout: ODOO_OP_TIMEOUT (a Go duration
// such as "10s") if set and valid, else defaultOPTimeout.
func opTimeout() time.Duration {
	if v := os.Getenv("ODOO_OP_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return defaultOPTimeout
}

// lockedMarkers are lowercase stderr fragments by which op reports a locked
// or signed-out vault.
var lockedMarkers = []string{
	"authorization timeout",
	"not signed in",
	"no accounts configured",
	"locked",
}

// runOPCommand runs bin with args, feeding stdin, and kills it after
// timeout. A timeout or a lock-related stderr wraps errOPLocked.
func runOPCommand(timeout time.Duration, bin string, args []string, stdin string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// Do not wait on pipes held open by grandchildren after the kill.
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("op inject: no answer within %s: %w", timeout, errOPLocked)
		}
		low := strings.ToLower(msg)
		for _, m := range lockedMarkers {
			if strings.Contains(low, m) {
				return "", fmt.Errorf("op inject: %s: %w", msg, errOPLocked)
			}
		}
		return "", fmt.Errorf("op inject: %s: %w", msg, err)
	}
	return stdout.String(), nil
}

// defaultOPInjectRunner calls the real op inject CLI with a timeout, so a
// locked 1Password fails fast instead of hanging.
func defaultOPInjectRunner(template string) (string, error) {
	return runOPCommand(opTimeout(), "op", []string{"inject"}, template)
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
		{"api-key", cfg.OPSecrets.APIKey, &cfg.Password},
		{"password", cfg.OPSecrets.Password, &cfg.WebPassword},
		{"totp_secret", cfg.OPSecrets.TOTPSecret, &cfg.TOTPSecret},
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
		if errors.Is(err, errOPLocked) {
			return apperr.Wrapf(apperr.KindSecretsLocked, err,
				"1Password is locked or not reachable: unlock it and retry")
		}
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
