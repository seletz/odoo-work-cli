package odoo

import (
	"errors"
	"testing"
)

// mockClient implements Client for testing.
type mockClient struct {
	info       *UserInfo
	err        error
	projects   []ProjectInfo
	projErr    error
	tasks      []TaskInfo
	taskErr    error
	timesheets []TimesheetEntry
	tsErr      error
	fields     []FieldInfo
	fieldsErr  error
	createID   int64
	createErr  error
	updateErr  error
	deleteErr  error
}

func (m *mockClient) WhoAmI() (*UserInfo, error) {
	return m.info, m.err
}

func (m *mockClient) ListProjects() ([]ProjectInfo, error) {
	return m.projects, m.projErr
}

func (m *mockClient) ListTasks(_ int64) ([]TaskInfo, error) {
	return m.tasks, m.taskErr
}

func (m *mockClient) ListTimesheets(_, _ string) ([]TimesheetEntry, error) {
	return m.timesheets, m.tsErr
}

func (m *mockClient) GetFields(_ string) ([]FieldInfo, error) {
	return m.fields, m.fieldsErr
}

func (m *mockClient) CreateTimesheet(_ TimesheetWriteParams) (int64, error) {
	return m.createID, m.createErr
}

func (m *mockClient) UpdateTimesheet(_ int64, _ TimesheetWriteParams) error {
	return m.updateErr
}

func (m *mockClient) DeleteTimesheet(_ int64) error {
	return m.deleteErr
}

func TestWhoAmI_Success(t *testing.T) {
	client := &mockClient{
		info: &UserInfo{
			ID:      42,
			Name:    "Test User",
			Login:   "test@example.com",
			Email:   "test@example.com",
			Company: "ACME Corp",
		},
	}

	info, err := client.WhoAmI()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ID != 42 {
		t.Errorf("ID = %d, want 42", info.ID)
	}
	if info.Name != "Test User" {
		t.Errorf("Name = %q, want %q", info.Name, "Test User")
	}
	if info.Login != "test@example.com" {
		t.Errorf("Login = %q, want %q", info.Login, "test@example.com")
	}
	if info.Company != "ACME Corp" {
		t.Errorf("Company = %q, want %q", info.Company, "ACME Corp")
	}
}

func TestWhoAmI_Error(t *testing.T) {
	client := &mockClient{
		err: errors.New("authentication failed"),
	}

	info, err := client.WhoAmI()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if info != nil {
		t.Errorf("expected nil info on error, got %+v", info)
	}
	if err.Error() != "authentication failed" {
		t.Errorf("error = %q, want %q", err.Error(), "authentication failed")
	}
}

func TestXMLRPCClient_ImplementsClient(t *testing.T) {
	// Compile-time check that XMLRPCClient satisfies the Client interface.
	var _ Client = (*XMLRPCClient)(nil)
}
