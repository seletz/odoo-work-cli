package main

import (
	"testing"
	"time"

	"github.com/seletz/odoo-work-cli/internal/odoo"
)

func TestWeekDateRange(t *testing.T) {
	tests := []struct {
		name     string
		week     string
		wantFrom string
		wantTo   string
		wantErr  bool
	}{
		{
			name:     "2026-W10",
			week:     "2026-W10",
			wantFrom: "2026-03-02",
			wantTo:   "2026-03-08",
		},
		{
			name:     "2026-W01",
			week:     "2026-W01",
			wantFrom: "2025-12-29",
			wantTo:   "2026-01-04",
		},
		{
			name:     "2025-W52",
			week:     "2025-W52",
			wantFrom: "2025-12-22",
			wantTo:   "2025-12-28",
		},
		{
			name:    "invalid format",
			week:    "not-a-week",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to, err := weekDateRange(tt.week)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if from != tt.wantFrom {
				t.Errorf("from = %q, want %q", from, tt.wantFrom)
			}
			if to != tt.wantTo {
				t.Errorf("to = %q, want %q", to, tt.wantTo)
			}
		})
	}
}

func TestParseDateRange(t *testing.T) {
	tests := []struct {
		name     string
		date     string
		wantFrom string
		wantTo   string
		wantErr  bool
	}{
		{
			name:     "valid date",
			date:     "2026-03-05",
			wantFrom: "2026-03-05",
			wantTo:   "2026-03-05",
		},
		{
			name:    "invalid date",
			date:    "not-a-date",
			wantErr: true,
		},
		{
			name:    "wrong format",
			date:    "05/03/2026",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to, err := parseDateRange(tt.date)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if from != tt.wantFrom {
				t.Errorf("from = %q, want %q", from, tt.wantFrom)
			}
			if to != tt.wantTo {
				t.Errorf("to = %q, want %q", to, tt.wantTo)
			}
		})
	}
}

func TestBuildTimesheetWriteParams(t *testing.T) {
	tests := []struct {
		name        string
		projectID   int64
		taskID      int64
		date        string
		description string
		hours       float64
		wantErr     bool
		wantMsg     string
	}{
		{
			name:        "valid all fields",
			projectID:   42,
			taskID:      7,
			date:        "2026-03-09",
			description: "coding",
			hours:       2.5,
		},
		{
			name:        "valid without task",
			projectID:   42,
			date:        "2026-03-09",
			description: "coding",
			hours:       2.5,
		},
		{
			name:        "valid with only task ID",
			taskID:      10,
			date:        "2026-03-09",
			description: "coding",
			hours:       1.0,
		},
		{
			name:        "missing both project and task ID",
			date:        "2026-03-09",
			description: "coding",
			hours:       1.0,
			wantErr:     true,
			wantMsg:     "project ID or task ID is required",
		},
		{
			name:        "empty date defaults to today",
			projectID:   42,
			description: "coding",
			hours:       1.0,
		},
		{
			name:        "invalid date format",
			projectID:   42,
			date:        "09/03/2026",
			description: "coding",
			hours:       1.0,
			wantErr:     true,
			wantMsg:     `invalid date "09/03/2026": expected YYYY-MM-DD`,
		},
		{
			name:      "missing description",
			projectID: 42,
			date:      "2026-03-09",
			hours:     1.0,
			wantErr:   true,
			wantMsg:   "description is required",
		},
		{
			name:        "zero hours",
			projectID:   42,
			date:        "2026-03-09",
			description: "coding",
			hours:       0,
			wantErr:     true,
			wantMsg:     "hours must be greater than zero",
		},
		{
			name:        "negative hours",
			projectID:   42,
			date:        "2026-03-09",
			description: "coding",
			hours:       -1.0,
			wantErr:     true,
			wantMsg:     "hours must be greater than zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := buildTimesheetWriteParams(tt.projectID, tt.taskID, tt.date, tt.description, tt.hours)

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
			if params.ProjectID != tt.projectID {
				t.Errorf("ProjectID = %d, want %d", params.ProjectID, tt.projectID)
			}
			if params.TaskID != tt.taskID {
				t.Errorf("TaskID = %d, want %d", params.TaskID, tt.taskID)
			}
			wantDate := tt.date
			if wantDate == "" {
				wantDate = time.Now().Format("2006-01-02")
			}
			if params.Date != wantDate {
				t.Errorf("Date = %q, want %q", params.Date, wantDate)
			}
			if params.Name != tt.description {
				t.Errorf("Name = %q, want %q", params.Name, tt.description)
			}
			if params.Hours != tt.hours {
				t.Errorf("Hours = %f, want %f", params.Hours, tt.hours)
			}
		})
	}
}

func TestFilterEntries(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{ID: 1, Date: "2026-03-02", Project: "Acme Corp", Task: "Backend Dev", Name: "Auth endpoint", Hours: 2.0, ValidatedStatus: "draft"},
		{ID: 2, Date: "2026-03-02", Project: "Acme Corp", Task: "QA Testing", Name: "Review PR", Hours: 1.5, ValidatedStatus: "validated"},
		{ID: 3, Date: "2026-03-03", Project: "Beta Project", Task: "Frontend Dev", Name: "Dashboard", Hours: 4.0, ValidatedStatus: "draft"},
	}

	tests := []struct {
		name    string
		project string
		task    string
		status  string
		wantIDs []int64
	}{
		{
			name:    "no filter",
			wantIDs: []int64{1, 2, 3},
		},
		{
			name:    "filter by project",
			project: "acme",
			wantIDs: []int64{1, 2},
		},
		{
			name:    "filter by task",
			task:    "dev",
			wantIDs: []int64{1, 3},
		},
		{
			name:    "filter by both",
			project: "acme",
			task:    "qa",
			wantIDs: []int64{2},
		},
		{
			name:    "no match",
			project: "nonexistent",
			wantIDs: nil,
		},
		{
			name:    "filter by status draft",
			status:  "draft",
			wantIDs: []int64{1, 3},
		},
		{
			name:    "filter by status validated",
			status:  "validated",
			wantIDs: []int64{2},
		},
		{
			name:    "filter by status and project",
			project: "acme",
			status:  "draft",
			wantIDs: []int64{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterEntries(entries, tt.project, tt.task, tt.status)
			if len(got) != len(tt.wantIDs) {
				t.Fatalf("got %d entries, want %d", len(got), len(tt.wantIDs))
			}
			for i, e := range got {
				if e.ID != tt.wantIDs[i] {
					t.Errorf("entry[%d].ID = %d, want %d", i, e.ID, tt.wantIDs[i])
				}
			}
		})
	}
}

