package main

import (
	"errors"
	"testing"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// TestInputErrorsBeforeSecrets checks that pure input errors are reported
// as validation without loading config or secrets (1Password), even when
// the secrets backend is locked (#84).
func TestInputErrorsBeforeSecrets(t *testing.T) {
	assertInputErrorsBeforeSecrets(t, inputErrorCases)
}

type inputErrorCase = struct {
	name string
	args []string
}

var inputErrorCases = []inputErrorCase{
	{"entries bad week", []string{"entries", "--week", "nope", "--json"}},
	{"entries bad date", []string{"entries", "--date", "nope", "--json"}},
	{"entries add bad hours", []string{"entries", "add", "--json", "--project-id", "1", "--hours", "x", "--description", "d"}},
	{"entries delete bad id", []string{"entries", "delete", "abc", "--json"}},
	{"tasks bad project id", []string{"tasks", "x", "--json"}},
	{"clock in future", []string{"clock", "in", "--json", "--at", "2999-01-01 08:00"}},
	{"entries update bad id", []string{"entries", "update", "abc", "--json", "--hours", "1"}},
	{"entries update no flags", []string{"entries", "update", "5", "--json"}},
	{"clock out future", []string{"clock", "out", "--json", "--at", "2999-01-01 08:00"}},
	{"clock list date and week", []string{"clock", "list", "--date", "2026-11-06", "--week", "2026-W45", "--json"}},
	{"clock list bad date", []string{"clock", "list", "--date", "nope", "--json"}},
	{"clock list bad week", []string{"clock", "list", "--week", "nope", "--json"}},
	{"entries add missing required", []string{"entries", "add", "--json", "--project-id", "1"}},
	{"entries delete no arg", []string{"entries", "delete", "--json"}},
	{"json with format table", []string{"whoami", "--json", "--format", "table"}},
	{"clock add missing out", []string{"clock", "add", "--json", "--in", "08:00"}},
	{"clock add future", []string{"clock", "add", "--json", "--in", "2999-01-01 08:00", "--out", "2999-01-01 09:00"}},
	{"clock add out before in", []string{"clock", "add", "--json", "--date", "2026-07-14", "--in", "12:00", "--out", "08:00"}},
}

func assertInputErrorsBeforeSecrets(t *testing.T, tests []inputErrorCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			locked := func(string) (*config.Config, odoo.Client, error) {
				calls++
				return nil, nil, apperr.New(apperr.KindSecretsLocked, "locked")
			}
			code, out, _ := execute(t, locked, tt.args...)
			if code != 1 {
				t.Fatalf("exit code = %d, want 1", code)
			}
			if calls != 0 {
				t.Fatalf("setup called %d times, want 0", calls)
			}
			if kind := errorKind(t, out); kind != apperr.KindValidation {
				t.Fatalf("kind = %q, want validation; out=%s", kind, out)
			}
		})
	}
}

func TestSetupErrorStillReported(t *testing.T) {
	locked := func(string) (*config.Config, odoo.Client, error) {
		return nil, nil, errors.New("boom")
	}
	code, out, _ := execute(t, locked, "whoami", "--json")
	if code != 1 {
		t.Fatalf("exit code = %d", code)
	}
	if kind := errorKind(t, out); kind != apperr.KindConfig && kind != apperr.KindInternal {
		t.Fatalf("kind = %q", kind)
	}
}
