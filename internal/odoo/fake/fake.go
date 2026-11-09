// Package fake provides an offline odoo.Client backed by a JSON state file.
//
// It is selected with ODOO_WORK_CLI_BACKEND=fake (or --backend fake) and
// lets agents and developers try every command without network, config or
// 1Password. Its data is made up and has nothing to do with any real Odoo
// database. Writes persist in the state file, so list → add → list works
// across separate invocations; deleting the file resets the data.
//
// The client mirrors the behaviour of odoo.XMLRPCClient that commands rely
// on: unique IDs, not_found for unknown IDs on reads, conflict for "already
// clocked in" / "not clocked in", and validation for what Odoo itself
// rejects (unknown project or task, overlapping attendances, check-out not
// after check-in, writes to a missing record). It is modelled on
// odootest.Fake but keeps state semantics instead of recording calls.
// Differences: worked_hours is the plain check-out − check-in span (Odoo
// subtracts the lunch break), and configured filters are ignored.
package fake

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

const (
	// BackendEnvVar selects the backend ("fake" or "odoo").
	BackendEnvVar = "ODOO_WORK_CLI_BACKEND"
	// StateEnvVar overrides the path of the fake state file.
	StateEnvVar = "ODOO_WORK_CLI_FAKE_STATE"
	// DefaultStateFile is the state file name under os.TempDir().
	DefaultStateFile = "odoo-work-cli-fake-state.json"
)

// StatePath returns the fake state file: ODOO_WORK_CLI_FAKE_STATE if set,
// else DefaultStateFile in os.TempDir().
//
// :returns: the state file path
func StatePath() string {
	if p := os.Getenv(StateEnvVar); p != "" {
		return p
	}
	return filepath.Join(os.TempDir(), DefaultStateFile)
}

// State is the persisted fake database.
type State struct {
	NextID      int64                   `json:"next_id"`
	User        odoo.UserInfo           `json:"user"`
	EmployeeID  int64                   `json:"employee_id"`
	Projects    []odoo.ProjectInfo      `json:"projects"`
	Tasks       []odoo.TaskInfo         `json:"tasks"`
	Timesheets  []odoo.TimesheetEntry   `json:"timesheets"`
	Attendances []odoo.AttendanceRecord `json:"attendances"`
}

// Client is the file-backed fake odoo.Client.
type Client struct {
	path string
	now  func() time.Time
	st   *State
}

var _ odoo.Client = (*Client)(nil)

// Open loads the state file at path, or seeds and writes a new one if it
// does not exist. A broken state file is a config error naming the file.
//
// :param path: the state file
// :param now: clock used for seeding and clock in/out
// :returns: the client
func Open(path string, now func() time.Time) (*Client, error) {
	c := &Client{path: path, now: now}
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		c.st = Seed(now())
		if err := c.save(); err != nil {
			return nil, err
		}
		return c, nil
	case err != nil:
		return nil, apperr.Wrapf(apperr.KindConfig, err, "reading fake state %s", path)
	}
	st := &State{}
	if err := json.Unmarshal(data, st); err != nil {
		return nil, apperr.Wrapf(apperr.KindConfig, err,
			"fake state %s is broken: delete it to reset", path)
	}
	for i := range st.Attendances {
		st.Attendances[i].CheckIn = st.Attendances[i].CheckIn.Local()
		if o := st.Attendances[i].CheckOut; o != nil {
			l := o.Local()
			st.Attendances[i].CheckOut = &l
		}
	}
	c.st = st
	return c, nil
}

// save writes the state atomically (temp file + rename).
func (c *Client) save() error {
	data, err := json.MarshalIndent(c.st, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(c.path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return apperr.Wrapf(apperr.KindConfig, err, "creating fake state dir %s", dir)
		}
	}
	tmp, err := os.CreateTemp(filepath.Dir(c.path), ".odoo-work-cli-fake-*")
	if err != nil {
		return apperr.Wrapf(apperr.KindConfig, err, "writing fake state %s", c.path)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return apperr.Wrapf(apperr.KindConfig, err, "writing fake state %s", c.path)
	}
	if err := tmp.Close(); err != nil {
		return apperr.Wrapf(apperr.KindConfig, err, "writing fake state %s", c.path)
	}
	if err := os.Rename(tmp.Name(), c.path); err != nil {
		return apperr.Wrapf(apperr.KindConfig, err, "writing fake state %s", c.path)
	}
	return nil
}

func (c *Client) nextID() int64 {
	id := c.st.NextID
	c.st.NextID++
	return id
}

// missing mirrors Odoo's MissingError on write/unlink (Fault 2 →
// validation).
func missing(model string, id int64) error {
	return apperr.New(apperr.KindValidation,
		"Record does not exist or has been deleted. (Record: %s(%d,))", model, id)
}

// Close implements odoo.Client.
func (c *Client) Close() {}

// WhoAmI implements odoo.Client.
func (c *Client) WhoAmI() (*odoo.UserInfo, error) {
	u := c.st.User
	return &u, nil
}

// ListProjects implements odoo.Client. Configured filters are ignored.
func (c *Client) ListProjects() ([]odoo.ProjectInfo, error) { return c.ListAllProjects() }

// ListAllProjects implements odoo.Client.
func (c *Client) ListAllProjects() ([]odoo.ProjectInfo, error) {
	return append([]odoo.ProjectInfo(nil), c.st.Projects...), nil
}

