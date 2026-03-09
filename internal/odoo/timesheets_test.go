package odoo

import (
	"errors"
	"testing"
)

func TestExtractMany2OneID(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		want int64
	}{
		{
			name: "int64 array",
			val:  []interface{}{int64(42), "Project Alpha"},
			want: 42,
		},
		{
			name: "float64 array",
			val:  []interface{}{float64(99), "Task Beta"},
			want: 99,
		},
		{
			name: "nil value",
			val:  nil,
			want: 0,
		},
		{
			name: "false value",
			val:  false,
			want: 0,
		},
		{
			name: "empty array",
			val:  []interface{}{},
			want: 0,
		},
		{
			name: "non-array value",
			val:  "not an array",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMany2OneID(tt.val)
			if got != tt.want {
				t.Errorf("extractMany2OneID(%v) = %d, want %d", tt.val, got, tt.want)
			}
		})
	}
}

func TestListTimesheets(t *testing.T) {
	tests := []struct {
		name    string
		client  *mockClient
		wantLen int
		wantErr bool
		wantMsg string
		checkFn func(t *testing.T, entries []TimesheetEntry)
	}{
		{
			name: "success returns only current user entries",
			client: &mockClient{
				timesheets: []TimesheetEntry{
					{ID: 100, Date: "2026-03-02", Project: "Alpha", ProjectID: 10, Task: "Task A", TaskID: 20, Name: "Dev work", Hours: 4.0, Employee: "Test User", ValidatedStatus: "draft"},
					{ID: 101, Date: "2026-03-02", Project: "Beta", ProjectID: 11, Task: "Task B", TaskID: 21, Name: "Review", Hours: 2.5, Employee: "Test User", ValidatedStatus: "validated"},
				},
			},
			wantLen: 2,
			checkFn: func(t *testing.T, entries []TimesheetEntry) {
				t.Helper()
				if entries[0].ID != 100 {
					t.Errorf("entries[0].ID = %d, want 100", entries[0].ID)
				}
				if entries[0].Date != "2026-03-02" {
					t.Errorf("entries[0].Date = %q, want %q", entries[0].Date, "2026-03-02")
				}
				if entries[0].Hours != 4.0 {
					t.Errorf("entries[0].Hours = %f, want 4.0", entries[0].Hours)
				}
				if entries[0].Project != "Alpha" {
					t.Errorf("entries[0].Project = %q, want %q", entries[0].Project, "Alpha")
				}
				if entries[0].ProjectID != 10 {
					t.Errorf("entries[0].ProjectID = %d, want 10", entries[0].ProjectID)
				}
				if entries[0].TaskID != 20 {
					t.Errorf("entries[0].TaskID = %d, want 20", entries[0].TaskID)
				}
				if entries[1].ProjectID != 11 {
					t.Errorf("entries[1].ProjectID = %d, want 11", entries[1].ProjectID)
				}
				if entries[1].TaskID != 21 {
					t.Errorf("entries[1].TaskID = %d, want 21", entries[1].TaskID)
				}
				if entries[0].ValidatedStatus != "draft" {
					t.Errorf("entries[0].ValidatedStatus = %q, want %q", entries[0].ValidatedStatus, "draft")
				}
				if entries[1].ValidatedStatus != "validated" {
					t.Errorf("entries[1].ValidatedStatus = %q, want %q", entries[1].ValidatedStatus, "validated")
				}
				// All entries should belong to the same user.
				for i, e := range entries {
					if e.Employee != "Test User" {
						t.Errorf("entries[%d].Employee = %q, want %q", i, e.Employee, "Test User")
					}
				}
			},
		},
		{
			name:    "empty list returns no error",
			client:  &mockClient{timesheets: []TimesheetEntry{}},
			wantLen: 0,
		},
		{
			name:    "error is propagated",
			client:  &mockClient{tsErr: errors.New("timeout")},
			wantErr: true,
			wantMsg: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := tt.client.ListTimesheets("2026-03-02", "2026-03-06")

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
			if len(entries) != tt.wantLen {
				t.Fatalf("len(entries) = %d, want %d", len(entries), tt.wantLen)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, entries)
			}
		})
	}
}
