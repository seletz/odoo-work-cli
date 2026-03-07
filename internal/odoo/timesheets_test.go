package odoo

import (
	"errors"
	"testing"
)

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
					{ID: 100, Date: "2026-03-02", Project: "Alpha", Task: "Task A", Name: "Dev work", Hours: 4.0, Employee: "Test User", ValidatedStatus: "draft"},
					{ID: 101, Date: "2026-03-02", Project: "Beta", Task: "Task B", Name: "Review", Hours: 2.5, Employee: "Test User", ValidatedStatus: "validated"},
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
