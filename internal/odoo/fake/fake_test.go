package fake

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// fixedNow is a Wednesday afternoon, local time.
var fixedNow = time.Date(2026, 7, 15, 15, 0, 0, 0, time.Local)

func open(t *testing.T, path string) *Client {
	t.Helper()
	c, err := Open(path, func() time.Time { return fixedNow })
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return c
}

func statePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "state.json")
}

func wantKind(t *testing.T, err error, kind apperr.Kind) {
	t.Helper()
	if err == nil {
		t.Fatalf("got no error, want %s", kind)
	}
	if got := apperr.KindOf(err); got != kind {
		t.Fatalf("kind = %s (%v), want %s", got, err, kind)
	}
}

func TestOpen_SeedsAndPersists(t *testing.T) {
	path := statePath(t)
	c := open(t, path)

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("state file not written: %v", err)
	}
	projects, err := c.ListProjects()
	if err != nil || len(projects) < 2 {
		t.Fatalf("projects = %v, %v; want seeded projects", projects, err)
	}
	tasks, err := c.ListTasks(projects[0].ID)
	if err != nil || len(tasks) == 0 {
		t.Fatalf("tasks = %v, %v; want seeded tasks", tasks, err)
	}
	for _, task := range tasks {
		if task.ProjectID != projects[0].ID {
			t.Errorf("task %d belongs to project %d, want %d", task.ID, task.ProjectID, projects[0].ID)
		}
	}
	monday := "2026-07-13"
	entries, err := c.ListTimesheets(monday, "2026-07-19")
	if err != nil || len(entries) == 0 {
		t.Fatalf("entries this week = %v, %v; want seeded entries", entries, err)
	}
	att, err := c.ListAttendance(time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local), time.Date(2026, 7, 20, 0, 0, 0, 0, time.Local))
	if err != nil || len(att) == 0 {
		t.Fatalf("attendance this week = %v, %v; want seeded records", att, err)
	}
	user, err := c.WhoAmI()
	if err != nil || user.Name == "" {
		t.Fatalf("WhoAmI = %v, %v", user, err)
	}
}

func TestTimesheets_PersistAcrossOpen(t *testing.T) {
	path := statePath(t)
	c := open(t, path)
	projects, _ := c.ListProjects()
	tasks, _ := c.ListTasks(projects[0].ID)

	id, err := c.CreateTimesheet(odoo.TimesheetWriteParams{
		ProjectID: projects[0].ID, TaskID: tasks[0].ID, Date: "2026-07-15", Name: "fake work", Hours: 1.5,
	})
	if err != nil {
		t.Fatalf("CreateTimesheet: %v", err)
	}

	c2 := open(t, path)
	e, err := c2.GetTimesheet(id)
	if err != nil {
		t.Fatalf("GetTimesheet after reopen: %v", err)
	}
	if e.Name != "fake work" || e.Hours != 1.5 || e.Project != projects[0].Name || e.Task != tasks[0].Name {
		t.Errorf("entry = %+v", e)
	}
	if e.ValidatedStatus != "draft" || e.Employee == "" {
		t.Errorf("status/employee = %q/%q", e.ValidatedStatus, e.Employee)
	}

	if err := c2.UpdateTimesheet(id, map[string]interface{}{"name": "changed", "unit_amount": 2.0}); err != nil {
		t.Fatalf("UpdateTimesheet: %v", err)
	}
	if err := c2.DeleteTimesheet(id); err != nil {
		t.Fatalf("DeleteTimesheet: %v", err)
	}
	_, err = open(t, path).GetTimesheet(id)
	wantKind(t, err, apperr.KindNotFound)
}

func TestTimesheets_Errors(t *testing.T) {
	c := open(t, statePath(t))
	projects, _ := c.ListProjects()
	other, _ := c.ListTasks(projects[1].ID)

	_, err := c.CreateTimesheet(odoo.TimesheetWriteParams{ProjectID: 999999, Date: "2026-07-15", Name: "x", Hours: 1})
	wantKind(t, err, apperr.KindValidation)

	_, err = c.CreateTimesheet(odoo.TimesheetWriteParams{ProjectID: projects[0].ID, TaskID: other[0].ID, Date: "2026-07-15", Name: "x", Hours: 1})
	wantKind(t, err, apperr.KindValidation)

	_, err = c.GetTimesheet(999999)
	wantKind(t, err, apperr.KindNotFound)
	wantKind(t, c.UpdateTimesheet(999999, map[string]interface{}{"name": "x"}), apperr.KindValidation)
	wantKind(t, c.DeleteTimesheet(999999), apperr.KindValidation)
}

