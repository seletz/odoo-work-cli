package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// noSetup fails the test when the real config/client setup is used: the
// fake backend must need no config, no 1Password and no network.
func noSetup(t *testing.T) setupFunc {
	return func(string) (*config.Config, odoo.Client, error) {
		t.Fatal("real setup called with the fake backend")
		return nil, nil, nil
	}
}

func useFakeBackend(t *testing.T) string {
	t.Helper()
	state := filepath.Join(t.TempDir(), "fake.json")
	t.Setenv("ODOO_WORK_CLI_BACKEND", "fake")
	t.Setenv("ODOO_WORK_CLI_FAKE_STATE", state)
	return state
}

// runJSON runs args, requires exit code wantCode and decodes stdout.
func runJSON(t *testing.T, setup setupFunc, wantCode int, out any, args ...string) {
	t.Helper()
	code, stdout, stderr := execute(t, setup, args...)
	if code != wantCode {
		t.Fatalf("%v: exit %d, want %d\nstdout: %s\nstderr: %s", args, code, wantCode, stdout, stderr)
	}
	if out != nil {
		if err := json.Unmarshal(stdout, out); err != nil {
			t.Fatalf("%v: decoding %q: %v", args, stdout, err)
		}
	}
}

type idOnly struct {
	ID int64 `json:"id"`
}

type errEnvelope struct {
	Error struct {
		Kind string `json:"kind"`
	} `json:"error"`
}

func TestFakeBackend_ClockWorkflow(t *testing.T) {
	useFakeBackend(t)
	setup := noSetup(t)
	const date = "2026-01-07"

	var list []idOnly
	runJSON(t, setup, 0, &list, "clock", "list", "--date", date, "--json")
	if len(list) != 0 {
		t.Fatalf("day not empty before create: %v", list)
	}

	var created struct {
		ID     int64           `json:"id"`
		Before json.RawMessage `json:"before"`
	}
	runJSON(t, setup, 0, &created, "clock", "edit", "--date", date, "--in", "08:00", "--out", "12:00", "--create-only", "--json")
	if created.ID == 0 || string(created.Before) != "null" {
		t.Fatalf("created = %+v (before %s)", created, created.Before)
	}

	runJSON(t, setup, 0, &list, "clock", "list", "--date", date, "--json")
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("clock list after create = %v, want record %d", list, created.ID)
	}

	var env errEnvelope
	runJSON(t, setup, 1, &env, "clock", "edit", "--date", date, "--in", "13:00", "--out", "14:00", "--create-only", "--json")
	if env.Error.Kind != "conflict" {
		t.Errorf("second --create-only kind = %q, want conflict", env.Error.Kind)
	}

	var edited struct {
		ID     int64   `json:"id"`
		Before *idOnly `json:"before"`
	}
	runJSON(t, setup, 0, &edited, "clock", "edit", "--date", date, "--out", "12:30", "--json")
	if edited.ID != created.ID || edited.Before == nil || edited.Before.ID != created.ID {
		t.Errorf("edit = %+v, want before set", edited)
	}
}

func TestFakeBackend_ClockInOutWebSession(t *testing.T) {
	useFakeBackend(t)
	setup := noSetup(t)

	var in idOnly
	runJSON(t, setup, 0, &in, "clock", "in", "--json")
	var env errEnvelope
	runJSON(t, setup, 1, &env, "clock", "in", "--json")
	if env.Error.Kind != "conflict" {
		t.Errorf("double clock in kind = %q, want conflict", env.Error.Kind)
	}
	var out idOnly
	runJSON(t, setup, 0, &out, "clock", "out", "--json")
	if out.ID != in.ID {
		t.Errorf("clock out closed %d, want %d", out.ID, in.ID)
	}
}

func TestFakeBackend_EntriesWorkflow(t *testing.T) {
	useFakeBackend(t)
	setup := noSetup(t)
	const date = "2026-01-07"

	var added idOnly
	runJSON(t, setup, 0, &added, "entries", "add", "--project-id", "1", "--task-id", "11",
		"--date", date, "--hours", "1:30", "--description", "fake work", "--json")

	var list []idOnly
	runJSON(t, setup, 0, &list, "entries", "--date", date, "--json")
	if len(list) != 1 || list[0].ID != added.ID {
		t.Fatalf("entries after add = %v, want %d", list, added.ID)
	}

	var env errEnvelope
	runJSON(t, setup, 1, &env, "entries", "delete", "999999", "--json")
	if env.Error.Kind != "not_found" {
		t.Errorf("delete unknown kind = %q, want not_found", env.Error.Kind)
	}

	runJSON(t, setup, 1, &env, "entries", "add", "--project-id", "999999",
		"--date", date, "--hours", "1", "--description", "x", "--json")
	if env.Error.Kind != "validation" {
		t.Errorf("add unknown project kind = %q, want validation", env.Error.Kind)
	}

	runJSON(t, setup, 0, nil, "entries", "delete", jsonID(added.ID), "--json")
	runJSON(t, setup, 0, &list, "entries", "--date", date, "--json")
	if len(list) != 0 {
		t.Errorf("entries after delete = %v", list)
	}
}

func TestFakeBackend_Selection(t *testing.T) {
	t.Run("flag selects fake", func(t *testing.T) {
		t.Setenv("ODOO_WORK_CLI_BACKEND", "")
		t.Setenv("ODOO_WORK_CLI_FAKE_STATE", filepath.Join(t.TempDir(), "s.json"))
		var user struct {
			Name string `json:"name"`
		}
		runJSON(t, noSetup(t), 0, &user, "--backend", "fake", "whoami", "--json")
		if user.Name != "Fake User" {
			t.Errorf("whoami name = %q", user.Name)
		}
	})
	t.Run("flag odoo overrides env", func(t *testing.T) {
		useFakeBackend(t)
		calls := 0
		setup := func(string) (*config.Config, odoo.Client, error) {
			calls++
			return &config.Config{}, nil, nil
		}
		execute(t, setup, "--backend", "odoo", "whoami", "--json")
		if calls != 1 {
			t.Errorf("real setup calls = %d, want 1", calls)
		}
	})
	t.Run("invalid flag", func(t *testing.T) {
		t.Setenv("ODOO_WORK_CLI_BACKEND", "")
		var env errEnvelope
		runJSON(t, noSetup(t), 1, &env, "--backend", "bogus", "whoami", "--json")
		if env.Error.Kind != "validation" {
			t.Errorf("kind = %q, want validation", env.Error.Kind)
		}
	})
	t.Run("invalid env", func(t *testing.T) {
		t.Setenv("ODOO_WORK_CLI_BACKEND", "bogus")
		var env errEnvelope
		runJSON(t, noSetup(t), 1, &env, "whoami", "--json")
		if env.Error.Kind != "config" {
			t.Errorf("kind = %q, want config", env.Error.Kind)
		}
	})
}

func jsonID(id int64) string {
	b, _ := json.Marshal(id)
	return string(b)
}
