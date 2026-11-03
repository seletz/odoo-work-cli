package output

import (
	"math"
	"time"

	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// The types in this file are the stable JSON shapes of the CLI. They are
// decoupled from the odoo package so internal refactorings cannot change
// the output. Every field is always present; optional relations are null
// instead of "" or 0. Field names and types are pinned by the schemas in
// schemas/ — change both together, and only additively.

// Project is a project as printed by projects.
type Project struct {
	ID             int64             `json:"id"`
	Name           string            `json:"name"`
	Active         bool              `json:"active"`
	Customer       *string           `json:"customer"`
	Company        *string           `json:"company"`
	Stage          *string           `json:"stage"`
	ProjectManager *string           `json:"project_manager"`
	ExtraFields    map[string]string `json:"extra_fields"`
}

// Task is a task as printed by tasks.
type Task struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	ProjectID *int64  `json:"project_id"`
	Project   *string `json:"project"`
	Company   *string `json:"company"`
	Stage     *string `json:"stage"`
	Active    bool    `json:"active"`
}

// Entry is a timesheet line as printed by entries, timesheets and
// the entries add/update/delete write commands.
type Entry struct {
	ID          int64   `json:"id"`
	Date        string  `json:"date"`
	ProjectID   *int64  `json:"project_id"`
	Project     *string `json:"project"`
	TaskID      *int64  `json:"task_id"`
	Task        *string `json:"task"`
	Description string  `json:"description"`
	Hours       float64 `json:"hours"`
	Minutes     int64   `json:"minutes"`
	Employee    *string `json:"employee"`
	Status      *string `json:"status"`
	Company     *string `json:"company"`
}

// User is the current user as printed by whoami.
type User struct {
	ID      int64   `json:"id"`
	Name    string  `json:"name"`
	Login   string  `json:"login"`
	Email   *string `json:"email"`
	Company *string `json:"company"`
}

// Attendance is an attendance record as printed by clock in/out/edit
// and in the periods of clock status. Times are RFC 3339 with the
// local UTC offset.
type Attendance struct {
	ID            int64   `json:"id"`
	EmployeeID    *int64  `json:"employee_id"`
	Employee      *string `json:"employee"`
	CheckIn       string  `json:"check_in"`
	CheckOut      *string `json:"check_out"`
	Running       bool    `json:"running"`
	WorkedHours   float64 `json:"worked_hours"`
	WorkedMinutes int64   `json:"worked_minutes"`
}

// ClockStatus is the attendance state as printed by clock status.
type ClockStatus struct {
	ClockedIn      bool         `json:"clocked_in"`
	CurrentID      *int64       `json:"current_id"`
	CheckIn        *string      `json:"check_in"`
	ElapsedMinutes *int64       `json:"elapsed_minutes"`
	Periods        []Attendance `json:"periods"`
	TotalHours     float64      `json:"total_hours"`
	TotalMinutes   int64        `json:"total_minutes"`
}

// NewProject converts an odoo.ProjectInfo.
func NewProject(p odoo.ProjectInfo) Project {
	extra := map[string]string{}
	for k, v := range p.ExtraFields {
		extra[k] = v
	}
	return Project{
		ID:             p.ID,
		Name:           p.Name,
		Active:         p.Active,
		Customer:       optString(p.Customer),
		Company:        optString(p.Company),
		Stage:          optString(p.Stage),
		ProjectManager: optString(p.ProjectManager),
		ExtraFields:    extra,
	}
}

// NewProjects converts a project list; never returns nil.
func NewProjects(ps []odoo.ProjectInfo) []Project {
	out := make([]Project, 0, len(ps))
	for _, p := range ps {
		out = append(out, NewProject(p))
	}
	return out
}

// NewTask converts an odoo.TaskInfo.
func NewTask(t odoo.TaskInfo) Task {
	return Task{
		ID:        t.ID,
		Name:      t.Name,
		ProjectID: optID(t.ProjectID),
		Project:   optString(t.Project),
		Company:   optString(t.Company),
		Stage:     optString(t.Stage),
		Active:    t.Active,
	}
}

// NewTasks converts a task list; never returns nil.
func NewTasks(ts []odoo.TaskInfo) []Task {
	out := make([]Task, 0, len(ts))
	for _, t := range ts {
		out = append(out, NewTask(t))
	}
	return out
}

// NewEntry converts an odoo.TimesheetEntry.
func NewEntry(e odoo.TimesheetEntry) Entry {
	return Entry{
		ID:          e.ID,
		Date:        e.Date,
		ProjectID:   optID(e.ProjectID),
		Project:     optString(e.Project),
		TaskID:      optID(e.TaskID),
		Task:        optString(e.Task),
		Description: e.Name,
		Hours:       e.Hours,
		Minutes:     minutes(e.Hours),
		Employee:    optString(e.Employee),
		Status:      optString(e.ValidatedStatus),
		Company:     optString(e.Company),
	}
}

// NewEntries converts a timesheet list; never returns nil.
func NewEntries(es []odoo.TimesheetEntry) []Entry {
	out := make([]Entry, 0, len(es))
	for _, e := range es {
		out = append(out, NewEntry(e))
	}
	return out
}

// NewUser converts an odoo.UserInfo.
func NewUser(u odoo.UserInfo) User {
	return User{
		ID:      u.ID,
		Name:    u.Name,
		Login:   u.Login,
		Email:   optString(u.Email),
		Company: optString(u.Company),
	}
}

// NewAttendance converts an odoo.AttendanceRecord.
func NewAttendance(r odoo.AttendanceRecord) Attendance {
	a := Attendance{
		ID:            r.ID,
		EmployeeID:    optID(r.EmployeeID),
		Employee:      optString(r.Employee),
		CheckIn:       timestamp(r.CheckIn),
		Running:       r.CheckOut == nil,
		WorkedHours:   r.WorkedHours,
		WorkedMinutes: minutes(r.WorkedHours),
	}
	if r.CheckOut != nil {
		s := timestamp(*r.CheckOut)
		a.CheckOut = &s
	}
	return a
}

// NewClockStatus converts an odoo.AttendanceStatus. now is used for the
// elapsed time of a running period.
func NewClockStatus(s *odoo.AttendanceStatus, now time.Time) ClockStatus {
	cs := ClockStatus{
		ClockedIn:    s.ClockedIn,
		CurrentID:    optID(s.CurrentID),
		Periods:      make([]Attendance, 0, len(s.Periods)),
		TotalHours:   s.TotalHours,
		TotalMinutes: minutes(s.TotalHours),
	}
	if s.ClockedIn && s.CheckIn != nil {
		in := timestamp(*s.CheckIn)
		cs.CheckIn = &in
		elapsed := int64(math.Max(0, now.Sub(*s.CheckIn).Minutes()))
		cs.ElapsedMinutes = &elapsed
	}
	for _, p := range s.Periods {
		cs.Periods = append(cs.Periods, NewAttendance(p))
	}
	return cs
}

// timestamp formats t as RFC 3339 in the local zone.
func timestamp(t time.Time) string { return t.Local().Format(time.RFC3339) }

// minutes converts fractional hours to whole minutes, rounded.
func minutes(hours float64) int64 { return int64(math.Round(hours * 60)) }

func optString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func optID(id int64) *int64 {
	if id <= 0 {
		return nil
	}
	return &id
}
