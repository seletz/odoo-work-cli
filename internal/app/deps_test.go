package app

import (
	"testing"

	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

type stubClient struct{}

func (s *stubClient) Close() {}

func (s *stubClient) WhoAmI() (*odoo.UserInfo, error) {
	return nil, nil
}

func (s *stubClient) ListProjects() ([]odoo.ProjectInfo, error) {
	return nil, nil
}

func (s *stubClient) ListAllProjects() ([]odoo.ProjectInfo, error) {
	return nil, nil
}

func (s *stubClient) ListTasks(int64) ([]odoo.TaskInfo, error) {
	return nil, nil
}

func (s *stubClient) ListAllTasks(int64) ([]odoo.TaskInfo, error) {
	return nil, nil
}

func (s *stubClient) ListTimesheets(string, string) ([]odoo.TimesheetEntry, error) {
	return nil, nil
}

func (s *stubClient) GetFields(string) ([]odoo.FieldInfo, error) {
	return nil, nil
}

func (s *stubClient) CreateTimesheet(odoo.TimesheetWriteParams) (int64, error) {
	return 0, nil
}

func (s *stubClient) UpdateTimesheet(int64, map[string]interface{}) error {
	return nil
}

func (s *stubClient) DeleteTimesheet(int64) error {
	return nil
}

func (s *stubClient) ClockIn() (int64, error) {
	return 0, nil
}

func (s *stubClient) ClockOut() (*odoo.AttendanceRecord, error) {
	return nil, nil
}

func (s *stubClient) AttendanceStatus() (*odoo.AttendanceStatus, error) {
	return nil, nil
}

func TestDepsRequireClientReturnsInterface(t *testing.T) {
	client := &stubClient{}
	deps := &Deps{Client: client}

	got, err := deps.RequireClient()
	if err != nil {
		t.Fatalf("RequireClient() error = %v, want nil", err)
	}
	if got != client {
		t.Fatalf("RequireClient() returned %T, want original client", got)
	}
}

func TestDepsRequireClientErrorsWhenUninitialized(t *testing.T) {
	tests := []struct {
		name string
		deps *Deps
	}{
		{name: "nil deps", deps: nil},
		{name: "nil client", deps: &Deps{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.deps.RequireClient()
			if err == nil {
				t.Fatal("RequireClient() error = nil, want non-nil")
			}
			if got != nil {
				t.Fatalf("RequireClient() client = %v, want nil", got)
			}
		})
	}
}

func TestDepsRequireConfig(t *testing.T) {
	cfg := &config.Config{URL: "https://example.com"}
	deps := &Deps{Config: cfg}

	got, err := deps.RequireConfig()
	if err != nil {
		t.Fatalf("RequireConfig() error = %v, want nil", err)
	}
	if got != cfg {
		t.Fatalf("RequireConfig() returned %+v, want original config", got)
	}
}

func TestDepsRequireConfigErrorsWhenUninitialized(t *testing.T) {
	tests := []struct {
		name string
		deps *Deps
	}{
		{name: "nil deps", deps: nil},
		{name: "nil config", deps: &Deps{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.deps.RequireConfig()
			if err == nil {
				t.Fatal("RequireConfig() error = nil, want non-nil")
			}
			if got != nil {
				t.Fatalf("RequireConfig() config = %+v, want nil", got)
			}
		})
	}
}
