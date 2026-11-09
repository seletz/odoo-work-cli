// Package odootest provides an in-memory odoo.Client for command tests.
package odootest

import (
	"time"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// Fake is a configurable odoo.Client. Each method returns the matching
// field; Err, when set, is returned by every method instead. Write calls
// are recorded.
type Fake struct {
	// Err is returned by every method when non-nil.
	Err error

	User        odoo.UserInfo
	Projects    []odoo.ProjectInfo
	Tasks       []odoo.TaskInfo
	Timesheets  []odoo.TimesheetEntry
	Attendances []odoo.AttendanceRecord
	Status      odoo.AttendanceStatus

	// NextID is returned by create calls.
	NextID int64

	Created  []odoo.TimesheetWriteParams
	Updated  map[int64]map[string]interface{}
	Deleted  []int64
	Closed   bool
	EditedIn *time.Time
}

var _ odoo.Client = (*Fake)(nil)

// Close implements odoo.Client.
func (f *Fake) Close() { f.Closed = true }

// WhoAmI implements odoo.Client.
func (f *Fake) WhoAmI() (*odoo.UserInfo, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	u := f.User
	return &u, nil
}

// ListProjects implements odoo.Client.
func (f *Fake) ListProjects() ([]odoo.ProjectInfo, error) { return f.Projects, f.Err }

// ListAllProjects implements odoo.Client.
func (f *Fake) ListAllProjects() ([]odoo.ProjectInfo, error) { return f.Projects, f.Err }

// ListTasks implements odoo.Client.
func (f *Fake) ListTasks(int64) ([]odoo.TaskInfo, error) { return f.Tasks, f.Err }

// ListAllTasks implements odoo.Client.
func (f *Fake) ListAllTasks(int64) ([]odoo.TaskInfo, error) { return f.Tasks, f.Err }

// ListTimesheets implements odoo.Client.
func (f *Fake) ListTimesheets(string, string) ([]odoo.TimesheetEntry, error) {
	return f.Timesheets, f.Err
}

// ListAllTimesheets implements odoo.Client.
func (f *Fake) ListAllTimesheets(string, string) ([]odoo.TimesheetEntry, error) {
	return f.Timesheets, f.Err
}

// GetTimesheet implements odoo.Client.
func (f *Fake) GetTimesheet(id int64) (*odoo.TimesheetEntry, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	for i := range f.Timesheets {
		if f.Timesheets[i].ID == id {
			e := f.Timesheets[i]
			return &e, nil
		}
	}
	return nil, apperr.New(apperr.KindNotFound, "timesheet entry %d not found", id)
}

// GetFields implements odoo.Client.
func (f *Fake) GetFields(string) ([]odoo.FieldInfo, error) { return nil, f.Err }

// CreateTimesheet implements odoo.Client. The created entry is added to
// Timesheets so that a subsequent GetTimesheet finds it.
func (f *Fake) CreateTimesheet(p odoo.TimesheetWriteParams) (int64, error) {
	if f.Err != nil {
		return 0, f.Err
	}
	f.Created = append(f.Created, p)
	f.Timesheets = append(f.Timesheets, odoo.TimesheetEntry{
		ID: f.NextID, Date: p.Date, ProjectID: p.ProjectID, TaskID: p.TaskID, Name: p.Name, Hours: p.Hours,
	})
	return f.NextID, nil
}

// UpdateTimesheet implements odoo.Client and applies name, unit_amount, project_id, task_id and date.
func (f *Fake) UpdateTimesheet(id int64, fields map[string]interface{}) error {
	if f.Err != nil {
		return f.Err
	}
	if f.Updated == nil {
		f.Updated = map[int64]map[string]interface{}{}
	}
	f.Updated[id] = fields
	for i := range f.Timesheets {
		if f.Timesheets[i].ID != id {
			continue
		}
		if v, ok := fields["name"].(string); ok {
			f.Timesheets[i].Name = v
		}
		if v, ok := fields["unit_amount"].(float64); ok {
			f.Timesheets[i].Hours = v
		}
		if v, ok := fields["project_id"].(int64); ok {
			f.Timesheets[i].ProjectID = v
		}
		if v, ok := fields["task_id"].(int64); ok {
			f.Timesheets[i].TaskID = v
		}
		if v, ok := fields["date"].(string); ok {
			f.Timesheets[i].Date = v
		}
		return nil
	}
	return apperr.New(apperr.KindNotFound, "timesheet entry %d not found", id)
}

// DeleteTimesheet implements odoo.Client.
func (f *Fake) DeleteTimesheet(id int64) error {
	if f.Err != nil {
		return f.Err
	}
	f.Deleted = append(f.Deleted, id)
	return nil
}

// ClockIn implements odoo.Client and records an open attendance.
func (f *Fake) ClockIn() (int64, error) { return f.ClockInAt(time.Now()) }

// ClockOut implements odoo.Client and returns the first attendance.
func (f *Fake) ClockOut() (*odoo.AttendanceRecord, error) { return f.firstAttendance() }

// ClockInAt implements odoo.Client and records an open attendance.
func (f *Fake) ClockInAt(in time.Time) (int64, error) {
	if f.Err != nil {
		return 0, f.Err
	}
	f.Attendances = append(f.Attendances, odoo.AttendanceRecord{ID: f.NextID, CheckIn: in})
	return f.NextID, nil
}

// ClockOutAt implements odoo.Client and returns the first attendance.
func (f *Fake) ClockOutAt(time.Time) (*odoo.AttendanceRecord, error) { return f.firstAttendance() }

// CreateAttendance implements odoo.Client and records a closed attendance.
func (f *Fake) CreateAttendance(in, out time.Time) (int64, error) {
	if f.Err != nil {
		return 0, f.Err
	}
	hours := out.Sub(in).Hours()
	f.Attendances = append(f.Attendances, odoo.AttendanceRecord{ID: f.NextID, CheckIn: in, CheckOut: &out, WorkedHours: hours})
	return f.NextID, nil
}

// EditAttendance implements odoo.Client: it applies the new times to the
// matching record and returns the updated record.
func (f *Fake) EditAttendance(id int64, in, out *time.Time) (*odoo.AttendanceRecord, error) {
	f.EditedIn = in
	if f.Err != nil {
		return nil, f.Err
	}
	for i := range f.Attendances {
		r := &f.Attendances[i]
		if r.ID != id {
			continue
		}
		if in != nil {
			r.CheckIn = *in
		}
		if out != nil {
			o := *out
			r.CheckOut = &o
		}
		if r.CheckOut != nil {
			r.WorkedHours = r.CheckOut.Sub(r.CheckIn).Hours()
		}
		c := *r
		return &c, nil
	}
	return nil, apperr.New(apperr.KindNotFound, "attendance record %d not found", id)
}

// GetAttendance implements odoo.Client.
func (f *Fake) GetAttendance(id int64) (*odoo.AttendanceRecord, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	for i := range f.Attendances {
		if f.Attendances[i].ID == id {
			r := f.Attendances[i]
			return &r, nil
		}
	}
	return nil, apperr.New(apperr.KindNotFound, "attendance record %d not found", id)
}

// AttendanceStatus implements odoo.Client.
func (f *Fake) AttendanceStatus() (*odoo.AttendanceStatus, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	s := f.Status
	return &s, nil
}

// ListAttendance implements odoo.Client: records with check-in in [from, to).
func (f *Fake) ListAttendance(from, to time.Time) ([]odoo.AttendanceRecord, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	var out []odoo.AttendanceRecord
	for _, r := range f.Attendances {
		if !r.CheckIn.Before(from) && r.CheckIn.Before(to) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *Fake) firstAttendance() (*odoo.AttendanceRecord, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	if len(f.Attendances) == 0 {
		return nil, apperr.New(apperr.KindConflict, "not clocked in")
	}
	r := f.Attendances[0]
	return &r, nil
}
