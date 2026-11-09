package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/odoo/odootest"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/seletz/odoo-work-cli/internal/output/outputtest"
)

func decodeObject(t *testing.T, out []byte) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("output is not a JSON object: %v\n%s", err, out)
	}
	return got
}

func assertNoWrites(t *testing.T, fake *odootest.Fake) {
	t.Helper()
	if len(fake.Created) != 0 || len(fake.Updated) != 0 || len(fake.Deleted) != 0 {
		t.Errorf("dry run wrote: created %v, updated %v, deleted %v", fake.Created, fake.Updated, fake.Deleted)
	}
}

func TestDryRun_WritesNothingAndMatchesRealCall(t *testing.T) {
	// Fields the dry run cannot know before Odoo has stored the entry.
	serverSide := []string{"id", "dry_run", "duplicates", "employee", "status", "company", "project", "task"}
	tests := []struct {
		name string
		args []string
	}{
		{"add", []string{"entries", "add", "--project-id", "3", "--task-id", "5", "--date", "2026-07-20", "--hours", "2:30", "--description", "new"}},
		{"add without task", []string{"entries", "add", "--project-id", "3", "--date", "2026-07-20", "--hours", "1", "--description", "new"}},
		{"update", []string{"entries", "update", "11", "--hours", "2", "--description", "changed"}},
		{"update moves project", []string{"entries", "update", "11", "--project-id", "4", "--task-id", "0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dryFake, realFake := populated(), populated()
			code, dryOut, stderr := execute(t, fakeSetup(dryFake), append(tt.args, "--json", "--dry-run")...)
			if code != 0 {
				t.Fatalf("dry run exit %d: %s %s", code, dryOut, stderr)
			}
			assertNoWrites(t, dryFake)
			if !reflect.DeepEqual(dryFake.Timesheets, populated().Timesheets) {
				t.Errorf("dry run changed the timesheets: %v", dryFake.Timesheets)
			}
			outputtest.AssertObject(t, output.SchemaEntryDryRun, dryOut)
			dry := decodeObject(t, dryOut)
			if dry["dry_run"] != true {
				t.Errorf("dry_run = %v, want true", dry["dry_run"])
			}

			code, realOut, stderr := execute(t, fakeSetup(realFake), append(tt.args, "--json")...)
			if code != 0 {
				t.Fatalf("real exit %d: %s %s", code, realOut, stderr)
			}
			real := decodeObject(t, realOut)
			for k, v := range real {
				if contains(serverSide, k) {
					continue
				}
				if !reflect.DeepEqual(dry[k], v) {
					t.Errorf("field %s: dry run %v, real call %v", k, dry[k], v)
				}
			}
		})
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func TestDryRun_AddHasNullIDAndResolvedNames(t *testing.T) {
	fake := populated()
	code, out, stderr := execute(t, fakeSetup(fake), "entries", "add", "--json", "--dry-run",
		"--project-id", "3", "--task-id", "5", "--date", "2026-07-20", "--hours", "2:30", "--description", "new")
	if code != 0 {
		t.Fatalf("exit %d: %s %s", code, out, stderr)
	}
	got := decodeObject(t, out)
	if id, ok := got["id"]; !ok || id != nil {
		t.Errorf("id = %v (present %v), want null", id, ok)
	}
	if got["project"] != "Internal" || got["task"] != "Review" || got["minutes"] != 150.0 {
		t.Errorf("got %v", got)
	}
}

func TestDryRun_UpdateKeepsIDAndMergesFlags(t *testing.T) {
	fake := populated()
	code, out, _ := execute(t, fakeSetup(fake), "entries", "update", "11", "--json", "--dry-run", "--hours", "2")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	got := decodeObject(t, out)
	if got["id"] != 11.0 || got["hours"] != 2.0 || got["description"] != "code review" || got["employee"] != "Pat" {
		t.Errorf("got %v", got)
	}
}

func TestDryRun_ValidationMatchesRealCall(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantKind apperr.Kind
	}{
		{"add unknown project", []string{"entries", "add", "--project-id", "999", "--hours", "1", "--description", "x"}, apperr.KindValidation},
		{"add unknown task", []string{"entries", "add", "--project-id", "3", "--task-id", "999", "--hours", "1", "--description", "x"}, apperr.KindValidation},
		{"add task of another project", []string{"entries", "add", "--project-id", "4", "--task-id", "5", "--hours", "1", "--description", "x"}, apperr.KindValidation},
		{"add bad hours", []string{"entries", "add", "--project-id", "3", "--hours", "x", "--description", "x"}, apperr.KindValidation},
		{"add missing project id", []string{"entries", "add", "--hours", "1", "--description", "x"}, apperr.KindValidation},
		{"update missing entry", []string{"entries", "update", "5", "--hours", "1"}, apperr.KindNotFound},
		{"update unknown project", []string{"entries", "update", "11", "--project-id", "999"}, apperr.KindValidation},
		{"update no flags", []string{"entries", "update", "11"}, apperr.KindValidation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := populated()
			code, out, _ := execute(t, fakeSetup(fake), append(tt.args, "--json", "--dry-run")...)
			if code != 1 {
				t.Fatalf("exit %d, want 1: %s", code, out)
			}
			var env struct{ Error struct{ Kind string } }
			if err := json.Unmarshal(out, &env); err != nil || env.Error.Kind != string(tt.wantKind) {
				t.Errorf("kind = %q, want %q (%s)", env.Error.Kind, tt.wantKind, out)
			}
			assertNoWrites(t, fake)
		})
	}
}

func TestDryRun_Duplicates(t *testing.T) {
	// Same day, project, task and description as entry 11.
	args := []string{"entries", "add", "--json", "--dry-run", "--project-id", "3", "--task-id", "5",
		"--date", "2026-07-14", "--hours", "1", "--description", "code review"}
	code, out, _ := execute(t, fakeSetup(populated()), args...)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	outputtest.AssertObject(t, output.SchemaEntryDryRun, out)
	dups, _ := decodeObject(t, out)["duplicates"].([]any)
	if len(dups) != 1 || dups[0].(map[string]any)["id"] != 11.0 {
		t.Errorf("duplicates = %v, want entry 11", dups)
	}

	// A different description is no duplicate.
	args[len(args)-1] = "something else"
	_, out, _ = execute(t, fakeSetup(populated()), args...)
	if dups, _ := decodeObject(t, out)["duplicates"].([]any); len(dups) != 0 {
		t.Errorf("duplicates = %v, want none", dups)
	}

	// An update does not report the entry itself.
	_, out, _ = execute(t, fakeSetup(populated()), "entries", "update", "11", "--json", "--dry-run", "--hours", "2")
	if dups, _ := decodeObject(t, out)["duplicates"].([]any); len(dups) != 0 {
		t.Errorf("update duplicates = %v, want none", dups)
	}
}

func TestDryRun_TableOutput(t *testing.T) {
	fake := populated()
	code, out, stderr := execute(t, fakeSetup(fake), "entries", "add", "--dry-run",
		"--project-id", "3", "--date", "2026-07-14", "--hours", "1", "--description", "code review", "--task-id", "5")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	for _, want := range []string{"Dry run", "Internal", "Review", "code review", "possible duplicate"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	assertNoWrites(t, fake)
}
