package odoo

import (
	"errors"
	"fmt"

	goOdoo "github.com/skilld-labs/go-odoo"
)

// ValidateTimesheetParams validates that the required fields are set.
func ValidateTimesheetParams(p TimesheetWriteParams) error {
	if p.ProjectID <= 0 && p.TaskID <= 0 {
		return errors.New("project ID or task ID is required")
	}
	if p.Date == "" {
		return errors.New("date is required")
	}
	if p.Name == "" {
		return errors.New("description is required")
	}
	if p.Hours <= 0 {
		return errors.New("hours must be greater than zero")
	}
	return nil
}

// timesheetValues builds the Odoo field map from write params.
func timesheetValues(p TimesheetWriteParams) map[string]interface{} {
	vals := map[string]interface{}{
		"project_id":  p.ProjectID,
		"date":        p.Date,
		"name":        p.Name,
		"unit_amount": p.Hours,
	}
	if p.TaskID > 0 {
		vals["task_id"] = p.TaskID
	}
	return vals
}

// CreateTimesheet creates a new timesheet entry and returns its ID.
func (x *XMLRPCClient) CreateTimesheet(params TimesheetWriteParams) (int64, error) {
	if err := ValidateTimesheetParams(params); err != nil {
		return 0, err
	}

	vals := timesheetValues(params)
	resp, err := x.client.ExecuteKw("create", "account.analytic.line",
		[]interface{}{vals}, goOdoo.NewOptions())
	if err != nil {
		return 0, fmt.Errorf("creating timesheet: %w", err)
	}

	switch id := resp.(type) {
	case int64:
		return id, nil
	case float64:
		return int64(id), nil
	default:
		return 0, fmt.Errorf("unexpected create response type %T", resp)
	}
}

// UpdateTimesheet partially updates an existing timesheet entry.
// Only the fields present in the map are sent to Odoo.
func (x *XMLRPCClient) UpdateTimesheet(id int64, fields map[string]interface{}) error {
	if id <= 0 {
		return errors.New("timesheet ID is required")
	}
	if len(fields) == 0 {
		return errors.New("at least one field to update is required")
	}

	_, err := x.client.ExecuteKw("write", "account.analytic.line",
		[]interface{}{[]int64{id}, fields}, goOdoo.NewOptions())
	if err != nil {
		return fmt.Errorf("updating timesheet %d: %w", id, err)
	}
	return nil
}

// DeleteTimesheet deletes a timesheet entry by ID.
func (x *XMLRPCClient) DeleteTimesheet(id int64) error {
	if id <= 0 {
		return errors.New("timesheet ID is required")
	}

	_, err := x.client.ExecuteKw("unlink", "account.analytic.line",
		[]interface{}{[]int64{id}}, goOdoo.NewOptions())
	if err != nil {
		return fmt.Errorf("deleting timesheet %d: %w", id, err)
	}
	return nil
}
