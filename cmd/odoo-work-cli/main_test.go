package main

import (
	"testing"
	"time"

	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/spf13/cobra"
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
		hours       string
		wantHours   float64
		wantErr     bool
		wantMsg     string
	}{
		{
			name:        "valid all fields decimal",
			projectID:   42,
			taskID:      7,
			date:        "2026-03-09",
			description: "coding",
			hours:       "2.5",
			wantHours:   2.5,
		},
		{
			name:        "valid H:MM format",
			projectID:   42,
			date:        "2026-03-09",
			description: "coding",
			hours:       "2:30",
			wantHours:   2.5,
		},
		{
			name:        "valid without task",
			projectID:   42,
			date:        "2026-03-09",
			description: "coding",
			hours:       "2.5",
			wantHours:   2.5,
		},
		{
			name:        "valid with only task ID",
			taskID:      10,
			date:        "2026-03-09",
			description: "coding",
			hours:       "1",
			wantHours:   1.0,
		},
		{
			name:        "missing both project and task ID",
			date:        "2026-03-09",
			description: "coding",
			hours:       "1",
			wantErr:     true,
			wantMsg:     "project ID or task ID is required",
		},
		{
			name:        "empty date defaults to today",
			projectID:   42,
			description: "coding",
			hours:       "1",
			wantHours:   1.0,
		},
		{
			name:        "invalid date format",
			projectID:   42,
			date:        "09/03/2026",
			description: "coding",
			hours:       "1",
			wantErr:     true,
			wantMsg:     `invalid date "09/03/2026": expected YYYY-MM-DD`,
		},
		{
			name:      "missing description",
			projectID: 42,
			date:      "2026-03-09",
			hours:     "1",
			wantErr:   true,
			wantMsg:   "description is required",
		},
		{
			name:        "zero hours",
			projectID:   42,
			date:        "2026-03-09",
			description: "coding",
			hours:       "0",
			wantErr:     true,
			wantMsg:     "hours must be greater than zero",
		},
		{
			name:        "negative hours",
			projectID:   42,
			date:        "2026-03-09",
			description: "coding",
			hours:       "-1",
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
			if params.Hours != tt.wantHours {
				t.Errorf("Hours = %f, want %f", params.Hours, tt.wantHours)
			}
		})
	}
}

func TestBuildUpdateFields(t *testing.T) {
	tests := []struct {
		name       string
		flags      []string
		wantFields []string
		wantErr    bool
	}{
		{
			name:       "single field hours",
			flags:      []string{"--hours", "2.5"},
			wantFields: []string{"unit_amount"},
		},
		{
			name:       "single field description",
			flags:      []string{"--description", "new desc"},
			wantFields: []string{"name"},
		},
		{
			name:       "single field date",
			flags:      []string{"--date", "2026-03-09"},
			wantFields: []string{"date"},
		},
		{
			name:       "single field project-id",
			flags:      []string{"--project-id", "42"},
			wantFields: []string{"project_id"},
		},
		{
			name:       "hours H:MM format",
			flags:      []string{"--hours", "1:30"},
			wantFields: []string{"unit_amount"},
		},
		{
			name:       "multiple fields",
			flags:      []string{"--hours", "3.0", "--description", "updated"},
			wantFields: []string{"unit_amount", "name"},
		},
		{
			name:    "no flags errors",
			flags:   []string{},
			wantErr: true,
		},
		{
			name:    "invalid date",
			flags:   []string{"--date", "not-a-date"},
			wantErr: true,
		},
		{
			name:    "zero hours",
			flags:   []string{"--hours", "0"},
			wantErr: true,
		},
		{
			name:    "negative hours",
			flags:   []string{"--hours", "-1"},
			wantErr: true,
		},
		{
			name:    "empty description",
			flags:   []string{"--description", ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh command with the same flags as entriesUpdateCmd.
			cmd := &cobra.Command{Use: "test"}
			var pID, tID int64
			var date, desc, hours string
			cmd.Flags().Int64Var(&pID, "project-id", 0, "")
			cmd.Flags().Int64Var(&tID, "task-id", 0, "")
			cmd.Flags().StringVar(&date, "date", "", "")
			cmd.Flags().StringVar(&hours, "hours", "", "")
			cmd.Flags().StringVar(&desc, "description", "", "")

			err := cmd.ParseFlags(tt.flags)
			if err != nil {
				t.Fatalf("parsing flags: %v", err)
			}

			// Point the package-level vars at the parsed values.
			updateProjectID = pID
			updateTaskID = tID
			updateDate = date
			updateHours = hours
			updateDescription = desc

			fields, err := buildUpdateFields(cmd)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(fields) != len(tt.wantFields) {
				t.Fatalf("got %d fields, want %d", len(fields), len(tt.wantFields))
			}
			for _, key := range tt.wantFields {
				if _, ok := fields[key]; !ok {
					t.Errorf("missing expected field %q", key)
				}
			}
		})
	}
}

func TestParseEntryID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{name: "valid", input: "42", want: 42},
		{name: "zero", input: "0", wantErr: true},
		{name: "negative", input: "-1", wantErr: true},
		{name: "non-numeric", input: "abc", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEntryID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("parseEntryID(%q) = %d, want %d", tt.input, got, tt.want)
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
