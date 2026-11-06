package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/odoo/odootest"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/seletz/odoo-work-cli/internal/output/outputtest"
	goOdoo "github.com/skilld-labs/go-odoo"
)

// fakeSetup returns a setup function that hands out fake instead of a
// real Odoo connection.
func fakeSetup(fake *odootest.Fake) setupFunc {
	return func(string) (*config.Config, odoo.Client, error) {
		return &config.Config{}, fake, nil
	}
}

func failingSetup(err error) setupFunc {
	return func(string) (*config.Config, odoo.Client, error) { return nil, nil, err }
}

// execute runs the CLI with args and returns exit code, stdout and stderr.
func execute(t *testing.T, setup setupFunc, args ...string) (int, []byte, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr, setup)
	return code, stdout.Bytes(), stderr.String()
}

var (
	day      = time.Date(2026, 7, 14, 0, 0, 0, 0, time.Local)
	in0800   = day.Add(8 * time.Hour)
	out1200  = day.Add(12 * time.Hour)
	in1300   = day.Add(13 * time.Hour)
	entryOne = odoo.TimesheetEntry{
		ID: 11, Date: "2026-07-14", ProjectID: 3, Project: "A very long project name that used to shift the table columns",
		TaskID: 5, Task: "Review", Name: "code review", Hours: 1.5, Employee: "Pat", ValidatedStatus: "draft", Company: "Co",
	}
)

func populated() *odootest.Fake {
	return &odootest.Fake{
		NextID:     99,
		User:       odoo.UserInfo{ID: 2, Name: "Pat", Login: "pat", Email: "pat@example.com", Company: "Co"},
		Projects:   []odoo.ProjectInfo{{ID: 3, Name: "Internal", Active: true, Customer: "ACME"}, {ID: 4, Name: "Bare"}},
		Tasks:      []odoo.TaskInfo{{ID: 5, Name: "Review", ProjectID: 3, Project: "Internal", Active: true}},
		Timesheets: []odoo.TimesheetEntry{entryOne, {ID: 12, Date: "2026-07-15", ProjectID: 3, Project: "Internal", Name: "no task", Hours: 0.25}},
		Attendances: []odoo.AttendanceRecord{
			{ID: 20, EmployeeID: 7, Employee: "Pat", CheckIn: in0800, CheckOut: &out1200, WorkedHours: 4},
		},
		Status: odoo.AttendanceStatus{
			ClockedIn: true, CurrentID: 21, CheckIn: &in1300, TotalHours: 4,
			Periods: []odoo.AttendanceRecord{
				{ID: 20, CheckIn: in0800, CheckOut: &out1200, WorkedHours: 4},
				{ID: 21, CheckIn: in1300},
			},
		},
	}
}

type shape int

const (
	list shape = iota
	object
)

