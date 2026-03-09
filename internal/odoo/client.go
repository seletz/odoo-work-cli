package odoo

import "time"

// UserInfo holds identity information for the current Odoo user.
type UserInfo struct {
	ID       int64
	Name     string
	Login    string
	Email    string
	Company  string
}

// ProjectInfo holds summary information for an Odoo project.
type ProjectInfo struct {
	ID             int64
	Name           string
	Active         bool
	Customer       string
	Company        string
	Stage          string
	ProjectManager string
	ExtraFields    map[string]string
}

// TaskInfo holds summary information for an Odoo task.
type TaskInfo struct {
	ID      int64
	Name    string
	Project string
	Stage   string
	Active  bool
}

// TimesheetEntry holds a single timesheet line from Odoo.
type TimesheetEntry struct {
	ID              int64
	Date            string
	ProjectID       int64
	Project         string
	TaskID          int64
	Task            string
	Name            string
	Hours           float64
	Employee        string
	ValidatedStatus string
}

// TimesheetWriteParams holds parameters for creating or updating a timesheet entry.
type TimesheetWriteParams struct {
	// ProjectID is the Odoo ID of the project (required for create).
	ProjectID int64
	// TaskID is the Odoo ID of the task (optional, 0 means no task).
	TaskID int64
	// Date is the entry date in "YYYY-MM-DD" format (required for create).
	Date string
	// Name is the description of work done (required for create).
	Name string
	// Hours is the number of hours logged (required for create, must be > 0).
	Hours float64
}

// FieldInfo holds metadata about a single model field.
type FieldInfo struct {
	Name     string
	Type     string
	String   string
	Required bool
}

// AttendanceRecord holds a single attendance period from Odoo.
type AttendanceRecord struct {
	ID          int64
	EmployeeID  int64
	Employee    string
	CheckIn     time.Time
	CheckOut    *time.Time // nil if still clocked in
	WorkedHours float64
}

// AttendanceStatus holds the current clock-in/out state and today's periods.
type AttendanceStatus struct {
	ClockedIn  bool
	CurrentID  int64              // open record ID (0 if not clocked in)
	CheckIn    *time.Time         // current period start
	Periods    []AttendanceRecord // all today's records
	TotalHours float64
}

// Client defines the interface for interacting with an Odoo instance.
type Client interface {
	// WhoAmI returns the identity of the currently authenticated user.
	WhoAmI() (*UserInfo, error)
	// ListProjects returns all projects from Odoo.
	ListProjects() ([]ProjectInfo, error)
	// ListTasks returns tasks, optionally filtered by project ID.
	ListTasks(projectID int64) ([]TaskInfo, error)
	// ListTimesheets returns timesheet entries for the given date range.
	ListTimesheets(dateFrom, dateTo string) ([]TimesheetEntry, error)
	// GetFields returns field metadata for the given Odoo model.
	GetFields(model string) ([]FieldInfo, error)
	// CreateTimesheet creates a new timesheet entry and returns its ID.
	CreateTimesheet(params TimesheetWriteParams) (int64, error)
	// UpdateTimesheet partially updates an existing timesheet entry.
	// Only the fields present in the map are updated.
	UpdateTimesheet(id int64, fields map[string]interface{}) error
	// DeleteTimesheet deletes a timesheet entry by ID.
	DeleteTimesheet(id int64) error
	// ClockIn creates an attendance record with check_in = now.
	ClockIn() (int64, error)
	// ClockOut writes check_out = now on the open attendance record.
	ClockOut() (*AttendanceRecord, error)
	// AttendanceStatus returns the current clock state and today's periods.
	AttendanceStatus() (*AttendanceStatus, error)
}
