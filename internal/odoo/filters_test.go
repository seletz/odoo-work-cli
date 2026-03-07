package odoo

import (
	"testing"

	"github.com/seletz/odoo-work-cli/internal/config"
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
