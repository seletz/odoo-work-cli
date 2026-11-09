package app

import (
	"errors"
	"fmt"
	"testing"
	"time"

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

func (s *stubClient) ListAllTimesheets(string, string) ([]odoo.TimesheetEntry, error) {
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

func (s *stubClient) ClockInAt(time.Time) (int64, error) {
	return 0, nil
}

func (s *stubClient) ClockOutAt(time.Time) (*odoo.AttendanceRecord, error) {
	return nil, nil
}

func (s *stubClient) CreateAttendance(time.Time, time.Time) (int64, error) {
	return 0, nil
}

func (s *stubClient) EditAttendance(int64, *time.Time, *time.Time) (*odoo.AttendanceRecord, error) {
	return nil, nil
}

func (s *stubClient) AttendanceStatus() (*odoo.AttendanceStatus, error) {
	return nil, nil
}

func (s *stubClient) ListAttendance(time.Time, time.Time) ([]odoo.AttendanceRecord, error) {
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

func (s *stubClient) GetTimesheet(int64) (*odoo.TimesheetEntry, error) { return nil, nil }

func (s *stubClient) GetAttendance(int64) (*odoo.AttendanceRecord, error) { return nil, nil }

func TestDepsLazyLoader(t *testing.T) {
	cfg := &config.Config{URL: "https://example.com"}
	client := &stubClient{}
	calls := 0
	deps := &Deps{Loader: func() (*config.Config, odoo.Client, error) {
		calls++
		return cfg, client, nil
	}}

	if calls != 0 {
		t.Fatalf("loader called before first use")
	}
	gotCfg, err := deps.RequireConfig()
	if err != nil || gotCfg != cfg {
		t.Fatalf("RequireConfig() = %v, %v", gotCfg, err)
	}
	gotClient, err := deps.RequireClient()
	if err != nil || gotClient != client {
		t.Fatalf("RequireClient() = %v, %v", gotClient, err)
	}
	if calls != 1 {
		t.Fatalf("loader calls = %d, want 1", calls)
	}
}

func TestDepsLazyLoaderError(t *testing.T) {
	wantErr := fmt.Errorf("locked")
	calls := 0
	deps := &Deps{Loader: func() (*config.Config, odoo.Client, error) {
		calls++
		return nil, nil, wantErr
	}}

	if _, err := deps.RequireClient(); !errors.Is(err, wantErr) {
		t.Fatalf("RequireClient() error = %v, want %v", err, wantErr)
	}
	if _, err := deps.RequireConfig(); !errors.Is(err, wantErr) {
		t.Fatalf("RequireConfig() error = %v, want %v", err, wantErr)
	}
	if calls != 1 {
		t.Fatalf("loader calls = %d, want 1 (error is cached)", calls)
	}
}

func TestDepsCloseOnlyAfterLoad(t *testing.T) {
	deps := &Deps{Loader: func() (*config.Config, odoo.Client, error) {
		t.Fatal("Close must not trigger the loader")
		return nil, nil, nil
	}}
	deps.Close()
}
