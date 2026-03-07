package odoo

import (
	"errors"
	"testing"
)

func TestListTasks(t *testing.T) {
	tests := []struct {
		name    string
		client  *mockClient
		projID  int64
		wantLen int
		wantErr bool
		wantMsg string
		checkFn func(t *testing.T, tasks []TaskInfo)
	}{
		{
			name: "success returns tasks",
			client: &mockClient{
				tasks: []TaskInfo{
					{ID: 10, Name: "Task A", Project: "Alpha", Stage: "In Progress", Active: true},
					{ID: 20, Name: "Task B", Project: "Alpha", Stage: "Done", Active: true},
				},
			},
			wantLen: 2,
			checkFn: func(t *testing.T, tasks []TaskInfo) {
				t.Helper()
				if tasks[0].ID != 10 {
					t.Errorf("tasks[0].ID = %d, want 10", tasks[0].ID)
				}
				if tasks[0].Name != "Task A" {
					t.Errorf("tasks[0].Name = %q, want %q", tasks[0].Name, "Task A")
				}
				if tasks[0].Project != "Alpha" {
					t.Errorf("tasks[0].Project = %q, want %q", tasks[0].Project, "Alpha")
				}
				if tasks[0].Stage != "In Progress" {
					t.Errorf("tasks[0].Stage = %q, want %q", tasks[0].Stage, "In Progress")
				}
			},
		},
		{
			name:    "empty list returns no error",
			client:  &mockClient{tasks: []TaskInfo{}},
			wantLen: 0,
		},
		{
			name:    "error is propagated",
			client:  &mockClient{taskErr: errors.New("access denied")},
			wantErr: true,
			wantMsg: "access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks, err := tt.client.ListTasks(tt.projID)

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
			if len(tasks) != tt.wantLen {
				t.Fatalf("len(tasks) = %d, want %d", len(tasks), tt.wantLen)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, tasks)
			}
		})
	}
}
