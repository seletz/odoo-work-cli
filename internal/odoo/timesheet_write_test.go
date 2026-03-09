package odoo

import (
	"errors"
	"testing"
)

func TestValidateTimesheetParams(t *testing.T) {
	tests := []struct {
		name    string
		params  TimesheetWriteParams
		wantErr bool
		wantMsg string
	}{
		{
			name: "valid params",
			params: TimesheetWriteParams{
				ProjectID: 1,
				Date:      "2026-03-09",
				Name:      "Dev work",
				Hours:     2.5,
			},
		},
		{
			name: "valid with task ID",
			params: TimesheetWriteParams{
				ProjectID: 1,
				TaskID:    10,
				Date:      "2026-03-09",
				Name:      "Task-level entry",
				Hours:     1.0,
			},
		},
		{
			name: "valid with only task ID",
			params: TimesheetWriteParams{
				TaskID: 10,
				Date:   "2026-03-09",
				Name:   "Task only entry",
				Hours:  1.0,
			},
		},
		{
			name: "missing both project and task ID",
			params: TimesheetWriteParams{
				Date:  "2026-03-09",
				Name:  "No project or task",
				Hours: 1.0,
			},
			wantErr: true,
			wantMsg: "project ID or task ID is required",
		},
		{
			name: "missing date",
			params: TimesheetWriteParams{
				ProjectID: 1,
				Name:      "No date",
				Hours:     1.0,
			},
			wantErr: true,
			wantMsg: "date is required",
		},
		{
			name: "missing description",
			params: TimesheetWriteParams{
				ProjectID: 1,
				Date:      "2026-03-09",
				Hours:     1.0,
			},
			wantErr: true,
			wantMsg: "description is required",
		},
		{
			name: "zero hours",
			params: TimesheetWriteParams{
				ProjectID: 1,
				Date:      "2026-03-09",
				Name:      "Zero hours",
				Hours:     0,
			},
			wantErr: true,
			wantMsg: "hours must be greater than zero",
		},
		{
			name: "negative hours",
			params: TimesheetWriteParams{
				ProjectID: 1,
				Date:      "2026-03-09",
				Name:      "Negative hours",
				Hours:     -1.0,
			},
			wantErr: true,
			wantMsg: "hours must be greater than zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimesheetParams(tt.params)

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
		})
	}
}

func TestCreateTimesheet(t *testing.T) {
	tests := []struct {
		name    string
		client  *mockClient
		params  TimesheetWriteParams
		wantID  int64
		wantErr bool
		wantMsg string
	}{
		{
			name:   "success returns new entry ID",
			client: &mockClient{createID: 42},
			params: TimesheetWriteParams{
				ProjectID: 1,
				Date:      "2026-03-09",
				Name:      "Implemented write API",
				Hours:     2.5,
			},
			wantID: 42,
		},
		{
			name: "error is propagated",
			client: &mockClient{
				createErr: errors.New("access denied"),
			},
			params: TimesheetWriteParams{
				ProjectID: 1,
				Date:      "2026-03-09",
				Name:      "Should fail",
				Hours:     1.0,
			},
			wantErr: true,
			wantMsg: "access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := tt.client.CreateTimesheet(tt.params)

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
			if id != tt.wantID {
				t.Errorf("ID = %d, want %d", id, tt.wantID)
			}
		})
	}
}

func TestUpdateTimesheet(t *testing.T) {
	tests := []struct {
		name    string
		client  *mockClient
		id      int64
		params  TimesheetWriteParams
		wantErr bool
		wantMsg string
	}{
		{
			name:   "success updates entry",
			client: &mockClient{},
			id:     42,
			params: TimesheetWriteParams{
				ProjectID: 1,
				Date:      "2026-03-09",
				Name:      "Updated description",
				Hours:     3.0,
			},
		},
		{
			name: "error is propagated",
			client: &mockClient{
				updateErr: errors.New("record locked"),
			},
			id: 42,
			params: TimesheetWriteParams{
				ProjectID: 1,
				Date:      "2026-03-09",
				Name:      "Should fail",
				Hours:     1.0,
			},
			wantErr: true,
			wantMsg: "record locked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.client.UpdateTimesheet(tt.id, tt.params)

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
		})
	}
}

func TestDeleteTimesheet(t *testing.T) {
	tests := []struct {
		name    string
		client  *mockClient
		id      int64
		wantErr bool
		wantMsg string
	}{
		{
			name:   "success deletes entry",
			client: &mockClient{},
			id:     42,
		},
		{
			name: "error is propagated",
			client: &mockClient{
				deleteErr: errors.New("not found"),
			},
			id:      999,
			wantErr: true,
			wantMsg: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.client.DeleteTimesheet(tt.id)

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
		})
	}
}
