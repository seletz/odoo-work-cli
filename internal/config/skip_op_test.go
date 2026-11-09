package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// opSecretsTOML references every field [op_secrets] can set.
const opSecretsTOML = `[op_secrets]
url         = "op://V/i/url"
database    = "op://V/i/database"
username    = "op://V/i/username"
api-key     = "op://V/i/api-key"
password    = "op://V/i/password"
totp_secret = "op://V/i/totp"
`

// allEnvVars are the env vars that can take the place of [op_secrets].
var allEnvVars = map[string]string{
	"ODOO_URL":          "https://env.example.com",
	"ODOO_DATABASE":     "envdb",
	"ODOO_USERNAME":     "env-user",
	"ODOO_PASSWORD":     "env-api-key",
	"ODOO_WEB_PASSWORD": "env-web-pw",
	"ODOO_TOTP_SECRET":  "ENVTOTP",
}

// countingRunner resolves every op:// reference to "op-<field>" and
// records the templates it was called with.
type countingRunner struct{ templates []string }

func (r *countingRunner) run(template string) (string, error) {
	r.templates = append(r.templates, template)
	var out strings.Builder
	for _, line := range strings.Split(template, "\n") {
		key, _, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out.WriteString(key + "=op-" + key + "\n")
	}
	return out.String(), nil
}

func TestDiscover_SkipsOP(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		// wantCalls is the number of op inject calls.
		wantCalls int
		// wantKeys are the keys the op template must contain (when called).
		wantKeys []string
		// wantMissing, when set, is a field Validate must name.
		wantMissing string
	}{
		{
			name:      "env complete: op not called",
			env:       allEnvVars,
			wantCalls: 0,
		},
		{
			name: "env partial: op resolves only the missing fields",
			env: map[string]string{
				"ODOO_URL": "https://env.example.com", "ODOO_DATABASE": "envdb",
				"ODOO_USERNAME": "env-user", "ODOO_PASSWORD": "env-api-key",
			},
			wantCalls: 1,
			wantKeys:  []string{"password", "totp_secret"},
		},
		{
			name:      "env empty: op resolves all fields",
			env:       map[string]string{},
			wantCalls: 1,
			wantKeys:  []string{"url", "database", "username", "api-key", "password", "totp_secret"},
		},
		{
			name: "NO_OP with complete env: op not called",
			env: merge(allEnvVars, map[string]string{
				"ODOO_WORK_CLI_NO_OP": "1",
			}),
			wantCalls: 0,
		},
		{
			name: "NO_OP with missing api key: op not called, config names field",
			env: map[string]string{
				"ODOO_WORK_CLI_NO_OP": "1",
				"ODOO_URL":            "https://env.example.com", "ODOO_DATABASE": "envdb",
				"ODOO_USERNAME": "env-user",
			},
			wantCalls:   0,
			wantMissing: "ODOO_PASSWORD",
		},
		{
			name:      "NO_OP=0 is off",
			env:       map[string]string{"ODOO_WORK_CLI_NO_OP": "0"},
			wantCalls: 1,
			wantKeys:  []string{"url", "database", "username", "api-key", "password", "totp_secret"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k := range allEnvVars {
				t.Setenv(k, "")
			}
			t.Setenv("ODOO_WORK_CLI_NO_OP", "")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			runner := &countingRunner{}
			stubOP(t, runner.run)

			cfgPath := filepath.Join(t.TempDir(), "c.toml")
			if err := os.WriteFile(cfgPath, []byte(opSecretsTOML), 0o644); err != nil {
				t.Fatal(err)
			}

			result, err := Discover(cfgPath)
			if err != nil {
				t.Fatalf("Discover: %v", err)
			}
			if got := len(runner.templates); got != tt.wantCalls {
				t.Fatalf("op inject calls = %d, want %d", got, tt.wantCalls)
			}
			if tt.wantCalls > 0 {
				tmpl := runner.templates[0]
				lines := strings.Count(tmpl, "\n")
				if lines != len(tt.wantKeys) {
					t.Errorf("template has %d lines, want %d:\n%s", lines, len(tt.wantKeys), tmpl)
				}
				for _, k := range tt.wantKeys {
					if !strings.Contains(tmpl, k+"={{") {
						t.Errorf("template lacks %s:\n%s", k, tmpl)
					}
				}
			}

			// Env always wins over op_secrets.
			for k, v := range tt.env {
				if k == "ODOO_URL" && result.Config.URL != v {
					t.Errorf("URL = %q, want env %q", result.Config.URL, v)
				}
				if k == "ODOO_PASSWORD" && result.Config.Password != v {
					t.Errorf("Password = %q, want env %q", result.Config.Password, v)
				}
			}

			verr := result.Config.Validate()
			if tt.wantMissing == "" {
				if tt.wantCalls == 0 && verr != nil {
					t.Errorf("Validate: %v", verr)
				}
				return
			}
			if verr == nil {
				t.Fatalf("Validate succeeded, want missing %s", tt.wantMissing)
			}
			if !strings.Contains(verr.Error(), tt.wantMissing) {
				t.Errorf("Validate error %q does not name %s", verr, tt.wantMissing)
			}
		})
	}
}

func TestDiscover_NoOPIgnoresPlainValues(t *testing.T) {
	for k := range allEnvVars {
		t.Setenv(k, "")
	}
	t.Setenv("ODOO_WORK_CLI_NO_OP", "true")
	runner := &countingRunner{}
	stubOP(t, runner.run)

	cfgPath := filepath.Join(t.TempDir(), "c.toml")
	if err := os.WriteFile(cfgPath, []byte("[op_secrets]\nurl = \"https://plain.example.com\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Discover(cfgPath)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if result.Config.URL != "" {
		t.Errorf("URL = %q, want empty: NO_OP ignores [op_secrets] completely", result.Config.URL)
	}
}

func TestNoOPEnabled(t *testing.T) {
	for v, want := range map[string]bool{
		"": false, "0": false, "false": false, "no": false,
		"1": true, "true": true, "TRUE": true, "yes": true,
	} {
		t.Setenv("ODOO_WORK_CLI_NO_OP", v)
		if got := NoOPEnabled(); got != want {
			t.Errorf("NoOPEnabled(%q) = %v, want %v", v, got, want)
		}
	}
}

// stubOP makes op look available and routes op inject to run.
func stubOP(t *testing.T, run opInjectRunner) {
	t.Helper()
	origRun, origAvail := opInject, opIsAvailable
	opInject = run
	opIsAvailable = func() bool { return true }
	t.Cleanup(func() { opInject, opIsAvailable = origRun, origAvail })
}

func merge(a, b map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}