func TestClockInOut_WebSessionPath(t *testing.T) {
	path := statePath(t)
	c := open(t, path)

	_, err := c.ClockOut()
	wantKind(t, err, apperr.KindConflict)

	id, err := c.ClockIn()
	if err != nil {
		t.Fatalf("ClockIn: %v", err)
	}
	_, err = c.ClockIn()
	wantKind(t, err, apperr.KindConflict)

	st, err := open(t, path).AttendanceStatus()
	if err != nil || !st.ClockedIn || st.CurrentID != id {
		t.Fatalf("status after clock in = %+v, %v", st, err)
	}

	rec, err := c.ClockOut()
	if err != nil {
		t.Fatalf("ClockOut: %v", err)
	}
	if rec.ID != id || rec.CheckOut == nil {
		t.Errorf("closed record = %+v", rec)
	}
}

func TestClockAt_And_Create(t *testing.T) {
	c := open(t, statePath(t))
	in := time.Date(2026, 7, 15, 8, 0, 0, 0, time.Local)

	_, err := c.ClockOutAt(in)
	wantKind(t, err, apperr.KindConflict)

	id, err := c.ClockInAt(in)
	if err != nil {
		t.Fatalf("ClockInAt: %v", err)
	}
	_, err = c.ClockOutAt(in.Add(-time.Hour))
	wantKind(t, err, apperr.KindValidation)
	rec, err := c.ClockOutAt(in.Add(4 * time.Hour))
	if err != nil {
		t.Fatalf("ClockOutAt: %v", err)
	}
	if rec.ID != id || rec.WorkedHours != 4 {
		t.Errorf("record = %+v", rec)
	}

	// Odoo refuses overlapping attendances (ValidationError).
	_, err = c.CreateAttendance(in.Add(time.Hour), in.Add(2*time.Hour))
	wantKind(t, err, apperr.KindValidation)
	_, err = c.CreateAttendance(in.Add(6*time.Hour), in.Add(5*time.Hour))
	wantKind(t, err, apperr.KindValidation)

	id2, err := c.CreateAttendance(in.Add(5*time.Hour), in.Add(6*time.Hour))
	if err != nil {
		t.Fatalf("CreateAttendance: %v", err)
	}
	if id2 == id {
		t.Errorf("IDs not unique: %d", id2)
	}

	newIn := in.Add(4*time.Hour + 30*time.Minute)
	edited, err := c.EditAttendance(id2, &newIn, nil)
	if err != nil {
		t.Fatalf("EditAttendance: %v", err)
	}
	if !edited.CheckIn.Equal(newIn) || edited.WorkedHours != 1.5 {
		t.Errorf("edited = %+v", edited)
	}
	early := in.Add(time.Hour)
	_, err = c.EditAttendance(id2, &early, nil)
	wantKind(t, err, apperr.KindValidation)

	_, err = c.EditAttendance(999999, &newIn, nil)
	wantKind(t, err, apperr.KindNotFound)
	_, err = c.GetAttendance(999999)
	wantKind(t, err, apperr.KindNotFound)
}

func TestListAttendance_IncludesOpenFromEarlier(t *testing.T) {
	c := open(t, statePath(t))
	in := time.Date(2026, 7, 14, 22, 0, 0, 0, time.Local)
	id, err := c.ClockInAt(in)
	if err != nil {
		t.Fatalf("ClockInAt: %v", err)
	}
	day := time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local)
	recs, err := c.ListAttendance(day, day.AddDate(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range recs {
		if r.ID == id {
			found = true
		}
	}
	if !found {
		t.Errorf("open record from the previous day missing: %+v", recs)
	}
}

func TestOpen_CorruptState(t *testing.T) {
	path := statePath(t)
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Open(path, time.Now)
	wantKind(t, err, apperr.KindConfig)
}

func TestStatePath(t *testing.T) {
	t.Setenv(StateEnvVar, "")
	if got, want := StatePath(), filepath.Join(os.TempDir(), DefaultStateFile); got != want {
		t.Errorf("StatePath() = %q, want %q", got, want)
	}
	t.Setenv(StateEnvVar, "/x/y.json")
	if got := StatePath(); got != "/x/y.json" {
		t.Errorf("StatePath() = %q, want env value", got)
	}
}

func TestSearch(t *testing.T) {
	c := open(t, statePath(t))
	names := func(ps []odoo.ProjectInfo) []string {
		var out []string
		for _, p := range ps {
			out = append(out, p.Name)
		}
		return out
	}

	tests := []struct {
		text string
		want int
	}{
		{"alpha", 1},        // name, case-insensitive
		{"customer b", 1},   // customer
		{"demo project", 2}, // substring
		{"", 3},             // empty: all
		{"no such project", 0},
	}
	for _, tt := range tests {
		got, err := c.SearchProjects(tt.text)
		if err != nil || len(got) != tt.want {
			t.Errorf("SearchProjects(%q) = %v, %v; want %d", tt.text, names(got), err, tt.want)
		}
	}

	tasks, err := c.SearchTasks(0, "IMPLEMENT")
	if err != nil || len(tasks) != 1 || tasks[0].ID != 12 {
		t.Errorf("SearchTasks(0, IMPLEMENT) = %v, %v", tasks, err)
	}
	tasks, _ = c.SearchTasks(2, "alpha")
	if len(tasks) != 0 {
		t.Errorf("SearchTasks(2, alpha) = %v, want none (other project)", tasks)
	}
}