// ListTasks implements odoo.Client. Configured filters are ignored.
func (c *Client) ListTasks(projectID int64) ([]odoo.TaskInfo, error) {
	return c.ListAllTasks(projectID)
}

// ListAllTasks implements odoo.Client; projectID 0 means all projects.
func (c *Client) ListAllTasks(projectID int64) ([]odoo.TaskInfo, error) {
	var out []odoo.TaskInfo
	for _, t := range c.st.Tasks {
		if projectID == 0 || t.ProjectID == projectID {
			out = append(out, t)
		}
	}
	return out, nil
}

// ListTimesheets implements odoo.Client. Configured filters are ignored.
func (c *Client) ListTimesheets(dateFrom, dateTo string) ([]odoo.TimesheetEntry, error) {
	return c.ListAllTimesheets(dateFrom, dateTo)
}

// ListAllTimesheets implements odoo.Client: entries with dateFrom <= date
// <= dateTo, newest first (Odoo's "date desc, id desc").
func (c *Client) ListAllTimesheets(dateFrom, dateTo string) ([]odoo.TimesheetEntry, error) {
	var out []odoo.TimesheetEntry
	for _, e := range c.st.Timesheets {
		if (dateFrom == "" || e.Date >= dateFrom) && (dateTo == "" || e.Date <= dateTo) {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date > out[j].Date
		}
		return out[i].ID > out[j].ID
	})
	return out, nil
}

// GetTimesheet implements odoo.Client.
func (c *Client) GetTimesheet(id int64) (*odoo.TimesheetEntry, error) {
	if i := c.timesheetIndex(id); i >= 0 {
		e := c.st.Timesheets[i]
		return &e, nil
	}
	return nil, apperr.New(apperr.KindNotFound, "timesheet entry %d not found", id)
}

func (c *Client) timesheetIndex(id int64) int {
	for i := range c.st.Timesheets {
		if c.st.Timesheets[i].ID == id {
			return i
		}
	}
	return -1
}

// GetFields implements odoo.Client with a short static field list.
func (c *Client) GetFields(model string) ([]odoo.FieldInfo, error) {
	f, ok := fakeFields[model]
	if !ok {
		return nil, apperr.New(apperr.KindValidation, "fake backend: unknown model %q (known: %s)",
			model, strings.Join(fakeModels(), ", "))
	}
	return append([]odoo.FieldInfo(nil), f...), nil
}

// CreateTimesheet implements odoo.Client.
func (c *Client) CreateTimesheet(p odoo.TimesheetWriteParams) (int64, error) {
	e := odoo.TimesheetEntry{
		Date: p.Date, Name: p.Name, Hours: p.Hours,
		Employee: c.st.User.Name, ValidatedStatus: "draft",
	}
	if err := c.applyProjectTask(&e, p.ProjectID, p.TaskID); err != nil {
		return 0, err
	}
	e.ID = c.nextID()
	c.st.Timesheets = append(c.st.Timesheets, e)
	return e.ID, c.save()
}

// applyProjectTask sets project and task on e, rejecting unknown IDs and a
// task of another project the way Odoo does (validation).
func (c *Client) applyProjectTask(e *odoo.TimesheetEntry, projectID, taskID int64) error {
	var proj *odoo.ProjectInfo
	for i := range c.st.Projects {
		if c.st.Projects[i].ID == projectID {
			proj = &c.st.Projects[i]
		}
	}
	if proj == nil {
		return missing("project.project", projectID)
	}
	e.ProjectID, e.Project, e.Company = proj.ID, proj.Name, proj.Company
	e.TaskID, e.Task = 0, ""
	if taskID == 0 {
		return nil
	}
	for _, t := range c.st.Tasks {
		if t.ID != taskID {
			continue
		}
		if t.ProjectID != projectID {
			return apperr.New(apperr.KindValidation,
				"task %d belongs to project %d, not %d", taskID, t.ProjectID, projectID)
		}
		e.TaskID, e.Task = t.ID, t.Name
		return nil
	}
	return missing("project.task", taskID)
}

// UpdateTimesheet implements odoo.Client for name, unit_amount, date,
// project_id and task_id.
func (c *Client) UpdateTimesheet(id int64, fields map[string]interface{}) error {
	i := c.timesheetIndex(id)
	if i < 0 {
		return missing("account.analytic.line", id)
	}
	e := c.st.Timesheets[i]
	if v, ok := fields["name"].(string); ok {
		e.Name = v
	}
	if v, ok := toFloat(fields["unit_amount"]); ok {
		e.Hours = v
	}
	if v, ok := fields["date"].(string); ok {
		e.Date = v
	}
	projectID, taskID := e.ProjectID, e.TaskID
	if v, ok := toInt(fields["project_id"]); ok {
		projectID = v
	}
	if v, ok := toInt(fields["task_id"]); ok {
		taskID = v
	} else if fields["task_id"] == false {
		taskID = 0
	}
	if err := c.applyProjectTask(&e, projectID, taskID); err != nil {
		return err
	}
	c.st.Timesheets[i] = e
	return c.save()
}

// DeleteTimesheet implements odoo.Client.
func (c *Client) DeleteTimesheet(id int64) error {
	i := c.timesheetIndex(id)
	if i < 0 {
		return missing("account.analytic.line", id)
	}
	c.st.Timesheets = append(c.st.Timesheets[:i], c.st.Timesheets[i+1:]...)
	return c.save()
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

func toInt(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	}
	return 0, false
}
