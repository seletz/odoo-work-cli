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
			name: "success returns projects",
			client: &mockClient{
				projects: []ProjectInfo{
					{ID: 1, Name: "Project Alpha", Active: true},
					{ID: 2, Name: "Project Beta", Active: false},
				},
			},
			wantLen: 2,
			checkFn: func(t *testing.T, projects []ProjectInfo) {
				t.Helper()
				if projects[0].ID != 1 {
					t.Errorf("projects[0].ID = %d, want 1", projects[0].ID)
				}
				if projects[0].Name != "Project Alpha" {
					t.Errorf("projects[0].Name = %q, want %q", projects[0].Name, "Project Alpha")
				}
				if !projects[0].Active {
					t.Error("projects[0].Active = false, want true")
				}
				if projects[1].ID != 2 {
					t.Errorf("projects[1].ID = %d, want 2", projects[1].ID)
				}
				if projects[1].Active {
					t.Error("projects[1].Active = true, want false")
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
