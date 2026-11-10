package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/seletz/odoo-work-cli/internal/output/outputtest"
)

// TestClockAdd runs clock add against the fake. populated() has record #20
// on 2026-07-14 08:00–12:00.
func TestClockAdd(t *testing.T) {
	prevDay := day.AddDate(0, 0, -1)
	overnightOut := day.Add(1*time.Hour + 30*time.Minute)

	tests := []struct {
		name      string
		extra     []odoo.AttendanceRecord
		args      []string
		wantKind  apperr.Kind
		wantNamed string // record the conflict message must name
	}{
		{
			name: "free day",
			args: []string{"--date", "2026-07-13", "--in", "08:00", "--out", "12:00"},
		},
		{
			name: "second record after a break",
			args: []string{"--date", "2026-07-14", "--in", "13:00", "--out", "17:30"},
		},
		{
			name: "adjacent edges are not an overlap",
			args: []string{"--date", "2026-07-14", "--in", "12:00", "--out", "13:00"},
		},
		{
			name:     "overlap at the start",
			args:     []string{"--date", "2026-07-14", "--in", "11:00", "--out", "14:00"},
			wantKind: apperr.KindConflict, wantNamed: "#20",
		},
		{
			name:     "overlap at the end",
			args:     []string{"--date", "2026-07-14", "--in", "07:00", "--out", "09:00"},
			wantKind: apperr.KindConflict, wantNamed: "#20",
		},
		{
			name:     "fully inside",
			args:     []string{"--date", "2026-07-14", "--in", "09:00", "--out", "10:00"},
			wantKind: apperr.KindConflict, wantNamed: "#20",
		},
		{
			name:     "previous-day record past midnight",
			extra:    []odoo.AttendanceRecord{{ID: 30, CheckIn: prevDay.Add(22 * time.Hour), CheckOut: &overnightOut, WorkedHours: 3.5}},
			args:     []string{"--date", "2026-07-14", "--in", "01:00", "--out", "04:00"},
			wantKind: apperr.KindConflict, wantNamed: "#30",
		},
		{
			name:     "running record",
			extra:    []odoo.AttendanceRecord{{ID: 31, CheckIn: day.Add(13 * time.Hour)}},
			args:     []string{"--date", "2026-07-14", "--in", "14:00", "--out", "15:00"},
			wantKind: apperr.KindConflict, wantNamed: "#31",
		},
		{
			name:     "running record from an earlier day",
			extra:    []odoo.AttendanceRecord{{ID: 32, CheckIn: day.AddDate(0, 0, -3).Add(9 * time.Hour)}},
			args:     []string{"--date", "2026-07-14", "--in", "13:00", "--out", "15:00"},
			wantKind: apperr.KindConflict, wantNamed: "#32",
		},
		{
			name:     "needs both times",
			args:     []string{"--date", "2026-07-14", "--in", "13:00"},
			wantKind: apperr.KindValidation,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := populated()
			fake.Attendances = append(fake.Attendances, tt.extra...)
			before := len(fake.Attendances)

			args := append([]string{"clock", "add", "--json"}, tt.args...)
			code, out, stderr := execute(t, fakeSetup(fake), args...)

			if tt.wantKind != "" {
				if code != 1 {
					t.Fatalf("exit code %d, want 1 (stdout %s, stderr %s)", code, out, stderr)
				}
				if kind := errorKind(t, out); kind != tt.wantKind {
					t.Errorf("kind = %q, want %q (%s)", kind, tt.wantKind, out)
				}
				if tt.wantNamed != "" && !strings.Contains(string(out), tt.wantNamed) {
					t.Errorf("conflict message should name %s: %s", tt.wantNamed, out)
				}
				if len(fake.Attendances) != before {
					t.Errorf("records written on failure: %d → %d", before, len(fake.Attendances))
				}
				return
			}

			if code != 0 {
				t.Fatalf("exit code %d, stdout %s, stderr %s", code, out, stderr)
			}
			outputtest.AssertObject(t, output.SchemaAttendance, out)
			if len(fake.Attendances) != before+1 {
				t.Fatalf("expected one created record, have %d (was %d)", len(fake.Attendances), before)
			}
			var got map[string]any
			_ = json.Unmarshal(out, &got)
			if got["id"] != 99.0 {
				t.Errorf("id = %v, want 99", got["id"])
			}
		})
	}
}

func TestClockAdd_DayWithBreakListsBoth(t *testing.T) {
	fake := populated()
	code, out, stderr := execute(t, fakeSetup(fake), "clock", "add", "--date", "2026-07-14", "--in", "13:00", "--out", "17:30")
	if code != 0 {
		t.Fatalf("exit code %d, stdout %s, stderr %s", code, out, stderr)
	}
	if !strings.Contains(string(out), "#99") || !strings.Contains(string(out), "13:00") {
		t.Errorf("table output should name the created record: %s", out)
	}

	code, out, stderr = execute(t, fakeSetup(fake), "clock", "list", "--json", "--date", "2026-07-14")
	if code != 0 {
		t.Fatalf("exit code %d, stderr %s", code, stderr)
	}
	if n := outputtest.AssertList(t, output.SchemaAttendance, out); n != 2 {
		t.Errorf("clock list shows %d records, want 2", n)
	}
}

func TestClockAdd_TableConflictNamesRecord(t *testing.T) {
	code, _, stderr := execute(t, fakeSetup(populated()), "clock", "add", "--date", "2026-07-14", "--in", "11:00", "--out", "14:00")
	if code != 1 {
		t.Fatalf("exit code %d, want 1", code)
	}
	for _, w := range []string{"overlaps", "#20", "08:00", "12:00", "nothing was changed"} {
		if !strings.Contains(stderr, w) {
			t.Errorf("stderr missing %q:\n%s", w, stderr)
		}
	}
}

func TestClockAdd_ConflictListsOverlapsOldestFirst(t *testing.T) {
	fake := populated()
	// Odoo returns records newest first; the running record comes first here.
	fake.Attendances = append([]odoo.AttendanceRecord{{ID: 31, CheckIn: day.Add(13 * time.Hour)}}, fake.Attendances...)
	code, _, stderr := execute(t, fakeSetup(fake), "clock", "add", "--date", "2026-07-14", "--in", "11:00", "--out", "14:00")
	if code != 1 {
		t.Fatalf("exit code %d, want 1", code)
	}
	if !strings.Contains(stderr, "overlaps attendance record #20;") {
		t.Errorf("message should name the earliest overlapping record #20:\n%s", stderr)
	}
	if strings.Index(stderr, "#20 ") > strings.Index(stderr, "#31 ") {
		t.Errorf("overlapping records not oldest first:\n%s", stderr)
	}
}

func TestClockAdd_OvernightShowsCheckOutDate(t *testing.T) {
	code, out, stderr := execute(t, fakeSetup(populated()), "clock", "add", "--date", "2026-07-14", "--in", "22:00", "--out", "2026-07-15 01:30")
	if code != 0 {
		t.Fatalf("exit code %d, stderr %s", code, stderr)
	}
	if !strings.Contains(string(out), "2026-07-14 22:00 - 2026-07-15 01:30") {
		t.Errorf("overnight record should show the check-out date: %s", out)
	}
}
