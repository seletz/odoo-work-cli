// Package docsgen generates the generated parts of the documentation site:
// the demo state of the fake backend, TUI screenshots (SVG plus plain-text
// goldens), CLI output samples and the command reference.
//
// Everything is rendered from a fixed demo state at a fixed clock
// (DemoNow, Europe/Berlin), so a rerun yields byte-identical files as long
// as the code does not change. That is what lets the check tasks fail when
// the committed files are stale.
package docsgen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
	// Embedded zone data: renders must not depend on the host's tzdata.
	_ "time/tzdata"

	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/odoo/fake"
)

// Names of the made-up demo world. Nothing here refers to a real person,
// customer or company; e-mail addresses use the reserved .invalid TLD.
const (
	DemoUser       = "Alex Demo"
	demoLogin      = "alex.demo@example.invalid"
	demoCompany    = "Example Consulting"
	demoLabCompany = "Wildflower Labs"
)

// Location is the time zone of every rendering.
var Location = mustLoad("Europe/Berlin")

// DemoNow is the clock of the docs: Thursday of ISO week 2026-W15, whose
// Monday is Easter Monday (a public holiday in Baden-Württemberg). The demo
// user clocked in at 08:41 and is still clocked in.
var DemoNow = time.Date(2026, 4, 9, 15, 12, 0, 0, Location)

