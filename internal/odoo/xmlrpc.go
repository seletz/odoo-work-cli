package odoo

import (
	"fmt"
	"sort"

	"github.com/seletz/odoo-work-cli/internal/config"
	goOdoo "github.com/skilld-labs/go-odoo"
)

// XMLRPCClient implements Client using the Odoo XML-RPC API.
type XMLRPCClient struct {
	client *goOdoo.Client
	login  string
	models map[string]config.ModelConfig
}

// NewXMLRPCClient creates a new Odoo client and authenticates.
// The models parameter provides per-model extra field configuration.
func NewXMLRPCClient(url, database, username, password string, models map[string]config.ModelConfig) (*XMLRPCClient, error) {
	c, err := goOdoo.NewClient(&goOdoo.ClientConfig{
		Admin:    username,
		Password: password,
		Database: database,
		URL:      url,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to odoo: %w", err)
	}
	return &XMLRPCClient{client: c, login: username, models: models}, nil
}

// Close closes the underlying XML-RPC connections.
func (x *XMLRPCClient) Close() {
	x.client.Close()
}

// searchReadRaw calls ExecuteKw("search_read", ...) and returns raw maps.
func (x *XMLRPCClient) searchReadRaw(model string, criteria *goOdoo.Criteria, opts *goOdoo.Options) ([]map[string]interface{}, error) {
	resp, err := x.client.ExecuteKw("search_read", model, []interface{}{*criteria}, opts)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	items, ok := resp.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response type %T", resp)
	}
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		result = append(result, m)
	}
	return result, nil
}

// extractMany2OneName extracts the display name from a Many2One field value.
// Many2One fields are represented as [id, name] or false in XML-RPC.
func extractMany2OneName(v interface{}) string {
	if v == nil || v == false {
		return ""
	}
	arr, ok := v.([]interface{})
	if !ok || len(arr) < 2 {
		return ""
	}
	name, _ := arr[1].(string)
	return name
}

// extractMany2OneID extracts the numeric ID from a Many2One field value.
// Many2One fields are represented as [id, name] or false in XML-RPC.
func extractMany2OneID(v interface{}) int64 {
	if v == nil || v == false {
		return 0
	}
	arr, ok := v.([]interface{})
	if !ok || len(arr) < 1 {
		return 0
	}
	switch id := arr[0].(type) {
	case int64:
		return id
	case float64:
		return int64(id)
	default:
		return 0
	}
}