// TestJSONOutput_Schemas runs every command that supports --json with a
// populated fake client and validates stdout against its schema.
func TestJSONOutput_Schemas(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		schema    string
		shape     shape
		wantItems int
	}{
		{name: "projects", args: []string{"projects", "--json"}, schema: output.SchemaProject, shape: list, wantItems: 2},
		{name: "tasks", args: []string{"tasks", "3", "--format", "json"}, schema: output.SchemaTask, shape: list, wantItems: 1},
		{name: "entries", args: []string{"entries", "--json", "--week", "2026-W29"}, schema: output.SchemaEntry, shape: list, wantItems: 2},
		{name: "entries filtered", args: []string{"entries", "--json", "--week", "2026-W29", "--task", "review"}, schema: output.SchemaEntry, shape: list, wantItems: 1},
		{name: "timesheets", args: []string{"timesheets", "--json", "--week", "2026-W29"}, schema: output.SchemaEntry, shape: list, wantItems: 2},
		{name: "whoami", args: []string{"whoami", "--json"}, schema: output.SchemaUser, shape: object},
		{name: "clock status", args: []string{"clock", "status", "--json"}, schema: output.SchemaClockStatus, shape: object},
		{name: "entries add", args: []string{"entries", "add", "--json", "--project-id", "3", "--date", "2026-07-14", "--hours", "2:30", "--description", "x"}, schema: output.SchemaEntry, shape: object},
		{name: "entries update", args: []string{"entries", "update", "11", "--json", "--hours", "2"}, schema: output.SchemaEntry, shape: object},
		{name: "entries delete", args: []string{"entries", "delete", "11", "--json"}, schema: output.SchemaEntry, shape: object},
		{name: "clock in", args: []string{"clock", "in", "--json"}, schema: output.SchemaAttendance, shape: object},
		{name: "clock in --at", args: []string{"clock", "in", "--json", "--at", "2026-07-14 13:00"}, schema: output.SchemaAttendance, shape: object},
		{name: "clock out", args: []string{"clock", "out", "--json"}, schema: output.SchemaAttendance, shape: object},
		{name: "clock out --at", args: []string{"clock", "out", "--json", "--at", "2026-07-14 12:00"}, schema: output.SchemaAttendance, shape: object},
		{name: "clock edit existing", args: []string{"clock", "edit", "--json", "--date", "2026-07-14", "--in", "08:15"}, schema: output.SchemaAttendance, shape: object},
		{name: "clock edit creates", args: []string{"clock", "edit", "--json", "--date", "2026-07-13", "--in", "08:00", "--out", "16:00"}, schema: output.SchemaAttendance, shape: object},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out, stderr := execute(t, fakeSetup(populated()), tt.args...)
			if code != 0 {
				t.Fatalf("exit code %d, stdout %s, stderr %s", code, out, stderr)
			}
			switch tt.shape {
			case list:
				if n := outputtest.AssertList(t, tt.schema, out); n != tt.wantItems {
					t.Errorf("got %d items, want %d", n, tt.wantItems)
				}
			case object:
				outputtest.AssertObject(t, tt.schema, out)
			}
		})
	}
}

func TestJSONLOutput_Schemas(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		schema    string
		wantLines int
	}{
		{name: "projects", args: []string{"projects", "--format", "jsonl"}, schema: output.SchemaProject, wantLines: 2},
		{name: "entries", args: []string{"entries", "--format=jsonl", "--week", "2026-W29"}, schema: output.SchemaEntry, wantLines: 2},
		{name: "whoami", args: []string{"whoami", "--format", "jsonl"}, schema: output.SchemaUser, wantLines: 1},
		{name: "clock status", args: []string{"clock", "status", "--format", "jsonl"}, schema: output.SchemaClockStatus, wantLines: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out, stderr := execute(t, fakeSetup(populated()), tt.args...)
			if code != 0 {
				t.Fatalf("exit code %d, stderr %s", code, stderr)
			}
			if n := outputtest.AssertLines(t, tt.schema, out); n != tt.wantLines {
				t.Errorf("got %d lines, want %d", n, tt.wantLines)
			}
		})
	}
}

func TestJSONOutput_EmptyResultsAreEmptyArrays(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "projects", args: []string{"projects", "--json"}},
		{name: "tasks", args: []string{"tasks", "--json"}},
		{name: "entries", args: []string{"entries", "--json"}},
		{name: "entries no match", args: []string{"entries", "--json", "--project", "nothing-matches"}},
		{name: "timesheets", args: []string{"timesheets", "--json"}},
		{name: "entries jsonl", args: []string{"entries", "--format", "jsonl"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out, stderr := execute(t, fakeSetup(&odootest.Fake{}), tt.args...)
			if code != 0 {
				t.Fatalf("exit code %d, stderr %s", code, stderr)
			}
			if got := strings.TrimSpace(string(out)); got != "[]" {
				t.Errorf("stdout = %q, want []", got)
			}
		})
	}
}

func TestJSONOutput_StatusNotClockedIn(t *testing.T) {
	code, out, _ := execute(t, fakeSetup(&odootest.Fake{}), "clock", "status", "--json")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	outputtest.AssertObject(t, output.SchemaClockStatus, out)
	var got map[string]any
	_ = json.Unmarshal(out, &got)
	if got["clocked_in"] != false || got["periods"] == nil {
		t.Errorf("unexpected status: %s", out)
	}
}

