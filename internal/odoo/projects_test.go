package odoo

import (
	"errors"
	"testing"
)

func TestListProjects(t *testing.T) {
	tests := []struct {
		name     string
		client   *mockClient
		wantLen  int
		wantErr  bool
		wantMsg  string
		checkFn  func(t *testing.T, projects []ProjectInfo)
	}{
		{
			name: "success returns projects with all fields",
			client: &mockClient{
				projects: []ProjectInfo{
					{
						ID: 1, Name: "Project Alpha", Active: true,
						Customer: "ACME Corp", Company: "nexiles",
						Stage: "In Progress", ProductOwner: "Jane Doe",
						ProjectManager: "John Smith",
					},
					{
						ID: 2, Name: "Project Beta", Active: false,
						Customer: "Globex", Company: "digitalgedacht",
						Stage: "Done", ProductOwner: "Bob",
						ProjectManager: "Alice",
					},
				},
			},
			wantLen: 2,
			checkFn: func(t *testing.T, projects []ProjectInfo) {
				t.Helper()
				p := projects[0]
				if p.ID != 1 {
					t.Errorf("ID = %d, want 1", p.ID)
				}
				if p.Name != "Project Alpha" {
					t.Errorf("Name = %q, want %q", p.Name, "Project Alpha")
				}
				if !p.Active {
					t.Error("Active = false, want true")
				}
				if p.Customer != "ACME Corp" {
					t.Errorf("Customer = %q, want %q", p.Customer, "ACME Corp")
				}
				if p.Company != "nexiles" {
					t.Errorf("Company = %q, want %q", p.Company, "nexiles")
				}
				if p.Stage != "In Progress" {
					t.Errorf("Stage = %q, want %q", p.Stage, "In Progress")
				}
				if p.ProductOwner != "Jane Doe" {
					t.Errorf("ProductOwner = %q, want %q", p.ProductOwner, "Jane Doe")
				}
				if p.ProjectManager != "John Smith" {
					t.Errorf("ProjectManager = %q, want %q", p.ProjectManager, "John Smith")
				}
			},
		},
		{
			name:    "empty list returns no error",
			client:  &mockClient{projects: []ProjectInfo{}},
			wantLen: 0,
		},
		{
			name:    "error is propagated",
			client:  &mockClient{projErr: errors.New("connection refused")},
			wantErr: true,
			wantMsg: "connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projects, err := tt.client.ListProjects()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err.Error() != tt.wantMsg {
					t.Errorf("error = %q, want %q", err.Error(), tt.wantMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(projects) != tt.wantLen {
				t.Fatalf("len(projects) = %d, want %d", len(projects), tt.wantLen)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, projects)
			}
		})
	}
}
