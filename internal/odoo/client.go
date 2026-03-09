package odoo

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
	Project         string
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
	// UpdateTimesheet updates an existing timesheet entry.
	UpdateTimesheet(id int64, params TimesheetWriteParams) error
	// DeleteTimesheet deletes a timesheet entry by ID.
	DeleteTimesheet(id int64) error
}
