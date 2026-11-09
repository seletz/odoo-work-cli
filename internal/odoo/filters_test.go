package odoo

import (
	"encoding/json"
	"testing"

	"github.com/seletz/odoo-work-cli/internal/config"
	goOdoo "github.com/skilld-labs/go-odoo"
)

func TestFiltersForModel(t *testing.T) {
	models := map[string]config.ModelConfig{
		"task": {
			Filters: []config.Filter{
				{Field: "company_id.name", Op: "=", Value: "Company A"},
				{Field: "active", Op: "=", Value: "true"},
			},
		},
		"project": {
			Filters: []config.Filter{
				{Field: "stage_id.name", Op: "!=", Value: "Cancelled"},
			},
		},
	}
	x := &XMLRPCClient{models: models}

	t.Run("known model returns filters", func(t *testing.T) {
		filters := x.filtersForModel("task")
		if len(filters) != 2 {
			t.Fatalf("len(filters) = %d, want 2", len(filters))
		}
		if filters[0].Field != "company_id.name" {
			t.Errorf("filters[0].Field = %q, want %q", filters[0].Field, "company_id.name")
		}
	})

	t.Run("unknown model returns nil", func(t *testing.T) {
		filters := x.filtersForModel("timesheet")
		if filters != nil {
			t.Errorf("expected nil, got %v", filters)
		}
	})

	t.Run("nil models map returns nil", func(t *testing.T) {
		x2 := &XMLRPCClient{}
		filters := x2.filtersForModel("task")
		if filters != nil {
			t.Errorf("expected nil, got %v", filters)
		}
	})
}

// criteriaFields extracts the field names from a criteria's criterions.
func criteriaFields(t *testing.T, criteria *goOdoo.Criteria) []string {
	t.Helper()
	var fields []string
	for _, c := range *criteria {
		tuple, ok := c.([]interface{})
		if !ok || len(tuple) < 1 {
			t.Fatalf("unexpected criterion shape: %#v", c)
		}
		field, ok := tuple[0].(string)
		if !ok {
			t.Fatalf("unexpected criterion field type: %#v", tuple[0])
		}
		fields = append(fields, field)
	}
	return fields
}

// Issue #58: timesheet entries booked on another company's project carry that
// company's company_id. A configured [models.timesheet] company filter must
// only be part of the domain when filters are requested.
func TestTimesheetCriteria(t *testing.T) {
	x := &XMLRPCClient{
		login: "user@example.com",
		models: map[string]config.ModelConfig{
			"timesheet": {
				Filters: []config.Filter{
					{Field: "company_id.name", Op: "=", Value: "digitalgedacht GmbH"},
				},
			},
		},
	}

	t.Run("filtered includes configured filters", func(t *testing.T) {
		fields := criteriaFields(t, x.timesheetCriteria("2026-07-13", "2026-07-19", true))
		want := []string{"date", "date", "user_id.login", "company_id.name"}
		if len(fields) != len(want) {
			t.Fatalf("criteria fields = %v, want %v", fields, want)
		}
		for i := range want {
			if fields[i] != want[i] {
				t.Errorf("fields[%d] = %q, want %q", i, fields[i], want[i])
			}
		}
	})

	t.Run("unfiltered omits configured filters", func(t *testing.T) {
		fields := criteriaFields(t, x.timesheetCriteria("2026-07-13", "2026-07-19", false))
		want := []string{"date", "date", "user_id.login"}
		if len(fields) != len(want) {
			t.Fatalf("criteria fields = %v, want %v", fields, want)
		}
		for i := range want {
			if fields[i] != want[i] {
				t.Errorf("fields[%d] = %q, want %q", i, fields[i], want[i])
			}
		}
	})
}

// criteriaString renders a domain as Odoo sees it, e.g.
// `["|", ["name","ilike","x"], ...]`, so tests can pin the exact structure.
func criteriaString(t *testing.T, criteria *goOdoo.Criteria) string {
	t.Helper()
	b, err := json.Marshal(*criteria)
	if err != nil {
		t.Fatalf("marshal criteria: %v", err)
	}
	return string(b)
}

// Issue #86: --search is an ilike match evaluated by Odoo. For projects it
// matches the name OR the customer; it is ANDed with the configured filters.
func TestProjectCriteria(t *testing.T) {
	x := &XMLRPCClient{
		models: map[string]config.ModelConfig{
			"project": {Filters: []config.Filter{{Field: "active", Op: "=", Value: "true"}}},
		},
	}
	tests := []struct {
		name     string
		filtered bool
		search   string
		want     string
	}{
		{"no search, filtered", true, "", `[["active","=","true"]]`},
		{"no search, unfiltered", false, "", `[]`},
		{"search, filtered", true, "ab-1",
			`[["active","=","true"],"|",["name","ilike","ab-1"],["partner_id","ilike","ab-1"]]`},
		{"search, unfiltered", false, "ab-1",
			`["|",["name","ilike","ab-1"],["partner_id","ilike","ab-1"]]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := criteriaString(t, x.projectCriteria(tt.filtered, tt.search))
			if got != tt.want {
				t.Errorf("domain = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTaskCriteria(t *testing.T) {
	x := &XMLRPCClient{
		models: map[string]config.ModelConfig{
			"task": {Filters: []config.Filter{{Field: "stage_id.fold", Op: "=", Value: "false"}}},
		},
	}
	tests := []struct {
		name      string
		projectID int64
		filtered  bool
		search    string
		want      string
	}{
		{"nothing, filtered", 0, true, "", `[["stage_id.fold","=","false"]]`},
		{"nothing, unfiltered", 0, false, "", `[]`},
		{"project only", 7, true, "", `[["stage_id.fold","=","false"],["project_id","=",7]]`},
		{"search only", 0, true, "bug",
			`[["stage_id.fold","=","false"],["name","ilike","bug"]]`},
		{"search and project", 7, true, "bug",
			`[["stage_id.fold","=","false"],["project_id","=",7],["name","ilike","bug"]]`},
		{"search, unfiltered", 7, false, "bug",
			`[["project_id","=",7],["name","ilike","bug"]]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := criteriaString(t, x.taskCriteria(tt.projectID, tt.filtered, tt.search))
			if got != tt.want {
				t.Errorf("domain = %s, want %s", got, tt.want)
			}
		})
	}
}
