package fake

import (
	"sort"
	"time"

	"github.com/seletz/odoo-work-cli/internal/odoo"
)

const (
	fakeCompany = "Example Co fake"
	fakeUser    = "Fake User"
)

// Seed returns a fresh fake database. All names are made up. Timesheet
// entries and closed attendance records lie on the three workdays before
// now (local time), so week views show data while today stays free.
//
// :param now: the reference time
// :returns: the seeded state
func Seed(now time.Time) *State {
	st := &State{
		NextID:     1000,
		EmployeeID: 7,
		User: odoo.UserInfo{
			ID: 2, Name: fakeUser, Login: "fake.user@example.invalid",
			Email: "fake.user@example.invalid", Company: fakeCompany,
		},
		Projects: []odoo.ProjectInfo{
			{ID: 1, Name: "Demo Project Alpha", Active: true, Customer: "Example Customer A", Company: fakeCompany, Stage: "In Progress", ProjectManager: fakeUser},
			{ID: 2, Name: "Demo Project Beta", Active: true, Customer: "Example Customer B", Company: fakeCompany, Stage: "In Progress", ProjectManager: fakeUser},
			{ID: 3, Name: "Internal (fake)", Active: true, Company: fakeCompany, Stage: "In Progress", ProjectManager: fakeUser},
		},
		Tasks: []odoo.TaskInfo{
			{ID: 11, Name: "Alpha: Design", Project: "Demo Project Alpha", ProjectID: 1, Company: fakeCompany, Stage: "In Progress", Active: true},
			{ID: 12, Name: "Alpha: Implementation", Project: "Demo Project Alpha", ProjectID: 1, Company: fakeCompany, Stage: "In Progress", Active: true},
			{ID: 21, Name: "Beta: Support", Project: "Demo Project Beta", ProjectID: 2, Company: fakeCompany, Stage: "In Progress", Active: true},
			{ID: 31, Name: "Meetings", Project: "Internal (fake)", ProjectID: 3, Company: fakeCompany, Stage: "In Progress", Active: true},
		},
	}

	type booking struct {
		project, task int64
		name          string
		hours         float64
	}
	bookings := [][]booking{
		{{1, 11, "Sketch screens", 3}, {3, 31, "Weekly sync", 1}, {2, 21, "Answer tickets", 3.5}},
		{{1, 12, "Implement login form", 6}, {3, 31, "Team meeting", 1.5}},
		{{2, 21, "Bug triage", 4}, {1, 12, "Code review", 3.5}},
	}
	c := &Client{st: st}
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for _, bs := range bookings {
		day = previousWorkday(day)
		for _, b := range bs {
			e := odoo.TimesheetEntry{
				ID: c.nextID(), Date: day.Format("2006-01-02"), Name: b.name, Hours: b.hours,
				Employee: fakeUser, ValidatedStatus: "draft",
			}
			_ = c.applyProjectTask(&e, b.project, b.task)
			st.Timesheets = append(st.Timesheets, e)
		}
		in := day.Add(8 * time.Hour)
		out := day.Add(17 * time.Hour)
		st.Attendances = append(st.Attendances, withHours(odoo.AttendanceRecord{
			ID: c.nextID(), EmployeeID: st.EmployeeID, Employee: fakeUser, CheckIn: in, CheckOut: &out,
		}))
	}
	return st
}

func previousWorkday(day time.Time) time.Time {
	for {
		day = day.AddDate(0, 0, -1)
		if wd := day.Weekday(); wd != time.Saturday && wd != time.Sunday {
			return day
		}
	}
}

// fakeFields is the static answer of GetFields.
var fakeFields = map[string][]odoo.FieldInfo{
	"account.analytic.line": {
		{Name: "date", Type: "date", String: "Date", Required: true},
		{Name: "name", Type: "char", String: "Description"},
		{Name: "project_id", Type: "many2one", String: "Project"},
		{Name: "task_id", Type: "many2one", String: "Task"},
		{Name: "unit_amount", Type: "float", String: "Quantity"},
		{Name: "validated_status", Type: "selection", String: "Validated Status", Required: true},
	},
	"hr.attendance": {
		{Name: "check_in", Type: "datetime", String: "Check In", Required: true},
		{Name: "check_out", Type: "datetime", String: "Check Out"},
		{Name: "employee_id", Type: "many2one", String: "Employee", Required: true},
		{Name: "worked_hours", Type: "float", String: "Worked Hours"},
	},
	"project.project": {
		{Name: "name", Type: "char", String: "Name", Required: true},
		{Name: "partner_id", Type: "many2one", String: "Customer"},
		{Name: "company_id", Type: "many2one", String: "Company"},
	},
	"project.task": {
		{Name: "name", Type: "char", String: "Title", Required: true},
		{Name: "project_id", Type: "many2one", String: "Project"},
		{Name: "stage_id", Type: "many2one", String: "Stage"},
	},
}

func fakeModels() []string {
	var out []string
	for m := range fakeFields {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}