func TestJSONOutput_WriteCommandsReturnChangedObject(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		check func(t *testing.T, fake *odootest.Fake, got map[string]any)
	}{
		{
			name: "add returns created entry",
			args: []string{"entries", "add", "--json", "--project-id", "3", "--date", "2026-07-14", "--hours", "2:30", "--description", "new"},
			check: func(t *testing.T, _ *odootest.Fake, got map[string]any) {
				if got["id"] != 99.0 || got["minutes"] != 150.0 || got["description"] != "new" {
					t.Errorf("got %v", got)
				}
			},
		},
		{
			name: "update returns updated entry",
			args: []string{"entries", "update", "11", "--json", "--description", "changed"},
			check: func(t *testing.T, _ *odootest.Fake, got map[string]any) {
				if got["id"] != 11.0 || got["description"] != "changed" {
					t.Errorf("got %v", got)
				}
			},
		},
		{
			name: "delete returns deleted entry",
			args: []string{"entries", "delete", "11", "--json"},
			check: func(t *testing.T, fake *odootest.Fake, got map[string]any) {
				if got["id"] != 11.0 || len(fake.Deleted) != 1 {
					t.Errorf("got %v, deleted %v", got, fake.Deleted)
				}
			},
		},
		{
			name: "clock edit creates missing record",
			args: []string{"clock", "edit", "--json", "--date", "2026-07-13", "--in", "08:00", "--out", "16:00"},
			check: func(t *testing.T, _ *odootest.Fake, got map[string]any) {
				if got["id"] != 99.0 || got["worked_minutes"] != 480.0 || got["running"] != false {
					t.Errorf("got %v", got)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := populated()
			code, out, stderr := execute(t, fakeSetup(fake), tt.args...)
			if code != 0 {
				t.Fatalf("exit code %d, stderr %s", code, stderr)
			}
			var got map[string]any
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatalf("decode: %v\n%s", err, out)
			}
			tt.check(t, fake, got)
		})
	}
}

// TestJSONOutput_Errors checks the error contract: exactly one JSON object
// on stdout with a stable kind, nothing else on stdout, exit code 1.
func TestJSONOutput_Errors(t *testing.T) {
	netErr := &url.Error{Op: "Post", URL: "https://odoo.invalid/xmlrpc/2/common", Err: &net.DNSError{Err: "no such host", Name: "odoo.invalid"}}

	tests := []struct {
		name     string
		setup    setupFunc
		args     []string
		wantKind apperr.Kind
	}{
		{name: "config error", setup: failingSetup(apperr.New(apperr.KindConfig, "url is required")), args: []string{"whoami", "--json"}, wantKind: apperr.KindConfig},
		{name: "missing config file", setup: defaultSetup, args: []string{"--config", "/nonexistent/odoo-work-cli.toml", "whoami", "--json"}, wantKind: apperr.KindConfig},
		{name: "auth error on connect", setup: failingSetup(fmt.Errorf("connecting to odoo: %w", goOdoo.ErrClientAuthentication)), args: []string{"projects", "--json"}, wantKind: apperr.KindAuth},
		{name: "network error on connect", setup: failingSetup(netErr), args: []string{"projects", "--json"}, wantKind: apperr.KindNetwork},
		{name: "access fault from odoo", setup: fakeSetup(&odootest.Fake{Err: errors.New("Fault(4): access denied")}), args: []string{"entries", "--json"}, wantKind: apperr.KindAccessDenied},
		{name: "conflict", setup: fakeSetup(&odootest.Fake{}), args: []string{"clock", "out", "--json"}, wantKind: apperr.KindConflict},
		{name: "not found on update", setup: fakeSetup(&odootest.Fake{}), args: []string{"entries", "update", "5", "--json", "--hours", "1"}, wantKind: apperr.KindNotFound},
		{name: "not found on delete", setup: fakeSetup(&odootest.Fake{}), args: []string{"entries", "delete", "5", "--json"}, wantKind: apperr.KindNotFound},
		{name: "invalid entry id", setup: fakeSetup(&odootest.Fake{}), args: []string{"entries", "delete", "abc", "--json"}, wantKind: apperr.KindValidation},
		{name: "invalid hours", setup: fakeSetup(&odootest.Fake{}), args: []string{"entries", "add", "--json", "--project-id", "1", "--hours", "x", "--description", "d"}, wantKind: apperr.KindValidation},
		{name: "invalid week", setup: fakeSetup(&odootest.Fake{}), args: []string{"entries", "--json", "--week", "nope"}, wantKind: apperr.KindValidation},
		{name: "invalid project id arg", setup: fakeSetup(&odootest.Fake{}), args: []string{"tasks", "x", "--json"}, wantKind: apperr.KindValidation},
		{name: "future clock time", setup: fakeSetup(&odootest.Fake{}), args: []string{"clock", "in", "--json", "--at", "2999-01-01 08:00"}, wantKind: apperr.KindValidation},
		{name: "unknown flag before --json", setup: fakeSetup(&odootest.Fake{}), args: []string{"entries", "--bogus", "--json"}, wantKind: apperr.KindValidation},
		{name: "missing required flag", setup: fakeSetup(&odootest.Fake{}), args: []string{"entries", "add", "--json", "--project-id", "1"}, wantKind: apperr.KindValidation},
		{name: "wrong arg count", setup: fakeSetup(&odootest.Fake{}), args: []string{"entries", "delete", "--json"}, wantKind: apperr.KindValidation},
		{name: "json conflicts with format", setup: fakeSetup(&odootest.Fake{}), args: []string{"whoami", "--json", "--format", "table"}, wantKind: apperr.KindValidation},
		{name: "jsonl error", setup: fakeSetup(&odootest.Fake{}), args: []string{"clock", "out", "--format", "jsonl"}, wantKind: apperr.KindConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out, _ := execute(t, tt.setup, tt.args...)
			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			outputtest.AssertObject(t, output.SchemaError, out)
			var env output.ErrorEnvelope
			if err := json.Unmarshal(out, &env); err != nil {
				t.Fatal(err)
			}
			if env.Error.Kind != tt.wantKind {
				t.Errorf("kind = %q (%s), want %q", env.Error.Kind, env.Error.Message, tt.wantKind)
			}
		})
	}
}

