package output_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/output"
	"github.com/seletz/odoo-work-cli/internal/output/outputtest"
)

func encode(t *testing.T, v any) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := output.WriteObject(&buf, output.FormatJSON, v); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

// fields decodes a JSON object into a generic map for value assertions.
func fields(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return m
}

var (
	checkIn  = time.Date(2026, 7, 16, 6, 0, 0, 0, time.UTC)
	checkOut = time.Date(2026, 7, 16, 10, 30, 0, 0, time.UTC)
	now      = time.Date(2026, 7, 16, 12, 15, 0, 0, time.UTC)
)

func TestSchemas_FullAndSparseValues(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		value  any
	}{
		{name: "project full", schema: output.SchemaProject, value: output.NewProject(odoo.ProjectInfo{
			ID: 1, Name: "Internal", Active: true, Customer: "ACME", Company: "Co", Stage: "Run",
			ProjectManager: "Pat", ExtraFields: map[string]string{"x_po": "Lee"},
		})},
		{name: "project sparse", schema: output.SchemaProject, value: output.NewProject(odoo.ProjectInfo{ID: 2, Name: "Bare"})},
		{name: "task full", schema: output.SchemaTask, value: output.NewTask(odoo.TaskInfo{
			ID: 5, Name: "Review", ProjectID: 1, Project: "Internal", Company: "Co", Stage: "Todo", Active: true,
		})},
		{name: "task sparse", schema: output.SchemaTask, value: output.NewTask(odoo.TaskInfo{ID: 6, Name: "Orphan"})},
		{name: "entry full", schema: output.SchemaEntry, value: output.NewEntry(odoo.TimesheetEntry{
			ID: 9, Date: "2026-07-16", ProjectID: 1, Project: "Internal", TaskID: 5, Task: "Review",
			Name: "code review", Hours: 1.25, Employee: "Pat", ValidatedStatus: "draft", Company: "Co",
		})},
		{name: "entry sparse", schema: output.SchemaEntry, value: output.NewEntry(odoo.TimesheetEntry{ID: 10, Date: "2026-07-16"})},
		{name: "user full", schema: output.SchemaUser, value: output.NewUser(odoo.UserInfo{ID: 2, Name: "Pat", Login: "pat", Email: "p@x", Company: "Co"})},
		{name: "user sparse", schema: output.SchemaUser, value: output.NewUser(odoo.UserInfo{ID: 2, Name: "Pat", Login: "pat"})},
		{name: "attendance closed", schema: output.SchemaAttendance, value: output.NewAttendance(odoo.AttendanceRecord{
			ID: 17, EmployeeID: 7, Employee: "Pat", CheckIn: checkIn, CheckOut: &checkOut, WorkedHours: 4.5,
		})},
		{name: "attendance running", schema: output.SchemaAttendance, value: output.NewAttendance(odoo.AttendanceRecord{ID: 18, CheckIn: checkIn})},
		{name: "status clocked in", schema: output.SchemaClockStatus, value: output.NewClockStatus(&odoo.AttendanceStatus{
			ClockedIn: true, CurrentID: 18, CheckIn: &checkIn, TotalHours: 4.5,
			Periods: []odoo.AttendanceRecord{{ID: 17, CheckIn: checkIn, CheckOut: &checkOut, WorkedHours: 4.5}, {ID: 18, CheckIn: checkIn}},
		}, now)},
		{name: "status clocked out, no periods", schema: output.SchemaClockStatus, value: output.NewClockStatus(&odoo.AttendanceStatus{}, now)},
		{name: "error envelope", schema: output.SchemaError, value: output.ErrorEnvelope{Error: output.ErrorBody{Kind: apperr.KindAuth, Message: "x"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputtest.AssertObject(t, tt.schema, encode(t, tt.value))
		})
	}
}

func TestNewEntry_Values(t *testing.T) {
	got := fields(t, encode(t, output.NewEntry(odoo.TimesheetEntry{
		ID: 9, Date: "2026-07-16", ProjectID: 1, Project: "Internal", Name: "work", Hours: 1.25,
	})))
	want := map[string]any{
		"id": 9.0, "project_id": 1.0, "project": "Internal", "task_id": nil, "task": nil,
		"description": "work", "hours": 1.25, "minutes": 75.0, "status": nil,
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %#v, want %#v", k, got[k], v)
		}
	}
}

func TestNewEntry_MinutesRounded(t *testing.T) {
	tests := []struct {
		hours float64
		want  int64
	}{
		{hours: 0, want: 0},
		{hours: 0.333333, want: 20},
		{hours: 7.999, want: 480},
		{hours: 2.5, want: 150},
	}
	for _, tt := range tests {
		if got := output.NewEntry(odoo.TimesheetEntry{Hours: tt.hours}).Minutes; got != tt.want {
			t.Errorf("minutes(%v) = %d, want %d", tt.hours, got, tt.want)
		}
	}
}

func TestNewAttendance_Values(t *testing.T) {
	closed := fields(t, encode(t, output.NewAttendance(odoo.AttendanceRecord{
		ID: 17, EmployeeID: 7, CheckIn: checkIn, CheckOut: &checkOut, WorkedHours: 4.5,
	})))
	if closed["running"] != false || closed["worked_minutes"] != 270.0 {
		t.Errorf("closed record: %v", closed)
	}
	in, err := time.Parse(time.RFC3339, closed["check_in"].(string))
	if err != nil || !in.Equal(checkIn) {
		t.Errorf("check_in = %v (%v), want instant %v", closed["check_in"], err, checkIn)
	}

	running := fields(t, encode(t, output.NewAttendance(odoo.AttendanceRecord{ID: 18, CheckIn: checkIn})))
	if running["running"] != true || running["check_out"] != nil {
		t.Errorf("running record: %v", running)
	}
}

func TestNewClockStatus_Values(t *testing.T) {
	in := fields(t, encode(t, output.NewClockStatus(&odoo.AttendanceStatus{
		ClockedIn: true, CurrentID: 18, CheckIn: &checkIn, TotalHours: 1.5,
	}, now)))
	if in["clocked_in"] != true || in["current_id"] != 18.0 || in["elapsed_minutes"] != 375.0 || in["total_minutes"] != 90.0 {
		t.Errorf("clocked in status: %v", in)
	}
	periods, ok := in["periods"].([]any)
	if !ok || len(periods) != 0 {
		t.Errorf("periods = %#v, want empty array", in["periods"])
	}

	out := fields(t, encode(t, output.NewClockStatus(&odoo.AttendanceStatus{}, now)))
	if out["clocked_in"] != false || out["current_id"] != nil || out["check_in"] != nil || out["elapsed_minutes"] != nil {
		t.Errorf("clocked out status: %v", out)
	}
}

func TestListConverters_NeverNil(t *testing.T) {
	if output.NewEntries(nil) == nil || output.NewProjects(nil) == nil || output.NewTasks(nil) == nil {
		t.Error("list converters must return empty, non-nil slices")
	}
	var buf bytes.Buffer
	if err := output.WriteList(&buf, output.FormatJSON, output.NewEntries(nil)); err != nil {
		t.Fatal(err)
	}
	if n := outputtest.AssertList(t, output.SchemaEntry, buf.Bytes()); n != 0 {
		t.Errorf("got %d items, want 0", n)
	}
}

func TestErrorEnvelope_MatchesSchema(t *testing.T) {
	var buf bytes.Buffer
	output.WriteError(&buf, output.FormatJSON, errors.New("boom"))
	outputtest.AssertObject(t, output.SchemaError, buf.Bytes())
}