// extractFieldValue extracts a field value as a string based on its Odoo type.
func extractFieldValue(v interface{}, fieldType string) string {
	if v == nil || v == false {
		return ""
	}
	switch fieldType {
	case "many2one":
		return extractMany2OneName(v)
	case "char", "selection", "text":
		s, _ := v.(string)
		return s
	case "boolean":
		b, _ := v.(bool)
		if b {
			return "true"
		}
		return "false"
	case "integer":
		switch n := v.(type) {
		case int64:
			return fmt.Sprintf("%d", n)
		case float64:
			return fmt.Sprintf("%d", int64(n))
		}
		return fmt.Sprintf("%v", v)
	case "float":
		switch n := v.(type) {
		case float64:
			return fmt.Sprintf("%.2f", n)
		case int64:
			return fmt.Sprintf("%.2f", float64(n))
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// extractExtraFields reads configured extra fields from a raw record map.
func (x *XMLRPCClient) extractExtraFields(modelKey string, record map[string]interface{}) map[string]string {
	mc, ok := x.models[modelKey]
	if !ok || len(mc.ExtraFields) == 0 {
		return nil
	}
	extras := make(map[string]string, len(mc.ExtraFields))
	for _, ef := range mc.ExtraFields {
		v := record[ef.Field]
		extras[ef.Name] = extractFieldValue(v, ef.Type)
	}
	return extras
}

// filtersForModel returns configured default filters for the given model key.
func (x *XMLRPCClient) filtersForModel(modelKey string) []config.Filter {
	if x.models == nil {
		return nil
	}
	mc, ok := x.models[modelKey]
	if !ok || len(mc.Filters) == 0 {
		return nil
	}
	return mc.Filters
}

// applyCriteriaFilters adds configured default filters for the given model
// to the criteria.
func (x *XMLRPCClient) applyCriteriaFilters(criteria *goOdoo.Criteria, modelKey string) {
	for _, f := range x.filtersForModel(modelKey) {
		criteria.Add(f.Field, f.Op, f.Value)
	}
}

// ListProjects returns all projects from Odoo.
func (x *XMLRPCClient) ListProjects() ([]ProjectInfo, error) {
	criteria := goOdoo.NewCriteria()
	x.applyCriteriaFilters(criteria, "project")

	// Build field list: standard fields + configured extra fields.
	fields := []string{"id", "name", "active", "partner_id", "company_id", "stage_id", "user_id"}
	if mc, ok := x.models["project"]; ok {
		for _, ef := range mc.ExtraFields {
			fields = append(fields, ef.Field)
		}
	}
	opts := goOdoo.NewOptions().FetchFields(fields...)

	records, err := x.searchReadRaw("project.project", criteria, opts)
	if err != nil {
		if IsNotFound(err) {
			return []ProjectInfo{}, nil
		}
		return nil, fmt.Errorf("fetching projects: %w", err)
	}

	result := make([]ProjectInfo, 0, len(records))
	for _, r := range records {
		info := ProjectInfo{
			Customer:       extractMany2OneName(r["partner_id"]),
			Company:        extractMany2OneName(r["company_id"]),
			Stage:          extractMany2OneName(r["stage_id"]),
			ProjectManager: extractMany2OneName(r["user_id"]),
			ExtraFields:    x.extractExtraFields("project", r),
		}
		if id, ok := r["id"].(int64); ok {
			info.ID = id
		} else if id, ok := r["id"].(float64); ok {
			info.ID = int64(id)
		}
		if name, ok := r["name"].(string); ok {
			info.Name = name
		}
		if active, ok := r["active"].(bool); ok {
			info.Active = active
		}
		result = append(result, info)
	}
	return result, nil
}

// ListTasks returns tasks from Odoo, optionally filtered by project ID.
// Pass projectID <= 0 to list all tasks.
func (x *XMLRPCClient) ListTasks(projectID int64) ([]TaskInfo, error) {
	criteria := goOdoo.NewCriteria()
	x.applyCriteriaFilters(criteria, "task")
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
	x.applyCriteriaFilters(criteria, "timesheet")

	fields := []string{"id", "date", "project_id", "task_id", "name", "unit_amount", "employee_id", "validated_status"}
	opts := goOdoo.NewOptions().FetchFields(fields...)

	records, err := x.searchReadRaw("account.analytic.line", criteria, opts)
	if IsNotFound(err) {
		return []TimesheetEntry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fetching timesheets: %w", err)
	}

	result := make([]TimesheetEntry, 0, len(records))
	for _, r := range records {
		entry := TimesheetEntry{
			ProjectID: extractMany2OneID(r["project_id"]),
			Project:   extractMany2OneName(r["project_id"]),
			TaskID:    extractMany2OneID(r["task_id"]),
			Task:      extractMany2OneName(r["task_id"]),
			Employee:  extractMany2OneName(r["employee_id"]),
		}
		if id, ok := r["id"].(int64); ok {
			entry.ID = id
		} else if id, ok := r["id"].(float64); ok {
			entry.ID = int64(id)
		}
		if name, ok := r["name"].(string); ok {
			entry.Name = name
		}
		if date, ok := r["date"].(string); ok {
			entry.Date = date
		}
		if hours, ok := r["unit_amount"].(float64); ok {
			entry.Hours = hours
		}
		if status, ok := r["validated_status"].(string); ok {
			entry.ValidatedStatus = status
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
