package entries

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

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
		name          string
		flags         []string
		ops           subOps
		wantFields    []string
		wantValues    map[string]interface{}
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:       "single field hours",
			flags:      []string{"--hours", "2.5"},
			ops:        subOps{hours: "2.5"},
			wantFields: []string{"unit_amount"},
			wantValues: map[string]interface{}{"unit_amount": 2.5},
		},
		{
			name:       "single field description",
			flags:      []string{"--description", "new desc"},
			ops:        subOps{description: "new desc"},
			wantFields: []string{"name"},
			wantValues: map[string]interface{}{"name": "new desc"},
		},
		{
			name:       "single field date",
			flags:      []string{"--date", "2026-03-09"},
			ops:        subOps{date: "2026-03-09"},
			wantFields: []string{"date"},
			wantValues: map[string]interface{}{"date": "2026-03-09"},
		},
		{
			name:       "single field project-id",
			flags:      []string{"--project-id", "42"},
			ops:        subOps{projectID: 42},
			wantFields: []string{"project_id"},
			wantValues: map[string]interface{}{"project_id": int64(42)},
		},
		{
			name:       "single field task-id",
			flags:      []string{"--task-id", "7"},
			ops:        subOps{taskID: 7},
			wantFields: []string{"task_id"},
			wantValues: map[string]interface{}{"task_id": int64(7)},
		},
		{
			name:       "hours H:MM format",
			flags:      []string{"--hours", "1:30"},
			ops:        subOps{hours: "1:30"},
			wantFields: []string{"unit_amount"},
			wantValues: map[string]interface{}{"unit_amount": 1.5},
		},
		{
			name:       "multiple fields",
			flags:      []string{"--hours", "3.0", "--description", "updated"},
			ops:        subOps{hours: "3.0", description: "updated"},
			wantFields: []string{"unit_amount", "name"},
			wantValues: map[string]interface{}{"unit_amount": 3.0, "name": "updated"},
		},
		{
			name:          "no flags errors",
			flags:         []string{},
			wantErr:       true,
			wantErrSubstr: "at least one flag is required",
		},
		{
			name:          "invalid date",
			flags:         []string{"--date", "not-a-date"},
			ops:           subOps{date: "not-a-date"},
			wantErr:       true,
			wantErrSubstr: `invalid date "not-a-date": expected YYYY-MM-DD`,
		},
		{
			name:          "zero hours",
			flags:         []string{"--hours", "0"},
			ops:           subOps{hours: "0"},
			wantErr:       true,
			wantErrSubstr: "hours must be greater than zero",
		},
		{
			name:          "negative hours",
			flags:         []string{"--hours", "-1"},
			ops:           subOps{hours: "-1"},
			wantErr:       true,
			wantErrSubstr: "hours must be greater than zero",
		},
		{
			name:          "empty description",
			flags:         []string{"--description", ""},
			ops:           subOps{description: ""},
			wantErr:       true,
			wantErrSubstr: "description must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().Int64("project-id", 0, "")
			cmd.Flags().Int64("task-id", 0, "")
			cmd.Flags().String("date", "", "")
			cmd.Flags().String("hours", "", "")
			cmd.Flags().String("description", "", "")

			if err := cmd.ParseFlags(tt.flags); err != nil {
				t.Fatalf("parsing flags: %v", err)
			}

			fields, err := buildUpdateFields(cmd, &tt.ops)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrSubstr != "" && !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("error = %q, want substring %q", err.Error(), tt.wantErrSubstr)
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
				got, ok := fields[key]
				if !ok {
					t.Errorf("missing expected field %q", key)
					continue
				}
				if want, ok := tt.wantValues[key]; ok && got != want {
					t.Errorf("field %q = %#v, want %#v", key, got, want)
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