func TestTableOutput_ErrorsGoToStderr(t *testing.T) {
	code, out, stderr := execute(t, fakeSetup(&odootest.Fake{}), "clock", "out")
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if len(out) != 0 {
		t.Errorf("stdout = %q, want empty", out)
	}
	if !strings.Contains(stderr, "Error: not clocked in") {
		t.Errorf("stderr = %q", stderr)
	}
	if strings.Contains(stderr, "Usage:") {
		t.Errorf("runtime errors must not print usage: %q", stderr)
	}
}

func TestTableOutput_Unchanged(t *testing.T) {
	code, out, _ := execute(t, fakeSetup(populated()), "entries", "--week", "2026-W29")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(string(out), "Week: 2026-07-13 to 2026-07-19") || !strings.Contains(string(out), "code review") {
		t.Errorf("unexpected table output:\n%s", out)
	}
}

func TestTableOutput_UsageErrorKeepsCobraFormat(t *testing.T) {
	code, out, stderr := execute(t, fakeSetup(&odootest.Fake{}), "entries", "--bogus")
	if code != 1 || len(out) != 0 {
		t.Errorf("exit = %d, stdout = %q", code, out)
	}
	if !strings.HasPrefix(stderr, "Error: unknown flag: --bogus") || !strings.Contains(stderr, "Usage:") {
		t.Errorf("stderr = %q, want cobra's error followed by usage", stderr)
	}
}

func TestJSONOutput_UsageErrorsOutsideOutputCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "command without output flags", args: []string{"fields", "--json"}},
		{name: "unknown command", args: []string{"bogus", "--format", "jsonl"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out, stderr := execute(t, fakeSetup(&odootest.Fake{}), tt.args...)
			if code != 1 || stderr != "" {
				t.Errorf("exit = %d, stderr = %q", code, stderr)
			}
			outputtest.AssertObject(t, output.SchemaError, out)
			var env output.ErrorEnvelope
			_ = json.Unmarshal(out, &env)
			if env.Error.Kind != apperr.KindValidation {
				t.Errorf("kind = %q, want validation", env.Error.Kind)
			}
		})
	}
}

func TestTableOutput_UnknownCommandHasNoUsage(t *testing.T) {
	code, out, stderr := execute(t, fakeSetup(&odootest.Fake{}), "bogus")
	if code != 1 || len(out) != 0 {
		t.Errorf("exit = %d, stdout = %q", code, out)
	}
	if !strings.HasPrefix(stderr, `Error: unknown command "bogus"`) || strings.Contains(stderr, "Usage:") {
		t.Errorf("stderr = %q", stderr)
	}
}