// DemoWeek is DemoNow's ISO week in the --week flag format.
const DemoWeek = "2026-W15"

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// DemoMonday returns the Monday of DemoNow's week, at local midnight.
//
// :returns: the Monday
func DemoMonday() time.Time {
	d := time.Date(DemoNow.Year(), DemoNow.Month(), DemoNow.Day(), 0, 0, 0, 0, Location)
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

// DemoCompanyColors are the [company_colors] the screenshots use.
var DemoCompanyColors = map[string]string{demoCompany: "5", demoLabCompany: "2"}

// demoBooking is one timesheet entry of the demo state.
type demoBooking struct {
	day   string // YYYY-MM-DD
	task  int64
	name  string
	hours float64
}

// demoShift is one closed attendance record: day, check-in, check-out.
type demoShift struct{ day, in, out string }

var (
	demoProjects = []odoo.ProjectInfo{
		{ID: 1, Name: "Lighthouse Portal", Customer: "Example Harbor Ltd", Company: demoCompany},
		{ID: 2, Name: "Orchard Mobile App", Customer: "Example Orchard Inc", Company: demoCompany},
		{ID: 3, Name: "Internal", Company: demoCompany},
		{ID: 4, Name: "Riverbank Data Migration", Customer: "Example River Co", Company: demoLabCompany},
	}
	demoTasks = []odoo.TaskInfo{
		{ID: 11, Name: "Backend API", ProjectID: 1},
		{ID: 12, Name: "Frontend", ProjectID: 1},
		{ID: 13, Name: "Code review", ProjectID: 1},
		{ID: 21, Name: "Release 2.4", ProjectID: 2},
		{ID: 22, Name: "Support", ProjectID: 2},
		{ID: 31, Name: "Meetings", ProjectID: 3},
		{ID: 32, Name: "Training", ProjectID: 3},
		{ID: 41, Name: "Data mapping", ProjectID: 4},
	}
	// Previous week (W14, Friday is Good Friday) and the demo week up to
	// today. Easter Monday has no bookings.
	demoBookings = []demoBooking{
		{"2026-03-30", 11, "Pagination for the search endpoint", 4},
		{"2026-03-30", 31, "Sprint planning", 1.5},
		{"2026-03-30", 22, "Ticket triage", 2.5},
		{"2026-03-31", 12, "Filter sidebar", 5},
		{"2026-03-31", 41, "Map legacy customer fields", 3},
		{"2026-04-01", 21, "Release notes", 2},
		{"2026-04-01", 13, "Review search endpoint", 1.5},
		{"2026-04-01", 41, "Map legacy order fields", 4.5},
		{"2026-04-02", 11, "Rate limiting", 6},
		{"2026-04-02", 31, "Retro", 1},
		{"2026-04-07", 11, "Search: fuzzy matching", 3.5},
		{"2026-04-07", 31, "Weekly sync", 1},
		{"2026-04-07", 41, "Import dry run", 3.5},
		{"2026-04-08", 12, "Result list layout", 4},
		{"2026-04-08", 22, "Crash report on login", 2.5},
		{"2026-04-08", 32, "Accessibility workshop", 1.5},
		{"2026-04-09", 21, "Store screenshots", 2},
		{"2026-04-09", 13, "Review fuzzy matching", 1.5},
		{"2026-04-09", 31, "Customer call", 0.75},
	}
	demoShifts = []demoShift{
		{"2026-03-30", "08:30", "16:45"},
		{"2026-03-31", "08:15", "16:30"},
		{"2026-04-01", "08:20", "16:20"},
		{"2026-04-02", "08:00", "15:00"},
		{"2026-04-07", "08:10", "16:40"},
		{"2026-04-08", "08:30", "12:00"},
		{"2026-04-08", "12:30", "17:00"},
	}
	demoClockIn = "2026-04-09 08:41"
)

// DemoState returns the demo database of the fake backend. It is built
// from the tables above, so it is the same on every call.
//
// :returns: the state
func DemoState() *fake.State {
	st := &fake.State{
		NextID:     5000,
		EmployeeID: 7,
		User: odoo.UserInfo{
			ID: 2, Name: DemoUser, Login: demoLogin, Email: demoLogin, Company: demoCompany,
		},
	}
	projects := map[int64]odoo.ProjectInfo{}
	for _, p := range demoProjects {
		p.Active, p.Stage, p.ProjectManager = true, "In Progress", DemoUser
		st.Projects = append(st.Projects, p)
		projects[p.ID] = p
	}
	tasks := map[int64]odoo.TaskInfo{}
	for _, t := range demoTasks {
		p := projects[t.ProjectID]
		t.Project, t.Company, t.Stage, t.Active = p.Name, p.Company, "In Progress", true
		st.Tasks = append(st.Tasks, t)
		tasks[t.ID] = t
	}
	id := int64(4000)
	for _, b := range demoBookings {
		t := tasks[b.task]
		p := projects[t.ProjectID]
		id++
		st.Timesheets = append(st.Timesheets, odoo.TimesheetEntry{
			ID: id, Date: b.day, ProjectID: p.ID, Project: p.Name, TaskID: t.ID, Task: t.Name,
			Name: b.name, Hours: b.hours, Employee: DemoUser, ValidatedStatus: status(b.day), Company: p.Company,
		})
	}
	id = 4600
	for _, s := range demoShifts {
		in, out := at(s.day+" "+s.in), at(s.day+" "+s.out)
		id++
		st.Attendances = append(st.Attendances, odoo.AttendanceRecord{
			ID: id, EmployeeID: st.EmployeeID, Employee: DemoUser,
			CheckIn: in, CheckOut: &out, WorkedHours: out.Sub(in).Hours(),
		})
	}
	st.Attendances = append(st.Attendances, odoo.AttendanceRecord{
		ID: id + 1, EmployeeID: st.EmployeeID, Employee: DemoUser, CheckIn: at(demoClockIn),
	})
	return st
}

// status marks last week's entries as validated, the demo week's as draft.
func status(day string) string {
	if day < DemoMonday().Format("2006-01-02") {
		return "validated"
	}
	return "draft"
}

// at parses "YYYY-MM-DD HH:MM" in Location.
func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, Location)
	if err != nil {
		panic(err)
	}
	return t
}

// WriteDemoState writes DemoState as the JSON state file of the fake
// backend (ODOO_WORK_CLI_FAKE_STATE).
//
// :param path: the file to write
// :returns: an error if writing fails
func WriteDemoState(path string) error {
	data, err := json.MarshalIndent(DemoState(), "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("demo state: %w", err)
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
