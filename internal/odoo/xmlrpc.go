package odoo

import (
	"fmt"
	"sort"

	goOdoo "github.com/skilld-labs/go-odoo"
)

// XMLRPCClient implements Client using the Odoo XML-RPC API.
type XMLRPCClient struct {
	client *goOdoo.Client
	login  string
}

// NewXMLRPCClient creates a new Odoo client and authenticates.
func NewXMLRPCClient(url, database, username, password string) (*XMLRPCClient, error) {
	c, err := goOdoo.NewClient(&goOdoo.ClientConfig{
		Admin:    username,
		Password: password,
		Database: database,
		URL:      url,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to odoo: %w", err)
	}
	return &XMLRPCClient{client: c, login: username}, nil
}

// Close closes the underlying XML-RPC connections.
func (x *XMLRPCClient) Close() {
	x.client.Close()
}

// projectRecord is a custom struct for deserializing project.project via SearchRead.
// It includes the custom field x_studio_productowner that is not in the generated go-odoo types.
type projectRecord struct {
	Id                   *goOdoo.Int     `xmlrpc:"id,omitempty"`
	Name                 *goOdoo.String  `xmlrpc:"name,omitempty"`
	Active               *goOdoo.Bool    `xmlrpc:"active,omitempty"`
	PartnerId            *goOdoo.Many2One `xmlrpc:"partner_id,omitempty"`
	CompanyId            *goOdoo.Many2One `xmlrpc:"company_id,omitempty"`
	StageId              *goOdoo.Many2One `xmlrpc:"stage_id,omitempty"`
	UserId               *goOdoo.Many2One `xmlrpc:"user_id,omitempty"`
	XStudioProductowner  *goOdoo.Many2One `xmlrpc:"x_studio_productowner,omitempty"`
}

type projectRecords []projectRecord

// ListProjects returns all projects from Odoo.
func (x *XMLRPCClient) ListProjects() ([]ProjectInfo, error) {
	criteria := goOdoo.NewCriteria()
	var records projectRecords
	err := x.client.SearchRead("project.project", criteria, goOdoo.NewOptions(), &records)
	if IsNotFound(err) {
		return []ProjectInfo{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fetching projects: %w", err)
	}

	result := make([]ProjectInfo, 0, len(records))
	for _, p := range records {
		info := ProjectInfo{
			ID:     p.Id.Get(),
			Name:   p.Name.Get(),
			Active: p.Active.Get(),
		}
		if p.PartnerId != nil {
			info.Customer = p.PartnerId.Name
		}
		if p.CompanyId != nil {
			info.Company = p.CompanyId.Name
		}
		if p.StageId != nil {
			info.Stage = p.StageId.Name
		}
		if p.UserId != nil {
			info.ProjectManager = p.UserId.Name
		}
		if p.XStudioProductowner != nil {
			info.ProductOwner = p.XStudioProductowner.Name
		}
		result = append(result, info)
	}
	return result, nil
}

// ListTasks returns tasks from Odoo, optionally filtered by project ID.
// Pass projectID <= 0 to list all tasks.
func (x *XMLRPCClient) ListTasks(projectID int64) ([]TaskInfo, error) {
	criteria := goOdoo.NewCriteria()
	if projectID > 0 {
		criteria.Add("project_id", "=", projectID)
	}
	tasks, err := x.client.FindProjectTasks(criteria, goOdoo.NewOptions())
	if IsNotFound(err) {
		return []TaskInfo{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fetching tasks: %w", err)
	}

	result := make([]TaskInfo, 0, len(*tasks))
	for _, t := range *tasks {
		info := TaskInfo{
			ID:     t.Id.Get(),
			Name:   t.Name.Get(),
			Active: t.Active.Get(),
		}
		if t.ProjectId != nil {
			info.Project = t.ProjectId.Name
		}
		if t.StageId != nil {
			info.Stage = t.StageId.Name
		}
		result = append(result, info)
	}
	return result, nil
}

// ListTimesheets returns timesheet entries for the given date range.
func (x *XMLRPCClient) ListTimesheets(dateFrom, dateTo string) ([]TimesheetEntry, error) {
	criteria := goOdoo.NewCriteria().
		Add("date", ">=", dateFrom).
		Add("date", "<=", dateTo).
		Add("user_id.login", "=", x.login)
	lines, err := x.client.FindAccountAnalyticLines(criteria, goOdoo.NewOptions())
	if IsNotFound(err) {
		return []TimesheetEntry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fetching timesheets: %w", err)
	}

	result := make([]TimesheetEntry, 0, len(*lines))
	for _, l := range *lines {
		entry := TimesheetEntry{
			ID:    l.Id.Get(),
			Name:  l.Name.Get(),
			Hours: l.UnitAmount.Get(),
		}
		if l.Date != nil {
			entry.Date = l.Date.Get().Format("2006-01-02")
		}
		if l.ProjectId != nil {
			entry.Project = l.ProjectId.Name
		}
		if l.TaskId != nil {
			entry.Task = l.TaskId.Name
		}
		if l.EmployeeId != nil {
			entry.Employee = l.EmployeeId.Name
		}
		result = append(result, entry)
	}
	return result, nil
}

// GetFields returns field metadata for the given Odoo model.
func (x *XMLRPCClient) GetFields(model string) ([]FieldInfo, error) {
	resp, err := x.client.FieldsGet(model, goOdoo.NewOptions())
	if err != nil {
		return nil, fmt.Errorf("fetching fields for %s: %w", model, err)
	}

	result := make([]FieldInfo, 0, len(resp))
	for name, raw := range resp {
		info := FieldInfo{Name: name}
		if attrs, ok := raw.(map[string]interface{}); ok {
			if t, ok := attrs["type"].(string); ok {
				info.Type = t
			}
			if s, ok := attrs["string"].(string); ok {
				info.String = s
			}
			if r, ok := attrs["required"].(bool); ok {
				info.Required = r
			}
		}
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// WhoAmI returns the identity of the currently authenticated user.
func (x *XMLRPCClient) WhoAmI() (*UserInfo, error) {
	criteria := goOdoo.NewCriteria().Add("login", "=", x.login)
	user, err := x.client.FindResUsers(criteria)
	if err != nil {
		return nil, fmt.Errorf("fetching user: %w", err)
	}

	info := &UserInfo{
		ID:    user.Id.Get(),
		Name:  user.Name.Get(),
		Login: user.Login.Get(),
		Email: user.Email.Get(),
	}
	if user.CompanyId != nil {
		info.Company = user.CompanyId.Name
	}

	return info, nil
}
